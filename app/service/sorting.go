package service

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/France-ioi/AlgoreaBackend/app/database"
)

// FieldType represents a type of tie-breaker field's value.
type FieldType string

// Value type of 'from.*' field in HTTP requests.
const (
	FieldTypeInt64  FieldType = "int64"
	FieldTypeBool   FieldType = "bool"
	FieldTypeString FieldType = "string"
	FieldTypeTime   FieldType = "time"
)

// FieldSortingParams represents sorting parameters for one field.
type FieldSortingParams struct {
	// ColumnName is a DB column name (should contain a table name as a prefix, e.g. "groups.id")
	ColumnName string
	// Nullable means that the field can be null
	Nullable bool
}

type sortingDirection int

const (
	asc  sortingDirection = 1
	desc sortingDirection = -1
)

type nullPlacement int

const (
	first nullPlacement = -1
	last  nullPlacement = 1
)

type sortingType struct {
	sortingDirection
	nullPlacement
}

func (t sortingType) asSQL(columnName string) string {
	var result string
	switch t.nullPlacement {
	case last:
		result += columnName + " IS NULL, "
	case first:
		result += columnName + " IS NOT NULL, "
	}
	result += columnName + " "
	if t.sortingDirection == asc {
		return result + "ASC"
	}
	return result + "DESC"
}

func (t sortingType) conditionSign() string {
	if t.sortingDirection == asc {
		return ">"
	}
	return "<"
}

// FromFirstRow is a special value of SortingAndPagingParameters.StartFromRowSubQuery
// needed to bypass pagination completely and ignore 'from.*' parameters of the HTTP request.
const FromFirstRow = iota

// SortingAndPagingFields is a type of SortingAndPagingParameters.Fields (field_name -> params).
type SortingAndPagingFields map[string]*FieldSortingParams

// SortingAndPagingTieBreakers is a type of SortingAndPagingParameters.TieBreakers (field_name -> type).
type SortingAndPagingTieBreakers map[string]FieldType

// SortingAndPagingParameters represents sorting and paging parameters to apply.
type SortingAndPagingParameters struct {
	Fields       SortingAndPagingFields
	DefaultRules string
	TieBreakers  SortingAndPagingTieBreakers

	// If IgnoreSortParameter is true, the 'sort' parameter of the HTTP request is ignored
	IgnoreSortParameter bool

	// If StartFromRowSubQuery is not nil, 'from.*' parameters of the HTTP request are not analyzed,
	// the sub-query should provide tie-breaker fields instead.
	//
	// If StartFromRowSubQuery is FromFirstRow, pagination gets skipped.
	StartFromRowSubQuery interface{}
}

// ApplySortingAndPaging applies ordering and paging according to the given parameters
// taking into the account the URL parameters 'from.*' if needed.
//
// When `parameters.IgnoreSortParameter` is true, the 'sort' request parameter is ignored.
//
// When `parameters.StartFromRowSubQuery` is set, the 'from.*' request parameters are ignored.
func ApplySortingAndPaging(r *http.Request, query *database.DB, parameters *SortingAndPagingParameters) (*database.DB, APIError) {
	mustHaveValidTieBreakerFieldsList(parameters.Fields, parameters.TieBreakers)
	sortingRules := chooseSortingRules(r, parameters.DefaultRules, parameters.IgnoreSortParameter)

	usedFields, fieldsSortingTypes, err := parseSortingRules(sortingRules, parameters.Fields, parameters.TieBreakers)
	if err != nil {
		return nil, ErrInvalidRequest(err)
	}

	query = applyOrder(query, usedFields, parameters.Fields, fieldsSortingTypes)

	var fromValues map[string]interface{}
	if parameters.StartFromRowSubQuery == nil {
		fromValues, err = ParsePagingParameters(r, parameters.TieBreakers)
		if err != nil {
			return nil, ErrInvalidRequest(err)
		}
	}

	query = applyPagingConditions(query, usedFields, fieldsSortingTypes, parameters.Fields, fromValues, parameters.StartFromRowSubQuery)
	return query, NoError
}

// chooseSortingRules chooses which sorting rules to use.
// If urlQuery["sort"] is not present, the default sorting rules are used.
func chooseSortingRules(r *http.Request, defaultRules string, ignoreSortParameter bool) (sortingRules string) {
	sortingRules = defaultRules
	if !ignoreSortParameter {
		urlQuery := r.URL.Query()
		if len(urlQuery["sort"]) > 0 {
			sortingRules = urlQuery["sort"][0]
		}
	}
	return sortingRules
}

