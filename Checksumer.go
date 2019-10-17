package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
	"sigs.k8s.io/yaml"
	"strings"
)

type FileSpec struct {
	Key     string `json:"key,omitempty" yaml:"key,omitempty"`
	Path    string `json:"path,omitempty" yaml:"path,omitempty"`
	Recurse bool   `json:"recurse,omitempty" yaml:"recurse,omitempty"`
}

type plugin struct {
	Files      []FileSpec         `json:"files,omitempty" yaml:"files,omitempty"`
	FieldSpecs []config.FieldSpec `json:"fieldSpecs,omitempty" yaml:"fieldSpecs,omitempty"`
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
		if fileSpec.Recurse {
			sigs := make([]string, 0)
			files, err := ioutil.ReadDir(fileSpec.Path)
			if err != nil {
				return "", err
			}
			for _, f := range files {
				// TODO: make that a FileSpec field parameter
				// ignore hidden files such as git ignore files, etc
				if strings.HasPrefix(f.Name(), ".") {
					continue
				}
				s, err := GetFileSpecSignature(FileSpec{
					Path:    fileSpec.Path + "/" + f.Name(),
					Recurse: fileSpec.Recurse,
				})
				if err != nil {
					return "", err
				}
				sigs = append(sigs, s)
			}
			b = []byte(strings.Join(sigs[:], ""))
		}
	case false:
		b, err = ioutil.ReadFile(fileSpec.Path)
		if err != nil {
			return "", err
		}
	}
	return SHA1Sum(b), nil
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
	t, err := transformers.NewMapTransformer(
		p.FieldSpecs,
		computedKVs,
	)
	if err != nil {
		return err
	}
	return t.Transform(m)
}
