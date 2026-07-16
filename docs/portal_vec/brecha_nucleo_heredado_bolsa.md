# Brecha del nucleo heredado de Bolsa (`internal/candidate`)

Ejecuta los pasos 1 y 2 de DEC-050 (inventario y analisis de brecha) y el
hallazgo H-04 de la
[auditoria de diseno y seguridad](auditoria_diseno_y_seguridad_2026-07-16.md).
Documento puramente descriptivo: no se ha modificado ningun fichero `.go`. El
paso 3 (porte) y el paso 4 (borrado) quedan pendientes de un trabajo aparte,
solo tras aprobar esta brecha.

## 1. Alcance y metodo

- Se ha leido integramente `internal/candidate/**` (dominio, puertos, casos de
  uso, aplicacion y adaptadores) y se ha comparado capacidad a capacidad con
  `internal/modules/bolsa/**`.
- Cada afirmacion cita el fichero y, cuando aporta precision, la linea o el
  simbolo exportado exacto en el estado del repo a 2026-07-16.
- No se ha tocado `internal/modules/bolsa` (otro agente trabaja ahi en
  paralelo) ni ningun otro fichero de codigo.

## 2. Verificacion de acoplamiento con `go list`

```
go list -f '{{.ImportPath}} {{join .Deps " "}}' ./... | awk '... /internal\/candidate/ ...'
vec-diputacion-granada/cmd/vec-server
vec-diputacion-granada/internal/app/bootstrap
```

Unicos dos paquetes de todo el repo cuyo grafo de dependencias transitivas
incluye `internal/candidate`: `cmd/vec-server` (que solo llama a
`bootstrap.NewHTTPServer`) y `internal/app/bootstrap`. Esto confirma la premisa
de H-04 y DEC-050: nada en `internal/modules/bolsa`, `internal/vec` ni el resto
del arbol depende del nucleo heredado.

El acoplamiento inverso si existe y es relevante para el porte:

- `internal/candidate/domain/procedure.go:9` importa
  `vec-diputacion-granada/internal/modules/bolsa/domain` como `dominiobolsa` y
  en las lineas 18-19 y 25-32 declara `Convocatoria`, `ProcedureState` y las
  constantes `ProcedureState*` como **alias de tipo** de
  `dominiobolsa.Convocatoria` / `dominiobolsa.EstadoConvocatoria`
  (`internal/modules/bolsa/domain/convocatoria.go:36-63`). El heredado ya no
  tiene su propio agregado de convocatoria: reutiliza literalmente el nuevo.
  El comentario de `convocatoria.go:38-40` es explicito: esas constantes
  "mantienen la compatibilidad temporal del prototipo candidate" y no son el
  catalogo gobernado final (eso sigue abierto por DEC-011, fuera de esta
  brecha).
- `internal/candidate/adapters/handler/vec_module.go:7,19,27` y
  `internal/candidate/adapters/handler/http.go:13` importan el paquete raiz
  `internal/modules/bolsa` (alias `bolsamodule`) solo para publicar
  `bolsamodule.ModuleID` y `bolsamodule.ModuleManifestForCandidatePortal()` en
  la ruta `/api/modules/bolsa`.

**Hallazgo relevante**: `internal/modules/bolsa/operational_contract.go:128-184`
define `ModuleManifestForCandidatePortal()`, que declara como rutas "reales"
(`Mode: "real"`) exactamente los endpoints heredados
(`/api/candidates`, `/api/candidates/{id}/documents`, `/api/candidates/{id}/claims`,
`/api/candidates/{id}/notifications`, `/api/notifications/{id}/send`,
`/api/candidates/{id}/audit`, `/api/admin/status`, ...). Es decir: el modulo
nuevo ya contiene, como *contrato declarado*, la lista completa de capacidades
HTTP que hoy sirve el handler heredado, pero ese contrato es documentacion
publicada por el propio heredado (unico llamador confirmado:
`internal/candidate/adapters/handler/vec_module.go:27`) y no tiene ningun
adaptador HTTP propio de `internal/modules/bolsa` detras. Confirmado con
`grep -rln "modules/bolsa" internal/vec/adapters/httpapi/*.go`: solo aparecen
`workspace.go` y `handler.go` (registro de manifiesto de menu), ningun handler
funcional de solicitudes/documentos/alegaciones/avisos.

## 3. Que monta `internal/app/bootstrap` en modo fake

`internal/app/bootstrap/bootstrap.go:50-72` (`NewDemoAPIWithConfig`):

- Si `cfg.AuthMode != config.AuthModeFake` (el valor por defecto es
  `config.AuthModeDisabled`, `config/config.go:61,220`): se monta
  `composeVECShellAPI` (linea 65), que solo registra `/api/vec` (carcasa VEC) y
  la consulta publica de convocatorias de Bolsa
  (`registrarBolsaPublica`, linea 141-144). **El nucleo heredado no se monta
  en absoluto** en este modo; ni siquiera se construye.
- Si `cfg.AuthMode == config.AuthModeFake`: se construye ademas
  `newBolsaAPIWithConfig` (linea 67, definida en 177-219) y se monta con
  `composeAPI` (linea 71, definida en 121-128), que registra la consulta
  publica, `/api/vec` y el handler heredado completo como **comodin de raiz**
  (`mux.Handle("/", fallback)`, linea 126). Todo lo que no sea `/api/vec*` ni
  la ruta publica de convocatorias lo sirve el heredado.

Lo que monta exactamente `newBolsaAPIWithConfig` (bootstrap.go:177-219) en modo
fake:

| Pieza | Constructor | Repositorios segun `cfg.StorageMode` |
| --- | --- | --- |
| Candidatos + meritos + autobaremo | `application.NewCandidateApplicationService` (linea 192) | memoria (`repository.NewCandidateRepository/NewMeritRepository/NewBaremoResultRepository`, linea 239) o fichero durable (`repository.NewDurableFileStore`, linea 232-236) segun `StorageModeFile` |
| Regla de baremo de demostracion | `demoRuleSet("convocatoria-demostracion", "v1")` (linea 188, definida 289-336) | constante en codigo, no gobernada |
| Convocatorias/solicitudes (`ProcedureUseCase`) | `usecases.NewProcedureUseCase` (linea 198) | `demoProcedureRepositories` (linea 221-226): durable o memoria |
| Documentos/alegaciones/avisos/auditoria | `demoAdministrativeFlow` (linea 202, definida 242-264) | `durable.*Repository()` o `repository.NewAdministrativeFlowMemoryStore` |
| Autenticacion | `demoAuthenticator` (linea 206, definida 273-287) | `authadapter.NewFakeAuthenticator()` si `AuthMode != fake`; si `fake`, el almacen de credenciales fake cargado por `cargarCredencialesFake` (unico admitido) |
| Handler HTTP | `handler.NewHTTPHandlerWithModulesAndStatus` (linea 211) | expone `bolsamodule.OperationalStatusForModes(...)` bajo `/api/modules/bolsa` |

Confirma con precision la frase de H-04/DEC-050: "su unico consumidor
restante es `internal/app/bootstrap`, que solo monta la API heredada en modo
`fake`".

## 4. Inventario de capacidades y clasificacion

Cada capacidad se marca **(a)** cubierta por el modulo nuevo, **(b)** falta y
se debe portar, o **(c)** se descarta, con motivo.

### 4.1 Candidatos y meritos (autodeclaracion ciudadana)

- `domain.Candidate` / `domain.NewCandidate`
  (`internal/candidate/domain/candidate.go:6-24`): agregado minimo (ID, DNI,
  Nombre, Email). **(b) falta**: `internal/modules/bolsa/domain` no tiene
  ningun agregado de candidato/aspirante/sujeto natural; solo referencias
  opacas `SujetoRef string` en `baremacion.go` y `llamamientos.go`
  (confirmado: `grep -rln "Candidato\|Aspirante" internal/modules/bolsa/` no
  devuelve ningun fichero de dominio). El nuevo modulo asume que la identidad
  ya viene resuelta por un puerto externo; falta decidir y portar el alta
  minima de un sujeto natural para bolsa si el modulo nuevo no la absorbe de
  otro sitio (p. ej. Personal/RRHH).
- `domain.Merit`, `domain.MeritType`, `domain.MeritState`,
  `domain.MeritData{Meses, Horas, PuntosFijos float64}`
  (`internal/candidate/domain/merit.go:22-159`): maquina de estados mutable
  (`Borrador->Presentado->Validado/Rechazado/Subsanacion`) con puntos en
  `float64`. **(a) cubierto y superado** en el circuito administrativo: el
  concepto equivalente en el modulo nuevo es `BaremacionMerito` con historial
  `[]DecisionTecnica` append-only firmado
  (`internal/modules/bolsa/domain/baremacion.go:812-917`) y `Puntos int64`
  micropuntos (`baremacion.go:34-53`), exactamente la sustitucion que DEC-015
  exige del "estado mutable, `float64` y reglas compiladas" heredado. La
  declaracion en borrador previa a presentar (uso ciudadano) no tiene
  equivalente propio: el modulo nuevo empieza en `AltaMeritoBaremable`
  (`baremacion.go:781-791`), que ya exige un `CalculoOficial` completo
  (motor, huellas, evidencias), no una anotacion libre del candidato. **(b)
  falta** la capa de "declaracion en borrador antes de presentar" si se quiere
  conservar autobaremo ciudadano informal; alternativamente se puede
  **(c) descartar** y sustituirla enteramente por el flujo firmado, pendiente
  de decision expresa (ver 4.9).
