package service

import (
	corev1 "k8s.io/api/core/v1"

	ks "github.com/romnn/kube-score/domain"
	"github.com/romnn/kube-score/score/checks"
	"github.com/romnn/kube-score/score/internal"
	"github.com/romnn/kube-score/scorecard"
)

type Options struct {
	Namespace string
}

func Register(allChecks *checks.Checks, pods ks.Pods, podspeccers ks.PodSpeccers, options Options) {
	allChecks.RegisterServiceCheck(
		"Service Targets Pod",
		`Makes sure that all Services targets a Pod`,
		serviceTargetsPod(pods.Pods(), podspeccers.PodSpeccers(), options),
	)
	allChecks.RegisterServiceCheck(
		"Service Type",
		`Makes sure that the Service type is not NodePort`,
		serviceType(options),
	)
}

// serviceTargetsPod checks if a Service targets a pod and issues a critical warning if no matching pod
// could be found
func serviceTargetsPod(
	pods []ks.Pod,
	podspecers []ks.PodSpecer,
	options Options,
) func(corev1.Service) (scorecard.TestScore, error) {
	podsInNamespace := make(map[string][]map[string]string)
	for _, p := range pods {
		pod := p.Pod()
		namespace := pod.Namespace
		if namespace == "" {
			namespace = options.Namespace
		}
		if _, ok := podsInNamespace[namespace]; !ok {
			podsInNamespace[namespace] = []map[string]string{}
		}
		podsInNamespace[namespace] = append(
			podsInNamespace[namespace],
			pod.Labels,
		)
	}
	for _, podSpec := range podspecers {
		podNamespace := podSpec.GetObjectMeta().Namespace
		if podNamespace == "" {
			podNamespace = options.Namespace
		}

		if _, ok := podsInNamespace[podNamespace]; !ok {
			podsInNamespace[podNamespace] = []map[string]string{}
		}
		podsInNamespace[podNamespace] = append(
			podsInNamespace[podNamespace],
			podSpec.GetPodTemplateSpec().Labels,
		)
	}

	return func(service corev1.Service) (scorecard.TestScore, error) {
		// Services of type ExternalName does not have a selector
		var score scorecard.TestScore
		if service.Spec.Type == corev1.ServiceTypeExternalName {
			score.Grade = scorecard.GradeAllOK
			return score, nil
		}

		hasMatch := false

		serviceNamespace := service.Namespace
		if serviceNamespace == "" {
			serviceNamespace = options.Namespace
		}

		for _, podLabels := range podsInNamespace[serviceNamespace] {
			if internal.LabelSelectorMatchesLabels(service.Spec.Selector, podLabels) {
				hasMatch = true
				break
			}
		}

		if hasMatch {
			score.Grade = scorecard.GradeAllOK
		} else {
			score.Grade = scorecard.GradeCritical
			score.AddComment("", "The services selector does not match any pods", "")
		}

		return score, nil
	}
}

func serviceType(options Options) func(service corev1.Service) (scorecard.TestScore, error) {
	return func(service corev1.Service) (scorecard.TestScore, error) {
		var score scorecard.TestScore
		if service.Spec.Type == corev1.ServiceTypeNodePort {
			score.Grade = scorecard.GradeWarning
			score.AddComment(
				"",
				"The service is of type NodePort",
				"NodePort services should be avoided as they are insecure, and can not be used together with NetworkPolicies. LoadBalancers or use of an Ingress is recommended over NodePorts.",
			)
			return score, nil
		}

		score.Grade = scorecard.GradeAllOK
		return score, nil
	}
}
