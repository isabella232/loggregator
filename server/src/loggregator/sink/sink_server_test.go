package sink

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"loggregator/messagestore"
	"net/http"
	"net/url"
	"testhelpers"
	"testing"
	"time"
)

const (
	VALID_AUTHENTICATION_TOKEN   = "bearer correctAuthorizationToken"
	INVALID_AUTHENTICATION_TOKEN = "incorrectAuthorizationToken"
	SERVER_PORT                  = "8081"
)

var TestSinkServer *sinkServer
var dataReadChannel chan []byte

func init() {
	// This needs be unbuffered as the channel we get from the
	// agent listener is unbuffered?
	//	dataReadChannel = make(chan []byte, 10)
	dataReadChannel = make(chan []byte, 10)
	TestSinkServer = NewSinkServer(dataReadChannel, messagestore.NewMessageStore(10), testhelpers.Logger(), "localhost:"+SERVER_PORT, testhelpers.SuccessfulAuthorizer, 50*time.Millisecond)
	go TestSinkServer.Start()
	time.Sleep(1 * time.Millisecond)
}

func WaitForWebsocketRegistration() {
	time.Sleep(50 * time.Millisecond)
}

func TestThatItSends(t *testing.T) {
	receivedChan := make(chan []byte, 2)

	expectedMessageString := "Some data"
	expectedMessage := testhelpers.MarshalledLogMessage(t, expectedMessageString, "mySpace", "myApp", "myOrg")
	otherMessageString := "Some more stuff"
	otherMessage := testhelpers.MarshalledLogMessage(t, otherMessageString, "mySpace", "myApp", "myOrg")

	testhelpers.AddWSSink(t, receivedChan, SERVER_PORT, TAIL_PATH+"?org=myOrg&space=mySpace&app=myApp", VALID_AUTHENTICATION_TOKEN)
	WaitForWebsocketRegistration()

	dataReadChannel <- expectedMessage
	dataReadChannel <- otherMessage

	select {
	case <-time.After(1 * time.Second):
		t.Errorf("Did not get message.")
	case message := <-receivedChan:
		testhelpers.AssertProtoBufferMessageEquals(t, expectedMessageString, message)
	}

	select {
	case <-time.After(1 * time.Second):
		t.Errorf("Did not get message.")
	case message := <-receivedChan:
		testhelpers.AssertProtoBufferMessageEquals(t, otherMessageString, message)
	}
}

func TestThatItSendsAllDataToAllSinks(t *testing.T) {
	client1ReceivedChan := make(chan []byte)
	client2ReceivedChan := make(chan []byte)
	space1ReceivedChan := make(chan []byte)
	space2ReceivedChan := make(chan []byte)

	testhelpers.AddWSSink(t, client1ReceivedChan, SERVER_PORT, TAIL_PATH+"?org=myOrg&space=mySpace&app=myApp", VALID_AUTHENTICATION_TOKEN)
	testhelpers.AddWSSink(t, client2ReceivedChan, SERVER_PORT, TAIL_PATH+"?org=myOrg&space=mySpace&app=myApp", VALID_AUTHENTICATION_TOKEN)
	testhelpers.AddWSSink(t, space1ReceivedChan, SERVER_PORT, TAIL_PATH+"?org=myOrg&space=mySpace", VALID_AUTHENTICATION_TOKEN)
	testhelpers.AddWSSink(t, space2ReceivedChan, SERVER_PORT, TAIL_PATH+"?org=myOrg&space=mySpace", VALID_AUTHENTICATION_TOKEN)
	WaitForWebsocketRegistration()

	expectedMessageString := "Some Data"
	expectedMarshalledProtoBuffer := testhelpers.MarshalledLogMessage(t, expectedMessageString, "mySpace", "myApp", "myOrg")

	dataReadChannel <- expectedMarshalledProtoBuffer

	testhelpers.AssertProtoBufferMessageEquals(t, expectedMessageString, <-client1ReceivedChan)
	testhelpers.AssertProtoBufferMessageEquals(t, expectedMessageString, <-client2ReceivedChan)
	testhelpers.AssertProtoBufferMessageEquals(t, expectedMessageString, <-space1ReceivedChan)
	testhelpers.AssertProtoBufferMessageEquals(t, expectedMessageString, <-space2ReceivedChan)
}

func TestThatItSendsLogsToProperAppSink(t *testing.T) {
	receivedChan := make(chan []byte)

	otherAppsMarshalledMessage := testhelpers.MarshalledLogMessage(t, "Some other message", "mySpace", "otherApp", "myOrg")

	expectedMessageString := "My important message"
	myAppsMarshalledMessage := testhelpers.MarshalledLogMessage(t, expectedMessageString, "mySpace", "myApp", "myOrg")

	testhelpers.AddWSSink(t, receivedChan, SERVER_PORT, TAIL_PATH+"?org=myOrg&space=mySpace&app=myApp", VALID_AUTHENTICATION_TOKEN)
	WaitForWebsocketRegistration()

	dataReadChannel <- otherAppsMarshalledMessage
	dataReadChannel <- myAppsMarshalledMessage

	testhelpers.AssertProtoBufferMessageEquals(t, expectedMessageString, <-receivedChan)
}

