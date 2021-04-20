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

// BlocksList asfdasfd asdf a
type BlocksList struct {
	Blocks []string `json:"blocks"`
}

// NewRBS Returns a new cloud object
func NewRBS(addr, magic string) *RBSMulti {
	newRemote := &RBSMulti{
		Main: blocksremote.New(addr, magic),
	}
	return newRemote
}

// SetBkp adfa adfs
func (r *RBSMulti) SetBkp(addr, magic string) {
	r.Bkp = blocksremote.New(addr, magic)
}

func (r *RBSMulti) GetBlock(hash string, user string) ([]byte, error) {
	hashs := []string{hash}
	bytesArray, err := r.GetBlocks(hashs, user)
	if bytesArray[0] == nil {
		return nil, errors.New("remote.current.GetBlocksList()", fmt.Sprintf("Block %s is ungettable", hash))
	}
	return bytesArray[0], err
}

// GetBlocks afda fa fasf
func (r *RBSMulti) GetBlocks(hashs []string, user string) (bytesArray [][]byte, err error) {
	op := "remotes.GetBlocks()"

	for retries := 0; retries < 2; retries++ {
		bytesArray, err = r.Main.RetrieveMultiple(hashs, user)
		if err == nil {
			log.Noticeln("Error nil, checking if Bkp exists")
			if r.Bkp == nil {
				log.Noticeln("Bkp NOt presernt")
				return
			}
			log.Noticeln("Bkp presernt")
		} else if retries == 2 {
			return nil, errors.New(op, fmt.Sprintf("failed to fetch blocks array: \n%v", hashs))
		}
	}

	log.Noticeln("went through???")

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

func (r *RBSMulti) GetBlocksList(hash string, user string) (blockList []string, err error) {
	hashs := []string{hash}
	bLists, err := r.GetBlocksLists(hashs, user)
	if bLists[0] == nil {
		return nil, errors.New("remote.current.GetBlocksList()", "Blocklist ungettable")
	}
	return bLists[0], err
}

// GetBlockslists asfd adf afd
func (r *RBSMulti) GetBlocksLists(hashs []string, user string) (blockLists [][]string, err error) {
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
