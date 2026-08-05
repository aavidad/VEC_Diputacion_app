# C4b-2/G1: autoprueba operativa del supervisor

Fecha: 5 de agosto de 2026.

## Alcance

Este corte amplía el supervisor probatorio M38 antes de conectarlo al caso
funcional. La autoprueba se ejecuta sin Docker, PostgreSQL, red, SQL ni datos
de expediente y falla cerrada con estado `65` antes de cualquier efecto.

G1 acredita sobre Linux/amd64:

- ABI `pidfd_open` y `pidfd_send_signal` con autoridad de grupo;
- creación atómica de cada hijo y su `PidFD` mediante el mismo `Start`;
- grupo cooperativo con `STOP` observado, `CONT`, `TERM` y recolección;
- miembro que ignora `TERM`, gracia acotada y extinción por `KILL` de grupo;
- líder recogido antes que su descendiente real, cuyo `pidfd` se transfiere al
  subreaper mediante un `socketpair` anónimo y `SCM_RIGHTS`;
- señuelo en otro grupo, `EINVAL`, `EBADF`, `ESRCH`, rollback posterior a
  `Start`, un solo `Wait` por hijo directo y ausencia final de hijos y FD;
- rechazo con estado `64` de modos desconocidos y de un ayudante sin la
  capacidad paterna acreditada.

No se incorporan aún el modo operativo, los FD 3..9 del caso, el protocolo
`ARMAR/ACK/INICIAR/CANCELAR/recibo`, el puente `coproc`, el plazo funcional de
180 segundos ni cambios en el adaptador. Todo ello pertenece a G2.

## Autoridad y limpieza

Las señales de grupo usan exclusivamente el `pidfd` del líder y
`PIDFD_SIGNAL_PROCESS_GROUP`. No existen llamadas a `kill` por PID/PGID,
`Process.Signal`, `Process.Kill`, `CommandContext` o `Cmd.Cancel`.

Los pidfd individuales solo se emplean para el rollback de procesos
sintéticos creados por la propia autoprueba. La limpieza acredita terminalidad
antes de `cmd.Wait`, recoge una sola vez cada hijo directo, drena adoptados
hasta `ECHILD` y cierra canales y descriptores. Un `Start` exitoso sin pidfd
se resuelve cerrando el canal de capacidad para que el ayudante termine por
EOF y pueda recogerse sin señalización numérica.

## Huellas reproducibles

| Artefacto | SHA-256 |
| --- | --- |
| Fuente Go, 674 líneas físicas | `736adf31dcafd6314fbd937aae9f906e636b5a0c8fa7d7dd68bc5fb0c03bc08a` |
| Binario Go 1.26.5, `CGO_ENABLED=0`, `GOAMD64=v1`, `-trimpath` | `fff235c390bf179e80e8a21101dd801e143fd5018c9871e6f64d9883a5cab3fb` |

Dos compilaciones aisladas consecutivas produjeron la misma huella binaria.
El runner fija ambas huellas antes de ejecutar la autoprueba.

## Presupuesto

Dirección autorizó priorizar claridad y controles sobre el objetivo local de
300 líneas. El fichero queda en 674 líneas, por debajo del tope duro de 800 de
DEC-051. Esta excepción no autoriza minificación ni crecimiento automático en
G2: antes de G2 debe fijarse una separación compatible con la captura privada
si la proyección amenaza el tope de 800.

## Verificación local previa a revisión independiente

- `gofmt -d`: sin diferencias;
- Go 1.26.5: `vet` y dos builds cerrados reproducibles;
- autoprueba operativa: `50/50` ejecuciones verdes;
- modo desconocido: `64`;
- ayudante sin capacidad paterna: `64`;
- `bash -n` y ShellCheck del runner: verdes;
- `git diff --check`: verde;
- procesos residuales tras el estrés: cero.

Faltan todavía la revisión funcional y la revisión de seguridad independientes,
las puertas globales, el commit de aceptación y la integración. Este documento
no cierra C4b-2, C4b, H0b, C2, F0, O4-05 ni cambia métricas o producción.
