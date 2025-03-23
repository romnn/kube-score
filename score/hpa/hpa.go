package hpa

import (
	"fmt"

	"github.com/romnn/kube-score/domain"
	"github.com/romnn/kube-score/score/checks"
	"github.com/romnn/kube-score/scorecard"
	"k8s.io/utils/ptr"
)

type Options struct {
	AllTargetableObjs []domain.BothMeta
	Namespace         string
}

func Register(allChecks *checks.Checks, options Options) {
	allChecks.RegisterHorizontalPodAutoscalerCheck(
		"HorizontalPodAutoscaler has target",
		`Makes sure that the HPA targets a valid object`,
		hpaHasTarget(options),
	)
	allChecks.RegisterHorizontalPodAutoscalerCheck(
		"HorizontalPodAutoscaler Replicas",
		`Makes sure that the HPA has multiple replicas`,
		hpaHasMultipleReplicas(options),
	)
}

func hpaHasTarget(
	options Options,
) func(hpa domain.HpaTargeter) (score scorecard.TestScore, err error) {
	verbose := false
	return func(hpa domain.HpaTargeter) (scorecard.TestScore, error) {
		targetRef := hpa.HpaTarget()
		var hasTarget bool
		for _, t := range options.AllTargetableObjs {

			hpaNamespace := hpa.GetObjectMeta().Namespace
			if hpaNamespace == "" {
				hpaNamespace = options.Namespace
			}

			namespace := t.ObjectMeta.Namespace
			if namespace == "" {
				namespace = options.Namespace
			}

			if verbose {
				fmt.Printf("hpa=%s\n", targetRef.Name)
				fmt.Printf(
					"\t apiVersion: %s == %s\n",
					targetRef.APIVersion,
					t.TypeMeta.APIVersion,
				)
				fmt.Printf("\t kind: %s == %s\n", targetRef.Kind, t.TypeMeta.Kind)
				fmt.Printf("\t name: %s == %s\n", targetRef.Name, t.ObjectMeta.Name)
				fmt.Printf("\t namespace: %s == %s\n", hpaNamespace, namespace)
			}
			if t.TypeMeta.APIVersion == targetRef.APIVersion &&
				t.TypeMeta.Kind == targetRef.Kind &&
				t.ObjectMeta.Name == targetRef.Name &&
				namespace == hpaNamespace {
				hasTarget = true
				break
			}
		}

		var score scorecard.TestScore
		if hasTarget {
			score.Grade = scorecard.GradeAllOK
		} else {
			score.Grade = scorecard.GradeCritical
			score.AddComment("", "The HPA target does not match anything", "")
		}
		return score, nil
	}
}

func hpaHasMultipleReplicas(
	options Options,
) func(hpa domain.HpaTargeter) (score scorecard.TestScore, err error) {
	return func(hpa domain.HpaTargeter) (score scorecard.TestScore, err error) {
		if ptr.Deref(hpa.MinReplicas(), 1) >= 2 {
			score.Grade = scorecard.GradeAllOK
		} else {
			score.Grade = scorecard.GradeWarning
			score.AddComment("", "HPA few replicas", "HorizontalPodAutoscalers are recommended to have at least 2 replicas to prevent unwanted downtime.")
		}
		return
	}
}
