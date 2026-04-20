package gounslop

type Architecture map[string]PolicyConfig

type Config struct {
	Disable      []string     `json:"disable" yaml:"disable"`
	Architecture Architecture `json:"architecture" yaml:"architecture"`
}

type PolicyConfig struct {
	Imports []string `json:"imports" yaml:"imports"`
	Exports []string `json:"exports" yaml:"exports"`
	Shared  bool     `json:"shared" yaml:"shared"`
	Mode    *string  `json:"mode" yaml:"mode"`
}

func StrPtr(s string) *string {
	return &s
}
