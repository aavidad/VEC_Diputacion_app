# Revisión independiente O2-08B

Fecha: 23 de julio de 2026.

## Dictamen

**GO** para el adaptador aislado exacto
`3e2885c30c8c0c849a7be9ef2c0f42b420b71639`.

La revisión no encontró hallazgos críticos, altos, medios ni bajos atribuibles
al candidato. El GO permitió integrar sus commits como
`42dc3ac`–`94e09e8` en `feature/contratacion-temporal`.

No acredita el registro de la ruta, la composición O2-07, la reconciliación
O2-06, el E2E O2-10 ni producción.

## Evidencia funcional y de seguridad

- El cuerpo contiene únicamente el sobre cerrado
  `{clave_idempotencia, solicitud}`.
- La UUIDv4 identifica una intención no confiable; no representa identidad,
  autenticación, perfil, organización, permiso ni capacidad.
- La autoridad llega por el contexto de servidor y debe aportar autenticación,
  sesión, perfil y organización, con solicitud y clave vacías.
- La aplicación liga mediante HMAC la UUID, actor, perfil, organización, flujo
  y contenido; PostgreSQL recibe sellos HMAC y no la UUID original.
- Query, cabeceras de autoridad, cookies, campos desconocidos o repetidos,
  `null`, segundo documento, UTF-8 no canónico y entradas fuera de límite se
  rechazan.
- La UUID no se devuelve en respuesta, error, cabecera, correlación ni estado
  durable del adaptador.
- El catálogo de errores es estable, redactado e i18n; un resultado
  indeterminado no induce un segundo efecto.
- OpenAPI 3.1 describe el mismo sobre y mantiene los objetos cerrados.
- El comando coincide con O2-09A y no requiere cookies, almacenamiento web ni
  cabeceras de autoridad. Web, escritorio, API, CLI y MCP comparten contrato.

## Puertas reproducidas

```text
go test ./internal/modules/contrataciontemporal/adapters/httpinterno -count=50
go test -race ./internal/modules/contrataciontemporal/adapters/httpinterno -count=5
go test ./...
go vet ./...
node --test web/static/portal-empleado/modulos/contratacion-temporal/contratacion-temporal.test.mjs
git diff --check a736105..3e2885c
git fsck --full
```

Además:

- aplicación, seguridad y PostgreSQL: veinte repeticiones;
- interfaz O2-09A exacta `1323b4b`: 20/20;
- tamaños dentro del límite;
- Gitleaks en el rango: cero fugas;
- integración virtual contra `6c56602`: sin conflictos;
- candidato y worktree: limpios.

Tras integrar, Dirección volvió a ejecutar veinte repeticiones HTTP, carrera
×3, todos los paquetes de contratación temporal, `go vet` focal y
`git diff --check`, sin fallos.

## Pendientes que conservan el NO-GO funcional

1. O2-05: confirmación atómica PostgreSQL.
2. O2-06: adaptador y reconciliación.
3. O2-07: composición y registro de la ruta con fallo cerrado.
4. Validación semántica externa del OpenAPI 3.1.
5. O2-09: integrar la vista revisada.
6. O2-10: E2E y aceptación RRHH.
