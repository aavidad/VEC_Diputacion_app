# Decisión O3A-LEASE-CLOSED-OPS-R3: operaciones privadas cerradas

Fecha: 1 de septiembre de 2026

Tarea: `O3A-LEASE-CLOSED-OPS-R3`

Estado: candidato documental; no autoriza código, dinámica, integración ni
producción

## Resultado contractual

Este R3 sustituye íntegramente R2 y fija un contrato implementable para retirar
la API de permisos, callbacks, acreditaciones y punteros de autoridad ligados a
TID. Una operación privada cerrada adquiere por un único CAS su slot antes de
leer o mutar estado compartido, posee el efecto físico completo, conserva el raw
y consolida el resultado dentro de la misma llamada. No devuelve ni deja
pendiente permiso, ticket de autoridad, callback, handle, puntero de capacidad o
acreditación consumible por otra llamada.

Un fallo al adquirir, validar o consolidar la autoridad es irreversible:
`estado=5`, celda fatal y salida exterior 65. Un raw funcional producido por
una operación ya autorizada no se convierte por ello en fallo de autoridad;
conserva exactamente la semántica de O3a, O3b, O3c, O4a, O4b u O4c que lo
posee.

Este documento no declara que el árbol actual cumpla el contrato. La retirada
instrumentada, las migraciones, pruebas, AST, mutantes, runners y manifiestos
son trabajo futuro granular. La puerta de cero referencias es la última puerta
material, no una precondición ficticia anterior a esos cambios.

## Base, genealogía y fuentes leídas

La base documental exacta es
`96ec857f434417f3a917b681464c5a36cc689941`. El producto publicado usado solo
para comprobar composición de árboles es
`c8cc4312b063ddca7294dfe27dc673ad1a4676d0`. No se hace `fetch`, `pull`,
`reset`, integración ni publicación.

Se leyeron completas las instrucciones aplicables y las autoridades materiales
siguientes:

- `decision_f0_h0b_c4b2_g2o_o3a_arranque_mapa_fd_2026-08-09.md`, su P0 de
  margen, P1 de ticket temprano y enmienda CI;
- `decision_f0_h0b_c4b2_g2o_o3b_ticket_stop_identidad_2026-08-10.md`;
- `decision_f0_h0b_c4b2_g2o_o3c_continuacion_salida_2026-08-11.md`;
- contratos O4a de causa, arbitraje, señales, raw y plazos;
- contrato O4b de autoridad de señales y su bloqueo O4B-P2;
- `decision_f0_h0b_c4b2_g2o_o4c_terminalidad_limpieza_2026-08-13.md` y su
  enmienda de terminalidad STOP;
- autoridad O5 de composición, preflight, registros, recursos y resultado
  privado;
- el candidato R2, la matriz y especificaciones normativas enlazadas;
- inventario dirigido `/tmp/o3a-lease-callers-20260831.txt`;
- dictamen remoto `/tmp/vec-review-o3a-lease-r2-20260901.log` y el dictamen
  local comunicado por Dirección.

Los ficheros bajo `/tmp` son entrada de revisión, no evidencia durable ni parte
del commit.

## Registro explícito de los dos NO-GO R2

R2 recibió dos rechazos que este documento no oculta:

- dictamen local: `NO-GO`, `P0=2`, `P1=3`;
- dictamen remoto: `NO-GO`, `P0=4`, `P1=2`, `P2=0`.

Dirección normalizó los cinco hallazgos locales así:

| ID | Severidad | Hallazgo R2 | Cierre contractual R3 |
| --- | --- | --- | --- |
| L0-1 | P0 | Inventario incompleto de permisos, preflight y acreditación TID diferida. | El inventario nominal y semántico de retirada de este R3 incluye O3a, O3b, O3c, O4a y O4b, campos TID y derivados equivalentes. |
| L0-2 | P0 | Callbacks y handles seguían vivos, incluidos dos `WithHandle`. | Se prohíben ambos y se sustituyen por preflight directo cerrado más postcondición de inventario de `Start`, sin callback ni handle visible. |
| L1-1 | P1 | `valido`, `sellarFisico`, `transferirCritico` y `liberar` eludían el CAS. | Se retiran; validación, sellado, cambio de propietario y liberación son fases internas de operaciones cerradas. |
| L1-2 | P1 | No había firmas M38 ni reconciliación funcional exactas. | Este R3 fija tipos, nombres, argumentos, resultados y propietario para cada efecto y separa raw funcional de fallo de autoridad. |
| L1-3 | P1 | Hito de cero prematuro y C1/C2 no implementables. | C1/C2 se descomponen con una responsabilidad, rutas y presupuesto por corte; cero referencias queda después de material y herramientas. |

El dictamen remoto enumeró seis hallazgos concretos:

| ID | Severidad | Hallazgo R2 | Cierre contractual R3 |
| --- | --- | --- | --- |
| R0-1 | P0 | Faltaban `primerPermiso`, `permisoPrimero`, `revalidacionO3cM38.permiso`, `permisoContMemoriaValidoO3cM38` y `consolidarContO3cM38`. | Todos figuran en la retirada nominal y CONT queda revalidado, ejecutado y consolidado dentro de una sola llamada. |
| R0-2 | P0 | Se omitieron `acreditacionPidfdGoM38`, su preflight y los dos `WithHandle`. | Se retiran tipo, productor, consumidor y callbacks; O5 usa preflight directo efímero y O3a valida el pidfd opaco por delta físico. |
| R0-3 | P0 | El CAS no dominaba `valido`, `sellarFisico`, `transferirCritico` ni `liberar`. | No sobreviven como métodos; toda lectura o mutación correspondiente queda tras el CAS de la operación exacta. |
| R0-4 | P0 | O4c no estaba en el mapa de autoridades. | O4c queda reconciliada expresamente para terminalidad, Wait, Wait4, cierres, TERMINAL y liberación final. |
| R1-1 | P1 | R2 confundía raw de `Start`/`Wait` con fallo de autoridad. | `Start` sin hijo admite retirada ordinaria y `*exec.ExitError` exacto de Wait es terminación válida; solo autoridad o consolidación fallida son fatal+5. |
| R1-2 | P1 | C1/C2 eran monolíticos y sin presupuesto por responsabilidad. | La tabla de minitareas fija criterio observable, rutas máximas y parada de líneas inferior a 800 para cada commit. |

