# Autoridad mutante O3b P7

La expansión contractual completa se divide sin solapamiento en dos ejecutores:

- `o3b_p7_mutantes_v3a`: 81 alternativas atómicas B01–B18;
- `o3b_p7_mutantes_v3b`: 50 alternativas atómicas B19–B30, B21A y B25A.

Los 131 mutantes compilan y mueren. V3A usa 71 oráculos conductuales y diez
aserciones AST de señalización que no se ejecutan por seguridad. V3B usa 14
oráculos conductuales, 33 aserciones AST/CFG positivas y tres meta-pruebas del
conductor B30. Ningún superviviente se clasifica por mera diferencia textual.

La autoridad conjunta se reproduce desde la raíz del repositorio con:

```sh
tools/o3b_p7_mutantes/validar.sh
```

El validador comprueba las huellas congeladas de ambos ejecutores, sus sumas,
la cardinalidad única de los IDs, las 32 familias y la ausencia de `SKIP` o
supervivientes. Los directorios `evidencia/` y `evidencia-v3/` conservan los
comandos completos de regeneración en sus manifiestos y README respectivos.
