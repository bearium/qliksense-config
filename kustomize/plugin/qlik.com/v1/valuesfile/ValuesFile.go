package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	ValuesFile string `json:"valuesFile,omitempty" yaml:"valuesFile,omitempty"`
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

func (p *plugin) Transform(m resmap.ResMap) error {
	filePath := filepath.Join(p.Root, p.ValuesFile)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return errors.New("Error: values.tml.yaml is not found")
	}

	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	resMap, err := p.rf.NewResMapFromBytes(fileData)
	if err != nil {
		return err
	}
	for _, r := range m.Resources() {
		_, err := r.AsYAML()
		if err != nil {
			return errors.New("Error: Not a valid yaml file")
		}
		r.Merge(resMap.Resources()[0])
	}

	return nil
}
