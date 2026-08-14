# Revisión funcional O3c P6: fallo durable del conductor

Fecha: 14 de agosto de 2026.

Tarea revisada: `O3C-P6-CONDUCTOR-FALLO-DURABLE-P0`.

Dictamen: **GO funcional**, con `P0=0`, `P1=0` y `P2=0`, exclusivamente para
el corte que conserva el primer fallo del conductor O3c. Este dictamen no
revoca `CAP_NORMAL_021`, no acredita O3a V3/C21 ni el toolchain y no abre O4,
publicación, CI o producción.

## Identidad, independencia y alcance

La revisión partió del candidato exacto
`55d6c9420e1adc3c37499b7b236ac0cfc101d6d2`, cuyo padre único es
`fea52f3ddf796991c93c85cae992ca695db1ae63` y cuyo árbol es
`b9857dc7d82c0a52bffa575da5957d103558408f`. El rango contiene un commit,
cero merges y tres rutas:

```text
A docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_2026-08-14.md
M tools/o3c_p6_conductor/conductor.sh
A tools/o3c_p6_conductor/fallo_durable.sh
```

El delta es `+234/-3`: enmienda `+120`, conductor `+20/-3` y publicador
`+94`. Los modos Git son `100644`, `100755` y `100755`, respectivamente. La
rama productora
`trabajo/o3c-p6-conductor-fallo-durable-p0-20260814` y este checkout revisor
estaban limpios antes de documentar. No se editó el candidato.

El único write-set revisor es esta acta. Se leyeron completos `AGENTS.md`, el
relevo de sesión, mapa, tablero, relevo de contratación temporal,
especificación RRHH, hoja de ruta, matriz normativa, decisión O3c, ledger y
revisiones O3c, las dos actas `NO-GO` de V3 y la enmienda del candidato.

## Huellas y presupuesto

Las huellas reproducidas son:

| Ruta | Líneas | Bytes | SHA-256 |
| --- | ---: | ---: | --- |
| `docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_2026-08-14.md` | 120 | 4.678 | `6b795bf0fe6b433414dfa8e684a77e7ea8ca9e934235d36b277ab62a0f3e03ea` |
| `tools/o3c_p6_conductor/conductor.sh` | 182 | 10.710 | `6f777f59d4e157b2c4eabb13f66789cfe11b9c01f646e9512af9af388b715538` |
| `tools/o3c_p6_conductor/fallo_durable.sh` | 94 | 3.771 | `8827cd2953aaf0787897062fd1652ef716eaef969717f803715df85573ba7641` |

El conductor y el helper quedan muy por debajo del tope DEC-051 de 800
líneas. No cambia ningún fichero Go, productivo o test-only, matriz, ledger,
workflow, Docker, PostgreSQL, estado transversal o métrica.

## Auditoría del contrato

El orden de ambas rutas negativas es correcto. `ejecutar` y `ejecutar_bf`
calculan el oráculo, añaden primero la fila `NO-GO` al TSV parcial y solo
entonces invocan el publicador. Tras publicarla imprimen el mismo diagnóstico
y devuelven uno; `set -e` detiene el conductor en el primer rojo. No existe una
ruta posterior que pueda escribir un resumen verde sobre ese paquete.

El publicador valida origen, raws, destino, metadatos, HEAD y Go antes de
crear nada. Con `umask 077` crea el temporal en el mismo padre, copia sin
interpretar stdout y stderr, incorpora fila/contexto, escribe
`resultado=NO-GO`, incluye todos los ficheros en `SHA256SUMS`, los verifica y
publica por un único `mv`. Rechaza un destino ya publicado. El conductor lo
usa exclusivamente bajo su `flock`, por lo que la comprobación y el rename no
compiten con otro conductor admitido. Los paquetes negativos observados son
`0700/0600` y no contienen sus raws en Git.

La ruta previa de éxito conserva los 244 casos, seis bloqueos fatales, las
100+100 capturas, residuos cero y el rename final. Sus únicos datos nuevos son
`contexto.tsv`, el hash del publicador en el resumen y la inclusión de ese
contexto en `SHA256SUMS`; no cambian oráculos, cardinalidades, plazos,
aislamiento, compilación ni matrices.

## Evidencia durable comprobada

Se verificaron sin reejecutarlas las dos integraciones sintéticas entregadas:

- caso ordinario, paquete `destinos/r3`: `C01_ENTRADA`, normal, estado uno,
  `resultado=NO-GO`, fila parcial presente, inventarios
  `6,0,0,0,0` antes/después, PGID ausente, exit no cero y
  `SHA256SUMS` `58b3bf27a74b6735c6bd8df9456bc65ad896f104382f3ad16ac6655d64ec41c3`;
- BF, paquete `destinos/r2`: `C01_BF_AUTO`, normal, estado uno,
  `resultado=NO-GO`, fila BF parcial presente, inventarios
  `5,0,0,0,0` antes/después, PGID ausente, exit no cero y
  `SHA256SUMS` `aea1db67a0bb10470a3a739c260651a538d8622d8fb264dbe91a8fefdc82ce86`.

