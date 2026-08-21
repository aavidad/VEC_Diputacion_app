# Revisión de seguridad O3a V5 CND V3 C21: cierre de la puerta O3c V4

Fecha: 21 de agosto de 2026.

Tarea:
`O3A-V5-CND-V3-C21-INVENTARIO-EXACTO-CIERRE-PUERTA-O3C-V4`.

Dictamen: **GO de seguridad, `P0=0`, `P1=0`, `P2=0`**.

Las prioridades usadas en este dictamen son:

- P0: hallazgo crítico;
- P1: hallazgo mayor que impide el cierre;
- P2: hallazgo menor que exige corrección antes del cierre.

No quedan hallazgos P0, P1 o P2 dentro del alcance exacto de esta revisión.

Este GO acredita únicamente que la puerta O3c obligatoria que bloqueó el
candidato histórico O3a V5 CND V3 dispone ahora de una evidencia posterior,
durable, independiente y cerrada sobre un descendiente distinto.

No convierte en verde la ejecución roja histórica, no reclasifica el candidato
`fea52f3ddf796991c93c85cae992ca695db1ae63`, no acredita estabilidad global
de C21, no cierra la cadena de herramientas, no abre O4 y no autoriza
integración, publicación externa, envío a GitHub, despliegue, producción,
credenciales, datos reales, métricas o estado transversal.

## Capability, invariante y write-set

Capability:

```text
O3A-V5-CND-V3-C21-INVENTARIO-EXACTO-CIERRE-PUERTA-O3C-V4
```

La capacidad es reconocer, sin compensación ni reintento, que la dependencia
O3c del inventario exacto de descriptores de fichero —FD, por su denominación
inglesa— queda satisfecha en un corte posterior mediante la evidencia canónica
V4 y sus dos revisiones postcanónicas independientes.

Invariante:

1. el candidato histórico `fea52f3` conserva para siempre sus dos dictámenes
   `NO-GO`, con `P0=0`, `P1=1`, `P2=0`;
2. sus evidencias verdes O3a/O3b no compensan su puerta O3c roja;
3. la evidencia V4 pertenece al commit posterior
   `484020703c683c324e9b2eaef5c43a56c1d95ea6`;
4. G7a, los ledgers O3a/O3b y la enmienda V3 permanecen byte a byte;
5. el único cambio del ledger O3c corresponde a su arnés temporal V4 y queda
   ligado a las 32 fuentes exactas;
6. ningún error, ausencia, discrepancia o límite externo se interpreta como
   autorización;
7. no se amplía el modelo de confianza del usuario Unix compartido, del
   cargador dinámico inicial ni de la frontera C.

Write-set futuro único:

```text
docs/portal_vec/revisiones/revision_seguridad_o3a_v5_cnd_v3_c21_inventario_exacto_cierre_o3c_v4_2026-08-21.md
```

La ruta estaba ausente antes y después de esta revisión. No se modificaron
producto, pruebas, conductores, fixtures, ledgers, evidencias, permisos,
procesos, ramas ajenas, documentos transversales ni métricas.

## Independencia y entorno revisor

La revisión se realizó estrictamente en solo lectura desde:

```text
/srv/fabrica/revisiones/o3a-v5-cnd-v3-c21-cierre-o3c-v4-seguridad-20260821
```

Rama exclusiva:

```text
revision/o3a-v5-cnd-v3-c21-cierre-o3c-v4-seguridad-20260821
```

Identidad final:

```text
HEAD  484020703c683c324e9b2eaef5c43a56c1d95ea6
tree  408ff618672f8faab56cbc46f33f7233c380f5f4
estado Git limpio
```

El worktree es `root:root`, modo `0755`. No se añadió configuración Git
global: todas las consultas que lo necesitaron usaron `safe.directory`
únicamente dentro de su propia invocación.

## Instrucciones y autoridades leídas

Se leyeron completos los dos `AGENTS.md` aplicables:

