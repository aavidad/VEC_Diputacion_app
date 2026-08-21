# Revisión funcional postcanónica O3c P6 CAP_NORMAL_021 runtime temporal V4

Fecha: 21 de agosto de 2026.

Tarea revisada: `O3C-P6-CAP021-RUNTIME-TMP-V4-FIXTURE-PASS`.

Dictamen: **GO funcional y de trazabilidad postcanónica**, con `P0=0`,
`P1=0` y `P2=0`.

No quedan bloqueadores funcionales ni de trazabilidad dentro del alcance
exacto de V4. El paquete canónico acredita `CAP_NORMAL_021`, su ejecución con
detector de carreras, las cien capturas por modo, las seis bifurcaciones
fatales directas y las 1.472 atestaciones de directorios temporales.

Este GO no revoca por sí mismo los rojos históricos más amplios de O3a, C21 o
el toolchain; no abre O4, integración, publicación externa, CI remota,
despliegue, producción ni métricas.

## Independencia, autoridades y write-set

La revisión se realizó de forma independiente y estrictamente de solo lectura
desde el worktree exclusivo:

```text
/srv/fabrica/revisiones/o3c-p6-cap021-runtime-tmp-v4-funcional-20260821
```

Rama:

```text
revision/o3c-p6-cap021-runtime-tmp-v4-funcional-20260821
```

Antes de redactar se revalidaron:

- rama exacta;
- commit y árbol exactos;
- cero cambios rastreados;
- cero cambios preparados;
- cero archivos no seguidos;
- ausencia previa del acta;
- identidad y limpieza del target inmutable;
- integridad actual del paquete canónico.

Se leyeron completos los dos `AGENTS.md` aplicables:

```text
/srv/fabrica/AGENTS.md
/srv/fabrica/revisiones/o3c-p6-cap021-runtime-tmp-v4-funcional-20260821/AGENTS.md
```

También se contrastaron como autoridades funcionales y de trazabilidad:

```text
docs/portal_vec/enmienda_o3c_p6_cap_normal_021_runtime_tmp_2026-08-21.md
tools/o3c_p6_conductor/conductor.sh
tools/o3c_p6_conductor/casos.tsv
tools/o3c_p6_conductor/fuentes.tsv
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff_test.go
docs/portal_vec/revisiones/revision_funcional_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md
docs/portal_vec/revisiones/revision_funcional_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md
docs/portal_vec/revisiones/revision_funcional_f0_h0b_c4b2_g2o_o3c_2026-08-11.md
```

El único write-set autorizado para materializar este dictamen es:

```text
docs/portal_vec/revisiones/revision_funcional_o3c_p6_cap_normal_021_runtime_tmp_v4_2026-08-21.md
```

No se modifican producto, conductores, casos, fuentes, fixtures, paquetes,
workflows, documentos transversales, métricas ni otras actas. El revisor no
integra ni aprueba su propio documento para publicación.

## Identidad exacta del candidato

| Propiedad | Valor reproducido |
| --- | --- |
| Commit candidato | `484020703c683c324e9b2eaef5c43a56c1d95ea6` |
| Padre directo | `7782e4679e546cde4d693633911d5ec3a47bf85b` |
| Árbol candidato | `408ff618672f8faab56cbc46f33f7233c380f5f4` |
| Asunto | `audit(o3c): corregir publicacion atomica V4` |
| Base V4 | `14c1f31079e466a82b8e1d390168078973cc6e05` |
| Árbol de la base | `1c736bcb555326841485a1f0c0486626ad6a5038` |
| Base antepasada del candidato | sí |
| Rutas modificadas por el candidato | dos |
| Target rastreado, preparado y no seguido | `0/0/0` |
| Worktree revisor rastreado, preparado y no seguido | `0/0/0` |

El commit modifica únicamente:

```text
M docs/portal_vec/enmienda_o3c_p6_cap_normal_021_runtime_tmp_2026-08-21.md
M tools/o3c_p6_conductor/conductor.sh
```

Huellas congeladas:

