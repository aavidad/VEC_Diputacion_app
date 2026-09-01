# Decisión O3A-LEASE-CLOSED-OPS-R4: custodia estable y operaciones privadas

Fecha: 1 de septiembre de 2026

Tarea: `O3A-LEASE-CLOSED-OPS-R4`

Estado: candidato documental; no autoriza código, dinámica, integración,
despliegue ni producción

## Capability y condición de apertura

Este R4 sustituye íntegramente R3 y cierra un contrato implementable para
reemplazar credenciales transferibles por operaciones privadas indivisibles
sobre una raíz O5 estable. La custodia conserva un TID ejecutor privado,
sellado durante la composición con el hilo fijado. Cada operación física
acredita de nuevo ese mismo hilo dentro de su propia invocación y de su slot.

No sale de una operación ninguna credencial, TID, PID, FD, pidfd, `Handle`,
puntero de autoridad, permiso, callback ni raw copiable. El único resultado es
una clase inerte. Los recursos y raws permanecen en la custodia definitiva.

`C1a` solo puede abrir después de dos revisiones independientes `GO` sobre el
hash exacto de este documento, integración por Dirección y CI documental
exigida. `O4B-P2` continúa bloqueado. Este corte no se autoaprueba, no integra y
no cambia porcentajes.

## Genealogía, write-set y lecturas

La base exacta de R4 es
`a63aa2f6e9229742b8746102c1299af6ec8cc474`. El producto local usado solo para
el `merge-tree` es `c8cc4312b063ddca7294dfe27dc673ad1a4676d0`.
El write-set exclusivo es este fichero.

Se leyeron completos `/srv/fabrica/AGENTS.md`, los `AGENTS.md` aplicables, O3,
la especificación común y las autoridades O3a, O3b, O3c, O4a, O4b, O4c y O5,
incluidas sus enmiendas enlazadas. También se leyó íntegro el dictamen remoto
`/tmp/vec-review-o3a-lease-r3-20260901.log` y el segundo dictamen entregado por
Dirección. Los ficheros de `/tmp` no son evidencia durable.

El dictamen remoto fue `NO-GO`, `P0=5`, `P1=2`, `P2=0`. El local resumió
`NO-GO`, `P0=2`, `P1=3`. R4 incorpora todos sus hallazgos: TID privado vivo,
CAS perdedor sin escritura, raíz O5 definitiva, preflight separado, productor
O4a, raw O4b literal, O4c completa, Wait/Wait4 cerrados, migración antes de
retirada y C1--C4 granulares con rutas exactas.

## Autoridades coordinadas, no reemplazadas

| Autoridad | Conserva | Enmienda de este seam |
| --- | --- | --- |
| O3a | preparación, mapa FD, `Start`, pidfd y retirada previa | efectos cerrados sobre la raíz; sin lease, testigo, permiso ni `WithHandle` |
| O3b | barrera, ticket temprano, STOP e identidad | barrera, escritura, cierre, observación y handoff cerrados; cero `primerPermiso` |
| O3c | revalidación, marca, primer CONT y salida | revalidación+CONT indivisibles; cero permiso diferido y segundo `WithHandle` |
| O4a | causa, precedencia, etapa, índice y plazos | prepara cada etapa bajo slot y consume solo clase inerte |
| O4b | raws STOP/TERM/CONT/KILL de P1 | syscall literal dentro de la ejecución de etapa; P2 no se abre |
| O4c | terminalidad, Wait, Wait4, ESRCH, cierres y TERMINAL | orquestador sin efectos más siete operaciones cerradas exactas |
| O5 | composición, preflight, recursos y salida privada | posee la única raíz estable y el TID ejecutor sellado; no registra tokens |

O3a conserva su Wait de retirada antes de entregar. O4c conserva el único Wait
funcional del caso entregado. R4 prevalece sobre O4c exclusivamente para el
PID cero devuelto por `Wait4`: sustituye la clasificación histórica `OCF` por
incidente de cierre, cuarentena y salida 65, sin repetición ni otro `Wait4`.
Ningún otro contrato, causa, precedencia, plazo o clasificación cambia.

## Inventario legacy exhaustivo que debe llegar a cero

La retirada nominal abarca definición, campo, método, llamada, test, AST,
mutante, conductor y manifiesto vivo de:

- `permisoGuardiaO3aM38`, `comenzar`, `comenzarCritico`, `permisoValido`,
  `consolidarFisico`, `consolidarCritico` y `fatalPendiente`;
- `primerPermiso`, `permisoPrimero`, `revalidacionO3cM38.permiso`,
  `permisoContMemoriaValidoO3cM38` y `consolidarContO3cM38`;
- `acreditacionPidfdGoM38`, `preflightPidfdGoM38` y
  `consumirPreflightPidfdGoM38`;
- `testigoVueltaM38`, `celdaVueltaM38`, `relojVueltaM38`, `emitir`, `consumir`
  y el argumento `testigo` de `avanzarArranqueO3aM38`;
- `operarConLeaseBarreraO3bM38`, `syscallLeaseO3cM38` y todo parámetro,
  campo o closure `func` usado para transportar un efecto;
- los dos `Process.WithHandle` actuales y cualquier `WithHandle`, `Handle`,
  `uintptr`, callback o getter que exponga el pidfd opaco;
- `valido`, `sellarFisico`, `transferirCritico` y `liberar`, junto con
  `leaseTransferidaO3bM38`, `observadorTransferidoO3bM38` y equivalentes;
- `leaseGuardiaO3aM38`, `observadorSenalO3aM38`,
  `registroAutoridadO3aM38`, mapas de punteros, generaciones, nonces,
  autopunteros y estados `pending`;
