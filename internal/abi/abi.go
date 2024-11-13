package abi

import (
	"encoding/json"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type MethodType string

const (
	ReadMethod  MethodType = "Read"
	WriteMethod MethodType = "Write"
	AllMethod   MethodType = "All"
)

func readABIFile(filepath string) (abi.ABI, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return abi.ABI{}, err
	}
	var contractABI abi.ABI
	err = json.Unmarshal(file, &contractABI)
	if err != nil {
		return abi.ABI{}, err
	}
	return contractABI, nil
}

// LoadABIs reads all abi files in the abiDir and returns a map of contract name to ABI
func LoadABIs(abiDir string) (map[string]abi.ABI, error) {
	// read all abi files in ABI_DIR
	files, err := os.ReadDir(abiDir)
	if err != nil {
		return nil, err
	}

	mAbi := make(map[string]abi.ABI)
	for _, f := range files {
		abiFilepath := abiDir + "/" + f.Name()
		contractABI, err := readABIFile(abiFilepath)
		if err != nil {
			return nil, err
		}
		mAbi[f.Name()] = contractABI
	}
	return mAbi, nil
}

func GetMethodsByType(contractABI abi.ABI, rw MethodType) map[string]abi.Method {
	readMethods := make(map[string]abi.Method)
	writeMethods := make(map[string]abi.Method)
	allMethods := make(map[string]abi.Method)

	for name, method := range contractABI.Methods {
		if method.IsConstant() {
			readMethods[name] = method
		} else {
			writeMethods[name] = method
		}
		allMethods[name] = method
	}

	switch rw {
	case ReadMethod:
		return readMethods
	case WriteMethod:
		return writeMethods
	case AllMethod:
		return allMethods
	default:
		return allMethods
	}
}

func ParseArrayOrSliceInput(input string, typ abi.Type) interface{} {
	// parse the input string for array or slice type
	// example input format: "[1,2,3]" or "1,2,3"
	// 1. trim whitespace
	input = strings.ReplaceAll(input, " ", "")
	// 2. remove brackets if present
	input = strings.Trim(input, "[]")
	// 3. split the input string by comma
	parts := strings.Split(input, ",")
	var addresses []common.Address
	var integers []*big.Int
	var booleans []bool
	var strings []string
	var bytes [][]byte
	var hashes []common.Hash
	for _, part := range parts {
		switch typ.Elem.T {
		case abi.IntTy, abi.UintTy:
			value, _ := new(big.Int).SetString(part, 10)
			integers = append(integers, value)
		case abi.BoolTy:
			booleans = append(booleans, part == "true")
		case abi.StringTy:
			strings = append(strings, part)
		case abi.AddressTy:
			v := common.HexToAddress(part)
			addresses = append(addresses, v)
		case abi.FixedBytesTy, abi.BytesTy:
			bytes = append(bytes, common.Hex2Bytes(part))
		case abi.HashTy:
			hashes = append(hashes, common.HexToHash(part))
		default:
			panic("unsupported array or slice element type: " + typ.Elem.String())
		}
	}
	if len(addresses) > 0 {
		return addresses
	} else if len(integers) > 0 {
		return integers
	} else if len(booleans) > 0 {
		return booleans
	} else if len(strings) > 0 {
		return strings
	} else if len(bytes) > 0 {
		return bytes
	} else if len(hashes) > 0 {
		return hashes
	}
	return nil
}

func ParseTupleInput(input string, typ abi.Type) interface{} {
	// parse the input string for tuple type
	// example input format: "(1,0xabc,true)"
	input = strings.Trim(input, "()")
	parts := strings.Split(input, ",")

	if len(parts) != len(typ.TupleElems) {
		panic("tuple input length mismatch")
	}

	values := make([]interface{}, len(parts))
	for i, part := range parts {
		elemType := typ.TupleElems[i]
		var value interface{}
		switch elemType.T {
		case abi.IntTy, abi.UintTy:
			value, _ = new(big.Int).SetString(part, 10)
		case abi.BoolTy:
			value = part == "true"
		case abi.StringTy:
			value = part
		case abi.AddressTy:
			value = common.HexToAddress(part)
		case abi.FixedBytesTy, abi.BytesTy:
			value = common.Hex2Bytes(part)
		case abi.HashTy:
			value = common.HexToHash(part)
		default:
			panic("unsupported tuple element type: " + elemType.String())
		}
		values[i] = value
	}
	return values
}
