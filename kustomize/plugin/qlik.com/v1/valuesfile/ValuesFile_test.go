package main_test

import (
	"testing"

	kusttest_test "sigs.k8s.io/kustomize/v3/pkg/kusttest"
	plugins_test "sigs.k8s.io/kustomize/v3/pkg/plugins/test"
)

func TestStrategicMergePatch(t *testing.T) {
	tc := plugins_test.NewEnvForTest(t).Set()
	defer tc.Reset()

	tc.BuildGoPlugin(
		"qlik.com", "v1", "ValuesFile")

	th := kusttest_test.NewKustTestPluginHarness(t, "/app")

	th.WriteF("/app/values.tml.yaml", `
apiVersion: app/v1
kind: HelmValues
metadata:
  name: collections
values:
  config:
    accessControl:
      testing: 1234
    qix-sessions:
      testing: true
    test123:
      working: 123
`)

	rm := th.LoadAndRunTransformer(`
apiVersion: qlik.com/v1
kind: ValuesFile
metadata:
  name: qliksense
enabled: true
valuesFile: values.tml.yaml
`,
		`apiVersion: apps/v1
kind: HelmValues
metadata:
  name: collections
values:
  config:
    qix-sessions:
      testing: false
`,
	)

	th.AssertActualEqualsExpected(rm, `
apiVersion: apps/v1
kind: HelmValues
metadata:
  name: collections
values:
  config:
    accessControl:
      testing: 1234
    qix-sessions:
      testing: true
    test123:
      working: 123
`)
}
