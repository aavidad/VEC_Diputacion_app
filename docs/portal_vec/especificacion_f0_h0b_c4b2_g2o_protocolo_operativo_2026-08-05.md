# Especificación F0-H0b/C4b-2 G2-O: protocolo operativo

Fecha: 5 de agosto de 2026.

Estado: **NO-GO documental**. La doble revisión detectó contradicciones de
estado, framing y presupuesto; no se programa esta propuesta. Debe sustituirse
por el desglose O0–O6 fijado en la
[revisión independiente](revisiones/revision_f0_h0b_c4b2_g2o_protocolo_operativo_2026-08-05.md).

## Alcance

Esta especificación completa únicamente los literales que la decisión C4b-2
dejó semánticos: tramas, dominios, recibos y relación causa/estado. Conserva
la topología runner → Go → Bash, FD 3..9, `/usr/bin/bash -p`, pidfd de grupo,
plazos monotónicos, propietario único de `Wait` y cuarentena ya aprobados.

## Lector incremental y límites

Control y recibo son streams. El lector incremental admite una trama
fragmentada en múltiples escrituras y varias tramas coalescidas en una sola;
conserva el sobrante después de cada `LF`. No confunde límites de `read` con
límites de mensaje.

Cada trama usa ASCII de siete bits y termina en un único `LF`. Se rechazan NUL,
CR, TAB, byte no ASCII, línea vacía, campo vacío no permitido o más de 1024
bytes incluido `LF`, antes de ampliar el buffer. FD 9 de Go es monoframa: tras
su `LF` solo admite EOF. Tras una trama terminal no se admite otra trama.

Mutantes obligatorios: un byte por escritura, dos tramas por escritura,
1023/1024/1025 bytes, EOF parcial y bytes después del terminal.

| Elemento | Dominio |
| --- | --- |
| Nonce/identidad | 64 hexadecimales minúsculos exactos |
| PID/PGID/PPID | decimal mínimo, `1..2147483647` |
| Inicio `/proc` | decimal mínimo, `1..uint64_max` |
| Estado | uno de `0,64,65,79,130,143` |
| Ticket | 1..2048 bytes imprimibles `0x20..0x7e` |
| Sobre FD 9 | máximo 4096 bytes incluido `LF` |

Se rechazan signo, espacios exteriores, ceros iniciales, desbordamiento,
hexadecimal mayúsculo y ticket con TAB/CR/LF/NUL. Se prueban 2048/2049,
4096/4097 y extremos decimales antes de convertir.

## Sobre inicial FD 9 de Go

```text
V1|SOBRE|NONCE|PID_RUNNER|SELECTOR|IDENTIDAD|LONGITUD_TICKET|TICKET\n
```

Se divide con siete cortes; el ticket puede contener `|`.
`LONGITUD_TICKET` es su longitud exacta. El selector pertenece al catálogo M38
vigente. Go no consume el sobre antes de aceptar `ARMAR`.

## Control Shell hacia Go

```text
V1|CONTROL|ARMAR|NONCE|PID_RUNNER\n
V1|CONTROL|INICIAR|NONCE\n
V1|CONTROL|CANCELAR|NONCE|CAUSA|ESTADO\n
```

Existe un `ARMAR` y después una sola orden. `CANCELAR` solo acepta:

| Causa | Estado |
| --- | ---: |
| `CANCELADO` | 65 |
| `PROTOCOLO` | 65 |
| `SENAL_INT` | 130 |
| `SENAL_TERM` | 143 |

Así el runner transporta y Go repite la primera señal observada. EOF equivale
a una cancelación única; nunca inicia Bash ni sustituye una causa ya enclavada.

## Recibos Go hacia Shell

```text
V1|RECIBO|ACK_LISTO|NONCE|PID_SUPERVISOR\n
V1|RECIBO|ACK_CASO|NONCE|PID_BASH|T|PPID|PGID|SID|INICIO\n
V1|RECIBO|TERMINAL|NONCE|ESTADO|CAUSA|BASH_CREADO|BASH_ESPERADO|ADOPTADOS_PENDIENTES|GRUPO_AUSENTE\n
```

Antes de `ACK_CASO`, Go acredita literalmente desde `/proc`:

```text
PID_BASH | estado=T | PPID=PID_SUPERVISOR |
PGID=PID_BASH | SID=SID_SUPERVISOR | INICIO>0
```

