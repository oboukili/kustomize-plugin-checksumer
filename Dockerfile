FROM golang:1.13

#RUN apk add --update --no-cache git gcc libc-dev

RUN GO111MODULE=on go get -trimpath sigs.k8s.io/kustomize/kustomize/v3@v3.5.4
RUN mkdir -p /root/.config/kustomize/plugin/github.com/oboukili
COPY . /root/.config/kustomize/plugin/github.com/oboukili/checksumer

RUN cd /root/.config/kustomize/plugin/github.com/oboukili/checksumer && \
    GO111MODULE=on go build -mod=readonly -trimpath -buildmode plugin -o Checksumer.so Checksumer.go

RUN cd /root/.config/kustomize/plugin/github.com/oboukili/checksumer && \
    cd tests/integration && \
    kustomize build --enable_alpha_plugins --load_restrictor none .
