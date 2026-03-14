package kube

import (
	kustomize "cue.dev/x/crd/fluxcd.io/kustomize/v1"
)

kustomize.#Kustomization & {
	metadata: {
		name:      "infra-configs"
		namespace: "flux-system"
	}
	spec: {
		interval:      "1h"
		retryInterval: "2m"
		timeout:       "5m"
		prune:         true
		sourceRef: {
			kind: "ExternalArtifact"
			name: "infrastructure"
		}
		path: "./configs"
		patches: [{
			patch: """
				- op: replace
					path: /spec/acme/server
					value: https://acme-staging-v02.api.letsencrypt.org/directory
				"""
			target: {
				kind: "ClusterIssuer"
				name: "letsencrypt"
			}
		}]
	}
}
