package kube

import (
	helm "cue.dev/x/crd/fluxcd.io/helm/v2"
)

helm.#HelmRelease & {
	metadata: {
		name:      "cert-manager-files"
		namespace: "cert-manager-files"
	}
	spec: {
		interval: "12h"
		install: {
			strategy: {
				name:          "RetryOnFailure"
				retryInterval: "2m"
			}
		}
		upgrade: {
			strategy: {
				name:          "RetryOnFailure"
				retryInterval: "3m"
			}
		}
		chartRef: {
			kind: "OCIRepository"
			name: "cert-manager-files"
		}
		values: {
			crds: {
				enabled: true
				keep:    false
			}
		}
	}
}
