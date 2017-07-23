package bindings

import (
  "github.com/tendermint/basecoin/client/commands"
  proofcmd "github.com/tendermint/basecoin/client/commands/proofs"
  "github.com/tendermint/basecoin/modules/coin"
  "github.com/tendermint/basecoin/stack"
)

type AccountResult struct {
  Height int
  Account coin.Account
  Error string
}

func GetAccount(hexAddr string) AccountResult {
  act, err := commands.ParseActor(hexAddr)
  if err != nil {
    return AccountResult{Error: err.Error()}
  }
  key := stack.PrefixedKey(coin.NameCoin, act.Bytes())

  acct := coin.Account{}
  proof, err := proofcmd.GetAndParseAppProof(key, &acct)
  if err != nil {
    return AccountResult{Error: err.Error()}
  }

  res := AccountResult{
    Height: int(proof.BlockHeight()),
    Account: acct,
  }
  return res
}
