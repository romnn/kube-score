package score

import (
	"errors"

	"github.com/romnn/kube-score/config"
	ks "github.com/romnn/kube-score/domain"
	"github.com/romnn/kube-score/score/apps"
	"github.com/romnn/kube-score/score/checks"
	"github.com/romnn/kube-score/score/container"
	"github.com/romnn/kube-score/score/cronjob"
	"github.com/romnn/kube-score/score/deployment"
	"github.com/romnn/kube-score/score/disruptionbudget"
	"github.com/romnn/kube-score/score/hpa"
	"github.com/romnn/kube-score/score/ingress"
	"github.com/romnn/kube-score/score/meta"
	"github.com/romnn/kube-score/score/networkpolicy"
	"github.com/romnn/kube-score/score/podtopologyspreadconstraints"
	"github.com/romnn/kube-score/score/probes"
	"github.com/romnn/kube-score/score/security"
	"github.com/romnn/kube-score/score/service"
	"github.com/romnn/kube-score/score/stable"
	"github.com/romnn/kube-score/scorecard"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func RegisterAllChecks(
	allObjects ks.AllTypes,
	checksConfig *checks.Config,
	runConfig *config.RunConfiguration,
) *checks.Checks {
	allChecks := checks.New(checksConfig)

	deployment.Register(allChecks, allObjects, deployment.Options{Namespace: runConfig.Namespace})
	ingress.Register(allChecks, allObjects, ingress.Options{Namespace: runConfig.Namespace})
	cronjob.Register(allChecks)
	container.Register(allChecks, container.Options{
		SkipInitContainers:                    runConfig.SkipInitContainers,
		IgnoreContainerCpuLimitRequirement:    runConfig.IgnoreContainerCpuLimitRequirement,
		IgnoreContainerMemoryLimitRequirement: runConfig.IgnoreContainerMemoryLimitRequirement,
	})
	disruptionbudget.Register(allChecks, allObjects, disruptionbudget.Options{
		Namespace: runConfig.Namespace,
	})
	networkpolicy.Register(
		allChecks,
		allObjects,
		allObjects,
		allObjects,
		networkpolicy.Options{
			Namespace: runConfig.Namespace,
		},
	)
	probes.Register(allChecks, allObjects, probes.Options{
		SkipInitContainers: runConfig.SkipInitContainers,
		Namespace:          runConfig.Namespace,
	})
	security.Register(allChecks, security.Options{
		SkipInitContainers: runConfig.SkipInitContainers,
	})
	service.Register(allChecks, allObjects, allObjects, service.Options{Namespace: runConfig.Namespace})
	stable.Register(runConfig.KubernetesVersion, allChecks)
	apps.Register(
		allChecks,
		allObjects.HorizontalPodAutoscalers(),
		allObjects.Services(),
		apps.Options{
			Namespace: runConfig.Namespace,
		},
	)
	meta.Register(allChecks)
	hpa.Register(allChecks, hpa.Options{
		AllTargetableObjs: allObjects.Metas(),
		Namespace:         runConfig.Namespace,
	})
	podtopologyspreadconstraints.Register(allChecks)

	return allChecks
}

type podSpeccer struct {
	typeMeta   metav1.TypeMeta
	objectMeta metav1.ObjectMeta
	spec       corev1.PodTemplateSpec
}

func (p *podSpeccer) GetTypeMeta() metav1.TypeMeta {
	return p.typeMeta
}

func (p *podSpeccer) GetObjectMeta() metav1.ObjectMeta {
	return p.objectMeta
}

func (p *podSpeccer) GetPodTemplateSpec() corev1.PodTemplateSpec {
	return p.spec
}

func (p *podSpeccer) FileLocation() ks.FileLocation {
	return ks.FileLocation{}
}

// Score runs a pre-configured list of tests against the files defined in the configuration, and returns a scorecard.
// Additional configuration and tuning parameters can be provided via the config.
func Score(
	allObjects ks.AllTypes,
	allChecks *checks.Checks,
	runConfig *config.RunConfiguration,
) (*scorecard.Scorecard, error) {
	if runConfig == nil {
		runConfig = &config.RunConfiguration{}
	}

	if allChecks == nil {
		return nil, errors.New("no checks registered")
	}

	scoreCard := scorecard.New()

	newObject := func(typeMeta metav1.TypeMeta, objectMeta metav1.ObjectMeta) *scorecard.ScoredObject {
		return scoreCard.NewObject(typeMeta, objectMeta, runConfig)
	}

	for _, ingress := range allObjects.Ingresses() {
		o := newObject(ingress.GetTypeMeta(), ingress.GetObjectMeta())
		for _, test := range allChecks.Ingresses() {
			fn, err := test.Fn(ingress)
			if err != nil {
				return nil, err
			}
			o.Add(fn, test.Check, ingress, ingress.GetObjectMeta().Annotations)
		}
	}

	for _, meta := range allObjects.Metas() {
		o := newObject(meta.TypeMeta, meta.ObjectMeta)
		for _, test := range allChecks.Metas() {
			fn, err := test.Fn(meta)
			if err != nil {
				return nil, err
			}
			o.Add(fn, test.Check, meta, meta.ObjectMeta.Annotations)
		}
	}

	for _, pod := range allObjects.Pods() {
		o := newObject(pod.Pod().TypeMeta, pod.Pod().ObjectMeta)
		for _, test := range allChecks.Pods() {
			podTemplateSpec := corev1.PodTemplateSpec{
				ObjectMeta: pod.Pod().ObjectMeta,
				Spec:       pod.Pod().Spec,
			}

			score, _ := test.Fn(&podSpeccer{
				typeMeta:   pod.Pod().TypeMeta,
				objectMeta: pod.Pod().ObjectMeta,
				spec:       podTemplateSpec,
			})
			o.Add(score, test.Check, pod, pod.Pod().Annotations)
		}
	}

	for _, podspecer := range allObjects.PodSpeccers() {
		if podspecer.GetTypeMeta().Kind == "Job" && runConfig.SkipJobs {
			continue
		}
		o := newObject(podspecer.GetTypeMeta(), podspecer.GetObjectMeta())
		for _, test := range allChecks.Pods() {
			score, _ := test.Fn(podspecer)
			o.Add(score, test.Check, podspecer,
				podspecer.GetObjectMeta().Annotations,
				podspecer.GetPodTemplateSpec().Annotations,
			)
		}
	}

	for _, service := range allObjects.Services() {
		o := newObject(service.Service().TypeMeta, service.Service().ObjectMeta)
		for _, test := range allChecks.Services() {
			fn, err := test.Fn(service.Service())
			if err != nil {
				return nil, err
			}
			o.Add(fn, test.Check, service, service.Service().Annotations)
		}
	}

	for _, statefulset := range allObjects.StatefulSets() {
		o := newObject(
			statefulset.StatefulSet().TypeMeta,
			statefulset.StatefulSet().ObjectMeta,
		)
		for _, test := range allChecks.StatefulSets() {
			fn, err := test.Fn(statefulset.StatefulSet())
			if err != nil {
				return nil, err
			}
			o.Add(
				fn,
				test.Check,
				statefulset,
				statefulset.StatefulSet().Annotations,
			)
		}
	}

	for _, deployment := range allObjects.Deployments() {
		o := newObject(
			deployment.Deployment().TypeMeta,
			deployment.Deployment().ObjectMeta,
		)
		for _, test := range allChecks.Deployments() {
			res, err := test.Fn(deployment.Deployment())
			if err != nil {
				return nil, err
			}
			o.Add(
				res,
				test.Check,
				deployment,
				deployment.Deployment().Annotations,
			)
		}
	}

	for _, netpol := range allObjects.NetworkPolicies() {
		o := newObject(
			netpol.NetworkPolicy().TypeMeta,
			netpol.NetworkPolicy().ObjectMeta,
		)
		for _, test := range allChecks.NetworkPolicies() {
			fn, err := test.Fn(netpol.NetworkPolicy())
			if err != nil {
				return nil, err
			}
			o.Add(fn, test.Check, netpol, netpol.NetworkPolicy().Annotations)
		}
	}

	for _, cjob := range allObjects.CronJobs() {
		if runConfig.SkipJobs {
			continue
		}
		o := newObject(cjob.GetTypeMeta(), cjob.GetObjectMeta())
		for _, test := range allChecks.CronJobs() {
			fn, err := test.Fn(cjob)
			if err != nil {
				return nil, err
			}
			o.Add(fn, test.Check, cjob, cjob.GetObjectMeta().Annotations)
		}
	}

	for _, hpa := range allObjects.HorizontalPodAutoscalers() {
		o := newObject(hpa.GetTypeMeta(), hpa.GetObjectMeta())
		for _, test := range allChecks.HorizontalPodAutoscalers() {
			fn, err := test.Fn(hpa)
			if err != nil {
				return nil, err
			}
			o.Add(fn, test.Check, hpa, hpa.GetObjectMeta().Annotations)
		}
	}

	for _, pdb := range allObjects.PodDisruptionBudgets() {
		o := newObject(pdb.GetTypeMeta(), pdb.GetObjectMeta())
		for _, test := range allChecks.PodDisruptionBudgets() {
			fn, err := test.Fn(pdb)
			if err != nil {
				return nil, err
			}
			o.Add(fn, test.Check, pdb, pdb.GetObjectMeta().Annotations)
		}
	}

	return &scoreCard, nil
}
