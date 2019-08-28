package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/imdario/mergo"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	DataSource map[string]interface{} `json:"dataSource,omitempty" yaml:"dataSource,omitempty"`
	ValuesFile string                 `json:"valuesFile,omitempty" yaml:"valuesFile,omitempty"`
	Root       string
	ldr        ifc.Loader
	rf         *resmap.Factory
}

//nolint: golint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(
	ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.ldr = ldr
	p.rf = rf
	p.Root = ldr.Root()
	return yaml.Unmarshal(c, p)
}

func mergeFiles(orig map[string]interface{}, tmpl map[string]interface{}) (map[string]interface{}, error) {
	var mergedData = orig

	err := mergo.Merge(&mergedData, tmpl)
	if err != nil {
		return nil, err
	}

	return mergedData, nil
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
	if os.Getenv("EJSON_KEY") != "" && ejsonKey == "" {
		ejsonKey = fmt.Sprintf("EJSON_KEY=%s", os.Getenv("EJSON_KEY"))
		env = append(env, ejsonKey)
	}

	var dataSource interface{}
	if ejsonKey != "" {
		dataSource = p.DataSource["ejson"].(map[string]interface{})["filePath"]
	} else if vaultAddress != "" && vaultToken != "" {
		dataSource = p.DataSource["vault"].(map[string]interface{})["secretPath"]
	} else {
		return errors.New("exit 1")
	}

	filePath := filepath.Join(p.Root, p.ValuesFile)
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.New("Error: values.tml.yaml is not found")
	}

	if err != nil {
		return err
	}
	for _, r := range m.Resources() {
		// gomplate the initial values file first
		_, err := r.AsYAML()
		if err != nil {
			return errors.New("Error: Not a valid yaml file")
		}
		output, err := runGomplate(dataSource, p.Root, env, string(fileData))
		if err != nil {
			return err
		}
		var Values map[string]interface{}
		err = yaml.Unmarshal(output, &Values)
		if err != nil {
			return err
		}
		ValuePrefixed := map[string]interface{}{"values": Values}

		mergedFile, err := mergeFiles(r.Map(), ValuePrefixed)
		if err != nil {
			return err
		}
		r.SetMap(mergedFile)
	}

	return nil
}

func runGomplate(dataSource interface{}, pwd string, env []string, temp string) ([]byte, error) {
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

	return out.Bytes(), nil
}
