# Autoprogramacion Orquesta pendientes 2026-07-16

## Alcance

- Origen: [auditoria de diseno y seguridad 2026-07-16](portal_vec/auditoria_diseno_y_seguridad_2026-07-16.md)
  y decisiones DEC-050 a DEC-053 adoptadas en el
  [registro de decisiones](portal_vec/registro_decisiones.md).
- Este backlog sigue el contrato del scanner documental: cada pendiente Txx
  lleva origen, estado, area hexagonal, accion y evidencia. Ninguna entrada
  abre codigo desde texto narrativo: todas remiten a una DEC adoptada o a un
  hallazgo verificado con comandos reproducibles.
- Orden recomendado: T01 y T06 son documentales y desbloquean al resto;
  T02 tiene evidencia de fallo real en CI y precede a cualquier ampliacion
  de `internal/vec/ports`.

## Pendientes Txx

### T01 — Analisis de brecha del nucleo heredado (DEC-050)

- `origen`: DEC-050 del registro de decisiones.
- `estado`: completado 2026-07-16 por direccion: [analisis de brecha](portal_vec/brecha_nucleo_heredado_bolsa.md) fusionado; el porte queda pendiente de la decision de alcance de la inscripcion ciudadana.
- `area_hexagonal`: docs primero; nucleo y composicion despues.
- `accion`: documentar el inventario de capacidades de `internal/candidate`
  y el analisis de brecha contra `internal/modules/bolsa`, dejando en el
  registro que se porta, que ya esta cubierto y que se descarta. Sin ese
  documento no se abre codigo de porte ni de borrado.
- `evidencia`: `go list -f` verifica que solo `internal/app/bootstrap`
  importa `internal/candidate`; la API heredada solo se monta en `fake`.

### T02 — Extraer la logica canonica de `internal/vec/ports` (H-01)

- `origen`: auditoria H-01 y fallo real de CI.
- `estado`: nuevo.
- `area_hexagonal`: puerto hacia nucleo/subpaquetes.
- `accion`: programar el troceo por capacidad (autorizacion, documental,
  almacen, auditoria) moviendo derivaciones canonicas y criptograficas a
  subpaquetes; `ports` conserva interfaces y tipos de intercambio. Aplicar
  DEC-051 a cada fichero resultante.
- `evidencia`: run de CI 29462846251 fallo por timeout de `go test -race`
  (600 s) en `internal/vec/ports`; el paquete tiene 19.733 lineas de fuente
  y ficheros de 4.122 lineas (`ejecuciones_documentales_v3.go`).

### T03 — Cableado de modulos fuera de `httpapi` (H-02)

- `origen`: auditoria H-02.
- `estado`: nuevo.
- `area_hexagonal`: adaptador hacia composicion.
- `accion`: programar el traslado del montaje de modulos de
  `internal/vec/adapters/httpapi` (`workspace.go`, `cronos.go`,
  `personal_rpt.go`) a `internal/app/bootstrap`; `httpapi` recibe handlers o
  interfaces ya compuestos y pierde todos los imports de
  `internal/modules/*`.
- `evidencia`: `go list -f '{{.ImportPath}} {{join .Imports " "}}'` muestra
  a `httpapi` importando cinco modulos, incluidos `application` y
  `adapters/file`/`memory` ajenos.

### T04 — Frontend en modulos ES con tokens (DEC-052, H-03)

- `origen`: DEC-052 del registro de decisiones.
- `estado`: en_curso, cedido al agente Orquesta el 2026-07-16: el subagente de direccion cayo sin commitear y el agente inicio la extraccion conforme a DEC-052 (`81a54d1` modulo cronos con tests JS, `5135eb4` arranque cerrado). La direccion no relanza subagente en este carril para evitar colision; revisa y sube.
- `area_hexagonal`: adaptador (frontend estatico).
- `accion`: programar la particion de `web/static/app.js` en modulos ES
  nativos por dominio funcional, eliminando en cada modulo migrado los
  estilos inline y colores literales; temas solo como redefinicion de tokens
  bajo atributo del documento. Criterio de terminado por modulo segun
  DEC-052.
