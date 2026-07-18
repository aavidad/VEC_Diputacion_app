# Documento de desarrollo VEC para Orquesta/Codex

Autor de la peticion: responsable del proyecto.
Ejecutor: Codex a traves de Orquesta.
Fecha: 2026-06-19.

> **Política de ejecución vigente desde 2026-07-18.** Este documento conserva
> decisiones de alcance e historia, pero ya no autoriza a Orquesta a programar
> ni a modificar el repositorio. Orquesta se usa exclusivamente en solo lectura
> para seguridad, auditoría, revisión cruzada y gates de integración. Codex y
> sus subagentes implementan con write-sets disjuntos; otro agente de Orquesta
> revisa la entrega y una metarrevisión independiente valida los hallazgos antes
> de integrarla. No se usa Orquesta para trabajo mecánico o rutinario cuando el
> coste no aporta garantías proporcionales. Un `stub`, `noop`, fixture o dato de
> presentación puede conservar un contrato o habilitar una prueba explícita,
> pero nunca se presenta como funcionalidad terminada ni sustituye la vertical
> real solicitada.

Este documento es la **orden de trabajo** para que Orquesta construya la primera
version del portal VEC (Ventanilla Electronica del/de la empleado/a publico/a).
Es ejecutable: define alcance, arquitectura, contratos, base de datos, gates de
aceptacion y el plan de tareas. Complementa, no sustituye, a
`contrato_modulos_vec.md` (contrato de modulos) que debe leerse antes.

---

## 0. Principios rectores (no negociables)

1. **Hexagonalidad pura.** El dominio y los casos de uso no importan HTTP, SQL,
   ficheros, red, protocolos externos ni UI. Todo lo externo entra por un
   **puerto** (interfaz Go) y se satisface con un **adaptador**.
2. **PostgreSQL es el primer adaptador de persistencia, pero es intercambiable.**
   El dominio habla con `Repository`/`Store` ports; PostgreSQL es una
   implementacion mas. Debe poder sustituirse (otra BD, memoria para tests) sin
   tocar dominio ni casos de uso.
3. **De menos a mas, con las puertas abiertas.** Se construye lo minimo que
   funciona de verdad. Los protocolos de interoperabilidad con otras
   administraciones se dejan **definidos como puertos** pero **la mayoria NO se
   implementan ahora**: se entregan con un adaptador `noop`/`stub` y un test de
   contrato. Asi el dia de manana se enchufan sin reescribir.
4. **No sobredimensionar.** Critica expresa del responsable al estudio extenso de
   Bolsa (`Bolsa_Diputacion/estudio_arquitectura_bolsa.md`):
   es demasiado grande y se extralimita en servicios. Aqui se hace lo contrario:
   superficie minima ejecutable + puertos preparados. **No** levantar Kubernetes,
   colas, MinIO, Redis, MCP-IA, HA multi-servidor ni microservicios en esta fase.
5. **Software libre.** EUPL v1.2 (recomendado) o GPLv3. Coherente con reutilizar
   AutofirmaV2 (GPL-3.0) y con el art. 157 de la Ley 40/2015.

---

## 1. Que es VEC y que NO es esta primera entrega

VEC es un **portal modular** del empleado publico: un shell con identidad, menu,
permisos, auditoria, notificaciones e i18n comunes, donde se enchufan modulos
(Bolsa, Nominas, Concursos, etc.) con un contrato unico.

Decision de alcance 2026-06-19: **Bolsa no es la aplicacion raiz**. El destino
de implementacion es un proyecto VEC (`VEC_Diputacion_app`). Personal/Nominas
(`vec.module.personal`), Cronos (`vec.module.cronos`), Dietas
(`vec.module.dietas`) y Bolsa (`vec.module.bolsa`) son modulos independientes
aunque relacionados por shell, identidad, permisos, auditoria,
empleado/expediente y justificantes. Cualquier nombre, README, UI o entrypoint
que presente Bolsa como producto principal debe considerarse legado o
compatibilidad.

