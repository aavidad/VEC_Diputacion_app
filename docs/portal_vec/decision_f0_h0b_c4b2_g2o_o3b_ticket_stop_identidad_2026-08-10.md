# Decisión F0-H0b/C4b-2/G2-O/O3b: ticket, STOP e identidad

Fecha: 10 de agosto de 2026.

Estado: **CANDIDATA DOCUMENTAL**. No autoriza código, integración, ejecución
operativa ni producción. Requiere dos revisiones independientes completas,
publicación y CI 5/5 antes de desbloquear la primera minitarea de código.

Identificador: `O3B-P0-CONTRATO`.

## Resultado observable y autoridad

O3b consume una sola vez el agregado opaco entregado por O3a y produce uno de
dos resultados cerrados:

- `capturado`: el Bash sigue detenido, su identidad física queda acreditada y
  todos los recursos se transfieren juntos a O3c, sin `CONT` ni plazo de caso;
- `retirado`: la primera causa queda congelada y el hijo se recoge mediante la
  retirada ya autorizada, sin devolver recursos utilizables.

El éxito exige, en este orden, releer la autoridad de señal, preparar la trama
en memoria y ejecutar la última barrera, revalidar el bootstrap, escribir una
sola trama `PID_SUPERVISOR|TICKET\n` como primer syscall posterior al verde,
cerrar exactamente una vez su escritor, acreditar el auto-`STOP`, acreditar
identidad `/proc` y transferir el agregado capturado. No se duplica ningún
pidfd.

Prevalecen:

1. la [decisión O3a](decision_f0_h0b_c4b2_g2o_o3a_arranque_mapa_fd_2026-08-09.md)
   para entrada, propiedad, reloj, lease, observador, retirada y fronteras;
2. la [enmienda O3a V5](enmienda_f0_h0b_c4b2_g2o_o3a_v5_autoridad_ci_sin_r_2026-08-09.md)
   para el ledger material y la autoridad CI sin R;
3. la [revisión final O3a V5](revisiones/revision_f0_h0b_c4b2_g2o_o3a_v5_codigo_final_2026-08-10.md)
   para la base publicada;
4. la decisión C4b-2 de [captura y supervisión exterior](decision_f0_h0b_c4b2_captura_quinta_supervision_exterior_2026-08-04.md)
   solo donde no fue sustituida por O1a--O3a;
5. este documento exclusivamente para O3b.

La especificación conjunta G2-O del 5 de agosto permanece `NO-GO` y solo es
genealogía. No se reintroducen `ACK_LISTO`, `ACK_CASO` ni un pipe de recibo:
TERMINAL sigue siendo un fichero regular 0600 cuyo uso pertenece a O4c.

## Alcance cerrado

O3b sí posee:

- consumo lineal del agregado O3a;
- última barrera anterior al primer efecto sobre el ticket;
- escritura y cierre únicos del pipe-ticket;
- observación no recolectora de ambos pidfd explícitos;
- señal cero de grupo exclusivamente mediante pidfd;
- lectura probatoria de `/proc/<PID>/stat`;
- acreditación de `T`, PPID, PGID, SID e instante de inicio;
- transferencia opaca y conjunta a O3c;
- retirada convergente ante cualquier fallo anterior a esa transferencia.

O3b no posee:

- otro `Start`, `Wait`, `waitid` recolector o drenaje de adoptados;
- `CONT`, STOP/TERM/KILL funcional, plazo de 180 segundos o gracia;
- causa terminal funcional, escritura de TERMINAL o recibo exterior;
- parser de CONTROL, selector, ticket, SQL, Docker, datos RRHH u oráculo;
- apertura operativa de Orquesta, Firecracker o Jailer;
- integración, despliegue, producción, datos o secretos reales.

O3c y O4a--O6 permanecen cerrados. O3b no incrementa métricas ni cierra
C4b-2, C4b, H0b, C2, F0 u O4-05.

## Entrada única y propiedad

La única entrada productiva es `*agregadoO3aM38`, no una lista de argumentos.
O3b la consume mediante CAS antes de observar un campo y pone a cero el puntero
del llamador. Nulo, alias, clon, reutilización o discriminante distinto de
O3a/A6 se retiran como `uso_consumido` o entran en fatal si la propiedad no es
acreditable; nunca se reconstruye un agregado.

