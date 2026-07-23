# Encargo aislado O2-08A — API interna definitiva del alta de contratación

## Mandato

Lee este documento completo y ejecuta el encargo sin ampliar alcance. Antes de
editar, lee `AGENTS.md`, `ORQUESTACION_AGENTES.md`,
`docs/instruccion_direccion_2026-07-18.md`,
`docs/portal_vec/tablero_tareas_contratacion_temporal_2026-07-23.md` y
`docs/portal_vec/diseno_transaccion_atomica_alta_o2_05_2026-07-23.md`.

Son obligatorias todas las reglas del proyecto: código, pruebas, mensajes y
documentación en castellano coherente; i18n mediante claves estables;
arquitectura hexagonal y adaptadores intercambiables; denegación por defecto;
neutralidad web/escritorio/CLI/MCP; ausencia de secretos y datos personales;
documentación simultánea; seguridad de Administración pública y todas las
puertas de `AGENTS.md`. Los términos técnicos universales pueden conservarse,
pero no se admite mezclar castellano e inglés en el dominio.

No uses Orquesta. No edites el árbol principal. No integres ni empujes tu rama.

## Subagentes obligatorios

El agente principal es el **único editor** del worktree. Debe crear dos
subagentes de revisión, de solo lectura y sin commits:

1. revisor de seguridad HTTP/administración pública: autenticación confiable,
   mínimo privilegio, límites, errores, cookies, cabeceras, replay y
   minimización;
2. revisor de contrato/OpenAPI y pruebas adversariales: correspondencia exacta
   Go↔JSON↔OpenAPI, clientes web/escritorio/CLI/MCP y compatibilidad futura.

Los revisores emiten GO/NO-GO con evidencia. El principal reproduce y corrige
los bloqueos. Ningún revisor modifica archivos ni integra.

## Preparación obligatoria

Desde `/home/alberto/Trabajo/VEC_Diputacion_app`:

```bash
git worktree list
test ! -e .worktrees/ct-o2-08-api
git worktree add .worktrees/ct-o2-08-api \
  -b agent/ct-o2-08-api feature/contratacion-temporal
cd .worktrees/ct-o2-08-api
```

Si rama o worktree ya existen, detente e informa. No crees otro nombre. Todo el
trabajo debe permanecer dentro de
`/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-o2-08-api`.

## Objetivo

Construir la parte paralelizable y definitiva de O2-08:

- contrato JSON cerrado del alta interna;
- adaptador HTTP interno con dependencias inyectadas;
- resolución de contexto autenticado exclusivamente mediante una autoridad
  confiable del servidor;
- límites, códigos, envelope, recibo minimizado y OpenAPI;
- pruebas adversariales exhaustivas.

O2-08A **no compone** PostgreSQL ni completa O2-08: O2-07 conectará el servicio
real cuando O2-05/O2-06 estén cerrados. No se permite un servicio falso,
repositorio en memoria, éxito simulado ni ruta de desarrollo disfrazada de
real.

## Fuentes que mandan

- `internal/modules/contrataciontemporal/application/registro_solicitud.go`;
- `internal/modules/contrataciontemporal/domain/solicitud.go`;
- `internal/modules/contrataciontemporal/ports/alta.go`;
- `internal/modules/contrataciontemporal/manifest.go`;
- patrones de seguridad en `internal/vec/adapters/httpseguridad`;
- patrón de adaptador interno en `internal/modules/bolsa/adapters/httpinterno`,
  sin copiar decisiones funcionales de Bolsa.

No cambies los contratos anteriores para acomodar HTTP. Si falta una frontera,
declárala mínima dentro del adaptador y documenta la necesidad de composición.

## Archivos permitidos

Crea exclusivamente:

```text
internal/modules/contrataciontemporal/adapters/httpinterno/
  alta.go
  alta_contrato.go
  alta_errores.go
  alta_test.go
  alta_seguridad_test.go
  alta_limites_test.go
  doc.go
docs/api/contratacion_temporal_alta_interna_v1.yaml
docs/portal_vec/o2_08a_api_interna_alta_2026-07-23.md
```

Puedes añadir archivos de prueba en ese mismo paquete si ningún fichero supera
800 líneas. No modifiques aplicación, dominio, `ports`, SQL, composición,
servidores, web, Bolsa ni documentación compartida.

## Contrato funcional

El cuerpo solo contiene datos funcionales de la solicitud. No admite:

- autenticación, sesión, perfil, actor, cuenta, nombre, DNI, correo o rol;
- organización declarada por el cliente;
- permisos, decisiones, capacidades, HMAC, claves de idempotencia en claro,
  atestaciones ni referencias internas de seguridad.

Una dependencia de servidor debe resolver un contexto ya autenticado y
autorizable con, al menos, autenticación, sesión, perfil, organización y una
clave de idempotencia/correlación opaca ligada al cliente y a la petición. El
handler no obtiene autoridad de cabeceras de texto libres, query params,
cookies ni campos JSON.

