package keyValueStorage

import "bytes"

// delete a key from the tree
func treeDelete(tree *BTree, node BNode, key []byte) BNode {
	// where to find the key?
	idx := findLessEqualNode(node, key)
	// act depending on the node type
	switch node.getType() {
	case BNodeLeaf:
		if !bytes.Equal(key, node.getKey(idx)) {
			return BNode{} // not found
		}
		// delete the key in the leaf
		newNode := BNode{data: make([]byte, BtreePageSize)}
		leafDelete(newNode, node, idx)
		return newNode
	case BNodeInternal:
		return nodeDelete(tree, node, idx, key)
	default:
		panic("bad node!")
	}
}

// remove a key from a leaf node
func leafDelete(new BNode, old BNode, idx uint16) {
	new.setHeader(BNodeLeaf, old.numberOfKeys()-1)
	nodeAppendRange(&new, old, 0, 0, idx)
	nodeAppendRange(&new, old, idx, idx+1, old.numberOfKeys()-(idx+1))
}

// todo
func nodeDelete(bTree *BTree, node BNode, idx uint16, key []byte) BNode {
	return BNode{}
}
