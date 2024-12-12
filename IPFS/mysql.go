package IPFS

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type IPFSTable struct {
	AlgorithmName    string
	AlgorithmFileCid string
	DatasetCid       string
}

func insertIPFSTable(db *sql.DB, table IPFSTable) error {
	query := `INSERT INTO ipfs_table (algorithm_name, algorithm_file_cid, dataset_cid) 
			VALUES (?, ?, ?)`
	_, err := db.Exec(query, table.AlgorithmName, table.AlgorithmFileCid, table.DatasetCid)
	return err
}

func createConnection() (*sql.DB, error) {
	dsn := "user:password@tcp(localhost:3306)/database_name"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping database: %v", err)
	}
	return db, nil
}

func getIPFSTableByName(db *sql.DB, algorithmName string) (*IPFSTable, error) {
	query := `SELECT algorithm_name, algorithm_file_cid, dataset_cid 
			FROM ipfs_table WHERE algorithm_name = ?`
	row := db.QueryRow(query, algorithmName)

	var table IPFSTable
	err := row.Scan(&table.AlgorithmName, &table.AlgorithmFileCid, &table.DatasetCid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no records found for algorithm_name: %s", algorithmName)
		}
		return nil, fmt.Errorf("error retrieving data: %v", err)
	}

	return &table, nil
}
