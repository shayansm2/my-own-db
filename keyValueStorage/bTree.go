package keyValueStorage

import (
	"bytes"
	"encoding/binary"
)

type BTree struct {
	// pointer (a nonzero page number)
	root uint64
	// callbacks for managing on-disk pages
	get func(uint64) BNode // dereference a pointer
	new func(BNode) uint64 // allocate a new page
	del func(uint64)       // deallocate a page
}

var tree *BTree

func init() {
	node1max := Header + 8 + 2 + 4 + BtreeMaxKeySize + BtreeMaxValSize
	assert(node1max <= BtreePageSize)
	// todo init tree
	tree = nil
}

// insert a KV into a node, the result might be split into 2 nodes.
// the caller is responsible for deallocating the input node
// and splitting and allocating result nodes.
func treeInsert(node BNode, key []byte, val []byte) BNode {
	// where to insert the key?
	index := bisectFindIndexLessEqual(node, key)
	assertThat(assertionArgs{condition: index == findLessEqualNode(node, key), message: "bisect search is wrong"}) // todo remove this
	// act depending on the node type
	return insertIntoNode(node, key, val, index)
}

// returns the first kid node whose range intersects the key. (kid[i] <= key)
func findLessEqualNode(node BNode, key []byte) uint16 {
	nkeys := node.numberOfKeys()
	found := uint16(0)
	// the first key is a copy from the parent node,
	// thus it's always less than or equal to the key.
	for i := uint16(1); i < nkeys; i++ {
		cmp := bytes.Compare(node.getKey(i), key)
		if cmp <= 0 {
			found = i
		}
		if cmp >= 0 {
			break
		}
	}
	return found
}

func bisectFindIndexLessEqual(node BNode, key []byte) uint16 {
	start, end := uint16(0), node.numberOfKeys()-1
	for start <= end {
		mid := start + (end-start)/2

		compare := bytes.Compare(node.getKey(mid), key)

		if compare == 0 {
			return mid
		}

		if compare > 0 {
			end = mid - 1
		} else {
			if end == start {
				return mid
			}
			start = mid + 1
		}
	}
	panic("no index found")
}

func insertIntoNode(node BNode, key []byte, val []byte, index uint16) BNode {
	// the result node.
	// it's allowed to be bigger than 1 page and will be split if so
	newNode := BNode{data: make([]byte, 2*BtreePageSize)}

	switch node.getType() {
	case BNodeLeaf:
		// leaf, node.getKey(index) <= key
		if bytes.Equal(key, node.getKey(index)) {
			// found the key, update it.
			leafNodeUpdate(&newNode, node, index, key, val)
		} else {
			// insert it after the position.
			leafNodeInsert(&newNode, node, index+1, key, val)
		}
	case BNodeInternal:
		// internal node, insert it to a kid node.
		internalNodeInsert(&newNode, node, index, key, val)
	default:
		panic("bad node!")
	}

	return newNode
}

// add a new key to a leaf node
func leafNodeInsert(new *BNode, old BNode, idx uint16, key []byte, val []byte) {
	new.setHeader(BNodeLeaf, old.numberOfKeys()+1)
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, 0, key, val)
	nodeAppendRange(new, old, idx+1, idx, old.numberOfKeys()-idx)
}

func leafNodeUpdate(new *BNode, old BNode, idx uint16, key []byte, val []byte) {
	new.setHeader(BNodeLeaf, old.numberOfKeys())
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, 0, key, val)
	nodeAppendRange(new, old, idx+1, idx+1, old.numberOfKeys()-idx-1)
}

// part of the treeInsert(): KV insertion to an internal node
func internalNodeInsert(new *BNode, node BNode, idx uint16, key []byte, val []byte) {
	// get and deallocate the kid node
	childPointer := node.getPointer(idx)
	childNode := tree.get(childPointer)
	tree.del(childPointer)
	// recursive insertion to the kid node
	childNode = treeInsert(childNode, key, val)
	// split the result
	numberOfSplits, splitedNodes := splitNode(childNode)
	// update the kid links
	nodeReplaceKidN(new, node, idx, splitedNodes[:numberOfSplits]...)
}

