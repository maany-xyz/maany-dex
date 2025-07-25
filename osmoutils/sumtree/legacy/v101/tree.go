package v101

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	storetypes "cosmossdk.io/store/types"
	stypes "cosmossdk.io/store/types"

	"github.com/neutron-org/neutron/v5/osmomath"
	"github.com/neutron-org/neutron/v5/osmoutils/sumtree"
)

type Child struct {
	Index []byte
	Acc   osmomath.Int
}

type Children []Child // branch nodes

func migrateBranchValue(oldValueBz []byte) *sumtree.Node {
	var oldValue Children
	fmt.Println(string(oldValueBz))
	err := json.Unmarshal(oldValueBz, &oldValue)
	if err != nil {
		panic(err)
	}
	cs := make([]*sumtree.Child, len(oldValue))
	for i, oldChild := range oldValue {
		cs[i] = &sumtree.Child{Index: oldChild.Index, Accumulation: oldChild.Acc}
	}
	return &sumtree.Node{Children: cs}
}

func migrateLeafValue(index []byte, oldValueBz []byte) *sumtree.Leaf {
	oldValue := osmomath.ZeroInt()
	err := json.Unmarshal(oldValueBz, &oldValue)
	if err != nil {
		panic(err)
	}
	return sumtree.NewLeaf(index, oldValue)
}

func nodeKey(level uint16, key []byte) []byte {
	bz := make([]byte, 2)
	binary.BigEndian.PutUint16(bz, level)
	return append(append([]byte("node/"), bz...), key...)
}

func leafKey(key []byte) []byte {
	return nodeKey(0, key)
}

func migrateTreeNode(store storetypes.KVStore, level uint16, key []byte) {
	if level == 0 {
		migrateTreeLeaf(store, key)
	} else {
		migrateTreeBranch(store, level, key)
	}
}

func migrateTreeBranch(store storetypes.KVStore, level uint16, key []byte) {
	keyBz := nodeKey(level, key)
	oldValueBz := store.Get(keyBz)
	fmt.Println("migrate", keyBz, string(oldValueBz), level)
	newValue := migrateBranchValue(oldValueBz)
	newValueBz, err := proto.Marshal(newValue)
	if err != nil {
		panic(err)
	}
	store.Set(keyBz, newValueBz)

	for _, child := range newValue.Children {
		migrateTreeNode(store, level-1, child.Index)
	}
}

func migrateTreeLeaf(store storetypes.KVStore, key []byte) {
	keyBz := leafKey(key)
	oldValueBz := store.Get(keyBz)
	newValue := migrateLeafValue(key, oldValueBz)
	newValueBz, err := proto.Marshal(newValue)
	if err != nil {
		panic(err)
	}
	store.Set(keyBz, newValueBz)
}

func MigrateTree(store storetypes.KVStore) {
	iter := stypes.KVStoreReversePrefixIterator(store, []byte("node/"))
	defer iter.Close()
	if !iter.Valid() {
		return
	}
	keybz := iter.Key()[5:]
	level := binary.BigEndian.Uint16(keybz[:2])
	key := keybz[2:]
	migrateTreeNode(store, level, key)
}
