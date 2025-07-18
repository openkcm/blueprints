
# SIS BuiltIn Plugin 


Configuration in the Host Application's config.yaml

```yaml
plugins:
  - name: <plugin name here>
    yamlConfiguration: |[main.go](../sis/external-plugin-binary/main.go)
      customx:
        fieldx: value-here
```

As shown above, each plugin entry includes a `yamlConfiguration` field. This section allows you to define the configuration 
parameters for the plugin’s binary or service at runtime. The contents of `yamlConfiguration` are automatically passed to the plugin through a gRPC call:
```go
(p *Plugin) Configure(_ context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error)`.
```
Example :
- Builtin plugin, see the [example](./internal/builtin/sis/plugin.go)
- Load a builtin plugin, see the [example](./internal/business/business.go)


Dummy service using sis builtin plugin [application](./cmd/main.go).


## Usage source code

Loading plugins given the configuration
```golang
plugins, err := catalog.Load(ctx, catalog.Config{
    Logger:        slog.Default(),
    PluginConfigs: cfg.Plugins,
}, builtin.BuiltIns()...)
if err != nil {
    return err
}
```

Closing all plugins as resources whenever the application is shutdown
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