Personal/Nominas es la pieza maestra de empleados, puestos, situaciones
administrativas, nomina, antiguedad, servicios prestados y certificados. Cronos,
Dietas y Bolsa deben consumir sus referencias y certificados, no duplicar esos
datos ni recalcularlos.

Cronos debe modelar control horario completo, no solo fichajes: perfiles de
jornada por puesto, flexibilidad configurable, puestos sin flexibilidad por
cobertura presencial (por ejemplo atencion a personas mayores), saldos
mensuales, movimientos, permisos/licencias, incidencias/acumulados y reducciones
por prejubilacion configurables (63 anos: 1 hora menos; 64 anos: 2 horas menos,
segun regla interna validada por RRHH).

Dietas debe modelar comisiones de servicio, kilometraje provincial, medias
dietas/dietas completas, justificantes, politicas, aprobaciones y liquidaciones.
El mapa de kilometros debe ser dato de modulo Dietas; VEC solo lo presenta y lo
relaciona con empleado, comision y auditoria.

**Esta primera entrega (vertical minima) SI incluye:**

- `vec-core`: contratos (puertos), tipos de dominio, manifiesto de modulo.
- `vec-shell`: identidad por certificado, sesion, registro de modulos, menu,
  permisos, auditoria, eventos in-memory.
- `vec-module-bolsa`: un modulo minimo de ejemplo que se engancha y demuestra el
  contrato (NO la Bolsa completa; solo lo justo para probar el enganche).
- `vec-module-personal`: modulo maestro de empleado publico, puestos, situaciones
  administrativas, nomina, antiguedad, servicios prestados y certificados.
- `vec-module-cronos`: modulo de control horario, saldos, fichajes, horarios por
  puesto, permisos, vacaciones, incidencias y reducciones 63/64.
- `vec-module-dietas`: modulo de comisiones, rutas, kilometraje provincial,
  medias dietas/dietas completas, justificantes y aprobaciones.
- Adaptador PostgreSQL para lo que persista el shell (sesiones de auditoria,
  registro de modulos instalados, eventos).
- Adaptador de firma/identidad que invoca **AutofirmaV2** local.
- Puertos de interoperabilidad AAPP **definidos** + adaptadores `stub`.

**Esta primera entrega NO incluye (se deja como puerto stub para el futuro):**

- Implementacion real de @firma estatal, SCSP, SIR/ORVE, Notific@/DEHu, InSiDe,
  Apodera, FACe, SARA. Solo el **puerto** y un **stub** con test de contrato.
- Toda la funcionalidad de negocio de Bolsa (baremo, listados, alegaciones).
- HA, colas, object storage, IA, micro-frontends federados en produccion.

---

## 2. Arquitectura de capas (hexagonal)

```text
vec-core/                      (dominio + puertos, CERO dependencias externas)
  domain/        tipos: Principal, Module, MenuEntry, Event, AuditEntry, ...
  ports/         interfaces: todos los puertos (persistencia, firma, interop...)
  application/   casos de uso: RegisterModule, BuildMenu, Authenticate, Audit...

vec-shell/                     (adaptadores inbound/outbound + bootstrap)
  adapters/inbound/http/       handlers REST finos (transporte + DTO + i18n)
  adapters/outbound/postgres/  PRIMER adaptador de persistencia (intercambiable)
  adapters/outbound/memory/    adaptador en memoria para tests
  adapters/outbound/autofirma/ adaptador de firma/identidad -> AutofirmaV2 local
  adapters/outbound/interop/   adaptadores a protocolos AAPP (la mayoria STUB)
  bootstrap/                   composicion: cablea puertos con adaptadores

vec-module-bolsa/              (modulo de ejemplo, mismo contrato que otros)
  domain/ application/ adapters/  igual patron, hexagonal
  manifest.go                  publica VECModuleManifest
```

Regla de oro de imports: las flechas apuntan **hacia dentro**. `application`
depende de `ports` y `domain`. Los `adapters` dependen de `ports`. `domain` no
depende de nada. `bootstrap` es el unico que conoce adaptadores concretos.

