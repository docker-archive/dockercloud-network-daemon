package nodes

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/tutumcloud/go-tutum/tutum"
)

type NodeNetwork struct {
	Public_Ip string
	cidrs     []tutum.Network
}

type PostForm struct {
	Interfaces []tutum.Network `json:"private_ips"`
}

const (
	Version = "0.20.0"
)

var (
	Tutum_Node_Api_Uri   = os.Getenv("TUTUM_NODE_API_URI")
	Tutum_Node_Public_Ip = ""
	Tutum_Node_CIDR      = []tutum.Network{}
	Tutum_Node_Uuid      = ""
	peer_ips             = []string{}
	peer_ips_public      = []string{}
)

func removeDuplicates(elements []string) []string {
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

func getInterfaces() []tutum.Network {
	rawInterfaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Cannot get network interfaces: %s", err.Error())
	}

	ifs := make([]tutum.Network, 0, 0)
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

				ifs = append(ifs, tutum.Network{Name: name, CIDR: cidr})
			}
		}
	}
	return ifs
}

func sendData(url string, data []byte) error {
	httpClient := &http.Client{}
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(data))
	if err != nil {
		log.Println(err)
		return err
	}
	tutumAuth := os.Getenv("TUTUM_AUTH")
	if tutumAuth != "" {
		req.Header.Add("Authorization", tutumAuth)
	}
	req.Header.Add("User-Agent", "weave-daemon/"+Version)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("Send metrics failed: %s", resp.Status)
		if resp.StatusCode >= 500 {
			return errors.New(resp.Status)
		}
	}
	return nil
}

func Send(url string, data []byte) {
	counter := 1
	for {
		err := sendData(url, data)
		if err == nil {
			break
		} else {
			if counter > 100 {
				log.Println("Too many reties, give up")
				break
			} else {
				counter *= 2
				log.Printf("%s: Retry in %d seconds", err, counter)
				time.Sleep(time.Duration(counter) * time.Second)
			}
		}
	}
}

func PostInterfaceData(url string) {
	interfaces := getInterfaces()
	Tutum_Node_CIDR = interfaces

	data := PostForm{Interfaces: interfaces}
	json, err := json.Marshal(data)
	if err != nil {
		log.Fatal("Cannot marshal the interface data: %v\n", data)
	}

	log.Printf("Posting to %s with %s", url, string(json))
	Send(url, json)
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

func CIDRToIP(array []string) []string {
	IpArray := []string{}
	for _, elem := range array {
		IP, _, err := net.ParseCIDR(elem)
		if err != nil {
			log.Println(err)
		}
		IpArray = append(IpArray, IP.String())
	}
	return IpArray
}

func CheckIfSameNetwork(cidr1 string, cidr2 string) bool {
	if cidr1 != cidr2 {
		_, ipNet1, err1 := net.ParseCIDR(cidr1)

		if err1 != nil {
			log.Println(err1)
		}

		_, ipNet2, err2 := net.ParseCIDR(cidr2)

		if err2 != nil {
			log.Println(err2)
		}

		if string(ipNet1.IP) == string(ipNet2.IP) {
			return true
		}
	}
	return false
}

func NodeAppend(nodeList tutum.NodeListResponse) ([]string, []string) {
	networkAvailable := make(map[string]NodeNetwork)
	node_public_ips := []string{}
	node_private_ips := []string{}

	for i := range nodeList.Objects {
		state := nodeList.Objects[i].State
		if state == "Deployed" || state == "Unreachable" {
			networkAvailable[nodeList.Objects[i].Uuid] = NodeNetwork{cidrs: nodeList.Objects[i].Private_ips, Public_Ip: nodeList.Objects[i].Public_ip}
		}
	}

	temp := []string{}
	for _, value := range networkAvailable {
		temp1 := []string{}
		if len(value.cidrs) > 0 {
			for _, networkAvailableCIDR := range value.cidrs {
			Loop1:
				for _, network := range Tutum_Node_CIDR {
					if os.Getenv("TUTUM_PRIVATE_CIDR") != "" {
						if CheckIfSameNetwork(os.Getenv("TUTUM_PRIVATE_CIDR"), networkAvailableCIDR.CIDR) {
							temp1 = append(node_private_ips, networkAvailableCIDR.CIDR)
							break Loop1
						}
					} else {
						if CheckIfSameNetwork(network.CIDR, networkAvailableCIDR.CIDR) {
							temp1 = append(node_private_ips, networkAvailableCIDR.CIDR)
							break Loop1
						}
					}
				}
				if len(temp1) == 0 && value.Public_Ip != Tutum_Node_Public_Ip {
					node_public_ips = append(node_public_ips, value.Public_Ip)
				} else {
					temp = append(temp, temp1...)
				}
			}
		} else {
			if value.Public_Ip != Tutum_Node_Public_Ip {
				node_public_ips = append(node_public_ips, value.Public_Ip)
			}
		}
	}
	if len(temp) > 0 {
		node_private_ips = append(node_private_ips, temp...)
	}

	node_private_ips = CIDRToIP(node_private_ips)
	return removeDuplicates(node_public_ips), removeDuplicates(node_private_ips)
}

func DiscoverPeers() error {
	tries := 0
	log.Println("[NODE DISCOVERY STARTED]")
	for {
		nodeList, err := tutum.ListNodes()
		if err != nil {
			time.Sleep(60 * time.Second)
			return err
		}

		if len(nodeList.Objects) == 0 {
			return nil
		}

		node_public_ips, node_private_ips := NodeAppend(nodeList)

		log.Println("[NODE DISCOVERY]: Current nodes available")
		log.Printf("Private Network: %s", node_private_ips)
		log.Printf("Public Network: %s", node_public_ips)

		var diff1 []string

		//Checking if there are nodes that are not in the peer_ips list
		diff1 = compareNodePeer(node_private_ips, peer_ips, diff1)

		for _, i := range diff1 {
			err := connectToPeers(i)
			if err != nil {
				tries++
				if tries > 3 {
					return err
				}
			}
		}

		var diff3 []string

		//Checking if there are nodes that are not in the peer_ips list

		diff3 = compareNodePeer(node_public_ips, peer_ips_public, diff3)

		for _, i := range diff3 {
			err := connectToPeers(i)
			if err != nil {
				tries++
				if tries > 3 {
					return err
				}
			}
		}

		//IF TERMINATED EVENT
		var diff2 []string

		//Checking if there are peers that are not in the node_private_ips list
		diff2 = compareNodePeer(peer_ips, node_private_ips, diff2)

		for _, i := range diff2 {
			err := forgetPeers(i)
			if err != nil {
				tries++
				if tries > 3 {
					return err
				}
			}
		}

		var diff4 []string

		//Checking if there are peers that are not in the node_private_ips list
		diff4 = compareNodePeer(peer_ips_public, node_public_ips, diff4)

		for _, i := range diff4 {
			err := forgetPeers(i)
			if err != nil {
				tries++
				if tries > 3 {
					return err
				}
			}
		}

		peer_ips = node_private_ips
		peer_ips_public = node_public_ips
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