El cierre aquí es contractual. Solo una revisión independiente del hash exacto
de R3 puede declarar que estos hallazgos documentales están resueltos.

## Prevalencia y autoridades coordinadas

R3 enmienda solo el seam común necesario. No mueve la política funcional de
sus propietarios:

| Autoridad | Conserva | Enmienda cerrada obligatoria |
| --- | --- | --- |
| O3a | preparación, mapa FD, `Start`, pidfd explícitos y retirada previa a entrega | reemplaza lease, testigo, permisos y callbacks por el slot embebido y operaciones exactas de FD/proceso |
| O3b | barrera, ticket temprano, auto-STOP e identidad `/proc` | primera escritura, parciales, EINTR, cierre, observaciones y handoff ocurren dentro de llamadas cerradas; no guarda permiso primero |
| O3c | revalidación, marca monotónica, primer CONT y observación de salida | revalidación y CONT forman una sola llamada; no devuelve `revalidacion` ni permiso pendiente |
| O4a | causa, precedencia, etapa, plazos e interpretación de raws | prepara por índice una etapa fija, pero no emite puntero de autorización ni TID; consume un resultado embebido |
| O4b | raw literal de STOP/TERM/CONT/KILL según etapa O4a | la futura ejecución es una llamada cerrada por índice; P2 continúa bloqueado y este R3 no autoriza su código ni dinámica |
| O4c | terminalidad, único Wait funcional, Wait4, ECHILD, ESRCH, cierres, TERMINAL y liberación | cada efecto usa el slot; el orquestador no lee estado compartido directamente y entrega a O5 solo un valor inerte |
| O5 | construcción en dirección estable, preflight, composición, recursos y consumo final privado | no crea registro de tokens; compone preflight y preparación sin devolver acreditación y nunca recibe lease, handle o callback |

O3a conserva su Wait de retirada anterior a la entrega. O4c conserva el único
Wait funcional del caso entregado. No se confunden ni se duplican.

## Estado cerrado embebido y CAS dominante

O5 construye una sola `custodiaO3aM38` en su dirección definitiva. Dentro de
ella se embeben por valor, no mediante registros de punteros, estas celdas:

```go
type estadoOperacionesCerradasM38 struct {
	slot        atomic.Uint32
	estado      atomic.Uint32
	propietario atomic.Uint32
	fisico      snapshotFDO3aM38
}

type estadoObservacionCerradaM38 struct {
	palabra atomic.Uint64
}
```

No tienen getter, interfaz, serialización ni método público. No se copian tras
el primer uso. `slot=0` significa libre; cada operación tiene un discriminante
constante no reutilizado. `estado=5` significa fatal irreversible. El
propietario es una fase cerrada O5→O3a→O3b→O3c→O4a→O4c→liberado, nunca un
puntero consumible.

Toda operación con estado o efecto sigue exactamente este orden:

1. comprobar solo nulo del argumento raíz;
2. CAS único de adquisición `slot: 0→operacion`; no hay lectura compartida
   anterior, salvo el propio CAS;
3. validar estado, propietario, fase, argumentos, plazos y snapshot;
4. capturar el preestado necesario dentro de la llamada;
5. ejecutar el raw literal o la secuencia compuesta expresamente autorizada;
6. capturar raw y postestado sin soltar el slot;
7. clasificar el raw conforme a su autoridad, actualizar resultado y snapshot;
8. consolidar mediante CAS `slot: operacion→0` como última acción compartida.

Fallo en 2, 3, 6, 7 por contradicción de autoridad, o en 8, fija primero
`estado=5` y la celda fatal y termina con 65. No restaura, no hace cleanup, no
registra, no llama otro efecto y no retorna al llamador. Un raw funcional
permitido sí llega a 7 y 8 aunque signifique retirada, incidente o resultado
negativo.

Una operación compuesta mantiene el mismo slot desde su primer acceso hasta su
última consolidación. No se implementa llamando otra operación que vuelva a
adquirirlo. Sus helpers privados no aceptan `func`, interfaz o autoridad, no se
almacenan y no son llamables fuera de la función propietaria según AST.

## Inventario exhaustivo de retirada

La retirada nominal obligatoria comprende definiciones, campos, métodos,
llamadas, pruebas, AST, mutantes, runners y manifiestos vivos de:

- `permisoGuardiaO3aM38`, `comenzar`, `comenzarCritico`, `permisoValido`,
  `consolidarFisico`, `consolidarCritico` y `fatalPendiente`;
- `primerPermiso`, `permisoPrimero`, `revalidacionO3cM38.permiso`,
  `permisoContMemoriaValidoO3cM38` y `consolidarContO3cM38`;
- `acreditacionPidfdGoM38`, `preflightPidfdGoM38` y
  `consumirPreflightPidfdGoM38`;
- `testigoVueltaM38`, `celdaVueltaM38`, `relojVueltaM38`, `emitir`, `consumir`
  y el argumento `testigo` de `avanzarArranqueO3aM38`;
- `operarConLeaseBarreraO3bM38`, `syscallLeaseO3cM38` y todo parámetro,
  campo, variable o cierre de tipo `func` usado para transportar un efecto;
- ambos `Process.WithHandle` vivos y cualquier `WithHandle`, `Handle`,
  `uintptr` o callback que revele o compare el pidfd opaco;
- los métodos `valido`, `sellarFisico`, `transferirCritico` y `liberar` de
  lease u observador, más sus wrappers `leaseTransferidaO3bM38`,
  `observadorTransferidoO3bM38` y equivalentes;
- `leaseGuardiaO3aM38`, `observadorSenalO3aM38`,
  `registroAutoridadO3aM38` y sus mapas de punteros, una vez migrado el último
  consumidor al estado embebido;
- los campos de autoridad `tid` en registro, lease, reloj, celda, bundle,
  custodia, sellos O3c/O4a/O4b y autorizaciones de etapa, y todo `Gettid`
  almacenado o comparado para permitir un efecto;
- `autorizacionEtapaO4aM38`, `tomarEntradaAutoridadO4bM38`,
  `slotPropioAutoridadO4bM38`, `slotCanonicoAutoridadO4bM38`, el puntero
  `pendiente`, enlaces autorización↔resultado y cualquier clon consumible;