| Artefacto | Líneas | SHA-256 |
| --- | ---: | --- |
| Enmienda V4 | 799 | `490fa132c48a973860edfb604d65ecb48eb783dc36779a151bf6718a436151a4` |
| Conductor V4 | 800 | `8918ef3b2ba7b343ff95468a95483e676ad18991715a8fd048bc716de5cb15df` |
| Publicador durable | — | `b8f91102a2e98ce1e2e79ed73bfa9bd48c5d1f8ca002271dc5fc2461512bf174` |
| Matriz de 22 casos | 23 | `1e2d93e4f24d53c470699fc2751deaba5ef5132aa173e41a02709c03ef7d547f` |
| Ledger de 32 fuentes | 33 | `cd2b633cf7c787a58fa787fea8b3cf71e9d725388cb0795539f2851f2a49ee83` |

El target inmutable revisado fue:

```text
/srv/fabrica/orquesta/home/revisiones/o3c-p6-cap021-runtime-tmp-v4-4840207-target
```

Su `HEAD` y su árbol coinciden con el candidato. No presenta cambios
rastreados, preparados o no seguidos. Su repositorio no usa `alternates`,
`shallow` ni `commondir` redirigido; `info/grafts` está ausente o vacío y todos
los registros inspeccionados del índice conservan el marcador normal `H `.

## Paquete canónico

Paquete revisado:

```text
/srv/fabrica/orquesta/home/evidencias/o3c-p6-cap021-runtime-tmp-v4-4840207-canonica-r1
```

El directorio es real, propiedad `orquesta:orquesta`, UID efectivo `999`, modo
`0700` y contiene exactamente doce entradas, sin subdirectorios ni enlaces
simbólicos.

Las once entradas de datos son regulares, propiedad de `orquesta`, modo `0600`
y `nlink=1`. El ayudante `rename_noreplace` es regular, propiedad de
`orquesta`, modo `0700` y `nlink=1`.

Inventario exacto:

```text
SHA256SUMS
bf_directos.tsv
binarios.tsv
casos.tsv
contexto.tsv
fuentes.tsv
publicacion.tsv
rename_noreplace
residuos.txt
resumen.txt
tmpdir_selectores.tsv
utilidades.tsv
```

Huellas recalculadas:

| Artefacto | SHA-256 |
| --- | --- |
| `SHA256SUMS` | `4ebe2573983ed9edecc83f86f9e639507f39b013b9976778d3179edca02d1330` |
| `bf_directos.tsv` | `ef861710f7194a9b0ee4926fc6f22dc1f0c2ec9c012770e398e61de2f0f1a13b` |
| `binarios.tsv` | `5aaef81951ec3f96c5a5858c49930dbd477b9bdde101458c1ef9c0279efbafbd` |
| `casos.tsv` | `fb0236774afeb5d4384bf65151738687f80aaa251dd0a5e8deeb5a96b2156dbe` |
| `contexto.tsv` | `43ae7b55935f23125fe8d4d7f815a27e797d64a6519575cbe7499b3df5609121` |
| `fuentes.tsv` | `5dd85475d86e0287727c2dbf1830e48b6fa2b6c71c49b64e70e85d728f6c8cfe` |
| `publicacion.tsv` | `9ac56d4d43b8054914debaebfac0bd9a8b431700937d3ada5339b1ca996e785d` |
| `rename_noreplace` | `08b3ed3abefd234d249db0c73ab1016ad255b3f06958d6411660cf72e46a4173` |
| `residuos.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `resumen.txt` | `2b2fcab73880ae79ec6f2a0e60084e966d63fd14147c7456806b9e3a26ef2aae` |
| `tmpdir_selectores.tsv` | `6e4af8d125c98bc21bef5ca890175ebea92154dca5701f5ff52d3378c59c2149` |
| `utilidades.tsv` | `13af73948e08055c7536c22dcb0311a418af44d4931560976b11fc94423b8742` |

`sha256sum -c SHA256SUMS` validó las once entradas tanto como `root` como
mediante lectura bajo el usuario `orquesta`.

## Contexto y trazabilidad de fuentes

`contexto.tsv` registra la misma identidad en los tres cortes:

```text
head_inicial      = 484020703c683c324e9b2eaef5c43a56c1d95ea6
head_snapshot     = 484020703c683c324e9b2eaef5c43a56c1d95ea6
head_publicacion  = 484020703c683c324e9b2eaef5c43a56c1d95ea6