- TID transferible en registro, lease, reloj, celda, entrada, sello O3c,
  autorización O4a/O4b, permiso o resultado;
- `autorizacionEtapaO4aM38`, `tomarEntradaAutoridadO4bM38`,
  `slotPropioAutoridadO4bM38`, `slotCanonicoAutoridadO4bM38`, `pendiente` y
  enlaces autorización↔resultado.

La lista no limita el cierre semántico. También se rechazan alias, interfaces,
canales, goroutines, `unsafe.Pointer`, closures, hooks o nombres nuevos que
permitan diferir autoridad. No se elimina el TID ejecutor privado descrito a
continuación; se eliminan todos sus transportes, getters y copias.

## Raíz O5 estable y custodia no copiable

La raíz final tiene esta forma contractual; los campos funcionales omitidos
siguen embebidos por valor y privados:

```go
type raizCerradaO5M38 struct {
	noCopiar noCopy
	entrada  entradaCerradaO5M38
	custodia custodiaCerradaM38
}

type custodiaCerradaM38 struct {
	operaciones estadoOperacionesCerradasM38
	tidEjecutor int
	fd          recursosFDCerradosM38
	proceso     estadoProcesoCerradoM38
	etapas      estadoEtapasCerradasM38
	terminal    estadoTerminalCerradoM38
}

type estadoOperacionesCerradasM38 struct {
	slot        atomic.Uint32
	estado      atomic.Uint32
	propietario atomic.Uint32
	secuencia   atomic.Uint64
	fatal       celdaFatalCerradaM38
}
```

`ejecutarCasoCerradoO5M38` crea una única raíz con `new`, en su dirección
definitiva, y no la copia, mueve ni publica. Entrada y custodia viven en ese
mismo objeto hasta la liberación final. No existe “slot del bundle”, registro
global, mapa de raíces, caché, singleton ni puntero retornado.

El dueño O5 hace `runtime.LockOSThread` antes de sellar. Durante la composición,
cuando la raíz aún no es observable, ejecuta literalmente `SYS_GETTID`, valida
el raw, escribe una sola vez `tidEjecutor` y activa el estado. El hilo permanece
fijado hasta que O4c haya cerrado y consolidado todo; solo entonces O5 hace
`UnlockOSThread`. Una ruta fatal termina el proceso y no desbloquea.

`tidEjecutor` no tiene getter, no aparece en argumentos o resultados, no se
serializa y no se copia a entrada, etapa, prueba o manifiesto. Su única lectura
permitida ocurre dentro de una operación que ya ganó el slot, después de
consolidar su propio `Gettid` raw.

La entrada de compatibilidad es un wrapper transitorio propiedad de O5. Posee
la raíz durante toda la llamada y conserva el camino legacy solo mientras se
migran consumidores. No convierte custodia nueva en bundle antiguo ni devuelve
su dirección. El camino legacy se elimina únicamente después de la puerta de
cero consumidores; cada commit intermedio compila y supera su focal.

## Protocolo indivisible de toda operación

Toda lectura o mutación compartida y todo efecto físico siguen este orden:

1. comprobar únicamente que el argumento raíz no sea nulo;
2. adquirir por un único CAS `slot: 0→discriminante`;
3. ejecutar literalmente el raw `SYS_GETTID` dentro de la función propietaria;
4. consolidar errno, retorno y validez del `Gettid` bajo el slot;
5. comparar el TID observado con `tidEjecutor` y validar estado, propietario,
   fase, índice, plazo, secuencia, argumentos inertes y preestado;
6. ejecutar el efecto literal autorizado;
7. capturar y consolidar raw, postestado, delta, clase funcional y snapshot;
8. liberar mediante CAS `slot: discriminante→0` como última acción compartida.

El perdedor del CAS no lee ni escribe ninguna otra memoria compartida: no fija
`estado=5`, no toca la celda fatal, no registra, no limpia y no llama a otro
efecto. Termina directamente con estado exterior 65.

Solo el ganador puede declarar contradicción. Una contradicción de autoridad,
TID, secuencia, raw/postestado o consolidación fija, mientras posee el slot,
la celda fatal y `estado=5`; después termina en 65 sin liberar el slot ni hacer
cleanup. Esta autoridad queda envenenada de forma irreversible.

Un raw funcional negativo previsto por la autoridad no es contradicción. Se
clasifica, se consolida y libera el slot. Si una operación contiene más de un
efecto literal autorizado, mantiene el mismo slot y repite antes de cada efecto
los pasos 3--5; nunca readquiere ni llama a otra operación cerrada.

Los orquestadores no adquieren slot, no ejecutan raws, no leen ni mutan la raíz
y deciden solo con clases inertes. Invocan operaciones cerradas en serie,
después de que la anterior haya consolidado. Un helper puro puede transformar
valores locales; no acepta raíz, custodia, FD, PID, TID, handle, callback ni
puntero de autoridad.

## Resultados inertes y raws internos

Cada firma cerrada devuelve un enum de tamaño fijo. Se permiten, según su
propietario, clases como `aplicado`, `retiradaSinHijo`, `entregable`,
`aplazado`, `rawFuncionalNegativo`, `liderRecogido`, `incidenteCierre`,
`ECHILD`, `grupoAusente`, `recursosCerrados`, `terminalEmitido` y `liberado`.

No puede formar parte de un resultado un `error` proveedor, `syscall.Errno`,
PID, TID, número de FD, `WaitStatus`, `Rusage`, `ProcessState`, señal, tiempo
crudo, bytes de CONTROL/TERMINAL, contador, índice mutable, causa copiable que
habilite efecto, puntero o interfaz. Los detalles quedan en celdas privadas de
la raíz y solo la siguiente operación propietaria puede leerlos bajo su slot.

