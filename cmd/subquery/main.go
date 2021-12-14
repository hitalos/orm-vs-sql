package main

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	DSN = "host=localhost port=5432 user=postgres dbname=postgres password=postgres sslmode=disable"
)

type Municipio struct {
	Nome string
	Ufs  string
	Qtd  uint64
}

func (m Municipio) String() string {
	return fmt.Sprintf("%s - %s - %d", m.Nome, m.Ufs, m.Qtd)
}

func main() {
	db := gormInit()

	municipios := []Municipio{}
	subquery := db.Table("municipios").
		Select("municipio AS nome, STRING_AGG(uf, ',') AS ufs, COUNT(*) AS qtd").
		Group("municipio").
		Order("qtd DESC, municipio")
	db.Model(&Municipio{}).
		Table("(?) AS homonimos", subquery).
		Where("qtd > 1").
		Scan(&municipios)

	for _, m := range municipios {
		fmt.Println(m)
	}
}

func gormInit() *gorm.DB {
	db, err := gorm.Open(postgres.Open(DSN), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	return db
}