| Autoridad | Líneas | SHA-256 |
| --- | ---: | --- |
| `/srv/fabrica/AGENTS.md` | 94 | `08324611474fedcd484e2603cebd2c492e1d022d5b4c1bee5dc1b86d5484b7d0` |
| `AGENTS.md` del worktree | 319 | `b830a866dc85714b32115b8594c906865065ddff0a76784fb1ce7b21f0a4d5e2` |

También se leyeron completas y se contrastaron:

```text
docs/portal_vec/decision_f0_h0b_c4b2_g2o_o3a_arranque_mapa_fd_2026-08-09.md
docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o3a_v5_autoridad_ci_sin_r_2026-08-09.md
docs/portal_vec/revisiones/revision_f0_h0b_c4b2_g2o_o3a_v5_codigo_final_2026-08-10.md
docs/portal_vec/enmienda_o3a_v5_cnd_estados_causales_go1_26_6_2026-08-14.md
docs/portal_vec/enmienda_o3a_v5_cnd_estados_causales_v2_c18_2026-08-14.md
docs/portal_vec/enmienda_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md
docs/portal_vec/revisiones/revision_funcional_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md
docs/portal_vec/revisiones/revision_seguridad_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md
docs/portal_vec/decision_f0_h0b_c4b2_g2o_o3c_continuacion_salida_2026-08-11.md
docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md
docs/portal_vec/revisiones/revision_funcional_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md
docs/portal_vec/revisiones/revision_seguridad_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md
docs/portal_vec/enmienda_o3c_p6_cap_normal_021_runtime_tmp_2026-08-21.md
```

También se leyeron completas desde sus respectivos commits las revisiones
postcanónicas funcional y de seguridad de O3c V4.

## Identidad y genealogía

### Candidato histórico rechazado

| Propiedad | Valor |
| --- | --- |
| Commit | `fea52f3ddf796991c93c85cae992ca695db1ae63` |
| Padre | `ec02febaa3080cb8e7894340075beb850ea54348` |
| Árbol | `c9d5a393f39d48a5015848d5fd49a0b69959c622` |
| Asunto | `test(O3A-V5-CND): compara inventario FD exacto en C21` |
| Resultado histórico | `NO-GO funcional` y `NO-GO de seguridad` |
| Prioridades históricas | `P0=0`, `P1=1`, `P2=0` |

El commit es antepasado de la base actual. La relación inversa no existe.

### Base de esta revisión

| Propiedad | Valor |
| --- | --- |
| Commit | `484020703c683c324e9b2eaef5c43a56c1d95ea6` |
| Padre | `7782e4679e546cde4d693633911d5ec3a47bf85b` |
| Árbol | `408ff618672f8faab56cbc46f33f7233c380f5f4` |
| Asunto | `audit(o3c): corregir publicacion atomica V4` |

La cadena entre `fea52f3` y `4840207` es lineal y conserva los commits de
evidencia durable, las dos actas históricas `NO-GO`, el trabajo temporal V4 y
la corrección final de publicación atómica.

## Preservación del NO-GO histórico

Los dictámenes históricos permanecen presentes y no han sido editados,
eliminados ni reclasificados:

| Acta | SHA-256 actual | Dictamen conservado |
| --- | --- | --- |
| `revision_funcional_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md` | `53a288ecec28ca3249de72df1b4a52b05db33dcf4c01f95025a91c2649ff7c13` | `NO-GO`, P1=1 |
| `revision_seguridad_o3a_v5_cnd_v3_c21_inventario_exacto_2026-08-14.md` | `fe8c06d0b3f319101760c737d04dee14eb5d1a6182d457cc6e7634e888949f4e` | `NO-GO`, P1=1 |

El motivo histórico continúa siendo el mismo: la única conducción O3c del
candidato terminó roja en `CAP_NORMAL_021`, estado 1, stdout de 197 bytes y
stderr vacío. El paquete no llegó a publicarse y los 197 bytes no quedaron
conservados.

No se reconstruye su contenido, no se atribuye retroactivamente una causa y no
se repite ese SHA para buscar verde.

## Preservación byte a byte de V3

Entre `fea52f3` y `4840207` permanecen idénticos:

