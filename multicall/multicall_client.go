package multicall

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
)

// A bunch of commonly used solidity types
var (
	defaultGasLimit  = big.NewInt(60_000)
	defaultSizeLimit = big.NewInt(1024)
)

// Errors
var (
	ErrQuickCallRequired = errors.New("quickcall is required for this function")
	ErrMulticallRequired = errors.New("multicall is required for this function")
)

var (
	//go:embed multicall_v1.json
	multicallV1Abi string
	multicallV1    = &bind.MetaData{ABI: multicallV1Abi}

	//go:embed quickcall.json
	quickCallAbi string
	quickcall    = &bind.MetaData{ABI: quickCallAbi}
)

type Version int

const (
	V1 Version = iota
	QuickCall
)

func (v Version) getAbi() (*abi.ABI, error) {
	switch v {
	case V1:
		return multicallV1.GetAbi()
	case QuickCall:
		return quickcall.GetAbi()
	default:
		return nil, fmt.Errorf("unknown version %d", v)
	}
}

type Client struct {
	client   *ethclient.Client
	address  common.Address
	contract *bind.BoundContract
	version  Version
}

// New creates a new client for the given address and version of multicall
func New(address string, version Version, client *ethclient.Client) (*Client, error) {
	parsed, err := version.getAbi()
	if err != nil {
		return nil, err
	}

	addr := common.HexToAddress(address)

	return &Client{
		client,
		addr,
		bind.NewBoundContract(addr, *parsed, client, client, client),
		version,
	}, nil
}

// Aggregate makes multiple calls to the target contracts and returns the results.
func (client *Client) Aggregate(opts *bind.CallOpts, methods []*MethodCall) (blockNumber *big.Int, results []any, err error) {
	if client.version == QuickCall {
		return nil, nil, ErrMulticallRequired
	}

	// Converts method calls into call data for calling the aggregate function.
	calls, err := methodCallsToCallData(methods)
	if err != nil {
		panic(err)
	}

	// callData the aggregate function.
	var out []interface{}
	err = client.contract.Call(opts, &out, "aggregate", calls)
	if err != nil {
		return nil, nil, err
	}

	// Build our result
	blockNumber = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	raw := *abi.ConvertType(out[1], new([][]byte)).(*[][]byte)

	// Unpack results from the raw bytes.
	err = unpackResults(results, raw, methods...)
	if err != nil {
		return nil, nil, err
	}

	return
}

type QuickCallResults struct {
	BlockNumber *big.Int
	Results     []QuickCallResult
}

type QuickCallResult struct {
	Status uint64
	Result any
}

// IsStatusZero returns true if the status is zero, false otherwise.
// This is useful for checking if a call was successful or not.
// Typically, a status of zero means the call has reverted or failed.
func (qcr *QuickCallResult) IsStatusZero() bool {
	return qcr.Status == 0
}

// Execute makes multiple calls to the target contracts and returns the results.
func (client *Client) Execute(opts *bind.CallOpts, gasLimitPerCall *big.Int, resultSizeLimit *big.Int, methods []*MethodCall) (result *QuickCallResults, err error) {
	if client.version != QuickCall {
		return nil, ErrQuickCallRequired
	}

	// set default gas limit per call to 60K
	if gasLimitPerCall == nil {
		gasLimitPerCall = defaultGasLimit
	}

	// set default size limit to 1024
	if resultSizeLimit == nil {
		resultSizeLimit = defaultSizeLimit
	}

	// Converts method calls into call data for calling the aggregate function.
	calls, err := methodCallsToCallData(methods)
	if err != nil {
		panic(err)
	}

	// extract addresses and call data from method calls
	var addresses []common.Address
	var data [][]byte
	for _, call := range calls {
		addresses = append(addresses, call.Target)
		data = append(data, call.CallData)
	}

	var out []interface{}
	err = client.contract.Call(opts, &out, "execute", gasLimitPerCall, resultSizeLimit, addresses, data)
	if err != nil {
		return nil, err
	}

	result.BlockNumber = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	statuses := *abi.ConvertType(out[1], new([]uint64)).(*[]uint64)
	raw := *abi.ConvertType(out[2], new([][]byte)).(*[][]byte)

	var results []any
	// Unpack results from the raw bytes.
	err = unpackResults(results, raw, methods...)
	if err != nil {
		return nil, err
	}

	result.Results = make([]QuickCallResult, len(statuses))
	for i, status := range statuses {
		result.Results[i] = QuickCallResult{
			Status: status,
			Result: results[i],
		}
	}
	return
}

