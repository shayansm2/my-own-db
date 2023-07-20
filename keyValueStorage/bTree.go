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

// do not need this due to number of keys in a node
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

func getSplitKeyIndex(node BNode) uint16 {
	for i := uint16(1); i <= node.numberOfKeys(); i++ {
		if node.kvPos(i) > BtreePageSize {
			return i - 1
		}
	}
	return node.numberOfKeys()
}

// do not need this due to number of keys in a node
func bisectFindSplitKeyIndex(node BNode) uint16 {
	start, end := uint16(0), node.numberOfKeys()-1
	for start <= end {
		mid := start + (end-start)/2

		pageSize := node.kvPos(mid)

		if pageSize == BtreePageSize {
			return mid
		}

		if pageSize > BtreePageSize {
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
func nodeReplaceChildren(new *BNode, old BNode, idx uint16, children ...BNode) {
	numberOfChildren := uint16(len(children))
	new.setHeader(BNodeInternal, old.numberOfKeys()+numberOfChildren-1)
	nodeAppendRange(new, old, 0, 0, idx)
	for i, child := range children {
		nodeAppendKV(new, idx+uint16(i), tree.new(child), child.getKey(0), nil)
	}
	nodeAppendRange(new, old, idx+numberOfChildren, idx+1, old.numberOfKeys()-(idx+1))
}
