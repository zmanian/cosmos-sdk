package bindings

import (
	"encoding/hex"

	"github.com/tendermint/basecoin/client/commands"
	"github.com/tendermint/basecoin/modules/coin"
	"github.com/tendermint/basecoin/stack"
  rpcclient "github.com/tendermint/tendermint/rpc/client"
  "github.com/tendermint/light-client/certifiers"
  "github.com/tendermint/light-client/certifiers/client"
  "github.com/tendermint/light-client/proofs"
  wire "github.com/tendermint/go-wire"
  lc "github.com/tendermint/light-client"
)

type ExportCoin struct {
	Denom  string `json:"denom"`
	Amount int    `json:"amount"`
}

type ExportAccount struct {
	Coins []ExportCoin
}

type AccountResult struct {
	Height  int
	Key     string
	Account ExportAccount
	Error   string
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

func GetAccount(hexAddr, url string) AccountResult {
	act, err := commands.ParseActor(hexAddr)
	if err != nil {
		return AccountResult{Error: err.Error()}
	}
	key := stack.PrefixedKey(coin.NameCoin, act.Bytes())

  return getHardcodedResult(key, url)

 //  res, err := getAppProof(key, url)
	// if err != nil {
	//   return AccountResult{Error: err.Error()}
	// }
 //  return res
}

func getHardcodedResult(key []byte, url string) AccountResult {
  return AccountResult{
    Height: 50,
    Key:    hex.EncodeToString(key),
    Account: ExportAccount{
      Coins: []ExportCoin{{
        Denom:  "atom",
        Amount: 420,
      }},
    },
  }
}

func getAppProof(key []byte, url string) (acct AccountResult, err error) {
  node := rpcclient.NewHTTP(url, "/websocket")
  prover := proofs.NewAppProver(node)

  var proof lc.Proof
  proof, err = prover.Get(key, 0)
  if err != nil {
    return
  }

  var cert *certifiers.InquiringCertifier
  cert, err = getCertifier(node)
  if err != nil {
    return
  }

  err = validateProof(proof, node, cert)
  if err != nil {
    return
  }

  var data coin.Account
  err = wire.ReadBinaryBytes(proof.Data(), &data)
  if err != nil {
    return
  }

  acct = AccountResult{
    Height: int(proof.BlockHeight()),
    Account: convertAccount(data),
  }
  return
}

func getCertifier(node *rpcclient.HTTP) (*certifiers.InquiringCertifier, error) {
  // here is the certifier, root of all knowledge
  trust := certifiers.NewCacheProvider(
    certifiers.NewMemStoreProvider(),
  )
  source := client.New(node)

  // get some data!
  seed, err := source.GetByHeight(0)
  if err != nil {
    return nil, err
  }

  chainID := seed.Checkpoint.Header.ChainID
  cert := certifiers.NewInquiring(
    chainID,
    seed.Validators,
    trust,
    source,
  )
  return cert, nil
}

func validateProof(proof lc.Proof, node *rpcclient.HTTP,
  cert *certifiers.InquiringCertifier) (err error) {

  // get and validate a signed header for this proof
  ph := int(proof.BlockHeight())

  // FIXME: cannot use cert.GetByHeight for now, as it also requires
  // Validators and will fail on querying tendermint for non-current height.
  // When this is supported, we should use it instead...
  rpcclient.WaitForHeight(node, ph, nil)
  commit, err := node.Commit(ph)
  if err != nil {
    return
  }
  check := lc.Checkpoint{
    Header: commit.Header,
    Commit: commit.Commit,
  }
  err = cert.Certify(check)
  if err != nil {
    return
  }

  // validate the proof against the certified header to ensure data integrity
  err = proof.Validate(check)
  return err
}