La custodia recibida contiene conjuntamente:

- controlador O2b exactamente S3 y lector CONTROL limpio L0;
- `exec.Cmd` y `os.Process` con PID positivo y handle pidfd opaco;
- pidfd primario y reserva explícitos, distintos y `CLOEXEC`;
- escritor único del ticket todavía abierto y sin bytes escritos;
- CONTROL lector y TERMINAL escritor, ambos no heredados;
- `leaseGuardiaO3aM38` transferida, sin transacción pendiente;
- observador transferido, baseline de señal y contador monótono;
- `finBootstrapGoM38`, TID y PPID originales;
- forma sellada y snapshot físico final de O3a;
- sobre retenido por el controlador, incluido el ticket opaco validado por
  O1a/O1b/O2a.

No se extrae, envuelve ni cierra el handle pidfd opaco. Los pidfd explícitos
son enteros prestados bajo la lease, no `*os.File`. No se crea una reserva
nueva ni se promueve una referencia. O3b conserva exactamente tres referencias
físicas de la misma identidad durante todo su camino verde.

## Máquina total

Estados privados, no serializables:

```text
B0 recibido
  -> B1 barrera_verde | B7 retirando | BF fatal
B1 barrera_verde
  -> B2 ticket_cerrado | B7 retirando | BF fatal
B2 ticket_cerrado
  -> B3 stop_observado | B7 retirando | BF fatal
B3 stop_observado
  -> B4 identidad_acreditada | B7 retirando | BF fatal
B4 identidad_acreditada
  -> B4T transfiriendo_no_retornable | B7 retirando | BF fatal
B4T transfiriendo_no_retornable
  -> B5 capturado | BF fatal
B7 retirando
  -> B8 retirado | BF fatal
```

No existe transición desde B5, B8 o BF. Un estado distinto, un recurso ausente
o una combinación imposible es invariante fatal, no éxito degradado.

Resultados privados:

- `capturado`: agregado O3c completo, estado B5, ticket ya cerrado y Bash T;
- `retirado`: origen `con_hijo` o `uso_consumido`, primera causa inmutable,
  cero hijo/zombi/FD de caso y custodia negativa de CONTROL/TERMINAL;
- `BF`: `fatalO3aM38`, estado 65 y EOF exterior, sin retorno, log ni E/S
  posterior a descubrir que no puede garantizarse retirada.

No existe resultado `preparado`, `aplazado`, parcialmente capturado ni
`nil,nil`.

## Autoridad de hilo, lease y observador

Toda O3b se ejecuta en la misma goroutine fijada y el mismo TID que O3a. Antes
de cualquier efecto:

1. revalida autoidentidad, registro, pertenencia, generación y TID de lease y
   observador;
2. exige `syscall.Gettid()` igual al TID sellado y PPID igual al original;
3. relee el observador y exige contador igual al baseline transferido y signo
   cero;
4. inicia una operación crítica de lease con permiso opaco y snapshot esperado;
5. cualquier señal, incremento, PPID/TID o autoridad divergente gana y retira.

El contador es `uint64` monótono: no se resetea, decrementa, copia ni acepta
desde el exterior. La última lectura que permite captura ocurre después de la
identidad `/proc` y justo antes del handoff. Un cambio en cualquier ventana
posterior a otra lectura retira; una limpieza no sustituye la primera causa.

Cada syscall sobre recursos poseídos usa la secuencia exacta
`comenzarCritico -> syscall -> consolidar`. No se ejecuta syscall al consolidar,
no se entrega `pending` y no se reutiliza un permiso.

## Última barrera antes del ticket

La barrera es anterior al primer byte del ticket y usa la misma precedencia de
O3a, sin segundo parser:

1. CONTROL en orden de stream se entrega a G4; trama terminal, framing o EOF
   retira con esa primera causa;
2. fragmento, presupuesto agotado, `EINTR` persistente u otro error de lectura
   retira; O3b nunca aplaza con hijo vivo;
3. señal, contador cambiado, `PDEATHSIG`, PPID o TID incoherente retira;
4. bootstrap vencido o menos de un segundo monotónico restante retira; ese
   segundo es la reserva conjunta O3b/O3c ya fijada por O3a, no una promesa de
   tiempo exclusivo para ninguna de las dos fases;
