Argo Image [![CircleCI](https://circleci.com/gh/qlik-oss/qliksense.svg?style=svg)](https://circleci.com/gh/qlik-oss/qliksense)

# Qlik Sense Enterprise on Kubernetes

## Installation of kustomize

THe version of kustomize being using is v3.x , can be installed with Homebrew:

 `brew install kustomize`

You will also need (the go version of) yq installed, available from here (https://mikefarah.github.io/yq/])
 
 `brew install yq`

Finally, you will need helm,

 `brew install kubernetes-helm`
 
 
Create a convienience function that points to the root of this cloned repo:

```
function kustomizeIt {
  XDG_CONFIG_HOME="<Parent directory of this locally cloned repo>"
  kustomize build --enable_alpha_plugins $1
}
```

Then you can run:

`kustomizeIt .`

at almost level in this repo with a kustomization.yaml file and get some kind of YAML resources output but if you looking for a complete "all-in-one" `qlikense for Docker Desktop` configuration, navigate to the parent directory of this locally cloned repo and execute (may take 1/2 a minute or s0):

`kustomizeIt configs/qseok_devmode | kubectl apply -f -`

If you only wish to update one component, you can subsequently execute (ex. chronos):

`kustomizeIt configs/qseok_devmode/chronos | kubectl apply --prune -l app=chronos -f -`

If you wish to change a component configuration (ex "QCS-style", chronos includes own redis) you can execute:

`kustomizeIt configs/qseok_ent_devmode/chronos | kubectl apply --prune -l app=chronos -f -`
