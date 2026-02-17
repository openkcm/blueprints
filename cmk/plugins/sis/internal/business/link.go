package business

import (
	"github.com/openkcm/plugin-sdk/pkg/catalog"
	"github.com/openkcm/sis/external-plugin-binary/plugin/sis"
)

func RegisterAllBuiltInPlugins(registry catalog.BuiltInPluginRegistry) {
	sis.Register(registry)
}
