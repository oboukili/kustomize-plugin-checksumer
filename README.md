# Checksumer (Kustomize plugin)

Kustomize transformer plugin that will calculate a unique sha1 checksum of a file or a file directory.
Useful when tracking a specific version of a non Kubernetes resource represented by (an) external file(s),
such as an external app configuration server state.


### Roadmap:

* Better integration tests for directories (need help with KustHarness test framework)

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
    apiVersion: gitlab.com/maltcommunity
    kind: Checksumer
    metadata:
      name: myTransformer
    files:
      arbitraryKey: path/to/file
      anotherArbitraryKey: path/to/folder
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
mkdir -p $HOME/.config/kustomize/plugin/gitlab.com/maltcommunity
cd $HOME/.config/kustomize/plugin/gitlab.com/maltcommunity
git clone https://gitlab.com/maltcommunity/public/checksumer -o checksumer
cd checksumer
go build -buildmode plugin -o Checksumer.so Checksumer.go
```

### Run

```
PLUGIN_ROOT=$HOME/.config/kustomize/plugin kustomize build --enable_alpha_plugins path/to/kustomization/folder
```


### Credits

Many thanks to the kustomize team for bringing us an awesome opensource configuration tool :)
