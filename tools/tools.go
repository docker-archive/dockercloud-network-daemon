package tools

import (
	"log"
	"net"
	"strings"

	"github.com/docker/go-dockercloud/dockercloud"
)

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

func RemoveDuplicates(elements []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		if encountered[elements[v]] == true {
			// Do not add duplicate.
		} else {
			encountered[elements[v]] = true
			result = append(result, elements[v])
		}
	}
	return result
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func GetInterfaces() []dockercloud.Network {
	rawInterfaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Cannot get network interfaces: %s", err.Error())
	}

	ifs := make([]dockercloud.Network, 0, 0)
	for _, iface := range rawInterfaces {
		name := strings.ToLower(iface.Name)
		addrs, err := iface.Addrs()
		if err != nil {
			log.Printf("Cannot get address from interface %s: %s", iface.Name, err.Error())
			continue
		}
		log.Printf("Found interface %s: %s", name, addrs)

		var cidr string

		if !contains([]string{"docker0", "weave", "lo"}, name) {
			for _, addr := range addrs {
				cidr = addr.String()
				if strings.ContainsAny(cidr, "abcdef:") {
					continue
				}

				ifs = append(ifs, dockercloud.Network{Name: name, CIDR: cidr})
			}
		}
	}
	return ifs
}
