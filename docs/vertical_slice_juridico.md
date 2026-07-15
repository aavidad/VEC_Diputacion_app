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
- Autenticador fake cerrado: resuelve una credencial Bearer opaca contra un
  fichero local obligatorio; no existen identidades ni secretos precargados.

Composicion:

- `cmd/vec-server/main.go` carga `config.Load()` y arranca el unico servidor
  canonico. El alias `cmd/bolsa-server` esta retirado y falla cerrado.
- Configuracion canonica actual: `VEC_HTTP_ADDR`.

## Flujo ciudadano

1. `POST /api/candidates` crea candidato.
2. `POST /api/candidates/{id}/merits` registra un merito.
3. `POST /api/candidates/{id}/baremo` calcula puntuacion.
4. `GET /api/candidates/{id}/expediente` devuelve candidato, meritos y baremo.

Autenticacion local: `Authorization: Bearer $VEC_FAKE_TOKEN`. El token aleatorio
no se incluye en codigo, JavaScript, documentacion ni imagenes; el servidor
obtiene sujeto, roles, mecanismo y garantia exactos del fichero indicado por
`VEC_FAKE_CREDENTIALS_FILE`.

## Flujo administrativo

`POST /api/demo` ejecuta una demo con:

- convocatoria `demo-convocatoria`;
- dos solicitudes;
- publicacion de listado provisional;
- publicacion de listado definitivo;
- desempate/ranking calculado por reglas de baremo.

La llamada usa igualmente el Bearer local. Las cabeceras `X-Auth-*` y `X-VEC-*`
no son autoridad en modo fake, aunque un cliente intente enviarlas.

## Reglas de baremo prototipo

Reglas cargadas para la convocatoria de demostracion indicada de forma
explicita. No existe una convocatoria de reserva: una peticion sin
`call_id`, con comodines o distinta de la configurada se deniega.

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
  -H "Authorization: Bearer $VEC_FAKE_TOKEN"
```

El primer comando devuelve `{"status":"ok"}`. El segundo devuelve un envelope
JSON con `data.convocatoria.id` igual a `demo-convocatoria`, siempre que el
operador haya generado el token fuera del repositorio y haya guardado solo su
SHA-256 en un fichero regular `0600`. La configuracion exacta y sus limites se
documentan en `docs/portal_vec/autenticacion_fake_local_segura.md`.
