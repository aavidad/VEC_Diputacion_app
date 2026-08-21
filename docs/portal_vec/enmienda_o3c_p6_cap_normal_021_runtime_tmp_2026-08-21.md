# Enmienda O3c P6 CAP_NORMAL_021: attestation del runtime temporal V4

Fecha: 21 de agosto de 2026.

Tarea: `O3C-P6-CAP021-RUNTIME-TMP-V4-FIXTURE-PASS`.

Estado: la conducción canónica de `7782e467` terminó `NO-GO` tras validar todos
los casos y antes de publicar. Los intentos `9749ddd`, `9ec119f`, `9b5b97d`,
`a47a7a7` y `7782e467`, junto con sus destinos, están consumidos. Esta corrección
local solo autoriza pruebas estáticas y probes desechables del publicador como
`orquesta`, sin build/test de producto, commit ni conducción. Requiere revisión
funcional y de seguridad independiente; no autoriza integración, CI remota,
despliegue, producción o métricas. El probe histórico Go/C sigue consumido.

## Orden de dirección: cierre mínimo FIXTURE-PASS

Capability: `O3C-P6-CAP021-RUNTIME-TMP-V4-FIXTURE-PASS`.

Invariante: el snapshot cerrado contiene las 32 fuentes Go compilables y un
fixture runtime byte-exacto; cada ejecución ordinaria verde produce `PASS\n`
—cinco bytes en stdout y cero en stderr— y cada BF conserva 0/0. Todo el resto
de V4 mantiene oráculos, cardinalidades, aislamiento, limpieza y publicación.

Write-set exacto de esta corrección:

```text
tools/o3c_p6_conductor/conductor.sh
docs/portal_vec/enmienda_o3c_p6_cap_normal_021_runtime_tmp_2026-08-21.md
```

La capability es publicar atómicamente y sin reemplazo el paquete GO validado.
Conserva target, resultados, producto, casos, fuentes y métricas; retira staging
y `tool-runtime` antes del rename. Sigue revisión independiente de los dos
archivos congelados; este corte no autoriza commit ni conducción.

El hallazgo NUL invalida `2021ab059b44cd9879a638119aa8a08129abbf0dc1be5c03049d8a2fe1215eb5`
y `0a0522260574b01bf4db5f1a3cbf724d36363752a7c4a6662fbafe97b1536b6c`:
Bash descarta NUL y `read` no acredita bytes binarios. También invalida
`7647c9a3a45a82980639884b1afed11bd602569a892fa576c2a4a6e2e1407c23` y
`3d669296b16c30190c61778094bd96b838fa0ff4b99efe497762ff736500170a`,
que aún usaban sustitución de comando para el digest ASCII.

El candidato vigente conserva `stdout_bytes=5` y entrega por pipeline a la
copia privada de `sha256sum --status -c -` un checklist con el literal de
`PASS\n`,
`c26de83abdc9496cd1301470918ec39ecca1cf389ef0ae1c6504da1800d1c431`.
Ni los bytes ni el digest se cargan en una variable Bash. No crea referencia
temporal ni conserva descriptor en el padre: el pipeline termina y cierra sus
extremos antes del predicado, y el wrapper privado cierra FD `>=3` antes de
`exec`. El probe adversarial `P\0ASS`, también de cinco bytes, da SHA-256
`d51d490384e39c795194ab773ffc525e162dff431773e2e115d1501bc7a87ad3`
y queda rechazado.

El mismo hallazgo exige que cada copia del snapshot sea regular, no symlink,
propiedad del EUID, modo 0400, `nlink=1` y SHA exacta. Una rutina única vuelve a
acreditar las 33 rutas tras los builds y después de la última ejecución, antes
de preparar o publicar. Esta lectura final reduce la ventana accidental, pero
no ofrece aislamiento frente a una sustitución transitoria por otro proceso
con el mismo UID 999; `orquesta` continúa siendo autoridad cooperativa, no una
frontera entre procesos de igual UID.

## Base y autoridad

La base exacta es
`14c1f31079e466a82b8e1d390168078973cc6e05`, con árbol
`1c736bcb555326841485a1f0c0486626ad6a5038`. Conserva el publicador durable
V2 de `9391243f98d232a8eb78ed20ba5306e31b864bad` y sus dos revisiones
independientes de alcance filesystem:

- funcional `19ed5f03ed7905f56dabaed883d771fbfb2bd7e7`;
- seguridad `d9629d787c351ac5592ea091535407ca8bdad75f`.

Esos GO no revocaron el rojo histórico `CAP_NORMAL_021`, los dos NO-GO de
O3A-V5-CND-V3 ni el P1 C21/toolchain. Esta enmienda tampoco los convierte en
GO. V4 acredita únicamente el vínculo entre las 32 fuentes compiladas, el
fixture runtime único, el checkout, el `TMPDIR` exacto de cada ejecución y el
paquete local publicado.

## NO-GO consumido y requisito del próximo clon

La única corrida de `9749ddd3ee1ecbd2e3a2c9db0d3eb9123d2ceb7c` terminó antes de
staging con `NO-GO base ausente` y exit 2 por `dubious ownership`: el checkout
era `root:root` modo 0755 al ejecutarse como `orquesta`. No hubo destino,
paquete de evidencia, staging ni procesos de build o test; la consulta Git
previa a la base falló por propiedad dudosa. Ese SHA no se repetirá.
El siguiente target debe ser un clon local nuevo, limpio, modo 0700 y propiedad
de `orquesta`; el conductor debe comprobar UID/modo y la sonda Git antes de
consultar la base, emitiendo `NO-GO checkout Git no acreditable` con el
diagnóstico real si falla. No se usa `safe.directory` global, fetch, pull ni
red.

La única corrida de `9ec119f679a2f00a3cf7f77fbd01bc0271141c89` terminó después de
crear el staging privado, con `NO-GO utilidad cat` y exit 1: `type -P cat` resolvió `/usr/bin/cat`,
un symlink propiedad de root hacia `/usr/lib/cargo/bin/coreutils/cat`; el
conductor había creado staging/evidencia parcial, que la trampa retiró, y no
hubo destino, build ni test. Ese SHA y su destino no se repetirán. La
capability correctora exige resolver cada utilidad con `realpath -e` y acreditar
solo la ruta canónica absoluta regular, no symlink y ejecutable, con su SHA-256;
cualquier resolución o validación fallida cierra en NO-GO. El bootstrap mínimo
previo a la atestación resuelve nombres desde el PATH fijo; no se afirma que
esas consultas iniciales queden pinadas retroactivamente. Tras resolverlas,
cada objeto se copia byte a byte a un `tool-runtime` privado 0700 con su basename
exacto, modo 0500, huella y validación; las llamadas por nombre se fijan a esas
copias para conservar la semántica de applets multicall. El runtime se retira
antes del helper final mediante el `rm` bootstrap y no hay utilidades después
del rename.

