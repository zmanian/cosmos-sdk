package init

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/common"

	"github.com/cosmos/cosmos-sdk/cmd/gaia/app"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

// AddGenesisAccountsForFundraiserContributors
func AddContributorAccounts(ctx *server.Context, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fundraiser",
		Short: "Add fundraiser contributors accounts to genesis.json",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			config := ctx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))
			doners := make(map[string]big.Int)

			doners = extractEthereum(doners)
			doners = extractBitcoin(doners)

			genFile := config.GenesisFile()
			if !common.FileExists(genFile) {
				return fmt.Errorf("%s does not exist, run `gaiad init` first", genFile)
			}
			genDoc, err := loadGenesisDoc(cdc, genFile)
			if err != nil {
				return err
			}

			var appState *app.GenesisState
			if err = cdc.UnmarshalJSON(genDoc.AppState, appState); err != nil {
				return err
			}

			var keys []string
			for k := range doners {
				keys = append(keys, k)
			}

			sort.Strings(keys)

			for _, account := range keys {
				accountBytes, err := hex.DecodeString(account)
				if err != nil {
					log.Fatalln(err)
				}
				acc := sdk.AccAddress(accountBytes)
				alloc := doners[account]
				allocationCoin := sdk.Coin{
					Denom:  "atom",
					Amount: sdk.NewIntFromBigInt(&alloc),
				}
				appState, err = addGenesisAccount(cdc, appState, acc, sdk.Coins{allocationCoin})
				if err != nil {
					return err
				}

			}
			appStateJSON, err := cdc.MarshalJSON(appState)

			return ExportGenesisFile(genFile, genDoc.ChainID, nil, appStateJSON)
		},
	}
	cmd.Flags().String(cli.HomeFlag, app.DefaultNodeHome, "node's home directory")
	return cmd
}

type EthereumResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  []struct {
		Address          string   `json:"address"`
		BlockHash        string   `json:"blockHash"`
		BlockNumber      string   `json:"blockNumber"`
		Data             string   `json:"data"`
		LogIndex         string   `json:"logIndex"`
		Removed          bool     `json:"removed"`
		Topics           []string `json:"topics"`
		TransactionHash  string   `json:"transactionHash"`
		TransactionIndex string   `json:"transactionIndex"`
	} `json:"result"`
}

type BlockInfoResponse struct {
	Hash160       string `json:"hash160"`
	Address       string `json:"address"`
	NTx           int    `json:"n_tx"`
	TotalReceived int64  `json:"total_received"`
	TotalSent     int64  `json:"total_sent"`
	FinalBalance  int64  `json:"final_balance"`
	Txs           []struct {
		Ver    int `json:"ver"`
		Inputs []struct {
			Sequence int64  `json:"sequence"`
			Witness  string `json:"witness"`
			PrevOut  struct {
				Spent             bool `json:"spent"`
				SpendingOutpoints []struct {
					TxIndex int `json:"tx_index"`
					N       int `json:"n"`
				} `json:"spending_outpoints"`
				TxIndex int    `json:"tx_index"`
				Type    int    `json:"type"`
				Addr    string `json:"addr"`
				Value   int    `json:"value"`
				N       int    `json:"n"`
				Script  string `json:"script"`
			} `json:"prev_out"`
			Script string `json:"script"`
		} `json:"inputs"`
		Weight      int    `json:"weight"`
		BlockHeight int    `json:"block_height"`
		RelayedBy   string `json:"relayed_by"`
		Out         []struct {
			Spent             bool `json:"spent"`
			SpendingOutpoints []struct {
				TxIndex int `json:"tx_index"`
				N       int `json:"n"`
			} `json:"spending_outpoints,omitempty"`
			TxIndex int    `json:"tx_index"`
			Type    int    `json:"type"`
			Addr    string `json:"addr,omitempty"`
			Value   int    `json:"value"`
			N       int    `json:"n"`
			Script  string `json:"script"`
		} `json:"out"`
		LockTime   int    `json:"lock_time"`
		Result     int    `json:"result"`
		Size       int    `json:"size"`
		BlockIndex int    `json:"block_index"`
		Time       int    `json:"time"`
		TxIndex    int    `json:"tx_index"`
		VinSz      int    `json:"vin_sz"`
		Hash       string `json:"hash"`
		VoutSz     int    `json:"vout_sz"`
		Rbf        bool   `json:"rbf,omitempty"`
	} `json:"txs"`
}

