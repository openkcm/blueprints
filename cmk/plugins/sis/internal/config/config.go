package config

import (
	config "github.com/openkcm/common-sdk/pkg/commoncfg"
)

type Config struct {
	config.BaseConfig `yaml:",inline"`

	CustomX CustomX `yaml:"customx"`
}

type CustomX struct {
	FieldX string
}
