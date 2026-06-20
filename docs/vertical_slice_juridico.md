# Vertical slice juridico-administrativo

## Objetivo

Documentar el corte funcional implementado para una bolsa de empleo:
candidatos, meritos, baremo, expediente y demo de convocatoria. Este documento
describe lo que existe hoy; no abre alcance productivo.

## Fronteras hexagonales

Nucleo neutral:

- `internal/candidate/domain` contiene entidades y reglas de validacion.
- No depende de HTTP, Docker, base de datos, Orquesta ni rutas locales.

Puertos:

- `CandidateRepository`, `MeritRepository`, `Authenticator` y puertos de
  procedimiento separan dominio/casos de uso de adaptadores.

Adaptadores opt-in:

- Handler HTTP para API local.
- Repositorios en memoria para prototipo.
- Autenticador fake con dos identidades precargadas.

Composicion:

- `cmd/bolsa-server/main.go` crea handler prototipo, carga `config.Load()` y
  arranca `server.NewHTTPServer`.
- Configuracion canonica actual: `BOLSA_HTTP_ADDR`.

## Flujo ciudadano

1. `POST /api/candidates` crea candidato.
2. `POST /api/candidates/{id}/merits` registra un merito.
3. `POST /api/candidates/{id}/baremo` calcula puntuacion.
4. `GET /api/candidates/{id}/expediente` devuelve candidato, meritos y baremo.

Autenticacion simulada:

- `X-Auth-Mechanism: clave`
- `X-Auth-Subject: candidate`
- `Authorization: Bearer citizen-token`

## Flujo administrativo

`POST /api/demo` ejecuta una demo con:

- convocatoria `demo-convocatoria`;
- dos solicitudes;
- publicacion de listado provisional;
- publicacion de listado definitivo;
- desempate/ranking calculado por reglas de baremo.

Autenticacion simulada:

- `X-Auth-Mechanism: kerberos_ad`
- `X-Auth-Subject: staff`
- `Authorization: Bearer staff-token`

## Reglas de baremo prototipo

Reglas cargadas por `defaultRuleSet()`:

| Merito | Seccion | Unidad | Puntos |
| --- | --- | --- | --- |
| `experiencia_misma_categoria` | experiencia | meses | 0.2 |
| `experiencia_otra_categoria` | experiencia | meses | 0.1 |
| `formacion_titulo` | formacion | puntos declarados | 1 |
| `formacion_curso` | formacion | horas | 0.05 |
| `otros` | otros | puntos declarados | 1 |

Topes:

| Seccion | Maximo |
| --- | --- |
| experiencia | 50 |
| formacion | 30 |

Desempates:

1. mayor experiencia;
2. mayor formacion;
3. letra de sorteo.

## Probado

- Tests unitarios y de handler con `go test ./...`.
- Health endpoint publico.
- Routing `/api` y rutas `/api/candidates`.
- Handler prototipo con flujo completo de candidato, merito, baremo y
  expediente.
- Demo administrativa con listado definitivo de dos solicitudes.

## Simulado

- Identidades ciudadana y personal interno.
- Repositorios en memoria.
- Datos de convocatoria demo.
- Catalogo i18n fallback si no hay locales externos.

## Pendiente productivo

- Integracion real con identidad, roles y sesiones.
- Persistencia duradera por convocatoria/solicitud/merito.
- Parametrizacion legal de reglas de baremo por convocatoria publicada.
- Auditoria, firma, evidencias documentales, notificaciones y trazabilidad.
- Control de concurrencia y ciclo completo de alegaciones/subsanaciones.

## Smoke esperado

Con servidor en `127.0.0.1:8080`:

```bash
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS -X POST http://127.0.0.1:8080/api/demo \
  -H 'X-Auth-Mechanism: kerberos_ad' \
  -H 'X-Auth-Subject: staff' \
  -H 'Authorization: Bearer staff-token'
```

El primer comando devuelve `{"status":"ok"}`. El segundo devuelve un envelope
JSON con `data.convocatoria.id` igual a `demo-convocatoria`.
