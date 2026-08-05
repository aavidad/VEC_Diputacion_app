# Implementación F0-H0b Q5a: quinta captura y autoprueba ABI

Fecha: 4 de agosto de 2026.

Estado: implementación Q5a **aceptada** sobre `649ee46` tras tres dictámenes
independientes con `P0=0`, `P1=0` y `P2=0`. Conserva los dos commits
autorizados por la enmienda y añade un tercer commit mínimo para cerrar la
allowlist y una supresión nueva de ShellCheck. Cierra solo Q5a; no cierra
C4b-2, H0b, C2, F0 ni habilita producción. La evidencia consolidada está en
[`revision_f0_h0b_q5a_captura_supervisor_2026-08-05.md`](revisiones/revision_f0_h0b_q5a_captura_supervisor_2026-08-05.md).

Nota posterior: este documento conserva la evidencia histórica exacta de
`649ee46`. La corrección de empaquetado detectada por su primera CI, incluidas
las nuevas huellas y la línea adicional del supervisor, se registra aparte en
[`correccion_f0_h0b_q5a_paquete_go_2026-08-05.md`](correccion_f0_h0b_q5a_paquete_go_2026-08-05.md).

## Alcance aplicado

El runner captura ahora en una sola operación privada exactamente estas cinco
fuentes:

1. D2c `arnes_fuente_corporativa_contexto_actor_v1.sh`;
2. D2d `operaciones_runner_fuente_corporativa_contexto_actor_v1.sh`;
3. H0b `arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh`;
4. adaptador `ciclo_recursos_m38_h0b_fuente_corporativa_contexto_actor_v1.sh`;
5. `supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go`.

El capturador ordena canónicamente el manifiesto por ruta. El runner lo compara
con cinco pares literales ruta/SHA-256. Este orden canónico no cambia el orden
de carga: las cuatro fuentes Shell se validan y cargan como
D2c → D2d → H0b → adaptador, cada una con una marca nueva
`VEC_F0_CARGA_PRIVADA=1`. El runner exige la ausencia de la marca antes de
crearla y después de cada carga; el `source` debe consumirla. Las cuatro
fuentes conservan el rechazo directo 64.

Tras cargar el adaptador se acreditan directamente la presencia, el atributo
`readonly` y el valor literal de sus seis destinos superiores: M080, T080,
directorio de wrappers y wrappers sin R0, nominal y de error. Una marca o un
destino preexistente y `readonly` cierra con 65 antes de una asignación fatal.

La fuente Go se reacredita justo antes de `vet` y compilación: fichero regular,
no simbólico, modo 0600, propietario efectivo, un solo enlace y SHA-256
literal. Después del build se recalcula su huella con `/usr/bin/sha256sum`.
Estas puertas rechazan una sustitución de la copia privada ocurrida después de
publicar el manifiesto, durante el build o antes de ejecutar el binario.

## Corte intermedio de refactorización

Las funciones `acreditar_snapshot_contenedor_f0`,
`rechazar_snapshot_adverso_f0` y `probar_snapshot_adverso_f0` se trasladaron
sin cambio semántico del runner a D2d, después de las funciones propietarias
del contenedor y antes de `comparar_huellas_f0`. El cuerpo trasladado coincide
byte a byte con las 47 líneas funcionales de origen. El separador retirado y
añadido, más una anotación local no ejecutable para que ShellCheck reconozca
`temporales` como dependencia inyectada, dejan un diff físico de 48 líneas
retiradas y 49 añadidas.

La captura continúa conteniendo exactamente cinco fuentes y carga D2d antes
del primer uso de las funciones trasladadas. Su huella en ese corte intermedio
fue `db462039da649c6e7e370ce2a3131eeca45ab513dd220099ff2a8d92d5d34502`.
D2c, H0b, adaptador, capturador y supervisor Go permanecieron byte a byte
invariantes en aquel commit.

## Segundo commit: arranque protegido y formato auditable

