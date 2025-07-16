package main

import (
	"github.com/openkcm/plugin-sdk/pkg/plugin"
	systeminformationv1 "github.com/openkcm/plugin-sdk/proto/plugin/systeminformation/v1"
	configv1 "github.com/openkcm/plugin-sdk/proto/service/common/config/v1"
)

func main() {
	p := NewPlugin()

	plugin.Serve(
		systeminformationv1.SystemInformationServicePluginServer(p),
		configv1.ConfigServiceServer(p),
	)
}
