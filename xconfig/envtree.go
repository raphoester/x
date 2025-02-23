package xconfig

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type envTree struct {
	mapping map[string]string
}

func ApplyEnv(value interface{}) error {
	tree := &envTree{mapping: make(map[string]string)}
	for _, env := range os.Environ() {
		idx := strings.Index(env, "=")
		if idx > 0 {
			key := env[:idx]
			val := env[idx+1:]
			tree.mapping[key] = val
		}
	}

	if err := tree.applyRec("", reflect.ValueOf(value), nil); err != nil {
		return fmt.Errorf("apply finished with errors: %w", err)
	}

	return nil
}

func (t *envTree) applyRec(currentPath string, value reflect.Value, retErr error) error {
	if t == nil {
		return errors.New("env tree is nil")
	}

	if val, ok := t.mapping[currentPath]; ok {
		switch value.Kind() {
		case reflect.String:
			value.SetString(val)
		case reflect.Int:
			intVal, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("unable to parse int value at %s: %v", currentPath, err))
			} else {
				value.SetInt(intVal)
			}
		case reflect.TypeOf(time.Duration(0)).Kind():
			durationVal, err := time.ParseDuration(val)
			if err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("unable to parse duration value at %s: %v", currentPath, err))
			} else {
				value.SetInt(int64(durationVal))
			}
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(val)
			if err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("unable to parse bool value at %s: %v", currentPath, err))
			} else {
				value.SetBool(boolVal)
			}
		case reflect.Slice:
			if value.Type().Elem().Kind() == reflect.Uint8 {
				decoded, err := base64.StdEncoding.DecodeString(val)
				if err != nil {
					retErr = errors.Join(retErr, fmt.Errorf("unable to decode base64 at %s: %v", currentPath, err))
				} else {
					newSlice := reflect.MakeSlice(value.Type(), len(decoded), len(decoded))
					reflect.Copy(newSlice, reflect.ValueOf(decoded))
					value.Set(newSlice)
				}
			} else {
				retErr = errors.Join(retErr, fmt.Errorf("unimplemented slice type %v at path %s", value.Type().Elem().Kind(), currentPath))
			}
		default:
			retErr = errors.Join(retErr, fmt.Errorf("unimplemented value type %v(%v) at path %s", val, value.Kind(), currentPath))
		}
	}

	typeVal := value.Type()
	for typeVal.Kind() == reflect.Ptr {
		if value.IsNil() {
			return retErr
		}
		typeVal = typeVal.Elem()
		value = value.Elem()
	}

	if typeVal.Kind() == reflect.Struct {
		for i := 0; i < typeVal.NumField(); i++ {
			field := typeVal.Field(i)
			var nextPath string
			if t, ok := field.Tag.Lookup("yaml"); ok {
				nextPath = strings.ToUpper(t)
			} else {
				nextPath = strings.ToUpper(field.Name)
			}

			if len(currentPath) != 0 {
				nextPath = fmt.Sprintf("%s_%s", currentPath, nextPath)
			}

			retErr = t.applyRec(nextPath, value.Field(i), retErr)
		}
	}

	return retErr
}