5. ambos pidfd explícitos se sondean sin recolectar: terminalidad, identidad o
   flags divergentes retiran; sin referencia fiable se entra en BF;
6. se exige nuevamente el inventario físico sellado, tres pidfd de una sola
   identidad, CONTROL/TERMINAL/ticket vivos y lease sin `pending`; la ausencia
   de aliases del escritor procede de esa propiedad/snapshot, no solo de
   `fstat` o `fcntl`;
7. se construye en memoria acotada la trama completa y se acredita el escritor
   mediante `F_GETFD`, `F_GETFL` y `fstat` contra su identidad sellada;
8. una relectura CONTROL/observador/PPID/TID y otra lectura monotónica deben
   seguir verdes y dejar al menos un segundo;
9. solo `EAGAIN`, S3/L0, hijo vivo, trama preparada y recursos exactos permiten
   que el primer syscall posterior sea la escritura.

Los límites permanecen literales: buffer 1024, cuatro lecturas, 4096 bytes y
ocho `EINTR`. No se reinicia `finBootstrapGoM38`. La reserva O3b/O3c permanece
en un segundo conjunto: si O3b no termina con tiempo positivo para el `CONT`
inmediato de O3c, no entrega.
Cancelación o EOF ya disponible gana a terminalidad observada después.

Entre el verde de la barrera y el primer byte no se permite otra operación
falible, lenta o de E/S.

## Trama ticket y escritura exactamente una vez

Antes del verde final de la barrera, el contenido se construye en memoria
privada con los bytes:

```text
decimal_canónico(os.Getpid()) + "|" + ticket_retenido + "\n"
```

`PID_SUPERVISOR` es el PID positivo del proceso Go actual, decimal mínimo sin
signo, espacio ni cero inicial. Debe coincidir con el PPID esperado del Bash.
`TICKET` son exactamente los 1..2048 bytes imprimibles `0x20..0x7e` ya
validados y retenidos; no se recodifican, normalizan, registran ni copian a
errores. La longitud total máxima es 2060 bytes: diez dígitos de PID, `|`,
2048 de ticket y `LF`.

El pipe-ticket vacío tiene un solo escritor. Antes del verde final, la barrera
acredita por `F_GETFD`, `F_GETFL` y `fstat` que sigue siendo el extremo escritor
sellado, `CLOEXEC` y sin bytes previos atribuibles; la ausencia de alias procede
del snapshot y la lease. Después del verde O3b:

1. ejecuta una sola emisión lógica mediante un bucle de syscalls acotado que
   avanza solo por bytes realmente
   escritos, admite escritura parcial y reintenta `EINTR` como máximo ocho
   veces, siempre contra el mismo deadline bootstrap;
2. trata `n=0`, `EPIPE`, presupuesto agotado o cualquier otro error como fallo;
3. exige haber escrito exactamente la trama completa, ni un byte más;
4. separa irrevocablemente el escritor de la custodia antes de intentar
   `close`, ejecuta exactamente un syscall de cierre bajo la lease y conserva
   nulo el campo cualquiera que sea su resultado.

No se reintenta la trama desde el principio, no hay segunda emisión, writer
alternativo ni recuperación después de cierre. La emisión lógica puede usar
varios syscalls solo para completar escrituras parciales. Si la escritura no
puede acreditarse, la retirada cierra una vez el escritor que todavía no fue
intentado. Si comenzó el syscall `close`, Linux consume irrevocablemente la
propiedad incluso ante `EINTR` u otro error distinto de `EBADF`: no se reintenta
ni en la retirada, porque el número podría haberse reutilizado. `EBADF` prueba
una divergencia previa de propiedad y tampoco autoriza otro cierre. Todo error
congela la causa y exige retirada; si esta no converge, BF. La retirada nunca
vuelve a escribir.

## Acreditación de auto-STOP

Cerrar el ticket permite al Bash completar su lectura, validar el sobre y
ejecutar su `builtin kill -STOP "$BASHPID"`. O3b no envía STOP.