// parseSortingRules returns a slice with used fields and a map fieldName -> sortingType
// It also checks that there are no unallowed fields in the rules.
func parseSortingRules(sortingRules string, configuredFields map[string]*FieldSortingParams, tieBreakerFields map[string]FieldType) (
	usedFields []string, fieldsSortingTypes map[string]sortingType, err error,
) {
	sortStatements := strings.Split(sortingRules, ",")
	usedFields = make([]string, 0, len(sortStatements)+1)
	fieldsSortingTypes = make(map[string]sortingType, len(sortStatements)+1)
	for _, sortStatement := range sortStatements {
		fieldName, sorting := getFieldNameAndSortingTypeFromSortStatement(sortStatement)
		err = validateSortingField(fieldsSortingTypes, fieldName, configuredFields, sorting)
		if err != nil {
			return nil, nil, err
		}
		if configuredFields[fieldName].Nullable && sorting.nullPlacement == 0 {
			sorting.nullPlacement = first
		}
		fieldsSortingTypes[fieldName] = sorting
		usedFields = append(usedFields, fieldName)
	}
	tieBreakerFieldsToAdd := make([]string, 0, len(tieBreakerFields))
	for tieBreakerFieldName := range tieBreakerFields {
		if fieldsSortingTypes[tieBreakerFieldName].sortingDirection == 0 {
			fieldsSortingTypes[tieBreakerFieldName] = sortingType{sortingDirection: 1}
			tieBreakerFieldsToAdd = append(tieBreakerFieldsToAdd, tieBreakerFieldName)
		}
	}
	sort.Strings(tieBreakerFieldsToAdd)
	usedFields = append(usedFields, tieBreakerFieldsToAdd...)
	return usedFields, fieldsSortingTypes, err
}

func mustHaveValidTieBreakerFieldsList(configuredFields map[string]*FieldSortingParams, tieBreakerFields map[string]FieldType) {
	for fieldName, fieldType := range tieBreakerFields {
		if !map[FieldType]bool{
			FieldTypeString: true,
			FieldTypeInt64:  true,
			FieldTypeBool:   true,
			FieldTypeTime:   true,
		}[fieldType] {
			panic(fmt.Errorf("unsupported type %q for field %q", fieldType, fieldName))
		}
		if configuredFields[fieldName] == nil {
			panic(fmt.Errorf("no such field %q, cannot use it as a tie-breaker field", fieldName))
		}
		if configuredFields[fieldName].Nullable {
			panic(fmt.Errorf("a nullable field %q cannot be a tie-breaker field", fieldName))
		}
	}
}

func validateSortingField(
	fieldsSortingTypes map[string]sortingType, fieldName string, configuredFields map[string]*FieldSortingParams, sorting sortingType,
) error {
	if _, ok := fieldsSortingTypes[fieldName]; ok {
		return fmt.Errorf("a field cannot be a sorting parameter more than once: %q", fieldName)
	}
	if _, ok := configuredFields[fieldName]; !ok {
		return fmt.Errorf("unallowed field in sorting parameters: %q", fieldName)
	}
	if !configuredFields[fieldName].Nullable && sorting.nullPlacement != 0 {
		return fmt.Errorf("'null last' sorting cannot be applied to a non-nullable field: %q", fieldName)
	}
	return nil
}

// getFieldNameAndSortingTypeFromSortStatement extracts a field name and a sorting type
// from a given sorting statement.
// "id"   -> ("id", 1)
// "-name -> ("name", {-1, 0})
// "-date$ -> ("date", {-1, 1}) # null last.
func getFieldNameAndSortingTypeFromSortStatement(sortStatement string) (string, sortingType) {
	var direction sortingDirection
	var np nullPlacement
	if len(sortStatement) > 0 && sortStatement[0] == '-' {
		sortStatement = sortStatement[1:]
		direction = desc
	} else {
		direction = asc
	}
	if len(sortStatement) > 0 && sortStatement[len(sortStatement)-1] == '$' {
		np = last
		sortStatement = sortStatement[:len(sortStatement)-1]
	}
	fieldName := sortStatement
	return fieldName, sortingType{direction, np}
}

