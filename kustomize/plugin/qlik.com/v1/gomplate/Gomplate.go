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
	var env []string
	var vaultAddressPath, vaultTokenPath interface{}
	var vaultAddress, vaultToken, ejsonKey string
	if p.DataSource["vault"] != nil {
		vaultAddressPath = p.DataSource["vault"].(map[string]interface{})["addressPath"]
		vaultTokenPath = p.DataSource["vault"].(map[string]interface{})["tokenPath"]

		if _, err := os.Stat(fmt.Sprintf("%v", vaultAddressPath)); os.IsNotExist(err) {
			readBytes, err := ioutil.ReadFile(fmt.Sprintf("%v", vaultAddressPath))
			if err != nil {
				return err
			}
			vaultAddress = fmt.Sprintf("VAULT_ADDR=%s", string(readBytes))
			env = append(env, vaultAddress)
		}
		if _, err := os.Stat(fmt.Sprintf("%v", vaultTokenPath)); os.IsNotExist(err) {
			readBytes, err := ioutil.ReadFile(fmt.Sprintf("%v", vaultTokenPath))
			if err != nil {
				return err
			}
			vaultToken = fmt.Sprintf("VAULT_TOKEN=%s", string(readBytes))
			env = append(env, vaultToken)
		}
	}

	var ejsonPrivateKeyPath interface{}
	if p.DataSource["ejson"] != nil {
		ejsonPrivateKeyPath = p.DataSource["ejson"].(map[string]interface{})["privateKeyPath"]
		if _, err := os.Stat(fmt.Sprintf("%v", ejsonPrivateKeyPath)); err == nil {
			readBytes, err := ioutil.ReadFile(fmt.Sprintf("%v", ejsonPrivateKeyPath))
			if err != nil {
				return err
			}
			ejsonKey = fmt.Sprintf("EJSON_KEY=%s", string(readBytes))
			env = append(env, ejsonKey)
		}
	}

	var dataSource interface{}
	if ejsonKey != "" {
		dataSource = p.DataSource["ejson"].(map[string]interface{})["filePath"]
	} else if vaultAddress != "" && vaultToken != "" {
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
		file, err := os.Create(dir + "/temp.tmpl.yaml")
		if err != nil {
			return err
		}
		_, err = file.Write(yamlByte)
		if err != nil {
			return err
		}
		output, err := runGomplate(dataSource, p.Pwd, dir, env, file, string(yamlByte))
		if err != nil {
			return err
		}
		res, _ := p.rf.RF().FromBytes(output)
		r.SetMap(res.Map())
	}
	return nil
}

func runGomplate(dataSource interface{}, pwd string, dir string, env []string, file *os.File, temp string) ([]byte, error) {
	dataLocation := filepath.Join(pwd, fmt.Sprintf("%v", dataSource))
	data := fmt.Sprintf(`--datasource=data=%s`, dataLocation)
	from := fmt.Sprintf(`--in=%s`, temp)

	gomplateCmd := exec.Command("gomplate", `--left-delim=((`, `--right-delim=))`, data, from)

	gomplateCmd.Env = append(os.Environ(), env...)

	var out bytes.Buffer
	gomplateCmd.Stdout = &out
	err := gomplateCmd.Run()

	if err != nil {
		return nil, err
	}
	err = os.RemoveAll(dir)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