En particular, `ejecutarWait4CerradoO4cM38` conserva PID reaprovechado,
`WaitStatus` y `Rusage` dentro de `estadoTerminalCerradoM38` y devuelve solo
`claseWait4O4cM38`. No existe `resultadoWait4CerradoO4cM38` copiable.

## Firmas exactas, productor y consumidor

Las firmas son privadas al paquete. `indice`, `rol` y `destino` son enums
inertes y acotados; nunca números de recurso. Cada fila tiene productor y
consumidor reales en el DAG, por lo que no se introduce una API huérfana.

| Autoridad | Firma exacta | Productor → consumidor |
| --- | --- | --- |
| O5 | `ejecutarCasoCerradoO5M38(config configuracionInerteO5M38) claseSalidaO5M38` | composición O5 → modo supervisor |
| O5 | `componerRaizCerradaO5M38(r *raizCerradaO5M38) claseComposicionO5M38` | wrapper O5 → preflight O5 |
| O5 | `ejecutarPreflightCerradoO5M38(r *raizCerradaO5M38) clasePreflightO5M38` | composición O5 → orquestador de preparación |
| O5 | `orquestarPreparacionCerradaO5M38(r *raizCerradaO5M38) clasePreparacionO5M38` | preflight válido → operaciones O3a |
| O3a | `cerrarFDCerradoO3aM38(r *raizCerradaO5M38, destino destinoFDCerradoM38) claseFDCerradoM38` | preparación → siguiente clase O3a |
| O3a | `duplicarFDCerradoO3aM38(r *raizCerradaO5M38, origen, destino destinoFDCerradoM38) claseFDCerradoM38` | preparación → apertura |
| O3a | `abrirFDCerradoO3aM38(r *raizCerradaO5M38, destino destinoFDCerradoM38) claseFDCerradoM38` | preparación → pipe |
| O3a | `crearPipeCerradoO3aM38(r *raizCerradaO5M38, par parFDCerradoM38) claseFDCerradoM38` | preparación → `Start` |
| O3a | `iniciarProcesoCerradoO3aM38(r *raizCerradaO5M38) claseStartCerradoO3aM38` | preparación → retirada o reserva |
| O3a | `reservarPidfdCerradoO3aM38(r *raizCerradaO5M38) clasePidfdCerradoM38` | `Start` entregable → sonda |
| O3a | `sondarPidfdCerradoO3aM38(r *raizCerradaO5M38, rol rolPidfdCerradoM38) clasePidfdCerradoM38` | reserva → entrega/retirada |
| O3a | `esperarRetiradaCerradoO3aM38(r *raizCerradaO5M38) claseWaitRetiradaO3aM38` | retirada previa → cierre O3a |
| O3a | `consumirControlCerradoO3aM38(r *raizCerradaO5M38) claseControlCerradoM38` | barrera → preparación/retirada |
| O3a | `entregarCapturaCerradaO3aM38(r *raizCerradaO5M38) claseHandoffCerradoM38` | O3a → O3b |
| O3b | `ejecutarBarreraCerradaO3bM38(r *raizCerradaO5M38) claseBarreraO3bM38` | handoff O3a → ticket |
| O3b | `escribirYCerrarTicketCerradoO3bM38(r *raizCerradaO5M38) claseTicketO3bM38` | barrera → STOP |
| O3b | `observarStopCerradoO3bM38(r *raizCerradaO5M38) claseStopO3bM38` | ticket → identidad |
| O3b | `acreditarIdentidadCerradaO3bM38(r *raizCerradaO5M38) claseIdentidadO3bM38` | STOP → handoff |
| O3b | `entregarContinuacionCerradaO3bM38(r *raizCerradaO5M38) claseHandoffCerradoM38` | O3b → O3c |
| O3c | `revalidarYEjecutarContCerradoO3cM38(r *raizCerradaO5M38) claseContO3cM38` | handoff O3b → observación |
| O3c | `observarSalidaCerradaO3cM38(r *raizCerradaO5M38) claseObservacionO3cM38` | CONT → O4a |
| O3c | `entregarCausaCerradaO3cM38(r *raizCerradaO5M38) claseHandoffCerradoM38` | O3c → O4a |
| O4a | `prepararEtapaCerradaO4aM38(r *raizCerradaO5M38, indice indiceEtapaO4aM38) clasePreparacionEtapaO4aM38` | orquestador O4a → O4b |
| O4b | `ejecutarEtapaCerradaO4bM38(r *raizCerradaO5M38, indice indiceEtapaO4aM38) claseEjecucionEtapaO4bM38` | etapa preparada → O4a |
| O4a | `consumirResultadoEtapaCerradaO4aM38(r *raizCerradaO5M38, indice indiceEtapaO4aM38, clase claseEjecucionEtapaO4bM38) claseResultadoEtapaO4aM38` | O4b → próxima etapa/O4c |
| O4a | `orquestarEtapasCerradasO4aM38(r *raizCerradaO5M38) claseCierreEtapasO4aM38` | handoff O3c → preparar/O4b/consumir → O4c |
| O4c | `confirmarTerminalidadCerradaO4cM38(r *raizCerradaO5M38) claseTerminalidadO4cM38` | O4a → Wait |
| O4c | `ejecutarWaitCerradoO4cM38(r *raizCerradaO5M38) claseWaitO4cM38` | terminalidad → Wait4 o incidente |
| O4c | `ejecutarWait4CerradoO4cM38(r *raizCerradaO5M38) claseWait4O4cM38` | líder recogido → ESRCH |
| O4c | `acreditarGrupoAusenteCerradoO4cM38(r *raizCerradaO5M38) claseGrupoO4cM38` | Wait4 final → cierres |
| O4c | `cerrarRecursosCerradoO4cM38(r *raizCerradaO5M38) claseCierreRecursosO4cM38` | ESRCH → TERMINAL |
| O4c | `escribirYCerrarTerminalCerradoO4cM38(r *raizCerradaO5M38) claseTerminalO4cM38` | recursos cerrados → liberación |
| O4c | `liberarCustodiaCerradaO4cM38(r *raizCerradaO5M38) claseLiberacionO4cM38` | TERMINAL cerrado → salida O5 |
| O4c | `orquestarCierreCerradoO4cM38(r *raizCerradaO5M38) claseSalidaO5M38` | O4a → operaciones O4c en orden |

