// Copyright © 2017 Weald Technology Trading
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unsafe"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/wealdtech/ethereal/cli"
	"github.com/wealdtech/ethereal/util"
)

var contractStr string
var contractAbi string
var contractJSON string
var contractName string

// contractCmd represents the contract command
var contractCmd = &cobra.Command{
	Use:   "contract",
	Short: "Manage contracts",
	Long:  `Call contracts directly.`,
}

func init() {
	RootCmd.AddCommand(contractCmd)
}

func contractFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&contractStr, "contract", "", "address of the contract")
	cmd.Flags().StringVar(&contractAbi, "abi", "", "ABI, or path to ABI, for the contract")
	cmd.Flags().StringVar(&contractJSON, "json", "", "JSON, or path to JSON, for the contract as output by solc --combined-json=bin,abi")
	cmd.Flags().StringVar(&contractName, "name", "", "Name of the contract (required when using json)")
}

// parse contract given the information from various flags
func parseContract(binStr string) *util.Contract {
	var contract *util.Contract
	if contractJSON != "" {
		if contractName == "" {
			// Attempt to obtain the contract name from the JSON file
			contractName = strings.Split(filepath.Base(contractJSON), ".")[0]
		}
		contract, err = util.ParseCombinedJSON(contractJSON, contractName)
		cli.ErrCheck(err, quiet, "Failed to parse JSON")
	} else {
		contract = &util.Contract{}

		// Add name if present
		if contractName != "" {
			contract.Name = contractName
		}

		// Add binary if present
		var bin []byte
		bin, err = hex.DecodeString(strings.TrimPrefix(binStr, "0x"))
		cli.ErrCheck(err, quiet, "Failed to decode data")
		contract = &util.Contract{Binary: bin}

		// Add ABI if present
		if contractAbi != "" {
			abi, err := contractParseAbi(contractAbi)
			cli.ErrCheck(err, quiet, fmt.Sprintf("Failed to parse ABI %s", contractAbi))
			contract.Abi = abi
		}
	}
	return contract
}

func contractParseAbi(input string) (output abi.ABI, err error) {
	var reader io.Reader

	if strings.HasPrefix(contractAbi, "[") {
		// ABI is direct
		reader = strings.NewReader(input)
	} else {
		// ABI value is a path
		reader, err = os.Open(input)
		if err != nil {
			return
		}
	}
	return abi.JSON(reader)
}

func contractUnpack(abi abi.ABI, name string, data []byte) (result *[]*interface{}, err error) {
	method, exists := abi.Methods[name]
	if !exists {
		return nil, fmt.Errorf("The method %s does not exist", name)
	}

	var res []*interface{}
	result = &res
	if len(method.Outputs) == 0 {
		return
	} else if len(method.Outputs) == 1 {
		output := reflect.New(method.Outputs[0].Type.Type).Elem().Interface()
		err = abi.Unpack(&output, name, data)
		res = append(res, &output)
	} else {
		for i := range method.Outputs {
			output := reflect.New(method.Outputs[i].Type.Type.Elem()).Elem().Interface()
			res = append(res, &output)
		}
		err = abi.Unpack(&res, name, data)
	}
	return
}

