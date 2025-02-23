package rabbitmq_broker_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/raphoester/x/xdockertest"
	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xevents/rabbitmq_broker"
	"github.com/raphoester/x/xid"
	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xtime"
	"github.com/stretchr/testify/suite"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	rabbitMQ *xdockertest.RabbitMQ
	broker   *rabbitmq_broker.Broker
	suite.Suite
}

func (s *testSuite) SetupSuite() {
	rabbitMQ, err := xdockertest.NewRabbitMQ(xlog.NewTestLogger(s.T()))
	s.Require().NoError(err)
	s.rabbitMQ = rabbitMQ
	s.broker, err = rabbitmq_broker.New(rabbitMQ.RabbitMQ)
}

func (s *testSuite) TearDownSuite() {
	if err := s.rabbitMQ.Destroy(); err != nil {
		s.T().Errorf("failed to destroy rabbitMQ: %v", err)
	}
}

func (s *testSuite) SetupTest() {
	_ = s.rabbitMQ.Clean()
}

func (s *testSuite) TestPublish() {
	customTopicName := "topic.custom"

	timeProvider := xtime.NewDefaultFixedProvider()
	idGenerator := xid.NewDefaultFixedGenerator()
	expectedEventData := xevents.EventData{
		ID:        idGenerator.Generate(),
		CreatedAt: timeProvider.Now(),
		Topic:     customTopicName,
		Payload:   nil, // type is dynamic, can't perform assertion on it
	}

	ran := false
	retrievedValue := ""
	retrievedEventData := xevents.EventData{}

	// need to cancel the listener context to stop the test at the end
	listenCtx, cancelListen := context.WithCancel(context.Background())
	defer cancelListen()

	ready, err := s.broker.Listen(
		listenCtx,
		[]string{customTopicName},
		rabbitmq_broker.HandlerPair{
			Topic: customTopicName, // route the event to its own handler through its topic name
			Handler: rabbitmq_broker.UnmarshalHelper(
				func(ctx context.Context, event xevents.Event, payload xevents.ExamplePayload) error {
					ran = true
					retrievedValue = payload.Key
					retrievedEventData = event.Data()
					retrievedEventData.Payload = nil
					return nil
				},
			),
		},
	)
	s.Require().NoError(err)

	<-ready

	valueShouldBe := "value"
	err = s.broker.Publish(
		context.Background(), xevents.New(
			timeProvider,
			idGenerator,
			xevents.ExamplePayload{Key: "value"}.
				WithTopic(customTopicName),
		),
	)
	s.Require().NoError(err)

	iterations := 10
	for i := range iterations {
		if ran {
			break
		}

		s.T().Logf("%d/%d seconds of waiting for callback to execute (0/1)", i, iterations)
		time.Sleep(1 * time.Second)
	}

	s.Require().True(ran)
	s.Assert().Equal(valueShouldBe, retrievedValue)
	s.Assert().Equal(expectedEventData, retrievedEventData)
}

