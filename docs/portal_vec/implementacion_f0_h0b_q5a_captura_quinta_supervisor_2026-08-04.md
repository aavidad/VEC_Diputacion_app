# Implementación F0-H0b Q5a: quinta captura y autoprueba ABI

Fecha: 4 de agosto de 2026.

Estado: candidato productor Q5a. No cierra C4b-2, H0b, C2, F0 ni habilita
producción. Requiere revisión independiente antes de integrarse.

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
`VEC_F0_CARGA_PRIVADA=1`. Las cuatro conservan el rechazo directo 64.

La fuente Go se reacredita justo antes de `vet` y compilación: fichero regular,
no simbólico, modo 0600, propietario efectivo, un solo enlace y SHA-256
literal. Esta puerta rechaza una sustitución de la copia privada ocurrida
después de publicar el manifiesto y antes del build.

## Build y autoprueba cerrados

Solo el supervisor se valida y compila con:

```text
GOOS=linux
GOARCH=amd64
CGO_ENABLED=0
GOTOOLCHAIN=local
GOWORK=off
GOPROXY=off
GOSUMDB=off
GONOSUMDB=*
GOFLAGS=-mod=readonly
go build -trimpath
```

El capturador conserva sin cambios su build `-race`. El binario del supervisor
queda como fichero privado 0700, del usuario efectivo y con un solo enlace. El
runner ejecuta su `--autoprueba` y exige además que un modo desconocido termine
con 64.

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
