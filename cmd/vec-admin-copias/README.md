# API de copias en ADMIN

Este corte prepara un manejador inyectable. No añade otro servidor, perfil ni sesión.
El punto de composición es `administracion.NuevoHandlerCopias`. Debe montarse después
de la comprobación vigente de CA, CRL, red y sesión del listener ADMIN existente.

La sesión procede del resolvedor ADMIN. Cada operación consulta de nuevo la autoridad
central con actor, perfil, acción y recurso exactos. Los adaptadores de negocio deben
revalidar y consumir esa autoridad junto al efecto, auditoría y recibo. Un callback
que devuelve `nil` no demuestra consumo V3. Las capacidades solo orientan la vista.

La raíz de la API es `/api/admin/copias/v1`:

| Método y sufijo | Entrada o salida |
| --- | --- |
| GET vacío | `cursor` y `limite` opcionales; página de copias y versión actual de la fuente. |
| GET `/capacidades` | Capacidades nominales; no contiene identidad ni concesiones. |
| GET `/{copia_ref}` | Estado, compatibilidad y metadatos del conjunto. |
| GET `/opciones-restauracion` | Destinos, motivos y ventanas de configuración, por referencia y clave i18n. |
| GET `/calendario` o `/retencion` | `{configuracion:{version,politica}}`. |
| GET `/propuestas` | Propuestas y datos necesarios para revisarlas. |
| POST `/lanzamientos` | `operacion_ref`, `version_esperada`, `tipo: "completa"`; devuelve recibo. |
| POST `/calendario` o `/retencion` | `operacion_ref`, `version_esperada` y política completa; devuelve recibo. |
| POST `/propuestas` | Referencias de conjunto, destino, motivo y ventana, operación y fechas UTC; devuelve propuesta. |
| POST `/propuestas/{ref}/revision` o `/ejecucion` | Operación, destino, huella de propuesta y versión esperada. |

Los modelos completos están en `internal/modules/administracion/ports/httpcopias`.
Calendario y retención comparten una política atómica. El adaptador de configuración
debe exigir las concesiones de todos los bloques que cambien; el permiso de un
bloque no permite modificar el otro. La versión de lanzamiento procede de su fuente
real y el adaptador la contrasta antes de reservar. No se deduce del número de filas
ni se reemplaza por una constante.

Las propuestas muestran la preimagen y las políticas selladas. La copia previa se
exige antes de sustituir; no se presenta como ya hecha al proponer. Los intervalos de
pérdida solo aparecen cuando una fuente los acredita. El servicio CS10 comprueba el
cercado, pero no ejecuta sustituciones. Sin ejecutor real, `/ejecucion` devuelve 503.

Solo se acepta JSON de hasta 16 KiB, con campos requeridos explícitos y sin claves
duplicadas, campos desconocidos, valores nulos ni documentos adicionales. Los POST
exigen origen y contexto de navegación del mismo sitio. Se rechazan cookies,
Authorization y cabeceras libres de identidad. Las respuestas no generan cookies,
redirecciones ni CORS y llevan `no-store`. Los fallos contienen código y clave i18n;
no incluyen certificados, secretos, rutas o causas internas.

La comprobación focal usa `go test -race` y `go vet` en los paquetes afectados.
Una prueba HTTP con mTLS local verifica denegación sin sesión o permiso, rechazo de
entradas falsificadas, autorización nominal y retirada de permiso en una conexión.
Son fixtures sintéticos: no acreditan montaje, asignaciones administrativas,
consumo V3, persistencia de una copia ni un recorrido de navegador.
