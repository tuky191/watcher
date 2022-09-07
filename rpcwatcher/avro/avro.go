package avro

import (
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
	record := getAvroRecords(reflect.TypeOf(model), reflect.TypeOf(model).Name())
	st, err := json.Marshal(record)
	if err != nil {
		log.Err(err).Msg("")
		return ""
	}
	return string(st)
}

func getAvroRecords(model reflect.Type, namespace string) AvroSchema {
	typ := strctTyp(model)

	record := AvroSchema{
		Type:      "record",
		Name:      typ.Name(),
		Namespace: namespace,
	}
	switch namespace {
	case "":
		namespace = typ.Name()
	default:
		namespace = namespace + "." + typ.Name()
	}
	fields := getAvroFields(typ, namespace)

	switch model.Kind() {
	case reflect.Struct, reflect.Ptr:
		record.Fields = fields
	case reflect.Array:
		record.Items = fields
	case reflect.Map:
		record.Values = fields
	}

	return record
}

func getAvroFields(model reflect.Type, namespace string) []AvroField {
	var fields []AvroField
	var variable_type interface{}

	typ := strctTyp(model)
	for i := 0; i < typ.NumField(); i++ {
		t := typ.Field(i)
		switch t.Type.Kind() {
		case reflect.Struct, reflect.Ptr, reflect.Array, reflect.Map:
			variable_type = getAvroRecords(t.Type, namespace)
		case reflect.Float32:
			variable_type = "float"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			variable_type = "int"
		case reflect.Int64, reflect.Uint64:
			variable_type = "long"
		case reflect.Float64:
			variable_type = "double"
		case reflect.Slice:
			variable_type = "string"
		case reflect.Interface:
		case reflect.Bool:
			variable_type = "boolean"

		case reflect.String:
			variable_type = "string"
		default:
			panic("unsupported type " + t.Type.String())
		}

		field := AvroField{
			Name: t.Name,
			Type: variable_type,
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
