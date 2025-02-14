//go:build !prod

package testhelpers

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages/go/v21"

	"github.com/France-ioi/AlgoreaBackend/v2/app/database"
)

type rowTransformation int

const (
	unchanged rowTransformation = iota + 1
	changed
	deleted
)

// DBHasTable inserts the data from the Godog table into the database table.
func (ctx *TestContext) DBHasTable(tableName string, data *godog.Table) error {
	if len(data.Rows) > 1 {
		referenceColumnIndex := -1
		head := data.Rows[0].Cells
		fields := make([]string, 0, len(head))
		marks := make([]string, 0, len(head))

		for i, cell := range head {
			if cell.Value == "@reference" {
				referenceColumnIndex = i
			}

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
			for j, cell := range data.Rows[i].Cells {
				if j != referenceColumnIndex {
					var err error
					if cell.Value, err = ctx.preprocessString(cell.Value); err != nil {
						return err
					}
				}
				vals = append(vals, dbDataTableValue(cell.Value))
			}
		}
		if err := ctx.executeOrQueueDBDataInsertionQuery(query, vals); err != nil {
			return err
		}
	}

	ctx.initializeOrCombineDBTableData(tableName, data)

	return nil
}

func (ctx *TestContext) initializeOrCombineDBTableData(tableName string, data *godog.Table) {
	if ctx.dbTableData[tableName] == nil {
		ctx.dbTableData[tableName] = data
	} else if len(data.Rows) > 1 {
		ctx.dbTableData[tableName] = combinePickleTables(ctx.dbTableData[tableName], data)
	}
}

func (ctx *TestContext) setDBTableRowColumnValues(tableName string, primaryKey, columnValues map[string]string) {
	if ctx.dbTableData[tableName] == nil {
		panic(fmt.Sprintf("cannot set value: table %q is not initialized", tableName))
	}

	columns := make([]string, 0, len(columnValues))
	values := make([]string, 0, len(columnValues))
	for column, value := range columnValues {
		columns = append(columns, column)
		values = append(values, value)
	}

	columnIndexes := getColumnIndexes(ctx.dbTableData[tableName], columns)
	for i, columnIndex := range columnIndexes {
		if columnIndex == -1 {
			ctx.dbTableData[tableName] = combinePickleTables(
				ctx.dbTableData[tableName],
				&godog.Table{Rows: []*messages.PickleTableRow{{Cells: []*messages.PickleTableCell{{Value: columns[i]}}}}},
			)
			columnIndexes[i] = len(ctx.dbTableData[tableName].Rows[0].Cells) - 1
		}
	}

	rowIndex := ctx.getDBTableRowIndexForPrimaryKey(tableName, primaryKey)
	if rowIndex == -1 {
		panic(fmt.Sprintf("no such row in table %q with primary key %v", tableName, primaryKey))
	}

	row := ctx.dbTableData[tableName].Rows[rowIndex]
	for i, columnIndex := range columnIndexes {
		if row.Cells[columnIndex] == nil {
			row.Cells[columnIndex] = &messages.PickleTableCell{}
		}
		row.Cells[columnIndex].Value = values[i]
	}

	ctx.executeOrQueueDBDataRowUpdate(tableName, primaryKey, columns, values)
}

func (ctx *TestContext) executeOrQueueDBDataRowUpdate(tableName string, primaryKey map[string]string, columns, values []string) {
	queryValues := make([]interface{}, 0, len(values)+len(primaryKey))
	quotedColumns := make([]string, 0, len(columns))
	for i, column := range columns {
		quotedColumns = append(quotedColumns, database.QuoteName(column)+" = ?")
		queryValues = append(queryValues, dbDataTableValue(values[i]))
	}

	primaryKeyColumns := make([]string, 0, len(primaryKey))
	for primaryKeyColumn, primaryKeyValue := range primaryKey {
		primaryKeyColumns = append(primaryKeyColumns, database.QuoteName(primaryKeyColumn))
		queryValues = append(queryValues, dbDataTableValue(primaryKeyValue))
	}

	err := ctx.executeOrQueueDBDataInsertionQuery("UPDATE "+database.QuoteName(tableName)+
		" SET "+strings.Join(quotedColumns, ", ")+" WHERE "+
		strings.Join(primaryKeyColumns, " = ? AND ")+" = ?", queryValues)
	if err != nil {
		panic(err)
	}
}

