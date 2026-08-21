# Revisión independiente de seguridad O3c P6 CAP_NORMAL_021 runtime temporal V4

Fecha: 21 de agosto de 2026.

Tarea: `O3C-P6-CAP021-RUNTIME-TMP-V4-SEGURIDAD-POSTCANONICA`.

Dictamen: **GO de seguridad, P0=0, P1=0, P2=0**.

El dictamen se limita a la seguridad del conductor V4, el cierre de
descriptores, la integridad del paquete canónico y su publicación local
atómica mediante `renameat2(RENAME_NOREPLACE)`. No acredita la estabilidad
global de C21, el parche de toolchain, O4, la aplicación completa ni
producción. Tampoco autoriza integración, publicación externa, `push`,
despliegue, datos reales, credenciales o cambio de métricas.

## Independencia, rama y write-set

La revisión se realizó en modo estrictamente de solo lectura desde el worktree
exclusivo:

```text
/srv/fabrica/revisiones/o3c-p6-cap021-runtime-tmp-v4-seguridad-20260821
```

Rama exclusiva:

```text
revision/o3c-p6-cap021-runtime-tmp-v4-seguridad-20260821
```

Antes y después de la revisión se observaron:

```text
HEAD   484020703c683c324e9b2eaef5c43a56c1d95ea6
tree   408ff618672f8faab56cbc46f33f7233c380f5f4
rama   revision/o3c-p6-cap021-runtime-tmp-v4-seguridad-20260821
estado limpio
```

El candidato tiene como padre único
`7782e4679e546cde4d693633911d5ec3a47bf85b` y asunto
`audit(o3c): corregir publicacion atomica V4`.

En el momento de entregar el contenido de esta acta no existe todavía un
commit revisor: la rama continúa directamente en el candidato. Dirección debe
comprobar al materializarla que el commit exclusivamente documental tenga como
padre exacto:

```text
484020703c683c324e9b2eaef5c43a56c1d95ea6
```

Write-set único autorizado:

```text
docs/portal_vec/revisiones/revision_seguridad_o3c_p6_cap_normal_021_runtime_tmp_v4_2026-08-21.md
```

La ruta estaba ausente antes de esta entrega. No se modificaron conductor,
producto, fixtures, ledgers, autoridades, evidencias canónicas, estado
transversal ni métricas.

## Autoridades leídas

Se localizaron y leyeron completos los dos `AGENTS.md` aplicables:

| Autoridad | Líneas | SHA-256 |
| --- | ---: | --- |
| `/srv/fabrica/AGENTS.md` | 94 | `08324611474fedcd484e2603cebd2c492e1d022d5b4c1bee5dc1b86d5484b7d0` |
| `AGENTS.md` del worktree revisor | 319 | `b830a866dc85714b32115b8594c906865065ddff0a76784fb1ce7b21f0a4d5e2` |

La autoridad material específica del corte es:

| Ruta | Modo Git | Líneas | SHA-256 |
| --- | ---: | ---: | --- |
| `tools/o3c_p6_conductor/conductor.sh` | `100755` | 800 | `8918ef3b2ba7b343ff95468a95483e676ad18991715a8fd048bc716de5cb15df` |
| `docs/portal_vec/enmienda_o3c_p6_cap_normal_021_runtime_tmp_2026-08-21.md` | `100644` | 799 | `490fa132c48a973860edfb604d65ecb48eb783dc36779a151bf6718a436151a4` |

El commit candidato solo modifica esas dos rutas respecto de su padre. La
capability revisada es
`O3C-P6-CAP021-RUNTIME-TMP-V4-FIXTURE-PASS`: atestación acotada del
runtime temporal, el fixture exacto, la construcción y la publicación local
del paquete bajo la autoridad operativa cooperativa `orquesta`.

## Identidad del target canónico

Target revisado:

```text
/srv/fabrica/orquesta/home/revisiones/o3c-p6-cap021-runtime-tmp-v4-4840207-target
```

Se verificó:

- directorio `orquesta:orquesta`, UID/GID `999:982`, modo `0700`;
- `HEAD` exacto
  `484020703c683c324e9b2eaef5c43a56c1d95ea6`;
- árbol exacto
  `408ff618672f8faab56cbc46f33f7233c380f5f4`;
- `HEAD` desacoplado y estado Git vacío;
- sin remotos configurados;
- ausencia de `.git/objects/info/alternates`, `.git/commondir` y
  `.git/shallow`;