- `evidencia`: `app.js` tiene 13.211 lineas, ~300 asignaciones `.style` y 67
  colores literales; `styles.css` ya define 49 tokens.

### T05 — Contrato API por modulo para clientes equivalentes (DEC-053)

- `origen`: DEC-053 del registro de decisiones; Ola 2 del plan de
  implantacion.
- `area_hexagonal`: puerto y adaptador HTTP.
- `estado`: completado 2026-07-16 la parte documental por direccion: [contratos API](portal_vec/contratos_api_modulos.md) fusionado; los endpoints de Ola 2 se abriran contra ese contrato.
- `accion`: documentar el contrato de endpoints por modulo (ruta, metodo,
  envelope, errores, version) y programar los endpoints finos de la Ola 2
  contra ese contrato; la web consume la API como un cliente mas y ningun
  cliente incorpora logica de negocio.
- `evidencia`: el frontend actual solo hace `fetch` real a la API heredada y
  `/api/demo`; las pantallas privadas operan con datos sinteticos locales.

### T06 — Nivel de madurez por modulo en el contrato (H-05)

- `origen`: auditoria H-05.
- `estado`: completado 2026-07-16 por direccion: seccion de niveles de madurez fusionada en el contrato de modulos.
- `area_hexagonal`: docs.
- `accion`: documentar en el
  [contrato de modulos](portal_vec/contrato_modulos_vec.md) el nivel que
  cumple cada modulo (completo, parcial, solo manifiesto) para que el shell
  no asuma capacidades inexistentes.
- `evidencia`: Bolsa tiene las cuatro capas; Cronos y Personal parciales;
  Dietas y Administracion solo manifiesto (arbol de `internal/modules/`).

### T07 — Coherencia frontend-API verificada en ejecucion

- `origen`: inconsistencias detectadas y verificadas en vivo por T05 en
  [contratos API por modulo](portal_vec/contratos_api_modulos.md), seccion
  "Inconsistencias detectadas".
- `estado`: nuevo. No ejecutar hasta fusionar el carril T04 (mismo write_set
  `web/static/**`).
- `area_hexagonal`: adaptador (frontend estatico) y composicion.
- `accion`: programar tres correcciones: (1) `staffHeaders()` y
  `candidateHeaders()` de `app.js` no envian `Authorization: Bearer`, que el
  modo `fake` exige — 401 verificado en `/api/vec/session`; (2) ningun perfil
  `fake` de serie tiene permisos funcionales de Cronos/Dietas/Personal — 403
  verificado con Bearer valido, y `loadPortal()` sin `.catch` por llamada
  deja la carcasa siempre en estado de error contra un despliegue demo
  limpio; (3) `app.js` llama a `GET /api` (solo existe en la API heredada
  fake) cuando el equivalente real de la carcasa es `GET /api/vec`.
- `evidencia`: curls documentados en la seccion de inconsistencias del
  contrato; codigos 401/403 reproducidos contra `cmd/vec-server` en fake.

### T08 — Codigo muerto de workspace en httpapi

- `origen`: hallazgo de lectura de T05.
- `estado`: nuevo. Encaja de forma natural dentro del carril T03
  (`httpapi` hacia `bootstrap`), como paso previo del traslado.
- `area_hexagonal`: adaptador.
- `accion`: eliminar `workspaceSnapshot`/`workspaceSnapshotWithCronos` de
  `internal/vec/adapters/httpapi/workspace.go` (compilan pero ningun camino
  HTTP los alcanza) o conectarlos si el workspace real de Ola 2 los necesita;
  decidirlo al ejecutar T03.
- `evidencia`: analisis de alcanzabilidad de T05 sobre `workspace.go`.