func (ctx *TestContext) setDBTableRowColumnValue(tableName string, primaryKey map[string]string, column, value string) {
	ctx.setDBTableRowColumnValues(tableName, primaryKey, map[string]string{column: value})
}

func getDBTableColumnIndex(dbTable *godog.Table, columnName string) int {
	for i, cell := range dbTable.Rows[0].Cells {
		if cell.Value == columnName {
			return i
		}
	}

	return -1
}

func isDBTableColumnSetInRow(dbTable *godog.Table, columnName string, rowIndex int) bool {
	columnIndex := getDBTableColumnIndex(dbTable, columnName)
	return columnIndex != -1 && dbTable.Rows[rowIndex].Cells[columnIndex] != nil
}

func (ctx *TestContext) getDBTableRowIndexForPrimaryKey(tableName string, primaryKey map[string]string) int {
	if _, ok := ctx.dbTableData[tableName]; !ok {
		return -1
	}

	primaryKeyIndexes := make(map[string]int, len(primaryKey))
	for primaryKeyColumn := range primaryKey {
		primaryKeyIndexes[primaryKeyColumn] = getDBTableColumnIndex(ctx.dbTableData[tableName], primaryKeyColumn)
		if primaryKeyIndexes[primaryKeyColumn] == -1 {
			return -1
		}
	}

	for i := 1; i < len(ctx.dbTableData[tableName].Rows); i++ {
		row := ctx.dbTableData[tableName].Rows[i]
		match := true
		for primaryKeyColumn, primaryKeyIndex := range primaryKeyIndexes {
			if row.Cells[primaryKeyIndex] == nil || row.Cells[primaryKeyIndex].Value != primaryKey[primaryKeyColumn] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func (ctx *TestContext) executeOrQueueDBDataInsertionQuery(query string, vals []interface{}) error {
	if ctx.inScenario {
		tx, err := ctx.db.Begin()
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
			_, _ = tx.Exec("SET FOREIGN_KEY_CHECKS=1")
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
	return nil
}

func getColumnIndex(table *godog.Table, columnName string) int {
	for i, cell := range table.Rows[0].Cells {
		if cell.Value == columnName {
			return i
		}
	}
	return -1
}

// DBHasUsers inserts the data from the Godog table into the users and groups tables.
func (ctx *TestContext) DBHasUsers(data *godog.Table) error {
	if len(data.Rows) > 1 {
		groupsToCreate := &godog.Table{
			Rows: make([]*messages.PickleTableRow, 1, (len(data.Rows)-1)*2+1),
		}
		groupsToCreate.Rows[0] = &messages.PickleTableRow{
			Cells: []*messages.PickleTableCell{
				{Value: "id"}, {Value: "name"}, {Value: "type"},
			},
		}

		groupIDColumnIndex := getColumnIndex(data, "group_id")
		loginColumnIndex := getColumnIndex(data, "login")

		for i := 1; i < len(data.Rows); i++ {
			var login string
			if loginColumnIndex != -1 {
				login = data.Rows[i].Cells[loginColumnIndex].Value
			}

			if groupIDColumnIndex != -1 &&
				ctx.getDBTableRowIndexForPrimaryKey(
					"groups",
					map[string]string{"id": data.Rows[i].Cells[groupIDColumnIndex].Value},
				) == -1 {
				groupsToCreate.Rows = append(groupsToCreate.Rows, &messages.PickleTableRow{
					Cells: []*messages.PickleTableCell{
						{Value: data.Rows[i].Cells[groupIDColumnIndex].Value}, {Value: login}, {Value: "User"},
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

// loadColumnsFromDBTable loads the content of a table from the database for some columns in order to check if the table
// had changed after some manipulations later.
func (ctx *TestContext) loadColumnsFromDBTable(gormDB *database.DB, dbTableName string, dbColumnNames []string) error {
	headerCells := make([]*messages.PickleTableCell, len(dbColumnNames))
	for i, columnName := range dbColumnNames {
		headerCells[i] = &messages.PickleTableCell{
			Value: columnName,
		}
	}

	ctx.dbTableData[dbTableName] = &godog.Table{
		Rows: []*messages.PickleTableRow{
			{Cells: headerCells},
		},
	}

	var rows []map[string]interface{}
	err := gormDB.Table(dbTableName).Select(strings.Join(dbColumnNames, ",")).
		Order(strings.Join(dbColumnNames, ",")).ScanIntoSliceOfMaps(&rows).Error()
	if err != nil {
		return err
	}

	for _, row := range rows {
		rowCells := make([]*messages.PickleTableCell, len(dbColumnNames))
		for j, columnName := range dbColumnNames {
			rowCells[j] = &messages.PickleTableCell{
				Value: fmt.Sprintf("%v", row[columnName]),
			}
		}

		ctx.dbTableData[dbTableName].Rows = append(ctx.dbTableData[dbTableName].Rows, &messages.PickleTableRow{
			Cells: rowCells,
		})
	}

	return nil
}

// DBGroupsAncestorsAreComputed computes the groups_ancestors table.
func (ctx *TestContext) DBGroupsAncestorsAreComputed() error {
	gormDB, err := database.Open(ctx.db)
	if err != nil {
		return err
	}

	err = database.NewDataStore(gormDB).InTransaction(func(store *database.DataStore) error {
		return store.GroupGroups().CreateNewAncestors()
	})
	if err != nil {
		return err
	}

	err = ctx.loadColumnsFromDBTable(gormDB, "groups_ancestors", []string{
		"ancestor_group_id",
		"child_group_id",
		"expires_at",
	})
	if err != nil {
		return err
	}

	return nil
}

// DBItemsAncestorsAndPermissionsAreComputed computes the items_ancestors and permissions_generated tables.
func (ctx *TestContext) DBItemsAncestorsAndPermissionsAreComputed() error {
	gormDB, err := database.Open(ctx.db)
	if err != nil {
		return err
	}

	err = database.NewDataStore(gormDB).InTransaction(func(store *database.DataStore) error {
		// We can consider keeping foreign_key_checks,
		// but it'll break all tests that didn't define items while having permissions.
		store.Exec("SET FOREIGN_KEY_CHECKS=0")
		defer store.Exec("SET FOREIGN_KEY_CHECKS=1")

		err = store.ItemItems().CreateNewAncestors()
		if err != nil {
			return err
		}
		store.SchedulePermissionsPropagation()
		store.ScheduleResultsPropagation()

		return nil
	})
	if err != nil {
		return err
	}

	err = ctx.loadColumnsFromDBTable(gormDB, "items_ancestors", []string{
		"ancestor_item_id",
		"child_item_id",
	})
	if err != nil {
		return err
	}

	err = ctx.loadColumnsFromDBTable(gormDB, "permissions_generated", []string{
		"group_id",
		"item_id",
		"can_view_generated",
		"can_grant_view_generated",
		"can_watch_generated",
		"can_edit_generated",
		"is_owner_generated",
		"can_view_generated_value",
		"can_grant_view_generated_value",
		"can_watch_generated_value",
		"can_edit_generated_value",
	})
	if err != nil {
		return err
	}

	return nil
}

// TableShouldBeEmpty verifies that the DB table is empty.
func (ctx *TestContext) TableShouldBeEmpty(tableName string) error {
	sqlRows, err := ctx.db.Query(fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", tableName)) //nolint:gosec
	if err != nil {
		return err
	}
	defer func() {
		_ = sqlRows.Close()

		if sqlRows.Err() != nil {
			panic(sqlRows.Err())
		}
	}()
	if sqlRows.Next() {
		return fmt.Errorf("the table %q should be empty, but it is not", tableName)
	}

	return nil
}

// TableAtColumnValueShouldBeEmpty verifies that the DB table does not contain rows having the provided values
// in the specified column.
func (ctx *TestContext) TableAtColumnValueShouldBeEmpty(tableName, columnName, valuesStr string) error {
	values := parseMultipleValuesString(valuesStr)

	where, parameters := constructWhereForColumnValues([]string{columnName}, values, true)
	sqlRows, err := ctx.db.Query(fmt.Sprintf("SELECT 1 FROM %s %s LIMIT 1", tableName, where), parameters...) //nolint:gosec
	if err != nil {
		return err
	}
	defer func() {
		_ = sqlRows.Close()

		if sqlRows.Err() != nil {
			panic(sqlRows.Err())
		}
	}()
	if sqlRows.Next() {
		return fmt.Errorf("the table %q should be empty, but it is not", tableName)
	}

	return nil
}

// TableShouldBe verifies that the DB table matches the provided data.
func (ctx *TestContext) TableShouldBe(tableName string, data *godog.Table) error {
	return ctx.tableAtColumnValueShouldBe(tableName, []string{""}, nil, unchanged, data)
}

// TableShouldRemainUnchanged checks that the DB table has not changed.
func (ctx *TestContext) TableShouldRemainUnchanged(tableName string) error {
	data := ctx.dbTableData[tableName]
	if data == nil {
		data = &godog.Table{
			Rows: []*messages.PickleTableRow{
				{Cells: []*messages.PickleTableCell{{Value: "1"}}},
			},
		}
	}
	return ctx.tableAtColumnValueShouldBe(tableName, []string{""}, nil, unchanged, data)
}

// TableShouldStayUnchangedRegardlessOfTheRowsWithColumnValue checks that the DB table has not changed except for rows
// with the specified values in the specified column.
func (ctx *TestContext) TableShouldStayUnchangedRegardlessOfTheRowsWithColumnValue(tableName, columnName, columnValues string) error {
	data := ctx.dbTableData[tableName]
	if data == nil {
		data = &godog.Table{Rows: []*messages.PickleTableRow{}}
	}
	return ctx.tableAtColumnValueShouldBe(tableName, []string{columnName}, parseMultipleValuesString(columnValues), changed, data)
}

// TableShouldRemainUnchangedExceptThatTheRowsWithColumnValueShouldBeDeleted checks for row deletion.
func (ctx *TestContext) TableShouldRemainUnchangedExceptThatTheRowsWithColumnValueShouldBeDeleted(
	tableName, columnNames, columnValues string,
) error {
	data := ctx.dbTableData[tableName]
	if data == nil {
		data = &godog.Table{Rows: []*messages.PickleTableRow{}}
	}

	return ctx.tableAtColumnValueShouldBe(
		tableName, parseMultipleValuesString(columnNames), parseMultipleValuesString(columnValues), deleted, data)
}

// TableAtColumnValueShouldBe verifies that the rows of the DB table having the provided values in the specified column
// match the provided data.
func (ctx *TestContext) TableAtColumnValueShouldBe(
	tableName, columnName, columnValues string, data *godog.Table,
) error {
	return ctx.tableAtColumnValueShouldBe(
		tableName,
		[]string{columnName},
		parseMultipleValuesString(ctx.replaceReferencesWithIDs(columnValues)),
		unchanged,
		data,
	)
}

// TableShouldNotContainColumnValue verifies that the DB table does not contain rows having the provided values
// in the specified column.
func (ctx *TestContext) TableShouldNotContainColumnValue(
	tableName, columnName, columnValues string,
) error {
	return ctx.tableAtColumnValueShouldBe(
		tableName, []string{columnName}, parseMultipleValuesString(ctx.replaceReferencesWithIDs(columnValues)), unchanged,
		&godog.Table{
			Rows: []*messages.PickleTableRow{
				{Cells: []*messages.PickleTableCell{{Value: columnName}}},
			},
		})
}

func combinePickleTables(table1, table2 *godog.Table) *godog.Table {
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

	combinedTable := &godog.Table{}
	combinedTable.Rows = make([]*messages.PickleTableRow, 0, len(table1.Rows)+len(table2.Rows)-1)

	header := &messages.PickleTableRow{
		Cells: make([]*messages.PickleTableCell, 0, columnNumber),
	}
	for _, columnName := range combinedColumnNames {
		header.Cells = append(header.Cells, &messages.PickleTableCell{Value: columnName})
	}
	combinedTable.Rows = append(combinedTable.Rows, header)

	copyCellsIntoCombinedTable(table1, combinedColumnNames, table1FieldMap, combinedTable)
	copyCellsIntoCombinedTable(table2, combinedColumnNames, table2FieldMap, combinedTable)
	return combinedTable
}

func copyCellsIntoCombinedTable(sourceTable *godog.Table, combinedColumnNames []string,
	sourceTableFieldMap map[string]int, combinedTable *godog.Table,
) {
	for rowNum := 1; rowNum < len(sourceTable.Rows); rowNum++ {
		newRow := &messages.PickleTableRow{
			Cells: make([]*messages.PickleTableCell, 0, len(combinedColumnNames)),
		}
		for _, columnName := range combinedColumnNames {
			var newCell *messages.PickleTableCell
			if sourceColumnNumber, ok := sourceTableFieldMap[columnName]; ok {
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

var columnRegexp = regexp.MustCompile(`^[a-zA-Z]\w*$`)

func (ctx *TestContext) tableAtColumnValueShouldBe(tableName string, columnNames, columnValues []string,
	rowTransformation rowTransformation, data *godog.Table,
) error {
	// For that, we build a SQL request with only the attributes we are interested about (those
	// for the test data table) and we convert them to string (in SQL) to compare to table value.
	// Expect 'null' string in the table to check for nullness

	if rowTransformation == deleted {
		nbRows, err := ctx.getNbRowsMatching(tableName, columnNames, columnValues)
		if err != nil {
			return err
		}
		if nbRows > 0 {
			return fmt.Errorf("found %d rows that should have been deleted", nbRows)
		}
	}

	dbColumnNames := getDBColumnNamesFromDataTable(data)

	// request for "unchanged": WHERE IN...
	// request for "changed" or "deleted": WHERE NOT IN...
	whereIn := rowTransformation == unchanged
	dbResult, closer, err := ctx.queryDBRowsMatching(tableName, dbColumnNames, columnNames, columnValues, whereIn)
	if err != nil {
		return err
	}
	defer closer()

	return ctx.dataTableShouldMatchDBResult(data, dbResult, rowTransformation, dbColumnNames, columnNames, columnValues)
}

// dataTableShouldMatchDBResult checks whether the provided data table matches the database rows result.
func (ctx *TestContext) dataTableShouldMatchDBResult(data *godog.Table, dbResult *sql.Rows,
	rowTransformation rowTransformation, dbColumnNames, columnNames, columnValues []string,
) error {
	iDataRow := 1
	columnIndexes := getColumnIndexes(data, columnNames)
	for dbResult.Next() {
		for shouldSkipRow(data, iDataRow, columnIndexes, columnValues, rowTransformation) {
			iDataRow++
		}

		// We need pointers to differentiate null columnValues
		dbRow, err := scanDBRow(dbResult, len(dbColumnNames))
		if err != nil {
			return err
		}

		if iDataRow >= len(data.Rows) {
			nextRow := ctx.formatDBRowAsTableRow(dbRow)
			return fmt.Errorf("there are more rows in the SQL results than expected. expected: %d, the next row:\n%s",
				len(data.Rows)-1, nextRow)
		}

		err = ctx.dataRowMatchesDBRow(data.Rows[iDataRow], dbRow, dbColumnNames, iDataRow)
		if err != nil {
			return err
		}

		iDataRow++
	}
	for shouldSkipRow(data, iDataRow, columnIndexes, columnValues, rowTransformation) {
		iDataRow++
	}

	// check that there are no rows in the test data table left for checking (this means there are fewer rows in the SQL result)
	if iDataRow < len(data.Rows) {
		return fmt.Errorf("there are fewer rows in the SQL result than expected")
	}

	return nil
}

func (ctx *TestContext) formatDBRowAsTableRow(dbRow []*string) string {
	dbRowValuesStr := make([]string, len(dbRow))
	for i, value := range dbRow {
		if value == nil {
			dbRowValuesStr[i] = "null"
			continue
		}

		dbRowValuesStr[i] = *value
		dbRowValuesStr[i] = ctx.readableValue(dbRowValuesStr[i])
	}

	return "| " + strings.Join(dbRowValuesStr, " | ") + " |"
}

func (ctx *TestContext) readableValue(value string) string {
	if id, err := strconv.ParseInt(value, 10, 64); err == nil {
		if reference, ok := ctx.idToReferenceMap[id]; ok {
			return reference
		}
		return value
	}

	if value == time.Now().Format(time.DateTime) {
		return "{{currentTimeDB()}}"
	}

	if value == time.Now().Format("2006-01-02 15:04:05.000") {
		return "{{currentTimeDBMs()}}"
	}

	return value
}

// dataRowMatchesSQLRow checks that a data row matches a row from database.
func (ctx *TestContext) dataRowMatchesDBRow(dataRow *messages.PickleTableRow,
	columnValues []*string, tableColumnNames []string, rowIndex int,
) error {
	// checking that all columns of the test data table match the SQL row
	for colIndex, dataCell := range dataRow.Cells {
		if dataCell == nil {
			continue
		}

		dataValue, err := ctx.preprocessString(dataCell.Value)
		if err != nil {
			return err
		}

		columnValue := columnValues[colIndex]
		if columnValue == nil {
			columnValue = pTableValueNull
		}

		if (dataValue == tableValueTrue && *columnValue == "1") || (dataValue == tableValueFalse && *columnValue == "0") {
			continue
		}

		if dataValue != *columnValue {
			return fmt.Errorf("not matching expected value at row %d, col %s, expected '%s', got: '%v'",
				rowIndex, tableColumnNames[colIndex], dataValue, *columnValue)
		}
	}

	return nil
}

// getDBColumnNamesFromDataTable gets the column names from the Godog data table.
func getDBColumnNamesFromDataTable(data *godog.Table) (dbColumnNames []string) {
	// the first row contains the column names
	headerCells := data.Rows[0].Cells
	for _, cell := range headerCells {
		dbColumnName := cell.Value
		if columnRegexp.MatchString(dbColumnName) {
			dbColumnName = database.QuoteName(dbColumnName)
		}

		dbColumnNames = append(dbColumnNames, dbColumnName)
	}

	return dbColumnNames
}

// scanDBRow scans a DB row from a DB result (*sql.Rows) into a slice of string pointers.
func scanDBRow(dbResult *sql.Rows, length int) ([]*string, error) {
	// Create a slice of values and a second slice with pointers to each item.
	rowValues := make([]*string, length)
	rowValPtr := make([]interface{}, length)
	for i := range rowValues {
		rowValPtr[i] = &rowValues[i]
	}

	// Scan the result into the column pointers...
	err := dbResult.Scan(rowValPtr...)

	return rowValues, err
}

// getColumnIndexes gets the indices of the columns referenced by the given names.
func getColumnIndexes(data *godog.Table, columnNames []string) []int {
	// the first row contains the column names
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

	return columnIndexes
}

// queryDBRowsMatching returns the MySQL result (*sql.Rows) with DB rows that match (if whereIn) or not (if !whereIn)
// one of filterColumnNames at any filterColumnValues.
func (ctx *TestContext) queryDBRowsMatching(tableName string, dbColumnNames, filterColumnNames, filterColumnValues []string, whereIn bool) (
	*sql.Rows, func(), error,
) {
	selectsJoined := strings.Join(dbColumnNames, ", ")

	where, parameters := constructWhereForColumnValues(filterColumnNames, filterColumnValues, whereIn)

	// exec sql
	query := fmt.Sprintf("SELECT %s FROM `%s` %s ORDER BY %s", selectsJoined, tableName, where, selectsJoined) //nolint: gosec
	sqlRows, err := ctx.db.Query(query, parameters...)

	closer := func() { _ = sqlRows.Close() }
	return sqlRows, closer, err
}

// getNbRowsMatching returns how many rows match one of values at any column.
func (ctx *TestContext) getNbRowsMatching(tableName string, columnNames, columnValues []string) (int, error) {
	// check that the rows are not present anymore
	where, parameters := constructWhereForColumnValues(columnNames, columnValues, true)

	// exec sql
	var nbRows int
	selectValuesInQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s` %s", tableName, where) //nolint: gosec
	err := ctx.db.QueryRow(selectValuesInQuery, parameters...).Scan(&nbRows)

	return nbRows, err
}

func shouldSkipRow(data *godog.Table, rowIndex int, columnIndexes []int,
	columnValues []string, rowTransformation rowTransformation,
) bool {
	return rowTransformation != unchanged &&
		rowIndex < len(data.Rows) &&
		rowMatchesColumnValues(data.Rows[rowIndex], columnIndexes, columnValues)
}

// rowMatchesColumnValues checks whether a column matches some values at some rows
// we do an OR operation, thus returning if any column is match one of the values.
func rowMatchesColumnValues(row *messages.PickleTableRow, columnIndexes []int, columnValues []string) bool {
	// Both loops should contain 1 or 2 elements only
	for _, columnIndex := range columnIndexes {
		for _, value := range columnValues {
			if row.Cells[columnIndex].Value == value {
				return true
			}
		}
	}

	return false
}

// constructWhereForColumnValues construct the WHERE part of a query matching column with values
// note: the same values are checked for every column
func constructWhereForColumnValues(columnNames, columnValues []string, whereIn bool) (
	where string, parameters []interface{},
) {
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

// DBTimeNow sets the current time in the database to the provided time.
func (ctx *TestContext) DBTimeNow(timeStrRaw string) error {
	var err error
	timeStrRaw, err = ctx.preprocessString(timeStrRaw)
	if err != nil {
		return err
	}
	MockDBTime(timeStrRaw)
	return nil
}

const (
	tableValueFalse = "false"
	tableValueTrue  = "true"
	tableValueNull  = "null"
)

var (
	tableValueNullVar = tableValueNull
	pTableValueNull   = &tableValueNullVar
)

// dbDataTableValue converts a string value that we can find the db seeding table to a valid type for the db
// e.g., the string "null" means the SQL `NULL`.
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
