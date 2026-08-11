# Revisión funcional final O3c

Fecha: 11 de agosto de 2026.

Identificador: `O3C-P7-EVIDENCIA-FUNCIONAL`.

Estado: **GO, P0=0, P1=0, P2=0**.

## Snapshot y alcance

La revisión fue independiente, de solo lectura y desde el worktree exclusivo
`o3c-p7-revision-funcional-20260811`. Contrastó íntegramente el contrato O3c
SHA-256 `f47395d68fa3f9e39e118f81b07fde8d8792aa61d4820dfb676ff4c7216515b6`
y el ledger final de 199 líneas, SHA-256
`617f2df9763dafaacd91d8a3f9651028804d08b286b74fe8cb5921c6153964a4`.
No editó el worktree productor.

## Reproducción

Se verificaron:

- cadena lineal, padres y ramas remotas P0--P6;
- producto P1--P5 byte-inmutable y sus cinco SHA-256 exactos;
- checksums de conductor y mutantes;
- 244/244 casos: 122 normal y 122 race;
- seis BF directos con estado 65, EOF/no retorno, `stdout=0` y `stderr=0`;
- residuos cero;
- 144/144 mutantes `COMPILA` y `MUERTO` en 24/24 familias;
- AST tipado/DAG, focales normal/race, `go vet`, formato y `git diff --check`.

## Hallazgo corregido

La primera lectura detectó un SHA mal transcrito para `retirada.go`:
`...1716...` en vez de `...1718...`. El ledger fue corregido al valor real
`55500193cce25d83b912f0b6298994864ff970cbd1718ef2dd1f0441ca6461cd`.
La relectura completa del snapshot corregido no encontró otro hallazgo.

## Fronteras

P7 solo documenta evidencia. No modifica producto, herramientas P6, master,
integración, métricas, producción ni O4. La publicación y CI propia P7 siguen
siendo condición posterior al GO material.
