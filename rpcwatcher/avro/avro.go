package avro

import (
	"encoding/json"
	"net"
	"net/url"
	"reflect"
	"strings"
	"time"
)

var AvroSchemas []AvroSchema
var (
	timeType = reflect.TypeOf(time.Time{}) // date-time RFC section 7.3.1
	ipType   = reflect.TypeOf(net.IP{})    // ipv4 and ipv6 RFC section 7.3.4, 7.3.5
	uriType  = reflect.TypeOf(url.URL{})   // uri RFC section 7.3.6
)

type AvroSchema struct {
	Type      string      `json:"type"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace,omitempty"`
	Fields    []AvroField `json:"fields,omitempty"`
	Values    []AvroField `json:"values,omitempty"`
	Items     interface{} `json:"items,omitempty"`
}

type AvroField struct {
	Name string      `json:"name"`
	Type interface{} `json:"type"`
}

func GenerateAvroSchema(model interface{}) (string, error) {

	typ := strctTyp(reflect.TypeOf(model))
	root := reflect.StructField{
		Name:    typ.Name(),
		Type:    typ,
		PkgPath: typ.PkgPath(),
	}
	record := getAvroRecords(root, "", true)

	st, err := json.Marshal(record)
	//fmt.Printf("%s", st)
	//spew.Dump(record)

	return string(st), err
}

func getAvroRecords(model reflect.StructField, namespace string, root bool) interface{} {
	typ := strctTyp(model.Type)
	//if the type is []type
	if model.Type.Kind() == reflect.Slice {
		//access the underlying slice type
		switch model.Type.Elem().Kind() {
		//if it's a complex type then do something with it - slice is represented as array in avro
		case reflect.Struct, reflect.Ptr, reflect.Array, reflect.Map, reflect.Slice:
			typ := model.Type.Elem()
			struct_type := reflect.StructField{

				Name:    typ.Name(),
				Type:    typ,
				PkgPath: typ.PkgPath(),
			}

			record := []interface{}{
				"null",
				AvroSchema{
					Type:      "array",
					Name:      getFieldName(struct_type),
					Namespace: namespace,
					Items:     getAvroRecords(struct_type, namespace, false),
				},
			}
			return record
			// if it's a primitive type, return string instead
		default:
			return "string"
		}
	}

	record := AvroSchema{
		Type:      "record",
		Name:      getFieldName(model),
		Namespace: namespace,
	}
	if !root {
		switch namespace {
		case "":
			namespace = getFieldName(model)
		default:
			namespace = namespace + "." + getFieldName(model)
		}
	} else {
		namespace = getFieldName(model)
	}
	fields := getAvroFields(typ, namespace)

	switch typ.Kind() {
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
		f := typ.Field(i)
		var kind reflect.Kind
		switch f.Type {
		case timeType:
			kind = reflect.String
		case ipType:
			kind = reflect.String
		case uriType:
			kind = reflect.String
		default:
			kind = f.Type.Kind()
		}

		switch kind {
		case reflect.Struct, reflect.Ptr, reflect.Array, reflect.Map, reflect.Slice:
			variable_type = getAvroRecords(f, namespace, false)
		case reflect.Float32:
			variable_type = "float"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			variable_type = "int"
		case reflect.Int64, reflect.Uint64:
			variable_type = "long"
		case reflect.Float64:
			variable_type = "double"
		case reflect.Interface:
		case reflect.Bool:
			variable_type = "boolean"

		case reflect.String:
			variable_type = "string"
		default:
			panic("unsupported type " + f.Type.String())
		}

		field := AvroField{
			Name: getFieldName(f),
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

func getFieldName(t reflect.StructField) string {
	var name string
	name = strings.Split(t.Tag.Get("json"), ",")[0]
	if name == "" {
		name = t.Name
	}

	return name
}
