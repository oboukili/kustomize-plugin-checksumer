package main_test

import (
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/testutils/kusttest"
	"testing"
)

func TestTransformer(t *testing.T) {
	fSys := filesys.MakeFsInMemory()
	testFiles := map[string]string{
		"/test/foobarbaz": "foobarbaz",
		"/test/bar": "bar",
		"/test/sub/foo": "foo",
	}
	
	thfs := kusttest_test.MakeHarnessWithFs(t, fSys)
	if err := fSys.MkdirAll("/test/sub"); err != nil {
		panic(err)
	}
	for path, data := range testFiles {
		thfs.WriteF(path, data)
	}
	th := kusttest_test.MakeEnhancedHarness(thfs.GetT()).
		BuildGoPlugin("github.com", "oboukili", "Checksumer")
	defer th.Reset()
	
	rm := th.LoadAndRunTransformer(`
apiVersion: github.com/oboukili
kind: Checksumer
metadata:
  name: notImportantHere
files:
  - key: directory
    path: /test
    recurse: false
  - key: directoryRecursive
    path: /test
    recurse: true
  - key: file
    path: /test/foobarbaz
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

	thfs.AssertActualEqualsExpected(rm, `
apiVersion: v1
kind: Service
metadata:
  annotations:
    directory: 7567aa21664edd14a9eeac26efd9c07a7eed914e
    directoryRecursive: 01cbe21c720fb38989f494dfcc3c4635c156bc04
    file: 5f5513f8822fdbe5145af33b64d8d970dcf95c6e
  name: myService
spec:
  ports:
  - port: 7002
`)
}
