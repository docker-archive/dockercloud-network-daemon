package main // import "github.com/docker/dockercloud-network-daemon"

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/docker/dockercloud-network-daemon/nodes"
	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/fsouza/go-dockerclient"
)

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
		err := containers.ContainerAttachThread(client)
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
		dockercloud.SetUserAgent("network-daemon/" + tools.Version)
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
