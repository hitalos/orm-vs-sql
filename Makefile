ANOS=2018|2019|2020|2021
REGIONS=1,2,3,4,5
URL='https://servicodados.ibge.gov.br/api/v3/agregados/6579/periodos/$(ANOS)/variaveis/9324?localidades=N6\[N2\[$(REGIONS)\]\]'
CSVFILTER='.[].resultados[].series[] | "\"\(.localidade.id)\",\"\(.localidade.nome)\",\"\(.serie."2018")\",\"\(.serie."2019")\",\"\(.serie."2020")\",\"\(.serie."2021")\""'
JSONFILTER='[ .[].resultados[].series[] | { id: .localidade.id, nome: .localidade.nome, populacao: .serie } ]'

dados/ibge.csv:
	curl $(URL) | jq -r $(CSVFILTER) | sed -e 's/ - /","/' > dados/ibge.csv

dados/ibge.json:
	curl $(URL) |jq $(JSONFILTER) > dados/ibge.json

importer:
	go build -o importer ./cmd/importer

counter:
	go build -o counter ./cmd/counter