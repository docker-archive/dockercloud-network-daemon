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

func DiscoverPeers() error {
	tries := 0
	for {
		node_ips := []string{}
		nodeList, err := tutum.ListNodes()
		if err != nil {
			return err
		}

		for i := range nodeList.Objects {
			state := nodeList.Objects[i].State

			if state == "Deployed" || state == "Unreachable" {
				if nodeList.Objects[i].Public_ip != Tutum_Node_Public_Ip {
					node_ips = append(node_ips, nodeList.Objects[i].Public_ip)
				}
			}
		}

		var diff1 []string

		//Checking if there are nodes that are not in the peer_ips list

		for _, s1 := range node_ips {
			found := false
			for _, s2 := range peer_ips {
				if s1 == s2 {
					found = true
					break
				}
			}
			if !found {
				diff1 = append(diff1, s1)
			}
		}

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

		for _, s1 := range peer_ips {
			found := false
			for _, s2 := range node_ips {
				if s1 == s2 {
					found = true
					break
				}
			}
			if !found {
				diff2 = append(diff2, s1)
			}
		}

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
	log.Println("Stopping discovery")
	return nil
}

func connectToPeers(node_ip string) error {
	tries := 0
	for {

		log.Printf("connecting to newly discovered peer: %s", node_ip)
		cmd := exec.Command("/weave", "--local", "connect", node_ip)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			tries++
			if tries > 3 {
				return err
			}
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			tries++
			if tries > 3 {
				return err
			}
		}

		if err := cmd.Start(); err != nil {
			tries++
			if tries > 3 {
				return err
			}
		}

		if err := cmd.Wait(); err != nil {
			log.Printf("%s: %s %s", node_ip, stdout, stderr)
			tries++
			if tries > 3 {
				log.Printf("Unable to 'weave connect: %s %s", stdout, stderr)
				return err
			}
		} else {
			break
		}
	}
	log.Println("Discover Peers: done!")
	return nil
}

func forgetPeers(node_ip string) error {
	tries := 0
	for {
		log.Printf("forgetting peer: %s", node_ip)
		cmd := exec.Command("/weave", "--local", "forget", node_ip)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			tries++
			if tries > 3 {
				return err
			}
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			tries++
			if tries > 3 {
				return err
			}
		}

		if err := cmd.Start(); err != nil {
			tries++
			if tries > 3 {
				return err
			}
		}

		if err := cmd.Wait(); err != nil {
			log.Printf("CMD ERRO : %s: %s %s", node_ip, stdout, stderr)
			tries++
			if tries > 3 {
				log.Printf("Unable to 'weave forget: %s %s", stdout, stderr)
				return err
			}
			time.Sleep(1)
		} else {
			break
		}
	}
	log.Println("Forget Peers: done!")
	return nil
}

func EventHandler(event tutum.Event) error {
	if event.Type == "node" && (event.State == "Deployed" || event.State == "Terminated") {
		err := DiscoverPeers()
		if err != nil {
			return err
		}
	}
	return nil
}
