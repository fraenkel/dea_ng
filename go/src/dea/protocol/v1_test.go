package protocol

import (
	"dea"
	"encoding/json"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/localip"
	"github.com/nu7hatch/gouuid"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestHelloMessage(t *testing.T) {
	localIp, err := localip.LocalIP()
	if err != nil {
		t.Fatal(err)
	}

	port := 2222
	uuid, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	message := NewHelloMessage(*uuid, localIp, uint16(port))
	bytes, err := json.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}

	decoded := &HelloMessage{}
	if err = json.Unmarshal(bytes, decoded); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, message, decoded)
}

func TestHeartbeatResponseV1(t *testing.T) {
	uuid, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}
	instances := make([]dea.Instance, 1)
	instances[0] = *dea.NewInstance(map[string]interface{}{
		"cc_partition":        "partition",
		"instance_id":         uuid.String(),
		"instance_index":      37,
		"application_id":      "37",
		"application_version": "some_version",
		"application_name":    "my_application",
		"state":               dea.BORN,
		"state_time":          time.Now(),
	})
	response := NewHeartbeatResponseV1(*uuid, instances)

	assert.NotNil(t, response)
	bytes, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}

	decoded := &HeartbeatResponseV1{}
	if err = json.Unmarshal(bytes, decoded); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uuid.String(), decoded.Uuid)
	assert.Equal(t, response, decoded)
}
