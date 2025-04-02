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

	s.broker, err = rabbitmq_broker.New(rabbitMQ.RabbitMQ, xlog.NewTestLogger(s.T()))
	s.Require().NoError(err)
}

func (s *testSuite) TearDownSuite() {
	if err := s.rabbitMQ.Destroy(); err != nil {
		s.T().Errorf("failed to destroy rabbitMQ: %v", err)
	}
}

func (s *testSuite) SetupTest() {
	_ = s.rabbitMQ.Clean()
	var err error
	s.broker, err = rabbitmq_broker.New(s.rabbitMQ.RabbitMQ, xlog.NewTestLogger(s.T()))
	s.Require().NoError(err)
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

	err := s.broker.Listen(
		listenCtx,
		"test",
		[]string{customTopicName},
		xevents.HandlerPair{
			Topic: customTopicName, // route the event to its own handler through its topic name
			Handler: xevents.UnmarshalHelper(
				func(ctx context.Context, event *xevents.Event, payload *xevents.ExamplePayload) error {
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

	valueShouldBe := "value"

	event, err := xevents.New(
		timeProvider,
		idGenerator,
		xevents.ExamplePayload{Key: "value"}.WithTopic(customTopicName),
	)
	s.Require().NoError(err)
	err = s.broker.Publish(context.Background(), event)
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
	pairs := make([]xevents.HandlerPair, 0, eventsSentCount)
	for key := range expectedKeysSet {
		pairs = append(pairs, xevents.HandlerPair{
			Topic: key,
			Handler: xevents.UnmarshalHelper(
				func(ctx context.Context, event *xevents.Event, payload xevents.ExamplePayload) error {
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
	err := s.broker.Listen(listenCtx, "test", []string{"topic.*"}, pairs...)
	s.Assert().NoError(err)

	for key := range expectedKeysSet {
		expectedKeysSet[key] = struct{}{}
		event, err := xevents.New(timeProvider, idGenerator, xevents.ExamplePayload{Key: key}.WithTopic(key))
		s.Require().NoError(err)
		err = s.broker.Publish(context.Background(), event)
		s.Require().NoError(err)
	}

	event, err := xevents.New(
		timeProvider,
		idGenerator,
		xevents.ExamplePayload{Key: "value"}.WithTopic("xyz.not.matching.topic.name"),
	)
	s.Require().NoError(err)
	err = s.broker.Publish(context.Background(), event)
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
	err := s.broker.Listen(context.Background(), "test", []string{"topic1"}, xevents.HandlerPair{
		Topic: "topic1",
		Handler: func(ctx context.Context, event *xevents.Event) error {
			return nil
		},
	})
	s.Assert().NoError(err)

	err = s.broker.Listen(context.Background(), "test", []string{"topic2"}, xevents.HandlerPair{
		Topic: "topic2",
		Handler: func(ctx context.Context, event *xevents.Event) error {
			return nil
		},
	})
	s.Assert().NoError(err)

}

func (s *testSuite) TestListenOnMultipleKeys() {

	ranTopicA := false
	ranTopicB := false
	ranTopicC := false

	err := s.broker.Listen(context.Background(), "test", []string{"topic.a.*", "topic.b.*"}, xevents.HandlerPair{
		Topic: "topic.a.test",
		Handler: func(ctx context.Context, event *xevents.Event) error {
			ranTopicA = true
			return nil
		},
	}, xevents.HandlerPair{
		Topic: "topic.b.test",
		Handler: func(ctx context.Context, event *xevents.Event) error {
			ranTopicB = true
			return nil
		},
	}, xevents.HandlerPair{ // this one should not be called
		Topic: "topic.c.test",
		Handler: func(ctx context.Context, event *xevents.Event) error {
			ranTopicC = true
			return nil
		},
	})
	s.Assert().NoError(err)

	makeEvent := func(key, topic string) *xevents.Event {
		event, err := xevents.New(xtime.NewDefaultFixedProvider(), xid.RandomGenerator{}, xevents.ExamplePayload{Key: key}.WithTopic(topic))
		s.Require().NoError(err)
		return event
	}

	err = s.broker.Publish(
		context.Background(),
		makeEvent("a", "topic.a.test"))
	s.Require().NoError(err)

	err = s.broker.Publish(
		context.Background(),
		makeEvent("b", "topic.b.test"))
	s.Require().NoError(err)

	err = s.broker.Publish(
		context.Background(),
		makeEvent("c", "topic.c.test"))
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
