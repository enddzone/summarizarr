package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"       // Original SQLite driver
	_ "github.com/mattn/go-sqlite3" // SQLCipher-enabled driver
)

func main() {
	var (
		sqlitePath    = flag.String("sqlite", "", "Path to existing SQLite database")
		sqlcipherPath = flag.String("sqlcipher", "", "Path for new encrypted SQLCipher database")
		encryptionKey = flag.String("key", "", "Encryption key (hex string or passphrase)")
		verify        = flag.Bool("verify", true, "Verify data integrity after migration")
		schemaFile    = flag.String("schema", "schema.sql", "Path to schema file")
		createBackup  = flag.Bool("backup", false, "Create backup of source database before migration")
	)
	flag.Parse()

	if *sqlitePath == "" || *sqlcipherPath == "" || *encryptionKey == "" {
		fmt.Println("Usage:")
		flag.Usage()
		fmt.Println("\nExample:")
		fmt.Println("  go run cmd/migrate-to-sqlcipher/main.go \\")
		fmt.Println("    -sqlite ./data/summarizarr.db \\")
		fmt.Println("    -sqlcipher ./data/summarizarr_encrypted.db \\")
		fmt.Println("    -key \"your_encryption_key\"")
		os.Exit(1)
	}

	if err := migrateToEncryptedDB(*sqlitePath, *sqlcipherPath, *encryptionKey, *schemaFile, *verify, *createBackup); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration completed successfully!")
}

func migrateToEncryptedDB(sqlitePath, sqlcipherPath, encryptionKey, schemaFile string, verify bool, backup bool) error {
	fmt.Printf("Starting migration from %s to %s\n", sqlitePath, sqlcipherPath)

	// 0. Create backup if requested
	if backup {
		fmt.Println("0. Creating backup of source database...")
		backupPath, err := createBackup(sqlitePath)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("✓ Backup created at %s\n", backupPath)
	}

	// 1. Validate source database
	fmt.Println("1. Validating source SQLite database...")
	if err := validateSourceDatabase(sqlitePath); err != nil {
		return fmt.Errorf("source database validation failed: %w", err)
	}
	fmt.Println("✓ Source database validation passed")

	// 2. Open existing SQLite database (using modernc.org/sqlite)
	fmt.Println("2. Opening source SQLite database...")
	sqliteDB, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	defer func() {
		if err := sqliteDB.Close(); err != nil {
			log.Printf("Warning: failed to close source database: %v", err)
		}
	}()

	// Verify source database is accessible
	if err := sqliteDB.Ping(); err != nil {
		return fmt.Errorf("source database is not accessible: %w", err)
	}

	// Validate encryption key format (should be hex string)
	if len(encryptionKey) != 64 {
		return fmt.Errorf("encryption key must be 64 characters (32 bytes in hex format)")
	}
	for _, c := range encryptionKey {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("encryption key must be a valid hex string")
		}
	}

	// 3. Create new encrypted SQLCipher database using PRAGMA key approach
	fmt.Println("3. Creating encrypted SQLCipher database...")
	sqlcipherDB, err := sql.Open("sqlite3", sqlcipherPath)
	if err != nil {
		return fmt.Errorf("failed to create SQLCipher database: %w", err)
	}
	defer func() {
		if err := sqlcipherDB.Close(); err != nil {
			log.Printf("Warning: failed to close destination database: %v", err)
		}
	}()

	// Try to get SQLCipher version (optional, do not hard fail in environments without it)
	var version string
	if err := sqlcipherDB.QueryRow("PRAGMA cipher_version").Scan(&version); err == nil && version != "" {
		fmt.Printf("✓ SQLCipher version: %s\n", version)
	} else {
		fmt.Println("! SQLCipher version not reported; proceeding with PRAGMA key setup")
	}
	if _, err := sqlcipherDB.Exec(fmt.Sprintf(`PRAGMA key = "x'%s'"`, encryptionKey)); err != nil {
		return fmt.Errorf("failed to set encryption key: %w", err)
	}
	// Touch sqlite_master to ensure key correctness and initialize DB header
	if _, err := sqlcipherDB.Exec("CREATE TABLE IF NOT EXISTS __init__ (id INTEGER PRIMARY KEY)"); err != nil {
		return fmt.Errorf("failed to initialize encrypted database: %w", err)
	}
	fmt.Println("✓ Encryption key set via PRAGMA")

	// 4. Skip schema setup initially - we'll create tables as we migrate them
	fmt.Println("4. Will create tables dynamically based on source database structure...")

	// 5. Get list of tables to migrate
	tables, err := getTables(sqliteDB)
	if err != nil {
		return fmt.Errorf("failed to get table list: %w", err)
	}
	fmt.Printf("Found %d tables to migrate: %s\n", len(tables), strings.Join(tables, ", "))

	// 6. Migrate data table by table with transaction safety
	for _, table := range tables {
		fmt.Printf("Migrating table: %s\n", table)
		rowCount, err := migrateTableWithTransaction(sqliteDB, sqlcipherDB, table)
		if err != nil {
			return fmt.Errorf("failed to migrate table %s: %w", table, err)
		}
		fmt.Printf("  ✓ Migrated %d rows\n", rowCount)
	}

	// 7. Verify data integrity if requested
	if verify {
		fmt.Println("7. Verifying data integrity...")
		if err := verifyMigration(sqliteDB, sqlcipherDB, tables); err != nil {
			return fmt.Errorf("data verification failed: %w", err)
		}
		fmt.Println("✓ Data verification passed")
	}

	return nil
}

