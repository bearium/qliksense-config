KUSTOMIZE_VERSION := v3.1.0

install:
	GO111MODULE=on go get sigs.k8s.io/kustomize/v3/cmd/kustomize@${KUSTOMIZE_VERSION}
	./scripts/build.sh
