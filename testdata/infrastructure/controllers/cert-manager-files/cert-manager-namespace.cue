package kube

import (
	core "cue.dev/x/k8s.io/api/core/v1"
)

core.#Namespace & {
	metadata: {
		name: "cert-manager-files"
		labels: {
			"toolkit.fluxcd.io/tenant": "sre-team"
		}
	}
}
