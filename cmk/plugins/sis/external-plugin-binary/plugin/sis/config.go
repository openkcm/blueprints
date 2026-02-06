package sis

type Config struct {
	//config.BaseConfig `yaml:",inline"`

	CustomX CustomX `yaml:"customx"`
}

type CustomX struct {
	FieldX string
}
