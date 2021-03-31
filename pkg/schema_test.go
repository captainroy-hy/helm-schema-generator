package pkg

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestGetChartValuesJSONSchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	tests := map[string]map[string][]string{
		"https://helm.elastic.co": {
			"elasticsearch": {"6.8.15", "7.11.1", "7.12.0"},
			"kibana":        {"7.11.1", "7.12.0"},
		},
		"http://oam.dev/catalog": {
			"podinfo": {"5.1.4"},
		},
		"https://helm.releases.hashicorp.com": {
			"consul":    {"0.31.1"},
			"terraform": {"1.0.0"},
			"vault":     {"0.10.0"},
		},
	}
	for repo, charts := range tests {
		for chart, versions := range charts {
			for _, version := range versions {
				t.Run(fmt.Sprintf("%s#%s@%s", repo, chart, version), func(t *testing.T) {
					values, err := loadChartFiles(ctx, repo, chart, version)
					if err != nil {
						t.Fatal(err, "failed get values file")
					}
					_, err = GenerateSchemaFromValues(values.Data)
					if err != nil {
						t.Error(err, "failed get schema")
					}
				})
			}
		}
	}

}
