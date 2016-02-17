package containers

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/docker/dockercloud-network-daemon/tools"
	"github.com/fsouza/go-dockerclient"
)

//Event struct for docker events
type Event struct {
	Node       string `json:"node,omitempty"`
	Status     string `json:"status"`
	ID         string `json:"id"`
	From       string `json:"from"`
	Time       int64  `json:"time"`
	HandleTime int64  `json:"handletime"`
	ExitCode   string `json:"exitcode"`
}

//AttachContainer inspects the container with the id containerID and weave attach it
func AttachContainer(c *docker.Client, containerID string) {
	log.Println("[CONTAINER ATTACH]: Inspecting Containers " + containerID)
	inspect, err := c.InspectContainer(containerID)

	if err != nil {
		log.Println("[CONTAINER ATTACH]: Inspecting Containers failed")
		return
	}

	log.Println("[CONTAINER ATTACH]: Attaching container " + containerID)

	cidr := ""
	envVars := inspect.Config.Env

	for i := range envVars {
		if strings.HasPrefix(envVars[i], "DOCKERCLOUD_IP_ADDRESS=") {
			cidr = envVars[i][len("DOCKERCLOUD_IP_ADDRESS="):]
			break
		}
	}

	if cidr != "" {
		tries := 0
	Loop:
		for {
			cmd := exec.Command("/weave", "--local", "attach", cidr, containerID)

			if _, err := cmd.StdoutPipe(); err != nil {
				break Loop
			}

			if _, err := cmd.StderrPipe(); err != nil {
				break Loop
			}

			if err := cmd.Start(); err != nil {
				tries++
				time.Sleep(2 * time.Second)
				log.Println("[CONTAINER ATTACH ERROR]: Start weave cmd failed:", err)
				if tries > 3 {
					break Loop
				}
			}

			if err := cmd.Wait(); err != nil {
				tries++
				time.Sleep(2 * time.Second)
				log.Println("[CONTAINER ATTACH ERROR]: Wait weave cmd failed:", err)
				if tries > 3 {
					break Loop
				}
			} else {
				log.Printf("[CONTAINER ATTACH]: Weave attach successful for %s with IP %s", containerID, cidr)
				break
			}
		}
	} else {
		log.Printf("[CONTAINER ATTACH]: Ignoring container %s - cannot find the IP address to add to weave", containerID)
	}
}

func monitorDockerEvents(c chan *Event, e chan error) {
	log.Println("docker events starts")
	cmd := exec.Command(tools.DockerPath, "events")
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

//ContainerAttachThread List containers on the current node and execute AttachContainer function at launch and whenever we receive a start event from docker
func ContainerAttachThread(c *docker.Client) error {
	var weaveID = ""
	listener := make(chan *Event)
	e := make(chan error)
	containerList := []string{}
	connectedContainerList := []string{}

	containers, err := c.ListContainers(docker.ListContainersOptions{All: false, Size: true, Limit: 0, Since: "", Before: ""})
	if err != nil {
		log.Println("[CONTAINER ATTACH THREAD ERROR]: Listing Containers failed")
		return err
	}

	for _, container := range containers {
		if !strings.HasPrefix(container.Image, "weaveworks/") {
			log.Println("[CONTAINER ATTACH THREAD]: Found running container with ID: " + container.ID)
			go AttachContainer(c, container.ID)
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
				go AttachContainer(c, msg.ID)
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

			containers, err := c.ListContainers(docker.ListContainersOptions{All: false, Size: true, Limit: 0, Since: "", Before: ""})
			if err != nil {
				log.Println("[CONTAINER ATTACH THREAD ERROR]: Listing Containers failed")
				return err
			}

			cmd, err := exec.Command("sh", "-c", "/weave --local ps | awk '{print $1}'").Output()
			if err != nil {
				log.Println(err)
			}

			output := strings.Split(string(cmd), "\n")

			for _, id := range output {
				if !strings.Contains(id, ":") && id != "" {
					connectedContainerList = append(connectedContainerList, id)
				}
			}

			for _, container := range containers {
				containerList = append(containerList, container.ID)
			}

			var containerToConnectList []string
			containerToConnectList = tools.CompareIDArrays(containerList, connectedContainerList, containerToConnectList)
			if len(containerToConnectList) > 0 {
				for _, id := range containerToConnectList {
					go AttachContainer(c, id)
				}
			}

			containerList = []string{}
			connectedContainerList = []string{}

			break
		}
	}
}
