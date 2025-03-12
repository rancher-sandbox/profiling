package hack

import "strings"

type Metadata struct {
	Namespace   string
	Name        string
	Target      string
	ProfileType string
}

func SplitPathToMd(path string) Metadata {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")
	return Metadata{
		Namespace:   parts[0],
		Name:        parts[1],
		Target:      parts[2],
		ProfileType: parts[3],
	}
}