func extractBitcoin(doners map[string]big.Int) map[string]big.Int {

	resp, err := http.Get("https://blockchain.info/rawaddr/35ty8iaSbWsj4YVkoHzs9pZMze6dapeoZ8")

	if err != nil {
		log.Fatalln(err)
	}

	var parsedResp BlockInfoResponse

	json.NewDecoder(resp.Body).Decode(&parsedResp)

	for _, tx := range parsedResp.Txs {
		//Ignore transctions not during the fundraiser
		if tx.BlockHeight < 460654 || tx.BlockHeight > 460662 {
			continue
		}
		if len(tx.Out) != 2 {
			continue
		}
		if tx.Out[0].Script != "a9142e232a65af2f891ccbb16023683b8dbea8ebccef87" {
			continue
		}
		tag := tx.Out[1].Script
		if len(tag) != 44 || tag[:4] != "6a14" {
			continue
		}
		balance := doners[tag[4:]]
		donation := big.NewInt(int64(11635 * tx.Out[0].Value))

		doners[tag[4:]] = *new(big.Int).Add(donation, &balance)

	}

	return doners
}

func extractEthereum(doners map[string]big.Int) map[string]big.Int {

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getLogs",
		"params": []map[string]interface{}{
			map[string]interface{}{
				"fromBlock": "0x352960",
				"toBlock":   "0x353CE8",
				"address":   "0xCF965Cfe7C30323E9C9E41D4E398e2167506f764",
				"topics":    []string{"0x14432f6e1dc0e8c1f4c0d81c69cecc80c0bea817a74482492b0211392478ab9b"},
			}},
		"id": 73,
	}

	bytesRepresentation, err := json.Marshal(message)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := http.Post("https://mainnet.infura.io/v3/1fa0be52251e4a1c9871ee9c854502d7", "application/json", bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		log.Fatalln(err)
	}

	var parsed_resp EthereumResponse

	json.NewDecoder(resp.Body).Decode(&parsed_resp)

	for _, tx := range parsed_resp.Result {
		txdata := tx.Data
		donor := tx.Topics[1][26:]
		amount := new(big.Int)
		_, err = fmt.Sscanf(txdata[66:130], "%x", amount)
		if err != nil {
			panic("parsing amount:" + err.Error())
		}

		rate := new(big.Int)
		_, err = fmt.Sscanf(txdata[130:], "%x", rate)
		if err != nil {
			panic("parsing rate" + err.Error())
		}
		res := new(big.Int).Div(amount, rate)

		doners[donor] = *res
	}
	return doners
}

// AddGenesisAccountCmd returns add-genesis-account cobra Command
func AddGenesisAccountCmd(ctx *server.Context, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-genesis-account [address] [coin][,[coin]]",
		Short: "Add genesis account to genesis.json",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			config := ctx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))

			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			coins, err := sdk.ParseCoins(args[1])
			if err != nil {
				return err
			}
			coins.Sort()

			genFile := config.GenesisFile()
			if !common.FileExists(genFile) {
				return fmt.Errorf("%s does not exist, run `gaiad init` first", genFile)
			}
			genDoc, err := loadGenesisDoc(cdc, genFile)
			if err != nil {
				return err
			}

			var appState *app.GenesisState
			if err = cdc.UnmarshalJSON(genDoc.AppState, appState); err != nil {
				return err
			}

			appState, err = addGenesisAccount(cdc, appState, addr, coins)
			if err != nil {
				return err
			}

			appStateJSON, err := cdc.MarshalJSON(appState)

			if err != nil {
				return err
			}

			return ExportGenesisFile(genFile, genDoc.ChainID, nil, appStateJSON)
		},
	}

	cmd.Flags().String(cli.HomeFlag, app.DefaultNodeHome, "node's home directory")
	return cmd
}

func addGenesisAccount(cdc *codec.Codec, appState *app.GenesisState, addr sdk.AccAddress, coins sdk.Coins) (*app.GenesisState, error) {
	for _, stateAcc := range appState.Accounts {
		if stateAcc.Address.Equals(addr) {
			return nil, fmt.Errorf("the application state already contains account %v", addr)
		}
	}

	acc := auth.NewBaseAccountWithAddress(addr)
	acc.Coins = coins
	appState.Accounts = append(appState.Accounts, app.NewGenesisAccount(&acc))
	return appState, nil
}
