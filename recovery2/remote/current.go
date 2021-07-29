package remote

import (
	"encoding/json"
	"fmt"

	"github.com/clonercl/blockserver/blocks"
	blocksremote "github.com/clonercl/blockserver/blocks/master/remote"
	"github.com/morrocker/errors"
	"github.com/morrocker/log"
)

// RBS stores the info to set-up and query remote Files and Blocksmaster
type RBSMulti struct {
	Main blocks.MasterStore
	Bkp  blocks.MasterStore
}

// BlocksList struct to store incoming blocks
type BlocksList struct {
	Blocks []string `json:"blocks"`
}

// NewRBS Returns a new RBSMulti object with a main remote initialized
func NewRBS(addr, magic string) *RBSMulti {
	newRemote := &RBSMulti{
		Main: blocksremote.New(addr, magic),
	}
	return newRemote
}

// SetBkp Sets and initializes a new remote RBS as a backup to the existing main
func (r *RBSMulti) SetBkp(addr, magic string) {
	r.Bkp = blocksremote.New(addr, magic)
}

// GetBlock takes a blocks parameters searches and returns it
func (r *RBSMulti) GetBlock(hash string, user string) (bytes []byte, err error) {
	for x := 0; x < 2; x++ {
		bytes, err = r.Main.Retrieve(hash, user)
		if len(bytes) != 0 {
			return
		}
		if r.Bkp != nil {
			bytes, err = r.Bkp.Retrieve(hash, user)
			if len(bytes) != 0 {
				return
			}
		}
	}
	return nil, errors.New("remote.GetBlock()", fmt.Sprintf("Block %s is ungettable", hash))
}

// GetBlocks takes an array of blocks from a single user and returns an array with them
func (r *RBSMulti) GetBlocks(hashs []string, user string) (bytesArray [][]byte, err error) {
	op := "remote.GetBlocks()"
	for retries := 0; retries < 3; retries++ {
		bytesArray, err = r.Main.RetrieveMultiple(hashs, user)
		if err == nil {
			// log.Noticeln("Error nil, checking if Bkp exists")
			if r.Bkp == nil {
				return
			}
			// log.Noticeln("Bkp presernt")
		} else if retries == 1 {
			return nil, errors.New(op, fmt.Sprintf("failed to fetch blocks array: \n%v\nError:%s", hashs, err))
		}
	}

	issuesMap := make(map[string]int)
	issuesArr := []string{}
	for i, content := range bytesArray {
		if len(content) == 0 {
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
		if len(content) != 0 {
			idx := issuesMap[issuesArr[i]]
			bytesArray[idx] = content
		}
	}

	return
}

// GetBlockslists receives a block and a user and returns an array of block names
func (r *RBSMulti) GetBlocksList(hash string, user string) (blocksList []string, err error) {
	op := "remote.GetBlockList()"
	var blockList *BlocksList
	var block []byte
	for x := 0; x < 2; x++ {
		blockList = &BlocksList{}
		block, err = r.GetBlock(hash, user)
		if err != nil {
			continue
		}
		if err = json.Unmarshal(block, blockList); err != nil {
			continue
		}
		break
	}
	return blockList.Blocks, errors.Extend(op, err)
}

// GetBlockslists receives a lists of blocks from a user and returns an array of arrays of block names
func (r *RBSMulti) GetBlocksLists(hashs []string, user string) (blockLists [][]string, err error) {
	op := "remote.GetBlockLists()"
	blocks, err := r.GetBlocks(hashs, user)
	if err != nil {
		return nil, errors.Extend(op, err)
	}

	for _, block := range blocks {
		if len(block) == 0 {
			blockLists = append(blockLists, []string{})
		} else {
			blockList := &BlocksList{}
			if err := json.Unmarshal(block, blockList); err != nil {
				return nil, errors.Extend(op, err)
			}
			blockLists = append(blockLists, blockList.Blocks)
		}
	}
	return
}
