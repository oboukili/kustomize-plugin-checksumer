package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/kustomize/api/ifc"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/transform"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"strings"
)

type FileSpec struct {
	Key     string `json:"key,omitempty" yaml:"key,omitempty"`
	Path    string `json:"path,omitempty" yaml:"path,omitempty"`
	Recurse bool   `json:"recurse,omitempty" yaml:"recurse,omitempty"`
}

type plugin struct {
	Files      []FileSpec         `json:"files,omitempty" yaml:"files,omitempty"`
	FieldSpecs []types.FieldSpec `json:"fieldSpecs,omitempty" yaml:"fieldSpecs,omitempty"`
	loader     *ifc.Loader
}

var KustomizePlugin plugin

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.Files = nil
	p.FieldSpecs = nil
	p.loader = &ldr
	return yaml.Unmarshal(c, p)
}

func GetFileSpecSignature(fileSpec FileSpec) (sig string, err error) {
	var b []byte
	i, err := os.Stat(fileSpec.Path)
	if err != nil {
		return "", err
	}
	switch t := i.IsDir(); t {
	case true:
		sigs := make([]string, 0)

		files, err := ioutil.ReadDir(fileSpec.Path)
		if err != nil {
			return "", err
		}
		for _, f := range files {
			filePath := fileSpec.Path + "/" + f.Name()
			// TODO: make that a FileSpec field parameter
			// ignore hidden files such as git ignore files, etc
			if strings.HasPrefix(f.Name(), ".") {
				continue
			}
			if f.IsDir() && fileSpec.Recurse {
				s, err := GetFileSpecSignature(FileSpec{
					Path:    filePath,
					Recurse: fileSpec.Recurse,
				})
				if err != nil {
					return "", err
				}
				b = []byte(s)
			}
			if !f.IsDir() {
				b, err = ioutil.ReadFile(filePath)
			}
			if err != nil {
				return "", err
			}
			sigs = append(sigs, SHA1Sum(b))
		}
		b = []byte(strings.Join(sigs[:], ""))
	case false:
		b, err = ioutil.ReadFile(fileSpec.Path)
		if err != nil {
			return "", err
		}
	}
	return SHA1Sum(b), err
}

func SHA1Sum(b []byte) string {
	h := sha1.New()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (p *plugin) Transform(m resmap.ResMap) (err error) {
	computedKVs := make(map[string]string, 0)
	for _, fileSpec := range p.Files {
		// Appending Kustomization root path if path is relative
		if !filepath.IsAbs(fileSpec.Path) {
			fileSpec.Path = (*p.loader).Root() + "/" + fileSpec.Path
		}
		computedKVs[fileSpec.Key], err = GetFileSpecSignature(fileSpec)
		if err != nil {
			return err
		}
	}
	t, err := transform.NewMapTransformer(
		p.FieldSpecs,
		computedKVs,
	)
	if err != nil {
		return err
	}
	return t.Transform(m)
}
