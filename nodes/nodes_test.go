package nodes

import (
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

func Test_nodeAppend(t *testing.T) {
	nodeList := tutum.NodeListResponse{Objects: []tutum.Node{{State: "Deployed", Public_ip: "10.0.0.1"}, {State: "Deployed", Public_ip: "10.0.0.2"}, {State: "Terminated", Public_ip: "10.0.0.3"}, {State: "Unreachable", Public_ip: "10.0.0.4"}}}
	node_ips := nodeAppend(nodeList)
	expectedList := []string{`10.0.0.1`, `10.0.0.2`, `10.0.0.4`}

	if !testEq(node_ips, expectedList) {
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
