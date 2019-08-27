package main

import (
	"fmt"
	"path/filepath"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	Enabled             bool            `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Path                string          `json:"path,omitempty" yaml:"path,omitempty"`
	Target              *types.Selector `json:"target,omitempty" yaml:"target,omitempty"`
	ldr                 ifc.Loader
	rf                  *resmap.Factory
	StrategicMergePatch *resource.Resource
	json6902Patch       jsonpatch.Patch
}

//nolint: go-lint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(
	ldr ifc.Loader, rf *resmap.Factory, c []byte) error {
	p.ldr = ldr
	p.rf = rf
	err := yaml.Unmarshal(c, p)
	if err != nil {
		return err
	}
	if p.Path == "" {
		return nil
	}

	loadPath := filepath.Join(ldr.Root(), p.Path)
	//load the patch
	patch, err := ldr.Load(loadPath)
	if err != nil {
		return err
	}

	p.StrategicMergePatch, err = p.rf.RF().FromBytes(patch)
	if err == nil {
		return nil
	}
	p.json6902Patch, err = jsonPatchFromBytes(patch)
	if err == nil {
		return nil
	}
	return errors.New("neither a strategic Merge patch or JSON6902 patch was Found")
}

func (p *plugin) Transform(m resmap.ResMap) error {

	if !p.Enabled {
		return nil
	}

	if p.Target == nil {
		return nil
	}
	resources, err := m.Select(*p.Target)
	if err != nil {
		return err
	}

	for _, r := range resources {
		if p.json6902Patch != nil {
			origObj, err := r.MarshalJSON()
			if err != nil {
				return err
			}
			patchedObj, err := p.json6902Patch.Apply(origObj)
			if err != nil {
				return err
			}
			err = r.UnmarshalJSON(patchedObj)
			if err != nil {
				return err
			}
		}
		if p.StrategicMergePatch != nil {
			patchCopy := p.StrategicMergePatch.DeepCopy()
			err := r.Patch(patchCopy.Kunstructured)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// jsonPatchFromBytes loads a Json 6902 patch from
// a bytes input
func jsonPatchFromBytes(in []byte) (jsonpatch.Patch, error) {
	ops := string(in)
	if ops == "" {
		return nil, fmt.Errorf("empty json patch operations")
	}

	if ops[0] != '[' {
		jsonOps, err := yaml.YAMLToJSON(in)
		if err != nil {
			return nil, err
		}
		ops = string(jsonOps)
	}
	return jsonpatch.DecodePatch([]byte(ops))
}
