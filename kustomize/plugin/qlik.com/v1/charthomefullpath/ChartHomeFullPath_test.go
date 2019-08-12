package main_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	kusttest_test "sigs.k8s.io/kustomize/v3/pkg/kusttest"
	plugins_test "sigs.k8s.io/kustomize/v3/pkg/plugins/test"
)

func TestDatePrefixerPlugin(t *testing.T) {
	tc := plugins_test.NewEnvForTest(t).Set()
	defer tc.Reset()

	// create a temp directory and test file
	dir, err := ioutil.TempDir("", "test")
	require.NoError(t, err)

	file, err := ioutil.TempFile(dir, "testFile")
	require.NoError(t, err)
	defer file.Close()

	fileContents := []byte("test")
	_, err = file.Write(fileContents)
	require.NoError(t, err)

	tc.BuildGoPlugin(
		"qlik.com", "v1", "ChartHomeFullPath")
	th := kusttest_test.NewKustTestPluginHarness(t, "/app")

	// make temp directory chartHome
	m := th.LoadAndRunTransformer(`
apiVersion: qlik.com/v1
kind: ChartHomeFullPath
metadata:
  name: qliksense
chartHome: `+dir, `
apiVersion: apps/v1
kind: HelmChart
metadata:
  name: qliksense
chartName: qliksense
releaseName: qliksense
`)

	// pull out modified chartHome for plugin
	var chartHome string
	for _, r := range m.Resources() {
		chartHome, err = r.GetString("chartHome")
		require.NoError(t, err)
	}

	require.NotEqual(t, dir, chartHome)

	//open modified directory
	directory, err := os.Open(chartHome)
	require.NoError(t, err)
	objects, err := directory.Readdir(-1)
	require.NoError(t, err)

	//check the temp file was coppied over correctly
	for _, obj := range objects {
		source := chartHome + "/" + obj.Name()
		readFileContents, err := ioutil.ReadFile(source)
		require.NoError(t, err)
		require.Equal(t, fileContents, readFileContents)
	}

	// insure ouput of yaml is correct
	th.AssertActualEqualsExpected(m, `
apiVersion: apps/v1
chartHome: `+chartHome+`
chartName: qliksense
kind: HelmChart
metadata:
  name: qliksense
releaseName: qliksense
`)
}