`orquestarPreparacionCerradaO5M38`, `orquestarEtapasCerradasO4aM38` y
`orquestarCierreCerradoO4cM38` son los tres únicos orquestadores: carecen de
raws y de acceso directo a campos. No adquieren slot. Cada operación que
invocan lo adquiere y libera por sí misma.

## Preflight O5 y sustitución de ambos `WithHandle`

`ejecutarPreflightCerradoO5M38` opera sobre `r.custodia` en su dirección final.
Adquiere su discriminante, acredita el TID sellado y ejecuta, literalmente y
sin helper de autoridad: `pidfd_open(getpid, 0)`, consulta `FD_CLOEXEC`, señal
cero y un único `close`. Consolida inventario basal y ausencia de residuo. No
usa `os.FindProcess`, `WithHandle`, `Release`, callback ni token.

Tras su clase válida libera el slot. Solo entonces el orquestador sin efectos
invoca, una a una, las operaciones O3a. No continúa la preparación dentro del
slot de preflight y no existe adquisición anidada.

Después de `cmd.Start`, `iniciarProcesoCerradoO3aM38` conserva dentro de la
raíz el pidfd primario de `SysProcAttr.PidFD` y el pidfd opaco de Go. El delta
físico exige exactamente ambos, `CLOEXEC` e identidad igual; ningún número o
handle sale. `reservarPidfdCerradoO3aM38` crea luego la reserva en otra
invocación cerrada. El segundo `WithHandle` de O3c se reemplaza por sonda
literal sobre el pidfd custodiado y nunca recibe callback.

## Semántica raw preservada

- `close`: un intento, cero retry; `EBADF` cuando debía estar abierto es
  contradicción. Un cierre funcional esperado se consolida antes de seguir.
- `dup3`/`fcntl(F_DUPFD_CLOEXEC)`, `openat2` y `pipe2`: raw literal, destino
  embebido, `CLOEXEC`, identidad y delta exactos. Cualquier rollback es un
  segundo efecto dentro del mismo slot y reacredita TID antes de ejecutarse.
- `Start`: una sola invocación. Error sin `Process` ni pidfd nuevo es
  `retiradaSinHijo`; error con hijo coherente es `retiradaConHijo`; éxito exige
  `Process`, pidfd primario, opaco y delta coherentes. Solo contradicción es
  fatal+5.
- sonda pidfd: señal cero literal. `nil` acredita vivo; `ESRCH`, terminal;
  cualquier otro errno conserva su clase funcional propietaria. Nunca usa PID.
- ticket O3b: escritura temprana, parciales y hasta ocho reintentos EINTR se
  mantienen dentro de una invocación y un slot; cada write reacredita TID. EOF,
  cierre y postestado se consolidan antes de retornar clase.
- CONT O3c: revalidación de deadline, marca, identidad y estado, raw CONT y
  consolidación son una sola operación. EINTR tiene un intento y cero retry.
  No existe `revalidacion.permiso` ni permiso primero retornado.
- O4b: STOP/TERM/CONT/KILL usan literalmente `pidfd_send_signal` dentro de
  `ejecutarEtapaCerradaO4bM38`. Se elimina
  `enviarSenalPidfdCerradaO4bM38`; no hay helper que acepte autoridad. Una etapa
  TERM+CONT conserva un slot y reacredita TID antes de cada syscall.
- CONTROL/TERMINAL: lecturas, escritura completa, parciales, EINTR, EOF y
  cierres conservan los límites y precedencias de sus autoridades; los bytes y
  offsets no salen de la raíz.

## O4a produce y consume cada etapa

`prepararEtapaCerradaO4aM38` recibe solo un índice acotado. Bajo el slot valida
causa primaria, precedencia, fase anterior, plazo y que el índice sea el único
siguiente; escribe por valor en `r.custodia.etapas` señal, rol pidfd, deadline y
estado preparado. Devuelve únicamente `clasePreparacionEtapaO4aM38`.

La primera etapa y todas las siguientes se producen con esa misma firma. O4b
recibe el índice, no un puntero a etapa. O4a consume después la clase de O4b
bajo un slot nuevo, contrasta el resultado interno y prepara el siguiente
índice o entrega O4c. Ninguna etapa, raw o autorización sale de la raíz.

## O4c completa sin adquisiciones anidadas

`orquestarCierreCerradoO4cM38` no tiene efectos ni inspecciona la custodia. Su
orden por clases inertes es único:

1. `confirmarTerminalidadCerradaO4cM38`;
2. `ejecutarWaitCerradoO4cM38` una vez;
3. solo con `liderRecogido`, `ejecutarWait4CerradoO4cM38` una vez; la operación
   completa el barrido o devuelve incidente, y PID cero prohíbe otro `Wait4`;
4. `acreditarGrupoAusenteCerradoO4cM38` por señal cero y `ESRCH`;
5. `cerrarRecursosCerradoO4cM38` en orden primario, reserva y CONTROL;
6. `escribirYCerrarTerminalCerradoO4cM38`;
7. `liberarCustodiaCerradaO4cM38` y salida inerte O5.

