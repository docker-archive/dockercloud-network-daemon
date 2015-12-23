package nodes

import (
	"os"
	"testing"

	"github.com/tutumcloud/go-tutum/tutum"
)

func testEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func Test_CIDRToIP(t *testing.T) {
	a := []string{"192.168.130.23/24", "192.168.130.24/24", "192.168.130.23/24", "192.168.130.24/24", "192.168.130.22/24"}
	b := []string{"192.168.130.23", "192.168.130.24", "192.168.130.23", "192.168.130.24", "192.168.130.22"}

	a = CIDRToIP(a)
	if !testEq(a, b) {
		t.Error("Unexpected node ips list")
	}
}

func Test_removeDuplicates(t *testing.T) {
	a := []string{"192.168.130.23", "192.168.130.24", "192.168.130.23", "192.168.130.24", "192.168.130.22"}
	a_without_duplicates := []string{"192.168.130.23", "192.168.130.24", "192.168.130.22"}

	a = removeDuplicates(a)

	if !testEq(a, a_without_duplicates) {
		t.Error("Unexpected node ips list")
	}
}

func Test_IsInPrivateRange(t *testing.T) {
	IP1 := "159.8.238.60/16"
	response1 := IsInPrivateRange(IP1)

	IP2 := "192.168.1.12/16"
	response2 := IsInPrivateRange(IP2)

	IP3 := "10.136.220.69/32"
	response3 := IsInPrivateRange(IP3)

	IP4 := "172.19.27.18/16"
	response4 := IsInPrivateRange(IP4)

	if response1 != false {
		t.Error("Unexpected response, got true expected false")
	}

	if response2 != true || response3 != true || response4 != true {
		t.Error("Unexpected response, got false expected true")
	}
}

func Test_nodeAppendAWS(t *testing.T) {
	Tutum_Node_Public_Ip = "178.100.50.34"
	Tutum_Region = "/1/2/3"
	Tutum_Node_CIDR = []tutum.Network{{Name: "eth0", CIDR: "192.168.130.23/24"}, {Name: "eth1", CIDR: "10.77.32.17/17"}}

	os.Setenv("TUTUM_PRIVATE_CIDR", "10.77.0.0/16")
	nodeList := tutum.NodeListResponse{Objects: []tutum.Node{
		{Uuid: "1", State: "Deployed", Region: "/1/2/3", Public_ip: "10.0.0.1", Private_ips: []tutum.Network{{Name: "eth0", CIDR: "10.77.250.17/17"}}},
		{Uuid: "2", State: "Deployed", Region: "/1/2/2", Public_ip: "10.0.0.2", Private_ips: []tutum.Network{{Name: "eth0", CIDR: "10.77.32.16/17"}}}}}

	node_public_ips, node_private_ips := NodeAppend(nodeList)

	expectedList := []string{"10.0.0.2"}
	expectedListPrivate := []string{"10.77.250.17"}

	if !testEq(node_public_ips, expectedList) && !testEq(node_private_ips, expectedListPrivate) {
		t.Error("Unexpected node ips list")
	}
}

func Test_compareNodePeer(t *testing.T) {
	var diff1 []string
	var diff2 []string

	node_ips := []string{`10.0.0.1`, `10.0.0.2`, `10.0.0.3`, `10.0.0.4`}
	node_ips2 := []string{`10.0.0.1`, `10.0.0.3`, `10.0.0.4`}
	peer_ips := []string{`10.0.0.2`}

	diff1 = compareNodePeer(node_ips, peer_ips, diff1)

	expectedNodeList := []string{`10.0.0.1`, `10.0.0.3`, `10.0.0.4`}

	if !testEq(diff1, expectedNodeList) {
		t.Error("Unexpected node ips list")
	}

	diff2 = compareNodePeer(peer_ips, node_ips2, diff2)

	expectedPeerList := []string{`10.0.0.2`}
	if !testEq(diff2, expectedPeerList) {
		t.Error("Unexpected peer ips list")
	}
}