La única conducción real de
`9b5b97d011ed9c86de731deace5329bf6c555994` hacia
`/srv/fabrica/orquesta/home/evidencias/o3c-p6-cap021-runtime-tmp-v4-9b5b97d-canonica-r1`
terminó con exit 1 tras crear el `tool-runtime` y antes de resolver target o
crear staging, build, test o destino. La copia privada de `find` emitió
`Failed to restore initial working directory: /root: Permission denied`: el
launcher inició el conductor con cwd `/root`, inaccesible al EUID `orquesta`, y
el conductor no lo sustituyó antes de su primera ejecución externa. El trap
retiró el `tool-runtime`; target permaneció limpio, el destino siguió ausente y
no apareció staging nuevo. Ese SHA y ese destino están consumidos y no se
repiten.

La corrección posterior, antes de cualquier comando externo, completa primero
los rechazos y limpiezas builtin ya definidos, ejecuta `hash -r` y después usa
`builtin cd /`, comprobando también `PWD=/`. Si el cambio de cwd o su identidad
lógica fallan, termina cerrado con `NO-GO cwd inicial`; ningún bootstrap ni
applet privado hereda ya el cwd inaccesible entregado por el launcher.

El probe ligero, sin conductor ni creación de temporales, lanzó `find` como
`orquesta` desde `/root` y reprodujo el control con exit 1 y el mismo error de
restauración. La variante que ejecutó primero `builtin cd /`, comprobó `PWD=/`
y lanzó después el mismo `find` terminó `GO`. No alcanzó target, staging, Go,
build, test o destino y no autoriza otra conducción.

La corrida canónica del commit
`a47a7a7a25eb31e1476c82a0131ff3cce187017a` contra el target
`/srv/fabrica/orquesta/home/revisiones/o3c-p6-cap021-runtime-tmp-v4-a47a7a7-target`
y el destino nuevo
`/srv/fabrica/orquesta/home/evidencias/o3c-p6-cap021-runtime-tmp-v4-a47a7a7-canonica-r1`
quedó consumida en `NO-GO C01_ENTRADA`: estado 1, stdout 512 bytes y stderr
cero. El paquete durable existe. La causa exacta fue que
`TestAutoridadO3cConsumeHandoffO3bReal` no pudo abrir en el snapshot
`deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh`.
Ese SHA y ese destino no se repiten.

La única conducción canónica de `7782e4679e546cde4d693633911d5ec3a47bf85b`
contra `/srv/fabrica/orquesta/home/revisiones/o3c-p6-cap021-runtime-tmp-v4-7782e46-target`
completó 244/244 casos, 6/6 BF directos y 1472/1472 atestaciones de selectores,
sin filas `NO-GO`. Retiró staging y `tool-runtime`, pero terminó exit 1 con
`NO-GO publicacion GO no acreditada`; el destino
`/srv/fabrica/orquesta/home/evidencias/o3c-p6-cap021-runtime-tmp-v4-7782e46-canonica-r1` y los
temporales propios quedaron ausentes y el target siguió limpio en ese commit; intento, commit y destino están consumidos.

La causa exacta precedió al helper: el padre cerró el FD 255 del conductor;
Bash reabrió su entrada en otro FD y el segundo inventario devolvió 2. El helper
no llegó a `exec` ni llamó `renameat2`. El wrapper hijo conserva solo stdio y
`destino_padre_fd`; el padre mantiene su FD interno y el helper pinado.

El criterio independiente adicional no sustituye esa causa. El helper antiguo
aceptó dos entradas 0600 hardlinkadas con `nlink=2`; además, precondición Bash,
reapertura y huella inválida colisionaban en 2, ejecución daba 127 y `EEXIST`
17. La variante usa `B_PRECONDICION`; `H_VALIDACION`; `X_REAPERTURA`,
`X_CIERRE_FD`, `X_EJECUCION`, `X_ESTADO`, `X_LIMPIEZA`; y
`R_EEXIST/R_ENOENT/R_EXDEV/R_NO_SOPORTE/R_AUTORIDAD/R_OTRO`. Los estados
internos 70–77 y 2/126/127/otro se traducen sin exponer stderr ni rutas. Las 12
entradas exigen regular, EUID, `nlink=1` y `Mode&07777` 0600, salvo helper 0700.

La autoridad funcional directa para el cierre mínimo es el código de
`crearFixtureO3aM38`/`leerRunnerPruebaO3aM38`, el runner citado y la evidencia
histórica `tools/o3c_p6_conductor/evidencia/casos.tsv`. La inspección estática
demuestra que la prueba solo abre ese fixture, exige fichero regular de 1 a 64
KiB, copia sus bytes a un runner privado y entrega el contenido al hijo. En el
modo hijo, el runner ejecuta solo builtins, lee el FD 9 y se auto-detiene antes
de alcanzar herramientas externas, Docker o auxiliares. Por ello no se copia
ningún otro elemento de `deploy`. En HEAD `a47a7a7` el fixture es Git 100755,
46123 bytes y SHA-256
`7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487`;
permanece fuera del write-set.

La frontera bootstrap es exacta: `/usr/bin/{realpath,stat,sha256sum,mktemp,install,rm}`
root-owned y con cadena de directorios no escribible. El runtime privado contiene
29 applets con basename exacto; Go y el publicador se declaran por ruta absoluta,
se pinan por FD y conservan su huella. La frontera bootstrap no se acredita
retroactivamente como parte del runtime.

## Hallazgo funcional paralelo sobre los hashes iniciales

El hallazgo paralelo es una lista de riesgos sobre los hashes iniciales del
conductor
`d94545dbdeaedb348f2735e7b9ddc0c4f925735ee8d18d8abf23de79134360e4`
y de esta enmienda
`1a605c26c195ae1df5c69b5c73a62e1eda728b16b774f0807f19c71b7adcc72b`;
no es una aprobación ni un dictamen sobre este candidato. Obliga a cerrar y
volver a revisar independientemente cuatro puntos:

- la lista cerrada contiene 29 utilidades, no 28: `basename` también es una
  utilidad externa y queda copiada, acreditada y fijada dentro del runtime;
