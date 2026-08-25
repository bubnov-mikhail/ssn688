package campaign

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer is semantic version major.minor.patch.
type SemVer struct {
	Major int
	Minor int
	Patch int
}

func (v SemVer) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func ParseSemVer(s string) SemVer {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ".")
	v := SemVer{}
	if len(parts) > 0 {
		v.Major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		v.Minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		v.Patch, _ = strconv.Atoi(parts[2])
	}
	return v
}

// Compare returns -1 if v < o, 0 if equal, +1 if v > o.
func (v SemVer) Compare(o SemVer) int {
	if v.Major != o.Major {
		if v.Major < o.Major {
			return -1
		}
		return 1
	}
	if v.Minor != o.Minor {
		if v.Minor < o.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != o.Patch {
		if v.Patch < o.Patch {
			return -1
		}
		return 1
	}
	return 0
}

func (v SemVer) AtLeast(o SemVer) bool {
	return v.Compare(o) >= 0
}
