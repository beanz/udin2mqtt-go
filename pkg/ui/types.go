package ui

type UIEventType int

const (
	UIRenameEvent UIEventType = iota
	UIEnableEvent
	UICreateEvent
)

type UIEvent struct {
	Kind UIEventType
	Args []string
}

func NewUIEvent(kind UIEventType, args ...string) UIEvent {
	return UIEvent{kind, args}
}
