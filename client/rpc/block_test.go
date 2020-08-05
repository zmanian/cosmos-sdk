package rpc

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

func Test_serializeBlock(t *testing.T) {

	lastID := makeBlockIDRandom()
	h := int64(3)

	voteSet, _, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, err := types.MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := types.NewMockDuplicateVoteEvidenceWithValidator(h, time.Now(), vals[0], "block-test-chain")
	evList := []types.Evidence{ev}

	resultBlock := coretypes.ResultBlock{
		makeBlockIDRandom(),
		types.MakeBlock(0, []types.Tx{}, commit, evList),
	}
	res, _ := legacy.Cdc.MarshalJSON(resultBlock)
	fmt.Println(string(res))
	t.Fail()

}

func randVoteSet(
	height int64,
	round int32,
	signedMsgType tmproto.SignedMsgType,
	numValidators int,
	votingPower int64,
) (*types.VoteSet, *types.ValidatorSet, []types.PrivValidator) {
	valSet, privValidators := types.RandValidatorSet(numValidators, votingPower)
	return types.NewVoteSet("test_chain_id", height, round, signedMsgType, valSet), valSet, privValidators
}

func makeBlockIDRandom() types.BlockID {
	var (
		blockHash   = make([]byte, tmhash.Size)
		partSetHash = make([]byte, tmhash.Size)
	)
	rand.Read(blockHash)   //nolint: gosec
	rand.Read(partSetHash) //nolint: gosec
	return types.BlockID{blockHash, types.PartSetHeader{123, partSetHash}}
}
