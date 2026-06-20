# Arquitectura tecnica modular del portal

## Objetivo

Definir la estructura tecnica para evolucionar el prototipo de bolsa de empleo
hacia un portal modular, manteniendo la arquitectura hexagonal existente, i18n
centralizada y pruebas verdes durante el refactor.

La base actual ya expone un servidor Go con HTML estatico, API bajo `/api`,
dominio de candidatos/meritos/baremo/procedimiento, repositorios en memoria,
autenticacion fake y configuracion canonica en `config`.

## Principios

- Nucleo neutral: dominio y casos de uso sin HTTP, HTML, base de datos,
  Orquesta, rutas locales, credenciales ni proveedor externo.
- Puertos pequenos: contratos por capacidad, no por tecnologia.
- Adaptadores opt-in: HTTP, frontend estatico, repositorio memoria, auth fake y
  futuras integraciones reales se conectan solo en composicion.
- Configuracion unica: variables y defaults documentados en una superficie
  canonica.
- i18n centralizada: textos de API y UI salen de catalogos, con fallback
  controlado.
- Persistencia sustituible: memoria por defecto en prototipo; almacenamiento
  duradero solo como adaptador, sin contaminar el nucleo.

## Estructura propuesta

```text
cmd/
  bolsa-server/
    main.go
config/
  config.go
internal/
  app/
    bootstrap/
      bootstrap.go
    server/
      server.go
      routes.go
  candidate/
    domain/
    usecases/
    ports/
    adapters/
      handler/
      repository/
      auth/
  portal/
    domain/
    usecases/
    ports/
    adapters/
      handler/
web/
  static/
    index.html
    app.js
    styles.css
    locales/
docs/
  portal_vec/
```

`internal/candidate` mantiene la vertical existente. `internal/portal` agrupa
capacidades transversales de pantalla, navegacion y resumen si crecen mas alla
de la demo actual. `internal/app/bootstrap` construye dependencias demo y
elige adaptadores; ningun paquete de dominio debe importar composicion.
`cmd/bolsa-server/main.go` solo arranca el servidor devuelto por bootstrap.
La composicion actual conecta repositorios en memoria, autenticacion fake de
demo, catalogo i18n, casos de uso y handler HTTP sin que el binario conozca
detalles de dominio o adaptadores concretos.

## Backend

La separacion deseada por capa queda asi:

- `domain`: entidades, estados, validaciones, ranking y reglas puras.
- `usecases`: casos de uso con puertos inyectados; sin `net/http`.
- `ports`: repositorios, autenticador, reloj, auditoria y notificaciones cuando
  existan.
- `adapters/handler`: DTOs, routing HTTP, validacion de transporte, envelopes e
  i18n de mensajes.
- `adapters/repository`: implementaciones en memoria y futuras
  implementaciones duraderas.
- `adapters/auth`: autenticacion fake actual y futuros adaptadores reales.
- `app/server`: mux, health, estaticos, prefijo API y timeouts.
- `app/bootstrap`: wiring de prototipo/produccion, carga de config y
  seleccion de adaptadores.

El handler actual puede dividirse sin cambiar comportamiento:

```text
internal/candidate/adapters/handler/
  http.go              # Handler, ServeHTTP y dependencias
  routes.go            # parseo de rutas y dispatch
  dto.go               # comandos/vistas/envelope
  candidate_service.go # aplicacion HTTP sobre casos de uso
  demo.go              # demo administrativa
  errors.go            # statusFromError, errorKey
```

Cada fichero Go debe permanecer por debajo de 300 lineas.

## Frontend

El frontend actual es estatico y se sirve desde `web/static`:

- `GET /` devuelve la pantalla principal.
- `app.js` invoca `POST /api/demo` con cabeceras fake de personal interno.
- La pantalla muestra convocatoria, estado, solicitudes y listados provisional
  y definitivo.

Estructura incremental recomendada sin introducir build step:

```text
web/static/
  index.html
  app.js
  styles.css
  locales/
    es.json
  views/
    convocatoria.js
    candidato.js
  services/
    api.js
  components/
    listing.js
    status.js
```

Si despues se adopta SPA con build, debe quedar detras de `web/portal` o
`frontend/portal`, manteniendo contrato HTTP estable y artefactos compilados
servidos por `internal/app/server`.

Pantallas/rutas frontend esperadas:

- `/`: tablero administrativo demo/listados.
- `/candidatos`: alta y consulta de candidato.
- `/candidatos/{id}`: expediente, meritos y autobaremo.
- `/convocatorias/{id}`: estado de convocatoria y listados.

En fase estatica, estas rutas pueden resolverse con hash o History API solo si
el servidor sirve fallback a `index.html`.

## API y rutas

Rutas publicas actuales:

| Metodo | Ruta | Uso |
| --- | --- | --- |
| `GET` | `/healthz` | Salud del proceso |
| `GET` | `/` | Frontend estatico |
| `GET` | `/api` | Descubrimiento minimo de rutas |
| `POST` | `/api/demo` | Demo administrativa con convocatoria y listados |
| `POST` | `/api/candidates` | Alta de candidato |
| `POST` | `/api/candidates/{id}/merits` | Registro de merito |
| `POST` | `/api/candidates/{id}/baremo` | Calculo de autobaremo |
| `GET` | `/api/candidates/{id}/expediente` | Exportacion de expediente |

Rutas recomendadas para portal productivo:

| Metodo | Ruta | Caso de uso |
| --- | --- | --- |
| `GET` | `/api/convocatorias` | Listar convocatorias visibles |
| `POST` | `/api/convocatorias` | Crear convocatoria administrativa |
| `GET` | `/api/convocatorias/{id}` | Consultar convocatoria |
| `POST` | `/api/convocatorias/{id}/solicitudes` | Registrar solicitud |
| `GET` | `/api/convocatorias/{id}/listados/provisional` | Ver listado provisional |
| `POST` | `/api/convocatorias/{id}/listados/provisional` | Publicar provisional |
| `GET` | `/api/convocatorias/{id}/listados/definitivo` | Ver listado definitivo |
| `POST` | `/api/convocatorias/{id}/listados/definitivo` | Publicar definitivo |
| `POST` | `/api/candidates/{id}/documents` | Adjuntar evidencia documental |

Las rutas nuevas deben entrar por handlers finos y llamar usecases. El prefijo
API sigue siendo configuracion de composicion, no constante repetida en
adaptadores.

## Configuracion

Superficie canonica actual:

| Variable | Default | Uso |
| --- | --- | --- |
| `BOLSA_HTTP_ADDR` | `:8080` | Direccion de escucha HTTP |

Superficie propuesta, a incorporar solo cuando exista adaptador real:

| Variable | Default | Uso |
| --- | --- | --- |
| `BOLSA_HTTP_ADDR` | `:8080` | Direccion HTTP |
| `BOLSA_API_BASE_PATH` | `/api` | Prefijo API |
| `BOLSA_READ_HEADER_TIMEOUT` | `5s` | Timeout de cabeceras |
| `BOLSA_STORAGE_DRIVER` | `memory` | `memory` o adaptador duradero |
| `BOLSA_DATABASE_URL` | vacio | DSN solo si `BOLSA_STORAGE_DRIVER` lo requiere |
| `BOLSA_I18N_DIR` | `web/static/locales` | Catalogos de mensajes |

Regla: cada variable se define, normaliza y documenta en `config`. Ningun
adaptador lee `os.Getenv` directamente.

## Persistencia

Estado actual:

- `repository.NewMemoryStore()` conserva candidatos, meritos, convocatorias y
  solicitudes en memoria.
- No hay migraciones ni base de datos productiva.

Evolucion propuesta:

```text
internal/candidate/ports/
  candidate_repository.go
  merit_repository.go
  procedure_repository.go
internal/candidate/adapters/repository/
  memory/
  sql/
    candidate_repository.go
    merit_repository.go
    procedure_repository.go
db/
  migrations/
```

El adaptador SQL debe implementar puertos existentes o puertos pequenos nuevos.
Las transacciones, migraciones, locks y serializacion son detalle del adaptador.
Los usecases solo ven repositorios y errores de puerto.

## i18n

`internal/shared/i18n` sigue como punto unico de catalogos. Reglas:

- API devuelve claves traducidas en envelopes, con fallback si falta catalogo.
- Frontend reutiliza catalogos estaticos bajo `web/static/locales`.
- Claves por dominio: `api.candidate.*`, `api.procedure.*`,
  `ui.portal.*`.
- No mezclar literales productivos dentro de handlers o componentes si el texto
  debe ser visible para usuario final.

## Tests

Pruebas actuales obligatorias:

```bash
go test ./...
```

Matriz de pruebas por frontera:

- Dominio: validaciones, ranking, baremo y transiciones sin mocks externos.
- Usecases: fakes de puertos y errores de repositorio/autenticacion.
- Handlers HTTP: status, envelopes, rutas, auth, DTOs y JSON estricto.
- Server: health, estaticos, prefijo API y metodo no permitido.
- Repositorios: contrato comun para memoria y futuros adaptadores duraderos.
- Frontend estatico: smoke manual o test de endpoint para `/`, `/app.js` y
  consumo de `/api/demo`.
- Config: defaults, normalizacion y lectura de entorno.

Todo refactor debe mantener `go test ./...` verde antes de ampliar alcance.

## Orden tecnico de refactor

1. Congelar comportamiento con tests actuales de handler, server, dominio y
   usecases.
2. Extraer DTOs, errores y routing del handler sin cambiar rutas.
3. Mover wiring de `NewPrototypeHandler` a `internal/app/composition`.
4. Hacer `BOLSA_API_BASE_PATH` y timeout configurables desde `config` si se
   necesitan en ejecucion.
5. Introducir servicios frontend pequenos (`api.js`, componentes de listado)
   manteniendo la pantalla actual.
6. Crear puertos faltantes solo cuando un caso de uso los necesite.
7. Anadir adaptador duradero detras de `BOLSA_STORAGE_DRIVER`, con suite de
   contrato compartida.
8. Separar `/api/demo` de rutas productivas; conservar demo como fixture o
   endpoint solo de desarrollo.
9. Anadir catalogos i18n externos para UI y API.
10. Ejecutar `go test ./...` tras cada paso y antes de integrar.

## Riesgos y limites

- El handler concentra demasiadas responsabilidades; dividirlo reduce riesgo
  antes de anadir rutas.
- La autenticacion fake no debe filtrarse a composicion productiva.
- La memoria no sirve para concurrencia, recuperacion ni auditoria legal.
- Las rutas frontend futuras requieren fallback del servidor si usan History
  API.
- La demo administrativa crea datos cada ejecucion; no debe confundirse con API
  productiva.

## Referencias de entrega

Se preservan como referencias opacas de Orquesta:

- `worktree_ref=worktree-bolsa-vec-portal-005`
- `branch_ref=branch-bolsa-vec-portal-005`

No son rutas locales ni nombres Git dentro del diseno tecnico.
