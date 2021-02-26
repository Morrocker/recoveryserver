package config

// Config stores the configuration values needed to run the recovery server
type Config struct {
	Clouds       map[string]Cloud
	SlackToken   string
	SlackChannel string
}

// Cloud stores the keys, address and number of storages from which to restrieve data
type Cloud struct {
	ClonerKey    string
	FilesAddress string
	Stores       []BlocksMaster
}

//BlocksMaster stores the address and magic to use for each store
type BlocksMaster struct {
	Magic   string
	Address string
}
