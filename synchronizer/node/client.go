package node

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"log"
	"math/big"
	"time"
)

type myClient struct {
	rpc RPC
}

func (m *myClient) BlockHeadersByRange(startHeight *big.Int, engHeight *big.Int, chainId uint) ([]types.Header, error) {
	if startHeight.Cmp(engHeight) == 0 {
		header, err := m.BlockHeaderByNumber(startHeight)
		if err != nil {
			return nil, err
		}
		return []types.Header{*header}, nil
	}

	count := new(big.Int).Sub(engHeight, startHeight).Uint64() + 1
	headers := make([]types.Header, count)
	batchElems := make([]rpc.BatchElem, count)

	for i := uint64(0); i < count; i++ {
		height := new(big.Int).Add(startHeight, new(big.Int).SetUint64(i))
		batchElems[i] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args:   []interface{}{toBlockNumArg(height), false},
			Result: &headers[i],
		}
	}

	ctxwt, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	err := m.rpc.BatchCallContext(ctxwt, batchElems)
	if err != nil {
		return nil, err
	}

	size := 0
	for i, batchElem := range batchElems {
		header, ok := batchElem.Result.(*types.Header)
		if !ok {
			return nil, fmt.Errorf("unable to transform rpc response %v into utils.Header", batchElem.Result)
		}
		headers[i] = *header

		size = size + 1
	}
	headers = headers[:size]

	return headers, nil
}

func (m *myClient) BlockHeaderByNumber(b *big.Int) (*types.Header, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var header *types.Header
	err := m.rpc.CallContext(ctxwt, &header, "eth_getBlockByNumber", toBlockNumArg(b), false)
	if err != nil {
		log.Fatalln("Call eth_getBlockByNumber method fail", "err", err)
		return nil, err
	} else if header == nil {
		log.Println("header not found")
		return nil, ethereum.NotFound
	}
	return header, nil
}

func (m *myClient) LatestBlockNumber() (*types.Header, error) {
	//TODO implement me
	panic("implement me")
}

func (m *myClient) LatestFinalizedBlockHeader() (*types.Header, error) {
	//TODO implement me
	panic("implement me")
}

func (m *myClient) BlockHeaderByHash(hash common.Hash) (*types.Header, error) {
	//TODO implement me
	panic("implement me")
}

func (m *myClient) TxByHash(hash common.Hash) (*types.Transaction, error) {
	//TODO implement me
	panic("implement me")
}

func (m *myClient) StorageHash(address common.Address, b *big.Int) (common.Hash, error) {
	//TODO implement me
	panic("implement me")
}

func (m *myClient) FilterLogs(query ethereum.FilterQuery) (Logs, error) {
	args, err := toFilterLog(query)
	if err != nil {
		return Logs{}, err
	}
	var header types.Header
	var logs []types.Log

	batchElems := make([]rpc.BatchElem, 2)

	batchElems[0] = rpc.BatchElem{
		Method: "eth_getBlockByNumber",
		Args:   []interface{}{toBlockNumArg(query.ToBlock), false},
		Result: &header,
	}
	batchElems[1] = rpc.BatchElem{
		Method: "eth_getLogs",
		Args:   []interface{}{args},
		Result: &logs,
	}
	ctxwt, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	err = m.rpc.BatchCallContext(ctxwt, batchElems)
	if err != nil {
		return Logs{}, err
	}
	if batchElems[0].Error != nil {
		return Logs{}, fmt.Errorf("unable to query for the `FilterQuery#ToBlock` header: %w", batchElems[0].Error)
	}
	if batchElems[1].Error != nil {
		return Logs{}, fmt.Errorf("unable to query logs: %w", batchElems[1].Error)
	}

	return Logs{Logs: logs, ToBlockHeader: &header}, nil
}

func (m *myClient) Close() {
	//TODO implement me
	panic("implement me")
}

type RPC interface {
	Close()
	CallContext(ctx context.Context, result any, method string, args ...any) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
}

type Logs struct {
	Logs          []types.Log
	ToBlockHeader *types.Header
}

type EthClient interface {
	BlockHeaderByNumber(*big.Int) (*types.Header, error)
	LatestBlockNumber() (*types.Header, error)
	LatestFinalizedBlockHeader() (*types.Header, error)
	BlockHeaderByHash(hash common.Hash) (*types.Header, error)
	BlockHeadersByRange(*big.Int, *big.Int, uint) ([]types.Header, error)

	TxByHash(hash common.Hash) (*types.Transaction, error)

	StorageHash(common.Address, *big.Int) (common.Hash, error)
	FilterLogs(query ethereum.FilterQuery) (Logs, error)

	Close()
}

func DialEthClient(ctx context.Context, rpcUrl string) (EthClient, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	rpcClient, err := rpc.DialContext(ctx, rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to dial address (%s): %w", rpcUrl, err)
	}

	return &myClient{
		rpc: NewRPC(rpcClient),
	}, nil
}

type rpcClient struct {
	rpc *rpc.Client
}

func NewRPC(client *rpc.Client) RPC {
	return &rpcClient{client}
}

func (c *rpcClient) Close() {
	c.rpc.Close()
}

func (c *rpcClient) CallContext(ctx context.Context, result any, method string, args ...any) error {
	err := c.rpc.CallContext(ctx, result, method, args...)
	return err
}

func (c *rpcClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	err := c.rpc.BatchCallContext(ctx, b)
	return err
}

func toBlockNumArg(b *big.Int) string {
	if b == nil {
		return "latest"
	}
	if b.Sign() >= 0 {
		return hexutil.EncodeBig(b)
	}
	return rpc.BlockNumber(b.Int64()).String()
}

func toFilterLog(q ethereum.FilterQuery) (interface{}, error) {
	arg := map[string]interface{}{"address": q.Addresses, "topics": q.Topics}
	if q.BlockHash != nil {
		arg["blockHash"] = *q.BlockHash
		if q.FromBlock != nil || q.ToBlock != nil {
			return nil, errors.New("cannot specify both BlockHash and FromBlock/ToBlock")
		}
	} else {
		if q.FromBlock != nil {
			arg["fromBlock"] = toBlockNumArg(q.FromBlock)
		} else {
			arg["fromBlock"] = "0x0"
		}
		if q.ToBlock != nil {
			arg["toBlock"] = toBlockNumArg(q.ToBlock)
		}
	}
	return arg, nil
}
