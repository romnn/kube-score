package internal

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
)

func LabelSelectorMatchesLabels(
	selectorLabels map[string]string,
	labels map[string]string,
) bool {
	labelSelector := &metav1.LabelSelector{
		MatchLabels: selectorLabels,
	}

	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return false
	}

	return selector.Matches(k8slabels.Set(labels))
}
