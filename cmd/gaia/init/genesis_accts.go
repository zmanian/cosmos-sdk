package init

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"

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
			extractEthereum()
			genFile := config.GenesisFile()
			if !common.FileExists(genFile) {
				return fmt.Errorf("%s does not exist, run `gaiad init` first", genFile)
			}
			genDoc, err := loadGenesisDoc(cdc, genFile)
			if err != nil {
				return err
			}

			var appState app.GenesisState
			if err = cdc.UnmarshalJSON(genDoc.AppState, &appState); err != nil {
				return err
			}

			// appStateJSON, err := addGenesisAccount(cdc, appState, addr, coins)
			// if err != nil {
			// 	return err
			// }

			return ExportGenesisFile(genFile, genDoc.ChainID, nil, genDoc.AppState)
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

func extractEthereum() (donors map[string]big.Int) {

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

		donors[donor] = *res
	}
	return donors
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

			var appState app.GenesisState
			if err = cdc.UnmarshalJSON(genDoc.AppState, &appState); err != nil {
				return err
			}

			appStateJSON, err := addGenesisAccount(cdc, appState, addr, coins)
			if err != nil {
				return err
			}

			return ExportGenesisFile(genFile, genDoc.ChainID, nil, appStateJSON)
		},
	}

	cmd.Flags().String(cli.HomeFlag, app.DefaultNodeHome, "node's home directory")
	return cmd
}

func addGenesisAccount(cdc *codec.Codec, appState app.GenesisState, addr sdk.AccAddress, coins sdk.Coins) (json.RawMessage, error) {
	for _, stateAcc := range appState.Accounts {
		if stateAcc.Address.Equals(addr) {
			return nil, fmt.Errorf("the application state already contains account %v", addr)
		}
	}

	acc := auth.NewBaseAccountWithAddress(addr)
	acc.Coins = coins
	appState.Accounts = append(appState.Accounts, app.NewGenesisAccount(&acc))
	return cdc.MarshalJSON(appState)
}
