package tools

import "testing"

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

func Test_compareIdArrays(t *testing.T) {
	//[f14315bb3620 ada93137e004 ]
	array1 := []string{"8a9049b57b356f3d31740aa64f95046e674fdd93a80ace75282547eca2559a64", "c55d26f0f0aad554b7b18ae59c7c644360b205b2f502a0162f216b5f7fdc7c29", "f14315bb36208e3431c31b2370ba24466973023f50f8e25741702770c522735f", "ada93137e00448774e196ef81099492d069cb64490f4b13bf42b7c8a802b0c2c", "21f45b6a5b1c3b0aa5201069cc222b655874177c1faee94a0178c20e58766d5e"}
	array2 := []string{"f14315bb3620", "ada93137e004"}
	array3 := []string{}
	array4 := []string{"8a9049b57b356f3d31740aa64f95046e674fdd93a80ace75282547eca2559a64", "c55d26f0f0aad554b7b18ae59c7c644360b205b2f502a0162f216b5f7fdc7c29", "21f45b6a5b1c3b0aa5201069cc222b655874177c1faee94a0178c20e58766d5e"}

	array3 = CompareIdArrays(array1, array2, array3)
	if !testEq(array3, array4) {
		t.Error("Unexpected id array")
	}
}

func Test_compareArrays(t *testing.T) {
	var diff1 []string
	var diff2 []string

	node_ips := []string{`10.0.0.1`, `10.0.0.2`, `10.0.0.3`, `10.0.0.4`}
	node_ips2 := []string{`10.0.0.1`, `10.0.0.3`, `10.0.0.4`}
	peer_ips := []string{`10.0.0.2`}

	diff1 = CompareArrays(node_ips, peer_ips, diff1)

	expectedNodeList := []string{`10.0.0.1`, `10.0.0.3`, `10.0.0.4`}

	if !testEq(diff1, expectedNodeList) {
		t.Error("Unexpected node ips list")
	}

	diff2 = CompareArrays(peer_ips, node_ips2, diff2)

	expectedPeerList := []string{`10.0.0.2`}
	if !testEq(diff2, expectedPeerList) {
		t.Error("Unexpected peer ips list")
	}
}

func Test_removeDuplicates(t *testing.T) {
	a := []string{"192.168.130.23", "192.168.130.24", "192.168.130.23", "192.168.130.24", "192.168.130.22"}
	a_without_duplicates := []string{"192.168.130.23", "192.168.130.24", "192.168.130.22"}

	a = RemoveDuplicates(a)

	if !testEq(a, a_without_duplicates) {
		t.Error("Unexpected node ips list")
	}
}
