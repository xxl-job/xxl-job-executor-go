package xxl

type Conf struct {
	ServerAddr   string `json:"server_addr"`
	ExecutorPort string `json:"executor_port"`
	RegistryKey  string `json:"registry_key"`
	AccessToken  string `json:"access_token"`
}
