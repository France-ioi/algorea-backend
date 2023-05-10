// Package formdata provides an interface to handle forms.
package formdata

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/France-ioi/mapstructure"
	"github.com/France-ioi/validator"
	"github.com/France-ioi/validator/translations/en"
	english "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"gorm.io/gorm/schema"

	"github.com/France-ioi/AlgoreaBackend/app/database"
)

// FormData can parse JSON, validate it and construct a map for updating DB.
type FormData struct {
	definitionStructure interface{}
	fieldErrors         FieldErrors
	metadata            mapstructure.Metadata
	usedKeys            map[string]bool
	decodeErrors        map[string]bool
	oldValues           interface{}
	allowUnknownFields  bool

	validate *validator.Validate
	trans    ut.Translator
}

const (
	set    = "set"
	null   = "null"
	squash = "squash"
)

// NewFormData creates a new FormData object for given definitions.
func NewFormData(definitionStructure interface{}) *FormData {
	// Initialize go-playground/validator
	validate := validator.New()

	// Initialize go-playground/validator's default error messages in English
	eng := english.New()
	uni := ut.New(eng, eng)
	trans, _ := uni.GetTranslator("en")
	_ = en.RegisterDefaultTranslations(validate, trans)

	// go-playground/validator should read field names from 'json' tag
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		parts := strings.Split(fld.Tag.Get("json"), ",")
		name := parts[0]
		if name == "-" {
			return ""
		}
		for i := 0; i < len(parts); i++ {
			if parts[i] == squash {
				return "<squash>"
			}
		}
		return name
	})

	formData := &FormData{
		definitionStructure: definitionStructure,
		validate:            validate,
		trans:               trans,
	}

	// Register global custom validations

	// This one is needed to check if the field is set
	formData.RegisterValidation(set, func(fl validator.FieldLevel) bool {
		path := formData.getUsedKeysPathFromValidatorPath(fl.Path())
		return formData.usedKeys[path]
	})
	formData.RegisterTranslation(set, "missing field")

	// This one is needed to check if the field is null
	formData.RegisterValidation(null, formData.ValidatorSkippingUnsetFields(validateNull))
	formData.RegisterTranslation(null, "should be null")

	formData.RegisterValidation("duration", validateDuration)
	formData.RegisterTranslation("duration", "invalid duration")

	formData.RegisterValidation("dmy-date", validateDMYDate)
	formData.RegisterTranslation("dmy-date", "should be dd-mm-yyyy")

	return formData
}

// RegisterValidation adds a validation with the given tag.
func (f *FormData) RegisterValidation(tag string, fn validator.Func) {
	_ = f.validate.RegisterValidation(tag, fn)
}

// RegisterTranslation registers translations against the provided tag.
func (f *FormData) RegisterTranslation(tag, text string) {
	_ = f.validate.RegisterTranslation(tag, f.trans,
		func(ut ut.Translator) (err error) {
			err = ut.Add(tag, text, false)
			if err != nil {
				panic(err)
			}
			return err
		}, func(_ ut.Translator, fe validator.FieldError) string {
			return fmt.Sprintf("%.0[1]s"+text, fe.Tag(), fe.Field(), fe.Param()) // %.0[1]s is needed to suppress the EXTRA suffix
		})
}

// SetOldValues sets the internal pointer to the structure containing old values for validation.
func (f *FormData) SetOldValues(oldValues interface{}) {
	f.oldValues = oldValues
}

// AllowUnknownFields disables the check for unknown fields in data.
func (f *FormData) AllowUnknownFields() {
	f.allowUnknownFields = true
}

// ParseJSONRequestData parses and validates JSON from the request according to the structure definition.
func (f *FormData) ParseJSONRequestData(r *http.Request) error {
	if err := f.decodeRequestJSONDataIntoStruct(r); err != nil {
		return err
	}

	return f.checkAndValidate()
}

