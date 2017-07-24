package bindings

import (
	"encoding/hex"

	"github.com/tendermint/basecoin/client/commands"
	"github.com/tendermint/basecoin/modules/coin"
	"github.com/tendermint/basecoin/stack"
)

// Coin is an exportable version of coin.Coin (no int64)
type MyCoin struct {
	Denom  string
	Amount int
}

// AccountResult captures all account info
// If error, len(Error) > 0 and Height == 0
type AccountResult struct {
	Height  int
	Key     string
	account coin.Account
	Error   string
}

// NumCoins returns the total number of coins in the account to loop through
func (a AccountResult) NumCoins() int {
	return len(a.account.Coins)
}

// Coin allows you to get a coin by index
// together with NumCoins, it allows us to avoid exposing a slice to
// the C ABI boundary
func (a *AccountResult) MyCoin(i int) *MyCoin {
	return convertCoin(a.account.Coins[i])
}

func convertCoin(c coin.Coin) *MyCoin {
	return &MyCoin{
		Denom:  c.Denom,
		Amount: int(c.Amount),
	}
}

func GetCoin() *MyCoin {
	return &MyCoin{"demo", 123}
}

// GetAccount provides a binding to call from C
func GetAccount(hexAddr, url string) *AccountResult {
	act, err := commands.ParseActor(hexAddr)
	if err != nil {
		return &AccountResult{Error: err.Error()}
	}
	key := stack.PrefixedKey(coin.NameCoin, act.Bytes())

	return getHardcodedResult(key, url)

	//  res, err := getAppProof(key, url)
	// if err != nil {
	//   return AccountResult{Error: err.Error()}
	// }
	//  return res
}

func getHardcodedResult(key []byte, url string) *AccountResult {
	return &AccountResult{
		Height: 50,
		Key:    hex.EncodeToString(key),
		account: coin.Account{
			Coins: []coin.Coin{{
				Denom:  "atom",
				Amount: 420,
			}},
		},
	}
}

/*
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

	acct = &AccountResult{
		Height:  int(proof.BlockHeight()),
		Key:     hex.EncodeToString(key),
		account: data,
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
*/