La barrera exige al menos un segundo. Se fija una sola marca
`finStop = min(ahora+1s, finBootstrapGoM38)`, con
componente monotónico y sin reinicio. Hasta esa marca se observa:

- CONTROL y observador con la misma precedencia anterior;
- ambos pidfd explícitos vivos y de identidad estable;
- señal cero de grupo mediante `pidfd_send_signal(pidfdFiable, 0,
  PIDFD_SIGNAL_PROCESS_GROUP)`; no se usa como sustituto de `/proc`;
- `/proc/<PID>/stat` leído de forma acotada para evidencia.

Solo estado `T` estable en dos lecturas consecutivas, separadas por al menos
una cesión del planificador y con identidad idéntica, acredita auto-STOP. Estado
terminal, ausencia, `R/S/D/I/Z/X`, parseo adverso, cambio entre lecturas o
vencimiento retira. La señal cero nunca cambia el estado del proceso.

## Identidad `/proc` exacta

La única ruta por PID permitida en O3b es `/proc/<PID>/stat`, abierta con el PID
positivo ya retenido; `/proc` es evidencia, nunca autoridad de señalización.
El parser soporta paréntesis y espacios en `comm`: localiza el último `)` que
precede al separador del estado y después exige la cardinalidad completa antes
de convertir. Rechaza truncado, campos extra imposibles, signo, cero inicial,
desbordamiento o lectura mayor de 4096 bytes.

Cada lectura verde exige exactamente:

```text
pid       = Process.Pid
estado    = T
ppid      = os.Getpid()
pgrp      = Process.Pid
session   = SID_supervisor sellado
starttime = inicio positivo sellado en la primera lectura verde
```

El SID del supervisor se obtiene una vez antes del ticket mediante la identidad
propia ya acreditada y no se iguala al PID. La segunda lectura debe conservar
PID, PPID, PGID, SID y `starttime`. Ambos pidfd explícitos y el handle opaco
deben seguir refiriéndose al mismo objeto; una cuarta referencia o un cambio de
flags/identidad retira.

No se abre `/proc/<PID>` como pin, no se usa `stat` para enviar señales y no se
acepta una identidad solo porque el número PID coincida.

## Última revalidación y handoff

Después de la segunda lectura T verde:

1. se relee CONTROL hasta `EAGAIN` con los límites cerrados;
2. se relee el contador y se exige baseline/signo/registro/TID intactos;
3. se revalida PPID, lease, CONTROL, TERMINAL y exactamente tres pidfd;
4. una lectura monotónica exige tiempo positivo hasta `finBootstrapGoM38`; la
   entrega solo es válida si O3c todavía puede ejecutar inmediatamente su
   revalidación y `CONT`; no se exige ni crea todavía el plazo de 180 segundos;
5. una operación propietaria `transferirCapturado` prevalida conjuntamente
   lease y observador sin mutarlos, entra en el subestado privado no entregable
   B4T y transfiere primero el observador y después la lease;
6. solo con ambas transferencias consolidadas se mueve B4T→B5 y se entrega
   inmediatamente. Si la primera consolidó y la segunda falla, la propiedad
   partida no retorna ni intenta rollback: domina BF directo, sin cierre, log
   u otra E/S.

Entre el último verde y el handoff no hay syscall, asignación falible, log o
E/S. El agregado O3c conserva conjuntamente `Cmd`, `Process`, primario,
reserva, handle opaco, CONTROL, TERMINAL, lease, observador, baseline actualizado,
`finBootstrapGoM38`, identidad `/proc` y primera observación vacía. No contiene
escritor ticket ni expone PID/pidfd por separado.

O3c será el único propietario de crear `finCaso=ahora+180s` inmediatamente
antes del primer `CONT`. O3b no prepara, adelanta ni reinicia esa marca.

## Retirada y fatalidad

Antes de B5, O3b es propietario único. Todo fallo controlable usa la retirada
O3a ya autorizada, con estas precisiones:

1. al observar el primer fallo se fija una sola vez
   `finRetiradaO3b=min(ahora+3s, finBootstrapGoM38)`; usa reloj monotónico, no
   se reinicia y no se transfiere. Al no existir ya O3c, no reserva tiempo para
   un `CONT`; si el límite ya venció, domina BF;
