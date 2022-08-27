# provenance go blockchain time
Simple program which tell's the average block cut time of a cosmos based chain.
Currently it is configured towards `provenance mainet` or `testnet`
`or any comsos based chain running locally`
example commands:
`go run main.go "mainnet"`

`go run main.go "testnet"`

for local:
`go run main.go`

by default this program will calculate avg block time over 100 blocks, but if you want shorter 
duration use "-i "<number of blocks wanted>""
e.g 
`go run main.go -n "mainnet" -i "5"`

To install the go-blockchain-time binary:

`go install github.com/arnabmitra/go-blockchain-time@latest` 