- todo estado `pending`, permiso, generación, nonce, auto-puntero o resultado
  que sobreviva a una llamada y habilite otra.

`Gettid` tampoco puede reaparecer con otro nombre como autoridad de goroutine.
Una observación técnica local que una autoridad previa aún exija solo puede
existir dentro de una operación cerrada, no se almacena, no sale y no decide
posesión; la implementación preferida elimina también esa observación.

La lista nominal no limita la retirada semántica. Un canal equivalente es
NO-GO aunque use nombres nuevos.

## Sustitución concreta de `WithHandle` y preflight

O5 deja de acreditar la biblioteca estándar con un token. La sustitución es la
operación compuesta exacta
`componerEntradaConPreflightCerradoO5M38`: adquiere el slot del bundle en su
dirección final; ejecuta `pidfd_open(getpid, 0)` directo, verifica
`FD_CLOEXEC`, ejecuta señal cero directa, cierra ese pidfd una vez, comprueba
inventario basal y continúa inmediatamente con la preparación O3a dentro de la
misma llamada. No usa `os.FindProcess`, `WithHandle`, `Release`, callback ni
resultado diferido. Ningún pidfd del preflight queda en el bundle.

La integración real de Go se acredita después de `cmd.Start`, todavía dentro
de `iniciarProcesoCerradoO3aM38`: el delta físico contiene el pidfd primario de
`SysProcAttr.PidFD` y exactamente un pidfd opaco adicional, ambos `CLOEXEC` y
de identidad física igual. Su número se conserva solo en la custodia para que
el único `cmd.Wait` acredite su desaparición. Nunca se obtiene, compara o
expone mediante handle. La reserva explícita se crea en la operación cerrada
posterior y debe tener la misma identidad.

Así se retiran los dos callbacks vivos sin sustituirlos por otro callback o
por una acreditación copiable.

## Tipos de resultado exactos sin autoridad

Los únicos resultados nuevos son valores inertes:

```go
type resultadoFDCerradoM38 struct {
	aplicado bool
	errno    syscall.Errno
}

type claseStartCerradoO3aM38 uint8
const (
	startSinHijoO3aM38 claseStartCerradoO3aM38 = iota + 1
	startConHijoRetirableO3aM38
	startEntregableO3aM38
)

type resultadoEscrituraTicketO3bM38 struct {
	completo bool
	escritos int
	errno    syscall.Errno
}

type resultadoContCerradoO3cM38 struct {
	clase      claseContO3cM38
	retornoRaw int
	ahoraCaso  time.Time
	finCaso    time.Time
}

type resultadoWait4CerradoO4cM38 struct {
	clase  claseWait4O4cM38
	pid    int
	estado syscall.WaitStatus
	uso    syscall.Rusage
}

type resultadoTerminalidadO4cM38 struct {
	estadoExterior uint8
	causa          causaPrimariaO4aM38
	terminalNormal bool
	cuarentena     bool
}
```

Los enums nuevos y sus únicos valores son exactamente:

```go
type clasePreparacionO5M38 uint8
const (
	preparacionListaO5M38 clasePreparacionO5M38 = iota + 1
	preparacionRechazadaO5M38
)

type clasePidfdCerradoM38 uint8
const (
	pidfdVivoCerradoM38 clasePidfdCerradoM38 = iota + 1
	pidfdTerminalCerradoM38
	pidfdRawFallidoCerradoM38
)

type rolPidfdCerradoM38 uint8
const (
	pidfdPrimarioCerradoM38 rolPidfdCerradoM38 = iota + 1
	pidfdReservaCerradoM38
	pidfdOpacoCerradoM38
)

type propietarioCerradoM38 uint8
const (
	propietarioO5CerradoM38 propietarioCerradoM38 = iota + 1
	propietarioO3aCerradoM38
	propietarioO3bCerradoM38
	propietarioO3cCerradoM38
	propietarioO4aCerradoM38
	propietarioO4cCerradoM38
	propietarioLiberadoCerradoM38
)

type claseWaitRetiradaO3aM38 uint8
const (
	waitRetiradaRecogidoO3aM38 claseWaitRetiradaO3aM38 = iota + 1
	waitRetiradaExitErrorO3aM38
)

type claseBarreraO3bM38 uint8
const (
	barreraVerdeCerradaO3bM38 claseBarreraO3bM38 = iota + 1
	barreraAplazadaCerradaO3bM38
	barreraRetiradaCerradaO3bM38
)

type claseStopO3bM38 uint8
const (
	stopAcreditadoCerradoO3bM38 claseStopO3bM38 = iota + 1
	stopRetiradoCerradoO3bM38
)

type claseContO3cM38 uint8
const (
	contRetiradoAntesRawO3cM38 claseContO3cM38 = iota + 1
	contRawIntentadoO3cM38
)

type claseObservacionO3cM38 uint8
const (
	observacionSinEventoO3cM38 claseObservacionO3cM38 = iota + 1
	observacionControlO3cM38
	observacionSenalO3cM38
	observacionTerminalO3cM38
)

type claseEjecucionEtapaO4bM38 uint8
const (
	etapaEjecutadaO4bM38 claseEjecucionEtapaO4bM38 = iota + 1
	etapaRawNegativoO4bM38
)

type claseResultadoEtapaO4aM38 uint8
const (
	resultadoEtapaConsumidoO4aM38 claseResultadoEtapaO4aM38 = iota + 1
	resultadoEtapaFuncionalNegativoO4aM38
)

type claseTerminalidadO4cM38 uint8
const (
	terminalidadVivaO4cM38 claseTerminalidadO4cM38 = iota + 1
	terminalidadConfirmadaO4cM38
)

type claseWaitO4cM38 uint8
const (
	waitRecogidoO4cM38 claseWaitO4cM38 = iota + 1
	waitExitErrorValidoO4cM38
)

type claseWait4O4cM38 uint8
const (
	wait4AdoptadoO4cM38 claseWait4O4cM38 = iota + 1
	wait4ECHILDO4cM38
	wait4HijoVivoO4cM38
	wait4RawFallidoO4cM38
)
```

