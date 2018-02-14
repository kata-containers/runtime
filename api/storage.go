package api

type Storage struct {
	Name          string
	Driver        string
	DriverOptions map[string]string
	Source        string
	Fstype        string
	MoutOptions   []string
}
