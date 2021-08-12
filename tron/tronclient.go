package tron

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/maticnetwork/bor/accounts/abi"
	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/heimdall/contracts/rootchain"
	"github.com/maticnetwork/heimdall/tron/pb"
	"github.com/maticnetwork/heimdall/types"
	"google.golang.org/grpc"
)

// Client defines typed wrappers for the Tron RPC API.
type Client struct {
	client       pb.WalletClient
	rootchainABI abi.ABI
}

// NewClient creates a client that uses the given RPC client.
func NewClient(url string) *Client {
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		os.Exit(0)
	}
	rootchainABI, err := getABI(rootchain.RootchainABI)
	if err != nil {
		os.Exit(0)
	}
	return &Client{
		client:       pb.NewWalletClient(conn),
		rootchainABI: rootchainABI,
	}
}

//
// private abi methods
//
func getABI(data string) (abi.ABI, error) {
	return abi.JSON(strings.NewReader(data))
}

func (tc *Client) TriggerContract(ownerAddress, contractAddress string, data []byte) (*pb.Transaction, error) {
	response, err := tc.client.TriggerContract(context.Background(),
		&pb.TriggerSmartContract{
			OwnerAddress:    common.FromHex("41" + ownerAddress),
			ContractAddress: common.FromHex(contractAddress),
			CallValue:       0,
			Data:            data,
			CallTokenValue:  0,
			TokenId:         0,
		})
	if err != nil {
		return nil, err
	}
	if response.Result.Code != pb.Return_SUCCESS {
		return nil, fmt.Errorf("code:%v message:%v", response.Result.Code, string(response.Result.Message))
	}
	return response.Transaction, nil
}

func (tc *Client) TriggerConstantContract(contractAddress string, data []byte) ([]byte, error) {
	response, err := tc.client.TriggerConstantContract(context.Background(),
		&pb.TriggerSmartContract{
			OwnerAddress:    nil,
			ContractAddress: common.FromHex(contractAddress),
			CallValue:       0,
			Data:            data,
			CallTokenValue:  0,
			TokenId:         0,
		})
	if err != nil {
		return nil, err
	}
	if response.Result.Code != pb.Return_SUCCESS {
		return nil, fmt.Errorf("code:%v message:%v", response.Result.Code, string(response.Result.Message))
	}
	return response.ConstantResult[0], nil
}

func (tc *Client) GetNowBlock(ctx context.Context) (int64, error) {
	block, err := tc.client.GetNowBlock2(ctx, &pb.EmptyMessage{})
	if err != nil {
		return 0, err
	}
	return block.BlockHeader.RawData.Number, nil
}

// CurrentHeaderBlock is a free data retrieval call binding the contract method 0xec7e4855.
//
// Solidity: function currentHeaderBlock() view returns(uint256)
func (tc *Client) CurrentHeaderBlock(contractAddress string, childBlockInterval uint64) (uint64, error) {
	// Pack the input
	btsPack, err := tc.rootchainABI.Pack("currentHeaderBlock")
	if err != nil {
		return 0, err
	}

	// Call
	data, err := tc.TriggerConstantContract(contractAddress, btsPack)
	if err != nil {
		return 0, err
	}

	// Unpack the results
	var (
		ret0 = new(*big.Int)
	)
	if err = tc.rootchainABI.Unpack(ret0, "currentHeaderBlock", data); err != nil {
		return 0, nil
	}
	return (*ret0).Uint64() / childBlockInterval, nil
}

// HeaderBlocks is a free data retrieval call binding the contract method 0x41539d4a.
//
// Solidity: function headerBlocks(uint256 ) view returns(bytes32 root, uint256 start, uint256 end, uint256 createdAt, address proposer)
func (tc *Client) GetHeaderInfo(number uint64, contractAddress string, childBlockInterval uint64) (
	root common.Hash,
	start uint64,
	end uint64,
	createdAt uint64,
	proposer types.HeimdallAddress,
	err error,
) {
	// Pack the input
	btsPack, err := tc.rootchainABI.Pack("headerBlocks",
		big.NewInt(0).Mul(big.NewInt(0).SetUint64(number), big.NewInt(0).SetUint64(childBlockInterval)))
	if err != nil {
		return root, 0, 0, 0, types.HeimdallAddress{}, err
	}

	// Call
	data, err := tc.TriggerConstantContract(contractAddress, btsPack)
	if err != nil {
		return root, 0, 0, 0, types.HeimdallAddress{}, err
	}

	// Unpack the results
	ret := new(struct {
		Root      [32]byte
		Start     *big.Int
		End       *big.Int
		CreatedAt *big.Int
		Proposer  common.Address
	})
	if err = tc.rootchainABI.Unpack(ret, "headerBlocks", data); err != nil {
		return root, 0, 0, 0, types.HeimdallAddress{}, err
	}

	return ret.Root, ret.Start.Uint64(), ret.End.Uint64(),
		ret.CreatedAt.Uint64(), types.HeimdallAddress(ret.Proposer), nil
}

// GetLastChildBlock is a free data retrieval call binding the contract method 0xb87e1b66.
//
// Solidity: function getLastChildBlock() view returns(uint256)
func (tc *Client) GetLastChildBlock(contractAddress string) (uint64, error) {
	// Pack the input
	btsPack, err := tc.rootchainABI.Pack("getLastChildBlock")
	if err != nil {
		return 0, err
	}
	data, err := tc.TriggerConstantContract(contractAddress, btsPack)
	if err != nil {
		return 0, err
	}
	// Unpack the results
	var (
		ret0 = new(*big.Int)
	)
	if err = tc.rootchainABI.Unpack(ret0, "getLastChildBlock", data); err != nil {
		return 0, nil
	}
	return (*ret0).Uint64(), nil
}

func (tc *Client) BroadcastTransaction(ctx context.Context, trx *pb.Transaction) (string, error) {
	result, err := tc.client.BroadcastTransaction(ctx, trx)
	if err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}
	if result.Code != pb.Return_SUCCESS {
		return "", fmt.Errorf("code:%v message:%v", result.Code, string(result.Message))
	}
	return hex.EncodeToString(result.Message), nil
}
