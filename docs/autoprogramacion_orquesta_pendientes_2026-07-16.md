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
- Prioridad de negocio (responsable, 2026-07-17): **Bolsa primero**. El
  objetivo inmediato es una Bolsa operable de punta a punta: levantar el
  NO-GO de la migracion V2, conectores productivos de la saga de firma,
  porte del ranking/desempate del heredado (hallazgo 4 de la brecha) y su
  API privada con UI conectada. Los demas frentes no se detienen, pero
  ceden el paso ante conflicto de recursos o de carril.
- Orden de ataque recomendado por direccion (17/07 tarde): primero T11
  (conformidad DEC-053, barato ahora y caro despues), despues T07 y T12
  (coherencia y durabilidad de Bolsa), despues T02 y el resto de T03.
  S-03 (identidad real por certificado/Kerberos) queda aparcada
  deliberadamente: depende de Sistemas y no hay produccion hasta su visto
  bueno; nada debe disenarse de forma que la estorbe.

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
- `estado`: en curso. Primera rebanada terminada el 17/07/2026: la raiz de
  composicion construye e inyecta el servicio de catalogo Personal y
  `httpapi` ya no importa ni decide entre sus adaptadores `file`/`memory`;
  tambien se retiro el servicio Cronos sintetico que solo alimentaba codigo
  muerto. Queda extraer los handlers/contratos de ruta para eliminar los
  imports funcionales restantes de `internal/modules/*`.
- `area_hexagonal`: adaptador hacia composicion.
- `accion`: programar el traslado del montaje de modulos de
  `internal/vec/adapters/httpapi` (`workspace.go`, `cronos.go`,
  `personal_rpt.go`) a `internal/app/bootstrap`; `httpapi` recibe handlers o
  interfaces ya compuestos y pierde todos los imports de
  `internal/modules/*`.
- `evidencia`: `nuevoServicioCatalogoPersonal` vive en
  `internal/app/bootstrap`; `HandlerOptions.PersonalCatalog` recibe el caso
  de uso compuesto. `go list` sigue mostrando imports funcionales de modulo,
  por lo que T03 no se declara cerrado todavia.

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
- `estado`: completado el 17/07/2026. Se conserva la ruta fail-closed y se
  elimino todo el grafo sintetico inalcanzable.
- `area_hexagonal`: adaptador.
- `accion`: cerrada por eliminacion; el futuro workspace interno se
  construira como caso de uso acotado por sujeto, finalidad y campos, no
  reactivando el agregado demo.
- `evidencia`: `workspace.go` contiene solo `handleWorkspace`; no quedan
  simbolos `workspaceSnapshot*`, semillas Cronos ni datos RPT/nomina/dietas.

### T09 — Regresion de tamano en la ola de autorizacion del 17/07

- `origen`: puerta `scripts/comprobar_tamano_ficheros.sh` ejecutada por
  direccion antes del push de la ola COSE/VEC-AD-2.
- `estado`: corregido por direccion el 17/07/2026 (`nucleo: dividir
  autorizacion para cumplir el tope de tamano`): `domain/autorizacion.go`
  quedo en 434 lineas mas `autorizacion_politicas_decision.go` (600), y
  `memory/autorizacion_test.go` en 667 mas `autorizacion_aislamiento_test.go`
  (540). Linea base ratificada a la baja.
- `area_hexagonal`: nucleo y adaptador de memoria.
- `accion` (preventiva, para el agente): ejecutar la puerta de tamanos
  antes de cada commit, no solo `go test`. Dos ficheros crecieron por
  encima del tope duro (1024 y 1190 lineas frente a bases de 933 y 860)
  a lo largo de 24 commits sin que ninguna puerta local lo detectara.
- `evidencia`: salida de la puerta el 17/07 a las 11:14; el tope duro es
  800 (DEC-051) y la linea base solo puede menguar.

### T10 — Estilo de mensajes de commit desviado

- `origen`: revision de direccion del 17/07 por la tarde.
- `estado`: nuevo (correccion de conducta, no de codigo; el historico no se reescribe).
- `area_hexagonal`: proceso.
- `accion`: volver al estilo del repositorio `area: verbo en infinitivo`.
  Los prefijos convencionales estan prohibidos por la directriz 5 de la
  auditoria, incluida su variante con parentesis: `fix(web):`, `test(web):`,
  `test(seguridad):`.
- `evidencia`: commits `8e95568`, `1448bc4`, `65d4bee`, `28721f2` del
  17/07/2026 usan prefijos `fix()`/`test()`.

### T11 — Retirar cookies del portal interno conforme a DEC-053

- `origen`: DEC-053 del registro de decisiones ("la web consume la API como
  un cliente mas"; sin cookies de sesion, para que un cliente de escritorio
  sea equivalente byte a byte). El cliente nuevo del portal la contradice.
- `estado`: completado el 17/07/2026 en `1a32b1c`. La regresion queda
  protegida por pruebas del cliente y del contrato de superficies.
- `area_hexagonal`: adaptador (frontend) y composicion.
- `accion`: el cliente del portal interno no debe usar
  `credentials: "same-origin"` ni asumir sesion de cookie: la autenticacion
  viaja en cabecera `Authorization` (token de sesion emitido por el
  adaptador de identidad; para tecnicos, derivado de mTLS+Kerberos segun
  DEC-053), y el cliente web usa `credentials: "omit"`. Bajo ese contrato web
  no hay credencial ambiental que el navegador adjunte entre sitios. La app de
  administracion sera de escritorio y no tiene contexto web entre sitios: la
  API no puede exigir nada que un cliente no navegador no pueda enviar. Si un
  futuro navegador negociase Kerberos/SPNEGO o mTLS automaticamente, habria que
  reevaluar CSRF y origen antes de habilitarlo.
- `evidencia`: `web/static/portal-empleado/portal.js`,
  `web/static/bolsa/bolsa.js` y `web/static/app.js` usan
  `credentials: "omit"`; `web/static/app.seguridad.test.mjs` impide
  reintroducir cookies, credenciales ambientales o identidad persistida en
  almacenamiento web. El contrato de superficies ya no contiene
  configuracion de cookies y el proxy local elimina cualquier cabecera
  `Cookie` entrante.

### T12 — Durabilidad probatoria productiva de Bolsa

- `origen`: prioridad de negocio "Bolsa primero" y revision de direccion del
  17/07: la semantica durable (saga de firma, auditoria encadenada,
  expediente probatorio) esta probada solo en memoria.
- `estado`: nuevo.
- `area_hexagonal`: adaptadores de persistencia y composicion.
- `accion`: llevar a PostgreSQL productivo, por este orden, la auditoria
  encadenada, el registro de decisiones de autorizacion ya migrado y los
  puntos de control de la saga de firma; incluir copias de seguridad
  ensayadas y prueba de recuperacion tras reinicio completo. Sin esto un
  incidente no pierde seguridad pero si trazabilidad.
- `evidencia`: adaptadores en memoria en `internal/modules/bolsa` y
  `internal/vec`; el flujo durable documenta sus conectores pendientes.