- el hash inicial del conductor ejecutaba herramientas externas antes de completar la
  transición. El candidato usa directamente el intérprete regular
  `/usr/bin/bash` en modo de arranque `-p`, sin el salto previo por
  `/usr/bin/env`, y exige ese modo con un builtin. Así Bash no procesa
  `BASH_ENV` ni importa funciones antes del rechazo; una vez iniciado, solo
  builtins limpian el ambiente y las únicas ejecuciones anteriores al runtime
  privado son las seis rutas bootstrap cerradas. Target, destino, candado,
  Git, staging, evidencia y Go se resuelven después de la transición;
- el hash inicial reabría y ejecutaba el publicador por su ruta. El candidato
  compara identidad física de ruta y FD al abrirlo, acredita y conserva ese FD,
  y todas sus ejecuciones usan Bash privado sobre `/proc/$$/fd`, con `env -i` y
  `PATH` privado;
- el hash inicial no acreditaba toda la cadena de padres bootstrap. El candidato
  recorre hasta `/` tanto la cadena del alias `/usr/bin/...` como la del objeto
  canónico, y exige en cada directorio propietario root y ausencia de escritura
  para grupo/otros.

Estas correcciones son afirmaciones verificables del productor y permanecen
pendientes de revisión funcional y de seguridad; no levantan ningún NO-GO.

## Segundo dictamen de seguridad sobre los hashes iniciales

El segundo dictamen es `NO-GO`. Se refiere a los mismos hashes iniciales
`d94545db…` y `1a605c26…`, complementa el hallazgo funcional anterior y no
aprueba los bytes posteriores. Identifica y obliga a volver a revisar estos
riesgos de seguridad:

- cada descriptor que pina el origen de una de las 29 utilidades debe cerrarse
  inmediatamente después de acreditar la copia, y el cierre debe comprobarse;
- toda consulta Git debe atravesar la copia privada con `env -i`, hogares
  privados, configuración de sistema anulada, global nulo, locks opcionales
  desactivados, atributos de sistema anulados y hooks, fsmonitor, diff externo
  y textconv anulados;
- el binario `go` no basta como frontera: deben fijarse y comprobarse antes y
  después de todo uso de Go el árbol GOROOT completo, `11536` ficheros y SHA
  `b53ebeab1542ea933c6f995a2bcf862d505cb8343ad2b0d1f7a7de3238157ae6`,
  y los ocho ficheros de GOTOOLDIR, SHA
  `1061bd99d16310f8f549e375a5c0cb18a79d66441ca0ed4dee60f70fde633f9b`;
- la carrera debe fijar `CC`, `CXX`, `cc1`, `collect2`, `as`, `ld` y
  `lto-wrapper` por ruta canónica, propietario/modo, padres y SHA, y declarar
  que cabeceras, objetos, bibliotecas y sysroot C del host siguen fuera de la
  atestación;
- ningún hijo debe heredar FD ambientales: utilidades, Git, Go, builds y
  publicador reciben solo stdio; el test recibe además únicamente su marcador,
  `flock` y el helper final únicamente el FD del padre que necesitan;
- el helper Go debe cerrar y comprobar todos los FD propios antes de
  `renameat2`; después del éxito termina inmediatamente con `os.Exit(0)`, sin
  `defer`, cierre ni otra operación falible posterior.

El candidato incorpora esos cierres como afirmaciones del productor. El
dictamen de seguridad permanece `NO-GO` hasta revisar independientemente los
nuevos hashes exactos.

### Autoridad operativa del publicador

La futura conducción se ejecuta con EUID `orquesta` (`999`), pero el conductor
y `fallo_durable.sh` se consumen desde este worktree de autoridad y son objetos
`root:root` 0755. Por ello el publicador no se compara con el EUID: se exige UID
root, modo 0755, fichero regular sin symlink, SHA exacta e igualdad de identidad
física entre ruta y FD. Su cadena `o3c_p6_conductor/` → `tools/` → raíz
canónica del worktree debe ser enteramente root-owned y no escribible por
grupo/otros. Esa raíz es el ancla de esta autoridad; los directorios externos
que alojan `.worktrees` pertenecen a `orquesta` y no se presentan como
root-owned. El UID `orquesta` es una autoridad operativa cooperativa para
target, temporales y destino, no una frontera adversarial frente a otro proceso
que ya ejecute con el mismo UID: ese proceso conserva capacidad Unix
equivalente sobre los objetos 0700/0600 de `orquesta` y queda fuera del modelo
cerrado por V4.

La prueba operativa no canónica ejecutada como `orquesta` usó un clon local
desechable de `9ec119f`, limpio, `999:982` modo 0700, y un destino nuevo. Una
fuente del ledger se marcó `skip-worktree` y se retiró para imponer un punto de
parada posterior al publicador y anterior a build. El conductor atravesó las
fronteras Go y C, acreditó el publicador root y terminó después con estado 1,
stdout 0 y `NO-GO fuente ...autoridad.go`; no creó destino ni dejó variación de
staging o `tool-runtime`. El clon se eliminó. No fue una conducción canónica.
Un intento de preparación anterior no alcanzó el conductor por `dubious
ownership`, sin añadir `safe.directory`; otro se detuvo antes del publicador al
detectar que la salida alias de `gcc -print-prog-name=ld` debía canonicalizarse.
Ninguno creó destino, build o test.

Tras el presente dictamen se ejecutó únicamente un probe aislado del wrapper,
sin target, staging, build, test ni destino. Un Bash pinado en el padre fue
abierto por el hijo mediante `/proc/$pid_conductor/fd/N` después de cerrar los
FD heredados: la variante Git/Go/publicador observó solo `0,1,2`, y la variante
del helper observó solo `0,1,2` más el FD explícito de un directorio padre. Las
dos aserciones terminaron `GO`; son evidencia focal del productor, no una
conducción canónica ni una aprobación del candidato.

## Dictamen independiente de seguridad sobre el corte endurecido

El dictamen independiente de seguridad es `NO-GO` sobre los hashes exactos
`02e27debd570ef3cc77b9df14dece0d0a5f8dde7a3d5e1cf93d454508ecda969`
del conductor y
`e5ce0dddbcd72828008704bdcefd680490fdb6c491940c1942f6a3d111d8cbe2`
de esta enmienda. No aprueba ningún byte posterior y exige cerrar:

