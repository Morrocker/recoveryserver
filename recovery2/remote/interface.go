package remote

type RBS interface {
	GetBlock(string, string) ([]byte, error)
	GetBlocks([]string, string) ([][]byte, error)
	GetBlocksList(string, string) ([]string, error)
	GetBlocksLists([]string, string) ([][]string, error)
}
