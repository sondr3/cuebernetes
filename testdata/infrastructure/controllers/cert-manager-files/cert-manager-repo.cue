package kube

import (
	source "cue.dev/x/crd/fluxcd.io/source/v1"
)

source.#OCIRepository & {
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