- conductor con el SHA-256 autorizado;
- 33 rutas publicadas coincidentes byte a byte con el target: 32 fuentes Go y
  el único fixture shell.

## Paquete canónico

Paquete postcanónico:

```text
/srv/fabrica/orquesta/home/evidencias/o3c-p6-cap021-runtime-tmp-v4-4840207-canonica-r1
```

Identidad física observada:

```text
dispositivo:inode:UID:modo = 2049:2267315:999:700
tipo                       = directorio real
propietario                 = orquesta:orquesta
nlink                       = 2
```

La identidad coincide exactamente con `huella_origen` en
`publicacion.tsv`. El directorio padre es real, propiedad de `orquesta`, modo
`0700`. No se observaron ACL ampliadas en padre, paquete o entradas, ni
capacidades Linux en el helper.

El paquete contiene exactamente 12 entradas, todas ficheros regulares, sin
subdirectorios, enlaces simbólicos ni entradas adicionales:

| Entrada | Modo | `nlink` | Bytes | SHA-256 |
| --- | ---: | ---: | ---: | --- |
| `SHA256SUMS` | `0600` | 1 | 885 | `4ebe2573983ed9edecc83f86f9e639507f39b013b9976778d3179edca02d1330` |
| `bf_directos.tsv` | `0600` | 1 | 2.229 | `ef861710f7194a9b0ee4926fc6f22dc1f0c2ec9c012770e398e61de2f0f1a13b` |
| `binarios.tsv` | `0600` | 1 | 159 | `5aaef81951ec3f96c5a5858c49930dbd477b9bdde101458c1ef9c0279efbafbd` |
| `casos.tsv` | `0600` | 1 | 66.649 | `fb0236774afeb5d4384bf65151738687f80aaa251dd0a5e8deeb5a96b2156dbe` |
| `contexto.tsv` | `0600` | 1 | 1.358 | `43ae7b55935f23125fe8d4d7f815a27e797d64a6519575cbe7499b3df5609121` |
| `fuentes.tsv` | `0600` | 1 | 6.583 | `5dd85475d86e0287727c2dbf1830e48b6fa2b6c71c49b64e70e85d728f6c8cfe` |
| `publicacion.tsv` | `0600` | 1 | 262 | `9ac56d4d43b8054914debaebfac0bd9a8b431700937d3ada5339b1ca996e785d` |
| `rename_noreplace` | `0700` | 1 | 3.015.841 | `08b3ed3abefd234d249db0c73ab1016ad255b3f06958d6411660cf72e46a4173` |
| `residuos.txt` | `0600` | 1 | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `resumen.txt` | `0600` | 1 | 1.591 | `2b2fcab73880ae79ec6f2a0e60084e966d63fd14147c7456806b9e3a26ef2aae` |
| `tmpdir_selectores.tsv` | `0600` | 1 | 117.855 | `6e4af8d125c98bc21bef5ca890175ebea92154dca5701f5ff52d3378c59c2149` |
| `utilidades.tsv` | `0600` | 1 | 4.676 | `13af73948e08055c7536c22dcb0311a418af44d4931560976b11fc94423b8742` |

`SHA256SUMS` contiene exactamente 11 registros, uno por cada entrada sellada
distinta del propio manifiesto. No hay nombres ausentes, duplicados o
adicionales. `sha256sum -c SHA256SUMS`, ejecutado como `orquesta`, devolvió
`OK` para los 11 artefactos.

## Evidencia funcional relevante para seguridad

`resumen.txt` registra `resultado=GO`, el conductor exacto, Go 1.26.5, cero
residuos y publicación sin reemplazo.

`contexto.tsv` liga conjuntamente:

```text
HEAD inicial/snapshot/publicación = 484020703c683c324e9b2eaef5c43a56c1d95ea6
tree inicial/snapshot/publicación = 408ff618672f8faab56cbc46f33f7233c380f5f4
estado Git                        = limpio
EUID                              = 999
SHA conductor                     = 8918ef3b2ba7b343ff95468a95483e676ad18991715a8fd048bc716de5cb15df
SHA fuentes                       = 5dd85475d86e0287727c2dbf1830e48b6fa2b6c71c49b64e70e85d728f6c8cfe
```

La auditoría semántica de los ledgers produjo:

