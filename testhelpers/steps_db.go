// +build !prod

package testhelpers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cucumber/messages-go/v10"

	"github.com/France-ioi/AlgoreaBackend/app/database"
)

type rowTransformation int

const (
	unchanged rowTransformation = iota + 1
	changed
	deleted
)

func (ctx *TestContext) DBHasTable(tableName string, data *messages.PickleStepArgument_PickleTable) error { // nolint
	db := ctx.db()

	if len(data.Rows) > 1 {
		head := data.Rows[0].Cells
		fields := make([]string, 0, len(head))
		marks := make([]string, 0, len(head))
		for _, cell := range head {
			fields = append(fields, database.QuoteName(cell.Value))
			marks = append(marks, "?")
		}

		marksString := "(" + strings.Join(marks, ", ") + ")"
		finalMarksString := marksString
		if len(data.Rows) > 2 {
			finalMarksString = strings.Repeat(marksString+", ", len(data.Rows)-2) + finalMarksString
		}
		query := "INSERT INTO " + database.QuoteName(tableName) + // nolint: gosec
			" (" + strings.Join(fields, ", ") + ") VALUES " + finalMarksString
		vals := make([]interface{}, 0, (len(data.Rows)-1)*len(head))
		for i := 1; i < len(data.Rows); i++ {
			for _, cell := range data.Rows[i].Cells {
				var err error
				if cell.Value, err = ctx.preprocessString(cell.Value); err != nil {
					return err
				}
				vals = append(vals, dbDataTableValue(cell.Value))
			}
		}
		if ctx.inScenario {
			tx, err := db.Begin()
			if err != nil {
				return err
			}
			_, err = tx.Exec("SET FOREIGN_KEY_CHECKS=0")
			if err != nil {
				_ = tx.Rollback()
				return err
			}
			_, err = tx.Exec(query, vals...)
			if err != nil {
				_ = tx.Rollback()
				return err
			}
			_, err = tx.Exec("SET FOREIGN_KEY_CHECKS=1")
			if err != nil {
				_ = tx.Rollback()
				return err
			}
			err = tx.Commit()
			if err != nil {
				return err
			}
		} else {
			ctx.featureQueries = append(ctx.featureQueries, dbquery{query, vals})
		}
	}

	if ctx.dbTableData[tableName] == nil {
		ctx.dbTableData[tableName] = data
	} else if len(data.Rows) > 1 {
		ctx.dbTableData[tableName] = combinePickleTables(ctx.dbTableData[tableName], data)
	}

	return nil
}

func (ctx *TestContext) DBHasUsers(data *messages.PickleStepArgument_PickleTable) error { // nolint
	if len(data.Rows) > 1 {
		groupsToCreate := &messages.PickleStepArgument_PickleTable{
			Rows: make([]*messages.PickleStepArgument_PickleTable_PickleTableRow, 1, (len(data.Rows)-1)*2+1),
		}
		groupsToCreate.Rows[0] = &messages.PickleStepArgument_PickleTable_PickleTableRow{
			Cells: []*messages.PickleStepArgument_PickleTable_PickleTableRow_PickleTableCell{
				{Value: "id"}, {Value: "name"}, {Value: "description"}, {Value: "type"},
			},
		}
		head := data.Rows[0].Cells
		groupIDColumnNumber := -1
		loginColumnNumber := -1
		for number, cell := range head {
			if cell.Value == "group_id" {
				groupIDColumnNumber = number
				continue
			}
			if cell.Value == "login" {
				loginColumnNumber = number
				continue
			}
		}

		for i := 1; i < len(data.Rows); i++ {
			login := "null"
			if loginColumnNumber != -1 {
				login = data.Rows[i].Cells[loginColumnNumber].Value
			}

			if groupIDColumnNumber != -1 {
				groupsToCreate.Rows = append(groupsToCreate.Rows, &messages.PickleStepArgument_PickleTable_PickleTableRow{
					Cells: []*messages.PickleStepArgument_PickleTable_PickleTableRow_PickleTableCell{
						{Value: data.Rows[i].Cells[groupIDColumnNumber].Value}, {Value: login}, {Value: login}, {Value: "User"},
					},
				})
			}
		}

		if err := ctx.DBHasTable("groups", groupsToCreate); err != nil {
			return err
		}
	}

	return ctx.DBHasTable("users", data)
}

