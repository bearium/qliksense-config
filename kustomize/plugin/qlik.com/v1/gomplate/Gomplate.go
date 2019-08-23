package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"

	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	DataSource map[string]interface{} `json:"dataSource,omitempty" yaml:"dataSource,omitempty"`
	ldr        ifc.Loader
	rf         *resmap.Factory
}

//nolint: go-lint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(
	ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.ldr = ldr
	p.rf = rf
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	var vaultAddressPath, vaultTokenPath string
	if !reflect.ValueOf(p.DataSource["vault"]).IsNil() {
		vaultAddressPath = p.DataSource["vault"].(map[string]string)["addressPath"]
		vaultTokenPath = p.DataSource["vault"].(map[string]string)["tokenPath"]

		if vaultAddressPath != "" {
			os.Setenv("VAULT_ADDR", vaultAddressPath)
		}
		if vaultTokenPath != "" {
			os.Setenv("VAULT_TOKEN", vaultTokenPath)
		}
	}

	var ejsonPrivateKeyPath string
	if !reflect.ValueOf(p.DataSource["ejson"]).IsNil() {
		ejsonPrivateKeyPath = p.DataSource["ejson"].(map[string]string)["privateKeyPath"]

		if ejsonPrivateKeyPath != "" {
			os.Setenv("EJSON_KEY", ejsonPrivateKeyPath)
		}
	}

	var dataSource string
	if os.Getenv("EJSON_KEY") != "" {
		dataSource = "temp"
	} else if os.Getenv("VAULT_ADDR") != "" && os.Getenv("VAULT_TOKEN ") != "" {
		dataSource = "temp"
	} else {
		return errors.New("exit 1")
	}

	dir, err := ioutil.TempDir("", "temp")
	if err != nil {
		return err
	}

	for _, r := range m.Resources() {

		yamlByte, err := r.AsYAML()
		if err != nil {
			return err
		}
		file, err := os.Create(dir + "/allresources.tmpl.yaml")
		if err != nil {
			return err
		}
		_, err = file.Write(yamlByte)
		if err != nil {
			return err
		}
		output, err := runGomplate(dataSource, dir)
		if err != nil {
			return err
		}
		resMap, err := p.rf.NewResMapFromBytes(output)
		if err != nil {
			return err
		}
		r.Replace(resMap.Resources()[0])
	}
	return nil
}

func runGomplate(dataSource string, dir string) ([]byte, error) {
	data := fmt.Sprintf("data=%s", dataSource)
	from := fmt.Sprintf("%s/allresources.tmpl.yaml", dir)
	out := fmt.Sprintf("%s/allresources.yaml", dir)
	gomplateCmd := exec.Command("gomplate", `--left-delim="((" --right-delim="))"`, "-d", data, "-f", from, "-o", out)

	err := gomplateCmd.Run()
	if err != nil {
		return nil, err
	}

	gomplatedBytes, err := ioutil.ReadFile(out)

	return gomplatedBytes, nil
}
