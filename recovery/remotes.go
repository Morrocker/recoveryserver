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
	newRemote := &RBS{CurrentStores: make([]blocks.MasterStore, 2)}
	newRemote.Legacy = c.Legacy
	if newRemote.Legacy {
		for _, bm := range c.Stores {
			newRemote.LegacyStores = append(newRemote.LegacyStores, legacyremote.New(bm.Address, bm.Magic))
		}
	} else {
		for _, bm := range c.Stores {
			if bm.Main {
				newRemote.CurrentStores[0] = blocksremote.New(bm.Address, bm.Magic)
			} else {
				newRemote.CurrentStores[1] = blocksremote.New(bm.Address, bm.Magic)
			}
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
func (c *RBS) GetBlock(hash, user string) ([]byte, error) {
	op := "remotes.GetBlock()"

	for retries := 0; retries < 2; retries++ {
		if c.Legacy {
			for _, bs := range c.LegacyStores {
				content, err := bs.Retrieve(hash)
				if err == nil {
					return content, nil
				}
			}
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

// GetBlocks
func (c *RBS) GetBlocks(hashs []string, user string) (contents map[string][]byte, err error) {
	op := "remotes.GetBlock()"
	contents = make(map[string][]byte)

	bArray := [][]byte{}
	for retries := 0; retries < 2; retries++ {
		if c.Legacy {
			return nil, errors.New(op, "Legacy Store does not support multiblock feature")
		} else {
			bArray, err = c.CurrentStores[0].RetrieveMultiple(hashs, user)
			if err == nil {
				break
			}
		}
		if retries == 2 {
			return nil, errors.New(op, fmt.Sprintf("failed to fetch blocks array: \n%v", hashs))
		}
	}

	issues := []string{}
	iArray := [][]byte{}
	for i, content := range bArray {
		if content == nil {
			issues = append(issues, hashs[i])
			continue
		}
		contents[hashs[i]] = content
	}

	for retries := 0; retries < 2; retries++ {
		iArray, err = c.CurrentStores[1].RetrieveMultiple(issues, user)
		if err == nil {
			break
		}
	}

	for i, content := range iArray {
		contents[issues[i]] = content
	}

	return
}
