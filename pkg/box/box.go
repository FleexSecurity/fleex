package box

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
}