// applyOrder appends the "ORDER BY" statement to given query according to the given list of used fields,
// the fields configuration (configuredFields) and sorting types.
func applyOrder(query *database.DB, usedFields []string, configuredFields map[string]*FieldSortingParams,
	fieldsSortingTypes map[string]sortingType,
) *database.DB {
	usedFieldsNumber := len(usedFields)
	orderStrings := make([]string, 0, usedFieldsNumber)
	for _, field := range usedFields {
		orderStrings = append(orderStrings, fieldsSortingTypes[field].asSQL(configuredFields[field].ColumnName))
	}
	if len(orderStrings) > 0 {
		query = query.Order(strings.Join(orderStrings, ", "))
	}
	return query
}

// ParsePagingParameters returns a map with values provided for paging in a request URL (none or all should be present).
func ParsePagingParameters(r *http.Request, expectedFromFields map[string]FieldType) (map[string]interface{}, error) {
	fromValueIsMissing := false
	fromValueAccepted := false
	fromValues := make(map[string]interface{}, len(expectedFromFields))

	const fromPrefix = "from."
	urlQuery := r.URL.Query()
	var unknownFromFields []string
	for fieldName := range urlQuery {
		if strings.HasPrefix(fieldName, fromPrefix) {
			if _, ok := expectedFromFields[fieldName[len(fromPrefix):]]; !ok {
				unknownFromFields = append(unknownFromFields, fieldName)
			}
		}
	}
	if len(unknownFromFields) > 0 {
		sort.Strings(unknownFromFields)
		return nil, fmt.Errorf("unallowed paging parameters (%s)", strings.Join(unknownFromFields, ", "))
	}

	expectedFromFieldNames := make([]string, 0, len(expectedFromFields))
	for fieldName, fieldType := range expectedFromFields {
		expectedFromFieldNames = append(expectedFromFieldNames, fieldName)
		var value interface{}
		fromFieldName := fromPrefix + fieldName
		if len(r.URL.Query()[fromFieldName]) > 0 {
			var err error
			value, err = getFromValueForField(r, fieldName, fieldType)
			if err != nil {
				return nil, err
			}
			fromValueAccepted = true
		} else {
			fromValueIsMissing = true
			continue
		}
		fromValues[fieldName] = value
	}
	if fromValueAccepted && fromValueIsMissing {
		sort.Strings(expectedFromFieldNames)
		fromParameters := strings.Join(expectedFromFieldNames, ", from.")
		return nil, fmt.Errorf("all 'from' parameters (from.%s) or none of them must be present", fromParameters)
	}

	return fromValues, nil
}

// getFromValueForField returns a 'from' value (a paging parameter) for a given field.
// The value is converted according to fieldType.
func getFromValueForField(r *http.Request, fieldName string, fieldType FieldType) (result interface{}, err error) {
	fromFieldName := "from." + fieldName
	switch fieldType {
	case FieldTypeString:
		result, err = r.URL.Query()[fromFieldName][0], nil
	case FieldTypeInt64:
		result, err = ResolveURLQueryGetInt64Field(r, fromFieldName)
	case FieldTypeBool:
		result, err = ResolveURLQueryGetBoolField(r, fromFieldName)
	case FieldTypeTime:
		var timeResult time.Time
		timeResult, err = ResolveURLQueryGetTimeField(r, fromFieldName)
		result = (*database.Time)(&timeResult)
	}
	return result, err
}

var safeColumnNameRegexp = regexp.MustCompile("[^a-zA-Z_0-9]")

// applyPagingConditions adds filtering on paging values into the query.
func applyPagingConditions(query *database.DB, usedFields []string, fieldsSortingTypes map[string]sortingType,
	configuredFields map[string]*FieldSortingParams, fromValues map[string]interface{}, startFromRowSubQuery interface{},
) *database.DB {
	if startFromRowSubQuery == nil && len(fromValues) == 0 || startFromRowSubQuery == FromFirstRow {
		return query
	}

	// Note: Since all the tie-breaker columns are always used and usedFields cannot contain duplicates,
	// having the same number of elements in usedFields and fromValues mean we have all the needed data
	// for paging. At the same time, fromValues are empty (unknown) when startFromRowSubQuery is given,
	// meaning the sub-query is needed in this case.
	subQueryNeeded := len(usedFields) != len(fromValues) || startFromRowSubQuery != nil

	conditions, queryValues, safeColumnNames := constructPagingConditions(
		usedFields, configuredFields, subQueryNeeded, fieldsSortingTypes, fromValues)
	if len(conditions) > 0 {
		if subQueryNeeded {
			query = joinSubQueryForPaging(query, usedFields, configuredFields, startFromRowSubQuery, safeColumnNames, fromValues, conditions)
		} else {
			query = query.Where(strings.Join(conditions, " OR "), queryValues...)
		}
	}
	return query
}