// ParseMapData parses and validates map[string]interface{} according to the structure definition.
func (f *FormData) ParseMapData(m map[string]interface{}) error {
	f.decodeMapIntoStruct(m)
	return f.checkAndValidate()
}

func (f *FormData) checkAndValidate() error {
	f.checkProvidedFields()
	f.validateFieldValues()

	if len(f.fieldErrors) > 0 {
		return f.fieldErrors
	}

	return nil
}

// ConstructMapForDB constructs a map for updating DB. It uses both the definition and the JSON input.
func (f *FormData) ConstructMapForDB() map[string]interface{} {
	result := map[string]interface{}{}
	f.addDBFieldsIntoMap(result, reflect.ValueOf(f.definitionStructure), "")
	return result
}

// ConstructPartialMapForDB constructs a map for updating DB. It uses both the definition and the JSON input.
func (f *FormData) ConstructPartialMapForDB(part string) map[string]interface{} {
	result := map[string]interface{}{}
	partField := reflect.ValueOf(f.definitionStructure).Elem().FieldByName(part)
	partFieldType, _ := reflect.TypeOf(f.definitionStructure).Elem().FieldByName(part)
	prefix := getJSONFieldName(&partFieldType)
	if getJSONSquash(&partFieldType) {
		prefix = ""
	}
	if partField.Kind() == reflect.Ptr {
		partField = partField.Elem()
	}
	f.addDBFieldsIntoMap(result, partField, prefix)
	return result
}

// IsSet returns true if the field is set.
func (f *FormData) IsSet(key string) bool {
	return f.usedKeys[key]
}

// IsValid returns true if the field is valid (there are no errors for this field).
func (f *FormData) IsValid(key string) bool {
	return len(f.fieldErrors[key]) == 0
}

// ValidatorSkippingUnsetFields constructs a validator checking only fields given by the user.
func (f *FormData) ValidatorSkippingUnsetFields(nestedValidator validator.Func) validator.Func {
	return func(fl validator.FieldLevel) bool {
		path := f.getUsedKeysPathFromValidatorPath(fl.Path())
		if !f.IsSet(path) {
			return true
		}
		return nestedValidator(fl)
	}
}

// ValidatorSkippingUnchangedFields constructs a validator checking only fields with changed values.
// You might want to call f.SetOldValues(oldValues) before in order to provide the form with previous field values.
func (f *FormData) ValidatorSkippingUnchangedFields(nestedValidator validator.Func) validator.Func {
	return f.ValidatorSkippingUnsetFields(func(fl validator.FieldLevel) bool {
		structPath := fl.StructPath()
		structPath = structPath[strings.IndexRune(structPath, '.')+1:]
		oldValue := getFieldValueByStructPath(reflect.ValueOf(f.oldValues), structPath)
		newValue := getFieldValueByStructPath(fl.Field(), "")
		if newValue == oldValue {
			return true
		}
		return nestedValidator(fl)
	})
}

var (
	mapstructTypeErrorRegexp     = regexp.MustCompile(`^'([^']*)'\s+(.*)$`)
	mapstructDecodingErrorRegexp = regexp.MustCompile(`^error decoding '([^']*)':\s+(.*)$`)
)

func (f *FormData) decodeRequestJSONDataIntoStruct(r *http.Request) error {
	var rawData map[string]interface{}
	defer func() { _, _ = io.Copy(ioutil.Discard, r.Body) }()
	err := json.NewDecoder(r.Body).Decode(&rawData)
	if err != nil {
		return err
	}
	f.decodeMapIntoStruct(rawData)
	return nil
}

