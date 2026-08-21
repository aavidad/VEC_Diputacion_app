# Revisión funcional O3a V5 CND V3 C21: cierre de la puerta O3c mediante V4

Fecha: 21 de agosto de 2026.

Tarea:
`O3A-V5-CND-V3-C21-INVENTARIO-EXACTO-CIERRE-PUERTA-O3C-V4`.

Dictamen: **GO funcional y de trazabilidad**, con:

```text
P0=0
P1=0
P2=0
```

P0 designa un hallazgo crítico, P1 uno mayor y P2 uno menor.

La evidencia O3c V4 satisface prospectivamente la única puerta O3c que dejó
rojo el candidato V3. El candidato histórico
`fea52f3ddf796991c93c85cae992ca695db1ae63` conserva íntegramente su
`NO-GO`: no se borra, repite, compensa ni reclasifica su ejecución fallida.

Este dictamen no acredita estabilidad global de C21, no cierra la confianza
completa del conjunto de herramientas, no abre O4 y no autoriza integración,
publicación externa, envío a GitHub, despliegue, producción, credenciales,
datos reales o cambio de métricas.

## Pregunta de cierre y respuesta exacta

La enmienda V3 exigía que O3c ejecutase su conductor porque compartía G7a y su
inventario vivo de fuentes. La única ejecución O3c de aquel candidato terminó
roja en:

```text
CAP_NORMAL_021
modo=normal
estado=1
stdout=197
stderr=0
grupo=si
inventario=5/5,0/0,0/0,0/0,0/0
```

El conductor histórico eliminó su staging y no conservó los 197 bytes. Por
ello el resultado demostraba inequívocamente una puerta roja, pero no permitía
atribuir su causa.

La nueva evidencia V4 demuestra, sobre un commit posterior exacto, que:

- `CAP_NORMAL_021` termina con estado cero;
- su salida estándar es el literal exacto `PASS\n`, cinco bytes;
- su error estándar está vacío;
- los siete selectores cumplen sus estados contractuales;
- sus descriptores de archivo, hijos, zombis, grupos y temporales no presentan
  delta;
- el mismo contrato queda acreditado con detector de carreras;
- el paquete resultante queda ligado al ledger O3c y a las fuentes exactas;
- la publicación local es íntegra, privada, atómica y sin reemplazo.

Por tanto V4 cierra la dependencia O3c para un nuevo cierre documental de V3.
No convierte la ejecución de `fea52f3` en verde ni demuestra que aquel fallo
tuviera exactamente la causa corregida por V4.

## Independencia y modo de trabajo

La revisión se realizó en modo estrictamente de solo lectura en
`root@cidonia.cloud`, desde el worktree exclusivo:

```text
/srv/fabrica/revisiones/o3a-v5-cnd-v3-c21-cierre-o3c-v4-funcional-20260821
```

Rama:

```text
revision/o3a-v5-cnd-v3-c21-cierre-o3c-v4-funcional-20260821
```

Estado observado antes y después:

```text
HEAD=484020703c683c324e9b2eaef5c43a56c1d95ea6
tree=408ff618672f8faab56cbc46f33f7233c380f5f4
cambios rastreados=0
cambios preparados=0
archivos no seguidos=0
```

El único write-set futuro autorizado para materializar este dictamen es:

```text
docs/portal_vec/revisiones/revision_funcional_o3a_v5_cnd_v3_c21_inventario_exacto_cierre_o3c_v4_2026-08-21.md
```

Durante esta revisión no se creó ni modificó ese archivo. No se editaron
producto, conductores, fuentes, ledgers, fixtures, autoridades, evidencias,
permisos, ramas, índices, procesos, estado transversal o métricas.

## Autoridades leídas

Se leyeron completos los dos `AGENTS.md` aplicables:

```text
/srv/fabrica/AGENTS.md
AGENTS.md del worktree revisor
```

También se leyeron completos los documentos obligatorios de dirección:

```text
docs/portal_vec/relevo_sesion_2026-07-29_inicio_ct47.md
docs/portal_vec/mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md
docs/portal_vec/tablero_tareas_contratacion_temporal_2026-07-23.md
docs/portal_vec/relevo_contratacion_temporal_2026-07-23.md
docs/portal_vec/expediente_contratacion_temporal_rrhh.md
docs/portal_vec/objetivos_y_hoja_ruta_rrhh_2026-07-23.md
docs/portal_vec/matriz_normativa_contratacion_temporal_2026-07-23.md
```

