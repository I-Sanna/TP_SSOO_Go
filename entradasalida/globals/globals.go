package globals

type Config struct {
	Port             int    `json:"port"`
	Type             string `json:"type"`
	UnitWorkTime     int    `json:"unit_work_time"`
	CompactationTime int    `json:"compactation_time"`
	IpKernel         string `json:"ip_kernel"`
	PortKernel       int    `json:"port_kernel"`
	IpMemory         string `json:"ip_memory"`
	PortMemory       int    `json:"port_memory"`
	DialfsPath       string `json:"dialfs_path"`
	DialfsBlockSize  int    `json:"dialfs_block_size"`
	DialfsBlockCount int    `json:"dialfs_block_count"`
}

var ClientConfig *Config
