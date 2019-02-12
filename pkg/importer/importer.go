package importer

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type State interface {
	AcceptAndVerifyBlockBinary(block []byte, initialisation bool) error
	GetBlockByHeight(height uint64) (*proto.Block, error)
	WavesAddressesNumber() (uint64, error)
	AccountBalance(addr proto.Address, asset []byte) (uint64, error)
}

func ApplyFromFile(st State, blockchainPath string, nBlocks int, checkBlocks bool) error {
	blockchain, err := os.Open(blockchainPath)
	if err != nil {
		return errors.Errorf("failed to open blockchain file: %v\n", err)
	}
	sb := make([]byte, 4)
	buf := make([]byte, 2*1024*1024)
	r := bufio.NewReader(blockchain)
	for i := 0; i < nBlocks; i++ {
		if _, err := io.ReadFull(r, sb); err != nil {
			return err
		}
		size := binary.BigEndian.Uint32(sb)
		block := buf[:size]
		if _, err := io.ReadFull(r, block); err != nil {
			return err
		}
		if err := st.AcceptAndVerifyBlockBinary(block, true); err != nil {
			return err
		}
		if checkBlocks {
			savedBlock, err := st.GetBlockByHeight(uint64(i))
			if err != nil {
				return err
			}
			savedBlockBytes, err := savedBlock.MarshalBinary()
			if err != nil {
				return err
			}
			if bytes.Compare(block, savedBlockBytes) != 0 {
				return errors.New("accepted and returned blocks differ\n")
			}
		}
	}
	if err := blockchain.Close(); err != nil {
		return errors.Errorf("failed to close blockchain file: %v\n", err)
	}
	return nil
}

func CheckBalances(st State, balancesPath string) error {
	balances, err := os.Open(balancesPath)
	if err != nil {
		return errors.Errorf("failed to open balances file: %v\n", err)
	}
	var state map[string]uint64
	jsonParser := json.NewDecoder(balances)
	if err := jsonParser.Decode(&state); err != nil {
		return errors.Errorf("failed to decode state: %v\n", err)
	}
	addressesNumber, err := st.WavesAddressesNumber()
	if err != nil {
		return errors.Errorf("failed to get number of waves addresses: %v\n", err)
	}
	properAddressesNumber := uint64(len(state))
	if properAddressesNumber != addressesNumber {
		return errors.Errorf("number of addresses differ: %d and %d\n", properAddressesNumber, addressesNumber)
	}
	for addrStr, properBalance := range state {
		addr, err := proto.NewAddressFromString(addrStr)
		if err != nil {
			return errors.Errorf("faied to convert string to address: %v\n", err)
		}
		balance, err := st.AccountBalance(addr, nil)
		if err != nil {
			return errors.Errorf("failed to get balance: %v\n", err)
		}
		if balance != properBalance {
			return errors.Errorf("balances for address %v differ: %d and %d\n", addr, properBalance, balance)
		}
	}
	if err := balances.Close(); err != nil {
		return errors.Errorf("failed to close balances file: %v\n", err)
	}
	return nil
}