Las autoridades materiales específicas contrastadas fueron:

```text
docs/portal_vec/decision_f0_h0b_c4b2_g2o_o3c_continuacion_salida_2026-08-11.md
docs/portal_vec/enmienda_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md
docs/portal_vec/revisiones/revision_funcional_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md
docs/portal_vec/revisiones/revision_seguridad_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md
docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_2026-08-14.md
docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md
docs/portal_vec/revisiones/revision_funcional_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md
docs/portal_vec/revisiones/revision_seguridad_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md
docs/portal_vec/enmienda_o3c_p6_cap_normal_021_runtime_tmp_2026-08-21.md
tools/o3c_p6_conductor/conductor.sh
tools/o3c_p6_conductor/casos.tsv
tools/o3c_p6_conductor/fuentes.tsv
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff_test.go
```

Se leyeron además, desde sus objetos Git exactos, las actas postcanónicas
funcional y de seguridad de V4.

## Identidades Git

### Candidato histórico V3

```text
commit=fea52f3ddf796991c93c85cae992ca695db1ae63
parent=ec02febaa3080cb8e7894340075beb850ea54348
tree=c9d5a393f39d48a5015848d5fd49a0b69959c622
```

Su delta histórico contiene cinco rutas: G7a, la enmienda V3 y los tres
ledgers O3a, O3b y O3c.

### Base exacta de esta revisión

```text
commit=484020703c683c324e9b2eaef5c43a56c1d95ea6
parent=7782e4679e546cde4d693633911d5ec3a47bf85b
tree=408ff618672f8faab56cbc46f33f7233c380f5f4
subject=audit(o3c): corregir publicacion atomica V4
```

`fea52f3` es antepasado de `4840207`. Esta genealogía permite contrastar qué
autoridades V3 permanecieron inmóviles y qué cambio O3c posterior queda
cubierto por V4.

## Revisión de los commits postcanónicos

Las dos revisiones V4 son commits hermanos e independientes, ambos hijos
directos de la base exacta.

| Revisión | Commit | Padre | Árbol | Write-set |
| --- | --- | --- | --- | --- |
| Funcional y trazabilidad | `bb23278e0b5660b611cb9c7faa19b197e0283a19` | `484020703c683c324e9b2eaef5c43a56c1d95ea6` | `fb1867ae0adea947ed6dd65b3b6588fd06af2442` | una única acta nueva |
| Seguridad | `22fde5f72b2ce4bf92c2ae32bee4d7ab47a4e36b` | `484020703c683c324e9b2eaef5c43a56c1d95ea6` | `1cfe901ac1b37f6e6ddb1dd3c66cb459632ec6c7` | una única acta nueva |

Acta funcional:

```text
docs/portal_vec/revisiones/revision_funcional_o3c_p6_cap_normal_021_runtime_tmp_v4_2026-08-21.md
líneas=722
bytes=23851
SHA-256=0770193993646b7c20e141be72a9bdc8c4abd7ce71017e09dc73a70d6ce6a06e
```

Acta de seguridad:

```text
docs/portal_vec/revisiones/revision_seguridad_o3c_p6_cap_normal_021_runtime_tmp_v4_2026-08-21.md
líneas=549
bytes=22285
SHA-256=551e91ea4fb52608c18482036b863b4279cb5fb61ff12ceae01584c9200ed1a4
```

No modifican producto, conductor, ledgers, fixtures o evidencia canónica. Sus
dictámenes coinciden en `GO`, sin bloqueadores dentro del alcance V4, y
conservan expresamente las fronteras históricas.

## Autoridades V3 que permanecen byte a byte

Se compararon los objetos de `fea52f3` y `4840207`.