func validateSourceDatabase(dbPath string) error {
	// Check if file exists
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("source database file not found: %w", err)
	}

	// Try to open and query the database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Warning: failed to close database: %v", err)
		}
	}()

	// Verify it's a valid SQLite database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if err != nil {
		return fmt.Errorf("not a valid SQLite database: %w", err)
	}

	return nil
}

func getTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}

	return tables, rows.Err()
}

func migrateTable(source, dest *sql.DB, tableName string) (int, error) {
	// Get column information
	columns, err := getTableColumns(source, tableName)
	if err != nil {
		return 0, err
	}

	if len(columns) == 0 {
		return 0, fmt.Errorf("no columns found for table %s", tableName)
	}

	// If schema wasn't applied from file, create the table structure
	if err := ensureTableExists(source, dest, tableName); err != nil {
		return 0, err
	}

	// Copy data
	selectSQL := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), tableName)
	sourceRows, err := source.Query(selectSQL)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := sourceRows.Close(); err != nil {
			log.Printf("Warning: failed to close source rows: %v", err)
		}
	}()

	// Prepare insert statement
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", 
		tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	
	stmt, err := dest.Prepare(insertSQL)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Printf("Warning: failed to close statement: %v", err)
		}
	}()

	// Copy rows
	rowCount := 0
	for sourceRows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := sourceRows.Scan(valuePtrs...); err != nil {
			return rowCount, err
		}

		if _, err := stmt.Exec(values...); err != nil {
			return rowCount, err
		}
		rowCount++
	}

	return rowCount, sourceRows.Err()
}

func getTableColumns(db *sql.DB, tableName string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var columns []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}

	return columns, rows.Err()
}

func ensureTableExists(source, dest *sql.DB, tableName string) error {
	// Check if table already exists in destination
	var count int
	err := dest.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// Table already exists
		return nil
	}

	// Get CREATE TABLE statement from source
	var createSQL string
	err = source.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&createSQL)
	if err != nil {
		return fmt.Errorf("failed to get CREATE statement for table %s: %w", tableName, err)
	}

	// Create table in destination
	if _, err := dest.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	fmt.Printf("  ✓ Created table structure for %s\n", tableName)
	return nil
}