2. la primera observación O3b nunca se sustituye;
3. si nunca se intentó cerrar el escritor, se separa de la custodia y se intenta
   una vez; si ya comenzó cualquier syscall de cierre, no se vuelve a cerrar
   aunque aquel devolviera error;
4. se observa terminalidad por primario o reserva sin reapertura por PID;
5. EOF puede bastar antes del STOP; después del ticket completo, si el Bash
   está detenido, se permite un único `SIGKILL` individual por pidfd fiable;
6. terminalidad precede al único `cmd.Wait` de retirada;
7. se cierran ambos pidfd explícitos y los demás recursos según O3a;
8. CONTROL y TERMINAL vuelven solo en custodia negativa; lease y observador no
   se resetean ni recrean.

No se usa `CONT` para retirar en O3b, señal de grupo, PID/PGID numérico,
`Process.Signal`, `Process.Kill` ni `pidfd_open`. Si no hay pidfd fiable, vence
`finRetiradaO3b`, no puede acreditarse el único Wait o queda hijo/zombi/FD, BF llama
directamente a `fatalO3aM38` sin retorno ni limpieza ficticia.

## Precedencias totales

En cualquier punto anterior a B5 gana la primera observación según este orden:

1. CONTROL terminal/framing/EOF ya disponible;
2. escritura/cierre del ticket ya iniciado y su error exacto;
3. señal, contador, `PDEATHSIG`, PPID o TID;
4. bootstrap;
5. terminalidad o corrupción pidfd;
6. STOP/identidad `/proc`;
7. inventario, lease o propiedad;
8. éxito.

Cuando dos condiciones se hacen observables en la misma ronda, el orden de
lectura anterior decide y queda congelado. Los errores de retirada son
secundarios. Ningún evento posterior cambia `CANCELADO/65`, `SENAL_INT/130` o
`SENAL_TERM/143` ya enclavado por G4/O2b.

## APIs y conductas prohibidas

En el grafo productivo O3b quedan prohibidos:

- `Start`, `Run`, `Output`, `CombinedOutput`, `StartProcess`, fork o clone;
- `cmd.Wait`, salvo la llamada única dominada por retirada compartida;
- `waitid` recolector, goroutine de espera o callback/hook/mock;
- `pidfd_open`, `dup*`, `F_DUPFD*`, `os.NewFile` sobre pidfd o cuarta referencia;
- señales por PID/PGID, `kill`, `Process.Signal/Kill` y señal de grupo distinta
  de la sonda cero;
- STOP, CONT, TERM o KILL funcional de grupo;
- `close_range`, cierre masivo, entorno, comando o `ExtraFiles` nuevos;
- cualquier `/proc` salvo `/proc/<PID>/stat` probatorio;
- escritura de CONTROL/TERMINAL, lectura de salida/error o interpretación de
  resultado del caso;
- log, serialización o exposición de ticket, nonce, identidad, PID o pidfd;
- reinicio de bootstrap, plazo STOP o contador; creación del plazo de caso;
- transferencia separada o retorno de un agregado parcial.

## Write-set futuro y grafo de minitareas

Este corte documental modifica solo este fichero. Los documentos transversales
de estado pertenecen a dirección.

La implementación futura se divide obligatoriamente; ninguna fila autoriza la
siguiente por sí sola:

| ID futuro | Responsabilidad observable | Write-set máximo | Criterio de cierre |
| --- | --- | --- | --- |
| O3B-P1-AUTORIDAD | Tipo opaco B0..B8/BF, consumo lineal y validación de entrada; no transfiere lease/observador. | Un Go productivo nuevo + prueba focal. | Nulos/aliases/TID/generación y CAS cerrados; cero efecto ticket. |
| O3B-P2-BARRERA | Última barrera, bootstrap y pidfd sin duplicación. | Un Go productivo nuevo + prueba focal; P1 inmutable. | Precedencias y simultaneidad, cero bytes ticket en negativo. |
| O3B-P3-TICKET | Construcción, escritura parcial acotada y cierre único. | Un Go productivo nuevo + prueba focal; P1/P2 inmutables. | Byte exacto, un escritor, un cierre, fallos convergentes. |
| O3B-P4-STOP | Sonda cero de grupo y dos lecturas T estables. | Un Go productivo nuevo + prueba focal; anteriores inmutables. | Auto-STOP real dentro de 1 s, sin STOP emitido por Go. |
| O3B-P5-IDENTIDAD | Parser `/proc/<PID>/stat` y acreditación de identidad, sin transferir recursos. | Un Go productivo nuevo + prueba focal; anteriores inmutables. | PID/T/PPID/PGID/SID/inicio exactos. |
| O3B-P6-HANDOFF | Última revalidación, transferencia conjunta de lease/observador y agregado O3c opaco. | Un Go productivo nuevo + prueba focal; anteriores inmutables. | Una transferencia, origen consumido y O3c todavía sin efectos. |
| O3B-P7-CONDUCTOR | Conductor durable, AST/DAG y expansión mutante. | Herramientas/pruebas externas; productivo inmutable. | Matriz completa normal/race, mutantes muertos, residuos cero. |
| O3B-P8-EVIDENCIA | Ledger, revisión material, documentación y CI. | Evidencia y documentos propios; código inmutable. | Doble GO, commits autónomos, push normal y CI 5/5. |

Si dos minitareas necesitan el mismo fichero se ejecutan secuencialmente o se
extrae antes una abstracción propietaria revisada. No se permiten ediciones
concurrentes. Cada fichero queda por debajo de 800 líneas; objetivo 500 y
parada local 750 para cualquier fuente O3b. Superar 750 exige refactor o nueva
decisión, no actualizar el ledger.

R, D, G2--G5, capturador, adaptador, SQL, migraciones, fuentes O3a y workflow
quedan byte a byte en P0. La proyección P1 debe fijar rutas concretas, líneas y
SHA antes de autorizar código; este documento no adivina nombres de fuente.

## Oráculos conductuales mínimos

1. Entrada nominal O3a/A6 se consume una vez; alias/nulo/clon/reuso no toca
   ticket ni ejecuta nueva operación.
2. Contador igual permite barrera; cambio antes, durante o justo antes del
   handoff retira y conserva signo/causa.
3. CONTROL `CANCELAR`, EOF, parcial, framing, presupuesto, `EINTR` persistente
   y error de lectura retiran antes del primer byte.
4. Bootstrap con 1 s exacto puede avanzar; un nanosegundo menos retira; el
   segundo es conjunto O3b/O3c y la entrega exige tiempo positivo para el
   `CONT` inmediato; ningún deadline se reinicia.
5. Primario/reserva/handle son tres referencias exactas; una cerrada retira con
   la fiable, ambas explícitas cerradas fatalizan y una cuarta rechaza.
6. Ningún camino O3b invoca `F_DUPFD*`, `pidfd_open` ni crea `*os.File` pidfd.
7. Trama nominal coincide byte a byte con `PID_SUPERVISOR|TICKET\n` para ticket
   1 y 2048; PID mínimo/máximo Linux y caracteres `|` interiores se conservan.
8. Escrituras parciales 1..N, `EINTR` hasta ocho y cierre nominal producen una
   sola trama; noveno `EINTR`, cero, `EPIPE`, corto final o cierre fallido
   retiran sin segunda emisión; un cierre que devolvió `EINTR`, `EBADF` u otro
   error nunca se reintenta.
9. El Bash no alcanza T antes de ticket completo y cierre; después se acredita
   T estable sin que Go emita STOP.
10. `/proc/<PID>/stat` con `comm` que contiene espacios y `)` se parsea; truncado,
    >4096, Z/X, PID/PPID/PGID/SID/inicio distinto rechaza.
11. Señal cero de grupo nominal no altera estado; `EINVAL`, `ENOSYS`, `EPERM`,
    `EBADF` o flag omitido retiran/fatalizan según disponibilidad de reserva.
12. CANCELAR/EOF simultáneo con T gana; señal simultánea con T gana; bootstrap
    simultáneo con T gana antes de identidad.
13. Terminalidad justo tras ticket o durante las dos lecturas T nunca entrega.
14. Fallo tras ticket cerrado retira con un Wait posterior a terminalidad y
    cero hijo/zombi/FD; no escribe ticket otra vez. La retirada fija una sola
    marca de 3 s limitada por bootstrap y prueba sus dos bordes.
