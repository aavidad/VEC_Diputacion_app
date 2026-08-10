# Conductor durable O3b P7

Entrada única externa, sin modificar ni arrancar el modo operativo:

```text
./conductor.sh RUTA_CHECKOUT DIRECTORIO_EVIDENCIA_INEXISTENTE
```

El conductor exige que la base `d9f8aeb547f5d1b3b9ab3eb786382f78ef964e28`
sea ancestro del checkout, Go 1.26.5 linux/amd64 y las 22 fuentes exactas de
`fuentes.tsv`. Así sigue siendo reproducible después de publicar el propio
conductor sin permitir que cambie ningún byte P1--P6. Compila dos
binarios privados (`CGO_ENABLED=0` y `-race`), ejecuta los 17 grupos de
oráculos en ambos modos y el oráculo 18 como cien capturas independientes por
modo. Cada fila
conserva ID, comando lógico, SHA del target, estado, tamaños de salida,
duración e inventarios antes/después. Un estado no cero, hijo, zombi o FD
residual hace fallar toda la corrida; no hay `SKIP` ni reintento.

`SHA256SUMS` se genera fuera del destino y se valida antes de `GO`. La
evidencia no contiene rutas de staging, cachés, binarios ni datos reales.
La única evidencia vigente es `evidencia-p7-final-r2`: 234 filas, 117 por
modo, con 100 capturas normal y 100 `race`, todos los inventarios sin delta y
`residuos.txt` vacío. `resumen.txt` liga además los SHA del conductor, matriz
y catálogo de fuentes usados para producirla.
