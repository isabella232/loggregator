package trafficcontroller_test

import (
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/grpclog"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	APP_ID          = "1234"
	AUTH_TOKEN      = "bearer iAmAnAdmin"
	SUBSCRIPTION_ID = "firehose-subscription-1"
)

var (
	localIPAddress                    string
	fakeDoppler                       *FakeDoppler
	tcCleanupFunc                     func()
	TRAFFIC_CONTROLLER_DROPSONDE_PORT int
)

func TestIntegrationTest(t *testing.T) {
	grpclog.SetLogger(log.New(GinkgoWriter, "", log.LstdFlags))
	log.SetOutput(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Traffic Controller Integration Suite")
}

var _ = BeforeSuite(func() {
	setupFakeAuthServer()
	setupFakeUaaServer()

	var err error
	trafficControllerExecPath, err := gexec.Build("code.cloudfoundry.org/loggregator/trafficcontroller", "-race")
	Expect(err).ToNot(HaveOccurred())
	os.Setenv("TRAFFIC_CONTROLLER_BUILD_PATH", trafficControllerExecPath)

	localIPAddress = "127.0.0.1"
})

var _ = AfterEach(func() {
	tcCleanupFunc()
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

var setupFakeAuthServer = func() {
	fakeAuthServer := &FakeAuthServer{ApiEndpoint: "localhost:42123"}
	fakeAuthServer.Start()

	Eventually(func() error {
		_, err := http.Get("http://" + localIPAddress + "localhost:42123")
		return err
	}).ShouldNot(HaveOccurred())
}

var setupFakeUaaServer = func() {
	fakeUaaServer := &FakeUaaHandler{}
	go http.ListenAndServe("localhost:5678", fakeUaaServer)
	Eventually(func() error {
		_, err := http.Get("http://" + localIPAddress + ":5678/check_token")
		return err
	}).ShouldNot(HaveOccurred())
}

func makeDropsondeMessage(messageString string, appId string, currentTime int64) []byte {
	logMessage := &events.LogMessage{
		Message:        []byte(messageString),
		MessageType:    events.LogMessage_ERR.Enum(),
		Timestamp:      proto.Int64(currentTime),
		AppId:          proto.String(appId),
		SourceType:     proto.String("DOP"),
		SourceInstance: proto.String("SN"),
	}

	envelope := &events.Envelope{
		LogMessage: logMessage,
		Origin:     proto.String("doppler"),
		EventType:  events.Envelope_LogMessage.Enum(),
	}
	msg, _ := proto.Marshal(envelope)

	return msg
}
func makeContainerMetricMessage(appId string, instanceIndex int, cpu int, membytes int, diskbytes int, currentTime int64) []byte {
	containerMetric := &events.ContainerMetric{
		ApplicationId: proto.String(appId),
		InstanceIndex: proto.Int32(int32(instanceIndex)),
		CpuPercentage: proto.Float64(float64(cpu)),
		MemoryBytes:   proto.Uint64(uint64(membytes)),
		DiskBytes:     proto.Uint64(uint64(diskbytes)),
	}

	envelope := &events.Envelope{
		ContainerMetric: containerMetric,
		Origin:          proto.String("doppler"),
		EventType:       events.Envelope_ContainerMetric.Enum(),
		Timestamp:       proto.Int64(currentTime),
	}
	msg, _ := proto.Marshal(envelope)

	return msg
}

type RouterStart struct {
	Id                               string   `json:"id"`
	Hosts                            []string `json:"hosts"`
	MinimumRegisterIntervalInSeconds int      `json:"minimumRegisterIntervalInSeconds"`
}
