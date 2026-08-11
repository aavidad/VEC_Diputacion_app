# Conductor durable O3c P6

Ejecución hermética desde un checkout limpio:

```text
./tools/o3c_p6_conductor/conductor.sh RUTA_CHECKOUT DIRECTORIO_EVIDENCIA_INEXISTENTE
```

El conductor exige la base P5, Go 1.26.5 y los hashes de las 32 fuentes
congeladas. Compila binarios normal y race privados, ejecuta C01--C22 en ambos
modos y añade cien capturas C22 independientes por modo. Cada fila conserva
comando lógico, SHA target, estado, tamaños de salida, duración, cinco
inventarios y oráculo causal. Tres BF se lanzan directamente por modo y deben
terminar 65, sin retorno, con EOF y cero bytes en stdout y stderr. No hay SKIP,
reintentos ni reutilización de resultados. La evidencia queda ligada por
SHA256SUMS y no contiene rutas privadas, caches, binarios ni datos reales.
