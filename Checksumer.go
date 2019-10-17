package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
	"sigs.k8s.io/yaml"
	"strings"
)

type plugin struct {
	Files      map[string]string  `json:"files,omitempty" yaml:"files,omitempty"`
	FieldSpecs []config.FieldSpec `json:"fieldSpecs,omitempty" yaml:"fieldSpecs,omitempty"`
	Loader     *ifc.Loader
}

var KustomizePlugin plugin

type ExtendedFileInfo struct {
	os.FileInfo
	FilePath string
}

func GetFilesFromDirectory(d string) ([]*ExtendedFileInfo, error) {

	results := make([]*ExtendedFileInfo, 0)
	var b strings.Builder

	_, err := os.Stat(d)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", d, err)
	}
	files, err := ioutil.ReadDir(d)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		b.Reset()
		b.WriteString(d)
		b.WriteString("/")
		b.WriteString(f.Name())
		// recursive subdirectory files lookup
		if f.IsDir() {
			results, err = GetFilesFromDirectory(b.String())
		}
		// ignore hidden files such as git ignore files, etc
		if !strings.HasPrefix(f.Name(), ".") {
			results = append(results, &ExtendedFileInfo{
				FileInfo: f,
				FilePath: b.String(),
			})
		}
	}
	return results, nil
}

func SHA1FileChecksum(f interface{}) (sig string, err error) {
	input := make([]byte, 0)
	switch f.(type) {
	// if a file was specified as the files value
	case string:
		_, err := os.Stat(f.(string))
		if err != nil {
			return sig, err
		}
		input, err = ioutil.ReadFile(f.(string))

	// if a directory was specified as the files value (passed through GetFilesFromDirectory first)
	case *ExtendedFileInfo:
		input, err = ioutil.ReadFile((f.(*ExtendedFileInfo)).FilePath)
	}
	if err != nil {
		return sig, err
	}
	h := sha1.New()
	h.Write(input)
	return fmt.Sprintf("%x", h.Sum(nil)), err
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.Files = nil
	p.FieldSpecs = nil
	p.Loader = &ldr
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	var sig string
	ldr := *p.Loader

	for k, v := range p.Files {
		filePath := ldr.Root() + "/" + v
		// Directory or file determination
		i, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		if i.IsDir() {
			files, err := GetFilesFromDirectory(filePath)
			if err != nil {
				return err
			}
			sigs := make([]byte, 0)

			for _, f := range files {
				sig, err := SHA1FileChecksum(f)
				if err != nil {
					return err
				}
				sigs = append(sigs, sig...)
			}
			h := sha1.New()
			h.Write(sigs)
			sig = fmt.Sprintf("%x", h.Sum(nil))
		} else {
			sig, err = SHA1FileChecksum(filePath)
			if err != nil {
				return err
			}
		}
		p.Files[k] = sig
	}
	t, err := transformers.NewMapTransformer(
		p.FieldSpecs,
		p.Files,
	)
	if err != nil {
		return err
	}
	return t.Transform(m)
}
