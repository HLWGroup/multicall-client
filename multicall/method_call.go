package multicall

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type MethodCall struct {
	Address common.Address
	Method  abi.Method
	Args    []any
}

// NewMethodCall creates a new method call
func NewMethodCall(address common.Address, method abi.Method, args ...any) *MethodCall {
	return &MethodCall{
		Address: address,
		Method:  method,
		Args:    args,
	}
}

// getCallData returns the call data for the method call
func (c *MethodCall) getCallData() ([]byte, error) {
	data, err := c.Method.Inputs.Pack(c.Args...)
	if err != nil {
		return nil, err
	}

	return append(c.Method.ID, data...), nil
}
