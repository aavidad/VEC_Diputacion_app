# Mutantes O3b P7 V3B

Este bloque cubre una transformación compilable por cada alternativa de
B19–B30, incluidas B21A y B25A: 50 mutantes en total. Las fuentes Go se toman
del commit base; B25A modifica la copia efímera de la retirada O3a real. B30
usa exclusivamente la copia del conductor cuyo SHA-256 es `5561475016a1cf2b59602e441dd825399356099f84bdcd33a3864169e9374343`.

Cada mutante Go se compila, ejecuta las pruebas O3b y se retira en un PGID
exclusivo hasta acreditar `ESRCH`. Sólo un superviviente conductual llega a la
regla AST/CFG positiva propia de su alternativa. B30 ejecuta fixtures meta
contra fallo/residuos, reutilización de caso y `SKIP`; no se mata buscando el
texto sustituido ni comparando la huella del mutante.

Ejecución completa reproducible desde la raíz del checkout:

```sh
go run ./tools/o3b_p7_mutantes_v3b \
  -repo . \
  -out tools/o3b_p7_mutantes_v3b/evidencia-v3
```

`-desde` y `-hasta` permiten segmentar la ejecución sin cambiar catálogo,
fuentes, transformación ni oráculo. La evidencia canónica conserva una fila
por ID, los 50 SHA mutados y sólo rutas relativas al repositorio.
