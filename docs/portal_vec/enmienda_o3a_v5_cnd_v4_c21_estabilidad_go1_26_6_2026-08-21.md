# Enmienda O3a V5 CND V4 C21: estabilidad unánime con Go 1.26.6

Fecha: 21 de agosto de 2026.

Tarea: `O3A-V5-CND-V4-C21-ESTABILIDAD-GO1.26.6`.

Estado: **CANDIDATO DOCUMENTAL RECONGELADO, PENDIENTE DE REVISIÓN FUNCIONAL
Y DE SEGURIDAD INDEPENDIENTES**. Este documento no ejecuta el conductor, no
acredita estabilidad, no corrige ni reclasifica un rojo histórico y no
autoriza commit, integración, convergencia, publicación, CI remota, O4,
despliegue, producción, credenciales, datos reales, estado transversal o
métricas.

## Resultado único del contrato

La capability que este contrato permitirá juzgar es:

```text
O3A-V5-CND-V4-C21-ESTABILIDAD-GO1.26.6
```

Consiste exclusivamente en obtener evidencia durable y unánime de tres
carriles predeclarados del conductor O3a V5, bajo la toolchain exacta Go
1.26.6, sobre el mismo futuro commit documental, mediante:

1. un carril productor;
2. un carril de revisión funcional;
3. un carril de revisión de seguridad.

Cada carril usa un clon local nuevo y limpio, propiedad del usuario Unix
`orquesta`, y un destino de evidencia nuevo. Después del preflight conjunto,
cada carril invoca exactamente una vez el conductor. No hay retry, segunda
muestra, mayoría, selección del mejor resultado, tolerancia, `SKIP`, cambio de
plazo ni sustitución de un carril.

La capability solo puede recibir `GO_UNANIME_REVISADO` cuando los tres
carriles son verdes y las dos revisiones independientes posteriores aceptan
el conjunto exacto. Un solo rojo, ausencia, interrupción o discrepancia
invalida todo el SHA. Dos verdes nunca compensan el tercero.

## Base exacta y dependencias cerradas

La base técnica de este contrato es:

| Propiedad | Valor |
| --- | --- |
| Commit | `484020703c683c324e9b2eaef5c43a56c1d95ea6` |
| Padre | `7782e4679e546cde4d693633911d5ec3a47bf85b` |
| Árbol | `408ff618672f8faab56cbc46f33f7233c380f5f4` |
| Asunto | `audit(o3c): corregir publicacion atomica V4` |

Las cuatro actas dependientes existen como objetos Git. Son commits hermanos,
hijos directos de `484020703c683c324e9b2eaef5c43a56c1d95ea6`, y cada uno
añade una sola acta:

| Autoridad | Commit | Blob de su única acta | Dictamen |
| --- | --- | --- | --- |
| O3c V4 funcional | `bb23278e0b5660b611cb9c7faa19b197e0283a19` | `b677ab3ad1427ce6c10f1725e2fab7bed3b69039` | `GO`, `P0=P1=P2=0` |
| O3c V4 seguridad | `22fde5f72b2ce4bf92c2ae32bee4d7ab47a4e36b` | `445617e8e35d6d21e996ea5b380871b2c0844151` | `GO`, `P0=P1=P2=0` |
| C21 cierre funcional | `baea7d014e403412dd77c7cbd380e89023dd9717` | `edf90770b9f98d3a03fdb842b4b6c070ca3cbbcc` | `GO`, `P0=P1=P2=0` |
| C21 cierre seguridad | `fd44303eaf5dd27d2fbbd221266b13fe1473b4ba` | `1c1d70ed2ae8d877e8af2ba90879577aa2bbec01` | `GO`, `P0=P1=P2=0` |

Las actas V4 acreditan prospectivamente `CAP_NORMAL_021` y
`CAP_RACE_021`. Las actas C21 permiten consumir esa puerta posterior sin
repetir, compensar ni borrar la ejecución roja de V3. Ninguna de las cuatro
actas afirma estabilidad estadística ni cierre global del toolchain.

## Rojo histórico no compensable

El commit
`74f249587d5c0da41092a2c017ccd6cb54817248` permanece histórico y rojo.
Sus revisiones funcional y de seguridad conservaron la ejecución C21 race
fallida en el índice 44, con estado 66, stdout y stderr vacíos, FD del
conductor iguales y residuos cero. Dos ejecuciones verdes no la compensaron.

También permanece histórico el `NO-GO` de
`fea52f3ddf796991c93c85cae992ca695db1ae63`: su única puerta O3c terminó
roja en `CAP_NORMAL_021`. La evidencia posterior V4 cierra su dependencia
para un corte documental nuevo, pero no convierte ese SHA en candidato verde.

Este contrato no reejecuta ninguno de esos SHAs. Un futuro triple verde
demostrará únicamente la estabilidad exigida para el nuevo SHA documental;
no reescribirá los hechos anteriores.

## Invariante de bytes técnicos inmóviles

El futuro commit documental debe conservar byte a byte el árbol técnico de
`4840207`. Como mínimo se vuelven a acreditar antes y después de cada carril:

| Autoridad viva | Modo | Líneas | SHA-256 |
| --- | ---: | ---: | --- |
| G7a, pruebas de arranque | `100644` | 750 | `2d5fc67e6aff0f6e11305e9d92690452f7ffb194b4bcef22cd2dd9121235bbb3` |
| G7b, pruebas adversas | `100644` | 744 | `5ec6be1ccb917ec2908bfa75d2d947ae7a107c39044239166cc10e1067d7677e` |
| `tools/o3a_v5_conductor/fuentes_v5.tsv` | `100644` | 11 | `ba2b0a1c9838f57ca43d53ec6133ae74bfa7452f7511421e367d7fc5f3d079b0` |
| `tools/o3a_v5_conductor/conductor.sh` | `100755` | 205 | `cc1f47f8d4dd49df71effd4886de732748a261f9fc1964a4c524037d5e4be0d7` |
| `tools/o3a_v5_conductor/README.md` | `100644` | 242 | `7933b7617a92e22fddf6f888b9fddcc74ea0224616e0340d9c9f36d7d2533ce8` |
| `go.mod` | `100644` | 45 | `a447f8cf5217130ed336bf78bcd93211659e1c7ba547673e9a1753ef6ba7eeca` |
| `Dockerfile` | `100644` | 253 | `6045bd15bf061de0db0e70c3858344b48db39fc31dbe9fc1f76c6e397f81bf5d` |

Los siete bloques externos también permanecen inmóviles:

| Bloque | Líneas | SHA-256 |
| --- | ---: | --- |
| `conductor_c01_c07.sh` | 61 | `9236cd1d7ca6cb1ebfbfb511c3913b96be980ba28acebeedad91e2f70deaedc3` |
| `conductor_c02_interleavings.sh` | 24 | `67d993b84c3c337a230a53f90d12ed91c08021c8ef88279cc93fd8557659f78a` |
| `conductor_c08_c11_c14.sh` | 83 | `a07c561ad3a616a59256f3bcb317cf803c3c0a0b01f6a09eabad3cfb8614329e` |
| `conductor_c10_c13.sh` | 98 | `70e2413081711d3f7019fa5daa4f5172d2ddde2b6eb5f2b121dc7d82f478cd46` |
| `conductor_c15_c21.sh` | 60 | `edf6a719eed6c96f920265c8c51df16b458d172744cb6cf06b581188e13ac0fa` |
| `conductor_c19.sh` | 50 | `cb8629842b2635ea36293d788ed9980f174b18e19d570b3f5e4f124f830d81f5` |
| `conductor_c20.sh` | 38 | `189c59f0ee235722a3aeac1491bb21de26015da99746027a4a20f489ec1b9b8e` |

`fuentes_v5.tsv` fija exactamente diez fuentes. Sus diez filas, líneas y
SHA-256 deben coincidir con el target; no se aprende una huella nueva, no se
actualiza el ledger y no se acepta una fuente adicional o ausente. G7a, G7b,
los demás ocho fuentes, los siete bloques, casos, oráculos, workflow,
toolchain, evidencias históricas y documentos transversales son inmutables en
este corte.
Cada fila de `fuentes_v5.tsv` aporta solo un basename no vacío, sin `/` ni
componentes `.` o `..`. El coordinador fija
`base_fuentes_rel=deploy/postgresql/autorizacion_atestada_v3/pruebas_sql`,
construye `ruta_relativa=base_fuentes_rel/basename` y
`fichero=FUENTE/ruta_relativa`; `bytes_tecnicos.tsv` registra siempre esa ruta
relativa canónica completa.

## Toolchain exacta del carril

`GOROOT_ORIGEN` es únicamente la fuente de lectura para la copia controlada; no
es una raíz ejecutable del carril. La única raíz ejecutable futura es
`GOROOT_COPIA`, y el único `PATH` permitido empieza en `GOROOT_COPIA/bin`:

```text
GOROOT_ORIGEN=/srv/fabrica/orquesta/home/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.6.linux-amd64
GOROOT_COPIA=/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-goroot
```

La prueba biyectiva origen→copia es `cmp` de los dos inventarios completos,
ordenados y sin duplicados, cuyas columnas exactas son `tipo`, `ruta_relativa`,
`modo` y `SHA-256`; cualquier diferencia, omisión o entrada adicional es rojo.
En `GOROOT_COPIA`:

| Ejecutable | Propietario | Modo | Enlaces (nlink) | SHA-256 |
| --- | --- | ---: | ---: | --- |
| `bin/go` | `root:root` (`0:0`) | `0555` | `1` | `29e6e0b8be61beb1489ceae62b304343566de8a1dc700af74bde7aeb9c80ad45` |
| `bin/gofmt` | `root:root` (`0:0`) | `0555` | `1` | `0ef6fe2d15c972d15b8ecb75dcb9c861e042ac94a0e48baf0202a840648b0fdc` |

`bin/go version` debe devolver exactamente:

```text
go version go1.26.6 linux/amd64
```

Los binarios Go y gofmt son ELF Linux/AMD64 estáticos. El `PATH` de cada
carril empieza en el `bin` anterior; no usa el lanzador
`/usr/local/bin/go`, no permite selección automática, no descarga otra
toolchain y fija `GOTOOLCHAIN=local`, `GOENV=off`, `GOPROXY=off` y
`GOSUMDB=off`.

### Incidencia de telemetría anterior y aislamiento obligatorio

Durante una revisión anterior a este contrato, Go 1.26.6 creó el contador:

```text
/srv/fabrica/orquesta/home/.config/go/telemetry/local/go@go1.26.6-go1.26.6-linux-amd64-2026-08-21.v1.count
```

Es una **incidencia preexistente ajena a los futuros carriles**. En el
instante de esta corrección era un fichero regular, `orquesta:orquesta`
(`999:982`), modo `0644`, `nlink=1`, 16384 bytes y SHA-256
`e588a3240689f992fda01d36b8d642edb893f0c6dea2ca72d6f12784aa220475`.
No se borra, trunca, renombra, toca, copia como resultado ni altera. Tampoco
se usa el `HOME` común que lo contiene durante preflight o carriles. El
paquete global vuelve a observar sus metadatos y huella antes y después; una
mutación atribuible a la ventana es roja, pero su mera existencia inicial no
lo es.

La fuente local de Go 1.26.6 establece que `os.UserConfigDir` consume
`XDG_CONFIG_HOME` y que el modo reside en
`XDG_CONFIG_HOME/go/telemetry/mode`. Antes de **cualquier** invocación futura
de `go` o `gofmt`, incluido cada `go version` del preflight, se materializa
con Bash y utilidades no Go el fichero privado de cada carril con estos bytes
exactos:

```text
contenido = off 2026-08-21
codificación = ASCII/UTF-8
longitud = 14 bytes
salto final = ninguno
SHA-256 = f03cbb57ff78c1021486549450b22a77fee846d2f4b9a04738ebceee643568fc
modo = 0600
propietario = orquesta:orquesta (999:982)
tipo = fichero regular
nlink = 1
```

`HOME`, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, `TMPDIR` y `GOCACHE` son
distintos y nuevos en cada carril. Todo proceso Go nace con esos cinco
valores privados, `GOROOT` global root:root explícito, `PATH` fijado y entorno
vacío. Antes del `go version`, inmediatamente
después de él y en el postflight del carril se exige ausencia bajo
`XDG_CONFIG_HOME/go/telemetry` de `local`, `upload`, `debug`, cualquier
`*.count` y cualquier entrada distinta de `mode`. Tras el carril, los bytes y
metadatos de `mode` se copian a su atestación durable; solo entonces se
autoriza retirar las cinco raíces privadas, y se acredita su ausencia. Esa
retirada no alcanza nunca al contador común preexistente.

## SHA documental futuro

Este borrador no conoce ni puede fabricar su propio commit. Después de que el
contenido congelado reciba dos revisiones documentales independientes verdes,
dirección podrá crear un único commit con estas propiedades:

```text
padre_documental = 861e22698dc43ee5919a9e01cddffaac40be0f2b
base_tecnica = 484020703c683c324e9b2eaef5c43a56c1d95ea6
delta = M docs/portal_vec/enmienda_o3a_v5_cnd_v4_c21_estabilidad_go1_26_6_2026-08-21.md
resto de rutas = cero
```

El hash completo de ese commit se denomina `SHA_DOCUMENTAL`; sus doce primeros
hexadecimales se denominan `SHA12`. El árbol del commit debe partir del padre
documental inmediato y conservar la base técnica `4840207` como su progenitor
histórico. Los tres carriles fijan el
mismo `SHA_DOCUMENTAL` y el mismo árbol. Un amend, rebase, cherry-pick, cambio
de padre o cambio de un byte produce otro SHA y deja sin consumir estas rutas.

No se ejecuta ningún carril antes de que dirección publique expresamente los
seis operandos `SHA_DOCUMENTAL`, `ARBOL_DOCUMENTAL`, `BLOB_DOCUMENTAL`,
`SHA256_DOCUMENTO`, `LINEAS_DOCUMENTO` y `BYTES_DOCUMENTO` de este mismo
candidato aprobado y confirme las dos revisiones documentales. Esos seis
valores se copian literalmente al argv del bootstrap; no se infieren del
commit que se pretende juzgar. El coordinador compara antes de reservar rutas
el padre, el delta Git de una sola modificación documental, el árbol, el blob, la huella, las
líneas y los bytes observados con esos operandos. Cualquier diferencia impide
el preflight y hace imposible consumir como verde una reedición, una ruta
adicional o un commit distinto del blob revisado.

El commit `861e22698dc43ee5919a9e01cddffaac40be0f2b` queda registrado como
`ROJO_PREFLIGHT_SIN_CARRILES_NO_CONSUMIBLE`: su única materialización terminó
antes de los carriles por doce ficheros vacíos que BusyBox `stat -c %F` rotula `regular empty
file`, con cero hijos, cero carriles, sin sellado y sin reintento. Sus rutas y
evidencias son históricas, inmutables, no consumibles y no reutilizables. El
commit `1789629ab78c5274356ae6998c4849254ff84c2b` permanece como intento
documental anterior inválido; el siguiente candidato tendrá como
`BASE_TECNICA=484020703c683c324e9b2eaef5c43a56c1d95ea6` →
`ANTECESOR_DOCUMENTAL=1789629ab78c5274356ae6998c4849254ff84c2b` →
`PADRE_DOCUMENTAL=861e22698dc43ee5919a9e01cddffaac40be0f2b` y conservará
esa genealogía; son autoridades
distintas y no se intercambian.

## Rutas consumibles predeclaradas

Después de publicar `SHA_DOCUMENTAL`, se sustituye literalmente `SHA12` por
sus doce primeros hexadecimales. No se admite ninguna otra interpolación. El
paquete de control global ocupa esta ruta nueva:

```text
CONTROL=/srv/fabrica/orquesta/home/evidencias/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-control-global
GOROOT_COPIA=/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-goroot
```

`GOROOT_COPIA` es una segunda raíz global reservada. Se crea directamente y
de forma atómica por `root` bajo `/var/tmp`, cuyo `lstat` debe ser directorio
`root:root`, modo `1777`. Al ser la entrada hija también `root:root`, el sticky
bit impide que UID 999 la renombre o retire; toda su jerarquía queda
`root:root`, sin escritura, antes de bajar privilegios.

Cada carril reserva ocho raíces externas nuevas:

| Carril | Símbolo | Ruta absoluta |
| --- | --- | --- |
| Productor | `TARGET` | `/srv/fabrica/orquesta/home/revisiones/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-productor-target` |
| Productor | `DESTINO` | `/srv/fabrica/orquesta/home/evidencias/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-productor-unica` |
| Productor | `ATESTACION` | `/srv/fabrica/orquesta/home/evidencias/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-productor-atestacion` |
| Productor | `HOME` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-productor-home` |
| Productor | `XDG_CONFIG_HOME` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-productor-xdg-config` |
| Productor | `XDG_CACHE_HOME` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-productor-xdg-cache` |
| Productor | `TMPDIR` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-productor-runtime` |
| Productor | `GOCACHE` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-productor-gocache` |
| Funcional | `TARGET` | `/srv/fabrica/orquesta/home/revisiones/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-funcional-target` |
| Funcional | `DESTINO` | `/srv/fabrica/orquesta/home/evidencias/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-funcional-unica` |
| Funcional | `ATESTACION` | `/srv/fabrica/orquesta/home/evidencias/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-funcional-atestacion` |
| Funcional | `HOME` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-funcional-home` |
| Funcional | `XDG_CONFIG_HOME` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-funcional-xdg-config` |
| Funcional | `XDG_CACHE_HOME` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-funcional-xdg-cache` |
| Funcional | `TMPDIR` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-funcional-runtime` |
| Funcional | `GOCACHE` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-funcional-gocache` |
| Seguridad | `TARGET` | `/srv/fabrica/orquesta/home/revisiones/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-seguridad-target` |
| Seguridad | `DESTINO` | `/srv/fabrica/orquesta/home/evidencias/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-seguridad-unica` |
| Seguridad | `ATESTACION` | `/srv/fabrica/orquesta/home/evidencias/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-seguridad-atestacion` |
| Seguridad | `HOME` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-seguridad-home` |
| Seguridad | `XDG_CONFIG_HOME` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-seguridad-xdg-config` |
| Seguridad | `XDG_CACHE_HOME` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-seguridad-xdg-cache` |
| Seguridad | `TMPDIR` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-seguridad-runtime` |
| Seguridad | `GOCACHE` | `/var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-SHA12-seguridad-gocache` |

El recuento total es **26/26 raíces externas reservadas**: `CONTROL`, una
`GOROOT_COPIA` global y ocho raíces por cada uno de tres carriles. Las capturas son nueve ficheros
internos de las tres atestaciones, no nueve raíces externas. Los ficheros
`mode` son tres entradas internas de los tres `XDG_CONFIG_HOME`. Las 21
entradas enumeradas por el `SHA256SUMS` de cada `DESTINO` son, a su vez,
entradas internas del resultado del conductor y conservan su cardinalidad
propia `21/21`; no forman parte del recuento 26/26.

Las 26 raíces deben estar ausentes antes de crear nada. El observador inicial
las evalúa en el orden publicado, sin seguir enlaces, y solo si obtiene 26
ausencias crea `CONTROL` y escribe en él el
manifiesto completo `ausencia_inicial.tsv`. La ausencia de una raíz implica
la de todos sus descendientes nominales. Toda entrada posterior se crea con
`umask 077` y noclobber. Ruta existente, enlace, renombrado, vaciado, sufijo
`r2`, sobreescritura o reutilización produce `ROJO_PREFLIGHT`, con cero
invocaciones de conductor.

Los clones se crean localmente, sin red, sin hardlinks y con `umask 077`, desde
el repositorio local que contiene `SHA_DOCUMENTAL`. Cada clon queda en HEAD
desacoplado, sin remoto, `alternates`, `shallow`, `commondir` redirigido ni
grafts. Todos sus objetos y ficheros pertenecen a `orquesta`; el target es un
directorio real, no un enlace. La limpieza exige simultáneamente cero cambios
rastreados, preparados y no seguidos.

## Bootstrap estático y coordinador materializable

El repositorio fuente local único es:

```text
FUENTE=/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-20260821
DOCUMENTO=docs/portal_vec/enmienda_o3a_v5_cnd_v4_c21_estabilidad_go1_26_6_2026-08-21.md
```

No se reserva una raíz bootstrap. El coordinador viaja por una tubería desde el
blob comprometido hasta stdin de Bash; así puede comprobar las 26 rutas,
incluido `CONTROL`, antes de crear ninguna. El primer ejecutable del argv es
BusyBox estático. Este es el único bootstrap autorizado, en una sola línea
lógica. Solo los seis operandos en mayúsculas publicados por dirección se
sustituyen en la futura línea de invocación; fuente, documento y huella del
coordinador ya son literales y nunca proceden de estado de shell heredado:

```text
/usr/bin/busybox env -i LC_ALL=C PATH=/usr/bin:/bin HOME=/nonexistent/o3a-v5-bootstrap XDG_CONFIG_HOME=/nonexistent/o3a-v5-bootstrap GIT_CONFIG_NOSYSTEM=1 GIT_TERMINAL_PROMPT=0 GIT_ALLOW_PROTOCOL=file /usr/bin/bash --noprofile --norc -c 'set -euo pipefail; fuente=$1; sha=$2; documento=$3; esperado=$4; shift 4; cd "$fuente"; extraer(){ /usr/bin/git -c core.hooksPath=/dev/null -c protocol.file.allow=always -C "$fuente" show "$sha:$documento" | /usr/bin/busybox awk '\''$0=="<!-- COORDINADOR_V3_INICIO -->"{d=1;s=1;next}$0=="<!-- COORDINADOR_V3_FIN -->"{d=0}d{if(s){s=0;next}a[++n]=$0}END{for(i=1;i<n;i++)print a[i]}'\''; }; observado=$(extraer | /usr/bin/busybox sha256sum | /usr/bin/busybox awk '\''{print $1}'\''); test "$observado" = "$esperado"; extraer | /usr/bin/busybox env -i LC_ALL=C PATH=/usr/bin:/bin HOME=/nonexistent/o3a-v5-bootstrap XDG_CONFIG_HOME=/nonexistent/o3a-v5-bootstrap GIT_CONFIG_NOSYSTEM=1 GIT_TERMINAL_PROMPT=0 GIT_ALLOW_PROTOCOL=file /usr/bin/bash --noprofile --norc -s -- "$fuente" "$sha" "$documento" "$esperado" "$@"' bootstrap /srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-20260821 SHA_DOCUMENTAL docs/portal_vec/enmienda_o3a_v5_cnd_v4_c21_estabilidad_go1_26_6_2026-08-21.md de397a8228459d7b25272332fc2b2bf19e0f5fe804d958de5ae092a8a34b0ed7 ARBOL_DOCUMENTAL BLOB_DOCUMENTAL SHA256_DOCUMENTO LINEAS_DOCUMENTO BYTES_DOCUMENTO FIN_OPERANDOS
```

La segunda instancia Bash recibe por stdin exclusivamente la plantilla y el
argv exacto `[/usr/bin/bash,--noprofile,--norc,-s,--,FUENTE_LITERAL,
SHA_DOCUMENTAL,DOCUMENTO_LITERAL,HUELLA_COORDINADOR,ARBOL_DOCUMENTAL,
BLOB_DOCUMENTAL,SHA256_DOCUMENTO,LINEAS_DOCUMENTO,BYTES_DOCUMENTO,FIN_OPERANDOS]`;
dentro de la plantilla esos diez operandos son `$1..$10`; exige exactamente
`FIN_OPERANDOS` en `$10` y rechaza cualquier otra cardinalidad o centinela. Los dos literales
de ruta y la huella completa del coordinador son exactamente los publicados
arriba. No se evalúa texto de variables heredadas.

BusyBox limpia el entorno antes de cargar Bash, Git o sus cargadores
dinámicos. El coordinador exige UID/GID iniciales `0:0`, cwd `FUENTE`,
`umask 077` y exactamente el entorno reaplicado por el segundo `env -i` más
`PWD`, `SHLVL` y `_` creados por Bash. El `OLDPWD` que el Bash bootstrap crea
al hacer `cd FUENTE` no cruza ese segundo vaciado. Abre y bloquea, sin crear
entradas, los directorios padres de
evidencias, revisiones y `/var/tmp`; bajo esos tres bloqueos cooperativos
observa las 26 ausencias. Solo entonces crea `CONTROL`, vuelve a extraer desde
el mismo blob su instancia `CONTROL/coordinador_v3.sh`, compara la huella
publicada y continúa. Así rompe la circularidad sin ruta adicional. Un actor
no cooperativo del mismo UID sigue fuera de garantía.

El coordinador completo tiene exactamente los bytes situados entre estos
marcadores, en UTF-8/ASCII, LF y un único LF final:

