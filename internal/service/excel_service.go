package service

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// GetSheets returns a list of sheet names from an Excel file
func GetSheets(path string) ([]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.GetSheetList(), nil
}

// ReadExcelFull reads all rows from the specified sheet
func ReadExcelFull(
	path string,
	sheetName string,
) ([]string, [][]string, error) {

	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	if sheetName == "" {
		sheetName = f.GetSheetName(0)
	}

	rs, err := f.Rows(sheetName)
	if err != nil {
		return nil, nil, err
	}

	var headers []string
	var rows [][]string

	idx := 0

	for rs.Next() {

		cols, _ := rs.Columns()

		if idx == 0 {
			headers = cols
		} else {
			rows = append(rows, cols)
		}

		idx++
	}

	return headers, rows, nil
}

// ReadExcelPreview reads only the first 50 data rows from the specified sheet
func ReadExcelPreview(
	path string,
	sheetName string,
) ([]string, [][]string, error) {

	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	if sheetName == "" {
		sheetName = f.GetSheetName(0)
	}

	rs, err := f.Rows(sheetName)
	if err != nil {
		return nil, nil, err
	}

	var headers []string
	var rows [][]string

	index := 0

	for rs.Next() {

		cols, err := rs.Columns()
		if err != nil {
			continue
		}

		if index == 0 {
			headers = cols
		} else {
			rows = append(rows, cols)
		}

		index++

		// Preview hanya 50 row data (index 1..50)
		if index >= 51 {
			break
		}
	}

	return headers, rows, nil
}

// CreateTable creates a new Oracle table using the selected Excel columns
// selectedColumns is a subset of headers chosen by the user
func CreateTable(
	db *sql.DB,
	tableName string,
	selectedColumns []string,
) error {

	var cols []string

	for _, h := range selectedColumns {

		col := strings.ToUpper(
			strings.ReplaceAll(h, " ", "_"),
		)

		cols = append(
			cols,
			fmt.Sprintf("%s VARCHAR2(4000)", col),
		)
	}

	sqlText := fmt.Sprintf(
		"CREATE TABLE %s (%s)",
		strings.ToUpper(tableName),
		strings.Join(cols, ", "),
	)

	_, err := db.Exec(sqlText)

	return err
}

// InsertRows inserts rows using columnMap: { excelColName -> oracleColName }
// Rows that already exist (either checked via DB or unique constraint violation) are skipped.
func InsertRows(
	db *sql.DB,
	tableName string,
	headers []string,
	rows [][]string,
	columnMap map[string]string,
) (int, int, error) {

	// Build ordered list of (headerIndex, oracleCol) for columns that are mapped
	type colEntry struct {
		headerIdx int
		oracleCol string
	}

	var entries []colEntry

	for i, h := range headers {
		if oracleCol, ok := columnMap[h]; ok && oracleCol != "" {
			entries = append(entries, colEntry{
				headerIdx: i,
				oracleCol: oracleCol,
			})
		}
	}

	if len(entries) == 0 {
		return 0, 0, fmt.Errorf("tidak ada kolom yang dipilih untuk di-insert")
	}

	// Build column list and bind params
	var oracleCols []string
	var binds []string
	var whereClauses []string

	for i, e := range entries {
		oracleCols = append(oracleCols, e.oracleCol)
		binds = append(binds, fmt.Sprintf(":%d", i+1))
		// Check for equality or if both are NULL (Oracle treats "" as NULL)
		// We use distinct bind indices for checkSQL to avoid driver issues with duplicated binds
		whereClauses = append(whereClauses, fmt.Sprintf("(%s = :%d OR (%s IS NULL AND :%d IS NULL))", e.oracleCol, i*2+1, e.oracleCol, i*2+2))
	}

	sqlText := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		strings.ToUpper(tableName),
		strings.Join(oracleCols, ", "),
		strings.Join(binds, ", "),
	)

	checkSQL := fmt.Sprintf(
		"SELECT 1 FROM %s WHERE %s AND ROWNUM <= 1",
		strings.ToUpper(tableName),
		strings.Join(whereClauses, " AND "),
	)

	stmt, err := db.Prepare(sqlText)
	if err != nil {
		return 0, 0, err
	}
	defer stmt.Close()

	checkStmt, err := db.Prepare(checkSQL)
	if err != nil {
		return 0, 0, err
	}
	defer checkStmt.Close()

	inserted := 0
	skipped := 0

	// In-memory duplicate tracking for rows within the same file
	seen := make(map[string]bool)

	for _, row := range rows {

		args := make([]interface{}, len(entries))
		checkArgs := make([]interface{}, len(entries)*2)
		keyParts := make([]string, len(entries))

		for i, e := range entries {
			val := ""
			if e.headerIdx < len(row) {
				val = row[e.headerIdx]
			}
			args[i] = val
			checkArgs[i*2] = val
			checkArgs[i*2+1] = val
			keyParts[i] = val
		}

		key := strings.Join(keyParts, "|#|")
		if seen[key] {
			skipped++
			continue
		}

		// Check if it exists in the database
		var exists int
		err = checkStmt.QueryRow(checkArgs...).Scan(&exists)
		if err == nil && exists == 1 {
			seen[key] = true
			skipped++
			continue
		} else if err != sql.ErrNoRows && err != nil {
			return inserted, skipped, err
		}

		_, err = stmt.Exec(args...)
		if err != nil {
			// ORA-00001 = unique constraint violated → skip
			if strings.Contains(err.Error(), "ORA-00001") ||
				strings.Contains(err.Error(), "unique constraint") {
				skipped++
				seen[key] = true
				continue
			}
			return inserted, skipped, err
		}

		seen[key] = true
		inserted++
	}

	return inserted, skipped, nil
}

// GetOracleTables returns all table names for the current user
func GetOracleTables(
	db *sql.DB,
) ([]string, error) {

	sqlText := `
		SELECT TABLE_NAME
		FROM USER_TABLES
		ORDER BY TABLE_NAME
	`

	rows, err := db.Query(sqlText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string

	for rows.Next() {

		var table string

		rows.Scan(&table)

		tables = append(tables, table)
	}

	return tables, nil
}

// GetOracleColumns returns column names for a given table
func GetOracleColumns(
	db *sql.DB,
	tableName string,
) ([]string, error) {

	sqlText := `
		SELECT COLUMN_NAME
		FROM USER_TAB_COLUMNS
		WHERE TABLE_NAME = :1
		ORDER BY COLUMN_ID
	`

	rows, err := db.Query(
		sqlText,
		strings.ToUpper(tableName),
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string

	for rows.Next() {

		var col string

		rows.Scan(&col)

		cols = append(cols, col)
	}

	return cols, nil
}
