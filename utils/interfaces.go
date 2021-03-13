package utils

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

func GetBytes(key interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return []byte{}
	}
	return buf.Bytes()
}

func GetType(myvar interface{}) string {
	t := reflect.TypeOf(myvar)
	if t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	}
	return t.Name()
}

func CastData(obj reflect.Type, data map[string]interface{}) reflect.Value {
	dataValue := reflect.New(obj).Elem()
	typeOfT := dataValue.Type()

	for i := 0; i < dataValue.NumField(); i++ {
		field := typeOfT.Field(i)
		tag := field.Tag.Get("json")

		fl := dataValue.FieldByName(field.Name)

		if fieldValue, ok := data[tag]; ok {
			switch fl.Interface().(type) {
			case bool:
				if val, ok := fieldValue.(bool); ok {
					fl.SetBool(val)
				}
			case int, int64:
				if val, ok := fieldValue.(int64); ok {
					fl.SetInt(val)
				}
			case float64:
				if val, ok := fieldValue.(float64); ok {
					fl.SetFloat(val)
				}
			case string:
				if val, ok := fieldValue.(string); ok {
					fl.SetString(val)
				}
			case time.Time:
				if val, ok := fieldValue.(string); ok {
					if t, err := time.Parse(time.RFC3339, val); err == nil {
						fl.Set(reflect.ValueOf(t))
					}
				} else {
					if val, ok := fieldValue.(time.Time); ok {
						fl.Set(reflect.ValueOf(val))
					}
				}
			case uuid.UUID:
				fmt.Printf("%#v\n", fieldValue)

				if val, ok := fieldValue.(string); ok {
					if t, err := uuid.Parse(val); err == nil {
						fl.Set(reflect.ValueOf(t))
					}
				} else {
					if val, ok := fieldValue.(uuid.UUID); ok {
						fl.Set(reflect.ValueOf(val))
					}
				}
				// default:
				// 	switch fl.Kind() {
				// 	case reflect.Slice, reflect.Array:
				// 		fmt.Printf("%#v\n", v)

				// 		for i := 0; i < v.Len(); i++ {
				// 			if d, ok := v.Index(i).Interface().(map[string]interface{}); ok {
				// 				fl.Set(CastData(v.Index(i).Type(), d))
				// 			} else {
				// 				fl.Set(reflect.ValueOf(v.Index(i).Interface()))
				// 			}
				// 		}
			default:
				if d, ok := fieldValue.(map[string]interface{}); ok {
					fl.Set(CastData(fl.Type(), d))
				}
				// }
			}
		}
	}
	return dataValue
}
