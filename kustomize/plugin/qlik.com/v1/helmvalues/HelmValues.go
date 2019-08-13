package main

import (
	"github.com/imdario/mergo"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	Overwrite        bool                   `json:"overwrite,omitempty" yaml:"overwrite,omitempty"`
	Chart            string                 `json:"chartName,omitempty" yaml:"chartName,omitempty"`
	ReleaseName      string                 `json:"releaseName,omitempty" yaml:"releaseName,omitempty"`
	ReleaseNamespace string                 `json:"releaseNamespace,omitempty" yaml:"releaseNamespace,omitempty"`
	FieldSpecs       []config.FieldSpec     `json:"fieldSpecs,omitempty" yaml:"fieldSpecs,omitempty"`
	Values           map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
	ValuesName       string
}

//nolint: golint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(
	ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	return yaml.Unmarshal(c, p)
}

func (p *plugin) mutateReleaseNameSpace(in interface{}) (interface{}, error) {
	return p.ReleaseNamespace, nil
}

func (p *plugin) mutateReleaseName(in interface{}) (interface{}, error) {
	return p.ReleaseName, nil
}

func (p *plugin) mutateValues(in interface{}) (interface{}, error) {
	var mergedData map[interface{}]interface{}

	// first merge in whats already in the document stream
	var mergeFrom = make(map[interface{}]interface{})
	mergeFrom["root"] = in
	err := mergeValues(&mergedData, mergeFrom, p.Overwrite)
	if err != nil {
		return nil, err
	}

	// second merge in new values then output
	if p.ValuesName != "" {
		mergeFrom["root"] = p.Values[p.ValuesName]
	} else {
		mergeFrom["root"] = p.Values
	}
	err = mergeValues(&mergedData, mergeFrom, p.Overwrite)
	if err != nil {
		return nil, err
	}
	return mergedData["root"], nil
}

func (p *plugin) Transform(m resmap.ResMap) error {
	for _, r := range m.Resources() {
		if isHelmChart(r) {
			if applyResources(r, p.Chart) {
				err := transformers.MutateField(
					r.Map(),
					[]string{"values"},
					true,
					p.mutateValues)
				if err != nil {
					return err
				}
			}
		}
		name, _ := r.GetString("chartName")
		if p.Values[name] != nil && p.Values[name] != "null" {
			p.ValuesName = name
			err := transformers.MutateField(
				r.Map(),
				[]string{"values", name},
				true,
				p.mutateValues)
			if err != nil {
				return err
			}
			p.ValuesName = ""
		}
		if len(p.ReleaseNamespace) > 0 && p.ReleaseNamespace != "null" {
			err := transformers.MutateField(
				r.Map(),
				[]string{"releaseNamespace"},
				true,
				p.mutateReleaseNameSpace)
			if err != nil {
				return err
			}
		}
		if len(p.ReleaseName) > 0 && p.ReleaseName != "null" {
			err := transformers.MutateField(
				r.Map(),
				[]string{"releaseName"},
				true,
				p.mutateReleaseName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func isHelmChart(obj ifc.Kunstructured) bool {
	kind := obj.GetKind()
	if kind == "HelmChart" {
		return true
	}
	return false
}

func applyResources(obj ifc.Kunstructured, chart string) bool {
	name, _ := obj.GetString("chartName")
	if name == chart || chart == "" || chart == "null" {
		return true
	}
	return false
}

func mergeValues(values1 interface{}, values2 interface{}, overwrite bool) error {
	if overwrite {
		return mergo.Merge(values1, values2, mergo.WithOverride)
	}
	return mergo.Merge(values1, values2)
}
