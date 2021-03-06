package config

// Config stores the configuration values needed to run the recovery server
type Config struct {
	AutoQueueRecoveries bool
	AutoRunRecoveries   bool
	LoginAddr           bool
	HostAddr            string
	RecoveriesJSON      string
	Clouds              map[string]Cloud
	SlackToken          string
	SlackChannel        string
}

// Cloud stores the keys, address and number of storages from which to restrieve data
type Cloud struct {
	ClonerKey    string
	FilesAddress string
	Stores       []BlocksMaster
	Legacy       bool
}

//BlocksMaster stores the address and magic to use for each store
type BlocksMaster struct {
	Magic   string
	Address string
}

// Data stores the recovery server's configuration data
var Data Config
