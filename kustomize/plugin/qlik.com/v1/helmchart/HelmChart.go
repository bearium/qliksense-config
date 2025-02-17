package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"os/exec"

	"github.com/imdario/mergo"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	ChartName        string                 `json:"chartName,omitempty" yaml:"chartName,omitempty"`
	ChartHome        string                 `json:"chartHome,omitempty" yaml:"chartHome,omitempty"`
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
	dir = path.Join(dir, "../")

	if p.HelmHome == "" {
		// make home for helm stuff
		directory := fmt.Sprintf("%s/%s", dir, "dotHelm")
		p.HelmHome = directory
	}

	if len(p.ChartHome) == 0 {
		// make home for chart stuff
		directory := fmt.Sprintf("%s/%s", dir, p.ChartName)
		p.ChartHome = directory
	}

	if p.HelmBin == "" {
		p.HelmBin = "helm"
	}

	if len(p.ChartVersion) > 0 {
		p.ChartVersionExp = fmt.Sprintf("--version=%s", p.ChartVersion)
	} else {
		p.ChartVersionExp = ""
	}

	if p.ChartRepo == "" {
		p.ChartRepo = "https://kubernetes-charts.storage.googleapis.com"
	}

	if p.ReleaseName == "" {
		p.ReleaseName = "release-name"
	}

	if p.ReleaseNamespace == "" {
		p.ReleaseName = "default"
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
	err = deleteRequirements(p.ChartHome)
	if err != nil {
		return nil, err
	}

	templatedYaml, err := p.templateHelm()
	if err != nil {
		return nil, err
	}

	if len(p.ChartPatches) > 0 {
		err := p.formatYaml()
		if err != nil {
			return nil, err
		}
		templatedYaml, err = p.applyPatches(templatedYaml)
		if err != nil {
			return nil, err
		}
	}

	return p.rf.NewResMapFromBytes(templatedYaml)
}