| Autoridad | Identidad conservada |
| --- | --- |
| G7a | blob `a5d987dc5afe7dc3ce09e4ea7c16f55e8c22b28c`, 750 líneas, 22.607 bytes, SHA-256 `2d5fc67e6aff0f6e11305e9d92690452f7ffb194b4bcef22cd2dd9121235bbb3` |
| Ledger O3a | blob `8e3d62a9f686e87525c81a4222aa24f445bf1dc6`, 11 líneas, SHA-256 `ba2b0a1c9838f57ca43d53ec6133ae74bfa7452f7511421e367d7fc5f3d079b0` |
| Ledger O3b | blob `4be981d323590693edece554739f8f184be92071`, 23 líneas, SHA-256 `238e019a8fe31c7de830e174a4308083452ef22ceb54999ca4a8aeff5aac5f4f` |
| Enmienda V3 | blob `2d5f18d3f3d377bfc917148cf0c6e1286e437dca`, 103 líneas, SHA-256 `10e1eda8a6a741326be9fb53c2566b655ae316b4af819c224f7b4f368a8a2753` |

La validación de todas las entradas contra el árbol actual dio:

```text
O3a: 10/10 fuentes, cero discrepancias de líneas o SHA-256
O3b: 22/22 fuentes, cero discrepancias
O3c: 32/32 fuentes, cero discrepancias
```

No existe una alteración silenciosa de G7a ni de los dos ledgers que ya habían
terminado verdes en V3.

## Cambio legítimo del ledger O3c

El ledger O3c conserva 33 líneas y 6.421 bytes, pero cambia de:

```text
blob=5ad17947fb4a75b9c0a37dceea4e91074687a9c3
SHA-256=5603a529fadbe424ccbb17a9be98dda58871bb31c3168a83048b61ab3a47064f
```

a:

```text
blob=118e0c4b5052b6c9aa67a41c707876ee9f22fcc8
SHA-256=cd2b633cf7c787a58fa787fea8b3cf71e9d725388cb0795539f2851f2a49ee83
```

Su único cambio textual es la huella de:

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff_test.go
```

Huella histórica:

```text
SHA-256=13bac7a37f5d82bc27d2cb4f767b963eee6f99961cf574a8616186afde391d76
líneas=263
bytes=8609
```

Huella V4:

```text
SHA-256=73b511eb634c5bfb2bcb8fa89e6fde0e58231c9553da13543c98930154b2d020
líneas=405
bytes=14885
delta=+145/-3
```

El cambio pertenece al arnés test-only. Añade, de forma cerrada:

- un directorio temporal privado por selector;
- propietario efectivo y modo `0700`;
- nacimiento vacío;
- identidad física anterior a la limpieza;
- recuento de residuos previo;
- retirada y acreditación mediante ausencia;
- raíz exterior vacía;
- atestación ligada a un fichero regular privado `0600`;
- bifurcación fatal directa de `particion`.

No modifica G7a, producto O3a/O3b, las máquinas productivas O3c ni la matriz de
22 oráculos.

El paquete V4 reproduce exactamente este ledger: sus primeras 33 líneas de
`fuentes.tsv` son byte a byte iguales a `tools/o3c_p6_conductor/fuentes.tsv`.
La entrada 33 adicional es el único fixture runtime autorizado:

```text
deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
SHA-256=7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487
```

## Evidencia histórica preservada

### O3a V3

Paquete:

```text
/srv/fabrica/revisiones/evidencia-o3a-v5-cnd-v3-inventario-fea52f3-r1
```

Su manifiesto conserva SHA-256:

```text
36e0d01cc361e4ffd1461cc104bc7a18fee9a1c1fac6afaeb024ae663097f773
```

Las 21 sumas declaradas fueron recalculadas correctamente. El resumen
histórico continúa declarando:

```text
resultado=GO
head=fea52f3ddf796991c93c85cae992ca695db1ae63
bloques=14
casos_registrados=74
fd_conductor_inicio=4
fd_conductor_fin=4
residuos=cero
```

### O3b V3

Paquete:

```text
/srv/fabrica/revisiones/evidencia-o3b-p7-v3-inventario-fea52f3/r1
```

Su manifiesto conserva SHA-256:

```text
99e6adc183f0ce0dea7b605f0369157bf1681f7debf158720bd6d7359f480ce4
```

Sus seis sumas fueron recalculadas correctamente. Conserva:

```text
resultado=GO
casos_totales=234
capturas_normal=100
capturas_race=100
o17_directos=6
o17_estado=65
o17_stdout_stderr=cero
residuos=cero
```

### O3c V3

El destino histórico:

```text
/srv/fabrica/revisiones/evidencia-o3c-p6-v3-inventario-fea52f3
```

continúa existiendo y vacío, coherente con el conductor histórico que retiró
su staging al primer fallo. No se ha fabricado retroactivamente un paquete que
nunca existió.

La sonda posterior permanece en:

```text
/srv/fabrica/revisiones/o3c-p6-cap021-selectores-fea52f3-r1
```

Su manifiesto conserva SHA-256:

```text
4e4f211e76b5b16f3796f0bdb692763e327cc8ed5abd1a7523f0ab4f86000131
```

Todas sus sumas validan. Su resumen continúa declarando:

```text
resultado=NO-GO
head=fea52f3ddf796991c93c85cae992ca695db1ae63
selectores=7
reintentos=0
```

Los estados y salidas individuales eran los esperados, pero el inventario
temporal crecía de cero a siete entradas. Ese `NO-GO` diagnóstico no se
reclasifica ni se presenta como ejecución del agregado `CAP_NORMAL_021`.

## Paquete canónico V4

Paquete:

```text
/srv/fabrica/orquesta/home/evidencias/o3c-p6-cap021-runtime-tmp-v4-4840207-canonica-r1
```

Propiedades observadas:

```text
tipo=directorio real
propietario=orquesta:orquesta
modo=0700
identidad física=2049:2267315:999:700
entradas=12
subdirectorios=0
enlaces simbólicos=0
```

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

Las entradas de datos son regulares, `0600`, propiedad de `orquesta` y
`nlink=1`. El helper es regular, `0700`, del mismo propietario y `nlink=1`.

El manifiesto tiene SHA-256:

```text
4ebe2573983ed9edecc83f86f9e639507f39b013b9976778d3179edca02d1330
```

Las once sumas declaradas validan.

`contexto.tsv` liga los tres cortes al mismo commit y árbol:

```text
head_inicial=head_snapshot=head_publicacion=
484020703c683c324e9b2eaef5c43a56c1d95ea6

