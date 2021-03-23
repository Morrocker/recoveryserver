package pdf

// Delivery asdf adf a
type Delivery struct {
	OrgName     string
	Requester   string
	Receiver    string
	Address     string
	ExtDelivery string
	Disks       []Disk
	Recoveries  []Recovery
	TotalSize   int64
}

// Recovery asdf a
type Recovery struct {
	User    string
	Machine string
	Disk    string
	Size    int64
}

// Disk asdf a
type Disk struct {
	Name   string
	Brand  string
	Serial string
	Size   string
	Value  int
}

const (
	lineheight float64 = 7
	cellheight float64 = 7
)