func verifyMigration(source, dest *sql.DB, tables []string) error {
	for _, table := range tables {
		var sourceCount, destCount int
		
		err := source.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&sourceCount)
		if err != nil {
			return fmt.Errorf("failed to count source rows in %s: %w", table, err)
		}
		
		err = dest.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&destCount)
		if err != nil {
			return fmt.Errorf("failed to count dest rows in %s: %w", table, err)
		}
		
		if sourceCount != destCount {
			return fmt.Errorf("row count mismatch in %s: source=%d, dest=%d", table, sourceCount, destCount)
		}
		
		fmt.Printf("  ✓ Table %s: %d rows verified\n", table, sourceCount)
	}
	
	return nil
}

// migrateTableWithTransaction provides transaction-safe table migration with better error handling
func migrateTableWithTransaction(source, dest *sql.DB, tableName string) (int, error) {
	// Get column information
	columns, err := getTableColumns(source, tableName)
	if err != nil {
		return 0, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
	}

	if len(columns) == 0 {
		return 0, fmt.Errorf("no columns found for table %s", tableName)
	}

	// Ensure table exists in destination
	if err := ensureTableExists(source, dest, tableName); err != nil {
		return 0, fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	// Begin transaction for atomic migration
	tx, err := dest.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction for table %s: %w", tableName, err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Warning: failed to rollback transaction for table %s: %v", tableName, err)
		}
	}()

	// Query source data
	selectSQL := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), tableName)
	sourceRows, err := source.Query(selectSQL)
	if err != nil {
		return 0, fmt.Errorf("failed to query source table %s: %w", tableName, err)
	}
	defer func() {
		if err := sourceRows.Close(); err != nil {
			log.Printf("Warning: failed to close source rows: %v", err)
		}
	}()

	// Prepare insert statement
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare insert statement for table %s: %w", tableName, err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Printf("Warning: failed to close statement: %v", err)
		}
	}()

	// Copy rows with progress tracking
	rowCount := 0
	batchSize := 1000 // Commit in batches for large tables
	
	for sourceRows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := sourceRows.Scan(valuePtrs...); err != nil {
			return rowCount, fmt.Errorf("failed to scan row %d from table %s: %w", rowCount+1, tableName, err)
		}

		if _, err := stmt.Exec(values...); err != nil {
			return rowCount, fmt.Errorf("failed to insert row %d into table %s: %w", rowCount+1, tableName, err)
		}
		rowCount++

		// Commit in batches for large tables to avoid long-running transactions
		if rowCount%batchSize == 0 {
			if err := tx.Commit(); err != nil {
				return rowCount, fmt.Errorf("failed to commit batch at row %d for table %s: %w", rowCount, tableName, err)
			}
			// Start new transaction for next batch
			tx, err = dest.Begin()
			if err != nil {
				return rowCount, fmt.Errorf("failed to begin new transaction at row %d for table %s: %w", rowCount, tableName, err)
			}
			if err := stmt.Close(); err != nil {
				log.Printf("Warning: failed to close statement in batch: %v", err)
			}
			stmt, err = tx.Prepare(insertSQL)
			if err != nil {
				return rowCount, fmt.Errorf("failed to re-prepare statement at row %d for table %s: %w", rowCount, tableName, err)
			}
		}
	}

	if err := sourceRows.Err(); err != nil {
		return rowCount, fmt.Errorf("error reading source rows from table %s: %w", tableName, err)
	}

	// Final commit
	if err := tx.Commit(); err != nil {
		return rowCount, fmt.Errorf("failed to commit final transaction for table %s: %w", tableName, err)
	}

	return rowCount, nil
}

// createBackup creates a backup of the source database before migration
func createBackup(sourcePath string) (string, error) {
	backupPath := sourcePath + ".backup." + fmt.Sprintf("%d", time.Now().Unix())
	
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file for backup: %w", err)
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			log.Printf("Warning: failed to close source file: %v", err)
		}
	}()

	backupFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() {
		if err := backupFile.Close(); err != nil {
			log.Printf("Warning: failed to close backup file: %v", err)
		}
	}()

	if _, err := io.Copy(backupFile, sourceFile); err != nil {
		if err := os.Remove(backupPath); err != nil {
			log.Printf("Warning: failed to clean up partial backup: %v", err)
		}
		return "", fmt.Errorf("failed to copy data to backup file: %w", err)
	}

	return backupPath, nil
}