package optshttp

import (
  "net/http"
  "time"
  "github.com/gorilla/mux"
  "fmt"
  "strconv"
  "reflect"
  "unsafe"
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
    if tagStr, ok := field.Tag.Lookup(tagKey); ok {
      if (len(tagStr) == 0) {
        continue
      }
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
          if err := setValue(v.Field(i), tagFields[0], tagFields[0]); err != nil {
            return err
          }
        }
      }
    }
  }
  return nil
}

var pStrType = reflect.TypeOf((*string)(nil))
var pIntType = reflect.TypeOf((*int)(nil))
var pUintType = reflect.TypeOf((*uint)(nil))
var timeType = reflect.TypeOf(time.Time{})
var pTimeType = reflect.PtrTo(timeType)

func setValue(v reflect.Value, formKey string, formStr string) error {
  switch v.Kind() {
    case reflect.String:
      v.SetString(formStr)
    case reflect.Int:
      if val, err := strconv.ParseInt(formStr, 10, 0); err != nil{
        return &optsError{"integer", formKey, formStr}     
      } else {
        v.SetInt(val)
      }
    case reflect.Uint:
      if val, err := strconv.ParseUint(formStr, 10, 0); err != nil{
        return &optsError{"unsigned integer", formKey, formStr}     
      } else {
        v.SetUint(val)
      }
    case reflect.Struct:
      if v.Type() == timeType {
        t := (*time.Time)(unsafe.Pointer(v.UnsafeAddr()))
        if err := t.UnmarshalText([]byte(formStr)); err != nil {
          return &optsError{"time in RFC3339 format", formKey, formStr}     
        }
      }
    case reflect.Ptr:
      if ! v.IsNil() {
        return setValue(v.Elem(), formKey, formStr)
      }
      switch v.Type() {
        case pStrType:
          p := (**string)(unsafe.Pointer(v.UnsafeAddr()))
          *p = &formStr
        case pIntType:
          if val, err := strconv.Atoi(formStr); err != nil{
            return &optsError{"integer", formKey, formStr}     
          } else {
            p := (**int)(unsafe.Pointer(v.UnsafeAddr()))
            *p = &val
          }
        case pUintType:
          if val, err := strconv.ParseUint(formStr, 10, 0); err != nil{
            return &optsError{"unsigned integer", formKey, formStr}     
          } else {
            p := (**uint64)(unsafe.Pointer(v.UnsafeAddr()))
            *p = &val
          }
        case pTimeType:
          if val, err := time.Parse(time.RFC3339Nano, formStr); err != nil{
            return &optsError{"time in RFC3339 format", formKey, formStr}     
          } else {
            p := (**time.Time)(unsafe.Pointer(v.UnsafeAddr()))
            *p = &val
          }
      }
  }
  return nil
}
