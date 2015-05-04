package docker

import (
	"log"
	"os/exec"
	"strings"
	"time"
)

func (c *Client) AttachContainer(container_id string) error {
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

func (c *Client) ContainerAttachThread() {

	listener := make(chan *APIEvents)

	containers, err := c.ListContainers(ListContainersOptions{All: false, Size: true, Limit: 0, Since: "", Before: ""})
	if err != nil {
		log.Fatal(err)
	}

	for _, container := range containers {
		//log.Println(container.ID, container.Names)
		err := c.AttachContainer(container.ID)
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
				c.AttachContainer(msg.ID)
				//DEBUG
				//fmt.Println("attached")
			}
		case <-timeout:
			break
		}
	}
}
