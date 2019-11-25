FROM golang:1.12-alpine

RUN apk add --update --no-cache git gcc libc-dev

RUN GO111MODULE=on go get sigs.k8s.io/kustomize/v3/cmd/kustomize@v3.1.0
RUN mkdir -p /root/.config/kustomize/plugin/github.com/oboukili
COPY . /root/.config/kustomize/plugin/github.com/oboukili/checksumer

RUN cd /root/.config/kustomize/plugin/github.com/oboukili/checksumer && \
    go test

RUN cd /root/.config/kustomize/plugin/github.com/oboukili/checksumer && \
    GO111MODULE=on go build -buildmode plugin -o Checksumer.so Checksumer.go

RUN cd /root/.config/kustomize/plugin/github.com/oboukili/checksumer && \
    cd tests/integration && \
    kustomize build --enable_alpha_plugins --load_restrictor none .
