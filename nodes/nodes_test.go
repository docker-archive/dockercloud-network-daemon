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

func Test_nodeAppendAWS(t *testing.T) {
	Tutum_Node_Public_Ip = "178.100.50.34"
	Tutum_NodeCluster_Uri = "/1/2/3"
	Tutum_Node_CIDR = []tutum.Network{{Name: "eth0", CIDR: "192.168.130.23/24"}, {Name: "eth1", CIDR: "10.77.32.17/17"}}

	os.Setenv("TUTUM_PRIVATE_CIDR", "10.77.0.0/16")
	nodeList := tutum.NodeListResponse{Objects: []tutum.Node{
		{Uuid: "1", State: "Deployed", Node_cluster: "/1/2/3", Public_ip: "10.0.0.1", Private_ips: []tutum.Network{{Name: "eth0", CIDR: "10.77.250.17/17"}}},
		{Uuid: "2", State: "Deployed", Node_cluster: "/1/2/2", Public_ip: "10.0.0.2", Private_ips: []tutum.Network{{Name: "eth0", CIDR: "10.77.32.16/17"}}}}}

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
