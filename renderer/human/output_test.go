package human

import (
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/romnn/kube-score/domain"
	"github.com/romnn/kube-score/scorecard"
)

func getTestCard() *scorecard.Scorecard {
	checks := []scorecard.TestScore{
		{
			Check: domain.Check{
				Name: "test-warning-two-comments",
			},
			Grade: scorecard.GradeWarning,
			Comments: []scorecard.TestScoreComment{
				{
					Path:        "a",
					Summary:     "summary",
					Description: "description",
				},
				{
					// No path
					Summary:          "summary",
					Description:      "description",
					DocumentationURL: "https://kube-score.com/whatever",
				},
			},
		},
		{
			Check: domain.Check{
				Name: "test-ok-comment",
			},
			Grade: scorecard.GradeAllOK,
			Comments: []scorecard.TestScoreComment{
				{
					Path:        "a",
					Summary:     "summary",
					Description: "description",
				},
			},
		},
		{
			Check: domain.Check{
				Name: "test-skipped-comment",
			},
			Skipped: true,
			Comments: []scorecard.TestScoreComment{
				{
					Path:        "a",
					Summary:     "skipped sum",
					Description: "skipped description",
				},
			},
		},
		{
			Check: domain.Check{
				Name: "test-skipped-no-comment",
			},
			Skipped: true,
		},
	}

	return &scorecard.Scorecard{
		"a": &scorecard.ScoredObject{
			TypeMeta: v1.TypeMeta{
				Kind:       "Testing",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "foofoo",
			},
			Checks: checks,
		},

		// No namespace
		"b": &scorecard.ScoredObject{
			TypeMeta: v1.TypeMeta{
				Kind:       "Testing",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "bar-no-namespace",
			},
			Checks: checks,
		},
	}
}

func TestHumanOutputDefault(t *testing.T) {
	t.Parallel()
	r, err := Human(getTestCard(), 0, 100, false)
	assert.Nil(t, err)
	all, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(
		t,
		`v1/Testing foo in foofoo                                                      🤔
    [WARNING] test-warning-two-comments
        · a -> summary
            description
        · summary
            description
            More information: https://kube-score.com/whatever
v1/Testing bar-no-namespace                                                   🤔
    [WARNING] test-warning-two-comments
        · a -> summary
            description
        · summary
            description
            More information: https://kube-score.com/whatever
`,
		string(all),
	)
}

func TestHumanOutputVerbose1(t *testing.T) {
	t.Parallel()
	r, err := Human(getTestCard(), 1, 100, false)
	assert.Nil(t, err)
	all, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(
		t,
		`v1/Testing foo in foofoo                                                      🤔
    [WARNING] test-warning-two-comments
        · a -> summary
            description
        · summary
            description
            More information: https://kube-score.com/whatever
    [OK] test-ok-comment
        · a -> summary
            description
v1/Testing bar-no-namespace                                                   🤔
    [WARNING] test-warning-two-comments
        · a -> summary
            description
        · summary
            description
            More information: https://kube-score.com/whatever
    [OK] test-ok-comment
        · a -> summary
            description
`,
		string(all),
	)
}

func TestHumanOutputVerbose2(t *testing.T) {
	t.Parallel()
	r, err := Human(getTestCard(), 2, 100, false)
	assert.Nil(t, err)
	all, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(
		t,
		`v1/Testing foo in foofoo                                                      🤔
    [WARNING] test-warning-two-comments
        · a -> summary
            description
        · summary
            description
            More information: https://kube-score.com/whatever
    [OK] test-ok-comment
        · a -> summary
            description
    [SKIPPED] test-skipped-comment
        · a -> skipped sum
            skipped description
    [SKIPPED] test-skipped-no-comment
v1/Testing bar-no-namespace                                                   🤔
    [WARNING] test-warning-two-comments
        · a -> summary
            description
        · summary
            description
            More information: https://kube-score.com/whatever
    [OK] test-ok-comment
        · a -> summary
            description
    [SKIPPED] test-skipped-comment
        · a -> skipped sum
            skipped description
    [SKIPPED] test-skipped-no-comment
`,
		string(all),
	)
}

