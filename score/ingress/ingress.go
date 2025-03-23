package ingress

import (
	"fmt"

	ks "github.com/romnn/kube-score/domain"
	"github.com/romnn/kube-score/score/checks"
	"github.com/romnn/kube-score/scorecard"
)

type Options struct {
	Namespace string
}

func Register(allChecks *checks.Checks, services ks.Services, options Options) {
	allChecks.RegisterIngressCheck(
		"Ingress targets Service",
		`Makes sure that the Ingress targets a Service`,
		ingressTargetsService(services.Services(), options),
	)
}

func ingressTargetsService(
	allServices []ks.Service,
	options Options,
) func(ks.Ingress) (scorecard.TestScore, error) {
	return func(ingress ks.Ingress) (scorecard.TestScore, error) {
		return ingressTargetsServiceCommon(ingress, allServices, options)
	}
}

func ingressTargetsServiceCommon(
	ingress ks.Ingress,
	allServices []ks.Service,
	options Options,
) (score scorecard.TestScore, err error) {
	allRulesHaveMatches := true

	for _, rule := range ingress.Rules() {
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}

		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			pathHasMatch := false

			for _, srv := range allServices {
				service := srv.Service()

				serviceNamespace := service.Namespace
				if serviceNamespace == "" {
					serviceNamespace = options.Namespace
				}
				ingressNamespace := ingress.GetObjectMeta().Namespace
				if ingressNamespace == "" {
					ingressNamespace = options.Namespace
				}

				if serviceNamespace != ingressNamespace {
					continue
				}
				if path.Backend.Service == nil {
					continue
				}

				if service.Name == path.Backend.Service.Name {
					for _, servicePort := range service.Spec.Ports {
						if path.Backend.Service.Port.Number > 0 &&
							servicePort.Port == path.Backend.Service.Port.Number {
							pathHasMatch = true
						} else if servicePort.Name == path.Backend.Service.Port.Name {
							pathHasMatch = true
						}
					}
				}
			}

			if !pathHasMatch {
				allRulesHaveMatches = false
				if path.Backend.Service != nil {
					if path.Backend.Service.Port.Number > 0 {
						score.AddComment(
							path.Path,
							"No service match was found",
							fmt.Sprintf(
								"No service with name %s and port number %d was found",
								path.Backend.Service.Name,
								path.Backend.Service.Port.Number,
							),
						)
					} else {
						score.AddComment(
							path.Path,
							"No service match was found",
							fmt.Sprintf(
								"No service with name %s and port named %s was found",
								path.Backend.Service.Name,
								path.Backend.Service.Port.Name,
							),
						)
					}
				} else {
					score.AddComment(path.Path, "No service match was found", "")
				}
			}
		}
	}

	if allRulesHaveMatches {
		score.Grade = scorecard.GradeAllOK
	} else {
		score.Grade = scorecard.GradeCritical
	}

	return
}
