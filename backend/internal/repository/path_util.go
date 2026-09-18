package repository

import (
	"strings"
)

// matchPath matches a configured endpoint path pattern against an incoming
// request path. Patterns may contain :param segments (e.g. /api/users/:id).
func matchPath(pattern, path string) (map[string]string, bool) {
	params := map[string]string{}
	ps := splitPath(pattern)
	ss := splitPath(path)
	if len(ps) != len(ss) {
		return nil, false
	}
	for i, seg := range ps {
		if strings.HasPrefix(seg, ":") {
			params[strings.TrimPrefix(seg, ":")] = ss[i]
			continue
		}
		if seg != ss[i] {
			return nil, false
		}
	}
	return params, true
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func equalFold(a, b string) bool {
	return strings.EqualFold(a, b)
}
