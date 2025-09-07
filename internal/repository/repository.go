package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"oracle-golang/internal/model/request"
	"strconv"
	"strings"
	"time"

	go_ora "github.com/sijms/go-ora/v2"
)

type OracleRepository struct {
	db *sql.DB
}

func NewOracleRepository(db *sql.DB) *OracleRepository {
	return &OracleRepository{db: db}
}

func (r *OracleRepository) CallProcedure(ctx context.Context, name string, params []request.ProcedureParam) (map[string]any, error) {
	log.Printf("Calling procedure: %s with %d parameters", name, len(params))
	for i, p := range params {
		log.Printf("  Param[%d]: name=%s, type=%s, direction=%s, value=%v", i, p.Name, p.Type, p.Direction, p.Value)
	}

	// Prepare named arguments for go-ora
	args := make([]interface{}, 0, len(params))
	outputParams := make(map[string]interface{}) // Store output parameter destinations

	for _, p := range params {
		switch strings.ToUpper(p.Direction) {
		case "IN":
			// For input parameters, use sql.Named with converted value
			args = append(args, sql.Named(p.Name, r.convertInputValue(p)))
		case "OUT":
			// For output parameters, use go_ora.Out with appropriate destination
			if strings.ToUpper(p.Type) == "REF CURSOR" || strings.ToUpper(p.Type) == "SYS_REFCURSOR" {
				// REF CURSOR requires special handling
				var cursor go_ora.RefCursor
				args = append(args, sql.Named(p.Name, go_ora.Out{Dest: &cursor}))
				outputParams[p.Name] = &cursor
			} else {
				outParam := r.createOutputParameter(p)
				args = append(args, sql.Named(p.Name, outParam))
				outputParams[p.Name] = outParam.Dest
			}
		case "INOUT":
			// For INOUT parameters, we need to handle both input value and output destination
			// This requires special handling since go_ora.Out.In is just a boolean flag
			inputValue := r.convertInputValue(p)
			if strings.ToUpper(p.Type) == "REF CURSOR" || strings.ToUpper(p.Type) == "SYS_REFCURSOR" {
				return nil, fmt.Errorf("REF CURSOR cannot be used as INOUT parameter")
			} else {
				outParam := r.createOutputParameter(p)
				// For INOUT, we need to pass the input value separately
				// This is a workaround since go_ora.Out doesn't support input values directly
				args = append(args, sql.Named(p.Name, struct {
					In  interface{}
					Out go_ora.Out
				}{
					In:  inputValue,
					Out: outParam,
				}))
				outputParams[p.Name] = outParam.Dest
			}
		default:
			return nil, fmt.Errorf("unsupported parameter direction: %s", p.Direction)
		}
	}

	// Construct the PL/SQL block with named parameters
	query := fmt.Sprintf("BEGIN %s(", name)
	for i, p := range params {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf(":%s", p.Name)
	}
	query += "); END;"

	log.Printf("Generated SQL: %s", query)

	// Execute the procedure
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("execution failed for procedure '%s': %w", name, err)
	}

	// Process output parameters
	return r.processOutputParameters(params, outputParams)
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
	case "NUMBER", "INTEGER", "INT", "FLOAT", "DOUBLE":
		switch v := p.Value.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case string:
			// Try to parse string to float
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
			return v
		default:
			return v
		}
	case "VARCHAR2", "VARCHAR", "CHAR", "CLOB", "NVARCHAR2", "NCHAR", "NCLOB":
		return fmt.Sprintf("%v", p.Value)
	case "DATE", "TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITH LOCAL TIME ZONE":
		// Handle date/time conversion
		switch v := p.Value.(type) {
		case string:
			// Try to parse as time
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				return t
			}
			return v
		case time.Time:
			return v
		default:
			return v
		}
	case "BOOLEAN":
		switch v := p.Value.(type) {
		case bool:
			return v
		case string:
			return strings.ToLower(v) == "true" || v == "1"
		case int, int64, float64:
			return v != 0
		default:
			return false
		}
	case "RAW", "BLOB":
		switch v := p.Value.(type) {
		case []byte:
			return v
		case string:
			return []byte(v)
		default:
			return v
		}
	default:
		return p.Value
	}
}

// createOutputParameter creates the appropriate output parameter based on type
func (r *OracleRepository) createOutputParameter(p request.ProcedureParam) go_ora.Out {
	switch strings.ToUpper(p.Type) {
	case "NUMBER", "INTEGER", "INT", "FLOAT", "DOUBLE":
		var out sql.NullFloat64
		return go_ora.Out{Dest: &out}
	case "VARCHAR2", "VARCHAR", "CHAR", "CLOB", "NVARCHAR2", "NCHAR", "NCLOB":
		var out sql.NullString
		// For strings, specify size to avoid ORA-06502 errors
		return go_ora.Out{Dest: &out, Size: 4000}
	case "DATE", "TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITH LOCAL TIME ZONE":
		var out sql.NullTime
		return go_ora.Out{Dest: &out}
	case "BOOLEAN":
		var out bool
		return go_ora.Out{Dest: &out}
	case "RAW", "BLOB":
		var out []byte
		return go_ora.Out{Dest: &out, Size: 4000}
	default:
		var out any
		return go_ora.Out{Dest: &out}
	}
}

// processOutputParameters processes the output parameters and returns the result
func (r *OracleRepository) processOutputParameters(params []request.ProcedureParam, outputParams map[string]interface{}) (map[string]any, error) {
	result := make(map[string]any)

	for _, p := range params {
		if strings.ToUpper(p.Direction) == "IN" {
			continue // Skip input-only parameters
		}

		dest := outputParams[p.Name]
		if dest == nil {
			continue
		}

		// Handle REF CURSOR parameters
		if strings.ToUpper(p.Type) == "REF CURSOR" || strings.ToUpper(p.Type) == "SYS_REFCURSOR" {
			if cursorPtr, ok := dest.(*go_ora.RefCursor); ok && cursorPtr != nil {
				if cursorPtr != nil {
					// Convert RefCursor to sql.Rows
					rows, err := go_ora.WrapRefCursor(context.Background(), r.db, cursorPtr)
					if err != nil {
						return nil, fmt.Errorf("failed to wrap REF CURSOR for parameter %s: %w", p.Name, err)
					}
					if rows != nil {
						rowsData, err := r.processRowsResult(rows)
						if err != nil {
							return nil, fmt.Errorf("failed to process REF CURSOR for parameter %s: %w", p.Name, err)
						}
						result[p.Name] = rowsData
					} else {
						result[p.Name] = nil
					}
				} else {
					result[p.Name] = nil
				}
			}
			continue
		}

		// Handle regular output parameters
		switch dest := dest.(type) {
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
		case *bool:
			result[p.Name] = *dest
		case *[]byte:
			result[p.Name] = *dest
		case *any:
			result[p.Name] = *dest
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
		// Create slice of interface{} to hold column values
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		rowMap := make(map[string]any)
		for i, colName := range cols {
			// Handle different types appropriately
			switch v := columns[i].(type) {
			case []byte:
				// Convert byte arrays to strings for better JSON serialization
				rowMap[colName] = string(v)
			case time.Time:
				// Format time for better JSON serialization
				rowMap[colName] = v.Format(time.RFC3339)
			default:
				rowMap[colName] = v
			}
		}
		allRows = append(allRows, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return allRows, nil
}
