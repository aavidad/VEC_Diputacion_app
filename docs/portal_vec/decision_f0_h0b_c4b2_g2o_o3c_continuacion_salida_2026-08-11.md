# Decisión F0-H0b/C4b-2/G2-O/O3c: continuación y primera observación

Fecha: 11 de agosto de 2026.

Identificador: `O3C-P0-CONTRATO`.

Estado: **CERRADA Y PUBLICADA EN SU RAMA EXCLUSIVA**. No autoriza integración,
ejecución operativa ni producción. Las revisiones funcional y de seguridad
independientes cerraron `P0=P1=P2=0`; el commit material `2761eca` se publicó
sin force y su CI `31454901886` terminó 5/5. Solo dirección puede asignar el
código O3c y decidir una integración posterior.

## Resultado observable

O3c consume exactamente una vez el agregado opaco capturado por O3b. Tras una
última revalidación conjunta, crea una sola marca monotónica de caso de 180
segundos inmediatamente antes del primer y único `CONT` de grupo, emite ese
`CONT` mediante un pidfd explícito y realiza una observación no recolectora
inmediata. Entrega a O4a un agregado opaco que conserva toda la custodia y una
primera observación raw inmutable de unión cerrada.

En la ruta positiva y desde el intento CONT, O3c no espera hasta el
vencimiento, no decide la causa primaria, no envía STOP/TERM/KILL, no recoge
el Bash, no drena adoptados, no escribe TERMINAL y no libera recursos. Esas
responsabilidades empiezan en O4a/O4b/O4c. La única excepción es la retirada
propia anterior a CONT, dominada por C7: puede emitir un SIGKILL individual,
ejecutar un Wait tras terminalidad y drenar solo adoptados ya terminales.

No existen lectores Go de salida/error pendientes en el handoff: O3a entregó
al Bash las copias destinadas a sus stdout/stderr y cerró en el padre tanto
los originales como sus copias parentales antes del handoff. El Bash conserva
sus FD heredados hasta salir. La mención genealógica a «drenar salida/error» en
G2-O no crea un owner ni una capacidad Go; queda sustituida. O4c drena solo
hijos adoptados y acredita `ECHILD`, sin leer esos streams.

Antes de `CONT`, un fallo controlable usa una retirada propietaria O3c con las
primitivas y semántica de retirada O3a ya autorizadas; nunca invoca la API O3b
cuya B5 fue consumida. Desde que se intenta `CONT`, el efecto es irreversible:
cualquier evento se entrega a O4a; si la propiedad ya no puede entregarse
completa, domina fatalidad 65 sin retorno, log, cierre ni otra E/S.

## Base, prevalencia y genealogía

La base exacta es `124aa60e19d8daf31906098d4d60b4a4d2c6e281`, publicada en
`trabajo/o3b-p8-evidencia-v2-20260811`. Su CI
[31452048607](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31452048607)
terminó `completed/success`, `attempt=1`, con cinco de cinco jobs verdes.

Prevalecen, por responsabilidad:

1. la [decisión O3a](decision_f0_h0b_c4b2_g2o_o3a_arranque_mapa_fd_2026-08-09.md)
   para arranque, mapa FD, lease, observador y retirada compartida;
2. la [decisión O3b](decision_f0_h0b_c4b2_g2o_o3b_ticket_stop_identidad_2026-08-10.md)
   para el agregado de entrada, ticket cerrado, auto-STOP e identidad;
3. la [revisión final O3b](revisiones/revision_f0_h0b_c4b2_g2o_o3b_codigo_final_2026-08-11.md)
   para bytes, ledger y evidencia publicados;
4. esta decisión exclusivamente para O3c.

La [especificación conjunta G2-O](especificacion_f0_h0b_c4b2_g2o_protocolo_operativo_2026-08-05.md)
permanece `NO-GO` genealógico. No se recuperan sus ACK, recibos ni reparto de
responsabilidades donde O3a/O3b/O3c lo sustituyen.

## Alcance cerrado

O3c posee solamente:

- consumo lineal del handoff O3b B5;
- máquina privada y custodia entre O3b y O4a;
- última revalidación anterior al primer `CONT`;
- creación única de `finCasoO3cM38 = ahora_monotónico + 180s`;
- primer y único `CONT` por pidfd con flag de grupo;
- primera observación inmediata y no recolectora de salida natural;
- transferencia conjunta, opaca e inmediata a O4a;
- retirada pre-`CONT` y fatalidad cerrada cuando no existe propiedad segura.

O3c no posee:

- parser nuevo de CONTROL, ticket, `/proc`, selector o salida/error;
- otro `Start`, ticket, STOP, TERM, KILL, `waitid` o drenaje en la ruta
  positiva/post-CONT; la retirada C7 conserva solo el SIGKILL individual,
  Wait único y drenaje no bloqueante expresamente acotados;
- decisión de causa, vencimiento, gracia, reanudación posterior o TERMINAL;
- exposición de PID, PGID, pidfd, `Cmd`, `Process`, lease u observador;
- Orquesta, Firecracker, Jailer, Docker, SQL, datos RRHH o secretos reales;
- producción, despliegue, métricas, O4a, O4b, O4c, O5 u O6.

## Entrada única, consumo y autoridad

La única entrada productiva es `**agregadoO3cM38`. La operación copia el
puntero a custodia privada y pone a cero el puntero del llamador antes de
validar. El campo B5 publicado por O3b es un discriminante read-only, no una
autoridad atómica y no se modifica.

El consumo lineal usa las dos autoridades atómicas ya entregadas y sus estados
publicados. Tras exigir B5 y que los handles mínimos de observador/lease no sean
nulos, prevalida conjuntamente observador en estado 1, lease en estado 1,
mismo registro/generación/TID, baseline exacto y lease sin `pending`; entra en
un subestado de entrada no retornable, llama
`observador.transferirCritico(baseline)` —CAS 1→2— y después
`lease.transferirCritico()` —CAS 1→3—. Solo ambos CAS crean C0 y permiten
observar los restantes campos. Antes de ambos CAS preasigna una autoridad
privada `autoridadCustodiaO3cM38`, con autoidentidad y dos owners atómicos aún
inactivos; solo después fija conjuntamente ambos en O3C. Si el primer CAS
subyacente consolida y el segundo falla,
domina CF sin rollback, cierre, log, E/S ni retorno. Nulo, alias, clon, reuso,
autoidentidad rota o B5 distinto nunca ejecutan `CONT`; tras un primer consumo,
cualquier alias ve 2/3 en vez de 1/1 y retorna `uso_consumido` sin tocar
recursos. No se añade estado a los tipos O3b congelados. Los estados
subyacentes 2/3 nunca vuelven a 1/1, ni siquiera después de C5.

La entrada conserva conjuntamente:

- `Cmd` y `Process` con PID positivo y handle pidfd opaco;
- primario y reserva pidfd explícitos, distintos, `CLOEXEC` y de la misma
  identidad que el handle opaco;
- CONTROL lector limpio, TERMINAL escritor 0600 y ticket writer ausente;
- ningún lector o escritor Go de salida/error pendiente;
- lease y observador ya transferidos, mismo registro/generación/TID;
- baseline actualizado, signo cero, PPID y TID sellados;
- controlador S3/L0 y límites de lectura 1024/4/4096/8;
- `finBootstrapGoM38` monotónico todavía positivo;
- identidad `/proc` sellada: PID, T, PPID, PGID, SID y starttime;
- snapshot físico completo e inventario de exactamente tres referencias;
- primera observación O3b vacía e inmutable.

No se extrae el handle opaco, no se crea pidfd, no se duplica ni promueve una
referencia. Los enteros pidfd son capacidades prestadas bajo la lease. Toda
validación de registro, pertenencia, identidad, generación, TID y ownership
atómico falla cerrada. No se usa `unsafe`, registro global ni sidecar.

## Máquina total

Estados privados y no serializables:

```text
C0 recibido
  -> C1 revalidado_pre_cont | C7 retirando_pre_cont | CF fatal
C1 revalidado_pre_cont
  -> C2 cont_intentado | CF fatal
C2 cont_intentado
  -> C3 observacion_inmediata | CF fatal
C3 observacion_inmediata
  -> C4T transfiriendo_no_retornable | CF fatal
C4T transfiriendo_no_retornable
  -> C5 entregado_o4a | CF fatal
C7 retirando_pre_cont
  -> C8 retirado_pre_cont | CF fatal
```

No existe transición desde C5, C8 o CF. C2 significa que el syscall `CONT`
fue intentado, aunque devolviera error; desde C2 no se llama a la retirada
pre-`CONT` ni se repite el efecto. C4T no es entregable.

Resultados cerrados:

- `entregado_o4a`: estado C5, efecto CONT intentado una vez, deadline único y
  agregado O4a completo;
- `retirado_pre_cont`: estado C8, cero CONT, hijo recogido por retirada
  compartida y cero recurso de caso;
- `uso_consumido`: estado terminal sin tocar recursos ya transferidos;
- `CF`: `fatalO3aM38`, estado exterior 65 y EOF, sin retorno ni E/S posterior.

No existe `nil,nil`, entrega parcial, segundo CONT ni retorno con ownership
incierto.

## Última revalidación anterior a CONT

