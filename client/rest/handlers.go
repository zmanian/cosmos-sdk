package rest

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	sdk "github.com/cosmos/cosmos-sdk"
	"github.com/tendermint/go-crypto"
	keysutils "github.com/tendermint/go-crypto/cmd"
	"github.com/tendermint/go-crypto/hd"
	"github.com/tendermint/go-crypto/keys"
	"github.com/tendermint/go-crypto/keys/cryptostore"
	"github.com/tendermint/tmlibs/common"
	"github.com/tyler-smith/go-bip39"
)

type Keys struct {
	algo    string
	manager keys.Manager
}

func DefaultKeysManager() keys.Manager {
	return keysutils.GetKeyManager()
}

func NewDefaultKeysManager(algo string) *Keys {
	return New(DefaultKeysManager(), algo)
}

func New(manager keys.Manager, algo string) *Keys {
	return &Keys{
		algo:    algo,
		manager: manager,
	}
}

func (k *Keys) GenerateKey(w http.ResponseWriter, r *http.Request) {
	ckReq := &CreateKeyRequest{
		Algo: k.algo,
	}
	if err := common.ParseRequestAndValidateJSON(r, ckReq); err != nil {
		common.WriteError(w, err)
		return
	}

	key, seed, err := k.manager.Create(ckReq.Name, ckReq.Passphrase, ckReq.Algo)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	res := &CreateKeyResponse{Key: key, Seed: seed}
	common.WriteSuccess(w, res)
}

func (k *Keys) GetKey(w http.ResponseWriter, r *http.Request) {
	query := mux.Vars(r)
	name := query["name"]
	key, err := k.manager.Get(name)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteSuccess(w, &key)
}

func (k *Keys) ListKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := k.manager.List()
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteSuccess(w, keys)
}

var (
	errNonMatchingPathAndJSONKeyNames = errors.New("path and json key names don't match")
)

func (k *Keys) UpdateKey(w http.ResponseWriter, r *http.Request) {
	uReq := new(UpdateKeyRequest)
	if err := common.ParseRequestAndValidateJSON(r, uReq); err != nil {
		common.WriteError(w, err)
		return
	}

	query := mux.Vars(r)
	name := query["name"]
	if name != uReq.Name {
		common.WriteError(w, errNonMatchingPathAndJSONKeyNames)
		return
	}

	if err := k.manager.Update(uReq.Name, uReq.OldPass, uReq.NewPass); err != nil {
		common.WriteError(w, err)
		return
	}

	key, err := k.manager.Get(uReq.Name)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteSuccess(w, &key)
}

func (k Keys) RecoverKey(w http.ResponseWriter, r *http.Request) {
	rReq := new(RecoverKeyRequest)
	if err := common.ParseRequestAndValidateJSON(r, rReq); err != nil {
		common.WriteError(w, err)
		return
	}

	query := mux.Vars(r)
	name := query["name"]
	if name != rReq.Name {
		common.WriteError(w, errNonMatchingPathAndJSONKeyNames)
		return
	}

	key, err := k.manager.Recover(rReq.Name, rReq.Passphrase, rReq.Seed)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteSuccess(w, &key)
}

func (k *Keys) DeleteKey(w http.ResponseWriter, r *http.Request) {
	dReq := new(DeleteKeyRequest)
	if err := common.ParseRequestAndValidateJSON(r, dReq); err != nil {
		common.WriteError(w, err)
		return
	}

	query := mux.Vars(r)
	name := query["name"]
	if name != dReq.Name {
		common.WriteError(w, errNonMatchingPathAndJSONKeyNames)
		return
	}

	if err := k.manager.Delete(dReq.Name, dReq.Passphrase); err != nil {
		common.WriteError(w, err)
		return
	}

	resp := &common.ErrorResponse{Success: true}
	common.WriteSuccess(w, resp)
}

func (k *Keys) DeriveHD(w http.ResponseWriter, r *http.Request) {
	dr := new(DeriveHDRequest)
	if err := common.ParseRequestAndValidateJSON(r, dr); err != nil {
		common.WriteError(w, err)
		return
	}

	seed := bip39.NewSeed(dr.Seed, "")

	_, secretHex, chainHex, _ := hd.ComputeMastersFromSeed(string(seed))

	keyBytes := hd.DerivePrivateKeyForPath(
		hd.HexDecode(secretHex),
		hd.HexDecode(chainHex),
		dr.Path,
	)
	// prepend type byte so the key can be unmarshalled by go-crypto
	// (HD keys follow the Bitcoin spec, so they are always secp256k1)
	keyBytes = append([]byte{crypto.TypeSecp256k1}, keyBytes...)
	key, err := crypto.PrivKeyFromBytes(keyBytes)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	// store in key manager
	cs, ok := k.manager.(cryptostore.Manager)
	if !ok {
		common.WriteError(w, errors.New("Invalid key manager"))
		return
	}
	err = cs.Import(dr.Name, dr.Password, "", keyBytes)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteSuccess(w, &key)
}

func doPostTx(w http.ResponseWriter, r *http.Request) {
	tx := new(sdk.Tx)
	if err := common.ParseRequestAndValidateJSON(r, tx); err != nil {
		common.WriteError(w, err)
		return
	}
	commit, err := PostTx(*tx)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteSuccess(w, commit)
}

func doSign(w http.ResponseWriter, r *http.Request) {
	sr := new(SignRequest)
	if err := common.ParseRequestAndValidateJSON(r, sr); err != nil {
		common.WriteError(w, err)
		return
	}

	tx := sr.Tx
	if err := SignTx(sr.Name, sr.Password, tx); err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteSuccess(w, tx)
}

// mux.Router registrars

// RegisterPostTx is a mux.Router handler that exposes POST
// method access to post a transaction to the blockchain.
func RegisterPostTx(r *mux.Router) error {
	r.HandleFunc("/tx", doPostTx).Methods("POST")
	return nil
}

// RegisterAllCRUD is a convenience method to register all
// CRUD for keys to allow access by methods and routes:
// POST:      /keys
// GET:	      /keys
// GET:	      /keys/{name}
// PUT:       /keys/{name}
// POST:      /keys/{name}
// DELETE:    /keys/{name}
func (k *Keys) RegisterAllCRUD(r *mux.Router) error {
	r.HandleFunc("/keys", k.GenerateKey).Methods("POST")
	r.HandleFunc("/keys", k.ListKeys).Methods("GET")
	r.HandleFunc("/keys/{name}", k.GetKey).Methods("GET")
	r.HandleFunc("/keys/{name}", k.UpdateKey).Methods("PUT")
	r.HandleFunc("/keys/{name}", k.RecoverKey).Methods("POST")
	r.HandleFunc("/keys/{name}", k.DeleteKey).Methods("DELETE")
	r.HandleFunc("/hdkeys", k.DeriveHD).Methods("POST", "PUT")

	return nil
}

// RegisterSignTx is a mux.Router handler that
// exposes POST method access to sign a transaction.
func RegisterSignTx(r *mux.Router) error {
	r.HandleFunc("/sign", doSign).Methods("POST")
	return nil
}

// End of mux.Router registrars
