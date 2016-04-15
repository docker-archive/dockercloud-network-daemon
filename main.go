package main // import "github.com/docker/dockercloud-network-daemon"

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/dockercloud-network-daemon/nodes"
	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/fsouza/go-dockerclient"
)

const (
	Version    = "1.0.3"
	DockerPath = "/usr/local/bin/docker"
)

type Event struct {
	Node       string `json:"node,omitempty"`
	Status     string `json:"status"`
	ID         string `json:"id"`
	From       string `json:"from"`
	Time       int64  `json:"time"`
	HandleTime int64  `json:"handletime"`
	ExitCode   string `json:"exitcode"`
}

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
		if strings.HasPrefix(env_vars[i], "DOCKERCLOUD_IP_ADDRESS=") {
			cidr = env_vars[i][len("DOCKERCLOUD_IP_ADDRESS="):]
			break
		}
	}

	if cidr != "" {
		tries := 0
		for {
			cmd := exec.Command("/weave", "--local", "attach", cidr, container_id)

			if output, err := cmd.CombinedOutput(); err != nil {
				tries++
				time.Sleep(2 * time.Second)
				log.Println("[CONTAINER ATTACH ERROR]: Weave attach failed:", err, string(output))
				if tries > 3 {
					return err
				}
			} else {
				log.Printf("[CONTAINER ATTACH]: Weave attach successful for %s: %s", container_id, string(output))
				break
			}
		}
	} else {
		log.Printf("[CONTAINER ATTACH]: Ignoring container %s - cannot find the IP address to add to weave", container_id)
	}
	return nil
}

func monitorDockerEvents(c chan *Event, e chan error) {
	log.Println("docker events starts")
	cmd := exec.Command(DockerPath, "events")
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		e <- err
	}

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			eventStr := scanner.Text()
			event := parseEvent(eventStr)
			if event != nil {
				c <- event
			}
		}
		if scanner.Err() == nil {
			e <- err
		} else {
			e <- err
		}
	}()

	err = cmd.Start()
	if err != nil {
		e <- err
	}

	err = cmd.Wait()
	if err != nil {
		e <- err
	}
	log.Println("docker events stops")
}

func parseEvent(eventStr string) (event *Event) {
	if eventStr == "" {
		return nil
	}

	re := regexp.MustCompile("(.*) (.{64}): \\(from (.*)\\) (.*)")
	terms := re.FindStringSubmatch(eventStr)
	if len(terms) == 5 {
		var event Event
		eventTime, err := time.Parse(time.RFC3339Nano, terms[1])
		if err == nil {
			event.Time = eventTime.UnixNano()
		} else {
			event.Time = time.Now().UnixNano()
		}
		event.ID = terms[2]
		event.From = terms[3]
		event.Status = terms[4]
		event.HandleTime = time.Now().UnixNano()
		return &event
	}

	// for docker event 1.10 or above
	re = regexp.MustCompile("(.*) container (\\w*) (.{64}) \\((.*)\\)")
	terms = re.FindStringSubmatch(eventStr)
	if len(terms) == 5 {
		var event Event
		eventTime, err := time.Parse(time.RFC3339Nano, terms[1])
		if err == nil {
			event.Time = eventTime.UnixNano()
		} else {
			event.Time = time.Now().UnixNano()
		}
		event.ID = terms[3]
		event.Status = terms[2]
		event.HandleTime = time.Now().UnixNano()

		if terms[4] != "" {
			attrs := strings.Split(terms[4], ",")
			for _, attr := range attrs {
				attr = strings.TrimSpace(attr)
				if strings.HasPrefix(strings.ToLower(attr), "image=") && len(attr) > 6 {
					event.From = attr[6:]
				}
			}
		}
		return &event
	}

	return nil
}

