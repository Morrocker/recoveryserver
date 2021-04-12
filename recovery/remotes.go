package recovery

import (
	"encoding/json"
	"fmt"

	"github.com/clonercl/blockserver/blocks"
	blocksremote "github.com/clonercl/blockserver/blocks/master/remote"
	legacy "github.com/clonercl/kaon/blocks"
	legacyremote "github.com/clonercl/kaon/blocks/master/remote"
	"github.com/morrocker/errors"
	"github.com/morrocker/recoveryserver/config"
)

// RBS stores the info to set-up and query remote Files and Blocksmaster
type RBS struct {
	LegacyStores  []legacy.MasterStore
	CurrentStores []blocks.MasterStore
	Legacy        bool
}

// BlocksList asfdasfd asdf a
type BlocksList struct {
	Blocks []string `json:"blocks"`
}

// NewRBS Returns a new cloud object
func NewRBS(c config.Cloud) *RBS {
	newRemote := &RBS{}
	newRemote.Legacy = c.Legacy
	if newRemote.Legacy {
		for _, bm := range c.Stores {
			newRemote.LegacyStores = append(newRemote.LegacyStores, legacyremote.New(bm.Address, bm.Magic))
		}
	} else {
		for _, bm := range c.Stores {
			newRemote.CurrentStores = append(newRemote.CurrentStores, blocksremote.New(bm.Address, bm.Magic))
		}
	}
	return newRemote
}

// GetBlockslist
func (c *RBS) GetBlocksList(hash, user string) (*BlocksList, error) {
	op := "remotes.GetBlockList()"
	block, err := c.GetBlock(hash, user)
	if err != nil {
		return nil, errors.Extend(op, err)
	}

	ret := &BlocksList{}
	if err := json.Unmarshal(block, ret); err != nil {
		return nil, errors.Extend(op, err)
	}
	return ret, nil
}

// GetBlocks
func (c *RBS) GetBlock(hash string, user string) ([]byte, error) {
	op := "remotes.GetBlock()"

	for retries := 0; retries < 2; retries++ {
		if c.Legacy {
			return nil, errors.New(op, "Legacy store cannot use multiple blocks function")
		} else {
			for _, bs := range c.CurrentStores {
				content, err := bs.Retrieve(hash, user)
				if err == nil {
					return content, nil
				}
			}
		}
	}

	return nil, errors.New(op, fmt.Sprintf("block %q is ungettable", hash))
}

// GetBlockslist
func (c *RBS) GetBlocksLists(hashs []string, user string) ([]*BlocksList, error) {
	op := "remotes.GetBlockList()"
	blocks, err := c.GetBlocks(hashs, user)
	if err != nil {
		return nil, errors.Extend(op, err)
	}

	ret := make([]*BlocksList, len(blocks))
	for i, block := range blocks {
		blks := &BlocksList{}
		if err := json.Unmarshal(block, blks); err != nil {
			return nil, errors.Extend(op, err)
		}
		ret[i] = blks
	}
	return ret, nil
}

// GetBlocks
func (c *RBS) GetBlocks(hashs []string, user string) ([][]byte, error) {
	op := "remotes.GetBlock()"

	for retries := 0; retries < 2; retries++ {
		if c.Legacy {
			return nil, errors.New(op, "Legacy doesn't support multiblock download")
		} else {
			for _, bs := range c.CurrentStores {
				content, err := bs.RetrieveMultiple(hashs, user)
				if err == nil {
					return content, nil
				}
			}
		}
	}

	return nil, errors.New(op, fmt.Sprintf("blocklist %v is ungettable", hashs))
}