No contienen FD, PID/PGID persistible, handle, TID, nonce, generación,
puntero, interfaz de autoridad, función, error libre, ruta o dato personal.
Los raws que no caben en un valor seguro se fijan en la custodia dentro de la
llamada y solo se devuelve su clase cerrada. Copiar un resultado no permite
ningún efecto.

`destinoFDCerradoO3aM38` es un enum cerrado con exactamente estos literales:
`fdRaizO3aM38`, `fdRunnerO3aM38`, `fdSalidaO3aM38`, `fdErrorCasoO3aM38`,
`fdControlO3aM38`, `fdTerminalO3aM38`, `fdTicketLectorO3aM38`,
`fdTicketEscritorO3aM38`, `fdPidfdPrimarioO3aM38` y
`fdPidfdReservaO3aM38`. La operación escribe el recurso en la custodia; nunca
lo devuelve.

Los argumentos físicos tampoco son libres: duplicación exige
`minimo==minFDDuplicadoM38`; apertura O3a admite solo `/dev/null`,
`O_RDONLY|O_CLOEXEC|O_NOFOLLOW`, modo cero y el nombre interno fijado por el
destino; pipe admite solo `O_CLOEXEC` y los destinos ticket lector/escritor.
Close admite solo un destino actualmente poseído. La apertura `/proc` de O3b
permanece dentro de `evaluarBarreraCerradaO3bM38`, con PID sellado y flags de
su autoridad; no reutiliza la apertura O3a. Todo otro tuple falla antes del raw
como autoridad inválida y, tras CAS adquirido, es fatal+5.

## Nombres y firmas M38 exactos

No son familias ni ejemplos. Cualquier otra firma que ejecute el mismo efecto
es NO-GO.

| Propietario | Firma exacta | Argumentos y resultado |
| --- | --- | --- |
| O5 | `func componerEntradaConPreflightCerradoO5M38(e *bundleEntradaO3aM38) clasePreparacionO5M38` | bundle en dirección final; preflight, cierre y preparación en una llamada; devuelve solo clase |
| O3a | `func cerrarFDCerradoO3aM38(c *custodiaO3aM38, destino destinoFDCerradoO3aM38) resultadoFDCerradoM38` | cierra una vez el campo enumerado, lo anula si el inventario acredita baja |
| O3a | `func duplicarFDCerradoO3aM38(c *custodiaO3aM38, origen, destino destinoFDCerradoO3aM38, minimo int, nombre string) resultadoFDCerradoM38` | `F_DUPFD_CLOEXEC`; almacena el nuevo archivo en `destino`; no devuelve FD/puntero |
| O3a | `func abrirFDCerradoO3aM38(c *custodiaO3aM38, destino destinoFDCerradoO3aM38, ruta string, flags int, modo uint32, nombre string) resultadoFDCerradoM38` | `open` literal y sellado físico; almacena en destino |
| O3a | `func crearPipeCerradoO3aM38(c *custodiaO3aM38, lector, escritor destinoFDCerradoO3aM38, flags int, nombres [2]string) resultadoFDCerradoM38` | `pipe2` literal; almacena ambos extremos o ninguno |
| O3a | `func consumirControlCerradoO3aM38(c *custodiaO3aM38, fragmento []byte, fin bool) resultadoControlPreinicioM38` | consumo y contador internos; no devuelve testigo/vuelta |
| O3a | `func iniciarProcesoCerradoO3aM38(c *custodiaO3aM38) claseStartCerradoO3aM38` | único `cmd.Start`, raw fijado en custodia, delta pidfd completo y clase sin autoridad |
| O3a | `func reservarPidfdCerradoO3aM38(c *custodiaO3aM38) resultadoFDCerradoM38` | duplica primario a reserva y lo almacena; no devuelve FD |
| O3a | `func sondearPidfdCerradoO3aM38(c *custodiaO3aM38, rol rolPidfdCerradoM38) clasePidfdCerradoM38` | `poll`/identidad bajo slot; devuelve solo vivo, terminal o raw fallido |
| O3a | `func esperarRetiradaCerradoO3aM38(c *custodiaO3aM38) claseWaitRetiradaO3aM38` | único Wait de retirada; consume pidfd opaco y fija `ProcessState` dentro |
| O3a | `func avanzarPropietarioCerradoO3aM38(c *custodiaO3aM38, desde, hacia propietarioCerradoM38) bool` | transición interna exacta; no devuelve lease/observador |
| O3b | `func evaluarBarreraCerradaO3bM38(a *autoridadCapturaO3bM38) claseBarreraO3bM38` | lecturas, identidad, inventario y plazo dentro de una llamada compuesta |
| O3b | `func emitirYCerrarTicketCerradoO3bM38(a *autoridadCapturaO3bM38) resultadoEscrituraTicketO3bM38` | primera escritura inmediata, parciales/reintentos acotados y cierre en el mismo slot |
| O3b | `func observarStopCerradoO3bM38(a *autoridadCapturaO3bM38) claseStopO3bM38` | `/proc`, pidfd e inventario; fija identidad, no devuelve acreditación |
| O3c | `func revalidarEIntentarContCerradoO3cM38(a *autoridadContinuacionO3cM38) resultadoContCerradoO3cM38` | toda revalidación, marca exacta y un CONT; no retorna `revalidacion` ni permiso |
| O3c | `func observarCasoCerradoO3cM38(a *autoridadContinuacionO3cM38) claseObservacionO3cM38` | CONTROL/señal/pidfd bajo slot; fija primera observación dentro |
| O4a | `func consumirResultadoEtapaCerradaO4aM38(e *autoridadEtapasO4aM38, indice uint8) claseResultadoEtapaO4aM38` | consume por valor la celda embebida del índice; no recibe puntero O4b |
| O4b | `func ejecutarEtapaCerradaO4bM38(e *autoridadEtapasO4aM38, indice uint8) claseEjecucionEtapaO4bM38` | lee tras CAS la etapa ya fijada, ejecuta raws literales y sella resultado embebido |
| O4b | `func enviarSenalPidfdCerradaO4bM38(e *autoridadEtapasO4aM38, indice uint8, senal syscall.Signal, flags uintptr) syscall.Errno` | helper privado de la anterior; un raw, solo flags canónicos, no se llama fuera según AST |
| O4c | `func confirmarTerminalidadCerradaO4cM38(a *autoridadTerminalidadO4cM38) claseTerminalidadO4cM38` | pareja primario/reserva y límite prestado; no recoge |
| O4c | `func esperarCmdCerradoO4cM38(a *autoridadTerminalidadO4cM38) claseWaitO4cM38` | exactamente un `cmd.Wait`; fija estado real internamente |
| O4c | `func drenarWait4CerradoO4cM38(a *autoridadTerminalidadO4cM38) resultadoWait4CerradoO4cM38` | cada raw es `Wait4(-1,&estado,WNOHANG,&uso)`; itera dentro hasta `ECHILD` |
| O4c | `func cerrarTerminalidadCerradaO4cM38(a *autoridadTerminalidadO4cM38) resultadoTerminalidadO4cM38` | orquesta solo llamadas cerradas: terminalidad, Wait, Wait4, ESRCH, cierres, TERMINAL y liberación |
| O4c | `func liberarCustodiaCerradaO4cM38(a *autoridadTerminalidadO4cM38) bool` | libera primero observación y al final operaciones; fija propietario liberado; ningún efecto posterior |

