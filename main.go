package main // import "github.com/tutumcloud/weave-daemon"

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/tutumcloud/go-tutum/tutum"
	"github.com/tutumcloud/weave-daemon/nodes"
)

const version = "0.15.2"

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func inHashWithValue(containerAttached map[string]string, id string, value string) bool {
	if val, ok := containerAttached[id]; ok && val == value {
		return true
	}
	return false
}

func removeMissing(containerAttached map[string]string, containerList []string) map[string]string {
	for k, _ := range containerAttached {
		if !stringInSlice(k, containerList) {
			delete(containerAttached, k)
		}
	}
	return containerAttached
}

func AttachContainer(c *docker.Client, container_id string) error {
	log.Println("[CONTAINER ATTACH]: Inspecting Containers " + container_id)
	inspect, err := c.InspectContainer(container_id)

	if err != nil {
		log.Println("[CONTAINER ATTACH]: Inspecting Containers failed")
		return err
	}

	log.Println("[CONTAINER ATTACH]: Attaching container " + container_id)

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
		for {

			cmd := exec.Command("/weave", "--local", "attach", cidr, container_id)

			_, err := cmd.StdoutPipe()
			if err != nil {
				return err
			}

			if err := cmd.Start(); err != nil {
				tries++
				time.Sleep(2 * time.Second)
				log.Println("[CONTAINER ATTACH ERROR]: Start weave cmd failed")
				if tries > 3 {
					return err
				}
			}

			if err := cmd.Wait(); err != nil {
				tries++
				time.Sleep(2 * time.Second)
				log.Println("[CONTAINER ATTACH ERROR]: Wait weave cmd failed")
				if tries > 3 {
					return err
				}
			} else {
				log.Printf("%s: adding to weave with IP %s", container_id, cidr)
				break
			}
		}
	} else {
		log.Printf("%s: cannot find the IP address to add to weave", container_id)
	}
	return nil
}

func ContainerAttachThread(c *docker.Client) error {
	var weaveID = ""
	listener := make(chan *docker.APIEvents)
	containerAttached := make(map[string]string)
	containerList := []string{}

	containers, err := c.ListContainers(docker.ListContainersOptions{All: false, Size: true, Limit: 0, Since: "", Before: ""})
	if err != nil {
		log.Println("[CONTAINER ATTACH THREAD ERROR]: Listing Containers failed")
		return err
	}

	for _, container := range containers {

		runningContainer, err := c.InspectContainer(container.ID)

		if err != nil {
			return err
		}

		log.Println("[CONTAINER ATTACH THREAD]: Found running container with ID: " + container.ID)

		if strings.HasPrefix(container.Image, "weaveworks/weave:") {
			weaveID = container.ID
		}

		err = AttachContainer(c, container.ID)
		if err != nil {
			log.Println("[CONTAINER ATTACH THREAD ERROR]: Attaching Containers failed")
			return err
		}
		containerAttached[container.ID] = runningContainer.State.StartedAt.Format(time.RFC3339)
	}

	err = c.AddEventListener(listener)
	if err != nil {
		log.Println("[CONTAINER ATTACH THREAD ERROR]: Listening Containers Events failed")
		return err
	}

	defer func() error {

		err = c.RemoveEventListener(listener)
		if err != nil {
			return err
		}
		return nil
	}()

	if weaveID == "" {
		os.Exit(1)
	}

	log.Println("WEAVE ID is : " + weaveID)

	for {
		timeout := time.Tick(2 * time.Minute)
		select {
		case msg := <-listener:
			if msg.Status == "die" && strings.HasPrefix(msg.From, "weaveworks/weave:") {
				os.Exit(1)
			}
			if msg.Status == "start" {
				startingContainer, err := c.InspectContainer(msg.ID)

				if err != nil {
					return err
				}

				if inHashWithValue(containerAttached, msg.ID, startingContainer.State.StartedAt.Format(time.RFC3339)) {
					break
				} else {
					err := AttachContainer(c, msg.ID)
					if err != nil {
						log.Println("[CONTAINER ATTACH THREAD ERROR]: " + err.Error())
						break
					}
					containerAttached[msg.ID] = startingContainer.State.StartedAt.Format(time.RFC3339)
				}
			}
		case <-timeout:

			weave, err := c.InspectContainer(weaveID)
			if err != nil {
				return err
			}

			if weave.State.Running != true {
				os.Exit(1)
			}

			containers, err := c.ListContainers(docker.ListContainersOptions{All: false, Size: true, Limit: 0, Since: "", Before: ""})
			if err != nil {
				log.Println("[CONTAINER ATTACH THREAD ERROR]: Listing Containers failed")
				return err
			}

			for _, container := range containers {

				containerList = append(containerList, container.ID)

				containerAttached = removeMissing(containerAttached, containerList)

				containerList = []string{}

				runningContainer, err := c.InspectContainer(container.ID)

				if err != nil {
					return err
				}
				if inHashWithValue(containerAttached, container.ID, runningContainer.State.StartedAt.Format(time.RFC3339)) {
					break
				} else {
					log.Println("[CONTAINER ATTACH THREAD]: Found running container with ID: " + container.ID)
					err := AttachContainer(c, container.ID)
					if err != nil {
						log.Println("[CONTAINER ATTACH THREAD ERROR]: Attaching Containers failed")
						return err
					}

					containerAttached[container.ID] = runningContainer.State.StartedAt.Format(time.RFC3339)

					err = c.AddEventListener(listener)
					if err != nil {
						log.Println("[CONTAINER ATTACH THREAD ERROR]: Listening Containers Events failed")
						return err
					}

					defer func() error {

						err = c.RemoveEventListener(listener)
						if err != nil {
							return err
						}
						return nil
					}()
				}
				break
			}
			break
		}
	}
}

