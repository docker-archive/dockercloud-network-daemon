package tools

import "strings"

const (
	Version    = "0.21.1"
	DockerPath = "/usr/local/bin/docker"
)

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func InHashWithValue(containerAttached map[string]string, id string, value string) bool {
	if val, ok := containerAttached[id]; ok && val == value {
		return true
	}
	return false
}

func RemoveMissing(containerAttached map[string]string, containerList []string) map[string]string {
	for k, _ := range containerAttached {
		if !stringInSlice(k, containerList) {
			delete(containerAttached, k)
		}
	}
	return containerAttached
}

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