`cerrarTerminalidadCerradaO4cM38` no inspecciona campos directamente. Sus
decisiones proceden de los valores cerrados devueltos por las operaciones de
la tabla. Los helpers de raw permanecen privados a su operación compuesta y no
forman una API alternativa.

## Semántica exacta de raws funcionales

La regla `raw != éxito ⇒ fatal` de R2 queda anulada.

### FD, close, dup, open y pipe

Cada raw se intenta una vez, salvo el bucle de escritura de ticket autorizado
por O3b. En dup/open/pipe, error más inventario sin cambio es resultado
funcional negativo consolidado; éxito exige el delta exacto y `CLOEXEC`.
Contradicción raw/inventario o inventario ilegible es fallo de consolidación y
fatal+5.

`Close` no reintenta `EINTR`. El inventario decide si hubo baja. O3a conserva
la primera retirada; O4c enclava incidente y cuarentena cuando su contrato lo
ordena. Un resultado incierto que impida acreditar la postcondición no se
presenta como éxito: si rompe consolidación es fatal+5.

### `Start` y pidfd

`errStart != nil`, `cmd.Process == nil` y pidfd primario `-1`, con inventario
sin hijo ni nuevas referencias, es `startSinHijoO3aM38` y retirada ordinaria.
No es fallo de autoridad.

Cualquier tupla con hijo observable se clasifica con hijo aunque `errStart` no
sea nulo y entra en retirada protegida. `Process`, pidfd o delta mutuamente
contradictorios, o consolidación imposible, son fatal+5. Éxito entregable exige
Process válido, primario, opaco, reserva posterior, identidades iguales,
`CLOEXEC` y delta exacto.

### Ticket O3b

Tras B1, el primer syscall de la llamada es literalmente `Write`; el CAS y las
lecturas de memoria necesarias no son syscalls. No existe permiso preparado.
Un parcial avanza solo por el sufijo no escrito. `EINTR` conserva `n`, cuenta
como máximo ocho interrupciones y reintenta según O3b; antes de cada reintento,
no antes del primero, se relee el límite. Error funcional retira y el cierre se
intenta una vez dentro de la misma llamada. Fallo de slot o consolidación es
fatal+5.

### CONT O3c

La llamada cerrada revalida CONTROL, señal, bootstrap, pidfd, inventario,
identidad, STOP y segunda ronda; si una autoridad O3c permite retirada antes
del CONT, devuelve esa clase sin efecto. Inmediatamente antes del raw crea la
marca monotónica exacta de 180 segundos. Ejecuta una sola vez
`pidfd_send_signal(primario,SIGCONT,0,PIDFD_SIGNAL_PROCESS_GROUP)`.

El raw, incluido `EINTR`, se conserva sin retry ni interpretación en
`retornoRaw`; O4a decide su significado. La misma llamada fija `ahoraCaso`,
`finCaso`, raw y estado C2 antes de soltar el slot. No existe fase C1 que
devuelva permiso a P3.

### O4b

Las cardinalidades, orden STOP/TERM/CONT/KILL, flags y deadlines siguen siendo
los fijados por O4a/O4b. Cada señal literal tiene un intento. `EINTR`, EPERM,
ESRCH y cualquier otro raw se sellan exactamente, no se reinterpretan como
autoridad ni se reintentan. O4a conserva precedencia y clasificación.

Estas firmas son contrato futuro, no apertura de O4B-P2. P2 sigue bloqueado;
no se programa ni se ejecuta dinámica por este R3.

### Wait, Wait4 y O4c

O4c confirma terminalidad antes de Wait. `cmd.Wait()==nil` es válido.
`*exec.ExitError` es terminación válida solo si pertenece al mismo Cmd y su
`ProcessState` exacto queda fijado; no es por sí solo fatal ni incidente.
Otro error que no acredite recolección sigue la fatalidad O4c.

Después del Wait, `drenarWait4CerradoO4cM38` usa exclusivamente objetivo `-1`,
`WNOHANG` y `Rusage` local. PID positivo recoge un adoptado ya terminal;
`ECHILD` termina el drenaje; PID cero acredita un hijo vivo y no se acepta como
cierre. No hay Wait4 bloqueante, retry libre ni Wait del líder. La semántica de
`EINTR` y otros errores permanece la de O4c: no se inventa ECHILD ni éxito.

El orden O4c es inmutable:

1. terminalidad concordante antes del límite;
2. único Wait funcional y consumo físico del pidfd opaco;
3. Wait4 no bloqueante hasta ECHILD;
4. grupo ausente por ESRCH según su autoridad;
5. cierre único de primario, reserva y CONTROL, anulando cada campo;
6. escritura lógica única y cierre de TERMINAL;
7. liberación de observación;
8. liberación final del estado de operaciones y propietario `liberado`;
9. cero lectura, syscall, E/S, snapshot, log o asignación falible posterior.

El resultado a O5 es el valor inerte `resultadoTerminalidadO4cM38`. No es
autoidentificado ni consumible como permiso, no tiene getter o serialización y
no sustituye el recibo Shell de O5.

## DAG futuro y condición para abrir código