func ContainerAttachThread(c *docker.Client) error {
	var weaveID = ""
	listener := make(chan *Event)
	e := make(chan error)
	containerAttached := make(map[string]string)
	containerList := []string{}

	containers, err := c.ListContainers(docker.ListContainersOptions{All: false, Size: false, Limit: 0, Since: "", Before: ""})
	if err != nil {
		log.Println("[CONTAINER ATTACH THREAD ERROR]: Listing Containers failed")
		return err
	}

	for _, container := range containers {
		if !strings.HasPrefix(container.Image, "weaveworks/") {
			runningContainer, err := c.InspectContainer(container.ID)
			if err != nil {
				return err
			}
			log.Println("[CONTAINER ATTACH THREAD]: Found running container with ID: " + container.ID)

			err = AttachContainer(c, container.ID)
			if err != nil {
				log.Println("[CONTAINER ATTACH THREAD ERROR]: Attaching Containers failed")
				return err
			}
			containerAttached[container.ID] = runningContainer.State.StartedAt.Format(time.RFC3339)
		}

		if strings.HasPrefix(container.Image, "weaveworks/weave:") {
			weaveID = container.ID
		}
	}

	go monitorDockerEvents(listener, e)

	if weaveID == "" {
		os.Exit(1)
	}

	log.Println("WEAVE ID is : " + weaveID)

	for {
		timeout := time.Tick(2 * time.Minute)
		select {
		case msg := <-listener:
			if msg.Status == "die" && msg.ID == weaveID {
				os.Exit(1)
			}
			if msg.Status == "start" && !strings.HasPrefix(msg.From, "weaveworks/") {
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
		case err := <-e:
			return err
		case <-timeout:

			weave, err := c.InspectContainer(weaveID)
			if err != nil {
				return err
			}

			if weave.State.Running != true {
				os.Exit(1)
			}

			containers, err := c.ListContainers(docker.ListContainersOptions{All: false, Size: false, Limit: 0, Since: "", Before: ""})
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
				}
				break
			}
			break
		}
	}
}

func nodeEventHandler(eventType string, state string, action string) error {
	if (eventType == "node" && state == "Deployed" && action == "update") || (eventType == "node" && state == "Terminated") {
		err := nodes.DiscoverPeers()
		if err != nil {
			return err
		}
	}
	return nil
}

func tutumEventHandler(wg *sync.WaitGroup, c chan dockercloud.Event, e chan error) {
Loop:
	for {
		select {
		case event := <-c:
			err := nodeEventHandler(event.Type, event.State, event.Action)
			if err != nil {
				log.Println(err)
			}
			break
		case err := <-e:
			log.Println("[NODE DISCOVERY ERROR]: " + err.Error())
			time.Sleep(10 * time.Second)
			wg.Add(1)
			go discovering(wg)
			break Loop
		}
	}
}

func discovering(wg *sync.WaitGroup) {
	defer wg.Done()
	c := make(chan dockercloud.Event)
	e := make(chan error)

	nodes.DiscoverPeers()

	go dockercloud.Events(c, e)
	tutumEventHandler(wg, c, e)
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
	log.Println("===> Start running daemon")
	wg := &sync.WaitGroup{}
	wg.Add(2)
	//Init Docker client
	client, err := connectToDocker()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println("===> Starting container discovery goroutine")
	go containerThread(client, wg)

	tries := 0
	if nodes.Node_Api_Uri != "" {
		dockercloud.SetUserAgent("network-daemon/" + Version)
	Loop:
		for {
			node, err := dockercloud.GetNode(nodes.Node_Api_Uri)
			if err != nil {
				tries++
				log.Println(err)
				time.Sleep(5 * time.Second)
				if tries > 3 {
					time.Sleep(60 * time.Second)
					tries = 0
				}
				continue Loop
			} else {
				nodes.Region = node.Region
				nodes.Node_Public_Ip = node.Public_ip
				nodes.Node_Uuid = node.Uuid

				log.Println("===> Posting interface data to database")
				nodes.PostInterfaceData(os.Getenv("DOCKERCLOUD_REST_HOST") + nodes.Node_Api_Uri)

				log.Printf("This node IP is %s", nodes.Node_Public_Ip)
				if os.Getenv("DOCKERCLOUD_AUTH") != "" {
					log.Println("===> Detected Dockecloud API access - starting peer discovery goroutine")
					go discovering(wg)
					break Loop
				}
			}
		}
	}
	wg.Wait()
}
