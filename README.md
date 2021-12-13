# ORM vs SQL (usando golang)

O objetivo desse artigo é demonstrar porque muitos desenvolvedores preferem montar suas próprias *queries* a deixar as decisões por conta de um [ORM](https://pt.wikipedia.org/wiki/Mapeamento_objeto-relacional). Logicamente não se aplica a todos os casos e vai depender bastante da experiência do desenvolvedor e das situações envolvidas. Todos os códigos usados neste artigo ficarão disponíveis em: [https://github.com/hitalos/orm-vs-sql](https://github.com/hitalos/orm-vs-sql)

Muitos argumentam que não precisam conhecer bem de SQL, já que o ORM resolve tudo sozinho. Em primeiro lugar, nem tudo! Além disso, a modelagem de um banco influencia muito nos resultados. Nem sempre seguir as "receitas" dos tutoriais vai ser suficiente.

Como exemplo, decidi importar alguns dados do [IBGE](https://servicodados.ibge.gov.br/) sobre a estimativa de população dos municípios brasileiros. Não entrarei em detalhes sobre o script que usei para extrair e preparar os dados para o banco por não ser o foco deste artigo, mas aqui está:

```makefile
ANOS=2018|2019|2020|2021
REGIONS=1,2,3,4,5
URL='https://servicodados.ibge.gov.br/api/v3/agregados/6579/periodos/$(ANOS)/variaveis/9324?localidades=N6\[N2\[$(REGIONS)\]\]'
CSVFILTER='.[].resultados[].series[] | "\"\(.localidade.id)\",\"\(.localidade.nome)\",\"\(.serie."2018")\",\"\(.serie."2019")\",\"\(.serie."2020")\",\"\(.serie."2021")\""'

ibge.csv:
	curl $(URL) | jq -r $(CSVFILTER) | sed -e 's/ - /","/' > ibge.csv

```

Levantamos um banco de teste rapidamente no docker:

```bash
docker run -d --name orm_vs_sql -p 5432:5432 postgres:alpine
```

Esta será a estrutura da tabela usada em todos os nossos testes.

```sql
CREATE TABLE IF NOT EXISTS municipios (
    id INTEGER NOT NULL PRIMARY KEY,
    nome VARCHAR NOT NULL,
    uf VARCHAR(2) NOT NULL,
    populacao_2018 INTEGER NOT NULL,
    populacao_2019 INTEGER NOT NULL,
    populacao_2020 INTEGER NOT NULL,
    populacao_2021 INTEGER NOT NULL,
    UNIQUE (nome, uf)
);
```

Poderíamos ter uma coluna "ano" e apenas um coluna para população (quadruplicaria o número de registros) ou mesmo separar essa informação numa outra tabela, mas vamos manter assim para fins de simplificação.

## Importando os dados

Já vamos aproveitar para experimentar 5 métodos. Usaremos o **[GORM](https://gorm.io/)** e depois a biblioteca **[PGX](https://github.com/jackc/pgx)**. Após ler todos os registros do CSV e montar um *slice* com todos os valores devidamente convertidos, percorremos essa lista e contamos o tempo (daqui pra frente apenas) que demora para inserir todos os registros.

|  | GORM | PGX |
| --- | --- | --- |
| Média sem Tx | 6,5s | 5,5s |
| Média com Tx | 550ms | 300ms |
| Com Batch | n/a | 68ms |

Só aqui já vemos como pode fazer diferença.

Com os dados já importados, agora considere a seguinte pergunta:

## Quantos municípios e habitantes tem em cada Unidade Federativa? (liste em ordem alfabética)

Um iniciante que optaria pela comodidade do ORM talvez desconheça os recursos de agregação, comuns no SQL. Provavelmente ele iria consultar todos os registros, iterar sobre eles e fazer as devidas somas e contagens. Simulando aqui, ficaria assim:

```go
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
```

Um programador mais experiente, faria a *query* manualmente e deixaria o processamento por conta do próprio banco de dados. Perceba que os cálculos serão feitos "na fonte" e otimizados da melhor forma por quem mais entende do assunto. Só em gerar um conjunto de dados menor, também teremos menos tráfego. Essa operação já tende a ser mais rápida! Do nosso lado, só precisaríamos mostrar os resultados:

```go
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
		err = rows.Scan(&uf, &count, &habitantes)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s: %d habitantes em %d municípios\n", uf, habitantes, count)
	}

	fmt.Println("Tempo:", time.Since(start))
}
```

Média de tempo de execução para gerar exatamente a mesma saída:

| GORM | PGX |
| --- | --- |
| 25,3ms | 2,2ms |

Além do tempo de execução ser bem mais rápido, até o código ficou mais simples! Pode-se dizer que um usuário experiente do GORM criaria um tipo específico pra essa saída e talvez alcançasse o mesmo resultado, mas vejam como não precisamos complicar nada disso!

Mesmo não sendo um tempo alto, lembrem-se que quase sempre uma consulta não basta para gerar a resposta de uma API ou mesmo para montar uma página de site. Num conjunto de dados maior também poderíamos ter criado um índice para a coluna "uf", já que ela foi usada para agrupar e ordenar (isso favorecia ainda mais a consulta sem ORM).

## Usando a imaginação

É importante conhecer as funcionalidade que o banco tem nativamente. O postgres tem muitos tipo de dados e funções nativas ausentes em outros bancos e que são bem interessantes. Segundo a documentação do PGX, ele reconhece mais de 70 desses tipos. Deixarei aqui alguns exemplos de perguntas e as queries que resolveriam cada caso para vocês exercitarem o conhecimento em GORM e postgresql. Tomem suas próprias conclusões.

### Liste os 10 municípios com nomes mais repetidos (homônimos)

```sql
SELECT
	municipio,
	STRING_AGG(uf, ',') AS uf,
	COUNT(municipio) AS qtd 
FROM
	municipios 
GROUP BY
	municipio 
ORDER BY
	qtd DESC,
	municipio
LIMIT 10;
```

> Este campo "uf" poderia retornar um array e o mesmo seria convertido naturalmente sem precisar "splitar" (veja na próxima query).

### E se eu quisesse a lista de todos que se repetem?

```sql
SELECT * FROM (
	SELECT
		municipio,
		ARRAY_AGG(uf) AS uf,
		COUNT(municipio) AS qtd 
	FROM
		municipios 
	GROUP BY
		municipio 
	ORDER BY
		qtd DESC,
		municipio
) AS homonimos
WHERE qtd > 1
```

> Subqueries complicam um pouco, mas ao meu ver o GORM consegue piorar essa complexidade!

## Concluindo…

Se você tiver que usar um ORM porque a equipe decidiu adotar ou porque "pegou o bonde andando", certifique-se de conhecer um pouco mais que o básico do seu banco de dados para poder fazer o melhor possível e não ser surpreendido por gargalos facilmente evitáveis. Se for "começar do zero", vale mais a pena direcionar seus esforços para ter o melhor domínio possível do seu BD e daí então tomar as suas próprias conclusões sobre a melhor solução para o projeto em questão.