func deleteRequirements(dir string) error {

	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	files, err := d.Readdir(-1)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.Mode().IsRegular() {
			ext := filepath.Ext(file.Name())
			name := file.Name()[0 : len(file.Name())-len(ext)]
			if name == "requirements" {
				err := os.Remove(dir + "/" + file.Name())
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (p *plugin) initHelm() error {
	// build helm flags
	home := fmt.Sprintf("--home=%s", p.HelmHome)
	helmCmd := exec.Command(p.HelmBin, "init", home, "--client-only")
	err := helmCmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (p *plugin) fetchHelm() error {

	// build helm flags
	home := fmt.Sprintf("--home=%s", p.HelmHome)
	untarDir := fmt.Sprintf("--untardir=%s", p.ChartHome)
	repo := fmt.Sprintf("--repo=%s", p.ChartRepo)
	helmCmd := exec.Command("helm", "fetch", home, "--untar", untarDir, repo, p.ChartVersionExp, p.ChartName)

	var out bytes.Buffer
	helmCmd.Stdout = &out
	err := helmCmd.Run()
	if err != nil {
		return err
	}

	fileLocation := fmt.Sprintf("%s/%s", p.ChartHome, p.ChartName)
	tempFileLocation := fileLocation + "-temp"

	err = os.Rename(fileLocation, tempFileLocation)
	if err != nil {
		return err
	}

	err = copyDir(fileLocation+"-temp", p.ChartHome)
	if err != nil {
		return err
	}

	err = os.RemoveAll(tempFileLocation)
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

	// build helm flags
	home := fmt.Sprintf("--home=%s", p.HelmHome)
	values := fmt.Sprintf("--values=%s", file.Name())
	name := fmt.Sprintf("--name=%s", p.ReleaseName)
	nameSpace := fmt.Sprintf("--namespace=%s", p.ReleaseNamespace)
	helmCmd := exec.Command("helm", "template", home, values, name, nameSpace, p.ChartHome)

	if len(p.ExtraArgs) > 0 && p.ExtraArgs != "null" {
		helmCmd.Args = append(helmCmd.Args, p.ExtraArgs)
	}

	if len(p.ValuesFrom) > 0 && p.ValuesFrom != "null" {
		templatedValues := fmt.Sprintf("--values=%s", p.ValuesFrom)
		helmCmd.Args = append(helmCmd.Args, templatedValues)
	}

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

func (p *plugin) formatYaml() error {
	dir, err := os.Open(p.ChartHome + "/" + p.ChartPatches)
	if err != nil {
		return err
	}
	defer dir.Close()

	objs, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		filePath := filepath.Join(dir.Name(), obj)
		if filepath.Ext(filePath) == ".yaml" {
			var parsedString string
			yamlBytes, err := ioutil.ReadFile(filePath)
			if err != nil {
				return err
			}
			if strings.Contains(p.ReleaseName, p.ChartName) {
				parsedString = strings.Replace(string(yamlBytes), "?", "", -1)
			} else {
				parsedString = strings.Replace(string(yamlBytes), "?", p.ReleaseName+"-", -1)
			}
			parsedYaml := strings.Replace(parsedString, "*", p.ReleaseName+"-", -1)

			err = ioutil.WriteFile(filePath, []byte(parsedYaml), 0644)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *plugin) applyPatches(templatedHelm []byte) ([]byte, error) {
	// get the patches
	path := filepath.Join(p.ChartHome + "/" + p.ChartPatches + "/kustomization.yaml")
	origYamlBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var originalYamlMap map[string]interface{}

	yaml.Unmarshal(origYamlBytes, &originalYamlMap)
	patches := originalYamlMap["patchesJson6902"]
	patchArray := patches.([]interface{})

	// helmoutput file for kustomize build
	f, err := os.Create(p.ChartHome + "/" + p.ChartPatches + "/helmoutput.yaml")
	if err != nil {
		return nil, err
	}

	// loop through all patches
	for _, patch := range patchArray {

		_, err = f.Write(templatedHelm)
		if err != nil {
			return nil, err
		}

		kustomizeYaml, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		var kustomizeYamlMap map[string]interface{}
		yaml.Unmarshal(kustomizeYaml, &kustomizeYamlMap)

		// delete old resources in map
		delete(kustomizeYamlMap, "patchesJson6902")
		delete(kustomizeYamlMap, "resources")

		//merge patch data together
		mergedData, err := mergeValues(kustomizeYamlMap["patchesJson6902"], patch)

		// update yaml
		kustomizeYamlMap["patchesJson6902"] = []interface{}{mergedData}
		kustomizeYamlMap["resources"] = []string{"helmoutput.yaml"}

		yamlM, err := yaml.Marshal(kustomizeYamlMap)
		if err != nil {
			return nil, err
		}

		ioutil.WriteFile(path, yamlM, 0644)
		// kustomize build
		templatedHelm, err = p.buildPatches()
		if err != nil {
			return nil, err
		}

	}
	return templatedHelm, nil
}

func (p *plugin) buildPatches() ([]byte, error) {
	path := filepath.Join(p.ChartHome + "/" + p.ChartPatches)
	kustomizeCmd := exec.Command("kustomize", "build", path)

	var out bytes.Buffer
	kustomizeCmd.Stdout = &out

	err := kustomizeCmd.Run()
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func mergeValues(root interface{}, copy interface{}) (interface{}, error) {
	var mergedData map[interface{}]interface{}
	var mergeFrom = make(map[interface{}]interface{})

	mergeFrom["root"] = root
	err := mergo.Merge(&mergedData, mergeFrom)
	if err != nil {
		return nil, err
	}

	mergeFrom["root"] = copy
	err = mergo.Merge(&mergedData, mergeFrom)
	if err != nil {
		return nil, err
	}
	return mergedData["root"], nil
}

// copy source file to destination location
func copyFile(source string, dest string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
			return err
		}
	}
	return nil
}

//copy source directory to destination
func copyDir(source string, dest string) error {
	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}
	sourceDirectory, _ := os.Open(source)
	// read everything within source directory
	objects, _ := sourceDirectory.Readdir(-1)

	// go through all files/directories
	for _, obj := range objects {

		sourceFileName := source + "/" + obj.Name()

		destinationFileName := dest + "/" + obj.Name()

		if obj.IsDir() {
			err := copyDir(sourceFileName, destinationFileName)
			if err != nil {
				return err
			}
		} else {
			err := copyFile(sourceFileName, destinationFileName)
			if err != nil {
				return err
			}
		}

	}
	return nil
}
