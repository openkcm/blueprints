# SIS Plugin 


Configuration in the Host Application's config.yaml

```yaml
plugins:
  - name: <plugin name here>
    path: <binary path>
    type: SystemInformationService
    yamlConfiguration:
      logger:
        level: debug
      customx:
        fieldx: value-here
```

As shown above, each plugin entry includes a `yamlConfiguration` field. This section allows you to define the configuration 
parameters for the plugin’s binary or service at runtime. The contents of `yamlConfiguration` are automatically passed to the plugin through a gRPC call:
```go
(p *Plugin) Configure(_ context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error)`.
```

[Here](./internal/sis/plugin.go) you can find the example how to read it.