- P1 Git: acreditar antes de `status`/`diff` un `$target/.git` exacto,
  canónico, directorio sin symlink, del EUID y sin escritura de grupo/otros;
  auditar con `git config --local --no-includes` todas las claves locales y
  rechazar sin sensibilidad a mayúsculas `include.*`, `includeIf.*`,
  `filter.*`, `core.worktree`, `core.attributesFile` y
  `extensions.worktreeConfig`; exigir `--show-toplevel=$target`,
  `--absolute-git-dir=$target/.git`, `info/attributes` ausente o regular vacío,
  y valor efectivo `unspecified` de `filter` para cada ruta entregada por
  `ls-files -z` a `check-attr -z --stdin`;
- P1 destino: rechazar inmediatamente después de canonicalizar los argumentos
  una evidencia igual a target o descendiente suyo, incluida `.git`;
- P2 lanzamiento: declarar que el propio `exec` inicial solo es admisible desde
  un launcher confiable/root que ya haya retirado `LD_PRELOAD`, `LD_AUDIT` y
  `LD_LIBRARY_PATH`; el `env -i` interior llega después de cargar Bash y no
  protege ese primer límite dinámico.

El candidato posterior usa ficheros NUL privados para conservar y comprobar de
forma directa los estados de `config`, `ls-files` y `check-attr`; no los oculta
en process substitution ni ejecuta filtros durante la auditoría. Estas son
afirmaciones del productor y el dictamen independiente permanece `NO-GO` hasta
revisar los nuevos hashes.

Los probes estáticos posteriores, sin conductor ni creación de temporales,
confirmaron que la forma exacta de `config`/`ls-files`/`check-attr` es aceptada
por el Git instalado, que las seis familias prohibidas se rechazan también con
mayúsculas mezcladas y que target, `.git` y cualquier descendiente activan el
predicado de separación mientras un prefijo vecino no lo activa. Estos probes
no acreditan un repositorio candidato, no ejecutan una conducción y no cambian
el `NO-GO` independiente.

## Dictamen independiente funcional adicional

El dictamen funcional adicional es `NO-GO` sobre los hashes viejos exactos
`939eebc2f059d286a7b2258c5b25d9840366beaaebd802e3dc534bddc9bb75db`
del conductor y
`5186fb189ebdd7c68cf4391374b41e33f7470ab3d22a5c50a0a74e1242aeda12`
de esta enmienda. No aprueba los bytes posteriores y registra tres hallazgos:

- P1: `ejecutar()` podía decidir `GO` sin acreditar la salida ordinaria. La
  corrección provisional de aquel dictamen propuso 0/0, pero la autoridad
  histórica posterior demuestra que un testbin Go verde produce exactamente
  `PASS\n`: cinco bytes en stdout y cero en stderr. El candidato vigente mide
  ambos una sola vez, exige esos bytes exactos y reutiliza tamaños en fila y
  diagnóstico; BF conserva 0/0;
- P2: con `set -e`, el trap podía abandonar las retiradas restantes tras el
  primer `rm` fallido. El candidato posterior captura el estado original,
  desarma `EXIT` para evitar recursión, intenta siempre
  `temporal_publicacion_go`, `tool-runtime` y staging, acumula fallos y conserva
  un NO-GO original; si el estado original era cero y falla alguna retirada,
  termina no cero;
- P2 documental: la regla general limita los probes al tramo anterior a
  staging/Go, mientras el acta también conserva un probe que cruzó Go/C y
  publicador. La cronología correcta es que dirección autorizó explícitamente
  esa única excepción histórica para verificar la autoridad root del
  publicador; quedó consumida antes de esta edición y no modifica la prohibición
  vigente ni convierte aquel probe en conducción canónica.

Las correcciones son afirmaciones verificables del productor. El dictamen
funcional adicional permanece `NO-GO` hasta una nueva revisión independiente
de los hashes congelados.

Los probes estáticos de aquel candidato comprobaron que `ejecutar()` contenía
solo los dos cálculos `wc` y que fila y diagnóstico reutilizaban `so`/`se`; su
expectativa ordinaria 0/0 queda sustituida por la evidencia `PASS\n` 5/0. Un
primer arnés de limpieza fue inválido por
quoting y no ejecutó la aserción; la repetición válida, enteramente en memoria,
confirmó tres intentos en orden y los resultados `original=7+fallo → 7`,
`original=0+fallo → 2` y `original=0+limpieza verde → 0`. No se invocaron el
conductor, staging, Go, build, test ni destino.

## Segunda revisión independiente de seguridad

La segunda revisión independiente de seguridad emite `NO-GO` sobre los hashes
viejos exactos
`1df4840d5ae40c49ceedc8646c362704e58a35155fb52118cea78acd81638f7d`
del conductor y
`45f28d0ee204d21bfbb61b9e9fbeac3e53fd21cf08fc0ddc4180837546ea4850`
de esta enmienda. No aprueba bytes posteriores y exige cerrar:

- P1 índice: obtener registros NUL con `git ls-files -t -v -z`, aceptar
  exclusivamente el tag exacto `H `, derivar de esos registros la lista NUL
  para `check-attr` y rechazar `skip-worktree`, `assume-unchanged`, unmerged o
  cualquier estado no normal. Tags y lista exacta deben revalidarse en cuatro
  cortes: inicial, tras snapshot, antes de publicación y tras retirar staging;
- P1 genealogía: `info/grafts` solo puede estar ausente o ser regular vacío sin
  symlink; `.git/shallow` y `.git/commondir` deben estar totalmente ausentes, y
  `--path-format=absolute --git-common-dir` debe resolver exactamente al
  `$target/.git` ya acreditado;
- P1 lazy fetch: toda consulta debe fijar `GIT_NO_LAZY_FETCH=1`, y la
  configuración LOCAL cruda debe rechazar `extensions.partialClone`,
  `remote.*.promisor` y `remote.*.partialCloneFilter`. Un objeto ausente debe
  producir error local; no se autoriza transporte, fetch ni helper remoto;
- toda consulta Git debe añadir `-c core.fileMode=true` y conservar los cierres
  previos de entorno, atributos, filtros, reemplazos y diff.

El candidato posterior centraliza los cuatro cortes en una única auditoría
fail-closed: conserva la salida etiquetada, genera una lista NUL separada,
comprueba cada `H ` y compara las listas posteriores byte a byte con la inicial.
Solo la lista derivada alimenta `check-attr`. Estas son afirmaciones del
productor; el dictamen permanece `NO-GO` hasta revisar los nuevos hashes.

