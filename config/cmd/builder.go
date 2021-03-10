package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Stoakes/go-pkg/log"
	"github.com/Stoakes/go-pkg/types"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	defaults "github.com/mcuadros/go-defaults"
	toml "github.com/pelletier/go-toml"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"
)

var configNewAsEnvFlag bool

// ConfigFileExportFormat controls format for configuration to file export
type ConfigFileExportFormat int

const (
	// Hcl export config as HCL file
	Hcl ConfigFileExportFormat = iota
	// Toml export config as Toml file
	Toml
	// Yaml export config as Yaml file. default
	Yaml
	// None does not export config as file
	None
)

// NewConfigCommand initialize a cobra config command tree
func NewConfigCommand(conf interface{}, envPrefix string, exportFormat ConfigFileExportFormat) *cobra.Command {
	// Uppercase the prefix
	upPrefix := strings.ToUpper(envPrefix)

	// config
	configCmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "Manage Service Configuration",
	}

	// config new
	configNewCmd := &cobra.Command{
		Use:   "new",
		Short: "Initialize a default configuration",
		Run: func(cmd *cobra.Command, args []string) {
			defaults.SetDefaults(conf)

			if !configNewAsEnvFlag {
				if exportFormat == Toml {
					btes, err := toml.Marshal(conf)
					if err != nil {
						log.For(cmd.Context()).Fatal("Error during configuration export", zap.Error(err))
					}
					fmt.Fprintf(cmd.OutOrStdout(), string(btes))
					return
				}
				if exportFormat == Hcl {
					f := hclwrite.NewEmptyFile()
					gohcl.EncodeIntoBody(conf, f.Body())
					fmt.Fprintf(cmd.OutOrStdout(), string(f.Bytes()))
					return
				}
				if exportFormat == None {
					return
				}
				// Yaml is default export format
				btes, err := yaml.Marshal(conf)
				if err != nil {
					log.For(cmd.Context()).Fatal("Error during configuration export", zap.Error(err))
				}
				fmt.Fprintf(cmd.OutOrStdout(), string(btes))
				return
			}

			m, err := types.AsEnvVariables(conf, upPrefix, true)
			if err != nil {
				log.For(cmd.Context()).Fatal("Error during environment variables processing", zap.Error(err))
			}
			keys := []string{}

			for k := range m {
				keys = append(keys, k)
			}

			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(cmd.OutOrStdout(), "export %s=\"%s\"\n", k, m[k])
			}

		},
	}

	// flags
	configNewCmd.Flags().BoolVar(&configNewAsEnvFlag, "env", false, "Print configuration as environment variable")
	configCmd.AddCommand(configNewCmd)

	// Return base command
	return configCmd
}