func (f *FormData) decodeMapIntoStruct(m map[string]interface{}) {
	f.fieldErrors = make(FieldErrors)
	f.usedKeys = make(map[string]bool)
	f.decodeErrors = make(map[string]bool)
	f.metadata = mapstructure.Metadata{}

	var decoder *mapstructure.Decoder
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: f.definitionStructure,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeHookFunc(time.RFC3339),
			stringToDatabaseTimeUTCHookFunc(time.RFC3339),
			toAnythingHookFunc(),
			stringToInt64HookFunc(),
		),
		ErrorUnused:      false, // we will check this on our own
		Metadata:         &f.metadata,
		TagName:          "json",
		ZeroFields:       true, // this marks keys with null values as used
		WeaklyTypedInput: false,
	})
	if err != nil {
		panic(err) // this error can only be caused by bugs in the code
	}

	if err = decoder.Decode(m); err != nil {
		mapstructureErr := err.(*mapstructure.Error)
		for _, fieldErrorString := range mapstructureErr.Errors { // Convert mapstructure's errors to our format
			if matches := mapstructTypeErrorRegexp.FindStringSubmatch(fieldErrorString); len(matches) > 0 {
				key := make([]byte, len(matches[1]))
				copy(key, matches[1])
				value := make([]byte, len(matches[2]))
				copy(value, matches[2])
				f.fieldErrors[string(key)] = append(f.fieldErrors[string(key)], string(value))
				f.usedKeys[string(key)] = true
				f.decodeErrors[string(key)] = true
			} else if matches := mapstructDecodingErrorRegexp.FindStringSubmatch(fieldErrorString); len(matches) > 0 {
				key := make([]byte, len(matches[1]))
				copy(key, matches[1])
				f.fieldErrors[string(key)] = append(f.fieldErrors[string(key)], "decoding error: "+matches[2])
				f.usedKeys[string(key)] = true
				f.decodeErrors[string(key)] = true
			} else {
				f.fieldErrors[""] = append(f.fieldErrors[""], fieldErrorString) // should never happen
			}
		}
	}
}

func (f *FormData) validateFieldValues() {
	if err := f.validate.Struct(f.definitionStructure); err != nil {
		f.processValidatorErrors(err)
	}
}

func (f *FormData) processValidatorErrors(err error) {
	validatorErrors := err.(validator.ValidationErrors)
	for _, validatorError := range validatorErrors {
		path := validatorError.Namespace()
		path = f.getUsedKeysPathFromValidatorPath(path)
		if (f.usedKeys[path] || validatorError.Tag() == set) && !f.decodeErrors[path] {
			errorMsg := validatorError.Translate(f.trans)
			f.fieldErrors[path] = append(f.fieldErrors[path], errorMsg)
		}
	}
}

func (f *FormData) getUsedKeysPathFromValidatorPath(path string) string {
	prefix := ""

	structType := reflect.TypeOf(f.definitionStructure)
	for structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	structName := structType.Name()
	if structName != "" {
		prefix = structName + "."
	}

	path = strings.TrimPrefix(path, prefix)
	path = strings.Replace(path, ".<squash>", "", -1)
	path = strings.Replace(path, "<squash>.", "", -1)
	path = strings.Replace(path, "<squash>", "", -1)

	return path
}

func (f *FormData) checkProvidedFields() {
	if !f.allowUnknownFields {
		for _, unusedKey := range f.metadata.Unused {
			f.fieldErrors[unusedKey] = append(f.fieldErrors[unusedKey], "unexpected field")
		}
	}
	for _, usedKey := range f.metadata.Keys {
		f.usedKeys[usedKey] = true
	}
}

func (f *FormData) addDBFieldsIntoMap(resultMap map[string]interface{}, reflValue reflect.Value, prefix string) {
	traverseStructure(func(field reflect.Value, structField reflect.StructField, jsonName string) bool {
		if _, ok := f.usedKeys[jsonName]; !ok {
			return false
		}

		dbName := schema.NamingStrategy{}.ColumnName("", structField.Name)

		for _, str := range []string{structField.Tag.Get("sql"), structField.Tag.Get("gorm")} {
			tags := strings.Split(str, ";")
			for _, value := range tags {
				v := strings.Split(value, ":")
				key := strings.TrimSpace(v[0])
				if key == "-" {
					return false // skip this field
				}
				var value string
				if len(v) >= 2 {
					value = strings.Join(v[1:], ":")
				} else {
					value = key
				}
				if key == "column" {
					dbName = value
				}
			}
		}

		fieldType := field.Type()

		// For now, all the fields from nested structures will be set in the root map.
		// Yet the nested structures themselves will not be in the map
		if field.Kind() != reflect.Struct || fieldType.PkgPath()+"/"+fieldType.Name() == "time/Time" {
			resultMap[dbName] = field.Interface()
		}

		return true
	}, reflValue, prefix)
}