func TestThatItSendsLogsToProperSpaceSink(t *testing.T) {
	receivedChan := make(chan []byte)

	otherSpaceMarshalledMessage := testhelpers.MarshalledLogMessage(t, "Some other message", "otherSpace", "otherApp", "myOrg")

	expectedMessageString := "My important message"
	mySpaceMarshalledMessage := testhelpers.MarshalledLogMessage(t, expectedMessageString, "mySpace", "myApp", "myOrg")

	testhelpers.AddWSSink(t, receivedChan, SERVER_PORT, TAIL_PATH+"?org=myOrg&space=mySpace", VALID_AUTHENTICATION_TOKEN)
	WaitForWebsocketRegistration()

	dataReadChannel <- otherSpaceMarshalledMessage
	dataReadChannel <- mySpaceMarshalledMessage

	testhelpers.AssertProtoBufferMessageEquals(t, expectedMessageString, <-receivedChan)
}

func TestDropUnmarshallableMessage(t *testing.T) {
	receivedChan := make(chan []byte)

	sink, _ := testhelpers.AddWSSink(t, receivedChan, SERVER_PORT, TAIL_PATH+"?org=myOrg&space=mySpace&app=myApp", VALID_AUTHENTICATION_TOKEN)
	WaitForWebsocketRegistration()

	dataReadChannel <- make([]byte, 10)

	time.Sleep(1 * time.Millisecond)
	select {
	case msg1 := <-receivedChan:
		t.Error("We should not have received a message, but got: %v", msg1)
	default:
		//no communication. That's good!
	}

	sink.Close()
	expectedMessageString := "My important message"
	mySpaceMarshalledMessage := testhelpers.MarshalledLogMessage(t, expectedMessageString, "mySpace", "myApp", "myOrg")
	dataReadChannel <- mySpaceMarshalledMessage
}

//func TestDropSinkWithoutAppAndContinuesToWork(t *testing.T) {
//	AssertConnecitonFails(t, SERVER_PORT, TAIL_PATH+"", "")
//	TestThatItSends(t)
//}
//
//func TestDropSinkWithoutAuthorizationAndContinuesToWork(t *testing.T) {
//	AssertConnecitonFails(t, SERVER_PORT, TAIL_PATH+"?space=mySpace&app=myApp", "")
//	TestThatItSends(t)
//}
//
//func TestDropSinkWhenAuthorizationFailsAndContinuesToWork(t *testing.T) {
//	AssertConnecitonFails(t, SERVER_PORT, TAIL_PATH+"?space=mySpace&app=myApp", INVALID_AUTHENTICATION_TOKEN)
//	TestThatItSends(t)
//}

func TestKeepAlive(t *testing.T) {
	receivedChan := make(chan []byte)

	_, killKeepAliveChan := testhelpers.AddWSSink(t, receivedChan, SERVER_PORT, TAIL_PATH+"?org=MyOrg&space=mySpace&app=myApp", VALID_AUTHENTICATION_TOKEN)
	WaitForWebsocketRegistration()

	killKeepAliveChan <- true

	time.Sleep(60 * time.Millisecond) //wait a little bit to make sure the keep-alive has successfully been stopped

	expectedMessageString := "My important message"
	myAppsMarshalledMessage := testhelpers.MarshalledLogMessage(t, expectedMessageString, "mySpace", "myApp", "myOrg")
	dataReadChannel <- myAppsMarshalledMessage

	time.Sleep(10 * time.Millisecond) //wait a little bit to give a potential message time to arrive

	select {
	case msg1 := <-receivedChan:
		t.Error("We should not have received a message, but got: %v", msg1)
	default:
		//no communication. That's good!
	}
}

// *** Start dump tests

func TestItDumpsAllMessages(t *testing.T) {
	expectedMessageString := "Some data"
	expectedMessage := testhelpers.MarshalledLogMessage(t, expectedMessageString, "mySpace", "myApp", "myOrg")

	dataReadChannel <- expectedMessage

	req, err := http.NewRequest("GET", "http://localhost:"+SERVER_PORT+DUMP_PATH+"?org=MyOrg&space=mySpace&app=myApp", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", VALID_AUTHENTICATION_TOKEN)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, resp.Header.Get("Content-Type"), "application/octet-stream")
	assert.Equal(t, resp.StatusCode, 200)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	resp.Body.Close()

	messages, err := testhelpers.ParseDumpedMessages(body)
	assert.NoError(t, err)

	testhelpers.AssertProtoBufferMessageEquals(t, expectedMessageString, messages[len(messages)-1])
}

func TestItReturns401WithIncorrectAuthToken(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost:"+SERVER_PORT+DUMP_PATH+"?org=MyOrg&space=mySpace&app=myApp", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", INVALID_AUTHENTICATION_TOKEN)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, resp.StatusCode, 401)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "", string(body))
}

func TestItReturns404WithoutSpaceId(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost:"+SERVER_PORT+DUMP_PATH, nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", VALID_AUTHENTICATION_TOKEN)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, resp.StatusCode, 404)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "", string(body))
}

func TestExtractTarget(t *testing.T) {
	theUrl, err := url.Parse("wss://loggregator.loggregatorci.cf-app.com:4443/tail/?org=6e6926ce-bd94-428d-944f-9446ae446deb&space=e0c78fc4-443b-43d0-840f-ed8b0823b4fd&app=11bfecc7-7128-4e56-83a0-d8e0814ed7e6")
	assert.NoError(t, err)
	target := extractTarget(theUrl)
	assert.Equal(t, "6e6926ce-bd94-428d-944f-9446ae446deb", target.OrgId)
	assert.Equal(t, "e0c78fc4-443b-43d0-840f-ed8b0823b4fd", target.SpaceId)
	assert.Equal(t, "11bfecc7-7128-4e56-83a0-d8e0814ed7e6", target.AppId)
}
