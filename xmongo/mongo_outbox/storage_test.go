package mongo_outbox_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xid"
	"github.com/raphoester/x/xmongo/mongo_outbox"
	"github.com/raphoester/x/xtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventConversion makes sure that the event is not altered when converting it to a DAO and back
func TestEventConversion(t *testing.T) {
	timeProvider := xtime.CustomProvider{NowFunc: func() time.Time { return time.Date(2024, time.October, 10, 0, 0, 0, 0, time.UTC) }}
	idGenerator := xid.CustomGenerator{GenFunc: func() string { return "id" }}
	event := xevents.New(
		timeProvider,
		idGenerator,
		&TestPayload{
			String: "string",
			Int:    42,
			Nested: struct {
				Bool bool
			}{
				Bool: true,
			},
		})

	dao, err := mongo_outbox.EventToDAO(event)
	require.NoError(t, err)

	event2, err := mongo_outbox.DAOToEvent(*dao)
	require.NoError(t, err)

	require.Equal(t, event.Data().Topic, event2.Data().Topic)
	require.Equal(t, event.Data().ID, event2.Data().ID)
	require.Equal(t, event.Data().CreatedAt, event2.Data().CreatedAt)

	payload2, err := json.Marshal(event.Data().Payload)
	require.NoError(t, err)

	assert.JSONEq(t, `{"String":"string","Int":42,"Nested":{"Bool":true}}`, string(payload2))
}

type TestPayload struct {
	String string
	Int    int
	Nested struct {
		Bool bool
	}
}

func (p *TestPayload) Topic() string {
	return "test"
}

func (p *TestPayload) IsValid() bool {
	return true
}
