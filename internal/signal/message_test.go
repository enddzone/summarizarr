package signal

import (
	"encoding/json"
	"testing"
)

const sampleMessage = `
{
    "envelope": {
      "source": "+18177392137",
      "sourceNumber": "+18177392137",
      "sourceUuid": "6f18ddfb-c3f4-4ce9-9da0-d90efd7f2e2b",
      "sourceName": "Nipuna Perera",
      "sourceDevice": 1,
      "timestamp": 1754277894400,
      "serverReceivedTimestamp": 1754277892899,
      "serverDeliveredTimestamp": 1754277901483,
      "syncMessage": {
        "sentMessage": {
          "destination": null,
          "destinationNumber": null,
          "destinationUuid": null,
          "timestamp": 1754277894400,
          "message": "This is a test message.",
          "expiresInSeconds": 0,
          "viewOnce": false,
          "groupInfo": {
            "groupId": "PLraV/rOQu4vyodMyn9fG2sgH1P+F9S+8iikkrGfNn0=",
            "groupName": "Setups and Suggestions",
            "type": "DELIVER"
          }
        }
      }
    },
    "account": "+18177392137"
}
`

func TestUnmarshalMessage(t *testing.T) {
	var msg EnvelopeWrapper
	err := json.Unmarshal([]byte(sampleMessage), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if msg.Envelope.Source != "+18177392137" {
		t.Errorf("Expected source to be '+18177392137', but got '%s'", msg.Envelope.Source)
	}

	if msg.Envelope.SourceName != "Nipuna Perera" {
		t.Errorf("Expected source name to be 'Nipuna Perera', but got '%s'", msg.Envelope.SourceName)
	}

	if msg.Envelope.SyncMessage.SentMessage.Message != "This is a test message." {
		t.Errorf("Expected message to be 'This is a test message.', but got '%s'", msg.Envelope.SyncMessage.SentMessage.Message)
	}

	if msg.Envelope.SyncMessage.SentMessage.GroupInfo.GroupName != "Setups and Suggestions" {
		t.Errorf("Expected group name to be 'Setups and Suggestions', but got '%s'", msg.Envelope.SyncMessage.SentMessage.GroupInfo.GroupName)
	}
}