El runner entra por `#!/usr/bin/bash -p`, comprueba inmediatamente la bandera
`p` y acredita mediante primitivas de Bash que `/usr/bin/bash`,
`/usr/bin/env` y `/usr/bin/grep` son ficheros regulares ejecutables y no
escribibles por el usuario efectivo. Antes de exportar, borrar variables o
seleccionar herramientas, la tubería física `/usr/bin/env -0 |
/usr/bin/grep -zE '^(BASH_FUNC_|LD_)'` conserva los dos elementos de
`PIPESTATUS`. Solo la pareja `(0 1)` puede avanzar; una coincidencia, un fallo
de productor o consumidor, otra cardinalidad u otra pareja terminan en 65.

La puerta no resuelve programas por `PATH`, no usa `grep -q` y no consulta
`$?` antes de copiar `PIPESTATUS`. El cuerpo de cualquier función exportada se
descarta en `/dev/null` y no se importa por el modo protegido. La frontera Go
mantiene además `/usr/bin/env -i` y el rechazo independiente de un `GOAMD64`
heredado.

La garantía empieza en el launcher administrado, su EUID, el cargador y los
binarios del sistema. El runner rechaza `LD_*`, pero no afirma deshacer una
biblioteca que un proceso padre comprometido hubiera cargado antes de entrar
en Bash. Las pruebas nominales partieron de un launcher sin el
`LD_LIBRARY_PATH` presente en el entorno interactivo del host.

Las dos sondas adversas por descriptor arrancan la copia mediante un entorno
vacío, `PATH=/ruta-no-resoluble`, `LC_ALL=C`, `BASH_XTRACEFD=6` y
`/usr/bin/bash -p -x`. Su lista de traza admite únicamente las instrucciones
de arranque previstas y conserva el prefijo literal `+ `. El adaptador privado
abre igualmente cada hijo con `/usr/bin/bash -p`; su nueva huella literal es
`98d22a302bfd8ad3964b9135ce78c655f7a31171088ad9c5c49c285f647a8cb7`.

Se restauraron saltos legibles en la frontera Go, los dos builds, el
manifiesto, las cuatro cargas, las seis postcondiciones, las comprobaciones de
forma y huella y el rechazo del modo desconocido. No se unieron controles para
simular el presupuesto físico.

## Corrección tras la contrarrevisión

Las revisiones independientes de `2bd0311` emitieron `NO-GO` con un P1 y un
P2. Bash 5.3 representa `exec 9<&-` en xtrace como `+ exec`, sin espacio, y la
allowlist admitía solo `exec `. La alternativa quedó limitada a
`exec( |$)`: acepta únicamente `exec` terminal o seguido de argumento. Los
cinco defectos `vacio`, `desconocido`, `repetido`, `sin-ticket` y
`discrepante` conservan hijo 64 y ahora hacen que la prueba agregada termine en
0, sin ampliar otra orden.

El traslado a D2d había añadido una supresión `SC2154`, contraria a la puerta
aceptada. Se sustituyó por `declare -g temporales` inmediatamente después de
consumir la marca privada. La declaración explicita la dependencia inyectada,
conserva el valor y atributos existentes y deja ShellCheck limpio sin una
supresión nueva. D2d mantiene 145 líneas y su huella final es
`9b137f1302c5672e9fd5c0c8df169810cbc7e57a11fa2129bf79a777e92c5e81`.

## Build y autoprueba cerrados

Toda la frontera Go —selección, versión, GOROOT, `vet` y build del capturador y
del supervisor— pasa por `/usr/bin/env -i`. El entorno permitido común es:

```text
HOME=<temporal privado>
TMPDIR=<temporal privado>
GOCACHE=<temporal privado>/cache-go
PATH=/usr/local/go/bin:/usr/bin:/bin
LC_ALL=C
GOENV=off
GOAMD64=v1
GOTELEMETRY=off
GOTOOLCHAIN=local
GOWORK=off
GOPROXY=off
GOSUMDB=off
GONOSUMDB=*
GOFLAGS=-mod=readonly
go build -trimpath
```

El capturador añade explícitamente `GOOS=linux`, `GOARCH=amd64` y
`CGO_ENABLED=1`, y conserva `-race`. El supervisor usa esos GOOS/GOARCH con
`CGO_ENABLED=0`. Una función heredada llamada `env` o un `GOAMD64` heredado se
rechazan antes de invocar Go.

