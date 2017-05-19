package optshttp

import (
  "net/http"
  "time"
  "github.com/gorilla/mux"
  "fmt"
  "strconv"
  "reflect"
  "strings"
)

const(
  flagForm = "form"
  flagPath = "path"
  flagInline = "inline"
)

func UnmarshalForm(req *http.Request, v interface{}) error {
  return unmarshalStruct(reflect.ValueOf(v).Elem(), flagForm, func(s string) string {return req.FormValue(s)})
}

func UnmarshalPath(req *http.Request, v interface{}) error {
  vars := mux.Vars(req)
  return unmarshalStruct(reflect.ValueOf(v).Elem(), flagPath, func(s string) string {return vars[s]})
}

type optsError struct{
  optType string
  optKey string
  optVal string
}

func (e *optsError) Error() string {
  return fmt.Sprintf("Invalid %s %s: %s", e.optType, e.optKey, e.optVal)
}

func unmarshalStruct(v reflect.Value, tagKey string, varLookup func(string) string) error {
  vt := v.Type()
  numField := vt.NumField()
  for i := 0; i < numField; i++ {
    field := vt.Field(i)
    if tagStr := field.Tag.Get(tagKey); len(tagStr) != 0 {
      tagFields := strings.Split(tagStr, ",")
      if len(tagFields) > 1 {
        for _, flag := range tagFields[1:] {
          switch flag {
            case flagInline:
              if err := unmarshalStruct(v.Field(i), tagKey, varLookup); err != nil {
                return err
              }
          }
        }
      }
      if len(tagFields) > 0 && len(tagFields[0]) > 0 {
        if formStr := varLookup(tagFields[0]); len(formStr) > 0 {
          if err := setValue(v.Field(i), tagFields[0], formStr); err != nil {
            return err
          }
        }
      }
    }
  }
  return nil
}

var timeType = reflect.TypeOf(time.Time{})
var monthType = reflect.TypeOf(time.Month(1))

func setValue(v reflect.Value, formKey string, formStr string) error {
  switch v.Kind() {
    case reflect.String:
      v.SetString(formStr)
    case reflect.Int:
      val, err := strconv.ParseInt(formStr, 10, 0)
      if  err != nil{
        return &optsError{"integer", formKey, formStr}     
      }
      if v.Type() == monthType {
        if val < 0 || val > 12 {
          return &optsError{"date", formKey, formStr}     
        }
      }
      v.SetInt(val)
    case reflect.Uint:
      val, err := strconv.ParseUint(formStr, 10, 0)
      if err != nil{
        return &optsError{"unsigned integer", formKey, formStr}     
      }
      v.SetUint(val)
    case reflect.Struct:
      if v.Type() == timeType {
        t := &time.Time{}
        if err := t.UnmarshalText([]byte(formStr)); err != nil {
          return &optsError{"time in RFC3339 format", formKey, formStr}     
        }
        v.Set(reflect.ValueOf(*t))
      }
    case reflect.Ptr:
      if v.IsNil() {
          v.Set(reflect.New(v.Type().Elem()))
      }
      return setValue(v.Elem(), formKey, formStr)
  }
  return nil
}
