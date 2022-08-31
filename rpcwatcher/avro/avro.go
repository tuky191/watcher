package avro

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/rs/zerolog/log"
)

var AvroSchemas []AvroSchema

type AvroSchema struct {
	Type      string      `json:"type"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Fields    []AvroField `json:"fields"`
}

type AvroField struct {
	Name string      `json:"name"`
	Type interface{} `json:"type"`
}

func GenerateAvroSchema(model interface{}) string {
	fmt.Printf("%#v\n", model)
	record := getAvroRecords(reflect.TypeOf(model))
	st, err := json.Marshal(record)
	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, st, "", " ")
	if err != nil {
		log.Err(err).Msg("")
		return ""
	}
	if error != nil {
		log.Err(error).Msg("failed to parse json")
	}
	return prettyJSON.String()
}

func getAvroRecords(model reflect.Type) AvroSchema {
	val := strctTyp(model)

	record := AvroSchema{
		Type:      "record",
		Name:      val.Name(),
		Namespace: "some.namespace",
		Fields:    getAvroFields(val),
	}

	return record
}
func getAvroFields(model reflect.Type) []AvroField {
	fields := make([]AvroField, 0, 10)

	val := strctTyp(model)
	for i := 0; i < val.NumField(); i++ {
		t := val.Field(i)

		if val.Field(i).Type.Kind() == reflect.Struct || val.Field(i).Type.Kind() == reflect.Ptr {

			// switch val.Field(i).Type.Kind() {

			// }
			field := AvroField{
				Name: t.Name,
				Type: getAvroRecords(val.Field(i).Type),
			}
			fields = append(fields, field)
			continue
		}

		field := AvroField{
			Name: t.Name,
			Type: val.Field(i).Type.Kind().String(),
		}
		fields = append(fields, field)
	}
	return fields
}

func strctTyp(s reflect.Type) reflect.Type {
	if s.Kind() == reflect.Ptr {
		s = s.Elem()
	}

	return s
}
