package main_test

import (
	"testing"

	kusttest_test "sigs.k8s.io/kustomize/v3/pkg/kusttest"
	plugins_test "sigs.k8s.io/kustomize/v3/pkg/plugins/test"
)

func TestPatchTransformerFromFiles(t *testing.T) {
	tc := plugins_test.NewEnvForTest(t).Set()
	defer tc.Reset()

	tc.BuildGoPlugin(
		"qlik.com", "v1", "SelectivePatch")

	th := kusttest_test.NewKustTestPluginHarness(t, "/app")

	th.WriteF("/app/patch.yaml", `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: qliksense
spec:
  template:
    metadata:
      labels:
        working: true
`)

	rm := th.LoadAndRunTransformer(`
apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: qliksense
enabled: true
path: patch.yaml
target:
  name: qliksense
`,
		`apiVersion: apps/v1
kind: Deployment
metadata:
  name: qliksense
spec:
  template:
    metadata:
      labels:
        working: false
`,
	)

	th.AssertActualEqualsExpected(rm, `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: qliksense
spec:
  template:
    metadata:
      labels:
        working: true
`)
}
