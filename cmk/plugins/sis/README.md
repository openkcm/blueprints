# SIS Plugin 


Configuration in the Host Application's config.yaml

```yaml
plugins:
  - name: <plugin name here>
    path: <binary path>
    type: SystemInformationService
    logLevel: debug #oneof: info, debug, error, warn
    yamlConfiguration: |
      customx:
        fieldx: value-here
```

As shown above, each plugin entry includes a `yamlConfiguration` field. This section allows you to define the configuration 
parameters for the plugin’s binary or service at runtime. The contents of `yamlConfiguration` are automatically passed to the plugin through a gRPC call:
```go
(p *Plugin) Configure(_ context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error)`.
```

Example :
- External plugin, see the [example](./external-plugin-binary/plugin.go)
- Load a plugins, see the [example](./internal/business/business.go)

Build sis plugin as separate binaries
```bash
go build -o sis-plugin ./external-plugin-binary
```

Dummy service using sis-plugin [application](./cmd/main.go).

## Usage source code

Loading all plugins given through config.yaml file as configuration
```glang
	plugins, err := catalog.Load(ctx, catalog.Config{
		Logger:        slog.Default(),
		PluginConfigs: cfg.Plugins,
	})
	if err != nil {
		return err
	}
```

Closing all plugins as resources
```golang
	err := plugins.Close()
	if err != nil {
		// do something with the error
	}
```

Load configured sis plugin and create the grpc client
```golang
sisPlugin := plugins.LookupByTypeAndName("SystemInformationService", "sis")
sisClient := systeminformationv1.NewSystemInformationServiceClient(sisPlugin.ClientConnection())
```

Call a grpc `Get` method out of sis plugin
```golang
_, err := sisClient.Get(ctx, &systeminformationv1.GetRequest{
    Id:   uuid.New().String(),
    Type: systeminformationv1.RequestType_REQUEST_TYPE_SYSTEM,
})
```
