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
		log.Println("Inspecting Containers failed")
		return err
	}

	cidr := ""
	env_vars := inspect.Config.Env

	for i := range env_vars {
		log.Println("Checking containers ENV")
		if strings.HasPrefix(env_vars[i], "TUTUM_IP_ADDRESS=") {
			cidr = env_vars[i][len("TUTUM_IP_ADDRESS="):]
			break
		}
	}

	if cidr != "" {
		tries := 0
		for {
			log.Printf("%s: adding to weave with IP %s", container_id, cidr)
			cmd := exec.Command("/weave", "--local", "attach", cidr, container_id)

			_, err := cmd.StdoutPipe()
			if err != nil {
				tries++
				time.Sleep(2 * time.Second)
				log.Println("Stdout error")
				if tries > 3 {
					return err
				}
			}

			if err := cmd.Start(); err != nil {
				tries++
				time.Sleep(2 * time.Second)
				log.Println("Start weave cmd failed")
				if tries > 3 {
					return err
				}
			}

			if err := cmd.Wait(); err != nil {
				tries++
				time.Sleep(2 * time.Second)
				log.Println("Wait weave cmd failed")
				if tries > 3 {
					return err
				}
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
		log.Println("Listing Containers failed")
		return err
	}

	for _, container := range containers {
		log.Println("Attaching Containers")
		err := AttachContainer(c, container.ID)
		if err != nil {
			log.Println("Attaching Containers failed")
			return err
		}
		time.Sleep(2 * time.Second)
	}

	err = c.AddEventListener(listener)
	if err != nil {
		log.Println("Listening Containers Events failed")
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
		log.Println("LOOP: Container")
		select {
		case msg := <-listener:
			log.Println("New Container Event")
			if msg.Status == "start" && !strings.HasPrefix(msg.From, "weaveworks/weave") {
				err := AttachContainer(c, msg.ID)
				if err != nil {
					log.Println("[CONTAINER ATTACH THREAD ERROR]: " + err.Error())
					break
				}
				time.Sleep(2 * time.Second)
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
	for {
		log.Println("LOOP: Node")
		select {
		case event := <-c:
			log.Println("New Node Event")
			if event.Type == "node" && (event.State == "Deployed" || event.State == "Terminated") {
				err := nodes.DiscoverPeers()
				if err != nil {
					log.Println(err)
				}
				time.Sleep(2 * time.Second)
			}
			break
		case err := <-e:
			log.Println("[NODE DISCOVERY ERROR]: " + err.Error())
			time.Sleep(5 * time.Second)
			go discovering()
			break Loop
		}
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
		os.Exit(1)
	}

	node, err := tutum.GetNode(nodes.Tutum_Node_Api_Uri)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	nodes.Tutum_Node_Public_Ip = node.Public_ip
	log.Printf("This node IP is %s", nodes.Tutum_Node_Public_Ip)
	if os.Getenv("TUTUM_AUTH") != "" {
		log.Println("Detected Tutum API access - starting peer discovery thread")
		go discovering()
	}
	for {
		err := ContainerAttachThread(client)
		if err != nil {
			log.Println(err)
			time.Sleep(15 * time.Second)
		}
	}
}
