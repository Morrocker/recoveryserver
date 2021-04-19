package remote

import (
	"encoding/json"
	"fmt"

	legacy "github.com/clonercl/kaon/blocks"
	legacyremote "github.com/clonercl/kaon/blocks/master/remote"
	"github.com/morrocker/errors"
)

// RBSLegacy stores the info to set-up and query remote Files and Blocksmaster
type RBSLegacy struct {
	Main legacy.MasterStore
	Bkp  legacy.MasterStore
}

func NewRBSLegacy(addr, magic string) *RBSLegacy {
	newRemote := &RBSLegacy{
		Main: legacyremote.New(addr, magic),
	}
	return newRemote
}

// SetBkp asdf
func (r *RBSLegacy) SetBkp(addr, magic string) {
	r.Bkp = legacyremote.New(addr, magic)
}

// GetBlocks
func (r *RBSLegacy) GetBlock(hash, user string) ([]byte, error) {
	op := "remotes.GetBlock()"

	for retries := 0; retries < 2; retries++ {
		content, err := r.Main.Retrieve(hash)
		if err == nil {
			return content, nil
		}
		content, err = r.Bkp.Retrieve(hash)
		if err == nil {
			return content, nil
		}
	}
	return nil, errors.New(op, fmt.Sprintf("block %q is ungettable", hash))
}

func (r *RBSLegacy) GetBlocks(hashs []string, user string) ([][]byte, error) {
	return nil, errors.New("remote.legacy.GetBlocks()", "Multi-block feature not available on legacy remote")
}

// GetBlockslist
func (r *RBSLegacy) GetBlocksList(hash, user string) (blocks []string, err error) {
	op := "remotes.GetBlockList()"
	block, err := r.GetBlock(hash, user)
	if err != nil {
		return nil, errors.Extend(op, err)
	}

	blocksLists := &BlocksList{}
	if err := json.Unmarshal(block, blocksLists); err != nil {
		return nil, errors.Extend(op, err)
	}
	return blocksLists.Blocks, nil
}

func (r *RBSLegacy) GetBlocksLists(hashs []string, user string) ([][]string, error) {
	return nil, errors.New("remote.legacy.GetBlocks()", "Multi-block feature not available on legacy remote")
}
