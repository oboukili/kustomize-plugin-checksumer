package main_test

import (
	"testing"

	"sigs.k8s.io/kustomize/v3/pkg/kusttest"
	"sigs.k8s.io/kustomize/v3/pkg/plugins/testenv"
)

func TestChecksumerTransformer(t *testing.T) {
	tc := testenv.NewEnvForTest(t).Set()
	defer tc.Reset()
	tc.BuildGoPlugin("gitlab.com/maltcommunity", "", "Checksumer")
	th := kusttest_test.NewKustTestPluginHarness(t, "/app")
	rm := th.LoadAndRunTransformer(`
apiVersion: gitlab.com/maltcommunity
kind: Checksumer
metadata:
  name: notImportantHere
files:
  sha1DirectorySignature: test/bar
  sha1FileSignature: test/foo
fieldSpecs:
  - path: metadata/annotations
    create: true
`, `
apiVersion: v1
kind: Service
metadata:
  name: myService
spec:
  ports:
  - port: 7002
`)

	th.AssertActualEqualsExpected(rm, `
apiVersion: v1
kind: Service
metadata:
  annotations:
    sha1DirectorySignature: c3ead417bcb50931c13cac6c72a13a30276364ab
    sha1FileSignature: 082ed81741b5b4450b29c8934adb6fea0778c1ce
  name: myService
spec:
  ports:
  - port: 7002
`)
}