| Evidencia | Resultado |
| --- | --- |
| Casos ordinarios | 244: 122 normal y 122 race |
| Estado y salida ordinaria | 244 con estado 0, `PASS\n` de 5 bytes y stderr 0 |
| Cierres fatales directos | 6: 3 normal y 3 race |
| Estado y salida fatal | 6 con estado 65, stdout 0, stderr 0, EOF y no retorno |
| Selectores acreditados | 1.472: 736 normal y 736 race |
| Enlace ejecución-selector | 250 claves esperadas, 250 observadas, 0 discrepancias |
| Duplicados o huérfanos | 0 |
| Filas `NO-GO` | 0 |
| Deltas FD/hijos/zombis/grupos/temporales | 0 |
| Contenedores runtime retirados | todos |
| FD ambientales cerrados | todos |
| Fuentes | 32 Go y un fixture, 0 discrepancias de SHA |
| Residuos finales | 0 |

Los 1.470 selectores declarados por casos ordinarios y los dos declarados por
cierres fatales suman exactamente las 1.472 filas publicadas. Todas acreditan
UID 999, modo `0700`, identidad física, inicio vacío, retirada, ausencia por
`lstat`, raíz exterior vacía y resultado `GO`.

Las 38 procedencias enumeradas en `utilidades.tsv` —29 utilidades privadas,
Go, siete ejecutables de la frontera C y `fallo_durable.sh`— conservaban
propietario, modo y SHA registrados al revisar. Esto es una comprobación
postcanónica de coherencia, no una nueva ejecución.

Los temporales exactos de la corrida quedaron ausentes:

```text
/var/tmp/o3c-p6.NJGSHV
/var/tmp/o3c-p6-tools.OFl6xW
```

## Publicación atómica y ausencia de reemplazo

El conductor prepara un directorio temporal `0700` hermano del destino y fija
la identidad física del padre y del origen. El destino debe estar ausente
antes de preparar y antes de invocar el helper.

La revisión de `conductor.sh:523-702` comprobó:

1. el padre debe ser un directorio real, no simbólico, del EUID y modo `0700`;
2. `destino_padre_fd` conserva un descriptor al padre y se compara con su
   huella `dispositivo:inode:UID:modo`;
3. el helper se abre y se valida por descriptor antes de ejecutarlo;
4. el hijo cierra todos los descriptores ambientales y conserva únicamente
   `stdin`, `stdout`, `stderr` y `destino_padre_fd`;
5. el helper recibe nombres base, no rutas arbitrarias de origen o destino;
6. no existe fallback a `mv`, `rename`, copia, reemplazo o reintento;
7. cualquier estado distinto de cero se traduce a una categoría cerrada y
   mantiene el destino sin publicar.

El helper Go embebido en `conductor.sh:555-645`:

- abre padre, origen, manifiesto y entradas mediante descriptores;
- añade `O_CLOEXEC|O_NOFOLLOW` a cada `openat`;
- exige padre y origen directorios reales del EUID y modo `0700`;
- compara las huellas físicas esperadas de padre y origen;
- reabre por ruta el padre con `O_NOFOLLOW` y contrasta su identidad con el
  descriptor pinado;
- enumera exactamente estas 12 entradas y rechaza cualquier otra;
- exige cada entrada regular, del EUID, `nlink=1` y
  `Mode&07777 == 0600`, salvo el helper, que debe ser `0700`;
- ancla el SHA de `SHA256SUMS`;
- verifica los 11 hashes de contenido;
- cierra y comprueba cada descriptor de fichero;
- cierra y comprueba `pathParent` y `origin` antes del rename;
- llama a `renameat2` con `RENAME_NOREPLACE`;
- termina inmediatamente mediante `os.Exit(0)` tras el retorno verde.

La taxonomía cerrada distingue:

```text
B_PRECONDICION
H_VALIDACION
X_REAPERTURA
X_CIERRE_FD
X_EJECUCION
X_ESTADO
X_LIMPIEZA
R_EEXIST
R_ENOENT
R_EXDEV
R_NO_SOPORTE
R_AUTORIDAD
R_OTRO
GO
```

No se filtran stderr internos, rutas temporales ni errores del sistema. Una
colisión posterior a la precondición se convierte en `R_EEXIST`; no se
interpreta como éxito ni permite reemplazar la entrada concurrente.

## Correspondencia del binario publicado

El helper publicado tiene SHA-256:

```text
08b3ed3abefd234d249db0c73ab1016ad255b3f06958d6411660cf72e46a4173
```