En ambos paquetes, todos los checksums validan; directorio y ficheros son
`0700/0600`. Sus `contexto.tsv` identifican el checkout sintético padre y el
helper exacto, por lo que no se confunden con la corrida canónica candidata.

La única evidencia canónica entregada está en
`/srv/fabrica/revisiones/evidencia-o3c-p6-fallo-durable-55d6c94/r1`. Se
verificó su manifiesto completo: 244/244 casos `GO`, seis/seis BF con estado
65, E/S cero y no retorno, 100+100 capturas, inventarios iguales, PGID ausente
y residuos cero. Las huellas principales son:

```text
resumen.txt     26a15b3be925fefd9d0bb119bed76f67a39281075e1d0cb8a92c168268d8def0
contexto.tsv    79b304751c182175245f9d4b9665347da8dd3665bb3ac3c02f52f19f5f5d0cae
casos.tsv       95e40988a5ed393dff4156c011e353ae589ad784f8220f6d1df63b34c7958163
bf_directos.tsv 5356a28e1254ef72645d96778949264f8d40933efe4b682af4297e8319b37472
SHA256SUMS      54535fafa63bae9a1b95b8d0ad9cb8e7b66466ff2ccf41d76d8ed1c76b0e8e01
```

`contexto.tsv` liga el HEAD exacto, Go 1.26.5 y los hashes exactos de
conductor, publicador, matriz y fuentes. Esta corrida verde solo acredita que
el publicador no altera el camino exitoso: no se repitió y no compensa el rojo
`CAP_NORMAL_021` del padre.

## Puertas reproducidas

```bash
git show -s --format='commit=%H%nparent=%P%ntree=%T' 55d6c9420e1adc3c37499b7b236ac0cfc101d6d2
git merge-base --is-ancestor fea52f3ddf796991c93c85cae992ca695db1ae63 55d6c9420e1adc3c37499b7b236ac0cfc101d6d2
git diff --name-status fea52f3ddf796991c93c85cae992ca695db1ae63..55d6c9420e1adc3c37499b7b236ac0cfc101d6d2
git diff --numstat fea52f3ddf796991c93c85cae992ca695db1ae63..55d6c9420e1adc3c37499b7b236ac0cfc101d6d2
wc -lc docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_2026-08-14.md tools/o3c_p6_conductor/{conductor,fallo_durable}.sh
sha256sum docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_2026-08-14.md tools/o3c_p6_conductor/{conductor,fallo_durable}.sh
bash -n tools/o3c_p6_conductor/{conductor,fallo_durable}.sh
shellcheck tools/o3c_p6_conductor/{conductor,fallo_durable}.sh
tools/o3c_p6_conductor/fallo_durable.sh --autoprueba
git diff --check fea52f3ddf796991c93c85cae992ca695db1ae63..55d6c9420e1adc3c37499b7b236ac0cfc101d6d2
GOWORK=off go run github.com/zricethezav/gitleaks/v8@v8.30.0 git --no-banner --redact --log-opts='fea52f3ddf796991c93c85cae992ca695db1ae63..55d6c9420e1adc3c37499b7b236ac0cfc101d6d2'
GOWORK=off go run github.com/zricethezav/gitleaks/v8@v8.30.0 git --no-banner --redact --log-opts='5345d5d097b51ab3567983f048feabeceaf2957b..55d6c9420e1adc3c37499b7b236ac0cfc101d6d2'
```

`bash -n`, ShellCheck 0.11.0, autoprueba, checksums, modos y `diff --check`
terminaron verdes. Los cuatro mutantes exigidos murieron una sola vez: omitir
stdout devolvió 2; falsear `GO`, permitir sobrescritura y excluir raws del
manifiesto devolvieron 1. Gitleaks no encontró filtraciones: un commit y 10,21
KB en el rango focal; 25 commits y 199,42 KB en el acumulado desde `5345d5d`.

No se repitió la corrida O3c canónica ni se ejecutaron gates Go globales,
PostgreSQL, Docker, HTTP o E2E: el corte solo cambia Bash documental/conductor,
la evidencia canónica exacta ya estaba sellada y la orden prohíbe convertir
reintentos en mayoría.

## Hallazgos, límites y relevo

No se encontraron P0, P1 ni P2 dentro del contrato acotado. Permanece abierto,
sin mitigación ni reclasificación, el `NO-GO P1` de
`CAP_NORMAL_021` conservado por las revisiones funcional
`4bfb7b43069e20a0ccb1eaf6e2a80b2ff462a14f` y de seguridad
`29430b4ce8592e6e2bde74b69e37e44437091a77`. El corte mejora qué quedará
disponible ante el siguiente fallo; no diagnostica el rojo previo ni acredita
estabilidad.

El candidato puede pasar a revisión de seguridad independiente de este mismo
write-set. Dirección decidirá cualquier integración y conservará bloqueados
O3a V3/C21, toolchain, O4, publicación, CI y producción hasta que sus propios
prerrequisitos obtengan la acreditación exigida.