El cierre lazy-fetch impide la recuperación prometida en las consultas del
conductor y no se ejecuta ningún comando de transporte. No equivale a `fsck`,
no acredita exhaustivamente el object store ni sus alternates y no protege
frente a mutación cooperativa por otro proceso UID 999.

Los probes estáticos, sin conductor ni temporales, confirmaron que el Git
instalado acepta common-dir absoluto y `ls-files -t -v -z`; el índice del
worktree de autoridad produjo 3101 registros `H `. Registros sintéticos `S`,
`h`, `M`, `R`, `C` y `?` fueron rechazados, y las tres nuevas familias de
configuración se detectaron con mayúsculas mezcladas. La inspección mecánica
confirmó exactamente cuatro llamadas y que cada una precede a su `status`.
Esto prueba sintaxis y predicados, no atestigua el futuro clon ni levanta el
`NO-GO`.

## Tercera revisión independiente de seguridad

La tercera revisión independiente de seguridad emite `NO-GO` sobre los hashes
viejos exactos
`72e675e2324ecf940d79c0f320a62231be78744ffb349b6cc462a1ed10ada313`
del conductor y
`f560f856e28323c86978fef5af3b0df18fbe09479115d893210034352cb4f105`
de esta enmienda. No aprueba bytes posteriores y exige:

- P1 caché estadística: todas las consultas Git deben fijar
  `core.fileMode=true`, `core.trustctime=true`, `core.checkStat=default`,
  `core.ignoreStat=false` y `core.untrackedCache=false`; la auditoría LOCAL
  normalizada debe rechazar explícitamente las cuatro últimas claves para que
  la prueba no dependa de configuración persistente ambigua;
- P2 documental: no puede afirmarse que todo hijo hereda solo stdio. Cero FD
  ambientales cruzan `exec`, pero las capabilities explícitas se distinguen:
  `flock` conserva `destino_padre_fd`, el wrapper de test conserva
  temporalmente su marcador y el helper final conserva `destino_padre_fd`;
  Git, Go y publicador sí reciben exclusivamente stdin/stdout/stderr.

El candidato posterior añade los cuatro overrides en el wrapper único Git y
las cuatro claves normalizadas a la lista cerrada de rechazo. Corrige asimismo
el criterio FD sin relajar el cierre ambiental. Son afirmaciones del productor;
el dictamen permanece `NO-GO` hasta revisar los nuevos hashes.

Los probes estáticos confirmaron que el Git instalado devuelve los cinco
valores forzados exactos, que las cuatro claves de caché se rechazan también
con mayúsculas mezcladas y que ya no existe la frase excesiva «cada hijo hereda
solo». La inspección documental conserva por separado stdio, FD ambientales y
las tres clases de capability explícita. No se invocaron conductor, staging,
Go, build, test ni destino; los probes no levantan el `NO-GO`.

Existe una precondición externa anterior al paso 1: el launcher confiable/root
debe entregar al `exec` inicial un ambiente sin `LD_PRELOAD`, `LD_AUDIT` ni
`LD_LIBRARY_PATH`. V4 no acredita al launcher, no neutraliza lo que el cargador
dinámico pudiera haber consumido antes de iniciar Bash y no es autosuficiente
frente a esa frontera.

El orden V4 es causal dentro de esa precondición externa:

1. solo builtins rechazan `BASH_ENV`, `ENV`, `CDPATH`, cualquier función o
   alias heredado, retiran todo `GIT_*` heredado y ejecutan `hash -r`;
2. el bootstrap acredita propietario root del alias, objetivo canónico regular
   root, modo sin escritura de grupo/otros y todos sus directorios padres;
3. crea el `tool-runtime` 0700 antes de resolver target, destino, candado, Git,
   staging o evidencia;
4. copia con modo 0500 y basename exacto `basename bash cat chmod cmp cp cut dirname env
   find flock git grep install ln mkdir mkfifo mktemp mv realpath rm seq setsid
   sha256sum sleep sort stat timeout wc`; acredita las 29 entradas, sus modos,
   propietarios y hashes, cierra y verifica inmediatamente cada FD fuente,
   fija `hash -p` y contrasta cada `hash -t`;
5. solo entonces canonicaliza los dos argumentos y rechaza evidencia igual o
   descendiente de target antes de resolver su padre o candado. Exige target
   0700 y `$target/.git` canónico exacto, no symlink, del EUID y sin escritura
   de grupo/otros; `--show-toplevel`, `--absolute-git-dir` y el common-dir
   absoluto deben devolver exactamente target y `$target/.git`, mientras
   `.git/commondir` y `.git/shallow` deben estar ausentes. Antes de
   `merge-base`, `info/grafts` está ausente o es regular vacío sin symlink.
   Antes del primer `status`/`diff` audita
   las claves LOCAL crudas con `config --local --no-includes`, normalizadas en
   locale C, y aplica la lista cerrada de rechazo, incluidas configuración
   partial-clone y remotos promisor. Exige `info/attributes` ausente o regular
   vacío. `ls-files -t -v -z` solo admite tags `H ` y de ellos deriva la lista
   NUL cuyo `filter` efectivo debe ser `unspecified`; tags y lista byte-exacta
   se revalidan en los cortes inicial, post-snapshot, pre-publicación y
   post-staging. Los errores de `config`, `ls-files`, `check-attr` o `cmp`, una
   terna truncada/excedente o cualquier discrepancia fallan cerrados. Cada Git
   añade `-c core.fileMode=true`, `-c core.trustctime=true`,
   `-c core.checkStat=default`, `-c core.ignoreStat=false` y
   `-c core.untrackedCache=false`; las cuatro últimas claves también se
   rechazan en la configuración LOCAL normalizada. Git se ejecuta con `env -i`,
   `HOME`/`XDG_CONFIG_HOME` privados,
   `GIT_CONFIG_NOSYSTEM=1`, `GIT_ATTR_NOSYSTEM=1`, global `/dev/null`,
   `GIT_OPTIONAL_LOCKS=0`, `GIT_NO_REPLACE_OBJECTS=1`,
   `GIT_NO_LAZY_FETCH=1`, hooks en `/dev/null`, fsmonitor desactivado y
   `diff.external` vacío; las ocho comprobaciones `diff` llevan conjuntamente
   `--no-ext-diff --no-textconv`.
   `GIT_NO_REPLACE_OBJECTS` impide la sustitución por `refs/replace` en estas
   consultas; no equivale a `fsck`, no atestigua el object store ni protege
   contra otro proceso cooperativo del mismo UID;
