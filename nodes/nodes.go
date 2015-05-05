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

func DiscoverPeers(ch chan string) {
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

			if state == "Deployed" { //or Unreachable
				if nodeList.Objects[i].Public_ip != Tutum_Node_Public_Ip {
					node_ips = append(node_ips, nodeList.Objects[i].Public_ip)
				}
			}
		}

		ch <- fmt.Sprintf("Discovering peers")
		var diff1 []string
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
			ch <- fmt.Sprintln(diff1)
			for _, i := range diff1 {
				connectToPeers(i)
			}
		}
		ch <- fmt.Sprintln(node_ips)
		ch <- fmt.Sprintln(peer_ips)

		ch <- fmt.Sprintf("Forgetting peers")
		var diff2 []string
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
			ch <- fmt.Sprintln(diff2)
			for _, i := range diff2 {
				forgetPeers(i)
			}
		}
		/*for _, ip := range peer_ips {
			for _, i := range node_ips {
				if ip != i {
					forgetPeers(ip)
				}
			}
		}*/
		//peer_ips = node_ips
		ch <- fmt.Sprintln(node_ips)
		ch <- fmt.Sprintln(peer_ips)
		break
	}
	ch <- fmt.Sprint("STOP DISCOVER FUNCTION")
}

func connectToPeers(node_ip string) {
	//tries := 0

	for {
		log.Printf("connecting to newly discovered peer: %s", node_ip)
		cmd := exec.Command("/weave", "--local", "connect", node_ip)
		/*stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}*/

		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		} else {
			break
		}

		if err := cmd.Wait(); err != nil {
			log.Fatal(err)
		} else {
			fmt.Println("Connected !")
			break
		}
		time.Sleep(1)
	}
}

/*
log.Printf("%s: %s %s", node_ip, stdout, stderr)
tries++
if tries > 3 {
	log.Printf("Unable to 'weave connect: %s %s", stdout, stderr)
	break
*/

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
	ch := make(chan string)
	if event.Type == "node" && (event.State == "Deployed" || event.State == "Terminated") {
		DiscoverPeers(ch)
	}
}
