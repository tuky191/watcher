package avro

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/rs/zerolog/log"
)

var AvroSchemas []AvroSchema

type AvroSchema struct {
	Type      string      `json:"type"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace,omitempty"`
	Fields    []AvroField `json:"fields,omitempty"`
	Values    []AvroField `json:"values,omitempty"`
	Items     []AvroField `json:"items,omitempty"`
}

type AvroField struct {
	Name string      `json:"name"`
	Type interface{} `json:"type"`
}

func GenerateAvroSchema(model interface{}) string {
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
func getAvroMaps(model reflect.Type) AvroSchema {
	val := strctTyp(model)

	record := AvroSchema{
		Type:   "map",
		Name:   val.Name(),
		Values: getAvroFields(val),
	}

	return record
}
func getAvroArrays(model reflect.Type) AvroSchema {
	val := strctTyp(model)

	record := AvroSchema{
		Type:  "array",
		Name:  val.Name(),
		Items: getAvroFields(val),
	}

	return record
}

func getAvroFields(model reflect.Type) []AvroField {
	fields := make([]AvroField, 0, 10)

	val := strctTyp(model)
	var typ interface{}
	for i := 0; i < val.NumField(); i++ {
		t := val.Field(i)
		switch t.Type.Kind() {

		case reflect.Struct, reflect.Ptr:
			typ = getAvroRecords(t.Type)
		case reflect.Array:
			typ = getAvroArrays(t.Type)
		case reflect.Map:
			typ = getAvroMaps(t.Type)
		case reflect.Float32:
			typ = "float"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			typ = "int"
		case reflect.Int64, reflect.Uint64:
			typ = "long"
		case reflect.Float64:
			typ = "double"
		case reflect.Slice:
			typ = "bytes"
		case reflect.Interface:
		case reflect.Bool:
			typ = "boolean"

		case reflect.String:
			typ = "string"
		default:
			panic("unsupported type " + t.Type.String())
		}

		field := AvroField{
			Name: t.Name,
			Type: typ,
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