func getTestCardAllOK() *scorecard.Scorecard {
	checks := []scorecard.TestScore{
		{
			Check: domain.Check{
				Name: "test-warning-two-comments",
			},
			Grade: scorecard.GradeAllOK,
			Comments: []scorecard.TestScoreComment{
				{
					Path:        "a",
					Summary:     "summary",
					Description: "description",
				},
				{
					// No path
					Summary:     "summary",
					Description: "description",
				},
			},
		},
		{
			Check: domain.Check{
				Name: "test-ok-comment",
			},
			Grade: scorecard.GradeAllOK,
			Comments: []scorecard.TestScoreComment{
				{
					Path:        "a",
					Summary:     "summary",
					Description: "description",
				},
			},
		},
		{
			Check: domain.Check{
				Name: "test-skipped-comment",
			},
			Skipped: true,
			Comments: []scorecard.TestScoreComment{
				{
					Path:        "a",
					Summary:     "skipped sum",
					Description: "skipped description",
				},
			},
		},
		{
			Check: domain.Check{
				Name: "test-skipped-no-comment",
			},
			Skipped: true,
		},
	}

	return &scorecard.Scorecard{
		"a": &scorecard.ScoredObject{
			TypeMeta: v1.TypeMeta{
				Kind:       "Testing",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "foofoo",
			},
			Checks: checks,
		},

		// No namespace
		"b": &scorecard.ScoredObject{
			TypeMeta: v1.TypeMeta{
				Kind:       "Testing",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "bar-no-namespace",
			},
			Checks: checks,
		},
	}
}

func TestHumanOutputAllOKDefault(t *testing.T) {
	t.Parallel()
	r, err := Human(getTestCardAllOK(), 0, 100, false)
	assert.Nil(t, err)
	all, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(
		t,
		`v1/Testing foo in foofoo                                                      ✅
v1/Testing bar-no-namespace                                                   ✅
`,
		string(all),
	)
}

func getTestCardLongDescription() *scorecard.Scorecard {
	checks := []scorecard.TestScore{
		{
			Check: domain.Check{
				Name: "test-warning-two-comments",
			},
			Grade: scorecard.GradeWarning,
			Comments: []scorecard.TestScoreComment{
				{
					Path:        "a",
					Summary:     "summary",
					Description: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras elementum sagittis lacus, a dictum tortor lobortis vel. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Nulla eu neque erat. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Maecenas et nisl venenatis, elementum augue a, porttitor libero.",
				},
			},
		},
	}

	return &scorecard.Scorecard{
		"a": &scorecard.ScoredObject{
			TypeMeta: v1.TypeMeta{
				Kind:       "Testing",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "foofoo",
			},
			Checks: checks,
		},
	}
}

func TestHumanOutputLogDescription120Width(t *testing.T) {
	t.Parallel()
	r, err := Human(getTestCardLongDescription(), 0, 120, false)
	assert.Nil(t, err)
	all, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(
		t,
		`v1/Testing foo in foofoo                                                      🤔
    [WARNING] test-warning-two-comments
        · a -> summary
            Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras elementum sagittis lacus, a dictum tortor
            lobortis vel. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas.
            Nulla eu neque erat. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae;
            Maecenas et nisl venenatis, elementum augue a, porttitor libero.
`,
		string(all),
	)
}

func TestHumanOutputLogDescription100Width(t *testing.T) {
	t.Parallel()
	r, err := Human(getTestCardLongDescription(), 0, 100, false)
	assert.Nil(t, err)
	all, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(
		t,
		`v1/Testing foo in foofoo                                                      🤔
    [WARNING] test-warning-two-comments
        · a -> summary
            Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras elementum sagittis lacus,
            a dictum tortor lobortis vel. Pellentesque habitant morbi tristique senectus et netus et
            malesuada fames ac turpis egestas. Nulla eu neque erat. Vestibulum ante ipsum primis in
            faucibus orci luctus et ultrices posuere cubilia Curae; Maecenas et nisl venenatis,
            elementum augue a, porttitor libero.
`,
		string(all),
	)
}