func (ctx *TestContext) DBGroupsAncestorsAreComputed() error { // nolint
	gormDB, err := database.Open(ctx.db())
	if err != nil {
		return err
	}

	err = database.NewDataStore(gormDB).InTransaction(func(store *database.DataStore) error {
		return store.GroupGroups().After()
	})
	if err != nil {
		return err
	}

	ctx.dbTableData["groups_ancestors"] = &messages.PickleStepArgument_PickleTable{
		Rows: []*messages.PickleStepArgument_PickleTable_PickleTableRow{
			{Cells: []*messages.PickleStepArgument_PickleTable_PickleTableRow_PickleTableCell{
				{Value: "ancestor_group_id"}, {Value: "child_group_id"}, {Value: "expires_at"},
			}},
		},
	}

	var groupsAncestors []map[string]interface{}
	err = gormDB.Table("groups_ancestors").Select("ancestor_group_id, child_group_id, expires_at").
		Order("ancestor_group_id, child_group_id, expires_at").ScanIntoSliceOfMaps(&groupsAncestors).Error()

	if err != nil {
		return err
	}

	for _, row := range groupsAncestors {
		ctx.dbTableData["groups_ancestors"].Rows = append(ctx.dbTableData["groups_ancestors"].Rows,
			&messages.PickleStepArgument_PickleTable_PickleTableRow{
				Cells: []*messages.PickleStepArgument_PickleTable_PickleTableRow_PickleTableCell{
					{Value: row["ancestor_group_id"].(string)}, {Value: row["child_group_id"].(string)}, {Value: row["expires_at"].(string)},
				},
			})
	}
	return nil
}