`SID` se conserva, no se iguala al PID: `Setpgid` crea un grupo, no una sesión
nueva. Después fija el plazo de 180 segundos, escribe el ticket y ejecuta
`CONT`. Un intento de `setsid` o cambio de grupo sigue prohibido.

`BASH_CREADO`, `BASH_ESPERADO` y `GRUPO_AUSENTE` son `0|1`.
`ADOPTADOS_PENDIENTES` es la cantidad aún no recolectada al emitir el terminal.
Una terminación controlada exige cero.

La relación terminal es canónica:

| Causa | Estado permitido |
| --- | --- |
| `SALIDA` | `0`, `64`, `65` o `79`, estado real del Bash |
| `CANCELADO` | `65` |
| `PLAZO` | `65` |
| `PROTOCOLO` | `65` |
| `INCIDENTE` | `65` |
| `SENAL_INT` | `130` |
| `SENAL_TERM` | `143` |

Después de `ACK_CASO`, una salida controlada exige
`BASH_CREADO=1`, `BASH_ESPERADO=1`, `ADOPTADOS_PENDIENTES=0` y
`GRUPO_AUSENTE=1`. `CANCELAR` antes del inicio exige ambos indicadores Bash a
cero, adoptados cero y grupo ausente. Un incidente externo puede declarar una
postcondición incompleta, pero siempre es 65, cuarentena y cero caso siguiente.

El supervisor sale con el mismo estado del terminal. El adaptador rechaza una
pareja causa/estado no canónica, un recibo duplicado, un Go todavía vivo o un
estado de `wait -f` distinto.

## Secuencia y descriptores

```text
FD estables -> ARMAR -> ACK_LISTO
ACK_LISTO -> INICIAR -> ACK_CASO -> TERMINAL -> terminalidad -> wait -f
          -> CANCELAR ------------> TERMINAL -> terminalidad -> wait -f
```

FD 3/4 son raíz y runner; 5/6 salida y error; 7 control; 8 recibo; 9 sobre. El
Bash recibe raíz, runner y ticket exclusivamente como 7, 8 y 9. El ticket es:

```text
PID_SUPERVISOR|TICKET\n
```

El comando exacto es:

```text
/usr/bin/bash -p /proc/self/fd/8 --caso-inyeccion-h0b SELECTOR
```

El entorno es lista positiva y no admite `BASH_ENV`, `ENV`, `BASH_FUNC_*` ni
rutas vivas.

## Plazos y terminación

- Shell: armado `t0+2 s`, ACK `t0+4 s`, orden/bootstrap `t0+6 s`;
- Go: bootstrap de seis segundos desde entrada;
- caso: 180 segundos desde el primer `CONT`;
- STOP inicial/final: un segundo; gracia TERM: dos; drenaje: cinco;
- Shell: 190 segundos desde `ACK_CASO` y un segundo de recibo a terminalidad.

Ningún plazo se reinicia. Una goroutine fijada a un hilo posee `Start`, pidfd,
señal terminal, `cmd.Wait`, drenaje, cierres y recibo. Terminalidad previa usa
`poll(pidfd)` o `waitid(P_PIDFD, WEXITED|WNOHANG|WNOWAIT)`. Después ejecuta un
solo `cmd.Wait`, drena adoptados hasta `ECHILD`, exige señal cero=`ESRCH` y
cierra pidfds.

Permanecen prohibidos `CommandContext`, `Cmd.Cancel`, goroutine con `Wait`,
`kill` numérico, `Process.Signal/Kill`, fallback PID/PGID, `setsid` y
daemonización.

## Cierre de G2-O

La matriz cubre cardinalidad `coproc`, FD estables, cada trama omitida,
fragmentada, coalescida, duplicada o truncada, cancelación/EOF, `Start`, pidfd
y duplicado, ticket, muerte paterna, 180 segundos, STOP/TERM/CONT/KILL, líder
y adoptados, recibo adverso, `wait` interrumpido y cuarentena.

Cada mutante controlado deja un terminal, una sola espera, `ECHILD`, `ESRCH`,
FD iniciales restaurados y residuos cero. Las fronteras externas prueban 65,
cuarentena y no reutilización; nunca postausencia ficticia.

G2-O necesita doble revisión independiente `P0=P1=P2=0`, puertas focales y
globales, y conserva byte a byte capturador, D2c, D2d y H0b. No cierra C4b-3,
C4c, C4d, H0b, F0, O4-05 ni producción.