---

## 3. Persistencia: PostgreSQL primero, intercambiable

El dominio define puertos de repositorio. Ejemplo:

```go
// vec-core/ports/persistence.go
type ModuleRegistryStore interface {
    Save(ctx context.Context, m domain.ModuleManifest) error
    List(ctx context.Context) ([]domain.ModuleManifest, error)
}

type AuditStore interface {
    Append(ctx context.Context, entry domain.AuditEntry) error
    ListBySubject(ctx context.Context, subjectRef string) ([]domain.AuditEntry, error)
}

type EventStore interface {
    Publish(ctx context.Context, evt domain.Event) error
    Subscribe(ctx context.Context, types []string) (<-chan domain.Event, error)
}
```

Adaptadores a entregar:

- `adapters/outbound/postgres/`: implementacion real con `pgx` (sin ORM pesado;
  SQL explicito o `sqlc`). **Es el adaptador por defecto en bootstrap.**
- `adapters/outbound/memory/`: implementacion en memoria. **Obligatoria**, porque
  prueba que el dominio NO depende de PostgreSQL (si los tests de casos de uso
  pasan con memoria, la intercambiabilidad esta demostrada).

Reglas:

- Migraciones versionadas (`golang-migrate` o equivalente), en `vec-shell/migrations/`.
- Nada de SQL fuera de `adapters/outbound/postgres/`.
- Conexion por config/env (`VEC_DB_DSN`), nunca hardcodeada.
- Bloqueo optimista (`version`) en entidades mutables.
- Borrado logico (no fisico) en datos con valor probatorio.

---

## 4. Identidad y firma: AutofirmaV2 (decision tomada)

Identidad **por certificado** como metodo preferente (decision del responsable).
Reutilizamos software propio: **AutofirmaV2** (`AutofirmaV2`,
Go, hexagonal, GPL-3.0, Oficina de Software Libre Dipgra), cliente local del
empleado en loopback. Endpoints relevantes ya existentes:

- `POST /auth/challenge` + `POST /auth/verify`: autenticacion reto/respuesta por
  certificado.
- `POST /sign`, `POST /sign-batch`: firma (y firma trifasica de lotes).
- `POST /verify`: verificacion de firma.
- `GET /certificates`: certificados del empleado.

Puertos en `vec-core`:

```go
type CertAuthPort interface {
    Challenge(ctx context.Context) (domain.AuthChallenge, error)
    Verify(ctx context.Context, resp domain.AuthChallengeResponse) (domain.Principal, error)
}

type SignaturePort interface {
    Sign(ctx context.Context, p domain.Principal, req domain.SignRequest) (domain.SignReceipt, error)
    VerifySignature(ctx context.Context, ref string) (domain.SignVerification, error)
}
```

Adaptador: `adapters/outbound/autofirma/` que habla con AutofirmaV2 local por
REST. **Salvedad arquitectonica:** AutofirmaV2 vive en el puesto del empleado
(la clave nunca sale del puesto). El flujo de firma es **frontend del shell ->
AutofirmaV2 local del puesto** (patron tipo `afirma://`), no servidor-a-servidor.
El backend NO firma por su cuenta: orquesta, recibe el resultado firmado +
justificante, y entonces registra y audita.

`Principal` lleva `AuthMethod` (certificado | sso | clave) y `AuthAssurance`
(bajo | sustancial | alto). Acciones que generan acto administrativo exigen
`alto` (certificado). Login de solo lectura puede admitir SSO/Cl@ve mas adelante.

---

## 5. Interoperabilidad con otras AAPP: puertos ahora, implementacion despues

Esta es la peticion central del responsable: **VEC debe poder conectarse con todo
el ecosistema de interoperabilidad de las AAPP espanolas, presente y futuro.**
Estrategia hexagonal: **definir todos los puertos ahora**, implementar **solo lo
imprescindible**, dejar el resto como **stub con test de contrato**. Asi nada
obliga a reescribir el dominio cuando se active un servicio.

