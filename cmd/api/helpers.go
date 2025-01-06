package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/s-devoe/greenlight-go/internal/validator"
)

type envelope map[string]interface{}

func (app *application) background(fn func()) {
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		defer func() {
			if err := recover(); err != nil {
				app.logger.PrintError(fmt.Errorf("%s", err), nil)
			}

		}()
		fn()
	}()
}

// qs = query string
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}
	return s
}

func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)

	if csv == "" {
		return defaultValue
	}
	return strings.Split(csv, ",")
}

func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}

func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())
	// below means, convert the id string(all values in params are strings ) to a base 10 64 bit interger
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}
	return id, nil
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	js = append(js, '\n')
	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	//use max byte to limit the size of the request body to 1MB to prevent against DOS attacks
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		// errors.As() function to check whether the error has the type *json.SyntaxError.
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON at character %d", syntaxError.Offset)
		//In causes where Decode()  may return an io.ErrUnexpectedEOF error for syntax errors in the JSON. so we check using errors.Is()
		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("body contains badly-formed JSON")
			// *json.UnmarshalTypeError occurs when the JSON value id the wrong type for the target destination
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
			// am io.EOF error will be returned by Decode() if the request body is empty.
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
			//if JSON contains an incorrect field value, thw disallowed fields will let Decode() return an error in the format
			//"json:unknown field "<fieldname>""
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("body contains unknown field %s", fieldName)
		// if request body exceeds 1MB in size the Decode() returns an error "http: request body too large"
		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)
		//a json.invalidUnmarshalError error will be returned if we pass a non-nil pointer to Decode(). catch this an panic, rather than returning an error.
		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}
	// if the body contains multiple JSON fields in one request, i.e { "foo":"bar"} and another one again {"foo1":"bar1"}
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must contain a single JSON value")
	}

	return nil
}
