# Decisión O3A-LEASE-SEAM-P0-R1: acreditación y efecto indivisibles

Fecha: 1 de septiembre de 2026.

Estado: **CANDIDATA DOCUMENTAL R1 A DOBLE REVISIÓN**. Sustituye por completo
la propuesta transferible de este mismo fichero en
`5a4fad03035f67b46c248cbbed64cfeda5ec71c7`. No autoriza código, pruebas,
manifiestos, CI, integración, publicación, O4B-P2, producción ni despliegue.
Solo una doble revisión documental independiente con `GO`, `P0=P1=P2=0`,
seguida de integración y publicación por Dirección, permite crear el precorte
de código definido al final.

## Base y preflight factual

La base material exacta y limpia de R1 es
`5a4fad03035f67b46c248cbbed64cfeda5ec71c7`, cuyo padre es
`58800913b32e22b0f77eb8d62900d95c452e98fa`. La rama de trabajo de este
documento es `trabajo/o3a-lease-seam-p0-contrato-20260901`; su único write-set
es este fichero.

Entre `f1c7f8f957cf1bfa478414f0cf24702cd49768f2` y
`58800913b32e22b0f77eb8d62900d95c452e98fa` aparecieron exactamente cuatro
rutas:

1. `docs/portal_vec/ct_lite_o5_01_web_a_contrato_asignacion_2026-08-31.md`;
2. `docs/portal_vec/decision_o6_03_pre_cap_orden_terminal_autorizada_v2_2026-09-01.md`;
3. `web/static/portal-empleado/modulos/contratacion-temporal/contrato-asignacion.js`;
4. `web/static/portal-empleado/modulos/contratacion-temporal/contrato-asignacion.test.mjs`.

Son dos documentos, el JavaScript O5 y su prueba. El delta contiene cero bytes
O3a/O3b/O3c/O4a/O4b. En la base, G6a sigue teniendo 559 líneas, modo `0644` y
SHA-256
`9015dff049f04f839920c964a5d8471c1b3f7f9e3dcab339266cf2e13f155bd8`.
Esa huella continúa activa en los cuatro consumidores que el futuro corte debe
actualizar conjuntamente:

- `tools/o3a_v5_conductor/fuentes_v5.tsv`;
- `tools/o3b_p7_conductor/fuentes.tsv`;
- `tools/o3c_p6_conductor/fuentes.tsv`;
- `tools/o3c_p6_mutantes/main.go`.

No se ha usado red para convertir referencias locales en hechos remotos. El
preflight de R1 no acredita publicación, CI ni producto arrancable.

## Hallazgos que invalidan la propuesta anterior

### P0 funcional: la acreditación era transferible

Un puntero, permiso o acreditación devuelto por
`comenzarConTIDAcreditado` puede pasar a otra goroutine. La goroutine receptora
puede ganar `3→2` y ejecutar el efecto desde otro TID. Que la goroutine
original mantenga `runtime.LockOSThread` no vuelve intransferible el objeto ni
acredita el TID de la goroutine que ejecuta el efecto.

Por tanto, el patrón separado
`check → token/acreditación → efecto → consolidación` es insuficiente. También
lo es ocultarlo tras un callback o closure: el llamador puede capturar,
desviar, diferir o sustituir el comportamiento y la revisión no puede probar
que el TID acreditado sea el del syscall funcional. Ninguna identidad física,
autoidentidad, secuencia, one-shot o slot corrige esa separación causal.

R1 elimina `permisoGettidO3aM38`, `acreditacionTIDO3aM38`,
`permisoConTIDAcreditadoO3aM38`, `comenzarAcreditacionTID`,
`consolidarAcreditacionTID`, `comenzarConTIDAcreditado` y
`consolidarConTIDAcreditado` del diseño futuro. Esos nombres no se
materializan ni se conservan como compatibilidad.

### P0 de seguridad: existía `pending` con slot nulo