6. después de crear staging/evidencia abre por descriptor únicamente
   `/srv/fabrica/orquesta/home/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.5.linux-amd64/bin/go`,
   SHA-256 `8da5fd321795754b994c64e3eb8a5a14ff47bd285559a7e876f3c79abafc67f9`,
   exige versión `go1.26.5`, `GOROOT` y `GOTOOLDIR` derivados exactos y
   contrasta las huellas/cardinalidades transitivas anteriores antes del primer
   uso y después de compilar el helper final;
7. abre por descriptor `fallo_durable.sh`, regular, no symlink, UID root,
   modo 0755, con todos sus padres root y sin escritura de grupo/otros hasta
   la raíz canónica del worktree de autoridad, y SHA-256
   `b8f91102a2e98ce1e2e79ed73bfa9bd48c5d1f8ca002271dc5fc2461512bf174`;
   lo ejecuta con Bash privado, `env -i`, `PATH` privado y el objeto pinado en
   `/proc/$pid_conductor/fd`, no mediante reapertura de la ruta;
8. una frontera común cierra todo FD `>=3` antes de cada `exec`; solo conserva
   el FD expresamente declarado para `flock`, el marcador del test o el padre
   del helper. Git, Go y publicador reciben exclusivamente `0,1,2`: sus
   ejecutables pinados se abren mediante `/proc/$pid_conductor/fd/N` después de
   que el wrapper hijo haya cerrado la copia heredada del FD. El wrapper de
   test acredita el cierre y retira también el marcador antes de `exec` del
   binario;
9. antes del rename final ya no existen staging ni `tool-runtime`; en el padre
   quedan `0,1,2`, su descriptor interno de lectura del conductor,
   `destino_padre_fd` y `helper_fd`. El subshell `hijo_fd` cierra sus copias
   ambientales y el helper ejecutado recibe solo `0,1,2` y `destino_padre_fd`;
   cierra sus FD `origin` y `pathParent` y, tras el rename, hace `os.Exit(0)`.

La build normal queda enteramente ligada al Go/GOROOT/GOTOOLDIR anteriores.
La build `race` necesita CGO: usa `PATH` privado, fija `CC`/`CXX` absolutos y
`COMPILER_PATH`, y acredita la procedencia de siete ejecutables C root-owned
por ruta y SHA. La frontera C permanece externa: fijar esa procedencia no
acredita transitivamente cabeceras, objetos, bibliotecas, cargador ni sysroot
del host. V4 registra
`race_c_tcb=ejecutables_7_fijados_sysroot_host_no_atestado`; no incluye esos
ejecutables entre las 29 utilidades copiadas ni afirma una atestación completa,
un cierre total de la frontera C o el cierre del parche/toolchain C21.

## V3: GO productor revocado por revisión

La base V4 `14c1f31` obtuvo una única corrida productora verde y durable:
244 casos, seis BF, cien capturas normal y cien race, Go 1.26.5 y residuos
cero. Su `SHA256SUMS` es
`f1edf1a274843221a12c56f5cfc1abe9efd4bb4b460888f8948aa9699e26877b`.
Ese GO productor no es un dictamen y no se repetirá.

Las revisiones funcional y de seguridad independientes emitieron `NO-GO`
sobre ese SHA exacto. Al iniciar V4 sus actas todavía no estaban presentes
como objetos Git locales; los hallazgos transmitidos por dirección son:

- P1 funcional: la fila medía el contenedor `runtime_tmp` después de borrarlo,
  pero no el `TMPDIR` efectivo `runtime_aislado/tmp` antes de retirar el
  contenedor;
- P1 seguridad: la publicación verde usaba `mv` simple, sin no-replace ni
  identidad física;
- P1 seguridad: existía una ventana entre el hash de las fuentes del target y
  su reapertura posterior para compilar;
- P2 seguridad: el helper no acreditaba UID efectivo y modo 0700;
- P2 seguridad: utilidades y FD ambientales no quedaban acreditados.

V4 no convierte esos NO-GO en GO ni abre O3A-V4, C21, toolchain u O4.

## Candidatos rojos preservados y atribución corregida

El primer candidato `4d2951f83057490390c4e4e25927c5b06e0f868c` recibió una
única corrida canónica. Terminó `NO-GO` en `C17_OWNERS`, normal, estado 1,
stdout 200, stderr 0, grupo ausente e inventarios
`6/6,0/0,0/0,0/0,0/0`. El publicador V2 conservó el paquete exacto bajo la
raíz privada de evidencias del usuario `orquesta`; `SHA256SUMS` es
`a5c4434ee7e04c3877641ed1b1d6d2c63e3e66bfc43bb77ca8708249419fda64`.
No se repitió esa corrida.

El paquete sellado conserva solamente el fallo exterior y la línea que mide
la ejecución interior: `err=exit status 2`, stdout 48 y stderr 1927. No
conserva los 48 o 1927 bytes interiores, por lo que no acredita su contenido.
La primera atribución a una doble liberación del hilo fue una inferencia del
código —`defer runtime.UnlockOSThread()` local junto al `testing.Cleanup` de
`autoridadRealBarreraO3bPruebaM38`—, no una conclusión contenida en el raw.

El segundo candidato `6407fad5feb91330be3425c8feaea3fd31886460`
retiró el `defer` y recibió una sola focal agregada de los siete selectores.
Terminó también `NO-GO`, estado 1, stdout 200, stderr 0, con la misma medición
interior 48/1927. Su paquete tiene `SHA256SUMS`
`94e5fdca7f0de9297032f389e87c286042ec74ad4a7954293e69d04aacce42f2`.
No recibió corrida canónica y no se repitió la focal. Este resultado refuta la
atribución a la doble liberación y obliga a retirarla.

La causa estructural visible es el `testing.Cleanup` heredado:
`consolidarHandoffO3bM38` entrega la custodia y fija
`autoridadCapturaO3bM38.custodia=nil`, mientras el cleanup registrado por
`autoridadRealBarreraO3bPruebaM38` dereferencia después esa custodia. El
retorno normal activa ese cleanup fuera de su precondición. Este corte no
modifica el helper O3b ajeno: restaura los dos `os.Exit(0)` originales del
hijo y el `defer runtime.UnlockOSThread()` original. La nueva capacidad de
retirada temporal pertenece al padre y ocurre después de `cmd.Run`, con
independencia de que el hijo termine mediante `os.Exit`.

Una sonda anterior, ejecutada fuera del aislamiento canónico, heredó un FD
ambiental y falló antes de alcanzar los selectores. Se conserva como intento
inválido y no cuenta como GO, NO-GO ni evidencia causal.

