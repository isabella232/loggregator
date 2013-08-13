package sink

import (
	"cfcomponent/instrumentation"
	"code.google.com/p/go.net/websocket"
	"code.google.com/p/gogoprotobuf/proto"
	"fmt"
	"github.com/cloudfoundry/gosteno"
	"logMessage"
	"loggregator/logtarget"
	"loggregator/messagestore"
	"net/http"
	"net/url"
	"time"
)

const (
	TAIL_PATH = "/tail/"
	DUMP_PATH = "/dump/"
)

type sinkServer struct {
	logger            *gosteno.Logger
	dataChannel       chan []byte
	listenHost        string
	listenerChannels  *groupedChannels
	authorize         LogAccessAuthorizer
	sinkCloseChan     chan chan []byte
	keepAliveInterval time.Duration
	messageStore      *messagestore.MessageStore
}

func NewSinkServer(givenChannel chan []byte, messageStore *messagestore.MessageStore, logger *gosteno.Logger, listenHost string, authorize LogAccessAuthorizer, keepAliveInterval time.Duration) *sinkServer {
	listeners := newGroupedChannels()
	sinkCloseChan := make(chan chan []byte, 4)
	return &sinkServer{logger, givenChannel, listenHost, listeners, authorize, sinkCloseChan, keepAliveInterval, messageStore}
}

func (sinkServer *sinkServer) sinkRelayHandler(ws *websocket.Conn) {
	clientAddress := ws.RemoteAddr()

	target := extractTarget(ws.Request().URL)
	authToken := ws.Request().Header.Get("Authorization")

	if !target.IsValid() {
		message := fmt.Sprintf("Did not accept sink connection with invalid org, space, app combination: %s.", clientAddress)
		sinkServer.logger.Warn(message)
		return
	}

	if authToken == "" {
		message := fmt.Sprintf("Did not accept sink connection from %s without authorization.", clientAddress)
		sinkServer.logger.Warnf(message)
		return
	}

	if !sinkServer.authorize(authToken, target.SpaceId, target.AppId, sinkServer.logger) {
		message := fmt.Sprintf("Auth token [%s] not authorized to access space [%s].", authToken, target.SpaceId)
		sinkServer.logger.Warn(message)
		return
	}

	sink := newCfSink(target, sinkServer.logger, ws, clientAddress, sinkServer.keepAliveInterval)

	sinkServer.listenerChannels.add(sink.listenerChannel, target.SpaceId, target.AppId)
	sink.Run(sinkServer.sinkCloseChan)
}

func (sinkServer *sinkServer) dumpHandler(rw http.ResponseWriter, req *http.Request) {
	target := extractTarget(req.URL)
	authToken := req.Header.Get("Authorization")

	if !target.IsValid() {
		sinkServer.logger.Warn("Did not accept dump connection with invalid org, space, app combination.")
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	if !sinkServer.authorize(authToken, target.SpaceId, target.AppId, sinkServer.logger) {
		message := fmt.Sprintf("Auth token [%s] not authorized to access space [%s].", authToken, target.SpaceId)
		sinkServer.logger.Warn(message)
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	rw.Header().Set("Content-Type", "application/octet-stream")
	rw.Write(sinkServer.messageStore.DumpFor(target.SpaceId, target.AppId))
}

func (sinkServer *sinkServer) relayMessagesToAllSinks() {
	extractReceivedSpaceAndAppId := func(data []byte) (string, string) {
		receivedMessage := &logMessage.LogMessage{}
		err := proto.Unmarshal(data, receivedMessage)
		if err != nil {
			sinkServer.logger.Debugf("Log message could not be unmarshaled. Dropping it... Error: %v. Data: %v", err, data)
			return "", ""
		}
		return *receivedMessage.SpaceId, *receivedMessage.AppId
	}

	for {
		select {
		case sin := <-sinkServer.sinkCloseChan:
			sinkServer.listenerChannels.delete(sin)
			close(sin)
			sinkServer.logger.Info("A Tail client went have away. Closed it.")
		case data := <-sinkServer.dataChannel:
			sinkServer.logger.Debugf("Received %d bytes of data from agent listener.", len(data))
			receivedSpaceId, receivedAppId := extractReceivedSpaceAndAppId(data)

			sinkServer.messageStore.Add(data, receivedSpaceId, receivedAppId)

			sinkServer.logger.Debugf("Searching for channels with spaceId [%s] and appId [%s].", receivedSpaceId, receivedAppId)
			for _, listenerChannel := range sinkServer.listenerChannels.get(receivedSpaceId, receivedAppId) {
				sinkServer.logger.Debugf("Sending Message to channel %s for space [%s] and app [%s].", listenerChannel, receivedSpaceId, receivedAppId)
				listenerChannel <- data
			}
			sinkServer.logger.Debugf("Searching for channels with spaceId [%s].", receivedSpaceId)
			for _, listenerChannel := range sinkServer.listenerChannels.get(receivedSpaceId) {
				sinkServer.logger.Debugf("Sending Message to channel %s for space [%s].", listenerChannel, receivedSpaceId)
				listenerChannel <- data
			}
			sinkServer.logger.Debugf("Done sending message to tail clients.")
		}
	}
}

func (sinkServer *sinkServer) Start() {
	go sinkServer.relayMessagesToAllSinks()

	sinkServer.logger.Infof("Listening on port %s", sinkServer.listenHost)
	http.Handle(TAIL_PATH, websocket.Handler(sinkServer.sinkRelayHandler))
	http.HandleFunc(DUMP_PATH, sinkServer.dumpHandler)
	err := http.ListenAndServe(sinkServer.listenHost, nil)
	if err != nil {
		panic(err)
	}
}

func (sinkServer *sinkServer) Emit() instrumentation.Context {
	return instrumentation.Context{
		Name: "sinkServer",
		Metrics: []instrumentation.Metric{
			instrumentation.Metric{"numberOfSinks", sinkServer.listenerChannels.NumberOfChannels()},
		},
	}
}

func extractTarget(u *url.URL) *logtarget.LogTarget {
	appId := u.Query().Get("app")
	spaceId := u.Query().Get("space")
	orgId := u.Query().Get("org")
	target := &logtarget.LogTarget{orgId, spaceId, appId}
	return target
}
