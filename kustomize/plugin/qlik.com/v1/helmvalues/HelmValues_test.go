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
		"qlik.com", "v1", "HelmValues")
	th := kusttest_test.NewKustTestPluginHarness(t, "/app")

	// make temp directory chartHome
	m := th.LoadAndRunTransformer(`
apiVersion: qlik.com/v1
kind: HelmValues
metadata:
  name: qliksense
chartName: qliksense
releaseName: qliksense
values:
  config:
    accessControl:
      testing: 1234
  qix-sessions:
    testing: true`, `
apiVersion: apps/v1
kind: HelmChart
metadata:
  name: qliksense
chartName: qliksense
releaseName: qliksense
values:
  config:
    accessControl:
      testing: 4321
---
apiVersion: apps/v1
kind: HelmChart
metadata:
  name: qix-sessions
chartName: qix-sessions
releaseName: qix-sessions
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