func traverseStructure(fn func(fieldValue reflect.Value, structField reflect.StructField, jsonName string) bool,
	reflValue reflect.Value, prefix string,
) {
	for reflValue.Kind() == reflect.Ptr {
		reflValue = reflValue.Elem()
	}
	numberOfFields := reflValue.NumField()

	for i := 0; i < numberOfFields; i++ {
		field := reflValue.Field(i)
		structField := reflValue.Type().Field(i)

		jsonName := getJSONFieldName(&structField)
		if jsonName == "-" { // skip fields ignored in json
			continue
		}
		squash := getJSONSquash(&structField)
		result := true
		if !squash {
			if len(prefix) > 0 {
				jsonName = prefix + "." + jsonName
			}

			result = fn(field, structField, jsonName)
		} else {
			jsonName = prefix
		}

		if result && field.Kind() == reflect.Struct {
			traverseStructure(fn, field, jsonName)
		}
	}
}

func getJSONFieldName(structField *reflect.StructField) string {
	jsonTagParts := strings.Split(structField.Tag.Get("json"), ",")
	if jsonTagParts[0] == "" {
		return "-"
	}
	return jsonTagParts[0]
}

func getJSONSquash(structField *reflect.StructField) bool {
	jsonTagParts := strings.Split(structField.Tag.Get("json"), ",")
	for i := 1; i < len(jsonTagParts); i++ {
		if jsonTagParts[i] == squash {
			return true
		}
	}
	return false
}

func getFieldValueByStructPath(value reflect.Value, structPath string) interface{} {
	if !value.IsValid() {
		return nil
	}
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	if structPath == "" {
		return value.Interface()
	}

	var fieldName string
	nameLength := strings.IndexRune(structPath, '.')
	if nameLength >= 0 {
		fieldName = structPath[0:nameLength]
		structPath = structPath[nameLength+1:]
	} else {
		fieldName = structPath
		structPath = ""
	}

	return getFieldValueByStructPath(value.FieldByName(fieldName), structPath)
}

// toAnythingHookFunc returns a DecodeHookFunc that converts
// any value to payloads.Anything.
func toAnythingHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if t.Name() != "Anything" || t.PkgPath() != "github.com/France-ioi/AlgoreaBackend/app/formdata" {
			return data, nil
		}

		if f.Kind() == reflect.Slice && f.Elem().Kind() == reflect.Uint8 {
			return *AnythingFromBytes(data.([]byte)), nil
		}
		bytes, _ := json.Marshal(data)
		return *AnythingFromBytes(bytes), nil
	}
}

// stringToInt64HookFunc returns a DecodeHookFunc that converts
// strings to int64.
func stringToInt64HookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if t.Kind() != reflect.Int64 || f.Kind() != reflect.String {
			return data, nil
		}
		return strconv.ParseInt(data.(string), 10, 64)
	}
}

// stringToDatabaseTimeUTCHookFunc returns a DecodeHookFunc that converts strings to database.Time in UTC.
func stringToDatabaseTimeUTCHookFunc(layout string) mapstructure.DecodeHookFunc {
	timeDecodeFunc := mapstructure.StringToTimeHookFunc(layout)

	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.String || t.Name() != "Time" || t.PkgPath() != "github.com/France-ioi/AlgoreaBackend/app/database" {
			return data, nil
		}
		converted, err := mapstructure.DecodeHookExec(timeDecodeFunc, f, reflect.TypeOf((*time.Time)(nil)).Elem(), data)
		return database.Time(converted.(time.Time).UTC()), err
	}
}
