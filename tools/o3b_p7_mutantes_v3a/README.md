# Mutantes O3b P7, B01–B18

Este conductor expande las 81 alternativas contractuales B01–B18 en
transformaciones atómicas distintas. Cada árbol se extrae de la base sellada,
se transforma con cardinalidad uno y se compila con las pruebas O3b.

Las 71 transformaciones sin autoridad de señalización deben morir ejecutando
las pruebas O3b. Las diez variantes B08, B14 y B15 no se ejecutan porque
introducen precisamente llamadas o argumentos de señalización prohibidos: un
analizador AST inspecciona la API, el handle, la señal cero y el flag de grupo.
Una prueba verde o una aserción AST verde es siempre una supervivencia y cierra
el ejecutor con estado no cero.

La ejecución monolítica es reproducible con:

```sh
go run ./tools/o3b_p7_mutantes_v3a -repo . \
  -out tools/o3b_p7_mutantes_v3a/evidencia
```

Los flags `-desde` y `-hasta` permiten lotes cerrados cuando el supervisor
exterior limita la duración. `-solo-catalogo` reproduce el catálogo canónico.
La evidencia publicada está consolidada por ID y no conserva rutas temporales,
salidas crudas ni lotes parciales.
