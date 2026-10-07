# Consulta de usuarios desde ADMIN

`NuevaFuenteLecturasUsuariosMetadatos` adapta el puerto nominal de usuarios al
contrato HTTP existente. Conserva filtros, cursor y todos los perfiles de cada
Persona. La versión de perfil es la de CA; no se sustituye por la versión de
asignación AUT. Devuelve datos únicamente si el puerto confirma su consulta.
Un resultado parcial acompañado de error no sale por HTTP.

Una versión de denominación ausente se muestra como nombre sin registrar. Una
versión presente queda como nombre no consultado: este consumidor no usa el
certificado, cargo, perfil o un identificador como nombre. La posterior lectura
de nombre debe pasar por CA32 y su lector autorizado. Historial y actos no
consultados se omiten; no se presentan como listas vacías.

El montaje activa `SoloUsuariosMetadatos` desde dependencias privadas. El
handler reutiliza las rutas de lista y ficha, y rechaza las consultas auxiliares
y búsqueda por nombre en la frontera auditada, antes de llamar a su fuente.
Las escrituras siguen cerradas. Este ajuste de montaje no concede permisos:
cada consulta pasa por el puerto nominal AUT43/AD185 y el perfil APP real.

La ficha ausente o ajena devuelve el mismo resultado después del consumo
nominal de la fuente. Los errores privados se normalizan. La capa de
presentación comprueba la proyección y su referencia antes de serializar.

Las pruebas con dobles verifican el transporte del contrato, omisión de campos,
copia de versión, rechazo de rutas y errores sin datos parciales. No acreditan
mTLS, PostgreSQL, V3 ni auditoría durable. El montaje operativo requiere la
fuente PostgreSQL y el emisor real, sus LOGIN mínimos y configuración de
sesiones/contextos; no se incluyen configuraciones privadas o claves en Git.
