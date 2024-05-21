package globals

type Config struct {
	Port             int    `json:"port"`
	IpMemory         string `json:"ip_memory"`
	PortMemory       int    `json:"port_memory"`
	NumberFellingTbl int    `json:"number_felling_tbl"`
	AlgorithmTbl     string `json:"algorithm_tbl"`
	PortKernel       int    `json:"port_kernel"`
	IpKernel         string `json:"ip_kernel"`
}

var ClientConfig *Config
