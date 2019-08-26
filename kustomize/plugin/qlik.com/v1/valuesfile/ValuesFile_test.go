package main_test

import (
	"testing"

	kusttest_test "sigs.k8s.io/kustomize/v3/pkg/kusttest"
	plugins_test "sigs.k8s.io/kustomize/v3/pkg/plugins/test"
)

func TestDatePrefixerPlugin(t *testing.T) {
	tc := plugins_test.NewEnvForTest(t).Set()
	defer tc.Reset()

	tc.BuildGoPlugin(
		"qlik.com", "v1", "ValuesFile")
	th := kusttest_test.NewKustTestPluginHarness(t, "/app")

	th.WriteF("/app/values.tmpl.yaml", `
values:
  config:
	accessControl:
	  testing: 4321
	qix-sessions:
	  testing: true
`)

	// make temp directory chartHome
	m := th.LoadAndRunTransformer(`
apiVersion: qlik.com/v1
kind: ValuesFile
metadata:
  name: collections
valuesFile: "values.tmpl.yaml"`, `
apiVersion: apps/v1
kind: HelmChart
metadata:
  name: qliksense
values:
  config:
    accessControl:
      testing: 4321
  qix-sessions:
    testing: true
`)

	// insure output of yaml is correct
	th.AssertActualEqualsExpected(m, `
apiVersion: apps/v1
chartName: qliksense
kind: HelmChart
metadata:
  name: qliksense
releaseName: qliksense
values:
  config:
    accessControl:
      testing: 4321
  qix-sessions:
    testing: true
---
apiVersion: apps/v1
chartName: qix-sessions
kind: HelmChart
metadata:
  name: qix-sessions
releaseName: qliksense
values:
  qix-sessions:
    testing: true
`)

}
