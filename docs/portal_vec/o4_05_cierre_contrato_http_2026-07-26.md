# O4-05: cierre del contrato HTTP de cobertura

**Fecha:** 26 de julio de 2026

**Commit verificado:** `1a764cf`

**Resultado:** `GO` independiente para el adaptador aislado

## Alcance cerrado

El corte alinea alta y cobertura con la lista positiva que el portal interno
sirve realmente:

- `POST /api/vec/contratacion-temporal/solicitudes`;
- `POST /api/vec/contratacion-temporal/cobertura/propuesta`;
- `POST /api/vec/contratacion-temporal/cobertura/decisiones`;
- `POST /api/vec/contratacion-temporal/cobertura/rectificaciones`.

Las rutas son exactas, no incluyen versión y rechazan el alias histórico
`/api/interno/v1/...`, barras, escapes, consultas y métodos no admitidos.

La autoridad sigue fuera de HTTP. El adaptador clasifica el contexto
corporativo ausente o caducado como `401`, otra organización como `403` y la
autoridad no disponible como `503`. Los detalles privados no salen en JSON y
ningún caso de uso se ejecuta sin contexto confiable.

## Resultado incierto

Si la transacción pudo alcanzar `COMMIT` y la lectura primaria aún no concluye,
la aplicación devuelve un error compuesto que satisface
`ErrConfirmacionDecisionCoberturaPendiente` y conserva compatibilidad con
`ErrConfirmacionDecisionCoberturaNoDisponible`.

HTTP lo proyecta como `503 operacion_pendiente`, elimina `Retry-After` y no
publica el detalle interno. Un fallo acreditado antes de `COMMIT` continúa
como indisponibilidad ordinaria, no queda pendiente y no reconcilia.

## Evidencia

- pruebas focales con detector de carreras en HTTP y aplicación;
- suite Go completa;
- `go vet ./...`;
- formato y `git diff --check`;
- matriz positiva y negativa de reglas Gitleaks;
- revisión independiente sin bloqueos.

## Lo que no acredita

Este `GO` no publica por sí solo una ruta productiva. Permanecen pendientes la
raíz de composición, la frontera mTLS/Kerberos, los pools PostgreSQL de mínimo
privilegio, el cliente web, la recuperación protegida de recibos y el E2E
productivo. Hasta cerrar esos cortes, `cmd/vec-interno` debe seguir fallando
cerrado.

El siguiente corte mecánico ya registra los manejadores de forma atómica detrás
de una autoridad obligatoria. Su alcance y límites constan en
[O4-05: registro seguro de rutas y composición](o4_05_registro_rutas_y_composicion_2026-07-26.md).
