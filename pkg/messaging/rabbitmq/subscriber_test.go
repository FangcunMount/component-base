package rabbitmq

import (
	"strings"
	"testing"

	"github.com/FangcunMount/component-base/pkg/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestNewSubscriberWithOptionsRequiresFailedMessageHandlerBeforeDial(t *testing.T) {
	_, err := NewSubscriberWithOptions("amqp://invalid", messaging.SubscriberOptions{MaxAttempts: 8})
	if err == nil || !strings.Contains(err.Error(), "failed-message handler") {
		t.Fatalf("bounded subscriber error = %v", err)
	}
}

func TestNewSubscriberWithOptionsRejectsNegativeMaxAttemptsBeforeDial(t *testing.T) {
	_, err := NewSubscriberWithOptions("amqp://invalid", messaging.SubscriberOptions{MaxAttempts: -1})
	if err == nil || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("negative max-attempts error = %v", err)
	}
}

func TestRabbitDeliveryMetadata(t *testing.T) {
	tests := []struct {
		name           string
		headers        amqp.Table
		wantAttempt    int
		wantFailedOnly bool
		wantErr        bool
	}{
		{name: "legacy first delivery", wantAttempt: 1},
		{name: "attempt seven", headers: amqp.Table{rabbitDeliveryAttemptHeader: int32(7)}, wantAttempt: 7},
		{name: "failed-only attempt eight", headers: amqp.Table{rabbitDeliveryAttemptHeader: "8", rabbitFailedOnlyHeader: true}, wantAttempt: 8, wantFailedOnly: true},
		{name: "attempt nine", headers: amqp.Table{rabbitDeliveryAttemptHeader: int64(9)}, wantAttempt: 9},
		{name: "invalid attempt", headers: amqp.Table{rabbitDeliveryAttemptHeader: "bad"}, wantAttempt: 1, wantFailedOnly: true, wantErr: true},
		{name: "zero attempt", headers: amqp.Table{rabbitDeliveryAttemptHeader: int32(0)}, wantAttempt: 1, wantFailedOnly: true, wantErr: true},
		{name: "invalid failed-only", headers: amqp.Table{rabbitDeliveryAttemptHeader: int32(3), rabbitFailedOnlyHeader: "not-bool"}, wantAttempt: 3, wantFailedOnly: true, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attempt, failedOnly, err := rabbitDeliveryMetadata(test.headers)
			if attempt != test.wantAttempt || failedOnly != test.wantFailedOnly || (err != nil) != test.wantErr {
				t.Fatalf("metadata = %d/%v/%v", attempt, failedOnly, err)
			}
		})
	}
}

func TestRabbitAttemptCompatibilityHelperDefaultsOnInvalidHeader(t *testing.T) {
	if got := rabbitAttempt(amqp.Table{rabbitDeliveryAttemptHeader: "bad"}); got != 1 {
		t.Fatalf("rabbitAttempt() = %d", got)
	}
}