func nodeEventHandler(eventType string, state string) error {
	if eventType == "node" && (state == "Deployed" || state == "Terminated") {
		err := nodes.DiscoverPeers()
		if err != nil {
			return err
		}
	}
	return nil
}

func discovering(wg *sync.WaitGroup) {
	defer wg.Done()
	c := make(chan tutum.Event)
	e := make(chan error)

	nodes.DiscoverPeers()

	go tutum.TutumEvents(c, e)
Loop:
	for {
		select {
		case event := <-c:
			err := nodeEventHandler(event.Type, event.State)
			if err != nil {
				log.Println(err)
			}
			break
		case err := <-e:
			log.Println("[NODE DISCOVERY ERROR]: " + err.Error())
			time.Sleep(5 * time.Second)
			wg.Add(1)
			go discovering(wg)
			break Loop
		}
	}
}

func containerThread(client *docker.Client, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		err := ContainerAttachThread(client)
		if err != nil {
			log.Println(err)
			time.Sleep(15 * time.Second)
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
	wg := &sync.WaitGroup{}
	wg.Add(2)
	//Init Docker client
	client, err := connectToDocker()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println("Starting container discovery goroutine")
	go containerThread(client, wg)

	tries := 0
	if nodes.Tutum_Node_Api_Uri != "" {
		tutum.SetUserAgent("weave-daemon/" + version)
	Loop:
		for {
			node, err := tutum.GetNode(nodes.Tutum_Node_Api_Uri)
			if err != nil {
				tries++
				log.Println(err)
				time.Sleep(5 * time.Second)
				if tries > 3 {
					time.Sleep(30 * time.Second)
					tries = 0
				}
				continue Loop
			} else {
				nodes.Tutum_Node_Public_Ip = node.Public_ip
				log.Printf("This node IP is %s", nodes.Tutum_Node_Public_Ip)
				if os.Getenv("TUTUM_AUTH") != "" {
					log.Println("Detected Tutum API access - starting peer discovery goroutine")
					go discovering(wg)
					break Loop
				}
			}
		}
	}
	wg.Wait()
}