La propuesta anterior hacía primero `3→2` y publicaba después el slot. Esa
ventana permite observar estado pendiente sin celda propietaria; las API
anteriores tampoco tenían una guarda inicial común. Publicar el slot después
del CAS, o retirarlo antes de restaurar el estado, no es corregible mediante
validaciones posteriores.

R1 exige una celda canónica publicada antes de cualquier transición a estado
2. La celda permanece ocupada durante todo el ciclo. El cierre restaura el
estado mientras la celda sigue publicada, declara después el cierre verde y
solo entonces retira el slot. Todo fallo posterior a la publicación deja la
celda en fase fatal y la lease en estado 5; no existe rollback a 1 o 3.

## Prevalencia y reparto de autoridades

Permanecen simultáneamente vigentes:

1. O3a posee registro, generación, TID, lease, slot, mapa FD y ejecución
   física de la primitiva cerrada;
2. O3b posee identidad `/proc`, auto-STOP y las primitivas probatorias que le
   fueron asignadas;
3. O3c posee CONT inicial, primera observación y custodia C5;
4. O4a decide causa, precedencia, reloj, etapa, cardinalidad y autorización;
5. O4b consume la autorización O4a, selecciona etapa, operación y pidfd y
   produce la evidencia física no recolectora.

El ejecutor O3a no elige causa, etapa, plazo, cardinalidad funcional, rol de
pidfd ni política de señal. Su enum cerrado es vocabulario físico del
adaptador, no una segunda autorización funcional. O4b no ejecuta directamente
`Gettid` ni `pidfd_send_signal`, no recibe una acreditación TID y no puede
inyectar una función. O4a sigue sin syscall. O3b y O3c no adquieren esta API.

Esta decisión sustituye únicamente la mecánica O4b antigua que separaba
comienzo, syscall y consolidación para las cuatro señales de grupo. No cambia
las tablas O4a/O4b, la enmienda de terminalidad y STOP final, la selección de
primario/reserva, los deadlines, la evidencia posterior, `Wait`, drenaje,
limpieza o resultado.

## Capability e invariantes no negociables

Capability: ofrecer a O4b un ejecutor O3a privado y finito que, en una sola
invocación síncrona sobre la goroutine llamadora bloqueada al OS thread,
acredite explícitamente su TID actual y ejecute exactamente un
`pidfd_send_signal` de la unión cerrada, sin autoridad transferible entre
acreditación y efecto.

Invariantes:

- no existe token, puntero, permiso, callback, closure, hook, interfaz ni
  función variable entre `Gettid` y el efecto;
- el ejecutor no lanza goroutine ni cambia el bloqueo del OS thread;
- la única llamada `Gettid` y el único syscall funcional ocurren en la misma
  invocación y en la misma goroutine llamadora;
- el slot canónico se publica antes de `3→2` y nunca es nulo mientras el estado
  es 2;
- toda API heredada carga el slot como primer guard y, si no cumple la
  condición que le corresponde —`nil` al comenzar o celda propia al
  consolidar—, termina antes de cualquier `Gettid`;
- toda lectura raw se consolida sin interpretación antes de cualquier
  comparación o clasificación;
- error, `EINTR` o duda consumen el intento: no hay retry, fallback, segundo
  pidfd ni segunda señal;
- cualquier fallo posterior a publicar la celda deja `FATAL` y estado 5;
- el único cierre verde hace `2→3` con el slot ocupado, sella después el cierre
  y retira al final el puntero exacto;
- el resultado raw solo sale del ejecutor después de ese cierre verde; nunca
  transporta autoridad TID.

## Tipos y operación cerrada implementables

G6a añadirá exactamente estos nombres privados, todos con sufijo
`O3aM38`:

```go
type celdaEjecucionLeaseO3aM38 struct { /* sellos privados */ }
type faseCeldaLeaseO3aM38 uint8
type operacionEfectoLeaseO3aM38 uint8
type resultadoEfectoLeaseO3aM38 struct {
	intentadoO3aM38  bool
	retornoRawO3aM38 syscall.Errno
}

const (
	fasePreparadaO3aM38 faseCeldaLeaseO3aM38 = iota + 1
	faseGettidEmitidoO3aM38
	faseGettidConsolidadoO3aM38
	faseEfectoEmitidoO3aM38
	faseEfectoConsolidadoO3aM38
	faseCierreVerdeO3aM38
	faseFatalO3aM38
)

const (
	operacionPararGrupoO3aM38 operacionEfectoLeaseO3aM38 = iota + 1
	operacionTerminarGrupoO3aM38
	operacionReanudarGrupoO3aM38
	operacionMatarGrupoO3aM38
)

func (l *leaseGuardiaO3aM38) ejecutarEfectoConTIDActualO3aM38(
	operacion operacionEfectoLeaseO3aM38,
	pidfd int,
) (resultadoEfectoLeaseO3aM38, bool)
```

La lease añade el campo exacto
`slotEjecucionO3aM38 atomic.Pointer[celdaEjecucionLeaseO3aM38]`. La celda
conserva, como mínimo, autoidentidad, lease, registro, generación, secuencia
reservada, estado de origen 3, fase atómica, operación cerrada, pidfd exacto,
snapshot físico, TID raw y raw de efecto. No guarda autorización O4a,
callback, función, interfaz, texto, PID, referencia humana, reloj o recurso
alternativo. El pidfd recibido no es autoridad TID: antes de publicar la
celda se acredita como objetivo único no negativo y presente en el snapshot
físico sellado de la lease, sin nueva sonda ni syscall. O4b sigue siendo
responsable de haber seleccionado su rol correcto conforme a la autorización
consumida.

La operación se traduce dentro del ejecutor, mediante un `switch` exhaustivo,
a `SIGSTOP`, `SIGTERM`, `SIGCONT` o `SIGKILL`. La única primitiva funcional es
literalmente:

```go
syscall.Syscall6(
	sysPidfdSendSignal,
	uintptr(pidfd),
	uintptr(senalCerrada),
	0,
	pidfdSignalProcessGroup,
	0,
	0,
)
```

`senalCerrada` solo puede proceder del `switch`; `pidfdSignalProcessGroup`
permanece `uintptr(1 << 2)`. No se acepta señal numérica libre, flags, siginfo,
segundo objetivo, función de syscall ni modo de prueba inyectable. Los tests
usan procesos desechables y AST, no sustitutos ejecutores.

## Celda canónica y compatibilidad de la API heredada

`leaseGuardiaO3aM38` incorpora un único
`atomic.Pointer[celdaEjecucionLeaseO3aM38]`. La misma celda gobierna el nuevo
ejecutor y los ciclos de las API heredadas. Así la invariante «estado 2 implica
slot canónico no nulo» se aplica a G6a completo, no solo al camino O4b.

Para `comenzar` y `comenzarCritico`, el orden obligatorio es:

1. cargar el slot como primera operación observable y rechazar si no es nulo;
2. prevalidar y preasignar una celda heredada sin syscall;
3. publicar por CAS `nil→celda`;
4. desde ese punto, cualquier fallo fija fase fatal y estado 5;
5. acreditar el TID conforme al orden raw de esta decisión;
6. ejecutar el CAS autorizado `1/3→2` con la celda todavía publicada;
7. devolver el permiso heredado ligado al puntero exacto de la celda.

`permisoGuardiaO3aM38` añade solo el campo privado exacto
`celdaO3aM38 *celdaEjecucionLeaseO3aM38`. Las firmas de la API heredada y
todos sus llamadores permanecen byte-inmutables.
`permisoValido`, `consolidarFisico`, `consolidarCritico` y `fatalPendiente`
cargan como primer guard el slot y exigen que sea exactamente la celda del
permiso antes de cualquier `Gettid`, snapshot o mutación. En cierre verde,
`consolidarFisico` y `consolidarCritico` restauran `2→estadoPrevio` mientras
el slot sigue ocupado, sellan el cierre y solo después hacen
`slot.CompareAndSwap(celda, nil)`.

