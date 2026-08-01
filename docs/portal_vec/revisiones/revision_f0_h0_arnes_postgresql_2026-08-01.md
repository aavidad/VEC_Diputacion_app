# Revisión F0-H0: arnés base PostgreSQL 18.4

Fecha: 1 de agosto de 2026.

## Resultado

**GO técnico**, confirmado por dos revisores independientes con
`P0=0`, `P1=0` y `P2=0`.

El commit productor es `b790c97` y su integración exacta en
`integracion/ct-o4-04e-20260726` es `a0d63df`:

```text
test(F0-H0): acreditar arnés PostgreSQL 18.4
```

La ejecución CI `30706398093` terminó completamente verde en sus cinco
puertas: calidad, secretos, artefactos productivos, PostgreSQL de Bolsa y
PostgreSQL de ContextoActor V3.

H0 aporta únicamente el arnés que permitirá probar los componentes dormidos
de F0. No crea todavía la migración `000007`, una fachada productiva ni una
autoridad nueva. Las métricas funcionales no cambian y producción continúa en
`NO-GO`.

## Artefactos congelados

| Artefacto | Líneas | SHA-256 |
| --- | ---: | --- |
| `probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh` | 550 | `96be18afdb1ce4eece9db40bb4e01e0eb84c36d5db066ff9e5e4b6b62a172333` |
| `arnes_fuente_corporativa_contexto_actor_v1.sh` | 531 | `34a5c7b29d4b20eebc9db97d2250a12b5e9f2549f9e6d5732e7db6cabbf42a3e` |
| `operaciones_runner_fuente_corporativa_contexto_actor_v1.sh` | 96 | `8281ac2fe10a2c4609bfb7a87f68f69a1e71189d0d7a3ed946af231b866e2075` |
| `capturar_snapshot_fuente_corporativa_contexto_actor_v1.go` | 799 | `4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902` |

El write-set final contiene exactamente esos cuatro ficheros. El runner queda
en el máximo reservado por D2d; los dos auxiliares y el capturador permanecen
por debajo de sus topes. I0 es el único nodo posterior autorizado a modificar
el runner y debe conservar inmutables los otros tres artefactos y sus huellas.

## NO-GO corregidos antes del cierre

El primer candidato no se confirmó. Las revisiones adversariales obligaron a
corregir, entre otros, estos defectos:

- separar el ciclo de vida Docker del auxiliar SQL y autenticar ambos
  auxiliares y el capturador mediante huellas literales;
- proteger el `cidfile` con modo `0600` antes de interpretarlo y aceptar solo
  64 hexadecimales o 64 más un único salto de línea;
- impedir que la sustitución de comandos de Bash ocultara NUL u otros bytes
  tras un identificador aparentemente válido;
- aplicar el límite de un mebibyte antes de copiar, recorrer, calcular huellas
  o reservar memoria;
- instalar trampas antes de reservar recursos y acreditar propiedad por
  identidad, no solo por nombre;
- evitar borrar una colisión temporal preexistente y retirar un temporal solo
  tras comprobar dispositivo, inode, propietario, tipo y modo;
- conservar el estado causal de `SIGINT`, `SIGTERM`, fallos de `docker run` y
  fallos o falsos éxitos de `docker rm`;
- eliminar un falso positivo de Gitleaks mediante vocabulario no ambiguo, sin
  listas de permitidos.

La primera congelación documental recibió `NO-GO`, `P1=1`, porque el productor
transcribió mal la huella del runner. No hubo cambio de bytes: se redeclararon
las cuatro huellas obtenidas directamente y ambos revisores repitieron desde
cero. Este incidente queda conservado como evidencia de que una congelación
no se acepta por semejanza parcial.

## Evidencia reproducida

Productor, dirección y revisores acreditaron sobre los bytes finales:

- `bash -n` y ShellCheck de los tres shells;
- ejecución directa de ambos auxiliares rechazada con estado `64`;
- `gofmt`, `go vet`, compilación con Go 1.26.5, detector de carreras y
  autoprueba de seis sustituciones por descriptor;
- topes de un mebibyte y byte adicional de sonda antes de `awk`, huella o
  copia;
- parser positivo y negativo de control transaccional y clasificación de
  SQLSTATE real, vacía, múltiple y truncada;
- PostgreSQL 18.4 fijado por digest, sin red y con
  `max_prepared_transactions=0`;
- instalación de `000001..000006`, línea base, dependencias reales, ensayo
  dormido nominal y rollback por error;
- `cidfile` de 64 bytes y de 65 con salto final, y rechazo de NUL, segundo
  salto, exceso, enlace o identificador ajeno;
- supervivencia de nombre, etiqueta, identificador o `cidfile` parciales;
- recuperación de contenedor propio sin `cidfile` y limpieza tras error de
  `docker run` posterior a la creación;
- `SIGINT=130` y `SIGTERM=143` sin contenedor ni temporal residual;
- fallo de `docker rm` aceptado únicamente tras reacreditar ausencia y falso
  éxito rechazado mientras el recurso permanecía;
- colisión temporal previa preservada, creación seguida de error retirada y
  sustitución de inode no eliminada;
- tres ejecuciones nominales sobre el freeze antes de revisión, una ejecución
  completa por cada revisor y otra reproducción sobre la integración;
- `git diff --check`, `git show --check`, Gitleaks por artefacto, huellas
  estables y cero recursos `vec-f0-h0-*` al terminar.

Los revisores no modificaron ni confirmaron el candidato. La integración la
realizó dirección después del doble GO.

## Alcance del GO

El cierre acredita un punto de entrada probatorio reproducible, el snapshot
exacto, la propiedad y limpieza de recursos y las invariantes transaccionales
base. No acredita los validadores, cánones, tablas, consumidor, retirada ni
envoltorios de `000007`.

La siguiente minitarea es A1, limitada a:

```text
deploy/postgresql/autorizacion_atestada_v3/migraciones/
  000007_componentes/010_validadores.sql
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  000007_componentes/010_validadores.sql
```

A1 debe ejecutarse mediante `--etapa A1` y cerrar UTF-8, identificadores,
números, instantes y límites sin ampliar el write-set ni anticipar A2--A4.