La ronda final preserva la precedencia autoritativa y ejecuta cada syscall
sobre un recurso poseído con un permiso lease distinto mediante los métodos
publicados que aceptan el estado transferido 3:
`lease.comenzar -> syscall -> lease.consolidarCritico`. No llama al método
`comenzarCritico`, que solo admite estado 1. Un permiso no se comparte,
reutiliza ni entrega pendiente.

Orden exacto:

1. CONTROL se lee hasta `EAGAIN` mediante el parser G4 ya existente; trama
   terminal, framing, EOF, parcial, presupuesto o `EINTR` agotado ganan;
2. observador: autoidentidad, registro, pertenencia, generación y TID; contador
   igual al baseline y signo cero;
3. `Gettid`, `Getppid` y `PR_GET_PDEATHSIG` coinciden con TID, PPID y
   `SIGTERM` sellados;
4. bootstrap tiene tiempo monotónicamente positivo; no se recrea ni reinicia;
5. primario y reserva se acreditan vivos con sondeo no recolector; identidad,
   flags o terminalidad divergente impiden el verde, y ambos no fiables son CF;
6. el handle opaco permanece dentro de `Process` y el inventario físico exige
   exactamente tres referencias de la misma identidad, sin cuarta;
7. CONTROL, TERMINAL, `Cmd`, `Process`, lease y observador coinciden con sus
   sellos; ticket writer continúa nulo; el lector/parser O3b ya existente
   relee `/proc/<PID>/stat` una vez y exige de nuevo T y la identidad completa
   PID/PPID/PGID/SID/starttime, sin abrir `/proc/<PID>` ni crear otro parser;
8. una segunda lectura CONTROL/observador/PPID/TID posterior al inventario
   debe seguir verde;
9. se preautoriza bajo lease un único permiso opaco para el primer CONT, sin
   syscall en su consolidación futura;
10. se preasigna el agregado O4a y todos sus contenedores antes de la lectura
    monotónica final;
11. una lectura monotónica final exige `ahora < finBootstrapGoM38`.

Cancelación o señal ya observable precede a bootstrap, pidfd e inventario. La
terminalidad natural anterior a CONT retira: O3c no reanuda un proceso ya
terminal. Cualquier error lease o propiedad no acreditable domina CF.

## Marca única de 180 segundos y primer CONT

Después de C1 no se ejecuta syscall, log, asignación falible ni E/S antes de
crear la marca. La operación de reloj monotónico que produce `ahoraCaso` es la
última operación anterior al efecto y fija una sola vez:

```text
finCasoO3cM38 = ahoraCaso + 180 segundos
```

Se rechaza overflow, componente monotónico ausente o marca no estrictamente
posterior. La marca no se adelanta, copia desde el exterior, recalcula,
extiende, pausa ni reinicia. Se conserva el `ahoraCaso` sellado para acreditar
la relación exacta de 180 segundos.

El siguiente syscall literal es exactamente:

```text
pidfd_send_signal(pidfdFiable, SIGCONT, NULL, PIDFD_SIGNAL_PROCESS_GROUP)
```

El flag de grupo es `1<<2`. El syscall usa exclusivamente el primario
acreditado en el último verde. La reserva nunca se promueve ni se usa para el
CONT; queda disponible solo para la retirada pre-CONT. Si el primario pierde
fiabilidad después del verde, el único intento devuelve su error raw y no se
reintenta con la reserva.

El permiso lease preautorizado se consolida inmediatamente después del syscall
y antes de interpretar retorno o mutar ownership. Fallo de consolidación es
CF. El intento consume irrevocablemente la autoridad de primer CONT: ningún
error, `EINTR`, cancelación o resultado parcial permite reintento. C2 registra
solo `intentado`; O4a recibirá el retorno raw para fijar causa conforme a su
contrato futuro.

La elegibilidad contractual se decide en `ahoraCaso`: debe cumplir
`ahoraCaso < finBootstrapGoM38`, y el CONT es el siguiente syscall literal sin
otra operación intermedia. No se sobredeclara el instante no observable en que
el kernel comienza a ejecutar el syscall ni se inventa una reserva temporal
adicional. O3c no promete cuánto presupuesto bootstrap queda después del
intento.

## Primera observación raw inmediata

Después de consolidar CONT se hace una ronda inmediata, no un bucle de espera:

1. se relee CONTROL; cualquier resultado distinto del verde exacto fija
   `control_raw` y termina la ronda;
2. solo si CONTROL quedó verde se relee el observador; cualquier cambio fija
   `senal_raw` y termina la ronda;
3. solo si ambos quedaron verdes se acreditan los pidfd explícitos y se ejecuta
   un único `poll` no recolector con timeout cero;
