package tools

import "strings"

const (
	Version    = "0.21.1"
	DockerPath = "/usr/local/bin/docker"
)

func CompareArrays(array1, array2, diff []string) []string {
	for _, s1 := range array1 {
		found := false
		for _, s2 := range array2 {
			if s1 == s2 {
				found = true
				break
			}
		}
		if !found {
			diff = append(diff, s1)
		}
	}
	return diff
}

func CompareIdArrays(array1, array2, diff []string) []string {
	for _, s1 := range array1 {
		found := false
		for _, s2 := range array2 {
			if strings.Contains(s1, s2) {
				found = true
				break
			}
		}
		if !found {
			diff = append(diff, s1)
		}
	}
	return diff
}