func (s *testSuite) TestStreamOnlyAPartOfThePublishedEvents() {
	timeProvider := xtime.NewDefaultFixedProvider()
	idGenerator := xid.RandomGenerator{}

	ranCounter := 0

	eventsSentCount := 10
	expectedKeysSet := map[string]struct{}{}

	for i := 0; i < eventsSentCount; i++ {
		expectedKeysSet[fmt.Sprintf("topic.%d", i+1)] = struct{}{}
	}

	retrievedKeysSet := map[string]struct{}{}
	pairs := make([]rabbitmq_broker.HandlerPair, 0, eventsSentCount)
	for key := range expectedKeysSet {
		pairs = append(pairs, rabbitmq_broker.HandlerPair{
			Topic: key,
			Handler: rabbitmq_broker.UnmarshalHelper(
				func(ctx context.Context, event xevents.Event, payload xevents.ExamplePayload) error {
					ranCounter++
					retrievedKeysSet[payload.Key] = struct{}{}
					return nil
				},
			),
		})
	}

	// need to cancel the listener context to stop the test at the end
	listenCtx, cancelListen := context.WithCancel(context.Background())
	defer cancelListen()
	ready, err := s.broker.Listen(listenCtx, []string{"topic.*"}, pairs...)
	s.Assert().NoError(err)

	<-ready

	for key := range expectedKeysSet {
		expectedKeysSet[key] = struct{}{}
		err := s.broker.Publish(context.Background(),
			xevents.New(timeProvider, idGenerator, xevents.ExamplePayload{Key: key}.WithTopic(key)))
		s.Require().NoError(err)
	}

	err = s.broker.Publish(
		context.Background(),
		xevents.New(
			timeProvider,
			idGenerator,
			xevents.ExamplePayload{Key: "value"}.WithTopic("xyz.not.matching.topic.name"),
		),
	)
	s.Require().NoError(err)

	retriesCount := 10
	for i := range retriesCount {
		if ranCounter >= eventsSentCount {
			break
		}
		s.T().Logf("%d/%d seconds of waiting for all callbacks to execute (%d/%d)",
			i, retriesCount, ranCounter, eventsSentCount)
		time.Sleep(1 * time.Second)
	}

	s.Assert().Equal(eventsSentCount, ranCounter)
	s.Assert().Equal(expectedKeysSet, retrievedKeysSet)
}

func (s *testSuite) TestListenTwice() {

	ready1, err := s.broker.Listen(context.Background(), []string{"topic1"}, rabbitmq_broker.HandlerPair{
		Topic: "topic1",
		Handler: func(ctx context.Context, event xevents.Event) error {
			return nil
		},
	})
	s.Assert().NoError(err)
	<-ready1

	ready2, err := s.broker.Listen(context.Background(), []string{"topic2"}, rabbitmq_broker.HandlerPair{
		Topic: "topic2",
		Handler: func(ctx context.Context, event xevents.Event) error {
			return nil
		},
	})
	s.Assert().NoError(err)

	<-ready2
}

func (s *testSuite) TestListenOnMultipleKeys() {

	ranTopicA := false
	ranTopicB := false
	ranTopicC := false

	ready, err := s.broker.Listen(context.Background(), []string{"topic.a.*", "topic.b.*"}, rabbitmq_broker.HandlerPair{
		Topic: "topic.a.test",
		Handler: func(ctx context.Context, event xevents.Event) error {
			ranTopicA = true
			return nil
		},
	}, rabbitmq_broker.HandlerPair{
		Topic: "topic.b.test",
		Handler: func(ctx context.Context, event xevents.Event) error {
			ranTopicB = true
			return nil
		},
	}, rabbitmq_broker.HandlerPair{ // this one should not be called
		Topic: "topic.c.test",
		Handler: func(ctx context.Context, event xevents.Event) error {
			ranTopicC = true
			return nil
		},
	})
	s.Assert().NoError(err)

	<-ready

	err = s.broker.Publish(
		context.Background(),
		xevents.New(xtime.NewDefaultFixedProvider(), xid.RandomGenerator{}, xevents.ExamplePayload{Key: "a"}.WithTopic("topic.a.test")))
	s.Require().NoError(err)

	err = s.broker.Publish(
		context.Background(),
		xevents.New(xtime.NewDefaultFixedProvider(), xid.RandomGenerator{}, xevents.ExamplePayload{Key: "a"}.WithTopic("topic.b.test")))
	s.Require().NoError(err)

	err = s.broker.Publish(
		context.Background(),
		xevents.New(xtime.NewDefaultFixedProvider(), xid.RandomGenerator{}, xevents.ExamplePayload{Key: "a"}.WithTopic("topic.c.test")))
	s.Require().NoError(err)

	iterations := 10
	for i := range iterations {
		if ranTopicA && ranTopicB {
			break
		}

		s.T().Logf("%d/%d seconds of waiting for all callbacks to execute (0/2)", i, iterations)
		time.Sleep(1 * time.Second)
	}

	s.Assert().True(ranTopicA)
	s.Assert().True(ranTopicB)
	s.Assert().False(ranTopicC)
}
