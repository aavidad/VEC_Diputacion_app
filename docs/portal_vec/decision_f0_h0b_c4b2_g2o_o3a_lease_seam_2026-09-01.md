# Decisión O3A-LEASE-SEAM-P0: permisos TID sin syscall oculto

Fecha: 1 de septiembre de 2026.

Estado: **CANDIDATA DOCUMENTAL**. No autoriza editar código, integrar,
publicar, ejecutar O4B-P2 ni declarar producto terminado. Requiere dos
revisiones independientes completas —funcional y de seguridad— con `GO`,
`P0=P1=P2=0`, y su integración posterior por Dirección antes de asignar el
precorte de código O3a.

## Capability, autoridad e invariante

Capability: emitir bajo la autoridad única O3a dos permisos canónicos,
físicamente distintos y de un solo uso: uno exclusivamente para `Gettid` y
otro para un único syscall de efecto ya ligado al TID acreditado, sin syscall
oculto dentro de comenzar, validar o consolidar.

O3a conserva la única autoridad sobre lease, TID, registro, generación,
secuencia, slot y permisos. O4b se limita a elegir, por una autorización O4a
ya consumida, el efecto cerrado y a ejecutar su syscall; no crea ni duplica
autoridad O3a. No existe una segunda autoridad.

Invariante literal:

```text
lease estado 3, slot nil
→ comenzar acreditación TID sin syscall
→ lease 2, permiso Gettid canónico
→ syscall.Gettid único
→ consolidar acreditación sin syscall
→ lease 3, acreditación canónica pendiente
→ comenzar efecto con TID acreditado
→ lease 2, permiso de efecto distinto y ligado
→ syscall único
→ consolidar sin syscall
→ lease 3, slot nil.
```

`runtime.LockOSThread` es precondición viva antes del primer comienzo y no se
libera hasta finalizar el segundo permiso. La acreditación no es una sesión
TID reutilizable: solo enlaza esos dos permisos consecutivos.

## Preflight local anterior a la edición

Se acreditó, sin red ni escritura, lo siguiente:

- worktree y rama exactos, `HEAD`
  `58800913b32e22b0f77eb8d62900d95c452e98fa` y árbol limpio;
- destino ausente, ningún `AGENTS.md` adicional bajo `docs/portal_vec` y
  rutas del write-set posterior existentes;
- log de análisis completo: 16.978 líneas, SHA-256
  `4938b16c933d1c6cf574dd07559fcc388a3a81d69218df8e2e9c0d0c51ff3c4d`;
- decisiones y enmiendas citadas leídas desde sus ficheros; desde la base del
  análisis `f1c7f8f957cf1bfa478414f0cf24702cd49768f2` no cambió ninguna fuente
  O3a/O3b/O3c/O4a/O4b ni el lease: solo aparecieron dos documentos ajenos;
- G6a conserva 559 líneas, modo 0644 y SHA-256
  `9015dff049f04f839920c964a5d8471c1b3f7f9e3dcab339266cf2e13f155bd8`,
  coincidente con `tools/o3a_v5_conductor/fuentes_v5.tsv`;
- producto local, seguimiento remoto conocido y base de esta rama coinciden
  en `58800913b32e22b0f77eb8d62900d95c452e98fa`; el `merge-tree` local de esa
  base no mostró delta.

## Causa exacta y prevalencia

La API actual contradice la regla «un syscall por permiso»: `comenzar`
ejecuta `syscall.Gettid` antes de entregar el permiso y
`consolidarCritico` llama `permisoValido`, que vuelve a ejecutar
`syscall.Gettid`. O4B-P2 tiene prohibido usar `comenzar`, `comenzarCritico`,
`permisoValido`, `consolidarCritico` o cualquier envoltorio que conserve esos
syscalls ocultos.

Prevalecen por responsabilidad las decisiones O3a, O3b, O3c y O4a, la
decisión O4b de señales de grupo y la enmienda O4a/O4b de terminalidad y STOP.
Esta decisión corrige solo el seam propietario de O3a. No cambia causa,
etapas, deadlines, señal, cardinalidad funcional, terminalidad, `Wait`,
limpieza ni resultado O4b.

## Slot, tipos y API privada mínima

`leaseGuardiaO3aM38` incorpora un único slot privado canónico, inicialmente
nulo, mediante un `atomic.Pointer` a una celda O3a. La celda conserva
autoidentidad, lease, registro, generación, las dos secuencias, fase, TID
acreditado, operación, cardinalidad, objetivos y las identidades canónicas de
sus tres artefactos. No existe registro global ni sidecar.

Los tipos privados son distintos y no convertibles:

- `permisoGettidO3aM38`;
- `acreditacionTIDO3aM38`;
- `permisoConTIDAcreditadoO3aM38`;
- `celdaLeaseSeamO3aM38`, solo accesible desde la lease.

Cada artefacto es puntero opaco autoidentificado y debe coincidir por identidad
física con el guardado en el slot. La acreditación es one-shot y no tiene
getter del TID raw, método de efecto, serialización ni conversión a entero.

Las cuatro firmas privadas mínimas quedan fijadas así:

```go
func (l *leaseGuardiaO3aM38) comenzarAcreditacionTID() (
	*permisoGettidO3aM38, bool,
)

func (l *leaseGuardiaO3aM38) consolidarAcreditacionTID(
	permiso *permisoGettidO3aM38,
	tidRaw int,
) (*acreditacionTIDO3aM38, bool)

func (l *leaseGuardiaO3aM38) comenzarConTIDAcreditado(
	acreditacion *acreditacionTIDO3aM38,
	operacion operacionGuardiaO3aM38,
	cardinalidad int,
	objetivos [2]int,
) (*permisoConTIDAcreditadoO3aM38, bool)

func (l *leaseGuardiaO3aM38) consolidarConTIDAcreditado(
	permiso *permisoConTIDAcreditadoO3aM38,
) bool
```

Ninguna de las cuatro funciones ejecuta syscall, E/S, reloj, callback,
closure ejecutora, hook ni función variable. No acepta ni ejecuta una función
de efecto.

## Secuencia, CAS, fase y fallo cerrado

Las fases publicadas del slot son `GETTID_EMITIDO`, `TID_ACREDITADO` y
`EFECTO_EMITIDO`; `CONSUMIDO` y `FATAL` son terminales. Todo objeto se
preasigna antes del primer CAS que pueda hacerlo visible.

1. `comenzarAcreditacionTID` acredita sin syscall autoidentidad, registro,
   generación, estado 3, slot nulo y snapshot físico. Un único CAS gana
   `3→2`; después incrementa una vez la secuencia y publica la celda y el
   permiso Gettid canónicos. Overflow, slot extranjero o fallo tras ganar el
   CAS llevan a estado fatal 5; nunca restauran 3 ni emiten permiso.
2. El llamador ejecuta exactamente un `syscall.Gettid`. La consolidación
   consume el permiso una vez, sin syscall. Solo `tidRaw>0` e igual al TID
   sellado instala el TID dentro de la celda, avanza a `TID_ACREDITADO` y hace
   `2→3`; el slot continúa ocupado. TID cero, incorrecto, copia, forja o duda
   de CAS consume la capacidad y lleva a estado 5, sin acreditación.
3. `comenzarConTIDAcreditado` exige estado 3, fase y slot canónicos, identidad
   exacta de la acreditación y TID interno igual al de la lease. Sin syscall,
   un único CAS gana `3→2`, consume la acreditación, incrementa por segunda
   vez la secuencia y publica un permiso de efecto físicamente distinto,
   ligado a operación, cardinalidad y objetivos. Overflow o duda lleva a 5.
4. El llamador ejecuta exactamente el syscall autorizado. La consolidación
   consume el segundo permiso sin syscall, sella `CONSUMIDO`, elimina por CAS
   el slot exacto y solo entonces hace `2→3`. Si cualquier paso falla, la
   lease queda en 5 o con capacidad irrevocablemente consumida; nunca se
   reconstruye, restaura o devuelve una credencial utilizable.

Nulo o material demostrado como antiguo/ajeno se rechaza antes de CAS y no
afecta un ciclo canónico distinto. Si un objeto no canónico pretende la lease,
generación, secuencia o fase activas, se sella el ciclo actual como fatal.
Una copia superficial falla por autoidentidad; una copia autoidentificada, una
celda reconstruida, una forja same-package y un slot extranjero fallan por no
ser el puntero canónico del slot. Un replay consumido nunca recupera fase,
estado ni secuencia y no puede dañar un ciclo posterior demostrado distinto.

Un fallo de consolidación después del syscall hace que su raw sea no
interpretable y prohíbe otro syscall. Raw cero, cualquier error y `EINTR` del
efecto se consolidan exactamente una vez antes de ser interpretados fuera del
seam; ninguno autoriza retry, fallback al pidfd de reserva ni otro efecto. El
seam no interpreta raw de efecto. `Gettid` cero o distinto se trata únicamente
como TID no acreditado.

## Compatibilidad y guardas de la API anterior

