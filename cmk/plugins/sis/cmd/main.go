package main

import (
	"github.com/openkcm/plugin-sdk/pkg/plugin"
	systeminformationv1 "github.com/openkcm/plugin-sdk/proto/plugin/systeminformation/v1"
	configv1 "github.com/openkcm/plugin-sdk/proto/service/common/config/v1"
	"github.tools.sap/kms/cert-issuer-plugin/internal/sis"
)

func main() {
	server := sis.NewPlugin()

	plugin.Serve(
		systeminformationv1.SystemInformationServicePluginServer(server),
		configv1.ConfigServiceServer(server),
	)
}
