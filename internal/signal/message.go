package signal

// Envelope represents the top-level structure of a received message.
type Envelope struct {
	Source    string `json:"source"`
	SourceNumber string `json:"sourceNumber"`
	SourceUUID   string `json:"sourceUuid"`
	SourceName   string `json:"sourceName"`
	SourceDevice int    `json:"sourceDevice"`
	Timestamp    int64  `json:"timestamp"`
	SyncMessage  *SyncMessage `json:"syncMessage"`
}

// SyncMessage contains the actual message content.
type SyncMessage struct {
	SentMessage *SentMessage `json:"sentMessage"`
}

// SentMessage contains the details of the sent message.
type SentMessage struct {
	Destination string   `json:"destination"`
	Timestamp   int64    `json:"timestamp"`
	Message     string   `json:"message"`
	GroupInfo   *GroupInfo `json:"groupInfo"`
	Reaction    *Reaction  `json:"reaction"`
}

// GroupInfo contains information about the group the message was sent to.
type GroupInfo struct {
	GroupID string `json:"groupId"`
	GroupName string `json:"groupName"`
	Type    string `json:"type"`
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
	return e.SourceNumber
}