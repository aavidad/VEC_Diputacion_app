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
  exigen `VEC_CRONOS_RESOLUCION_ENABLED=true`, AD3-57 y `cronos_v1` 000009
  instaladas antes del binario, los cuatro motivos de la configuración privada
  y el circuito publicado (`permiso_resolutor`, provisional hasta la duda 47).

Si falta cualquier vista o cliente, Cronos no se ofrece (falla cerrado). Cada
vista consulta su propia API: con una capacidad desactivada el servidor da
404 y esa vista muestra su estado sin afectar a las demás. La jornada genérica
(`vista.js`) y el recorrido de Permisos (`vista-recorridos.js`) quedan solo en
presentación.

## Pendiente

- Notificaciones de la persona a RRHH (tipo, fecha, texto de hasta 512
  caracteres y adjunto por referencia y huella) y su bandeja en RRHH: siguiente
  corte de C9. Justificación de permisos (C8) y resolución de olvidos: aparte.
- La pestaña «Solicitudes por resolver» se ofrece a toda persona con Cronos;
  quien no tiene concesión ve la denegación del servidor. Ocultarla exige la
  matriz de roles pendiente (dudas 29, 31 y 47).
- El marcaje desde el portal se limitará al circuito remoto que acredite
  teletrabajo vigente para esa persona y periodo en el servidor.
- Falta vincular y probar navegador → identidad y autorización → caso de uso →
  PostgreSQL → recibo recuperable, incluida la recuperación tras reiniciar
  aplicación y base. No se atribuye esta evidencia a las pruebas de contrato.
- La obtención del empleado y de la jornada corresponde al servidor; el
  navegador no elegirá otra persona ni deducirá su vínculo de la cuenta.
- La emisión o descarga de recibos debe proceder del servicio documental
  autorizado. Este módulo no genera PDF local ni una ruta de cotejo simulada.

Los instantes se muestran en `Europe/Madrid` a partir de UTC y las fechas civiles
se mantienen como fechas. La vista escapa los datos recibidos y usa el catálogo
i18n compartido. No conserva operaciones ni credenciales en almacenamiento web.
