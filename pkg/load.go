package pkg

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

var (
	getters = getter.Providers{
		getter.Provider{
			Schemes: []string{"http", "https"},
			New:     getter.NewHTTPGetter,
		},
	}
)

func loadChartFiles(ctx context.Context, repoURL, chart, version string) (*loader.BufferedFile, error) {
	url, err := repo.FindChartInRepoURL(repoURL, chart, version, "", "", "", getters)
	if err != nil {
		return nil, errors.Wrap(err, "cannot find Chart URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot fetch Chart from remote URL:%s", url)
	}
	//nolint:errcheck
	defer resp.Body.Close()
	files, err := loader.LoadArchiveFiles(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "cannot load Chart files")
	}
	var valuesFile *loader.BufferedFile
	for _, f := range files {
		switch f.Name {
		case "values.yaml", "values.yml":
			valuesFile = f
			break
		default:
			continue
		}
	}
	if valuesFile == nil {
		return nil, errors.New("cannot find values.yml nor values.yaml file in the Chart")
	}
	return valuesFile, nil
}
