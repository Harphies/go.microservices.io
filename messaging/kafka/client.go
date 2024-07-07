package kafka

type EventType string

type event struct {
	Type  string
	Value interface{}
}
