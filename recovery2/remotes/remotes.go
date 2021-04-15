package remote

import (
	"encoding/json"
	"fmt"

	"github.com/clonercl/blockserver/blocks"
	blocksremote "github.com/clonercl/blockserver/blocks/master/remote"
	"github.com/clonercl/blockserver/log"
	"github.com/morrocker/errors"
)

// RBS stores the info to set-up and query remote Files and Blocksmaster
type RBS struct {
	Main blocks.MasterStore
	Bkp  blocks.MasterStore
}

// BlocksList asfdasfd asdf a
type BlocksList struct {
	Blocks []string `json:"blocks"`
}

// NewRBS Returns a new cloud object
func NewRBS(addr, magic string) *RBS {
	newRemote := &RBS{
		Main: blocksremote.New(addr, magic),
	}
	return newRemote
}

// SetBkp adfa adfs
func (r *RBS) SetBkp(addr, magic string) {
	r.Bkp = blocksremote.New(addr, magic)
}

// GetBlocks afda fa fasf
func (r *RBS) GetBlocks(hashs []string, user string) (bytesArray [][]byte, err error) {
	op := "remotes.GetBlocks()"

	for retries := 0; retries < 2; retries++ {
		bytesArray, err = r.Main.RetrieveMultiple(hashs, user)
		if err == nil {
			if r.Bkp == nil {
				return
			}
		} else if retries == 2 {
			return nil, errors.New(op, fmt.Sprintf("failed to fetch blocks array: \n%v", hashs))
		}
	}

	issuesMap := make(map[string]int)
	issuesArr := []string{}
	for i, content := range bytesArray {
		if content == nil {
			issuesArr = append(issuesArr, hashs[i])
			issuesMap[hashs[i]] = i
		}
	}

	issBytesArray := [][]byte{}
	for retries := 0; retries < 2; retries++ {
		issBytesArray, err = r.Bkp.RetrieveMultiple(issuesArr, user)
		if err == nil {
			break
		} else if retries == 2 {
			log.Errorln(errors.New(op, "failed to fetch blocks from Backup"))
			return
		}
	}

	for i, content := range issBytesArray {
		if content != nil {
			idx := issuesMap[issuesArr[i]]
			bytesArray[idx] = content
		}
	}

	return
}

// GetBlockslists asfd adf afd
func (r *RBS) GetBlocksLists(hashs []string, user string) (blockLists [][]string, err error) {
	op := "remotes.GetBlockLists()"
	blocks, err := r.GetBlocks(hashs, user)
	if err != nil {
		return nil, errors.Extend(op, err)
	}

	for _, block := range blocks {
		blockList := &BlocksList{}
		if err := json.Unmarshal(block, blockList); err != nil {
			return nil, errors.Extend(op, err)
		}
		blockLists = append(blockLists, blockList.Blocks)
	}
	return
}