4. cero eventos fija `pidfd_vacio`; `POLLIN` exacto fija
   `pidfd_terminal_natural`; `POLLNVAL`, `POLLERR`, `POLLHUP`, bits extra o
   ambas referencias no fiables fijan `pidfd_infraestructura`;
5. exactamente uno de esos cinco discriminantes se instala por un único CAS;
   no se combinan eventos ni se consulta pidfd después de CONTROL/señal raw;
6. la constancia raw del intento CONT es un campo separado y nunca se usa como
   discriminante de observación;
7. una observación de infraestructura se entrega a O4a si la custodia completa
   sigue transferible; solo ownership no acreditable es CF;
8. no se obtiene código de salida, no se usa `waitid`, no se llama `Wait` y no
   se decide `SALIDA`, `PLAZO`, `CANCELADO` o señal.

La primera observación es un valor privado sellado, con autoidentidad, orden,
origen y discriminante cerrado; se fija por CAS una vez y nunca se sustituye
por una observación de limpieza. Si CONTROL, señal y pidfd son simultáneos,
prevalece estrictamente CONTROL -> observador -> pidfd. O4a será el único dueño
de convertir el valor raw y eventos posteriores en causa primaria.

O3c no espera 180 segundos: la observación inmediata evita perder una salida
natural anterior al handoff y entrega sin demora el control temporal a O4a.

## Handoff opaco a O4a

El agregado O4a se preasigna antes del último verde y contiene conjuntamente:

- `Cmd`, `Process` y handle pidfd opaco no extraído;
- primario y reserva explícitos, CONTROL y TERMINAL;
- lease y observador, registro/generación/TID y baseline actualizado;
- controlador S3/L0, PPID, SID e identidad `/proc` completa;
- snapshot físico y autoridad de exactamente tres pidfd;
- `ahoraCaso`, `finCasoO3cM38` y relación exacta 180 s;
- autoridad conjunta O3c/O4a con sus dos owners;
- constancia raw del único intento CONT y su retorno;
- primera observación raw con exactamente uno de `control_raw`, `senal_raw`,
  `pidfd_vacio`, `pidfd_terminal_natural` o `pidfd_infraestructura`;
- campos de causa/estado funcional todavía vacíos.

No contiene ticket writer, ticket, nonce ni PID/pidfd expuestos mediante
getters. No tiene métodos de efecto O4a/O4b/O4c.

Al entrar en O3c, lease y observador llegan en estados atómicos 1/1; el consumo
de entrada los lleva a 3/2. Permanecen 3/2 durante O3c y todo O4; solo O4c los
libera. Así un clon antiguo B5 nunca vuelve a superar la entrada. La
transferencia O4a prevalida conjuntamente lease=3, observador=2, mismo
registro/generación/TID, ausencia de `pending`, autoidentidad de la autoridad
conjunta y ambos owners O3C, sin mutar los subyacentes. Entra en C4T, ejecuta
CAS `ownerObservador O3C→O4A` y después `ownerLease O3C→O4A`, consume el
origen y consolida C5. La autoridad viaja solo dentro del agregado O4a; no es
global, serializable ni accesible desde un clon B5. Solo ambas transiciones
permiten retorno inmediato. Si el owner del observador consolidó y el de la
lease falla, domina CF directo: no rollback, cierre, log, E/S ni retorno.
Entre el último verde de transferencia y C5 solo hay CAS y asignaciones
infalibles sobre memoria preasignada.

## Cancelación y fallos cerrados

Antes de intentar CONT, O3c usa una retirada propia; no llama a la API O3b cuya
autoridad B5 ya fue consumida:

- CONTROL terminal, señal, bootstrap, terminalidad o error controlable congelan
  la primera observación y entran en C7;
- se fija una única `finRetiradaO3cPreCont = min(ahora+3s,
  finBootstrapGoM38)`, monotónica y no reiniciable;
- valida que lease/observador tienen estados 3/2, registro/generación/TID
  exactos y lease sin `pending`; de otro modo CF;
- conserva CONTROL y TERMINAL abiertos mientras observa y nunca los escribe;
- elige primario o, solo para retirar, reserva fiable; si el Bash aún no es
  terminal envía una sola vez `SIGKILL` individual por pidfd, sin flag de grupo;
- acredita terminalidad dentro de la marca y solo entonces ejecuta exactamente
  un `cmd.Wait`, que libera Process y su handle opaco;
- drena únicamente adoptados ya terminales mediante espera no bloqueante hasta
  `ECHILD`; cualquier adoptado vivo es CF;