Las firmas y la conducta anterior permanecen intactas cuando el seam no se
usa: el nuevo slot conserva su valor cero nulo. `valido`, `sellarFisico`,
`comenzar`, `comenzarCritico`, `permisoValido`, `consolidarFisico`,
`consolidarCritico`, `fatalPendiente`, `transferirCritico` y `liberar` deben
exigir además `slot == nil`; así no se intercalan con una acreditación
pendiente. No se reescribe ningún consumidor.

O3a, O3b y O3c siguen usando exclusivamente la API antigua y conservan sus
bytes. O4B-P1 y sus pruebas permanecen también byte-inmutables. O4B-P2 será el
primer consumidor del seam y deberá ejecutar exteriormente, en orden,
comienzo Gettid, Gettid único, consolidación, comienzo efecto, syscall único y
consolidación; no llamará a la API antigua.

## Write-set exacto del precorte posterior

Tras doble `GO` de este documento e integración por Dirección, el precorte de
código O3a solo podrá tocar:

1. `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go`;
2. `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/lease_tid_acreditado_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_test.go`;
3. `tools/o3a_v5_conductor/fuentes_v5.tsv`, exclusivamente la nueva longitud
   y SHA-256 de G6a.

O4B-P1 y todos los consumidores O3a/O3b/O3c quedan byte-inmutables. No se
reescriben ledgers, revisiones ni evidencias históricas. G6a parte de 559
líneas y tiene parada dura antes de 800; si el corte no cabe, se vuelve a una
nueva decisión y no se crea otra fuente por conveniencia.

## Matriz adversa obligatoria del precorte

- nominal: estado 3/slot nulo, dos permisos distintos, dos secuencias, un
  syscall exacto por permiso y final 3/slot nulo;
- AST y call graph: cero syscall, E/S, reloj o API antigua dentro del seam, y
  secuencia exterior exacta;
- TID erróneo y cero; nulo, permiso ajeno, forja, copia superficial, copia
  autoidentificada, celda reconstruida, slot extranjero y replay en cada fase;
- carreras de cada comienzo y consolidación, con un único ganador, sin ABA,
  credencial duplicada ni slot huérfano;
- overflow en cada incremento, fallo forzado de cada CAS y fase/secuencia,
  operación, cardinalidad, objetivos, registro o generación adversos;
- fallo post-syscall o de consolidación: raw no interpretable, estado 5 y cero
  syscall posterior;
- raw de efecto cero, error y `EINTR`: un intento, una consolidación, cero
  retry y cero fallback;
- fatalidad black-box: estado 65, EOF/no retorno y stdout/stderr cero;
- focal normal, repetida y `-race`, `go vet`, AST/tipos/DAG y
  `git diff --check`;
- repetición focal de consumidores O3a/O3b/O3c y O4B-P1, sin editar sus
  fuentes ni pruebas.

## Exclusiones, seguridad y rollback lógico

No se abre O4B-P2, STOP estable, TERM-CONT, KILL, resultados totales, O4c,
conductor operativo, proceso real, Docker, PostgreSQL, despliegue, producción,
datos o credenciales. No hay global mutable, callback, closure ejecutora,
hook, reloj, E/S o syscall dentro del seam. No aparece identidad humana, PII,
texto visible, i18n o interfaz; accesibilidad no cambia.

El rollback lógico nunca restaura una acreditación o permiso. Ante duda se
consume la capacidad o se fija estado 5. Un candidato documental o de código
con `NO-GO` no se integra ni se enmienda ocultando su genealogía; se conserva
y se corrige en un nuevo commit revisable. Si el diseño no cabe o exige otro
fichero, se vuelve a decisión. Este contrato no cambia métricas ni estado de
producto.

## Criterios GO/NO-GO y DAG

Este documento puede recibir `GO` solo si dos revisores independientes
confirman autoridad única, invariante literal, secuencia CAS completa,
compatibilidad, write-set, límites y matriz, y si modo 0644, enlaces, ausencia
de marcadores, secretos y `git diff --check` quedan verdes sobre los mismos
bytes. Cualquier syscall oculto, segunda autoridad, getter TID, restauración de
credencial, permiso compartido, retry/fallback, cambio de consumidor/P1,
cuarta ruta, reescritura histórica o ambigüedad de fase es `NO-GO`.

DAG exacto:

```text
esta decisión documental
→ doble revisión funcional y de seguridad
→ precorte de código O3a
→ doble revisión e integración
→ O4B-P2 por separado.
```

El siguiente corte es únicamente la doble revisión de este documento. Ni este
texto ni su commit autorizan todavía editar código: hacen falta los dos `GO`
independientes y la integración de la decisión por Dirección. Integrar el seam
posterior solo desbloqueará O4B-P2; no lo cerrará ni autorizará producción.