El binario del supervisor queda como fichero privado 0700, del usuario
efectivo, con un solo enlace y sin enlace simbólico. `/usr/bin/stat` y
`/usr/bin/sha256sum` acreditan su forma y su SHA-256 determinista literal
`eb0764c58c7eb2d954abdecc65769a10cfb1f1b4080e5f8c23e417af2df78d86`
antes de ejecutarlo. El runner ejecuta su `--autoprueba` y exige además que un
modo desconocido termine con 64.

La fuente nueva tiene build constraint `linux && amd64`, no usa dependencias
externas y no crea hijos, canales, SQL, Docker ni recursos de caso. La
autoprueba:

- convierte el proceso en líder de su propio grupo y comprueba PID/PGID;
- ejecuta `pidfd_open(self)` mediante el ABI Linux/amd64;
- acredita señal cero con `PIDFD_SIGNAL_PROCESS_GROUP`;
- exige `EINVAL` con una bandera desconocida y vuelve a acreditar la referencia;
- cierra el pidfd y exige `EBADF` al reutilizarlo.

Todo error real, incluidos `ENOSYS`, `EPERM`, `EINVAL` fuera del mutante
esperado u otro fallo del kernel, termina en 65. Q5a no implementa todavía
`CLONE_PIDFD`, supervisor operativo, proceso Bash, plazo, espera ni extinción.

## Evidencia focal del productor

Entorno acreditado: Linux 7.0 amd64 y toolchain local Go 1.26.5. No se usaron
red, Docker ni PostgreSQL.

Resultados:

- `gofmt`: limpio;
- `go vet` cerrado sobre la fuente: 0;
- `go build -trimpath` cerrado: 0;
- autoprueba ABI real: 0, `autoprueba=ok pidfd_grupo=disponible`;
- modo desconocido: 64;
- mutantes cerrados de `pidfd_open` con `ENOSYS` y `EPERM`: 65 y 65;
- `bash -n` del runner: 0;
- ShellCheck `-x` del runner: 0, sin avisos ni supresiones nuevas;
- prueba focal del bootstrap real: manifiesto 5, snapshot 5, binarios 0700,
  autopruebas del capturador y supervisor verdes, residuos 0;
- mutante que sustituye la copia Go después del manifiesto y antes del build:
  65, sin binario de supervisor y residuos 0;
- shebang y `/usr/bin/bash -p`: puerta aceptada; Bash sin `-p`: 65;
- entorno limpio: `PIPESTATUS=(0 1)`; productor, consumidor, cardinalidad y
  pareja mutados: 65;
- funciones `env`, `type` y ambas exportadas: 65 sin ejecutar sus cuerpos ni
  publicar su definición;
- `LD_Q5A_ADVERSO=1` y `GOAMD64=v3`: 65 y 65;
- `BASH_ENV` y `ENV` adversos bajo `-p`: marcadores ausentes;
- `PS4`, `SHELLOPTS` y `BASHOPTS` adversos en las sondas: marcador ausente y
  prefijo de traza literal `+ `;
- binario privado sustituido después del build: 65, sin autoprueba y residuos 0;
- marca de carga y destino superior del adaptador preexistentes como
  `readonly`: 65 y 65, residuos 0;
- SHA-256 determinista del binario limpio: `eb0764c58c7eb2d954abdecc65769a10cfb1f1b4080e5f8c23e417af2df78d86`;
- huellas D2d `9b137f13…5e81` y adaptador `98d22a30…8cb7` fijadas en el
  runner; hashes de D2c, H0b, capturador y supervisor Go invariantes;
- `git diff --check`: limpio.

Límites físicos `wc -l`:

| Fichero o bloque | Líneas |
| --- | ---: |
| Runner | 789 |
| D2d | 145 |
| Adaptador | 527 |
| `capturar_auxiliares_privados_f0` | 66 |
| Supervisor Go Q5a | 131 |

Los dos commits autorizados y su corrección posterior modifican runner, D2d y
adaptador. D2c, H0b, capturador y supervisor Go permanecen byte a byte
invariantes. Los tres dictámenes finales autorizan su integración como base de
C4b-2 operativo.
