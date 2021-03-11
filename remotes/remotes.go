package remotes

import (
	"encoding/json"
	"fmt"

	"github.com/clonercl/blockserver/blocks"
	blocksremote "github.com/clonercl/blockserver/blocks/master/remote"
	legacy "github.com/clonercl/kaon/blocks"
	legacyremote "github.com/clonercl/kaon/blocks/master/remote"
	"github.com/morrocker/errors"
	"github.com/morrocker/logger"
	"github.com/morrocker/recoveryserver/config"
)

// Cloud stores the info to set-up and query remote Files and Blocksmaster
type Cloud struct {
	ClonerKey     string
	FilesAddress  string
	LegacyStores  []legacy.MasterStore
	CurrentStores []blocks.MasterStore
	Legacy        bool
}

// BlocksList asfdasfd asdf a
type BlocksList struct {
	Blocks []string `json:"blocks"`
}

// NewCloud Returns a new cloud object
func NewCloud(c config.Cloud) *Cloud {
	newCloud := &Cloud{}
	newCloud.ClonerKey = c.ClonerKey
	newCloud.FilesAddress = c.FilesAddress
	newCloud.Legacy = c.Legacy
	if newCloud.Legacy {
		for _, bm := range c.Stores {
			newCloud.LegacyStores = append(newCloud.LegacyStores, legacyremote.New(bm.Address, bm.Magic))
		}
	} else {
		for _, bm := range c.Stores {
			newCloud.CurrentStores = append(newCloud.CurrentStores, blocksremote.New(bm.Address, bm.Magic))
		}
	}
	return newCloud
}

// GetBlockslist
func (c *Cloud) GetBlocksList(hash, user string) *BlocksList {
	errPath := "remotes.GetBlockList()"
	block, err := c.GetBlock(hash, user)
	if err != nil {
		logger.Error("%s", errors.Extend(errPath, err))
		return nil
	}

	ret := &BlocksList{}
	if err := json.Unmarshal(block, ret); err != nil {
		logger.Error("%s", errors.Extend(errPath, err))
		return nil
	}
	return ret
}

// GetBlocks
func (c *Cloud) GetBlock(hash, user string) ([]byte, error) {
	for retries := 0; retries < 5; retries++ { // we still have opaque errors
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
	return nil, fmt.Errorf("block %q is ungettable", hash)
}
