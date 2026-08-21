# Revisión funcional O3c P6 V4 — candidato `5658742`

Tarea: `O3C-P6-CAP021-RUNTIME-TMP-V4-ATTESTATION`  
Base: `14c1f31`  
Candidato exacto: `5658742` (hijo de `7b24171`)

## Dictamen

**GO funcional independiente, en alcance estático.** No se ejecutó conducta ni
se editaron fuentes. Esta acta es el único write-set de la revisión.

## Comprobaciones

- El conductor asigna siete selectores a C17, C18, C19, C20 y C22; C21 queda
  en cero; cada CAP recibe siete; C17-BF recibe uno. La cuenta exigida es
  `(5×2×7) + (100×2×7) + (1×2) = 1472`.
- `casos.tsv` y `fuentes.tsv` se copian al staging como ledgers privados 0400,
  se cachean sus hashes y las fases posteriores usan esas copias.
- El contexto final usa ruta explícita dentro del paquete y se escribe después
  de retirar staging; el paquete incluye `publicacion.tsv`, `SHA256SUMS`,
  `utilidades.tsv`, `residuos.txt`, `resumen.txt`, `tail` ya no necesario y el
  helper `rename_noreplace`.
- `SHA256SUMS` incluye los binarios normal/race y el helper de publicación;
  este se compila con el Go fijado, se sella y se verifica antes de invocar
  `renameat2(RENAME_NOREPLACE)`. Se conservan las comprobaciones de origen
  ausente, destino real e identidad dev:inode/UID/modo.
- La atestación de selectores y el marcador FD se leen mediante descriptores
  previamente abiertos y se valida su identidad física; el marcador exige
  contenido exacto `si` y cierre de FD posterior.

## Pruebas no conductuales

- `bash -n`: GO.
- `shellcheck -x -e SC2154`: GO, sin diagnósticos.
- `git diff --check 14c1f31..5658742`: GO.
- No se ejecutó conductor, corrida canónica, PostgreSQL ni comportamiento de
  selectores.

## Límites

Este GO no autoriza integración, publicación, CI remota, despliegue ni cambio
de métricas; queda sujeto a revisión de seguridad y puertas posteriores.

