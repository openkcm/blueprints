package builtin

import (
	"github.com/openkcm/plugin-sdk/pkg/catalog"
	"github.tools.sap/kms/cert-issuer-builtin-plugin/internal/builtin/sis"
)

func BuiltIns() []catalog.BuiltIn {
	return []catalog.BuiltIn{
		sis.BuiltIn(),
	}
}
