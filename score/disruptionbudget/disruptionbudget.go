package disruptionbudget

import (
	"fmt"

	ks "github.com/romnn/kube-score/domain"
	"github.com/romnn/kube-score/score/checks"
	"github.com/romnn/kube-score/scorecard"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
)

type Options struct {
	Namespace string
}

func Register(
	allChecks *checks.Checks,
	budgets ks.PodDisruptionBudgets,
	options Options,
) {
	allChecks.RegisterStatefulSetCheck(
		"StatefulSet has PodDisruptionBudget",
		`Makes sure that all StatefulSets are targeted by a PDB`,
		statefulSetHas(budgets.PodDisruptionBudgets(), options),
	)
	allChecks.RegisterDeploymentCheck(
		"Deployment has PodDisruptionBudget",
		`Makes sure that all Deployments are targeted by a PDB`,
		deploymentHas(budgets.PodDisruptionBudgets(), options),
	)
	allChecks.RegisterPodDisruptionBudgetCheck(
		"PodDisruptionBudget has policy",
		`Makes sure that PodDisruptionBudgets specify minAvailable or maxUnavailable`,
		hasPolicy,
	)
}

func hasMatching(
	budgets []ks.PodDisruptionBudget,
	namespace string,
	labels map[string]string,
	options Options,
) (bool, string, error) {
	verbose := false
	var hasNamespaceMismatch []string

	if namespace == "" {
		namespace = options.Namespace
	}

	for _, budget := range budgets {
		selector, err := metav1.LabelSelectorAsSelector(
			budget.PodDisruptionBudgetSelector(),
		)
		if err != nil {
			return false, "", fmt.Errorf("failed to create selector: %w", err)
		}

		budgetNamespace := budget.Namespace()
		if budgetNamespace == "" {
			budgetNamespace = options.Namespace
		}

		// var requirements []k8slabels.Requirement
		// for k, v := range labels {
		// 	req, err := k8slabels.NewRequirement(k, k8sselection.Equals, []string{v})
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	requirements = append(requirements, *req)
		// }
		// test := k8slabels.NewSelector().Add(requirements...)

		if verbose {
			fmt.Printf("selector = %+v\n", selector)
			fmt.Printf("labels = %+v\n", k8slabels.Set(labels))
			fmt.Printf(
				"\t pdbNamespace = %q namespace=%q\n",
				budgetNamespace,
				namespace,
			)
			fmt.Printf("\t match = %t\n", selector.Matches(k8slabels.Set(labels)))
		}
		if !selector.Matches(k8slabels.Set(labels)) {
			continue
		}

		// matches, but in different namespace
		if budgetNamespace != namespace {
			hasNamespaceMismatch = append(hasNamespaceMismatch, budgetNamespace)
			continue
		}

		return true, "", nil
	}

	if len(hasNamespaceMismatch) > 0 {
		return false, fmt.Sprintf(
			"A matching budget was found, but in a different namespace. expected='%s' got='%+v'",
			namespace,
			hasNamespaceMismatch,
		), nil
	}

	return false, "", nil
}

func statefulSetHas(
	budgets []ks.PodDisruptionBudget,
	options Options,
) func(appsv1.StatefulSet) (scorecard.TestScore, error) {
	return func(statefulset appsv1.StatefulSet) (score scorecard.TestScore, err error) {
		if statefulset.Spec.Replicas != nil && *statefulset.Spec.Replicas < 2 {
			score.Skipped = true
			score.AddComment(
				"",
				"Skipped because the statefulset has less than 2 replicas",
				"",
			)
			return
		}

		match, comment, matchErr := hasMatching(
			budgets,
			statefulset.Namespace,
			statefulset.Spec.Template.Labels,
			options,
		)
		if matchErr != nil {
			err = matchErr
			return
		}

		if match {
			score.Grade = scorecard.GradeAllOK
		} else {
			score.Grade = scorecard.GradeCritical
			score.AddComment("", "No matching PodDisruptionBudget was found", "It's recommended to define a PodDisruptionBudget to avoid unexpected downtime during Kubernetes maintenance operations, such as when draining a node. "+comment)
		}

		return
	}
}

func deploymentHas(
	budgets []ks.PodDisruptionBudget,
	options Options,
) func(appsv1.Deployment) (scorecard.TestScore, error) {
	return func(deployment appsv1.Deployment) (score scorecard.TestScore, err error) {
		if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas < 2 {
			score.Skipped = true
			score.AddComment(
				"",
				"Skipped because the deployment has less than 2 replicas",
				"",
			)
			return
		}

		match, comment, matchErr := hasMatching(
			budgets,
			deployment.Namespace,
			deployment.Spec.Template.Labels,
			options,
		)
		if matchErr != nil {
			err = matchErr
			return
		}

		if match {
			score.Grade = scorecard.GradeAllOK
		} else {
			score.Grade = scorecard.GradeCritical
			score.AddComment("", "No matching PodDisruptionBudget was found", "It's recommended to define a PodDisruptionBudget to avoid unexpected downtime during Kubernetes maintenance operations, such as when draining a node. "+comment)
		}

		return
	}
}

func hasPolicy(pdb ks.PodDisruptionBudget) (score scorecard.TestScore, err error) {
	spec := pdb.Spec()
	if spec.MinAvailable == nil && spec.MaxUnavailable == nil {
		score.AddComment(
			"",
			"PodDisruptionBudget missing policy",
			"PodDisruptionBudget should specify minAvailable or maxUnavailable.",
		)
		score.Grade = scorecard.GradeCritical
	} else {
		score.Grade = scorecard.GradeAllOK
	}

	return
}
