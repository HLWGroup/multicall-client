package multicall

import "github.com/ethereum/go-ethereum/common"

type callData struct {
	Target   common.Address
	CallData []byte
}

func newCallData(method *MethodCall) (*callData, error) {
	data, err := method.getCallData()
	if err != nil {
		return nil, err
	}

	return &callData{
		Target:   method.Address,
		CallData: data,
	}, nil
}
