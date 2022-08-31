package avro

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
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
	st, err := json.Marshal(getAvroRecords(reflect.TypeOf(model)))
	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, st, "", "\t")
	if err != nil {
		log.Err(err).Msg("")
		return ""
	}
	if error != nil {
		log.Err(error).Msg("failed to parse json")
	}
	return prettyJSON.String()
}

func getAvroRecords(model reflect.Type) []AvroSchema {

	val := strctTyp(model)
	spew.Dump(val)
	record := AvroSchema{
		Type:      "record",
		Name:      val.Name(),
		Namespace: "some.namespace",
		Fields:    getAvroFields(val),
	}
	AvroSchemas = append(AvroSchemas, record)
	return AvroSchemas
}
func getAvroFields(model reflect.Type) []AvroField {
	fields := make([]AvroField, 0, 10)

	val := strctTyp(model)
	//spew.Dump(val)

	if val.Kind() != reflect.Struct {
		field := AvroField{
			Name: val.Name(),
			Type: val.Kind().String(),
		}
		fields = append(fields, field)
		return fields
	}
	for i := 0; i < val.NumField(); i++ {
		t := val.Field(i)

		if val.Field(i).Type.Kind() == reflect.Struct || val.Field(i).Type.Kind() == reflect.Ptr {
			field := AvroField{
				Name: t.Name,
				Type: val.Field(i).Type.Kind().String(),
			}
			fields = append(fields, field)
			//Skip private/non exportable fields
			_ = getAvroRecords(val.Field(i).Type)
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
		spew.Dump(s)
	}
	//spew.Dump(t)
	//n := reflect.TypeOf(s).Elem().Name()
	//fmt.Println(n)
	// if pointer get the value
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		spew.Dump(f)
	}

	// for v.Kind() == reflect.Ptr {

	// 	spew.Dump(v.Elem())

	// 	v = reflect.Indirect(reflect.ValueOf(s))
	// 	// spew.Dump(reflect.ValueOf(&s))
	// 	// spew.Dump(reflect.Indirect(reflect.ValueOf(&s)))
	// 	// spew.Dump(reflect.New(v.Type().Elem()))
	// }

	return s
}