La entrada funcional debe corresponder exactamente con
`domain.SolicitudCentro`: centro, contacto referenciado, categoría,
grupo/subgrupo, motivo, detalle, periodo, retención de crédito y referencias
opacas de adjuntos. Rechaza campos desconocidos, duplicados, `null` ambiguos,
números no enteros, fechas no UTC, tamaños y colecciones excesivos.

La salida de éxito usa el envelope institucional y proyecta únicamente:

- referencia de expediente;
- número visible;
- versión;
- referencia de recibo;
- fecha de confirmación.

No devuelve auditoría, evento, decisión, correlación, identidad ni evidencia
criptográfica. Un recurso de consulta autorizado podrá exponerlos después.

## Diseño hexagonal

- Define una interfaz mínima del ejecutor que acepte `context.Context` y
  `application.SolicitudRegistrarExpediente`, y devuelva `ports.ReciboAlta`.
- Define una autoridad mínima de contexto de canal. Debe ser inyectada y
  neutral respecto de certificado, Kerberos, app de escritorio, CLI o MCP.
- El handler traduce HTTP↔comando; no contiene reglas de negocio ni conoce
  PostgreSQL.
- No expongas un constructor público capaz de fabricar contexto autenticado
  desde cadenas del cliente.
- La ruta propuesta es
  `POST /api/interno/v1/contratacion-temporal/solicitudes`.
  No la registres todavía en el servidor global: O2-07 hará esa composición.

## HTTP y seguridad

- Solo `POST`; `GET`, `PUT`, `PATCH`, `DELETE`, `OPTIONS` no concedidos fallan
  de forma cerrada según el router institucional.
- `Content-Type: application/json`; `Accept` compatible; cuerpo limitado antes
  de decodificar; exactamente un documento JSON y EOF.
- Sin `Cookie` ni `Set-Cookie`; no leer `Authorization`, `Remote-User`,
  `X-Remote-User`, `X-Forwarded-User` ni cabeceras de rol dentro del handler.
  La autoridad inyectada consume únicamente el contexto confiable preparado
  por la superficie interna.
- No reflejar mensajes privados. Define catálogo público estable de errores con
  `codigo`, clave i18n y correlación opaca del servidor.
- Propaga cancelación y plazos. No convierte un `COMMIT` confirmado en error si
  recibe un recibo válido; un resultado indeterminado pertenece a O2-06 y se
  proyecta como estado recuperable, no como reintento ciego.
- No CORS permisivo, redirecciones, caché compartida, compresión de secretos,
  telemetría externa ni datos personales en logs.
- Cabeceras de respuesta: tipo, no-store y defensas institucionales aplicables;
  no dupliques la terminación TLS.
- Denegación por defecto y comparaciones de error mediante `errors.Is`.

## OpenAPI

El YAML debe ser OpenAPI 3.1, sin servidores ni credenciales reales. Incluye:

- esquema cerrado con `additionalProperties: false`;
- límites exactos alineados con Go;
- UTC, enteros y céntimos sin coma flotante;
- respuestas de éxito y error minimizadas;
- `operationId` y nombres coherentes en castellano;
- seguridad descrita como contexto interno aportado por la plataforma, sin
  afirmar que una cabecera libre autentica.

Añade una prueba que lea el YAML con herramientas ya disponibles o, si no hay
parser sin dependencia nueva, una comprobación estructural suficiente y deja
documentada la validación externa pendiente. No añadas librerías.

## Pruebas mínimas

1. éxito con ejecutor y contexto confiables;
2. campos extra/duplicados, segundo JSON, cuerpo vacío/grande y tipos erróneos;
3. fechas, periodo, céntimos, moneda, detalle y adjuntos en límites;
4. contexto ausente, caducado, cancelado o de otra organización;
5. intento de inyectar identidad/rol/organización por body, query y cabeceras;
6. errores privados no salen en cuerpo, cabeceras ni logs;
7. recibo incompleto, adulterado, futuro o no ligado se rechaza;
8. doble envío y propagación de clave opaca sin exponerla;
9. cancelación/timeout antes del ejecutor y resultado recuperable posterior;
10. ninguna respuesta contiene `Set-Cookie` ni datos prohibidos;
11. carrera concurrente y ausencia de estado mutable global;
12. correspondencia exacta entre DTO, OpenAPI y caso de uso.

Ejecuta:

```bash
go test -race ./internal/modules/contrataciontemporal/adapters/httpinterno -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

Comprueba tamaño, Gitleaks si está disponible y árbol limpio.

## Entrega

Commits pequeños en castellano; no empujes. Entrega SHA, archivos, pruebas,
dictámenes de los dos subagentes, riesgos y declaración expresa:

**candidato O2-08A; no registrado, no compuesto, no integrado y no productivo
hasta O2-07/O2-06**.
