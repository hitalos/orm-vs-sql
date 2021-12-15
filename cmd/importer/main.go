package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	DSN            = "host=localhost port=5432 user=postgres dbname=postgres password=postgres sslmode=disable"
	insertSQL      = "INSERT INTO municipios (id, nome, uf, populacao_2018, populacao_2019, populacao_2020, populacao_2021) VALUES ($1, $2, $3, $4, $5, $6, $7);"
	createTableSQL = `CREATE TABLE municipios (
		id INTEGER NOT NULL PRIMARY KEY,
		nome VARCHAR NOT NULL,
		uf VARCHAR(2) NOT NULL,
		populacao_2018 INTEGER NOT NULL,
		populacao_2019 INTEGER NOT NULL,
		populacao_2020 INTEGER NOT NULL,
		populacao_2021 INTEGER NOT NULL,
		UNIQUE (nome, uf)
	);`
)

type Municipio struct {
	ID             uint64 `gorm:"type:INTEGER;primary_key"`
	Nome           string `gorm:"uniqueIndex:idx_nome_uf"`
	UF             string `gorm:"uniqueIndex:idx_nome_uf"`
	Populacao_2018 uint64
	Populacao_2019 uint64
	Populacao_2020 uint64
	Populacao_2021 uint64
}

func main() {
	gormImport()
	gormTransactionImport()
	gormBatchImport()
	pgxImport()
	pgxTransactionImport()
	pgxBatchImport()
}

func readAll() []Municipio {
	f, err := os.Open("dados/ibge.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		panic(err)
	}

	municipios := make([]Municipio, len(records))
	for i, record := range records {
		id, err := strconv.ParseUint(record[0], 10, 64)
		if err != nil {
			panic(err)
		}

		p2018, err := strconv.ParseUint(record[3], 10, 64)
		if err != nil {
			panic(err)
		}

		p2019, err := strconv.ParseUint(record[4], 10, 64)
		if err != nil {
			panic(err)
		}

		p2020, err := strconv.ParseUint(record[5], 10, 64)
		if err != nil {
			panic(err)
		}

		p2021, err := strconv.ParseUint(record[6], 10, 64)
		if err != nil {
			panic(err)
		}

		municipios[i] = Municipio{
			ID:             id,
			Nome:           record[1],
			UF:             record[2],
			Populacao_2018: p2018,
			Populacao_2019: p2019,
			Populacao_2020: p2020,
			Populacao_2021: p2021,
		}
	}

	return municipios
}

func gormInit() *gorm.DB {
	db, err := gorm.Open(postgres.Open(DSN), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	fmt.Println("Removendo tabela…")
	db.Exec("DROP TABLE municipios;")

	fmt.Println("Criando tabela…")
	if err := db.AutoMigrate(&Municipio{}); err != nil {
		panic(err)
	}

	return db
}

func gormImport() {
	db := gormInit()

	fmt.Println("Importando dados…")
	municipios := readAll()
	start := time.Now()
	for _, m := range municipios {
		if err := db.Create(&m).Error; err != nil {
			panic(err)
		}
	}
	fmt.Println("Tempo de importação com GORM:", time.Since(start))
}

func gormTransactionImport() {
	db := gormInit()

	fmt.Println("Importando dados…")
	municipios := readAll()
	start := time.Now()
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, m := range municipios {
			if err := tx.Create(&m).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Tempo de importação com GORM (usando transação):", time.Since(start))
}

func gormBatchImport() {
	db := gormInit()

	fmt.Println("Importando dados…")
	municipios := readAll()

	start := time.Now()
	db.Create(&municipios)
	fmt.Println("Tempo de importação com GORM (usando Batch):", time.Since(start))
}

func pgxInit() *pgx.Conn {
	db, err := pgx.Connect(context.Background(), DSN)
	if err != nil {
		panic(err)
	}

	fmt.Println("Removendo tabela…")
	_, err = db.Exec(context.Background(), "DROP TABLE municipios;")
	if err != nil {
		panic(err)
	}

	fmt.Println("Criando tabela…")
	_, err = db.Exec(context.Background(), createTableSQL)
	if err != nil {
		panic(err)
	}

	return db
}

func pgxImport() {
	db := pgxInit()

	municipios := readAll()

	_, err := db.Prepare(context.Background(), "insertMunicipio", insertSQL)
	if err != nil {
		panic(err)
	}

	fmt.Println("Importando dados…")
	start := time.Now()
	for _, m := range municipios {
		_, err = db.Exec(context.Background(), "insertMunicipio", m.ID, m.Nome, m.UF, m.Populacao_2018, m.Populacao_2019, m.Populacao_2020, m.Populacao_2021)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Tempo de importação com PGX:", time.Since(start))
}

func pgxTransactionImport() {
	db := pgxInit()

	municipios := readAll()

	tx, err := db.Begin(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println("Importando dados…")
	start := time.Now()
	for _, m := range municipios {
		_, err = tx.Exec(context.Background(), insertSQL, m.ID, m.Nome, m.UF, m.Populacao_2018, m.Populacao_2019, m.Populacao_2020, m.Populacao_2021)
		if err != nil {
			panic(err)
		}
	}
	tx.Commit(context.Background())

	fmt.Println("Tempo de importação com PGX (usando transação):", time.Since(start))
}

func pgxBatchImport() {
	db := pgxInit()

	municipios := readAll()
	batch := pgx.Batch{}

	fmt.Println("Importando dados…")
	start := time.Now()
	for _, m := range municipios {
		batch.Queue(insertSQL, m.ID, m.Nome, m.UF, m.Populacao_2018, m.Populacao_2019, m.Populacao_2020, m.Populacao_2021)
	}

	results := db.SendBatch(context.Background(), &batch)
	results.Close()

	fmt.Println("Tempo de importação com PGX (usando Batch):", time.Since(start))
}
