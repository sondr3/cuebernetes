package kube

import (
//	"tool/cli"
//	"encoding/yaml"
//	helm "cue.dev/x/crd/fluxcd.io/helm/v2"
//	source "cue.dev/x/crd/fluxcd.io/source/v1"
	core "cue.dev/x/k8s.io/api/core/v1"
)

core.#Namespace & {
	metadata: {
		name: "cert-manager"
		labels: {
			"toolkit.fluxcd.io/tenant": "sre-team"
		}
	}
}
