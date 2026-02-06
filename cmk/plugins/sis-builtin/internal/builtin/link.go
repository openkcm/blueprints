package builtin

import (
	"github.com/openkcm/plugin-sdk/pkg/catalog"
	"github.tools.sap/kms/sis-builtin-plugin/internal/builtin/sis"
)

func RegisterAllBuiltInPlugins(registry catalog.BuiltInPluginRegistry) {
	sis.Register(registry)
}
