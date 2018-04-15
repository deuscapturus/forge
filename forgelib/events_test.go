package forgelib

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
)

type mockEvents struct {
	stackEventsOutput cloudformation.DescribeStackEventsOutput
	cloudformationiface.CloudFormationAPI
}

func (m mockEvents) DescribeStackEventsPages(input *cloudformation.DescribeStackEventsInput, function func(*cloudformation.DescribeStackEventsOutput, bool) bool) error {
	// Paginate events to test that the destination functions concatenate the
	// entries correctly
	for i := 0; i < len(m.stackEventsOutput.StackEvents); i++ {
		thisOutput := &cloudformation.DescribeStackEventsOutput{
			StackEvents: []*cloudformation.StackEvent{
				m.stackEventsOutput.StackEvents[i],
			},
		}
		if nextPage := function(thisOutput, true); !nextPage {
			return nil
		}
	}
	return nil
}

func TestListEvents(t *testing.T) {
	cases := []struct {
		after    time.Time
		expected []*cloudformation.StackEvent
		resp     cloudformation.DescribeStackEventsOutput
	}{
		{
			after: time.Unix(150, 0),
			resp: cloudformation.DescribeStackEventsOutput{
				StackEvents: []*cloudformation.StackEvent{
					{Timestamp: aws.Time(time.Unix(300, 0))},
					{Timestamp: aws.Time(time.Unix(100, 0))},
					{Timestamp: aws.Time(time.Unix(200, 0))},
				},
			},
			expected: []*cloudformation.StackEvent{
				{Timestamp: aws.Time(time.Unix(200, 0))},
				{Timestamp: aws.Time(time.Unix(300, 0))},
			},
		},
		{
			after: time.Unix(100, 0),
			resp: cloudformation.DescribeStackEventsOutput{
				StackEvents: []*cloudformation.StackEvent{
					{Timestamp: aws.Time(time.Unix(300, 0))},
					{Timestamp: aws.Time(time.Unix(100, 0))},
					{Timestamp: aws.Time(time.Unix(200, 0))},
				},
			},
			expected: []*cloudformation.StackEvent{
				{Timestamp: aws.Time(time.Unix(200, 0))},
				{Timestamp: aws.Time(time.Unix(300, 0))},
			},
		},
		{
			after: time.Unix(50, 0),
			resp: cloudformation.DescribeStackEventsOutput{
				StackEvents: []*cloudformation.StackEvent{
					{Timestamp: aws.Time(time.Unix(300, 0))},
					{Timestamp: aws.Time(time.Unix(100, 0))},
					{Timestamp: aws.Time(time.Unix(200, 0))},
				},
			},
			expected: []*cloudformation.StackEvent{
				{Timestamp: aws.Time(time.Unix(100, 0))},
				{Timestamp: aws.Time(time.Unix(200, 0))},
				{Timestamp: aws.Time(time.Unix(300, 0))},
			},
		},
		{
			after: time.Unix(350, 0),
			resp: cloudformation.DescribeStackEventsOutput{
				StackEvents: []*cloudformation.StackEvent{
					{Timestamp: aws.Time(time.Unix(300, 0))},
					{Timestamp: aws.Time(time.Unix(100, 0))},
					{Timestamp: aws.Time(time.Unix(200, 0))},
				},
			},
			expected: []*cloudformation.StackEvent{},
		},
	}
	oldCFNClient := cfnClient
	defer func() { cfnClient = oldCFNClient }()
	for i, c := range cases {
		cfnClient = mockEvents{stackEventsOutput: c.resp}

		s := Stack{StackID: "whatever"}
		events, err := s.ListEvents(&c.after)
		if err != nil {
			t.Fatalf("%d, unexpected error, %v", i, err)
		}
		for j, event := range events {
			if a, e := *event.Timestamp, *c.expected[j].Timestamp; a != e {
				t.Errorf("%d, expected %v event, got %v", i, e, a)
			}
		}
	}
}

func TestListEventsNoStackID(t *testing.T) {
	s := Stack{}

	if _, err := s.ListEvents(aws.Time(time.Unix(0, 0))); err == nil {
		t.Errorf("expected error, got success")
	}
}

func TestGetLastEventTime(t *testing.T) {
	cases := []struct {
		resp     cloudformation.DescribeStackEventsOutput
		expected time.Time
	}{
		{
			resp: cloudformation.DescribeStackEventsOutput{
				StackEvents: []*cloudformation.StackEvent{
					{Timestamp: aws.Time(time.Unix(100, 0))},
					{Timestamp: aws.Time(time.Unix(300, 0))},
					{Timestamp: aws.Time(time.Unix(200, 0))},
				},
			},
			expected: time.Unix(300, 0),
		},
	}
	oldCFNClient := cfnClient
	defer func() { cfnClient = oldCFNClient }()
	for i, c := range cases {
		cfnClient = mockEvents{stackEventsOutput: c.resp}

		s := Stack{StackID: "whatever"}
		result, err := s.GetLastEventTime()
		if err != nil {
			t.Fatalf("%d, unexpected error, %v", i, err)
		}
		if *result != c.expected {
			t.Errorf("%d, expected %v time, got %v", i, c.expected, result)
		}
	}
}
