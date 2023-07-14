package keyValueStorage

const (
	BNodeInternal = 1 // internal nodes without values
	BNodeLeaf     = 2 // leaf nodes with values
)

const Header = 4

const BtreePageSize = 4096
const BtreeMaxKeySize = 1000
const BtreeMaxValSize = 3000
