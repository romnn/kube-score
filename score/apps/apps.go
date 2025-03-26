package apps

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"

	ks "github.com/romnn/kube-score/domain"
	"github.com/romnn/kube-score/score/checks"
	"github.com/romnn/kube-score/score/internal"
	"github.com/romnn/kube-score/scorecard"
)

type Options struct {
	Namespace string
}

func Register(
	allChecks *checks.Checks,
	allHPAs []ks.HpaTargeter,
	allServices []ks.Service,
	options Options,
) {
	allChecks.RegisterDeploymentCheck(
		"Deployment has host PodAntiAffinity",
		"Makes sure that a podAntiAffinity has been set that prevents multiple pods from being scheduled on the same node. https://kubernetes.io/docs/concepts/configuration/assign-pod-node/",
		deploymentHasAntiAffinity(options),
	)
	allChecks.RegisterStatefulSetCheck(
		"StatefulSet has host PodAntiAffinity",
		"Makes sure that a podAntiAffinity has been set that prevents multiple pods from being scheduled on the same node. https://kubernetes.io/docs/concepts/configuration/assign-pod-node/",
		statefulsetHasAntiAffinity(options),
	)

	allChecks.RegisterDeploymentCheck(
		"Deployment targeted by HPA does not have replicas configured",
		"Makes sure that Deployments using a HorizontalPodAutoscaler doesn't have a statically configured replica count set",
		hpaDeploymentNoReplicas(allHPAs, options),
	)
	allChecks.RegisterStatefulSetCheck(
		"StatefulSet has ServiceName",
		"Makes sure that StatefulSets have an existing headless serviceName.",
		statefulsetHasServiceName(allServices, options),
	)

	allChecks.RegisterDeploymentCheck(
		"Deployment Pod Selector labels match template metadata labels",
		"Ensure the StatefulSet selector labels match the template metadata labels.",
		deploymentSelectorLabelsMatching(options),
	)
	allChecks.RegisterStatefulSetCheck(
		"StatefulSet Pod Selector labels match template metadata labels",
		"Ensure the StatefulSet selector labels match the template metadata labels.",
		statefulSetSelectorLabelsMatching(options),
	)
}

func hpaDeploymentNoReplicas(
	allHPAs []ks.HpaTargeter,
	options Options,
) func(deployment appsv1.Deployment) (scorecard.TestScore, error) {
	return func(deployment appsv1.Deployment) (scorecard.TestScore, error) {
		var score scorecard.TestScore
		// If is targeted by a HPA
		for _, hpa := range allHPAs {
			target := hpa.HpaTarget()

			hpaNamespace := hpa.GetObjectMeta().Namespace
			if hpaNamespace == "" {
				hpaNamespace = options.Namespace
			}

			deploymentNamespace := deployment.Namespace
			if deploymentNamespace == "" {
				deploymentNamespace = options.Namespace
			}

			if hpaNamespace == deploymentNamespace &&
				strings.EqualFold(target.Kind, deployment.Kind) &&
				target.Name == deployment.Name {

				if deployment.Spec.Replicas == nil {
					score.Grade = scorecard.GradeAllOK
					return score, nil
				}

				score.Grade = scorecard.GradeCritical
				score.AddComment(
					"",
					"The deployment is targeted by a HPA, but a static replica count is configured in the DeploymentSpec",
					"When replicas are both statically set and managed by the HPA, the replicas will be changed to the statically configured count when the spec is applied, even if the HPA wants the replica count to be higher.",
				)
				return score, nil
			}
		}

		score.Grade = scorecard.GradeAllOK
		score.Skipped = true
		score.AddComment(
			"",
			"Skipped because the deployment is not targeted by a HorizontalPodAutoscaler",
			"",
		)
		return score, nil
	}
}

