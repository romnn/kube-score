package networkpolicy

import (
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ks "github.com/zegl/kube-score/domain"
	"github.com/zegl/kube-score/score/checks"
	"github.com/zegl/kube-score/score/internal"
	"github.com/zegl/kube-score/scorecard"
)

type Options struct {
	Namespace string
}

func Register(allChecks *checks.Checks, netpols ks.NetworkPolicies, pods ks.Pods, podspecers ks.PodSpeccers, options Options) {
	allChecks.RegisterPodCheck("Pod NetworkPolicy", `Makes sure that all Pods are targeted by a NetworkPolicy`, podHasNetworkPolicy(netpols.NetworkPolicies(), options))
	allChecks.RegisterNetworkPolicyCheck("NetworkPolicy targets Pod", `Makes sure that all NetworkPolicies targets at least one Pod`, networkPolicyTargetsPod(pods.Pods(), podspecers.PodSpeccers(), options))
}

// podHasNetworkPolicy returns a function that tests that all pods have matching NetworkPolicies
// podHasNetworkPolicy takes a list of all defined NetworkPolicies as input
func podHasNetworkPolicy(allNetpols []ks.NetworkPolicy, options Options) func(ks.PodSpecer) (scorecard.TestScore, error) {
	return func(ps ks.PodSpecer) (score scorecard.TestScore, err error) {
		hasMatchingEgressNetpol := false
		hasMatchingIngressNetpol := false

		podNamespace := ps.GetPodTemplateSpec().Namespace
		if podNamespace == "" {
			podNamespace = options.Namespace
		}

		for _, n := range allNetpols {
			netPol := n.NetworkPolicy()

			netPolNamespace := netPol.Namespace

			if netPolNamespace == "" {
				netPolNamespace = options.Namespace
			}

			// fmt.Printf(
			// 	"policyNamespace=%s/%s podNamespace=%s/%s\n",
			// 	netPolNamespace,
			// 	netPol.Name,
			// 	podNamespace,
			// 	ps.GetPodTemplateSpec().Name,
			// )

			// Make sure that the pod and networkpolicy is in the same namespace
			if podNamespace != netPolNamespace {
				continue
			}

			if selector, err := metav1.LabelSelectorAsSelector(&netPol.Spec.PodSelector); err == nil {
				if selector.Matches(internal.MapLabels(ps.GetPodTemplateSpec().Labels)) {

					// Documentation of PolicyTypes
					//
					// List of rule types that the NetworkPolicy relates to.
					// Valid options are "Ingress", "Egress", or "Ingress,Egress".
					// If this field is not specified, it will default based on the existence of Ingress or Egress rules;
					// policies that contain an Egress section are assumed to affect Egress, and all policies
					// (whether or not they contain an Ingress section) are assumed to affect Ingress.
					// If you want to write an egress-only policy, you must explicitly specify policyTypes [ "Egress" ].
					// Likewise, if you want to write a policy that specifies that no egress is allowed,
					// you must specify a policyTypes value that include "Egress" (since such a policy would not include
					// an Egress section and would otherwise default to just [ "Ingress" ]).

					if netPol.Spec.PolicyTypes == nil || len(netPol.Spec.PolicyTypes) == 0 {
						hasMatchingIngressNetpol = true
						if len(netPol.Spec.Egress) > 0 {
							hasMatchingEgressNetpol = true
						}
					} else {
						for _, policyType := range netPol.Spec.PolicyTypes {
							if policyType == networkingv1.PolicyTypeIngress {
								hasMatchingIngressNetpol = true
							}
							if policyType == networkingv1.PolicyTypeEgress {
								hasMatchingEgressNetpol = true
							}
						}
					}
				}
			}
		}

		switch {
		case hasMatchingEgressNetpol && hasMatchingIngressNetpol:
			score.Grade = scorecard.GradeAllOK
		case hasMatchingEgressNetpol && !hasMatchingIngressNetpol:
			score.Grade = scorecard.GradeWarning
			score.AddComment("", "The pod does not have a matching ingress NetworkPolicy", "Add a ingress policy to the pods NetworkPolicy")
		case hasMatchingIngressNetpol && !hasMatchingEgressNetpol:
			score.Grade = scorecard.GradeWarning
			score.AddComment("", "The pod does not have a matching egress NetworkPolicy", "Add a egress policy to the pods NetworkPolicy")
		default:
			score.Grade = scorecard.GradeCritical
			score.AddComment("", "The pod does not have a matching NetworkPolicy", "Create a NetworkPolicy that targets this pod to control who/what can communicate with this pod. Note, this feature needs to be supported by the CNI implementation used in the Kubernetes cluster to have an effect.")
		}

		return
	}
}

func networkPolicyTargetsPod(pods []ks.Pod, podspecers []ks.PodSpecer, options Options) func(networkingv1.NetworkPolicy) (scorecard.TestScore, error) {
	return func(netpol networkingv1.NetworkPolicy) (score scorecard.TestScore, err error) {
		hasMatch := false

		netPolNamespace := netpol.Namespace
		if netPolNamespace == "" {
			netPolNamespace = options.Namespace
		}

		for _, p := range pods {
			pod := p.Pod()
			podNamespace := pod.Namespace

			if podNamespace == "" {
				podNamespace = options.Namespace
			}

			// fmt.Printf(
			// 	"policyNamespace=%s/%s podNamespace=%s/%s\n",
			// 	netPolNamespace,
			// 	netpol.Name,
			// 	podNamespace,
			// 	pod.Name,
			// )
			if podNamespace != netPolNamespace {
				continue
			}

			if selector, err := metav1.LabelSelectorAsSelector(&netpol.Spec.PodSelector); err == nil {
				if selector.Matches(internal.MapLabels(pod.Labels)) {
					hasMatch = true
					break
				}
			}
		}

		if !hasMatch {
			for _, pod := range podspecers {
				if pod.GetObjectMeta().Namespace != netPolNamespace {
					continue
				}

				if selector, err := metav1.LabelSelectorAsSelector(&netpol.Spec.PodSelector); err == nil {
					if selector.Matches(internal.MapLabels(pod.GetPodTemplateSpec().Labels)) {
						hasMatch = true
						break
					}
				}
			}
		}

		if hasMatch {
			score.Grade = scorecard.GradeAllOK
		} else {
			score.Grade = scorecard.GradeCritical
			score.AddComment("", "The NetworkPolicys selector doesn't match any pods", "")
		}

		return
	}
}
