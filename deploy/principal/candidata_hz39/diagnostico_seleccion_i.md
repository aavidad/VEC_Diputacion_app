# F1-27: diagnóstico de selección en I

El único envío de selección de I devolvió 503 después de abrir la ventana
`solicitar_llamamiento`. La intención quedó indeterminada, sin recibo ni origen
de Bolsa persistido. No se ha repetido la selección.

El inventario de la fuente de I, `e39c95ab7605aaf8be3ba74e9077603574cdf422`,
no permite nombrar la causa. Los archivos de preparación, reanudación y puente
no crean errores con texto dinámico sin envolver: propagan errores de sus
dependencias. La aplicación conserva el error devuelto por
`SolicitarLlamamiento` al marcar la intención indeterminada. El tipo registrado,
`*errors.errorString`, también corresponde a los errores fijos de Bolsa que el
clasificador de Contratación aún no distingue. No hay cambios Go en esos módulos
ni en composición entre `483931b0c` y la fuente de I.

Este corte añade un registro en el puente nativo, antes de devolver el mismo
error. Reutiliza `causaFalloBolsaRRHHDesarrollo` y reconoce una lista cerrada de
errores fijos de origen, emisión, contexto y correo. Solo escribe el texto del
error fijo reconocido; nunca el mensaje de la causa ni el del envoltorio. Las
causas desconocidas siguen mostrando su tipo y PostgreSQL su código seguro.
No añade datos del expediente, la persona ni la conexión. La prueba focal
comprueba causas directas, envueltas, desconocidas y SQL con contenido privado.

## Activación y comprobación

No hay SQL, configuración, actos ni datos nuevos. Se compila y arranca por la
misma receta y configuración existentes. El registro se usa cuando está activo
el puente de llamamiento de Bolsa constituida; sin él, la ruta previa conserva
su comportamiento. No añade requisitos al arranque.

I conserva su fuente y binario originales. Aplicar este cambio a una copia
requiere revisión e integración de dirección, anotar el nuevo binario y su
huella, y reiniciar la aplicación de forma vigilada. La recuperación debe usar
la petición y clave originales, por la vía del producto y con autorización de
dirección. Este corte no acredita esa recuperación ni resuelve el 503.

Los pasos 10 a 21 y el reinicio final siguen pendientes. La evidencia privada
permanece fuera de Git. La hoja de I conserva el resultado observado.