- después de Wait y drenaje, pero antes de cerrar los pidfd explícitos, exige
  que la sonda cero de grupo por el pidfd fiable devuelva exactamente `ESRCH`,
  sin fallback PID/PGID; cualquier otro resultado es CF;
- cierra bajo permisos lease separados primario, reserva, CONTROL y TERMINAL,
  pone a cero cada campo y acredita cero FD/hijo/zombi/grupo/temporal;
- libera el observador desde estado 2 y, como última capacidad, la lease desde
  estado 3 mediante sus métodos publicados; después sella la autoridad conjunta
  como LIBERADA; ninguna liberación se reintenta;
- consume y pone a cero el agregado; vencimiento, ausencia de referencia
  fiable, Wait/cierre/liberación no acreditable o residuo entra en CF.

Desde el intento CONT:

- no se ejecuta retirada pre-CONT ni cierre local;
- error de CONT, cancelación, señal, terminalidad o corrupción se conserva raw
  y se transfiere a O4a;
- si el agregado completo no puede transferirse, CF termina el proceso; no se
  fabrica limpieza, rollback o postausencia.

La fatalidad usa directamente `fatalO3aM38`; el conductor exterior exige 65,
EOF/no retorno y stdout/stderr exactamente cero. No se cierra TERMINAL, no se
escribe log y no se realiza otra E/S después de decidir CF.

## Custodia y limpieza por frontera

| Recurso | Entrada O3c | Verde/CONT | Entrega O4a | Fallo pre-CONT | Fallo post-CONT |
| --- | --- | --- | --- | --- | --- |
| `Cmd`/`Process`/handle | O3c | O3c | O4a | terminalidad + único Wait O3c | O4a o CF |
| pidfd primario/reserva | O3c bajo lease | O3c | O4a juntos | cierre tras Wait | O4a o CF |
| CONTROL | O3c | lectura solamente | O4a | cierre O3c tras Wait | O4a o CF |
| TERMINAL | O3c, sin escribir | inmóvil | O4a | cierre O3c tras Wait | O4a o CF |
| lease/observador | 1/1→3/2 | permanecen 3/2 | owners O3C→O4A, observador primero | liberar desde 3/2, lease última | O4a o CF |
| deadline bootstrap | heredado | solo lectura | O4a | limita retirada | no se reinicia |
| deadline 180 s | no existe | se crea una vez | O4a | no existe | O4a |
| primera observación | vacía O3b | CAS una vez | O4a | causa retirada | O4a raw |

O3c no cierra recursos en la ruta positiva. O4c será el único propietario del
`cmd.Wait` funcional, drenaje, verificación `ECHILD`/`ESRCH`, cierre final de
pidfd/CONTROL/TERMINAL y liberación de lease/observador.

## Frontera exacta O4a/O4b/O4c

- **O4a — causa y tiempo:** consume el agregado O4a, preserva la primera
  observación, evalúa `finCasoO3cM38`, ordena eventos simultáneos y fija una
  sola causa primaria. No envía señales ni recoge.
- **O4b — señales funcionales:** ejecuta como máximo las transiciones de grupo
  STOP/TERM/CONT/KILL autorizadas por O4a, siempre por pidfd y lease. No decide
  causa ni llama Wait.
- **O4c — terminalidad y limpieza:** observa terminalidad, ejecuta el único
  `cmd.Wait` funcional, drena adoptados, exige `ECHILD` y `ESRCH`, escribe y
  cierra TERMINAL y libera la custodia en orden. No cambia causa ni plazo.

O3c no anticipa ninguna de esas implementaciones. O4a, O4b y O4c requieren
contratos separados y doble revisión antes de código.

## Precedencias totales

Antes de CONT gana la primera condición observable en este orden:

1. CONTROL terminal/framing/EOF;
2. señal, contador, registro, TID o PPID;
3. bootstrap vencido;
4. terminalidad/corrupción pidfd;
5. inventario, identidad, lease o propiedad;
6. creación válida de marca;
7. intento CONT.

Después del intento CONT:

1. error raw del intento y consolidación lease;
2. CONTROL;
3. observador/señal;
4. observación pidfd natural/infrastructura;
5. handoff.

La primera observación nunca se reemplaza. Una causa canónica no se fija en
O3c. Ninguna limpieza gana a un evento previo.

## APIs y conductas prohibidas

En el grafo productivo O3c se prohíben:

- `Start`, `Run`, `Output`, `CombinedOutput`, `StartProcess` y otro proceso;
- `cmd.Wait` salvo la única llamada C7 dominada por terminalidad; `waitid`
  recolector, goroutine de espera, callback, hook o función variable;
