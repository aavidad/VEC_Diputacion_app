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
- `estado`: en_curso (reservado por direccion, subagentes en worktree aislado, 2026-07-16).
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
- `estado`: en_curso (reservado por direccion, subagentes en worktree aislado, 2026-07-16).
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
- `estado`: en_curso (reservado por direccion, subagentes en worktree aislado, 2026-07-16).
- `accion`: documentar el contrato de endpoints por modulo (ruta, metodo,
  envelope, errores, version) y programar los endpoints finos de la Ola 2
  contra ese contrato; la web consume la API como un cliente mas y ningun
  cliente incorpora logica de negocio.
- `evidencia`: el frontend actual solo hace `fetch` real a la API heredada y
  `/api/demo`; las pantallas privadas operan con datos sinteticos locales.

### T06 — Nivel de madurez por modulo en el contrato (H-05)

- `origen`: auditoria H-05.
- `estado`: en_curso (reservado por direccion, subagentes en worktree aislado, 2026-07-16).
- `area_hexagonal`: docs.
- `accion`: documentar en el
  [contrato de modulos](portal_vec/contrato_modulos_vec.md) el nivel que
  cumple cada modulo (completo, parcial, solo manifiesto) para que el shell
  no asuma capacidades inexistentes.
- `evidencia`: Bolsa tiene las cuatro capas; Cronos y Personal parciales;
  Dietas y Administracion solo manifiesto (arbol de `internal/modules/`).
