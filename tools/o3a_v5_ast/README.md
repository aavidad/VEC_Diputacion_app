# Analizador O3a V5 (proyección externa)

Herramienta externa a la integradora que parsea y tipa conjuntamente
G1, G4, G6a--c y G7a--b. Emite definiciones, raíces y DAG de llamadas en JSON;
además falla si la cardinalidad de `Start` no es uno, si la llamada no vive en
G6c o si aparecen APIs directas prohibidas del contrato.

Ejecución:

```sh
go run ./tools/o3a_v5_ast -dir deploy/postgresql/autorizacion_atestada_v3/pruebas_sql > o3a-v5-dag.json
```

Estado: candidato material completo, pendiente de cierre de revisión
independiente. El analizador
codifica las propiedades estructurales del contrato y
`manifest_parcial.json` conserva 195 mutantes atómicos M001..M195, las 66
familias, patrón anterior/posterior literal único y oráculo causal ejecutado.
El manifiesto queda `COMPLETO_EJECUTADO`: la reproducción durable autoritativa
V21 generó BASE+M001..M195, con 195 transformaciones compilables muertas y cero
supervivientes. Un ledger V21 separado conserva BASE+cuatro mutantes SEC
muertos. Cada par incluye inventarios de las diez fuentes, GOROOT completo,
GOTOOLDIR, fuentes/receta de las herramientas y el snapshot ejecutable.
La BASE autorizada de las diez fuentes es
`60def025db7355b81d3036998996e8e6643da1d8713a0510c218259d2921ba75`.

`ejecutar_p1.sh` copia las diez fuentes a un directorio aislado, aplica un único
patrón por ID, exige `gofmt`, `go vet` y build antes del oráculo y distingue
`muerto_ast` de muerte conductual. El cierre exige un ledger integral único,
controles verdes, 195 transformaciones compilables y cero supervivientes.

Validación reproducible:

```sh
pruebas_sql=/ruta/proyeccion/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql
go run ./tools/o3a_v5_ast/validar_manifest \
  ./tools/o3a_v5_ast/manifest_parcial.json "$pruebas_sql" \
  ./tools/o3a_v5_ast/catalogo_expansion_m57_m63_m66.json \
  ./tools/o3a_v5_ast/evidencia/ledger_v21_m001_m195.tsv \
  ./tools/o3a_v5_ast/evidencia/ledger_v21_sec.tsv
go run ./tools/o3a_v5_ast -dir "$pruebas_sql" >o3a-v5-dag.json
```

Los ledgers V16--V20 son evidencia genealógica revocada. Se conservan para
explicar las correcciones, pero no son entrada autorizada ni deben publicarse
como resultado vigente; su fallo ante el validador actual es deliberado. La
única autoridad mutante del corte es el par V21 nombrado en el comando anterior
y todos sus sidecars homónimos.

Esto acredita el candidato externo; no autoriza por sí solo integración,
commit, publicación ni producción.
