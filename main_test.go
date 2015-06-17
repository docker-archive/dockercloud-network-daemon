package main

import "testing"

func Test_stringInSlice(t *testing.T) {
	list := []string{`a`, `b`, `c`}
	astring := `a`
	anotherstring := `d`

	success := stringInSlice(astring, list)
	if success != true {
		t.Error("Expected true, got ", success)
	}

	failure := stringInSlice(anotherstring, list)
	if failure != false {
		t.Error("Expected false, got ", failure)
	}
}

func Test_removeMissing(t *testing.T) {
	containerAttached := make(map[string]string)
	containerAttached[`Hello`] = `world`
	containerAttached[`Test`] = `true`

	containerList := []string{`Hello`, `GoodBye`}

	containerAttached = removeMissing(containerAttached, containerList)

	if val, ok := containerAttached[`Test`]; ok {
		t.Error("Expected no value got " + val)
	}
}

func Test_inHashWithValue(t *testing.T) {
	containerAttached := make(map[string]string)
	containerAttached[`Hello`] = `world`
	containerAttached[`Test`] = `true`

	if !inHashWithValue(containerAttached, `Hello`, `world`) {
		t.Error("Expected true, got false")
	}

	if inHashWithValue(containerAttached, `Test`, `false`) {
		t.Error("Expected false, got true")
	}
}

func Test_nodeEventHandler(t *testing.T) {
	eventType1 := "node"
	eventType2 := "service"
	state1 := "Deployed"
	state2 := "Deploying"
	state3 := "Terminated"
	Msg := "Couldn't find any Tutum credentials in ~/.tutum or environment variables TUTUM_USER and TUTUM_APIKEY"

	err := nodeEventHandler(eventType1, state1)

	err2 := nodeEventHandler(eventType2, state1)

	if err2 != nil {
		t.Error("Expected empty error message, got ", err2.Error())
	}

	err3 := nodeEventHandler(eventType2, state3)

	if err3 != nil {
		t.Error("Expected empty error message, got ", err3.Error())
	}

	err4 := nodeEventHandler(eventType1, state2)

	if err4 != nil {
		t.Error("Expected empty error message, got ", err4.Error())
	}

	err5 := nodeEventHandler(eventType1, state3)

	if err.Error() != Msg || err5.Error() != Msg {
		t.Error("Expected error, got ", err.Error())
	}
}

/*func Test_containerEventHandler(t *testing.T) {
	status1 := "start"
	from1 := "weaveworks/weave"
	from2 := "dockercontainer"
	id1 := "0123"

	endpoint := "unix:///var/run/docker.sock"
	c, _ := docker.NewClient(endpoint)

	err := containerEventHandler(status1, from1, id1, c)
	if err != nil {
		t.Error("Got error expected nil")
	}

	err2 := containerEventHandler(status1, from2, id1, c)
	if err2 == nil {
		t.Error("Got nil expected error")
	}
}*/
