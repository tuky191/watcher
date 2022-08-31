package avro

import (
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
	st, err := json.Marshal(getAvroRecords(model))
	if err != nil {
		log.Err(err).Msg("")
		return ""
	}
	return string(st)
}

func getAvroRecords(model interface{}) []AvroSchema {

	val := strctVal(model)
	spew.Dump(val)
	v := val.Type()
	record := AvroSchema{
		Type:      "record",
		Name:      v.Name(),
		Namespace: "some.namespace",
		Fields:    getAvroFields(val.Interface()),
	}
	AvroSchemas = append(AvroSchemas, record)
	return AvroSchemas
}
func getAvroFields(model interface{}) []AvroField {
	fields := make([]AvroField, 0, 10)

	val := strctVal(model)
	spew.Dump(val)
	v := val.Type()

	if val.Kind() != reflect.Struct {
		field := AvroField{
			Name: v.Name(),
			Type: val.Kind().String(),
		}
		fields = append(fields, field)
		return fields
	}
	for i := 0; i < val.NumField(); i++ {
		t := v.Field(i)

		if val.Field(i).Kind() == reflect.Struct || val.Field(i).Kind() == reflect.Ptr {
			field := AvroField{
				Name: t.Name,
				Type: val.Field(i).Kind().String(),
			}
			fields = append(fields, field)
			//Skip private/non exportable fields
			if val.Field(i).CanInterface() {
				_ = getAvroRecords(val.Field(i).Interface())
			}
			continue
		}

		field := AvroField{
			Name: t.Name,
			Type: val.Field(i).Kind().String(),
		}
		fields = append(fields, field)
	}
	return fields
}

func strctVal(s interface{}) reflect.Value {
	v := reflect.ValueOf(&s)
	//spew.Dump(v)

	//n := reflect.TypeOf(s).Elem().Name()
	//fmt.Println(n)
	// if pointer get the value
	for v.Kind() == reflect.Ptr {
		v = reflect.Indirect(reflect.ValueOf(s))
		// spew.Dump(reflect.ValueOf(&s))
		// spew.Dump(reflect.Indirect(reflect.ValueOf(&s)))
		// spew.Dump(reflect.New(v.Type().Elem()))
	}

	return v
}