func deploymentHasAntiAffinity(
	options Options,
) func(deployment appsv1.Deployment) (scorecard.TestScore, error) {
	return func(deployment appsv1.Deployment) (scorecard.TestScore, error) {
		// Ignore if the deployment only has a single replica
		// If replicas is not explicitly set, we'll still warn if the anti affinity is missing
		// as that might indicate use of a Horizontal Pod Autoscaler
		var score scorecard.TestScore
		if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas < 2 {
			score.Skipped = true
			score.AddComment(
				"",
				"Skipped because the deployment has less than 2 replicas",
				"",
			)
			return score, nil
		}

		warn := func() {
			score.Grade = scorecard.GradeWarning
			score.AddComment(
				"",
				"Deployment does not have a host podAntiAffinity set",
				"It's recommended to set a podAntiAffinity that stops multiple pods from a deployment from being scheduled on the same node. This increases availability in case the node becomes unavailable.",
			)
		}

		affinity := deployment.Spec.Template.Spec.Affinity
		if affinity == nil || affinity.PodAntiAffinity == nil {
			warn()
			return score, nil
		}

		labels := k8slabels.Set(deployment.Spec.Template.GetObjectMeta().GetLabels())
		if hasPodAntiAffinity(labels, affinity) {
			score.Grade = scorecard.GradeAllOK
			return score, nil
		}

		warn()
		return score, nil
	}
}

func statefulsetHasAntiAffinity(
	options Options,
) func(statefulset appsv1.StatefulSet) (scorecard.TestScore, error) {
	return func(statefulset appsv1.StatefulSet) (scorecard.TestScore, error) {
		// Ignore if the statefulset only has a single replica
		// If replicas is not explicitly set, we'll still warn if the anti affinity is missing
		// as that might indicate use of a Horizontal Pod Autoscaler
		var score scorecard.TestScore
		if statefulset.Spec.Replicas != nil && *statefulset.Spec.Replicas < 2 {
			score.Skipped = true
			score.AddComment(
				"",
				"Skipped because the statefulset has less than 2 replicas",
				"",
			)
			return score, nil
		}

		warn := func() {
			score.Grade = scorecard.GradeWarning
			score.AddComment(
				"",
				"StatefulSet does not have a host podAntiAffinity set",
				"It's recommended to set a podAntiAffinity that stops multiple pods from a statefulset from being scheduled on the same node. This increases availability in case the node becomes unavailable.",
			)
		}

		affinity := statefulset.Spec.Template.Spec.Affinity
		if affinity == nil || affinity.PodAntiAffinity == nil {
			warn()
			return score, nil
		}

		labels := k8slabels.Set(statefulset.Spec.Template.GetObjectMeta().GetLabels())

		if hasPodAntiAffinity(labels, affinity) {
			score.Grade = scorecard.GradeAllOK
			return score, nil
		}

		warn()
		return score, nil
	}
}

func hasPodAntiAffinity(selfLabels k8slabels.Labels, affinity *corev1.Affinity) bool {
	approvedTopologyKeys := map[string]struct{}{
		"kubernetes.io/hostname":        {},
		"topology.kubernetes.io/region": {},
		"topology.kubernetes.io/zone":   {},

		// Deprecated in Kubernetes v1.17
		"failure-domain.beta.kubernetes.io/region": {},
		"failure-domain.beta.kubernetes.io/zone":   {},
	}

	for _, pref := range affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
		if _, ok := approvedTopologyKeys[pref.PodAffinityTerm.TopologyKey]; ok {
			if selector, err := metav1.LabelSelectorAsSelector(pref.PodAffinityTerm.LabelSelector); err == nil {
				if selector.Matches(selfLabels) {
					return true
				}
			}
		}
	}

	for _, req := range affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
		if _, ok := approvedTopologyKeys[req.TopologyKey]; ok {
			if selector, err := metav1.LabelSelectorAsSelector(req.LabelSelector); err == nil {
				if selector.Matches(selfLabels) {
					return true
				}
			}
		}
	}

	return false
}