<!-- COORDINADOR_V3_INICIO -->
```bash
#!/usr/bin/bash
# shellcheck disable=SC2016,SC2129,SC2153,SC2317,SC2329
set -euo pipefail
set -o noclobber
umask 077
BB=/usr/bin/busybox
BASH=/usr/bin/bash
GIT=/usr/bin/git
FLOCK=/usr/bin/flock
SETPRIV=/usr/bin/setpriv
SETSID=/usr/bin/setsid
GOROOT_ORIGEN=/srv/fabrica/orquesta/home/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.6.linux-amd64
GOROOT_IDENTIDAD_SHA=9931a14ad98c26369bf964958b9adeec9edd26162086a51c1842aebe59135511
GO_SHA=29e6e0b8be61beb1489ceae62b304343566de8a1dc700af74bde7aeb9c80ad45
GOFMT_SHA=0ef6fe2d15c972d15b8ecb75dcb9c861e042ac94a0e48baf0202a840648b0fdc
PADRE_DOCUMENTAL=861e22698dc43ee5919a9e01cddffaac40be0f2b
ANTECESOR_DOCUMENTAL=1789629ab78c5274356ae6998c4849254ff84c2b
BASE_TECNICA=484020703c683c324e9b2eaef5c43a56c1d95ea6
MODE_SHA=f03cbb57ff78c1021486549450b22a77fee846d2f4b9a04738ebceee643568fc
PRIVILEGIO=(--reuid 999 --regid 982 --clear-groups --inh-caps=-all --ambient-caps=-all --bounding-set=-all --no-new-privs)

utc() { "$BB" date -u +%Y-%m-%dT%H:%M:%SZ; }
sha() { "$BB" sha256sum "$1" | "$BB" awk '{print $1}'; }
emitir_inventario_goroot() (
    local raiz=$1
    cd "$raiz"
    printf 'tipo\truta_relativa\tmodo\tsha256\n'
    "$BB" find . -xdev -mindepth 1 -type d -perm 0555 -print | "$BB" sort |
        "$BB" awk '{sub(/^\.\//,"");print "D\t"$0"\t555\t-"}'
    local modo
    for modo in 0444 0555; do
        "$BB" find . -xdev -mindepth 1 -type f -perm "$modo" -print | "$BB" sort |
            "$BB" xargs -r -n 256 "$BB" sha256sum |
            "$BB" awk -v m="${modo#0}" '{h=$1;$1="";sub(/^ /,"");sub(/^\.\//,"");print "F\t"$0"\t"m"\t"h}'
    done
)
validar_inventario_goroot() {
    local fichero=$1 filas directorios regulares regulares_444 regulares_555 identidad
    filas=$(($("$BB" wc -l <"$fichero")-1))
    directorios=$("$BB" awk -F '\t' 'NR>1&&$1=="D"{n++}END{print n+0}' "$fichero")
    regulares=$("$BB" awk -F '\t' 'NR>1&&$1=="F"{n++}END{print n+0}' "$fichero")
    regulares_444=$("$BB" awk -F '\t' 'NR>1&&$1=="F"&&$3=="444"{n++}END{print n+0}' "$fichero")
    regulares_555=$("$BB" awk -F '\t' 'NR>1&&$1=="F"&&$3=="555"{n++}END{print n+0}' "$fichero")
    identidad=$("$BB" awk 'NR>1' "$fichero" | "$BB" sha256sum | "$BB" awk '{print $1}')
    [[ $filas -eq 12870 && $directorios -eq 1334 && $regulares -eq 11536 && $regulares_444 -eq 11524 && $regulares_555 -eq 12 && $identidad == "$GOROOT_IDENTIDAD_SHA" ]]
    "$BB" awk -F '\t' 'NR==1{if($0!="tipo\truta_relativa\tmodo\tsha256")exit 1;next}NF!=4||($1!="D"&&$1!="F")||$2==""||($1=="D"&&($3!="555"||$4!="-"))||($1=="F"&&($3!="444"&&$3!="555"))||($1=="F"&&($4!~/^[0-9a-f]+$/||length($4)!=64)){exit 1}' "$fichero"
}
extraer() {
    "$GIT" -c core.hooksPath=/dev/null -c protocol.file.allow=always -C "$fuente" show "$sha_documental:$documento" |
        "$BB" awk '$0=="<!-- COORDINADOR_V3_INICIO -->"{d=1;s=1;next}$0=="<!-- COORDINADOR_V3_FIN -->"{d=0}d{if(s){s=0;next}a[++n]=$0}END{for(i=1;i<n;i++)print a[i]}'
}
campos_proceso() {
    local pid=$1 linea resto
    linea=$(<"/proc/$pid/stat")
    resto=${linea##*) }
    printf '%s\n' "$resto" | "$BB" awk -v pid="$pid" '{print pid"\t"$2"\t"$3"\t"$4"\t"$20}'
}
foto() {
    local fichero=$1 fase=$2 p linea pid ppid pgid sid inicio cmd cwd uid gid env_sha clase
    printf 'fase\tinstante_utc\tpid\tppid\tpgid\tsid\tuid\tgid\tstarttime_ticks\tcmdline_sha256\tenviron_sha256\tcwd\tclasificacion\n' >"$fichero"
    for p in /proc/[0-9]*; do
        pid=${p##*/}
        if [[ ! -r $p/stat || ! -r $p/status || ! -r $p/cmdline || ! -r $p/environ ]]; then [[ -d $p ]] && return 2; continue; fi
        if ! linea=$(campos_proceso "$pid"); then [[ -d $p ]] && return 2; continue; fi
        IFS=$'\t' read -r pid ppid pgid sid inicio <<<"$linea"
        uid=$("$BB" awk '/^Uid:/{print $2;exit}' "$p/status")
        gid=$("$BB" awk '/^Gid:/{print $2;exit}' "$p/status")
        cmd=$(sha "$p/cmdline")
        env_sha=$(sha "$p/environ")
        if ! cwd=$("$BB" readlink "$p/cwd" 2>/dev/null); then [[ -e $p/cwd || -L $p/cwd ]] && return 2; continue; fi
        clase=otro
        if [[ -r $p/cmdline ]] && "$BB" tr '\0' '\n' <"$p/cmdline" | "$BB" grep -q 'tools/o3a_v5_conductor/conductor.sh'; then clase=conductor_o3a_v5; fi
        printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$fase" "$(utc)" "$pid" "$ppid" "$pgid" "$sid" "$uid" "$gid" "$inicio" "$cmd" "$env_sha" "$cwd" "$clase" >>"$fichero"
    done
}
estado() {
    local fichero=$1 secuencia=$2 transicion=$3 carril=$4 pid=$5 ppid=$6 pgid=$7 sid=$8 inicio=$9 argv_sha=${10} salida=${11} previo=${12} resultado=${13}
    printf 'secuencia\ttransicion\tinstante_utc\tcarril\tpid\tppid\tpgid\tsid\tstarttime_ticks\targv_sha256\testado_salida\tprev_sha256\tresultado\n' >"$fichero"
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$secuencia" "$transicion" "$(utc)" "$carril" "$pid" "$ppid" "$pgid" "$sid" "$inicio" "$argv_sha" "$salida" "$previo" "$resultado" >>"$fichero"
}
registrar_entorno() {
    local fichero=$1 fase=$2 proceso=$3 pid=$4 ppid=$5 origen=$6 nombre valor
    while IFS='=' read -r nombre valor; do
        [[ -n $nombre ]] || continue
        printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$fase" "$proceso" "$pid" "$ppid" "$nombre" "$valor" "$origen" GO >>"$fichero"
    done
}
registrar_privilegio() {
    local fichero=$1 fase=$2 proceso=$3 observado_pid=${4:-$$} observado_ppid estado_proc uid gid uid_efectivo gid_efectivo grupos capinh capprm capeff capbnd capamb nnp resultado=NO-GO
    estado_proc=/proc/$observado_pid/status
    [[ -r $estado_proc ]]
    if [[ $observado_pid == "$$" ]]; then observado_ppid=$PPID; else IFS=$'\t' read -r _ observado_ppid _ _ _ <<<"$(campos_proceso "$observado_pid")"; fi
    uid=$("$BB" awk '/^Uid:/{print $2":"$3":"$4":"$5;exit}' "$estado_proc")
    gid=$("$BB" awk '/^Gid:/{print $2":"$3":"$4":"$5;exit}' "$estado_proc")
    uid_efectivo=${uid#*:}; uid_efectivo=${uid_efectivo%%:*}
    gid_efectivo=${gid#*:}; gid_efectivo=${gid_efectivo%%:*}
    grupos=$("$BB" awk '/^Groups:/{sub(/^Groups:[[:space:]]*/,"");print;exit}' "$estado_proc")
    capinh=$("$BB" awk '/^CapInh:/{print $2;exit}' "$estado_proc")
    capprm=$("$BB" awk '/^CapPrm:/{print $2;exit}' "$estado_proc")
    capeff=$("$BB" awk '/^CapEff:/{print $2;exit}' "$estado_proc")
    capbnd=$("$BB" awk '/^CapBnd:/{print $2;exit}' "$estado_proc")
    capamb=$("$BB" awk '/^CapAmb:/{print $2;exit}' "$estado_proc")
    nnp=$("$BB" awk '/^NoNewPrivs:/{print $2;exit}' "$estado_proc")
    [[ $uid == '999:999:999:999' && $gid == '982:982:982:982' && -z $grupos && $capinh == 0000000000000000 && $capprm == 0000000000000000 && $capeff == 0000000000000000 && $capbnd == 0000000000000000 && $capamb == 0000000000000000 && $nnp == 1 ]] && resultado=GO
    if [[ $fichero != - ]]; then
        printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$fase" "$proceso" "$observado_pid" "$observado_ppid" "$uid_efectivo" "$gid_efectivo" "${grupos:--}" "$capinh" "$capprm" "$capeff" "$capbnd" "$capamb" "$nnp" "$resultado" >>"$fichero"
    fi
    [[ $resultado == GO ]]
}
git_hijo() {
    shift
    registrar_privilegio - GIT git
    exec "$BB" env -i LC_ALL=C PATH=/usr/bin:/bin HOME=/nonexistent/o3a-v5-git XDG_CONFIG_HOME=/nonexistent/o3a-v5-git GIT_CONFIG_NOSYSTEM=1 GIT_TERMINAL_PROMPT=0 GIT_ALLOW_PROTOCOL=file "$GIT" "$@"
}
go_version_hijo() {
    shift
    local pidfile=$1 privilegios=$2 binario=$3 linea resto
    registrar_privilegio "$privilegios" GO_VERSION go
    linea=$(<"/proc/$$/stat"); resto=${linea##*) }
    printf '%s\t%s\t%s\n' "$$" "$PPID" "$(printf '%s\n' "$resto" | "$BB" awk '{print $20}')" >"$pidfile"
    exec "$BB" env -i "HOME=$HOME" "XDG_CONFIG_HOME=$XDG_CONFIG_HOME" "XDG_CACHE_HOME=$XDG_CACHE_HOME" "TMPDIR=$TMPDIR" "GOCACHE=$GOCACHE" "GOROOT=$GOROOT" LC_ALL=C "PATH=$PATH" GOTOOLCHAIN=local GOENV=off GOPROXY=off GOSUMDB=off "$binario" version
}
como_orquesta_git() {
    "$BB" env -i LC_ALL=C PATH=/usr/bin:/bin "$SETPRIV" "${PRIVILEGIO[@]}" "$BB" env -i LC_ALL=C PATH=/usr/bin:/bin HOME=/nonexistent/o3a-v5-git XDG_CONFIG_HOME=/nonexistent/o3a-v5-git GIT_CONFIG_NOSYSTEM=1 GIT_TERMINAL_PROMPT=0 GIT_ALLOW_PROTOCOL=file "$BASH" --noprofile --norc "$control/coordinador_v3.sh" --git "$@" 9>&- {efd}>&- {le}>&- {lr}>&- {lt}>&- {lv}>&-
}
telemetria() {
    local fichero=$1 fase=$2 xdg=$3 mode="$3/go/telemetry/mode" local_n=0 upload_n=0 debug_n=0 count_n=0 otros=0 mode_ok=0 resultado
    [[ -d $xdg/go/telemetry/local ]] && local_n=1
    [[ -d $xdg/go/telemetry/upload ]] && upload_n=1
    [[ -d $xdg/go/telemetry/debug ]] && debug_n=1
    count_n=$("$BB" find "$xdg/go/telemetry" -name '*.count' -print 2>/dev/null | "$BB" wc -l)
    otros=$("$BB" find "$xdg/go/telemetry" -mindepth 1 -maxdepth 1 ! -name mode -print 2>/dev/null | "$BB" wc -l)
    [[ ! -L $mode && -f $mode && $(sha "$mode") == "$MODE_SHA" && $("$BB" stat -c '%U|%G|%a|%h|%s|%F' "$mode") == 'orquesta|orquesta|600|1|14|regular file' ]] && mode_ok=1
    resultado=NO-GO; [[ $local_n$upload_n$debug_n$count_n$otros$mode_ok == 000001 ]] && resultado=GO
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$fase" "$xdg" "$mode" 14 "$(sha "$mode")" "$local_n" "$upload_n" "$debug_n" "$count_n" "$otros" "$resultado" >>"$fichero"
    [[ $resultado == GO ]]
}
limpiar_privadas() {
    "$BB" rm -rf -- "$1" "$2" "$3" "$4" "$5"
}
coincide_ruta() {
    local enlace=$1 prefijo
    shift
    for prefijo in "$@"; do
        [[ $enlace == "$prefijo" || $enlace == "$prefijo"/* || $enlace == "$prefijo (deleted)" || $enlace == "$prefijo"/*' (deleted)' ]] && return 0
    done
    return 1
}
auditar_fd_rutas() {
    local total=0 ambigua=0 p fd enlace antes despues
    for p in /proc/[0-9]*; do
        [[ -d $p ]] || continue
        if ! antes=$(campos_proceso "${p##*/}"); then [[ -d $p ]] && ambigua=1; continue; fi
        if [[ ! -d $p/fd || ! -r $p/fd ]]; then [[ -d $p ]] && ambigua=1; continue; fi
        for fd in "$p"/fd/*; do
            [[ -e $fd || -L $fd ]] || continue
            if enlace=$("$BB" readlink "$fd" 2>/dev/null); then
                coincide_ruta "$enlace" "$@" && ((total+=1))
            elif [[ -e $fd || -L $fd ]]; then
                ambigua=1
            fi
        done
        if ! despues=$(campos_proceso "${p##*/}"); then [[ -d $p ]] && ambigua=1; continue; fi
        [[ $antes == "$despues" ]] || { ambigua=1; continue; }
    done
    [[ $ambigua -eq 0 ]] && printf '%s\tGO\n' "$total" || printf '%s\tIDENTIDAD_AMBIGUA\n' "$total"
    [[ $ambigua -eq 0 ]]
}
auditar_maps_rutas() {
    local total=0 ambigua=0 p antes despues mapas enlace
    for p in /proc/[0-9]*; do
        [[ -d $p ]] || continue
        if ! antes=$(campos_proceso "${p##*/}"); then [[ -d $p ]] && ambigua=1; continue; fi
        if [[ ! -r $p/maps ]]; then [[ -d $p ]] && ambigua=1; continue; fi
        if ! mapas=$("$BB" awk 'NF>=6{$1=$2=$3=$4=$5="";sub(/^ +/,"");print}' "$p/maps" 2>/dev/null); then
            [[ -d $p ]] && ambigua=1
            continue
        fi
        if ! despues=$(campos_proceso "${p##*/}"); then [[ -d $p ]] && ambigua=1; continue; fi
        [[ $antes == "$despues" ]] || { ambigua=1; continue; }
        while IFS= read -r enlace; do
            [[ -n $enlace ]] || continue
            coincide_ruta "$enlace" "$@" && ((total+=1))
        done <<<"$mapas"
    done
    [[ $ambigua -eq 0 ]] && printf '%s\tGO\n' "$total" || printf '%s\tIDENTIDAD_AMBIGUA\n' "$total"
    [[ $ambigua -eq 0 ]]
}
auditar_usuarios_rutas() {
    local total=0 ambigua=0 p enlace clase antes despues
    for p in /proc/[0-9]*; do
        [[ -d $p ]] || continue
        if ! antes=$(campos_proceso "${p##*/}"); then [[ -d $p ]] && ambigua=1; continue; fi
        for clase in cwd root exe; do
            [[ -e $p/$clase || -L $p/$clase ]] || continue
            if enlace=$("$BB" readlink "$p/$clase" 2>/dev/null); then
                coincide_ruta "$enlace" "$@" && ((total+=1))
            elif [[ -e $p/$clase || -L $p/$clase ]]; then
                ambigua=1
            fi
        done
        if ! despues=$(campos_proceso "${p##*/}"); then [[ -d $p ]] && ambigua=1; continue; fi
        [[ $antes == "$despues" ]] || { ambigua=1; continue; }
    done
    [[ $ambigua -eq 0 ]] && printf '%s\tGO\n' "$total" || printf '%s\tIDENTIDAD_AMBIGUA\n' "$total"
    [[ $ambigua -eq 0 ]]
}
monitorizar() {
    local raiz_pid=$1 raiz_inicio=$2 fichero=$3 entornos=$4 p linea pid ppid pgid sid inicio uid gid cmd env_sha cwd fase proceso clase clave estado_identidad
    local -A entorno_visto=()
    printf 'fase\tinstante_utc\tpid\tppid\tpgid\tsid\tuid\tgid\tstarttime_ticks\tcmdline_sha256\tenviron_sha256\tcwd\tclasificacion\n' >"$fichero"
    while :; do
        if identidad_pid "$raiz_pid" "$raiz_inicio"; then estado_identidad=0; else estado_identidad=$?; fi
        case $estado_identidad in 0) kill -0 "$raiz_pid" 2>/dev/null || return 2 ;; 1) break ;; 2) return 2 ;; esac
        for p in /proc/[0-9]*; do
            pid=${p##*/}
            if [[ ! -r $p/stat || ! -r $p/status || ! -r $p/environ || ! -r $p/cmdline ]]; then [[ -d $p ]] && return 2; continue; fi
            if ! linea=$(campos_proceso "$pid"); then [[ -d $p ]] && return 2; continue; fi
            IFS=$'\t' read -r pid ppid pgid sid inicio <<<"$linea"
            uid=$("$BB" awk '/^Uid:/{print $2;exit}' "$p/status")
            gid=$("$BB" awk '/^Gid:/{print $2;exit}' "$p/status")
            cmd=$(sha "$p/cmdline"); env_sha=$(sha "$p/environ"); cwd=$("$BB" readlink "$p/cwd" 2>/dev/null || printf NO_LEIBLE)
            if [[ $sid != "$raiz_pid" ]]; then
                if "$BB" tr '\0' '\n' <"$p/cmdline" | "$BB" grep -q 'tools/o3a_v5_conductor/conductor.sh'; then
                    printf 'CONCURRENCIA\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\tconductor_fuera_sesion\n' "$(utc)" "$pid" "$ppid" "$pgid" "$sid" "$uid" "$gid" "$inicio" "$cmd" "$env_sha" "$cwd" >>"$fichero"
                fi
                continue
            fi
            fase=CONDUCCION; proceso=$("$BB" tr '\0' '\n' <"$p/cmdline" | "$BB" sed -n '1p')
            if "$BB" tr '\0' '\n' <"$p/cmdline" | "$BB" grep -qx -- '--hijo'; then fase=PUENTE; fi
            "$BB" tr '\0' '\n' <"$p/environ" | "$BB" grep -q '^CND_RACE=0$' && fase=BLOQUE_NORMAL
            "$BB" tr '\0' '\n' <"$p/environ" | "$BB" grep -q '^CND_RACE=1$' && fase=BLOQUE_RACE
            "$BB" tr '\0' '\n' <"$p/environ" | "$BB" grep -q '^CGO_ENABLED=0$' && fase=GO_BUILD_NORMAL
            "$BB" tr '\0' '\n' <"$p/environ" | "$BB" grep -q '^CGO_ENABLED=1$' && fase=GO_BUILD_RACE
            if "$BB" tr '\0' '\n' <"$p/environ" | "$BB" grep -q '^GOFLAGS=$'; then
                fase=C20
                "$BB" tr '\0' '\n' <"$p/environ" | "$BB" grep -q '^CGO_ENABLED=0$' && fase=C20_BUILD
            fi
            printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\tgobernado\n' "$fase" "$(utc)" "$pid" "$ppid" "$pgid" "$sid" "$uid" "$gid" "$inicio" "$cmd" "$env_sha" "$cwd" >>"$fichero"
            [[ $fase == PUENTE ]] && continue
            clave=$fase:$pid
            if [[ -z ${entorno_visto[$clave]+x} ]]; then
                "$BB" tr '\0' '\n' <"$p/environ" | registrar_entorno "$entornos" "$fase" "$proceso" "$pid" "$ppid" OBSERVADO_PROC
                entorno_visto[$clave]=1
            fi
        done
        "$BB" sleep 0.1
    done
}
sellar_paquete() (
    local dir=$1 total=$2 plantilla=$3 perm="$1/permisos_finales.tsv" sum="$1/SHA256SUMS" f n=0 pfd='' sfd='' mode ordinal=1 u g m t l metadatos esperado esperado_l esperado_t resultado permisos_ok=1 huella
    cerrar_pfd() {
        [[ ${pfd:-} =~ ^[0-9]+$ ]] || return 0
        local rc=0
        exec {pfd}>&- || rc=$?
        pfd=
        return "$rc"
    }
    cerrar_sfd() {
        [[ ${sfd:-} =~ ^[0-9]+$ ]] || return 0
        local rc=0
        exec {sfd}>&- || rc=$?
        sfd=
        return "$rc"
    }
    cerrar_sellado() {
        local rc=$?
        trap - EXIT
        cerrar_pfd || true
        cerrar_sfd || true
        exit "$rc"
    }
    trap cerrar_sellado EXIT
    [[ ! -e $perm && ! -L $perm && ! -e $sum && ! -L $sum ]] || exit 90
    for f in "$dir"/*; do [[ -f $f && ! -L $f ]] || exit 91; ((n+=1)); done
    [[ $n -eq $((total-2)) ]] || exit 92
    exec {pfd}>"$perm" || exit 93
    exec {sfd}>"$sum" || exit 94
    "$BB" chown -R 999:982 "$dir" || exit 95
    "$BB" chmod 0500 "$dir" || exit 95
    for f in "$dir"/*; do mode=0400; [[ $f == "$plantilla" ]] && mode=0500; "$BB" chmod "$mode" "$f" || exit 95; done
    printf 'ordinal\truta\tpropietario_esperado\tgrupo_esperado\tmodo_esperado\ttipo_esperado\tnlink_esperado\tpropietario_observado\tgrupo_observado\tmodo_observado\ttipo_observado\tnlink_observado\tresultado\n' >&"$pfd" || exit 96
    for f in "$dir" "$dir"/*; do
        metadatos=$("$BB" stat -c '%U|%G|%a|%F|%h' "$f") || permisos_ok=0
        IFS='|' read -r u g m t l <<<"$metadatos"
        esperado=400; esperado_l=1; esperado_t='regular file'
        [[ $f == "$dir" ]] && { esperado=500; esperado_l=2; esperado_t=directory; }
        [[ $f == "$plantilla" ]] && esperado=500
        resultado=NO-GO
        tipo_valido=0; [[ $t == "$esperado_t" || ($t == 'regular empty file' && $esperado_t == 'regular file' && $("$BB" stat -c %s "$f") -eq 0) ]] && tipo_valido=1
        [[ $u:$g == orquesta:orquesta && $m == "$esperado" && $l == "$esperado_l" && $tipo_valido -eq 1 ]] && resultado=GO
        [[ $resultado == GO ]] || permisos_ok=0
        printf '%s\t%s\torquesta\torquesta\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$ordinal" "$f" "$esperado" "$esperado_t" "$esperado_l" "$u" "$g" "$m" "$t" "$l" "$resultado" >&"$pfd" || exit 96
        ((ordinal+=1))
    done
    cerrar_pfd || exit 97
    [[ $permisos_ok -eq 1 ]] || exit 98
    for f in "$dir"/*; do
        [[ $f == "$sum" ]] && continue
        huella=$(sha "$f") || exit 99
        printf '%s  %s\n' "$huella" "${f##*/}" >&"$sfd" || exit 99
    done
    cerrar_sfd || exit 100
    (cd "$dir" && "$BB" sha256sum -c SHA256SUMS >/dev/null) || exit 101
    trap - EXIT
    exit 0
)
hijo() {
    shift
    local carril=$1 target=$2 destino=$3 atestacion=$4 home=$5 xdg=$6 xcache=$7 tmp=$8 gocache=$9 sha_documental=${10} goroot=${11}
    local espera linea pid ppid pgid sid inicio conductor argv_sha previo ruta_fd fd
    for ruta_fd in /proc/self/fd/*; do
        fd=${ruta_fd##*/}; [[ $fd =~ ^[0-9]+$ && $fd -ge 3 ]] || continue
        { exec {fd}>&-; } 2>/dev/null || true
    done
    registrar_privilegio "$atestacion/privilegios.tsv" CONDUCCION conductor
    linea=$(campos_proceso $$)
    IFS=$'\t' read -r pid ppid pgid sid inicio <<<"$linea"
    [[ $pid == "$pgid" && $pid == "$sid" ]]
    conductor=$target/tools/o3a_v5_conductor/conductor.sh
    argv_sha=$(printf '%s\0' /usr/bin/bash "$conductor" "$target" "$destino" | "$BB" sha256sum | "$BB" awk '{print $1}')
    previo=$(sha "$atestacion/01_RESERVADO.tsv")
    estado "$atestacion/02_HIJO_NACIDO.tsv" 2 HIJO_NACIDO "$carril" "$pid" "$ppid" "$pgid" "$sid" "$inicio" "$argv_sha" PENDIENTE "$previo" GO
    for ((espera=0; espera<300; espera++)); do
        [[ -f $atestacion/03_LIBERADO.tsv ]] && break
        "$BB" sleep 1
    done
    [[ -f $atestacion/03_LIBERADO.tsv ]]
    cd "$target"
    [[ $goroot == /var/tmp/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-${sha_documental:0:12}-goroot && ! -L $goroot && -d $goroot ]]
    [[ $("$BB" stat -c '%U|%G|%a|%h|%F' "$goroot") == 'root|root|555|6|directory' ]]
    [[ ! -L $goroot/bin/go && ! -L $goroot/bin/gofmt && $(sha "$goroot/bin/go") == "$GO_SHA" && $(sha "$goroot/bin/gofmt") == "$GOFMT_SHA" ]]
    [[ $("$BB" stat -c '%U|%G|%a|%h|%F' "$goroot/bin/go") == 'root|root|555|1|regular file' && $("$BB" stat -c '%U|%G|%a|%h|%F' "$goroot/bin/gofmt") == 'root|root|555|1|regular file' ]]
    exec "$BB" env -i "HOME=$home" "XDG_CONFIG_HOME=$xdg" "XDG_CACHE_HOME=$xcache" "TMPDIR=$tmp" "GOCACHE=$gocache" "GOROOT=$goroot" LC_ALL=C "PATH=$goroot/bin:/usr/bin:/bin" GOTOOLCHAIN=local GOENV=off GOPROXY=off GOSUMDB=off "$BASH" "$conductor" "$target" "$destino"
}
case ${1:-} in
    --git) git_hijo "$@"; exit ;;
    --go-version) go_version_hijo "$@"; exit ;;
    --hijo) hijo "$@"; exit ;;
esac
[[ $# -eq 10 ]]
fuente=$1; sha_documental=$2; documento=$3; coordinador_sha=$4
arbol_esperado=$5; blob_esperado=$6; doc_sha_esperado=$7; doc_lineas_esperadas=$8; doc_bytes_esperados=$9; fin_operandos=${10}
[[ $fin_operandos == FIN_OPERANDOS ]]
[[ $fuente == /srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-20260821 ]]
[[ $documento == docs/portal_vec/enmienda_o3a_v5_cnd_v4_c21_estabilidad_go1_26_6_2026-08-21.md ]]
[[ $sha_documental =~ ^[0-9a-f]{40}$ && $coordinador_sha =~ ^[0-9a-f]{64}$ && $arbol_esperado =~ ^[0-9a-f]{40}$ && $blob_esperado =~ ^[0-9a-f]{40}$ && $doc_sha_esperado =~ ^[0-9a-f]{64}$ && $doc_lineas_esperadas =~ ^[1-9][0-9]*$ && $doc_bytes_esperados =~ ^[1-9][0-9]*$ ]]
uid=$("$BB" awk '/^Uid:/{print $2;exit}' /proc/self/status)
gid=$("$BB" awk '/^Gid:/{print $2;exit}' /proc/self/status)
[[ $uid:$gid == 0:0 && $PWD == "$fuente" ]]
[[ $("$GIT" -C "$fuente" rev-parse "$sha_documental^") == "$PADRE_DOCUMENTAL" ]]
[[ $("$GIT" -C "$fuente" rev-parse HEAD) == "$sha_documental" && -z $("$GIT" -C "$fuente" status --porcelain=v2 --untracked-files=all) ]]
[[ $("$GIT" -C "$fuente" rev-parse "$PADRE_DOCUMENTAL^") == "$ANTECESOR_DOCUMENTAL" ]]
[[ $("$GIT" -C "$fuente" rev-parse "$ANTECESOR_DOCUMENTAL^") == "$BASE_TECNICA" ]]
delta=$("$GIT" -C "$fuente" diff-tree --no-commit-id --name-status -r "$PADRE_DOCUMENTAL" "$sha_documental")
[[ $delta == $'M\tdocs/portal_vec/enmienda_o3a_v5_cnd_v4_c21_estabilidad_go1_26_6_2026-08-21.md' ]]
arbol=$("$GIT" -C "$fuente" rev-parse "$sha_documental^{tree}")
blob=$("$GIT" -C "$fuente" rev-parse "$sha_documental:$documento")
doc_sha=$("$GIT" -C "$fuente" show "$sha_documental:$documento" | "$BB" sha256sum | "$BB" awk '{print $1}')
doc_lineas=$("$GIT" -C "$fuente" show "$sha_documental:$documento" | "$BB" wc -l)
doc_bytes=$("$GIT" -C "$fuente" cat-file -s "$sha_documental:$documento")
[[ $arbol == "$arbol_esperado" && $blob == "$blob_esperado" && $doc_sha == "$doc_sha_esperado" && $doc_lineas == "$doc_lineas_esperadas" && $doc_bytes == "$doc_bytes_esperados" ]]
sha12=${sha_documental:0:12}
prefijo=o3a-v5-cnd-v4-c21-estabilidad-go1.26.6-$sha12
control=/srv/fabrica/orquesta/home/evidencias/$prefijo-control-global
goroot=/var/tmp/$prefijo-goroot
GO=$goroot/bin/go
GOFMT=$goroot/bin/gofmt
carriles=(productor funcional seguridad)
casos_deterministas=(c01_c07_normal.tsv c01_c07_race.tsv c02_interleavings_normal.tsv c02_interleavings_race.tsv c08_c11_c14_normal.tsv c08_c11_c14_race.tsv c10_c13_normal.tsv c10_c13_race.tsv c15_c21_normal.tsv c15_c21_normal_c21_indices.tsv c15_c21_race.tsv c15_c21_race_c21_indices.tsv c19_normal.tsv c19_race.tsv c20_normal.tsv c20_race.tsv)
manifiesto_esperado=(
    'c01_c07|9236cd1d7ca6cb1ebfbfb511c3913b96be980ba28acebeedad91e2f70deaedc3'
    'c02_interleavings|67d993b84c3c337a230a53f90d12ed91c08021c8ef88279cc93fd8557659f78a'
    'c08_c11_c14|a07c561ad3a616a59256f3bcb317cf803c3c0a0b01f6a09eabad3cfb8614329e'
    'c10_c13|70e2413081711d3f7019fa5daa4f5172d2ddde2b6eb5f2b121dc7d82f478cd46'
    'c15_c21|edf6a719eed6c96f920265c8c51df16b458d172744cb6cf06b581188e13ac0fa'
    'c19|cb8629842b2635ea36293d788ed9980f174b18e19d570b3f5e4f124f830d81f5'
    'c20|189c59f0ee235722a3aeac1491bb21de26015da99746027a4a20f489ec1b9b8e'
)
declare -A target destino atestacion home xdg xcache tmp gocache
declare -A resultado_carril estado_carril
rutas=("$control" "$goroot"); clases_ruta=(paquete_control toolchain_global); carriles_ruta=(global global); simbolos_ruta=(CONTROL GOROOT_COPIA)
preflight_cerrado=0
control_creado=0
diario_creado=0
for c in "${carriles[@]}"; do
    target[$c]=/srv/fabrica/orquesta/home/revisiones/$prefijo-$c-target
    destino[$c]=/srv/fabrica/orquesta/home/evidencias/$prefijo-$c-unica
    atestacion[$c]=/srv/fabrica/orquesta/home/evidencias/$prefijo-$c-atestacion
    home[$c]=/var/tmp/$prefijo-$c-home
    xdg[$c]=/var/tmp/$prefijo-$c-xdg-config
    xcache[$c]=/var/tmp/$prefijo-$c-xdg-cache
    tmp[$c]=/var/tmp/$prefijo-$c-runtime
    gocache[$c]=/var/tmp/$prefijo-$c-gocache
    rutas+=("${target[$c]}" "${destino[$c]}" "${atestacion[$c]}" "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}")
    clases_ruta+=(clon destino paquete_atestacion privada privada privada privada privada)
    carriles_ruta+=("$c" "$c" "$c" "$c" "$c" "$c" "$c" "$c")
    simbolos_ruta+=(TARGET DESTINO ATESTACION HOME XDG_CONFIG_HOME XDG_CACHE_HOME TMPDIR GOCACHE)
done
job=0; supervisor_pid=0; supervisor_start=''; supervisor_lanzado=0; supervisor_recolectado=1; supervisor_detenido=0; intento_supervisor_pendiente=0
pid=0; inicio=''; estado_sesion=AUSENTE
monitor=0; monitor_start=''; monitor_pid_atestado=0; monitor_start_atestado=''; monitor_lanzado=0; monitor_recolectado=1; salida_monitor=125
carril_activo=NINGUNO; salida_job=125; limpieza_activos=GO; argv_sha=''; hijo_argv_sha=''
fase_actual=PREFLIGHT; motivo_terminal=NINGUNO; terminal_escrito=0
reservas_total=0; hijos_total=0; exec_total=0; finalizaciones_total=0; carriles_go=0
identidad_pid() {
    local esperado_pid=$1 esperado_inicio=$2 linea real_pid _ppid _pgid _sid real_inicio
    [[ $esperado_pid =~ ^[1-9][0-9]*$ && $esperado_inicio =~ ^[0-9]+$ ]] || return 2
    [[ -d /proc/$esperado_pid ]] || return 1
    [[ -r /proc/$esperado_pid/stat ]] || return 2
    linea=$(campos_proceso "$esperado_pid") || { [[ -d /proc/$esperado_pid ]] && return 2 || return 1; }
    IFS=$'\t' read -r real_pid _ppid _pgid _sid real_inicio <<<"$linea"
    [[ $real_pid == "$esperado_pid" && $real_inicio == "$esperado_inicio" ]] && return 0
    return 2
}
estado_kernel_pid() {
    local objetivo=$1 linea resto
    [[ -d /proc/$objetivo ]] || return 1
    [[ -r /proc/$objetivo/stat ]] || return 2
    linea=$(<"/proc/$objetivo/stat")
    resto=${linea##*) }
    printf '%s\n' "${resto%% *}"
}
asegurar_supervisor() {
    local linea real_pid real_ppid _pgid _sid real_inicio
    [[ $supervisor_lanzado -eq 1 && $supervisor_recolectado -eq 0 && $job =~ ^[1-9][0-9]*$ ]] || return 2
    if [[ $supervisor_start =~ ^[0-9]+$ ]]; then identidad_pid "$job" "$supervisor_start"; return; fi
    [[ -d /proc/$job ]] || return 1
    [[ -r /proc/$job/stat ]] || return 2
    linea=$(campos_proceso "$job") || { [[ -d /proc/$job ]] && return 2 || return 1; }
    IFS=$'\t' read -r real_pid real_ppid _pgid _sid real_inicio <<<"$linea"
    [[ $real_pid == "$job" && $real_ppid == "$$" && $real_inicio =~ ^[0-9]+$ ]] || return 2
    supervisor_pid=$job; supervisor_start=$real_inicio
}
asegurar_monitor() {
    local linea real_pid real_ppid _pgid _sid real_inicio
    [[ $monitor_lanzado -eq 1 && $monitor_recolectado -eq 0 && $monitor =~ ^[1-9][0-9]*$ ]] || return 2
    if [[ $monitor_start =~ ^[0-9]+$ ]]; then identidad_pid "$monitor" "$monitor_start"; return; fi
    [[ -d /proc/$monitor ]] || return 1
    [[ -r /proc/$monitor/stat ]] || return 2
    linea=$(campos_proceso "$monitor") || { [[ -d /proc/$monitor ]] && return 2 || return 1; }
    IFS=$'\t' read -r real_pid real_ppid _pgid _sid real_inicio <<<"$linea"
    [[ $real_pid == "$monitor" && $real_ppid == "$$" && $real_inicio =~ ^[0-9]+$ ]] || return 2
    monitor_start=$real_inicio; monitor_pid_atestado=$monitor; monitor_start_atestado=$real_inicio
}
descubrir_sesion() {
    local cabecera sec trans _instante carril marcador_pid marcador_ppid marcador_pgid marcador_sid marcador_inicio marcador_argv marcador_salida _marcador_previo marcador_resultado
    local candidato='' candidato_inicio='' linea real_pid real_ppid real_pgid real_sid real_inicio p hijos=0 candidatos=0 uid_proceso cmd_sha estado
    if [[ $supervisor_lanzado -eq 0 ]]; then estado_sesion=AUSENTE; pid=0; inicio=''; return 1; fi
    if [[ ${a:-} && -f $a/02_HIJO_NACIDO.tsv ]]; then
        [[ ! -L $a/02_HIJO_NACIDO.tsv && $("$BB" wc -l <"$a/02_HIJO_NACIDO.tsv") -eq 2 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
        IFS= read -r cabecera <"$a/02_HIJO_NACIDO.tsv"
        [[ $cabecera == $'secuencia\ttransicion\tinstante_utc\tcarril\tpid\tppid\tpgid\tsid\tstarttime_ticks\targv_sha256\testado_salida\tprev_sha256\tresultado' ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
        IFS=$'\t' read -r sec trans _instante carril marcador_pid marcador_ppid marcador_pgid marcador_sid marcador_inicio marcador_argv marcador_salida _marcador_previo marcador_resultado < <("$BB" sed -n '2p' "$a/02_HIJO_NACIDO.tsv")
        [[ $("$BB" awk -F '\t' 'NR==2{print NF}' "$a/02_HIJO_NACIDO.tsv") -eq 13 && $sec == 2 && $trans == HIJO_NACIDO && $carril == "$carril_activo" && $marcador_pid =~ ^[1-9][0-9]*$ && $marcador_ppid == "$supervisor_pid" && $marcador_pgid == "$marcador_pid" && $marcador_sid == "$marcador_pid" && $marcador_inicio =~ ^[0-9]+$ && $marcador_argv == "$argv_sha" && $marcador_salida == PENDIENTE && $marcador_resultado == GO ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
        if [[ $supervisor_detenido -eq 1 ]]; then
            hijos=0
            for p in /proc/[0-9]*; do
                if [[ ! -r $p/stat ]]; then [[ -d $p ]] && { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }; continue; fi
                linea=$(campos_proceso "${p##*/}") || { if [[ -d $p ]]; then estado_sesion=IDENTIDAD_AMBIGUA; return 2; else continue; fi; }
                IFS=$'\t' read -r real_pid real_ppid _ _ real_inicio <<<"$linea"
                [[ $real_ppid == "$supervisor_pid" ]] || continue
                ((hijos+=1)); [[ $real_pid == "$marcador_pid" && $real_inicio == "$marcador_inicio" ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
            done
            [[ $hijos -le 1 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
        fi
        pid=$marcador_pid; inicio=$marcador_inicio; estado_sesion=VIVO_IDENTICO
        if sesion_viva; then estado=0; else estado=$?; fi
        return "$estado"
    fi
    [[ $supervisor_detenido -eq 1 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    if identidad_pid "$supervisor_pid" "$supervisor_start"; then estado=0; else estado=$?; fi
    [[ $estado -eq 0 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    for p in /proc/[0-9]*; do
        if [[ ! -r $p/stat ]]; then [[ -d $p ]] && { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }; continue; fi
        linea=$(campos_proceso "${p##*/}") || { if [[ -d $p ]]; then estado_sesion=IDENTIDAD_AMBIGUA; return 2; else continue; fi; }
        IFS=$'\t' read -r real_pid real_ppid real_pgid real_sid real_inicio <<<"$linea"
        [[ $real_ppid == "$supervisor_pid" ]] || continue
        ((hijos+=1))
        [[ -r $p/status && -r $p/cmdline ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
        uid_proceso=$("$BB" awk '/^Uid:/{print $2;exit}' "$p/status")
        cmd_sha=$(sha "$p/cmdline")
        if [[ $real_pid == "$real_pgid" && $real_pid == "$real_sid" && $uid_proceso == 999 && (($hijo_argv_sha =~ ^[0-9a-f]{64}$ && $cmd_sha == "$hijo_argv_sha") || ($argv_sha =~ ^[0-9a-f]{64}$ && $cmd_sha == "$argv_sha")) ]]; then
            candidato=$real_pid; candidato_inicio=$real_inicio; ((candidatos+=1))
        fi
    done
    if [[ $hijos -eq 0 ]]; then estado_sesion=AUSENTE; pid=0; inicio=''; return 1; fi
    [[ $hijos -eq 1 && $candidatos -eq 1 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    if identidad_pid "$candidato" "$candidato_inicio"; then estado=0; else estado=$?; fi
    [[ $estado -eq 0 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    pid=$candidato; inicio=$candidato_inicio; estado_sesion=VIVO_IDENTICO
    return 0
}
sesion_viva() {
    local p linea real_pid _ppid _pgid real_sid real_inicio estado vivos=0
    [[ $estado_sesion == AUSENTE ]] && return 1
    [[ $estado_sesion == VIVO_IDENTICO && $pid =~ ^[1-9][0-9]*$ && $inicio =~ ^[0-9]+$ ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    if identidad_pid "$pid" "$inicio"; then estado=0; else estado=$?; fi
    [[ $estado -eq 2 ]] && { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    [[ $estado -eq 0 ]] && vivos=1
    for p in /proc/[0-9]*; do
        if [[ ! -r $p/stat ]]; then [[ -d $p ]] && { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }; continue; fi
        linea=$(campos_proceso "${p##*/}") || { if [[ -d $p ]]; then estado_sesion=IDENTIDAD_AMBIGUA; return 2; else continue; fi; }
        IFS=$'\t' read -r real_pid _ppid _pgid real_sid real_inicio <<<"$linea"
        [[ $real_sid == "$pid" ]] || continue
        if identidad_pid "$real_pid" "$real_inicio"; then estado=0; else estado=$?; fi
        [[ $estado -eq 2 ]] && { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
        [[ $estado -eq 0 ]] && vivos=1
    done
    [[ $vivos -eq 1 ]] && return 0
    estado_sesion=AUSENTE
    return 1
}
senalar_sesion() {
    local senal=$1 p linea real_pid _ppid real_pgid real_sid real_inicio estado miembros=0 lider=0 estado_supervisor
    [[ $estado_sesion == AUSENTE ]] && return 1
    [[ $estado_sesion == VIVO_IDENTICO && $supervisor_detenido -eq 1 ]] || return 2
    if identidad_pid "$supervisor_pid" "$supervisor_start"; then estado_supervisor=0; else estado_supervisor=$?; fi
    [[ $estado_supervisor -eq 0 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    estado_supervisor=$(estado_kernel_pid "$supervisor_pid") || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    [[ $estado_supervisor == T || $estado_supervisor == t ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    if identidad_pid "$pid" "$inicio"; then estado=0; else estado=$?; fi
    [[ $estado -eq 0 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    for p in /proc/[0-9]*; do
        if [[ ! -r $p/stat ]]; then [[ -d $p ]] && { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }; continue; fi
        linea=$(campos_proceso "${p##*/}") || { if [[ -d $p ]]; then estado_sesion=IDENTIDAD_AMBIGUA; return 2; else continue; fi; }
        IFS=$'\t' read -r real_pid _ppid real_pgid real_sid real_inicio <<<"$linea"
        [[ $real_sid == "$pid" ]] || continue
        if identidad_pid "$real_pid" "$real_inicio"; then estado=0; else estado=$?; fi
        [[ $estado -eq 0 && $real_pgid == "$pid" ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
        ((miembros+=1)); [[ $real_pid == "$pid" ]] && lider=1
    done
    [[ $miembros -ge 1 && $lider -eq 1 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    kill -"$senal" -- "-$pid" 2>/dev/null || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    if identidad_pid "$supervisor_pid" "$supervisor_start"; then estado_supervisor=0; else estado_supervisor=$?; fi
    [[ $estado_supervisor -eq 0 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    estado_supervisor=$(estado_kernel_pid "$supervisor_pid") || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    [[ $estado_supervisor == T || $estado_supervisor == t ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    if identidad_pid "$pid" "$inicio"; then estado=0; else estado=$?; fi
    [[ $estado -eq 0 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    miembros=0; lider=0
    for p in /proc/[0-9]*; do
        if [[ ! -r $p/stat ]]; then [[ -d $p ]] && { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }; continue; fi
        linea=$(campos_proceso "${p##*/}") || { if [[ -d $p ]]; then estado_sesion=IDENTIDAD_AMBIGUA; return 2; else continue; fi; }
        IFS=$'\t' read -r real_pid _ppid real_pgid real_sid real_inicio <<<"$linea"
        [[ $real_sid == "$pid" ]] || continue
        if identidad_pid "$real_pid" "$real_inicio"; then estado=0; else estado=$?; fi
        [[ $estado -eq 0 && $real_pgid == "$pid" ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
        ((miembros+=1)); [[ $real_pid == "$pid" ]] && lider=1
    done
    [[ $miembros -ge 1 && $lider -eq 1 ]] || { estado_sesion=IDENTIDAD_AMBIGUA; return 2; }
    return 0
}
esperar_sesion_quiescente() {
    local ciclos=$1 i estado p linea real_pid _ppid _pgid real_sid real_inicio estado_kernel miembros
    for ((i=0; i<ciclos; i++)); do
        miembros=0; estado=0
        if identidad_pid "$pid" "$inicio"; then :; else return 2; fi
        for p in /proc/[0-9]*; do
            if [[ ! -r $p/stat ]]; then [[ -d $p ]] && return 2; continue; fi
            if ! linea=$(campos_proceso "${p##*/}"); then [[ -d $p ]] && return 2; continue; fi
            IFS=$'\t' read -r real_pid _ppid _pgid real_sid real_inicio <<<"$linea"
            [[ $real_sid == "$pid" ]] || continue
            identidad_pid "$real_pid" "$real_inicio" || return 2
            estado_kernel=$(estado_kernel_pid "$real_pid") || return 2
            ((miembros+=1)); [[ $estado_kernel == Z ]] || estado=1
        done
        [[ $miembros -ge 1 ]] || return 2
        [[ $estado -eq 0 ]] && return 0
        "$BB" sleep 0.1
    done
    return 124
}
esperar_supervisor_acotado() {
    local ciclos=$1 i estado identidad
    [[ $supervisor_lanzado -eq 1 && $supervisor_recolectado -eq 0 && $job =~ ^[1-9][0-9]*$ ]] || return 2
    if [[ ! -d /proc/$job ]]; then
        if wait "$job"; then salida_job=0; else salida_job=$?; fi
        job=0; supervisor_recolectado=1; supervisor_detenido=0
        return 0
    fi
    [[ $supervisor_start =~ ^[0-9]+$ ]] || return 2
    for ((i=0; i<ciclos; i++)); do
        if identidad_pid "$job" "$supervisor_start"; then identidad=0; else identidad=$?; fi
        [[ $identidad -eq 2 ]] && return 2
        [[ $identidad -eq 1 ]] && break
        estado=$(estado_kernel_pid "$job") || { if [[ -d /proc/$job ]]; then return 2; else break; fi; }
        [[ $estado == Z ]] && break
        "$BB" sleep 0.1
    done
    if [[ -d /proc/$job ]]; then
        if identidad_pid "$job" "$supervisor_start"; then identidad=0; else identidad=$?; fi
        [[ $identidad -eq 0 ]] || { [[ $identidad -eq 1 ]] || return 2; }
    fi
    if [[ -d /proc/$job ]]; then
        [[ $(estado_kernel_pid "$job") == Z ]] || return 124
    fi
    if wait "$job"; then salida_job=0; else salida_job=$?; fi
    job=0; supervisor_recolectado=1; supervisor_detenido=0
    return 0
}
esperar_monitor_acotado() {
    local ciclos=$1 i estado identidad monitor_guardado=$monitor
    [[ $monitor_lanzado -eq 1 && $monitor_recolectado -eq 0 && $monitor_guardado =~ ^[1-9][0-9]*$ ]] || return 2
    if [[ ! -d /proc/$monitor_guardado ]]; then
        if wait "$monitor_guardado" 2>/dev/null; then salida_monitor=0; else salida_monitor=$?; fi
        monitor=0; monitor_start=''; monitor_recolectado=1
        return 0
    fi
    [[ $monitor_start =~ ^[0-9]+$ ]] || return 2
    for ((i=0; i<ciclos; i++)); do
        if identidad_pid "$monitor_guardado" "$monitor_start"; then identidad=0; else identidad=$?; fi
        [[ $identidad -eq 2 ]] && return 2
        [[ $identidad -eq 1 ]] && break
        estado=$(estado_kernel_pid "$monitor_guardado") || { if [[ -d /proc/$monitor_guardado ]]; then return 2; else break; fi; }
        [[ $estado == Z ]] && break
        "$BB" sleep 0.1
    done
    if [[ -d /proc/$monitor_guardado ]]; then
        if identidad_pid "$monitor_guardado" "$monitor_start"; then identidad=0; else identidad=$?; fi
        [[ $identidad -eq 0 ]] || { [[ $identidad -eq 1 ]] || return 2; }
    fi
    if [[ -d /proc/$monitor_guardado ]]; then
        [[ $(estado_kernel_pid "$monitor_guardado") == Z ]] || return 124
    fi
    if wait "$monitor_guardado" 2>/dev/null; then salida_monitor=0; else salida_monitor=$?; fi
    monitor=0; monitor_start=''; monitor_recolectado=1
    return 0
}
detener_monitor() {
    local estado
    [[ $monitor_lanzado -eq 0 || $monitor_recolectado -eq 1 ]] && return 0
    if asegurar_monitor; then estado=0; else estado=$?; fi
    [[ $estado -eq 2 ]] && return 2
    if [[ $estado -eq 0 ]]; then
        if identidad_pid "$monitor" "$monitor_start"; then kill -TERM "$monitor" 2>/dev/null || true; else estado=$?; [[ $estado -eq 1 ]] || return 2; fi
    fi
    esperar_monitor_acotado 50 && return 0
    estado=$?
    [[ $estado -eq 2 ]] && return 2
    if identidad_pid "$monitor" "$monitor_start"; then estado=0; else estado=$?; fi
    [[ $estado -eq 2 ]] && return 2
    if [[ $estado -eq 0 ]]; then kill -KILL "$monitor" 2>/dev/null || true; fi
    esperar_monitor_acotado 50
}
detener_supervisor_verificado() {
    local i estado identidad
    if asegurar_supervisor; then identidad=0; else identidad=$?; fi
    [[ $identidad -eq 0 ]] || return "$identidad"
    if identidad_pid "$job" "$supervisor_start"; then identidad=0; else identidad=$?; fi
    [[ $identidad -eq 0 ]] || return "$identidad"
    kill -STOP "$job" 2>/dev/null || { if identidad_pid "$job" "$supervisor_start"; then return 2; else return $?; fi; }
    for ((i=0; i<100; i++)); do
        if identidad_pid "$job" "$supervisor_start"; then identidad=0; else identidad=$?; fi
        [[ $identidad -eq 0 ]] || return "$identidad"
        estado=$(estado_kernel_pid "$job") || { if [[ -d /proc/$job ]]; then return 2; else return 1; fi; }
        if [[ $estado == T || $estado == t ]]; then supervisor_detenido=1; return 0; fi
        "$BB" sleep 0.01
    done
    return 2
}
terminar_activos() {
    local motivo=$1 estado ambigua=0 linea_diario
    limpieza_activos=NO-GO
    if [[ $intento_supervisor_pendiente -eq 1 ]]; then
        estado_sesion=IDENTIDAD_AMBIGUA; ambigua=1
        if [[ $control_creado -eq 1 && $diario_creado -eq 1 && -d $control && -f ${diario:-/no-diario} ]]; then
            linea_diario=$(printf '0\t%s\t%s\tLANZAMIENTO_SUPERVISOR_PENDIENTE\t%s\tpendiente=1\tNO_SELLADO' "$(utc)" "$motivo" "$$")
            if [[ ${diario_abierto:-0} -eq 1 ]]; then printf '%s\n' "$linea_diario" >&9; else printf '%s\n' "$linea_diario" >>"$control/diario.tsv"; fi
        fi
    fi
    if [[ $supervisor_lanzado -eq 1 && $supervisor_recolectado -eq 0 ]]; then
        if detener_supervisor_verificado; then estado=0; else estado=$?; fi
        [[ $estado -eq 0 ]] || ambigua=1
        if descubrir_sesion; then estado=0; else estado=$?; fi
        [[ $estado -eq 2 ]] && ambigua=1
    elif [[ $supervisor_lanzado -eq 0 && $intento_supervisor_pendiente -eq 0 ]]; then
        estado_sesion=AUSENTE
    elif [[ $estado_sesion != VIVO_IDENTICO && $estado_sesion != AUSENTE ]]; then
        estado_sesion=IDENTIDAD_AMBIGUA; ambigua=1
    fi
    if detener_monitor; then estado=0; else estado=$?; fi
    [[ $estado -eq 0 ]] || ambigua=1
    if [[ $estado_sesion == VIVO_IDENTICO ]]; then
        if senalar_sesion TERM; then estado=0; else estado=$?; fi
        [[ $estado -eq 2 ]] && ambigua=1
        if esperar_sesion_quiescente 50; then estado=0; else estado=$?; fi
        if [[ $estado -eq 124 ]]; then
            if senalar_sesion KILL; then estado=0; else estado=$?; fi
            [[ $estado -eq 2 ]] && ambigua=1
            if esperar_sesion_quiescente 50; then estado=0; else estado=$?; fi
        fi
        [[ $estado -eq 0 ]] || ambigua=1
    elif [[ $estado_sesion != AUSENTE ]]; then
        ambigua=1
    fi
    if [[ $supervisor_lanzado -eq 1 && $supervisor_recolectado -eq 0 && $supervisor_detenido -eq 1 ]]; then
        if identidad_pid "$job" "$supervisor_start"; then estado=0; else estado=$?; fi
        if [[ $estado -eq 0 ]]; then
            kill -TERM "$job" 2>/dev/null || true
            if identidad_pid "$job" "$supervisor_start"; then estado=0; else estado=$?; fi
            if [[ $estado -eq 0 ]]; then
                if ! kill -CONT "$job" 2>/dev/null; then
                    if identidad_pid "$job" "$supervisor_start"; then estado=0; else estado=$?; fi
                    [[ $estado -eq 1 ]] || ambigua=1
                fi
            elif [[ $estado -eq 2 ]]; then
                ambigua=1
            fi
        elif [[ $estado -eq 2 ]]; then
            ambigua=1
        fi
    fi
    if [[ $supervisor_lanzado -eq 1 && $supervisor_recolectado -eq 0 ]]; then
        if esperar_supervisor_acotado 50; then estado=0; else estado=$?; fi
        if [[ $estado -eq 124 ]]; then
            if identidad_pid "$job" "$supervisor_start"; then estado=0; else estado=$?; fi
            if [[ $estado -eq 0 ]]; then kill -KILL "$job" 2>/dev/null || true; elif [[ $estado -eq 2 ]]; then ambigua=1; fi
            if esperar_supervisor_acotado 50; then estado=0; else estado=$?; fi
        fi
        [[ $estado -eq 0 ]] || ambigua=1
    fi
    if detener_monitor; then estado=0; else estado=$?; fi
    [[ $estado -eq 0 ]] || ambigua=1
    if sesion_viva; then estado=0; else estado=$?; fi
    [[ $estado -eq 1 && $estado_sesion == AUSENTE ]] || ambigua=1
    [[ $supervisor_lanzado -eq 0 || ($supervisor_recolectado -eq 1 && $job == 0) ]] || ambigua=1
    [[ $monitor_lanzado -eq 0 || ($monitor_recolectado -eq 1 && $monitor == 0) ]] || ambigua=1
    [[ $ambigua -eq 0 ]] && limpieza_activos=GO
    if [[ $control_creado -eq 1 && $diario_creado -eq 1 && -d $control && -f ${diario:-/no-diario} ]]; then
        linea_diario=$(printf '0\t%s\t%s\tLIMPIEZA_ACTIVOS\t%s\tsupervisor=%s:%s:lanzado_%s:recolectado_%s,sesion=%s:%s:%s,monitor=%s:%s:lanzado_%s:recolectado_%s,resultado=%s\tNO_SELLADO' "$(utc)" "$motivo" "$$" "$supervisor_pid" "$supervisor_start" "$supervisor_lanzado" "$supervisor_recolectado" "$pid" "$inicio" "$estado_sesion" "$monitor_pid_atestado" "$monitor_start_atestado" "$monitor_lanzado" "$monitor_recolectado" "$limpieza_activos")
        if [[ ${diario_abierto:-0} -eq 1 ]]; then printf '%s\n' "$linea_diario" >&9; else printf '%s\n' "$linea_diario" >>"$control/diario.tsv"; fi
    fi
    [[ $limpieza_activos == GO ]]
}
clasificar_fallo() {
    if [[ $preflight_cerrado -eq 0 ]]; then printf 'ROJO_PREFLIGHT\n'; return; fi
    case $fase_actual in
        PROTOCOLO) printf 'ROJO_PROTOCOLO\n' ;;
        EJECUCION) printf 'ROJO_EJECUCION\n' ;;
        EVIDENCIA|SELLADO) printf 'ROJO_EVIDENCIA\n' ;;
        POSTCONDICION) printf 'ROJO_POSTCONDICION\n' ;;
        UNANIMIDAD) printf 'ROJO_UNANIMIDAD\n' ;;
        *) printf 'ROJO_PROTOCOLO\n' ;;
    esac
}
escribir_terminal() {
    local clasificacion=$1 motivo=$2 rc=$3 resultado=$4
    [[ $terminal_escrito -eq 0 ]] || return 0
    [[ ${efd:-} =~ ^[0-9]+$ ]] || return 1
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$(utc)" "$fase_actual" "$carril_activo" "$clasificacion" "$motivo" "$rc" "$reservas_total" "$hijos_total" "$exec_total" "$finalizaciones_total" "$carriles_go" "$resultado" >&"$efd"
    terminal_escrito=1; motivo_terminal=$motivo
}
fallar_carril() {
    local motivo=$1 clasificacion
    global_rojo=1; motivo_terminal=$motivo
    clasificacion=$(clasificar_fallo)
    log "$carril_activo" ROJO_TERMINAL "$clasificacion:$motivo" || true
    terminar_activos "$motivo" || true
    escribir_terminal "$clasificacion" "$motivo:limpieza_$limpieza_activos" 111 NO-GO || true
    exit 111
}
cierre_emergencia() {
    local rc=$? clasificacion linea_diario
    [[ $rc -eq 0 ]] && return
    trap - EXIT HUP INT TERM
    set +e
    terminar_activos "EXIT_$rc" || true
    clasificacion=$(clasificar_fallo)
    escribir_terminal "$clasificacion" "${motivo_terminal:-EXIT_$rc}:EXIT_$rc:limpieza_$limpieza_activos" "$rc" NO-GO || true
    if [[ $control_creado -eq 1 && $diario_creado -eq 1 && -d $control && -f ${diario:-/no-diario} ]]; then
        linea_diario=$(printf '0\t%s\tEMERGENCIA\tLIMPIEZA\t%s\trc=%s,activos=%s\tNO_SELLADO' "$(utc)" "$$" "$rc" "$limpieza_activos")
        if [[ ${diario_abierto:-0} -eq 1 ]]; then printf '%s\n' "$linea_diario" >&9; else printf '%s\n' "$linea_diario" >>"$control/diario.tsv"; fi
    fi
    if [[ ${efd:-} =~ ^[0-9]+$ ]]; then exec {efd}>&-; efd=; fi
    if [[ ${diario_abierto:-0} -eq 1 ]]; then exec 9>&-; diario_abierto=0; fi
    exit "$rc"
}
trap cierre_emergencia EXIT
trap 'motivo_terminal=SENAL_HUP; exit 129' HUP
trap 'motivo_terminal=SENAL_INT; exit 130' INT
trap 'motivo_terminal=SENAL_TERM; exit 143' TERM
exec {le}</srv/fabrica/orquesta/home/evidencias
exec {lr}</srv/fabrica/orquesta/home/revisiones
exec {lt}</var/tmp
"$FLOCK" -n "$le"; "$FLOCK" -n "$lr"; "$FLOCK" -n "$lt"
inicio_control=$(utc)
declare -a ausencias=()
fallo_ausencia=0
for r in "${rutas[@]}"; do
    if [[ -e $r || -L $r ]]; then ausencias+=(EXISTE); fallo_ausencia=1; else ausencias+=(AUSENTE); fi
done
[[ ! -e $control && ! -L $control ]]
[[ $fallo_ausencia -eq 0 ]]
"$BB" mkdir -m 0700 "$control"
control_creado=1
printf 'instante_utc\tfase\tcarril\tclasificacion\tmotivo\testado_salida_coordinador\treservas\thijos_nacidos\texec_confirmados\tfinalizaciones\tcarriles_go\tresultado\n' >"$control/estado_terminal.tsv"
exec {efd}>>"$control/estado_terminal.tsv"
diario=$control/diario.tsv
printf 'secuencia\tinstante_utc\tfase\tevento\tpid\tdetalle\tprev_sha256\n' >"$diario"
diario_creado=1
exec 9>>"$diario"
diario_abierto=1
secuencia=0; prev=INICIO
log() {
    local fase=$1 evento=$2 detalle=$3 linea
    ((secuencia+=1))
    linea=$(printf '%s\t%s\t%s\t%s\t%s\t%s\t%s' "$secuencia" "$(utc)" "$fase" "$evento" "$$" "$detalle" "$prev")
    printf '%s\n' "$linea" >&9
    prev=$(printf '%s\n' "$linea" | "$BB" sha256sum | "$BB" awk '{print $1}')
}
printf 'ordinal\truta_absoluta\tinstante_utc\tpid_observador\tuid_efectivo\tgid_efectivo\ttipo_observado\tresultado\n' >"$control/ausencia_inicial.tsv"
for i in "${!rutas[@]}"; do
    resultado=NO-GO; [[ ${ausencias[$i]} == AUSENTE ]] && resultado=GO
    printf '%s\t%s\t%s\t%s\t0\t0\t%s\t%s\n' "$((i+1))" "${rutas[$i]}" "$(utc)" "$$" "${ausencias[$i]}" "$resultado" >>"$control/ausencia_inicial.tsv"
done
log PREFLIGHT AUSENCIAS "26:$fallo_ausencia"
extraer >"$control/coordinador_v3.sh"
[[ $(sha "$control/coordinador_v3.sh") == "$coordinador_sha" ]]
"$BB" chown 999:982 "$control" "$control/coordinador_v3.sh"
"$BB" chmod 0700 "$control"; "$BB" chmod 0500 "$control/coordinador_v3.sh"
log PREFLIGHT COORDINADOR "$coordinador_sha"
: >"$control/ventana.lock"
exec {lv}<"$control/ventana.lock"
"$FLOCK" -n "$lv"
printf 'sha_documental\tpadre_documental\tantecesor_documental\tbase_tecnica\tarbol\tblob_documento\tsha256_documento\tlineas_documento\tbytes_documento\n%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$sha_documental" "$PADRE_DOCUMENTAL" "$ANTECESOR_DOCUMENTAL" "$BASE_TECNICA" "$arbol" "$blob" "$doc_sha" "$doc_lineas" "$doc_bytes" >"$control/contrato.tsv"
printf 'ruta\tmodo\tlineas\tsha256\n' >"$control/bytes_tecnicos.tsv"
ledger=$fuente/tools/o3a_v5_conductor/fuentes_v5.tsv
base_fuentes_rel=deploy/postgresql/autorizacion_atestada_v3/pruebas_sql
while IFS=$'\t' read -r ruta lineas huella; do
    [[ $ruta == archivo ]] && continue
    [[ -n $ruta && $ruta != */* && $ruta != . && $ruta != .. ]] || exit 1
    ruta_relativa=$base_fuentes_rel/$ruta
    fichero=$fuente/$ruta_relativa
    [[ ! -L $fichero && -f $fichero && $("$BB" stat -c %a "$fichero") == 644 && $("$BB" wc -l <"$fichero") -eq $lineas && $(sha "$fichero") == "$huella" ]]
    printf '%s\t100644\t%s\t%s\n' "$ruta_relativa" "$lineas" "$huella" >>"$control/bytes_tecnicos.tsv"
done <"$ledger"
tecnicos_fijos=(
    "tools/o3a_v5_conductor/fuentes_v5.tsv|100644|11|ba2b0a1c9838f57ca43d53ec6133ae74bfa7452f7511421e367d7fc5f3d079b0"
    "tools/o3a_v5_conductor/conductor.sh|100755|205|cc1f47f8d4dd49df71effd4886de732748a261f9fc1964a4c524037d5e4be0d7"
    "tools/o3a_v5_conductor/README.md|100644|242|7933b7617a92e22fddf6f888b9fddcc74ea0224616e0340d9c9f36d7d2533ce8"
    "go.mod|100644|45|a447f8cf5217130ed336bf78bcd93211659e1c7ba547673e9a1753ef6ba7eeca"
    "Dockerfile|100644|253|6045bd15bf061de0db0e70c3858344b48db39fc31dbe9fc1f76c6e397f81bf5d"
    "tools/o3a_v5_conductor/conductor_c01_c07.sh|100755|61|9236cd1d7ca6cb1ebfbfb511c3913b96be980ba28acebeedad91e2f70deaedc3"
    "tools/o3a_v5_conductor/conductor_c02_interleavings.sh|100755|24|67d993b84c3c337a230a53f90d12ed91c08021c8ef88279cc93fd8557659f78a"
    "tools/o3a_v5_conductor/conductor_c08_c11_c14.sh|100755|83|a07c561ad3a616a59256f3bcb317cf803c3c0a0b01f6a09eabad3cfb8614329e"
    "tools/o3a_v5_conductor/conductor_c10_c13.sh|100755|98|70e2413081711d3f7019fa5daa4f5172d2ddde2b6eb5f2b121dc7d82f478cd46"
    "tools/o3a_v5_conductor/conductor_c15_c21.sh|100755|60|edf6a719eed6c96f920265c8c51df16b458d172744cb6cf06b581188e13ac0fa"
    "tools/o3a_v5_conductor/conductor_c19.sh|100755|50|cb8629842b2635ea36293d788ed9980f174b18e19d570b3f5e4f124f830d81f5"
    "tools/o3a_v5_conductor/conductor_c20.sh|100755|38|189c59f0ee235722a3aeac1491bb21de26015da99746027a4a20f489ec1b9b8e"
)
for spec in "${tecnicos_fijos[@]}"; do
    IFS='|' read -r ruta modo lineas huella <<<"$spec"; fichero=$fuente/$ruta
    [[ ! -L $fichero && -f $fichero && 100$("$BB" stat -c %a "$fichero") == "$modo" && $("$BB" wc -l <"$fichero") -eq $lineas && $(sha "$fichero") == "$huella" ]]
    printf '%s\t%s\t%s\t%s\n' "$ruta" "$modo" "$lineas" "$huella" >>"$control/bytes_tecnicos.tsv"
done
for acta in bb23278e0b5660b611cb9c7faa19b197e0283a19 22fde5f72b2ce4bf92c2ae32bee4d7ab47a4e36b baea7d014e403412dd77c7cbd380e89023dd9717 fd44303eaf5dd27d2fbbd221266b13fe1473b4ba; do
    "$GIT" -C "$fuente" cat-file -e "$acta^{commit}"
    [[ $("$GIT" -C "$fuente" rev-parse "$acta^") == "$BASE_TECNICA" ]]
    [[ $("$GIT" -C "$fuente" diff-tree --no-commit-id --name-only -r "$acta" | "$BB" wc -l) -eq 1 ]]
done
log PREFLIGHT OBJETOS_Y_BYTES VERIFICADOS
printf 'ordinal\tclase\tcarril\tsimbolo\truta_absoluta\tdebe_ausente_inicial\n' >"$control/rutas_reservadas.tsv"
for i in "${!rutas[@]}"; do printf '%s\t%s\t%s\t%s\t%s\tsi\n' "$((i+1))" "${clases_ruta[$i]}" "${carriles_ruta[$i]}" "${simbolos_ruta[$i]}" "${rutas[$i]}" >>"$control/rutas_reservadas.tsv"; done
printf 'fase\truta\tclasificacion\tinstante_utc\tpropietario\tgrupo\tmodo\ttipo\tnlink\tbytes\tsha256\taccion\n' >"$control/incidencia_telemetria_pre.tsv"
incidencia=/srv/fabrica/orquesta/home/.config/go/telemetry/local/go@go1.26.6-go1.26.6-linux-amd64-2026-08-21.v1.count
[[ ! -L $incidencia && -f $incidencia && $("$BB" stat -c '%U|%G|%a|%h|%s|%F' "$incidencia") == 'orquesta|orquesta|644|1|16384|regular file' ]]
incidencia_pre=$(sha "$incidencia")
[[ $incidencia_pre == e588a3240689f992fda01d36b8d642edb893f0c6dea2ca72d6f12784aa220475 ]]
printf 'PRE\t%s\tPREEXISTENTE_AJENA\t%s\torquesta\torquesta\t644\tregular\t1\t16384\t%s\tNO_TOCAR\n' "$incidencia" "$(utc)" "$incidencia_pre" >>"$control/incidencia_telemetria_pre.tsv"
[[ ! -L /var/tmp && $("$BB" stat -c '%U|%G|%a|%F' /var/tmp) == 'root|root|1777|directory' ]]
[[ ! -L $GOROOT_ORIGEN && -d $GOROOT_ORIGEN && $("$BB" stat -c '%U|%G|%a|%h|%F' "$GOROOT_ORIGEN") == 'orquesta|orquesta|555|6|directory' ]]
emitir_inventario_goroot "$GOROOT_ORIGEN" >"$control/toolchain_goroot_inventario.tsv"
validar_inventario_goroot "$control/toolchain_goroot_inventario.tsv"
goroot_bytes=0
while IFS= read -r entrada; do ((goroot_bytes+=$("$BB" stat -c %s "$entrada"))); done < <("$BB" find "$GOROOT_ORIGEN" -xdev -type f -print)
[[ $goroot_bytes -eq 215332664 ]]
"$BB" mkdir -m 0700 "$goroot"
[[ ! -L $goroot && $("$BB" stat -c '%U|%G|%a|%F' "$goroot") == 'root|root|700|directory' ]]
"$BB" cp -R -P "$GOROOT_ORIGEN/." "$goroot/"
"$BB" chown -R 0:0 "$goroot"
"$BB" find "$goroot" -xdev -type d -exec "$BB" chmod 0555 '{}' ';'
"$BB" find "$goroot" -xdev -type f -exec "$BB" chmod 0444 '{}' ';'
while IFS= read -r entrada; do
    relativa=${entrada#"$GOROOT_ORIGEN"/}
    "$BB" chmod 0555 "$goroot/$relativa"
done < <("$BB" find "$GOROOT_ORIGEN" -xdev -type f -perm 0555 -print)
[[ $("$BB" find "$goroot" -xdev -mindepth 1 -print | "$BB" wc -l) -eq 12870 ]]
[[ $("$BB" find "$goroot" -xdev -mindepth 1 -type d -print | "$BB" wc -l) -eq 1334 ]]
[[ $("$BB" find "$goroot" -xdev -mindepth 1 -type f -print | "$BB" wc -l) -eq 11536 ]]
[[ $("$BB" find "$goroot" -xdev -mindepth 1 ! -type d ! -type f -print | "$BB" wc -l) -eq 0 ]]
goroot_metadatos_bad=0; goroot_bytes_copia=0; goroot_regulares_444=0; goroot_regulares_555=0
while IFS= read -r entrada; do
    IFS='|' read -r tipo u g m nlink < <("$BB" stat -c '%F|%U|%G|%a|%h' "$entrada")
    if [[ $tipo == directory ]]; then
        [[ $u:$g:$m == root:root:555 ]] || ((goroot_metadatos_bad+=1))
    elif [[ $tipo == 'regular file' || $tipo == 'regular empty file' ]]; then
        [[ $u:$g:$nlink == root:root:1 && ($m == 444 || $m == 555) ]] || ((goroot_metadatos_bad+=1))
        if [[ $m == 444 ]]; then ((goroot_regulares_444+=1)); else ((goroot_regulares_555+=1)); fi
        ((goroot_bytes_copia+=$("$BB" stat -c %s "$entrada")))
    else
        ((goroot_metadatos_bad+=1))
    fi
done < <("$BB" find "$goroot" -xdev -print)
[[ $goroot_metadatos_bad -eq 0 && $goroot_regulares_444 -eq 11524 && $goroot_regulares_555 -eq 12 && $goroot_bytes_copia -eq 215332664 ]]
[[ $("$BB" stat -c '%U|%G|%a|%h|%F' "$goroot") == 'root|root|555|6|directory' ]]
"$BB" cmp -s "$control/toolchain_goroot_inventario.tsv" <(emitir_inventario_goroot "$goroot")
[[ $(sha "$GO") == "$GO_SHA" && $(sha "$GOFMT") == "$GOFMT_SHA" ]]
printf 'fase\truta_origen\truta_copia\tidentidad_sha256\tdirectorios_totales\tdirectorios_internos\tficheros_regulares\tregulares_0444\tregulares_0555\totras_entradas\tbytes_regulares\tpropietario_origen\tgrupo_origen\tmodo_origen\ttipo_origen\tnlink_origen\tpropietario_copia\tgrupo_copia\tmodo_copia\ttipo_copia\tnlink_copia\tpadre_copia\tpropietario_padre\tgrupo_padre\tmodo_padre\ttipo_padre\tgo_sha256\tgofmt_sha256\tresultado\nMATERIALIZADA\t%s\t%s\t%s\t1335\t1334\t11536\t11524\t12\t0\t215332664\torquesta\torquesta\t555\tdirectory\t6\troot\troot\t555\tdirectory\t6\t/var/tmp\troot\troot\t1777\tdirectory\t%s\t%s\tGO\n' "$GOROOT_ORIGEN" "$goroot" "$GOROOT_IDENTIDAD_SHA" "$GO_SHA" "$GOFMT_SHA" >"$control/toolchain_goroot.tsv"
log PREFLIGHT GOROOT_COPIA "$GOROOT_IDENTIDAD_SHA:12870:215332664"
printf 'carril\ttarget\thead\tarbol\tstatus_porcelain_v2_sha256\tremotos\talternates\tshallow\tcommondir\tgrafts\thardlinks\tpropietario\tgrupo\tmodo\ttipo\tnlink\tresultado\n' >"$control/clones.tsv"
printf 'carril\tbinario\truta\tsha256\tversion_esperada\tversion_observada\tstdout_sha256\tstderr_sha256\testado\tpid\tstarttime_ticks\tresultado\n' >"$control/toolchain.tsv"
printf 'funcion\truta_lstat\ttipo_lstat\ttexto_enlace\tpropietario_lstat\tgrupo_lstat\tmodo_lstat\tnlink_lstat\truta_real\ttipo_real\tpropietario_real\tgrupo_real\tmodo_real\tnlink_real\tsha256_real\tresultado\n' >"$control/binarios_lanzamiento.tsv"
printf 'carril\tfase\tproceso\torden\tnombre\tvalor\torigen\tresultado\n' >"$control/entornos_permitidos.tsv"
printf 'control\tinicio_utc\tfin_utc\tpid\tppid\tpgid\tsid\tuid\tgid\tumask\treservas\texec_confirmados\tfinalizaciones\tconductores_ejecutados\tresultado\tdetalle\n' >"$control/preflight.tsv"
foto "$control/procesos_preflight.tsv" PREFLIGHT
[[ $("$BB" awk -F '\t' 'NR>1 && $13=="conductor_o3a_v5"{n++}END{print n+0}' "$control/procesos_preflight.tsv") -eq 0 ]]
regulares=(
    "/usr/bin/busybox|df12634c17fcdca839ae5dc47d7627b7558511f7645de7c99ccf097a0f28ed5b|root|root|755|1"
    "/usr/bin/bash|3efccc187bafa75ff1e37d246270ab3e7aa559f242c7a52bf3ec2a1b5450bdbd|root|root|755|1"
    "/usr/bin/git|5516c9f362c29376ab9a499a33082f9f611941d8c75930c880e30ad109e39c9a|root|root|755|1"
    "/usr/bin/flock|59bc254984eefd83939a22a590d746942a4583a702b8fd2753bbb92d956e7d4c|root|root|755|1"
    "/usr/bin/setpriv|9e0d70d26a02c1cb4b984ab6f49a582b7a2c3508b1063ac23adc60073292ae7e|root|root|755|1"
    "/usr/bin/setsid|ade4a0f0c5df4627b456ab19a1869670e7a58a34b04d817e8703c0fa7b852478|root|root|755|1"
    "$GO|29e6e0b8be61beb1489ceae62b304343566de8a1dc700af74bde7aeb9c80ad45|root|root|555|1"
    "$GOFMT|0ef6fe2d15c972d15b8ecb75dcb9c861e042ac94a0e48baf0202a840648b0fdc|root|root|555|1"
    "/usr/bin/find|efe4843f166525b02f328f0c456a884d6a597f32d2c78cd615b6be0b64279bcf|root|root|755|1"
    "/usr/bin/pgrep|b1e03079307c5c602dd59afdb36f249e262ff2724f98e35a8ddc91d2ea2dffec|root|root|755|1"
    "/usr/bin/grep|4874cbc734792ae29aa38cd7978aedf1e646563af9e8ee0d5dab59ecb6406417|root|root|755|1"
)
for spec in "${regulares[@]}"; do
    IFS='|' read -r p esperado propietario grupo modo nlink <<<"$spec"; observado=$(sha "$p")
    [[ ! -L $p && -f $p && $observado == "$esperado" && $("$BB" stat -c '%U|%G|%a|%h|%F' "$p") == "$propietario|$grupo|$modo|$nlink|regular file" ]]
    printf 'ejecutable\t%s\tregular\t-\t%s\t%s\t%s\t%s\t%s\tregular\t%s\t%s\t%s\t%s\t%s\tGO\n' "$p" "$propietario" "$grupo" "$modo" "$nlink" "$p" "$propietario" "$grupo" "$modo" "$nlink" "$observado" >>"$control/binarios_lanzamiento.tsv"
done
enlaces=(
    "/usr/bin/env|../lib/cargo/bin/coreutils/env|/usr/lib/cargo/bin/coreutils/env|48893b0fb21436b54619db80486e83ef39dfccaf1aefe83dfa00c02d6146e8c0|115"
    "/usr/bin/dirname|../lib/cargo/bin/coreutils/dirname|/usr/lib/cargo/bin/coreutils/dirname|48893b0fb21436b54619db80486e83ef39dfccaf1aefe83dfa00c02d6146e8c0|115"
    "/usr/bin/mkdir|../lib/cargo/bin/coreutils/mkdir|/usr/lib/cargo/bin/coreutils/mkdir|48893b0fb21436b54619db80486e83ef39dfccaf1aefe83dfa00c02d6146e8c0|115"
    "/usr/bin/mktemp|../lib/cargo/bin/coreutils/mktemp|/usr/lib/cargo/bin/coreutils/mktemp|48893b0fb21436b54619db80486e83ef39dfccaf1aefe83dfa00c02d6146e8c0|115"
    "/usr/bin/sha256sum|../lib/cargo/bin/coreutils/sha256sum|/usr/lib/cargo/bin/coreutils/sha256sum|48893b0fb21436b54619db80486e83ef39dfccaf1aefe83dfa00c02d6146e8c0|115"
    "/usr/bin/date|../lib/cargo/bin/coreutils/date|/usr/lib/cargo/bin/coreutils/date|48893b0fb21436b54619db80486e83ef39dfccaf1aefe83dfa00c02d6146e8c0|115"
    "/usr/bin/stat|../lib/cargo/bin/coreutils/stat|/usr/lib/cargo/bin/coreutils/stat|48893b0fb21436b54619db80486e83ef39dfccaf1aefe83dfa00c02d6146e8c0|115"
    "/usr/bin/sort|../lib/cargo/bin/coreutils/sort|/usr/lib/cargo/bin/coreutils/sort|48893b0fb21436b54619db80486e83ef39dfccaf1aefe83dfa00c02d6146e8c0|115"
    "/usr/bin/wc|../lib/cargo/bin/coreutils/wc|/usr/lib/cargo/bin/coreutils/wc|48893b0fb21436b54619db80486e83ef39dfccaf1aefe83dfa00c02d6146e8c0|115"
    "/usr/bin/timeout|../lib/cargo/bin/coreutils/timeout|/usr/lib/cargo/bin/coreutils/timeout|48893b0fb21436b54619db80486e83ef39dfccaf1aefe83dfa00c02d6146e8c0|115"
    "/usr/bin/cp|gnucp|/usr/bin/gnucp|ddffb913956d7f8cbdd7722b8c331cea51e9a13c59e92de8c194ae30e5eb0b1e|1"
    "/usr/bin/rm|gnurm|/usr/bin/gnurm|175a15a35617f84bb86692b50a3e21b41d8e21fdea242e952b31f1e2e80b1a54|1"
    "/usr/bin/mv|gnumv|/usr/bin/gnumv|3a56b5e8f9b40ca7fb484bb00ba048c705f6520c7a9092b45394a9c916ba0b9b|1"
    "/usr/bin/awk|/etc/alternatives/awk|/usr/bin/gawk|a695aa25fdd3f207ece8b3628514c6506c45216ce5486f63d293a83e12dc9798|1"
    "/etc/alternatives/awk|/usr/bin/gawk|/usr/bin/gawk|a695aa25fdd3f207ece8b3628514c6506c45216ce5486f63d293a83e12dc9798|1"
    "/bin/sh|dash|/usr/bin/dash|c626229526bb58ec2d0f585f3c3ae1412e6f973b4353385042d11c38d8426917|1"
    "/usr/lib/git-core/git-upload-pack|git|/usr/lib/git-core/git|5516c9f362c29376ab9a499a33082f9f611941d8c75930c880e30ad109e39c9a|1"
)
for spec in "${enlaces[@]}"; do
    IFS='|' read -r p texto real esperado nlink_real <<<"$spec"
    [[ -L $p && $("$BB" readlink "$p") == "$texto" && $("$BB" realpath "$p") == "$real" ]]
    [[ $("$BB" stat -c '%U|%G|%a|%h|%F' "$p") == 'root|root|777|1|symbolic link' ]]
    observado=$(sha "$real")
    [[ $observado == "$esperado" && $("$BB" stat -L -c '%U|%G|%a|%h|%F' "$p") == "root|root|755|$nlink_real|regular file" ]]
    printf 'dependencia_script\t%s\tsymlink\t%s\troot\troot\t777\t1\t%s\tregular\troot\troot\t755\t%s\t%s\tGO\n' "$p" "$texto" "$real" "$nlink_real" "$observado" >>"$control/binarios_lanzamiento.tsv"
done
for c in "${carriles[@]}"; do
    orden=0
    for par in 'LC_ALL=C' 'PATH=/usr/bin:/bin' 'HOME=/nonexistent/o3a-v5-bootstrap' 'XDG_CONFIG_HOME=/nonexistent/o3a-v5-bootstrap' 'GIT_CONFIG_NOSYSTEM=1' 'GIT_TERMINAL_PROMPT=0' 'GIT_ALLOW_PROTOCOL=file' "PWD=$fuente" 'SHLVL=DERIVADO' '_=DERIVADO'; do
        ((orden+=1)); printf '%s\tPREPARACION\tcoordinador\t%s\t%s\t%s\tBOOTSTRAP_BASH\tGO\n' "$c" "$orden" "${par%%=*}" "${par#*=}" >>"$control/entornos_permitidos.tsv"
    done
    for fase_proceso in 'GO_VERSION|go' 'INTENTO_SUPERVISOR|setsid' 'PUENTE|coordinador_hijo' 'CONDUCCION|conductor' 'BLOQUE_NORMAL|bloque' 'BLOQUE_RACE|bloque' 'GO_BUILD_NORMAL|go' 'GO_BUILD_RACE|go' 'C20|go' 'C20_BUILD|go'; do
        IFS='|' read -r fase proceso <<<"$fase_proceso"; orden=0
        for par in "HOME=${home[$c]}" "XDG_CONFIG_HOME=${xdg[$c]}" "XDG_CACHE_HOME=${xcache[$c]}" "TMPDIR=${tmp[$c]}" "GOCACHE=${gocache[$c]}" "GOROOT=$goroot" 'LC_ALL=C' "PATH=$goroot/bin:/usr/bin:/bin" 'GOTOOLCHAIN=local' 'GOENV=off' 'GOPROXY=off' 'GOSUMDB=off'; do
            ((orden+=1)); printf '%s\t%s\t%s\t%s\t%s\t%s\tCONTRATO\tGO\n' "$c" "$fase" "$proceso" "$orden" "${par%%=*}" "${par#*=}" >>"$control/entornos_permitidos.tsv"
        done
    done
    printf '%s\tPUENTE\tcoordinador_hijo\t13\tPWD\tFUENTE\tBASH\tGO\n%s\tPUENTE\tcoordinador_hijo\t14\tSHLVL\tDERIVADO\tBASH\tGO\n%s\tPUENTE\tcoordinador_hijo\t15\t_\tDERIVADO\tBASH\tGO\n' "$c" "$c" "$c" >>"$control/entornos_permitidos.tsv"
    printf '%s\tCONDUCCION\tconductor\t13\tPWD\tTARGET\tBASH\tGO\n%s\tCONDUCCION\tconductor\t14\tSHLVL\tDERIVADO\tBASH\tGO\n%s\tCONDUCCION\tconductor\t15\t_\tDERIVADO\tBASH\tGO\n' "$c" "$c" "$c" >>"$control/entornos_permitidos.tsv"
    printf '%s\tBLOQUE_NORMAL\tbloque\t13\tCND_RUNTIME_TARGET\tTARGET\tCONDUCTOR_VERSIONADO\tGO\n%s\tBLOQUE_NORMAL\tbloque\t14\tCND_RACE\t0\tCONDUCTOR_VERSIONADO\tGO\n%s\tBLOQUE_NORMAL\tbloque\t15\tCND_CGO_ENABLED\t0\tCONDUCTOR_VERSIONADO\tGO\n' "$c" "$c" "$c" >>"$control/entornos_permitidos.tsv"
    printf '%s\tBLOQUE_RACE\tbloque\t13\tCND_RUNTIME_TARGET\tTARGET\tCONDUCTOR_VERSIONADO\tGO\n%s\tBLOQUE_RACE\tbloque\t14\tCND_RACE\t1\tCONDUCTOR_VERSIONADO\tGO\n%s\tBLOQUE_RACE\tbloque\t15\tCND_CGO_ENABLED\t1\tCONDUCTOR_VERSIONADO\tGO\n' "$c" "$c" "$c" >>"$control/entornos_permitidos.tsv"
    for fase_modo in 'GO_BUILD_NORMAL|0' 'GO_BUILD_RACE|1' 'C20|0_O_1' 'C20_BUILD|0_O_1'; do
        IFS='|' read -r fase modo_cnd <<<"$fase_modo"
        printf '%s\t%s\tproceso_versionado\t13\tCND_RUNTIME_TARGET\tTARGET\tCONDUCTOR_VERSIONADO\tGO\n%s\t%s\tproceso_versionado\t14\tCND_RACE\t%s\tCONDUCTOR_VERSIONADO\tGO\n%s\t%s\tproceso_versionado\t15\tCND_CGO_ENABLED\t%s\tCONDUCTOR_VERSIONADO\tGO\n' "$c" "$fase" "$c" "$fase" "$modo_cnd" "$c" "$fase" "$modo_cnd" >>"$control/entornos_permitidos.tsv"
    done
    printf '%s\tGO_BUILD_NORMAL\tgo\t16\tCGO_ENABLED\t0\tBLOQUE_VERSIONADO\tGO\n%s\tGO_BUILD_RACE\tgo\t16\tCGO_ENABLED\t1\tBLOQUE_VERSIONADO\tGO\n%s\tC20\tgo\t16\tGOFLAGS\tVACIO\tBLOQUE_VERSIONADO\tGO\n%s\tC20_BUILD\tgo\t16\tGOFLAGS\tVACIO\tBLOQUE_VERSIONADO\tGO\n%s\tC20_BUILD\tgo\t17\tCGO_ENABLED\t0\tBLOQUE_VERSIONADO\tGO\n' "$c" "$c" "$c" "$c" "$c" >>"$control/entornos_permitidos.tsv"
    for fase_bash in BLOQUE_NORMAL BLOQUE_RACE GO_BUILD_NORMAL GO_BUILD_RACE C20 C20_BUILD; do
        inicio_bash=16
        [[ $fase_bash == GO_BUILD_NORMAL || $fase_bash == GO_BUILD_RACE || $fase_bash == C20 ]] && inicio_bash=17
        [[ $fase_bash == C20_BUILD ]] && inicio_bash=18
        printf '%s\t%s\tproceso_versionado\t%s\tPWD\tTARGET_DERIVADO\tBASH\tGO\n%s\t%s\tproceso_versionado\t%s\tOLDPWD\tTARGET_DERIVADO\tBASH\tGO\n%s\t%s\tproceso_versionado\t%s\tSHLVL\tDERIVADO\tBASH\tGO\n%s\t%s\tproceso_versionado\t%s\t_\tDERIVADO\tBASH\tGO\n' "$c" "$fase_bash" "$inicio_bash" "$c" "$fase_bash" "$((inicio_bash+1))" "$c" "$fase_bash" "$((inicio_bash+2))" "$c" "$fase_bash" "$((inicio_bash+3))" >>"$control/entornos_permitidos.tsv"
    done
done
for c in "${carriles[@]}"; do
    "$BB" mkdir -m 0700 "${target[$c]}" "${atestacion[$c]}"
    "$BB" chown 999:982 "${target[$c]}" "${atestacion[$c]}"
    como_orquesta_git -c "safe.directory=$fuente" clone --no-local --no-hardlinks --no-checkout --no-tags --single-branch "$fuente" "${target[$c]}"
    como_orquesta_git -C "${target[$c]}" remote remove origin
    como_orquesta_git -C "${target[$c]}" checkout --detach "$sha_documental"
    [[ -z $(como_orquesta_git -C "${target[$c]}" remote) ]]
    [[ -z $(como_orquesta_git -C "${target[$c]}" status --porcelain=v2 --untracked-files=all) ]]
    [[ ! -e ${target[$c]}/.git/objects/info/alternates && ! -e ${target[$c]}/.git/shallow && ! -e ${target[$c]}/.git/commondir && ! -e ${target[$c]}/.git/info/grafts ]]
    hardlinks=0
    while IFS= read -r objeto; do [[ $("$BB" stat -c %h "$objeto") -eq 1 ]] || ((hardlinks+=1)); done < <("$BB" find "${target[$c]}/.git/objects" -type f -print)
    [[ $hardlinks -eq 0 ]]
    metadatos_clone=0
    while IFS= read -r entrada; do [[ $("$BB" stat -c '%u:%g' "$entrada") == 999:982 ]] || ((metadatos_clone+=1)); done < <("$BB" find "${target[$c]}" -print)
    [[ $metadatos_clone -eq 0 && ! -L ${target[$c]} && -d ${target[$c]} ]]
    printf '%s\t%s\t%s\t%s\t%s\t0\tausente\tausente\tausente\tausente\t0\torquesta\torquesta\t700\tdirectory\t%s\tGO\n' "$c" "${target[$c]}" "$sha_documental" "$(como_orquesta_git -C "${target[$c]}" rev-parse 'HEAD^{tree}')" e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 "$("$BB" stat -c %h "${target[$c]}")" >>"$control/clones.tsv"
    "$BB" mkdir -m 0700 "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" "${xdg[$c]}/go" "${xdg[$c]}/go/telemetry"
    printf '%s' 'off 2026-08-21' >"${xdg[$c]}/go/telemetry/mode"
    "$BB" chown -R 999:982 "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}"
    [[ $(sha "${xdg[$c]}/go/telemetry/mode") == "$MODE_SHA" ]]
    [[ $("$BB" stat -c '%U|%G|%a|%h|%F' "$goroot") == 'root|root|555|6|directory' ]]
    [[ $(sha "$GO") == "$GO_SHA" && $(sha "$GOFMT") == "$GOFMT_SHA" ]]
    a=${atestacion[$c]}
    printf 'fase\tproceso\torden\tnombre\tvalor\torigen\tresultado\n' >"$a/entorno_permitido.tsv"
    printf 'fase\tproceso\tpid\tppid\tnombre\tvalor\torigen\tresultado\n' >"$a/entorno_efectivo.tsv"
    printf 'fase\tproceso\tpid\tppid\tuid\tgid\tgrupos_suplementarios\tcap_inh\tcap_prm\tcap_eff\tcap_bnd\tcap_amb\tno_new_privs\tresultado\n' >"$a/privilegios.tsv"
    "$BB" env | registrar_entorno "$a/entorno_efectivo.tsv" PREPARACION coordinador "$$" "$PPID" HEREDADO_BOOTSTRAP
    "$BB" chown 999:982 "$a/entorno_efectivo.tsv" "$a/privilegios.tsv"
    orden=0
    for par in 'LC_ALL=C' 'PATH=/usr/bin:/bin' 'HOME=/nonexistent/o3a-v5-bootstrap' 'XDG_CONFIG_HOME=/nonexistent/o3a-v5-bootstrap' 'GIT_CONFIG_NOSYSTEM=1' 'GIT_TERMINAL_PROMPT=0' 'GIT_ALLOW_PROTOCOL=file' "PWD=$fuente" 'SHLVL=DERIVADO' '_=DERIVADO'; do
        ((orden+=1)); printf 'PREPARACION\tcoordinador\t%s\t%s\t%s\tBOOTSTRAP_BASH\tGO\n' "$orden" "${par%%=*}" "${par#*=}" >>"$a/entorno_permitido.tsv"
    done
    printf 'fase\tproceso\tindice\tvalor\tsha256_bytes\n' >"$a/argv.tsv"
    printf 'carril\tsha_documental\tpadre_documental\tbase_tecnica\tarbol\tblob_documento\ttarget\tdestino\tatestacion\tuid\tgid\tumask\tpropietario\tgrupo\tmodo\ttipo\tnlink\tresultado\n' >"$a/identidad.tsv"
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t999\t982\t077\torquesta\torquesta\t700\tdirectory\t%s\tGO\n' "$c" "$sha_documental" "$PADRE_DOCUMENTAL" "$BASE_TECNICA" "$(como_orquesta_git -C "${target[$c]}" rev-parse 'HEAD^{tree}')" "$blob" "${target[$c]}" "${destino[$c]}" "$a" "$("$BB" stat -c %h "$a")" >>"$a/identidad.tsv"
    idx=0; for arg in /usr/bin/busybox mkdir -m 0700 "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" "${xdg[$c]}/go" "${xdg[$c]}/go/telemetry"; do printf 'PREPARACION\tbusybox_mkdir\t%s\t%s\t%s\n' "$idx" "$arg" "$(printf %s "$arg" | "$BB" sha256sum | "$BB" awk '{print $1}')" >>"$a/argv.tsv"; ((idx+=1)); done
    idx=0; for arg in /usr/bin/busybox env -i LC_ALL=C PATH=/usr/bin:/bin /usr/bin/setpriv "${PRIVILEGIO[@]}" /usr/bin/busybox env -i "HOME=${home[$c]}" "XDG_CONFIG_HOME=${xdg[$c]}" "XDG_CACHE_HOME=${xcache[$c]}" "TMPDIR=${tmp[$c]}" "GOCACHE=${gocache[$c]}" "GOROOT=$goroot" LC_ALL=C "PATH=$goroot/bin:/usr/bin:/bin" GOTOOLCHAIN=local GOENV=off GOPROXY=off GOSUMDB=off /usr/bin/bash --noprofile --norc "$control/coordinador_v3.sh" --go-version "${tmp[$c]}/go-version.pid.tsv" "$a/privilegios.tsv" "$GO"; do printf 'GO_VERSION\tbusybox_env_setpriv_env_bash\t%s\t%s\t%s\n' "$idx" "$arg" "$(printf %s "$arg" | "$BB" sha256sum | "$BB" awk '{print $1}')" >>"$a/argv.tsv"; ((idx+=1)); done
    idx=0; for arg in /usr/bin/busybox env -i "HOME=${home[$c]}" "XDG_CONFIG_HOME=${xdg[$c]}" "XDG_CACHE_HOME=${xcache[$c]}" "TMPDIR=${tmp[$c]}" "GOCACHE=${gocache[$c]}" "GOROOT=$goroot" LC_ALL=C "PATH=$goroot/bin:/usr/bin:/bin" GOTOOLCHAIN=local GOENV=off GOPROXY=off GOSUMDB=off "$GO" version; do printf 'GO_VERSION_EXEC\tbusybox_env_go\t%s\t%s\t%s\n' "$idx" "$arg" "$(printf %s "$arg" | "$BB" sha256sum | "$BB" awk '{print $1}')" >>"$a/argv.tsv"; ((idx+=1)); done
    printf 'fase\txdg_config_home\truta_mode\tbytes_mode\tsha256_mode\tlocal\tupload\tdebug\tcontadores\totras_entradas\tresultado\n' >"$a/telemetria_pre.tsv"
    telemetria "$a/telemetria_pre.tsv" ANTES_GO_VERSION "${xdg[$c]}"
    log PREFLIGHT CARRIL_PREPARADO "$c"
done
for c in "${carriles[@]}"; do
    a=${atestacion[$c]}; gov_pid="${tmp[$c]}/go-version.pid.tsv"; gov_err="${tmp[$c]}/go-version.stderr"
    set +e
    gov_out=$("$BB" env -i LC_ALL=C PATH=/usr/bin:/bin "$SETPRIV" "${PRIVILEGIO[@]}" "$BB" env -i "HOME=${home[$c]}" "XDG_CONFIG_HOME=${xdg[$c]}" "XDG_CACHE_HOME=${xcache[$c]}" "TMPDIR=${tmp[$c]}" "GOCACHE=${gocache[$c]}" "GOROOT=$goroot" LC_ALL=C "PATH=$goroot/bin:/usr/bin:/bin" GOTOOLCHAIN=local GOENV=off GOPROXY=off GOSUMDB=off "$BASH" --noprofile --norc "$control/coordinador_v3.sh" --go-version "$gov_pid" "$a/privilegios.tsv" "$GO" 2>"$gov_err" 9>&- {efd}>&- {le}>&- {lr}>&- {lt}>&- {lv}>&-)
    gov_status=$?
    set -e
    read -r gov_real_pid gov_ppid gov_start <"$gov_pid"
    [[ $gov_status -eq 0 && $gov_out == 'go version go1.26.6 linux/amd64' && ! -s $gov_err ]]
    printf '%s\tgo\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\tGO\n' "$c" "$GO" "$(sha "$GO")" 'go version go1.26.6 linux/amd64' "$gov_out" "$(printf '%s\n' "$gov_out" | "$BB" sha256sum | "$BB" awk '{print $1}')" "$(sha "$gov_err")" "$gov_status" "$gov_real_pid" "$gov_start" >>"$control/toolchain.tsv"
    telemetria "$a/telemetria_pre.tsv" DESPUES_GO_VERSION "${xdg[$c]}"
    telemetria "$a/telemetria_pre.tsv" ANTES_CONDUCCION "${xdg[$c]}"
    registrar_entorno "$a/entorno_efectivo.tsv" GO_VERSION go "$gov_real_pid" "$gov_ppid" CONTRATO <<EOF
HOME=${home[$c]}
XDG_CONFIG_HOME=${xdg[$c]}
XDG_CACHE_HOME=${xcache[$c]}
TMPDIR=${tmp[$c]}
GOCACHE=${gocache[$c]}
GOROOT=$goroot
LC_ALL=C
PATH=$goroot/bin:/usr/bin:/bin
GOTOOLCHAIN=local
GOENV=off
GOPROXY=off
GOSUMDB=off
EOF
    log PREFLIGHT GO_VERSION "$c:$gov_real_pid:$gov_start"
    base_permitido=("HOME=${home[$c]}" "XDG_CONFIG_HOME=${xdg[$c]}" "XDG_CACHE_HOME=${xcache[$c]}" "TMPDIR=${tmp[$c]}" "GOCACHE=${gocache[$c]}" "GOROOT=$goroot" 'LC_ALL=C' "PATH=$goroot/bin:/usr/bin:/bin" 'GOTOOLCHAIN=local' 'GOENV=off' 'GOPROXY=off' 'GOSUMDB=off')
    for fase_proceso in 'GO_VERSION|go' 'INTENTO_SUPERVISOR|setsid' 'PUENTE|coordinador_hijo' 'CONDUCCION|conductor' 'BLOQUE_NORMAL|bloque' 'BLOQUE_RACE|bloque' 'GO_BUILD_NORMAL|go' 'GO_BUILD_RACE|go' 'C20|go' 'C20_BUILD|go'; do
        IFS='|' read -r fase proceso <<<"$fase_proceso"; orden=0
        for par in "${base_permitido[@]}"; do ((orden+=1)); printf '%s\t%s\t%s\t%s\t%s\tCONTRATO\tGO\n' "$fase" "$proceso" "$orden" "${par%%=*}" "${par#*=}" >>"$a/entorno_permitido.tsv"; done
    done
    printf 'PUENTE\tcoordinador_hijo\t13\tPWD\tFUENTE\tBASH\tGO\nPUENTE\tcoordinador_hijo\t14\tSHLVL\tDERIVADO\tBASH\tGO\nPUENTE\tcoordinador_hijo\t15\t_\tDERIVADO\tBASH\tGO\n' >>"$a/entorno_permitido.tsv"
    printf 'CONDUCCION\tconductor\t13\tPWD\tTARGET\tBASH\tGO\nCONDUCCION\tconductor\t14\tSHLVL\tDERIVADO\tBASH\tGO\nCONDUCCION\tconductor\t15\t_\tDERIVADO\tBASH\tGO\n' >>"$a/entorno_permitido.tsv"
    printf 'BLOQUE_NORMAL\tbloque\t13\tCND_RUNTIME_TARGET\tTARGET\tCONDUCTOR_VERSIONADO\tGO\nBLOQUE_NORMAL\tbloque\t14\tCND_RACE\t0\tCONDUCTOR_VERSIONADO\tGO\nBLOQUE_NORMAL\tbloque\t15\tCND_CGO_ENABLED\t0\tCONDUCTOR_VERSIONADO\tGO\n' >>"$a/entorno_permitido.tsv"
    printf 'BLOQUE_RACE\tbloque\t13\tCND_RUNTIME_TARGET\tTARGET\tCONDUCTOR_VERSIONADO\tGO\nBLOQUE_RACE\tbloque\t14\tCND_RACE\t1\tCONDUCTOR_VERSIONADO\tGO\nBLOQUE_RACE\tbloque\t15\tCND_CGO_ENABLED\t1\tCONDUCTOR_VERSIONADO\tGO\n' >>"$a/entorno_permitido.tsv"
    for fase_modo in 'GO_BUILD_NORMAL|0' 'GO_BUILD_RACE|1' 'C20|0_O_1' 'C20_BUILD|0_O_1'; do
        IFS='|' read -r fase modo_cnd <<<"$fase_modo"
        printf '%s\tproceso_versionado\t13\tCND_RUNTIME_TARGET\tTARGET\tCONDUCTOR_VERSIONADO\tGO\n%s\tproceso_versionado\t14\tCND_RACE\t%s\tCONDUCTOR_VERSIONADO\tGO\n%s\tproceso_versionado\t15\tCND_CGO_ENABLED\t%s\tCONDUCTOR_VERSIONADO\tGO\n' "$fase" "$fase" "$modo_cnd" "$fase" "$modo_cnd" >>"$a/entorno_permitido.tsv"
    done
    printf 'GO_BUILD_NORMAL\tgo\t16\tCGO_ENABLED\t0\tBLOQUE_VERSIONADO\tGO\nGO_BUILD_RACE\tgo\t16\tCGO_ENABLED\t1\tBLOQUE_VERSIONADO\tGO\nC20\tgo\t16\tGOFLAGS\tVACIO\tBLOQUE_VERSIONADO\tGO\nC20_BUILD\tgo\t16\tGOFLAGS\tVACIO\tBLOQUE_VERSIONADO\tGO\nC20_BUILD\tgo\t17\tCGO_ENABLED\t0\tBLOQUE_VERSIONADO\tGO\n' >>"$a/entorno_permitido.tsv"
    for fase_bash in BLOQUE_NORMAL BLOQUE_RACE GO_BUILD_NORMAL GO_BUILD_RACE C20 C20_BUILD; do
        inicio_bash=16
        [[ $fase_bash == GO_BUILD_NORMAL || $fase_bash == GO_BUILD_RACE || $fase_bash == C20 ]] && inicio_bash=17
        [[ $fase_bash == C20_BUILD ]] && inicio_bash=18
        printf '%s\tproceso_versionado\t%s\tPWD\tTARGET_DERIVADO\tBASH\tGO\n%s\tproceso_versionado\t%s\tOLDPWD\tTARGET_DERIVADO\tBASH\tGO\n%s\tproceso_versionado\t%s\tSHLVL\tDERIVADO\tBASH\tGO\n%s\tproceso_versionado\t%s\t_\tDERIVADO\tBASH\tGO\n' "$fase_bash" "$inicio_bash" "$fase_bash" "$((inicio_bash+1))" "$fase_bash" "$((inicio_bash+2))" "$fase_bash" "$((inicio_bash+3))" >>"$a/entorno_permitido.tsv"
    done
done
read -r coord_pid coord_ppid coord_pgid coord_sid _coord_start <<<"$(campos_proceso $$)"
printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t0\t0\t077\t0\t0\t0\t0\tGO\tpreflight_completo\n' "$control" "$inicio_control" "$(utc)" "$coord_pid" "$coord_ppid" "$coord_pgid" "$coord_sid" >>"$control/preflight.tsv"
preflight_files=(contrato.tsv bytes_tecnicos.tsv rutas_reservadas.tsv ausencia_inicial.tsv incidencia_telemetria_pre.tsv clones.tsv toolchain.tsv toolchain_goroot.tsv toolchain_goroot_inventario.tsv binarios_lanzamiento.tsv entornos_permitidos.tsv preflight.tsv procesos_preflight.tsv coordinador_v3.sh ventana.lock)
for f in "${preflight_files[@]}"; do printf '%s  %s\n' "$(sha "$control/$f")" "$f"; done >"$control/SHA256SUMS.preflight"
(cd "$control" && "$BB" sha256sum -c SHA256SUMS.preflight >/dev/null)
preflight_cerrado=1
log PREFLIGHT CERRADO 15/15
global_rojo=0
fase_actual=PROTOCOLO
orden=0
for c in "${carriles[@]}"; do
    [[ $global_rojo -eq 0 ]] || exit 111
    fase_actual=PROTOCOLO
    ((orden+=1))
    carril_activo=$c; job=0; supervisor_pid=0; supervisor_start=''; supervisor_lanzado=0; supervisor_recolectado=1; supervisor_detenido=0; intento_supervisor_pendiente=0
    pid=0; inicio=''; estado_sesion=NO_OBSERVADA
    monitor=0; monitor_start=''; monitor_pid_atestado=0; monitor_start_atestado=''; monitor_lanzado=0; monitor_recolectado=1; salida_job=125; salida_monitor=125
    a=${atestacion[$c]}
    conductor=${target[$c]}/tools/o3a_v5_conductor/conductor.sh
    argv_sha=$(printf '%s\0' /usr/bin/bash "$conductor" "${target[$c]}" "${destino[$c]}" | "$BB" sha256sum | "$BB" awk '{print $1}')
    supervisor_argv_sha=$(printf '%s\0' "$SETSID" --fork --wait "$BASH" --noprofile --norc "$control/coordinador_v3.sh" --hijo "$c" "${target[$c]}" "${destino[$c]}" "$a" "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" "$sha_documental" "$goroot" | "$BB" sha256sum | "$BB" awk '{print $1}')
    hijo_argv_sha=$(printf '%s\0' "$BASH" --noprofile --norc "$control/coordinador_v3.sh" --hijo "$c" "${target[$c]}" "${destino[$c]}" "$a" "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" "$sha_documental" "$goroot" | "$BB" sha256sum | "$BB" awk '{print $1}')
    idx=0
    for arg in /usr/bin/bash "$conductor" "${target[$c]}" "${destino[$c]}"; do
        printf 'CONDUCCION\tconductor\t%s\t%s\t%s\n' "$idx" "$arg" "$(printf %s "$arg" | "$BB" sha256sum | "$BB" awk '{print $1}')" >>"$a/argv.tsv"
        ((idx+=1))
    done
    idx=0
    for arg in /usr/bin/busybox env -i "HOME=${home[$c]}" "XDG_CONFIG_HOME=${xdg[$c]}" "XDG_CACHE_HOME=${xcache[$c]}" "TMPDIR=${tmp[$c]}" "GOCACHE=${gocache[$c]}" "GOROOT=$goroot" LC_ALL=C "PATH=$goroot/bin:/usr/bin:/bin" GOTOOLCHAIN=local GOENV=off GOPROXY=off GOSUMDB=off /usr/bin/bash "$conductor" "${target[$c]}" "${destino[$c]}"; do
        printf 'CONDUCCION_EXEC\tbusybox_env_bash\t%s\t%s\t%s\n' "$idx" "$arg" "$(printf %s "$arg" | "$BB" sha256sum | "$BB" awk '{print $1}')" >>"$a/argv.tsv"
        ((idx+=1))
    done
    idx=0
    for arg in /usr/bin/busybox env -i LC_ALL=C PATH=/usr/bin:/bin /usr/bin/setpriv "${PRIVILEGIO[@]}" /usr/bin/busybox env -i "HOME=${home[$c]}" "XDG_CONFIG_HOME=${xdg[$c]}" "XDG_CACHE_HOME=${xcache[$c]}" "TMPDIR=${tmp[$c]}" "GOCACHE=${gocache[$c]}" "GOROOT=$goroot" LC_ALL=C "PATH=$goroot/bin:/usr/bin:/bin" GOTOOLCHAIN=local GOENV=off GOPROXY=off GOSUMDB=off /usr/bin/setsid --fork --wait /usr/bin/bash --noprofile --norc "$control/coordinador_v3.sh" --hijo "$c" "${target[$c]}" "${destino[$c]}" "$a" "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" "$sha_documental" "$goroot"; do
        printf 'INTENTO\tbusybox_env_setpriv_env_setsid_wait_bash\t%s\t%s\t%s\n' "$idx" "$arg" "$(printf %s "$arg" | "$BB" sha256sum | "$BB" awk '{print $1}')" >>"$a/argv.tsv"
        ((idx+=1))
    done
    estado "$a/01_RESERVADO.tsv" 1 RESERVADO "$c" PENDIENTE PENDIENTE PENDIENTE PENDIENTE PENDIENTE "$argv_sha" PENDIENTE INICIO GO
    ((reservas_total+=1))
    "$BB" chown 999:982 "$a/01_RESERVADO.tsv"
    log "$c" RESERVADO "$argv_sha"
    intento_supervisor_pendiente=1
    set +e
    "$BB" env -i LC_ALL=C PATH=/usr/bin:/bin "$SETPRIV" "${PRIVILEGIO[@]}" "$BB" env -i "HOME=${home[$c]}" "XDG_CONFIG_HOME=${xdg[$c]}" "XDG_CACHE_HOME=${xcache[$c]}" "TMPDIR=${tmp[$c]}" "GOCACHE=${gocache[$c]}" "GOROOT=$goroot" LC_ALL=C "PATH=$goroot/bin:/usr/bin:/bin" GOTOOLCHAIN=local GOENV=off GOPROXY=off GOSUMDB=off "$SETSID" --fork --wait "$BASH" --noprofile --norc "$control/coordinador_v3.sh" --hijo "$c" "${target[$c]}" "${destino[$c]}" "$a" "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" "$sha_documental" "$goroot" >"$a/captura_stdout.bin" 2>"$a/captura_stderr.bin" 9>&- {efd}>&- {le}>&- {lr}>&- {lt}>&- {lv}>&- &
    launch_rc=$?; job=$!; set -e
    if [[ $launch_rc -ne 0 || ! $job =~ ^[1-9][0-9]*$ ]]; then
        fallar_carril LANZAMIENTO_SUPERVISOR_NO_CONFIRMADO
    fi
    supervisor_pid=$job; supervisor_lanzado=1; supervisor_recolectado=0; supervisor_detenido=0; intento_supervisor_pendiente=0
    for ((i=0; i<100; i++)); do
        if asegurar_supervisor; then estado_identidad=0; else estado_identidad=$?; fi
        case $estado_identidad in
            0) break ;;
            1) "$BB" sleep 0.01 ;;
            2) fallar_carril IDENTIDAD_AMBIGUA_SUPERVISOR ;;
        esac
    done
    [[ $supervisor_start =~ ^[0-9]+$ ]] || fallar_carril SUPERVISOR_NO_OBSERVADO
    identidad_pid "$supervisor_pid" "$supervisor_start" || fallar_carril IDENTIDAD_AMBIGUA_SUPERVISOR
    log "$c" SETSID_SUPERVISOR "$supervisor_pid:$supervisor_start"
    for ((i=0; i<300; i++)); do
        [[ -f $a/02_HIJO_NACIDO.tsv ]] && break
        if identidad_pid "$job" "$supervisor_start"; then estado_identidad=0; else estado_identidad=$?; fi
        case $estado_identidad in 0) ;; 1) break ;; 2) fallar_carril PID_REUTILIZADO_SUPERVISOR ;; esac
        "$BB" sleep 1
    done
    [[ -f $a/02_HIJO_NACIDO.tsv ]] || fallar_carril TIMEOUT_HIJO_NACIDO
    IFS=$'\t' read -r supervisor_pid_real supervisor_ppid supervisor_pgid supervisor_sid supervisor_start_real <<<"$(campos_proceso "$supervisor_pid")"
    [[ $supervisor_pid_real == "$supervisor_pid" && $supervisor_ppid == "$coord_pid" && $supervisor_pgid == "$coord_pgid" && $supervisor_sid == "$coord_sid" && $supervisor_start_real == "$supervisor_start" ]]
    registrar_privilegio "$a/privilegios.tsv" INTENTO_SUPERVISOR setsid "$supervisor_pid"
    [[ $(sha "/proc/$supervisor_pid/cmdline") == "$supervisor_argv_sha" ]]
    "$BB" tr '\0' '\n' <"/proc/$supervisor_pid/environ" | registrar_entorno "$a/entorno_efectivo.tsv" INTENTO_SUPERVISOR setsid "$supervisor_pid" "$supervisor_ppid" OBSERVADO_PROC
    if descubrir_sesion; then estado_identidad=0; else estado_identidad=$?; fi
    [[ $estado_identidad -eq 0 && $estado_sesion == VIVO_IDENTICO ]] || fallar_carril IDENTIDAD_AMBIGUA_HIJO
    ((hijos_total+=1))
    IFS=$'\t' read -r pid_real ppid pgid sid inicio <<<"$(campos_proceso "$pid")"
    [[ $pid == "$pid_real" && $pid == "$pgid" && $pid == "$sid" && $ppid == "$supervisor_pid" ]]
    "$BB" tr '\0' '\n' <"/proc/$pid/environ" | registrar_entorno "$a/entorno_efectivo.tsv" PUENTE coordinador_hijo "$pid" "$ppid" OBSERVADO_PROC
    foto "$a/procesos_antes.tsv" HIJO_BLOQUEADO
    monitorizar "$pid" "$inicio" "$a/procesos_durante.tsv" "$a/entorno_efectivo.tsv" & monitor=$!; monitor_start=''; monitor_pid_atestado=$monitor; monitor_lanzado=1; monitor_recolectado=0
    asegurar_monitor
    previo=$(sha "$a/02_HIJO_NACIDO.tsv")
    estado "$a/03_LIBERADO.tsv" 3 LIBERADO "$c" "$pid" "$ppid" "$pgid" "$sid" "$inicio" "$argv_sha" PENDIENTE "$previo" GO
    log "$c" HIJO_LIBERADO "$pid:$inicio:$pgid:$sid"
    exec_ok=0
    for ((i=0; i<300; i++)); do
        if identidad_pid "$pid" "$inicio"; then estado_identidad=0; else estado_identidad=$?; fi
        case $estado_identidad in
            0) [[ -r /proc/$pid/cmdline && $(sha "/proc/$pid/cmdline") == "$argv_sha" ]] && { exec_ok=1; break; } ;;
            1) break ;;
            2) fallar_carril PID_REUTILIZADO_HIJO ;;
        esac
        if identidad_pid "$job" "$supervisor_start"; then estado_identidad=0; else estado_identidad=$?; fi
        case $estado_identidad in 0) ;; 1) break ;; 2) fallar_carril PID_REUTILIZADO_SUPERVISOR ;; esac
        "$BB" sleep 1
    done
    if [[ $exec_ok -eq 1 ]]; then
        IFS=$'\t' read -r pid_exec ppid_exec pgid_exec sid_exec inicio_exec <<<"$(campos_proceso "$pid")"
        [[ $pid_exec == "$pid" && $inicio_exec == "$inicio" && $pgid_exec == "$pid" && $sid_exec == "$pid" ]]
        previo=$(sha "$a/03_LIBERADO.tsv")
        estado "$a/04_EXEC_CONFIRMADO.tsv" 4 EXEC_CONFIRMADO "$c" "$pid" "$ppid_exec" "$pgid_exec" "$sid_exec" "$inicio_exec" "$argv_sha" PENDIENTE "$previo" GO
        ((exec_total+=1))
        log "$c" EXEC_CONFIRMADO "$pid:$inicio_exec:$argv_sha"
    else
        fallar_carril EXEC_NO_CONFIRMADO
    fi
    fase_actual=EJECUCION
    esperar_supervisor_acotado 72000 || fallar_carril TIMEOUT_SUPERVISOR
    salida=$salida_job
    "$BB" chown 999:982 "$a/captura_stdout.bin" "$a/captura_stderr.bin"
    fase_actual=PROTOCOLO
    esperar_monitor_acotado 50 || fallar_carril MONITOR_NO_RECOLECTADO
    [[ $salida_monitor -eq 0 ]] || fallar_carril MONITOR_SALIDA_NO_CERO
    if sesion_viva; then estado_identidad=0; else estado_identidad=$?; fi
    [[ $estado_identidad -eq 1 && $estado_sesion == AUSENTE ]] || fallar_carril SESION_NO_RECOLECTADA
    fase_actual=EJECUCION
    env_fases_ok=1
    for fase_requerida in PREPARACION GO_VERSION INTENTO_SUPERVISOR PUENTE CONDUCCION BLOQUE_NORMAL BLOQUE_RACE GO_BUILD_NORMAL GO_BUILD_RACE C20 C20_BUILD; do
        "$BB" awk -F '\t' -v f="$fase_requerida" 'NR>1 && $1==f{ok=1}END{exit !ok}' "$a/entorno_efectivo.tsv" || env_fases_ok=0
    done
    "$BB" awk -F '\t' -v home="${home[$c]}" -v xdg="${xdg[$c]}" -v xcache="${xcache[$c]}" -v tmp="${tmp[$c]}" -v gocache="${gocache[$c]}" -v goroot="$goroot" -v source="$fuente" -v target="${target[$c]}" '
        NR==1{if($0!="fase\tproceso\tpid\tppid\tnombre\tvalor\torigen\tresultado")exit 1;next}
        function base(n,v){return (n=="HOME"&&v==home)||(n=="XDG_CONFIG_HOME"&&v==xdg)||(n=="XDG_CACHE_HOME"&&v==xcache)||(n=="TMPDIR"&&v==tmp)||(n=="GOCACHE"&&v==gocache)||(n=="GOROOT"&&v==goroot)||(n=="LC_ALL"&&v=="C")||(n=="PATH"&&v==goroot "/bin:/usr/bin:/bin")||(n=="GOTOOLCHAIN"&&v=="local")||(n=="GOENV"&&v=="off")||(n=="GOPROXY"&&v=="off")||(n=="GOSUMDB"&&v=="off")}
        function bashv(n,v,where,old){return (n=="PWD"&&((where=="source"&&v==source)||(where=="target"&&(v==target||index(v,target "/")==1))))||(old&&n=="OLDPWD"&&(v==source||v==target||index(v,target "/")==1))||(n=="SHLVL"&&v~/^[0-9]+$/)||(n=="_"&&v!="")}
        function cnd(n,v,f){return (n=="CND_RUNTIME_TARGET"&&v==target)||(n=="CND_RACE"&&((f~/_NORMAL$/&&v=="0")||(f~/_RACE$/&&v=="1")||(f~/^C20/&&(v=="0"||v=="1"))))||(n=="CND_CGO_ENABLED"&&((f~/_NORMAL$/&&v=="0")||(f~/_RACE$/&&v=="1")||(f~/^C20/&&(v=="0"||v=="1"))))}
        function exige(k,n){if(!((k SUBSEP n) in visto))exit 1}
        function cardinalidad(f){if(f=="PREPARACION")return 10;if(f=="GO_VERSION"||f=="INTENTO_SUPERVISOR")return 12;if(f=="PUENTE"||f=="CONDUCCION")return 15;if(f=="BLOQUE_NORMAL"||f=="BLOQUE_RACE")return 19;if(f=="GO_BUILD_NORMAL"||f=="GO_BUILD_RACE"||f=="C20")return 20;if(f=="C20_BUILD")return 21;return -1}
        {if(NF!=8||$2==""||$3!~/^[1-9][0-9]*$/||$4!~/^[0-9]+$/||$5==""||$7==""||$8!="GO")exit 1;k=$1 SUBSEP $3;fases[k]=$1;filas[k]++;kn=k SUBSEP $5;if(kn in visto)duplicado[k]=1;else{visto[kn]=1;distintos[k]++}}
        $1=="PREPARACION"{if(!(($5=="LC_ALL"&&$6=="C")||($5=="PATH"&&$6=="/usr/bin:/bin")||($5=="HOME"&&$6=="/nonexistent/o3a-v5-bootstrap")||($5=="XDG_CONFIG_HOME"&&$6=="/nonexistent/o3a-v5-bootstrap")||($5=="GIT_CONFIG_NOSYSTEM"&&$6=="1")||($5=="GIT_TERMINAL_PROMPT"&&$6=="0")||($5=="GIT_ALLOW_PROTOCOL"&&$6=="file")||bashv($5,$6,"source",0)))exit 1;next}
        $1=="GO_VERSION"{if(!base($5,$6))exit 1;next}
        $1=="INTENTO_SUPERVISOR"{if(!base($5,$6))exit 1;next}
        $1=="PUENTE"{if(!(base($5,$6)||bashv($5,$6,"source",0)))exit 1;next}
        $1=="CONDUCCION"{if(!(base($5,$6)||bashv($5,$6,"target",0)))exit 1;next}
        $1=="BLOQUE_NORMAL"||$1=="BLOQUE_RACE"{if(!(base($5,$6)||bashv($5,$6,"target",1)||cnd($5,$6,$1)))exit 1;next}
        $1=="GO_BUILD_NORMAL"||$1=="GO_BUILD_RACE"{if(!(base($5,$6)||bashv($5,$6,"target",1)||cnd($5,$6,$1)||($5=="CGO_ENABLED"&&$6==($1=="GO_BUILD_NORMAL"?"0":"1"))))exit 1;next}
        $1=="C20"||$1=="C20_BUILD"{if(!(base($5,$6)||bashv($5,$6,"target",1)||cnd($5,$6,$1)||($5=="CGO_ENABLED"&&$6=="0")||($5=="GOFLAGS"&&$6=="")))exit 1;next}
        {exit 1}
        END{
            for(k in fases){
                f=fases[k]
                esperado=cardinalidad(f);if(esperado<0||duplicado[k]||filas[k]!=esperado||distintos[k]!=esperado)exit 1
                if(f=="PREPARACION"){split("LC_ALL PATH HOME XDG_CONFIG_HOME GIT_CONFIG_NOSYSTEM GIT_TERMINAL_PROMPT GIT_ALLOW_PROTOCOL PWD SHLVL _",r," ");for(i in r)exige(k,r[i]);continue}
                split("HOME XDG_CONFIG_HOME XDG_CACHE_HOME TMPDIR GOCACHE GOROOT LC_ALL PATH GOTOOLCHAIN GOENV GOPROXY GOSUMDB",r," ");for(i in r)exige(k,r[i])
                if(f=="PUENTE"||f=="CONDUCCION"){exige(k,"PWD");exige(k,"SHLVL");exige(k,"_")}
                if(f~/^(BLOQUE_|GO_BUILD_|C20)/){exige(k,"PWD");exige(k,"OLDPWD");exige(k,"SHLVL");exige(k,"_");exige(k,"CND_RUNTIME_TARGET");exige(k,"CND_RACE");exige(k,"CND_CGO_ENABLED")}
                if(f~/^GO_BUILD_/||f=="C20_BUILD")exige(k,"CGO_ENABLED")
                if(f=="C20"||f=="C20_BUILD")exige(k,"GOFLAGS")
            }
        }
    ' "$a/entorno_efectivo.tsv" || env_fases_ok=0
    privilegios_ok=0
    "$BB" awk -F '\t' 'NR==1{if($0!="fase\tproceso\tpid\tppid\tuid\tgid\tgrupos_suplementarios\tcap_inh\tcap_prm\tcap_eff\tcap_bnd\tcap_amb\tno_new_privs\tresultado")bad=1;next}{filas++;if(NF!=14||$3!~/^[1-9][0-9]*$/||$4!~/^[0-9]+$/||$5!="999"||$6!="982"||$7!="-"||$8!="0000000000000000"||$9!="0000000000000000"||$10!="0000000000000000"||$11!="0000000000000000"||$12!="0000000000000000"||$13!="1"||$14!="GO")bad=1;par=$1 SUBSEP $2;if(visto[par]++)bad=1;if($1=="GO_VERSION"&&$2=="go")g++;else if($1=="INTENTO_SUPERVISOR"&&$2=="setsid")s++;else if($1=="CONDUCCION"&&$2=="conductor")c++;else bad=1}END{exit !(filas==3&&g==1&&s==1&&c==1&&!bad)}' "$a/privilegios.tsv" && privilegios_ok=1
    printf '%s\n' "$salida" >"$a/captura_estado.txt"
    "$BB" chown 999:982 "$a/captura_estado.txt"
    previo=$([[ -f $a/04_EXEC_CONFIRMADO.tsv ]] && sha "$a/04_EXEC_CONFIRMADO.tsv" || printf HUECO)
    resultado=NO-GO; [[ $exec_ok -eq 1 && $salida -eq 0 ]] && resultado=GO
    estado "$a/05_FINALIZADO.tsv" 5 FINALIZADO "$c" "$pid" "$ppid" "$pgid" "$sid" "$inicio" "$argv_sha" "$salida" "$previo" "$resultado"
    ((finalizaciones_total+=1))
    log "$c" FINALIZADO "$pid:$salida:$resultado"
    fase_actual=EVIDENCIA
    foto "$a/procesos_despues.tsv" FINALIZADO
    resultado=NO-GO; [[ $exec_ok -eq 1 && $salida -eq 0 ]] && resultado=GO
    printf 'carril\treservas\thijos_nacidos\tliberaciones\texec_confirmados\tfinalizaciones\tpid_supervisor_setsid\tstarttime_supervisor_setsid\targv_sha256_supervisor_setsid\tpid_monitor\tstarttime_monitor\testado_monitor\tmonitor_recolectado\tpid\tppid\tpgid\tsid\tstarttime_ticks\tcwd\testado_conductor\tresultado\n%s\t1\t1\t1\t%s\t1\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$c" "$exec_ok" "$supervisor_pid" "$supervisor_start" "$supervisor_argv_sha" "$monitor_pid_atestado" "$monitor_start_atestado" "$salida_monitor" "$monitor_recolectado" "$pid" "$ppid" "$pgid" "$sid" "$inicio" "${target[$c]}" "$salida" "$resultado" >"$a/invocacion.tsv"
    printf 'nombre\truta\tbytes\tsha256\tpropietario\tgrupo\tmodo\ttipo\tnlink\tresultado\n' >"$a/capturas.tsv"
    for cap in captura_stdout.bin captura_stderr.bin captura_estado.txt; do
        cap_resultado=NO-GO; [[ -f $a/$cap && ! -L $a/$cap ]] && cap_resultado=GO
        IFS='|' read -r cap_u cap_g cap_m cap_t cap_l < <("$BB" stat -c '%U|%G|%a|%F|%h' "$a/$cap")
        [[ $cap_u:$cap_g:$cap_m:$cap_l == 'orquesta:orquesta:600:1' && ( $cap_t == 'regular file' || $cap_t == 'regular empty file' ) ]] || cap_resultado=NO-GO
        printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$cap" "$a/$cap" "$("$BB" stat -c %s "$a/$cap")" "$(sha "$a/$cap")" "$cap_u" "$cap_g" "$cap_m" "$cap_t" "$cap_l" "$cap_resultado" >>"$a/capturas.tsv"
    done
    stdout_sha=$(sha "$a/captura_stdout.bin"); stderr_sha=$(sha "$a/captura_stderr.bin"); estado_sha=$(sha "$a/captura_estado.txt")
    capturas_ok=0
    [[ $stdout_sha == 13763c7184f311500fa5b80d76abcca71e22d8c7928b9dc5ca6b8840a50a558d && $stderr_sha == e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 && $estado_sha == 9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa ]] && capturas_ok=1
    fase_actual=EJECUCION
    [[ $salida -eq 0 && $capturas_ok -eq 1 ]] || fallar_carril EJECUCION_O_CAPTURAS_NO_GO
    fase_actual=EVIDENCIA
    printf 'fase\txdg_config_home\truta_mode\tbytes_mode\tsha256_mode\tlocal\tupload\tdebug\tcontadores\totras_entradas\tresultado\n' >"$a/telemetria_post.tsv"
    telemetria "$a/telemetria_post.tsv" DESPUES_CONDUCCION "${xdg[$c]}"
    "$BB" cp "${xdg[$c]}/go/telemetry/mode" "$a/telemetria_mode"
    total_dest=$("$BB" find "${destino[$c]}" -mindepth 1 -maxdepth 1 -print 2>/dev/null | "$BB" wc -l)
    regulares_dest=$("$BB" find "${destino[$c]}" -mindepth 1 -maxdepth 1 -type f -print 2>/dev/null | "$BB" wc -l)
    enlaces_dest=0
    for f in "${destino[$c]}"/*; do [[ -f $f && ! -L $f && $("$BB" stat -c %h "$f") -eq 1 ]] || ((enlaces_dest+=1)); done
    sum_ok=0
    if [[ $total_dest -eq 22 && $regulares_dest -eq 22 && $enlaces_dest -eq 0 && $("$BB" wc -l <"${destino[$c]}/SHA256SUMS") -eq 21 ]] &&
        "$BB" cmp -s <("$BB" awk '{print $2}' "${destino[$c]}/SHA256SUMS" | "$BB" sort) <(for f in "${destino[$c]}"/*; do [[ ${f##*/} == SHA256SUMS ]] || printf '%s\n' "${f##*/}"; done | "$BB" sort) &&
        (cd "${destino[$c]}" && "$BB" sha256sum -c SHA256SUMS >/dev/null); then sum_ok=21; fi
    bloques=0; casos=0; c21_normal=0; c21_race=0; fd_inicio=-1; fd_fin=-2; residuos_bytes=-1; manifiesto_ok=0; c21_ok=0; resumen_ok=0
    [[ -f ${destino[$c]}/manifiesto.tsv ]] && bloques=$(($("$BB" wc -l <"${destino[$c]}/manifiesto.tsv")-1))
    [[ -f ${destino[$c]}/casos.ndjson ]] && casos=$("$BB" wc -l <"${destino[$c]}/casos.ndjson")
    [[ -f ${destino[$c]}/c15_c21_normal_c21_indices.tsv ]] && c21_normal=$(($("$BB" wc -l <"${destino[$c]}/c15_c21_normal_c21_indices.tsv")-1))
    [[ -f ${destino[$c]}/c15_c21_race_c21_indices.tsv ]] && c21_race=$(($("$BB" wc -l <"${destino[$c]}/c15_c21_race_c21_indices.tsv")-1))
    if [[ -f ${destino[$c]}/resumen.txt ]]; then
        fd_inicio=$("$BB" awk -F= '$1=="fd_conductor_inicio"{print $2}' "${destino[$c]}/resumen.txt")
        fd_fin=$("$BB" awk -F= '$1=="fd_conductor_fin"{print $2}' "${destino[$c]}/resumen.txt")
    fi
    [[ -f ${destino[$c]}/residuos.txt ]] && residuos_bytes=$("$BB" stat -c %s "${destino[$c]}/residuos.txt")
    if [[ -f ${destino[$c]}/manifiesto.tsv ]] && ! "$BB" grep -Eq '(^|[[:space:]])(NO-GO|SKIP)([[:space:]]|$)' "${destino[$c]}/manifiesto.tsv"; then
        manifiesto_ok=1
        for spec in "${manifiesto_esperado[@]}"; do
            bloque=${spec%%|*}; huella=${spec##*|}
            for modo in normal race; do
                "$BB" awk -F '\t' -v b="$bloque" -v m="$modo" -v h="$huella" 'NR>1&&$1==b&&$2==m{n++;if($3!=h||$6!=0||$7<1||$8!="GO")bad=1}END{exit !(n==1&&!bad)}' "${destino[$c]}/manifiesto.tsv" || manifiesto_ok=0
            done
        done
    fi
    if [[ $c21_normal -eq 100 && $c21_race -eq 100 ]] &&
        "$BB" awk -F '\t' 'FNR==1{archivos++;if($0!="indice\tcaso\tesperado\testado\tstdout\tstderr\tresultado")bad=1;next}FNR>1{n[FILENAME]++;if($1!=n[FILENAME]||$2!="TUPLA_C"||$3!=0||$4!=0||$5!=0||$6!=0||$7!="GO")bad=1}END{vistos=0;for(f in n){vistos++;if(n[f]!=100)bad=1}exit !(archivos==2&&vistos==2&&!bad)}' "${destino[$c]}/c15_c21_normal_c21_indices.tsv" "${destino[$c]}/c15_c21_race_c21_indices.tsv"; then c21_ok=1; fi
    casos_ok=1
    for f in "${casos_deterministas[@]}"; do
        "$BB" awk -F '\t' 'NR==1{for(i=1;i<=NF;i++){if($i=="esperado")e=i;if($i=="estado")s=i;if($i=="stdout")o=i;if($i=="stderr")r=i;if($i=="resultado")g=i}next}{if(!g||$g!="GO"||(e&&$e!=$s)||(o&&$o!=0)||(r&&$r!=0))exit 1}' "${destino[$c]}/$f" || casos_ok=0
    done
    if [[ -f ${destino[$c]}/resumen.txt ]] && "$BB" grep -qx 'resultado=GO' "${destino[$c]}/resumen.txt" && "$BB" grep -qx 'go_version=go version go1.26.6 linux/amd64' "${destino[$c]}/resumen.txt" && "$BB" grep -qx "head=$sha_documental" "${destino[$c]}/resumen.txt" && "$BB" grep -qx 'bloques=14' "${destino[$c]}/resumen.txt" && "$BB" grep -qx 'casos_registrados=74' "${destino[$c]}/resumen.txt"; then resumen_ok=1; fi
    concurrencia_ok=0
    [[ $("$BB" awk -F '\t' 'FNR>1 && ($13=="conductor_o3a_v5"||$13=="conductor_fuera_sesion"){n++}END{print n+0}' "$a/procesos_antes.tsv" "$a/procesos_despues.tsv") -eq 0 && $("$BB" awk -F '\t' 'FNR>1 && $13=="conductor_fuera_sesion"{n++}END{print n+0}' "$a/procesos_durante.tsv") -eq 0 ]] && concurrencia_ok=1
    evidencia_resultado=NO-GO
    [[ $salida -eq 0 && $exec_ok -eq 1 && $env_fases_ok -eq 1 && $privilegios_ok -eq 1 && $concurrencia_ok -eq 1 && $capturas_ok -eq 1 && $total_dest -eq 22 && $sum_ok -eq 21 && $bloques -eq 14 && $casos -eq 74 && $casos_ok -eq 1 && $manifiesto_ok -eq 1 && $c21_ok -eq 1 && $resumen_ok -eq 1 && $fd_inicio == "$fd_fin" && $residuos_bytes -eq 0 ]] && evidencia_resultado=GO
    printf 'destino\tficheros_totales\tentradas_sha256sums\tsha256sum_validas\tbloques\tcasos\tc21_normal\tc21_race\tfd_inicio\tfd_fin\tresiduos_bytes\tsha256_sha256sums\tresultado\n%s\t%s\t21\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "${destino[$c]}" "$total_dest" "$sum_ok" "$bloques" "$casos" "$c21_normal" "$c21_race" "$fd_inicio" "$fd_fin" "$residuos_bytes" "$([[ -f ${destino[$c]}/SHA256SUMS ]] && sha "${destino[$c]}/SHA256SUMS" || printf AUSENTE)" "$evidencia_resultado" >"$a/evidencia_conductor.tsv"
    [[ $evidencia_resultado == GO ]] || fallar_carril EVIDENCIA_CONDUCTOR_NO_GO
    fase_actual=POSTCONDICION
    target_limpio=NO-GO; [[ -z $(como_orquesta_git -C "${target[$c]}" status --porcelain=v2 --untracked-files=all) ]] && target_limpio=GO
    procesos_restantes=0; procesos_ambiguos=0
    for p in /proc/[0-9]*; do
        if [[ ! -r $p/stat ]]; then [[ -d $p ]] && procesos_ambiguos=1; continue; fi
        linea_post=$(campos_proceso "${p##*/}") || { [[ -d $p ]] && procesos_ambiguos=1; continue; }
        IFS=$'\t' read -r _ _ _ sid_post _ <<<"$linea_post"
        [[ $sid_post == "$pid" ]] && ((procesos_restantes+=1))
    done
    [[ $target_limpio == GO && $procesos_ambiguos -eq 0 && $procesos_restantes -eq 0 && $supervisor_recolectado -eq 1 && $monitor_recolectado -eq 1 && $salida_monitor -eq 0 && $estado_sesion == AUSENTE ]] || fallar_carril POSTFLIGHT_IDENTIDAD_AMBIGUA
    IFS=$'\t' read -r usuarios_antes usuarios_estado < <(auditar_usuarios_rutas "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" || true)
    IFS=$'\t' read -r fd_antes fd_estado < <(auditar_fd_rutas "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" || true)
    IFS=$'\t' read -r maps_antes maps_estado < <(auditar_maps_rutas "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" || true)
    [[ $usuarios_estado == GO && $fd_estado == GO && $maps_estado == GO && $usuarios_antes -eq 0 && $fd_antes -eq 0 && $maps_antes -eq 0 ]] || fallar_carril POSTFLIGHT_USO_PRIVADAS_AMBIGUO
    limpiar_privadas "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}"
    privadas_ausentes=NO-GO
    [[ ! -e ${home[$c]} && ! -e ${xdg[$c]} && ! -e ${xcache[$c]} && ! -e ${tmp[$c]} && ! -e ${gocache[$c]} ]] && privadas_ausentes=GO
    IFS=$'\t' read -r usuarios_despues usuarios_estado_post < <(auditar_usuarios_rutas "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" || true)
    IFS=$'\t' read -r fd_despues fd_estado_post < <(auditar_fd_rutas "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" || true)
    IFS=$'\t' read -r maps_despues maps_estado_post < <(auditar_maps_rutas "${home[$c]}" "${xdg[$c]}" "${xcache[$c]}" "${tmp[$c]}" "${gocache[$c]}" || true)
    [[ $usuarios_estado_post == GO && $fd_estado_post == GO && $maps_estado_post == GO && $usuarios_despues -eq 0 && $fd_despues -eq 0 && $maps_despues -eq 0 ]] || privadas_ausentes=NO-GO
    printf 'recurso\truta\tantes\tdespues\tprocesos_usuarios\tfd_usuarios\tmaps_usuarios\tresiduos\tretirada_autorizada\tresultado\tdetalle\nprivadas\t%s\tpresentes\tausentes\t%s\t%s\t%s\t%s\tsi\t%s\ttarget_%s,usuarios_post_%s,fd_post_%s,maps_post_%s\n' "$c" "$usuarios_antes" "$fd_antes" "$maps_antes" "$residuos_bytes" "$privadas_ausentes" "$target_limpio" "$usuarios_despues" "$fd_despues" "$maps_despues" >"$a/postflight.tsv"
    estado_carril[$c]=$salida; resultado_carril[$c]=NO-GO
    if [[ $evidencia_resultado == GO && $target_limpio == GO && $procesos_restantes -eq 0 && $privadas_ausentes == GO ]]; then resultado_carril[$c]=GO; ((carriles_go+=1)); else global_rojo=1; fi
    log "$c" POSTFLIGHT "$evidencia_resultado:$target_limpio:$procesos_restantes:$privadas_ausentes"
    [[ $global_rojo -eq 0 ]] || fallar_carril POSTFLIGHT_NO_GO
    carril_activo=NINGUNO; supervisor_pid=0; supervisor_start=''; supervisor_lanzado=0; supervisor_recolectado=1; supervisor_detenido=0
    pid=0; inicio=''; estado_sesion=AUSENTE
    monitor=0; monitor_start=''; monitor_pid_atestado=0; monitor_start_atestado=''; monitor_lanzado=0; monitor_recolectado=1
done
fase_actual=UNANIMIDAD
ref=${destino[productor]}
for c in funcional seguridad; do
    for f in fuentes-verificadas.tsv "${casos_deterministas[@]}"; do
        "$BB" cmp -s "$ref/$f" "${destino[$c]}/$f" || global_rojo=1
    done
    "$BB" cmp -s <("$BB" awk -F '\t' 'BEGIN{OFS="\t"}NR==1{print $1,$2,$3,$6,$7,$8;next}{print $1,$2,$3,$6,$7,$8}' "$ref/manifiesto.tsv") <("$BB" awk -F '\t' 'BEGIN{OFS="\t"}NR==1{print $1,$2,$3,$6,$7,$8;next}{print $1,$2,$3,$6,$7,$8}' "${destino[$c]}/manifiesto.tsv") || global_rojo=1
    "$BB" cmp -s <("$BB" sed -E -e 's/"duracion_bloque_ms":[0-9]+/"duracion_bloque_ms":DURACION/g' -e "s#${target[productor]}#TARGET#g" "$ref/casos.ndjson") <("$BB" sed -E -e 's/"duracion_bloque_ms":[0-9]+/"duracion_bloque_ms":DURACION/g' -e "s#${target[$c]}#TARGET#g" "${destino[$c]}/casos.ndjson") || global_rojo=1
done
log UNANIMIDAD COMPARACION_CRUZADA "$global_rojo"
[[ $global_rojo -eq 0 ]] || fallar_carril COMPARACION_CRUZADA_NO_UNANIME
fase_actual=POSTCONDICION
IFS=$'\t' read -r goroot_usuarios_antes goroot_usuarios_estado < <(auditar_usuarios_rutas "$goroot" || true)
IFS=$'\t' read -r goroot_fd_antes goroot_fd_estado < <(auditar_fd_rutas "$goroot" || true)
IFS=$'\t' read -r goroot_maps_antes goroot_maps_estado < <(auditar_maps_rutas "$goroot" || true)
[[ $goroot_usuarios_estado == GO && $goroot_fd_estado == GO && $goroot_maps_estado == GO && $goroot_usuarios_antes -eq 0 && $goroot_fd_antes -eq 0 && $goroot_maps_antes -eq 0 ]] || fallar_carril GOROOT_EN_USO_O_IDENTIDAD_AMBIGUA
"$BB" rm -rf -- "$goroot"
[[ ! -e $goroot && ! -L $goroot ]] || fallar_carril GOROOT_NO_RETIRADO
IFS=$'\t' read -r goroot_usuarios_despues goroot_usuarios_estado_post < <(auditar_usuarios_rutas "$goroot" || true)
IFS=$'\t' read -r goroot_fd_despues goroot_fd_estado_post < <(auditar_fd_rutas "$goroot" || true)
IFS=$'\t' read -r goroot_maps_despues goroot_maps_estado_post < <(auditar_maps_rutas "$goroot" || true)
[[ $goroot_usuarios_estado_post == GO && $goroot_fd_estado_post == GO && $goroot_maps_estado_post == GO && $goroot_usuarios_despues -eq 0 && $goroot_fd_despues -eq 0 && $goroot_maps_despues -eq 0 ]] || fallar_carril GOROOT_POSTFLIGHT_AMBIGUO
goroot_retirado=GO
log POSTCONDICION GOROOT_RETIRADO "$goroot:usuarios_$goroot_usuarios_antes:fd_$goroot_fd_antes:maps_$goroot_maps_antes"
fase_actual=SELLADO
printf 'orden\tcarril\treservas\thijos_nacidos\texec_confirmados\tfinalizaciones\testado_conductor\tstdout_sha256\tstderr_sha256\testado_sha256\tsha256sum_atestacion\tresultado\n' >"$control/carriles.tsv"
printf 'carril\truta\tsha256_sha256sums\tficheros_regulares\totras_entradas\tpropietario\tgrupo\tmodo\ttipo\tnlink\tresultado\n' >"$control/atestaciones.tsv"
printf 'carril\tnombre\truta\tbytes\tsha256\tresultado\n' >"$control/capturas_global.tsv"
orden=0
for c in "${carriles[@]}"; do
    ((orden+=1)); a=${atestacion[$c]}
    sellar_paquete "$a" 25 /ninguna || fallar_carril "SELLADO_ATESTACION_$c"
    [[ $("$BB" wc -l <"$a/SHA256SUMS") -eq 24 && $("$BB" find "$a" -mindepth 1 -maxdepth 1 -type f -print | "$BB" wc -l) -eq 25 && $("$BB" find "$a" -mindepth 1 -maxdepth 1 -print | "$BB" wc -l) -eq 25 ]] || fallar_carril "CARDINALIDAD_ATESTACION_$c"
    (cd "$a" && "$BB" sha256sum -c SHA256SUMS >/dev/null) || fallar_carril "VERIFICACION_ATESTACION_$c"
    estado_c=${estado_carril[$c]}
    exec_c=0; [[ -f $a/04_EXEC_CONFIRMADO.tsv ]] && exec_c=1
    resultado=${resultado_carril[$c]}
    printf '%s\t%s\t1\t1\t%s\t1\t%s\t%s\t%s\t%s\t%s\t%s\n' "$orden" "$c" "$exec_c" "$estado_c" "$(sha "$a/captura_stdout.bin")" "$(sha "$a/captura_stderr.bin")" "$(sha "$a/captura_estado.txt")" "$(sha "$a/SHA256SUMS")" "$resultado" >>"$control/carriles.tsv"
    printf '%s\t%s\t%s\t25\t0\torquesta\torquesta\t500\tdirectory\t2\tGO\n' "$c" "$a" "$(sha "$a/SHA256SUMS")" >>"$control/atestaciones.tsv"
    "$BB" awk -F '\t' -v c="$c" 'FNR>1{print c"\t"$1"\t"$2"\t"$3"\t"$4"\t"$10}' "$a/capturas.tsv" >>"$control/capturas_global.tsv"
done
[[ $(($("$BB" wc -l <"$control/capturas_global.tsv")-1)) -eq 9 ]]
resultado_global_material=CANDIDATO_UNANIME_PENDIENTE_SELLADO
printf 'recurso\truta\tantes\tdespues\tprocesos_usuarios\tfd_usuarios\tmaps_usuarios\tresiduos\tretirada_autorizada\tresultado\tdetalle\nventana\tglobal\tabierta\tcerrable\t0\t0\t0\t0\tsi\t%s\tpostflight_completo_global_rojo_%s\ngoroot\t%s\tpresente\tausente\t%s\t%s\t%s\t0\tsi\t%s\tpost_usuarios_%s,post_fd_%s,post_maps_%s\n' "$resultado_global_material" "$global_rojo" "$goroot" "$goroot_usuarios_antes" "$goroot_fd_antes" "$goroot_maps_antes" "$goroot_retirado" "$goroot_usuarios_despues" "$goroot_fd_despues" "$goroot_maps_despues" >"$control/postflight_global.tsv"
incidencia_post=$(sha "$incidencia")
[[ ! -L $incidencia && -f $incidencia && $("$BB" stat -c '%U|%G|%a|%h|%s|%F' "$incidencia") == 'orquesta|orquesta|644|1|16384|regular file' && $incidencia_post == "$incidencia_pre" ]]
printf 'fase\truta\tclasificacion\tinstante_utc\tpropietario\tgrupo\tmodo\ttipo\tnlink\tbytes\tsha256\taccion\nPOST\t%s\tPREEXISTENTE_AJENA\t%s\torquesta\torquesta\t644\tregular\t1\t16384\t%s\tNO_TOCADO\n' "$incidencia" "$(utc)" "$incidencia_post" >"$control/incidencia_telemetria_post.tsv"
log CIERRE RESULTADO "$global_rojo"
escribir_terminal CANDIDATO_UNANIME_PENDIENTE_SELLADO NO_ACREDITADO_HASTA_SHA256SUMS_CONTROL 0 NO_ACREDITADO_HASTA_SHA256SUMS_CONTROL
exec {efd}>&-; efd=
exec 9>&-; diario_abierto=0
sellar_paquete "$control" 25 "$control/coordinador_v3.sh"
trap - EXIT HUP INT TERM
exec {lv}>&-; exec {lt}>&-; exec {lr}>&-; exec {le}>&-
exit "$global_rojo"
```
<!-- COORDINADOR_V3_FIN -->