15. Handoff contiene CONTROL/TERMINAL/lease/observador/pidfd/Process juntos,
    ticket writer nulo y ninguna capacidad accesible desde el origen.
    Fallo de lease después de transferir observador entra en BF y nunca retorna
    propiedad partida.
16. O3c sigue inalcanzable: cero `CONT`, cero marca 180 s y cero TERMINAL.
17. Conductor exterior acredita BF 65/EOF, stdout/stderr vacíos y no retorno.
18. Cien capturas normales y cien race dejan inventario inicial/final, hijos,
    zombis, grupos, temporales y FD en cero delta.

Cada caso tiene ID estable, comando, SHA del target, estado, stdout/stderr,
duración, inventarios y oráculo explícito. No hay `SKIP`, reintento que oculte
un fallo ni evidencia agregada sin caso causal.

## Familias mutantes mínimas

Cada alternativa se expande a mutantes atómicos compilables `M001..MN`, con
patrón anterior/posterior de cardinalidad uno y oráculo causal:

- B01 aceptar entrada no A6, nula, alias o consumida;
- B02 omitir autoidentidad, registro, pertenencia, generación, TID o CAS;
- B03 omitir primera/última lectura del contador, resetearlo o sustituir causa;
- B04 reordenar CONTROL, señal, bootstrap, pidfd, STOP e identidad;
- B05 aceptar parcial, EOF, framing, `EINTR` o presupuesto como verde;
- B06 reiniciar bootstrap, aceptar menos de 1 s al entrar, prometer una reserva
  exclusiva inexistente o cambiar el plazo STOP de 1 s;
- B07 omitir sondeo de uno de los pidfd, aceptar bit poll adverso o cuarta
  referencia;
- B08 duplicar/reabrir pidfd, usar `pidfd_open`, `dup` u `os.NewFile`;
- B09 construir PID recibido, PID del Bash, signo/cero inicial o separador
  distinto;
- B10 alterar, truncar, normalizar, registrar o exponer el ticket;
- B11 omitir `LF`, duplicar trama, escribir bytes extra o reintentar desde cero;
- B12 aceptar `Write` cero/corto, noveno `EINTR`, `EPIPE` o cierre fallido;
- B13 omitir, anticipar, duplicar, retrasar o reintentar el cierre del escritor
  tras éxito, `EINTR`, `EBADF` u otro error;
- B14 enviar STOP desde Go, usar PID/PGID o confundir señal cero con STOP;
- B15 omitir flag de grupo, usar señal distinta de cero o promover reserva;
- B16 aceptar un único T, estado distinto o lecturas sin estabilidad;
- B17 parser `/proc` por espacios simples, primer `)`, buffer sin límite o
  conversión con signo/desbordamiento;
- B18 omitir comparación PID, PPID, PGID, SID o `starttime`;
- B19 usar `/proc` como autoridad de señal o pin de identidad;
- B20 entregar tras terminalidad, cancelación, señal o bootstrap vencido;
- B21 transferir PID/pidfd/Process/CONTROL/TERMINAL por separado;
- B21A invertir las transferencias observador/lease, omitir prevalidación,
  retornar desde B4T o intentar rollback tras una consolidación parcial;
- B22 conservar writer en capturado o perder CONTROL/TERMINAL;
- B23 crear plazo 180 s o enviar `CONT` en O3b;
- B24 usar `Wait` fuera de retirada o antes de terminalidad;
- B25 retirar con señal de grupo, PID numérico o reabrir identidad;
- B25A omitir, recrear, reiniciar o ampliar `finRetiradaO3b`, o no limitarlo
  por `finBootstrapGoM38`;
- B26 permitir retorno BF, log, cierre o E/S antes del fatal dominante;
- B27 añadir goroutine, callback, global mutable, `init`, hook o segundo owner;
- B28 añadir arista a O3c/O4/O5, parser G4 alternativo o dependencia inversa;
- B29 omitir inventario, lease pending, revalidación física o cierre causal;
- B30 falsear residuos, reutilizar caso fallido o aceptar `SKIP`.

La presencia de una familia no acredita expansión completa. El manifiesto
declara catálogo esperado, todas las alternativas, compilación, ejecución y
resultado; un mutante no compilable o superviviente es `NO-GO`.

