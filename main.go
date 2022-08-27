package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/sacOO7/gowebsocket"
	"gopkg.in/resty.v1"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

// optional port variable. example: `gex -p 30057`
var givenPort = flag.String("p", "26657", "port to connect to as a string")
var givenNetwork = flag.String("n", "localhost", "network type")
var numberOfIterations = flag.String("i", "100", "number of blocks to calculate this value.")

var appHost = "localhost"

// Info describes a list of types with data that are used in the explorer
type Info struct {
	blocks       *Blocks
	transactions *Transactions
}

// Blocks describe content that gets parsed for blocks
type Blocks struct {
	amount               int
	secondsPassed        int
	totalGasWanted       int64
	gasWantedLatestBlock int64
	maxGasWanted         int64
	lastTx               int64
}

// Transactions describe content that gets parsed for transactions
type Transactions struct {
	amount uint64
}

// WaitGroup is used to wait for the program to finish goroutines.
var wg sync.WaitGroup
var actualNumberofIterations int
var targetNumberofIterations int

func main() {

	// Init internal variables
	info := Info{}
	info.blocks = new(Blocks)
	info.transactions = new(Transactions)

	connectionSignal := make(chan string)
	flag.Parse()

	if *givenNetwork == "localhost" {
		appHost = "localhost"
	} else if *givenNetwork == "mainnet" {
		appHost = "34.94.27.30"
	} else if *givenNetwork == "testnet" {
		appHost = "34.82.40.187"
	}

	networkInfo := getFromRPC("status")
	networkStatus := gjson.Parse(networkInfo)
	if !networkStatus.Exists() {
		panic("Application not running on localhost:" + fmt.Sprintf("%s", *givenPort))
	}

	if *numberOfIterations == "" {
		targetNumberofIterations = 100
	} else {
		targetNumberofIterations, _ = strconv.Atoi(*numberOfIterations)
	}
	ctx, _ := context.WithCancel(context.Background())
	// system powered widgets
	go writeTime(ctx, info, 1*time.Second)

	// websocket powered widgets
	result := make(chan BlockTime, 1)
	go writeHealth(ctx, 500*time.Millisecond, connectionSignal)
	go writeBlocks(ctx, info, connectionSignal)
	wg.Add(1)

	//fmt.Printf("the block time is %f over %v blocks", bt.blockTime, bt.numberOfBlocks)
	// Wait for the goroutines to finish.
	go writeSecondsPerBlock(ctx, info, 1*time.Second, result)
	value := <-result
	fmt.Printf("Time between blocks: %v\n", value)
	close(result)
	wg.Wait()
}

type BlockTime struct {
	blockTime      float64
	numberOfBlocks int
}

// Get Data from RPC Endpoint
func getFromRPC(endpoint string) string {
	port := *givenPort
	resp, _ := resty.R().
		SetHeader("Cache-Control", "no-cache").
		SetHeader("Content-Type", "application/json").
		Get("http://" + appHost + ":" + port + "/" + endpoint)

	return resp.String()
}

// writeSecondsPerBlock writes the status to the Time per block.
// Exits when the context expires.
func writeSecondsPerBlock(ctx context.Context, info Info, delay time.Duration, result chan BlockTime) {

	defer wg.Done()
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var blocksPerSecond float64
			if info.blocks.secondsPassed != 0 {
				blocksPerSecond = float64(info.blocks.secondsPassed) / float64(info.blocks.amount)
			}
			fmt.Printf("On iteration: %v of total iteration wanted %v ... \n", actualNumberofIterations, targetNumberofIterations)
			if actualNumberofIterations >= targetNumberofIterations {
				result <- BlockTime{
					blockTime:      blocksPerSecond,
					numberOfBlocks: info.blocks.amount,
				}
				return
			}

		case <-ctx.Done():
			return
		}
	}
	return
}

// writeTime writes the current system time to the timeWidget.
// Exits when the context expires.
func writeTime(ctx context.Context, info Info, delay time.Duration) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			info.blocks.secondsPassed++
		case <-ctx.Done():
			return
		}
	}
}

// WEBSOCKET WIDGETS

// writeHealth writes the status to the healthWidget.
// Exits when the context expires.
func writeHealth(ctx context.Context, delay time.Duration, connectionSignal chan string) {
	reconnect := false

	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			health := gjson.Get(getFromRPC("health"), "result")
			if health.Exists() {
				if reconnect == true {
					connectionSignal <- "reconnect"
					connectionSignal <- "reconnect"
					connectionSignal <- "reconnect"
					reconnect = false
				}
			} else {
				if reconnect == false {
					connectionSignal <- "no_connection"
					connectionSignal <- "no_connection"
					connectionSignal <- "no_connection"
					reconnect = true
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// writeBlocks writes the latest Block to the blocksWidget.
// Exits when the context expires.
func writeBlocks(ctx context.Context, info Info, connectionSignal <-chan string) {

	port := *givenPort
	socket := gowebsocket.New("ws://" + appHost + ":" + port + "/websocket")

	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		currentBlock := gjson.Get(message, "result.data.value.block.header.height")
		if currentBlock.String() != "" {
			info.blocks.amount++
			actualNumberofIterations++
		}

	}

	socket.Connect()

	socket.SendText("{ \"jsonrpc\": \"2.0\", \"method\": \"subscribe\", \"params\": [\"tm.event='NewBlock'\"], \"id\": 1 }")

	for {
		select {
		case s := <-connectionSignal:
			if s == "no_connection" {
				socket.Close()
			}
			if s == "reconnect" {
				writeBlocks(ctx, info, connectionSignal)
			}
		case <-ctx.Done():
			log.Println("interrupt")
			socket.Close()
			return
		}
	}
}

func numberWithComma(n int64) string {
	in := strconv.FormatInt(n, 10)
	numOfDigits := len(in)
	if n < 0 {
		numOfDigits-- // First character is the - sign (not a digit)
	}
	numOfCommas := (numOfDigits - 1) / 3

	out := make([]byte, len(in)+numOfCommas)
	if n < 0 {
		in, out[0] = in[1:], '-'
	}

	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ','
		}
	}
}
