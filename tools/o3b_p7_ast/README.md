# Analizador estructural O3b P7

Analizador externo y fail-closed de las seis fuentes productivas O3b P1--P6.
Verifica sus SHA-256 sellados, parsea y tipa el paquete productivo sin pruebas,
construye el DAG O3b y acredita las invariantes estructurales del contrato P0.
No ejecuta el modo supervisor ni depende de Git, rutas absolutas, cachés o
variables heredadas.

Ejecución reproducible desde la raíz del repositorio:

```sh
go test ./tools/o3b_p7_ast
go run ./tools/o3b_p7_ast \
  -dir deploy/postgresql/autorizacion_atestada_v3/pruebas_sql \
  >o3b-p7-ast.json
```

El JSON liga la base publicada `d9f8aeb547f5d1b3b9ab3eb786382f78ef964e28`,
la versión efectiva de Go, las seis huellas, el resumen de fuentes y el SHA del
DAG. Una fuente distinta, un fallo de parseo/tipado, una cardinalidad ambigua,
un ciclo o una API prohibida devuelve estado distinto de cero.
