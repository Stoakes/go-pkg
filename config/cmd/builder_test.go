package cmd

import (
	"bytes"
	"io/ioutil"
	"testing"
)

type ConfigurationYamlTest struct {
	Mode  bool `yaml:"mode" toml:"mode" hcl:"mode" default:"false" comment:"Use remote or local as backend"`
	Local struct {
		ConnectionString string `yaml:"connection_string" toml:"connection_string" hcl:"connection_string" default:"postgresql://user:password@localhost:5432/postgres" comment:"Database connection string"`
	} `yaml:"Local" toml:"Local" hcl:"Local,block" comment:" Settings for local connection"`
	Remote struct {
		Address string `yaml:"address" toml:"address" hcl:"address" default:"" comment:"Remote database connection string"`
		UseTLS  bool   `yaml:"useTLS" toml:"useTLS" hcl:"useTLS" default:"false" comment:"Enable TLS listener"`
	} `hcl:"remote,block"`
}

func Test_FileExportCommand(t *testing.T) {

	tomlExpectedOutput := `
# Use remote or local as backend
mode = false

#  Settings for local connection
[Local]

  # Database connection string
  connection_string = "postgresql://user:password@localhost:5432/postgres"

[Remote]

  # Remote database connection string
  address = ""

  # Enable TLS listener
  useTLS = false
`

	yamlExpectedOutput := `mode: false
Local:
  connection_string: postgresql://user:password@localhost:5432/postgres
remote:
  address: ""
  useTLS: false
`

	hclExpectedOutput := `mode = false

Local {
  connection_string = "postgresql://user:password@localhost:5432/postgres"
}

remote {
  address = ""
  useTLS  = false
}
`

	envExpectedOutput := `export TEST_LOCAL_CONNECTIONSTRING="postgresql://user:password@localhost:5432/postgres"
export TEST_MODE="false"
export TEST_REMOTE_ADDRESS=""
export TEST_REMOTE_USETLS="false"
`

	// test table
	tt := []struct {
		name           string
		format         ConfigFileExportFormat
		expectedOutput string
		args           []string
	}{
		{
			name:           "toml",
			format:         Toml,
			expectedOutput: tomlExpectedOutput,
			args:           []string{"new"},
		},
		{
			name:           "yaml",
			format:         Yaml,
			expectedOutput: yamlExpectedOutput,
			args:           []string{"new"},
		},
		{
			name:           "hcl",
			format:         Hcl,
			expectedOutput: hclExpectedOutput,
			args:           []string{"new"},
		},
		{
			name:           "none",
			format:         None,
			expectedOutput: envExpectedOutput,
			args:           []string{"new", "--env"},
		},
	}

	// for each test case in test table
	for _, tc := range tt {
		cmd := NewConfigCommand(&ConfigurationYamlTest{}, "TEST", tc.format)
		b := bytes.NewBufferString("")
		cmd.SetArgs(tc.args)
		cmd.SetOut(b)
		cmd.Execute()
		out, err := ioutil.ReadAll(b)
		if err != nil {
			t.Fatalf("Error reading new %s command output: %s", tc.name, err.Error())
		}
		if string(out) != tc.expectedOutput {
			t.Errorf("%s test error. Expected \"%s\" got \"%s\"", tc.name, tc.expectedOutput, string(out))
		}
	}
}
