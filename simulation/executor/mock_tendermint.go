package simulation

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto"
	cryptoenc "github.com/cometbft/cometbft/crypto/encoding"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"golang.org/x/exp/maps"

	markov "github.com/neutron-org/neutron/v5/simulation/simtypes/transitionmatrix"
)

type mockValidator struct {
	val           abci.ValidatorUpdate
	livenessState int
}

func (mv mockValidator) String() string {
	return fmt.Sprintf("mockValidator{%s power:%v state:%v}",
		mv.val.PubKey.String(),
		mv.val.Power,
		mv.livenessState)
}

type mockValidators map[string]mockValidator

// get mockValidators from abci validators
func newMockValidators(r *rand.Rand, abciVals []abci.ValidatorUpdate, params Params) mockValidators {
	validators := make(mockValidators)

	for _, validator := range abciVals {
		str := fmt.Sprintf("%X", validator.PubKey.GetEd25519())
		liveliness := markov.GetMemberOfInitialState(r, params.InitialLivenessWeightings())

		validators[str] = mockValidator{
			val:           validator,
			livenessState: liveliness,
		}
	}

	return validators
}

func (mv mockValidators) Clone() mockValidators {
	validators := make(mockValidators, len(mv))
	keys := mv.getKeys()
	for _, k := range keys {
		validators[k] = mv[k]
	}
	return validators
}

// TODO describe usage
func (mv mockValidators) getKeys() []string {
	keys := maps.Keys(mv)
	sort.Strings(keys)
	return keys
}

// randomProposer picks a random proposer from the current validator set
func (mv mockValidators) randomProposer(r *rand.Rand) crypto.PubKey {
	keys := mv.getKeys()
	if len(keys) == 0 {
		return nil
	}

	key := keys[r.Intn(len(keys))]

	proposer := mv[key].val
	pk, err := cryptoenc.PubKeyFromProto(proposer.PubKey)
	if err != nil { //nolint:wsl
		panic(err)
	}

	return pk
}

func (mv mockValidators) toTmProtoValidators(proposerPubKey crypto.PubKey) (cmtypes.ValidatorSet, error) {
	var tmProtoValSet cmtproto.ValidatorSet
	var tmTypesValSet *cmtypes.ValidatorSet
	// iterate through current validators and add them to the TM ValidatorSet struct
	for _, key := range mv.getKeys() {
		var validator cmtproto.Validator
		mapVal := mv[key]
		validator.PubKey = mapVal.val.PubKey
		currentPubKey, err := cryptoenc.PubKeyFromProto(mapVal.val.PubKey)
		if err != nil {
			return cmtypes.ValidatorSet{}, err
		}
		validator.Address = currentPubKey.Address()
		tmProtoValSet.Validators = append(tmProtoValSet.Validators, &validator)
	}

	// set the proposer chosen earlier as the validator set block proposer
	var proposerVal cmtypes.Validator
	proposerVal.PubKey = proposerPubKey
	proposerVal.Address = proposerPubKey.Address()
	blockProposer, err := proposerVal.ToProto()
	if err != nil {
		return cmtypes.ValidatorSet{}, err
	}
	tmProtoValSet.Proposer = blockProposer

	// create a validatorSet type from the tmproto created earlier
	tmTypesValSet, err = cmtypes.ValidatorSetFromProto(&tmProtoValSet)
	return *tmTypesValSet, err
}