func statefulsetHasServiceName(
	allServices []ks.Service,
	options Options,
) func(statefulset appsv1.StatefulSet) (scorecard.TestScore, error) {
	verbose := false
	return func(statefulset appsv1.StatefulSet) (scorecard.TestScore, error) {
		var score scorecard.TestScore
		for _, service := range allServices {
			svc := service.Service()
			serviceNamespace := svc.Namespace
			if serviceNamespace == "" {
				serviceNamespace = options.Namespace
			}

			sfsNamespace := statefulset.Namespace
			if sfsNamespace == "" {
				sfsNamespace = options.Namespace
			}

			labels := statefulset.Spec.Template.GetObjectMeta().GetLabels()

			if verbose {
				fmt.Printf("service %q\n", svc.Name)
				fmt.Printf("\t name: %q == %q\n", svc.Name, statefulset.Spec.ServiceName)
				fmt.Printf("\t clusterIP: %q\n", svc.Spec.ClusterIP)
				fmt.Printf("\t selector: %+q\n", svc.Spec.Selector)
				fmt.Printf("\t labels: %+q\n", labels)
			}

			if serviceNamespace != sfsNamespace ||
				svc.Name != statefulset.Spec.ServiceName ||
				svc.Spec.ClusterIP != "None" {
				continue
			}

			if verbose {
				fmt.Printf("\t match: %t\n", internal.LabelSelectorMatchesLabels(
					svc.Spec.Selector,
					labels,
				))
			}

			if internal.LabelSelectorMatchesLabels(
				svc.Spec.Selector,
				labels,
			) {
				score.Grade = scorecard.GradeAllOK
				return score, nil
			}
		}

		score.Grade = scorecard.GradeCritical
		score.AddComment(
			"",
			"StatefulSet does not have a valid serviceName",
			"StatefulSets currently require a Headless Service to be responsible for the network identity of the Pods. You are responsible for creating this Service. https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#limitations",
		)
		return score, nil
	}
}

func statefulSetSelectorLabelsMatching(
	opions Options,
) func(statefulset appsv1.StatefulSet) (scorecard.TestScore, error) {
	return func(statefulset appsv1.StatefulSet) (scorecard.TestScore, error) {
		var score scorecard.TestScore
		selector, err := metav1.LabelSelectorAsSelector(statefulset.Spec.Selector)
		if err != nil {
			score.Grade = scorecard.GradeCritical
			score.AddComment(
				"",
				"StatefulSet selector labels are not matching template metadata labels",
				fmt.Sprintf("Invalid selector: %s", err),
			)
			return score, err
		}

		labels := k8slabels.Set(statefulset.Spec.Template.GetObjectMeta().GetLabels())
		if selector.Matches(labels) {
			score.Grade = scorecard.GradeAllOK
			return score, nil
		}

		score.Grade = scorecard.GradeCritical
		score.AddComment(
			"",
			"StatefulSet selector labels not matching template metadata labels",
			"StatefulSets require `.spec.selector` to match `.spec.template.metadata.labels`. https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#pod-selector",
		)
		return score, nil
	}
}

func deploymentSelectorLabelsMatching(
	options Options,
) func(deployment appsv1.Deployment) (scorecard.TestScore, error) {
	return func(deployment appsv1.Deployment) (scorecard.TestScore, error) {
		var score scorecard.TestScore
		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
		if err != nil {
			score.Grade = scorecard.GradeCritical
			score.AddComment(
				"",
				"Deployment selector labels are not matching template metadata labels",
				fmt.Sprintf("Invalid selector: %s", err),
			)
			return score, err
		}

		labels := k8slabels.Set(deployment.Spec.Template.GetObjectMeta().GetLabels())
		if selector.Matches(labels) {
			score.Grade = scorecard.GradeAllOK
			return score, nil
		}

		score.Grade = scorecard.GradeCritical
		score.AddComment(
			"",
			"Deployment selector labels not matching template metadata labels",
			"Deployment require `.spec.selector` to match `.spec.template.metadata.labels`. https://kubernetes.io/docs/concepts/workloads/controllers/deployment/",
		)
		return score, nil
	}
}
