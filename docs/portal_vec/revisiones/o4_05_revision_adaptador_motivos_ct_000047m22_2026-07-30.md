# Revisión del adaptador nominal de motivos M2.2

Fecha: 30 de julio de 2026.

Resultado final: **GO**, con `P0=0`, `P1=0` y `P2=0`.

Commit integrado:

```text
45f985c6a4b349c66636a4c69ec6cb82f6a09b60
```

## Alcance

M2.2 implementa `ports.ResolutorMotivoConsultaRRHH` sobre el pool nominal
M2.1. El adaptador:

- expone solo las operaciones nominales de cuadro y detalle;
- usa dos consultas SQL literales, sin nombre de función ni selector aportado
  por el llamador;
- limita a dos resultados y exige cardinalidad exactamente uno;
- abre `SERIALIZABLE READ WRITE`, reacredita dentro de la misma transacción y
  confirma antes de devolver;
- valida UTC, año, precisión de microsegundo, versión portable y referencia
  de motivo V2;
- revierte con un contexto técnico independiente y acotado;
- devuelve referencia cero y el centinela opaco ante toda incertidumbre.

El constructor público solo acepta `PoolResolucionMotivosRRHHPostgreSQL`. La
frontera estrecha para dobles permanece privada al paquete.

## Hallazgo y corrección durante la revisión

La revisión de seguridad detectó antes de confirmar un `P1`: un
`context.Context` que contuviera un puntero nulo tipado podía provocar un
segundo pánico al normalizar el error.

La corrección:

- usa `dependenciaNula` antes de invocar el contexto;
- endurece el normalizador compartido con centinela por defecto, recuperación
  propia y una sola llamada a `ctx.Err`;
- incorpora una regresión cuyo método `Err` panica si llega a invocarse.

La revisión funcional encontró además dos carencias probatorias `P2`. Se
cerraron fijando en las pruebas los nombres y campos SQL exactos, y cubriendo
nulos tipados, errores de reversión y pánicos de inicio, reacreditación,
consulta, escaneo y confirmación.

## Evidencia

Superaron:

```text
go test ./internal/modules/contrataciontemporal/adapters/postgres
go test -race ./internal/modules/contrataciontemporal/adapters/postgres \
  -run 'TestResolutorMotivosRRHHPostgreSQL|TestNuevoResolutorMotivoConsultaRRHHPostgreSQL|TestNuevoPoolResolucionMotivosRRHHPostgreSQL' \
  -count=1
go test ./internal/modules/contrataciontemporal/...
go vet ./internal/modules/contrataciontemporal/adapters/postgres
gofmt
git diff --check
gitleaks protect --staged --redact --no-banner
```

Tamaños y huellas del corte:

| Fichero | Líneas | SHA-256 |
| --- | ---: | --- |
| `fabrica_pool_resolucion_motivos_rrhh.go` | 355 | `7e812d0d89a2126c53155b286b2985aea7e51b79b442fd0fbd996935f553472f` |
| `resolucion_motivos_rrhh_postgresql.go` | 179 | `8698240f42b80146c73cec94532946aa366250a2c161c5f1ac5db3655762fc14` |
| `resolucion_motivos_rrhh_postgresql_test.go` | 795 | `b04955a597fdd23c92de93f74bfe8f1c1750e0c8548dd22210e940ea4d32122d` |

M2.2 no acredita PostgreSQL real: esa evidencia corresponde a M2.3. Tampoco
compone todavía el resolutor en la raíz productiva ni autoriza producción.
