# multicall-client

A wrapper for [Multicall](https://github.com/makerdao/multicall) that allows you to make batched calls to the blockchain.

### Usage
```go
// Create a new multicall client
client, err := multicall.NewClient(multicall.CronosMainnet, multicall.MulticallV1, ethClient)
if err != nil {
	panic(err)
}

calls := []multicall.MethodCall{
    // dead address balance of token
    multicall.NewMethodCall("0x5C7F8A570d578ED84E63fdFA7b1eE72dEae1AE23", ERC20Abi.Methods["balanceOf"], "0x000000000000000000000000000000000000dead"),
	// name of token
    multicall.NewMethodCall("0x5C7F8A570d578ED84E63fdFA7b1eE72dEae1AE23", ERC20Abi.Methods["name"]),
}

blockNumber, results, err := client.Aggregate(nil, calls)
if err != nil {
	panic(err)
}

balance := results[0].(*big.Int)
name := results[1].(string)
fmt.Printf("Balance of %s in dead address: %s\n", name, balance.String())
```