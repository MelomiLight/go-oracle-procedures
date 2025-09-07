package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"oracle-golang/internal/model/request"
	"strings"
)

type OracleRepository struct {
	db *sql.DB
}

func NewOracleRepository(db *sql.DB) *OracleRepository {
	return &OracleRepository{db: db}
}

func (r *OracleRepository) CallProcedure(ctx context.Context, name string, params []request.ProcedureParam) (map[string]any, error) {
	// Log the procedure call for debugging
	log.Printf("Calling procedure: %s with %d parameters", name, len(params))
	for i, p := range params {
		log.Printf("  Param[%d]: name=%s, type=%s, direction=%s, value=%v", i, p.Name, p.Type, p.Direction, p.Value)
	}

	// Prepare placeholders and arguments
	placeholders := make([]string, len(params))
	args := make([]any, len(params))

	for i, p := range params {
		// Use named placeholders instead of positional ones
		ph := fmt.Sprintf(":%s", p.Name)
		placeholders[i] = ph

		switch strings.ToUpper(p.Direction) {
		case "IN":
			args[i] = r.convertInputValue(p)
		case "OUT":
			args[i] = r.createOutputParameter(p)
		case "INOUT":
			args[i] = r.createInOutParameter(p)
		default:
			return nil, fmt.Errorf("unsupported parameter direction: %s", p.Direction)
		}
	}

	// Construct the PL/SQL block
	query := fmt.Sprintf("BEGIN %s(%s); END;", name, strings.Join(placeholders, ", "))
	log.Printf("Generated SQL: %s", query)

	// Prepare and execute
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("prepare failed for query '%s': %w", query, err)
	}
	defer func() {
		if closeErr := stmt.Close(); closeErr != nil {
			log.Printf("Warning: failed to close statement: %v", closeErr)
		}
	}()

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("execution failed for procedure '%s': %w", name, err)
	}

	// Process output parameters
	return r.processOutputParameters(params, args)
}

// GetProcedureInfo retrieves information about a stored procedure from Oracle's data dictionary
func (r *OracleRepository) GetProcedureInfo(ctx context.Context, procedureName string) ([]map[string]any, error) {
	query := `
        SELECT 
            ARGUMENT_NAME,
            DATA_TYPE,
            IN_OUT,
            POSITION,
            DEFAULT_VALUE
        FROM ALL_ARGUMENTS 
        WHERE OBJECT_NAME = UPPER(:1)
        AND OWNER = USER
        ORDER BY POSITION
    `

	rows, err := r.db.QueryContext(ctx, query, procedureName)
	if err != nil {
		return nil, fmt.Errorf("failed to query procedure info: %w", err)
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var argName, dataType, inOut, defaultValue sql.NullString
		var position sql.NullInt64

		err := rows.Scan(&argName, &dataType, &inOut, &position, &defaultValue)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		row := map[string]any{
			"argument_name": argName.String,
			"data_type":     dataType.String,
			"in_out":        inOut.String,
			"position":      position.Int64,
			"default_value": defaultValue.String,
		}
		result = append(result, row)
	}

	return result, nil
}

// convertInputValue converts the input value to the appropriate Go type for Oracle
func (r *OracleRepository) convertInputValue(p request.ProcedureParam) any {
	switch strings.ToUpper(p.Type) {
	case "NUMBER", "INTEGER", "INT":
		// Handle different numeric types
		switch v := p.Value.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case string:
			// Try to parse string numbers
			return v // Let Oracle handle the conversion
		default:
			return v
		}
	case "VARCHAR2", "VARCHAR", "CHAR", "CLOB":
		return fmt.Sprintf("%v", p.Value)
	case "DATE", "TIMESTAMP":
		return p.Value
	default:
		return p.Value
	}
}

// createOutputParameter creates the appropriate output parameter based on type
func (r *OracleRepository) createOutputParameter(p request.ProcedureParam) any {
	switch strings.ToUpper(p.Type) {
	case "REF_CURSOR", "SYS_REFCURSOR":
		var rows *sql.Rows
		return sql.Out{Dest: &rows}
	case "NUMBER", "INTEGER", "INT":
		var out sql.NullFloat64
		return sql.Out{Dest: &out}
	case "VARCHAR2", "VARCHAR", "CHAR", "CLOB":
		var out sql.NullString
		return sql.Out{Dest: &out}
	case "DATE", "TIMESTAMP":
		var out sql.NullTime
		return sql.Out{Dest: &out}
	default:
		var out any
		return sql.Out{Dest: &out}
	}
}

// createInOutParameter creates the appropriate in-out parameter
func (r *OracleRepository) createInOutParameter(p request.ProcedureParam) any {
	switch strings.ToUpper(p.Type) {
	case "NUMBER", "INTEGER", "INT":
		var out sql.NullFloat64
		return sql.Out{Dest: &out, In: true}
	case "VARCHAR2", "VARCHAR", "CHAR", "CLOB":
		var out sql.NullString
		return sql.Out{Dest: &out, In: true}
	case "DATE", "TIMESTAMP":
		var out sql.NullTime
		return sql.Out{Dest: &out, In: true}
	default:
		var out any
		return sql.Out{Dest: &out, In: true}
	}
}

// processOutputParameters processes the output parameters and returns the result
func (r *OracleRepository) processOutputParameters(params []request.ProcedureParam, args []any) (map[string]any, error) {
	result := make(map[string]any)

	for i, p := range params {
		if strings.ToUpper(p.Direction) == "IN" {
			continue // Skip input-only parameters
		}

		out, ok := args[i].(sql.Out)
		if !ok {
			continue
		}

		switch dest := out.Dest.(type) {
		case *sql.NullString:
			if dest.Valid {
				result[p.Name] = dest.String
			} else {
				result[p.Name] = nil
			}
		case *sql.NullFloat64:
			if dest.Valid {
				result[p.Name] = dest.Float64
			} else {
				result[p.Name] = nil
			}
		case *sql.NullTime:
			if dest.Valid {
				result[p.Name] = dest.Time
			} else {
				result[p.Name] = nil
			}
		case *any:
			result[p.Name] = *dest
		case **sql.Rows:
			rows := *dest
			if rows == nil {
				result[p.Name] = nil
				continue
			}

			rowsData, err := r.processRowsResult(rows)
			if err != nil {
				return nil, fmt.Errorf("failed to process rows for parameter %s: %w", p.Name, err)
			}
			result[p.Name] = rowsData
		}
	}

	return result, nil
}

// processRowsResult processes cursor results
func (r *OracleRepository) processRowsResult(rows *sql.Rows) ([]map[string]any, error) {
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Printf("Warning: failed to close rows: %v", closeErr)
		}
	}()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var allRows []map[string]any
	for rows.Next() {
		columns := make([]any, len(cols))
		columnPointers := make([]any, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		rowMap := make(map[string]any)
		for i, colName := range cols {
			rowMap[colName] = columns[i]
		}
		allRows = append(allRows, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return allRows, nil
}
