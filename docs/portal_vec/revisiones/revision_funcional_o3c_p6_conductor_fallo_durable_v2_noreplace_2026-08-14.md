# Revisión funcional O3c P6 V2: publicación sin reemplazo

Fecha: 14 de agosto de 2026.

Tarea revisada: `O3C-P6-CONDUCTOR-FALLO-DURABLE-V2-NOREPLACE`.

Dictamen: **GO funcional**, con `P0=0`, `P1=0` y `P2=0`, limitado a la
corrección defensiva del publicador de evidencia. El hallazgo P1 de V1 queda
cerrado para el candidato y el entorno exactos revisados. Este GO no revoca
`CAP_NORMAL_021`, no acredita O3a V3/C21 o toolchain, y no abre O4,
publicación remota, CI ni producción.

## Identidad, independencia y write-set

Se revisó el candidato exacto
`9391243f98d232a8eb78ed20ba5306e31b864bad`, con padre único
`55d6c9420e1adc3c37499b7b236ac0cfc101d6d2` y árbol
`6aa08b678c1f4ee268b57ba2872c4586fc04085e`. El rango contiene un commit,
cero merges y solamente:

```text
A docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md
M tools/o3c_p6_conductor/fallo_durable.sh
```

El delta es `+159/-10`: `+113` en la enmienda y `+46/-10` en el helper. Sus
modos son `100644` y `100755`. `tools/o3c_p6_conductor/conductor.sh` conserva
el blob `66390171c97e99dab9c47e78a605b18bce94f23d`, byte a byte idéntico al
padre. La rama productora y el checkout revisor estaban limpios antes de
documentar; no se modificó el candidato.

El único write-set de esta revisión es esta acta. Antes de editar se leyeron
completos `AGENTS.md`, relevo de sesión, mapa, tablero, relevo de contratación
temporal, expediente RRHH, roadmap, matriz normativa, decisiones O3a/O3b/O3c,
ledger final O3b, ledger y revisiones finales O3c, las dos actas V1 y ambas
enmiendas del fallo durable.

## Huellas y límites

| Ruta | Líneas | Bytes | SHA-256 |
| --- | ---: | ---: | --- |
| `docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md` | 113 | 5.437 | `abf76b1f72424844571791fbb82d65ea892eeb872c5fd019c569663a3abd316d` |
| `tools/o3c_p6_conductor/fallo_durable.sh` | 130 | 5.368 | `b8f91102a2e98ce1e2e79ed73bfa9bd48c5d1f8ca002271dc5fc2461512bf174` |
| `tools/o3c_p6_conductor/conductor.sh` | 182 | 10.710 | `6f777f59d4e157b2c4eabb13f66789cfe11b9c01f646e9512af9af388b715538` |

Helper y conductor quedan por debajo de DEC-051. No cambian fuentes Go,
oráculos, plazos, cardinalidades, inventarios, matrices, ledgers, workflows,
Docker, PostgreSQL, estado transversal o métricas. El DAG conserva P6 como
herramienta/evidencia externa y no abre P7, O4 ni una ruta productiva.

## Cierre requisito por requisito del P1 V1

La revisión de seguridad V1
`c0c6242a22a0e3b8aad074bac5f4472c96b6f6e3` demostró que el `mv` anterior
podía anidar el staging a través de un enlace insertado entre el precheck y la
publicación. La V2 elimina esa semántica:

1. `publicar` exige origen y padre directorios reales, destino ausente incluso
   como enlace, y crea el staging mediante `mktemp` en el mismo padre del
   destino;
2. `mv -n -T` trata el destino como entrada exacta y solicita no reemplazar;
3. antes del rename se fija `(st_dev, st_ino)` del staging;
4. un cero de `mv -n` no basta: el origen debe desaparecer, el destino debe
   existir como directorio no enlace y conservar exactamente aquella
   identidad;
5. colisión, enlace, anidamiento, ausencia o identidad divergente retornan
   error; `set -e` impide que el conductor los transforme en `GO`;
6. `temporal_publicacion` solo se desarma después de validar la identidad; en
   el rechazo concurrente observado, el trap eliminó el staging restante;