`valido`, `sellarFisico`, `transferirCritico` y `liberar` cargan también el
slot como primer guard y exigen `nil` antes de cualquier `Gettid` o cambio de
estado. Un slot no nulo corta la operación. Ninguna API puede saltar a la
validación del TID, tocar el estado o ejecutar un efecto antes de completar su
guard de slot.

Una pérdida de carrera antes de publicar la celda no cambia la lease. Después
de publicarla no existe retorno ordinario de error: overflow, TID inválido,
CAS perdido, fase inesperada, permiso forjado, snapshot divergente o fallo al
cerrar sellan la celda fatal y llevan el estado a 5. La celda fatal no se
retira ni se reutiliza. Una operación expresable por la API nunca deja estado
2 con slot nulo ni estado 1/3 con una celda pendiente utilizable.

## Orden causal indivisible del nuevo ejecutor

El ejecutor no llama a `comenzar`, `comenzarCritico`, `permisoValido`,
`consolidarFisico`, `consolidarCritico` ni `fatalPendiente`. Tampoco es un
envoltorio de la API transferible. Su secuencia completa es:

1. carga el slot como primer guard; prevalida lease, registro, generación,
   estado 3, operación, pidfd y snapshot sin `Gettid` ni efecto;
2. preasigna la celda y calcula la secuencia siguiente en un local, sin mutar
   todavía `l.secuencia` ni hacer autoridad ese valor;
3. gana `slot.CompareAndSwap(nil, celda)`;
4. con la celda publicada, copia la secuencia a la celda, avanza una vez
   `l.secuencia` y gana `estado.CompareAndSwap(3, 2)`; si falla, celda fatal y
   estado 5;
5. avanza a `faseGettidEmitidoO3aM38` (`GETTID_EMITIDO`);
6. ejecuta exactamente una llamada literal `syscall.Gettid()`;
7. copia el raw a la celda y lo consolida sin comparar, validar ni interpretar;
8. avanza a `faseGettidConsolidadoO3aM38` (`GETTID_CONSOLIDADO`);
9. solo ahora exige raw positivo e igualdad con `l.tid` y
   `l.registro.tid`; cualquier divergencia fija fatal+5 y ejecuta cero efecto;
10. avanza a `faseEfectoEmitidoO3aM38`, resuelve el `switch` cerrado y ejecuta
    exactamente el `syscall.Syscall6` anterior;
11. copia el raw de efecto a la celda y lo consolida sin interpretarlo;
12. avanza a `faseEfectoConsolidadoO3aM38`;
13. hace `estado.CompareAndSwap(2, 3)` mientras el slot todavía apunta a la
    celda;
14. avanza a `faseCierreVerdeO3aM38` y solo después hace
    `slot.CompareAndSwap(celda, nil)`;
15. devuelve `resultadoEfectoLeaseO3aM38` y `true` únicamente tras retirar el
    slot exacto.

El orden raw es, literalmente:

```text
GETTID_EMITIDO
→ syscall.Gettid único
→ consolidación raw sin interpretar
→ GETTID_CONSOLIDADO
→ comparación TID
→ EFECTO_EMITIDO
→ pidfd_send_signal único
→ consolidación raw sin interpretar
→ EFECTO_CONSOLIDADO
→ 2→3 con slot ocupado
→ CIERRE_VERDE
→ retirada del slot
→ interpretación exterior por O4b.
```

Nunca se compara el TID antes de `GETTID_CONSOLIDADO`. Nunca se clasifica raw
del efecto dentro del ejecutor. Si el cierre estructural falla, el raw no sale,
no se interpreta y no habilita otra llamada. Transferir la lease, la operación
o el pidfd a otra goroutine tampoco transfiere una acreditación: esa goroutine
ejecutaría su propio `Gettid` dentro de la invocación y fallaría antes del
efecto si no coincide con el TID sellado.

## Write-set máximo exacto del futuro productor

Después del cierre documental completo, el precorte deberá declarar como
máximo estas once rutas nominales y ninguna otra:

1. `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go`;
2. `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/lease_seam_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_test.go`;
3. `tools/o3a_v5_conductor/conductor.sh`;
4. `tools/o3a_v5_conductor/conductor_c22_lease_seam.sh`;
5. `tools/o3a_v5_conductor/fuentes_v5.tsv`;
6. `tools/o3a_v5_conductor/manifiesto_c01_c21.tsv`, exclusivamente para su
   baja;
7. `tools/o3a_v5_conductor/manifiesto_c01_c22.tsv`, exclusivamente para su
   alta como sustituto;
8. `tools/o3b_p7_conductor/fuentes.tsv`;
9. `tools/o3c_p6_conductor/fuentes.tsv`;
10. `tools/o3c_p6_mutantes/main.go`;
11. `docs/portal_vec/revisiones/evidencia_f0_h0b_c4b2_g2o_o3a_lease_seam_p0_codigo_2026-09-01.md`.

Las rutas 6 y 7 forman una sustitución gobernada y no pueden coexistir al
cerrar el candidato. La ruta 11 es evidencia nueva y autocontenida: fija el
commit, las once rutas, sus huellas y los resultados reproducibles, pero no
es revisión independiente ni permite sobrescribir un acta o evidencia
anterior.

G6a conserva parada dura en 800 líneas. La prueba y cada script nuevo quedan
también por debajo de 800. Si la celda universal y el ejecutor cerrado no caben
sin extraer una segunda fuente productiva, se emite `NO-GO` y se vuelve a una
nueva decisión; no se oculta código en tests, conductor, O4b u otra autoridad.

Los tres `fuentes*.tsv` deben fijar la misma longitud y el mismo SHA nuevos de
G6a. `tools/o3c_p6_mutantes/main.go` debe retirar la huella G
`9015dff049f04f839920c964a5d8471c1b3f7f9e3dcab339266cf2e13f155bd8`,
fijar la nueva y adaptar sus mutaciones estructurales a la celda/orden nuevos
sin relajar muertes. Después del candidato, ninguna de esas cuatro autoridades
vivas puede conservar `9015dff…`.

Las evidencias ya versionadas con `9015dff…` permanecen inmutables como
genealogía histórica y no acreditan el candidato nuevo. La evidencia nueva
debe registrar el nuevo G6a en todos los conductores reproducidos; una mezcla
de huellas, un reporte que use el manifiesto anterior o una evidencia histórica
presentada como vigente es `NO-GO`.

## Runner adicional y matriz mínima obligatoria

Añadir un `_test.go` no basta: el conductor O3a actual copia diez fuentes por
prefijo, construye un binario y no ejecuta ese fichero. El futuro
`conductor_c22_lease_seam.sh` debe ejecutar explícitamente el test focal con
las fuentes verificadas, en los modos normal y `-race` que ya gobierna
`conductor.sh`. `conductor.sh` debe exigir once entradas exactas —diez fuentes
existentes y la prueba C22—, invocar C22, registrar sus filas y fallar ante
`SKIP`, `NO-GO`, huella, salida o residuos divergentes. La job CI existente ya
invoca ese conductor; no se modifica `.github/workflows/ci.yml`.

Matriz mínima C22:

- goroutine propietaria bloqueada: TID consolidado igual, un único efecto,
  `2→3`, cierre verde y slot nulo final;
- transferencia a otra goroutine: su TID se consolida antes de comparar,
  falla con celda fatal+estado 5 y ejecuta cero señal;
- AST de dominancia y orden para cada API heredada: carga del slot como primer
  guard y toda salida adversa domina cualquier `syscall.Gettid`;
- AST del ejecutor: un `Gettid`, un `Syscall6`, orden de fases literal, cero
  llamada a la API heredada, callback, closure, función variable, goroutine o
  interpretación raw;
- observación concurrente en cada borde: nunca `estado==2 && slot==nil`;
- carreras de publicación, `3→2`, consolidaciones, `2→3`, cierre y retirada:
  un ganador, sin ABA ni celda huérfana verde;