// unpackResults unpacks the raw bytes into the results.
func unpackResults(results []any, raw [][]byte, methods ...*MethodCall) error {
	// If we have an unexpected length, return an error.
	if len(raw) != len(methods) {
		return errors.New(fmt.Sprintf("unexpected number of results, expected %d but got %d", len(methods), len(raw)))
	}

	// Unpack results from the raw bytes.
	for i, call := range methods {

		// Our result is empty, so we'll just return nil.
		if len(raw[i]) == 0 {
			results[i] = nil
			continue
		}

		// Unpack the result into the method output types.
		// Sadly we'll need to cast this later on.
		unpack, err := call.Method.Outputs.Unpack(raw[i])
		if err != nil {
			return err
		}

		// Unpacks multiple results into a slice, if we return more than one value.
		for _, u := range unpack {
			results = append(results, u.(any))
		}
	}
	return nil
}

// methodCallsToCallData converts method calls into call data, used internally by the client.
func methodCallsToCallData(calls []*MethodCall) ([]callData, error) {
	var structs []callData
	for _, call := range calls {
		data, err := call.getCallData()
		if err != nil {
			return nil, err
		}

		structs = append(structs, callData{
			Target:   call.Address,
			CallData: data,
		})
	}

	return structs, nil
}

// GetEthBalance returns the balance of the given address. Note: not available with quickcall.
func (client *Client) GetEthBalance(address common.Address) (*big.Int, error) {
	if client.version == QuickCall {
		return nil, ErrMulticallRequired
	}

	var out []interface{}
	err := client.contract.Call(nil, &out, "getEthBalance", address)
	if err != nil {
		return nil, err
	}

	return *abi.ConvertType(out[0], new(*big.Int)).(**big.Int), nil
}

// GetBlockHash returns the block hash for the given block number. Note: not available with quickcall.
func (client *Client) GetBlockHash(blockNumber *big.Int) (*common.Hash, error) {
	if client.version == QuickCall {
		return nil, ErrMulticallRequired
	}

	var out []interface{}
	err := client.contract.Call(nil, &out, "getBlockHash", blockNumber)
	if err != nil {
		return nil, err
	}

	return *abi.ConvertType(out[0], new(*common.Hash)).(**common.Hash), nil
}

// GetLastBlockHash returns the hash of the last block. Note: not available with quickcall.
func (client *Client) GetLastBlockHash() (*common.Hash, error) {
	if client.version == QuickCall {
		return nil, ErrMulticallRequired
	}

	var out []interface{}
	err := client.contract.Call(nil, &out, "getLastBlockHash")
	if err != nil {
		return nil, err
	}

	return *abi.ConvertType(out[0], new(*common.Hash)).(**common.Hash), nil
}

// GetCurrentBlockTimestamp returns the current block timestamp. Note: not available with quickcall.
func (client *Client) GetCurrentBlockTimestamp() (uint64, error) {
	if client.version == QuickCall {
		return 0, ErrMulticallRequired
	}

	var out []interface{}
	err := client.contract.Call(nil, &out, "getCurrentBlockTimestamp")
	if err != nil {
		return 0, err
	}

	return *abi.ConvertType(out[0], new(*uint64)).(*uint64), nil
}

// GetCurrentBlockDifficulty returns the current block difficulty. Note: not available with quickcall.
func (client *Client) GetCurrentBlockDifficulty() (*big.Int, error) {
	if client.version == QuickCall {
		return nil, ErrMulticallRequired
	}

	var out []interface{}
	err := client.contract.Call(nil, &out, "getCurrentBlockDifficulty")
	if err != nil {
		return nil, err
	}

	return *abi.ConvertType(out[0], new(*big.Int)).(**big.Int), nil
}

// GetCurrentBlockGasLimit returns the current block gas limit. Note: not available with quickcall.
func (client *Client) GetCurrentBlockGasLimit() (*big.Int, error) {
	if client.version == QuickCall {
		return nil, ErrMulticallRequired
	}

	var out []interface{}
	err := client.contract.Call(nil, &out, "getCurrentBlockGasLimit")
	if err != nil {
		return nil, err
	}

	return *abi.ConvertType(out[0], new(*big.Int)).(**big.Int), nil
}

// GetCurrentBlockCoinbase returns the current block coinbase. Note: not available with quickcall.
func (client *Client) GetCurrentBlockCoinbase() (*common.Address, error) {
	if client.version == QuickCall {
		return nil, ErrMulticallRequired
	}

	var out []interface{}
	err := client.contract.Call(nil, &out, "getCurrentBlockCoinbase")
	if err != nil {
		return nil, err
	}

	return *abi.ConvertType(out[0], new(*common.Address)).(**common.Address), nil
}