// updateValidators mimics Tendermint's update logic.
func updateValidators(
	r *rand.Rand,
	params simulation.Params,
	current map[string]mockValidator,
	updates []abci.ValidatorUpdate,
	event func(route, op, evResult string),
) (map[string]mockValidator, error) {
	nextSet := mockValidators(current).Clone()

	// Count the number of validators that are about to be kicked
	kickedValidators := 0

	for _, update := range updates {
		str := fmt.Sprintf("%X", update.PubKey.GetEd25519())

		if update.Power == 0 {
			if _, ok := nextSet[str]; !ok {
				return nil, fmt.Errorf("tried to delete a nonexistent validator: %s", str)
			}

			kickedValidators++

			event("end_block", "validator_updates", "kicked")
		} else if _, ok := nextSet[str]; ok {
			// validator already exists, update weight
			nextSet[str] = mockValidator{update, nextSet[str].livenessState}
			event("end_block", "validator_updates", "updated")
		} else {
			// Set this new validator
			nextSet[str] = mockValidator{
				update,
				markov.GetMemberOfInitialState(r, params.InitialLivenessWeightings()),
			}
			event("end_block", "validator_updates", "added")
		}
	}

	// Check if all the validators are about to be kicked, if so, don't perform the deletions
	if kickedValidators == len(nextSet) {
		return nextSet, nil
	}

	// Perform the deletions for validators that are to be kicked
	for _, update := range updates {
		str := fmt.Sprintf("%X", update.PubKey.GetEd25519())
		if update.Power == 0 {
			delete(nextSet, str)
		}
	}

	return nextSet, nil
}

func RandomRequestFinalizeBlock(
	r *rand.Rand,
	params Params,
	validators mockValidators,
	pastTimes []time.Time,
	pastVoteInfos [][]abci.VoteInfo,
	event func(route, op, evResult string),
	blockHeight int64,
	time time.Time,
	proposer []byte,
) *abci.RequestFinalizeBlock {
	if len(validators) == 0 {
		return &abci.RequestFinalizeBlock{
			Height:          blockHeight,
			Time:            time,
			ProposerAddress: proposer,
		}
	}

	voteInfos := make([]abci.VoteInfo, len(validators))

	for i, key := range validators.getKeys() {
		mVal := validators[key]
		mVal.livenessState = params.LivenessTransitionMatrix().NextState(r, mVal.livenessState)
		signed := true

		if mVal.livenessState == 1 {
			// spotty connection, 50% probability of success
			// See https://github.com/golang/go/issues/23804#issuecomment-365370418
			// for reasoning behind computing like this
			signed = r.Int63()%2 == 0
		} else if mVal.livenessState == 2 {
			// offline
			signed = false
		}

		if signed {
			event("begin_block", "signing", "signed")
		} else {
			event("begin_block", "signing", "missed")
		}

		pubkey, err := cryptoenc.PubKeyFromProto(mVal.val.PubKey)
		if err != nil {
			panic(err)
		}

		voteInfos[i] = abci.VoteInfo{
			Validator: abci.Validator{
				Address: pubkey.Address(),
				Power:   mVal.val.Power,
			},
			BlockIdFlag: cmtproto.BlockIDFlagCommit,
		}
	}

	// return if no past times
	if len(pastTimes) == 0 {
		return &abci.RequestFinalizeBlock{
			Height:          blockHeight,
			Time:            time,
			ProposerAddress: proposer,
			DecidedLastCommit: abci.CommitInfo{
				Votes: voteInfos,
			},
		}
	}

	// TODO: Determine capacity before allocation
	evidence := make([]abci.Misbehavior, 0)

	for r.Float64() < params.EvidenceFraction() {
		vals := voteInfos
		height := blockHeight
		misbehaviorTime := time
		if r.Float64() < params.PastEvidenceFraction() && height > 1 {
			height = int64(r.Intn(int(height)-1)) + 1 // CometBFT starts at height 1
			// array indices offset by one
			misbehaviorTime = pastTimes[height-1]
			vals = pastVoteInfos[height-1]
		}

		validator := vals[r.Intn(len(vals))].Validator

		var totalVotingPower int64
		for _, val := range vals {
			totalVotingPower += val.Validator.Power
		}

		evidence = append(evidence,
			abci.Misbehavior{
				Type:             abci.MisbehaviorType_DUPLICATE_VOTE,
				Validator:        validator,
				Height:           height,
				Time:             misbehaviorTime,
				TotalVotingPower: totalVotingPower,
			},
		)

		event("begin_block", "evidence", "ok")
	}

	return &abci.RequestFinalizeBlock{
		Height:          blockHeight,
		Time:            time,
		ProposerAddress: proposer,
		DecidedLastCommit: abci.CommitInfo{
			Votes: voteInfos,
		},
		Misbehavior: evidence,
	}
}

