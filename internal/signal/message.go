package signal

// EnvelopeWrapper represents the full message structure from the Signal API.
type EnvelopeWrapper struct {
	Envelope *Envelope `json:"envelope"`
}

// Envelope represents the top-level structure of a received message.
type Envelope struct {
	Source                   string       `json:"source"`
	SourceNumber             string       `json:"sourceNumber"`
	SourceUUID               string       `json:"sourceUuid"`
	SourceName               string       `json:"sourceName"`
	SourceDevice             int          `json:"sourceDevice"`
	Timestamp                int64        `json:"timestamp"`
	ServerReceivedTimestamp  int64        `json:"serverReceivedTimestamp"`
	ServerDeliveredTimestamp int64        `json:"serverDeliveredTimestamp"`
	SyncMessage              *SyncMessage `json:"syncMessage"`
	DataMessage              *DataMessage `json:"dataMessage"`
}

// SyncMessage contains the actual message content.
type SyncMessage struct {
	SentMessage *SentMessage `json:"sentMessage"`
}

// DataMessage contains the actual message content.
type DataMessage struct {
	Timestamp int64      `json:"timestamp"`
	Message   string     `json:"message"`
	GroupInfo *GroupInfo `json:"groupInfo"`
}

// SentMessage contains the details of the sent message.
type SentMessage struct {
	Destination string     `json:"destination"`
	Timestamp   int64      `json:"timestamp"`
	Message     string     `json:"message"`
	GroupInfo   *GroupInfo `json:"groupInfo"`
	Reaction    *Reaction  `json:"reaction"`
}

// GroupInfo contains information about the group the message was sent to.
type GroupInfo struct {
	GroupID   string `json:"groupId"`
	GroupName string `json:"groupName"`
	Type      string `json:"type"`
}

// Reaction contains information about a message reaction.
type Reaction struct {
	Emoji               string `json:"emoji"`
	TargetAuthorUUID    string `json:"targetAuthorUuid"`
	TargetSentTimestamp int64  `json:"targetSentTimestamp"`
	IsRemove            bool   `json:"isRemove"`
}

// DisplayName returns the source name, or the source number if the name is empty.
func (e *Envelope) DisplayName() string {
	if e.SourceName != "" {
		return e.SourceName
	}
	return e.SourceUUID
}
