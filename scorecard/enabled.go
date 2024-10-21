package scorecard

import (
	"fmt"
	"strings"

	ks "github.com/zegl/kube-score/domain"
)

func (so *ScoredObject) isEnabled(check ks.Check, annotations, childAnnotations map[string]string) bool {
	isIn := func(annotations map[string]string, csv string, key string) bool {
		// see if the check is explicitly allowed or denied
		if checkAnnotation, ok := annotations[fmt.Sprintf("kube-score/%s", check.ID)]; ok {
			switch strings.TrimSpace(strings.ToLower(checkAnnotation)) {
			case "allow", "allowed", "enable", "enabled", "yes":
				// fmt.Printf("enabling check %s\n", check.ID)
				return true
			case "deny", "denied", "disable", "disabled", "no":
				// fmt.Printf("disabling check %s\n", check.ID)
				return false
			}
		}

		// search comma separated list of checks
		for _, v := range strings.Split(csv, ",") {
			v = strings.TrimSpace(v)
			if v == key {
				return true
			}
			if v == "*" {
				// "*" wildcard matches all checks
				return true
			}
			if vals, ok := impliedIgnoreAnnotations[v]; ok {
				for i := range vals {
					if vals[i] == key {
						return true
					}
				}
			}
		}
		return false
	}

	if childAnnotations != nil && so.useIgnoreChecksAnnotation && isIn(childAnnotations, childAnnotations[ignoredChecksAnnotation], check.ID) {
		return false
	}
	if childAnnotations != nil && so.useOptionalChecksAnnotation && isIn(childAnnotations, childAnnotations[optionalChecksAnnotation], check.ID) {
		return true
	}
	if so.useIgnoreChecksAnnotation && isIn(annotations, annotations[ignoredChecksAnnotation], check.ID) {
		return false
	}
	if so.useOptionalChecksAnnotation && isIn(annotations, annotations[optionalChecksAnnotation], check.ID) {
		return true
	}

	// Enabled optional test from command line arguments
	if _, ok := so.enabledOptionalTests[check.ID]; ok {
		return true
	}

	// Optional checks are disabled unless explicitly allowed above
	if check.Optional {
		return false
	}

	// Enabled by default
	return true
}
