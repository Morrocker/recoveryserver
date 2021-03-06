package director

import (
	"github.com/clonercl/blockserver/blocks"
	blocksremote "github.com/clonercl/blockserver/blocks/master/remote"
	legacy "github.com/clonercl/kaon/blocks"
	legacyremote "github.com/clonercl/kaon/blocks/master/remote"
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

// NewCloud Returns a new cloud object
func NewCloud(c config.Cloud) *Cloud {
	var newCloud Cloud
	newCloud.ClonerKey = c.ClonerKey
	newCloud.FilesAddress = c.FilesAddress
	newCloud.Legacy = c.Legacy
	newCloud.InitStores(c.Stores)
	return &newCloud
}

// InitStores initializes a NewCloud's stores
func (c *Cloud) InitStores(s []config.BlocksMaster) {
	if c.Legacy {
		for _, bm := range s {
			c.LegacyStores = append(c.LegacyStores, legacyremote.New(bm.Address, bm.Magic))
		}
	} else {
		for _, bm := range s {
			c.CurrentStores = append(c.CurrentStores, blocksremote.New(bm.Address, bm.Magic))
		}
	}
}
