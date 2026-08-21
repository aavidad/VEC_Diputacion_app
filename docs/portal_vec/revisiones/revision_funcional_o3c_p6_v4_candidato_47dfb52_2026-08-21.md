# Revisión funcional O3c P6 V4 — candidato `47dfb52`

Tarea: `O3C-P6-CAP021-RUNTIME-TMP-V4-ATTESTATION`  
Base declarada: `14c1f31`  
Candidato auditado: `47dfb52` (`fix(O3C-P6): acredita temporales por selector`)

## Estado

**NO-GO funcional.** La revisión es independiente y read-only respecto de las
fuentes. El único fichero escrito por esta revisión es esta acta.

## Hallazgo bloqueante

`tools/o3c_p6_conductor/conductor.sh` fija `selectores_esperados=7` para cada
caso `C18_O4A_OPACO`–`C22_RESIDUOS` y para cada `CAP_*`. El ledger de casos
contiene 22 casos y el conductor ejecuta cada uno en los dos modos; además
ejecuta 100 capturas CAP por modo. Por tanto, la cantidad que el propio código
añade a `tmpdir_selectores.tsv` es:

```
(22 × 2 + 100 × 2) × 7 + (C17_BF × 2 × 1) = 1710
```

Sin embargo, al final exige `filas_selectores -eq 1472`. La especificación y la
enmienda también declaran 1472, pero esa cifra no corresponde al conjunto de
invocaciones del candidato. La condición hace imposible un GO canónico aunque
todas las atestaciones sean válidas.

## Auditoría del contrato

Se verificó por inspección estática que el candidato sí intenta cubrir los
siete selectores de C18–C22/CAP y uno (`particion`) para C17-BF, y que registra
dev/inode/UID/modo, entradas, residuos, `lstat_enoent`, raíz exterior, retirada
y resultado. También se observan el marcador de cierre de FD del intermediario,
snapshot con rehash, ledger único, `mv -n -T`, `SHA256SUMS`, `utilidades.tsv`,
`tail` y retirada explícita del staging antes de la segunda captura de contexto.
Estas comprobaciones no mitigan la contradicción aritmética bloqueante.

## Pruebas no conductuales

- `bash -n`: GO.
- `shellcheck -x -e SC2154`: GO, sin diagnósticos.
- `gofmt -d` sobre el Go modificado: GO, salida vacía.
- `git diff --check 14c1f31..47dfb52`: GO.
- No se ejecutó el conductor, la corrida canónica, PostgreSQL ni conducta de
  los selectores.

## Alcance y seguridad

No se editaron fuentes, ledgers, workflow ni documentos transversales; no se
realizó publicación ni despliegue. El write-set de esta revisión contiene
únicamente este Markdown.

## Cierre

La corrección mínima debe resolver explícitamente la cardinalidad esperada y
actualizar la evidencia contractual antes de una nueva revisión independiente.
Este acta no autoriza integración, publicación, CI remota, O3c posterior ni
cambio de métricas.