La recongelación fija la plantilla en **1382 líneas**, **108906 bytes** y
SHA-256
`de397a8228459d7b25272332fc2b2bf19e0f5fe804d958de5ae092a8a34b0ed7`.
La instancia de
`CONTROL` debe coincidir con esos valores y con el blob documental antes de
cualquier preparación.

## Frontera de ejecutables y enlaces

El primer ejecutable es `/usr/bin/busybox`, fichero regular estático ELF
x86-64, `root:root`, modo `0755`, `nlink=1`, 2190672 bytes y SHA-256
`df12634c17fcdca839ae5dc47d7627b7558511f7645de7c99ccf097a0f28ed5b`.
El coordinador usa siempre los applets explícitos `/usr/bin/busybox env`,
`chmod`, `chown`, `cp`, `rm`, `mkdir`, `stat`, `readlink`,
`realpath`, `sha256sum`, `date`, `find`, `cmp`, `sort`, `awk`, `sed`, `grep`,
`tr`, `sleep`, `wc` y `xargs`; todos son el mismo binario pinado, no enlaces separados.
Las construcciones y builtins usados por la plantilla son parte del Bash
pinado y no se resuelven como ejecutables adicionales mediante `PATH`.

Los únicos ejecutables dinámicos directos del bootstrap/coordinador son:

| Ruta | Tipo/lstat | Propietario | Modo | `nlink` | SHA-256 del fichero |
| --- | --- | --- | ---: | ---: | --- |
| `/usr/bin/bash` | regular | `root:root` | `0755` | 1 | `3efccc187bafa75ff1e37d246270ab3e7aa559f242c7a52bf3ec2a1b5450bdbd` |
| `/usr/bin/git` | regular | `root:root` | `0755` | 1 | `5516c9f362c29376ab9a499a33082f9f611941d8c75930c880e30ad109e39c9a` |
| `/usr/bin/flock` | regular | `root:root` | `0755` | 1 | `59bc254984eefd83939a22a590d746942a4583a702b8fd2753bbb92d956e7d4c` |
| `/usr/bin/setpriv` | regular | `root:root` | `0755` | 1 | `9e0d70d26a02c1cb4b984ab6f49a582b7a2c3508b1063ac23adc60073292ae7e` |
| `/usr/bin/setsid` | regular | `root:root` | `0755` | 1 | `ade4a0f0c5df4627b456ab19a1869670e7a58a34b04d817e8703c0fa7b852478` |

`setpriv` tiene 47576 bytes y `setsid`, 14800 bytes. No se usa el applet
BusyBox `su`: en este host conservaría el grupo suplementario 986 (`docker`).
Tampoco se usa el applet BusyBox `setsid`, porque no aporta la espera con
propagación de estado requerida por el protocolo.