Primero el hash exacto de este documento necesita dos revisiones materiales
independientes `GO`, ambas `P0=P1=P2=0`; Dirección decide integración. Solo
entonces puede abrirse la primera minitarea C1. Ningún productor revisa o
integra su trabajo.

El DAG exacto es:

```text
R3 doble GO e integración documental
→ C1a … C1l, en el orden de su tabla
→ C2a … C2v, en el orden de su tabla
→ C3 pruebas y AST
→ C4 mutantes, runners y manifiestos
→ C5 cero nominal y semántico final
→ V26 material → doble revisión → acta V26 separada
→ C22 material → doble revisión → evidencia C22 separada
→ integración, repetición de puertas y CI 5/5
→ decisión separada sobre O4B-P2.
```

Cada flecha exige commit limpio, compilable y pruebas focales verdes. Cada hash
material recibe dos revisiones independientes antes de integración. Un corte
no mezcla responsabilidad ni acta.

En las tablas, `S/` significa
`deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/`. “Parada” es el máximo
de líneas del fichero final, siempre menor de 800; si no cabe, se crea la ruta
nueva indicada y no se amplía el write-set.

Las rutas de C1/C2 se nombran mediante estos aliases exactos, sin globs:

- `A0` = `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go`;
- `A1N/A1NT` = `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_operaciones_cerradas_nucleo.go` y su `_test.go`;
- `A1F/A1FT` = `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_operaciones_cerradas_fd.go` y su `_test.go`;
- `A1P/A1PT` = `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_operaciones_cerradas_proceso.go` y su `_test.go`;
- `A1S/A1ST` = `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_operaciones_cerradas_estado.go` y su `_test.go`;
- `A2` = `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go`;
- `A3` = `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go`;
- `A4` = `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go`;
- `A5` = `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go`;
- `B0/B0T` = `S/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas.go` y su `_test.go`;
- `B1/B1T` = `S/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_barrera.go` y su `_test.go`;
- `B2/B2T` = `S/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_ticket.go` y su `_test.go`;
- `B3/B3T` = `S/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_stop.go` y su `_test.go`;
- `B4/B4T` = `S/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_identidad.go` y su `_test.go`;
- `B5/B5T` = `S/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go` y su `_test.go`;
- `C0/C0T` = `S/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas.go` y su `_test.go`;
- `C1/C1T` = `S/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_revalidacion.go` y su `_test.go`;
- `C2/C2T` = `S/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_cont.go` y su `_test.go`;
- `C3/C3T` = `S/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_observacion.go` y su `_test.go`;
- `C4/C4T` = `S/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go` y su `_test.go`;
- `D0/D0T` = `S/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas.go` y su `_test.go`;
- `D1/D1T` = `S/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas.go` y su `_test.go`;
- `D2/D2T` = `S/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go` y su `_test.go`;
- `E0/E0T` = `S/senales_grupo_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go` y su `_test.go`;
- `F0` = `S/terminalidad_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go`;
- `F1/F1T` = `S/terminalidad_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas.go` y su `_test.go`;
- `OPR` = `S/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operativo.go`.

## C1 descompuesto: núcleo sin consumidores migrados

| ID | Responsabilidad observable y cierre | Write-set exacto máximo | Presupuesto/parada |
| --- | --- | --- | --- |
| C1a | Definir estado embebido, discriminantes, CAS y fatal+5, sin efecto físico. | `A0`, nuevas `A1N/A1NT` | A0 559→máx. 650; A1N/A1NT 300/420; parada 700 |
| C1b | Implementar únicamente `cerrarFDCerradoO3aM38`, sin migrar callers. | nuevas `A1F/A1FT` | 120/180; parada 700 |
| C1c | Implementar únicamente `duplicarFDCerradoO3aM38`. | `A1F/A1FT` | +100/+150; acumulado 220/330; parada 700 |
| C1d | Implementar únicamente `abrirFDCerradoO3aM38`. | `A1F/A1FT` | +120/+160; acumulado 340/490; parada 700 |
| C1e | Implementar únicamente `crearPipeCerradoO3aM38`. | `A1F/A1FT` | +110/+150; acumulado 450/640; parada 700 |
| C1f | Implementar únicamente el preflight directo cerrado, sin `WithHandle`. | nuevas `A1P/A1PT` | 130/200; parada 760 |
| C1g | Implementar únicamente `iniciarProcesoCerradoO3aM38`. | `A1P/A1PT` | +190/+220; acumulado 320/420; parada 760 |
| C1h | Implementar únicamente reserva y sonda pidfd cerradas. | `A1P/A1PT` | +150/+170; acumulado 470/590; parada 760 |
| C1i | Implementar únicamente `esperarRetiradaCerradoO3aM38`. | `A1P/A1PT` | +100/+130; acumulado 570/720; parada 760 |
| C1j | Implementar únicamente consumo de CONTROL sin testigo. | nuevas `A1S/A1ST` | 140/200; parada 700 |
| C1k | Implementar únicamente observación cerrada sin TID. | `A1S/A1ST` | +130/+180; acumulado 270/380; parada 700 |
| C1l | Implementar únicamente cambio interno de propietario. | `A1S/A1ST` | +120/+170; acumulado 390/550; parada 700 |

Los nombres abreviados empiezan por el stem literal
`supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1`. C1 conserva
compilable temporalmente la API antigua, pero ninguna función nueva puede
devolverla o llamarla. C1l no abre O4B-P2.

## C2 descompuesto: un propietario por migración

