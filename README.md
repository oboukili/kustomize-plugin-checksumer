# Checksumer (Kustomize plugin)

Kustomize transformer plugin that will calculate a unique sha1 checksum of a file or a file directory.
Useful when tracking a specific version of a non Kubernetes resource represented by (an) external file(s),
such as an external app configuration server state.

### Roadmap:

* Support for IgnoredFilePrefix attribute within FileSpec

### Requirements:

* Go 1.12
* Kustomize 3.1.0 **built from source**
    ```
    go get sigs.k8s.io/kustomize/v3/cmd/kustomize@v3.1.0
    ```

### Usage:

* kustomization.yml
    ```
    apiVersion: kustomize.config.k8s.io/v1beta1
    kind: Kustomization
    resources:
      - deployment.yml
    transformers:
      - checksumer.yml
    ```

* deployment.yml
  ```
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: myapp
     labels:
       app: myapp
   spec:
     selector:
       matchLabels:
         app: myapp
     template:
       metadata:
         labels:
           app: myapp
       spec:
         containers:
           - image: myapp:somerevision
             name: myapp
    ```

* checksumer.yml
    ```
    apiVersion: github.com/oboukili
    kind: Checksumer
    metadata:
      name: myTransformer
    files:
      - key: arbitraryKey
        path: path/to/file
      - key: anotherArbitraryKey
        path: path/to/folder
        # Only applicable on folders, when true, will also take into account
        # all files present in subdirs to calculate a unique sha1 sum
        # (default to false)  
        recurse: true
    # Where the above keys will be inserted in the resulting transformed resources
    fieldSpecs:
      - path: metadata/annotations
        create: true
      - path: spec/template/metadata/annotations
        create: true
    ```

* results
    ```
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: myapp
     labels:
       app: myapp
     annotations:
       arbitraryKey: (sha1sum string)
       anotherArbitraryKey: (sha1sum string)
   spec:
     selector:
       matchLabels:
         app: myapp
     template:
       metadata:
         labels:
           app: myapp
         annotations:
           arbitraryKey: (sha1sum string)
           anotherArbitraryKey: (sha1sum string)
       spec:
         containers:
           - image: myapp:somerevision
             name: myapp
    ```

### Build

```
mkdir -p $HOME/.config/kustomize/plugin/github.com/oboukili
cd $HOME/.config/kustomize/plugin/github.com/oboukili
git clone https://github.com/oboukili/kustomize-plugin-checksumer -o checksumer
cd checksumer
go build -buildmode plugin -o Checksumer.so Checksumer.go
```

### Run

```
PLUGIN_ROOT=$HOME/.config/kustomize/plugin kustomize build --enable_alpha_plugins path/to/kustomization/folder
```


### Credits

Many thanks to the kustomize team for bringing us an awesome opensource configuration tool :)
