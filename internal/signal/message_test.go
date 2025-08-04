package signal

import (
	"encoding/json"
	"testing"
)

const sampleMessage = `
{
    "envelope": {
      "source": "e1a8c050-2b12-440b-9814-f993c07a758e",
      "sourceNumber": null,
      "sourceUuid": "e1a8c050-2b12-440b-9814-f993c07a758e",
      "sourceName": "PR",
      "sourceDevice": 1,
      "timestamp": 1754295444829,
      "serverReceivedTimestamp": 1754295445161,
      "serverDeliveredTimestamp": 1754298658564,
      "dataMessage": {
        "timestamp": 1754295444829,
        "message": "Together.2025.1080p.SCREENER.WEB-DL.X264.AC3-AOC",
        "expiresInSeconds": 0,
        "viewOnce": false,
        "groupInfo": {
          "groupId": "MvIF76urVKX1Zc2gPDciy/7V3P5xLtuQHk6zMkeTZtU=",
          "groupName": "Trackers / Usenet Indexers",
          "revision": 715,
          "type": "DELIVER"
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

	if msg.Envelope.Source != "e1a8c050-2b12-440b-9814-f993c07a758e" {
		t.Errorf("Expected source to be 'e1a8c050-2b12-440b-9814-f993c07a758e', but got '%s'", msg.Envelope.Source)
	}

	if msg.Envelope.SourceName != "PR" {
		t.Errorf("Expected source name to be 'PR', but got '%s'", msg.Envelope.SourceName)
	}

	if msg.Envelope.DataMessage.Message != "Together.2025.1080p.SCREENER.WEB-DL.X264.AC3-AOC" {
		t.Errorf("Expected message to be 'Together.2025.1080p.SCREENER.WEB-DL.X264.AC3-AOC', but got '%s'", msg.Envelope.DataMessage.Message)
	}

	if msg.Envelope.DataMessage.GroupInfo.GroupName != "Trackers / Usenet Indexers" {
		t.Errorf("Expected group name to be 'Trackers / Usenet Indexers', but got '%s'", msg.Envelope.DataMessage.GroupInfo.GroupName)
	}
}