| ID | Responsabilidad observable y cierre | Write-set exacto máximo | Presupuesto/parada |
| --- | --- | --- | --- |
| C2a | Migrar solo los cierres O3a y acreditar bajas exactas. | `A2`, `A3`, `A1F/A1FT` | A2 máx. 690; A3 máx. 680; A1F/A1FT 700 |
| C2b | Migrar solo la duplicación O3a. | `A2`, `A1F/A1FT` | A2 máx. 700; A1F/A1FT 700 |
| C2c | Migrar solo la apertura O3a. | `A2`, `A1F/A1FT` | A2 máx. 710; A1F/A1FT 700 |
| C2d | Migrar solo la creación de pipe O3a. | `A2`, `A1F/A1FT` | A2 500→máx. 720; A1F/A1FT 700 |
| C2e | Sustituir solo preflight/token y retirar el primer `WithHandle`. | `A0`, `A2`, `A4`, `A1P/A1PT` | 680, 740, 760, 760/760 |
| C2f | Migrar solo `Start` y sus tres clases funcionales. | `A3`, `A1P/A1PT` | A3 548→máx. 720; A1P/A1PT 760 |
| C2g | Migrar solo delta, reserva y sondas pidfd O3a. | `A3`, `A1P/A1PT` | A3 máx. 740; A1P/A1PT 760 |
| C2h | Migrar solo el Wait de retirada O3a. | `A3`, `A1P/A1PT` | A3 máx. 750; A1P/A1PT 760 |
| C2i | Migrar solo los consumidores de CONTROL sin testigo. | `A3`, `A4`, `A1S/A1ST` | 760, 760, 700/700 |
| C2j | Retirar reloj/testigo y autoridad TID restantes de O3a. | `A0`, `A4`, `A5`, `A1S/A1ST` | 700, 770, 780, 700/700 |
| C2k | Migrar solo la barrera O3b a operación cerrada. | nuevas `B0/B0T`; `B1/B1T` | 360/520; B1 428→máx. 690; B1T máx. 760 |
| C2l | Migrar solo emisión/cierre de ticket; cero permiso primero. | `B0/B0T`, `B2/B2T` | B0/B0T máx. 620/720; B2 230→máx. 560; B2T máx. 760 |
| C2m | Migrar solo observación STOP e identidad O3b. | `B0/B0T`, `B3/B3T`, `B4/B4T` | cada producción máx. 700; cada prueba máx. 760 |
| C2n | Migrar solo handoff O3b sin lease/TID transferible. | `B0/B0T`, `B5/B5T` | producción máx. 700; pruebas máx. 760 |
| C2o | Migrar solo revalidación O3c y retirar el segundo `WithHandle`. | nuevas `C0/C0T`; `C1/C1T` | 360/520; C1 293→máx. 620; C1T máx. 760 |
| C2p | Migrar solo CONT compuesto; cero `revalidacion.permiso`. | `C0/C0T`, `C2/C2T` | C0/C0T máx. 620/720; C2 100→máx. 360; C2T máx. 760 |
| C2q | Migrar solo observación O3c sin broker/callback. | `C0/C0T`, `C3/C3T` | producción máx. 700; pruebas máx. 760 |
| C2r | Migrar solo handoff O3c sin acreditación TID. | `C0/C0T`, `C4/C4T` | producción máx. 700; pruebas máx. 760 |
| C2s | Hacer que O4a emita solo índice/resultado embebido, sin puntero ni TID. | nuevas `D0/D0T`; `D1/D1T`, `D2/D2T` | D0/D0T 420/620; D1 563→730; D2 253→520; tests máx. 760 |
| C2t | Migrar solo O4b P1 al índice embebido, sin implementar ni ejecutar P2. | `E0/E0T` | E0 428→máx. 690; E0T 710→máx. 790 |
| C2u | Reconciliar solo el orquestador O4c estático y su orden total. | nuevas `F0`, `F1`, `F1T` | 420, 620 y 740; parada 780 |
| C2v | Migrar solo composición O5 y resultado final inerte; retirar registros/tokens. | `A0`, `A2`, `A4`; `OPR` byte a byte | 720, 740 y 780; OPR conserva 798 |

Cada fila es un commit material y una responsabilidad observable. Si una fila
necesita más de sus rutas o excede su parada, se vuelve a contrato; no se
fusionan filas, no se permite edición concurrente y no se reparte una misma
ruta entre productores simultáneos.

Ningún corte toca `OPR` mientras mida 798 líneas; cualquier
necesidad crea una fuente propietaria nueva. Ningún fichero puede llegar a 800.

## C3, C4 y la puerta final C5

C3 actualiza pruebas conductuales y todos los analizadores AST vivos. C4
actualiza mutantes, fuentes verificadas, runners y manifiestos. Solo después
C5 ejecuta el cero final.

Write-set máximo C3/C4:

- `tools/o3a_v5_ast/**` completo;
- `tools/o3b_p7_ast/{README.md,main.go,main_test.go}`;
- `tools/o3c_p6_ast/{README.md,invariantes.go,main.go,main_test.go,retirada.go,seguridad.go}`;
- `tools/o3b_p7_mutantes/**`, `tools/o3b_p7_mutantes_v3a/**` y
  `tools/o3b_p7_mutantes_v3b/**`;
- `tools/o3c_p6_mutantes/{README.md,fusion.go,main.go,main_test.go}`;
- `README.md`, `conductor.sh`, `casos.tsv`, `fuentes.tsv` y nuevos manifiestos
  V26 de `tools/o3b_p7_conductor/` y `tools/o3c_p6_conductor/`;
- pruebas y manifiestos vivos de O3a, O3b, O3c, O4a, O4b y O4c afectados.

Los AST prueban CAS dominante, ausencia de lectura compartida previa, raw y
consolidación en la misma llamada, call graph exacto y cero salida de
autoridad. Los mutantes matan CAS tardío, raw fuera de función, consolidación
omitida, raw funcional convertido en fatal, permiso diferido, callback, TID,
puntero O4a/O4b, retry indebido, Wait duplicado y liberación reordenada.

C5 se ejecuta sobre producción, tests, herramientas, runners y manifiestos
vivos ya actualizados. Excluye solo evidencia histórica V21/V25, que debe
permanecer byte a byte y verificarse por SHA. Exige cero referencias de cada
símbolo nominal del inventario, cero `WithHandle`, cero campo/callback `func`
que transporte efecto, cero mapa/registro de punteros de autoridad, cero TID o
`Gettid` en predicados de autorización, cero permiso pendiente entre retornos,
cero goroutine/canal/interfaz/`unsafe.Pointer`/`uintptr` que lleve autoridad y
cero syscall físico fuera de las firmas exactas autorizadas.

El analizador deriva además canales equivalentes: funciones que devuelven o
escriben por argumento un FD/handle/puntero de capacidad, aliases de slot,
resultados autoconsumibles, generaciones/nonce usados como permiso, closures,
hooks, métodos variables y callbacks. Cualquier hallazgo es NO-GO y vuelve al
nodo propietario; C5 no añade adaptadores.

## V26, revisión material y evidencia no autorreferente

