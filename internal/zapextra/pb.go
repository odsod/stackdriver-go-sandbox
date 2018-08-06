package zapextra

import (
	"encoding/json"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

func Proto(key string, message proto.Message) zap.Field {
	marshaller := jsonpb.Marshaler{}
	jsonString, err := marshaller.MarshalToString(message)
	if err != nil {
		return zap.Any(key, message)
	}
	return zap.Any(key, json.RawMessage(jsonString))
}
