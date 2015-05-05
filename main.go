package main // import "github.com/tutumcloud/weave-daemon"

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/tutumcloud/go-tutum/tutum"
	"github.com/tutumcloud/weave-daemon/nodes"
)

func AttachContainer(c *docker.Client, container_id string) error {
	inspect, err := c.InspectContainer(container_id)

	if err != nil {
		log.Printf("%s: exception when inspecting the container", err)
	}

	cidr := ""
	//log.Println(inspect)
	env_vars := inspect.Config.Env

	for i := range env_vars {
		if strings.HasPrefix(env_vars[i], "TUTUM_IP_ADDRESS=") {
			cidr = env_vars[i][len("TUTUM_IP_ADDRESS="):]
			break
		}
	}

	if cidr != "" {
		tries := 0
		for tries < 3 {

			log.Printf("%s: adding to weave with IP %s", container_id, cidr)
			cmd := exec.Command("/weave", "--local", "attach", cidr, container_id)

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
				log.Printf("%s: %s %s", container_id, stdout, stderr)
				tries++
				time.Sleep(1)
			} else {
				break
			}
		}
	} else {
		log.Printf("%s: cannot find the IP address to add to weave", container_id)
	}

	if err != nil {
		return err
	}

	return nil
}

func ContainerAttachThread(c *docker.Client) {

	listener := make(chan *docker.APIEvents)

	containers, err := c.ListContainers(docker.ListContainersOptions{All: false, Size: true, Limit: 0, Since: "", Before: ""})
	if err != nil {
		log.Fatal(err)
	}

	for _, container := range containers {
		//log.Println(container.ID, container.Names)
		err := AttachContainer(c, container.ID)
		if err != nil {
			log.Println(err)
		}

	}
	err = c.AddEventListener(listener)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {

		err = c.RemoveEventListener(listener)
		if err != nil {
			log.Fatal(err)
		}

	}()

	timeout := time.After(1 * time.Second)

	for {
		select {
		case msg := <-listener:
			//DEBUG
			//log.Print(msg.Status + " " + msg.ID + " " + msg.From)
			if msg.Status == "start" && !strings.HasPrefix(msg.From, "weaveworks/weave") {
				AttachContainer(c, msg.ID)
				//DEBUG
				//fmt.Println("attached")
			}
		case <-timeout:
			break
		}
	}
}

func main() {

	//Init client

	//BOOT2DOCKER NEW TLS CLIENT
	/*endpoint := "tcp://192.168.59.103:2376"
	path := os.Getenv("DOCKER_CERT_PATH")
	ca := fmt.Sprintf("%s/ca.pem", path)
	cert := fmt.Sprintf("%s/cert.pem", path)
	key := fmt.Sprintf("%s/key.pem", path)
	client, err := docker.NewTLSClient(endpoint, cert, key, ca)*/

	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)

	if err != nil {
		log.Fatal(err)
	}

	node, err := tutum.GetNode(nodes.Tutum_Node_Api_Uri)
	if err != nil {
		log.Fatal(err)
	}
	nodes.Tutum_Node_Public_Ip = node.Public_ip
	log.Printf("This node IP is %s", nodes.Tutum_Node_Public_Ip)
	if os.Getenv("TUTUM_AUTH") != "" {
		log.Println("Detected Tutum API access - starting peer discovery thread")
		c := make(chan tutum.Event)
		go tutum.TutumEvents(c)
		for {
			events := <-c
			nodes.EventHandler(events)
			nodes.DiscoverPeers()
		}
	}
	ContainerAttachThread(client)
}