// func randomVoteInfos(r *rand.Rand, simParams Params, validators mockValidators,
// ) []abci.VoteInfo {
// 	voteInfos := make([]abci.VoteInfo, len(validators))

// 	for i, key := range validators.getKeys() {
// 		mVal := validators[key]
// 		mVal.livenessState = simParams.LivenessTransitionMatrix().NextState(r, mVal.livenessState)
// 		signed := true

// 		if mVal.livenessState == 1 {
// 			// spotty connection, 50% probability of success
// 			// See https://github.com/golang/go/issues/23804#issuecomment-365370418
// 			// for reasoning behind computing like this
// 			signed = r.Int63()%2 == 0
// 		} else if mVal.livenessState == 2 {
// 			// offline
// 			signed = false
// 		}

// 		// TODO: Do we want to log any data to statsdb here?

// 		pubkey, err := cryptoenc.PubKeyFromProto(mVal.val.PubKey)
// 		if err != nil {
// 			panic(err)
// 		}

// 		singedFlag := cmtproto.BlockIDFlagCommit
// 		if !signed {
// 			singedFlag = cmtproto.BlockIDFlagNil
// 		}

// 		voteInfos[i] = abci.VoteInfo{
// 			Validator: abci.Validator{
// 				Address: pubkey.Address(),
// 				Power:   mVal.val.Power,
// 			},
// 			BlockIdFlag: singedFlag,
// 		}
// 	}

// 	return voteInfos
// }

// func randomDoubleSignEvidence(r *rand.Rand, params Params, pastTimes []time.Time,
// 	pastVoteInfos [][]abci.VoteInfo,
// 	event func(route, op, evResult string), header cmtproto.Header, voteInfos []abci.VoteInfo,
// ) []abci.Misbehavior {
// 	evidence := []abci.Misbehavior{}
// 	// return if no past times or if only 10 validators remaining in the active set
// 	if len(pastTimes) == 0 {
// 		return evidence
// 	}
// 	var n float64 = 1
// 	// TODO: Change this to be markov based & clean this up
// 	// Right now we incrementally lower the evidence fraction to make
// 	// it less likely to jail many validators in one run.
// 	// We should also add some method of including new validators into the set
// 	// instead of being stuck with the ones we start with during initialization.
// 	for r.Float64() < (params.EvidenceFraction() / n) {
// 		// if only one validator remaining, don't jail any more validators
// 		if len(voteInfos)-int(n) <= 0 {
// 			return nil
// 		}
// 		height := header.Height
// 		time := header.Time
// 		vals := voteInfos

// 		if r.Float64() < params.PastEvidenceFraction() && header.Height > 1 {
// 			height = int64(r.Intn(int(header.Height)-1)) + 1 // Tendermint starts at height 1
// 			// array indices offset by one
// 			time = pastTimes[height-1]
// 			vals = pastVoteInfos[height-1]
// 		}

// 		validator := vals[r.Intn(len(vals))].Validator

// 		var totalVotingPower int64
// 		for _, val := range vals {
// 			totalVotingPower += val.Validator.Power
// 		}

// 		evidence = append(evidence,
// 			abci.Misbehavior{
// 				Type:             abci.MisbehaviorType_DUPLICATE_VOTE,
// 				Validator:        validator,
// 				Height:           height,
// 				Time:             time,
// 				TotalVotingPower: totalVotingPower,
// 			},
// 		)

// 		event("begin_block", "evidence", "ok")
// 		n++
// 	}
// 	return evidence
// }
