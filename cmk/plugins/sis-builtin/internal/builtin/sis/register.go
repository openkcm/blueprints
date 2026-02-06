package sis

import "github.com/openkcm/plugin-sdk/pkg/catalog"

func Register(registry catalog.BuiltInPluginRegistry) {
	registry.Register(BuiltIn())
}