| Objeto | Modo | Líneas | Bytes | SHA-256 en ambos commits |
| --- | ---: | ---: | ---: | --- |
| G7a `supervisor_..._arranque_pruebas.go` | `100644` | 750 | 22.607 | `2d5fc67e6aff0f6e11305e9d92690452f7ffb194b4bcef22cd2dd9121235bbb3` |
| `tools/o3a_v5_conductor/fuentes_v5.tsv` | `100644` | 11 | 1.554 | `ba2b0a1c9838f57ca43d53ec6133ae74bfa7452f7511421e367d7fc5f3d079b0` |
| `tools/o3b_p7_conductor/fuentes.tsv` | `100644` | 23 | 4.400 | `238e019a8fe31c7de830e174a4308083452ef22ceb54999ca4a8aeff5aac5f4f` |
| Enmienda O3a V3 | `100644` | 103 | 4.605 | `10e1eda8a6a741326be9fb53c2566b655ae316b4af819c224f7b4f368a8a2753` |

La comparación Git entre ambos commits no devuelve ninguna ruta para esos
cuatro objetos.

Los ledgers actuales se revalidaron contra el árbol exacto:

```text
O3a: 10/10 filas, cero discrepancias de líneas o SHA-256
O3b: 22/22 filas, cero discrepancias
O3c: 32/32 filas, cero discrepancias
```

La fila de G7a conserva en los tres ledgers la huella exacta
`2d5fc67e…`.

## Cambio justificado del ledger O3c

El ledger O3c mantiene:

```text
modo=100644
líneas=33
bytes=6421
32 fuentes exactas
```

Su SHA-256 cambia de:

```text
5603a529fadbe424ccbb17a9be98dda58871bb31c3168a83048b61ab3a47064f
```

a:

```text
cd2b633cf7c787a58fa787fea8b3cf71e9d725388cb0795539f2851f2a49ee83
```

