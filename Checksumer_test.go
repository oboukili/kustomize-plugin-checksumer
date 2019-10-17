package main_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sigs.k8s.io/kustomize/v3/pkg/kusttest"
	"sigs.k8s.io/kustomize/v3/pkg/loader"
	"sigs.k8s.io/kustomize/v3/pkg/pgmconfig"
	"sigs.k8s.io/kustomize/v3/pkg/plugins"
	"strings"
	"testing"
)

// Start "sigs.k8s.io/kustomize/v3/pkg/plugins/test" override
// We need EnvForTest "workDir" attribute to be accessible

// EnvForTest manages the plugin test environment.
// It sets/resets XDG_CONFIG_HOME, makes/removes a temp objRoot.
type EnvForTest struct {
	t        *testing.T
	compiler *plugins.Compiler
	workDir  string
	oldXdg   string
	wasSet   bool
}

func NewEnvForTest(t *testing.T) *EnvForTest {
	return &EnvForTest{t: t}
}

func (x *EnvForTest) Set() *EnvForTest {
	x.createWorkDir()
	x.compiler = x.makeCompiler()
	x.setEnv()
	return x
}

func (x *EnvForTest) Reset() {
	x.resetEnv()
	x.removeWorkDir()
}

func (x *EnvForTest) BuildGoPlugin(g, v, k string) {
	err := x.compiler.Compile(g, v, k)
	if err != nil {
		x.t.Errorf("compile failed: %v", err)
	}
}

func (x *EnvForTest) BuildExecPlugin(g, v, k string) {
	lowK := strings.ToLower(k)
	obj := filepath.Join(x.compiler.ObjRoot(), g, v, lowK, k)
	src := filepath.Join(x.compiler.SrcRoot(), g, v, lowK, k)
	if err := os.MkdirAll(filepath.Dir(obj), 0755); err != nil {
		x.t.Errorf("error making directory: %s", filepath.Dir(obj))
	}
	cmd := exec.Command("cp", src, obj)
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		x.t.Errorf("error copying %s to %s: %v", src, obj, err)
	}
}

func (x *EnvForTest) makeCompiler() *plugins.Compiler {
	// The plugin loader wants to find object code under
	//    $XDG_CONFIG_HOME/kustomize/plugins
	// and the compiler writes object code to
	//    $objRoot
	// so set things up accordingly.
	objRoot := filepath.Join(
		x.workDir, pgmconfig.ProgramName, pgmconfig.PluginRoot)
	err := os.MkdirAll(objRoot, os.ModePerm)
	if err != nil {
		x.t.Error(err)
	}
	srcRoot, err := plugins.DefaultSrcRoot()
	if err != nil {
		x.t.Error(err)
	}
	return plugins.NewCompiler(srcRoot, objRoot)
}

func (x *EnvForTest) createWorkDir() {
	var err error
	x.workDir, err = ioutil.TempDir("", "kustomize-plugin-tests")
	if err != nil {
		x.t.Errorf("failed to make work dir: %v", err)
	}
}

func (x *EnvForTest) removeWorkDir() {
	err := os.RemoveAll(x.workDir)
	if err != nil {
		x.t.Errorf(
			"removing work dir: %s %v", x.workDir, err)
	}
}

func (x *EnvForTest) setEnv() {
	x.oldXdg, x.wasSet = os.LookupEnv(pgmconfig.XDG_CONFIG_HOME)
	os.Setenv(pgmconfig.XDG_CONFIG_HOME, x.workDir)
}

func (x *EnvForTest) resetEnv() {
	if x.wasSet {
		os.Setenv(pgmconfig.XDG_CONFIG_HOME, x.oldXdg)
	} else {
		os.Unsetenv(pgmconfig.XDG_CONFIG_HOME)
	}
}

// End plugins_test override

func createFileWithinTestEnv(path string, content string) error {
	err := os.MkdirAll(filepath.Dir(path), os.FileMode(int(0750)))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, []byte(content), os.FileMode(int(0640)))
	return err
}

func TestTransformer(t *testing.T) {
	tc := NewEnvForTest(t).Set()
	defer tc.Reset()
	kustomizeRootPath := tc.workDir + "/app"

	err := os.MkdirAll(kustomizeRootPath, os.FileMode(int(0750)))
	if err != nil {
		panic(err)
	}
	tc.BuildGoPlugin("gitlab.com/maltcommunity", "", "Checksumer")

	th := kusttest_test.NewKustTestHarnessFull(t, kustomizeRootPath, loader.RestrictionNone, plugins.ActivePluginConfig())

	err = createFileWithinTestEnv(kustomizeRootPath+"/../test/foobarbaz", "foobarbaz")
	if err != nil {
		panic(err)
	}
	err = createFileWithinTestEnv(kustomizeRootPath+"/../test/bar", "bar")
	if err != nil {
		panic(err)
	}
	err = createFileWithinTestEnv(kustomizeRootPath+"/../test/sub/foo", "foo")
	if err != nil {
		panic(err)
	}
	rm := th.LoadAndRunTransformer(`
apiVersion: gitlab.com/maltcommunity
kind: Checksumer
metadata:
  name: notImportantHere
files:
  - key: directory
    path: ../test
    recurse: false
  - key: directoryRecursive
    path: ../test
    recurse: true
  - key: file
    path: ../test/foobarbaz
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
    directory: 7567aa21664edd14a9eeac26efd9c07a7eed914e
    directoryRecursive: 01cbe21c720fb38989f494dfcc3c4635c156bc04
    file: 5f5513f8822fdbe5145af33b64d8d970dcf95c6e
  name: myService
spec:
  ports:
  - port: 7002
`)
}
