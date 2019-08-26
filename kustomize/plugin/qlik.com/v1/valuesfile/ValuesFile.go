package main

import (
	"errors"
	"path/filepath"

	"github.com/imdario/mergo"
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

func mergeFiles(orig map[string]interface{}, tmpl map[string]interface{}) (map[string]interface{}, error) {
	var mergedData = orig

	err := mergo.Merge(&mergedData, tmpl)
	if err != nil {
		return nil, err
	}

	return mergedData, nil
}

func (p *plugin) Transform(m resmap.ResMap) error {

	filePath := filepath.Join(p.Root, p.ValuesFile)

	fileData, err := p.ldr.Load(filePath)
	if err != nil {
		return errors.New("Error: values.tml.yaml is not found")
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
		mergedFile, err := mergeFiles(r.Map(), resMap.Resources()[0].Map())
		if err != nil {
			return err
		}
		r.SetMap(mergedFile)
	}

	return nil
}
