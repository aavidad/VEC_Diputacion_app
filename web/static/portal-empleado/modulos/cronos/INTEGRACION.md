# Estado de integración de Cronos en el Portal del Empleado

El portal interno registra Cronos desde el catálogo y compone
`componerCronosInterno` (`portal-composicion-empleado.js`):

- «Jornada» cuelga en contenedores hijos el saldo propio
  (`vista-saldo-conectado.js`), el fichaje remoto (`vista-remoto.js`), los
  movimientos del día (`vista-movimientos-conectado.js`) y el calendario anual
  con ausencias y olvidos (`vista-movimientos-propios.js`). «Olvido de
  marcaje» abre el formulario de olvido del calendario.
- «Permisos» monta `vista-permisos-propios.js`.
- «Avisos de resolución» (`vista-avisos-propios.js`) y «Solicitudes por
  resolver» (`vista-bandeja-permisos.js`) usan `cliente-resolucion-http.js`.
  Son opcionales y van juntas: sin sus piezas no se ofrecen. En el servidor
  exigen `VEC_CRONOS_RESOLUCION_ENABLED=true`, los cuatro motivos de la
  configuración privada y el circuito publicado (`permiso_resolutor`,
  provisional hasta la duda 47).
- «Notificaciones a RRHH» (`vista-notificaciones-propias.js`) y
  «Notificaciones recibidas» (`vista-bandeja-notificaciones.js`) usan
  `cliente-notificaciones-http.js`. También son opcionales y van juntas. En el
  servidor exigen `VEC_CRONOS_NOTIFICACIONES_ENABLED=true` y sus cuatro
  motivos (`notificacion`, `notificaciones`, `bandeja_notificaciones`,
  `atencion_notificacion`). El documento adjunto no se sube: el navegador
  calcula su huella SHA-256 y solo viajan esa huella y la referencia de
  custodia que indica la persona. Tipos provisionales: duda 48.

Si falta cualquier vista o cliente, Cronos no se ofrece (falla cerrado). Cada
vista consulta su propia API: con una capacidad desactivada el servidor da
404 y esa vista muestra su estado sin afectar a las demás. La jornada genérica
(`vista.js`) y el recorrido de Permisos (`vista-recorridos.js`) quedan solo en
presentación. Cada vista nueva tiene su guía tras el botón «?» (ayudante de
trámites: resolver permisos, avisos, notificar a RRHH y atender).

## Circuito de resolución

Por regla general (Alberto, 25/09/2026) resuelve primero la jefatura y por
último RRHH, sea cual sea el circuito escrito en el catálogo. Solo va directo
a RRHH quien tenga una marca expresa y vigente de su unidad
(`permiso_circuito_directo`, publicada por RRHH, provisional). Sin jefatura
asignada y sin esa marca, la solicitud queda pendiente de asignación: RRHH la
ve en su bandeja con ese estado y nadie puede resolverla (409
`pendiente_asignacion`).

## Orden de instalación

Antes del binario y en este orden, sin reaplicar ninguna ya instalada:
AD3-53 → AD3-70 → AD3-57 → `cronos_v1` 000009 → AD3-58 → `cronos_v1` 000010.
La resolución (selector propio) exige ya 000010, porque su bandeja devuelve lo
pendiente de asignación; las notificaciones exigen AD3-58 y 000010. Los tipos
de notificación sintéticos se publican con
`datos_sinteticos/tipos_notificacion_sintetico_duda48.sql`.

## Pendiente

- La pestaña «Solicitudes por resolver» y «Notificaciones recibidas» se
  ofrecen a toda persona con Cronos; quien no tiene concesión ve la denegación
  del servidor. Ocultarlas exige la matriz de roles pendiente (dudas 29, 31,
  47 y 48).
- La vista de permisos propios sigue mostrando el circuito del catálogo
  («Administración» o «Jefatura y administración»), que ya no decide el
  recorrido; se ajustará con la respuesta de la duda 41.
- El marcaje desde el portal se limitará al circuito remoto que acredite
  teletrabajo vigente para esa persona y periodo en el servidor.
- Falta vincular y probar navegador → identidad y autorización → caso de uso →
  PostgreSQL → recibo recuperable, incluida la recuperación tras reiniciar
  aplicación y base. No se atribuye esta evidencia a las pruebas de contrato
  ni a los ensayos PostgreSQL con fachadas V3 de prueba.
- La obtención del empleado y de la jornada corresponde al servidor; el
  navegador no elegirá otra persona ni deducirá su vínculo de la cuenta.
- La emisión o descarga de recibos debe proceder del servicio documental
  autorizado. Este módulo no genera PDF local ni una ruta de cotejo simulada.

Los instantes se muestran en `Europe/Madrid` a partir de UTC y las fechas civiles
se mantienen como fechas. La vista escapa los datos recibidos y usa el catálogo
i18n compartido. No conserva operaciones ni credenciales en almacenamiento web.
