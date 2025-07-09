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
