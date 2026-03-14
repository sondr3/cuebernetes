package kube

import (
	helm "cue.dev/x/crd/fluxcd.io/helm/v2"
	source "cue.dev/x/crd/fluxcd.io/source/v1"
	core "cue.dev/x/k8s.io/api/core/v1"
)

ns: core.#Namespace & {
	metadata: {
		name: "cert-manager-files"
		labels: {
			"toolkit.fluxcd.io/tenant": "sre-team"
		}
	}
}

repo: source.#OCIRepository & {
	metadata: {
		name:      "cert-manager-files"
		namespace: "cert-manager-files"
	}
	spec: {
		interval: "24h"
		url:      "oci://quay.io/jetstack/charts/cert-manager-files"
		layerSelector: {
			mediaType: "application/vnd.cncf.helm.chart.content.v1.tar+gzip"
			operation: "copy"
		}
		ref: {
			semver: "1.x"
		}
	}
}

chart: helm.#HelmRelease & {
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
