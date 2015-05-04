package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tutumcloud/weave-daemon/docker"
)

func main() {

	//Init client

	//BOOT2DOCKER NEW TLS CLIENT
	endpoint := "tcp://192.168.59.103:2376"
	path := os.Getenv("DOCKER_CERT_PATH")
	ca := fmt.Sprintf("%s/ca.pem", path)
	cert := fmt.Sprintf("%s/cert.pem", path)
	key := fmt.Sprintf("%s/key.pem", path)
	client, err := docker.NewTLSClient(endpoint, cert, key, ca)

	/*
		endpoint := "unix:///var/run/docker.sock"
		client, err := docker.NewClient(endpoint)
	*/
	if err != nil {
		log.Fatal(err)
	}

	/*node, err := tutum.GetNode(nodes.Tutum_Node_Api_Uri)
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
	}*/
	client.ContainerAttachThread()
}
