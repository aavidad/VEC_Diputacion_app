# Revisión independiente CT-000043A

Fecha: 29 de julio de 2026.

## Alcance

CT-000043A corrige de forma aditiva el contrato ya publicado de detalle RRHH:
`version_observada=0` significa primera carga de la versión actual. La consulta
y su huella VEC conservan el cero; el registro de acceso, la prueba durable y
el Recibo RRHH V2 conservan la versión positiva realmente materializada.

Commit productor:

```text
d2ba87e200a9f104c412e50ced98c55983afcec2
```

Commit integrado:

```text
35cba6afc76fa444e6af8ec521570fbbfbf7e4fa
```

## Resultado

Dos revisores distintos del productor reprodujeron el ejecutor PostgreSQL
completo y emitieron `GO`. Revisaron además, de forma separada, las puertas
estáticas aplicables.

| Severidad | Revisor 1 | Revisor 2 |
| --- | ---: | ---: |
| P0 | 0 | 0 |
| P1 | 0 | 0 |
| P2 | 2 coberturas heredadas | 0 |

Las coberturas P2 del primer revisor no alteran el veredicto: CT-000043 ya
prueba el canon completo del Recibo y sus mutaciones, mientras CT-000043A
acredita de forma específica el canon de consulta con cero y las versiones
positivas en las filas durables. La carrera exige un único ganador y la misma
matriz inyecta por separado los cuatro estados transitorios exactos.

## Evidencia reproducida

```bash
./deploy/postgresql/contratacion_temporal/\
probar_o4_05_corrector_detalle_version_actual_pg18_4.sh
```

El ejecutor PostgreSQL acredita:

- PostgreSQL 18.4 fijado por resumen;
- CT-000039 a CT-000043 y AUT-000003 a AUT-000006;
- `UP`, reentrada, `DOWN`, reentrada y segundo `UP`;
- bloqueo de la reversión CT-000043 mientras el corrector está aplicado;
- cuerpo, atributos, configuración, ACL, comentario y dependencias sellados;
- restauración byte a byte del cuerpo anterior;
- `0 → versión actual positiva`, `N = N` y rechazo de `N ≠ actual`;
- canon y huella VEC originales, sin normalizar el cero;
- mutación y cruce VEC sin efectos;
- rollback, replay, concurrencia y revocación viva;
- propagación exacta de `40001`, `40P01`, `55P03` y `57014`.

Dirección repitió la puerta sobre el commit integrado y obtuvo verde.

El productor ejecutó además:

```bash
TMPDIR="$HOME/.cache" go test ./internal/modules/contrataciontemporal/...
TMPDIR="$HOME/.cache" go test -race ./internal/modules/contrataciontemporal/...
TMPDIR="$HOME/.cache" go vet ./internal/modules/contrataciontemporal/...
TMPDIR="$HOME/.cache" go test ./...
TMPDIR="$HOME/.cache" go vet ./...
bash -n deploy/postgresql/contratacion_temporal/\
probar_o4_05_corrector_detalle_version_actual_pg18_4.sh
shellcheck deploy/postgresql/contratacion_temporal/\
probar_o4_05_corrector_detalle_version_actual_pg18_4.sh
git diff --check
gitleaks protect --staged --redact --no-banner
```

Los resultados fueron verdes. Dirección repitió `bash -n`, ShellCheck,
`git diff --check`, el límite de 800 líneas y Gitleaks sobre el conjunto
preparado. Los revisores comprobaron esos resultados y repitieron las puertas
estáticas compatibles; no se atribuyen al ejecutor PostgreSQL.

## Seguridad y límites

El corrector no crea tablas, roles, permisos, fachadas ni nuevas autoridades.
Mantiene las barreras `23/7`. No contiene datos personales, secretos ni rutas
privadas. Los mensajes, comentarios e identificadores propios permanecen en
castellano coherente.

CT-000043A no habilita producción. Siguen pendientes CT-000044, CT-000045,
adaptador Go, composición raíz, TLS viva, E2E y las aprobaciones formales.
