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
	"time"

	"github.com/docker/dockercloud-network-daemon/tools"
	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/getsentry/raven-go"
)

//NodeNetwork type
type NodeNetwork struct {
	PublicIP string
	cidrs    []dockercloud.Network
	region   string
}

//PostForm type contains the interfaces of the current node to be PATCHed
type PostForm struct {
	Interfaces []dockercloud.Network `json:"private_ips"`
}

const (
	Version = "1.0.3"
)

var (
	//NodeAPIURI resource uri of the current node
	NodeAPIURI = os.Getenv("DOCKERCLOUD_NODE_API_URI")
	//NodePublicIP public IP of the current node
	NodePublicIP = ""
	//NodeCIDR private IPs of the current node
	NodeCIDR = []dockercloud.Network{}
	//NodeUUID UUID of the current node
	NodeUUID = ""
	//Region region of the current node
	Region        = ""
	peerIps       = []string{}
	peerIpsPublic = []string{}
)

func sendData(url string, data []byte) error {
	httpClient := &http.Client{}
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(data))
	if err != nil {
		log.Println(err)
		return err
	}
	dcAuth := os.Getenv("DOCKERCLOUD_AUTH")
	if dcAuth != "" {
		req.Header.Add("Authorization", dcAuth)
	}
	req.Header.Add("User-Agent", "network-daemon/"+tools.Version)

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

//Send sends PATCH request on the database to update the current node with its private IPs
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

//PostInterfaceData triggers Send function
func PostInterfaceData(url string) {
	interfaces := tools.GetInterfaces()
	NodeCIDR = interfaces

	data := PostForm{Interfaces: interfaces}
	json, err := json.Marshal(data)
	if err != nil {
		log.Printf("Cannot marshal the interface data: %v\n", data)
	}

	log.Printf("Posting to %s with %s", url, string(json))
	Send(url, json)
}

//CIDRToIP converts array of CIDRs to array of IPs
func CIDRToIP(array []string) []string {
	ipArray := []string{}
	for _, elem := range array {
		IP, _, err := net.ParseCIDR(elem)
		if err != nil {
			log.Println(err)
		}
		ipArray = append(ipArray, IP.String())
	}
	return ipArray
}

//IsInPrivateRange check if the requested CIDR is in the private IP range
func IsInPrivateRange(cidr string) bool {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Println(err)
	}

	_, ipNet2, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		log.Println(err)
	}

	_, ipNet3, err := net.ParseCIDR("172.16.0.0/12")
	if err != nil {
		log.Println(err)
	}

	_, ipNet4, err := net.ParseCIDR("192.168.0.0/16")
	if err != nil {
		log.Println(err)
	}

	if ipNet2.Contains(ip) || ipNet3.Contains(ip) || ipNet4.Contains(ip) {
		return true
	}

	return false
}

//CheckIfSameNetwork returns true if one of the CIDR contains the other, otherwise returns false
func CheckIfSameNetwork(cidr1 string, cidr2 string) bool {
	ip1, ipNet1, err1 := net.ParseCIDR(cidr1)
	if err1 != nil {
		log.Println(err1)
	}

	ip2, ipNet2, err2 := net.ParseCIDR(cidr2)
	if err2 != nil {
		log.Println(err2)
	}

	if ipNet1.Contains(ip2) || ipNet2.Contains(ip1) {
		return true
	}
	return false
}

//NodeAppend returns the list of Public and Private IPs
func NodeAppend(nodeList dockercloud.NodeListResponse) ([]string, []string) {
	networkAvailable := make(map[string]NodeNetwork)
	NodePublicIPs := []string{}
	nodePrivateIps := []string{}

	for i := range nodeList.Objects {
		state := nodeList.Objects[i].State
		if state == "Deployed" || state == "Unreachable" {
			networkAvailable[nodeList.Objects[i].Uuid] = NodeNetwork{cidrs: nodeList.Objects[i].Private_ips, PublicIP: nodeList.Objects[i].Public_ip, region: nodeList.Objects[i].Region}
		}
	}

	temp := []string{}
	for _, value := range networkAvailable {
		temp1 := []string{}
		if len(value.cidrs) > 0 {
			for _, networkAvailableCIDR := range value.cidrs {
			Loop1:
				for _, network := range NodeCIDR {
					if networkAvailableCIDR.CIDR != network.CIDR && IsInPrivateRange(networkAvailableCIDR.CIDR) && IsInPrivateRange(network.CIDR) {
						if os.Getenv("DOCKERCLOUD_PRIVATE_CIDR") != "" {
							if value.region == Region && CheckIfSameNetwork(os.Getenv("DOCKERCLOUD_PRIVATE_CIDR"), networkAvailableCIDR.CIDR) {
								temp1 = append(nodePrivateIps, networkAvailableCIDR.CIDR)
								break Loop1
							}
						} else {
							if CheckIfSameNetwork(network.CIDR, networkAvailableCIDR.CIDR) {
								temp1 = append(nodePrivateIps, networkAvailableCIDR.CIDR)
								break Loop1
							}
						}
					}
				}
				if len(temp1) == 0 && value.PublicIP != NodePublicIP {
					NodePublicIPs = append(NodePublicIPs, value.PublicIP)
				} else {
					temp = append(temp, temp1...)
				}
			}
		} else {
			if value.PublicIP != NodePublicIP {
				NodePublicIPs = append(NodePublicIPs, value.PublicIP)
			}
		}
	}
	if len(temp) > 0 {
		nodePrivateIps = append(nodePrivateIps, temp...)
	}

	nodePrivateIps = CIDRToIP(nodePrivateIps)
	return tools.RemoveDuplicates(NodePublicIPs), tools.RemoveDuplicates(nodePrivateIps)
}

