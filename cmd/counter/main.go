package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	DSN = "host=localhost port=5432 user=postgres dbname=postgres password=postgres sslmode=disable"
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
	gormCounter()
	pgxCounter()
}

func gormCounter() {
	db, err := gorm.Open(postgres.Open(DSN), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	start := time.Now()
	rows := []Municipio{}
	result := db.Find(&rows)
	if result.Error != nil {
		panic(result.Error)
	}
	fmt.Println("Encontrados", result.RowsAffected)

	UFs := map[string][]Municipio{}
	for _, m := range rows {
		UFs[m.UF] = append(UFs[m.UF], m)
	}

	orderingUFs := []string{}
	for uf := range UFs {
		orderingUFs = append(orderingUFs, uf)
	}
	sort.Strings(orderingUFs)

	var habitantes uint64
	for _, uf := range orderingUFs {
		habitantes = 0
		for _, m := range UFs[uf] {
			habitantes += m.Populacao_2020
		}
		fmt.Printf("%s: %d habitantes em %d municípios\n", uf, habitantes, len(UFs[uf]))
	}

	fmt.Println("Tempo:", time.Since(start))
}

func pgxCounter() {
	ctx := context.Background()
	db, err := pgx.Connect(ctx, DSN)
	if err != nil {
		panic(err)
	}
	defer db.Close(ctx)

	start := time.Now()
	rows, err := db.Query(ctx, `SELECT uf, COUNT(id), SUM(populacao_2021)
		FROM municipios GROUP BY uf ORDER BY uf`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var (
		uf                string
		count, habitantes uint64
	)
	for rows.Next() {
		if err = rows.Scan(&uf, &count, &habitantes); err != nil {
			panic(err)
		}
		fmt.Printf("%s: %d habitantes em %d municípios\n", uf, habitantes, count)
	}

	fmt.Println("Tempo:", time.Since(start))
}
