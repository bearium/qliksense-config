package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	DataSource map[string]interface{} `json:"dataSource,omitempty" yaml:"dataSource,omitempty"`
	Pwd        string
	ldr        ifc.Loader
	rf         *resmap.Factory
}

//nolint: go-lint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(
	ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.ldr = ldr
	p.rf = rf
	p.Pwd = ldr.Root()
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	var vaultAddressPath, vaultTokenPath interface{}
	if p.DataSource["vault"] != nil {
		vaultAddressPath = p.DataSource["vault"].(map[string]interface{})["addressPath"]
		vaultTokenPath = p.DataSource["vault"].(map[string]interface{})["tokenPath"]

		if vaultAddressPath != "" {
			os.Setenv("VAULT_ADDR", fmt.Sprintf("%v", vaultAddressPath))
		}
		if vaultTokenPath != "" {
			os.Setenv("VAULT_TOKEN", fmt.Sprintf("%v", vaultTokenPath))
		}
	}

	var ejsonPrivateKeyPath interface{}
	if p.DataSource["ejson"] != nil {
		ejsonPrivateKeyPath = p.DataSource["ejson"].(map[string]interface{})["privateKeyPath"]

		if ejsonPrivateKeyPath != "" {
			os.Setenv("EJSON_KEY", fmt.Sprintf("%v", ejsonPrivateKeyPath))
		}
	}

	var dataSource interface{}
	if os.Getenv("EJSON_KEY") != "" {
		dataSource = p.DataSource["ejson"].(map[string]interface{})["filePath"]
	} else if os.Getenv("VAULT_ADDR") != "" && os.Getenv("VAULT_TOKEN ") != "" {
		dataSource = p.DataSource["vault"].(map[string]interface{})["secretPath"]
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
		fmt.Println(string(yamlByte))
		file, err := os.Create(dir + "/allresources.tmpl.yaml")
		if err != nil {
			return err
		}
		_, err = file.Write(yamlByte)
		if err != nil {
			return err
		}
		output, err := runGomplate(dataSource, p.Pwd, dir)
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

func runGomplate(dataSource interface{}, pwd string, dir string) ([]byte, error) {
	dataLocation := filepath.Join(pwd, fmt.Sprintf("%v", dataSource))
	data := fmt.Sprintf("-d data=%s", dataLocation)
	from := fmt.Sprintf("-f %s/allresources.tmpl.yaml", dir)
	out := fmt.Sprintf("-o %s/allresources.yaml", dir)
	gomplateCmd := exec.Command("gomplate", `--left-delim="((" --right-delim="))"`, data, from, out)
	fmt.Println(gomplateCmd.Args)
	err := gomplateCmd.Run()
	var stderr bytes.Buffer
	gomplateCmd.Stderr = &stderr

	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return nil, err
	}

	gomplatedBytes, err := ioutil.ReadFile(out)

	return gomplatedBytes, nil
}
