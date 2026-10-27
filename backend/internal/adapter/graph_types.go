package adapter

// GraphMessage Graph API 邮件响应结构
type GraphMessage struct {
	ID                      string           `json:"id"`
	Subject                 string           `json:"subject"`
	BodyPreview             string           `json:"bodyPreview"`
	Body                    GraphItemBody    `json:"body"`
	From                    GraphRecipient   `json:"from"`
	ToRecipients            []GraphRecipient `json:"toRecipients"`
	CcRecipients            []GraphRecipient `json:"ccRecipients"`
	BccRecipients           []GraphRecipient `json:"bccRecipients"`
	ReplyTo                 []GraphRecipient `json:"replyTo"`
	SentDateTime            string           `json:"sentDateTime"`
	ReceivedDateTime        string           `json:"receivedDateTime"`
	HasAttachments          bool             `json:"hasAttachments"`
	InternetMessageID       string           `json:"internetMessageId"`
	ConversationID          string           `json:"conversationId"`
	IsRead                  bool             `json:"isRead"`
	Categories              []string         `json:"categories"`
	InferenceClassification string           `json:"inferenceClassification"`
}

// GraphItemBody 邮件正文
type GraphItemBody struct {
	ContentType string `json:"contentType"` // text 或 html
	Content     string `json:"content"`
}

// GraphRecipient 收件人信息
type GraphRecipient struct {
	EmailAddress GraphEmailAddress `json:"emailAddress"`
}

// GraphEmailAddress 邮件地址
type GraphEmailAddress struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// GraphMessageList 邮件列表响应
type GraphMessageList struct {
	Value    []GraphMessage `json:"value"`
	NextLink string         `json:"@odata.nextLink"`
}

// GraphAttachment 附件信息
type GraphAttachment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	IsInline    bool   `json:"isInline"`
	ContentID   string `json:"contentId"`
}

// GraphAttachmentList 附件列表响应
type GraphAttachmentList struct {
	Value []GraphAttachment `json:"value"`
}
