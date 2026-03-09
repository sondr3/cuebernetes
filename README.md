<h1 align="center">cuebernetes</h1>
<p align="center">
    <a href="https://github.com/sondr3/cuebernetes/actions"><img alt="GitHub Actions Status" src="https://github.com/sondr3/cuebernetes/workflows/pipeline/badge.svg" /></a>
</p>

<p align="center">
    <b>A converter for Kubernetes files in Cue to YAML</b>
</p>

> [!NOTE]
> This is in many ways a fork, or at the very least based on [cke-cue-kubernetes-resource-exporter](https://github.com/Kystverket/cke-cue-kubernetes-resource-exporter)

# Quickstart

```sh
git clone github.com/sondr3/cuebernetes
cd cuebernetes
go install .
```

## Usage

You can either render directly to YAML (the default) or write the files back
out to a output directory (defaults to `_yaml`).

### Example data

You can either use exported fields or just a naked export. See the following
examples. One has exported named fields and one is just a single exported
namespace.

```cue
package kube

import (
	helm "cue.dev/x/crd/fluxcd.io/helm/v2"
	source "cue.dev/x/crd/fluxcd.io/source/v1"
	core "cue.dev/x/k8s.io/api/core/v1"
)

ns: core.#Namespace & {
    ...
}

repo: source.#HelmRepository & {
    ...
}
```

```cue
package kube

import (
	core "cue.dev/x/k8s.io/api/core/v1"
)

core.#Namespace & {
    ...
}
```



### Straight out

Just point it at a file (or directory), and it'll print all the Cue files that are
Kubernetes manifests to `stdout`.

```shell
$ cuebernetes testdata/apps/podinfo.cue
# generated from testdata/apps/podinfo.cue -- DO NOT EDIT
apiVersion: v1
kind: Namespace
...
```

### Written out

Point it at a file or directory, and it'll write those Cue manifests to the
output directory as a 1-to-1 mapping, essentially just changing the `.cue` to
`.yaml`.

```shell
$ cuebernetes -m write testdata/apps/podinfo.cue
Wrote 1 file(s)
```

If you want to split the exported fields in the Cue files, you can use `--split`
to write each file to its own file instead of a multi-document one.

# License

Apache.