Git se invoca con `GIT_ALLOW_PROTOCOL=file`, fuente absoluta, `--no-local`,
`--no-hardlinks`, `--no-checkout`, `--no-tags` y `--single-branch`; después
retira `origin`, hace checkout desacoplado del SHA y verifica cero remotos,
alternates, shallow, commondir, grafts, hardlinks y estado. No existe URL ni
protocolo de red. El ayudante que puede abrir el transporte local queda
sellado como `/usr/lib/git-core/git-upload-pack -> git`, destino regular
`/usr/lib/git-core/git`, `root:root`, `0755`, `nlink=1` y la misma huella de
Git `5516c9f362c29376ab9a499a33082f9f611941d8c75930c880e30ad109e39c9a`.

Los scripts versionados sí resuelven utilidades mediante `PATH`. Para cada
enlace, `binarios_lanzamiento.tsv` conserva **por separado** lstat
(`tipo=symlink`, modo `0777`, `nlink=1`, texto), `realpath`, y stat/hash del
destino. No se usa `stat -L` como sustituto de lstat. El grupo Cargo comparte
destino regular `root:root`, `0755`, `nlink=115`, SHA-256
`48893b0fb21436b54619db80486e83ef39dfccaf1aefe83dfa00c02d6146e8c0`:

```text
/usr/bin/env       -> ../lib/cargo/bin/coreutils/env
/usr/bin/dirname   -> ../lib/cargo/bin/coreutils/dirname
/usr/bin/mkdir     -> ../lib/cargo/bin/coreutils/mkdir
/usr/bin/mktemp    -> ../lib/cargo/bin/coreutils/mktemp
/usr/bin/sha256sum -> ../lib/cargo/bin/coreutils/sha256sum
/usr/bin/date      -> ../lib/cargo/bin/coreutils/date
/usr/bin/stat      -> ../lib/cargo/bin/coreutils/stat
/usr/bin/sort      -> ../lib/cargo/bin/coreutils/sort
/usr/bin/wc        -> ../lib/cargo/bin/coreutils/wc
/usr/bin/timeout   -> ../lib/cargo/bin/coreutils/timeout
```

