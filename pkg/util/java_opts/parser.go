package java_opts

import "strings"

type JavaOpts map[string]string

func (a JavaOpts) String(name string, _default ...string) string {
	if len(_default) == 0 {
		return a[name]
	}
	rs, ok := a[name]
	if ok {
		return rs
	}
	return _default[0]
}
func (a JavaOpts) Array(name string) []string {
	var out []string
	for k, v := range a {
		if strings.HasPrefix(k, name) {
			out = append(out, v)
		}
	}
	return out
}

func ParseEnvString(env string) JavaOpts {
	out := make(map[string]string)
	strs := strings.Split(env, " ")
	for _, s := range strs {
		if strings.HasPrefix(s, "-D") {
			if idx := strings.Index(s, "="); idx > 0 {
				out[s[2:idx]] = s[idx+1:]
			}
		}
	}
	return out
}
