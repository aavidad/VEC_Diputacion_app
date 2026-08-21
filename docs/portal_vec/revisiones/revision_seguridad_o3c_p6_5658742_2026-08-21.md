# Revisión independiente de seguridad O3C-P6 — `5658742`

Tarea: `O3C-P6`, candidato exacto `5658742` (hijo de `7b24171`).

Estado: **GO** (revisión de seguridad estática independiente).

## Alcance y write-set

Auditoría read-only del conductor y del material de publicación. El único
fichero escrito es esta acta; no se modificaron fuentes, pruebas, conducta,
contrato ni estado canónico.

## Evidencia revisada

- La atestación se crea antes del `exec`, se abre una vez en descriptor de
  lectura y se valida con `stat -L` sobre `/proc/$$/fd/<fd>` comparando
  dispositivo, inode, UID y modo. El parser lee cabecera y filas desde ese
  mismo descriptor; no usa `tail` ni vuelve a leer el pathname.
- El marcador de FD sigue el mismo esquema: descriptor preabierto, identidad
  validada por `/proc/$$/fd`, lectura completa y cierre posterior. El proceso
  intermedio filtra descriptores >=3 antes de convertirse en líder.
- Los ledgers `casos.tsv` y `fuentes.tsv` se copian al staging privado con modo
  `0400` y sus hashes se conservan en el contexto. El snapshot exige fuentes
  regulares sin symlink, hash coincidente y revalidación del checkout.
- El helper Go se compila con Go `1.26.5`, `GOENV=off`, `GOTOOLCHAIN=local`,
  cache dentro del staging y `-trimpath`; se publica dentro del paquete y su
  SHA entra en `SHA256SUMS`. El staging y la caché se retiran antes de la
  revalidación final.
- La publicación usa `renameat2(RENAME_NOREPLACE)`, comprueba ausencia del
  destino, identidad UID/mode/inode del origen y del padre, revalida el padre
  justo antes del rename y verifica `SHA256SUMS` desde el destino. El paquete
  incluye `tmpdir_selectores.tsv`, `utilidades.tsv` y el binario helper.

No observé una brecha bloqueante en los puntos solicitados. La revisión no
declara pruebas dinámicas ni GO funcional; queda limitada a seguridad estática
del árbol exacto.

Commit de acta: pendiente de commit local único.