- operación, pidfd, snapshot, registro, generación, secuencia, fase y TID
  adversos; todo fallo posterior a publicar produce celda fatal+estado 5;
- raw de efecto cero, error y `EINTR`: un intento, una consolidación, cero
  retry/fallback y clasificación solo tras retorno verde a O4b;
- las cuatro señales cerradas usan pidfd, `NULL` y
  `pidfdSignalProcessGroup` exactos; señal o flag libre no compila/no entra;
- regresión completa de O3a, O3b y O3c con sus manifiestos nuevos, y mutantes
  O3c con la nueva huella G; no cuenta una prueba que no haya sido invocada por
  un runner registrado;
- fatalidad black-box conserva 65, EOF/no retorno, stdout/stderr cero y cero
  efecto adicional;
- `gofmt`, focal normal/repetido/`-race`, `go vet`, conductores, mutantes,
  calidad global y `git diff --check` verdes en el futuro corte.

Este documento no ejecuta ninguna de esas pruebas dinámicas.

## Seguridad, exclusiones y paradas

La corrección reduce privilegios: el efecto no puede separarse de la
acreditación del TID ejecutor. No añade identidad humana, dato personal,
secreto, token, log, texto visible, i18n, interfaz, HTTP, SQL, red, Docker o
producción. Accesibilidad no cambia porque no existe superficie de usuario.

Se detiene con `NO-GO` ante cualquiera de estas condiciones:

- permiso/acreditación TID transferible, callback, closure o función de efecto;
- `Gettid` fuera del ejecutor para el nuevo camino, o escondido en comenzar,
  validar o consolidar;
- comparación antes de `GETTID_CONSOLIDADO`;
- estado 2 con slot nulo, retirada previa al cierre verde o restauración antes
  de consolidar el raw;
- fallo posterior a publicación que vuelva a 1/3, retire la celda o permita
  reutilización;
- señal, flags, siginfo, cardinalidad o segundo objetivo libres;
- retry, fallback, segundo syscall o interpretación raw dentro de O3a;
- consumidor vivo con el SHA antiguo, C22 no ejecutado o evidencia mixta;
- cambio de O4a/O4b funcional, O3b/O3c productivo, workflow, manifiesto ajeno,
  código fuera del write-set o fichero productivo mayor de 800 líneas.

O4B-P2 permanece bloqueado incluso después de publicar este documento. El
seam de código tampoco cierra ni abre automáticamente O4B-P2: solo su
integración, publicación y CI 5/5 permitirán a Dirección valorar un candidato
O4B-P2 separado, con contrato y write-set propios.

## DAG, cierre documental y siguiente corte

El DAG exacto e indivisible es:

```text
doble revisión documental
→ integración/publicación por Dirección
→ precorte código
→ productor
→ doble revisión código
→ integración/publicación
→ CI 5/5.
```

No puede adelantarse el precorte, producirse código sobre una decisión sin
publicar, revisar solo una disciplina, integrar antes del doble GO ni declarar
el cierre antes de CI 5/5. `O4B-P2` sigue bloqueado al final de este DAG hasta
una decisión posterior de Dirección.

El siguiente corte es exclusivamente la doble revisión independiente,
funcional y de seguridad, de los bytes exactos de R1. Debe verificar los dos
P0 por separado, los hallazgos secundarios, la compatibilidad de autoridades,
el write-set máximo, el runner ejecutable y estas puertas documentales:

- `git diff --check`;
- existencia y modo de todas las rutas afectadas, sin cambio fuera de este
  documento;
- Gitleaks con el binario exacto `/tmp/vec-gitleaks-20260831`, SHA-256
  `c100de843d374f76143b03487de20fe341fb20cae8a71b6fdff896aec561391d`;
- `merge-tree` contra la integración local conocida, sin marcadores ni
  resolución implícita.

R1 no se autoaprueba ni cambia métricas. Durante este corte están prohibidos
`fetch`, `pull`, `reset`, `push`, despliegue, pruebas dinámicas, producción y
limpieza. Dirección conserva en exclusiva integración, publicación y estado
transversal.
