PROJECT_DIR="$( pwd )"
KUSTOMIZE_DIR="$(pwd)/kustomize"
DEST_DIR=$HOME/.config/kustomize

cd $KUSTOMIZE_DIR
find . -iname '*.go' -exec sh -c 'f="{}"; GO111MODULE=on go build -buildmode plugin -o "${f/.go/.so}" $f' \;
if [ ! -z "${XDG_CONFIG_HOME}" ]; then
DEST_DIR=$XDG_CONFIG_HOME
fi
find . -name '*.so' | cpio -pdm $DEST_DIR
