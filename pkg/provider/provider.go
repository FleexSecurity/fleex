package provider

type Box struct {
	ID     string
	Label  string
	Group  string
	Status string
	IP     string
}

type Image struct {
	ID      string
	Label   string
	Created string
	Size    int
	Vendor  string
	Status  string
	Regions []string
}

type Provider interface {
	SpawnFleet(fleetName string, fleetCount int) error
	GetBoxes() (boxes []Box, err error)
	GetFleet(fleetName string) (fleet []Box, err error)
	GetBox(boxName string) (Box, error)
	ListImages() error
	GetImages() (images []Image, err error)
	RemoveImages(name string) error
	RunCommand(name, command string, port int, username, password string) error
	CountFleet(fleetName string, boxes []Box) (count int)
	DeleteFleet(name string) error
	DeleteBoxByID(id string) error
	DeleteBoxByLabel(label string) error
	CreateImage(diskID int, label string) error
	TransferImage(imageID int, region string) error
	GetImageRegions(imageID int) ([]string, error)
}