Cada fila es una operación cerrada distinta y secuencial. La anterior libera
antes de que la siguiente adquiera. `cerrarRecursosCerradoO4cM38` es una sola
operación compuesta expresamente autorizada: mantiene su slot durante los tres
cierres y reacredita el TID antes de cada raw. TERMINAL se escribe y cierra en
su propia operación. No hay raw huérfano ni orquestador con efecto.

## Wait y Wait4 exactos

`ejecutarWaitCerradoO4cM38` llama `cmd.Wait` exactamente una vez y no hace
retry. La clasificación bajo slot es:

- `nil` es válido solo con el `cmd.ProcessState` no nulo, perteneciente al
  mismo `Cmd`, terminal y coherente; acredita líder recogido;
- `*exec.ExitError` es válido solo si su `ProcessState` es el mismo estado no
  nulo y terminal del `Cmd`; también acredita líder recogido;
- EINTR consume su único intento. Con estado ausente o no terminal produce
  incidente de cierre, marca cuarentena dentro de la raíz, consolida y entrega
  estado exterior 65; no llama Wait4;
- otro error, o un `ProcessState` ausente/incompleto que no acredite recogida,
  produce igualmente incidente de cierre, cuarentena y exterior 65, sin Wait4;
- evidencia positiva irreconciliable —por ejemplo EINTR u otro error junto a
  un postestado que afirma recogida, `ExitError` de otro estado, o transición
  raw/postestado imposible— es contradicción no acreditable: fatal+5, sin
  liberar autoridad.

Solo las dos primeras clases permiten Wait4. No se infiere éxito de un error ni
se interpreta indisponibilidad como terminalidad.

`ejecutarWait4CerradoO4cM38` mantiene su slot durante el barrido. Antes de cada
`wait4(-1, ..., WNOHANG, ...)` reacredita TID. PID positivo, `WaitStatus` y
`Rusage` se consolidan dentro de la raíz y permiten continuar dentro de esa
misma operación. PID cero produce inmediatamente incidente de cierre,
cuarentena y exterior 65, sin espera, repetición ni otro `Wait4`. `ECHILD`
cierra el barrido; EINTR u otro error siguen la política de incidente de cierre
salvo contradicción raw/postestado, que es fatal+5. La función devuelve solo
`claseWait4O4cM38`.

## DAG compilable y retirada posterior a migración

Todas las rutas de C1 y C2 se escriben completas y literalmente. Las rutas
nuevas se marcan “nueva”; las demás existen en la base y se comprobaron antes
de este contrato. Cada minitarea modifica como máximo tres ficheros, añade como
máximo 200 líneas productivas y mantiene cada fichero por debajo de 800. Si
necesita otra ruta, responsabilidad o presupuesto, se detiene y vuelve a
contrato.

Rutas nuevas propietarias:

- `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_raiz_cerrada_o5.go` (`R5`) y `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_raiz_cerrada_o5_test.go` (`R5T`);
- `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas_fd.go` (`RF`) y `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas_fd_test.go` (`RFT`);
- `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas_proceso.go` (`RP`) y `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas_proceso_test.go` (`RPT`);
- `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas.go` (`RB`) y `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas_test.go` (`RBT`);
- `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas.go` (`RC`) y `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas_test.go` (`RCT`);
- `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas.go` (`RD`) y `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas_test.go` (`RDT`);
- `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/terminalidad_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas.go` (`RT`) y `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/terminalidad_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operaciones_cerradas_test.go` (`RTT`).

### C1: introducir raíz, wrapper y operaciones con consumidor

Cada commit añade la firma, su prueba y una llamada del wrapper transitorio o
del orquestador propietario; ninguna API queda sin consumidor.