El delta contiene una sola sustitución de huella:

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff_test.go
```

Huella histórica:

```text
13bac7a37f5d82bc27d2cb4f767b963eee6f99961cf574a8616186afde391d76
```

Huella V4:

```text
73b511eb634c5bfb2bcb8fa89e6fde0e58231c9553da13543c98930154b2d020
```

El fuente pasa de 263 líneas y 8.609 bytes a 405 líneas y 14.885 bytes. El
delta acumulado es `+145/-3` y se materializó mediante cuatro commits
lineales de O3c:

```text
4d2951f  aisla runtime temporal de CAP_NORMAL_021
6407fad  evita doble liberación del hilo Handoff
14c1f31  restaura salida aislada de selectores
47dfb52  acredita temporales por selector
```

El cambio:

- exige un directorio temporal absoluto, real, vacío, del identificador de
  usuario Unix efectivo y modo `0700`;
- crea un temporal privado por selector;
- conserva su identidad física;
- espera al hijo y contiene descendientes;
- cuenta residuos antes de retirarlos;
- retira el temporal desde el padre;
- acredita ausencia mediante `Lstat=ENOENT`;
- exige la raíz exterior vacía;
- registra una atestación cerrada;
- añade la bifurcación fatal directa de partición.

No cambia G7a, los ledgers O3a/O3b ni código productivo O3a. El ledger O3c
final coincide 32/32 con el árbol de `4840207`.

Las primeras 32 filas de `fuentes.tsv` del paquete V4 coinciden byte a byte con
este ledger. La fila 33 es el único fixture runtime:

```text
deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
```

SHA-256:

```text
7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487
```

## Evidencia histórica revalidada

### O3a del candidato V3

Ruta:

```text
/srv/fabrica/revisiones/evidencia-o3a-v5-cnd-v3-inventario-fea52f3-r1
```

Resultado proporcional:

```text
SHA256SUMS=36e0d01cc361e4ffd1461cc104bc7a18fee9a1c1fac6afaeb024ae663097f773
manifiesto completo=OK
head=fea52f3ddf796991c93c85cae992ca695db1ae63
resultado=GO
Go=1.26.6
bloques=14
casos=74
FD del conductor=4→4
residuos=cero
```

### O3b del candidato V3

Ruta:

```text
/srv/fabrica/revisiones/evidencia-o3b-p7-v3-inventario-fea52f3/r1
```

Resultado proporcional:

```text
SHA256SUMS=99e6adc183f0ce0dea7b605f0369157bf1681f7debf158720bd6d7359f480ce4
manifiesto completo=OK
resultado=GO
Go=1.26.5
casos=234
capturas=100 normal + 100 carrera
bifurcaciones fatales directas=6
estado fatal=65
salida fatal=0/0
residuos=cero
```

Estas dos evidencias verdes continúan sin compensar la puerta O3c roja.

### O3c rojo del candidato V3

El destino histórico permanece presente y vacío:

```text
/srv/fabrica/revisiones/evidencia-o3c-p6-v3-inventario-fea52f3
entradas=0
```

La sonda diagnóstica posterior y no compensatoria permanece en:

```text
/srv/fabrica/revisiones/o3c-p6-cap021-selectores-fea52f3-r1
```

Resultado revalidado:

```text
SHA256SUMS=4e4f211e76b5b16f3796f0bdb692763e327cc8ed5abd1a7523f0ab4f86000131
manifiesto completo=OK
resultado=NO-GO
head=fea52f3ddf796991c93c85cae992ca695db1ae63
selectores=7
reintentos=0
```

La sonda no reproduce el agregado rojo original, no recupera sus 197 bytes y
no cambia el dictamen histórico.

## Evidencia canónica O3c V4

Paquete:

```text
/srv/fabrica/orquesta/home/evidencias/o3c-p6-cap021-runtime-tmp-v4-4840207-canonica-r1
```

Target:

```text
/srv/fabrica/orquesta/home/revisiones/o3c-p6-cap021-runtime-tmp-v4-4840207-target
```

El target es `orquesta:orquesta`, identificador de usuario/grupo `999:982`,
modo `0700`, sin remotos, alternates, shallow o common-dir redirigido. Está
limpio en:

```text
HEAD=484020703c683c324e9b2eaef5c43a56c1d95ea6
tree=408ff618672f8faab56cbc46f33f7233c380f5f4
```

### Inventario y permisos

El paquete es un directorio real `orquesta:orquesta`, modo `0700`, identidad:

```text
dispositivo:inode:UID:modo = 2049:2267315:999:700
```

Contiene exactamente doce ficheros regulares:

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

Las once entradas de datos son modo `0600`; `rename_noreplace` es modo
`0700`. Todas pertenecen a `orquesta`, tienen un solo enlace físico y no son
enlaces simbólicos. No se encontraron listas de control de acceso ampliadas ni
capacidades Linux.

El manifiesto tiene SHA-256:

```text
4ebe2573983ed9edecc83f86f9e639507f39b013b9976778d3179edca02d1330
```

Sus once entradas se verificaron correctamente como `orquesta`.

### Cardinalidades y resultados

```text
casos ordinarios=244/244 GO
modo normal=122
modo con detector de carreras=122
bifurcaciones fatales directas=6/6 GO
atestaciones temporales=1472/1472 GO
filas NO-GO=0
duplicados=0
huérfanos=0
faltantes=0
excesos=0
```

Los 250 pares ejecución/modo se enlazan correctamente con sus selectores:

```text
claves con selectores=212
claves con cero selectores=38
claves coincidentes=250
filas faltantes=0
filas excesivas=0
filas huérfanas=0
```

Distribución temporal:

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

### CAP_NORMAL_021 y carrera

`CAP_NORMAL_021` acredita:

```text
modo=normal
estado=0
stdout=5
stderr=0
duración=987 ms
FD=7→7
hijos=0→0
zombis=0→0
grupos=0→0
temporales=0→0
selectores=7/7
resultado=GO
```

`CAP_RACE_021` acredita:

```text
modo=carrera
estado=0
stdout=5
stderr=0
duración=4249 ms
FD=7→7
hijos=0→0
zombis=0→0
grupos=0→0
temporales=0→0
selectores=7/7
resultado=GO
```

Las catorce filas temporales asociadas contienen exactamente los siete
selectores por modo y acreditan conjuntamente:

```text
UID=999
modo=0700
entradas iniciales=0
identidad conservada=true
residuos antes de limpieza=1
Lstat=ENOENT
raíz exterior vacía=true
retirada=true
resultado=GO
```

### Fuentes y contexto

Las 33 rutas del paquete están seguidas por Git, son regulares, no simbólicas
y coinciden con su SHA-256. Las primeras 32 coinciden con el ledger O3c y la
última con el fixture exacto.

`contexto.tsv` fija la misma identidad en los tres cortes:

```text
HEAD inicial/snapshot/publicación =
484020703c683c324e9b2eaef5c43a56c1d95ea6

