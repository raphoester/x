package mongo_outbox_test

import (
	"context"

	"testing"

	"github.com/raphoester/chaos"
	"github.com/raphoester/x/xdockertest"
	"github.com/raphoester/x/xerrs"
	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xid"
	"github.com/raphoester/x/xmongo/mongo_helpers"
	"github.com/raphoester/x/xmongo/mongo_outbox"
	"github.com/raphoester/x/xtime"
	"github.com/raphoester/x/xver"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	mongo *xdockertest.Mongo
	chaos *chaos.Chaos
}

func (s *testSuite) SetupSuite() {
	db, err := xdockertest.NewMongo()
	s.Require().NoError(err)
	s.mongo = db
}

func (s *testSuite) TearDownSuite() {
	_ = s.mongo.Destroy()
}

func (s *testSuite) SetupTest() {
	err := s.mongo.Clean()
	if err != nil {
		s.T().Log("failed to clean database:", err)
	}
	s.chaos = chaos.New(s.T().Name())
}

func (s *testSuite) getTestCollection() *mongo.Collection {
	return s.mongo.Client.Database("test_outbox").Collection("test_aggregates")
}

type testAggregate struct {
	id        string
	someField string
	*xevents.Buffer
	*xver.Version
}

func (s *testAggregate) ID() string {
	return s.id
}

type testSnapshot struct {
	ID        string `bson:"_id"`
	SomeField string
	Version   int
}

func (s *testSnapshot) Restore() (*testAggregate, error) {
	return &testAggregate{
		id:        s.ID,
		someField: s.SomeField,
		Version:   xver.Restore(s.Version),
		Buffer:    xevents.NewBuffer(),
	}, nil
}

func (s *testAggregate) TakeSnapshot() testSnapshot {
	return testSnapshot{
		ID:        s.id,
		SomeField: s.someField,
		Version:   s.Version.Current(),
	}
}

func (s *testSuite) TestSaveAggregateWithEvents() {
	aggregate := &testAggregate{
		someField: "someValue",
		id:        xid.NewDefaultFixedGenerator().Generate(),
		Buffer:    xevents.NewBuffer(),
		Version:   xver.New(),
	}

	event, err := xevents.New(
		xtime.NewDefaultFixedProvider(),
		xid.NewDefaultFixedGenerator(),
		&xevents.ExamplePayload{Key: "value"},
	)

	s.Require().NoError(err)
	aggregate.AddEvent(event)

	collection := s.getTestCollection()
	err = mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		aggregate,
	)
	s.Require().NoError(err)

	// assert that the aggregate is saved
	findOneRes := collection.FindOne(context.Background(), bson.M{
		"_id": aggregate.id,
	})
	s.Require().NoError(err)

	var foundSnapshot testSnapshot
	err = findOneRes.Decode(&foundSnapshot)
	s.Require().NoError(err)

	// assert aggregate equality
	s.Assert().Equal("someValue", foundSnapshot.SomeField)
	s.Assert().Equal(aggregate.id, foundSnapshot.ID)

	// assert that the event is saved
	eventRes, err := mongo_outbox.GetEventByID(context.Background(), s.getTestCollection().Database(), event.Data().ID)
	s.Require().NoError(err)

	// assert payload equality
	var payload xevents.ExamplePayload
	err = eventRes.UnmarshalPayload(&payload)
	s.Require().NoError(err)

	s.Assert().Equal("value", payload.Key)
}

func (s *testSuite) TestSaveAggregateWithoutEvents() {
	aggregate := &testAggregate{
		someField: "someValue",
		id:        xid.NewDefaultFixedGenerator().Generate(),
		Buffer:    xevents.NewBuffer(),
		Version:   xver.New(),
	}

	collection := s.getTestCollection()
	err := mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		aggregate,
	)

	s.Require().NoError(err)

	// assert that the aggregate is saved
	found, err := mongo_helpers.FindOne[*testSnapshot](context.Background(), collection, bson.M{"_id": aggregate.id})
	s.Require().NoError(err)

	foundSnapshot := found.TakeSnapshot()

	s.Assert().Equal("someValue", foundSnapshot.SomeField)
	s.Assert().Equal(aggregate.id, foundSnapshot.ID)

	// assert that the event is not saved
	allEvents, err := mongo_outbox.FindAllEvents(context.Background(), s.getTestCollection().Database())
	s.Require().NoError(err)
	s.Assert().Empty(allEvents)
}

func (s *testSuite) TestSaveEventsOnUnmodifiedAggregate() {
	// save the aggregate one first time
	aggregate := &testAggregate{
		someField: "someValue",
		id:        xid.NewDefaultFixedGenerator().Generate(),
		Buffer:    xevents.NewBuffer(),
		Version:   xver.New(),
	}

	collection := s.getTestCollection()
	err := mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		aggregate,
	)
	s.Require().NoError(err)

	// pretend like we just queried a new instance of the aggregate
	aggregate.Version = xver.Restore(aggregate.Version.Current())

	// add events without modifying the aggregate

	event, err := xevents.New(
		xtime.NewDefaultFixedProvider(),
		xid.NewDefaultFixedGenerator(),
		&xevents.ExamplePayload{Key: "value"},
	)

	s.Require().NoError(err)
	aggregate.AddEvent(event)

	err = mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		aggregate,
	)
	s.Require().NoError(err)

	// assert that the event is saved
	eventRes, err := mongo_outbox.GetEventByID(context.Background(), s.getTestCollection().Database(), event.Data().ID)
	s.Require().NoError(err)

	var payload xevents.ExamplePayload
	err = eventRes.UnmarshalPayload(&payload)
	s.Require().NoError(err)

	s.Assert().Equal("value", payload.Key)
}

