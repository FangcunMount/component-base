package eventmessaging

import (
	"github.com/FangcunMount/component-base/pkg/event"
	"github.com/FangcunMount/component-base/pkg/eventcodec"
	"github.com/FangcunMount/component-base/pkg/messaging"
)

func BuildMessage(evt event.DomainEvent, source string, encoders ...eventcodec.PayloadEncoder) (*messaging.Message, error) {
	encoder := eventcodec.EncodeDomainEvent
	if len(encoders) > 0 && encoders[0] != nil {
		encoder = encoders[0]
	}
	payload, err := encoder(evt)
	if err != nil {
		return nil, err
	}
	msg := messaging.NewMessage(evt.EventID(), payload)
	msg.Metadata = eventcodec.MetadataFromEvent(evt, source)
	return msg, nil
}
