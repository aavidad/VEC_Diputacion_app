# Contratos API por modulo (T05 / DEC-053)

Este documento existe para que un cliente de escritorio (o cualquier cliente
nuevo) pueda construirse **contra el contrato**, sin leer el codigo del
servidor, tal como exige DEC-053 ("API primero con clientes web y de
escritorio equivalentes"). No es una propuesta de arquitectura futura: es el
inventario de lo que el binario actual sirve realmente, extraido de
`internal/vec/adapters/httpapi/*.go`, `internal/modules/bolsa/adapters/httppublico/*.go`,
`internal/app/server/server.go` y `internal/app/bootstrap/bootstrap.go`, mas los
tests que fijan su comportamiento. Donde el codigo cierra una superficie a
proposito (503, 403 permanente), este documento lo dice tal cual: no se
inventa un endpoint ni un campo que no exista.

Distinto de `contrato_modulos_vec.md` (propuesta de arquitectura de modulo
"instalable" con manifiesto, Module Federation, etc.): aquel es una discusion
de diseno todavia no implementada; este documento describe unicamente lo que
el servidor responde hoy.

## Indice

1. [Convenciones comunes](#convenciones-comunes)
2. [Modulo: Carcasa VEC (shell)](#modulo-carcasa-vec-shell)
3. [Modulo: Cronos](#modulo-cronos)
4. [Modulo: Dietas](#modulo-dietas)
5. [Modulo: Personal](#modulo-personal)
6. [Modulo: Bolsa publica](#modulo-bolsa-publica)
7. [Fuera de alcance: Bolsa heredada (`internal/candidate`)](#fuera-de-alcance-bolsa-heredada-internalcandidate)
8. [Transporte y arranque comunes](#transporte-y-arranque-comunes)
9. [Huecos para la Ola 2](#huecos-para-la-ola-2)
10. [Inconsistencias detectadas (verificadas en ejecucion)](#inconsistencias-detectadas-verificadas-en-ejecucion)

---

## Convenciones comunes

Aplican a **todas** las rutas bajo `/api/vec` (`internal/vec/adapters/httpapi`),
salvo que se diga lo contrario.

### Envelope de respuesta

Exito (cualquier codigo `2xx`):

```json
{ "data": { "...": "..." } }
```

Error:

```json
{ "error": "mensaje" }
```

`Handler.writeJSON`/`Handler.writeError` en `handler.go` son los unicos puntos
que serializan; ambos son consistentes en todo el shell VEC. `Bolsa publica`
usa un envelope distinto propio (ver su seccion).

### Autenticacion

No hay cookies de sesion en ninguna superficie (DEC-053). La identidad se
resuelve en cada peticion segun `VEC_AUTH_MODE` (`config.AuthMode`):

| Modo | Como se resuelve el `Principal` | Cabeceras que acepta |
| --- | --- | --- |
| `disabled` (por defecto) | Nunca hay identidad (`identityFromRequest` con `acceptHeaders=false`); toda peticion falla `401`. | Ninguna. |
| `fake` | `DemoIdentityResolver.ResolveDemoIdentity` exige `Authorization: Bearer <token>` opaco (43-128 car. base64url), resuelto contra SHA-256 de un fichero local (`VEC_FAKE_CREDENTIALS_FILE`). Solo desde loopback. | `Authorization` unicamente; `X-VEC-*`/`X-Auth-*` se ignoran. |
| `trusted_headers` | Solo si la IP de origen esta en `VEC_TRUSTED_PROXY_CIDRS`; entonces lee `X-VEC-Subject`, `X-VEC-Roles`, `X-VEC-Auth-Mechanism` (nombres configurables) y variantes `X-Auth-*`. Alias contradictorios deniegan. | Cabeceras de identidad + certificado TLS de cliente verificado. |

En **ningun** modo el rol o permiso declarado por el cliente concede nada por
si mismo:

- En `fake`, el fichero declara `roles`; los **permisos** los calcula el
  servidor (`permissionsForRoles` en `handler.go`) a partir de una lista
  positiva cerrada por perfil (ver DEC-030). El cliente nunca envia permisos.
- En `trusted_headers`, `Principal.Permissions` llega siempre `nil`
  (`principalFromRequest`, rama no-demo): los roles/mecanismo de la cabecera
  son informativos, no autorizan nada. **A dia de hoy ningun caso de uso del
  shell VEC concede permisos a partir de esta rama**: el modo esta preparado
  para transportar identidad, pero el PDP real que traduciria esa identidad en
  permisos todavia no existe (DEC-020, "conexion del arranque productivo
  pendiente"). En la practica, `trusted_headers` autentica sujeto pero no
  autoriza ninguna operacion del shell.

Cada handler comprueba `principal.HasPermission("clave.exacta")` antes de
tocar datos; ausencia, permiso repetido no exacto o comodin siempre deniegan
(no hay coincidencia parcial ni herencia).

### Errores HTTP y su significado

| Codigo | Cuando aparece |
| --- | --- |
| `400` | JSON invalido, filtro fuera de rango, cuerpo con campos desconocidos (`DisallowUnknownFields`), o error de dominio no tipado como 404/403. |
| `401` | No hay identidad resoluble (`principal.Validate() != nil` o falta de credencial). Mensaje fijo: `"authentication required"`. |
| `403` | Falta el permiso exacto exigido por la ruta. Mensaje fijo: `domain.ErrPermissionDenied.Error()` = `"vec permission denied"`. |
| `404` | Ruta no reconocida por el `switch` de `ServeHTTP`, o recurso con codigo/slug inexistente en un caso de uso. |
| `405` | Metodo distinto del exigido; la respuesta incluye cabecera `Allow`. |
| `500` | Error interno no tipado del caso de uso (poco frecuente; casi todo pasa por 400/403/503). |
| `502` | Solo en `/dietas/road-route`: el conector OSRM fallo, devolvio un estado no-200, JSON invalido o una respuesta sin `code="Ok"`/`routes`. |
| `503` | Dependencia cerrada a proposito (workspace, Cronos heredado) o backend interno no disponible (falta `InternalOperations`, OSRM no configurado). |

### Version

No existe un segmento de version en la URL (`/api/vec/...`, sin `/v1/`). El
unico versionado explicito hoy es:

- `ModuleManifest.Version` (`"v0.1.0"` en los cinco modulos registrados:
  personal, cronos, dietas, bolsa, administracion), visible en la respuesta de
  `GET /api/vec/modules`.
- El campo `esquema` de las respuestas de Bolsa publica (p. ej.
  `"vec.bolsa.publico.convocatorias.v1"`).

No hay todavia una politica de versionado de contrato a nivel de ruta o de
cabecera `Accept`; un cliente de escritorio debe asumir que romper el envelope
de un endpoint existente romperia a todos los consumidores por igual. Esto es
un hueco a resolver antes de comprometerse con clientes externos estables (ver
seccion de huecos).

### Cache

Todas las respuestas heredan `Cache-Control: no-store` desde
`server.securityHeaders`; el shell nunca marca una respuesta de API como
cacheable.

---

## Modulo: Carcasa VEC (shell)

Fuente: `internal/vec/adapters/httpapi/handler.go`,
`auditoria_consulta.go`. Prefijo servido: `/api/vec` (`server.go` monta el
handler en `cfg.APIBasePath` = `/api` y en `/api/vec`+`/api/vec/`; en la
practica el shell responde con datos propios solo en las rutas registradas en
su `switch` interno bajo `/api/vec/*`).

| Metodo | Ruta | Permiso exigido | Estado |
| --- | --- | --- | --- |
| `GET` | `/api/vec` | `vec.modules.read` | Activo |
| `GET` | `/api/vec/session` | `vec.session.read` | Activo |
| `GET` | `/api/vec/modules` | `vec.modules.read` | Activo |
| `GET` | `/api/vec/menu` | `vec.menu.read` | Activo |
| `GET` | `/api/vec/audit` | `vec.audit.read` (= `adminmodule.PermissionAuditRead`) | Activo, requiere `InternalOperations` |
| `POST` | `/api/vec/modules/{key}/action` | Variable por `key` (tabla abajo) | Activo |
| `GET` | `/api/vec/workspace` | `vec.workspace.read` | **Cerrado (503 permanente)** |

### `GET /api/vec`

Lista las rutas que el propio binario declara (util para descubrimiento
minimo, no es un manifiesto OpenAPI):

```json
{ "data": { "routes": ["/api/vec/session", "/api/vec/modules", "..."] } }
```

La lista completa la genera `vecRoutes()` (23 entradas, incluye las 9
variantes de `/modules/{key}/action` y los dos parametrizados de Personal con
plantilla literal `{code}`/`{slug}`, no expandidos).

### `GET /api/vec/session`

Devuelve el `Principal` resuelto tal cual (mismo struct de dominio,
`internal/vec/domain/types.go`):

```json
{
  "data": {
    "principal": {
      "id": "demo.administrativo",
      "display_name": "Administrativo demo",
      "email": "",
      "roles": ["administrativo"],
      "permissions": ["vec.session.read", "vec.modules.read", "vec.menu.read"],
      "auth_method": "clave",
      "auth_assurance": "sustancial",
      "attributes": {}
    }
  }
}
```

Verificado en ejecucion (modo `fake`, ver seccion de inconsistencias).

### `GET /api/vec/modules`

Devuelve el catalogo completo de `ModuleManifest` registrados en el arranque
(`internal/app/bootstrap.go` registra Personal, Cronos, Dietas, Bolsa y
Administracion vía `internal_operations.RegisterModule`), **sin filtrar por
permiso del principal** (a diferencia de `/menu`): cualquier identidad con
`vec.modules.read` ve los cinco manifiestos completos, incluidos los permisos
y entradas de menu de modulos a los que no tiene acceso funcional. Cada
manifiesto trae `id`, `name_key`, `description_key`, `version`, `group`,
`base_path`, `permissions[]` (`key`, `label_key`) y `menu[]` (`id`,
`module_id`, `label_key`, `path`, `icon`, `group`, `order`,
`required_permissions[]`).

### `GET /api/vec/menu`

Aplica el filtro de permisos: solo se devuelven las `MenuEntry` cuyo
`RequiredPermissions` el principal cumple **integramente**
(`HasAllPermissions`, nunca "al menos uno"). Ordena por `order`, luego
`module_id`, luego `id`.

```json
{ "data": { "menu": [ /* MenuEntry[] filtradas */ ], "principal": { "...": "..." } } }
```

Con los perfiles `fake` tal como estan configurados hoy (DEC-030), el menu
resultante para **cualquier** perfil interno no-administrador es una lista
**vacia**: ningun perfil interno recibe permisos funcionales de Cronos,
Dietas, Personal ni Bolsa por defecto (ver inconsistencias).

### `GET /api/vec/audit`

Requiere `vec.audit.read` y un unico parametro de consulta exacto
`subject_ref` (lista positiva; `referenciaAuditoriaDesdeConsulta` en
`auditoria_consulta.go` rechaza mas de un parametro, ausencia, comodines
(`*`), control chars o mas de 512 bytes). Ejemplo:

```
GET /api/vec/audit?subject_ref=cronos-incidencia-demo
```

```json
{ "data": { "audit": [ /* domain.AuditEntry[] */ ] } }
```

`AuditEntry` trae, entre otros, `id`, `seq`, `actor_id`, `action`,
`module_id`, `subject_ref`, `result`, `occurred_at`, `signature`. No hay
consulta "global" (sin `subject_ref` no hay respuesta: 403, nunca "todo").
Ningun perfil `fake` de fabrica tiene `vec.audit.read` (ni siquiera
`administrador`, deliberadamente, por DEC-030), asi que en la practica esta
ruta siempre devuelve 403 con la configuracion de demostracion actual.

### `POST /api/vec/modules/{key}/action`

Dispatcher generico de "accion de demostracion auditada": crea una entrada de
auditoria y publica un evento, mismo patron para nueve claves fijas
(`actionForPath` en `handler.go`). No admite cuerpo de peticion (no se
decodifica JSON) y siempre usa un `subject_ref` fijo por clave, no aportado
por el cliente.

| `key` | Modulo | Permiso exigido | `subject_ref` fijo | `event.type` |
| --- | --- | --- | --- | --- |
| `cronos` | Cronos | `cronos.aprobacion.manage` | `cronos-incidencia-demo` | `vec.module.cronos.action.executed` |
| `horarios` | Cronos | `cronos.horario.manage` | `cronos-horario-demo` | `vec.module.cronos.schedule.executed` |
| `permisos` | Cronos | `cronos.permiso.manage` | `cronos-permiso-demo` | `vec.module.cronos.leave.executed` |
| `dietas` | Dietas | `dietas.aprobacion.manage` | `dietas-comision-demo` | `vec.module.dietas.action.executed` |
| `rutas` | Dietas | `dietas.ruta.manage` | `dietas-ruta-demo` | `vec.module.dietas.route.executed` |
| `bolsa` | Bolsa | `bolsa.demo.action` | `bolsa-demo-action` | `vec.module.bolsa.action.executed` |
| `administracion` | Administracion | `vec.catalogs.manage` | `admin-catalogos-demo` | `vec.module.administracion.catalog.executed` |
| `personal` | Personal | `personal.certificado.manage` | `personal-certificado-servicios-demo` | `vec.module.personal.action.executed` |
| `nominas` | Personal | `personal.nomina.manage` | `personal-nomina-demo` | `vec.module.personal.payroll.executed` |

Una `key` fuera de esta tabla (p. ej. `/api/vec/modules/inexistente/action`)
responde `404`. Exito devuelve `202 Accepted`:

```json
{ "data": { "receipt": { /* domain.AuditEntry completo, con signature */ } } }
```

Esto **no** ejecuta ninguna accion administrativa real de Cronos/Dietas/etc.:
es exclusivamente un registro de auditoria + evento de demostracion. Un
cliente de escritorio no debe asumir que llamar a esta ruta aprueba una
incidencia, una dieta o cualquier otra cosa real.

### `GET /api/vec/workspace` — cerrado a proposito

```go
func (h *Handler) handleWorkspace(...) {
    if !h.requireMethod(w, r, http.MethodGet) { return }
    if !principal.HasPermission("vec.workspace.read") { 403; return }
    h.escribirSuperficieHTTPCronosNoDisponible(w) // siempre 503
}
```

Requiere `vec.workspace.read` (permiso que **ningun** perfil `fake` de
fabrica posee) y, aunque se concediera, responde siempre `503` con
`{"error":"superficie HTTP de Cronos no disponible"}`. Es DEC-026 aplicado:
"el workspace mezcla datos de varias personas y modulos, pero carece de
resolver de recursos en el servidor [...] La ruta queda cerrada hasta que el
PDP aporte un ambito positivo exacto para cada seccion y campo".

Dato relevante para quien lea el codigo: existen funciones
`workspaceSnapshot`/`workspaceSnapshotWithCronos` en `workspace.go` que
construyen un payload muy completo (KPIs, `operational_records`, catalogos
RPT, datos de Cronos, rutas provinciales...). **Ese codigo no es alcanzable
por HTTP**: solo lo invocan tests (`handler_test.go`,
`cronos_security_test.go`). Un cliente de escritorio no puede ni debe
construirse contra ese payload; es codigo muerto desde la perspectiva del
contrato HTTP.

---

## Modulo: Cronos

Fuente: `internal/vec/adapters/httpapi/cronos.go`. Todo el modulo esta cerrado
a proposito (DEC-026): ambas rutas siempre responden `503` tras superar la
comprobacion de permiso.

| Metodo | Ruta | Permiso exigido | Estado |
| --- | --- | --- | --- |
| `GET` | `/api/vec/cronos/timecards` | `cronos.fichaje.read` | 403 sin permiso; si no, **503** |
| `POST` | `/api/vec/cronos/timecards` | `cronos.fichaje.manage` | 403 sin permiso; si no, **503** |
| `GET` | `/api/vec/cronos/leave-requests` | `cronos.permiso.read` | 403 sin permiso; si no, **503** |
| `POST` | `/api/vec/cronos/leave-requests` | `cronos.permiso.manage` | 403 sin permiso; si no, **503** |

Mensaje de error fijo en los cuatro casos:
`{"error":"superficie HTTP de Cronos no disponible"}`, cabecera
`Cache-Control: no-store` explicita ademas de la global.

Motivo documentado: los `GET` heredados entregaban una instantanea completa
con permiso grueso y los `POST` aceptaban el identificador de empleado
enviado por el navegador; no hay resolver de persona/organigrama/delegacion
en el servidor todavia. **Ningun dato de Cronos llega hoy al cliente por
HTTP**, ni en lectura ni en escritura. Todo lo que el modulo Cronos ofrece
(perfiles horarios, fichajes, saldos, permisos, vacaciones, reducciones
63/64) vive solo en el motor `internal/modules/cronos/application` usado
internamente por `workspaceSnapshot` (inalcanzable, ver arriba) y por los
tests.

---

## Modulo: Dietas

Fuente: `internal/vec/adapters/httpapi/dietas_road_route.go`. Unico endpoint
activo del modulo:

| Metodo | Ruta | Permiso exigido | Estado |
| --- | --- | --- | --- |
| `POST` | `/api/vec/dietas/road-route` | `dietas.ruta.read` | Activo si el conector OSRM esta configurado; si no, `503` |

### Peticion

```json
{
  "coordinates": [
    { "lat": 37.18, "lon": -3.6, "name": "opcional" },
    { "lat": 37.20, "lon": -3.7 }
  ],
  "alternatives": 1
}
```

- `coordinates`: 2 a 25 elementos. `lat` en `[-90, 90]`, `lon` en `[-180,
  180]`; ademas cada punto debe caer dentro del rectangulo configurado en
  `VEC_OSRM_SCOPE_BOUNDS` (p. ej. ambito "Granada"). `name` no se usa para
  enrutar, es informativo.
- `alternatives`: entero `0..3`. `0` equivale a `1` (una unica ruta); fuera de
  rango es `400`.
- El decodificador rechaza campos desconocidos (`DisallowUnknownFields`) y un
  segundo documento JSON en el mismo cuerpo. Limite de cuerpo: 1 MiB.

### Respuesta

Reenvia el JSON de OSRM (`GET {base}/route/v1/driving/{lon,lat;...}?overview=full&geometries=geojson&steps=false&alternatives=N`)
tal cual, con dos campos anadidos:

```json
{
  "data": {
    "code": "Ok",
    "routes": [ /* geometria GeoJSON, distancia, duracion... tal como OSRM las emite */ ],
    "engine": "osrm_on_premise",
    "route_scope": "Granada"
  }
}
```

La URL interna del motor OSRM **no** se expone nunca al cliente (solo
`engine` y `route_scope`).

### Errores especificos

| Codigo | Causa |
| --- | --- |
| `400` | Coordenadas fuera de rango/ambito, `alternatives` fuera de `0..3`, JSON con campos desconocidos o doble documento. |
| `403` | Falta `dietas.ruta.read`. |
| `502` | OSRM no responde `200`, responde JSON invalido, no confirma `code="Ok"` o no trae `routes` como lista; tambien si el conector no puede conectar o la respuesta excede 20 MiB. |
| `503` | `VEC_OSRM_BASE_URL`/`VEC_OSRM_SCOPE_NAME`/`VEC_OSRM_SCOPE_BOUNDS`/`VEC_OSRM_ALLOWED_CIDRS` no estan configurados de forma completa (los cuatro son atomicos: o los cuatro o ninguno). |

El resto de pantallas de Dietas (dashboard, comisiones, kilometraje anual,
mapa provincial, justificantes, aprobaciones, liquidaciones) **no tiene
ningun otro endpoint**: solo el calculo de ruta punto-a-punto esta expuesto.

---

## Modulo: Personal

Fuente: `internal/vec/adapters/httpapi/personal_rpt.go`. Es el modulo con mas
superficie CRUD real del shell. Almacen: memoria por defecto, o fichero
durable si `VEC_PERSONAL_CATALOG_PATH` apunta a una ruta (no es "memory").
**Importante**: el arranque productivo (`bootstrap.go`) no precarga ningun
dato de ejemplo en este catalogo; los fixtures que aparecen en
`workspace.go` (`workspaceRPTPositionSamples`, etc.) solo se cargan
explicitamente en tests (`personal_rpt_startup_test.go`). Un despliegue
limpio arranca con el catalogo vacio hasta que alguien haga `POST
/personal/rpt/imports` o `PUT`/`POST` individuales.

| Metodo | Ruta | Permiso exigido | Descripcion |
| --- | --- | --- | --- |
| `GET` | `/api/vec/personal/rpt/positions` | `personal.puesto.read` | Lista paginada de puestos RPT |
| `GET` | `/api/vec/personal/rpt/positions/{code}` | `personal.puesto.read` | Detalle por codigo (con fallback por "Codigo RPT oficial" en observaciones) |
| `PUT` | `/api/vec/personal/rpt/positions/{code}` | `personal.puesto.manage` | Alta/edicion (upsert) de un puesto + recibo de auditoria |
| `DELETE` | `/api/vec/personal/rpt/positions/{code}` | `personal.puesto.manage` | Baja + recibo de auditoria |
| `POST` | `/api/vec/personal/rpt/imports` | `personal.puesto.manage` | Importacion masiva (reemplaza o añade) + recibo |
| `GET` | `/api/vec/personal/rpt/stats` | `personal.puesto.read` | Contadores agregados del catalogo |
| `GET` | `/api/vec/personal/categories` | `personal.puesto.read` | Lista paginada de categorias profesionales |
| `POST` | `/api/vec/personal/categories` | `personal.puesto.manage` | Alta de categoria + recibo |
| `GET` | `/api/vec/personal/categories/{slug}` | `personal.puesto.read` | Detalle de categoria |
| `PUT` | `/api/vec/personal/categories/{slug}` | `personal.puesto.manage` | Edicion (upsert) + recibo |
| `DELETE` | `/api/vec/personal/categories/{slug}` | `personal.puesto.manage` | Baja + recibo |
| `GET` | `/api/vec/personal/catalogs` | `personal.puesto.read` | Lista de entradas de catalogo auxiliar (tipos de contrato, etc.) |

Nota: todas las escrituras usan el mismo permiso "manage", **no** hay
distincion entre crear/editar/borrar puestos y categorias (a diferencia de
Cronos/Dietas, donde cada accion tiene su propio permiso segun DEC-020). Un
cliente de escritorio que quiera ese nivel de granularidad no puede
expresarlo hoy: quien puede editar puede tambien borrar.

### `RPTPosition` (JSON)

```json
{
  "code": "118-DEMO", "name": "Puesto tecnico sintetico", "dot": 1,
  "type": "", "administration": "", "provision": "L", "group": "A1",
  "area": "", "scale": "", "category_code": "", "category_slug": "",
  "delegation": "", "center_code": "", "center_name": "",
  "destination_level": 0, "specific_kind": "", "annual_amount_cents": 0,
  "gcp_ct_level": "", "specific_complement": "", "geo_dispersion": "",
  "telework": "", "coverage": "", "state": "Vigente", "source": "fixture",
  "requirements": "", "observations": "", "page": 0, "raw": ""
}
```

`Validate()` exige `code` y `name` no vacios, `state` sin espacios exteriores
y no vacio, y `dot`/`destination_level`/`annual_amount_cents` >= 0.

### `RPTPositionPage` / `ProfessionalCategoryPage`

```json
{ "items": [ /* ... */ ], "total": 0, "limit": 100, "offset": 0 }
```

Filtros de consulta admitidos (`?q=&group=&center_code=&provision=&state=&limit=&offset=`
para posiciones; `?q=&area=&limit=&offset=` para categorias). `limit` por
defecto 100, tope 2000 (posiciones) / 500 (categorias); valores fuera de
rango se normalizan silenciosamente al valor por defecto (no es un error
400).

### `RPTImportCommand` / `RPTImportReceipt`

Peticion:

```json
{ "source": "rpt_2026", "version": "v3", "replace": true, "positions": [ /* RPTPosition[] */ ] }
```

Respuesta (`201 Created`):

```json
{ "data": { "import": { "source": "rpt_2026", "version": "v3", "imported": 42, "replaced": true }, "receipt": { /* AuditEntry */ } } }
```

### `ProfessionalCategory`

```json
{ "slug": "cocinero", "name": "Cocinero/a", "area": "Servicios", "source": "USO/OPES seed", "source_path": "", "module_key": "", "state": "Activo", "usage": "" }
```

### `CatalogStats` (`GET /personal/rpt/stats`)

```json
{ "positions": 0, "categories": 0, "catalog_entries": 0, "positions_by_group": {}, "categories_by_area": {}, "pending_legend": 0 }
```

### `CatalogEntry` (`GET /personal/catalogs`)

```json
{ "catalog": "tipo_contrato", "code": "L", "label": "Laboral", "source": "", "module_key": "", "state": "Activo", "usage": "" }
```

El resto de pantallas de Personal (expedientes, situaciones administrativas,
antiguedad/trienios, servicios prestados, certificados, nominas,
retribuciones, incidencias, integraciones) **no tienen ningun endpoint**:
solo el catalogo RPT/categorias/catalogos auxiliares esta expuesto por HTTP.

---

## Modulo: Bolsa publica

Fuente: `internal/modules/bolsa/adapters/httppublico/handler.go` +
`internal/modules/bolsa/application/consulta_publica.go`. Es el unico modulo
sin autenticacion: proyeccion publica minimizada, sin datos personales, para
visitantes anonimos. Montado en `/api/publico/bolsa/convocatorias` en
**todos** los modos de `VEC_AUTH_MODE` (no depende de `fake`).

| Metodo | Ruta | Auth | Descripcion |
| --- | --- | --- | --- |
| `GET`/`HEAD` | `/api/publico/bolsa/convocatorias` | Ninguna | Listado paginado y filtrable |
| `GET`/`HEAD` | `/api/publico/bolsa/convocatorias/{id}` | Ninguna | Detalle de una convocatoria publicada |

Cualquier otro metodo devuelve `405` con `Allow: GET, HEAD`. Una ruta con `%`
codificado o `RawPath` no vacio devuelve `400` (`ruta_invalida`) — no se
normaliza, se rechaza.

### Envelope de error (propio de este modulo, distinto del shell)

```json
{ "esquema": "vec.error.publico.v1", "error": { "codigo": "consulta_invalida", "mensaje": "La consulta no es válida." } }
```

Codigos usados: `consulta_invalida` (400), `recurso_no_encontrado` (404),
`convocatoria_no_encontrada` (404), `metodo_no_permitido` (405),
`error_interno` (500), `servicio_no_disponible` (503, solo si el servicio no
se pudo construir en el arranque).

### `GET /api/publico/bolsa/convocatorias`

Parametros de consulta (lista blanca exacta, cualquier otro parametro o
repetido es `400`): `texto` (<=100 runas), `tipo`, `categoria`, `estado`
(claves de catalogo, patron `^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$`), `plazo`
(unico valor admitido: `abierto`), `pagina` (1..500), `tamano` (1..24,
defecto 12). Query string completa limitada a 2048 bytes.

Respuesta (`200`), esquema `vec.bolsa.publico.convocatorias.v1`:

```json
{
  "esquema": "vec.bolsa.publico.convocatorias.v1",
  "fuente": { "revision": "2026-07", "actualizada_en": "2026-07-16T00:00:00Z", "demostracion": true, "aviso": "..." },
  "facetas": { "tipos": [ /* ValorCatalogoPublico[] */ ], "categorias": [ ], "estados": [ ] },
  "paginacion": { "pagina": 1, "tamano": 12, "total": 0, "paginas": 0 },
  "convocatorias": [ /* ResumenConvocatoriaPublica[] */ ]
}
```

`ValorCatalogoPublico`: `{ "clave", "version", "etiqueta", "descripcion?", "semantica" }`.

`ResumenConvocatoriaPublica`: `identificador_publico`, `version`,
`huella_sha256`, `titulo`, `resumen`, `tipo` (`ValorCatalogoPublico`),
`estado` (`ValorCatalogoPublico`), `categorias[]`, `plazo_destacado?`
(`PlazoPublico`), `numero_requisitos`, `numero_documentos`, `numero_ayudas`,
`publicada_en`, `actualizada_en`.

`PlazoPublico`: `referencia`, `tipo`, `titulo`, `descripcion?`, `abre_en`,
`cierra_en`, `situacion` (`proximo`|`abierto`|`cerrado`), `etiqueta_situacion`,
`semantica_situacion`.

### `GET /api/publico/bolsa/convocatorias/{id}`

`{id}` debe casar `^[a-z0-9][a-z0-9-]{2,79}$` y no admite query string (400 si
la trae). Respuesta (`200`), esquema `vec.bolsa.publico.convocatoria.v1`:

```json
{
  "esquema": "vec.bolsa.publico.convocatoria.v1",
  "fuente": { "...": "..." },
  "convocatoria": { /* ResumenConvocatoriaPublica */ },
  "descripcion": "texto largo",
  "plazos": [ /* PlazoPublico[] */ ],
  "requisitos": [ { "referencia", "orden", "titulo", "descripcion", "obligatorio" } ],
  "documentos": [ { "referencia", "tipo", "orden", "titulo", "descripcion", "formato", "url", "publicado_en" } ],
  "ayuda": [ { "referencia", "categoria", "orden", "pregunta", "respuesta" } ]
}
```

Fuente de datos: fichero JSON configurado en `VEC_BOLSA_PUBLIC_SOURCE_PATH`
(por defecto `data/demo/convocatorias_publicas.demo.json`), no base de datos.
Este es el modulo con el cliente frontend mas fiel al contrato real:
`web/static/bolsa/bolsa.js` llama exactamente a estas dos rutas (listado y
detalle) sin datos sinteticos locales de por medio.

---

## Fuera de alcance: Bolsa heredada (`internal/candidate`)

`internal/candidate/adapters/handler` (paquete en ingles, distinto de
`internal/modules/bolsa`) expone otra API bajo `/candidates`, `/api/modules/bolsa`,
`/api/admin/status`, `/api/admin/capabilities`, `/api/demo`,
`/api/notifications/...`, etc. Existe y responde, pero **solo cuando
`VEC_AUTH_MODE=fake`** (`bootstrap.NewDemoAPIWithConfig`: en cualquier otro
modo esta rama ni se monta). DEC-050 la declara "en solo-mantenimiento" con
retirada planificada tras portar lo necesario a `internal/modules/bolsa`. Por
eso este documento **no la detalla endpoint por endpoint**: un cliente de
escritorio nuevo no debe construirse contra una API que va a desaparecer. Si
se necesita, `internal/candidate/adapters/handler/routes.go` es su unico
punto de entrada de enrutado.

---

## Transporte y arranque comunes

- `internal/app/server/server.go` monta: `/` (estatico `web/static`),
  `/locales/` (JSON de idioma), `/healthz` (`{"status":"ok"}`, sin auth),
  `cfg.APIBasePath` (`/api`, por defecto) y `/candidates` apuntando al mismo
  handler compuesto por `bootstrap`.
- Limite de cuerpo de peticion: `cfg.MaxRequestBodyBytes` (2 MiB por
  defecto), aplicado a todo metodo salvo `GET/HEAD/OPTIONS`.
- `VEC_HTTP_ALLOWED_CIDRS` es una lista positiva sin valor universal: sin
  configurar, el listener normaliza a loopback (`127.0.0.1/32`, `::1/128`); a
  cualquier IP fuera de esas redes se responde `403`/`503` antes de llegar al
  handler de aplicacion.
- Modo `fake` exige ademas que `VEC_HTTP_ADDR` sea una IP loopback literal y
  que las redes permitidas sean exclusivamente loopback (`server.go`,
  `redesExclusivamenteLocales`); si no, el servidor rehusa arrancar.
- Cabeceras de seguridad fijas en toda respuesta: CSP restrictiva,
  `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, HSTS,
  `Cache-Control: no-store` (recursos estaticos versionados con `?v=` son la
  unica excepcion, cacheables un año).

---

## Huecos para la Ola 2

Comparacion entre lo que `web/static/app.js` (13.211 lineas, unico bundle del
shell interno, cargado desde `web/static/index.html`) renderiza y las
llamadas `fetch`/`getData`/`sendJSON` reales que emite. `app.js` solo declara
nueve constantes de endpoint real:

```
VEC_SHELL_API=/api/vec, VEC_WORKSPACE_API=/api/vec/workspace,
VEC_SESSION_API=/api/vec/session, CRONOS_LEAVE_API=/api/vec/cronos/leave-requests,
PERSONAL_RPT_POSITIONS_API, PERSONAL_RPT_STATS_API, PERSONAL_CATEGORIES_API,
PERSONAL_CATALOGS_API, DIETAS_ROAD_ROUTE_API
```

(mas las rutas heredadas de Bolsa: `BOLSA_PORTAL_API`, `ADMIN_STATUS_API`,
`ADMIN_CAPABILITIES_API`, `/api/candidates...`, fuera de alcance de este
documento). Todo lo demas que las pantallas del shell muestran es JSON/JS
local embebido en el propio `app.js` (arrays y objetos constantes, no
resultado de un `fetch`).

| Pantalla / seccion en `app.js` | Endpoint real que la respalda hoy | Origen real del dato mostrado |
| --- | --- | --- |
| Sesion, nombre de usuario, rol activo | `GET /api/vec/session` | Real |
| Catalogo de modulos / permisos declarados | `GET /api/vec/modules` | Real |
| Menu lateral | Ninguno propio (deriva de `/api/vec/modules`, no de `/api/vec/menu`) | Real pero sin filtrar por permiso (ver inconsistencia) |
| Cronos: fichajes, saldo horario, movimientos | Ninguno (`/cronos/timecards` cerrado 503) | 100% local (`state.workspace` nunca llega a rellenarse; ver inconsistencia) |
| Cronos: permisos y vacaciones (lectura) | Ninguno (`/cronos/leave-requests` GET cerrado 503) | Local |
| Cronos: alta de solicitud de permiso | `POST /api/vec/cronos/leave-requests` (tambien cerrado 503) | Local (el POST nunca prospera) |
| Cronos: reducciones 63/64, incidencias, aprobaciones, saldos | Ninguno | 100% local |
| Dietas: dashboard, comisiones, kilometraje anual, dietas, justificantes, aprobaciones, liquidaciones | Ninguno | 100% local |
| Dietas: mapa provincial, calculo de ruta punto a punto | `POST /api/vec/dietas/road-route` | Real solo para el calculo de distancia/geometria; el resto de la pantalla (listas de comisiones, dietas asociadas) es local |
| Personal: puestos RPT (listado/detalle) | `GET /api/vec/personal/rpt/positions[/{code}]` | Real, pero catalogo vacio salvo import previo |
| Personal: categorias profesionales, catalogos auxiliares | `GET /api/vec/personal/categories`, `GET /api/vec/personal/catalogs` | Real, mismo matiz |
| Personal: estadisticas de catalogo | `GET /api/vec/personal/rpt/stats` | Real |
| Personal: expedientes, situaciones administrativas, antiguedad/trienios, servicios prestados, certificados | Ninguno | 100% local |
| Personal: nominas, retribuciones, incidencias, integraciones | Ninguno | 100% local (incluye tablas `PAYROLL_HISTORY_MONTHS` y ajustes de productividad hardcodeados en el JS) |
| Administracion: usuarios/roles, catalogos, integraciones, monitorizacion | Ninguno propio del shell nuevo (usa `/api/admin/status` y `/api/admin/capabilities` heredados) | Heredado, fuera de alcance |
| Acciones "aprobar/validar" de cada modulo (botones) | `POST /api/vec/modules/{key}/action` | Real solo como recibo de auditoria generico; no ejecuta la accion de negocio real (ver seccion del dispatcher) |

Resumen cuantitativo: de las ~30 pantallas/secciones funcionales que expone
`app.js` para el shell interno, **9** tienen un endpoint real detras (sesion,
catalogo de modulos, RPT positions+detalle, categorias, catalogos, stats,
road-route, y el dispatcher de acciones como recibo de auditoria — sin contar
aqui el modulo Bolsa publica, que es un cliente distinto y ya correcto). El
resto (Cronos completo, Dietas salvo el calculo de ruta, y la mayor parte de
Personal: expedientes, situaciones, antiguedad, servicios, certificados,
nominas, retribuciones, incidencias, integraciones) se renderiza con datos
sinteticos locales embebidos en `app.js`, exactamente la deuda que DEC-053
identifica como "a extinguir con la Ola 2, no un patron a repetir". Estas son
las pantallas candidatas a recibir endpoints finos en la Ola 2.

---

## Inconsistencias detectadas (verificadas en ejecucion)

Estas tres se comprobaron arrancando `cmd/vec-server` en modo `fake` con un
fichero de credenciales valido y usando `curl` (no son solo lectura de
codigo):

1. **`app.js` nunca envia el `Authorization: Bearer` que el propio modo
   `fake` exige.** `staffHeaders()`/`candidateHeaders()` en `app.js` solo
   fijan `Content-Type: application/json`. Verificado: `GET /api/vec/session`
   y `GET /api/vec/modules` sin cabecera `Authorization` responden `401
   {"error":"authentication required"}`. Esto es coherente con DEC-025 ("el
   JavaScript no contiene tokens... ni selector de roles") pero significa que,
   **tal como esta hoy el bundle**, el shell no puede autenticarse solo
   abriendo `index.html` en un navegador contra el servidor en modo `fake`;
   hace falta una pieza externa (proxy, extension, credencial inyectada) que
   este repositorio no incluye.

2. **Ningun perfil `fake` de fabrica tiene permisos funcionales de
   Cronos/Dietas/Personal**, ni siquiera `administrador`. Verificado con
   Bearer valido: `GET /api/vec/workspace`, `GET
   /api/vec/cronos/leave-requests`, `GET /api/vec/personal/rpt/positions`,
   `POST /api/vec/dietas/road-route` y `POST
   /api/vec/modules/personal/action` devuelven `403` para los perfiles
   `administrativo` y `administrador`; `GET /api/vec/menu` devuelve
   `"menu": []` para `administrativo` y solo las 4 entradas de
   Administracion para `administrador` (nunca Personal/Cronos/Dietas/Bolsa).
   Esto es DEC-030 aplicado a proposito, pero combinado con el punto 1
   significa que **`loadPortal()` en `app.js` (la funcion que arranca todo el
   shell, invocada una sola vez al final del fichero) nunca puede completar
   su `Promise.all`** en un despliegue de demostracion recien instalado: ese
   `Promise.all` incluye `getData(VEC_WORKSPACE_API, ...)`,
   `getData(PERSONAL_RPT_POSITIONS_API, ...)`,
   `getData(PERSONAL_RPT_STATS_API, ...)`, `getData(PERSONAL_CATEGORIES_API,
   ...)` y `getData(PERSONAL_CATALOGS_API, ...)` sin `.catch` local; el primer
   `401`/`403` de cualquiera de ellas hace que `getData` lance, el `Promise.all`
   se rechace entero y el `catch` de `loadPortal` solo ejecute
   `setStatus(error.message, "error")` — la pagina se queda en estado de
   error, no en una pantalla parcialmente rellenada con datos locales. Antes
   de dar por cerrada la Ola 1 conviene decidir si el propio `fake` va a
   conceder los permisos que las pantallas de demostracion necesitan, o si
   `loadPortal()` debe tolerar el fallo de cada llamada por separado.

3. **Ambiguedad en `/api` raiz.** `app.js` llama a `GET /api` esperando
   `{"routes": [...]}` (`loadAPIRootRoutes`), pero esa ruta exacta la sirve
   `internal/candidate/adapters/handler` (la API heredada, solo montada en
   modo `fake`), no `internal/vec/adapters/httpapi` (que expone su propia
   raiz equivalente en `/api/vec`, con una lista de rutas distinta y mas
   completa). En modos `disabled`/`trusted_headers` (sin la rama heredada
   montada) `GET /api` no tiene handler registrado y depende del
   comportamiento por defecto de Go (`404 page not found`, sin el envelope
   JSON del shell). Un cliente de escritorio que quiera "listar rutas
   disponibles" debe usar `GET /api/vec`, nunca `GET /api`.

Ademas, dos observaciones de solo lectura de codigo (no requieren arrancar el
servidor):

- `workspaceSnapshot`/`workspaceSnapshotWithCronos` (`workspace.go`) generan
  un payload rico (KPIs, `operational_records`, `cronos_*`,
  `province_route_*`...) que **ningun** handler HTTP invoca; solo lo llaman
  tests. Si la intencion de la Ola 2 es reabrir el workspace con permisos
  finos, ese codigo es un punto de partida razonable para el nuevo caso de
  uso, pero hoy es puro codigo muerto desde el contrato HTTP.
- El catalogo de Personal (`personal/rpt/*`, `personal/categories`,
  `personal/catalogs`) arranca vacio en un despliegue limpio: los fixtures
  de `workspace.go` que parecen "poblar" RPT/categorias solo se cargan en
  tests (`cargarFixturesCatalogoPersonalPrueba`), nunca desde
  `bootstrap.go`. Quien pruebe estos endpoints en un servidor recien
  arrancado debe importar datos explicitamente (`POST
  /personal/rpt/imports`) antes de ver resultados no vacios.