func (s *testSuite) TestErrorOnVersionConflict() {
	// perform a first save
	aggregate := &testAggregate{
		someField: "someValue",
		id:        xid.NewDefaultFixedGenerator().Generate(),
		Buffer:    xevents.NewBuffer(),
		Version:   xver.New(),
	}

	collection := s.getTestCollection()
	err := mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		aggregate,
	)
	s.Require().NoError(err)

	// simulate a new instance to represent the updates that will successfully be saved
	wantedSomeFieldValue := "value1"
	wantedPayloadKey := "value1"
	storedVersion := aggregate.Version.Current()
	passingAggregate := &testAggregate{
		id:        aggregate.id,
		someField: wantedSomeFieldValue,
		Buffer:    xevents.NewBuffer(),
		Version:   xver.Restore(storedVersion),
	}
	passingEvent, err := xevents.New(
		xtime.NewDefaultFixedProvider(),
		xid.NewDefaultFixedGenerator(),
		&xevents.ExamplePayload{Key: wantedPayloadKey},
	)
	passingAggregate.RecordNewModification()
	passingAggregate.AddEvent(passingEvent)

	// simulate a new instance of the aggregate whose update will conflict
	failingAggregate := &testAggregate{
		someField: "evenAnotherValue",
		id:        aggregate.id,
		Buffer:    xevents.NewBuffer(),
		Version:   xver.Restore(storedVersion),
	}
	failingEvent, err := xevents.New(
		xtime.NewDefaultFixedProvider(),
		xid.NewDefaultFixedGenerator(),
		&xevents.ExamplePayload{Key: "value2"},
	)
	failingAggregate.RecordNewModification()
	failingAggregate.AddEvent(failingEvent)

	// save the passing aggregate
	err = mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		passingAggregate,
	)
	s.Require().NoError(err)

	// try to save the failing aggregate
	err = mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		failingAggregate,
	)

	// will conflict because upsert where version not found attempts to insert a new document with the same ID
	s.Assert().Error(err)
	s.Assert().ErrorIs(err, xerrs.ErrConflict)

	// assert that the right event is saved
	allEvents, err := mongo_outbox.FindAllEvents(context.Background(), s.getTestCollection().Database())
	s.Require().NoError(err)
	s.Require().Len(allEvents, 1)
	retrievedEvent := allEvents[0]

	// assert payload equality
	payload := xevents.ExamplePayload{}
	err = retrievedEvent.UnmarshalPayload(&payload)
	s.Require().NoError(err)
	s.Assert().Equal(wantedPayloadKey, payload.Key)

	// assert that the right aggregate is saved
	findOneRes := collection.FindOne(context.Background(), bson.M{
		"_id": aggregate.id,
	})

	s.Require().NoError(err)
	var foundSnapshot testSnapshot
	err = findOneRes.Decode(&foundSnapshot)
	s.Require().NoError(err)

	s.Assert().Equal(wantedSomeFieldValue, foundSnapshot.SomeField)
}

func (s *testSuite) TestEventsNotSavedOnUnmodifiedConflictingAggregate() {
	// perform a first save
	aggregate := &testAggregate{
		someField: "someValue",
		id:        xid.NewDefaultFixedGenerator().Generate(),
		Buffer:    xevents.NewBuffer(),
		Version:   xver.New(),
	}

	collection := s.getTestCollection()
	err := mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		aggregate,
	)
	s.Require().NoError(err)
	storedVersion := aggregate.Version.Current()

	// simulate a new instance to represent the updates that will successfully be saved
	passingAggregate := &testAggregate{
		id:        aggregate.id,
		someField: "newValue",
		Buffer:    xevents.NewBuffer(),
		Version:   xver.Restore(storedVersion),
	}
	passingAggregate.RecordNewModification() // need the first update to increment the version for the second to fail
	passingPayloadValue := "value1"
	eventIDGenerator := xid.NewChaoticGenerator(s.chaos)
	passingEvent, err := xevents.New(
		xtime.NewDefaultFixedProvider(),
		eventIDGenerator,
		&xevents.ExamplePayload{Key: passingPayloadValue},
	)
	passingAggregate.AddEvent(passingEvent)

	// simulate a new instance of the aggregate whose update will conflict
	failingAggregate := &testAggregate{
		id:        aggregate.id,
		someField: aggregate.someField,
		Buffer:    xevents.NewBuffer(),
		Version:   xver.Restore(storedVersion),
	}
	// don't record a new modification to check if even then the conflict is detected
	failingEvent, err := xevents.New(
		xtime.NewDefaultFixedProvider(),
		eventIDGenerator,
		&xevents.ExamplePayload{Key: "value2"},
	)
	failingAggregate.AddEvent(failingEvent)

	// save the passing aggregate
	err = mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		passingAggregate,
	)
	s.Require().NoError(err)

	// try to save the failing aggregate
	err = mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		failingAggregate,
	)

	s.Assert().Error(err)
	s.Assert().ErrorIs(err, xerrs.ErrConflict)

	// assert that the event is not saved
	allEvents, err := mongo_outbox.FindAllEvents(context.Background(), s.getTestCollection().Database())
	s.Require().NoError(err)
	s.Require().Len(allEvents, 1)

	testEvent := allEvents[0]
	var payload xevents.ExamplePayload
	err = testEvent.UnmarshalPayload(&payload)
	s.Require().NoError(err)
	s.Assert().Equal(passingPayloadValue, payload.Key)
}