tree_inicial      = 408ff618672f8faab56cbc46f33f7233c380f5f4
tree_snapshot     = 408ff618672f8faab56cbc46f33f7233c380f5f4
tree_publicacion  = 408ff618672f8faab56cbc46f33f7233c380f5f4
```

Las tres comprobaciones de limpieza son `si`. El contexto fija además:

```text
euid=999
go_version=go version go1.26.5 linux/amd64
sha_conductor=8918ef3b2ba7b343ff95468a95483e676ad18991715a8fd048bc716de5cb15df
sha_matriz=1e2d93e4f24d53c470699fc2751deaba5ef5132aa173e41a02709c03ef7d547f
sha_fuentes=cd2b633cf7c787a58fa787fea8b3cf71e9d725388cb0795539f2851f2a49ee83
sha_target=5dd85475d86e0287727c2dbf1830e48b6fa2b6c71c49b64e70e85d728f6c8cfe
```

Las primeras 32 entradas de `fuentes.tsv` coinciden byte a byte con el ledger
del conductor. La entrada 33 es exactamente el fixture:

```text
deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
```

Su SHA-256 es:

```text
7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487
```

Las 33 rutas:

- son ficheros regulares y no enlaces simbólicos;
- están seguidas por Git en el commit exacto;
- coinciden con el hash declarado;
- se distribuyen en 32 entradas Git `100644` y el fixture `100755`;
- no contienen rutas duplicadas.

El hash del `fuentes.tsv` empaquetado,
`5dd85475d86e0287727c2dbf1830e48b6fa2b6c71c49b64e70e85d728f6c8cfe`,
es el `sha_target` incorporado en las 244 filas ordinarias y en las seis filas
fatales.

Los binarios construidos quedaron ligados mediante:

```text
normal b81841b35b0629d565ef5601a01d001a678993ffaa35d4c65192b941388891cb
race   826c2540136525083372f376863ac6dbc0dcb0b7d3d1c54cd38eda95d14af89e
```

## Matriz funcional reproducida

`casos.tsv` tiene treinta columnas y 244 registros de evidencia más cabecera.

Distribución exacta:

| Grupo | Normal | Carrera | Total |
| --- | ---: | ---: | ---: |
| Matriz base | 22 | 22 | 44 |
| Capturas `CAP_*` | 100 | 100 | 200 |
| Total | 122 | 122 | 244 |

Se comprobó que:

- cada uno de los 22 identificadores base aparece exactamente una vez por modo;
- comando y oráculo coinciden exactamente con `tools/o3c_p6_conductor/casos.tsv`;
- las secuencias `CAP_NORMAL_001..100` y `CAP_RACE_001..100` están completas;
- no hay identificadores duplicados o ajenos;
- las 244 filas terminan en `GO`;
- las 244 ejecuciones terminan con estado cero;
- las 244 salidas ordinarias miden exactamente cinco bytes;
- los 244 errores miden cero bytes;
- descriptor, hijos, zombis, grupos y temporales conservan inventarios iguales
  al inicio y al final;
- el directorio temporal efectivo empieza y termina vacío;
- el contenedor queda retirado;
- los descriptores ambientales quedan cerrados;
- los selectores esperados y obtenidos coinciden;
- la ausencia final del grupo de ejecución queda acreditada.

El conductor exacto no decide `GO` únicamente por el tamaño de la salida:
contrasta mediante `sha256sum --status` el contenido literal `PASS\n`, cuyo
SHA-256 es:

```text
c26de83abdc9496cd1301470918ec39ecca1cf389ef0ae1c6504da1800d1c431
```

El paquete no conserva el raw ordinario, pero cada fila `GO` queda causalmente
ligada a esa comprobación por el hash exacto del conductor registrado en
`contexto.tsv`.

## CAP_NORMAL_021 y carrera asociada

Las filas exactas son:

| Identificador | Modo | Estado | stdout | stderr | Duración | FD | Hijos | Zombis | Grupos | Temporales | Selectores | Resultado |
| --- | --- | ---: | ---: | ---: | ---: | --- | --- | --- | --- | --- | ---: | --- |
| `CAP_NORMAL_021` | normal | 0 | 5 | 0 | 987 ms | 7→7 | 0→0 | 0→0 | 0→0 | 0→0 | 7/7 | `GO` |
| `CAP_RACE_021` | race | 0 | 5 | 0 | 4.249 ms | 7→7 | 0→0 | 0→0 | 0→0 | 0→0 | 7/7 | `GO` |

Ambas ejecutan exclusivamente:

```text
testbin -test.run=^(TestHandoffO3cP5CasosAislados)$ -test.count=1
```

La prueba fuente exacta exige los siete selectores y sus estados:

| Selector | Estado esperado |
| --- | ---: |
| `positivo` | 0 |
| `retirada` | 0 |
| `retirada_terminal` | 0 |
| `reuso` | 65 |
| `particion` | 65 |
| `retirada_sin_ref` | 65 |
| `retirada_plazo` | 65 |

Las catorce filas de atestación pertenecientes a `CAP_NORMAL_021` y
`CAP_RACE_021` contienen exactamente esos siete nombres por modo. Todas
acreditan:

```text
uid=999
modo_dir=700
entradas_inicio=0
identidad_pre_limpieza=true
residuos_pre_limpieza=1
lstat_enoent=true
raiz_exterior_vacia=true
retirada=true
resultado=GO
```

El test espera el proceso hijo, comprueba el estado correspondiente, contiene
descendientes, preserva el preconteo, retira el directorio selector, acredita
`Lstat=ENOENT` y exige que la raíz exterior quede vacía. Los caminos verdes
conservan sus `os.Exit(0)` y los fatales el estado 65 previsto.

Por ello el paquete acredita específicamente el defecto histórico observado en
`CAP_NORMAL_021` dentro del alcance temporal V4, y reproduce el mismo criterio
con el binario construido con detector de carreras.

## Bifurcaciones fatales directas

`bf_directos.tsv` tiene 33 columnas y seis registros:

| Caso | Normal | Carrera | Estado esperado |
| --- | ---: | ---: | ---: |
| `C01_BF_AUTO` | 1 | 1 | 65 |
| `C08_BF_LEASE` | 1 | 1 | 65 |
| `C17_BF_PARTICION` | 1 | 1 | 65 |

Las seis filas conservan conjuntamente:

```text
estado=65
stdout_bytes=0
stderr_bytes=0
stdout_eof=si
stderr_eof=si
no_retorno=si
resultado=GO
```

Inventarios, directorio temporal, contenedor, descriptores ambientales y grupo
de ejecución terminan acreditados. `C17_BF_PARTICION` aporta un selector
`particion` por modo, ambos `GO`.

## Atestaciones temporales

`tmpdir_selectores.tsv` contiene catorce columnas y 1.472 registros más
cabecera.

La suma independiente es:

```text
selectores declarados por casos ordinarios = 1470
selectores declarados por BF directos       =    2
total esperado                              = 1472
total obtenido                              = 1472
```

No hay ejecuciones desajustadas, filas huérfanas ni claves
`identificador/modo/selector` duplicadas.

Distribución exacta:

| Selector | Filas |
| --- | ---: |
| `positivo` | 210 |
| `retirada` | 210 |
| `retirada_terminal` | 210 |
| `reuso` | 210 |
| `particion` | 212 |
| `retirada_sin_ref` | 210 |
| `retirada_plazo` | 210 |
| Total | 1.472 |

Las dos filas adicionales de `particion` pertenecen a
`C17_BF_PARTICION`, una normal y otra con detector de carreras.

Todas las filas conservan `GO`, UID 999, modo 0700, nacimiento vacío, identidad
previa conservada, `Lstat=ENOENT`, raíz exterior vacía y retirada acreditada.

## Publicación y limpieza

`publicacion.tsv` declara:

```text
metodo=renameat2_RENAME_NOREPLACE
sha_helper_noreplace=08b3ed3abefd234d249db0c73ab1016ad255b3f06958d6411660cf72e46a4173
```

La identidad física declarada del origen publicado es:

```text
2049:2267315:999:700
```

La identidad recalculada del paquete canónico es exactamente la misma:

```text
2049:2267315:999:700
```

Las postcondiciones declaradas son:

```text
origen_ausente
destino_real
identidad_conservada
entradas_uid_modo_nlink1
codigos_cerrados
```

No quedan temporales hermanos de publicación. Los temporales propios
observados durante la corrida están ausentes:

```text
/var/tmp/o3c-p6.NJGSHV
/var/tmp/o3c-p6-tools.OFl6xW
```

Los procesos padre y conductor de la corrida, PID 213161 y PID 213175, están
ausentes. No se encontró una conducción canónica activa sobre el target. El
agente Codex productor permanece como proceso independiente, pero no ejecuta
el conductor canónico.

`residuos.txt` tiene cero bytes. `resumen.txt` conserva conjuntamente:

```text
resultado=GO
casos_totales=244
bf_directos=6
capturas_normal=100
capturas_race=100
tmpdir_selectores=1472
snapshot_revalidado_post_ejecucion=33
target_head_tree_limpieza_revalidados=si
tmpdir_exacto_inicio_preconteo_fin_cero=si
contenedor_runtime_retirado=si
fd_ambiente_cerrado=si
publicacion_go_no_replace=si
residuos=cero
staging_retirado=si
```

## Hallazgos

No se encontraron hallazgos funcionales o de trazabilidad pendientes en el
alcance exacto de esta revisión:

```text
P0=0
P1=0
P2=0
```

En particular:

- no existe divergencia entre commit, árbol, target y contexto;
- no existe divergencia entre ledger y las 33 rutas atestadas;
- no existen sumas incorrectas;
- no existen filas `NO-GO`;
- no faltan casos, capturas, bifurcaciones o selectores;
- no existen selectores huérfanos o duplicados;
- `CAP_NORMAL_021` y `CAP_RACE_021` cumplen el mismo contrato;
- la matriz base coincide exactamente con los 22 oráculos;
- la publicación conserva la identidad declarada;
- no quedan residuos propios de la corrida.

## Incidencias no materiales de la auditoría

Dos primeras consultas auxiliares no aportaron evidencia y fueron repetidas
correctamente:

1. un `printf` usado únicamente para separar visualmente se interpretó como
   opción y terminó con estado 2 después de que las sumas y metadatos ya se
   hubieran leído;
2. una primera expresión `awk` no alcanzó el servidor por un error local de
   comillas.

Ninguna de las dos consultas escribió archivos, ejecutó producto o modificó
paquete, target, índices, permisos o procesos. Las comprobaciones se repitieron
con comandos corregidos y la revalidación final terminó `GO`.

## Comandos de solo lectura reproducidos

Las comprobaciones principales se realizaron por SSH contra
`root@cidonia.cloud`. Se usó `GIT_OPTIONAL_LOCKS=0` para evitar actualizaciones
opcionales del índice.

```bash
realpath -e "$WORKTREE"
stat -c '%U:%G %a %F' "$WORKTREE"

GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$WORKTREE" \
  -C "$WORKTREE" branch --show-current
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$WORKTREE" \
  -C "$WORKTREE" rev-parse HEAD HEAD^{tree}
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$WORKTREE" \
  -C "$WORKTREE" diff-files --name-only
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$WORKTREE" \
  -C "$WORKTREE" diff-index --cached --name-only HEAD --
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$WORKTREE" \
  -C "$WORKTREE" ls-files --others --exclude-standard

find "$WORKTREE/docs/portal_vec/revisiones" -type f -name AGENTS.md -print
wc -l -c /srv/fabrica/AGENTS.md "$WORKTREE/AGENTS.md"
cat /srv/fabrica/AGENTS.md "$WORKTREE/AGENTS.md"

sha256sum \
  "$WORKTREE/docs/portal_vec/enmienda_o3c_p6_cap_normal_021_runtime_tmp_2026-08-21.md" \
  "$WORKTREE/tools/o3c_p6_conductor/conductor.sh"

GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$TARGET" \
  -C "$TARGET" rev-parse HEAD HEAD^ HEAD^{tree}
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$TARGET" \
  -C "$TARGET" merge-base --is-ancestor \
  14c1f31079e466a82b8e1d390168078973cc6e05 HEAD
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$TARGET" \
  -C "$TARGET" diff-tree --no-commit-id --name-status -r HEAD
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$TARGET" \
  -C "$TARGET" diff-files --quiet
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$TARGET" \
  -C "$TARGET" diff-index --cached --quiet HEAD --
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$TARGET" \
  -C "$TARGET" ls-files --others --exclude-standard

