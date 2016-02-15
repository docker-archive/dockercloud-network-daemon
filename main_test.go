package main

import "testing"

func Test_nodeEventHandler(t *testing.T) {
	eventType1 := "node"
	eventType2 := "service"
	state1 := "Deployed"
	state2 := "Deploying"
	state3 := "Terminated"
	action1 := "create"
	action2 := "update"
	Msg := "Couldn't find any DockerCloud credentials in ~/.docker/config.json or environment variables DOCKERCLOUD_USER and DOCKERCLOUD_APIKEY"

	err := nodeEventHandler(eventType1, state1, action2)
	err2 := nodeEventHandler(eventType2, state1, action2)
	err3 := nodeEventHandler(eventType2, state3, action2)
	err4 := nodeEventHandler(eventType1, state2, action2)
	err5 := nodeEventHandler(eventType1, state3, action1)

	if err2 != nil {
		t.Error("Expected empty error message, got ", err2.Error())
	}
	if err3 != nil {
		t.Error("Expected empty error message, got ", err3.Error())
	}
	if err4 != nil {
		t.Error("Expected empty error message, got ", err4.Error())
	}
	if err != nil && err5 != nil {
		if err.Error() != Msg || err5.Error() != Msg {
			t.Error("Expected error, got ", err.Error())
		}
	}
}
