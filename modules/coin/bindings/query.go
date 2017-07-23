package main

import (
  "encoding/hex"

  "github.com/tendermint/basecoin/client/commands"
  // proofcmd "github.com/tendermint/basecoin/client/commands/proofs"
  "github.com/tendermint/basecoin/modules/coin"
  "github.com/tendermint/basecoin/stack"
)

func main() {}

type ExportCoin struct {
  Denom  string `json:"denom"`
  Amount int  `json:"amount"`
}

type ExportAccount struct {
  Coins []ExportCoin
}

type AccountResult struct {
  Height int
  Key string
  Account ExportAccount
  Error string
}

func convertAccount(acct coin.Account) ExportAccount {
  return ExportAccount{
    Coins: convertCoins(acct.Coins),
  }
}

func convertCoins(coins []coin.Coin) []ExportCoin {
  res := make([]ExportCoin, len(coins))
  for i, c := range coins {
    res[i] = ExportCoin{
      Denom: c.Denom,
      Amount: int(c.Amount),
    }
  }
  return res
}

func GetAccount(hexAddr string) AccountResult {
  act, err := commands.ParseActor(hexAddr)
  if err != nil {
    return AccountResult{Error: err.Error()}
  }
  key := stack.PrefixedKey(coin.NameCoin, act.Bytes())

  return AccountResult{
    Height: 50,
    Key: hex.EncodeToString(key),
    Account: ExportAccount{
      Coins: []ExportCoin{{
        Denom: "atom",
        Amount: 420,
      }},
    },
  }

  // acct := coin.Account{}
  // proof, err := proofcmd.GetAndParseAppProof(key, &acct)
  // if err != nil {
  //   return AccountResult{Error: err.Error()}
  // }

  // res := AccountResult{
  //   Height: int(proof.BlockHeight()),
  //   Account: convertAccount(acct),
  // }
  // return res
}