Catalogo de puertos a definir en `vec-core/ports/interop/` (uno por familia):

| Puerto | Servicio AAPP real | Estandar/protocolo | Para que sirve | Fase |
| --- | --- | --- | --- | --- |
| `IdentityPort` | Cl@ve | SAML 2.0 / OpenID Connect | Identidad estatal del ciudadano/empleado | stub |
| `EUIDWalletPort` | EUDI Wallet (eIDAS 2) | OpenID4VP | Cartera Europea de Identidad Digital (obligatoria sector publico antes 24-dic-2026) | stub |
| `SignatureValidationPort` | @firma / VALIDe | OASIS DSS (XML-SOAP) | Validacion oficial de firma/certificado | stub |
| `TimestampPort` | TS@ (TSA) | RFC 3161 | Sello de tiempo oficial sincronizado con ROA | stub |
| `DataIntermediationPort` | Plataforma de Intermediacion de Datos (PID) | SCSP v3 (XML-SOAP) | Consultar datos a AEAT, TGSS, INE, DGT, INSS, Catastro... (sustituye certificados en papel) | stub |
| `RegistryInterconnectPort` | SIR / ORVE / GEISER | SICRES 3.0 | Intercambio de asientos registrales entre AAPP | stub |
| `CommonRegistryPort` | Registro Electronico Comun (REC) | servicio web AGE | Asiento de registro de entrada/salida | stub |
| `NotificationPort` | Notific@ / DEHu | servicio web AGE | Notificaciones y comunicaciones fehacientes al interesado | stub |
| `DocumentArchivePort` | InSiDe / ARCHIVE | ENI (documento/expediente electronico) | Documento y expediente conforme ENI, archivo definitivo | stub |
| `RepresentationPort` | Apodera (REA) | servicio web AGE | Registro electronico de apoderamientos | stub |
| `InvoicePort` | FACe | Facturae / servicio web | Punto general de entrada de facturas (si aplica) | stub |
| `OrgDirectoryPort` | DIR3 | catalogo de unidades organicas | Codigos de organos y unidades | stub |
| `SecureNetworkPort` | Red SARA | red de comunicaciones AAPP | Conectividad segura inter-AAPP (transporte de los anteriores) | infra |

Reglas para estos puertos:

- Cada puerto se entrega con: (1) la interfaz Go en `vec-core`, (2) un adaptador
  `stub` en `vec-shell/adapters/outbound/interop/<servicio>_stub.go` que devuelve
  un error tipado `ErrInteropNotEnabledV0` o un resultado simulado controlado, y
  (3) un **test de contrato** que verifica la forma del puerto (entradas/salidas)
  independientemente de la implementacion.
- **No** se implementa el cliente SOAP/SCSP/SICRES real en esta fase. Se deja el
  hueco con TODO y la referencia al estandar.
- El dominio y los casos de uso usan SIEMPRE el puerto, nunca el servicio
  concreto. Activar un servicio en el futuro = escribir un adaptador nuevo y
  cablearlo en `bootstrap`, sin tocar nada mas.
- `bootstrap` decide por config que adaptador usar (`stub` por defecto).

De esta tabla, **lo unico que se implementa de verdad ahora** es lo que ya
tenemos en casa (AutofirmaV2, seccion 4). Todo lo demas: puerto + stub + test.

---

## 6. Capacidades transversales del shell (minimas)

| Capacidad | Implementacion ahora |
| --- | --- |
| Identidad | Certificado via AutofirmaV2 (real). SSO/Cl@ve: puerto + stub. |
| Sesion | Real, en PostgreSQL. |
| Registro de modulos | Real (manifiesto -> validacion -> menu). |
| Menu segun permisos | Real. |
| Permisos | Claves namespaced (`bolsa.solicitud.read`), validadas en backend. |
| Auditoria | Real, en PostgreSQL, con encadenamiento de hash (`prev_signature`, `seq`). |
| Eventos | Bus in-memory real (interfaz lista para adaptador persistente). |
| i18n | Claves de catalogo, sin texto hardcodeado en handlers. |
| Firma | Real via AutofirmaV2. |
| Notificaciones/Registro/Documento ENI | Puerto + stub. |

