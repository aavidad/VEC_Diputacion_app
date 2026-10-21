# F1-27: sesión interna para consultar la aceptación de Mi Bolsa

Al guardar la propuesta del expediente 95034, la aceptación firme de Mi Bolsa
ya está confirmada. La preparación llega a consultar su anclaje en Bolsa y
falla antes de esa consulta: la sesión del perfil fijo de Bolsa exige que su
perfil figure en la frontera pública de la propuesta de Contratación.
La guarda privada permite esa consulta desde la propuesta, pero la sesión
todavía no conserva esa ligadura.

El segundo envío de J, con diagnóstico y autorización serializada, identifica
`actor_anclaje` y «autorizacion denegada». Se hizo sobre
`295a0a1461cf3720906245badb7d89b35dda8181`, con el mismo cuerpo y la misma
clave del primer envío. No hay un tercer intento.

## Permiso existente y alcance

| Elemento | Valor ya configurado |
| --- | --- |
| Perfil fijo | `incorporacion_b2_bolsa` |
| Acción | `bolsa.aceptacion_ct.anclaje.consultar` |
| Módulo y recurso | `bolsa`, `anclaje_aceptacion_ct` |
| Finalidad | `preparar_incorporacion_ct` |
| Ámbito | `unidad_ref` de la configuración del expediente |
| Audiencia | `vec_bolsa_llamamientos.aceptacion_ct.anclaje.v1` |

La propuesta clásica con aceptación anterior consulta el recibo de esa
operación de Bolsa. La rama nativa, incluida la aceptación por Mi Bolsa,
vuelve a consultar la emisión y su anclaje con el permiso de la tabla.
El defecto está en la selección de la sesión que ya tiene ese permiso.

La corrección enlaza esa sesión interna con el propietario, el perfil y la
captura de la misma petición. Reutiliza la guarda de ruta, sello y selector
del anclaje. La frontera pública conserva su descriptor y su perfil de
Contratación. La consulta de Bolsa conserva su autorización, asignación
publicada, cápsula de sesión y comprobaciones de recurso y unidad.
No se añaden concesiones ni SQL.

La prueba focal debe reproducir el rechazo con la sesión real y comprobar
el acceso ligado, además del rechazo de perfiles, rutas, sellos, selectores
y capturas cruzadas o de una asignación ausente. Protege frente al uso de
esa sesión desde una petición distinta o con otro perfil.

## Activación y límites

Consumidor: «Guardar la propuesta» del expediente de Contratación temporal,
después de confirmar la aceptación de Mi Bolsa. No hay configuración,
actos ni datos nuevos. Se recompila y actualiza la aplicación por la receta
de la copia; se conservan base, certificados, recibos y petición original.
La comprobación en J queda pendiente de integración y orden de dirección.

La denegación actual acaba presentada como 503. Dirección ha pedido tratar
su clasificación 403/404 por separado; este corte conserva esa respuesta.
El éxito de la propuesta y los pasos siguientes solo se acreditan con el
recorrido, sus recibos y la recuperación tras reiniciar.