func constructPagingConditions(usedFields []string, configuredFields map[string]*FieldSortingParams,
	subQueryNeeded bool, fieldsSortingTypes map[string]sortingType, fromValues map[string]interface{}) (
	conditions []string, queryValues []interface{}, safeColumnNames []string,
) {
	usedFieldsNumber := len(usedFields)
	safeColumnNames = make([]string, usedFieldsNumber)
	conditions = make([]string, 0, usedFieldsNumber)
	queryValuesPart := make([]interface{}, 0, usedFieldsNumber)
	queryValues = make([]interface{}, 0, (usedFieldsNumber+1)*usedFieldsNumber/2)
	var conditionPrefix string

	// Here we're constructing an expression like this one:
	// (col1 > val1) OR (col1 = val1 AND col2 > val2) OR (col1 = val1 AND col2 = val2 AND col3 > val3) OR ...
	for index, fieldName := range usedFields {
		safeColumnName := safeColumnNameRegexp.ReplaceAllLiteralString(fieldName, "_")
		safeColumnNames[index] = safeColumnName

		if len(conditionPrefix) > 0 {
			conditionPrefix += " AND "
		}
		columnName := configuredFields[fieldName].ColumnName
		if subQueryNeeded {
			condition := fmt.Sprintf("%s %s from_page.%s", columnName, fieldsSortingTypes[fieldName].conditionSign(), safeColumnName)
			if fieldsSortingTypes[fieldName].nullPlacement == first {
				condition = fmt.Sprintf("IF(from_page.%s IS NULL, %s IS NOT NULL, %s)", safeColumnName, columnName, condition)
			} else if fieldsSortingTypes[fieldName].nullPlacement == last {
				condition = fmt.Sprintf("IF(from_page.%s IS NULL, FALSE, %s IS NULL OR %s)", safeColumnName, columnName, condition)
			}
			conditions = append(conditions, fmt.Sprintf("(%s%s)", conditionPrefix, condition))
			conditionPrefix = fmt.Sprintf("%s%s <=> from_page.%s", conditionPrefix, columnName, safeColumnName)
		} else {
			// As here we deal with primary key columns which cannot be nullable,
			// we can omit tricks related to null placement.
			condition := fmt.Sprintf("%s %s ?", columnName, fieldsSortingTypes[fieldName].conditionSign())
			conditions = append(conditions, fmt.Sprintf("(%s%s)", conditionPrefix, condition))
			conditionPrefix = fmt.Sprintf("%s%s = ?", conditionPrefix, columnName)

			fromValue := fromValues[fieldName]
			queryValuesPart = append(queryValuesPart, fromValue)
			queryValues = append(queryValues, queryValuesPart...)
		}
	}
	return conditions, queryValues, safeColumnNames
}

func joinSubQueryForPaging(query *database.DB, usedFields []string, configuredFields map[string]*FieldSortingParams,
	startFromRowSubQuery interface{}, safeColumnNames []string, fromValues map[string]interface{}, conditions []string,
) *database.DB {
	if startFromRowSubQuery == nil {
		startFromRowQuery := query
		fieldsToSelect := make([]string, 0, len(fromValues))
		for index, fieldName := range usedFields {
			fieldsToSelect = append(fieldsToSelect,
				fmt.Sprintf("%s AS %s", configuredFields[fieldName].ColumnName, safeColumnNames[index]))
		}
		for fieldName := range fromValues {
			startFromRowQuery = startFromRowQuery.
				Where(fmt.Sprintf("%s <=> ?", configuredFields[fieldName].ColumnName), fromValues[fieldName])
		}
		startFromRowQuery = startFromRowQuery.Select(strings.Join(fieldsToSelect, ", "))
		startFromRowSubQuery = startFromRowQuery.Limit(1).SubQuery()
	}
	query = query.
		Joins("JOIN (?) AS from_page", startFromRowSubQuery).
		Where(strings.Join(conditions, " OR "))
	return query
}
