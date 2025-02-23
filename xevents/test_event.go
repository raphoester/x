package xevents

type ExamplePayload struct {
	Key string `json:"key"`

	topicName string
	isValid   *bool
}

func (p ExamplePayload) Topic() string {
	if p.topicName != "" {
		return p.topicName
	}
	return ExamplePayloadDefaultTopicName
}

func (p ExamplePayload) WithTopic(str string) ExamplePayload {
	return ExamplePayload{
		Key:       p.Key,
		topicName: str,
	}
}

func (p ExamplePayload) WithIsValid(value bool) ExamplePayload {
	return ExamplePayload{
		Key:     p.Key,
		isValid: &value,
	}
}

func (p ExamplePayload) IsValid() bool {
	if p.isValid != nil {
		return *p.isValid
	}

	return true
}

const ExamplePayloadDefaultTopicName = "example"
