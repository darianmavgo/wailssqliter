package main

import (
	"context"

	"github.com/darianmavgo/sqliter/sqliter"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx    context.Context
	engine *sqliter.Engine
	// We might need to keep track of current selection for pure file opening if desired,
	// but Engine works with relative paths from ServeFolder.
	// For Wails Desktop, ServeFolder might be "/" (root) or user defined.
	// Let's assume we allow full system access if ServeFolder is "/" ?
	// Engine restricts ".." so we must set ServeFolder to root to access anything.
}

// NewApp creates a new App application struct
func NewApp() *App {
	// For desktop app, we probably want to allow access to user files.
	// So we set ServeFolder to root? Or User Home?
	// Setting it to "/" on unix allows everything (but Engine checks for ".." which stops breaking out of ServeFolder.
	// If ServeFolder is /, you can't break out.)
	cfg := &sqliter.Config{
		ServeFolder: "/",
	}
	return &App{
		engine: sqliter.NewEngine(cfg),
	}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// Shutdown is called at termination
func (a *App) Shutdown(ctx context.Context) {
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

	return selection, nil
}

func (a *App) ListFiles(dir string) ([]sqliter.FileEntry, error) {
	return a.engine.ListFiles(dir)
}

func (a *App) ListTables(db string) ([]sqliter.TableInfo, error) {
	return a.engine.ListTables(db)
}

func (a *App) Query(opts sqliter.QueryOptions) (*sqliter.QueryResult, error) {
	return a.engine.Query(opts)
}
