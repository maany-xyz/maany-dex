package sumtree_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/iavl"

	dbm "github.com/cosmos/cosmos-db"

	iavlstore "cosmossdk.io/store/iavl"

	"github.com/neutron-org/neutron/v5/osmomath"
	"github.com/neutron-org/neutron/v5/osmoutils/sumtree"
	"github.com/neutron-org/neutron/v5/osmoutils/wrapper"
)

type TreeTestSuite struct {
	suite.Suite

	tree sumtree.Tree
}

func (suite *TreeTestSuite) SetupTest() {
	db := wrapper.NewIAVLDB(dbm.NewMemDB())
	tree := iavl.NewMutableTree(db, 100, false, log.NewNopLogger())
	_, _, err := tree.SaveVersion()
	suite.Require().Nil(err)
	kvstore := iavlstore.UnsafeNewStore(tree)
	suite.tree = sumtree.NewTree(kvstore, 10)
}

func TestTreeTestSuite(t *testing.T) {
	suite.Run(t, new(TreeTestSuite))
}

type pair struct {
	key   []byte
	value uint64
}

type pairs []pair

var _ sort.Interface = pairs{}

func (p pairs) Len() int {
	return len(p)
}

func (p pairs) Less(i, j int) bool {
	return bytes.Compare(p[i].key, p[j].key) < 0
}

func (p pairs) Swap(i, j int) {
	temp := p[i]
	p[i] = p[j]
	p[j] = temp
}

func (p pairs) sum() (res uint64) {
	for _, pair := range p {
		res += pair.value
	}
	return
}

func (suite *TreeTestSuite) TestTreeInvariants() {
	suite.SetupTest()

	pairs := pairs{pair{[]byte("hello"), 100}}
	suite.tree.Set([]byte("hello"), osmomath.NewIntFromUint64(100))

	seed := rand.Int63()
	fmt.Printf("running seed %d: \n", seed)
	r := rand.New(rand.NewSource(seed))

	// tested up to 2000
	for i := 0; i < 250; i++ {
		// add a single element
		key := make([]byte, r.Int()%20)
		value := r.Uint64() % 100
		r.Read(key)
		idx := sort.Search(len(pairs), func(n int) bool { return bytes.Compare(pairs[n].key, key) >= 0 })
		if idx < len(pairs) {
			if bytes.Equal(pairs[idx].key, key) {
				pairs[idx] = pair{key, value}
			} else {
				pairs = append(pairs, pair{key, value})
				sort.Sort(pairs)
			}
		} else {
			pairs = append(pairs, pair{key, value})
		}

		suite.tree.Set(key, osmomath.NewIntFromUint64(value))

		// check all is right
		for _, pair := range pairs {
			suite.Require().Equal(suite.tree.Get(pair.key).Uint64(), pair.value)
			// XXX: check all branch nodes
		}

		// check accumulation calc is alright
		left, exact, right := uint64(0), pairs[0].value, pairs[1:].sum()
		for idx, pair := range pairs {
			tleft, texact, tright := suite.tree.SplitAcc(pair.key)
			suite.Require().Equal(left, tleft.Uint64())
			suite.Require().Equal(exact, texact.Uint64())
			suite.Require().Equal(right, tright.Uint64())

			key := append(pair.key, 0x00)
			if idx == len(pairs)-1 {
				break
			}
			if bytes.Equal(key, pairs[idx+1].key) {
				break
			}

			tleft, texact, tright = suite.tree.SplitAcc(key)
			suite.Require().Equal(left+exact, tleft.Uint64())
			suite.Require().Equal(uint64(0), texact.Uint64())
			suite.Require().Equal(right, tright.Uint64())

			left += exact
			exact = pairs[idx+1].value
			right -= exact
		}

		if r.Int()%2 == 0 {
			idx := r.Int() % len(pairs)
			pair := pairs[idx]
			pairs = append(pairs[:idx], pairs[idx+1:]...)
			suite.tree.Remove(pair.key)
		}
	}
}
