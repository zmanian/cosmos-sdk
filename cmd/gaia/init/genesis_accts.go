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

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
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
			donors := make(map[string]big.Int)

			//extractEthereum(donors)
			extractBitcoin(donors)

			genFile := config.GenesisFile()
			if !common.FileExists(genFile) {
				return fmt.Errorf("%s does not exist, run `gaiad init` first", genFile)
			}
			genDoc, err := loadGenesisDoc(cdc, genFile)
			if err != nil {
				return err
			}

			appState := new(app.GenesisState)

			if err = cdc.UnmarshalJSON(genDoc.AppState, appState); err != nil {
				return err
			}

			var keys []string
			sum_alloc := new(big.Int)
			for k, alloc := range donors {
				keys = append(keys, k)
				sum_alloc = new(big.Int).Add(sum_alloc, &alloc)
			}
			fmt.Printf("Total allocation: %s", sum_alloc.String())

			sort.Strings(keys)

			for _, account := range keys {
				accountBytes, err := hex.DecodeString(account)
				if err != nil {
					log.Fatalln(err)
				}
				acc := sdk.AccAddress(accountBytes)
				alloc := donors[account]
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

func extractBitcoin(donors map[string]big.Int) map[string]big.Int {

	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:8332",
		User:         "user",
		Pass:         "password",
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Shutdown()

	blockCount, err := client.GetBlockCount()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Block count: %d", blockCount)

	txs, err := client.ListTransactionsCountFromWatchOnly("*", 1000, 0)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(txs)

	heightLookup := make(map[string]int32)

	for _, tx := range txs {
		// if tx.Category != "recieve" {
		// 	continue
		// }
		txid, err := chainhash.NewHashFromStr(tx.TxID)

		if err != nil {
			log.Fatal(err)
		}

		raw, err := client.GetRawTransactionVerbose(txid)

		if err != nil {
			log.Fatal(err)
		}

		_, lookup := heightLookup[raw.BlockHash]
		if !lookup {

			blockhash, err := chainhash.NewHashFromStr(raw.BlockHash)
			if err != nil {
				log.Fatal(err)
			}

			headerOfBlock, err := client.GetBlockHeaderVerbose(blockhash)
			if err != nil {
				log.Fatal(err)
			}

			heightLookup[raw.BlockHash] = headerOfBlock.Height
		}
		height := heightLookup[raw.BlockHash]

		//Ignore transctions not during the fundraiser
		if height < 460654 || height > 460661 {
			continue
		}
		if len(raw.Vout) != 2 {
			continue
		}
		if raw.Vout[0].ScriptPubKey.Hex != "a9142e232a65af2f891ccbb16023683b8dbea8ebccef87" {
			continue
		}
		tag := raw.Vout[1].ScriptPubKey.Hex
		if len(tag) != 44 || tag[:4] != "6a14" {
			continue
		}
		balance := donors[tag[4:]]
		donation := big.NewInt(int64(11635 * raw.Vout[0].Value))

		fmt.Println(donation.String())

		donors[tag[4:]] = *new(big.Int).Add(donation, &balance)

	}

	return donors
}

func extractEthereum(donors map[string]big.Int) map[string]big.Int {

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
		amount.SetString(txdata[66:130], 16)

		rate := new(big.Int)
		rate.SetString(txdata[130:], 16)

		res := new(big.Int).Div(amount, rate)

		balance := donors[donor]

		donors[donor] = *new(big.Int).Add(res, &balance)
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

			appState := new(app.GenesisState)

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