func contractStringToValue(argType abi.Type, val string) (interface{}, error) {
	val = strings.Trim(val, " ")
	switch argType.T {
	case abi.IntTy:
		res := big.NewInt(0)
		res, success := res.SetString(val, 10)
		if !success {
			return nil, fmt.Errorf("Bad integer %s", val)
		}
		switch argType.Size {
		case 8:
			return int8(res.Uint64()), nil
		case 16:
			return int16(res.Uint64()), nil
		case 32:
			return int32(res.Uint64()), nil
		case 64:
			return int64(res.Uint64()), nil
		default:
			return res, nil
		}
	case abi.UintTy:
		res := big.NewInt(0)
		res, success := res.SetString(val, 10)
		if !success {
			return nil, fmt.Errorf("Bad integer %s", val)
		}
		switch argType.Size {
		case 8:
			return uint8(res.Uint64()), nil
		case 16:
			return uint16(res.Uint64()), nil
		case 32:
			return uint32(res.Uint64()), nil
		case 64:
			return uint64(res.Uint64()), nil
		default:
			return res, nil
		}
	case abi.BoolTy:
		if val == "true" || val == "True" || val == "1" {
			return true, nil
		}
		return false, nil
	case abi.StringTy:
		return val, nil
	case abi.SliceTy:
		return nil, fmt.Errorf("Unhandled type slice (%s)", argType)
	case abi.ArrayTy:
		return nil, fmt.Errorf("Unhandled type array (%s)", argType)
	case abi.AddressTy:
		return common.HexToAddress(val), nil
	case abi.FixedBytesTy:
		slice := make([]byte, argType.Size)
		var decoded []byte
		decoded, err = hex.DecodeString(strings.TrimPrefix(val, "0x"))
		if err == nil {
			copy(slice[argType.Size-len(decoded):argType.Size], decoded)
		}
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
		switch argType.Size {
		case 1:
			return *(*[1]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 2:
			return *(*[2]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 3:
			return *(*[3]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 4:
			return *(*[4]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 5:
			return *(*[5]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 6:
			return *(*[6]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 7:
			return *(*[7]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 8:
			return *(*[8]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 9:
			return *(*[9]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 10:
			return *(*[10]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 11:
			return *(*[11]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 12:
			return *(*[12]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 13:
			return *(*[13]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 14:
			return *(*[14]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 15:
			return *(*[15]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 16:
			return *(*[16]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 17:
			return *(*[17]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 18:
			return *(*[18]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 19:
			return *(*[19]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 20:
			return *(*[20]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 21:
			return *(*[21]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 22:
			return *(*[22]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 23:
			return *(*[23]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 24:
			return *(*[24]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 25:
			return *(*[25]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 26:
			return *(*[26]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 27:
			return *(*[27]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 28:
			return *(*[28]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 29:
			return *(*[29]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 30:
			return *(*[30]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 31:
			return *(*[31]uint8)(unsafe.Pointer(hdr.Data)), nil
		case 32:
			return *(*[32]uint8)(unsafe.Pointer(hdr.Data)), nil
		}
		return nil, fmt.Errorf("Invalid byte size %d", argType.Size)
	case abi.BytesTy:
		return hex.DecodeString(strings.TrimPrefix(val, "0x"))
	case abi.HashTy:
		return common.HexToHash(val), nil
	case abi.FixedPointTy:
		return nil, fmt.Errorf("Unhandled type %v", argType)
	case abi.FunctionTy:
		return nil, fmt.Errorf("Unhandled type %v", argType)
	default:
		return nil, fmt.Errorf("Unknown type %v", argType)
	}
}

func contractValueToString(argType abi.Type, val interface{}) (string, error) {
	switch argType.T {
	case abi.IntTy:
		return fmt.Sprintf("%v", val), nil
	case abi.UintTy:
		return fmt.Sprintf("%v", val), nil
	case abi.BoolTy:
		if val.(bool) == true {
			return "true", nil
		}
		return "false", nil
	case abi.StringTy:
		return val.(string), nil
	case abi.SliceTy:
		res := make([]string, 0)
		arrayVal := reflect.ValueOf(val)
		for i := 0; i < arrayVal.Len(); i++ {
			elemRes, err := contractValueToString(*argType.Elem, arrayVal.Index(i).Interface())
			if err != nil {
				return "", err
			}
			res = append(res, elemRes)
		}
		return "[" + strings.Join(res, ",") + "]", nil
	case abi.ArrayTy:
		res := make([]string, 0)
		arrayVal := reflect.ValueOf(val)
		for i := 0; i < arrayVal.Len(); i++ {
			elemRes, err := contractValueToString(*argType.Elem, arrayVal.Index(i).Interface())
			if err != nil {
				return "", err
			}
			res = append(res, elemRes)
		}
		return "[" + strings.Join(res, ",") + "]", nil
	case abi.AddressTy:
		return val.(common.Address).Hex(), nil
	case abi.FixedBytesTy:
		arrayVal := reflect.ValueOf(val)
		castVal := make([]byte, arrayVal.Len())
		for i := 0; i < arrayVal.Len(); i++ {
			castVal[i] = byte(arrayVal.Index(i).Uint())
		}
		return fmt.Sprintf("0x%s", hex.EncodeToString(castVal)), nil
	case abi.BytesTy:
		return fmt.Sprintf("0x%s", hex.EncodeToString(val.([]byte))), nil
	case abi.HashTy:
		return val.(common.Hash).Hex(), nil
	case abi.FixedPointTy:
		return "", fmt.Errorf("Unhandled type %v", argType)
	case abi.FunctionTy:
		return "", fmt.Errorf("Unhandled type %v", argType)
	default:
		return "", fmt.Errorf("Unknown type %v", argType)
	}
}