---

## 7. Gates de aceptacion (Orquesta debe certificarlos)

Un entregable solo se acepta si pasa TODOS:

- `gate-hexagonal`: dominio y application no importan HTTP, SQL, red, AutofirmaV2,
  protocolos externos ni UI. (Verificable: grep de imports prohibidos en
  `domain/` y `application/`.)
- `gate-db-intercambiable`: los tests de casos de uso pasan con el adaptador
  `memory` Y con `postgres`. Si solo pasan con uno, falla.
- `gate-postgres`: migraciones aplican limpio sobre PostgreSQL vacio y los tests
  de integracion del adaptador pasan.
- `gate-identidad`: autenticacion por certificado contra AutofirmaV2 funciona;
  `Principal` lleva `AuthMethod` y `AuthAssurance`.
- `gate-firma`: una accion que genera acto administrativo pasa por `SignaturePort`
  y produce justificante; el backend no firma por su cuenta.
- `gate-interop-puertos`: todos los puertos de la tabla seccion 5 existen, tienen
  stub y test de contrato. Ninguno se invoca desde dominio como tipo concreto.
- `gate-modulo`: `vec-module-bolsa` se registra, aparece en menu solo con
  permiso, ejecuta una accion, emite evento auditable; un usuario sin permiso no
  lo ve.
- `gate-ui-root`: `GET /` sirve el shell VEC. Marca, titulo, navegacion primaria,
  estados de conexion y descargas visibles deben hablar de VEC; Bolsa solo puede
  aparecer como modulo o como compatibilidad legada.
- `gate-no-landing-density`: la raiz no es landing ni hero. Debe ser un espacio
  administrativo denso con topbar, KPIs, filtros, cola/listado, panel de detalle
  y acciones repetibles.
- `gate-i18n`: textos visibles por clave de catalogo.
- `gate-auditoria`: acciones administrativas generan entrada auditable encadenada.
- `gate-tests`: `go test ./...` verde en todos los modulos; cobertura de los
  casos de uso del shell.
- `gate-sin-sobredimension`: no se introducen Kubernetes, Redis, colas, MinIO,
  IA/MCP ni microservicios en esta fase. (Verificable: no aparecen en go.mod ni
  en compose.)

---

## 8. Plan de tareas para Orquesta

Orden con dependencias. Cada tarea es hexagonal y termina con tests.

1. **bootstrap-core**: crear modulos Go (`vec-core`, `vec-shell`,
   `vec-module-bolsa`), `go.mod` canonicos, AGENTS.md y docs/contratos.md que
   declaren hexagonalidad estricta como criterio de aceptacion.
   *Dep:* ninguna.
2. **domain-core**: tipos de dominio (`Principal`, `ModuleManifest`, `MenuEntry`,
   `Event`, `AuditEntry`, `AuthChallenge`, `SignRequest`...). Sin imports externos.
   *Dep:* 1.
3. **ports-core**: todos los puertos: persistencia, `CertAuthPort`,
   `SignaturePort`, eventos, auditoria, y los puertos de interop (seccion 5).
   *Dep:* 2.
4. **application-core**: casos de uso (`RegisterModule`, `BuildMenu`,
   `Authenticate`, `Audit`, `PublishEvent`, `RequireAssurance`). Probados contra
   puertos con dobles en memoria.
   *Dep:* 3.
5. **adapter-memory**: adaptador en memoria de los stores. Tests de application
   pasan con el. (Prueba la intercambiabilidad.)
   *Dep:* 4.
6. **adapter-postgres**: adaptador PostgreSQL (`pgx`), migraciones, tests de
   integracion. Mismos tests de contrato que `memory`.
   *Dep:* 4.
