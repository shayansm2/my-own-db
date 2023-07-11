package keyValueStorage

import "encoding/binary"

type BNode struct {
	data []byte // can be dumped to the disk
}

// header
func (node BNode) getType() uint16 {
	return binary.LittleEndian.Uint16(node.data)
}

func (node BNode) numberOfKeys() uint16 {
	return binary.LittleEndian.Uint16(node.data[2:4])
}

func (node BNode) setHeader(nodeType uint16, numberOfKeys uint16) {
	binary.LittleEndian.PutUint16(node.data[0:2], nodeType)
	binary.LittleEndian.PutUint16(node.data[2:4], numberOfKeys)
}

// pointers
func (node BNode) getPointer(idx uint16) uint64 {
	assert(idx < node.numberOfKeys())
	pos := HEADER + 8*idx
	return binary.LittleEndian.Uint64(node.data[pos:])
}

func (node BNode) setPointer(idx uint16, val uint64) {
	assert(idx < node.numberOfKeys())
	pos := HEADER + 8*idx
	binary.LittleEndian.PutUint64(node.data[pos:], val)
}

// offset list
func offsetPosition(node BNode, idx uint16) uint16 {
	assert(1 <= idx && idx <= node.numberOfKeys())
	return HEADER + 8*node.numberOfKeys() + 2*(idx-1)
}

func (node BNode) getOffset(idx uint16) uint16 {
	if idx == 0 {
		return 0
	}
	return binary.LittleEndian.Uint16(node.data[offsetPosition(node, idx):])
}

func (node BNode) setOffset(idx uint16, offset uint16) {
	binary.LittleEndian.PutUint16(node.data[offsetPosition(node, idx):], offset)
}

// key-values
func (node BNode) kvPos(idx uint16) uint16 {
	assert(idx <= node.numberOfKeys())
	return HEADER + 8*node.numberOfKeys() + 2*node.numberOfKeys() + node.getOffset(idx)
}

func (node BNode) getKey(idx uint16) []byte {
	assert(idx < node.numberOfKeys())
	pos := node.kvPos(idx)
	keyLen := binary.LittleEndian.Uint16(node.data[pos:])
	return node.data[pos+4:][:keyLen]
}

func (node BNode) getVal(idx uint16) []byte {
	assert(idx < node.numberOfKeys())
	pos := node.kvPos(idx)
	keyLen := binary.LittleEndian.Uint16(node.data[pos+0:])
	valLen := binary.LittleEndian.Uint16(node.data[pos+2:])
	return node.data[pos+4+keyLen:][:valLen]
}

// node size in bytes
func (node BNode) numberOfBytes() uint16 {
	return node.kvPos(node.numberOfKeys())
}
