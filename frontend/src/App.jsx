import {useState, useEffect, useCallback, useRef} from 'react';
import {OpenDatabase, ExecuteQuery, StreamQuery} from '../wailsjs/go/main/App';
import {EventsOn, EventsOff} from '../wailsjs/runtime/runtime';
import { AgGridReact } from 'ag-grid-react'; 
import { ModuleRegistry, AllCommunityModule } from 'ag-grid-community';
import "ag-grid-community/styles/ag-grid.css"; 
import "ag-grid-community/styles/ag-theme-quartz.css"; 
import './App.css';

// Register all community features
ModuleRegistry.registerModules([ AllCommunityModule ]);

function App() {
    const [dbPath, setDbPath] = useState("");
    const [tables, setTables] = useState([]);
    const [selectedTable, setSelectedTable] = useState("");
    
    // Grid State
    const [rowData, setRowData] = useState([]);
    const [colDefs, setColDefs] = useState([]);
    const [error, setError] = useState(null);
    const [loading, setLoading] = useState(false);
    
    const gridApiRef = useRef(null);

    // Event Listeners Setup
    useEffect(() => {
        const onColumns = (cols) => {
             const defs = cols.map(col => ({ field: col, filter: true, sortable: true }));
             setColDefs(defs);
             setRowData([]); // Clear previous data
        };

        const onRows = (chunk) => {
            if (gridApiRef.current) {
                gridApiRef.current.applyTransaction({ add: chunk });
            } else {
                 // Fallback if grid not ready (shouldn't happen often if we mount correctly)
                 setRowData(prev => [...prev, ...chunk]);
            }
        };

        const onDone = () => {
            setLoading(false);
        };

        const onError = (msg) => {
            setError(msg);
            setLoading(false);
        }

        // Subscribe
        EventsOn("query_columns", onColumns);
        EventsOn("query_rows", onRows);
        EventsOn("query_done", onDone);
        EventsOn("query_error", onError);

        return () => {
            // Cleanup effectively impossible with current Wails runtime js without strict identity, 
            // but we can just leave them or use EventsOff if we had the handler ref.
            // Simplified for this demo.
        }
    }, []);

    const onGridReady = (params) => {
        gridApiRef.current = params.api;
    };

    const pickFile = async () => {
        try {
            const path = await OpenDatabase();
            if (path) {
                setDbPath(path);
                setError(null);
                loadTables();
            }
        } catch(e) {
            console.error(e);
            setError(String(e));
        }
    };

    const loadTables = async () => {
        const res = await ExecuteQuery("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name");
        if (res.error) {
            setError(res.error);
            setTables([]);
        } else {
            const tableNames = res.rows.map(r => r.name);
            setTables(tableNames);
            if (tableNames.length > 0) {
                setSelectedTable(tableNames[0]);
                startStream(tableNames[0]);
            } else {
                setSelectedTable("");
                setRowData([]);
                setColDefs([]);
            }
        }
    };

    const startStream = (tableName) => {
        if (!tableName) return;
        setLoading(true);
        setError(null);
        setRowData([]); // Clear UI Immediately
        
        // This is async but returns immediately, events will follow
        StreamQuery(`SELECT * FROM "${tableName}"`);
    }

    const handleTableChange = (e) => {
        const tbl = e.target.value;
        setSelectedTable(tbl);
        startStream(tbl);
    };

    return (
        <div className="app-container">
            <div className="top-bar">
                <button onClick={pickFile}>Open DB...</button>
                
                {dbPath ? (
                    <>
                        <select value={selectedTable} onChange={handleTableChange} disabled={tables.length === 0}>
                            {tables.map(t => <option key={t} value={t}>{t}</option>)}
                            {tables.length === 0 && <option>No Tables</option>}
                        </select>
                        <span className="path-label" title={dbPath}>{dbPath}</span>
                        {loading && <span style={{color: '#4caf50', marginLeft: '10px'}}>Streaming data...</span>}
                    </>
                ) : (
                    <span className="path-label">No Database Selected</span>
                )}

                {error && <div className="error-message">{error}</div>}
            </div>

            <div className="grid-container">
                 <AgGridReact
                    className="ag-theme-quartz-dark"
                    style={{ width: '100%', height: '100%' }}
                    rowData={rowData}
                    columnDefs={colDefs}
                    defaultColDef={{
                        flex: 1,
                        minWidth: 100,
                        resizable: true,
                    }}
                    onGridReady={onGridReady}
                    pagination={true} 
                    paginationPageSize={100}
                    paginationPageSizeSelector={[100, 500, 1000]}
                />
            </div>
        </div>
    )
}

export default App;
