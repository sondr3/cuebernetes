package kube

import (
	helm "cue.dev/x/crd/fluxcd.io/helm/v2"
	source "cue.dev/x/crd/fluxcd.io/source/v1"
	core "cue.dev/x/k8s.io/api/core/v1"
)

ns: core.#Namespace & {
	metadata: {
		name: "podinfo"
		labels: {
			"toolkit.fluxcd.io/tenant": "dev-team"
		}
	}
}

repo: source.#HelmRepository & {
	metadata: {
		name:      "podinfo"
		namespace: "podinfo"
	}
	spec: {
		interval: "5m"
		url:      "https://stefanprodan.github.io/podinfo"
	}
}

chart: helm.#HelmRelease & {
	metadata: {
		name: "podinfo"
		namespace: "podinfo"
	}
	spec: {
		releaseName: "podinfo"
		chart: spec: {
			chart: "podinfo"
			sourceRef: {
				kind: "HelmRepository"
				name: "podinfo"
			}
		}
		interval: "50m"
		install: remediation: {
			retries: 3
		}
		values: {
			redis: {
				enabled: true
				repository: "public.ecr.aws/docker/library/redis"
				tag: "7.0.6"
			}
			ingress: {
				enabled: true
				className: "nginx"
			}
		}
	}
}