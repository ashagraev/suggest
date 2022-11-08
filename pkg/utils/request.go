package utils

import (
  "github.com/spf13/cast"
  "google.golang.org/protobuf/proto"
  "google.golang.org/protobuf/reflect/protoreflect"
  "net/http"
  "net/url"
)

func ReadRequest(request *http.Request, message proto.Message) error {
  if request.Method == http.MethodGet {
    err := request.ParseForm()
    var fieldsMap map[string]protoreflect.FieldDescriptor
    if err == nil {
      if fieldsMap == nil {
        fieldsMap = analyseProtoMessage(message)
      }
      readRequestValues(request.Form, message, fieldsMap)
    }
  }
  return nil
}

func readRequestValues(query url.Values, message proto.Message, messageTags map[string]protoreflect.FieldDescriptor) {
  for key, values := range query {
    if fieldDescriptor, ok := messageTags[key]; ok {
      if fieldDescriptor.Cardinality() == protoreflect.Repeated {
        l := message.ProtoReflect().Mutable(fieldDescriptor).List()
        for _, value := range values {
          l.Append(getReflectedQueryValue(value, fieldDescriptor))
        }
        message.ProtoReflect().Set(fieldDescriptor, protoreflect.ValueOfList(l))
      } else {
        for _, value := range values {
          message.ProtoReflect().Set(fieldDescriptor, getReflectedQueryValue(value, fieldDescriptor))
          break
        }
      }
    }
  }
}

func analyseProtoMessage(message proto.Message) map[string]protoreflect.FieldDescriptor {
  fields := message.ProtoReflect().Type().Descriptor().Fields()
  fieldsMap := make(map[string]protoreflect.FieldDescriptor)
  for ind := 0; ind < fields.Len(); ind++ {
    fieldDescriptor := fields.Get(ind)
    fieldsMap[fieldDescriptor.TextName()] = fieldDescriptor
  }
  return fieldsMap
}

func getReflectedQueryValue(value string, fieldDescriptor protoreflect.FieldDescriptor) protoreflect.Value {
  var reflectedValue protoreflect.Value
  switch fieldDescriptor.Kind() {
  case protoreflect.BoolKind:
    reflectedValue = protoreflect.ValueOfBool(cast.ToBool(value))
  case protoreflect.StringKind:
    reflectedValue = protoreflect.ValueOfString(value)
  case protoreflect.Int64Kind:
    reflectedValue = protoreflect.ValueOfInt64(cast.ToInt64(value))
  case protoreflect.Int32Kind:
    reflectedValue = protoreflect.ValueOfInt32(cast.ToInt32(value))
  default:
    reflectedValue = protoreflect.ValueOf(value)
  }
  return reflectedValue
}
