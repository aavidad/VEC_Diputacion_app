# Decisión F0-H0b C4b-2: quinta captura y supervisor padre con pidfd

Fecha: 4 de agosto de 2026.

Estado: cuarta corrección de dirección **aceptada** sobre `dd6d619`, posterior
a los `NO-GO` documentales de `f7ca3a9`, `a92ee4b`, `51860cb` y `31c9169`.
El 4 de agosto de 2026 obtuvo tres contrarrevisiones independientes con
`P0=0`, `P1=0` y `P2=0`; autoriza únicamente la implementación incremental
descrita en la secuencia autorizable y no modifica por sí misma C4b-2, H0b, C2,
F0 o producción.

## Historial vinculante de la decisión

`f7ca3a9` propuso un quinto auxiliar Shell. Sus tres revisiones rechazaron la
ambigüedad de autoridades, el solapamiento de write-sets, la falta de reserva
y la carrera `/proc -> kill`.

`a92ee4b` sustituyó aquel auxiliar por un supervisor Go lateral con pidfd. Las
tres contrarrevisiones aceptaron la captura única de cinco, los dos temporales,
el handoff y la separación de responsabilidades, pero volvieron a detener el
código porque:

- el Bash podía existir antes de que el supervisor lateral obtuviera el pidfd;
- la pérdida del supervisor dejaba un Bash que aquel no podía esperar;
- un pidfd de proceso no ligaba los posteriores `kill(-PGID, ...)` numéricos;
- no se acreditaba `PIDFD_SIGNAL_PROCESS_GROUP`;
- `Pdeathsig`, espera, cierre de pidfd y presupuesto no formaban un protocolo
  único reproducible.

La segunda corrección conservó lo aprobado y sustituyó por completo la topología
lateral. No se ha escrito código de Q5a o C4b-2 durante la decisión.

La revisión de `51860cb` aceptó la topología padre, pidfd de grupo, subreaper,
mapa de FD y presupuesto general, pero detuvo de nuevo el código porque no
cerraba `fork -> $!` del propio supervisor, no fijaba el `exec.Cmd`, relegaba
parte de los canales a C4b-3 y admitía reserva cero del runner. Esta tercera
corrección incorpora los cuatro hallazgos.

La revisión de `31c9169` confirmó con sondas Bash que los FDs publicados por
un `coproc` pueden quedar cerrados y sus variables desasignadas al terminar el
coproceso incluso antes de `wait`. También detectó deadlines incompletos de
bootstrap, una rama STOP sin convergencia definida, el flag `WEXITED` ausente y
el caso de caída total del host en el runbook. Esta cuarta corrección resuelve
los cinco hallazgos; el código continúa intacto.

## Prevalencia exacta

Esta decisión desarrolla la cláusula de parada de la
[enmienda de presupuesto y topología](enmienda_f0_h0b_m38_presupuesto_real_topologia_2026-08-02.md)
y sustituye únicamente estos extremos de las enmiendas M38 anteriores:

1. la captura exacta de cuatro pasa a ser una captura exacta de cinco fuentes;
2. la prohibición del quinto auxiliar se sustituye por una fuente Go capturada;
3. el runner deja de ser padre directo del Bash: es padre directo exclusivo
   del supervisor, y el supervisor es padre directo exclusivo del Bash;
4. la exigencia Shell `fork -> $! -> jobs 0/1` del Bash se sustituye por
   `exec.Cmd.Start` con `CLONE_PIDFD` atómico y pidfd no negativo;
5. el runner espera solo a su supervisor; el supervisor espera y recolecta una
   vez su Bash y los descendientes que adopte como subreaper;
6. la señalización del caso deja de usar por completo PID o PGID numéricos y
   usa solo `pidfd_send_signal(..., PIDFD_SIGNAL_PROCESS_GROUP)`;
7. los write-sets y presupuestos de Q5a, C4b-2, C4b-3 y C4c son los fijados en
   este documento.

El límite local del adaptador para C4c pasa de 580 a 600 líneas: las veinte
líneas adicionales se asignan exclusivamente a duplicados estables y deadlines
del protocolo `coproc`, no a alcance funcional. DEC-051 permanece en 800.

Permanecen vigentes la semántica C4b-1 de primera señal observada, oráculo,
causales `0/64/65/79`, topología H0b, D2c, D2d, capturador, propiedad exacta de
Docker y ficheros y las demás puertas no sustituidas expresamente.

## Hechos técnicos acreditados

El corte estable `91cb804` conserva esta base física:

| Fichero | Líneas | SHA-256 |
| --- | ---: | --- |
| Runner | 769 | se recalcula en cada candidato |
| Adaptador privado de ciclo | 527 | `d9b61a183e5a32c321a3eeb48483ce40c83551bc7a700354ccc88e8206d9ee1f` |
| D2c | inmutable | `a07057fb15315c5d2d0d10d6f3beea85f196fc78598cfcc4d1f63918bcbadde5` |
| H0b | 580 e inmutable | `02a00f2fc49e181d1cf8ed147a927155899956dbdbd7f36f3443ee4d7cbafded` |
| D2d | inmutable | `8281ac2fe10a2c4609bfb7a87f68f69a1e71189d0d7a3ed946af231b866e2075` |
| Capturador | inmutable | `4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902` |

