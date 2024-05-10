package globals

type Config struct {
	Port             int    `json:"port"`
	IpMemory         string `json:"ip_memory"`
	PortMemory       int    `json:"port_memory"`
	NumberFellingTbl int    `json:"number_felling_tbl"`
	AlgorithmTbl     string `json:"algorithm_tbl"`
}

var ClientConfig *Config