// split a node if it's too big. the results are 1~3 nodes.
func splitNode(node BNode) (uint16, [3]BNode) {
	if node.numberOfBytes() <= BtreePageSize {
		node.data = node.data[:BtreePageSize]
		return 1, [3]BNode{node}
	}

	left := BNode{make([]byte, 2*BtreePageSize)} // might be split later
	right := BNode{make([]byte, BtreePageSize)}
	splitIntoTwoNodes(&left, &right, node)
	if left.numberOfBytes() <= BtreePageSize {
		left.data = left.data[:BtreePageSize]
		return 2, [3]BNode{left, right}
	}
	// the left node is still too large
	leftLeft := BNode{make([]byte, BtreePageSize)}
	middle := BNode{make([]byte, BtreePageSize)}
	splitIntoTwoNodes(&leftLeft, &middle, left)
	assert(leftLeft.numberOfBytes() <= BtreePageSize)
	return 3, [3]BNode{leftLeft, middle, right}
}

// split a bigger-than-allowed node into two.
// the second node always fits on a page.
func splitIntoTwoNodes(left *BNode, right *BNode, old BNode) {
	numberOfKeys := getSplitKeyIndex(old)

	right.setHeader(old.getType(), numberOfKeys)
	nodeAppendRange(right, old, 0, 0, numberOfKeys)

	left.setHeader(old.getType(), old.numberOfKeys()-numberOfKeys)
	nodeAppendRange(left, old, 0, numberOfKeys, old.numberOfKeys()-numberOfKeys)
}

// todo bisect
func getSplitKeyIndex(node BNode) uint16 {
	for i := uint16(1); i <= node.numberOfKeys(); i++ {
		if node.kvPos(i) > BtreePageSize {
			return i - 1
		}
	}
	return node.numberOfKeys()
}

// copy multiple KVs into the position
func nodeAppendRange(new *BNode, old BNode, newIndex uint16, oldIndex uint16, length uint16) {
	assert(oldIndex+length <= old.numberOfKeys())
	assert(newIndex+length <= new.numberOfKeys())
	if length == 0 {
		return
	}

	// pointers
	for i := uint16(0); i < length; i++ {
		new.setPointer(newIndex+i, old.getPointer(oldIndex+i))
	}
	// offsets
	dstBegin := new.getOffset(newIndex)
	srcBegin := old.getOffset(oldIndex)
	for i := uint16(1); i <= length; i++ { // NOTE: the range is [1, length]
		offset := dstBegin + old.getOffset(oldIndex+i) - srcBegin
		new.setOffset(newIndex+i, offset)
	}
	// KVs
	begin := old.kvPos(oldIndex)
	end := old.kvPos(oldIndex + length)
	copy(new.data[new.kvPos(newIndex):], old.data[begin:end])
}

// copy a KV into the position
func nodeAppendKV(new *BNode, idx uint16, ptr uint64, key []byte, val []byte) {
	// pointers
	new.setPointer(idx, ptr)
	// Key-Values
	pos := new.kvPos(idx)
	binary.LittleEndian.PutUint16(new.data[pos+0:], uint16(len(key)))
	binary.LittleEndian.PutUint16(new.data[pos+2:], uint16(len(val)))
	copy(new.data[pos+4:], key)
	copy(new.data[pos+4+uint16(len(key)):], val)
	// the offset of the next key
	new.setOffset(idx+1, new.getOffset(idx)+4+uint16(len(key)+len(val)))
}

// replace a link with multiple links
func nodeReplaceKidN(new *BNode, old BNode, idx uint16, kids ...BNode) {
	inc := uint16(len(kids))
	new.setHeader(BNodeInternal, old.numberOfKeys()+inc-1)
	nodeAppendRange(new, old, 0, 0, idx)
	for i, node := range kids {
		nodeAppendKV(new, idx+uint16(i), tree.new(node), node.getKey(0), nil)
	}
	nodeAppendRange(new, old, idx+inc, idx+1, old.numberOfKeys()-(idx+1))
}