Este valor coincide entre el fichero, `publicacion.tsv` y `SHA256SUMS`.

Metadatos observados:

```text
ELF 64-bit LSB executable, x86-64
enlazado estáticamente
Go 1.26.5
CGO_ENABLED=0
GOOS=linux
GOARCH=amd64
trimpath=true
sin sección dinámica ni intérprete ELF
```

El desensamblado de `main.main` confirmó materialmente:

- llamadas a `syscall.Close` antes de publicar;
- carga de `0x13c`, número 316 de syscall en Linux/AMD64;
- bandera `1`, correspondiente a `RENAME_NOREPLACE`;
- llamada a `syscall.Syscall6`;
- clasificación del error si el retorno no es cero;
- llamada inmediata a `os.Exit(0)` en la rama verde.

No se encontró una ruta compilada alternativa de sobrescritura.

## Cierre de descriptores y exposición de datos

`cerrar_fd_excepto` y `hijo_fd` cierran en cada hijo todo descriptor igual o
superior a tres salvo la capability expresamente declarada. El descriptor
interno con el que Bash lee el conductor permanece únicamente en el padre; el
subshell final no lo transmite al helper.

Antes de ejecutar el helper:

- se cierran `go_fd` y `publicador_fallo_fd`;
- se vacía la tabla de comandos;
- se retira y acredita la ausencia de `tool-runtime`;
- staging ya está retirado;
- el helper queda pinado en el padre y se ejecuta a través de
  `/proc/$pid_conductor/fd/N`;
- la copia heredada de ese descriptor se cierra en el hijo;
- el helper recibe únicamente el descriptor del directorio padre además de
  los tres descriptores estándar.

El paquete privado conserva rutas locales de GOROOT, GOTOOLDIR y de la
autoridad de `fallo_durable.sh`. Están dentro de un directorio `0700` y
ficheros `0600`; no son credenciales ni se publicaron externamente. El escaneo
de los artefactos de datos no encontró claves privadas, tokens, contraseñas,
DSN, claves de acceso o rutas `/root`.

No se observaron capacidades Linux ni ACL ampliadas. `getfattr` no estaba
disponible para una enumeración general de atributos extendidos; `getcap` sí
comprobó específicamente la ausencia de capacidades en el ejecutable. Esta
omisión no altera el dictamen dentro de la autoridad Unix declarada.

## Rutas de ataque examinadas

| Riesgo | Resultado |
| --- | --- |
| Destino ya existente antes de publicar | rechazo en precondición |
| Destino creado durante la ventana final | `renameat2` devuelve `EEXIST`; no reemplaza |
| Padre sustituido por enlace o inode distinto | rechazo por `O_NOFOLLOW` y huella |
| Origen sustituido por enlace | rechazo por `O_NOFOLLOW` |
| Entrada no regular | rechazo |
| Hardlink de una entrada | rechazo por `nlink != 1` |
| Bits especiales o modo incorrecto | rechazo por `Mode&07777` |
| Entrada extra, ausente o duplicada | rechazo por inventario exacto |
| Datos o helper adulterados | rechazo por SHA |
| Manifiesto adulterado | rechazo por SHA anclado |
| Reapertura fallida | `X_REAPERTURA` |
| Cierre de descriptor fallido | `X_CIERRE_FD` |
| Ejecución 126/127 | `X_EJECUCION` |
| Estado inesperado | `X_ESTADO` |
| Retirada del runtime fallida | `X_LIMPIEZA` |
| Fuga de stderr o ruta interna | suprimida; solo categoría cerrada |
| Fallback de publicación | inexistente |
| Operación falible tras rename verde | inexistente; `os.Exit(0)` inmediato |

Dentro del modelo cooperativo declarado no se encontró una ruta de
sobrescritura, sustitución, publicación parcial, fuga persistente o aceptación
de evidencia incompleta.

## Hallazgos

| Prioridad | Número | Estado |
| --- | ---: | --- |
| P0 — crítico | 0 | sin hallazgos |
| P1 — mayor | 0 | sin hallazgos |
| P2 — menor | 0 | sin hallazgos |

Resultado conjunto:

```text
P0=0
P1=0
P2=0
```

## Puertas reproducidas en solo lectura