- `pidfd_open`, `dup*`, `F_DUPFD*`, `os.NewFile` sobre pidfd o cuarta referencia;
- PID/PGID numérico, `kill`, `Process.Signal/Kill` o señal distinta del único
  `SIGCONT` de grupo, el SIGKILL individual C7 y su sonda cero de grupo ESRCH;
- segundo CONT, STOP, TERM o KILL funcional;
- cualquier `/proc` salvo la relectura probatoria exacta
  `/proc/<PID>/stat` mediante el lector/parser O3b; parser nuevo de CONTROL,
  ticket, salida, error o TERMINAL;
- escritura de TERMINAL en cualquier ruta; cierre de TERMINAL fuera del único
  cierre C7, que ocurre después de Wait, ECHILD y grupo ESRCH; lectura de
  salida/error o emisión de recibos;
- crear otro deadline, reiniciar bootstrap, extender 180 s o esperar su fin;
- exponer PID/pidfd/ticket/nonce/identidad, loguearlos o serializarlos;
- entorno, comando, FD, goroutine, timer o canal nuevos;
- transferencia parcial, rollback de ownership o agregado con efecto O4.

## DAG obligatorio de minitareas

Cada minitarea tiene una responsabilidad observable y la siguiente permanece
bloqueada hasta commit publicado, revisión independiente y CI 5/5:

| ID | Responsabilidad | Write-set máximo | Cierre |
| --- | --- | --- | --- |
| O3C-P1-AUTORIDAD | Máquina C0..CF, consumo CAS conjunto de autoridades y agregado O4a opaco sin efecto. | Un Go productivo + prueba focal. | B5 read-only; nulos/alias/clon/reuso y ownership cerrados. |
| O3C-P2-REVALIDACION | Ronda final, precedencias, pidfd/inventario y permiso pre-CONT. | Un Go productivo + prueba focal; P1 inmóvil salvo sello mínimo. | Cero CONT y cero efecto en negativos. |
| O3C-P3-CONT | Marca única 180 s y único CONT de grupo. | Un Go productivo + prueba focal; anteriores inmóviles. | Siguiente syscall tras marca, un intento y lease consolidada. |
| O3C-P4-OBSERVACION | Ronda inmediata y primera observación raw por CAS. | Un Go productivo + prueba focal. | Vacía/natural/simultánea sin Wait ni causa. |
| O3C-P5-HANDOFF | Transferencia conjunta a O4a y retirada pre-CONT. | Un Go productivo + prueba focal. | C5 o C8/CF, sin custodia parcial. |
| O3C-P6-CONDUCTOR | Conductor durable, AST/tipado/DAG y mutantes. | Solo herramientas/pruebas externas. | Matriz normal/race, mutantes muertos, residuos cero. |
| O3C-P7-EVIDENCIA | Ledger, actas, documentación y CI. | Solo evidencia/documentación propia. | Doble GO material, push y CI 5/5. |

Aristas únicas: P1→P2→P3→P4→P5→P6→P7. O4a-P0 depende de P7; O4b-P0
depende del contrato O4a, y O4c-P0 de ambos. No hay trabajo paralelo sobre el
mismo fichero ni arista O4→O3c.

## Presupuesto y write-set futuro

Este corte modifica solo esta decisión y `docs/portal_vec/ultima_entrega.md`.
Los documentos transversales de dirección permanecen inmóviles.

Para código futuro:

- máximo cinco fuentes productivas nuevas P1--P5, una por minitarea;
- máximo una prueba focal nueva por fuente;
- objetivo 300--500 líneas y parada local 650 por fichero O3c;
- tope duro DEC-051: 800; superar 650 obliga a refactor/decisión previa;
- P6 usa un directorio de conductor, uno AST y uno mutantes, sin código
  productivo;
- P7 solo evidencia y revisiones.

Fuentes O3a/O3b, runner, workflow, SQL, adaptadores, R, documentos de dirección
y módulos ajenos permanecen byte a byte. Una corrección estrictamente necesaria
en un sello anterior exige minitarea separada, contrato y doble revisión; no se
absorbe dentro de O3c.

## Matriz mínima de pruebas y oráculos