7. el origen aportado al publicador nunca se mueve: se copia al staging y
   permaneció intacto tanto en éxito como en todos los rechazos;
8. `SHA256SUMS` se construye dentro del staging con nombres relativos,
   ordenados y enumerados con NUL, sin interpolar una ruta en `sed`, y se
   valida antes de publicar;
9. el paquete exitoso conserva directorio `0700`, ficheros `0600`, raws byte a
   byte y manifiesto completo.

En GNU coreutils 9.7 sobre el Linux revisado, `strace` confirmó tres llamadas
relevantes: el éxito ejecutó
`renameat2(..., RENAME_NOREPLACE) = 0`; las colisiones con directorio y enlace
ejecutaron la misma primitiva y obtuvieron `EEXIST`. No se observó una ruta de
publicación sin garantía atómica de no reemplazo.

## Sintéticos funcionales propios

Se realizaron una sola vez y exclusivamente bajo
`/srv/fabrica/revisiones`:

- publicación nominal: estado cero, staging y destino conservaron la misma
  identidad `2049:8075411`, el origen siguió intacto, checksums relativos
  verdes, `0700/0600` y ambos raws exactos;
- intercalación controlada del antiguo P1: después de observar el staging se
  insertó un enlace en el destino; el helper devolvió 2, conservó el enlace y
  el origen, dejó vacío el directorio apuntado y eliminó todo staging
  residual, con stdout/stderr cero;
- destino directorio previo: estado 2, inode y guardia inmutables, sin
  contenido añadido y origen preservado;
- destino enlace previo: estado 2, enlace intacto, objetivo vacío y origen
  preservado;
- padre enlace: estado 2, enlace intacto, objetivo vacío y origen
  preservado.

Las evidencias quedaron fuera de Git en los directorios
`o3c-p6-v2-exito-funcional.lK1o5k`,
`o3c-p6-v2-race-funcional.IT39xy`,
`o3c-p6-v2-rechazos-funcional.KcmDGN` y
`o3c-p6-v2-identidad-funcional.TDonLt` bajo esa raíz. Solo contienen datos
sintéticos.

## Mutantes e integraciones entregadas

El manifiesto de mutantes validó completo. Su `resultados.tsv`, SHA-256
`318f6e2ea908c90221df61829e3f8155491e91a02ac96103b5a84e5ef781d9d5`,
registra base verde y cinco mutantes muertos. La reproducción funcional única
obtuvo:

```text
m1_sustituye=1
m2_anida=1
m3_confia_mv=1
m4_sin_identidad: helper base=2, mutante=0
m5_sin_raw_sumas=1
```

Así se distinguen sustitución, anidamiento, confianza exclusiva en el estado
de `mv`, omisión de identidad y exclusión de los raws del manifiesto. Los cinco
scripts superaron `bash -n` antes del oráculo; una rotura sintáctica no se
contabilizó como muerte.

Los paquetes sintéticos entregados para caso ordinario y BF validaron todos
sus checksums, publicaron en la ruta exacta, conservaron `NO-GO`, estado uno,
inventarios iguales, PGID ausente y salida no cero del conductor. Sus
`SHA256SUMS` son, respectivamente:

```text
0bf790fde24608d1988b8ad844bd3071f51c1c9f2316bb0b847a44b37a9a7104
fce51de9c3f4007f60c726173a2f5369f56695baa1b51f85b10e89a7862c3834
```

Ambos paquetes tienen `0700/0600` y ligan el helper V2 exacto. No se usaron
para atribuir estabilidad al producto.

## Evidencia canónica sellada

No se repitió O3c. Se verificó la evidencia única ya sellada en
`/srv/fabrica/revisiones/evidencia-o3c-p6-fallo-durable-v2-9391243/r1`,
propiedad `orquesta:orquesta`. `sha256sum -c SHA256SUMS` validó sus siete
ficheros:

```text
resumen.txt     c7b80b882938d0a7492cfc6256cd1e65d8598f1c89ea007f9a37175515356089
contexto.tsv    c78e2554651f09e0e6e23c2a71435ea3d70488b4b09f36f988eba7ad0be5b57b
casos.tsv       f5e682cdcb08899fa443c828d5b2c5bc4d1e55966669e046131b38c35f13940a
bf_directos.tsv 28ccd3cde5516ff2d72dc0ec0dc3b609592bd85a5c15c3a0278c4277de33e6f0
SHA256SUMS      865ac3ce603eb460907aa59b7bf6ea06dfa415a0f6616d1331040a9700c5ea45
```

El contexto liga HEAD `9391243f…`, Go 1.26.5 y los hashes exactos del
conductor y helper. Hay 244/244 casos verdes, 122 normal y 122 race, 100+100
capturas, seis/seis BF en 65/EOF/0/0, inventarios iguales y residuos cero. El
stdout exterior fue `GO\n` y stderr cero. Este resultado acredita solo que el
helper V2 no altera el camino exitoso en esa ejecución; no compensa el rojo
histórico.

## Puertas reproducidas

```bash
git show -s --format='commit=%H%nparent=%P%ntree=%T' 9391243f98d232a8eb78ed20ba5306e31b864bad
git merge-base --is-ancestor 55d6c9420e1adc3c37499b7b236ac0cfc101d6d2 9391243f98d232a8eb78ed20ba5306e31b864bad
git diff --name-status 55d6c9420e1adc3c37499b7b236ac0cfc101d6d2..9391243f98d232a8eb78ed20ba5306e31b864bad
git diff --numstat 55d6c9420e1adc3c37499b7b236ac0cfc101d6d2..9391243f98d232a8eb78ed20ba5306e31b864bad
cmp -s <(git show 55d6c9420e1adc3c37499b7b236ac0cfc101d6d2:tools/o3c_p6_conductor/conductor.sh) tools/o3c_p6_conductor/conductor.sh
bash -n tools/o3c_p6_conductor/{conductor,fallo_durable}.sh
shellcheck -x tools/o3c_p6_conductor/{conductor,fallo_durable}.sh
tools/o3c_p6_conductor/fallo_durable.sh --autoprueba
strace -f -e trace=renameat2 tools/o3c_p6_conductor/fallo_durable.sh --autoprueba
git diff --check 55d6c9420e1adc3c37499b7b236ac0cfc101d6d2..9391243f98d232a8eb78ed20ba5306e31b864bad
GOWORK=off go run github.com/zricethezav/gitleaks/v8@v8.30.0 git --no-banner --redact --log-opts='55d6c9420e1adc3c37499b7b236ac0cfc101d6d2..9391243f98d232a8eb78ed20ba5306e31b864bad'
GOWORK=off go run github.com/zricethezav/gitleaks/v8@v8.30.0 git --no-banner --redact --log-opts='5345d5d097b51ab3567983f048feabeceaf2957b..9391243f98d232a8eb78ed20ba5306e31b864bad'
```

`bash -n`, ShellCheck 0.11.0, autoprueba, mutantes, sintéticos, checksums,
modos, genealogía y `diff --check` terminaron verdes. Gitleaks encontró cero
fugas: un commit y 7,49 KB en el rango V2; 26 commits y 206,91 KB en el
acumulado desde `5345d5d`.

No se ejecutaron Go global/race, PostgreSQL, Docker, HTTP o E2E, ni se repitió
el conductor canónico: el delta es Bash test-only/documentación y la corrida
canónica única ya estaba sellada.

## Hallazgos, límites y relevo

No quedan hallazgos P0, P1 o P2 en el alcance V2. El dictamen se limita al
Linux observado, donde GNU coreutils 9.7 materializó efectivamente
`RENAME_NOREPLACE`; otro entorno debe reproducir esta puerta y no hereda el GO
por nombre de opción. No se atribuye protección frente a una mutación posterior
al instante linealizado por un actor con autoridad sobre el directorio padre.

Permanecen abiertos, sin mitigación ni reclasificación,
`CAP_NORMAL_021`, los dos `NO-GO` de O3a V3 y el P1 C21/toolchain. La revisión
de seguridad independiente debe decidir sobre estos mismos bytes; solo
dirección puede integrar o cambiar el estado.
