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
	Namespace string      `json:"namespace"`
	Fields    []AvroField `json:"fields,omitempty"`
	Values    []AvroField `json:"values,omitempty"`
	Items     []AvroField `json:"items,omitempty"`
}

type AvroField struct {
	Name string      `json:"name"`
	Type interface{} `json:"type"`
}

func GenerateAvroSchema(model interface{}) string {
	record := getAvroRecords(reflect.TypeOf(model), reflect.TypeOf(model).Name())
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

func getAvroRecords(model reflect.Type, namespace string) AvroSchema {
	val := strctTyp(model)
	switch namespace {
	case "":
		namespace = val.Name()
	default:
		namespace = namespace + "." + val.Name()
	}

	fields := getAvroFields(val, namespace)
	record := AvroSchema{
		Type: "record",
		//Name:      model.String(),
		Name:      val.Name(),
		Namespace: namespace,
		Fields:    fields,
	}

	return record
}
func getAvroMaps(model reflect.Type, namespace string) AvroSchema {
	val := strctTyp(model)
	namespace = namespace + "." + val.Name()

	fields := getAvroFields(val, namespace)
	record := AvroSchema{
		Type:      "map",
		Name:      model.String(),
		Namespace: namespace + "." + val.Name(),
		Values:    fields,
	}

	return record
}
func getAvroArrays(model reflect.Type, namespace string) AvroSchema {
	val := strctTyp(model)
	namespace = namespace + "." + val.Name()

	fields := getAvroFields(val, namespace)
	record := AvroSchema{
		Type:      "array",
		Name:      model.String(),
		Namespace: namespace + "." + val.Name(),
		Items:     fields,
	}

	return record
}

func getAvroFields(model reflect.Type, namespace string) []AvroField {
	fields := make([]AvroField, 0, 10)
	val := strctTyp(model)
	//spew.Dump(val)
	var typ interface{}
	for i := 0; i < val.NumField(); i++ {
		t := val.Field(i)
		switch t.Type.Kind() {

		case reflect.Struct, reflect.Ptr:
			typ = getAvroRecords(t.Type, namespace)
		case reflect.Array:
			typ = getAvroArrays(t.Type, namespace)
		case reflect.Map:
			typ = getAvroMaps(t.Type, namespace)
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
