package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"os/exec"

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
	ldr              ifc.Loader
	rf               *resmap.Factory
}

//nolint: go-lint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(
	ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.ldr = ldr
	p.rf = rf
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Generate() (resmap.ResMap, error) {

	// make temp directory
	dir, err := ioutil.TempDir("", "tempRoot")
	if err != nil {
		return nil, err
	}
	if p.HelmHome == "" || p.HelmHome == "null" {
		// make home for helm stuff
		directory := fmt.Sprintf("%s/%s", dir, "dotHelm")
		err := os.Mkdir(directory, 0777)
		if err != nil {
			return nil, err
		}
		p.HelmHome = directory
	}

	if len(p.ChartHome) == 0 || p.ChartHome == "null" {
		// make home for chart stuff
		directory := fmt.Sprintf("%s/%s", dir, p.ChartName)
		p.ChartHome = directory
	}

	if p.HelmBin == "" || p.HelmBin == "null" {
		p.HelmBin = "helm"
	}

	if len(p.ValuesFrom) > 0 && p.ValuesFrom != "null" {
		p.ValuesFrom = fmt.Sprintf("--values=%s", p.ValuesFrom)
	} else {
		p.ValuesFrom = ""
	}

	if len(p.ChartVersion) > 0 && p.ChartVersion != "null" {
		p.ChartVersionExp = fmt.Sprintf("--version=%s", p.ChartVersion)
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

	if len(p.ExtraArgs) < 0 || p.ExtraArgs == "null" {
		p.ExtraArgs = ""
	}

	err = p.initHelm()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(p.ChartHome); os.IsNotExist(err) {
		err = p.fetchHelm()
		if err != nil {
			return nil, err
		}
	}

	templatedYaml, err := p.templateHelm()
	if err != nil {
		return nil, err
	}
	fmt.Println("here")
	return p.rf.NewResMapFromBytes(templatedYaml)
}

func (p *plugin) initHelm() error {
	home := fmt.Sprintf("--home=%s", p.HelmHome)
	helmCmd := exec.Command(p.HelmBin, "init", home, "--client-only")
	err := helmCmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (p *plugin) fetchHelm() error {
	home := fmt.Sprintf("--home=%s", p.HelmHome)
	untarDir := fmt.Sprintf("--untardir=%s", p.ChartHome)
	repo := fmt.Sprintf("--repo=%s", p.ChartName)
	helmCmd := exec.Command("helm", "fetch", home, "--untar", untarDir, repo, p.ChartVersionExp, p.ChartName)
	err := helmCmd.Run()
	var out bytes.Buffer
	var stderr bytes.Buffer
	helmCmd.Stdout = &out
	helmCmd.Stderr = &stderr
	if err != nil {
		return err
	}
	return nil

}

func (p *plugin) templateHelm() ([]byte, error) {

	valuesYaml, err := yaml.Marshal(p.Values)
	if err != nil {
		return nil, err
	}
	file, err := ioutil.TempFile("", "yaml")
	if err != nil {
		return nil, err
	}
	_, err = file.Write(valuesYaml)
	if err != nil {
		return nil, err
	}

	home := fmt.Sprintf("--home=%s", p.HelmHome)
	values := fmt.Sprintf("--values=%s", file.Name())
	name := fmt.Sprintf("--name=%s", p.ReleaseName)
	nameSpace := fmt.Sprintf("--namespace=%s", p.ReleaseNamespace)
	helmCmd := exec.Command("helm", "template", home, values, name, nameSpace, p.ChartHome)
	var out bytes.Buffer
	var stderr bytes.Buffer
	helmCmd.Stdout = &out
	helmCmd.Stderr = &stderr
	err = helmCmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return nil, err
	}
	return out.Bytes(), nil
}

func deleteDirectory(directory string) error {
	dir, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer dir.Close()

	objs, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		fmt.Println(obj)
		err = os.RemoveAll(filepath.Join(directory, obj))
		if err != nil {
			return err
		}
	}
	return nil
}
