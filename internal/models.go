package internal

type ServiceConfig struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Count   int    `json:"count"`
	Version string `json:"version"`
}
