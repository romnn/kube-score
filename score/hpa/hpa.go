package hpa

import (
	"github.com/zegl/kube-score/domain"
	"github.com/zegl/kube-score/score/checks"
	"github.com/zegl/kube-score/scorecard"
	"k8s.io/utils/ptr"
)

type Options struct {
	AllTargetableObjs []domain.BothMeta
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

func hpaHasTarget(options Options) func(hpa domain.HpaTargeter) (score scorecard.TestScore, err error) {
	return func(hpa domain.HpaTargeter) (score scorecard.TestScore, err error) {
		targetRef := hpa.HpaTarget()
		var hasTarget bool
		for _, t := range options.AllTargetableObjs {
			if t.TypeMeta.APIVersion == targetRef.APIVersion &&
				t.TypeMeta.Kind == targetRef.Kind &&
				t.ObjectMeta.Name == targetRef.Name &&
				t.ObjectMeta.Namespace == hpa.GetObjectMeta().Namespace {
				hasTarget = true
				break
			}
		}

		if hasTarget {
			score.Grade = scorecard.GradeAllOK
		} else {
			score.Grade = scorecard.GradeCritical
			score.AddComment("", "The HPA target does not match anything", "")
		}
		return
	}
}

func hpaHasMultipleReplicas(options Options) func(hpa domain.HpaTargeter) (score scorecard.TestScore, err error) {
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