tree_inicial=tree_snapshot=tree_publicacion=
408ff618672f8faab56cbc46f33f7233c380f5f4
```

También registra:

```text
estado Git limpio
EUID=999
Go=go version go1.26.5 linux/amd64
SHA conductor=8918ef3b2ba7b343ff95468a95483e676ad18991715a8fd048bc716de5cb15df
SHA ledger O3c=cd2b633cf7c787a58fa787fea8b3cf71e9d725388cb0795539f2851f2a49ee83
SHA fuentes empaquetadas=5dd85475d86e0287727c2dbf1830e48b6fa2b6c71c49b64e70e85d728f6c8cfe
```

## Reconciliación funcional V4

La matriz publicada contiene:

| Grupo | Normal | Detector de carreras | Total |
| --- | ---: | ---: | ---: |
| 22 casos base | 22 | 22 | 44 |
| Capturas | 100 | 100 | 200 |
| Total | 122 | 122 | 244 |

Los 22 identificadores base, comandos y oráculos coinciden exactamente con
`tools/o3c_p6_conductor/casos.tsv`, sin discrepancias.

Resultado independiente:

```text
casos=244
GO=244
estado_cero=244
stdout_cinco_bytes=244
stderr_cero=244
filas_NO-GO=0
```

Las bifurcaciones fatales directas conservan:

```text
filas=6
normal=3
detector_de_carreras=3
estado=65 en 6/6
stdout=0 en 6/6
stderr=0 en 6/6
resultado=GO en 6/6
```

La reconciliación de selectores dio:

```text
selectores de casos ordinarios=1470
selectores de bifurcaciones fatales=2
total esperado=1472
total publicado=1472
duplicados=0
invariantes inválidas=0
filas NO-GO=0
```

Distribución:

```text
positivo=210
retirada=210
retirada_terminal=210
reuso=210
particion=212
retirada_sin_ref=210
retirada_plazo=210
```

Las dos filas adicionales de `particion` corresponden a la bifurcación fatal
directa, una por modo.

## Cierre específico de CAP_NORMAL_021

Filas publicadas:

| Identificador | Modo | Estado | stdout | stderr | Duración | Descriptores | Selectores | Resultado |
| --- | --- | ---: | ---: | ---: | ---: | --- | --- | --- |
| `CAP_NORMAL_021` | normal | 0 | 5 | 0 | 987 ms | 7→7 | 7/7 | `GO` |
| `CAP_RACE_021` | detector de carreras | 0 | 5 | 0 | 4.249 ms | 7→7 | 7/7 | `GO` |

Ambas filas conservan:

```text
hijos=0→0
zombis=0→0
grupos=0→0
temporales=0→0
TMPDIR=0→0
contenedor retirado=si
descriptores ambientales cerrados=si
grupo de ejecución ausente=si
```

Las catorce atestaciones asociadas contienen los siete selectores exactos por
modo. Todas acreditan UID 999, modo `0700`, nacimiento vacío, identidad
conservada antes de limpiar, ausencia final, raíz exterior vacía, retirada y
resultado `GO`.

El conductor no decide el éxito por tamaño únicamente: contrasta el literal
`PASS\n`, cuya huella es:

```text
c26de83abdc9496cd1301470918ec39ecca1cf389ef0ae1c6504da1800d1c431
```

## Publicación y limpieza

`publicacion.tsv` declara:

```text
metodo=renameat2_RENAME_NOREPLACE
huella_origen=2049:2267315:999:700
sha_helper_noreplace=08b3ed3abefd234d249db0c73ab1016ad255b3f06958d6411660cf72e46a4173
```

La identidad del directorio publicado coincide con la huella previa. No existe
alternativa mediante copia, `mv`, reemplazo o reintento.

`residuos.txt` tiene cero bytes y `resumen.txt` declara:

```text
resultado=GO
casos_totales=244
bf_directos=6
capturas_normal=100
capturas_race=100
tmpdir_selectores=1472
snapshot_revalidado_post_ejecucion=33
target_head_tree_limpieza_revalidados=si
publicacion_go_no_replace=si
residuos=cero
staging_retirado=si
```

El target canónico continúa limpio en el commit y árbol exactos:

```text
/srv/fabrica/orquesta/home/revisiones/o3c-p6-cap021-runtime-tmp-v4-4840207-target
```

## Dictamen razonado

No se encontraron hallazgos funcionales o de trazabilidad:

```text
P0=0
P1=0
P2=0
```

V4 satisface la única puerta O3c que impedía un nuevo cierre de V3 porque:

1. conserva G7a y los ledgers O3a/O3b byte a byte;
2. explica y liga el único cambio legítimo del ledger O3c;
3. publica las 32 fuentes exactas y el fixture único;
4. ejecuta la matriz completa sin filas rojas;
5. acredita específicamente `CAP_NORMAL_021` y su versión con detector de
   carreras;
6. conserva 1.472 atestaciones reconciliables;
7. liga commit, árbol, target, conductor, ledger, fuentes y binarios;
8. publica sin reemplazo y sin residuos;
9. recibe revisiones funcional y de seguridad independientes sobre el mismo
   commit, árbol y paquete.

La conclusión es prospectiva: permite documentar que la dependencia O3c está
cerrada en el corte `4840207`. No altera el hecho de que `fea52f3` recibió
`NO-GO` y no debe integrarse ni describirse como candidato verde.

## Incidencias no materiales de la auditoría

Tres consultas auxiliares se corrigieron sin producir cambios:

1. un `printf` usado para separar cabeceras interpretó el texto como opción; la
   lectura se repitió con una forma segura;
2. un primer verificador genérico asumió erróneamente que el ledger O3a tenía
   las mismas columnas que O3b/O3c; se sustituyó por un verificador consciente
   de cada formato, que obtuvo `10/10`, `22/22` y `32/32`;
3. Git rechazó inicialmente la lectura del target propiedad de `orquesta` por
   propiedad dudosa; se repitió con `safe.directory` exclusivamente en la
   línea de órdenes, sin escribir configuración, y el target resultó limpio.

Ninguna incidencia ejecutó producto, escribió archivos, cambió permisos,
modificó Git o alteró evidencia.

## Comprobaciones ejecutadas

Solo se realizaron comprobaciones de lectura:

```text
identidad, padre, árbol y genealogía Git
estado limpio del worktree y del target
write-set de commits históricos y postcanónicos
comparación de blobs y SHA-256 entre fea52f3 y 4840207
diff exacto del ledger O3c
validación de las 10/22/32 fuentes de los tres ledgers
sha256sum -c de evidencias históricas O3a y O3b
sha256sum -c de la sonda diagnóstica O3c histórica
sha256sum -c del paquete V4, 11/11
inventario físico del paquete
correspondencia del ledger con fuentes.tsv
reconciliación tabular de 244 casos, seis bifurcaciones y 1.472 selectores
comparación exacta de los 22 comandos y oráculos base
revalidación final de limpieza
```

No se ejecutaron:

```text
conductor O3c
build normal o con detector de carreras
binarios de producto
mutantes
go test
go test -race
go vet
puertas globales
PostgreSQL
Docker
red de producto
integración continua remota
integración Git
push
despliegue
producción
```

Estas omisiones son obligatorias y proporcionales: la corrida canónica y sus
destinos están consumidos, y esta tarea es una revisión documental de solo
lectura.

## Límites conservados

Este GO no afirma:

- que el antiguo fallo de `fea52f3` haya sido reconstruido;
- estabilidad estadística global de C21;
- cierre del parche o confianza completa del conjunto de herramientas;
- aislamiento frente a otro proceso con el mismo UID 999;
- atestación transitiva de cabeceras, objetos, bibliotecas, cargador o sistema
  raíz C del host;
- atestación del cargador dinámico anterior al primer Bash;
- cierre de O4, fases posteriores o la aplicación completa;
- conformidad legal, producción o uso de datos reales.

La frontera C y el lanzador confiable permanecen declarados. La autoridad Unix
`orquesta` sigue siendo cooperativa.

## Relevo y siguiente corte

Dirección puede materializar exclusivamente esta acta desde la base:

```text
484020703c683c324e9b2eaef5c43a56c1d95ea6
```

Debe comprobar antes del commit:

```text
una única ruta añadida
cero cambios en producto, ledgers o evidencias
padre exacto 484020703c683c324e9b2eaef5c43a56c1d95ea6
git diff --check
worktree limpio después del commit
```

El siguiente corte seguro es una revisión independiente de seguridad del
cierre O3a V3 frente a la nueva puerta O3c V4, o la decisión documental de
dirección que corresponda tras disponer de ambos dictámenes.

O4 continúa cerrado. Cualquier integración, publicación externa, envío a
GitHub, despliegue o cambio de métricas requiere una orden separada y expresa.

## Entrega

```text
Tarea:
O3A-V5-CND-V3-C21-INVENTARIO-EXACTO-CIERRE-PUERTA-O3C-V4

