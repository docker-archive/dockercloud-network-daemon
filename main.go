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
		return err
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

			_, err := cmd.StdoutPipe()
			if err != nil {
				return err
			}

			if err := cmd.Start(); err != nil {
				return err
			}

			if err := cmd.Wait(); err != nil {
				tries++

				if tries > 3 {
					break
				}
				return err
			} else {
				break
			}
		}
	} else {
		log.Printf("%s: cannot find the IP address to add to weave", container_id)
	}
	return nil
}

func ContainerAttachThread(c *docker.Client) error {

	listener := make(chan *docker.APIEvents)

	containers, err := c.ListContainers(docker.ListContainersOptions{All: false, Size: true, Limit: 0, Since: "", Before: ""})
	if err != nil {
		return err
	}

	for _, container := range containers {
		err := AttachContainer(c, container.ID)
		if err != nil {
			return err
		}

	}
	err = c.AddEventListener(listener)
	if err != nil {
		return err
	}

	defer func() error {

		err = c.RemoveEventListener(listener)
		if err != nil {
			return err
		}
		return nil
	}()

	timeout := time.After(1 * time.Second)

	for {
		select {
		case msg := <-listener:
			if msg.Status == "start" && !strings.HasPrefix(msg.From, "weaveworks/weave") {
				err := AttachContainer(c, msg.ID)
				if err != nil {
					log.Println(err)
					break
				}
			}
		case <-timeout:
			break
		}
	}
}

func discovering() {
	c := make(chan tutum.Event)
	e := make(chan error)
	nodes.DiscoverPeers()
	go tutum.TutumEvents(c, e)
Loop:
	for {
		select {
		case event := <-c:
			if event.Type == "node" && (event.State == "Deployed" || event.State == "Terminated") {
				err := nodes.DiscoverPeers()
				if err != nil {
					log.Println(err)
				}
			}
		case err := <-e:
			log.Println(err)
			go discovering()
			break Loop
		}
	}
}

func containerThread(client *docker.Client) {
	tries := 0
	if tries > 3 {
		time.Sleep(30 * time.Second)
	}
	err := ContainerAttachThread(client)
	if err != nil {
		log.Println("ATTACH THREAD ERR")
		log.Println(err)
		containerThread(client)
		tries++
		time.Sleep(1500 * time.Millisecond)
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
		log.Println(err)
	}

	node, err := tutum.GetNode(nodes.Tutum_Node_Api_Uri)
	if err != nil {
		log.Println(err)
	}
	nodes.Tutum_Node_Public_Ip = node.Public_ip
	log.Printf("This node IP is %s", nodes.Tutum_Node_Public_Ip)
	if os.Getenv("TUTUM_AUTH") != "" {
		log.Println("Detected Tutum API access - starting peer discovery thread")
		go discovering()
	}
	containerThread(client)
}
