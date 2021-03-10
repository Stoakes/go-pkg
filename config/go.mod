module github.com/Stoakes/go-pkg/config

go 1.12

replace github.com/Stoakes/go-pkg/log => ../log

require (
	github.com/Stoakes/go-pkg/log v0.4.0
	github.com/Stoakes/go-pkg/types v0.0.3
	github.com/hashicorp/hcl/v2 v2.9.0
	github.com/mcuadros/go-defaults v1.1.1-0.20161116231230-e1c978be3307
	github.com/pelletier/go-toml v1.4.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.4.0
	go.uber.org/zap v1.10.0
	gopkg.in/yaml.v2 v2.2.4
)
