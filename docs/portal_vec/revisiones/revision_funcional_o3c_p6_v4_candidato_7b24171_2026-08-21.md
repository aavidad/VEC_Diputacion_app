# Revisión funcional O3c P6 V4 — candidato `7b24171`

Tarea: `O3C-P6-CAP021-RUNTIME-TMP-V4-ATTESTATION`  
Base: `14c1f31`  
Candidato: `7b24171` (hijo de `47dfb52`)

## Dictamen

**GO funcional independiente, con alcance estático.** No se ejecutó conducta
canónica ni se modificaron fuentes. El único write-set de esta revisión es
esta acta.

## Correcciones verificadas

- El conjunto de siete selectores se aplica exactamente a `C17_OWNERS`,
  `C18_O4A_OPACO`, `C19_RETIRADA`, `C20_POST_CONT`, `C22_RESIDUOS` y `CAP_*`;
  `C21` queda fuera como exige el contrato corregido.
- La cardinalidad es consistente: `(5 agregados + 100 CAP) × 2 modos × 7`
  más `C17_BF × 2 modos × 1` = `1472`.
- La atestación conserva y valida dev/inode/UID/modo, entradas, residuos,
  `lstat_enoent`, raíz exterior y retirada. La identidad física de la TSV de
  selectores se captura antes y se valida después.
- El marcador FD se precrea, se fija a 0600, se sella con dev/inode/UID/modo y
  se acepta solo con contenido exacto de tres bytes (`si`); el intermediario
  comprueba descriptores existentes antes del cierre.
- `registrar_contexto` recibe una ruta explícita y el `trap` no intenta borrar
  staging cuando ya fue retirado. Se mantiene la retirada de staging antes de
  la segunda captura de contexto.
- Snapshot y rehash del ledger, publicación no-replace con identidad física,
  `SHA256SUMS`, `utilidades.tsv` y uso de `tail` permanecen presentes.

## Pruebas no conductuales

- `bash -n` sobre el conductor: GO.
- `shellcheck -x -e SC2154`: GO, sin diagnósticos.
- `gofmt -d` sobre el Go modificado: GO, salida vacía.
- `git diff --check 14c1f31..7b24171`: GO.
- No se ejecutó el conductor, la corrida canónica, PostgreSQL ni pruebas de
  comportamiento.

## Límites

Este GO no autoriza integración, publicación, CI remota, despliegue ni cambio
de métricas. Requiere la revisión de seguridad independiente y las puertas
posteriores previstas por el procedimiento.

