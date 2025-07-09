# SIS Plugin 


Usage into the host application config.yaml file:

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

As you see from plugin definition configuration there is `yamlConfiguration` field under which can be 
defined configuration for plugin binary/service runtime behavior. The content of `yamlConfiguration` field is sent to plugin
automatically by calling a gRPC method 
`(p *Plugin) Configure(_ context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error)`.

[Here](./internal/sis/plugin.go) you can find the example how to read it.