Son autoridad funcional directa:

- la decisión O3c, en particular C21 y C22;
- el conductor O3c P6 y su ledger de 32 fuentes;
- las dos revisiones NO-GO de O3A-V5-CND-V3;
- la enmienda y las dos revisiones del publicador durable V2.

## Capability y criterio único

La única capability vigente es
`O3C-P6-CAP021-RUNTIME-TMP-V4-FIXTURE-PASS`, una atestación acotada bajo la
autoridad operativa cooperativa `orquesta`: el snapshot privado contiene las
32 fuentes Go exactas y el único fixture runtime byte-exacto; el checkout
mantiene HEAD, árbol y limpieza; cada selector usa un `TMPDIR` privado de UID
efectivo y modo 0700; y el paquete GO se publica localmente sin reemplazo,
conservando identidad dev:inode. No añade aislamiento frente a procesos con el
mismo UID.

Cada selector aislado de `TestHandoffO3cP5CasosAislados` recibe un directorio
runtime privado. Go elimina por completo el selector `runtimeSelector` y acredita
su `Lstat=ENOENT`; Bash mide las entradas del `TMPDIR` exterior antes y después
del hijo, registra también los residuos que existen antes de la limpieza, y
retira solamente esas entradas conservando el directorio y su identidad física.
La atestación TSV se precrea fuera del `TMPDIR` del hijo,
congela su huella dev:inode/UID/modo y se filtra de su entorno; cualquier
sustitución de esa ruta invalida el agregado. Después de `Lstat=ENOENT` del selector, el padre
acredita que la raíz exterior queda vacía; la retirada del contenedor se
acredita en una columna separada, también cuando el hijo termina con estado
fatal 65. Solo Bash retira después las entradas del `TMPDIR` exterior y luego
el contenedor; no se atribuye esa limpieza a Go.

## Invariantes

Para cada uno de los siete selectores:

| Selector | Estado esperado |
| --- | ---: |
| `positivo` | 0 |
| `retirada` | 0 |
| `retirada_terminal` | 0 |
| `reuso` | 65 |
| `particion` | 65 |
| `retirada_sin_ref` | 65 |
| `retirada_plazo` | 65 |

Se exige conjuntamente:

1. los hijos selectores internos conservan salida 0/0; el testbin ordinario que
   los contiene termina con `PASS\n` exacto, stdout de cinco bytes y stderr
   cero, mientras los BF directos conservan 0/0;
2. entradas del `TMPDIR` efectivo parten de cero; el preconteo de residuos se
   conserva por selector, el padre los retira, y el conteo final es cero antes
   de retirar el contenedor;
3. ningún hijo, zombi o grupo residual;
4. la limpieza temporal la realiza el padre después de `cmd.Run` y no depende
   de `defer` o `testing.Cleanup` en el hijo;
5. los recorridos verdes conservan sus `os.Exit(0)` originales para no activar
   un cleanup O3b cuya autoridad ya fue transferida;
6. el contenedor runtime queda retirado y acreditado separadamente;
7. cada selector acredita dev/inode/UID/modo, entradas iniciales, residuos
   pre-limpieza, `Lstat=ENOENT` y raíz exterior vacía en `tmpdir_selectores.tsv`;
8. las fuentes compiladas y el fixture son regulares, no symlink y
   byte-exactos, del EUID, modo 0400 y `nlink=1`; las 32 fuentes se leen
   únicamente desde el snapshot privado y el único fixture queda disponible
   allí solo para el consumo runtime citado;
9. HEAD, árbol y limpieza del target son iguales antes y después del snapshot;
10. ningún FD ambiental `>=3` cruza `exec`. Git, Go y publicador reciben solo
   stdin/stdout/stderr; como capabilities explícitas, `flock` conserva
   `destino_padre_fd`, el wrapper de test conserva temporalmente el marcador y
   lo cierra antes del binario, y el helper final conserva `destino_padre_fd`;
11. la publicación usa helper pinado y `renameat2(RENAME_NOREPLACE)`; ancla
   `SHA256SUMS`, revalida por FD las 12 entradas como regulares, EUID,
   `Mode&07777` 0600 —helper 0700—, `nlink=1` y SHA, y contrasta ruta y FD del padre;
12. ningún estado se reintenta, tolera, reclasifica, salta o decide por
   mayoría.

La corrida canónica conserva además 244 casos, seis BF directos, cien
capturas normal y cien race, 1.472 filas de atestación de selectores (cinco
agregados y cien capturas por modo, más C17 por modo), FD/hijos/zombis/grupos/temporales delta cero y
el toolchain literal `go version go1.26.5 linux/amd64`.

## Diseño acotado

### Arnés Go

El padre valida que `TMPDIR` sea un directorio absoluto real, vacío, propiedad
del UID efectivo y de modo 0700. Antes de cada selector crea una raíz con la
misma autoridad, reemplaza de forma explícita `HOME`,
`TMPDIR`, `GOTMPDIR` y `O3C_P5_CASO` en el entorno hijo, espera su estado,
contiene descendientes y retira la raíz. Solo entonces comprueba estado y
salidas.

Los dos puntos de éxito conservan sus `os.Exit(0)` originales y la fijación de
hilo conserva su `defer` original. El proceso padre no delega en esas salidas
la limpieza temporal: espera el estado, contiene descendientes y retira la
raíz privada por igual en éxito y en fatal 65.

El BF directo de partición usa un padre test-only que ejecuta una sola vez el
selector fatal, acredita y retira su temporal privado y solo entonces termina
él mismo en 65. La variable del BF no se hereda al selector, evitando recursión
y conservando estado 65, EOF y salida 0/0.

### Conductor

Compilación y ejecución tienen temporales distintos. Una única rutina cerrada
copia al snapshot 0700 las 32 fuentes del ledger y el fixture literal, siempre
con sus rutas relativas. Rechaza ruta absoluta o con ascenso, duplicados,
origen o copia no regular/symlink y cualquier hash distinto. `archivos[]`
conserva solo las 32 rutas Go; `evidencia/fuentes.tsv` enumera las 33 rutas,
`sha_target` cubre esas 33 y el rehash posterior al build recorre esa evidencia
para revalidar también el fixture. Cada copia y cada revalidación exigen además
UID efectivo, modo 0400 y un único enlace. La misma función vuelve a recorrer
las 33 rutas después de la última ejecución y antes de publicación. Revalida
HEAD, árbol y limpieza después de copiar y compila normal y race exclusivamente
desde las 32 fuentes del snapshot. El corte final no elimina la carrera frente
a otro proceso UID 999 cooperativo que sustituya transitoriamente una ruta.