| ID | Oráculo causal |
| --- | --- |
| C01 | Entrada B5 se consume una vez; alias/clon/reuso, incluso tras C5, no toca recursos. |
| C02 | Agregado incompleto, ticket writer presente o primera observación no vacía rechaza. |
| C03 | Registro/generación/TID/PPID/PDEATHSIG/lease/observador exactos. |
| C04 | CONTROL EAGAIN nominal; cancelar/EOF/framing/parcial/presupuesto/EINTR gana. |
| C05 | Bootstrap no se reinicia y vencimiento produce cero CONT. |
| C06 | Dos pidfd explícitos + handle opaco, sin cuarta referencia ni duplicación. |
| C07 | Terminalidad previa o `/proc stat` distinto de T/PID/PPID/PGID/SID/inicio produce retirada, nunca CONT. |
| C08 | Cada syscall tiene permiso lease único y consolidación inmediata. |
| C09 | Marca monotónica exacta 180 s, creada una vez sin overflow. |
| C10 | El primer syscall posterior a la marca es CONT. |
| C11 | CONT usa señal SIGCONT, flag de grupo y pidfd fiable sellado. |
| C12 | EINTR/error de CONT no reintenta y conserva intento raw. |
| C13 | CONT se inicia antes del bootstrap; ningún reloj se intercala. |
| C14 | Si CONTROL/observador están verdes, poll inmediato clasifica exactamente vacío, POLLIN natural o infraestructura. |
| C15 | CONTROL raw corta antes que señal; señal raw corta antes que pidfd; no se mezclan discriminantes simultáneos. |
| C16 | Exactamente un discriminante cerrado se fija una vez y limpieza no lo sustituye. |
| C17 | Autoridad conjunta transfiere owner observador antes de lease; parcial es CF y subyacentes siguen 3/2. |
| C18 | Agregado O4a conserva autoridad/custodia, plazo, retorno CONT raw separado y unión cerrada de observación; sin ticket/getters. |
| C19 | Fallo pre-CONT retira sin CONT; Wait tras terminalidad, ECHILD y grupo ESRCH preceden el único cierre C7 de TERMINAL y los demás cierres; cero residuos. |
| C20 | Fallo post-CONT nunca retira localmente; entrega a O4a o CF 65/EOF/0/0. |
| C21 | O3c no espera 180 s, no decide causa, nunca escribe TERMINAL y no la cierra fuera de C7; no usa APIs O4a/b/c. |
| C22 | Cien iteraciones normal y cien race: FD/hijos/zombis/grupos/temporales delta cero. |

Cada caso conserva ID, comando, SHA target, modo, estado, stdout/stderr,
duración, inventarios y oráculo explícito. No hay `SKIP` ni reintentos que
oculten fallos. Los BF se ejecutan como procesos directos y acreditan 65,
EOF/no retorno y cero bytes, sin sustituirlos por `PASS` del harness.

## Plan de mutantes

Cada alternativa se expande a mutantes atómicos compilables `C001..CN`, con
un patrón anterior/posterior de cardinalidad uno y oráculo causal. La presencia
de una familia no acredita sus alternativas.

- C01 omitir/reordenar uno de los CAS conjuntos, mutar B5, no anular llamador,
  aceptar alias/clon/reuso antes/después de C5 o retornar tras partición;
- C02 aceptar estado distinto B5, estados atómicos distintos de 1/1, recurso nulo, ticket
  writer o observación previa;
- C03 omitir registro/pertenencia/generación/TID/PPID/PDEATHSIG/baseline/signo;
- C04 reordenar CONTROL, aceptar EOF/parcial/framing/presupuesto/EINTR;
- C05 recrear, extender o ignorar bootstrap;
- C06 omitir primario/reserva/handle, aceptar cuarta referencia, duplicar o promover;
- C07 aceptar terminalidad previa, flags pidfd adversos, ruta/parser distinto o
  T/PID/PPID/PGID/SID/starttime cambiado;
- C08 compartir/reusar/forjar permiso lease, omitir comenzar o consolidar;
- C09 crear deadline antes, después, dos veces, con 179/181 s o aceptar overflow;
- C10 insertar syscall, reloj, log, asignación o E/S entre marca y CONT;
- C11 usar señal 0/STOP/TERM/KILL, flag 0 o PID/PGID numérico;
- C12 reintentar CONT, interpretar EINTR como no intento o perder retorno raw;
- C13 permitir CONT tras bootstrap;
- C14 poll bloqueante, omitir poll, confundir vacío/POLLIN/infraestructura o
  aceptar una variante fuera de la unión;
- C15 invertir precedencia CONTROL/observador/pidfd, consultar pidfd tras un raw
  anterior o combinar dos discriminantes;
- C16 omitir discriminante, sustituirlo, permitir segundo CAS o fijar causa;
- C17 transferir lease owner primero, revertir subyacentes a 1/1, omitir
  autoridad conjunta/prevalidación/B4T o retornar parcial;
- C18 exponer PID/pidfd, omitir autoridad/recurso/plazo/retorno CONT raw/unión
  de observación, mezclarlos o incluir ticket;
- C19 usar CONT/grupo funcional en retirada, Wait antes de terminalidad,
  omitir ECHILD/ESRCH, cerrar TERMINAL fuera de C7 o antes de Wait/ECHILD/ESRCH,
  cerrar pidfd antes de ESRCH o reiniciar 3 s;