```text
AGENTS.md aplicables completos                         GO
rama, HEAD, árbol, padre y limpieza                    GO
write-set revisor único y ruta inicialmente ausente   GO
SHA conductor y enmienda                              GO
bash -n conductor.sh                                  GO
ShellCheck conductor.sh                               GO
git diff --check 7782e467..48402070                   GO
inventario físico exacto del paquete                  GO
propietario/modos/tipos/nlink                         GO
ACL y capacidades Linux                               GO
SHA256SUMS, 11/11                                     GO
244 casos ordinarios                                  GO
6 cierres fatales                                     GO
1.472 selectores y enlaces con ejecuciones            GO
33 rutas fuente contra target                         GO
38 procedencias de herramientas                       GO
helper: formato ELF y metadatos Go                    GO
helper: sección dinámica/intérprete ausentes          GO
helper: símbolos y desensamblado renameat2             GO
residuos y temporales de la corrida                    cero
escaneo proporcional de literales sensibles           sin coincidencias
revalidación final de target y paquete                 GO
```

No se reejecutaron el conductor, el helper, la matriz dinámica, builds de
producto ni la conducción canónica. Esas operaciones habrían creado
temporales o cambiado estado y estaban prohibidas por el alcance
postcanónico de solo lectura. Tampoco se ejecutaron Go global, PostgreSQL,
Docker, HTTP, navegador, integración continua remota ni pruebas de extremo a
extremo.

## Comandos principales de solo lectura

```bash
W=/srv/fabrica/revisiones/o3c-p6-cap021-runtime-tmp-v4-seguridad-20260821
T=/srv/fabrica/orquesta/home/revisiones/o3c-p6-cap021-runtime-tmp-v4-4840207-target
P=/srv/fabrica/orquesta/home/evidencias/o3c-p6-cap021-runtime-tmp-v4-4840207-canonica-r1

find "$W" -type f -name AGENTS.md -print
sha256sum /srv/fabrica/AGENTS.md "$W/AGENTS.md"

GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$W" -C "$W" rev-parse HEAD
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$W" -C "$W" rev-parse 'HEAD^{tree}'
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$W" -C "$W" symbolic-ref --short HEAD
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$W" -C "$W" status --short --untracked-files=all
git -c safe.directory="$W" -C "$W" show -s --format='commit=%H%ntree=%T%nparent=%P%nsubject=%s' HEAD
git -c safe.directory="$W" -C "$W" diff-tree --no-commit-id --name-status -r HEAD

GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$T" -C "$T" rev-parse HEAD
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$T" -C "$T" rev-parse 'HEAD^{tree}'
GIT_OPTIONAL_LOCKS=0 git -c safe.directory="$T" -C "$T" status --short --untracked-files=all
git -c safe.directory="$T" -C "$T" remote -v

sha256sum \
  "$W/tools/o3c_p6_conductor/conductor.sh" \
  "$W/docs/portal_vec/enmienda_o3c_p6_cap_normal_021_runtime_tmp_2026-08-21.md"

stat -Lc '%F|%U:%G|%u:%g|%a|%h|%d:%i|%s|%n' "$P" "$P"/*
find "$P" -mindepth 1 -maxdepth 1 -printf '%P|%y\n'
sha256sum "$P"/*
runuser -u orquesta -- env -i LC_ALL=C PATH=/usr/bin:/bin \
  bash --noprofile --norc -c 'cd "$1" && sha256sum -c SHA256SUMS' bash "$P"

awk -F '\t' '...' "$P/casos.tsv"
awk -F '\t' '...' "$P/bf_directos.tsv"
awk -F '\t' '...' "$P/tmpdir_selectores.tsv"
awk -F '\t' '...' "$P/fuentes.tsv"

file "$P/rename_noreplace"
go version -m "$P/rename_noreplace"
readelf -lW "$P/rename_noreplace"
readelf -dW "$P/rename_noreplace"
nm -n "$P/rename_noreplace"
objdump -d --disassemble='main.main' "$P/rename_noreplace"

getfacl -p /srv/fabrica/orquesta/home/evidencias "$P" "$P"/*
getcap -r "$P"

bash -n "$W/tools/o3c_p6_conductor/conductor.sh"
shellcheck "$W/tools/o3c_p6_conductor/conductor.sh"
git -c safe.directory="$W" -C "$W" diff --check \
  7782e4679e546cde4d693633911d5ec3a47bf85b..484020703c683c324e9b2eaef5c43a56c1d95ea6

rg -n -i --glob '!rename_noreplace' \
  '(BEGIN [A-Z ]*PRIVATE KEY|AKIA[0-9A-Z]{16}|postgres(ql)?://|password[[:space:]]*=|secret[[:space:]]*=|token[[:space:]]*=|root@|/root/)' \
  "$P"
```

