package xevents

type Buffer struct {
	events []Event
}

func NewBuffer() *Buffer {
	return &Buffer{
		events: make([]Event, 0, 1),
	}
}

func (b *Buffer) AddEvent(event Event) {

	b.events = append(b.events, event)
}

func (b *Buffer) Collect() []Event {
	events := b.events
	b.events = make([]Event, 0, 1)
	return events
}
