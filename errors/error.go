package errors

import (
	"reflect"

	"github.com/golly-go/golly/utils"
	"github.com/sirupsen/logrus"
)

// Error struct holds the wrapped error
type Error struct {
	Key         string                 `json:"key"`
	Err         error                  `json:"-"`
	Status      int                    `json:"-"`
	Caller      string                 `json:"-"`
	ErrorString string                 `json:"error,omitempty"`
	Data        map[string]interface{} `json:"field_errors,omitempty"`
}

func (ae Error) Error() string {
	return ae.ErrorString
}

func (ae Error) ToLogFields() logrus.Fields {
	return logrus.Fields{
		"key":    ae.Key,
		"error":  ae.Err,
		"caller": ae.Caller,
	}
}

// NewError returns a new error resolving format and other stuff
func (ae Error) NewError(err error) Error {
	source := ""
	if e, ok := err.(Error); ok {
		source = e.Caller
	} else {
		source = utils.FileWithLineNum()
	}

	e := Error{Key: ae.Key, Err: err, Caller: source, Status: ae.Status}

	er := ae.Err
	if er == nil {
		er = err
	}

	e.ErrorString = er.Error()
	return e
}

func SetData(err error, key string, value interface{}) error {
	if err != nil {
		if ae, ok := err.(Error); ok {
			if ae.Data == nil {
				ae.Data = map[string]interface{}{}
			}
			ae.Data[key] = value
			return ae
		}
	}
	return err
}

// WrapWithStatus wraps a standard error with given status used in return on rigging
func WrapWithStatus(ae Error, err error, status int) error {
	if err == nil {
		return nil
	}

	if e, ok := reflect.ValueOf(err).Interface().(Error); ok {
		if e.Status == 0 {
			status = e.Status
		}
		return e
	}

	n := ae.NewError(err)
	n.Status = status
	return n
}

// Wrap wraps an error
func Wrap(ae Error, err error) error {
	if err == nil {
		return nil
	}

	if e, ok := err.(Error); ok {
		return ae.NewError(e)
	}

	return ae.NewError(err)
}
