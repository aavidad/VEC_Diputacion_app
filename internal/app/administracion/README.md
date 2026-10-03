# Montaje privado de Usuarios y perfiles

El único compositor `ComponerServidorPerfiles` reutiliza la frontera, cuentas,
sesión, contexto y confianza V3 de ADMIN. Construye el modo de lectura sin
abrir una autoridad de actos ni exigir su pool. Sirve `/admin/usuarios/` y el
alias `/administracion-perfiles/` con la misma hoja; no añade otra pantalla al
portal público.

La lista fija de activos incluye Usuarios, propuestas, selector, catálogos,
favicon y estilos comunes. Excluye pruebas y datos de ejemplo. Para servirla,
la observación coteja mTLS, CA, CRL, host, audiencia, red y vigencia. La fuente
previa al perfil debe devolver una lista propia válida y no vacía junto al
recibo de auditoría común confirmado en la misma transacción. No se resuelve
un perfil activo ni se selecciona uno al solicitar un archivo.

`FuenteSeleccionAuditadaADMIN` es el punto de conexión pendiente para esa
fuente y la selección con CAS y auditoría común. El lector antiguo de IS12
no se conecta como si acreditase ese recibo. Una referencia aislada tampoco
acredita instalación: la implementación y su ensayo corresponden a identidad
y auditoría. Sin esa fuente, activos y selector responden 503.

El selector HTTP reutilizado conserva sus rutas `/api/admin/seleccion-perfil/v1/`.
La elección no concede acceso ni suma perfiles. La vista vacía y cancela el
contexto anterior antes de consultar el nuevo. Las consultas de negocio siguen
exigiendo sus permisos centrales en `/api/admin/perfiles/v1/`. Si falta la
fuente durable, `lecturasNoDisponibles` devuelve error de servicio, nunca datos,
capacidades ni recibos. Los POST de perfiles permanecen cerrados y auditados.

El proceso admite únicamente la política temporal configurada de desarrollo
y pruebas. Conserva TLS directo, canal original y prohibición de sesiones
reanudadas. No arranca una aplicación ni aplica SQL o bootstrap por preparar
este ensamblaje. Faltan las fuentes auditadas, su instalación, configuración
privada y el recorrido con navegador y base reales.

Las dependencias que impiden activar la consulta son concretas:

- `FuenteSeleccionAuditadaADMIN.ListarPropiosAuditadosADMIN` y
  `SeleccionarPerfilAuditadoADMIN`: no hay proveedor admitido que una la
  operación previa al perfil con auditoría común y su recibo.
- `postgres.NuevaFuenteLecturas`: conserva `NoDisponible` hasta disponer del
  contrato técnico L, acuse común y fachadas compatibles. Las lecturas
  heredadas `consultar_capacidades_admin_v1`, `buscar_personas_admin_v1`,
  `consultar_persona_admin_v1`, `listar_roles_admin_v1`,
  `listar_propuestas_admin_v1`, `consultar_propuesta_admin_v1` y
  `consultar_recibo_admin_v1` no acreditan el protocolo sucesor por existir.
  Faltan búsqueda inicial autorizada, filtros antes de paginar y proyección
  nominal admitida; no se eliminan filtros para usar una fachada antigua.
- `registrar_denegacion_frontera_admin_v1`: el adaptador E está preparado,
  pero aquí no se acredita instalación ni auditoría común. Sin respuesta
  válida conserva el error, sin simular un acuse.
- `postgres.NuevaFuenteCapacidades`: prepara exclusivamente
  `consultar_capacidades_admin_v1`, con la correlación de la frontera y el
  acuse común permitido de AD168. Devuelve `acciones: []`; las otras seis
  consultas y los actos siguen cerrados. Su constructor conserva
  `NoDisponible` hasta que L entregue el registrador común durable de
  denegaciones y errores y se acredite el LOGIN del rol técnico
  `vec_admin_perfiles_lector`. Se inyectará por
  `DependenciasComposicionPerfiles.Lecturas`; no usa el pool de actos ni
  abre el constructor general de lecturas. Las pruebas de contrato con
  dobles no acreditan instalación ni auditoría real.
- Configuración privada, fuentes causales y bootstrap por el canal admitido:
  la CLI actual prepara o coteja; `-aplicar` permanece indisponible.

Las fachadas de actos y lotes no se conectan en este modo. Ninguna prueba de
ensamblaje con dobles acredita cuentas, perfiles activos, instalación o un
recorrido con PostgreSQL.