//DiscoverPeers queries DockerCloud API for the list of nodes and checks if nodes must be attached or forgotten
func DiscoverPeers() error {
	tries := 0
	counter := 1
	log.Println("[NODE DISCOVERY STARTED]")
	for {
		nodeList, err := dockercloud.ListNodes()
		if err != nil {
			if counter > 100 {
				log.Println("Too many retries, give up")
				return err
			}
			counter *= 2
			log.Printf("%s: Retry in %d seconds", err, counter)
			time.Sleep(time.Duration(counter) * time.Second)
		} else {
			if len(nodeList.Objects) == 0 {
				return nil
			}

			NodePublicIPs, nodePrivateIps := NodeAppend(nodeList)

			log.Println("[NODE DISCOVERY]: Current nodes available")
			log.Printf("Private Network: %s", nodePrivateIps)
			log.Printf("Public Network: %s", NodePublicIPs)

			var diff1 []string

			//Checking if there are nodes that are not in the peerIps list
			diff1 = tools.CompareArrays(nodePrivateIps, peerIps, diff1)

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

			//Checking if there are nodes that are not in the peerIps list

			diff3 = tools.CompareArrays(NodePublicIPs, peerIpsPublic, diff3)

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

			//Checking if there are peers that are not in the nodePrivateIps list
			diff2 = tools.CompareArrays(peerIps, nodePrivateIps, diff2)

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

			//Checking if there are peers that are not in the nodePrivateIps list
			diff4 = tools.CompareArrays(peerIpsPublic, NodePublicIPs, diff4)

			for _, i := range diff4 {
				err := forgetPeers(i)
				if err != nil {
					tries++
					if tries > 3 {
						return err
					}
				}
			}

			peerIps = nodePrivateIps
			peerIpsPublic = NodePublicIPs
			break
		}
	}

	log.Println("[NODE DISCOVERY STOPPED]")
	return nil
}

func connectToPeers(nodeIP string) error {
	log.Println("[NODE DISCOVERY UPDATE]: Some nodes are not peers")
	tries := 0
Loop:
	for {

		log.Printf("[NODE DISCOVERY]: Connecting to newly discovered peer: %s", nodeIP)
		cmd := exec.Command("/weave", "--local", "connect", nodeIP)
		if output, err := cmd.CombinedOutput(); err != nil {
			packet := raven.Packet{Message: "Node Connect failed", Extra: map[string]interface{}{"output": string(output)}, Release: tools.Version}
			raven.Capture(&packet, map[string]string{"type": "nodeConnect", "nodeURI": NodeAPIURI})
			tries++
			time.Sleep(2 * time.Second)
			log.Println("[NODE DISCOVERY ERROR]: Unable to 'weave connect':", err, string(output))
			if tries > 3 {
				break Loop
			}
		} else {
			break Loop
		}
	}
	log.Println("[NODE DISCOVERY]: Discover Peers: done!")
	return nil
}

func forgetPeers(nodeIP string) error {
	log.Println("[NODE DISCOVERY UPDATE]: Some peers are not nodes anymore")
	tries := 0
Loop:
	for {
		log.Printf("[NODE DISCOVERY]: Forgetting peer: %s", nodeIP)
		cmd := exec.Command("/weave", "--local", "forget", nodeIP)
		if output, err := cmd.CombinedOutput(); err != nil {
			packet := raven.Packet{Message: "Node Forget failed", Extra: map[string]interface{}{"output": string(output)}, Release: tools.Version}
			raven.Capture(&packet, map[string]string{"type": "nodeForget", "nodeURI": NodeAPIURI})
			tries++
			time.Sleep(2 * time.Second)
			log.Println("[NODE DISCOVERY ERROR]: Unable to 'weave forget':", err, string(output))
			if tries > 3 {
				break Loop
			}
		} else {
			break Loop
		}
	}
	log.Println("[NODE DISCOVERY]: Forget Peers: done!")
	return nil
}
