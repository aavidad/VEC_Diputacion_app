# F1-27: diagnóstico del fallo al guardar la propuesta

La copia J completó la aceptación en Mi Bolsa y su confirmación en
Contratación temporal. El primer envío de propuesta devolvió 503 el
21 de octubre de 2026 a las 11:12:33, hora de Madrid. Una lectura posterior
de la base encontró cero propuestas con su clave; el expediente seguía
en la versión 6. El registro no permitía distinguir qué etapa falló.

Este corte añade registro antes de normalizar los errores del antecedente,
la lectura de la emisión, la identidad para consultar el anclaje y la
consulta del anclaje. También distingue la preimagen, las dependencias
ausentes y la confirmación de la propuesta procedente del portal.
La etapa y la causa salen de listas cerradas. Un error desconocido queda
como `no_clasificado`: no se registra su mensaje ni el de su envoltorio.
No se incorporan cuerpos de petición, certificados, identidades ni recibos.

Se reutilizan los puertos, las consultas y el registro estructurado
existentes, siguiendo el diagnóstico de selección. Cada rama conserva su
resultado y su error. No se cambian autorización, transacciones, SQL,
catálogos ni pantallas.

## Activación y comprobación

La pantalla consumidora es la propuesta del expediente de Contratación
temporal, después de confirmar la aceptación del portal. No hay nueva
configuración, actos de administración ni datos. El registro funciona con
la configuración existente y permanece inactivo cuando no se usa esa ruta.

Tras integrar, se recompila y actualiza solo la aplicación por la receta
de la copia. Se conservan la base, los certificados, los recibos y la clave
de la petición original. La repetición del envío queda pendiente de la
decisión de dirección; este corte no demuestra que la propuesta funcione.

La prueba del clasificador comprueba errores reconocidos, envoltorios con
contenido privado, mensajes SQL y una etapa desconocida. Las pruebas
existentes del antecedente portal comprueban el rechazo de respuestas,
participaciones o fechas cruzadas y de un anclaje no disponible.
La ejecución focal, la revisión independiente y las puertas de calidad
se comunicarán sobre la cabeza final antes de pedir la integración.