- C20 cerrar/loguear/retornar tras CF o retirar localmente post-CONT;
- C21 añadir `waitid`, segundo Wait, Wait fuera de C7 o antes de terminalidad,
  CONT extra, TERM/KILL funcional, escritura de TERMINAL, cierre fuera de C7 o
  fin alternativo;
- C22 falsear inventario, aceptar residuos, SKIP o reutilizar evidencia;
- C23 añadir segundo owner fuera de la autoridad conjunta, ciclo,
  hook/callback/mock/goroutine/global/init;
- C24 cambiar límites 1024/4/4096/8, 180 s, 3 s, mínimo FD o parada 650;

Cada mutante pasa primero `gofmt`, build y `go vet`; un no compilable no cuenta
como muerto. El analizador AST/tipado/DAG verifica máquina total, B5 read-only,
CAS subyacente 1/1→2/3 sin reversión, autoridad conjunta única, owners
observador→lease, orden
marca→CONT, señal/flag, unión cerrada de observación con asignación única y
precedencia CONTROL→observador→pidfd, ownership acíclico, handoff conjunto,
Wait único solo en C7 y dominado por terminalidad, `waitid` ausente, APIs
prohibidas, escritura TERMINAL ausente, cierre TERMINAL único solo en C7 tras
Wait/ECHILD/ESRCH y separación O4. Todo mutante debe morir; la
huella global no es oráculo.

## Evidencia y puertas futuras

P6/P7 deberán ligar base, commits, fuentes, toolchain Go, conductor, catálogo,
mutantes y resultados mediante SHA-256 reproducible desde checkout limpio.
Puertas mínimas:

- focales normales y race, `gofmt`, `go vet` y `git diff --check`;
- conductor normal/race y 100+100 capturas con residuos cero;
- AST/tipado/DAG y expansión completa de mutantes;
- `go test ./...`, `go test -race ./...`, `go vet ./...` y calidad global;
- Gitleaks y escaneo de rutas privadas/secretos;
- doble revisión funcional/seguridad sobre bytes congelados;
- push normal sin force y CI exacta 5/5.

## Paradas duras

Se detiene si O3c crea más de una marca o CONT; usa un plazo distinto de 180 s;
intercala un syscall entre marca y CONT; reinicia bootstrap; duplica/reabre
pidfd; usa señal numérica; decide causa; espera el plazo; llama `waitid`, un
segundo Wait, Wait fuera de C7 o Wait antes de terminalidad;
escribe TERMINAL o la cierra fuera de C7/antes de Wait, ECHILD y ESRCH; usa
Wait/drenaje fuera de C7 o sin terminalidad, ECHILD y ESRCH previos; pierde la
primera observación; entrega ownership parcial;
ejecuta retirada local post-CONT; abre O4; supera write-set/parada; cambia
producción O3a/O3b; falta una alternativa mutante; sobrevive un mutante; queda
un FD/hijo/zombi/grupo/temporal; una revisión no es GO o CI no llega a 5/5.

## Seguridad, datos y métricas

No se tratan datos personales ni secretos. PID/pidfd, ticket e identidades son
opacos y no se registran. No hay HTTP, SQL, red ni decisión sobre personas. Las
puertas CT-CUM siguen bloqueando producción y datos reales.

Este contrato no cambia F0 `10/23`, O4-05 `3/5`, Contratación temporal
`24/46`, Bolsa productiva `1/14` ni producción `NO-GO`. Solo dirección puede
actualizar esos contadores después de integración funcional.

## Secuencia de autorización

1. revisión funcional y de seguridad independientes completas;
2. corrección de todo P0/P1/P2 y nueva revisión sobre bytes congelados;
3. commit documental pequeño y push normal de la rama exclusiva;
4. CI exacta 5/5;
5. solo entonces dirección puede asignar `O3C-P1-AUTORIDAD`;
6. O4a y fases posteriores permanecen cerradas hasta O3C-P7.

El productor no aprueba ni integra su propio trabajo.

## Revisiones materiales congeladas

La revisión funcional V9 y la revisión de seguridad V9 releyeron íntegramente
el mismo contrato de 596 líneas, SHA-256
`90d33545fb8c6304770a5a34e06af348db553d5d229e1e2c78364dab205fb453`,
y `ultima_entrega.md` de 30 líneas, SHA-256
`85341c5ec2cfa38fc65e26d505f221fe9a9a2359c4c623026457a3db40de976f`.
Ambas emitieron `GO`, `P0=0`, `P1=0`, `P2=0`. Esta incorporación documental
no altera las invariantes revisadas; sus nuevas huellas se vuelven a comprobar
antes del commit.
