package globals

type Config struct {
	PortKernel         int      `json:"port_kernel"`
	IpMemory           string   `json:"ip_memory"`
	PortMemory         int      `json:"port_memory"`
	IpCPU              string   `json:"ip_cpu"`
	PortCPU            int      `json:"port_cpu"`
	PortIO             int      `json:"port_io"`
	PlanningAlgorithm  string   `json:"planning_algorithm"`
	Quantum            int      `json:"quantum"`
	Resources          []string `json:"resources"`
	Resource_instances []int    `json:"resource_instances"`
	Multiprogramming   int      `json:"multiprogramming"`
}

var ClientConfig *Config