También se sellan: `/usr/bin/cp -> gnucp` y destino SHA
`ddffb913956d7f8cbdd7722b8c331cea51e9a13c59e92de8c194ae30e5eb0b1e`;
`/usr/bin/rm -> gnurm`, SHA
`175a15a35617f84bb86692b50a3e21b41d8e21fdea242e952b31f1e2e80b1a54`;
`/usr/bin/mv -> gnumv`, SHA
`3a56b5e8f9b40ca7fb484bb00ba048c705f6520c7a9092b45394a9c916ba0b9b`;
`/usr/bin/awk -> /etc/alternatives/awk -> /usr/bin/gawk`, SHA
`a695aa25fdd3f207ece8b3628514c6506c45216ce5486f63d293a83e12dc9798`;
y los regulares `/usr/bin/find` (`efe4843f166525b02f328f0c456a884d6a597f32d2c78cd615b6be0b64279bcf`),
`/usr/bin/pgrep` (`b1e03079307c5c602dd59afdb36f249e262ff2724f98e35a8ddc91d2ea2dffec`)
y `/usr/bin/grep` (`4874cbc734792ae29aa38cd7978aedf1e646563af9e8ee0d5dab59ecb6406417`).
`/bin/sh -> dash` se sella aunque el coordinador no lo invoca. GCC y la
frontera C permanecen en el límite explícito de revisión global.