## Analizador estructural y evidencia

El analizador AST/tipado futuro debe demostrar como mínimo:

- una entrada productiva O3b y cero entradas desde CLI/operativo antes de O5;
- máquina B total, consumo CAS único y owners sin ciclos;
- cero `Start`, duplicación pidfd, `CONT`, plazo de caso o escritura TERMINAL;
- única escritura ticket dominada por barrera y único cierre posterior;
- `Wait` únicamente dominado por retirada y posterior a terminalidad;
- llamada de señal solo pidfd, flag de grupo y señal cero en éxito;
- `/proc/<PID>/stat` como única ruta PID y parser acotado;
- dos lecturas T, comparaciones completas y última revalidación;
- transferencia conjunta a un tipo O3c aún sin consumidor operativo;
- APIs prohibidas ausentes y grafo sin G7/test hacia productivo.

La evidencia durable se liga a base, todas las fuentes, toolchain Go efectivo,
catálogo mutante, conductor y SHA de cada resultado. Debe ser reproducible en
checkout limpio y no depender de rutas privadas, blobs ignorados, entorno
heredado o caché no inventariada.

## Puertas de cada implementación futura

Como mínimo y sin arrancar el modo operativo:

- `gofmt` y `go vet` de las fuentes cerradas;
- build Linux/amd64 CGO=0 y build `-race` con Go autorizado;
- pruebas focales normales y race;
- conductor durable de todos los oráculos y residuos cero;
- AST/tipado/DAG y todos los mutantes compilables muertos;
- `go test ./...`, `go test -race ./...`, `go vet ./...`;
- `scripts/verificar_calidad.sh`, Gitleaks y `git diff --check`;
- dos revisiones independientes `P0=P1=P2=0`;
- commits pequeños, push sin force y CI 5/5.

Docker/PostgreSQL solo se ejecutarán si una minitarea posterior los necesita y
los autoriza. O3b por sí misma no los usa.

## GO y NO-GO de este contrato

El contrato obtiene GO documental solo si:

1. todas las invariantes, estados, precedencias, límites, ownership y negativos
   son completos y no contradicen O3a publicado;
2. el grafo divide implementación y evidencia en unidades compilables;
3. dos revisores independientes emiten `GO`, `P0=P1=P2=0` sobre los mismos
   bytes congelados;
4. formato, enlaces, secretos y `git diff --check` quedan verdes;
5. el commit toca solo este documento, se publica sin force y su CI termina
   5/5.

Es `NO-GO` si falta una precedencia, un recurso no tiene owner, se permite
éxito parcial, se duplica pidfd, se usa `/proc` para señalizar, se escribe más
de una trama, no se acredita el cierre, STOP lo emite Go, identidad no incluye
`starttime`, se abre O3c, falta expansión mutante, sobrevive un mutante, se
supera un límite, aparece un secreto/ruta privada, una revisión discrepa o una
puerta no termina verde.

## Paradas duras

Se detiene inmediatamente si se modifica otra ruta en P0; se toca un documento
transversal reservado; cambia la base `672b67102be91d33c2a60ca7dfa4d45d6dbd643d`;
se implementa código; se ejecuta Orquesta, Firecracker o Jailer; se abre
producción o datos reales; se presenta O3c como disponible; se integra o
publica sin doble GO; se usa force-push; o cualquier revisión/CI queda no verde.

## Seguridad, privacidad, i18n y accesibilidad

O3b trata únicamente metadatos técnicos y un ticket opaco ya validado. No
introduce identidad humana, perfil, permiso, dato personal, documento, texto
visible ni interfaz. El ticket no se registra ni se expone. Denegación
predeterminada, límites previos y evidencia minimizada son obligatorios.

La matriz normativa de Contratación temporal sigue vigente: este contrato no
autocertifica RGPD, ENS, ENI, accesibilidad ni producción, y no habilita IA ni
decisiones sobre personas. La validación organizativa permanece en las tareas
transversales de dirección.

## Siguiente paso condicionado

Tras doble GO, commit, push y CI 5/5 de este documento, dirección podrá asignar
`O3B-P1-AUTORIDAD`. No queda autoasignada. Hasta entonces O3b, O3c y todas las
fases posteriores permanecen cerradas.
