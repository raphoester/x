package mongo_versionning_test

import (
	"context"
	"testing"

	"github.com/raphoester/x/xdockertest"
	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xmongo/mongo_versionning"
	"github.com/raphoester/x/xver"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	coll := s.mongo.Client.Database("versioning_test").Collection("test_aggregates")
	return coll
}

type testAggregate struct {
	id        string
	someField string
	*xevents.Buffer
	*xver.Version
}

type TestAggregateSnapshot struct {
	ID        string `bson:"_id"`
	SomeField string `bson:"somefield"`
	Version   int
}

func (t *testAggregate) TakeSnapshot() *TestAggregateSnapshot {
	return &TestAggregateSnapshot{
		ID:        t.id,
		SomeField: t.someField,
		Version:   t.Version.Current(),
	}
}

func (t *testAggregate) ID() string {
	return t.id
}

func (s *testSuite) TestUpsert_WithConflictOnExtraField() {
	collection := s.getTestCollection()
	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.M{
			"somefield": 1,
		},
		Options: options.Index().SetUnique(true),
	})
	s.Require().NoError(err)

	toSave := &testAggregate{
		id:        "someID",
		someField: "someValue",
		Buffer:    xevents.NewBuffer(),
		Version:   xver.New(),
	}

	err = mongo_versionning.Upsert(context.Background(), collection, toSave)
	s.Require().NoError(err)

	toSave.Version = xver.Restore(0)

	// simulate a new instance of the aggregate
	toSave.someField = "someOtherValue"
	toSave.RecordNewModification()

	// Upsert again with a different value for the extra field
	err = mongo_versionning.Upsert(context.Background(), collection, toSave)
	s.Require().NoError(err)
}
