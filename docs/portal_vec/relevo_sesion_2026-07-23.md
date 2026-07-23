# Relevo de sesión — 23/07/2026

Estado al cerrar la sesión por petición del responsable. No se debe presentar
ningún checkpoint parcial como capacidad productiva.

## Estado conservado

### Presentación remota

- La versión de código desplegada continúa siendo `ae84f1c`.
- `https://vep.dipgra.es` vuelve a responder correctamente tras autenticación.
- La causa del rechazo era operativa: el Nginx de entrada seguía buscando
  `htpasswd` en `/tmp`, que se pierde al recrear el contenedor.
- Se recreó únicamente `entrada-remota-presentacion` con la ACL que referencia
  el secreto persistente `/run/secrets/vec_presentacion_htpasswd`.
- El contenedor quedó `running/healthy`; la prueba autenticada terminó en
  `200`, la no autenticada en `401`, y los otros servicios no se reiniciaron.
- La configuración externa activa para esta corrección se versionó con el
  sufijo `ae84f1c-r2`. No contiene la contraseña.
- La credencial compartida durante la conversación debe rotarse. No se añadió
  a Git ni a este documento.

### C3 — consulta pública PostgreSQL

- Rama publicada: `real/c3-convergencia`.
- Corte exacto: `eeac3a2`.
- Dictamen independiente: **GO, cero hallazgos MED+**.
- Incluye PostgreSQL 18, TLS verificado, ACL de mínimo privilegio, publicación
  atómica, catálogos históricos, lectores MVCC sin candado compartido,
  `readiness` con comprobación integral, fallo cerrado y gobierno del LOGIN
  publicador.
- Se corrigieron dos incompatibilidades detectadas en PostgreSQL real:
  serialización del timeout en milisegundos y sintaxis de `extract`.
- Evidencia verde: pruebas Go focales, `-race`, `go vet`, 19 pruebas web y
  runner PostgreSQL 18/TLS completo.
- La rama está limpia y sincronizada con `origin`.

### T17A — persistencia del importador Convoca

- Rama publicada: `real/t17a-postgres-importador`.
- Checkpoint exacto: `017a82a`.
- Estado declarado: **fundamento compilable; T17A no implementado**.
- Incluye el contrato de protección sustituible del staging, sobres cifrados,
  derivación ciega HMAC-SHA-256, serialización cerrada, copias defensivas y
  clasificación saneada de errores.
- El relevo detallado está en
  `internal/modules/bolsa/adapters/postgresimportacionconvoca/README_RELEVO.md`.
- Faltan repositorio transaccional/CAS, migraciones y roles, pruebas unitarias
  e integración PostgreSQL 18/TLS.
- La rama está limpia y sincronizada con `origin`.

### T20 — autorización V3 de borradores

- Rama publicada: `real/t20-borradores-postgresql`.
- Checkpoint documental exacto: `36c5925`.
- Se conservó un diseño WIP de DEC-103 que llevaba pendiente sin commit.
- El propio documento mantiene el estado **NO-GO**, no implementado y
  pendiente de revisión. No debe mezclarse ni darse por aprobado sin una nueva
  auditoría.
- La rama está limpia y sincronizada con `origin`.

## Foto de avance mantenida

| Medida | Estado |
|---|---:|
| Presentación DEMO de Bolsa | 100 % |
| Componentes reales construidos/probados por separado | aproximadamente 74 % |
| Flujos reales integrados E2E | aproximadamente 42 % |
| Producción aceptada | pendiente de integraciones y validaciones formales |

No se eleva el porcentaje E2E solo por cerrar contratos o checkpoints
aislados.

## Orden de reanudación

1. Integrar el corte auditado de C3 sin rehacerlo y actualizar la matriz de
   capacidades con su evidencia.
2. Auditar y reconciliar `real/c4-raiz-interna` con C3. La raíz interna existe,
   pero permanece cerrada hasta recibir las dependencias reales de C5/C6.
3. Cerrar la vertical real de gestión de convocatorias/borradores: identidad,
   PDP, KMS, PostgreSQL, confirmación, recibo y E2E.
4. Continuar T17A en su write-set aislado y someterlo a revisión antes de
   integrarlo.
5. Seguir después con bases/baremo, candidatura, revisión, alegaciones,
   llamamientos, comunicaciones, contratos/ceses, documentación y firma.

## Comprobaciones de cierre

- No quedaron procesos `go test`, `ports.test` ni runners PostgreSQL del
  proyecto activos.
- Los worktrees revisados quedaron limpios tras guardar T17A y el borrador
  documental T20.
- No se desplegó código real de C3/T17A/T20 en el servidor de presentación.
- La presentación remota permaneció operativa durante el cierre.
