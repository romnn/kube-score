package score

import (
	"testing"

	"github.com/zegl/kube-score/scorecard"
)

func TestPodTopologySpreadConstraintsWithOneConstraint(t *testing.T) {
	t.Parallel()
	testExpectedScore(
		t,
		"pod-topology-spread-constraints-one-constraint.yaml",
		"Pod Topology Spread Constraints",
		scorecard.GradeAllOK,
	)
}

func TestPodTopologySpreadConstraintsWithTwoConstraints(t *testing.T) {
	t.Parallel()
	testExpectedScore(
		t,
		"pod-topology-spread-constraints-two-constraints.yaml",
		"Pod Topology Spread Constraints",
		scorecard.GradeAllOK,
	)
}

func TestPodTopologySpreadConstraintsNoLabelSelector(t *testing.T) {
	t.Parallel()
	testExpectedScore(
		t,
		"pod-topology-spread-constraints-no-labelselector.yaml",
		"Pod Topology Spread Constraints",
		scorecard.GradeCritical,
	)
}

func TestPodTopologySpreadConstraintsInvalidMaxSkew(t *testing.T) {
	t.Parallel()
	testExpectedScore(
		t,
		"pod-topology-spread-constraints-invalid-maxskew.yaml",
		"Pod Topology Spread Constraints",
		scorecard.GradeCritical,
	)
}

func TestPodTopologySpreadConstraintsInvalidMinDomains(t *testing.T) {
	t.Parallel()
	testExpectedScore(
		t,
		"pod-topology-spread-constraints-invalid-mindomains.yaml",
		"Pod Topology Spread Constraints",
		scorecard.GradeCritical,
	)
}

func TestPodTopologySpreadConstraintsNoTopologyKey(t *testing.T) {
	t.Parallel()
	testExpectedScore(
		t,
		"pod-topology-spread-constraints-no-topologykey.yaml",
		"Pod Topology Spread Constraints",
		scorecard.GradeCritical,
	)
}

func TestPodTopologySpreadConstraintsInvalidDirective(t *testing.T) {
	t.Parallel()
	testExpectedScore(
		t,
		"pod-topology-spread-constraints-invalid-whenunsatisfiable.yaml",
		"Pod Topology Spread Constraints",
		scorecard.GradeCritical,
	)
}