| ID | Única responsabilidad observable | Dependencia | Write-set máximo |
| --- | --- | --- | --- |
| C1a | raíz definitiva y wrapper O5 aún sobre camino legacy | R4 integrado | nueva `R5`, `R5T`; `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go` |
| C1b | sellado privado LockOSThread/Gettid | C1a | `R5`, `R5T` |
| C1c | CAS, perdedor sin escritura y fatal del ganador | C1b | `R5`, `R5T` |
| C1d | preflight O5 cerrado | C1c | `R5`, `R5T` |
| C1e | orquestador O5 sin efectos | C1d | `R5`, `R5T` |
| C1f | cierre FD O3a | C1c | nueva `RF`, `RFT`; `R5` |
| C1g | duplicación FD O3a | C1f | `RF`, `RFT`; `R5` |
| C1h | apertura FD O3a | C1g | `RF`, `RFT`; `R5` |
| C1i | creación de pipe O3a | C1h | `RF`, `RFT`; `R5` |
| C1j | `Start` cerrado O3a | C1e,C1i | nueva `RP`, `RPT`; `R5` |
| C1k | reserva pidfd, sin sonda | C1j | `RP`, `RPT`; `R5` |
| C1l | sonda pidfd, sin reserva | C1k | `RP`, `RPT`; `R5` |
| C1m | Wait de retirada O3a | C1j | `RP`, `RPT`; `R5` |
| C1m1 | consumo CONTROL O3a | C1c | `RP`, `RPT`; `R5` |
| C1m2 | handoff O3a→O3b | C1m1 | `RP`, `RPT`; `R5` |
| C1n | barrera O3b | C1m2 | nueva `RB`, `RBT`; `R5` |
| C1o | ticket O3b | C1n | `RB`, `RBT`; `R5` |
| C1p | STOP O3b | C1o | `RB`, `RBT`; `R5` |
| C1q | identidad O3b | C1p | `RB`, `RBT`; `R5` |
| C1q1 | handoff O3b→O3c | C1q | `RB`, `RBT`; `R5` |
| C1r | CONT compuesto O3c | C1q1 | nueva `RC`, `RCT`; `R5` |
| C1s | observación O3c | C1r | `RC`, `RCT`; `R5` |
| C1s1 | handoff O3c→O4a | C1s | `RC`, `RCT`; `R5` |
| C1t | productor de etapa O4a | C1s1 | nueva `RD`, `RDT`; `R5` |
| C1u | consumidor de resultado O4a | C1t | `RD`, `RDT`; `R5` |
| C1v | ejecución O4b con raw literal | C1t | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/senales_grupo_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/senales_grupo_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad_test.go`; `RD` |
| C1v1 | orquestador O4a sin efectos | C1t,C1u,C1v | `RD`, `RDT`; `R5` |
| C1w | terminalidad O4c | C1c | nueva `RT`, `RTT`; `R5` |
| C1x | Wait O4c exacto | C1w | `RT`, `RTT`; `R5` |
| C1y | Wait4 O4c sin raws de salida | C1x | `RT`, `RTT`; `R5` |
| C1z | ESRCH O4c | C1y | `RT`, `RTT`; `R5` |
| C1aa | cierre primario/reserva/CONTROL | C1z | `RT`, `RTT`; `R5` |
| C1ab | escritura y cierre TERMINAL | C1aa | `RT`, `RTT`; `R5` |
| C1ac | liberación O4c | C1ab | `RT`, `RTT`; `R5` |

Las dependencias de la tabla son mínimos funcionales. Como los write-sets C1
se solapan, su orden de edición efectivo es total y obligatorio:
`C1a -> C1b -> C1c -> C1d -> C1e -> C1f -> C1g -> C1h -> C1i -> C1j ->
C1k -> C1l -> C1m -> C1m1 -> C1m2 -> C1n -> C1o -> C1p -> C1q -> C1q1 ->
C1r -> C1s -> C1s1 -> C1t -> C1u -> C1v -> C1v1 -> C1w -> C1x -> C1y ->
C1z -> C1aa -> C1ab -> C1ac`. No se abren dos nodos C1 a la vez.

### C2: migrar un consumidor por commit, sin retirar legacy

| ID | Única responsabilidad observable | Dependencia | Write-set literal máximo |
| --- | --- | --- | --- |
| C2a | consumidor de preflight, sin tocar `WithHandle` | C1d | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go`, `R5` |
| C2a1 | primer `WithHandle`, sin cambiar preflight | C2a | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go`, `R5` |
| C2b | cierres O3a | C1f | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go`, `RF` |
| C2c | duplicación O3a | C1g | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go`, `RF`, `RFT` |
| C2d | apertura O3a | C1h | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go`, `RF`, `RFT` |
| C2e | pipe O3a | C1i | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go`, `RF`, `RFT` |
| C2f | `Start` O3a | C1j | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go`, `RP`, `RPT` |
| C2g | delta pidfd, sin reserva ni sonda | C2f | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go`, `RP`, `RPT` |
| C2h | reserva pidfd, sin delta ni sonda | C1k,C2g | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go`, `RP`, `RPT` |
| C2i | sondas pidfd, sin delta ni reserva | C1l,C2h | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go`, `RP`, `RPT` |
| C2j | Wait de retirada O3a | C1m | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go`, `RP`, `RPT` |
| C2k | consumidores CONTROL | C1m1 | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go`, `R5` |
| C2k1 | handoff O3a→O3b | C1m2,C2i,C2k | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go`, `RP`, `R5` |
| C2l | barrera O3b | C1n | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_barrera.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_barrera_test.go`; `RB` |
| C2m | ticket O3b, sin retirar `primerPermiso` | C1o,C2l | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_ticket.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_ticket_test.go`; `RB` |
| C2m1 | consumidor de `primerPermiso`, sin cambiar ticket | C2m | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_ticket.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_ticket_test.go`; `RB` |
| C2n | STOP O3b, sin identidad | C1p,C2m1 | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_stop.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_stop_test.go`; `RB` |
| C2o | identidad O3b, sin STOP | C1q,C2n | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_identidad.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_identidad_test.go`; `RB` |
| C2p | handoff O3b | C2o | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff_test.go`; `RB` |
| C2q | revalidación O3c, sin tocar `WithHandle` | C1r,C2p | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_revalidacion.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_revalidacion_test.go`; `RC` |
| C2q1 | segundo `WithHandle`, sin cambiar revalidación | C2q | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_revalidacion.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_revalidacion_test.go`; `RC` |
| C2r | CONT y permiso diferido | C2q1 | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_cont.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_cont_test.go`; `RC` |
| C2s | observación O3c | C1s,C2r | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_observacion.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_observacion_test.go`; `RC` |
| C2t | handoff O3c | C2s | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff_test.go`; `RC` |
| C2u | productor O4a | C1t,C2t | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_etapas_test.go`; `RD` |
| C2v | consumidor O4a | C1u,C2u | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/causa_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad_test.go`; `RD` |
| C2w | O4b P1 sin helper raw | C1v,C2v | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/senales_grupo_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/senales_grupo_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad_test.go`; `RD` |
| C2w1 | orquestador O4a sin efectos | C1v1,C2w | `RD`, `RDT`, `R5` |
| C2x1 | consumidor de terminalidad O4c | C1w,C2w1 | `RT`, `RTT`, `R5` |
| C2x2 | consumidor de Wait O4c | C1x,C2x1 | `RT`, `RTT`, `R5` |
| C2x3 | consumidor de Wait4 O4c | C1y,C2x2 | `RT`, `RTT`, `R5` |
| C2x4 | consumidor de ESRCH O4c | C1z,C2x3 | `RT`, `RTT`, `R5` |
| C2x5 | consumidor de cierres O4c | C1aa,C2x4 | `RT`, `RTT`, `R5` |
| C2x6 | consumidor de TERMINAL O4c | C1ab,C2x5 | `RT`, `RTT`, `R5` |
| C2x7 | consumidor de liberación O4c | C1ac,C2x6 | `RT`, `RTT`, `R5` |
| C2x8 | orquestador O4c sin efectos | C2x7 | `RT`, `RTT`, `R5` |
| C2y | consumidor raíz del modo supervisor | C2x8 | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operativo.go`, `R5`, `R5T` |

Los write-sets C2 se solapan por bloques; por ello también se serializan en
orden total: `C2a -> C2a1 -> C2b -> C2c -> C2d -> C2e -> C2f -> C2g ->
C2h -> C2i -> C2j -> C2k -> C2k1 -> C2l -> C2m -> C2m1 -> C2n -> C2o ->
C2p -> C2q -> C2q1 -> C2r -> C2s -> C2t -> C2u -> C2v -> C2w -> C2w1 ->
C2x1 -> C2x2 -> C2x3 -> C2x4 -> C2x5 -> C2x6 -> C2x7 -> C2x8 -> C2y`.
La columna de dependencia añade prerrequisitos funcionales, pero nunca permite
ejecución concurrente de dos nodos C2.

Cada ruta existente de C2 queda expandida y exige `test -f` antes del corte.
`operativo.go` mide 798 líneas: C2y no lo amplía; extrae el adaptador a `R5` y
reduce el fichero. Si no puede, vuelve a contrato.

### C3: AST exacto, una responsabilidad por corte

Todas estas rutas existen en la base. Si cualquiera falta, se para; no se crea
un sustituto. Ninguna fila toca evidencia histórica.

| ID | Único cierre | Rutas literales exactas |
| --- | --- | --- |
| C3a | dominancia CAS/TID privado O3a-O5 | `tools/o3a_v5_ast/main.go`; `tools/o3a_v5_ast/README.md`; `tools/o3a_v5_ast/ejecutar_p1.sh` |
| C3b | call graph O3a-O5 y firmas usadas | `tools/o3a_v5_ast/validar_manifest/main.go`; `tools/o3a_v5_ast/manifest_parcial.json`; `tools/o3a_v5_ast/manifest_seguridad_pendiente_target.json` |
| C3c | operaciones cerradas O3b | `tools/o3b_p7_ast/main.go`; `tools/o3b_p7_ast/main_test.go`; `tools/o3b_p7_ast/README.md` |
| C3d | CAS/raw/consolidación O3c | `tools/o3c_p6_ast/invariantes.go`; `tools/o3c_p6_ast/main.go`; `tools/o3c_p6_ast/main_test.go` |
| C3e | cero permiso/callback O3c | `tools/o3c_p6_ast/retirada.go`; `tools/o3c_p6_ast/seguridad.go`; `tools/o3c_p6_ast/README.md` |

Los AST enumeran las firmas de este documento y prueban CAS dominante, ausencia
de acceso previo, `Gettid` raw+consolidación+comparación, efecto literal,
consolidación final, CAS perdedor sin escritura, fatal del ganador sin release,
cero helper O4b y cero raw/autoridad en resultados.

### C4: mutantes, conductores y manifiestos exactos

| ID | Único cierre | Rutas literales exactas |
| --- | --- | --- |
| C4a | catálogo mutante O3a/O5 | `tools/o3a_v5_ast/catalogo_m57_m62.json`; `tools/o3a_v5_ast/catalogo_expansion_m57_m63_m66.json`; `tools/o3a_v5_ast/manifest_parcial.json` |
| C4b | validador mutantes O3b | `tools/o3b_p7_mutantes/validar.sh`; `tools/o3b_p7_mutantes/README.md` |
| C4c | motor mutante O3b v3a | `tools/o3b_p7_mutantes_v3a/main.go`; `tools/o3b_p7_mutantes_v3a/README.md` |
| C4d | catálogo mutante O3b v3b | `tools/o3b_p7_mutantes_v3b/main.go`; `tools/o3b_p7_mutantes_v3b/catalogo.tsv`; `tools/o3b_p7_mutantes_v3b/README.md` |
| C4e | motor mutante O3c | `tools/o3c_p6_mutantes/main.go`; `tools/o3c_p6_mutantes/main_test.go`; `tools/o3c_p6_mutantes/fusion.go` |
| C4f | contrato mutante O3c | `tools/o3c_p6_mutantes/README.md` |
| C4g | conductor O3a y fuentes | `tools/o3a_v5_conductor/conductor.sh`; `tools/o3a_v5_conductor/fuentes_v5.tsv`; `tools/o3a_v5_conductor/README.md` |
| C4h | manifiesto y casos O3a vivos | `tools/o3a_v5_conductor/manifiesto_c01_c21.tsv`; `tools/o3a_v5_conductor/conductor_c15_c21.sh`; `tools/o3a_v5_conductor/conductor_c19.sh` |
| C4i | conductor O3b | `tools/o3b_p7_conductor/conductor.sh`; `tools/o3b_p7_conductor/casos.tsv`; `tools/o3b_p7_conductor/README.md` |
| C4j | manifiesto de fuentes O3b | `tools/o3b_p7_conductor/fuentes.tsv` |
| C4k | conductor O3c | `tools/o3c_p6_conductor/conductor.sh`; `tools/o3c_p6_conductor/casos.tsv`; `tools/o3c_p6_conductor/README.md` |
| C4l | manifiesto de fuentes O3c | `tools/o3c_p6_conductor/fuentes.tsv` |

`C4a` depende además de `C3b`: ambos escriben
`tools/o3a_v5_ast/manifest_parcial.json` y nunca se abren en paralelo. Los
demás C3/C4 solo pueden paralelizarse cuando sus tres rutas literales son
disjuntas; cualquier solape nuevo obliga a añadir una arista antes de editar.

Los mutantes matan CAS tardío, escritura del perdedor, TID ausente o expuesto,
`Gettid` fuera de invocación, raw antes de comparación, consolidación omitida,
callback, permiso diferido, helper O4b, preparación O4a ausente, raw O4c
huérfano, Wait duplicado, EINTR reintentado, Wait4 con raw de salida y cierre
reordenado. Ninguna fórmula abierta, brace expansion o glob define cobertura.

### Z1, retirada y C5

Tras C2y y C3/C4 verdes, `Z1` acredita cero consumidores legacy permitiendo
solo sus definiciones aún compilables. Después, commits separados retiran en
orden: permisos; primer permiso/CONT; acreditación preflight y ambos
`WithHandle`; testigos; lease/observador/registro; autorización O4a/O4b; y el
wrapper O5 legacy. Cada retirada toca como máximo tres rutas, compila y supera
su focal. Ninguna definición se elimina antes de `Z1`.

`C5` exige cero referencias finales a todos los símbolos inventariados, cero
TID salvo `custodiaCerradaM38.tidEjecutor` y sus lecturas intrafunción, cero
PID/FD/raw en resultados, cero callback/handle/puntero de autoridad, cero raw
fuera de firmas exactas y cero API declarada sin consumidor. También ejecuta
AST, mutantes, conductores, normal/race y calidad aplicables. Un fallo vuelve al
nodo propietario; C5 no corrige código.

Dependencia total:

```text
R4 doble GO + integración + CI documental
  -> C1a -> C1b -> C1c
  -> C1d..C1ac, cada rama según su columna
  -> C2a..C2y, en dependencias declaradas y sin retirada
  -> C3a..C3e y C4a..C4l
  -> Z1 cero consumidores legacy
  -> retiradas granulares compilables
  -> C5 cero final
  -> V26 material -> doble GO -> integración -> acta separada
  -> C22
