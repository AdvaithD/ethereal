// Copyright © 2022 Weald Technology Trading
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

package conn

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/params"
	"github.com/spf13/viper"
	"github.com/wealdtech/go-string2eth"
)

// CurrentBaseFee returns the current base fee of the chain.
func (c *Conn) CurrentBaseFee(ctx context.Context) (*big.Int, error) {
	if c.client == nil {
		if c.baseFeePerGas != nil {
			return c.baseFeePerGas, nil
		}
		var err error
		c.baseFeePerGas, err = string2eth.StringToWei(viper.GetString("base-fee-per-gas"))
		if err != nil {
			return nil, err
		}
		return c.baseFeePerGas, nil
	}

	blockNum, err := c.client.BlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	block, err := c.client.BlockByNumber(context.Background(), big.NewInt(int64(blockNum)))
	if err != nil {
		return nil, err
	}

	if c.baseFeePerGas != nil {
		return c.baseFeePerGas, nil
	}

	baseFee := misc.CalcBaseFee(&params.ChainConfig{
		LondonBlock: big.NewInt(0),
	}, block.Header())
	return baseFee, nil
}
