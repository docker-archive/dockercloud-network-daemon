package nodes

import (
	"fmt"
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

func DiscoverPeers() {
	tries := 0
	for {
		node_ips := []string{}
		nodeList, err := tutum.ListNodes()
		if err != nil {
			tries++
			time.Sleep(1)
			if tries > 3 {
				log.Fatal(err)
			}
		}
		for i := range nodeList.Objects {
			state := nodeList.Objects[i].State

			if state == "Deployed" {
				if nodeList.Objects[i].Public_ip != Tutum_Node_Public_Ip {
					node_ips = append(node_ips, nodeList.Objects[i].Public_ip)
				}
			}
		}

		fmt.Println("Discovering peers")
		for _, i := range node_ips {
			for _, ip := range peer_ips {
				if i != ip {
					connectToPeers(i)
				}
			}
		}

		for _, ip := range peer_ips {
			for _, i := range node_ips {
				if ip != i {
					forgetPeers(ip)
				}
			}
		}
		peer_ips = node_ips
		break
	}
}

func connectToPeers(node_ip string) {
	tries := 0
	for {
		log.Printf("connecting to newly discovered peer: %s", node_ip)
		cmd := exec.Command("/weave", "--local", "connect", node_ip)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}

		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}

		if err := cmd.Wait(); err != nil {
			log.Printf("%s: %s %s", node_ip, stdout, stderr)
			tries++
			if tries > 3 {
				log.Printf("Unable to 'weave connect: %s %s", stdout, stderr)
			}
			time.Sleep(1)
		} else {
			break
		}
	}
}

func forgetPeers(node_ip string) {
	tries := 0
	for {
		log.Printf("forgetting peer: %s", node_ip)
		cmd := exec.Command("/weave", "--local", "forget", node_ip)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}

		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}

		if err := cmd.Wait(); err != nil {
			log.Printf("%s: %s %s", node_ip, stdout, stderr)
			tries++
			if tries > 3 {
				log.Printf("Unable to 'weave forget: %s %s", stdout, stderr)
			}
			time.Sleep(1)
		} else {
			break
		}
	}
}

func EventHandler(event tutum.Event) {
	if event.Type == "node" && (event.State == "Deployed" || event.State == "Terminated") {
		DiscoverPeers()
	}
}
