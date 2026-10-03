# Montaje privado de Usuarios y perfiles

`NuevoServidorConLecturas` monta `/admin/usuarios/` y la API común
`/api/admin/perfiles/v1/`. Recibe el resolvedor de sesiones, las lecturas
autorizadas y el auditor de la composición. Rechaza dependencias ausentes y
no admite una autoridad de actos en este modo. Los intentos de escritura
quedan cerrados y auditados por el adaptador HTTP.

`NuevoServidorConPerfiles` reutiliza el mismo servidor y el servicio
`ServicioAdministracionPerfiles`. Exige catálogo publicado y autoridad
durable de actos. La ausencia del proveedor de escritura impide ese montaje.

Ambos conservan la frontera ADMIN: TLS directo con certificado de cliente,
CA admitida, hoja verificada, CRL vigente, host y red configurados y fecha de
retirada. Se rechazan sesiones TLS reanudadas. Los activos se sirven desde
una lista fija, después de comprobar la sesión y la capacidad central de
lectura. El montaje no incorpora datos, certificados ni claves de personas.

`NuevoContextoConexionPerfiles` liga la antigüedad de autenticación a la
conexión TLS original. El resolvedor debe consultar esa conexión, las
asignaciones centrales y su vigencia; ningún campo recibido desde el cliente
concede autoridad. Los proveedores deben respetar la separación entre el
administrador de Aplicación y Sistemas y el único perfil activo.

Este corte prepara la composición inyectada. Falta conectar el proveedor
durable de lecturas y de actos y acreditar su instalación y el recorrido
desde el navegador. No arranca un proceso ADMIN ni concede permisos por sí
solo. La frontera temporal admite desarrollo y el entorno de pruebas; el
montaje de producción requiere su política corporativa.
