package main_test

import (
	"io/ioutil"
	"log"
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

func TestChecksumerTransformer(t *testing.T) {
	tc := NewEnvForTest(t).Set()
	defer tc.Reset()
	tc.BuildGoPlugin("gitlab.com/maltcommunity", "", "Checksumer")
	th := kusttest_test.NewKustTestHarnessFull(t, tc.workDir, loader.RestrictionNone, plugins.ActivePluginConfig())
	err := createFileWithinTestEnv(tc.workDir + "/test/foo", "foobarbaz")
	if err != nil {
		log.Println(err)
	}
	err = createFileWithinTestEnv(tc.workDir + "/test/bar", "bar")
	if err != nil {
		log.Println(err)
	}
	rm := th.LoadAndRunTransformer(`
apiVersion: gitlab.com/maltcommunity
kind: Checksumer
metadata:
  name: notImportantHere
files:
  sha1DirectorySignature: test
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
    sha1DirectorySignature: 983babaa26198dbf060b4f52e3c85b3b456b8b91
    sha1FileSignature: 5f5513f8822fdbe5145af33b64d8d970dcf95c6e
  name: myService
spec:
  ports:
  - port: 7002
`)
}