La versión local acreditada es Go 1.26.5, Linux/amd64, y el host de prueba usa
Linux 7.0. `syscall.SysProcAttr.PidFD` solicita `CLONE_PIDFD`; Go puede usar
`clone` o `clone3`, por lo que la especificación exige la propiedad atómica,
no un syscall concreto.

Se ejecutaron dos sondas desechables, fuera del repositorio y ya retiradas:

1. creación con `Setpgid`, `PidFD` y señal cero, STOP, CONT y TERM mediante
   `PIDFD_SIGNAL_PROCESS_GROUP`: verde;
2. líder esperado y recolectado con un descendiente todavía vivo, seguido de
   señal cero y KILL de grupo mediante el pidfd original: verde y sin residuo.

Estas sondas solo demuestran viabilidad en el host. No sustituyen la autoprueba
capturada, sus mutantes ni las puertas de integración.

`/proc/PID/stat` sigue aportando estado, PPID, PGID, SID e inicio para
evidencia, pero no es autoridad de señalización. Mantener abierto
`/proc/PID` no impide reciclar el número. Las fronteras adoptadas son
`CLONE_PIDFD` y `PIDFD_SIGNAL_PROCESS_GROUP`, disponible desde Linux 6.9.
Referencias: [pidfd_send_signal(2)](https://man7.org/linux/man-pages/man2/pidfd_send_signal.2.html),
[proc_pid_stat(5)](https://man7.org/linux/man-pages/man5/proc_pid_stat.5.html),
[procfs](https://docs.kernel.org/filesystems/proc.html) y
[SysProcAttr de Go](https://go.dev/src/syscall/exec_linux.go).

## Alternativa adoptada

Se incorpora una sola fuente Go nueva:

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go
```

El runner captura esa fuente junto a las cuatro Shell, la compila desde la
copia privada y ejecuta únicamente el binario privado. La fuente viva no vuelve
a abrirse.

El supervisor no es lateral:

```text
runner Bash
└── supervisor Go               [único hijo esperado del runner]
    └── runner Bash --caso...   [único Bash de caso; grupo propio]
        └── clientes del caso   [mismo PGID y sesión]
```

La pérdida de paternidad directa runner/Bash es la excepción estricta necesaria
para que el proceso que crea al Bash reciba su pidfd en el mismo `clone`. El
supervisor no interpreta oráculo, causal funcional, `RESULTADO`, SQL, Docker o
RRHH. Solo crea y supervisa el árbol, transporta un ticket opaco, aplica plazo,
señaliza, espera, recolecta y emite un recibo mecánico.

Quedan prohibidos:

- `kill(PID)`, `kill(-PGID)`, `Process.Signal`, `Process.Kill`,
  `syscall.Kill`, `pkill`, `killall`, `setsid`, `nohup` o `disown` para el caso;
- `exec.CommandContext`, `Cmd.Cancel` y una goroutine que invoque `Cmd.Wait`;
- abrir pidfd después de crear al Bash como mecanismo nominal;
- degradar a PID/PGID numérico si falla pidfd o el flag de grupo;
- permitir que un descendiente cambie de PGID o sesión;
- dependencias Go externas, CGO, red, módulos descargados o un sexto fuente.

## Cinco fuentes y composición cerrada

La operación de captura contiene exactamente:

| Orden de carga/compilación | Fuente | Tratamiento |
| ---: | --- | --- |
| 1 | D2c `arnes_fuente_corporativa_contexto_actor_v1.sh` | `source` privado |
| 2 | D2d `operaciones_runner_fuente_corporativa_contexto_actor_v1.sh` | `source` privado |
| 3 | H0b `arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh` | `source` privado |
| 4 | `ciclo_recursos_m38_h0b_fuente_corporativa_contexto_actor_v1.sh` | `source` privado |
| 5 | `supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go` | `vet`, compilación y autoprueba |

El manifiesto se ordena canónicamente por ruta y se compara con cinco pares
literales ruta/SHA antes de cargar o compilar. El orden del manifiesto no
decide dependencias. Las cuatro fuentes Shell directas devuelven 64. Cada
`source` consume una marca nueva `VEC_F0_CARGA_PRIVADA=1`.

La fuente Go contiene `//go:build ignore && linux && amd64`: el tag convencional
`ignore` impide que el ejecutable probatorio colisione con el `main` del
capturador durante `go test ./...`, mientras la compilación por nombre de
fichero ignora la restricción y conserva la frontera cerrada del runner. Fija
el número ABI amd64 de `pidfd_send_signal` y
`PIDFD_SIGNAL_PROCESS_GROUP = 1 << 2`. Se valida desde la copia privada con
`go vet` y se compila por fichero con `GOOS=linux`, `GOARCH=amd64`,
`CGO_ENABLED=0`, `GOTOOLCHAIN=local`, `GOWORK=off`, `GOPROXY=off`,
`GOSUMDB=off`, `GONOSUMDB=*`, `GOFLAGS=-mod=readonly` y `-trimpath`. El binario
queda 0700 dentro del temporal acreditado.

Q5a incorpora solo `--autoprueba`: se aísla como líder de su propio grupo sin
crear hijos, abre pidfd sobre su propio PID, exige señal cero con el flag de
grupo, comprueba errores cerrados y cierra el descriptor. No crea
`ruta_caso_m38`, FIFO, canal, hijo, Docker o SQL. Cualquier otro modo devuelve
64. Con ABI disponible conserva H0; ausencia, `EINVAL`, `ENOSYS`, `EPERM` u
otro error devuelve 65 antes de recursos de caso y no acredita H0.

C4b-2 amplía esa misma autoprueba, antes del primer caso, con los grupos
sintéticos detallados más adelante. Q5a no pretende acreditar todavía
`CLONE_PIDFD`, proceso, plazo o extinción operativa.

## Temporales y handoff

Se mantienen separados:

1. `temporales`: raíz global de captura, compilación, manifiestos y matriz;
2. `ruta_caso_m38`: temporal exclusivo de un caso, creado después de cargar y
   autoprobar las cinco fuentes.

Antes del handoff no existe identidad de caso, canal operativo, supervisor,
Docker o SQL. El bootstrap solo retira su raíz global acreditada. Tras la marca
irreversible `bootstrap -> ciclo`, la trampa llama una vez al epílogo del
adaptador; no invoca directamente Docker, D2d o borrado de caso.

La identidad del caso se arma antes de crear su temporal. El adaptador crea y
acredita los canales bajo ese temporal antes de iniciar el supervisor. Un fallo
anterior al Bash no deja hijo de caso y converge en el mismo epílogo.

## Responsabilidades

### Runner

Conserva parser, selector, catálogo, oráculo, causales, ticket funcional,
`RESULTADO`, finalizador interior F01..F15, trampa C4b-1, captura de fuentes y
decisión de detener la matriz. Tras el handoff:

- construye el ticket sin PPID; el supervisor antepone su propio PID;
- ordena al adaptador lanzar y esperar un único supervisor;
- espera indirectamente el Bash mediante el recibo, nunca con `wait` sobre él;
- interpreta la salida solo después de un recibo terminal válido;
- invoca una vez el epílogo exterior.

### Adaptador privado

Conserva régimen Shell, canales anónimos, temporal, Docker, transporte por FD,
material H0b y el único `retirar_recursos_m38_f0`. Ya no crea, señaliza o espera Bash,
ni guarda su `$!` o PGID. Antes de lanzar el supervisor acredita `jobs` vacío,
arma `supervisor_provisional` y mantiene la sección crítica C4b-1. Lanza como
orden simple al supervisor y copia `$!` inmediatamente.

Una señal o fallo durante `fork -> $!` solo enclava C4b-1: no inicia epílogo,
Bash de caso ni otro trabajo. Al reanudar, el adaptador recupera por cardinalidad
exacta. Cero acredita que no existe supervisor; uno se registra como candidato;
más de uno es incidente 65 y no se escoge PID. El PID del candidato sirve para
`wait -f` y evidencia de hijo directo, nunca para señalizar.

El `coproc` crea los dos pipes anónimos antes de su fork interno. La primera
acción controlada del Go operativo es esperar `ARMAR`; antes de recibirlo no
valida el caso, crea Bash ni termina por un error controlado. Inmediatamente
después de recuperar el candidato, el adaptador duplica los dos FDs publicados
por `coproc` a FDs ordinarios mediante `exec {control_estable}>&...` y
`exec {recibo_estable}<&...`, acredita dirección/identidad y cierra los
originales. Desde entonces no vuelve a consultar el array ni el PID variable
del `coproc`.

Solo con ambos duplicados estables envía `ARMAR`. Go responde `ACK_LISTO` o un
recibo de error y ya puede terminar. El candidato se convierte en
`supervisor_capturado` únicamente si su trabajo directo, ejecutable privado,
nonce y ACK coinciden. Entonces el adaptador drena la señal pendiente y envía
exactamente uno de `INICIAR` o `CANCELAR`. En `INICIAR`, Go debe emitir además
`ACK_CASO` después de pidfd doble, barrera FD 9, STOP, deadline y CONT. Toda
lectura posterior, incluido el recibo terminal anterior al `wait` acreditado,
usa exclusivamente `recibo_estable`.

El padre fija con reloj monotónico antes del `coproc` cuatro límites absolutos
no reiniciables: `fin_armado = t0+2 s`, `fin_ack = t0+4 s`,
`fin_orden = t0+6 s` y `fin_bootstrap = fin_orden`. Lee ACK/recibos con espera
Shell acotada contra `/proc/uptime`, no con un `read` indefinido. Go fija al
entrar su propio `fin_bootstrap_go = ahora+6 s`; autoexpira sin Bash si no
recibe a tiempo ARMAR y después INICIAR/CANCELAR, y debe emitir `ACK_CASO`
antes de ese límite. Si el padre agota un límite, cierra control; solo llama
`wait -f` si acredita antes que el trabajo ya es terminal. Un supervisor vivo
o inerte activa incidente externo y cuarentena sin espera bloqueante.

Tras `ACK_CASO`, el padre fija `fin_supervision = ahora_monotónico + 190 s`
(180 de caso, nueve de limpieza y uno de salida). El recibo terminal se lee por
el FD estable con ese límite. Después del recibo concede como máximo un segundo
para acreditar terminalidad del trabajo y solo entonces ejecuta `wait -f`. Un
recibo ausente, un Go vivo tras el límite o un Go que escribe y no termina son
frontera externa: no se espera de forma bloqueante ni se reutiliza el worker.

En caminos controlados, el epílogo exige supervisor recolectado y recibo
coherente antes de declarar postausencia. Si falta el recibo por una frontera
externa, registra incidente, preserva temporal y raíz como cuarentena y no
desarma ni declara limpia la identidad de proceso dudosa. Puede inspeccionar o
retirar el contenedor solo por su identidad independiente exacta, pero el
worker sigue prohibido hasta reprovisión. D2d permanece inmutable y
subordinado: solo el adaptador invoca `retirar_contenedor_propio_f0`.

### Supervisor Go

Es el único propietario de:

- `PR_SET_PDEATHSIG` respecto del runner y `PR_SET_CHILD_SUBREAPER`;
- `exec.Cmd`, pidfd, grupo del caso y estado terminal;
- plazo monotónico de 180 segundos y gracia monotónica de dos segundos;
- señalización de grupo ligada, espera única y drenaje de adoptados;
- recibo mecánico final.

Tras ARMAR fija el hilo, instala el receptor Go de SIGTERM, ejecuta literalmente
`prctl(PR_SET_PDEATHSIG, SIGTERM)`, vuelve a comprobar el PPID recibido por FD
confiable y comprueba tanto señal ya encolada como EOF de control antes de
crear al Bash. Así cierra la carrera en la que el runner muere antes del
`prctl`. Canal, EOF o muerte del runner se traducen a una única cancelación;
nunca inicia una segunda.

## Mapa cerrado de descriptores

El adaptador entrega al supervisor solo descriptores ya abiertos:

| FD en Go | Contenido |
| ---: | --- |
| 3 | raíz acreditada que el Bash recibirá como FD 7 |
| 4 | copia privada del runner que el Bash recibirá como FD 8 |
| 5 | salida del caso |
| 6 | error del caso |
| 7 | pipe de control de `coproc`, solo lectura en Go |
| 8 | pipe de recibo de `coproc`, solo escritura en Go |
| 9 | sobre inicial opaco para Go; no es el FD 9 posterior del Bash |

Para el Bash, `exec.Cmd.ExtraFiles` asigna desde FD 3. El supervisor rellena
3..6 con descriptores de `/dev/null`, entrega raíz en 7, runner en 8 y un pipe
anónimo todavía vacío en 9. El Bash queda bloqueado al leer ese FD antes de
STOP y antes de cualquier efecto. Solo después de validar y duplicar el pidfd,
Go escribe `PID_supervisor|sobre`; si cierra el pipe, Bash recibe EOF y sale
sin STOP, descendientes ni efectos. `Stdout` y `Stderr` usan los FD 5 y 6. Go
no abre una ruta viva del repositorio.

El adaptador usa un `coproc` nombrado cuyo cuerpo hace `exec` del binario
privado: no crea FIFO, socket con nombre, keeper ni proceso auxiliar estable.
Los dos pipes anónimos son unidireccionales; el adaptador conserva escritura de
control y lectura de recibo mediante duplicados ordinarios estables, mientras
Go recibe únicamente los extremos opuestos. Los FDs del array `coproc` pueden
ser desasignados por Bash y nunca se guardan solo como números para usarlos
después. EOF/HUP conserva significado. Las redirecciones copian raíz/runner a
3/4 y control/recibo a 7/8 antes de cerrar los originales heredados. Una puerta
acredita que el cuerpo fue reemplazado por el ejecutable Go y que ningún FD de
control del adaptador llega al Bash. El protocolo limita versión, dirección,
longitud, cardinalidad y salto final.

## Comando exacto del Bash

El supervisor no usa `LookPath`, `PATH` para resolver el ejecutable ni una ruta
recibida. Tras acreditar físicamente `/usr/bin/bash`, construye exactamente:

```text
Cmd.Path = /usr/bin/bash
Cmd.Args = [/usr/bin/bash, /proc/self/fd/8,
            --caso-inyeccion-h0b, SELECTOR_LITERAL]
```

El entorno del Bash se construye mediante lista permitida, no se hereda en
bloque. Contiene `LC_ALL=C`, un `PATH` literal de herramientas del sistema y,
solo si fue validado, el digest `VEC_POSTGRES_TEST_IMAGE`. Ninguna variable
puede introducir `BASH_ENV`, `ENV`, opción Shell, ruta del repositorio o
configuración Go. FD 3, 4 y 9 del supervisor son las únicas raíz, fuente runner
y envoltorio admitidos; el Bash recibe únicamente sus duplicados 7, 8 y 9.

## Máquina Shell del supervisor directo

La máquina exterior precede a la máquina Go:

```text
ninguno -> supervisor_provisional -> cardinal_0 | candidato_unico | incidente
candidato_unico -> FD_estables -> ARMAR -> ACK_LISTO
ACK_LISTO -> INICIAR -> ACK_CASO -> esperado -> recolectado
          -> CANCELAR -> recibido_sin_Bash -> recolectado
```

- `supervisor_provisional` se arma antes del fork y exige tabla vacía;
- entre fork y `$!` no hay retorno ni efecto distinto del latch de señal;
- cardinalidad cero cierra los extremos `coproc` y acredita ausencia;
- cardinalidad uno conserva el hijo directo hasta `wait -f` aunque ya hubiera
  terminado; el estado cacheado del trabajo no se confunde con reciclado de PID;
- cardinalidad mayor que uno no elige ni señaliza por número, cierra el
  protocolo, devuelve 65 y espera/recolecta la tabla completa sin iniciar Bash;
  un trabajo ajeno que no termina convierte el caso en incidente operativo, no
  habilita un borrado o señalización ambiguos ni permite el caso siguiente;
- los duplicados estables preceden a ARMAR y sobreviven a que Bash desasigne el
  array del coproceso;
- Go no puede salir por un error controlado antes de ARMAR, ni crear al Bash
  antes de ACK_LISTO e INICIAR;
- cada transición se completa antes de su deadline; ACK omitido o proceso
  colgado nunca lleva a `read` o `wait -f` indefinidos.

## Máquina de estados del proceso

Una sola goroutine, fijada a un solo hilo de sistema, es dueña de `Start`,
pidfd, señalización terminal, `Wait`, cierre y recibo:

```text
arranque -> esperando_ARMAR -> listo -> esperando_orden
ninguno -> provisional -> capturado -> acreditado -> ejecutando
         -> terminando -> recolectado -> postausente -> recibido
```

1. Tras reconocer el modo operativo fijo, arma `fin_bootstrap_go`, espera
   `ARMAR` y no ejecuta otra transición controlada. EOF o timeout termina sin Bash.
2. Ejecuta `runtime.LockOSThread()` al aceptar ARMAR y antes de construir
   `exec.Cmd`; no libera
   el hilo hasta haber esperado Bash/adoptados, cerrado pidfd y emitido recibo.
3. Instala señal/prctl, valida PPID/FD/nonce, emite ACK_LISTO y espera una única
   orden antes de `fin_bootstrap_go`; CANCELAR termina sin Bash.
4. Arma `provisional` antes de `cmd.Start()` con `Setpgid: true`, `Pgid: 0`,
   `PidFD: &fd` y `Pdeathsig: SIGKILL`.
5. Error de `Start` significa que no existe hijo. En éxito, el Bash permanece
   bloqueado por el pipe de ticket. Go exige `fd >= 0` y crea inmediatamente
   `fd_reserva = fcntl(fd, F_DUPFD_CLOEXEC, ...)`.
6. Si falta el pidfd o su duplicado, cierra el pipe: Bash sale por EOF antes de
   STOP y Go acredita terminación con plazo bootstrap antes de llamar a
   `cmd.Wait`. Solo dos referencias válidas permiten escribir el ticket y
   transferir atómicamente a `capturado`.
7. Revalida por evidencia `PID|T|PPID|PGID=PID|SID|starttime`, además de señal
   cero de grupo por pidfd. `/proc` no autoriza ninguna señal.
8. Fija `fin_caso = time.Now().Add(180*time.Second)` antes del primer CONT; el
   valor conserva su componente monotónico y nunca se reinicia.
9. Emite ACK_CASO y un único bucle de eventos observa control, pidfd y tiempo.
   No existe un
   `cmd.Wait()` concurrente.
10. Todo error controlado posterior a `Start` entra en `terminando`; un `defer`
   de último recurso repite esa convergencia y nunca devuelve dejando una vía
   controlada sin `Wait`.

`fd` y `fd_reserva` representan el mismo objeto kernel. Un `EBADF` del primario
permite una única promoción del duplicado; no abre pidfd por PID. Un fallo
persistente de ambas referencias o del syscall de grupo es frontera externa
65 y activa cuarentena, nunca fallback numérico.

`Pdeathsig` del Bash no se usa para timeout, cancelación o error normal. Solo
reduce el daño de una muerte abrupta del supervisor. Fijar el hilo evita que
una retirada ordinaria de un hilo del runtime dispare la señal mientras el
supervisor sigue vivo.

## Secuencia terminal única

La terminación normal observa al líder sin recogerlo prematuramente. Si no hay
otro miembro, ejecuta `cmd.Wait()` exactamente una vez, drena adoptados, exige
grupo ausente y conserva el estado real. Si hay descendientes, usa la misma
secuencia de extinción que cancelación o plazo.

Para cancelar o extinguir:

1. enclava una causa única y conserva el deadline original;
2. fija `fin_parada = ahora_monotónico + 1 s`, envía STOP al grupo mediante el
   pidfd y acredita inventario estable detenido;
3. si STOP queda estable, fija `fin_gracia = ahora_monotónico + 2 s`, envía
   TERM y seguidamente CONT por el mismo pidfd y observa sin `Wait`;
4. al vencer la gracia fija `fin_parada_final = ahora_monotónico + 1 s`, vuelve
   a detener/estabilizar y envía KILL si persiste cualquier miembro;
5. esa rama fija `fin_drenaje = fin_parada_final + 5 s`;
6. si el primer STOP no estabiliza antes de `fin_parada`, omite por completo
   TERM, CONT, gracia y parada final: envía KILL inmediato y fija
   `fin_drenaje = fin_parada + 5 s`;
7. ambas ramas convergen en un `fin_drenaje` ya inicializado y esperan
   terminalidad mediante `poll(pidfd)` o
   `waitid(P_PIDFD, WEXITED|WNOHANG|WNOWAIT)`, sin recoger todavía;
8. solo con terminalidad acreditada ejecuta `cmd.Wait` una vez y drena con
   `wait4(..., WNOHANG)` los adoptados hasta `ECHILD`, siempre antes de
   `fin_drenaje`;
9. exige que señal cero de grupo por pidfd devuelva `ESRCH`, que no haya hijo
   adoptado ni miembro del PGID y solo entonces cierra pidfd;
10. escribe exactamente un recibo, cierra canales y libera el hilo.

El plazo funcional continúa siendo 180 segundos desde CONT. La rama cooperativa
puede añadir como máximo nueve segundos monotónicos: una parada inicial, dos de
gracia, una parada final y cinco de drenaje. La rama STOP fallida añade como
máximo seis. Ningún subplazo se reinicia. Si
vence `fin_drenaje`, no se llama a `cmd.Wait`: se declara frontera externa 65,
se emite recibo de incidente si aún es posible y se activa cuarentena.

El pidfd permanece abierto hasta la última señal y postcondición, incluso si
el líder es zombi o ya fue recogido. `EBADF`, `EINVAL` o `EPERM` son incidente
65. `ESRCH` solo acredita éxito después de la postcondición completa. No hay
fallback numérico.

`PR_SET_CHILD_SUBREAPER` se activa y comprueba antes de crear al Bash. Ningún
descendiente puede salir del PGID o sesión; las fuentes y el comportamiento se
auditan para rechazar `setsid`, `setpgid` o daemonización. Sin esa precondición
la arquitectura no afirma extinción completa.

## Finalización funcional y exterior

En el camino natural:

```text
Bash -> finalizar_h0b_f0 una vez -> RESULTADO -> salida
     -> supervisor acredita/recolecta -> recibo
     -> runner interpreta -> retirar_recursos_m38_f0 una vez
     -> D2d -> temporal
```

En señal, plazo o error, el Bash puede alcanzar su trampa/finalizador durante
TERM. Si KILL lo impide, el recibo declara que no existe resultado funcional
acreditable; el runner no inventa ni repite el finalizador, devuelve la causal
de señal o 65 y ejecuta solo el epílogo exterior. Go nunca llama al finalizador
interior, Docker o SQL.

## Límite explícito de garantía en proceso

Todos los errores controlados, cancelaciones, plazos y fallos inyectables de
protocolo deben dejar recibo, Bash/adoptados recolectados y residuos cero.

Como ya fijaba la enmienda anterior, SIGKILL directo al supervisor, caída del
host, fallo permanente del kernel tras una autoprueba verde o actuación de un
administrador Docker quedan fuera de la garantía en proceso. En esos casos:

- nunca se emite recibo verde ni se lanza el caso siguiente;
- el runner devuelve incidente 65 si sigue vivo;
- `Pdeathsig` extingue al líder, pero no se presenta falsamente como garantía
  de extinción de todo el grupo;
- temporal y raíz se preservan como cuarentena;
- evidencia y recuperación siguen obligatoriamente el
  [procedimiento de supervisor irrecuperable](procedimiento_operativo_f0_h0b_supervisor_irrecuperable_2026-08-04.md).

Esta excepción no cubre muerte o migración normal del hilo Go: queda impedida
por `LockOSThread`. Tampoco se usa para debilitar ningún mutante controlado.

## Presupuesto físico reproducible

Todas las cifras son líneas físicas `wc -l`. Cada candidato adjunta
`git diff --numstat`, rangos retirados y añadidos y el nuevo total.

En el adaptador base se han contado exactamente:

| Fragmento actual | Rango base | Acción C4b-2 |
| --- | ---: | ---: |
| `lanzar_hijo_m38_f0` | 132..147 = 16 | retirar 16 |
| `esperar_hijo_con_plazo_m38_f0` | 149..158 = 10 | retirar 10 |
| proceso en `retirar_recursos_m38_f0` | 162..167 = 6 | retirar 6 |
| **Total** | **32** | **-32** |
| `esperar_terminal_m38_f0` | 57..64 = 8 | conservar |
| `esperar_cliente_m38_f0` | 65..94 = 30 | conservar para Docker |

Ledger conservador:

| Corte y ecuación | Runner | Adaptador | Supervisor Go |
| --- | ---: | ---: | ---: |
| Base `91cb804` | 769 | 527 | 0 |
| Q5a: reemplazar función de captura de 39 por 39..40 y añadir 3 literales | 772..773 | 527 exactas | 130..190 |
| C4b-2: sustituciones sin crecimiento material; `527-32+(48..60)` | 772..773 | 543..555 | 330..450 |
| C4b-3: solo Docker/epílogo; `anterior+(8..14)` | 772..774 | 551..569 | 365..490 |
| Reserva hasta límites C4b/C4c | 1..3 hasta 775 | 31..49 hasta 600 | 10..135 hasta 500 |

Las 48..60 líneas nuevas de C4b-2 se presupuestan de forma explícita:

| Puente definitivo C4b-2 | Mínimo | Máximo |
| --- | ---: | ---: |
| `coproc`, redirecciones, duplicados estables y validación | 12 | 16 |
| provisional, lanzamiento y recuperación cardinal | 10 | 13 |
| reloj monotónico, ACK, órdenes, espera y recibo | 18 | 22 |
| cierre/postcondiciones comunes | 8 | 9 |
| **Total añadido** | **48** | **60** |

Q5a se detiene si runner supera 773, cambia el adaptador o Go supera 190.
C4b-2 se detiene si runner supera 773, adaptador supera 555 o Go supera 450.
C4b-3 se detiene si runner supera 774, adaptador supera 569 o Go supera 490.
C4c no toca Go y conserva al menos 31 líneas del adaptador. Un candidato que
exceda un checkpoint requiere otra decisión; no puede comprimir sentencias,
retirar controles o contar el límite como reserva.

## Write-set por minitarea

| Minitarea | Write-set productivo/probatorio |
| --- | --- |
| Q5a | runner + nueva fuente Go; adaptador, H0b, D2c, D2d y capturador invariantes |
| C4b-2 | runner solo para huella/protocolo + adaptador + Go |
| C4b-3 | runner solo para huella/protocolo + adaptador + Go |
| C4c | runner + adaptador; Go, H0b, D2c, D2d y capturador invariantes |

Q5a no retira todavía las funciones de proceso del adaptador; esa retirada
pertenece a C4b-2. Productor y revisor son distintos y no comparten edición.
C4b-2 entrega desde el primer candidato los canales definitivos completos:
`coproc`, extremos anónimos, FD, ACK, orden, recibo y cierres. C4b-3 no contiene
un canal provisional ni cambia ese protocolo.

## Autopruebas y mutantes obligatorios

### Captura y ABI

- cardinalidad 4/5/6, duplicado, quinta ausente, ruta o huella mutada;
- ruta viva modificada después de captura y copia sustituida antes de build;
- binario sustituido, modo no privado, build con CGO/red o autoprueba omitida;
- cuatro marcas Shell ausentes/reutilizadas y modo Go desconocido: 64;
- pidfd self, señal cero con flag, `EINVAL`, `ENOSYS`, `EPERM` y `EBADF`;
- todo fallo anterior a C4b-2 deja solo la raíz global, después retirada.

### Autoprueba operativa previa a casos

La misma fuente crea y limpia, sin Docker ni SQL:

1. cada líder y ayudante nace con `PidFD` obtenido por su mismo `Start`; la
   autoprueba conserva además cada pidfd individual para rollback sintético;
2. grupo cooperativo de líder y miembro: señal cero, STOP realmente observado,
   CONT y TERM con ambas terminaciones;
3. grupo con miembro que ignora TERM: STOP, TERM, CONT, gracia y KILL;
4. líder que termina antes, se recoge, descendiente vivo y actuación de grupo
   mediante el pidfd original;
5. señuelo en otro grupo que permanece intacto;
6. `ESRCH` terminal, pidfd cerrado/`EBADF`, flag adverso/`EINVAL` y cero hijos,
   grupos o descriptores al salir, incluso si una comprobación falla.

La autoprueba acredita conducta, no solo versión o cabecera. Si seccomp, kernel
o permisos impiden cualquier paso, devuelve 65 antes de identidad de caso,
Docker o SQL. El rollback puede usar los pidfds individuales solo sobre sus
procesos sintéticos conocidos; esa vía no existe ni se permite en el caso
operativo.

### Proceso operativo

- señal/fallo antes del fork del supervisor, durante `fork -> $!`, después de
  `$!`, cardinalidad 0/1/>1, candidato distinto y supervisor ya terminado;
- coproceso que intenta terminar antes de ARMAR, variables `coproc` desasignadas
  antes de `wait` y recibo aún legible por el duplicado estable después de él;
- ARMAR omitido/colgado, ACK_LISTO omitido/colgado, orden omitida/colgada y
  ACK_CASO omitido/colgado, agotando cada deadline sin espera bloqueante;
- señal antes/después de ACK, `INICIAR` omitido, cancelación antes de iniciar y
  comprobación de que no nació Bash ni se lanzó el caso siguiente;
- error `Start` sin hijo y éxito con pidfd negativo prohibido;
- cierre del pipe-ticket con EOF antes de STOP, fallo de `F_DUPFD_CLOEXEC`,
  `EBADF` primario con promoción única y fallo persistente de ambas referencias;
- presión y replanificación del runtime con hilo fijado, sin `Pdeathsig` espurio;
- Bash que no alcanza STOP, tupla truncada/adversa y grupo con miembro nuevo;
- cancelación antes/después de ACK, duplicada y simultánea con muerte paterna;
- fin en el borde de 180 s, reloj civil alterado y cien interrupciones Shell;
- TERM cooperativo, TERM ignorado, líder muerto durante gracia y KILL;
- STOP inicial estable y STOP inicial no estabilizable; ambas ramas crean su
  `fin_drenaje` exacto y no leen una variable de la otra;
- proceso KILL-pendiente, `fin_drenaje` agotado y prueba de que no se invocó
  `cmd.Wait` bloqueante ni se reutilizó el worker;
- `waitid` sin `WEXITED` debe fallar la puerta; la llamada admitida combina
  exactamente `WEXITED|WNOHANG|WNOWAIT` antes del único `cmd.Wait`;
- salida normal con descendiente, líder zombi y líder ya recogido;
- `wait -f` del supervisor interrumpido; estado real 0/64/65/79/130/143;
- recibo ausente, duplicado, truncado, de versión o identidad adversa, y Go
  que escribe el recibo pero no termina antes de `fin_supervision`;
- intento de `setsid`, `setpgid`, daemonizar o salir del grupo/sesión;
- fallo transitorio de señal con reintento acotado y fallo permanente como
  incidente sin falso verde ni caso siguiente;
- epílogo reentrante, canal roto y error controlado en cada transición.

Cada mutante controlado acredita supervisor y Bash recolectados, adoptados
cero, pidfd y canales cerrados, PGID sin miembros, temporal de caso retirado,
contenedor ausente y ningún siguiente caso. Los mutantes que modelan la frontera
externa prueban 65/no verde y generan la evidencia operativa correspondiente,
no una postausencia ficticia.

## Puertas

Además de Bash, ShellCheck, Go, `git diff --check`, Gitleaks, tamaños, H0
PostgreSQL 18.4 y C4b-1, se exige:

- manifiesto exacto de cinco sin reapertura viva;
- build Go cerrado, ABI básica y autoprueba operativa completas;
- búsqueda negativa de todas las APIs PID/PGID y esperas prohibidas;
- un único propietario de `cmd.Wait`, pidfd, cierre y recibo;
- call graph único de finalizador funcional y epílogo exterior;
- D2d invocado solo por el adaptador; runner y Go no ejecutan `docker rm`;
- hashes invariantes de H0b, D2c, D2d y capturador;
- SHA de Go fijada tras C4b e inmutable en C4c;
- ledger físico y matriz de mutantes con ejecución positiva y residuos cero;
- doble revisión funcional y de seguridad independiente.

## Secuencia autorizable

1. Dos contrarrevisiones documentales de esta cuarta corrección.
2. Q5a: quinta captura, build cerrado y autoprueba ABI sin recursos de caso.
3. Revisión independiente de Q5a.
4. C4b-2: supervisor padre, CLONE_PIDFD, grupo, plazo, espera y mutantes sin
   modificar todavía la reconciliación Docker de C4b-3.
5. Doble revisión independiente de C4b-2.
6. C4b-3: reconciliación Docker y epílogo exterior único; no rediseña canales.
7. C4c y C4d según el DAG vigente.

Ningún checkpoint intermedio aumenta F0 `10/23`, O4-05 `3/5`, Contratación
temporal `24/46`, Bolsa productiva `1/14` o cambia el `NO-GO` de producción.

## Criterio de aprobación

Dos revisores distintos deben confirmar `P0=0`, `P1=0`, `P2=0` para:

1. snapshot único de cinco y build cerrado;
2. paternidad runner -> Go -> Bash y CLONE_PIDFD sin ventana;
3. señalización exclusiva de grupo por pidfd y autoprueba real;
4. hilo fijado, subreaper y una sola máquina terminal;
5. finalizador funcional y epílogo exterior únicos;
6. temporales, canales y handoff sin autoridad concurrente;
7. ledger con reserva positiva y write-sets por fase;
8. conservación de C4b-1, H0b, D2c, D2d y capturador.

Un solo `NO-GO` mantiene detenido el código y exige una nueva corrección.
