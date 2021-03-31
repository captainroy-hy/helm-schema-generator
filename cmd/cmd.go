package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	gen "github.com/captainroy-hy/helm-schema-generator/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmd = &cobra.Command{
	Use:           "schema-gen <chart-values-file>",
	SilenceUsage:  true,
	SilenceErrors: true,
	Short:         "Generate OpenAPI v3 JSON schema for Helm chart values",
	Long: `Generate OpenAPI v3 JSON schema for Helm chart values"

Examples:
  $ schema-gen values.yaml
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("values file is required")
		}
		valuesFilePath := args[0]
		values, err := ioutil.ReadFile(valuesFilePath)
		if err != nil {
			return errors.Wrap(err, "cannot read values file")
		}
		schema, err := gen.GenerateSchemaFromValues(values)
		if err != nil {
			return errors.Wrap(err, "cannot generate schema")
		}
		fmt.Println(string(schema))
		return nil
	},
}

func Execute() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