- `domain.CalcularAutobaremo` (autodeclaracion, incluye borradores) vs
  `domain.CalcularBaremoOficial` (solo validados)
  (`internal/candidate/domain/baremo.go:147-164`, comentario propio en
  155-159: "El flujo productivo nuevo sustituira el estado mutable... esta
  funcion cierra mientras tanto el fallo del prototipo heredado"). **(a)
  cubierto por diseno superior**: el calculo oficial equivalente en bolsa es
  `CalculoOficialBaremacion` + `ContenidoDecisionTecnica.Validar()`
  (`internal/modules/bolsa/domain/baremacion.go:93-448`), que exige entrada
  reproducible, motor gobernado, huellas y firma; no admite el equivalente al
  "simplemente sumar validados". El autobaremo declarativo (simulacion previa
  a presentar, sin efectos administrativos) no tiene equivalente y queda como
  **(b) falta** si se decide mantenerlo de cara al ciudadano.
- `BaremoRuleSet`/`BaremoMeritRule`/`BaremoSectionCap`
  (`internal/candidate/domain/baremo.go:44-145`): reglas compiladas en Go
  (ver tambien `bootstrap.go:289-336`, `demoRuleSet` con `PointsPerUnit`
  literal en el codigo). **(b) falta**: el equivalente gobernado en bolsa es
  `ReferenciaCriterio`/`ReferenciaReglaCalculo`
  (`internal/modules/bolsa/domain/baremacion.go:56-91`), que referencia una
  regla publicada y versionada por fuera del nucleo (no la reimplementa). No
  existe todavia el adaptador/catalogo que sustituya `demoRuleSet`; es trabajo
  de porte, no solo de borrado.
- `RankSolicitudes` / `Desempate` (mayor experiencia, mayor formacion, letra
  de sorteo, id de candidato como ultimo criterio)
  (`internal/candidate/domain/procedure.go:141-227,280-333` en
  `baremo.go` para `Desempate`, invocado realmente desde
  `internal/candidate/usecases/procedure.go:311,350` en
  `PublicarListadoProvisional`/`PublicarListadoDefinitivo`). **(b) falta por
  completo**: `grep -rln "Desempate\|Sorteo\|Ranking\|Ranked\|desempate"
  internal/modules/bolsa/` no devuelve ningun fichero. El modulo nuevo calcula
  decisiones tecnicas por merito y listas de llamamiento
  (`InstantaneaOrdenBolsa`), pero no tiene ninguna funcion de ordenacion de
  solicitudes por puntuacion total con desempate. Es una capacidad real (con
  tests y con un llamador en produccion dentro del heredado) que debe portarse
  si se van a publicar listados provisionales/definitivos con la misma
  garantia de desempate reproducible.

### 4.2 Convocatorias y solicitudes (procedimiento)

- `usecases.ProcedureUseCase` completo: `CrearConvocatoria`,
  `EnsureConvocatoria`, `RegistrarSolicitud`, `EnsureSolicitud`,
  `PublicarListadoProvisional`, `PublicarListadoDefinitivo`, `ListadoActual`
  (`internal/candidate/usecases/procedure.go:54-361`). **Matiz importante**:
  el unico invocador real de todo este caso de uso es
  `internal/candidate/adapters/handler/demo.go:57-91`, dentro de
  `handleDemoRoute` (solo `POST /api/demo`, solo `requireStaff`,
  `routes.go:22`). No existe ningun endpoint HTTP real de inscripcion
  ciudadana en el heredado: el ciudadano nunca registra una `Solicitud` real
  por API; solo se generan datos sinteticos (`demoSolicitudFixtures`,
  `demo.go:140-186`) para poblar la demo.
  - **(b) falta** la logica de agregado (`Convocatoria`+`RuleSet` combinados
    en `ConvocatoriaRecord`, publicacion de listado provisional/definitivo con
    ranking): no existe agregado "Solicitud" equivalente en
    `internal/modules/bolsa/domain` (confirmado:
    `grep -rln "Solicitud\b" internal/modules/bolsa/` solo encuentra
    `SolicitudRef string` opaco dentro de `baremacion.go` y `llamamientos.go`,
    nunca un tipo `Solicitud` con estados propios). El modulo nuevo no cubre
    el ciclo de vida "Borrador -> Inscrita -> SubsanacionRequerida ->
    Subsanada -> AdmitidaProvisional/ExcluidaProvisional ->
    AlegacionPresentada -> AdmitidaDefinitiva/ExcluidaDefinitiva"
    (`internal/candidate/domain/procedure.go:34-79`).
  - **(a) cubierto** el agregado `Convocatoria` en si (ver 4.1, alias directo).
  - Dado que el unico consumidor de todo el ciclo de solicitud es un
    generador de datos de demostracion staff-only, el porte real no es
    "trasladar el generador de fixtures" sino **construir de cero, en
    espanol y con autorizacion por caso de uso, el ciclo de vida de
    Solicitud** que hoy no existe como endpoint real ni en el heredado ni en
    el modulo nuevo.
- `BolsaState` (`SinConstituir/Provisional/EnAlegaciones/Definitiva/
  Agotada/Cerrada`) con transiciones
  (`internal/candidate/domain/procedure.go:81-122,239-245`).
  **(c) se descarta**: `grep -rn "BolsaState\b" internal/candidate/
  --include=*.go` solo encuentra referencias dentro del propio
  `procedure.go` (definicion) y ningun uso en `usecases/`, `ports/`,
  `adapters/handler/` ni `adapters/repository/`. Es una maquina de estados
  declarada pero nunca conectada a ningun repositorio, caso de uso o ruta:
  codigo muerto del prototipo. El concepto equivalente en bolsa
  (`BolsaConstituida.VigenteEn`, `SituacionParticipacionBolsa` con periodos
  `[Desde,Hasta)` versionados en
  `internal/modules/bolsa/domain/llamamientos.go:44-278`) es estrictamente
  superior (temporal, con huella y referencias, no un enum global) y no
  necesita heredar el enum descartado.

### 4.3 Documentos y evidencia (`ElectronicFile`, `DocumentEvidence`)

- `domain.DocumentEvidence`, `domain.ENIMetadata`, `domain.SignatureEvidence`,
  `domain.ElectronicFile`, `domain.CSV`
  (`internal/candidate/domain/evidence.go:16-291`): modelo de evidencia
  documental con estado antivirus (`AVStatus`), metadatos ENI y manifiesto de
  expediente electronico exportable (`ExportManifest`, linea 239-256).
  **(b) falta** como agregado generico reutilizable: el modulo nuevo no tiene
  ningun tipo `DocumentEvidence`/`ElectronicFile`; su unico concepto de
  evidencia documental es `ReferenciaEvidencia`/`EvidenciaMerito`
  (`internal/modules/bolsa/domain/baremacion.go:211-246`), que es **especifico
  del merito baremado** (liga documento+version+representacion+huella al
  merito), no un expediente electronico general con CSV/ENI/firma reusable
  para cualquier proposito (Alegacion, Notificacion, etc.). El estado
  `AVStatus`/cuarentena (`evidence.go:189-201`) tampoco tiene equivalente
  explicito en bolsa (el nuevo diseno de carga documental vive en DEC-021/022,
  fuera de este modulo, con su propio puerto de almacen — ver
  `docs/portal_vec/almacen_documental_seguro.md`). El porte real de esta
  capacidad probablemente no es "copiar `evidence.go`" sino conectar bolsa con
  el almacen documental seguro ya especificado por DEC-021/DEC-022, que es un
  disenio mas estricto y todavia no conectado a ningun modulo funcional.
- `usecases.AdministrativeFlowUseCase.RegisterCandidateDocument`
  (`internal/candidate/usecases/administrative_flow.go:85-110`) y su ruta
  `POST /api/candidates/{id}/documents`
  (`internal/candidate/adapters/handler/administrative_flow.go:141-192`).
  **(c) se descarta el mecanismo concreto, no la necesidad**: DEC-021 ya cerro
  expresamente este `POST` heredado ("acepta CSV, SHA-256, referencia de
  almacenamiento, sello de tiempo, firma y fecha... construidos por el
  navegador") y decidio "el `POST` heredado de documentos responde con
  servicio no disponible y no persiste nada" — pero eso aplica a la
  superficie HTTP publica bajo `/api/vec/*`, mientras que el heredado bajo
  `/api/candidates/{id}/documents` en modo fake **si sigue aceptando y
  persistiendo evidencia declarada por el cliente** sin verificacion de
  servidor (revisado en `administrative_flow.go:79-104`: construye
  `domain.DocumentEvidence` directamente desde el `administrativeDocumentRequest`
  del cuerpo JSON, sin antivirus real ni firma servidor). Confirma la
  necesidad de no portar este flujo tal cual: cualquier porte debe seguir el
  patron ya decidido en DEC-021/DEC-022 (reserva a cuarentena, huella
  calculada por el servidor, verificacion antes de promocion), no la
  aceptacion directa heredada.

### 4.4 Alegaciones (`Claim`)

- `domain.Claim`, `domain.ClaimState`, `domain.ClaimReceipt`
  (`internal/candidate/domain/claim.go:15-161`): maquina de estados
  (`Presentada -> EnRevision -> Estimada/Desestimada/Archivada`) con recibo
  CSV+hash de auditoria.
  **(b) falta por completo**: `grep -rln "Alegacion" internal/modules/bolsa/`
  no devuelve ningun fichero de dominio o aplicacion (el unico "alegaciones"
  del modulo nuevo es la entrada de menu declarativa en
  `internal/modules/bolsa/manifest.go:41` y en
  `operational_contract.go:143,151`, sin logica detras). El flujo de
  presentacion ya esta cerrado en el heredado por DEC-021 ("la presentacion
  heredada de alegaciones... queda cerrada, porque recibia el CSV probatorio
  del mismo cliente que solicitaba la operacion"), confirmado en
  `internal/candidate/adapters/handler/routes.go:172-180`: el `POST` de
  `handleCandidateClaimsRoute` devuelve `503 api.error.probative_flow_unavailable`
  y no persiste nada; solo el `GET` (listar) sigue activo. El porte necesario
  no es trasladar `PresentClaim` (ya cerrado y con motivo justificado), sino
  disenar de cero, en el modulo nuevo, un flujo de alegaciones con evidencia
  emitida por el servidor (mismo patron DEC-021/022), algo que hoy no existe
  en ningun sitio del repo.

### 4.5 Avisos y notificaciones (`Notification`)

- `domain.Notification`, `domain.NotificationState`,
  `domain.NotificationReceipt`, transiciones `Send`/`MarkRead`
  (`internal/candidate/domain/notification.go:15-187`).
  **(b) falta por completo**: `grep -rln "Notificacion" internal/modules/bolsa/`
  no devuelve ficheros de dominio/aplicacion (solo la entrada de menu en
  `manifest.go:42` y las rutas declarativas en `operational_contract.go`).
  Igual que las alegaciones, DEC-021 ya cerro las transiciones de
  envio/lectura del heredado ("las transiciones de envio/lectura de
  notificaciones tambien quedan cerradas"), confirmado en
  `internal/candidate/adapters/handler/routes.go:270-283`
  (`notificationReceiptRequest` siempre devuelve `503`). La creacion de
  notificacion (`POST /api/candidates/{id}/notifications`,
  `administrative_flow.go` vía `handleCandidateNotificationsRoute`) sigue
  activa y acepta asunto/cuerpo libres del emisor interno (no del ciudadano;
  requiere `requireStaff`, `routes.go:184`), sin generar un CSV real. El
  porte necesario es, de nuevo, un diseno nuevo con recibo emitido por el
  conector de notificaciones, no existente hoy en `internal/modules/bolsa`.

### 4.6 Auditoria (`AuditEntry`, cadena firmada)

- `domain.AuditEntry`, `domain.AuditEnvelope`, `NewAuditEntry`,
  `VerifyAuditChain`, `signAuditEntry`
  (`internal/candidate/domain/audit.go:18-138`): cadena de auditoria
  encadenada por firma HMAC-like (`PrevSignature`/`Signature` derivados con
  `signingRef`), verificable con `VerifyAuditChain`.
  **(a) cubierto conceptualmente por un diseno mas fuerte, pero sin
  sustituto directo activo**: el modulo nuevo no tiene un tipo `AuditEntry`
  generico, pero el patron de append-only firmado con huella encadenada que
  aqui se improvisa (`signAuditEntry`, `audit.go:126-138`, firma simetrica
  ad-hoc con una unica `signingRef` de proceso) esta reemplazado en su dominio
  mas exigente (baremacion) por `DecisionTecnica`/`FirmaDecisionTecnica`
  reales con politica de firma, sello de tiempo y validacion por conector
  (`internal/modules/bolsa/domain/baremacion.go:504-711`). No existe, sin
  embargo, una auditoria *generica* de proposito general (para documentos,
  alegaciones, avisos) en el modulo nuevo: cada dominio tendria que definir su
  propio encadenamiento append-only siguiendo el patron de `DecisionTecnica`
  en vez de reutilizar `domain.AuditEntry`. Se considera **(c) se descarta el
  mecanismo generico heredado** (firma simetrica de proceso, sin politica ni
  conector real) **y (b) falta** definir, cuando se porten alegaciones/avisos,
  su propio encadenamiento siguiendo el patron ya validado de baremacion en
  lugar de reintroducir `AuditEntry`.
- Rutas de consulta `GET /api/candidates/{id}/audit` y `GET /api/audit?...`
  (`internal/candidate/adapters/handler/routes.go:206-213,285-305`):
  **(b) falta** un endpoint de auditoria equivalente y gobernado por
  RBAC+ABAC (DEC-020) en el modulo nuevo; hoy no existe ninguna ruta de
  auditoria funcional para bolsa fuera de la heredada.

### 4.7 Autenticacion `fake` heredada

- `ports.AuthPrincipal`/`AuthRole`/`AuthMechanism`/`AuthCredentials`
  (`internal/candidate/ports/auth.go:23-201`): catalogo cerrado de 4 roles
  (`candidate`, `validator_l1`, `validator_l2`, `system_admin`) y 3 mecanismos
  (`kerberos_ad`, `dnie`, `clave`), con `PrimaryRole()`/`AuthMethod()` que
  deniegan ante alias contradictorios (linea 115-163) — ya alineado con
  DEC-027 ("dos alias autoritativos contradictorios invalidan toda la
  asercion").
- `authadapter.FakeAuthenticator`
  (`internal/candidate/adapters/auth/fake.go:10-125`): registro en memoria sin
  precarga de tokens (linea 21-26, comentario explicito de que "la
  composicion fake real usa exclusivamente el fichero seguro de bootstrap"),
  usado solo cuando `AuthMode != fake` (ver 4.2/seccion 3).
- `authadapter.TrustedHeadersAuthenticator`
  (`internal/candidate/adapters/auth/trusted_headers.go:15-154`): existe pero
  **no se instancia desde `bootstrap.go`** (grep confirma que solo aparece en
  `fake_credentials.go` como referencia de tipo del `AuthRole`, no como
  constructor invocado); DEC-020 ya establece que la API heredada de Bolsa
  "no acepta identidades por cabeceras en modos reales".
  **(c) se descarta** trasladar esta autenticacion tal cual: el modulo nuevo
  no necesita su propio `Authenticator` de 4 roles gruesos porque DEC-020/025
  ya definen el reemplazo (fichero fake seguro de bootstrap con
  `almacenCredencialesFake`, RBAC+ABAC por caso de uso) como la via
  productiva; el `ports.Authenticator` heredado y su `FakeAuthenticator` viven
  y mueren con `internal/candidate`. No hay nada que portar mas alla de lo
  que DEC-020/025/030 ya cubren en la carcasa VEC real.

### 4.8 Persistencia (memoria y `local_durable`/`file`)

- `repository.MemoryStore`/`CandidateRepository`/`MeritRepository`
  (`internal/candidate/adapters/repository/memory.go:18-232`),
  `ProcedureMemoryStore` (`procedure_memory.go:18-224`),
  `AdministrativeFlowMemoryStore` (`administrative_flow_memory.go:23-296`),
  `BaremoResultRepository` (`baremo_result_memory.go:14-66`): adaptadores de
  memoria con mutex, genericos via `indexedMemory[T]`/`sortedKeys[V any]`.
  **(c) se descarta el codigo concreto**: el modulo nuevo ya tiene sus propios
  adaptadores de memoria con contrato mas exigente (reserva/confirmacion con
  huella y sellos —
  `internal/modules/bolsa/adapters/memory/baremacion.go:75-1122`,
  `llamamientos.go:38-290`). No hay valor en portar los repositorios
  genericos heredados; cualquier persistencia nueva (candidato, solicitud,
  alegacion, aviso) deberia seguir el patron transaccional ya usado por
  `RepositorioBaremaciones`, no el patron simple `Save/GetByID/List` heredado.
- `repository.DurableFileStore` (`durable_file.go:20-321`,
  `baremo_result_memory.go:72-239`): snapshot JSON en fichero con backup
  (`copyBackup`, `durable_file.go:304-320`) y recarga (`load`, linea 237-250).
  Es la unica persistencia "durable" real del heredado (`config.StorageModeFile`
  / `local_durable`, `config/config.go:41-42,204-207`).
  **(c) se descarta**: el modulo nuevo se dirige a PostgreSQL con
  transacciones/CAS reales
  (`internal/modules/bolsa/adapters/postgres/baremacion.go:27-919`), un salto
  de madurez que hace innecesario portar un snapshot-a-fichero intermedio.
  Mantenerlo alargaria el doble nucleo sin aportar nada que PostgreSQL no vaya
  a resolver mejor.

### 4.9 API HTTP heredada bajo `/api` (`internal/candidate/adapters/handler`)

Superficie completa (`routes.go:16-48`, `dispatch`): `/`, `/demo`, `/portal`,
`/admin/status`, `/admin/capabilities`, `/modules/bolsa*`, `/notifications*`,
`/audit`, `/candidates`, `/candidates/{id}/{documents|claims|notifications|audit}`,
`/candidates/{id}/{merits|baremo|expediente}` (via `handleCandidateAction`,
`candidates.go:37-124`).

- **(a) cubierto para las rutas ya cerradas por decision expresa**: los `POST`
  de alegaciones (4.4) y las transiciones de notificacion (4.5) ya devuelven
  `503` por DEC-021; no hace falta portar su implementacion, solo el flujo
  sustituto (ver 4.4/4.5, clasificado alli como falta).
- **(b) falta** la practica totalidad de las rutas activas si se quiere una
  API equivalente en espanol bajo `internal/modules/bolsa`: alta de candidato
  (`POST /candidates`), merito (`POST /candidates/{id}/merits`), autobaremo
  (`GET /candidates/{id}/baremo`), expediente (`GET /candidates/{id}/expediente`),
  documentos (`GET/POST /candidates/{id}/documents`), avisos
  (`GET/POST /candidates/{id}/notifications`, `POST /notifications/{id}/send|read`),
  auditoria (`GET /candidates/{id}/audit`, `GET /audit?...`) y el portal
  profesional agregador (`GET /portal`, ver 4.10). Ninguna de ellas tiene hoy
  un handler HTTP dentro de `internal/modules/bolsa/adapters/*`; el unico
  adaptador HTTP del modulo nuevo es
  `internal/modules/bolsa/adapters/httppublico/handler.go` (consulta publica
  de convocatorias, solo lectura, sin autenticacion de candidato).
- **(c) se descarta** la ruta `/demo` (`handleDemoRoute`,
  `routes.go:50-59`) y todo `demo.go`: es un generador de datos sinteticos
  para poblar la interfaz de demostracion (`demoSolicitudFixtures`,
  `demoPuestoID`, etc.), no una capacidad de negocio. Si se necesita un modo
  de demostracion en el modulo nuevo debe construirse siguiendo el contrato de
  datos gobernados (DEC-011), no copiando fixtures compilados.
- **(c) se descarta** `/admin/status` y `/admin/capabilities`
  (`vec_module.go:31-52` via `handleAdminRoute`): ya tienen sustituto real y
  activo en el modulo nuevo:
  `internal/modules/bolsa/operational_contract.go:96-117`
  (`AdminCapabilitiesContract`, `AdminRoutes`, `AdminCapabilityList`) mas
  `OperationalStatusForModes` (linea 71-83), consumidos hoy mismo por el
  propio heredado (`bootstrap.go:217`). Es decir, esta pieza concreta **ya
  esta portada**; lo que falta es solo el cableado HTTP en
  `internal/modules/bolsa` que hoy delega en el heredado para servirla bajo
  `/api/admin/*` (marcar como (a) cubierto en dominio, (b) falta en cableado
  HTTP propio).

### 4.10 Portal profesional agregador y manifiesto operacional

- `application.ProfessionalPortalView` y su constructor
  `NewProfessionalPortalView`
  (`internal/candidate/application/professional_portal.go:5-248`): vista
  agregada de dashboard para el candidato (modulos, capacidades, resumen,
  acciones pendientes, autobaremo explicado, auditoria) usada por
  `GET /api/portal` (`handleProfessionalPortalRoute`,
  `professional_portal.go` del paquete `handler`, linea 56-90).
  **(b) falta enteramente**: no hay ningun agregador de vista equivalente en
  `internal/modules/bolsa`; es logica de presentacion (no de dominio) que
  tendria que reconstruirse contra los datos reales del modulo nuevo cuando
  existan los casos de uso que le dan contenido (documentos, alegaciones,
  avisos, autobaremo).
- `bolsamodule.ModuleManifestForCandidatePortal()`
  (`internal/modules/bolsa/operational_contract.go:128-184`): **(a) cubierto
  como contrato/documentacion**, ya analizado en la seccion 2 — es el modulo
  nuevo describiendose a si mismo con las rutas que hoy sirve el heredado.
  Sirve como especificacion util para el porte (enumera exactamente que
  rutas hay que reconstruir), pero no aporta implementacion.
- `bolsamodule.Manifest()` (menu de la carcasa VEC,
  `internal/modules/bolsa/manifest.go:17-48`): **(a) cubierto**, ya declara
  entradas de menu para dashboard, convocatorias, solicitudes, meritos,
  autobaremo, documentos, alegaciones, notificaciones, listados, auditoria y
  manifiestos — es decir, el espacio de navegacion objetivo ya esta
  reservado; falta la logica y los handlers detras de cada entrada (ver
  4.1-4.6).

## 5. Listas finales

### (a) Ya cubierto por el modulo nuevo (9)

1. Agregado `Convocatoria`/`EstadoConvocatoria` — alias directo
   (`internal/candidate/domain/procedure.go:18-32` ->
   `internal/modules/bolsa/domain/convocatoria.go:36-63`).
2. Calculo oficial de baremo con decision firmada, append-only, enteros
   (`internal/modules/bolsa/domain/baremacion.go:93-917`, sustituye el
   `CalcularBaremoOficial` heredado de
   `internal/candidate/domain/baremo.go:160-164`).
3. Llamamiento/orden de bolsa con evidencia completa
   (`internal/modules/bolsa/domain/llamamientos.go` completo), sustituye al
   `BolsaState` heredado descartado en 4.2.
4. Autorizacion de operaciones de baremacion por accion/recurso/contexto
   (`internal/modules/bolsa/ports/autorizacion_baremacion.go`), muy superior
   al `ports.AuthPrincipal` heredado de 4 roles gruesos.
5. Persistencia transaccional con reserva/CAS en memoria y PostgreSQL
   (`internal/modules/bolsa/adapters/memory/baremacion.go`,
   `adapters/postgres/baremacion.go`), sustituye a los repositorios de
   memoria/fichero heredados (4.8).
6. Contrato de capacidades administrativas (`/api/admin/status`,
   `/api/admin/capabilities`) ya definido en dominio
   (`internal/modules/bolsa/operational_contract.go:96-117`), solo falta
   cableado HTTP propio (ver lista (b), item 12).
7. Manifiesto de menu de la carcasa VEC con las 11 entradas objetivo
   (`internal/modules/bolsa/manifest.go:17-48`).
8. Contrato declarado de rutas HTTP objetivo para el portal del candidato
   (`internal/modules/bolsa/operational_contract.go:128-184`,
   `ModuleManifestForCandidatePortal`) — documentacion util para el porte.
9. Patron de encadenamiento append-only firmado con politica/conector real,
   que reemplaza conceptualmente (no literalmente) la cadena de auditoria
   simetrica del heredado (`FirmaDecisionTecnica`,
   `internal/modules/bolsa/domain/baremacion.go:504-711` vs
   `internal/candidate/domain/audit.go:126-138`).

### (b) Falta y se debe portar/reconstruir (14)

1. Agregado de sujeto/candidato para bolsa (o decision expresa de delegarlo
   a otro modulo) — hoy `internal/modules/bolsa` solo tiene `SujetoRef`
   opaco.
2. Ciclo de vida de `Solicitud` (Borrador..AdmitidaDefinitiva/ExcluidaDefinitiva)
   — no existe agregado `Solicitud` en el modulo nuevo
   (`internal/candidate/domain/procedure.go:34-79`).
3. Reglas de baremo gobernadas (sustituto de `demoRuleSet`,
   `bootstrap.go:289-336`, y de `BaremoRuleSet` compilado en
   `internal/candidate/domain/baremo.go:44-145`) enlazadas con
   `ReferenciaCriterio`/`ReferenciaReglaCalculo`.
4. Ranking de solicitudes con desempate reproducible (`RankSolicitudes`,
   `Desempate`, `internal/candidate/domain/procedure.go:141-333`) — ausente
   por completo en `internal/modules/bolsa` (confirmado por grep).
5. Publicacion de listado provisional/definitivo
   (`PublicarListadoProvisional/Definitivo`,
   `internal/candidate/usecases/procedure.go:221-361`), hoy solo invocada
   desde el generador de fixtures de demo.
6. Expediente documental general con CSV/ENI/firma/antivirus reutilizable
   (`domain.DocumentEvidence`/`ElectronicFile`,
   `internal/candidate/domain/evidence.go`), conectado al almacen documental
   ya especificado por DEC-021/022 en vez de reintroducir el modelo heredado
   sin verificacion de servidor.
7. Flujo de alegaciones con evidencia emitida por el servidor (sustituto del
   `domain.Claim` cerrado por DEC-021,
   `internal/candidate/domain/claim.go`).
8. Flujo de avisos/notificaciones con recibo real de envio/lectura
   (sustituto de `domain.Notification`,
   `internal/candidate/domain/notification.go`, tambien cerrado en sus
   transiciones por DEC-021).
9. Auditoria funcional de proposito general para documentos/alegaciones/avisos,
   siguiendo el patron `DecisionTecnica` en vez de `domain.AuditEntry`.
10. Endpoint de consulta de auditoria por candidato/ambito
    (`GET /candidates/{id}/audit`, `GET /audit?...`,
    `internal/candidate/adapters/handler/routes.go:206-213,285-305`).
11. Portal agregador para el candidato (`ProfessionalPortalView` y su ruta
    `/portal`, `internal/candidate/application/professional_portal.go`).
12. Cableado HTTP propio en `internal/modules/bolsa` para `/api/admin/status`
    y `/api/admin/capabilities` (el dominio ya existe, ver (a) item 6; falta
    el adaptador HTTP que hoy delega en el heredado).
13. Endpoints HTTP en espanol para candidato/merito/autobaremo/expediente
    (`POST /candidates`, `POST /candidates/{id}/merits`,
    `GET /candidates/{id}/baremo`, `GET /candidates/{id}/expediente`) dentro
    de `internal/modules/bolsa/adapters`.
14. RBAC+ABAC real (DEC-020) para todas las rutas anteriores, sustituyendo
    los 4 roles gruesos de `ports.AuthPrincipal`
    (`internal/candidate/ports/auth.go:40-59`).

### (c) Se descarta expresamente (9)

1. `ports.Authenticator`/`FakeAuthenticator`/`TrustedHeadersAuthenticator`
   heredados (`internal/candidate/ports/auth.go`,
   `adapters/auth/fake.go`, `adapters/auth/trusted_headers.go`) — DEC-020,
   DEC-025 y DEC-030 ya definen el reemplazo productivo (fichero fake seguro
   de bootstrap + RBAC+ABAC); no hay nada que portar.
2. `BolsaState` (enum global de la lista) — codigo muerto, sin ningun uso
   fuera de su propia definicion (`internal/candidate/domain/procedure.go:81-122,239-245`).
   Sustituido conceptualmente por `SituacionParticipacionBolsa`, muy superior.
3. Repositorios de memoria genericos heredados
   (`internal/candidate/adapters/repository/memory.go`,
   `procedure_memory.go`, `administrative_flow_memory.go`,
   `baremo_result_memory.go`) — el patron transaccional del modulo nuevo
   (reserva/CAS con huellas) es el que debe usarse en cualquier porte.
4. `DurableFileStore` (snapshot JSON a fichero,
   `internal/candidate/adapters/repository/durable_file.go`) — el destino es
   PostgreSQL con transacciones reales, no un fichero intermedio.
5. Ruta `/demo` y todo el generador de fixtures
   (`internal/candidate/adapters/handler/demo.go`) — datos sinteticos
   compilados, no una capacidad de negocio a preservar.
6. Mecanismo concreto del `POST` heredado de documentos que acepta CSV/SHA-256/firma
   declarados por el cliente (`administrative_flow.go:79-104`) — ya
   contradice DEC-021/022; se descarta el mecanismo, no la necesidad
   funcional (que pasa a la lista (b), item 6).
7. Presentacion heredada de alegaciones y transiciones de notificacion tal
   como estan implementadas (`routes.go:172-180,270-283`) — ya cerradas por
   DEC-021 con `503`; se descarta el codigo, la necesidad pasa a (b) items 7-8.
8. Cadena de auditoria simetrica ad-hoc de proceso
   (`internal/candidate/domain/audit.go`, `signAuditEntry` con una unica
   `signingRef`) — sin politica de firma, conector ni sello de tiempo real;
   el patron a seguir es `FirmaDecisionTecnica`.
9. Autobaremo declarativo tal cual (borrador libre del candidato sumado como
   simulacion, `CalcularAutobaremo` en
   `internal/candidate/domain/baremo.go:147-154`) si la decision del
   responsable funcional es que el modulo nuevo empiece directamente en
   `AltaMeritoBaremable` firmada, sin fase de simulacion previa — pendiente de
   decision expresa antes de portar (no se asume aqui; ver hallazgo 6.3).

## 6. Hallazgos inesperados

1. **El heredado ya depende del modulo nuevo, no solo al reves.**
   `internal/candidate/domain/procedure.go` importa
   `internal/modules/bolsa/domain` para `Convocatoria`/`EstadoConvocatoria`
   (seccion 2). El borrado final de `internal/candidate` no es un corte
   limpio en una sola direccion: hay que verificar que ningun otro tipo
   heredado dependa igual del modulo nuevo antes de retirarlo (en este
   analisis solo se encontro este caso mas el uso de `bolsamodule.ModuleID`/
   `ModuleManifestForCandidatePortal` en el handler).
2. **El manifiesto del modulo nuevo ya "promete" toda la API heredada como
   si fuera real.** `ModuleManifestForCandidatePortal()` declara `Mode: "real"`
   para rutas que hoy sirve exclusivamente el heredado en modo fake. Si algun
   consumidor (frontend, tests, documentacion) confia en ese manifiesto para
   saber "que existe", esta recibiendo una descripcion de capacidades del
   heredado disfrazada de contrato del modulo nuevo. Conviene revisar si debe
   marcarse como `Mode: "legacy"` o similar mientras no haya adaptador propio,
   para no inducir a error sobre el nivel de madurez real (relacionado con
   H-05/T06).
3. **El "autobaremo ciudadano" nunca tuvo un endpoint real de inscripcion.**
   Todo el ciclo `RegistrarSolicitud`/`PublicarListado*` solo se ejercita
   desde el generador de datos de demostracion (`demo.go`), nunca desde una
   ruta ciudadana real. Antes de portar el ciclo de vida de `Solicitud` (lista
   (b) item 2) conviene que el responsable funcional confirme si el objetivo
   es construir por primera vez una inscripcion real (no "migrar" una que ya
   funcionaba) y si el autobaremo declarativo previo a presentar sigue
   siendo necesario o se sustituye directamente por el flujo firmado (lista
   (c) item 9) — es una decision de alcance, no solo tecnica.
4. **Alegaciones y avisos estan casi vacios de logica en el heredado.** Tras
   DEC-021, sus mutaciones mas sensibles (presentar alegacion, enviar/leer
   aviso) ya devuelven `503`. Lo que queda vivo del heredado en estos dos
   dominios es poco (listar, crear notificacion interna sin recibo real): el
   porte es, en la practica, disenar estas dos capacidades desde cero
   siguiendo el patron de `DecisionTecnica`, no migrar codigo heredado
   funcional.
5. **Ninguna capacidad heredada activa carece de equivalente critico sin
   contencion ya aplicada.** Las unicas piezas del heredado con dependencia
   *tecnica* real y sin decision de cierre previa son: el ciclo de
   Solicitud/listado con desempate (4.1/4.2) y el expediente documental
   general (4.3). Todo lo demas (alegaciones, avisos, autenticacion,
   persistencia, demo) ya tiene una decision previa (DEC-020/021/022/025/030)
   que marca el camino a seguir y que este documento solo confirma con cita
   exacta de codigo.

## 7. Siguientes pasos (fuera de este documento)

Este documento cierra los pasos 1-2 de DEC-050. El paso 3 (porte de la lista
(b) al formato nuevo: espanol, hexagonal, fallo cerrado, autorizacion por
caso de uso, limite de tamano de DEC-051, con tests) y el paso 4 (borrado de
`internal/candidate`, su cableado en `bootstrap` y la configuracion residual)
requieren aprobacion expresa del responsable del proyecto y trabajo de codigo
independiente; no se ejecutan aqui.
