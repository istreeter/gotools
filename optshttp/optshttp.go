package optshttp

import (
  "net/http"
  "time"
  "github.com/gorilla/mux"
  "fmt"
  "strconv"
  "reflect"
  "unsafe"
)

func UnmarshalForm(req *http.Request, v interface{}) error {
  return unmarshal(v, "form", func(s string) string {return req.FormValue(s)})
}

func UnmarshalPath(req *http.Request, v interface{}) error {
  vars := mux.Vars(req)
  return unmarshal(v, "path", func(s string) string {return vars[s]})
}

type optsError struct{
  optType string
  optKey string
  optVal string
}

func (e *optsError) Error() string {
  return fmt.Sprintf("Invalid %s %s: %s", e.optType, e.optKey, e.optVal)
}

func unmarshal(v interface{}, tagKey string, varLookup func(string) string) error {
  vv := reflect.ValueOf(v).Elem()
  vt := vv.Type()
  numField := vt.NumField()
  for i := 0; i < numField; i++ {
    field := vt.Field(i)
    if formKey, ok := field.Tag.Lookup(tagKey); ok {
      formStr := varLookup(formKey)
      if (len(formStr) == 0) {
        continue
      }
      if err := setValue(vv.Field(i), formKey, formStr); err != nil {
        return err
      }
    }
  }
  return nil
}

var pStrType = reflect.TypeOf((*string)(nil))
var pIntType = reflect.TypeOf((*int)(nil))
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