tree inicial/snapshot/publicación =
408ff618672f8faab56cbc46f33f7233c380f5f4
```

También fija:

```text
estado Git=limpio
EUID=999
Go=go1.26.5 linux/amd64
SHA conductor=8918ef3b2ba7b343ff95468a95483e676ad18991715a8fd048bc716de5cb15df
SHA ledger O3c=cd2b633cf7c787a58fa787fea8b3cf71e9d725388cb0795539f2851f2a49ee83
SHA fuentes del target=5dd85475d86e0287727c2dbf1830e48b6fa2b6c71c49b64e70e85d728f6c8cfe
```

### Publicación y limpieza

`publicacion.tsv` declara:

```text
método=renameat2_RENAME_NOREPLACE
helper=08b3ed3abefd234d249db0c73ab1016ad255b3f06958d6411660cf72e46a4173
identidad publicada=2049:2267315:999:700
```

La identidad física actual coincide. El publicador no tiene alternativa de
copia, reemplazo, `mv`, reintento o éxito por colisión.

`residuos.txt` está vacío. Los temporales propios de la conducción están
ausentes:

```text
/var/tmp/o3c-p6.NJGSHV
/var/tmp/o3c-p6-tools.OFl6xW
```

## Revisiones postcanónicas independientes

### Revisión funcional

```text
commit=bb23278e0b5660b611cb9c7faa19b197e0283a19
padre=484020703c683c324e9b2eaef5c43a56c1d95ea6
tree=fb1867ae0adea947ed6dd65b3b6588fd06af2442
```

Añade exclusivamente:

```text
docs/portal_vec/revisiones/revision_funcional_o3c_p6_cap_normal_021_runtime_tmp_v4_2026-08-21.md
```

El acta es modo `100644`, tiene 722 líneas y SHA-256:

```text
0770193993646b7c20e141be72a9bdc8c4abd7ce71017e09dc73a70d6ce6a06e
```

Dictamen: `GO funcional y de trazabilidad`, `P0=0`, `P1=0`, `P2=0`.

### Revisión de seguridad

```text
commit=22fde5f72b2ce4bf92c2ae32bee4d7ab47a4e36b
padre=484020703c683c324e9b2eaef5c43a56c1d95ea6
tree=1cfe901ac1b37f6e6ddb1dd3c66cb459632ec6c7
```

Añade exclusivamente:

```text
docs/portal_vec/revisiones/revision_seguridad_o3c_p6_cap_normal_021_runtime_tmp_v4_2026-08-21.md
```

El acta es modo `100644`, tiene 549 líneas y SHA-256:

```text
551e91ea4fb52608c18482036b863b4279cb5fb61ff12ceae01584c9200ed1a4
```

Dictamen: `GO de seguridad`, `P0=0`, `P1=0`, `P2=0`.

Ambos commits son hijos directos e independientes de la misma base y modifican
una sola acta cada uno. Ninguno integra, altera producto o autoaprueba la
evidencia productora.

## Análisis de seguridad y fallo cerrado

El cierre es fail-closed, es decir, falla de forma cerrada:

1. el `NO-GO` histórico continúa siendo una evidencia terminal;
2. la nueva evidencia pertenece a otro SHA y otro destino;
3. no existe repetición del mismo candidato rojo;
4. toda discrepancia de target, fuente, modo, propietario, enlace, huella,
   descriptor, temporal, caso o publicación termina no verde;
5. la publicación usa `renameat2(RENAME_NOREPLACE)` y no sustituye una entrada
   concurrente;
6. el helper valida inventario exacto, identificador de usuario efectivo,
   modos, enlaces y hashes antes de publicar;
7. tras un rename verde termina inmediatamente, sin operación falible
   posterior;
8. no se usan mayoría, tolerancia, `SKIP`, reclasificación o evidencia de otra
   puerta como compensación.

La prueba V4 demuestra que el defecto temporal observado por la sonda
diagnóstica dispone de una corrección posterior y de cobertura exacta. No
demuestra que el candidato histórico hubiera sido verde, ni recupera los bytes
perdidos de aquella ejecución.

## Límites de confianza conservados

El GO no amplía el modelo de amenaza:

1. `orquesta`, identificador de usuario Unix 999, sigue siendo una autoridad
   cooperativa. Otro proceso malicioso con el mismo identificador puede
   modificar objetos accesibles a ese usuario.
2. El paquete `0700/0600` es privado, pero no se declara inmutable o firmado
   frente al propio usuario 999.
3. El lanzador confiable debe retirar `LD_PRELOAD`, `LD_AUDIT` y
   `LD_LIBRARY_PATH` antes del primer `exec`. V4 no puede atestar
   retroactivamente lo consumido por el cargador dinámico.
4. El build con detector de carreras fija siete ejecutables C, pero no
   atestigua transitivamente cabeceras, objetos, bibliotecas, cargador o
   sistema raíz C del host.
5. No se ejecutó `fsck` ni se declara atestado completo el almacén de objetos
   Git.
6. La evidencia corresponde a una única conducción y no constituye una prueba
   estadística de estabilidad.
7. No se acreditan PostgreSQL, Docker, SQL, red, HTTP, interfaz, accesibilidad,
   despliegue, producción o la aplicación completa.

Estos límites estaban declarados antes del cierre y permanecen sin
relajación.

## Hallazgos

| Prioridad | Número | Estado |
| --- | ---: | --- |
| P0 — crítico | 0 | sin hallazgos |
| P1 — mayor | 0 | sin hallazgos |
| P2 — menor | 0 | sin hallazgos |

Resultado:

```text
P0=0
P1=0
P2=0
```

## Puertas reproducidas en solo lectura

```text
AGENTS.md aplicables completos                         GO
autoridades O3a/V3/O3c/V4 completas                   GO
rama, HEAD, árbol y limpieza                          GO
genealogía fea52f3→4840207                            GO
dos actas O3c hijas directas y de una sola ruta       GO
G7a byte a byte                                        GO
ledger O3a byte a byte y 10/10                        GO
ledger O3b byte a byte y 22/22                        GO
enmienda V3 byte a byte                               GO
ledger O3c: delta único V4 y 32/32                    GO
evidencia histórica O3a                               SHA256SUMS OK
evidencia histórica O3b                               SHA256SUMS OK
destino O3c histórico                                 vacío, preservado
sonda diagnóstica                                     NO-GO, cero reintentos
paquete V4                                            11/11 SHA OK como orquesta
inventario, propietario, tipos, modos y nlink         GO
ACL ampliadas y capacidades Linux                     cero
244 casos ordinarios                                  GO
6 bifurcaciones fatales                               GO
1472 atestaciones                                     GO
CAP_NORMAL_021 y CAP_RACE_021                         GO
33 fuentes contra target                              GO
publicación sin reemplazo e identidad física          GO
residuos y temporales                                 cero
git diff --check de los rangos revisados              GO
estado final del worktree                             limpio
```

## Incidencias auxiliares no materiales

Dos primeras expresiones auxiliares se corrigieron sin escribir estado:

1. el primer lector del ledger O3a interpretó la cabecera `archivo` como una
   fuente; la repetición con el formato exacto acreditó 10/10 y cero
   discrepancias;
2. una primera aserción de enlace temporal exigía filas para las 38
   ejecuciones que declaran cero selectores. El oráculo corregido acreditó 250
   claves coincidentes: 212 con filas, 38 con cero filas, sin faltantes,
   excesos ni huérfanos.

Ninguna incidencia modificó repositorio, target, paquete, índice, permisos,
procesos o evidencias.

## Pruebas omitidas

No se ejecutaron:

```text
conductor O3c
helper de publicación
build normal o con detector de carreras
mutantes
producto
go test
go test -race
go vet global
scripts/verificar_calidad.sh
PostgreSQL
Docker
HTTP
navegador
pruebas de extremo a extremo
integración continua remota
```

La conducción y su destino están consumidos. Repetirlos contradiría la
disciplina de evidencia única. Los gates de producto o globales son
desproporcionados para una revisión documental de cierre y estaban
expresamente prohibidos.

Tampoco se ejecutaron `fetch`, `pull`, `reset`, `push`, merge, rebase,
despliegue, cambios de permisos, señales o gestión de procesos.

## Dictamen y no compensación

**GO de seguridad, `P0=0`, `P1=0`, `P2=0`.**

Este GO significa exclusivamente:

- G7a y las autoridades V3 permanecen byte a byte;
- la puerta O3c roja de `fea52f3` permanece consumida;
- O3c V4 aporta una evidencia posterior, durable y revisada sobre `4840207`;
- la nueva evidencia cierra la dependencia O3c para un futuro corte documental
  de O3a V3;
- no existe compensación, reintento o reescritura del pasado.

No significa:

- que `fea52f3` pueda integrarse;
- que sus dos actas `NO-GO` cambien;
- que C21 sea estable;
- que la cadena de herramientas esté cerrada;
- que O4 o fases posteriores estén abiertas;
- que la aplicación esté terminada o autorizada para producción.

## Siguiente corte seguro

Dirección puede materializar exclusivamente esta acta sobre la base exacta:

```text
484020703c683c324e9b2eaef5c43a56c1d95ea6
```

El commit resultante debe:

1. tener ese padre exacto;
2. añadir únicamente
   `docs/portal_vec/revisiones/revision_seguridad_o3a_v5_cnd_v3_c21_inventario_exacto_cierre_o3c_v4_2026-08-21.md`;
3. conservar G7a, los tres ledgers, la enmienda V3, producto y evidencias;
4. no fusionar ni reescribir las ramas de las actas postcanónicas;
5. recibir una comprobación independiente de padre, write-set y contenido.

El siguiente corte funcional debe ser una revisión independiente y separada
del mismo cierre O3a, con su propio write-set documental. Solo tras ambos
dictámenes puede dirección decidir una materialización o integración
documental posterior. La integración, el envío remoto y las fases de producto
continúan cerrados hasta orden expresa.

## Entrega

```text
Tarea:
O3A-V5-CND-V3-C21-INVENTARIO-EXACTO-CIERRE-PUERTA-O3C-V4

