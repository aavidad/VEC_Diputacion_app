# Implementación F0-H0b Q5a: quinta captura y autoprueba ABI

Fecha: 4 de agosto de 2026.

Estado: candidato productor Q5a corregido tras dos dictámenes `NO-GO P1` sobre
`4f36a9a`. No cierra C4b-2, H0b, C2, F0 ni habilita producción. Requiere una
nueva revisión independiente antes de integrarse.

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
- función `env` heredada y `GOAMD64=v3` heredado: 65 y 65, residuos 0;
- binario privado sustituido después del build: 65, sin autoprueba y residuos 0;
- marca de carga y destino superior del adaptador preexistentes como
  `readonly`: 65 y 65, residuos 0;
- SHA-256 determinista del binario limpio: `eb0764c58c7eb2d954abdecc65769a10cfb1f1b4080e5f8c23e417af2df78d86`;
- hashes de D2c, D2d, H0b, adaptador y capturador: invariantes;
- `git diff --check`: limpio.

Límites físicos `wc -l`:

| Fichero o bloque | Líneas |
| --- | ---: |
| Runner | 773 |
| `capturar_auxiliares_privados_f0` | 40 |
| Supervisor Go Q5a | 131 |

No se modificaron el adaptador, D2c, D2d, H0b ni el capturador. La revisión
independiente debe reproducir estas puertas y decidir `GO` o `NO-GO` antes de
autorizar C4b-2 operativo.