find "$PAQUETE" -mindepth 1 -maxdepth 1 \
  -printf '%f\t%y\t%u:%g\t%m\t%n\t%s\n'
stat -c '%d:%i:%u:%a:%h:%F' "$PAQUETE"
sha256sum "$PAQUETE/SHA256SUMS"

(
  cd "$PAQUETE"
  sha256sum -c SHA256SUMS
)

/usr/sbin/runuser -u orquesta -- /bin/sh -c \
  'cd "$1" && sha256sum -c SHA256SUMS' sh "$PAQUETE"

wc -l -c "$PAQUETE"/*
head -n 4 "$PAQUETE/casos.tsv"
tail -n 4 "$PAQUETE/casos.tsv"
head -n 4 "$PAQUETE/bf_directos.tsv"
tail -n 4 "$PAQUETE/bf_directos.tsv"
head -n 4 "$PAQUETE/tmpdir_selectores.tsv"
tail -n 4 "$PAQUETE/tmpdir_selectores.tsv"

awk -F '\t' '...' \
  "$PAQUETE/casos.tsv" \
  "$PAQUETE/bf_directos.tsv" \
  "$PAQUETE/tmpdir_selectores.tsv"

cmp -s \
  <(head -n 33 "$PAQUETE/fuentes.tsv") \
  "$TARGET/tools/o3c_p6_conductor/fuentes.tsv"

while IFS=$'\t' read -r sha ruta; do
  test "$sha" = sha256 && continue
  test -f "$TARGET/$ruta"
  test ! -L "$TARGET/$ruta"
  sha256sum "$TARGET/$ruta"
  GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$TARGET" \
    -C "$TARGET" ls-files --error-unmatch -- "$ruta"
done < "$PAQUETE/fuentes.tsv"

stat -c '%d:%i:%u:%a' "$PAQUETE"
awk -F '\t' 'NR==2 {print $2}' "$PAQUETE/publicacion.tsv"
wc -c "$PAQUETE/residuos.txt"

test ! -e /var/tmp/o3c-p6.NJGSHV
test ! -e /var/tmp/o3c-p6-tools.OFl6xW
test ! -d /proc/213161
test ! -d /proc/213175
```

No se ejecutaron comandos de escritura, `fetch`, `pull`, `reset`, `push`,
merge, rebase, despliegue, cambios de permisos, señales, eliminación de
procesos, Docker, PostgreSQL, red de producto ni gates globales.

## Pruebas omitidas y motivo

No se repitieron la conducción canónica, las 244 ejecuciones, los builds normal
y race ni los fallos directos. El contrato exige una única conducción por SHA y
destino; repetirla invalidaría la disciplina de evidencia consumida y no es
necesario para una revisión postcanónica de solo lectura.

Tampoco se ejecutaron:

```text
go test ./...
go test -race ./...
go vet ./...
scripts/verificar_calidad.sh
Docker
PostgreSQL
HTTP
E2E
CI remota
```

Son gates desproporcionados para esta acta documental, no pueden mejorar la
trazabilidad del paquete ya sellado y no están autorizados por el corte.

## Límites y riesgos conservados

El GO queda limitado a
`O3C-P6-CAP021-RUNTIME-TMP-V4-FIXTURE-PASS`.

Permanecen expresamente fuera de alcance:

1. revocar de forma global los dos `NO-GO` históricos de O3A-V5-CND-V3;
2. acreditar estabilidad estadística de C21;
3. cerrar el parche o la confianza completa del toolchain;
4. abrir O4, `Start`, mapa de descriptores o fases posteriores;
5. acreditar Docker, PostgreSQL, SQL, red, HTTP, E2E o producción;
6. cambiar métricas o estado transversal;
7. autorizar integración, push, publicación externa o despliegue.

V4 opera bajo la autoridad cooperativa del usuario Unix `orquesta`. No protege
frente a otro proceso que ya ejecute con el mismo UID 999 y pueda modificar
objetos accesibles a ese usuario.

El build con detector de carreras fija siete ejecutables de la frontera C,
pero no atestigua transitivamente cabeceras, objetos, bibliotecas, cargador ni
el sistema raíz C del host. El paquete declara correctamente:

```text
race_c_tcb=ejecutables_7_fijados_sysroot_host_no_atestado
```

La precondición del lanzador confiable anterior al primer `exec` tampoco queda
atestada por V4. En particular, el conductor no puede neutralizar
retroactivamente lo que un cargador dinámico hubiera consumido antes de iniciar
Bash.

El paquete conserva los hashes de los binarios normal y race, pero no incorpora
los binarios como entradas publicadas. La correspondencia queda ligada por la
ejecución del conductor exacto, el manifiesto de fuentes, el contexto y las
filas selladas; esta acta no afirma una reproducción independiente de aquellos
binarios.

Estos límites están declarados por la enmienda y no constituyen bloqueadores
nuevos del criterio funcional V4.

## Relevo

El candidato exacto
`484020703c683c324e9b2eaef5c43a56c1d95ea6` y el paquete canónico exacto reciben:

```text
GO funcional y de trazabilidad postcanónica
P0=0
P1=0
P2=0
```

Este dictamen acredita el objetivo temporal V4 de `CAP_NORMAL_021` y su carrera
asociada. No constituye autorización de integración ni publicación externa.

Dirección debe:

1. materializar esta acta como único archivo del write-set;
2. revalidar que el commit resultante solo incorpora el acta;
3. comprobar separadamente el dictamen postcanónico de seguridad sobre el mismo
   commit, árbol y paquete;
4. decidir mediante orden expresa cualquier integración posterior.

El productor no se autoaprueba y esta revisión no modifica porcentajes,
documentos transversales ni producción.

## Entrega

```text
Tarea:
O3C-P6-CAP021-RUNTIME-TMP-V4-FIXTURE-PASS — revisión funcional postcanónica

Estado:
GO funcional y de trazabilidad; P0=0, P1=0, P2=0

Commit revisado:
484020703c683c324e9b2eaef5c43a56c1d95ea6

Árbol revisado:
408ff618672f8faab56cbc46f33f7233c380f5f4

Archivo de revisión:
docs/portal_vec/revisiones/revision_funcional_o3c_p6_cap_normal_021_runtime_tmp_v4_2026-08-21.md

Resultado:
244/244 casos, 6/6 BF directos y 1472/1472 atestaciones GO;
CAP_NORMAL_021 y CAP_RACE_021 acreditados.

Pruebas ejecutadas:
Identidad Git, limpieza, genealogía, inventario, sumas SHA-256, consistencia
tabular, correspondencia de fuentes, publicación y residuos; todo en lectura.

Pruebas omitidas:
Conducción y gates pesados, porque el SHA y destino canónicos están consumidos
y la revisión es postcanónica de solo lectura.

Seguridad, privacidad, i18n y accesibilidad:
Sin cambios de producto, datos, interfaz o traducciones. La seguridad
postcanónica conserva dictamen independiente separado.

Limitaciones:
Autoridad cooperativa UID 999, frontera C no atestada transitivamente,
precondición externa del lanzador y ausencia de autorización de integración.

Riesgos:
Ningún P0, P1 o P2 funcional o de trazabilidad dentro de V4. Permanecen los
límites históricos externos al corte.

Siguiente tarea desbloqueada:
Materialización documental y comprobación separada del dictamen postcanónico
de seguridad. Integración cerrada hasta orden expresa de dirección.

Revisión independiente:
GO funcional y de trazabilidad; P0=0, P1=0, P2=0.
```
