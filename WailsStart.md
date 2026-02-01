# WailsStart - The Wails Way to SQLite

This document outlines the idiomatic "Wails Way" to build a local SQLite browser, contrasting it with the traditional web-server-in-a-binary approach.

## 1. The Core Philosophy: "Bindings over HTTP"

In a traditional web app (like `sqliter`), the frontend talks to the backend via HTTP requests (REST/JSON).
In Wails, the frontend talks to the backend via **Bindings**.

- **Don't**: Run a `net/http` server inside your app and fetch `http://localhost:port/api/...`.
- **Do**: Define methods on your `App` struct and call them directly from Javascript.

## 2. Architecture Overview

### Backend (Go)
The Go side shouldn't be a stateless request handler. It should hold the **Application State**.
- **State**: The currently open `database/sql` connection.
- **Methods**: Exposed functions to manipulate that state (Open, Query, Close).

### Frontend (React/Vue/TS)
The frontend is a pure view layer. It doesn't need to know about URLs or ports.
- **Actions**: Call `App.OpenDB()`, `App.Query()`.
- **Events**: Listen for signal updates (e.g., `runtime.EventsEmit` when a long query finishes).

## 3. Implementation Blueprint

### A. The Application Struct (`app.go`)

This struct matches the lifetime of your application.

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	_ "modernc.org/sqlite" // Use pure Go SQLite for easy cross-compilation
)

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
```

### B. Opening a Database (Native Dialogs)

Instead of the user typing a path or scanning a directory, use the OS native file picker.

```go
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
```

### C. Querying Data (Typed Returns)

Return Go structs. Wails automatically handles the JSON serialization and TypeScript definition generation.

```go
type QueryResult struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"` // flexible row structure
	Error   string           `json:"error,omitempty"`
}

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
```

### D. The Frontend Integration

When you run `wails dev`, it generates a `wailsjs` directory.

```typescript
// frontend/src/App.tsx
import { useState } from 'react';
import { OpenDatabase, ExecuteQuery } from '../wailsjs/go/main/App';

function App() {
    const [dbPath, setDbPath] = useState("No DB");
    const [data, setData] = useState<any>(null);

    const pickFile = async () => {
        try {
            const path = await OpenDatabase();
            if (path) {
                setDbPath(path);
                // Auto-load tables
                loadTables();
            }
        } catch(e) {
            console.error(e);
        }
    };

    const loadTables = async () => {
        const res = await ExecuteQuery("SELECT * FROM sqlite_master WHERE type='table'");
        setData(res);
    };

    return (
        <div>
            <h1>{dbPath}</h1>
            <button onClick={pickFile}>Open Database</button>
            {data && <pre>{JSON.stringify(data, null, 2)}</pre>}
        </div>
    )
}
```

## 4. Key Advantages over `sqliter` Port

1.  **Safety**: No open network ports. Everything is in-process memory calls.
2.  **Performance**: No HTTP overhead. Serialization is highly optimized by Wails.
3.  **UX**: Native file dialogs feel more "Pro" than pasting paths into an input box.
4.  **Distribution**: Single binary, no side cars.

## 5. Next Steps

1.  Initialize project: `wails init -n wailssqliter -t react`
2.  Copy the Go code above into `app.go`.
3.  Add `modernc.org/sqlite` to `go.mod`.
4.  Build a simple React UI to trigger these functions.
