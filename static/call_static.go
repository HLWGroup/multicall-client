package static

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
)

// Opts call opts for static calls
type Opts struct {
	From        common.Address
	To          common.Address
	Gas         uint64
	GasPrice    *big.Int
	GasFeeCap   *big.Int
	GasTipCap   *big.Int
	Value       *big.Int
	AccessLists types.AccessList
}

// CallStaticWithName calls a function statically (not be mined into the blockchain) and returns its results.
// using 0 gas will allow infinite gas to be used
func CallStaticWithName(client *ethclient.Client, abi *abi.ABI, opts *Opts, methodName string, args ...any) (results []any, err error) {
	method, ok := abi.Methods[methodName]
	if !ok {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	return callStatic(client, opts, method, args...)
}

// CallStaticWithMethod callStaticWithName calls a function statically (not be mined into the blockchain) and returns its results.
// using 0 gas will allow infinite gas to be used
func CallStaticWithMethod(client *ethclient.Client, opts *Opts, method abi.Method, args ...any) (results []any, err error) {
	return callStatic(client, opts, method, args...)
}

func callStatic(client *ethclient.Client, opts *Opts, method abi.Method, args ...any) (results []any, err error) {
	arguments, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{
		From:       opts.From,
		To:         &opts.To,
		Data:       append(method.ID, arguments...),
		Gas:        opts.Gas,
		GasPrice:   opts.GasPrice,
		GasFeeCap:  opts.GasFeeCap,
		GasTipCap:  opts.GasTipCap,
		Value:      opts.Value,
		AccessList: opts.AccessLists,
	}

	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, err
	}

	results, err = method.Outputs.Unpack(result)
	return
}
