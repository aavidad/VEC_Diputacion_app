# Estado de integración de Cronos en el Portal del Empleado

El portal interno registra Cronos y monta una vista de Jornada sin datos cuando
no recibe una proyección propia autorizada. La vista conserva estados de carga,
vacío, denegación y error; no ofrece controles de fichaje genérico. El
contrato de Jornada valida la referencia del actor de la sesión compartida, las
capacidades declaradas y el envelope `vec.cronos.area-personal.v1`. Una capacidad
en el navegador solo gobierna la presentación: la autorización del efecto
pertenece al servidor.

El recorrido de Permisos, sus hojas de catálogo, correcciones y notificaciones
son superficies separadas. El coordinador las monta sin inventar solicitudes,
saldos ni responsables. Su comportamiento y textos están en
`vista-recorridos.js` e `i18n.js`.

## Pendiente de composición

- La API de Jornada no está montada en el coordinador. La vista no recibe aún el
  envelope validado de un servicio interno ni registra fichajes.
- Existen clientes HTTP y vistas de saldo propio y marcaje remoto, pero aún no
  están montados en el coordinador ni acreditados con un recorrido completo.
- Movimientos propios (calendario anual por tipo de día, ausencias y olvidos
  con su solicitud de corrección) y permisos propios (listado anual, solicitud
  y pendientes de conceder y de justificar) tienen cliente
  `cliente-solicitudes-http.js` y vistas `vista-movimientos-propios.js` y
  `vista-permisos-propios.js`, igualmente sin montar. La concesión por jefatura
  o administración y los mensajes de resolución pertenecen a otro corte.
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