Los `awk` representados de forma abreviada comprobaron número exacto de
columnas, cardinalidad, distribución normal/race, estados, E/S, inventarios,
identidad temporal, descriptores, selectores, duplicados, huérfanos, SHA del
target y resultado final.

## Límites conservados

Este GO no amplía el modelo de amenaza:

1. `orquesta` es una autoridad cooperativa. Otro proceso que ya ejecute con
   UID 999 puede modificar objetos `0700/0600` de ese mismo UID. V4 no crea
   aislamiento entre procesos con igual identidad Unix.
2. El paquete no se declara de solo lectura ni está firmado frente al propio
   UID 999. Los hashes prueban coherencia e integridad dentro de la autoridad
   cooperativa, no inmutabilidad criptográfica frente a ella.
3. El nombre aleatorio del origen previo al rename no queda conservado en el
   paquete. La identidad física publicada coincide con la huella pre-rename y
   los temporales conocidos están ausentes; esto es suficiente dentro del
   modelo cooperativo, no frente a manipulación posterior del mismo UID.
4. El launcher confiable debe entregar un entorno inicial sin
   `LD_PRELOAD`, `LD_AUDIT` o `LD_LIBRARY_PATH`. V4 no atesta lo consumido por
   el cargador dinámico antes de iniciar Bash.
5. La compilación race fija siete ejecutables C por ruta y SHA, pero no atesta
   transitivamente cabeceras, objetos, bibliotecas, cargador o sysroot del
   host.
6. Git se observó sin alternates, common-dir o shallow en el target, pero este
   corte no ejecuta `fsck` ni declara atestado completo del almacén de objetos.
7. La evidencia prueba esta corrida única y este SHA. No constituye una prueba
   estadística de estabilidad ni autoriza repetir un destino consumido.
8. No se acreditan PostgreSQL, Docker, SQL, red, interfaz, accesibilidad,
   despliegue, producción, datos reales o la aplicación completa.

Estos límites están declarados en la enmienda y no contradicen la capability
acotada revisada.

## Dictamen y relevo

**GO de seguridad postcanónica, P0=0, P1=0, P2=0.**

El commit `484020703c683c324e9b2eaef5c43a56c1d95ea6` acredita, dentro de la
autoridad operativa cooperativa declarada:

- paquete canónico íntegro y privado;
- inventario cerrado;
- propietario, tipos, modos y enlaces exactos;
- hashes completos;
- cierre ambiental de descriptores;
- helper pinado;
- publicación atómica local mediante
  `renameat2(RENAME_NOREPLACE)`;
- ausencia de fallback, reemplazo o reintento;
- clasificación cerrada de errores;
- salida inmediata tras éxito;
- staging, runtime y residuos de la corrida retirados.

Tarea: `O3C-P6-CAP021-RUNTIME-TMP-V4-SEGURIDAD-POSTCANONICA`.

Estado: `GO`, `P0=0`, `P1=0`, `P2=0`.

Commit revisado:
`484020703c683c324e9b2eaef5c43a56c1d95ea6`.

Commit del acta: pendiente de dirección; debe tener como padre exacto el
candidato revisado.

Archivo revisor único:
`docs/portal_vec/revisiones/revision_seguridad_o3c_p6_cap_normal_021_runtime_tmp_v4_2026-08-21.md`.

Pruebas ejecutadas: exclusivamente comprobaciones estáticas y de evidencia en
solo lectura.

Pruebas omitidas: conductor, helper dinámico, builds, producto, PostgreSQL,
Docker, red, extremo a extremo e integración continua; omitidas por el alcance
postcanónico de solo lectura.

Seguridad y privacidad: GO dentro de los límites declarados; sin credenciales,
datos personales reales ni publicación externa.

Internacionalización y accesibilidad: no aplican al write-set documental ni a
la publicación local de evidencia; no se modificó interfaz.

Siguiente tarea desbloqueada: dirección puede aplicar únicamente esta acta,
revalidar write-set y padre, y crear el commit documental de la rama revisora.
Después deberá contrastar el dictamen funcional independiente del mismo
candidato. Cualquier integración o publicación requiere una orden separada y
expresa.

Revisión independiente: seguridad, completa y postcanónica.
