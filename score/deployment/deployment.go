package deployment

import (
	ks "github.com/romnn/kube-score/domain"
	"github.com/romnn/kube-score/score/checks"
	"github.com/romnn/kube-score/score/internal"
	"github.com/romnn/kube-score/scorecard"
	v1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/utils/ptr"
)

type Options struct {
	Namespace string
}

func Register(allChecks *checks.Checks, all ks.AllTypes, options Options) {
	allChecks.RegisterDeploymentCheck(
		"Deployment Strategy",
		`Makes sure that all Deployments targeted by service use RollingUpdate strategy`,
		deploymentRolloutStrategy(all.Services(), options),
	)
	allChecks.RegisterDeploymentCheck(
		"Deployment Replicas",
		`Makes sure that Deployment has multiple replicas`,
		deploymentReplicas(all.Services(), all.HorizontalPodAutoscalers(), options),
	)
}

// deploymentRolloutStrategy checks if a Deployment has the update strategy on RollingUpdate if targeted by a service
func deploymentRolloutStrategy(
	svcs []ks.Service,
	options Options,
) func(deployment v1.Deployment) (scorecard.TestScore, error) {
	svcsInNamespace := make(map[string][]map[string]string)
	for _, s := range svcs {
		svc := s.Service()
		namespace := svc.Namespace
		if namespace == "" {
			namespace = options.Namespace
		}

		if _, ok := svcsInNamespace[namespace]; !ok {
			svcsInNamespace[namespace] = []map[string]string{}
		}
		svcsInNamespace[namespace] = append(
			svcsInNamespace[namespace],
			svc.Spec.Selector,
		)
	}

	return func(deployment v1.Deployment) (score scorecard.TestScore, err error) {
		referencedByService := false

		deploymentNamespace := deployment.Namespace
		if deploymentNamespace == "" {
			deploymentNamespace = options.Namespace
		}

		for _, svcSelector := range svcsInNamespace[deploymentNamespace] {
			if internal.LabelSelectorMatchesLabels(
				svcSelector,
				deployment.Spec.Template.Labels,
			) {
				referencedByService = true
				break
			}
		}

		if referencedByService {
			if deployment.Spec.Strategy.Type == v1.RollingUpdateDeploymentStrategyType ||
				deployment.Spec.Strategy.Type == "" {
				score.Grade = scorecard.GradeAllOK
			} else {
				score.Grade = scorecard.GradeWarning
				score.AddCommentWithURL("", "Deployment update strategy", "The deployment is used by a service but not using the RollingUpdate strategy which can cause interruptions. Set .spec.strategy.type to RollingUpdate.", "https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy")
			}
		} else {
			score.Skipped = true
			score.AddComment("", "Skipped as the Deployment is not targeted by a service", "")
		}

		return
	}
}

// deploymentReplicas checks if a Deployment has >= 2 replicas if not (targeted by service || has HorizontalPodAutoscaler)
func deploymentReplicas(
	svcs []ks.Service,
	hpas []ks.HpaTargeter,
	options Options,
) func(deployment v1.Deployment) (scorecard.TestScore, error) {
	svcsInNamespace := make(map[string][]map[string]string)
	for _, s := range svcs {
		svc := s.Service()
		namespace := svc.Namespace
		if namespace == "" {
			namespace = options.Namespace
		}
		if _, ok := svcsInNamespace[namespace]; !ok {
			svcsInNamespace[namespace] = []map[string]string{}
		}
		svcsInNamespace[namespace] = append(
			svcsInNamespace[namespace],
			svc.Spec.Selector,
		)
	}

	hpasInNamespace := make(map[string][]autoscalingv1.CrossVersionObjectReference)
	for _, hpa := range hpas {
		hpaTarget := hpa.HpaTarget()
		hpaMeta := hpa.GetObjectMeta()

		hpaNamespace := hpaMeta.Namespace
		if hpaNamespace == "" {
			hpaNamespace = options.Namespace
		}

		if _, ok := hpasInNamespace[hpaNamespace]; !ok {
			hpasInNamespace[hpaNamespace] = []autoscalingv1.CrossVersionObjectReference{}
		}
		hpasInNamespace[hpaNamespace] = append(
			hpasInNamespace[hpaNamespace],
			hpaTarget,
		)
	}

	return func(deployment v1.Deployment) (score scorecard.TestScore, err error) {
		referencedByService := false
		hasHPA := false

		deploymentNamespace := deployment.Namespace
		if deploymentNamespace == "" {
			deploymentNamespace = options.Namespace
		}

		for _, svcSelector := range svcsInNamespace[deploymentNamespace] {
			if internal.LabelSelectorMatchesLabels(
				svcSelector,
				deployment.Spec.Template.Labels,
			) {
				referencedByService = true
				break
			}
		}

		for _, hpaTarget := range hpasInNamespace[deploymentNamespace] {
			if deployment.APIVersion == hpaTarget.APIVersion &&
				deployment.Kind == hpaTarget.Kind &&
				deployment.Name == hpaTarget.Name {
				hasHPA = true
				break
			}
		}

		switch {
		case !referencedByService:
			score.Skipped = true
			score.AddComment(
				"",
				"Skipped as the Deployment is not targeted by service",
				"",
			)
		case hasHPA:
			score.Skipped = true
			score.AddComment(
				"",
				"Skipped as the Deployment is controlled by a HorizontalPodAutoscaler",
				"",
			)
		default:
			if ptr.Deref(deployment.Spec.Replicas, 1) >= 2 {
				score.Grade = scorecard.GradeAllOK
			} else {
				score.Grade = scorecard.GradeWarning
				score.AddComment("", "Deployment few replicas", "Deployments targeted by Services are recommended to have at least 2 replicas to prevent unwanted downtime.")
			}
		}

		return
	}
}
