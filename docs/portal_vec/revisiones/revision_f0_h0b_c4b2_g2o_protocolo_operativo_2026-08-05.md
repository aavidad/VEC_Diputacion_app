# Revisión F0-H0b/C4b-2 G2-O: protocolo operativo

Fecha: 5 de agosto de 2026.

## Dictamen

Estado: **NO-GO documental**.

```text
P0=0
P1=4
P2=2
```

Dos revisores independientes contrastaron la segunda propuesta G2-O con G1,
G2-S, C4b-2 y las fronteras H0b/D2. No modificaron código ni ejecutaron
Docker, PostgreSQL, red o puertas globales.

## Hallazgos bloqueantes

1. La máquina declara una sola orden después de `ARMAR`. Tras `INICIAR` no
   puede transportar una cancelación por señal antes o después de `ACK_CASO`,
   ni aceptar un `TERMINAL` temprano por fallo de `Start`, pidfd o STOP.
2. EOF, bootstrap agotado, trama inválida, muerte paterna y fallos internos no
   tienen causa, estado, fase y precedencia terminal completos. Queda sin
   contrato el intervalo `Start → ACK_CASO`.
3. El máximo general de 1024 bytes contradice el ticket de 2048 y el sobre FD
   9 de 4096. `ADOPTADOS_PENDIENTES` tampoco tiene dominio numérico cerrado.
4. Se prohíbe consumir FD 9 antes de aceptar `ARMAR`, pero los fallos previos
   deben emitir un terminal ligado al nonce que solo contiene ese sobre.
5. No existe un lector Shell binario-seguro y acotado que detecte NUL y exceso
   antes de reservar; `read` ordinario no conserva NUL.
6. Falta una precedencia canónica entre EOF, muerte paterna, señal y
   cancelación simultáneos, y la matriz no separa mutantes controlados de
   fronteras externas irrecuperables.
7. No hay ledger posterior a G2-S. El runner está en 800/800 y G2 en 91/160,
   mientras la estimación histórica del supervisor operativo era 330–450
   líneas. Framing, bootstrap, ciclo de proceso, terminación, puente Shell y
   matriz no forman una sola minitarea revisable.

## Aspectos conservables

- autoridad separada entre runner, supervisor Go, Bash y adaptador Docker;
- pidfd de grupo, subreaper, `LockOSThread` y un único `cmd.Wait`;
- FD 3..9, `/usr/bin/bash -p`, entorno positivo y ausencia de rutas vivas;
- plazos absolutos y secuencia STOP → TERM/CONT → KILL;
- `ECHILD`, `ESRCH`, restauración de FD y cuarentena como postcondiciones.

## Desglose obligatorio antes de código

| Minitarea | Única responsabilidad observable |
| --- | --- |
| G2-O0 | Corregir contrato, precedencias, dominios, write-set y ledger físico. |
| G2-O1 | Framing, dominios y serialización puros en Go. |
| G2-O2 | Bootstrap `ARMAR/ACK_LISTO/CANCELAR/EOF` sin crear Bash. |
| G2-O3 | `Start`, FD, pidfd, STOP, `/proc`, `ACK_CASO` y salida natural. |
| G2-O4 | Motor terminal, plazos, señales, `Wait`, drenaje, `ESRCH` y recibo. |
| G2-O5 | Puente Shell `coproc`, FD estables, lectura acotada y cuarentena. |
| G2-O6 | Matriz integradora de mutantes, residuos y fronteras externas. |

Cada corte debe compilar, autoprobarse y mantener el modo operativo cerrado
hasta completar su responsabilidad. G2-O0 necesita doble `GO` documental. Si
el adaptador supera 600 líneas o el runner requiere otro cambio, se aprueba
antes una nueva separación; nunca se comprimen controles para hacer sitio.

G2-O, C4b-2, C4b, H0b, C2, F0, O4-05 y producción permanecen abiertos.