func TestHumanOutputLogDescription80Width(t *testing.T) {
	t.Parallel()
	r, err := Human(getTestCardLongDescription(), 0, 80, false)
	assert.Nil(t, err)
	all, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(
		t,
		`v1/Testing foo in foofoo                                                      🤔
    [WARNING] test-warning-two-comments
        · a -> summary
            Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras
            elementum sagittis lacus, a dictum tortor lobortis vel. Pellentesque
            habitant morbi tristique senectus et netus et malesuada fames ac
            turpis egestas. Nulla eu neque erat. Vestibulum ante ipsum primis in
            faucibus orci luctus et ultrices posuere cubilia Curae; Maecenas et
            nisl venenatis, elementum augue a, porttitor libero.
`,
		string(all),
	)
}

func TestHumanOutputLogDescription0Width(t *testing.T) {
	t.Parallel()
	r, err := Human(getTestCardLongDescription(), 0, 0, false)
	assert.Nil(t, err)
	all, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(t, `v1/Testing foo in foofoo🤔
    [WARNING] test-warning-two-comments
        · a -> summary
            Lorem ipsum dolor sit amet, consectetur
            adipiscing elit. Cras elementum sagittis
            lacus, a dictum tortor lobortis vel.
            Pellentesque habitant morbi tristique
            senectus et netus et malesuada fames ac
            turpis egestas. Nulla eu neque erat.
            Vestibulum ante ipsum primis in faucibus
            orci luctus et ultrices posuere cubilia
            Curae; Maecenas et nisl venenatis,
            elementum augue a, porttitor libero.
`, string(all))
}

func getTestCardLongTitle() *scorecard.Scorecard {
	checks := []scorecard.TestScore{
		{
			Check: domain.Check{
				Name: "test-warning-two-comments",
			},
			Grade: scorecard.GradeWarning,
			Comments: []scorecard.TestScoreComment{
				{
					Path:        "a",
					Summary:     "summary",
					Description: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras elementum sagittis lacus, a dictum tortor lobortis vel. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Nulla eu neque erat. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Maecenas et nisl venenatis, elementum augue a, porttitor libero.",
				},
			},
		},
	}

	return &scorecard.Scorecard{
		"a": &scorecard.ScoredObject{
			TypeMeta: v1.TypeMeta{
				Kind:       "Testing",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "this-is-a-very-long-title-this-is-a-very-long-title-this-is-a-very-long-title-this-is-a-very-long-title-this-is-a-very-long-title",
				Namespace: "foofoo",
			},
			Checks: checks,
		},
	}
}

func TestHumanOutputWithLongObjectNames(t *testing.T) {
	t.Parallel()
	r, err := Human(getTestCardLongTitle(), 0, 80, false)
	assert.Nil(t, err)
	all, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(
		t,
		`v1/Testing this-is-a-very-long-title-this-is-a-very-long-title-this-is-a-very-long-title-this-is-a-very-long-title-this-is-a-very-long-title in foofoo🤔
    [WARNING] test-warning-two-comments
        · a -> summary
            Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras
            elementum sagittis lacus, a dictum tortor lobortis vel. Pellentesque
            habitant morbi tristique senectus et netus et malesuada fames ac
            turpis egestas. Nulla eu neque erat. Vestibulum ante ipsum primis in
            faucibus orci luctus et ultrices posuere cubilia Curae; Maecenas et
            nisl venenatis, elementum augue a, porttitor libero.
`,
		string(all),
	)
}

var sgrPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Every colored line has to reset before its newline. A consumer that splits
// the stream and prefixes each line renders the next prefix in the previous
// line's color when the reset lands on the wrong side of the break.
func TestHumanOutputResetsColorBeforeNewline(t *testing.T) {
	// Not parallel: Human writes the package-global color.NoColor.
	previous := color.NoColor
	t.Cleanup(func() { color.NoColor = previous })

	r, err := Human(getTestCard(), 2, 100, true)
	assert.Nil(t, err)
	all, err := io.ReadAll(r)
	assert.Nil(t, err)

	output := string(all)
	assert.Contains(t, output, "\x1b[", "expected colored output")

	for index, line := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
		codes := sgrPattern.FindAllString(line, -1)
		if len(codes) == 0 {
			continue
		}
		assert.Equal(
			t,
			"\x1b[0m",
			codes[len(codes)-1],
			"line %d leaves a color open across the newline: %q",
			index+1,
			line,
		)
	}
}
