# Auditoría nominal de la frontera ADMIN

`NuevoAuditorFronteraNominal` recibe el registrador común de intentos AD169 y
una configuración privada con proceso, canal, motivos de catálogo, plazo de
auditoría y destinos de las nueve acciones cerradas del handler. Copia el mapa
al construirse. `buscar_personas` usa una referencia `conjunto_admin:` fija de
esa configuración. `consultar_persona` registra la referencia `per_` solicitada
si supera la validación de forma; si está malformada, usa una referencia opaca
derivada de la correlación interna. Los demás destinos tienen referencias fijas.

La identidad procede del actor y del par V2 copiados de `SesionConfiable`. El
auditor coteja ambos y exige cuenta privilegiada en la superficie ADMIN. Los
campos de actor y perfil en `DenegacionADMIN` son compatibles con el contrato
anterior, pero no intervienen en la orden común. Tampoco se conservan cabeceras,
certificados, rutas, consultas, cuerpos ni errores privados.
La correlación es la referencia opaca emitida por la sesión ADMIN; es distinta
de la correlación técnica de incidencias del middleware.
La composición deberá unificar ambas cuando el proveedor de sesión pueda
consumir la correlación interna de la petición. Este corte no cambia ese
proveedor ni usa la referencia como permiso.

Los códigos de rechazo de entrada se registran como `denegado` y
`servicio_no_disponible` y `respuesta_incompatible` como `error`. La orden se envía con un plazo propio
después de retirar la cancelación de la petición. Si el COMMIT del registrador
es ambiguo, se reintenta la misma orden; la respuesta HTTP solo sigue después
de recibir y validar el acuse. Sin identidad V2 acreditada se devuelve
indisponibilidad, sin crear un actor ni una familia de auditoría alternativa.

Esta pieza no usa `registrar_denegacion_frontera_admin_v1`, una función SQL que
no está instalada. Tampoco cierra la auditoría de todas las rutas ADMIN: el
montaje y las fronteras técnicas anteriores a la sesión requieren su propia
autoridad y comprobación. La instalación PostgreSQL y el recorrido real siguen
pendientes del director.