El origen completo del toolchain es
`/srv/fabrica/orquesta/home/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.6.linux-amd64`.
Es preexistente, `orquesta:orquesta`, modo `0555`, y por sí solo no constituye
una frontera no reemplazable para UID 999. Antes de cualquier bajada de
privilegios o invocación Go, el coordinador root genera sobre él el inventario
canónico ordenado `tipo/ruta_relativa/modo/SHA-256`, exige identidad
`9931a14ad98c26369bf964958b9adeec9edd26162086a51c1842aebe59135511`,
1.335 directorios totales (1.334 internos), 11.536 regulares, cero enlaces u
otros tipos y 215.332.664 bytes regulares. El modo forma parte del inventario:
los 1.334 directorios internos son `0555`, 11.524 regulares son `0444` y 12
regulares son `0555`. No se ejecuta ningún fichero del origen.

Después crea atómicamente la raíz reservada `GOROOT_COPIA` con modo `0700`
como hija root de `/var/tmp` root:root `1777`, copia con BusyBox `cp -R -P`
el GOROOT completo —no solo `go`/`gofmt`—, fija recursivamente `root:root`,
directorios `0555` y conserva exactamente la partición regular `0444`/`0555`,
y exige `nlink=1` para cada regular.
El árbol copiado debe reproducir byte por byte el mismo inventario canónico y
las mismas cardinalidades; su raíz debe ser directorio `root:root`, `0555`,
`nlink=6`. El sticky bit del padre y la propiedad root impiden a UID 999
renombrar o retirar esa entrada, y ningún descendiente es escribible por UID
999. `go` y `gofmt` de la copia son regulares root:root `0555`, `nlink=1`, con
SHA-256 respectivamente `29e6e0b8be61beb1489ceae62b304343566de8a1dc700af74bde7aeb9c80ad45`
y `0ef6fe2d15c972d15b8ecb75dcb9c861e042ac94a0e48baf0202a840648b0fdc`.

Todas las fases Go reciben `GOROOT=GOROOT_COPIA` y
`PATH=GOROOT_COPIA/bin:/usr/bin:/bin`; no existen enlaces bajo `TMPDIR` ni una
ventana hash-a-exec en una jerarquía controlada por UID 999. El inventario
completo, resumen, metadatos, cardinalidades y ambos binarios quedan sellados
antes del preflight. La raíz solo se retira por root después de los tres
carriles y de probar cero `cwd/root/exe`, FD y entradas de `/proc/PID/maps`
que la usen; cualquier lectura ambigua la conserva y da rojo.

## Entornos exactos por fase

`entornos_permitidos.tsv` y cada `entorno_efectivo.tsv` usan
fase/proceso/PID; este último tiene cabecera exacta
`fase\tproceso\tpid\tppid\tnombre\tvalor\torigen\tresultado`. Las listas son:

La lista procede de lectura estática de `conductor.sh` y de los siete bloques
externos sellados; no se deduce de ejecutar Go. El coordinador compara tanto
nombre como valor efectivo y exige presencia observada de cada fase declarada.

| Fase/proceso | Variables permitidas |
| --- | --- |
| Bootstrap externo | Las siete variables contractuales; Bash añade `PWD`, `SHLVL`, `_` y, tras `cd FUENTE`, `OLDPWD`; ninguna cruza sin declarar al coordinador. |
| Coordinador/preparación | El segundo BusyBox `env -i` reaplica las siete variables; Bash añade `PWD=FUENTE`, `SHLVL`, `_`, sin `OLDPWD`; no ejecuta Go. |
| Frontera `setpriv` de Git, `go version` e intento | El primer BusyBox estático entrega solo `LC_ALL=C` y `PATH=/usr/bin:/bin` a `/usr/bin/setpriv`; este recibe el entorno ya vacío. Sus flags literales son `--reuid 999 --regid 982 --clear-groups --inh-caps=-all --ambient-caps=-all --bounding-set=-all --no-new-privs`. El segundo BusyBox `env -i` sustituye otra vez el entorno antes de cargar `setsid`, Bash, Git o Go. |
| Git de clon | `LC_ALL`, `PATH`, `HOME`/`XDG_CONFIG_HOME` no existentes y las tres `GIT_*`; todas contractuales. |
| `go version` | Las doce contractuales `HOME`, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, `TMPDIR`, `GOCACHE`, `GOROOT=GOROOT_COPIA`, `LC_ALL`, `PATH=GOROOT_COPIA/bin:/usr/bin:/bin`, `GOTOOLCHAIN`, `GOENV`, `GOPROXY`, `GOSUMDB`. |
| Supervisor `/usr/bin/setsid` | Exactamente las mismas doce variables privadas del carril; se fotografía desde `/proc/PID/environ` y queda asociado a su PID/PPID/starttime y argv exacto. |
| Puente/lanzador | Las doce anteriores; el puente hereda `PWD=FUENTE`, y Bash deriva `SHLVL` y `_`. |
| Conductor | Las doce anteriores; tras la barrera, el hijo hace `cd TARGET`, y Bash deriva `PWD=TARGET`, `SHLVL` y `_`. |
| Bloque normal | Base del conductor más `CND_RUNTIME_TARGET=TARGET`, `CND_RACE=0`, `CND_CGO_ENABLED=0`, derivados por `conductor.sh`; tras el `cd` versionado, Bash deriva `PWD`, `OLDPWD`, `SHLVL`, `_` dentro de `TARGET`. |
| Bloque race | Igual, con `CND_RACE=1` y `CND_CGO_ENABLED=1`, derivados por `conductor.sh`, y los mismos cuatro valores Bash derivados. |
| `go build` de bloque | Entorno del bloque más `CGO_ENABLED=0` normal o `1` race, derivado por el bloque versionado. |
| C20 `go vet`/`go build` de `abuelo_c20.go` | Entorno del bloque más `GOFLAGS=''`; el build añade además `CGO_ENABLED=0`, todo derivado por C20. |

`CND_LEDGER` y `CND_WATCHDOG_SEGUNDOS` permanecen ausentes para seleccionar
los valores versionados; `CND_RUNTIME_TARGET`, `CND_RACE`,
`CND_CGO_ENABLED`, `CGO_ENABLED` y `GOFLAGS` ya no se prohíben cuando su fase
los deriva. Fuera de la lista aplicable a cada PID, cualquier variable es
roja. Se conserva una sola fotografía de entorno por pareja fase/PID; una
variable repetida también es roja. Las cardinalidades exactas son 10 para
`PREPARACION`; 12 para `GO_VERSION` e `INTENTO_SUPERVISOR`; 15 para `PUENTE`
y `CONDUCCION`; 19 para cada `BLOQUE_*`; 20 para cada `GO_BUILD_*` y `C20`;
y 21 para `C20_BUILD`. El validador exige simultáneamente cabecera, ocho
columnas, PID/PPID, presencia, unicidad, cardinalidad y el valor exacto de
cada variable fija o derivada según su fase; no basta que el nombre esté
permitido. Los snapshots `/proc/PID/environ`, `cmdline`, `stat`, `status` y
`cwd` se toman solo después de que el PID exista.

## Preflight, intento probado y postflight

El coordinador ejecuta realmente, en este orden: validación del padre, delta,
árbol, blob, SHA-256, líneas y bytes documentales contra el argv publicado;
bloqueo de padres; ausencia 26/26; creación de `CONTROL`; copia propia;
inventario y copia completa root:root de `GOROOT_COPIA`; diarios/tablas;
verificación de objetos, bytes, binarios y enlaces; tres
clones; tres atestaciones; cinco raíces privadas por carril; `mode` OFF;
`GOROOT`/`PATH` exactos; `go version`; fotografía global; y
`SHA256SUMS.preflight`. Un fallo da solo `ROJO_PREFLIGHT`, contador global de
conductores cero y conserva cualquier ruta parcial como evidencia roja sin
retry; si una de las 26 rutas no estaba ausente, no crea ninguna.

Cada carril, secuencialmente, materializa con noclobber la cadena:

```text
01_RESERVADO -> 02_HIJO_NACIDO(PID,starttime,argv) ->
03_LIBERADO -> 04_EXEC_CONFIRMADO -> 05_FINALIZADO(estado)
```

`01_RESERVADO` se crea atómicamente antes del primer fork/exec del intento.
La frontera literal es BusyBox estático `env -i` -> `/usr/bin/setpriv`
`--reuid 999 --regid 982 --clear-groups --inh-caps=-all
--ambient-caps=-all --bounding-set=-all --no-new-privs` -> BusyBox `env -i`
-> `/usr/bin/setsid --fork --wait` -> Bash. La atestación exige UID 999, GID
982, lista suplementaria vacía, las cinco máscaras de capacidades a cero y
`NoNewPrivs=1` en `go version`, en el supervisor `setsid` y en el conductor;
encontrar el grupo histórico 986 en cualquiera es rojo. El helper comprueba
además que los cuatro identificadores real, efectivo,
guardado y de filesystem son uniformemente `999`/`982`; la fila durable
publica los valores efectivos `uid=999` y `gid=982`.
El padre conserva PID y `starttime` del supervisor `setsid`; el hijo de
`setsid` comunica después
de nacer su PID y `starttime` y se bloquea. El padre verifica que su PPID es el
supervisor y que `PID=PGID=SID`, fotografía proceso y entorno, y crea una única
barrera `03_LIBERADO`. Tras liberarlo, exige el mismo PID/starttime y el hash
exacto de `/proc/PID/cmdline` correspondiente a `/usr/bin/bash`, conductor,
target y destino; solo entonces crea `04_EXEC_CONFIRMADO`. Espera al supervisor
durante como máximo 72000 observaciones separadas por 0,1 segundos
(7200 segundos) y lo recolecta con `wait`; `setsid --wait` propaga el estado
del hijo y solo entonces se crea `05_FINALIZADO`. Agotar ese límite es rojo,
no una segunda ejecución.

El seguimiento de vida se inicializa antes de entrar en los carriles. Justo
después de cada lanzamiento en segundo plano, sin ninguna orden falible entre
ambos puntos, el coordinador guarda `$!` como `job` y PID del supervisor; hace
lo mismo con el monitor. Obtiene y conserva después el `starttime` de cada uno.
La sesión hija se identifica por el marcador `02_HIJO_NACIDO`; si este aún no
existe, solo se declara tras detener el supervisor y observar exactamente cero
o un hijo directo: el único hijo admisible debe ser líder `PID=PGID=SID`, UID
999 y tener el hash exacto del argv del puente o del conductor. Ningún proceso
se señala si un nuevo muestreo de `/proc/PID/stat` no conserva la pareja
PID/starttime publicada; un PID reutilizado es rojo, nunca un objetivo de
`kill`. Supervisor y monitor son hijos directos no recolectados, por lo que su
PID no se libera entre esa comprobación y una señal. Para la sesión no se
itera `kill PID`: con el supervisor observado en `T`/`t`, el hijo líder queda
sin recolectar y ancla PID, SID y PGID; el coordinador exige que todos los
miembros observados compartan ese PGID y emite una sola señal al grupo
negativo. Otro PGID, líder ausente o lectura cambiante es ambigüedad y no se
señala.

La primitiva de señal sigue siendo numérica (`kill -SEÑAL -- -PGID`): justo
antes y justo después se revalidan PID+starttime del supervisor, su estado
detenido `T`/`t`, y PID+starttime+SID+PGID del líder y miembros conocidos. Esta
garantía cubre la frontera cooperativa y la carrera controlada por el
coordinador; no promete inmunidad si un actor externo privilegiado fuerza la
muerte o reutilización entre instrucciones del kernel. No se añaden `pkill`,
`pidwait` ni otra dependencia para afirmar una garantía global inexistente.

Cada observación de vida tiene uno y solo uno de tres resultados: retorno 1 y
estado `AUSENTE`; retorno 0 y estado `VIVO_IDENTICO`; o retorno 2 y estado
`IDENTIDAD_AMBIGUA`. Ningún `|| break`, `|| true` ni condición booleana
convierte retorno 2 en ausencia. La ambigüedad queda pegajosa para todo el
cierre y prohíbe tanto `GO` como retirada de evidencia.

Toda salida roja usa el mismo cierre: verifica PID/starttime del supervisor,
envía `STOP` y espera además a observar estado kernel `T`/`t`; solo entonces
vuelve a descubrir exactamente cero o una sesión hija, cerrando la carrera
descubrimiento-fork. Después recolecta o termina el monitor; señala con `TERM`
el único grupo de la sesión anclada y espera hasta 50 veces 0,1 segundos a que
todos sus miembros sean zombis no ejecutables. Solo si no ocurre, reaplica el
mismo control de identidad y grupo antes de una única `KILL`, y espera otras
50 veces. Entonces reanuda/termina y recolecta el supervisor, y solo al final
exige cero procesos del SID. Un hijo ausente se acredita únicamente bajo el
supervisor ya detenido; un hijo ilegible, múltiple, no líder, con otro PGID,
argv inesperado o cuyo PID cambie es `IDENTIDAD_AMBIGUA`, nunca ausencia.

Los flags distintos `lanzado` y `recolectado` impiden equiparar un PID cero
inicial con un `wait` terminado. Supervisor y monitor lanzados deben quedar
explícitamente `recolectado=1`; la sesión conocida debe terminar en `AUSENTE`
tras un barrido sin PID reutilizado. El estado real de `wait` del monitor se
conserva en `invocacion.tsv`; en la ruta verde debe ser exactamente cero y no
se sustituye por `|| true`. La ruta de `EXIT` repite idempotentemente este
cierre, registra una clasificación terminal y nunca retira rutas. Solo el
postflight ordinario, después de evidencia verde y de comprobar en lectura
cero procesos usuarios, cero FD y cero mapeos `/proc/PID/maps` sin
ambigüedad, retira las raíces privadas y repite las tres comprobaciones;
cualquier rojo conserva toda la evidencia. La misma guarda triple se aplica
a `GOROOT_COPIA` tras el tercer carril; una referencia o lectura ambigua
impide retirarla y prohíbe avanzar.
Timeout de `02_HIJO_NACIDO`, timeout de exec, timeout del supervisor, fallo del
monitor, aserción bajo `set -e`, señal o postflight rojo terminan el
coordinador: no usan `continue`, no reintentan y no empiezan ningún carril
posterior.

La lectura de `/proc/PID/stat` elimina el prefijo hasta el último `) ` y toma
los campos relativos restantes; nunca separa ingenuamente toda la línea por
espacios, por lo que un `comm` con espacios o paréntesis no desplaza PPID,
PGID, SID ni `starttime`. No se usa FIFO: la barrera son marcadores regulares
`noclobber`. Antes del primer `exec` de Git, `go version` o del intento, las
redirecciones cierran los FD del diario, del estado terminal y de los cuatro
locks heredados; el hijo vuelve a cerrar defensivamente todos los FD mayores o
iguales a 3 antes de registrar el handshake y ejecutar el conductor.

GO exige una fila y un fichero de cada transición, una reserva, un hijo, un
exec confirmado y una finalización. Hueco, caída, segundo marcador, segundo
intento, PID/starttime distinto, proceso directo observado fuera del grupo o
ausencia de cmdline exacto es rojo terminal sin retry. El monitor nace con el
hijo todavía bloqueado, antes de liberar la barrera, comprueba en cada ciclo
PID+starttime de la raíz y muestrea `/proc` cada 0,1 segundos hasta que esa
identidad desaparece; una lectura estable ilegible o su propia salida no cero
es roja. El coordinador preserva stdout, stderr y estado, fotografía
antes/durante/después, valida el destino, copia el
modo de telemetría y retira las cinco raíces privadas. Los tres carriles se
consumen en orden productor, funcional, seguridad y nunca concurren.

Los instantes usan RFC 3339 UTC a segundos (`YYYY-MM-DDTHH:MM:SSZ`); el orden
no depende del reloj, sino de `secuencia` y de la cadena SHA-256 del diario y
los cinco marcadores. El bloqueo exclusivo, la secuencia y las
fotografías/monitor acreditan ausencia de otra conducción dentro de la
frontera cooperativa declarada. No afirman observación continua del kernel ni
defensa frente a una ejecución directa que el mismo UID lance deliberadamente
fuera del lock y del protocolo; esa conducta está fuera de la capability y no
puede convertirse en GO si es observada.

## Entrada, salida y estados exactos

Un carril solo es `GO` si cumple conjuntamente:

```text
estado final del conductor = 0
reservas/hijos/exec/finalizaciones = 1/1/1/1
stdout exterior = "GO\n" (3 bytes)
SHA-256 stdout   = 13763c7184f311500fa5b80d76abcca71e22d8c7928b9dc5ca6b8840a50a558d
stderr exterior = vacío (0 bytes)
SHA-256 stderr   = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
captura_estado  = "0\n"
SHA-256 estado   = 9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa
```

El destino contiene exactamente 22 ficheros regulares: `SHA256SUMS` y las 21
entradas que este manifiesto enumera. No contiene enlaces, subdirectorios,
logs no vacíos ni entradas adicionales. `sha256sum -c SHA256SUMS` valida
21/21, y el conjunto de nombres del manifiesto es exactamente el inventario
del destino salvo el propio `SHA256SUMS`.

`resumen.txt` debe declarar, con valores derivados:

```text
resultado=GO
go_version=go version go1.26.6 linux/amd64
target=TARGET_REAL_EXACTO
head=SHA_DOCUMENTAL
sha_conductor=cc1f47f8d4dd49df71effd4886de732748a261f9fc1964a4c524037d5e4be0d7
fuentes=10
sha_fuentes=ba2b0a1c9838f57ca43d53ec6133ae74bfa7452f7511421e367d7fc5f3d079b0
bloques=14
casos_registrados=74
watchdog_segundos=300
fd_conductor_inicio=N
fd_conductor_fin=N
residuos=cero
```

`N` puede depender de los descriptores de captura del carril, pero debe ser un
entero no negativo idéntico al inicio y al final. No se permite comparar solo
la cardinalidad de C21: las cien entregas usan el inventario estructurado G7a
por número y tupla completa.

`manifiesto.tsv` tiene cabecera y catorce filas: siete bloques normal y los
mismos siete con build `-race` real. Cada fila conserva el SHA exacto de su
script, `estado_script=0`, al menos una fila y `resultado=GO`. No hay bloque
ausente, duplicado, `NO-GO` o `SKIP`.

`casos.ndjson` contiene exactamente 74 registros. Cada TSV de caso termina
exclusivamente con sus estados contractuales y `resultado=GO`; el valor
observado coincide con `esperado`, y las columnas de E/S coinciden con el
oráculo versionado. No se reinterpreta un estado negativo causal como éxito.

C21 queda fijado de forma especial en los dos sidecars:

```text
c15_c21_normal_c21_indices.tsv: cabecera + índices 1..100
c15_c21_race_c21_indices.tsv:   cabecera + índices 1..100
caso=TUPLA_C
esperado=0
estado=0
stdout=0
stderr=0
resultado=GO
```

Las dos filas agregadas `C21_CIEN_INVENTARIOS`, normal y race, contienen
también esperado 0, estado 0, stdout 0, stderr 0 y `GO`. Por tanto el cierre
exige C21 `100/100` normal y `100/100` race, sin reintento de índice, y no una
media ni una tasa de éxito. La validación conjunta pasa ambos TSV a un solo
`awk` y usa `FNR==1`/`FNR>1`: reinicia el tratamiento de cabecera para el
segundo fichero y nunca evalúa esa segunda cabecera como fila de datos.

`C15_AF` conserva estado esperado y observado 65, stdout y stderr cero. C16,
C17 y C18 conservan estado esperado y observado 0, stdout y stderr cero. Los
estados test-only 77--113 continúan siendo causas de rechazo, no resultados
verdes. Los estados 124, 126, 127 y 128+N reservados por `timeout`, una señal o
una interrupción son siempre rojos.

El target permanece limpio y en `SHA_DOCUMENTAL` después de la ejecución. Las
cinco raíces privadas `HOME`, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, `TMPDIR` y
`GOCACHE` quedan ausentes tras preservar su evidencia; `residuos.txt` tiene
cero bytes; no queda hijo, zombi, grupo, FD o proceso gobernado por el carril.

## Taxonomía cerrada

Antes de un preflight verde no se asigna `ROJO_CARRIL_AUSENTE` ni ningún
resultado a los tres carriles. Un fallo de preflight tiene un solo estado
global, `ROJO_PREFLIGHT`, y cero conductores. Después de preflight verde, la
fase fijada **antes** de cada operación y el primer evento terminal forman la
partición siguiente; no se vuelve a evaluar el mismo hecho contra una fase
posterior:

| Orden de fase | Resultado terminal | Única clase de eventos admitida |
| ---: | --- | --- |
| 1 | `ROJO_PROTOCOLO` | Fallo durante reserva/handshake/barrera/exec confirmado, identidad PID+starttime, sesión, monitor, exclusión cooperativa o interrupción antes de una finalización válida. |
| 2 | `ROJO_EJECUCION` | Con exec ya confirmado: timeout del supervisor, señal/estado del conductor distinto de cero, stdout distinto de `GO\n` o stderr no vacío. |
| 3 | `ROJO_EVIDENCIA` | Con finalización y E/S verdes: artefacto/captura/hash ausente o adicional, entorno/privilegio/oráculo/cardinalidad discrepante o sellado fallido. |
| 4 | `ROJO_POSTCONDICION` | Con ejecución y evidencia verdes: target, proceso, FD, hijo, zombi, grupo, telemetría, residuo o retirada final no cumplen. |
| 5 | `GO` | Exactamente una reserva, hijo, exec y finalización, y las cuatro fases anteriores cerradas verdes. |

`PENDIENTE_CARRIL` solo describe el carril cuyo turno aún no comenzó. Si un
carril anterior cae rojo, los posteriores quedan `NO_INICIADO_POR_ROJO_PREVIO`
sin recibir un segundo dictamen ni producir `ROJO_CARRIL_AUSENTE`. El primer
rojo es terminal, se escribe con fase y motivo en `estado_terminal.tsv` y no
se inicia otro carril.

El agregado se clasifica una sola vez por esta precedencia total:

| Precedencia | Resultado global | Condición exclusiva |
| ---: | --- | --- |
| 1 | `NO_EJECUTADO` | Aún no existen doble GO documental y commit `SHA_DOCUMENTAL`; no comenzó el preflight. |
| 2 | `PREFLIGHT_PENDIENTE` | El bootstrap autorizado comenzó y todavía no existe cierre verde ni rojo. Es transitorio. |
| 3 | `ROJO_PREFLIGHT` | Primer evento terminal anterior a `SHA256SUMS.preflight`; exige cero reservas y cero conductores. |
| 4 | `EJECUCION_PENDIENTE` | Preflight verde, ningún evento rojo y todavía no concluyeron los tres carriles. Es transitorio. |
| 5 | `ROJO_PROTOCOLO`, `ROJO_EJECUCION`, `ROJO_EVIDENCIA` o `ROJO_POSTCONDICION` | Primer evento terminal material, clasificado exclusivamente por la fase de la tabla anterior; los carriles posteriores no empiezan. |
| 6 | `ROJO_UNANIMIDAD` | Los tres carriles cerraron `GO`, pero la comparación cruzada discrepa. No absorbe un rojo anterior ni un carril no iniciado. |
| 7 | `ROJO_REVISION` | Existen tres `GO` sellados y cualquiera de las dos revisiones postcanónicas dicta `NO-GO`; es terminal aunque la otra siga pendiente o después sea GO. |
| 8 | `CANDIDATO_UNANIME_PENDIENTE_SELLADO` | Los tres carriles terminaron candidatos verdes, pero `CONTROL` aún no ha superado su sellado; es transitorio y nunca es aceptación. |
| 9 | `GO_UNANIME_PENDIENTE_REVISION` | Solo una verificación posterior, estrictamente de lectura, confirma `SHA256SUMS` 24/24, cardinalidad, permisos y la fila candidata; entonces deriva este estado. |
| 10 | `GO_UNANIME_REVISADO` | Existen tres `GO` y ambas revisiones postcanónicas dictan `GO`, cada una con `P0=P1=P2=0`, sobre el mismo conjunto sellado. |

Las filas transitorias solo describen una fotografía mientras la ventana
autorizada sigue activa; no son aceptación. Las revisiones postcanónicas solo
son admisibles después de tres `GO`, `ROJO_REVISION` precede siempre a
`GO_UNANIME_PENDIENTE_REVISION`, y cada evento real pertenece a una sola fase.
La taxonomía es así exhaustiva y produce exactamente una clasificación.

No existen `PARCIAL`, `ACEPTABLE`, `FLAKY`, `RETRY`, `SKIP`, mayoría,
promedio, cuarentena verde ni excepción por infraestructura. Una
indisponibilidad es roja o impide ejecutar; nunca equivale a éxito.

## Consistencia cruzada de los tres carriles

Después de terminar las tres invocaciones, y sin ejecutar el conductor de
nuevo, las revisiones comprueban:

1. mismo `SHA_DOCUMENTAL`, árbol, blob documental y bytes técnicos;
2. tres targets distintos, limpios y propiedad de `orquesta`;
3. tres destinos distintos y nuevos, sin hardlinks entre entradas;
4. igualdad byte a byte de `fuentes-verificadas.tsv` y de los dieciséis TSV
   de casos/sidecars entre carriles;
5. igualdad semántica de manifiestos y NDJSON ignorando exclusivamente
   instantes, duraciones y la ruta absoluta predeclarada del target;
6. los tres `SHA256SUMS` válidos individualmente, aunque sus hashes globales
   puedan diferir por los campos volátiles anteriores;
7. tres cadenas completas 1/1/1/1, tres estados cero, tres stdout `GO\n` y
   tres stderr vacíos;
8. FD inicio=fin, 14/14 bloques, 74 casos, C21 100/100 normal y 100/100 race,
   y residuos cero en cada carril.

Una diferencia en estado, E/S, caso, selector, fuente, huella, inventario o
resultado es `ROJO_UNANIMIDAD`. Solo fecha, duración, ruta exacta ya
predeclarada y el valor común interno de FD de cada carril pueden variar; el FD
puede variar entre carriles, pero nunca dentro de uno.

## Evidencia durable y doble revisión

La evidencia consumible está formada por `CONTROL`, los tres `ATESTACION`, los
tres `DESTINO` y las futuras actas funcional y de seguridad. No depende de
terminal, scrollback, historial de shell, `/proc` ya desaparecido, memoria del
operador ni ficheros que se vayan a limpiar.

### Inventarios cerrados V3

`CONTROL` termina con exactamente 25 ficheros regulares y cero entradas
adicionales. Su manifiesto final enumera y valida 24/24; el manifiesto de
preflight enumera 15/15 ficheros ya cerrados y excluye deliberadamente el
diario y el estado terminal todavía abiertos. Inventario y cabeceras exactas:

| # | Fichero | Cabecera exacta o contenido |
| ---: | --- | --- |
| 1 | `contrato.tsv` | `sha_documental\tpadre_documental\tantecesor_documental\tbase_tecnica\tarbol\tblob_documento\tsha256_documento\tlineas_documento\tbytes_documento` |
| 2 | `bytes_tecnicos.tsv` | `ruta\tmodo\tlineas\tsha256` |
| 3 | `rutas_reservadas.tsv` | `ordinal\tclase\tcarril\tsimbolo\truta_absoluta\tdebe_ausente_inicial` |
| 4 | `ausencia_inicial.tsv` | `ordinal\truta_absoluta\tinstante_utc\tpid_observador\tuid_efectivo\tgid_efectivo\ttipo_observado\tresultado` |
| 5 | `incidencia_telemetria_pre.tsv` | `fase\truta\tclasificacion\tinstante_utc\tpropietario\tgrupo\tmodo\ttipo\tnlink\tbytes\tsha256\taccion` |
| 6 | `clones.tsv` | `carril\ttarget\thead\tarbol\tstatus_porcelain_v2_sha256\tremotos\talternates\tshallow\tcommondir\tgrafts\thardlinks\tpropietario\tgrupo\tmodo\ttipo\tnlink\tresultado` |
| 7 | `toolchain.tsv` | `carril\tbinario\truta\tsha256\tversion_esperada\tversion_observada\tstdout_sha256\tstderr_sha256\testado\tpid\tstarttime_ticks\tresultado` |
| 8 | `toolchain_goroot.tsv` | `fase\truta_origen\truta_copia\tidentidad_sha256\tdirectorios_totales\tdirectorios_internos\tficheros_regulares\tregulares_0444\tregulares_0555\totras_entradas\tbytes_regulares\tpropietario_origen\tgrupo_origen\tmodo_origen\ttipo_origen\tnlink_origen\tpropietario_copia\tgrupo_copia\tmodo_copia\ttipo_copia\tnlink_copia\tpadre_copia\tpropietario_padre\tgrupo_padre\tmodo_padre\ttipo_padre\tgo_sha256\tgofmt_sha256\tresultado`; exactamente una fila `MATERIALIZADA` |
| 9 | `toolchain_goroot_inventario.tsv` | `tipo\truta_relativa\tmodo\tsha256`; exactamente 12.870 filas de datos: 1.334 `D/0555`, 11.524 `F/0444` y 12 `F/0555`, identidad de las filas `9931a14ad98c26369bf964958b9adeec9edd26162086a51c1842aebe59135511` |
| 10 | `binarios_lanzamiento.tsv` | `funcion\truta_lstat\ttipo_lstat\ttexto_enlace\tpropietario_lstat\tgrupo_lstat\tmodo_lstat\tnlink_lstat\truta_real\ttipo_real\tpropietario_real\tgrupo_real\tmodo_real\tnlink_real\tsha256_real\tresultado` |
| 11 | `entornos_permitidos.tsv` | `carril\tfase\tproceso\torden\tnombre\tvalor\torigen\tresultado` |
| 12 | `preflight.tsv` | `control\tinicio_utc\tfin_utc\tpid\tppid\tpgid\tsid\tuid\tgid\tumask\treservas\texec_confirmados\tfinalizaciones\tconductores_ejecutados\tresultado\tdetalle` |
| 13 | `procesos_preflight.tsv` | `fase\tinstante_utc\tpid\tppid\tpgid\tsid\tuid\tgid\tstarttime_ticks\tcmdline_sha256\tenviron_sha256\tcwd\tclasificacion` |
| 14 | `coordinador_v3.sh` | instancia exacta de la plantilla embebida, modo final `0500` |
| 15 | `ventana.lock` | fichero regular vacío, bloqueo exclusivo durante la ventana |
| 16 | `SHA256SUMS.preflight` | manifiesto verificable de los ficheros 1--15; se excluye a sí mismo y no incluye los ficheros posteriores todavía abiertos |
| 17 | `diario.tsv` | `secuencia\tinstante_utc\tfase\tevento\tpid\tdetalle\tprev_sha256` |
| 18 | `estado_terminal.tsv` | `instante_utc\tfase\tcarril\tclasificacion\tmotivo\testado_salida_coordinador\treservas\thijos_nacidos\texec_confirmados\tfinalizaciones\tcarriles_go\tresultado`; exactamente una fila si `CONTROL` llega a cierre terminal |
| 19 | `carriles.tsv` | `orden\tcarril\treservas\thijos_nacidos\texec_confirmados\tfinalizaciones\testado_conductor\tstdout_sha256\tstderr_sha256\testado_sha256\tsha256sum_atestacion\tresultado` |
| 20 | `atestaciones.tsv` | `carril\truta\tsha256_sha256sums\tficheros_regulares\totras_entradas\tpropietario\tgrupo\tmodo\ttipo\tnlink\tresultado` |
| 21 | `capturas_global.tsv` | `carril\tnombre\truta\tbytes\tsha256\tresultado`, exactamente 9 filas |
| 22 | `postflight_global.tsv` | `recurso\truta\tantes\tdespues\tprocesos_usuarios\tfd_usuarios\tmaps_usuarios\tresiduos\tretirada_autorizada\tresultado\tdetalle`; exactamente las filas `ventana` y `goroot` |
| 23 | `incidencia_telemetria_post.tsv` | misma cabecera que la incidencia preflight |
| 24 | `permisos_finales.tsv` | tabla de permisos descrita abajo, exactamente 26 filas |
| 25 | `SHA256SUMS` | manifiesto GNU verificable de 24/24 entradas |

`rutas_reservadas.tsv` y `ausencia_inicial.tsv` tienen 26 filas de datos. El
diario y `estado_terminal.tsv` se crean inmediatamente después de `CONTROL` y
mantienen FD separados append-only; el diario encadena cada fila con la huella
de la anterior y el estado terminal admite una sola fila. Ambos FD se cierran
antes del manifiesto final. Antes del sellado, la única fila candidata declara
`CANDIDATO_UNANIME_PENDIENTE_SELLADO` y `NO_ACREDITADO_HASTA_SHA256SUMS_CONTROL`;
solo una verificación posterior, estrictamente de lectura, de `SHA256SUMS`
24/24, cardinalidad, permisos y fila candidata deriva
`GO_UNANIME_PENDIENTE_REVISION`. Un manifiesto ausente o inválido deriva
`ROJO_EVIDENCIA` y nunca verde. En rojo conserva fase,
motivo, estado y contadores aun cuando el paquete parcial quede, por diseño,
sin sello final. Un artefacto o manifiesto ausente sigue siendo rojo, no se
completa tras el fallo. El probe aislado de FD abre un descriptor decimal,
ejecuta `{ exec {fd}>&-; } 2>/dev/null || true`, comprueba con un intento de
escritura silenciado que el descriptor queda cerrado y emite después una marca
por stderr restaurado; no crea temporales.

Cada `ATESTACION` termina con exactamente 25 ficheros regulares y cero
adicionales; su `SHA256SUMS` valida 24/24:

| # | Fichero | Cabecera exacta o contenido |
| ---: | --- | --- |
| 1 | `identidad.tsv` | `carril\tsha_documental\tpadre_documental\tbase_tecnica\tarbol\tblob_documento\ttarget\tdestino\tatestacion\tuid\tgid\tumask\tpropietario\tgrupo\tmodo\ttipo\tnlink\tresultado` |
| 2 | `entorno_permitido.tsv` | `fase\tproceso\torden\tnombre\tvalor\torigen\tresultado` |
| 3 | `entorno_efectivo.tsv` | `fase\tproceso\tpid\tppid\tnombre\tvalor\torigen\tresultado` |
| 4 | `privilegios.tsv` | `fase\tproceso\tpid\tppid\tuid\tgid\tgrupos_suplementarios\tcap_inh\tcap_prm\tcap_eff\tcap_bnd\tcap_amb\tno_new_privs\tresultado`; exactamente tres filas y, una sola vez cada una, las parejas `GO_VERSION/go`, `INTENTO_SUPERVISOR/setsid` y `CONDUCCION/conductor`; todas con `uid=999`, `gid=982` |
| 5 | `argv.tsv` | `fase\tproceso\tindice\tvalor\tsha256_bytes` |
| 6 | `telemetria_pre.tsv` | tres filas: `ANTES_GO_VERSION`, `DESPUES_GO_VERSION`, `ANTES_CONDUCCION` |
| 7 | `01_RESERVADO.tsv` | marcador inmutable de transición 1 |
| 8 | `02_HIJO_NACIDO.tsv` | marcador inmutable de transición 2 |
| 9 | `03_LIBERADO.tsv` | marcador inmutable de transición 3/barrera única |
| 10 | `04_EXEC_CONFIRMADO.tsv` | marcador inmutable de transición 4 |
| 11 | `05_FINALIZADO.tsv` | marcador inmutable de transición 5 |
| 12 | `procesos_antes.tsv` | fotografía del hijo bloqueado |
| 13 | `procesos_durante.tsv` | monitor de la sesión, conductor, bloques y descendientes |
| 14 | `procesos_despues.tsv` | fotografía posterior a la finalización |
| 15 | `captura_stdout.bin` | stdout exterior exacto |
| 16 | `captura_stderr.bin` | stderr exterior exacto |
| 17 | `captura_estado.txt` | estado decimal exterior y `\n` |
| 18 | `invocacion.tsv` | `carril\treservas\thijos_nacidos\tliberaciones\texec_confirmados\tfinalizaciones\tpid_supervisor_setsid\tstarttime_supervisor_setsid\targv_sha256_supervisor_setsid\tpid_monitor\tstarttime_monitor\testado_monitor\tmonitor_recolectado\tpid\tppid\tpgid\tsid\tstarttime_ticks\tcwd\testado_conductor\tresultado` |
| 19 | `capturas.tsv` | `nombre\truta\tbytes\tsha256\tpropietario\tgrupo\tmodo\ttipo\tnlink\tresultado`, exactamente 3 filas propias |
| 20 | `evidencia_conductor.tsv` | `destino\tficheros_totales\tentradas_sha256sums\tsha256sum_validas\tbloques\tcasos\tc21_normal\tc21_race\tfd_inicio\tfd_fin\tresiduos_bytes\tsha256_sha256sums\tresultado` |
| 21 | `telemetria_post.tsv` | una fila `DESPUES_CONDUCCION`, misma cabecera que `telemetria_pre.tsv` |
| 22 | `telemetria_mode` | copia de 14 bytes `off 2026-08-21`, sin LF |
| 23 | `postflight.tsv` | `recurso\truta\tantes\tdespues\tprocesos_usuarios\tfd_usuarios\tmaps_usuarios\tresiduos\tretirada_autorizada\tresultado\tdetalle` |
| 24 | `permisos_finales.tsv` | tabla de permisos descrita abajo, exactamente 26 filas |
| 25 | `SHA256SUMS` | manifiesto GNU verificable de 24/24 entradas |

El total de 25 incluye `privilegios.tsv`; por ello la tabla de permisos tiene
26 filas contando la raíz y el manifiesto final enumera exactamente las otras
24 entradas. Añadir, omitir, duplicar o sustituir ese fichero rompe ambos
cierres.

Los cinco marcadores comparten la cabecera exacta
`secuencia\ttransicion\tinstante_utc\tcarril\tpid\tppid\tpgid\tsid\tstarttime_ticks\targv_sha256\testado_salida\tprev_sha256\tresultado`.
Los 25 nombres de `ATESTACION` son únicos (sin duplicados ni ausencias) y se
enumeran una sola vez en el inventario cerrado. `argv.tsv`, no
`invocacion.tsv`, es la autoridad de los argv literales. A continuación
enumera, por fase y proceso, los argv de preparación y de cada lanzador, con
índice y huella de bytes: preparación, el lanzador completo BusyBox `env -i` ->
`setpriv` -> BusyBox `env -i` -> Bash `--go-version` -> BusyBox `env -i` -> Go
`version`, el conductor y el lanzador BusyBox `env -i` -> `setpriv` -> BusyBox
`env -i` -> `setsid --fork --wait` -> Bash `--hijo` -> BusyBox `env -i` -> Bash
conductor. `invocacion.tsv` solo resume contadores y estados y nunca sustituye
ni duplica esa autoridad.
La huella de un argv es SHA-256 de sus argumentos concatenados, cada uno
terminado en NUL. La plantilla fija además cwd y redirecciones: stdout y
stderr se abren en el proceso exterior antes del primer `exec`, sobreviven a
la cadena y capturan también cualquier error del lanzador; el estado es el
propagado por `setsid --wait`. Cada `capturas.tsv` tiene tres filas propias; solo
`capturas_global.tsv` agrega las nueve.

### Permisos sin autorreferencia

Cada tabla de permisos usa exactamente las columnas:

```text
ordinal ruta propietario_esperado grupo_esperado modo_esperado tipo_esperado nlink_esperado propietario_observado grupo_observado modo_observado tipo_observado nlink_observado resultado
```

El coordinador crea todos los contenidos excepto la tabla y el manifiesto;
antes de abrirlos exige que ambas rutas estén ausentes y que existan
exactamente `total-2` entradas regulares. Después abre con noclobber un FD
nuevo para cada uno de esos dos ficheros, aplica
`chown`, directorio `0500`, coordinador `0500` y demás ficheros `0400`, y
observa por lstat el directorio y todas las entradas en orden byte C. Escribe
por el FD ya abierto 26 filas globales (raíz + 25 ficheros) o 26 filas por
atestación (raíz + 25), cierra la tabla, calcula mediante el segundo FD el
manifiesto que incluye la tabla pero se excluye a sí mismo, cierra el
manifiesto y ejecuta `sha256sum -c` en solo lectura. Escribir por los FD no
cambia propietario, modo, tipo ni `nlink`; la tabla no contiene tamaños ni
hashes del manifiesto, de modo que no existe ciclo autorreferente. Los
revisores repiten lstat y sumas sin escribir.

La vida de los descriptores es cerrada: `pfd` solo escribe
`permisos_finales.tsv` y se cierra exactamente una vez antes de evaluar la
tabla; `sfd` solo escribe después `SHA256SUMS`, se cierra exactamente una vez
antes de validarlo y nunca se usa para la tabla. El sellado corre en un
subshell con trap de salida; cada primitiva de cierre vacía inmediatamente su
variable, por lo que una salida de error cierra únicamente los FD todavía
abiertos, sin doble cierre ni escritura posterior sobre un descriptor
cerrado. El manifiesto nunca se valida antes de cerrar `sfd`.

Artefacto o transición ausente/adicional, una tabla con cardinalidad distinta,
captura sin hash, permiso/enlace discrepante, sobrescritura o reuso produce
rojo terminal. No se completa evidencia después ni se repite un intento.

El revisor funcional reconstruye en solo lectura preflight, los tres argv y
las tres invocaciones, identidad, cardinalidades, estados, E/S, oráculos,
igualdad cruzada, C21 100/100 por modo y residuos. El revisor de seguridad
reconstruye en solo lectura genealogía, UID/GID/umask, propietarios, modos,
tipos, enlaces, toolchain, binarios, entorno permitido/efectivo, telemetría,
procesos, aislamiento, hashes, sumas, ausencia inicial, no reuso y límites del
TCB. Ninguno ejecuta conductor, Go, build o caso, ni necesita escribir en la
evidencia. Cada acta usa write-set propio y disjunto y dicta `GO` o `NO-GO`
con `P0/P1/P2`.

El productor no firma ni integra sus resultados. Solo dirección puede
confirmar la unanimidad, materializar actas o decidir la secuencia posterior.
Los modos y hashes son controles cooperativos: no prometen inmutabilidad
frente a otro proceso con el mismo UID 999.

## Límites de UID, cargador y frontera C

La capability es deliberadamente acotada:

1. Los tres carriles usan el mismo usuario cooperativo `orquesta`, UID 999.
   Clones y destinos separados evitan reutilización accidental, pero no aíslan
   frente a otro proceso malicioso con el mismo UID.
2. Los modos `0700/0600`, propietario y hashes acreditan privacidad y
   coherencia bajo esa autoridad; no hacen los paquetes inmutables frente al
   propio UID 999 ni equivalen a una firma externa.
3. El lanzador confiable debe vaciar `LD_PRELOAD`, `LD_AUDIT`,
   `LD_LIBRARY_PATH` y el resto del entorno antes del primer `exec`. El
   conductor no puede atestar retroactivamente lo que un cargador dinámico
   hubiera consumido antes de iniciar Bash.
4. `go` y `gofmt` fijados son estáticos, pero el build race usa CGO, GCC y la
   frontera C del host. Este contrato no atestigua transitivamente
   preprocesador, ensamblador, enlazador, cabeceras, objetos, bibliotecas,
   cargador, kernel o sysroot C.
5. Un triple verde demuestra tres ejecuciones únicas coincidentes bajo esas
   fronteras. No prueba independencia adversarial de UID, una distribución
   estadística ilimitada ni la confianza completa del toolchain.
6. No se acreditan Docker, PostgreSQL, SQL, red, HTTP, navegador, E2E,
   despliegue, producción, datos reales, cumplimiento normativo global ni la
   aplicación completa.

Estos límites son parte del resultado y no pueden omitirse en las actas. Una
revisión global posterior del toolchain es obligatoria precisamente porque la
capability presente no cierra toda esa confianza.

## Secuencia posterior cerrada

La secuencia es estricta:

1. doble revisión independiente de este contrato congelado;
2. commit documental de una sola ruta sobre `PADRE_DOCUMENTAL`, conservando
   `BASE_TECNICA=4840207`, sin amend posterior;
3. preflight conjunto y tres carriles únicos sobre `SHA_DOCUMENTAL`;
4. doble revisión postcanónica del conjunto y, solo si procede,
   `GO_UNANIME_REVISADO`;
5. convergencia documental de `SEC-TOOLCHAIN-P2-CORTE-DB-VULN`, ya aprobada
   funcionalmente y en seguridad para su criterio acotado de seis
   vulnerabilidades y base mutable;
6. candidato nuevo de revisión global del toolchain sobre el SHA ya
   convergido, con revisiones funcional y de seguridad completas;
7. solo tras doble GO global, publicación ordinaria y CI 5/5 puede dirección
   decidir si asigna la siguiente tarea anterior a O4.

La corrección P2 aprobada conserva como autoridades el contenido documental
de `f49c00376a49692991ebc3556b1dc5e8dcfd6f52` y sus dos actas
`00eb616de5cfb9453183b357d70ee757c8f4c8cf` y
`110b2d12f6d583cbea4c8daf70dc818c3973a0cc`; sus objetos equivalentes ya
convergidos en la rama diagnóstica no sustituyen la revisión global pendiente.

El ancestro `74f2495` sigue siendo rojo incluso si los pasos 1--5 terminan
verdes. La nueva revisión global juzga un descendiente nuevo y exacto; no
reclasifica ese commit. O4 permanece cerrado hasta completar todos los pasos,
la publicación y la CI autorizante. Este contrato no crea ni asigna O4.

## Write-set y exclusiones

Write-set único de esta minitarea:

```text
docs/portal_vec/enmienda_o3a_v5_cnd_v4_c21_estabilidad_go1_26_6_2026-08-21.md
```

Quedan expresamente excluidos y byte a byte:

```text
producto
fixtures
G7a y G7b
los tres conductores O3a/O3b/O3c
todos los scripts y casos del conductor
workflow
go.mod y Dockerfile
toolchain instalada
oráculos y ledgers
evidencias y actas históricas
estado transversal y métricas
otros worktrees y ramas
```

Esta fase solo permite lecturas y las puertas documentales/focales autorizadas.
Durante esta corrección y su recongelación tampoco permite invocar `go`,
`gofmt`, `go vet` ni ningún otro binario Go. No permite conductor, builds,
mutantes, Docker, PostgreSQL, red, CI, gates pesados, `fetch`, `pull`, `reset`,
`push`, merge, rebase, integración, despliegue, producción o credenciales.

## Criterio de revisión documental actual

Antes de cualquier commit o conducción, dos revisores independientes deben
comprobar al menos:

1. base y cuatro objetos dependientes exactos;
2. preservación explícita de los rojos `74f2495` y `fea52f3`;
3. capability, invariante y write-set sin ambigüedad;
4. recuento 26/26, ausencia inicial, `CONTROL`, `GOROOT_COPIA`, tres
   `ATESTACION` y cinco raíces privadas disjuntas por carril;
5. paquete global de 25 ficheros, tres atestaciones de 25, inventarios
   cerrados, permisos finales y cinco manifiestos verificables;
6. incidencia común preservada, `mode` privado exacto antes de Go y
   telemetría/residuos privados ausentes al cierre;
7. preflight conjunto anterior a toda ejecución, entorno permitido/efectivo y
   evidencia de ausencia de conducción concurrente dentro de la frontera
   cooperativa declarada, sin extender la promesa al mismo UID no cooperativo;
8. Go exacto 1.26.6, GOROOT completo 12.870/215.332.664 bajo jerarquía
   root:root no escribible, `GOROOT`/`PATH` exactos, guarda
   `cwd/root/exe`+FD+`maps`, binarios de lanzamiento y bytes inmóviles de
   G7a/G7b/ledger/conductor;
9. bootstrap BusyBox y coordinador byte-exacto, anclaje del delta/blob
   revisado, argv/cwd/PID/starttime, grupo/sesión, `STOP` observado `T`/`t`,
   revalidación anterior y posterior a la señal numérica dentro del límite
   cooperativo, estado terminal durable, cinco transiciones 1/1/1/1/1 y nueve
   capturas con hash;
10. estados, E/S, 14/14, 74, C21 100/100 por modo, FD inicio=fin, destino con
    22 ficheros y su `SHA256SUMS` interno 21/21, y residuos cero;
11. taxonomía disjunta con precedencia total, `ROJO_PREFLIGHT` único y
    `ROJO_REVISION` terminal;
12. doble revisión postcanónica de solo lectura y límites UID/cargador/C;
13. convergencia P2 antes de revisión global del toolchain;
14. O4, integración, publicación, CI y métricas todavía cerrados.

Cualquier P0, P1 o P2 impide materializar el commit documental y exige una
corrección del mismo único fichero seguida de nueva revisión completa. El
autor de este borrador no lo autoaprueba.

## Relevo

Mientras falten las revisiones independientes, el estado es:

```text
capability=definida_no_ejecutada
invariante=bytes_tecnicos_inmoviles
write_set=un_documento
resultado_global=NO_EJECUTADO
conductor=prohibido_en_esta_fase
commit=prohibido_en_esta_fase
O4=cerrado
```

El siguiente corte único es la revisión documental funcional y de seguridad
de este mismo blob congelado. No se desbloquea conducción material hasta doble
GO; no se desbloquea convergencia P2 hasta `GO_UNANIME_REVISADO`; y no se
desbloquea O4 hasta la revisión global, publicación y CI 5/5 posteriores.

Estado de espera: **pendiente de nueva revisión completa funcional y de
seguridad**.
