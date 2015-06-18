package nodes

import (
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/tutumcloud/go-tutum/tutum"
)

var (
	Tutum_Node_Api_Uri   = os.Getenv("TUTUM_NODE_API_URI")
	Tutum_Node_Public_Ip = ""
	peer_ips             = []string{""}
)

func nodeAppend(nodeList tutum.NodeListResponse) []string {
	node_ips := []string{}
	for i := range nodeList.Objects {
		state := nodeList.Objects[i].State

		if state == "Deployed" || state == "Unreachable" {
			if nodeList.Objects[i].Public_ip != Tutum_Node_Public_Ip {
				node_ips = append(node_ips, nodeList.Objects[i].Public_ip)
			}
		}
	}
	return node_ips
}

func compareNodePeer(array1, array2, diff []string) []string {
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

func DiscoverPeers() error {
	tries := 0
	log.Println("[NODE DISCOVERY STARTED]")
	for {
		nodeList, err := tutum.ListNodes()
		if err != nil {
			return err
		}

		if len(nodeList.Objects) == 0 {
			return nil
		}

		node_ips := nodeAppend(nodeList)

		log.Println("[NODE DISCOVERY]: Current nodes available")
		log.Println(node_ips)

		var diff1 []string

		//Checking if there are nodes that are not in the peer_ips list

		diff1 = compareNodePeer(node_ips, peer_ips, diff1)

		for _, i := range diff1 {
			err := connectToPeers(i)
			if err != nil {
				tries++
				if tries > 3 {
					return err
				}
			}
		}

		var diff2 []string

		//Checking if there are peers that are not in the node_ips list

		diff2 = compareNodePeer(peer_ips, node_ips, diff2)

		for _, i := range diff2 {
			err := forgetPeers(i)
			if err != nil {
				tries++
				if tries > 3 {
					return err
				}
			}
		}

		peer_ips = node_ips
		break
	}
	log.Println("[NODE DISCOVERY STOPPED]")
	return nil
}

func connectToPeers(node_ip string) error {
	log.Println("[NODE DISCOVERY UPDATE]: Some nodes are not peers")
	tries := 0
Loop:
	for {

		log.Printf("[NODE DISCOVERY]: Connecting to newly discovered peer: %s", node_ip)
		cmd := exec.Command("/weave", "--local", "connect", node_ip)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			tries++
			time.Sleep(2 * time.Second)
			if tries > 3 {
				return err
			}
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			tries++
			time.Sleep(2 * time.Second)
			if tries > 3 {
				return err
			}
		}

		if err := cmd.Start(); err != nil {
			tries++
			time.Sleep(2 * time.Second)
			if tries > 3 {
				return err
			}
		}

		if err := cmd.Wait(); err != nil {
			log.Printf("%s: %s %s", node_ip, stdout, stderr)
			tries++
			time.Sleep(2 * time.Second)
			if tries > 3 {
				log.Printf("[NODE DISCOVERY ERROR]: Unable to 'weave connect: %s %s", stdout, stderr)
				return err
			}
		} else {
			break Loop
		}
	}
	log.Println("[NODE DISCOVERY]: Discover Peers: done!")
	return nil
}

func forgetPeers(node_ip string) error {
	log.Println("[NODE DISCOVERY UPDATE]: Some peers are not nodes anymore")
	tries := 0
Loop:
	for {
		log.Printf("[NODE DISCOVERY]: Forgetting peer: %s", node_ip)
		cmd := exec.Command("/weave", "--local", "forget", node_ip)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			tries++
			time.Sleep(2 * time.Second)
			if tries > 3 {
				return err
			}
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			tries++
			time.Sleep(2 * time.Second)
			if tries > 3 {
				return err
			}
		}

		if err := cmd.Start(); err != nil {
			tries++
			time.Sleep(2 * time.Second)
			if tries > 3 {
				return err
			}
		}

		if err := cmd.Wait(); err != nil {
			log.Printf("CMD ERRO : %s: %s %s", node_ip, stdout, stderr)
			tries++
			time.Sleep(2 * time.Second)
			if tries > 3 {
				log.Printf("[NODE DISCOVERY ERROR]: Unable to 'weave forget: %s %s", stdout, stderr)
				return err
			}
		} else {
			break Loop
		}
	}
	log.Println("[NODE DISCOVERY]: Forget Peers: done!")
	return nil
}
