package keyValueStorage

import "bytes"

// insert a KV into a node, the result might be split into 2 nodes.
// the caller is responsible for deallocating the input node
// and splitting and allocating result nodes.
func treeInsert(node BNode, key []byte, val []byte) BNode {
	// where to insert the key?
	index := findLessEqualNode(node, key)
	// act depending on the node type
	return insertIntoNode(node, key, val, index)
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
func internalNodeInsert(new *BNode, old BNode, idx uint16, key []byte, val []byte) {
	// get and deallocate the kid old
	childPointer := old.getPointer(idx)
	childNode := tree.get(childPointer)
	tree.del(childPointer)
	// recursive insertion to the kid old
	childNode = treeInsert(childNode, key, val)
	// split the result
	numberOfSplits, splitedNodes := splitNode(childNode)
	// update the kid links
	nodeReplaceChildren(new, old, idx, splitedNodes[:numberOfSplits]...)
}
