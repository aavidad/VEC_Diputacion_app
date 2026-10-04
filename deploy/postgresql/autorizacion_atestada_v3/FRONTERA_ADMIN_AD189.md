# Auditoría de la frontera ADMIN anterior a V2 — AD189

AD189 prepara una familia técnica propia de la auditoría común. Antes de
resolver un contexto V2 no atribuye la petición a una Persona o perfil humano.
El operador registrado es el LOGIN real de PostgreSQL, con un grupo exclusivo
acotado al nuevo puerto. Proceso y canal proceden de configuración privada.

Contrato de once cadenas: `tipo_registro`, `evento_ref`, `operador_login`,
`accion`, `recurso_ref`, `resultado`, `codigo_ref`, `proceso`, `canal`,
`finalidad_ref`, `correlacion_ref`. No incorpora un hash de solicitud ficticia.
La acción es `controlar_frontera_admin_v1`, módulo `administracion`, finalidad
`control_frontera_admin` y canal `administracion_privilegiada`.

El recurso es `solicitud_admin:<32hex>`, los primeros 16 bytes de SHA256 sobre
UTF8 `vec.admin.frontera.solicitud.v1\n` seguido de la correlación privada de la
petición. No conserva certificado, cabeceras, ruta, consulta, cuerpo ni IP.
El evento tiene referencia propia y los resultados/códigos forman un catálogo
cerrado de datos SQL. El acuse real contiene referencia, secuencia, huella,
correlación e instante de la cadena común; prefijo `aud_v3_fat_`.

El registrador confirma su transacción antes de permitir la respuesta. Un
COMMIT incierto vuelve a presentar la misma orden bajo un plazo privado y
conservando los valores del contexto, sin fabricar otro evento. Si no puede
confirmar un acuse válido, la frontera responde indisponibilidad.

El auditor compuesto utiliza la autoridad nominal existente cuando tiene V2
original usable. Sólo usa esta familia antes de V2; nunca cambia a la familia
técnica para ocultar un error de AD169 ni rellena un actor desde un DTO.

Reserva AD189 anterior al código y base causal AD188
`8d213c0102258f98f2220b68bbf5a4be1223f888`. CHECK post188 medido por el escritor
único del clon: `auditoria_tipo_disjunto_v4`, SHA256
`92f884b9e70c65f720a1448c00269949f035d14d6dd466ebe0678b2f27cfc2b4`, UTF8 de
`pg_get_constraintdef(false)` sin salto final. AD188 no cambió el núcleo.

Estado: contrato y puerto preparados. SQL sólo será candidata, con dos GO
independientes y ensayo autorizado antes de UP. No se ha ejecutado Go,
PostgreSQL, instalado SQL ni modificado la principal. El montaje HTTP y la
unión del verificador compartido se harán después de ceder estos archivos.
