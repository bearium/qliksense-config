package helmchart

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/helm/pkg/helm"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/repo"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	ChartName        string                 `json:"chartName,omitempty" yaml:"chartName,omitempty"`
	ChartHome        string                 `json:"chartHome,omitempty" yaml:"chartHome,omitempty"`
	ChartGitUrl      string                 `json:"chartGitUrl,omitempty" yaml:"chartGitUrl,omitempty"`
	ChartVersion     string                 `json:"chartVersion,omitempty" yaml:"chartVersion,omitempty"`
	ChartRepo        string                 `json:"chartRepo,omitempty" yaml:"chartRepo,omitempty"`
	ValuesFrom       string                 `json:"valuesFrom,omitempty" yaml:"valuesFrom,omitempty"`
	Values           map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
	HelmHome         string                 `json:"helmHome,omitempty" yaml:"helmHome,omitempty"`
	HelmBin          string                 `json:"helmBin,omitempty" yaml:"helmBin,omitempty"`
	ReleaseName      string                 `json:"releaseName,omitempty" yaml:"releaseName,omitempty"`
	ReleaseNamespace string                 `json:"releaseNamespace,omitempty" yaml:"releaseNamespace,omitempty"`
	ExtraArgs        string                 `json:"extraArgs,omitempty" yaml:"extraArgs,omitempty"`
	ChartPatches     string                 `json:"chartPatches,omitempty" yaml:"chartPatches,omitempty"`
	ChartVersionExp  string
}

//nolint: go-lint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(
	ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {

	// make temp directory
	dir, err := ioutil.TempDir("", "tempRoot")
	if err != nil {
		return err
	}
	if p.HelmHome == "" || p.HelmHome == "null" {
		// make home for helm stuff
		directory := fmt.Sprintf("%s/%s", dir, "dotHelm")
		err = os.Mkdir(directory, 0777)
	}
	if p.ChartHome == "" || p.ChartHome == "null" {
		// make home for helm stuff
		directory := fmt.Sprintf("%s/%s", dir, p.ChartName)
		err = os.Mkdir(directory, 0777)
	}

	if p.HelmBin == "" || p.HelmBin == "null" {
		p.HelmBin = "helm"
	}

	if len(p.ValuesFrom) > 0 && p.ValuesFrom != "null" {
		p.ValuesFrom = fmt.Sprintf("--values %s", p.ValuesFrom)
	} else {
		p.ValuesFrom = ""
	}

	if len(p.ChartVersion) > 0 && p.ChartVersion != "null" {
		p.ChartVersionExp = fmt.Sprintf("--version %s", p.ChartVersion)
	} else {
		p.ChartVersionExp = ""
	}

	if p.ChartRepo == "" || p.ChartRepo == "null" {
		p.ChartRepo = "https://kubernetes-charts.storage.googleapis.com"
	}

	if p.ReleaseName == "" || p.ReleaseName == "null" {
		p.ReleaseName = "release-name"
	}

	if p.ReleaseNamespace == "" || p.ReleaseNamespace == "null" {
		p.ReleaseName = "default"
	}

	if p.ExtraArgs == "" || p.ExtraArgs == "null" {
		p.ExtraArgs = ""
	}

	for _, r := range m.Resources() {

	}
	return nil
}

func (p *plugin) fetch() error {

	client := helm.NewClient()

	c := downloader.ChartDownloader{
		Out:      nil,
		Keyring:  defaultKeyring(),
		Verify:   downloader.VerifyNever,
		Username: "",
		Password: "",
	}

	dest, err := ioutil.TempDir("", "helm-")
	if err != nil {
		return fmt.Errorf("Failed to untar: %s", err)
	}
	defer os.RemoveAll(dest)

	chartURL, err := repo.FindChartInAuthRepoURL(f.repoURL, f.username, f.password, f.chartRef, f.version, f.certFile, f.keyFile, f.caFile, getter.All(settings))
	if err != nil {
		return err
	}

	saved, v, err := c.DownloadTo(chartURL, p.ChartVersionExp, dest)
	if err != nil {
		return err
	}

	if !filepath.IsAbs(p.ChartHome) {
		ud = filepath.Join(f.destdir, p.ChartHome)
	}
	if fi, err := os.Stat(ud); err != nil {
		if err := os.MkdirAll(ud, 0755); err != nil {
			return fmt.Errorf("Failed to untar (mkdir): %s", err)
		}

	} else if !fi.IsDir() {
		return fmt.Errorf("Failed to untar: %s is not a directory", ud)
	}

	return chartutil.ExpandFile(ud, saved)

	return nil
}

func defaultKeyring() string {
	return os.ExpandEnv("$HOME/.gnupg/pubring.gpg")
}