```

## V26 y acta separada

V21 y V25 son historia inmutable. V26 fija rutas, modos, líneas, SHA-256 de
fuentes, GOROOT, Go tool, herramientas, casos, mutantes y resultados normal y
race del material exacto. No modifica actas, ledgers ni evidencia histórica.

La secuencia es: commit material limpio; dos revisiones independientes del
mismo hash; integración solo por Dirección con `P0=P1=P2=0`; después commit de
acta/evidencia separado cuyo padre ya contiene el material. El acta registra el
hash material y hashes de artefactos; nunca pretende contener su propio hash ni
acredita otro árbol.

## C22 derivado del manifiesto

C22 conserva el cierre válido de R3. Puede tocar únicamente rutas literales de
`tools/o3a_v5_conductor/`: `README.md`, `conductor.sh`, `fuentes_v5.tsv`, los
runners existentes cuyo SHA cambie, la baja de `manifiesto_c01_c21.tsv`, y las
nuevas `conductor_c22_lease_closed_ops.sh` y `manifiesto_c01_c22.tsv`. La
evidencia nueva `evidencia-cnd-c22-lease-closed-ops-r1/` pertenece solo al
commit de evidencia posterior.

El manifiesto C01--C22 contiene exactamente ID, ruta relativa normalizada del
runner, SHA-256 del runner, ruta del manifiesto de fuentes y su SHA-256.
`conductor.sh` deriva de él la lista única; rechaza cardinalidad distinta de
22, huecos, duplicados, escapes, rutas absolutas, modos o SHA inválidos y
ejecuta cada runner en normal y race. C22 verifica hashes antes de ejecutar,
enlaza V26 y no admite `SKIP`.

## Matriz mínima de aceptación futura

- dos contendientes: uno gana y produce efecto; el perdedor sale 65 sin una
  sola escritura compartida;
- TID distinto, Gettid inválido o comparación omitida: fatal+5 del ganador y
  slot no liberado;
- copia, getter o resultado de TID/PID/FD/handle/callback: rechazo estático;
- preflight trabaja sobre la raíz definitiva, deja cero residuo y libera antes
  de que el orquestador O5 invoque O3a;
- `Start` sin hijo y con hijo conservan sus clases funcionales exactas;
- `primerPermiso`, permiso CONT y ambos `WithHandle`: cero final;
- O4a prepara primera y siguientes etapas por índice; O4b contiene el syscall
  literal; no existe helper raw con autoridad;
- Wait nil y `ExitError` coherentes permiten Wait4; EINTR, otro error o estado
  no acreditable producen incidente 65/cuarentena; contradicción positiva es
  fatal+5;
- PID cero en Wait4 produce incidente 65/cuarentena, sin espera ni otro raw;
- Wait4 no devuelve PID, `WaitStatus` ni `Rusage`;
- O4c acredita ESRCH, cierra primario/reserva/CONTROL, escribe/cierra TERMINAL
  y libera solo en el orden total;
- toda flecha del DAG compila, supera focal y respeta rutas y presupuestos.

## Seguridad, privacidad, i18n y límites

El diseño es denegación predeterminada, no crea autoridad de identidad y no
registra PID, TID, FD, señal, contenido de CONTROL/TERMINAL ni datos personales.
Los fallos exteriores son clases y estado 65, sin detalles internos. No hay
texto de interfaz nuevo; i18n y accesibilidad no cambian.

Este documento no ejecuta Go, Docker, PostgreSQL, dinámica, red ni producción;
no acredita una vertical, aplicación arrancable, cumplimiento o despliegue.
No implementa O4b P2: `O4B-P2` sigue bloqueado hasta su contrato y puertas
propias. El siguiente corte sigue siendo `C1a`, condicionado al doble GO,
integración y CI documental de R4.
