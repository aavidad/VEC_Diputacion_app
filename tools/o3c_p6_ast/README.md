# Analizador estructural O3c P6

Analizador externo, hermético y fail-closed de las cinco fuentes productivas
O3c P1--P5. Sella sus SHA-256, parsea y tipa el paquete productivo sin pruebas,
construye el DAG de llamadas y ownership tipado (incluidos autociclos) y
comprueba los invariantes estructurales del contrato
O3c P0. No ejecuta el supervisor, procesos ni efectos y no depende de Git,
rutas absolutas, cachés ocultas o variables heredadas.

Los esquemas tipados de la autoridad conjunta y del agregado O4a tienen
cardinalidad y nombres cerrados: no se admite un segundo owner atómico.

Desde un checkout limpio y la raíz del repositorio:

```sh
go test ./tools/o3c_p6_ast
go run ./tools/o3c_p6_ast \
  -dir deploy/postgresql/autorizacion_atestada_v3/pruebas_sql \
  >o3c-p6-ast.json
```

El runner de mutantes usa el mismo analizador contra su checkout temporal y
añade `-permitir-sha`. Ese modo omite **solo** la comparación con las cinco
huellas base; parseo, tipado, DAG, cardinalidades, precedencias y prohibiciones
siguen siendo fail-closed. Así una mutación muere por su oráculo causal y no
por la mera huella global:

```sh
go run ./tools/o3c_p6_ast \
  -permitir-sha \
  -dir "$CHECKOUT_MUTANTE/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql"
```

El JSON liga la base publicada
`c0f2a9945ed2fc5648980ee48b91424a04977655`, la versión efectiva de Go, las
cinco huellas productivas, su resumen y el SHA del DAG. Una fuente distinta,
un error de parseo/tipado, cardinalidad u orden ambiguos, un ciclo o una API
prohibida produce estado distinto de cero.
