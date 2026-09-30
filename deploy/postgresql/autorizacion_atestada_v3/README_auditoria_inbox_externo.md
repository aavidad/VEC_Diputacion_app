# Auditoría técnica del inbox externo

AD3-119 añade una cadena exclusiva para el trabajador de avisos externos bajo
el propietario de AD3. No consume una decisión humana V3. Usuarios conserva
el inbox y su historia operacional; su propietario sólo puede ejecutar el
escritor nominal. No puede leer, alterar ni borrar esta auditoría. El login
del trabajador tampoco puede llamar al escritor directamente.

Instalar después de Usuarios 000010 y antes de Usuarios 000014, con la lista
`deploy/principal/lista_sql_codexb_auditoria_inbox_externo_20260930.txt`.
La migración rechaza una instalación anterior. No hay DOWN: la historia se
conserva. Este fichero no acredita instalación en la principal.

La función `registrar_auditoria_inbox_externo_v1` recibe acción, productor,
evento, recibo, correlación, huellas anterior y posterior, versión y resultado;
devuelve `auditoria_tecnica_externa:<32 dígitos hexadecimales>`. El actor procede
de `session_user`, que debe ser el login nominal
`vec_externo_avisos_usuarios` propio del trabajador externo, con una sola
membresía vigente en
`vec_usuarios_ejecutor_externo`, sin SET ni ADMIN. El perfil técnico es el
contrato fijo `usuarios.inbox_avisos_externos.v1`; el instante procede de la BD.
La transacción debe ser SERIALIZABLE. El trabajador usa su conexión propia
`VEC_EXTERNO_AVISOS_USUARIOS_DATABASE_URL`; el operador prepara el LOGIN y las
credenciales fuera de Git. La migración no crea un LOGIN ni una contraseña.

Las acciones son aceptar, reservar y confirmar. Los resultados admitidos
dependen de la acción: aceptación, reserva, respuesta del transporte, replay
o denegación. Una denegación sin recurso existente admite recibo nulo y
versión cero; no recibe texto libre ni datos de contacto. Cada reintento
genera una nueva entrada de auditoría y conserva el recibo original del inbox.

U14 debe llamar al escritor en la misma transacción del efecto y enlazar la
referencia devuelta. Si el escritor falla, se revierte el efecto. Para conservar
una denegación, el consumidor debe confirmar sólo su auditoría y devolver un
resultado de error; una excepción que aborte la transacción también revierte
esa auditoría. No se promete persistencia fuera de la transacción.

La prueba `pruebas_sql/auditoria_tecnica_inbox_externo.sql` usa identidades
sintéticas y ROLLBACK. Comprueba cadena y actor, vocabulario, denegaciones,
revocación, vigencia, sombreado mediante tablas temporales, ACL, historia
inmutable y reversión del efecto cuando
falla la auditoría. El ensayo completo y las dos revisiones independientes
del conjunto U14 corresponden a dirección. No acredita SMTP ni entrega al
destinatario.

El escritor fija `search_path=pg_catalog,pg_temp` y consulta los catálogos
de roles con nombres completos. Las tablas temporales del trabajador no
pueden sustituir la comprobación de su membresía ni de su vigencia.