func (ctx *TestContext) TableShouldBeEmpty(tableName string) error { // nolint
	db := ctx.db()
	sqlRows, err := db.Query(fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", tableName)) //nolint:gosec
	if err != nil {
		return err
	}
	defer func() { _ = sqlRows.Close() }()
	if sqlRows.Next() {
		return fmt.Errorf("the table %q should be empty, but it is not", tableName)
	}

	return nil
}

func (ctx *TestContext) TableAtColumnValueShouldBeEmpty(tableName string, columnName, columnValuesStr string) error { // nolint
	columnValues := parseMultipleValuesString(columnValuesStr)

	db := ctx.db()
	where, parameters := constructWhereForColumnValues([]string{columnName}, columnValues, true)
	sqlRows, err := db.Query(fmt.Sprintf("SELECT 1 FROM %s %s LIMIT 1", tableName, where), parameters...) //nolint:gosec
	if err != nil {
		return err
	}
	defer func() { _ = sqlRows.Close() }()
	if sqlRows.Next() {
		return fmt.Errorf("the table %q should be empty, but it is not", tableName)
	}

	return nil
}

func (ctx *TestContext) TableShouldBe(tableName string, data *messages.PickleStepArgument_PickleTable) error { // nolint
	return ctx.tableAtColumnValueShouldBe(tableName, []string{""}, nil, unchanged, data)
}

func (ctx *TestContext) TableShouldStayUnchanged(tableName string) error { // nolint
	data := ctx.dbTableData[tableName]
	if data == nil {
		data = &messages.PickleStepArgument_PickleTable{Rows: []*messages.PickleStepArgument_PickleTable_PickleTableRow{
			{Cells: []*messages.PickleStepArgument_PickleTable_PickleTableRow_PickleTableCell{{Value: "1"}}}},
		}
	}
	return ctx.tableAtColumnValueShouldBe(tableName, []string{""}, nil, unchanged, data)
}

func (ctx *TestContext) TableShouldStayUnchangedButTheRowWithColumnValue(tableName, columnName, columnValues string) error { // nolint
	data := ctx.dbTableData[tableName]
	if data == nil {
		data = &messages.PickleStepArgument_PickleTable{Rows: []*messages.PickleStepArgument_PickleTable_PickleTableRow{}}
	}
	return ctx.tableAtColumnValueShouldBe(tableName, []string{columnName}, parseMultipleValuesString(columnValues),
		changed, data)
}

func (ctx *TestContext) TableShouldStayUnchangedButTheRowsWithColumnValueShouldBeDeleted(tableName, columnNames,
	columnValues string) error {
	data := ctx.dbTableData[tableName]
	if data == nil {
		data = &messages.PickleStepArgument_PickleTable{Rows: []*messages.PickleStepArgument_PickleTable_PickleTableRow{}}
	}

	return ctx.tableAtColumnValueShouldBe(tableName, parseMultipleValuesString(columnNames),
		parseMultipleValuesString(columnValues), deleted, data)
}

func (ctx *TestContext) TableAtColumnValueShouldBe(tableName, columnName, columnValues string, data *messages.PickleStepArgument_PickleTable) error { // nolint
	return ctx.tableAtColumnValueShouldBe(tableName, []string{columnName}, parseMultipleValuesString(columnValues),
		unchanged, data)
}

func (ctx *TestContext) TableShouldNotContainColumnValue(tableName, columnName, columnValues string) error { // nolint
	return ctx.tableAtColumnValueShouldBe(tableName, []string{columnName}, parseMultipleValuesString(columnValues), unchanged,
		&messages.PickleStepArgument_PickleTable{

			Rows: []*messages.PickleStepArgument_PickleTable_PickleTableRow{
				{Cells: []*messages.PickleStepArgument_PickleTable_PickleTableRow_PickleTableCell{{Value: columnName}}}},
		})
}

func combinePickleTables(table1, table2 *messages.PickleStepArgument_PickleTable) *messages.PickleStepArgument_PickleTable {
	table1FieldMap := map[string]int{}
	combinedFieldMap := map[string]bool{}
	columnNumber := len(table1.Rows[0].Cells)
	combinedColumnNames := make([]string, 0, columnNumber+len(table2.Rows[0].Cells))
	for index, cell := range table1.Rows[0].Cells {
		table1FieldMap[cell.Value] = index
		combinedFieldMap[cell.Value] = true
		combinedColumnNames = append(combinedColumnNames, cell.Value)
	}
	table2FieldMap := map[string]int{}
	for index, cell := range table2.Rows[0].Cells {
		table2FieldMap[cell.Value] = index
		// only add a column if it hasn't been met in table1
		if !combinedFieldMap[cell.Value] {
			combinedFieldMap[cell.Value] = true
			columnNumber++
			combinedColumnNames = append(combinedColumnNames, cell.Value)
		}
	}

	combinedTable := &messages.PickleStepArgument_PickleTable{}
	combinedTable.Rows = make([]*messages.PickleStepArgument_PickleTable_PickleTableRow, 0, len(table1.Rows)+len(table2.Rows)-1)

	header := &messages.PickleStepArgument_PickleTable_PickleTableRow{
		Cells: make([]*messages.PickleStepArgument_PickleTable_PickleTableRow_PickleTableCell, 0, columnNumber),
	}
	for _, columnName := range combinedColumnNames {
		header.Cells = append(header.Cells, &messages.PickleStepArgument_PickleTable_PickleTableRow_PickleTableCell{Value: columnName})
	}
	combinedTable.Rows = append(combinedTable.Rows, header)

	copyCellsIntoCombinedTable(table1, combinedColumnNames, table1FieldMap, combinedTable)
	copyCellsIntoCombinedTable(table2, combinedColumnNames, table2FieldMap, combinedTable)
	return combinedTable
}

func copyCellsIntoCombinedTable(sourceTable *messages.PickleStepArgument_PickleTable, combinedColumnNames []string,
	sourceTableFieldMap map[string]int, combinedTable *messages.PickleStepArgument_PickleTable) {
	for rowNum := 1; rowNum < len(sourceTable.Rows); rowNum++ {
		newRow := &messages.PickleStepArgument_PickleTable_PickleTableRow{
			Cells: make([]*messages.PickleStepArgument_PickleTable_PickleTableRow_PickleTableCell, 0, len(combinedColumnNames)),
		}
		for _, column := range combinedColumnNames {
			var newCell *messages.PickleStepArgument_PickleTable_PickleTableRow_PickleTableCell
			if sourceColumnNumber, ok := sourceTableFieldMap[column]; ok {
				newCell = sourceTable.Rows[rowNum].Cells[sourceColumnNumber]
			}
			newRow.Cells = append(newRow.Cells, newCell)
		}
		combinedTable.Rows = append(combinedTable.Rows, newRow)
	}
}

func parseMultipleValuesString(valuesString string) []string {
	return strings.Split(valuesString, ",")
}

var columnNameRegexp = regexp.MustCompile(`^[a-zA-Z]\w*$`)

func (ctx *TestContext) tableAtColumnValueShouldBe(tableName string, columnNames, columnValues []string,
	rowTransformation rowTransformation, data *messages.PickleStepArgument_PickleTable) error { // nolint
	// For that, we build a SQL request with only the attributes we are interested about (those
	// for the test data table) and we convert them to string (in SQL) to compare to table value.
	// Expect 'null' string in the table to check for nullness

	db := ctx.db()

	var selects []string
	head := data.Rows[0].Cells
	for _, cell := range head {
		dataTableColumnName := cell.Value
		if columnNameRegexp.MatchString(dataTableColumnName) {
			dataTableColumnName = database.QuoteName(dataTableColumnName)
		}
		selects = append(selects, dataTableColumnName)
	}
	selectsJoined := strings.Join(selects, ", ")

	if rowTransformation == deleted {
		// check that the rows are not present anymore
		where, parameters := constructWhereForColumnValues(columnNames, columnValues, true)

		// exec sql
		var nbRows int
		selectValuesInQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s` %s", tableName, where) // nolint: gosec
		err := db.QueryRow(selectValuesInQuery, parameters...).Scan(&nbRows)
		if err != nil {
			return err
		}

		if nbRows > 0 {
			return fmt.Errorf("found %d rows that should have been deleted", nbRows)
		}
	}

	// request for "unchanged": WHERE IN...
	// request for "changed" or "deleted": WHERE NOT IN...
	whereIn := rowTransformation == unchanged
	where, parameters := constructWhereForColumnValues(columnNames, columnValues, whereIn)

	// exec sql
	query := fmt.Sprintf("SELECT %s FROM `%s` %s ORDER BY %s", selectsJoined, tableName, where, selectsJoined) // nolint: gosec
	sqlRows, err := db.Query(query, parameters...)
	if err != nil {
		return err
	}
	defer func() { _ = sqlRows.Close() }()

	headerColumns := data.Rows[0].Cells
	columnIndexes := make([]int, len(columnNames))
	for i := range columnIndexes {
		columnIndexes[i] = -1
	}
	for headerColumnIndex, headerColumn := range headerColumns {
		for columnIndex, columnName := range columnNames {
			if headerColumn.Value == columnName {
				columnIndexes[columnIndex] = headerColumnIndex
				break
			}
		}
	}

	iDataRow := 1
	sqlCols, _ := sqlRows.Columns() // nolint: gosec
	for sqlRows.Next() {
		for rowTransformation != unchanged && iDataRow < len(data.Rows) && rowMatchesColumnValues(data.Rows[iDataRow], columnIndexes, columnValues) {
			iDataRow++
		}
		if iDataRow >= len(data.Rows) {
			return fmt.Errorf("there are more rows in the SQL results than expected. expected: %d", len(data.Rows)-1)
		}

		// Create a slice of string to represent each attribute value,
		// and a second slice to contain pointers to each item.
		rowValues := make([]*string, len(sqlCols))
		rowValPtr := make([]interface{}, len(sqlCols))
		for i := range rowValues {
			rowValPtr[i] = &rowValues[i]
		}
		// Scan the result into the column pointers...
		if err := sqlRows.Scan(rowValPtr...); err != nil {
			return err
		}

		nullValue := tableValueNull
		pNullValue := &nullValue
		// checking that all columns of the test data table match the SQL row
		for iCol, dataCell := range data.Rows[iDataRow].Cells {
			if dataCell == nil {
				continue
			}
			colName := headerColumns[iCol].Value
			dataValue, err := ctx.preprocessString(dataCell.Value)
			if err != nil {
				return err
			}
			sqlValue := rowValPtr[iCol].(**string)

			if *sqlValue == nil {
				sqlValue = &pNullValue
			}

			if (dataValue == tableValueTrue && **sqlValue == "1") || (dataValue == tableValueFalse && **sqlValue == "0") {
				continue
			}

			if dataValue != **sqlValue {
				return fmt.Errorf("not matching expected value at row %d, col %s, expected '%s', got: '%v'", iDataRow, colName, dataValue, **sqlValue)
			}
		}

		iDataRow++
	}

	for rowTransformation != unchanged && iDataRow < len(data.Rows) && rowMatchesColumnValues(data.Rows[iDataRow], columnIndexes, columnValues) {
		iDataRow++
	}

	// check that there are no rows in the test data table left for checking (this means there are fewer rows in the SQL result)
	if iDataRow < len(data.Rows) {
		return fmt.Errorf("there are fewer rows in the SQL result than expected")
	}
	return nil
}

// rowMatchesColumnValues checks whether a column matches some values at some rows
// we do an OR operation, thus returning if any column is match one of the values
func rowMatchesColumnValues(row *messages.PickleStepArgument_PickleTable_PickleTableRow, columnIndexes []int, columnValues []string) bool {
	for _, columnIndex := range columnIndexes {
		for _, columnValue := range columnValues {
			if row.Cells[columnIndex].Value == columnValue {
				return true
			}
		}

	}

	return false
}

// constructWhereForColumnValues construct the WHERE part of a query matching column with values
// note: the same values are checked for every column
func constructWhereForColumnValues(columnNames, columnValues []string, whereIn bool) (
	where string, parameters []interface{}) {
	if len(columnValues) > 0 {
		questionMarks := "?" + strings.Repeat(", ?", len(columnValues)-1)

		isFirstCondition := true
		for _, columnName := range columnNames {
			if isFirstCondition {
				where += " WHERE "
			} else {
				where += " OR "
			}
			isFirstCondition = false

			if whereIn {
				where += fmt.Sprintf(" %s IN (%s) ", columnName, questionMarks) // #nosec
			} else {
				where += fmt.Sprintf(" %s NOT IN (%s) ", columnName, questionMarks) // #nosec
			}

			for _, columnValue := range columnValues {
				parameters = append(parameters, columnValue)
			}
		}
	}

	return where, parameters
}

func (ctx *TestContext) DbTimeNow(timeStrRaw string) error { // nolint
	MockDBTime(timeStrRaw)
	return nil
}

const tableValueFalse = "false"
const tableValueTrue = "true"
const tableValueNull = "null"

// dbDataTableValue converts a string value that we can find the db seeding table to a valid type for the db
// e.g., the string "null" means the SQL `NULL`
func dbDataTableValue(input string) interface{} {
	switch input {
	case tableValueFalse:
		return false
	case tableValueTrue:
		return true
	case tableValueNull:
		return nil
	default:
		return input
	}
}
