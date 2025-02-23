package mongo_outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphoester/x/xerrs"
	"github.com/raphoester/x/xevents"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func NewStorageFromClient(client *mongo.Client, databaseName string) *Storage {
	return &Storage{
		database: client.Database(databaseName),
	}
}

func NewStorage(db *mongo.Database) *Storage {
	return &Storage{database: db}
}

type Storage struct {
	database *mongo.Database
}

func (s *Storage) GetPendingEvents(ctx context.Context) ([]xevents.Event, error) {
	return FindAllEvents(context.Background(), s.database)
}

func (s *Storage) MarkAsPublished(ctx context.Context, id string) error {
	return MarkAsPublished(ctx, s.database, id)
}

const outboxEventsCollectionName = "OutboxEvents"

func MarkAsPublished(ctx context.Context, db *mongo.Database, id string) error {
	res, err := db.Collection(outboxEventsCollectionName).DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete event with ID %q: %w", id, err)
	}

	if res.DeletedCount == 0 {
		return xerrs.ErrNotFound
	}

	return nil
}

func SaveEvents(ctx context.Context, db *mongo.Database, events []xevents.Event) error {
	collection := db.Collection(outboxEventsCollectionName)

	daos, err := EventsToDAOs(events)
	if err != nil {
		return fmt.Errorf("failed to convert events to daos: %w", err)
	}

	if _, err := collection.InsertMany(ctx, daos); err != nil {
		return err
	}

	return nil
}

func FindAllEvents(ctx context.Context, db *mongo.Database) ([]xevents.Event, error) {
	collection := db.Collection(outboxEventsCollectionName)

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find all events: %w", err)
	}

	var daos []EventDAO
	if err := cursor.All(ctx, &daos); err != nil {
		return nil, fmt.Errorf("failed to decode all events: %w", err)
	}

	events, err := DAOsToEvents(daos)
	if err != nil {
		return nil, fmt.Errorf("failed to convert daos to events: %w", err)
	}

	return events, nil
}

func GetEventByID(ctx context.Context, db *mongo.Database, id string) (*xevents.Event, error) {
	collection := db.Collection(outboxEventsCollectionName)

	filter := map[string]string{"_id": id}
	res := collection.FindOne(ctx, filter)
	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("failed to find event with ID %q: %w", id, err)
	}

	var dao EventDAO
	if err := res.Decode(&dao); err != nil {
		return nil, fmt.Errorf("failed to decode event with ID %q: %w", id, err)
	}

	event, err := DAOToEvent(dao)
	if err != nil {
		return nil, fmt.Errorf("failed to convert dao to event: %w", err)
	}

	return event, nil
}

type EventDAO struct {
	ID        string `bson:"_id"`
	CreatedAt time.Time
	Topic     string
	Payload   payloadMap
}

func EventToDAO(event xevents.Event) (*EventDAO, error) {
	eventData := event.Data()
	payload := payloadMap{}

	if err := event.UnmarshalPayload(&payload); err != nil {
		return nil, err
	}

	return &EventDAO{
		ID:        eventData.ID,
		CreatedAt: eventData.CreatedAt,
		Topic:     eventData.Topic,
		Payload:   payload,
	}, nil
}

type payloadMap map[string]any

func (p payloadMap) marshalJSON() ([]byte, error) {
	return json.Marshal(p)
}

func EventsToDAOs(events []xevents.Event) ([]any, error) {
	var eventDAOs []any
	for _, event := range events {
		eventDAO, err := EventToDAO(event)
		if err != nil {
			return nil, fmt.Errorf("failed to convert event with ID %q: %w", event.Data().ID, err)
		}
		eventDAOs = append(eventDAOs, eventDAO)
	}
	return eventDAOs, nil
}

func DAOToEvent(dao EventDAO) (*xevents.Event, error) {
	if dao.ID == "" {
		return nil, fmt.Errorf("id is empty")
	}

	payload := make(map[string]any)
	jsonBytes, err := dao.Payload.marshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to convert jsonb to json: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	restored := xevents.Restore(dao.ID, dao.CreatedAt, dao.Topic, payload)
	return &restored, nil
}

func DAOsToEvents(daos []EventDAO) ([]xevents.Event, error) {
	var events []xevents.Event
	for _, dao := range daos {
		event, err := DAOToEvent(dao)
		if err != nil {
			return nil, fmt.Errorf("failed to convert dao with id %q to event: %w", dao.ID, err)
		}
		events = append(events, *event)
	}
	return events, nil
}
