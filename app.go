package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	_ "modernc.org/sqlite" // Pure Go SQLite for easy cross-compilation
)

// App struct
type App struct {
	ctx context.Context
	db  *sql.DB
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// Shutdown is called at termination
func (a *App) Shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
}

// OpenDatabase prompts the user to select a SQLite file
func (a *App) OpenDatabase() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select SQLite Database",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite Files", Pattern: "*.db;*.sqlite;*.sqlite3"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})

	if err != nil {
		return "", err
	}

	if selection == "" {
		return "", nil // User cancelled
	}

	// Close previous connection if exists
	if a.db != nil {
		a.db.Close()
	}

	db, err := sql.Open("sqlite", selection)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}

	a.db = db
	return selection, nil
}

type QueryResult struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"` // flexible row structure
	Error   string           `json:"error,omitempty"`
}

// ExecuteQuery runs a SQL query and returns the results
func (a *App) ExecuteQuery(query string) QueryResult {
	if a.db == nil {
		return QueryResult{Error: "No database loaded"}
	}

	rows, err := a.db.Query(query)
	if err != nil {
		return QueryResult{Error: err.Error()}
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return QueryResult{Error: err.Error()}
	}

	var results []map[string]any

	for rows.Next() {
		// Dynamic scanning for unknown columns
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return QueryResult{Error: err.Error()}
		}

		entry := make(map[string]any)
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		results = append(results, entry)
	}

	return QueryResult{
		Columns: columns,
		Rows:    results,
	}
}

// StreamQuery executes a query and streams results back via events to avoid freezing the UI
func (a *App) StreamQuery(query string) {
	fmt.Printf("[Go] StreamQuery called with: %s\n", query)
	go func() {
		if a.db == nil {
			fmt.Println("[Go] Error: No database loaded")
			runtime.EventsEmit(a.ctx, "query_error", "No database loaded")
			return
		}

		rows, err := a.db.Query(query)
		if err != nil {
			fmt.Printf("[Go] Query Error: %v\n", err)
			runtime.EventsEmit(a.ctx, "query_error", err.Error())
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			fmt.Printf("[Go] Columns Error: %v\n", err)
			runtime.EventsEmit(a.ctx, "query_error", err.Error())
			return
		}

		fmt.Printf("[Go] Emitting columns: %v\n", columns)
		// Emit columns first
		runtime.EventsEmit(a.ctx, "query_columns", columns)

		chunkSize := 500
		var chunk []map[string]any
		count := 0

		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				fmt.Printf("[Go] Scan Error: %v\n", err)
				continue
			}

			entry := make(map[string]any)
			for i, col := range columns {
				var v interface{}
				val := values[i]
				b, ok := val.([]byte)
				if ok {
					v = string(b)
				} else {
					v = val
				}
				entry[col] = v
			}
			chunk = append(chunk, entry)
			count++

			if len(chunk) >= chunkSize {
				runtime.EventsEmit(a.ctx, "query_rows", chunk)
				chunk = make([]map[string]any, 0, chunkSize)
			}
		}

		// Emit remaining
		if len(chunk) > 0 {
			runtime.EventsEmit(a.ctx, "query_rows", chunk)
		}

		fmt.Printf("[Go] Query done. Processed %d rows.\n", count)
		runtime.EventsEmit(a.ctx, "query_done", true)
	}()
}
