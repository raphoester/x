package mongo_outbox_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/raphoester/x/xdockertest"
	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xid"
	"github.com/raphoester/x/xmongo/mongo_outbox"
	"github.com/raphoester/x/xtime"
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
}

func (s *testSuite) getTestCollection() *mongo.Collection {
	return s.mongo.DB.Collection("test_aggregates")
}

type testAggregate struct {
	ID        string `bson:"_id"`
	SomeField string
	*xevents.Buffer
}

func (s *testSuite) TestSaveAggregateWithEvents() {
	aggregate := &testAggregate{
		SomeField: "someValue",
		ID:        xid.NewDefaultFixedGenerator().Generate(),
		Buffer:    xevents.NewBuffer(),
	}

	aggregate.AddEvent(
		xevents.New(
			xtime.NewDefaultFixedProvider(),
			xid.NewDefaultFixedGenerator(),
			&xevents.ExamplePayload{
				Key: "value",
			}),
	)

	collection := s.getTestCollection()
	var insertedID any
	err := mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		aggregate,
		func(ctx context.Context, coll *mongo.Collection, aggregate *testAggregate) error {
			res, err := coll.InsertOne(ctx, aggregate)
			insertedID = res.InsertedID
			return err
		},
	)
	s.Require().NoError(err)

	// assert that the aggregate is saved
	findOneRes := collection.FindOne(context.Background(), bson.M{
		"_id": insertedID,
	})
	s.Require().NoError(err)

	var foundAggregate testAggregate
	err = findOneRes.Decode(&foundAggregate)
	s.Require().NoError(err)

	s.Assert().Equal("someValue", foundAggregate.SomeField)
	s.Assert().Empty(foundAggregate.Buffer.Collect())
	s.Assert().Equal(aggregate.ID, foundAggregate.ID)

	// assert that the event is saved
	eventRes, err := mongo_outbox.GetEventByID(context.Background(), s.mongo.DB, fmt.Sprint(insertedID))
	s.Require().NoError(err)

	// assert payload equality
	var payload xevents.ExamplePayload
	err = eventRes.UnmarshalPayload(&payload)
	s.Require().NoError(err)

	s.Assert().Equal("value", payload.Key)
}

func (s *testSuite) TestSaveAggregateWithoutEvents() {
	aggregate := &testAggregate{
		SomeField: "someValue",
		ID:        xid.NewDefaultFixedGenerator().Generate(),
		Buffer:    xevents.NewBuffer(),
	}

	collection := s.getTestCollection()
	var insertedID any
	err := mongo_outbox.SaveAggregate(
		context.Background(),
		collection,
		aggregate,
		func(ctx context.Context, coll *mongo.Collection, aggregate *testAggregate) error {
			res, err := coll.InsertOne(ctx, aggregate)
			if err != nil {
				return fmt.Errorf("failed to insert aggregate: %w", err)
			}
			insertedID = res.InsertedID
			return nil
		},
	)

	s.Require().NoError(err)

	// assert that the aggregate is saved

	findOneRes := collection.FindOne(context.Background(), bson.M{
		"_id": insertedID,
	})

	s.Require().NoError(err)

	var foundAggregate testAggregate

	err = findOneRes.Decode(&foundAggregate)
	s.Require().NoError(err)

	s.Assert().Equal("someValue", foundAggregate.SomeField)
	s.Assert().Empty(foundAggregate.Buffer.Collect())
	s.Assert().Equal(aggregate.ID, foundAggregate.ID)

	// assert that the event is not saved
	allEvents, err := mongo_outbox.FindAllEvents(context.Background(), s.mongo.DB)
	s.Require().NoError(err)
	s.Assert().Empty(allEvents)
}
