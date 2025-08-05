package signal

// EnvelopeWrapper represents the full message structure from the Signal API.
type EnvelopeWrapper struct {
	Envelope *Envelope `json:"envelope"`
	Account  string    `json:"account"`
}

// Envelope represents the top-level structure of a received message.
type Envelope struct {
	Source                   string          `json:"source"`
	SourceNumber             string          `json:"sourceNumber"`
	SourceUUID               string          `json:"sourceUuid"`
	SourceName               string          `json:"sourceName"`
	SourceDevice             int             `json:"sourceDevice"`
	Timestamp                int64           `json:"timestamp"`
	ServerReceivedTimestamp  int64           `json:"serverReceivedTimestamp"`
	ServerDeliveredTimestamp int64           `json:"serverDeliveredTimestamp"`
	SyncMessage              *SyncMessage    `json:"syncMessage"`
	DataMessage              *DataMessage    `json:"dataMessage"`
	ReceiptMessage           *ReceiptMessage `json:"receiptMessage"`
}

// SyncMessage contains the actual message content.
type SyncMessage struct {
	SentMessage *SentMessage `json:"sentMessage"`
}

// DataMessage contains the actual message content.
type DataMessage struct {
	Timestamp        int64      `json:"timestamp"`
	Message          string     `json:"message"`
	ExpiresInSeconds int        `json:"expiresInSeconds"`
	ViewOnce         bool       `json:"viewOnce"`
	GroupInfo        *GroupInfo `json:"groupInfo"`
	Quote            *Quote     `json:"quote"`
	Reaction         *Reaction  `json:"reaction"`
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
	Revision  int    `json:"revision"`
}

// Quote contains information about a quoted/replied-to message.
type Quote struct {
	ID           int64    `json:"id"`
	Author       string   `json:"author"`
	AuthorNumber string   `json:"authorNumber"`
	AuthorUUID   string   `json:"authorUuid"`
	Text         string   `json:"text"`
	Attachments  []string `json:"attachments"`
}

// Reaction contains information about a message reaction.
type Reaction struct {
	Emoji               string `json:"emoji"`
	TargetAuthor        string `json:"targetAuthor"`
	TargetAuthorNumber  string `json:"targetAuthorNumber"`
	TargetAuthorUUID    string `json:"targetAuthorUuid"`
	TargetSentTimestamp int64  `json:"targetSentTimestamp"`
	IsRemove            bool   `json:"isRemove"`
}

// ReceiptMessage contains delivery/read receipt information.
type ReceiptMessage struct {
	When       int64   `json:"when"`
	IsDelivery bool    `json:"isDelivery"`
	IsRead     bool    `json:"isRead"`
	IsViewed   bool    `json:"isViewed"`
	Timestamps []int64 `json:"timestamps"`
}

// DisplayName returns the source name, or the source number if the name is empty.
func (e *Envelope) DisplayName() string {
	if e.SourceName != "" {
		return e.SourceName
	}
	return e.SourceUUID
}
