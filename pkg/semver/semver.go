package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents semantic version
type Version struct {
	Major int
	Minor int
	Patch int
}

// Parse parses semver string
func Parse(v string) (*Version, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")

	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid semver: %s", v)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}
	if major < 0 {
		return nil, fmt.Errorf("major version cannot be negative: %d", major)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	if minor < 0 {
		return nil, fmt.Errorf("minor version cannot be negative: %d", minor)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, err
	}
	if patch < 0 {
		return nil, fmt.Errorf("patch version cannot be negative: %d", patch)
	}

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// String converts version to string
func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare compares two versions (-1, 0, 1)
func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		return compare(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compare(v.Minor, other.Minor)
	}
	return compare(v.Patch, other.Patch)
}

func compare(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// Constraint represents version constraint
type Constraint struct {
	Operator string
	Version  *Version
}

// ParseConstraint parses version constraint
// Supports: ^1.2.0, ~1.2.0, >=1.0.0, >1.0.0, <=1.0.0, <1.0.0, 1.2.0, 1.x, 1.2.x
func ParseConstraint(c string) (*Constraint, error) {
	c = strings.TrimSpace(c)

	// Exact version
	if !strings.ContainsAny(c, "^~<>=x") {
		v, err := Parse(c)
		if err != nil {
			return nil, err
		}
		return &Constraint{Operator: "=", Version: v}, nil
	}

	// Caret (^1.2.0 = >=1.2.0 <2.0.0)
	if strings.HasPrefix(c, "^") {
		v, err := Parse(c[1:])
		if err != nil {
			return nil, err
		}
		return &Constraint{Operator: "^", Version: v}, nil
	}

	// Tilde (~1.2.0 = >=1.2.0 <1.3.0)
	if strings.HasPrefix(c, "~") {
		v, err := Parse(c[1:])
		if err != nil {
			return nil, err
		}
		return &Constraint{Operator: "~", Version: v}, nil
	}

	// Operators
	operators := []string{">=", "<=", ">", "<"}
	for _, op := range operators {
		if strings.HasPrefix(c, op) {
			v, err := Parse(c[len(op):])
			if err != nil {
				return nil, err
			}
			return &Constraint{Operator: op, Version: v}, nil
		}
	}

	// Wildcard (1.x or 1.2.x)
	if strings.Contains(c, "x") {
		return parseWildcard(c)
	}

	return nil, fmt.Errorf("invalid constraint: %s", c)
}

// parseWildcard handles wildcard constraints
func parseWildcard(c string) (*Constraint, error) {
	c = strings.TrimPrefix(c, "v")
	parts := strings.Split(c, ".")

	if len(parts) == 2 && parts[1] == "x" {
		// 1.x = >=1.0.0 <2.0.0
		major, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, err
		}
		return &Constraint{
			Operator: "^",
			Version:  &Version{Major: major, Minor: 0, Patch: 0},
		}, nil
	}

	if len(parts) == 3 && parts[2] == "x" {
		// 1.2.x = >=1.2.0 <1.3.0
		major, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid major version: %s", parts[0])
		}
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid minor version: %s", parts[1])
		}
		return &Constraint{
			Operator: "~",
			Version:  &Version{Major: major, Minor: minor, Patch: 0},
		}, nil
	}

	return nil, fmt.Errorf("invalid wildcard: %s", c)
}

// Satisfies checks if version satisfies constraint
func (c *Constraint) Satisfies(v *Version) bool {
	cmp := v.Compare(c.Version)

	switch c.Operator {
	case "=":
		return cmp == 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case "^":
		// ^1.2.0 = >=1.2.0 <2.0.0
		if v.Major != c.Version.Major {
			return false
		}
		return cmp >= 0
	case "~":
		// ~1.2.0 = >=1.2.0 <1.3.0
		if v.Major != c.Version.Major || v.Minor != c.Version.Minor {
			return false
		}
		return cmp >= 0
	}

	return false
}

// FindBestMatch finds best version from list that satisfies constraint
func (c *Constraint) FindBestMatch(versions []string) (string, error) {
	var best *Version
	var bestStr string

	for _, vStr := range versions {
		v, err := Parse(vStr)
		if err != nil {
			continue
		}

		if c.Satisfies(v) {
			if best == nil || v.Compare(best) > 0 {
				best = v
				bestStr = vStr
			}
		}
	}

	if best == nil {
		return "", fmt.Errorf("no version satisfies %s%s", c.Operator, c.Version)
	}

	return bestStr, nil
}