V21 y V25 son historia inmutable. La nueva autoridad es V26 y contiene
manifiestos, rutas, modos, líneas, SHA-256 de fuentes, GOROOT, Go tool,
herramientas, casos, mutantes y resultados normal/race del material exacto.

La secuencia por corte es:

1. commit material limpio de una minitarea;
2. dos revisiones independientes del mismo hash, con reproducción focal;
3. integración solo por Dirección si ambas dan `GO`, `P0=P1=P2=0`;
4. commit de acta/evidencia separado cuyo padre contiene el material;
5. el acta registra el hash material y hashes de artefactos, nunca pretende
   incluir su propio hash ni acredita un árbol distinto.

Las evidencias nuevas viven en directorios V26 nuevos. No se reescriben actas,
ledgers o evidencias históricas para acomodar el candidato.

## C22 derivado del manifiesto y de SHA

El nodo C22 puede tocar solo:

- `tools/o3a_v5_conductor/README.md` y `conductor.sh`;
- runners `conductor_c*.sh` cuyo SHA real deba publicarse;
- nuevo `conductor_c22_lease_closed_ops.sh`;
- `fuentes_v5.tsv`;
- baja de `manifiesto_c01_c21.tsv` y alta de
  `manifiesto_c01_c22.tsv`;
- prueba cerrada O3a necesaria para C22;
- nuevo `evidencia-cnd-c22-lease-closed-ops-r1/`, únicamente en el commit de
  evidencia posterior al material.

`manifiesto_c01_c22.tsv` contiene exactamente C01-C22. Cada fila fija ID, ruta
relativa normalizada del runner, SHA-256 del runner, ruta del manifiesto de
fuentes y su SHA-256. `conductor.sh` no conserva lista paralela: deriva del
manifiesto la lista única, rechaza cardinalidad distinta de 22, huecos,
duplicados, escapes, rutas absolutas, modos o SHA inválidos, y ejecuta cada
runner en normal y `-race`.

C22 verifica el hash material de cada runner antes de ejecutarlo, enlaza V26 y
prueba los veintidós casos sin `SKIP`. La evidencia separada liga manifiesto,
fuentes, runners, modos, resultados, residuos y commit material exacto.

## Matriz mínima de comportamiento futuro

- dos contendientes: solo uno gana el CAS y solo uno produce efecto;
- lectura o mutación compartida anterior al CAS: muerte por AST/mutante;
- callback, permiso, TID, handle o puntero diferido: rechazo estático;
- close/dup/open/pipe: raw y delta coherentes, contradicción fatal+5;
- preflight: cero callback, cero FD residual y preparación inmediata;
- `Start` sin hijo: retirada ordinaria; tupla con hijo: retirada con hijo;
- pidfd primario/opaco/reserva: identidad única, `CLOEXEC`, inventario exacto;
- ticket: primer Write, parciales, hasta ocho EINTR, sufijo exacto y cierre;
- CONT: marca 180 s, un raw incluso EINTR y resultado sellado para O4a;
- O4b: cardinalidad/orden/raw exactos, sin retry ni dinámica antes de P2;
- Wait: cardinal uno y `*exec.ExitError` exacto válido;
- Wait4: WNOHANG, PID positivo o ECHILD, PID cero nunca cierre;
- O4c: ECHILD antes de ESRCH, cierres ordenados, TERMINAL única, observación
  antes de operaciones y cero efecto tras liberación;
- resultado O5: valor inerte sin identidad, recurso o dato personal;
- final C5: cero símbolos y cero canales equivalentes en todo material vivo;
- C22: manifiesto único, SHA verificado y normal/race para C01-C22.

## Seguridad, privacidad, i18n, accesibilidad y límites

Todo permanece privado, Linux/amd64, fail-closed y sin API, red, SQL, HTTP,
Docker, Orquesta, Firecracker, Jailer, deploy o producción. No se registra ni
expone PID, PGID, pidfd, TID, nonce, raw libre, ruta privada, actor, tenant,
secreto o dato personal. Errores externos siguen siendo canónicos y sin
detalles internos.

No hay texto de interfaz, por lo que i18n y accesibilidad no cambian. TERMINAL
conserva su contrato regular 0600 y su codec canónico. La indisponibilidad no
se interpreta como autoridad, terminalidad, cierre o éxito.

La fatalidad no ejecuta cleanup ni E/S posterior. La limpieza ordinaria
mantiene primera causa, secundarios acotados, cierre único, inventario y
privacidad. Ningún fichero supera 799 líneas y cada minitarea fija una parada
inferior a 800.

O4B-P2 continúa bloqueado. Este R3 no autoriza su implementación, pruebas
dinámicas ni señales reales. Tampoco acredita O4c productivo, C22, V26, E2E,
aplicación arrancable o producción.

## Condiciones de NO-GO

Es NO-GO cualquiera de estos hechos:

- uno de los dos dictámenes R2 no queda trazado o un hallazgo se silencia;
- operación con lectura compartida anterior al CAS o efecto fuera de llamada;
- autoridad/consolidación que retorna como error ordinario en vez de fatal+5;
- raw funcional de Start, Wait, ticket, CONT u O4b convertido indebidamente en
  fatal o éxito;
- cualquier símbolo retirado, `WithHandle` o canal equivalente vivo tras C5;
- callback, TID, permiso, puntero, FD, handle o acreditación transportable;
- O4c sin Wait único, ECHILD, ESRCH, orden de cierres o liberación final;
- C1/C2 monolítico, write-set ampliado o fichero de 800 líneas;
- O4B-P2 programado o ejecutado antes de su decisión separada;
- V21/V25 modificados o presentados como V26;
- evidencia en el commit material o acta autorreferente;
- C22 no derivado solo del manifiesto/SHA o sin normal y race;
- uso de datos reales, secretos, red, despliegue o producción.

## Condición de cierre documental

El único cierre posible de esta tarea es documental: fichero UTF-8, modo 0644,
no más de 800 líneas, diff limpio y puertas documentales ligeras. El candidato
requiere después doble revisión independiente del hash exacto. Solo Dirección
puede integrarlo y abrir C1a. No se cierra O3a, O3b, O3c, O4a, O4b, O4c, O5,
C22, V26 ni una vertical productiva mediante este documento.