7. **adapter-autofirma**: adaptador a AutofirmaV2 local para `CertAuthPort` y
   `SignaturePort`. Test contra un AutofirmaV2 simulado (no requiere el binario en
   CI; si hay binario, smoke opcional).
   *Dep:* 4.
8. **adapter-interop-stubs**: stubs + tests de contrato para cada puerto de la
   tabla seccion 5. Devuelven `ErrInteropNotEnabledV0` o simulacion controlada.
   *Dep:* 4.
9. **shell-http**: handlers REST finos (login por certificado, listar menu,
   registrar modulo, ejecutar accion, healthz). Transporte + DTO + i18n; cero
   logica de negocio.
   *Dep:* 5,6,7.
10. **shell-bootstrap**: composicion. PostgreSQL por defecto, interop=stub.
    Config por env. Arranque del servidor.
    *Dep:* 6,7,8,9.
11. **module-bolsa-minimo**: `vec-module-bolsa` con manifiesto, una entrada de
    menu, una accion que emite evento auditable y `health_route`. NO logica real
    de bolsa.
    *Dep:* 3,4.
12. **integracion-vertical**: test de integracion que demuestra la vertical
    minima de `contrato_modulos_vec.md`: registra Bolsa, menu segun permisos,
    usuario por certificado entra y salta a Bolsa sin re-login, accion emite
    evento auditable, modulo sin permiso no aparece, accion firmada exige
    assurance alto.
    *Dep:* 10,11.

Cuando la tarea 12 cierra en verde por Orquesta, la base VEC esta lista y cada
opcion de menu real puede convertirse en modulo con el mismo contrato.

---

## 9. Restricciones de ejecucion para Codex/Orquesta

- Respetar `gate-sin-sobredimension`. Si una tarea "necesita" Redis/colas/K8s,
  esta mal dimensionada: parar y replantear, no anadir infraestructura.
- No copiar el alcance del estudio extenso de Bolsa. Usarlo solo como referencia
  de dominio (modelo de datos, reglas de baremo) cuando se construya el modulo
  Bolsa REAL, que **no** es esta entrega.
- Toda app/modulo generado: hexagonal puro (lo exige el factory/planner de
  Orquesta ya reforzado).
- PostgreSQL primero, pero `memory` obligatorio para probar intercambiabilidad.
- Cada protocolo AAPP: puerto + stub + test de contrato. Nada de clientes SOAP
  reales en esta fase.
- Licencia EUPL v1.2 o GPLv3 en cada modulo.

---

## 10. Fuentes (ecosistema de interoperabilidad AAPP)

- Plataforma de Intermediacion de Datos (SCSP v3):
  `https://es.wikipedia.org/wiki/Plataforma_de_intermediaci%C3%B3n_de_datos`
- @firma / VALIDE / TS@ (validacion de firma y sello de tiempo, OASIS DSS, RFC 3161):
  `https://rednerea.juntadeandalucia.es/drupal/catalogo_red_sara/afirma`
  `https://rednerea.juntadeandalucia.es/drupal/catalogo_red_sara/tsa`
- SIR / ORVE / GEISER / REC (interconexion de registros, SICRES 3.0):
  `https://administracionelectronica.gob.es/ctt/verPestanaGeneral.htm?idIniciativa=sir`
- InSiDe / ARCHIVE (documento y expediente ENI):
  `https://rednerea.juntadeandalucia.es/drupal/catalogo_red_sara/inside`
- Apodera (registro electronico de apoderamientos):
  `https://apodera.redsara.es/ciudadano/`
- Esquema Nacional de Interoperabilidad (RD 4/2010):
  `https://www.boe.es/buscar/act.php?id=BOE-A-2010-1331`
- Arquitectura de referencia Microfrontend (Junta de Andalucia):
  `https://desarrollo.juntadeandalucia.es/recursos/reglas-pautas/arquitectura-referencia-microfrontend`
- AutofirmaV2 (software propio): `AutofirmaV2`
- Estudio extenso de Bolsa (referencia de dominio, NO de alcance):
  `Bolsa_Diputacion/estudio_arquitectura_bolsa.md`