Cada invocación recibe un entorno vacío y cerrado con `HOME`, `TMPDIR`,
`GOTMPDIR`, `GOROOT`, `GOENV=off` y `GOTOOLCHAIN=local` explícitos. El proceso
intermedio cierra todos los FD `>=3` antes de `exec`; el lock nunca llega al
target. Las rutas y huellas SHA-256 de las utilidades externas se conservan en
la evidencia. Build normal, ejecución y publicador usan solo el `tool-runtime`;
race añade únicamente la frontera C absoluta y fijada descrita arriba, cuyo
sysroot host queda declarado, no acreditado.

Los ledgers `casos.tsv` y `fuentes.tsv` se copian al inicio a entradas privadas
0400; la compilación, ejecución, rehash y contexto solo leen esas copias. La
fila registra tanto el inventario del contenedor como el del `TMPDIR`
efectivo antes/después, el preconteo de residuos por selector, su acreditación,
la retirada del contenedor y el cierre de FD ambientales. El publicador V2
recibe sin cambios el primer fallo.
El padre crea una FIFO 0600 con extremos separados, desvincula su nombre antes
de lanzar, cierra su escritor tras fork, el wrapper conserva solo el escritor,
comunica `si`, lo cierra y reenumera por symlink que no deja ningún FD>=3 antes
de `exec`; el lector padre exige esa única línea y EOF.
Un GO se prepara en un temporal 0700 hermano
del destino; el helper se compila con el Go fijado mientras el staging existe.
Después se retira staging, se revalida target, se escriben contexto y sello y el
helper invoca `renameat2(RENAME_NOREPLACE)`. Antes acredita inventario, SHA,
huellas, UID, modos y `nlink=1`, y cierra sus FD propios. `os.Exit(0)` sigue
inmediatamente al rename verde, sin `defer`, cierre u operación falible; no se
afirma modo de solo lectura ni aislamiento frente a otro proceso UID 999.

## Capability, invariante y write-set exacto

Capability: `O3C-P6-CAP021-RUNTIME-TMP-V4-FIXTURE-PASS`, atestación acotada del
runtime temporal, el fixture único y las herramientas que construyen, ejecutan
y publican la evidencia local bajo la cooperación de la autoridad operativa
`orquesta`; no ofrece aislamiento frente a otro proceso con UID 999. Conserva
el anclaje builtin del cwd inicial a `/` antes de cualquier ejecución externa.

Invariante: ninguna decisión de capacidad posterior al bootstrap usa un
resultado ambiental interno; el cwd heredado del launcher se descarta mediante
`builtin cd /` antes del primer comando externo. Destino y target son disjuntos, Git queda ligado
al `.git` exacto sin configuración local activa capaz de incluir, filtrar o
redireccionar worktree/atributos o declarar partial clone/promisor; genealogía
no usa grafts, shallow ni common-dir redirigido. Cada tracked conserva tag `H `,
la misma lista NUL en los cuatro cortes y `filter=unspecified`; lazy fetch y
replace objects quedan anulados. `fileMode`, `trustctime`, `checkStat`,
`ignoreStat` y `untrackedCache` quedan fijados en cada Git y las cuatro claves
de caché se rechazan localmente. En cada corte, `git status` debe terminar cero
antes de evaluar su salida vacía; un error Git nunca se interpreta como
limpieza. El snapshot cerrado contiene 32 fuentes Go compilables y el fixture
runtime único byte-exacto, regular, no symlink, EUID:0400 y `nlink=1`, también
en la revalidación posterior a la última ejecución. Cada fila ordinaria exige
estado cero, stdout de cinco bytes con la SHA-256 literal de `PASS\n` y stderr
cero; cada BF directo exige 0/0.
Snapshot, Go, helper, `TMPDIR`, cierre de FD y publicación pertenecen después a
la misma cadena fail-closed; el trap intenta
todas las retiradas, preserva el error original y convierte en error cualquier
fallo de limpieza tras un resultado originalmente verde. Esta invariante
empieza tras la precondición del launcher externo y no afirma control sobre el
cargador dinámico inicial, todo el object store ni sus alternates.

```text
tools/o3c_p6_conductor/conductor.sh
docs/portal_vec/enmienda_o3c_p6_cap_normal_021_runtime_tmp_2026-08-21.md
```

`handoff_test.go`, `fuentes.tsv`, las 32 fuentes, producto, G7a/G7b,
`fallo_durable.sh`, casos, ledgers O3a/O3b/O3c, workflows, estado transversal
y métricas permanecen byte a byte.

## Puertas y secuencia sin reintentos

Matriz ligera como `orquesta`, sin conductor ni conducta de producto:

| Probe | Resultado cerrado exigido |
|---|---|
| positivo | GO, rename único, origen ausente |
| destino ya existente | `B_PRECONDICION`, destino intacto |
| carrera tras precondición | `R_EEXIST`, origen y destino intactos |
| hardlink/modos especiales/adulteración/helper/huella | `H_VALIDACION`, sin destino |
| reapertura/cierre/exec/estado/limpieza | `X_REAPERTURA`/`X_CIERRE_FD`/`X_EJECUCION`/`X_ESTADO`/`X_LIMPIEZA`, sin destino |

Se añaden `bash -n`, ShellCheck, `git diff --check`, líneas, write-set, hashes y
cero residuos.

El siguiente corte es la revisión funcional y de seguridad independiente de
los dos archivos y hashes congelados, incluido el publicador corregido. El
productor no emite GO. Solo después,
y mediante orden expresa de dirección, podrá existir un commit candidato, un
clon `orquesta:orquesta` 0700 nuevo y una única corrida canónica a un destino
nunca usado. Si termina roja, el publicador V2 conserva el primer paquete y
ese SHA no se repite; si queda verde, la evidencia del mismo SHA vuelve a
revisión independiente antes de cualquier integración.

## Límites

El corte usa únicamente la evidencia histórica que fija `PASS\n` 5/0 para las
filas ordinarias y 0/0 para BF; no borra residuos o evidencias previas ni relaja
cardinalidades, plazos, oráculos o contención.
No acredita la estabilidad C21 ni el parche de toolchain, no abre O4 y no
autoriza datos reales, red, SQL, Docker, PostgreSQL, push, despliegue,
producción o credenciales.
