# Estado de integración de Cronos en el Portal del Empleado

El portal interno registra Cronos desde el catálogo y compone
`componerCronosInterno` (`portal-composicion-empleado.js`):

- «Jornada» cuelga en contenedores hijos el saldo propio
  (`vista-saldo-conectado.js`), el fichaje remoto (`vista-remoto.js`), los
  movimientos del día (`vista-movimientos-conectado.js`) y el calendario anual
  con ausencias y olvidos (`vista-movimientos-propios.js`). «Olvido de
  marcaje» abre el formulario de olvido del calendario.
- «Permisos» monta `vista-permisos-propios.js`.

Si falta cualquier vista o cliente, Cronos no se ofrece (falla cerrado). Cada
vista consulta su propia API: con una capacidad desactivada el servidor da
404 y esa vista muestra su estado sin afectar a las demás. La jornada genérica
(`vista.js`) y el recorrido de Permisos (`vista-recorridos.js`) quedan solo en
presentación.

## Pendiente

- La concesión por jefatura o administración y los mensajes de resolución
  pertenecen a otro corte.
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