Estado:
GO funcional y de trazabilidad; P0=0, P1=0, P2=0

Base revisada:
484020703c683c324e9b2eaef5c43a56c1d95ea6

Candidato histórico preservado:
fea52f3ddf796991c93c85cae992ca695db1ae63 — NO-GO sin reclasificar

Revisiones O3c consumidas:
bb23278e0b5660b611cb9c7faa19b197e0283a19
22fde5f72b2ce4bf92c2ae32bee4d7ab47a4e36b

Resultado:
V4 satisface la puerta O3c para un nuevo cierre documental de V3;
CAP_NORMAL_021 y CAP_RACE_021 quedan acreditados en 4840207.

Pruebas:
Solo lectura; hashes, genealogía, bytes, ledgers, paquete, matrices,
selectores, publicación y residuos.

Pruebas omitidas:
Conductor, builds, producto, mutantes y gates pesados, por prohibición y
consumo de la corrida canónica.

Write-set:
docs/portal_vec/revisiones/revision_funcional_o3a_v5_cnd_v3_c21_inventario_exacto_cierre_o3c_v4_2026-08-21.md

Limitaciones:
Sin estabilidad global, toolchain completo, O4, integración, publicación
externa, despliegue, producción o métricas.

Siguiente tarea:
Revisión independiente de seguridad del cierre o decisión documental expresa
de dirección; O4 permanece cerrado.
```
