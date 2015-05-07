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
		log.Println("Inspecting error")
		log.Printf("%s: exception when inspecting the container", err)
	}

	cidr := ""
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
				log.Println("5")
				log.Println(err)
			}

			stderr, _ := cmd.StderrPipe()
			/*if err != nil {
				log.Println(err)
			}*/

			if err := cmd.Start(); err != nil {
				log.Println("6")
				log.Println(err)
			}

			if err := cmd.Wait(); err != nil {
				log.Println("7")
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
	return nil
}

func ContainerAttachThread(c *docker.Client) {

	listener := make(chan *docker.APIEvents)

	containers, err := c.ListContainers(docker.ListContainersOptions{All: false, Size: true, Limit: 0, Since: "", Before: ""})
	if err != nil {
		log.Println("1")
		log.Println(err)
	}

	for _, container := range containers {
		err := AttachContainer(c, container.ID)
		if err != nil {
			log.Println("2")
			log.Println(err)
		}

	}
	err = c.AddEventListener(listener)
	if err != nil {
		log.Println("3")
		log.Println(err)
	}

	defer func() {

		err = c.RemoveEventListener(listener)
		if err != nil {
			log.Println("4")
			log.Println(err)
		}

	}()

	timeout := time.After(1 * time.Second)

	for {
		select {
		case msg := <-listener:
			if msg.Status == "EOF" {
				log.Println("EOF INCOMING")
				c.RemoveEventListener(listener)
				connectToDocker()
				break
			}

			if msg.Status == "start" && !strings.HasPrefix(msg.From, "weaveworks/weave") {
				AttachContainer(c, msg.ID)
			}
		case <-timeout:
			break
		}
	}
}

func discovering() {
	c := make(chan tutum.Event)
	nodes.DiscoverPeers()
	go tutum.TutumEvents(c)
	for {
		events := <-c
		nodes.EventHandler(events)
	}
}

func connectToDocker() (*docker.Client, error) {
	endpoint := "unix:///var/run/docker.sock"

	client, err := docker.NewClient(endpoint)

	if err != nil {
		log.Println(err)
	}
	return client, nil
}

func main() {

	log.Println("Start running daemon")

	//Init Docker client
	client, err := connectToDocker()
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
		go discovering()
	}
	ContainerAttachThread(client)
}