Estado:
GO de seguridad; P0=0, P1=0, P2=0

Base revisada:
484020703c683c324e9b2eaef5c43a56c1d95ea6

Árbol:
408ff618672f8faab56cbc46f33f7233c380f5f4

Candidato histórico:
fea52f3ddf796991c93c85cae992ca695db1ae63
NO-GO consumido y preservado

Evidencia posterior:
o3c-p6-cap021-runtime-tmp-v4-4840207-canonica-r1

Revisiones O3c:
bb23278e0b5660b611cb9c7faa19b197e0283a19
22fde5f72b2ce4bf92c2ae32bee4d7ab47a4e36b

Archivo futuro:
docs/portal_vec/revisiones/revision_seguridad_o3a_v5_cnd_v3_c21_inventario_exacto_cierre_o3c_v4_2026-08-21.md

Resultado:
Puerta O3c V4 cerrada para consumo documental posterior, sin compensar,
borrar ni reclasificar el NO-GO histórico.

Pruebas:
Solo lectura: genealogía, bytes, ledgers, manifiestos, tablas, permisos,
publicación, residuos y límites.

Limitaciones:
UID 999 cooperativo, cargador inicial no atestado, frontera C incompleta,
estabilidad/toolchain/O4/integración/producción no acreditados.

Siguiente tarea:
Revisión funcional independiente del cierre y posterior decisión expresa de
dirección.

Revisión independiente:
GO de seguridad; P0=0, P1=0, P2=0.
```
