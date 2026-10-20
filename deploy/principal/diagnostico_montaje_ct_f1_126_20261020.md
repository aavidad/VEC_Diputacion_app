# F1-126: localizar un fallo al montar Contratación temporal

Este corte añade al registro técnico de arranque una etiqueta fija del
constructor que falla. Conserva la causa del error para las comprobaciones
internas. No cambia los permisos, los actos, los documentos ni las respuestas
HTTP.

## Activación

No requiere SQL, datos, actos de administración ni variables nuevas. Conservar
el ejecutable anterior y la configuración efectiva de la copia propia antes
de sustituir el ejecutable por el construido desde la cabeza revisada.

Reiniciar una vez bajo vigilancia. Comprobar la salud y el portal y después
el cuadro de Contratación temporal con su certificado. Si falla el montaje,
consultar el evento técnico `contratacion_temporal_no_disponible`: su atributo
`constructor` identifica el punto de fallo mediante una etiqueta
`ct.montaje.<constructor>`. La etiqueta procede del código; no contiene el
mensaje de la causa, conexiones, rutas ni datos de las personas.

Si no hay error de montaje, no se espera ese evento. Comprobar también el
arranque con la configuración de firma desactivada. La ausencia de firma no
debe impedir abrir el portal ni las funciones disponibles de Contratación
temporal.

## Recuperación y evidencia

Ante una regresión, restaurar el ejecutable y la configuración conservados,
reiniciar y comprobar de nuevo salud, portal y cuadro. No repetir los actos
de administración ni publicar otra versión del plan para obtener un mensaje
de diagnóstico.

Guardar fuera de Git la revisión, la huella del ejecutable, los estados HTTP
y la etiqueta del constructor. Un diagnóstico temporal sobre una fuente
anterior debe declarar esa fuente y el único cambio aplicado; no acredita
por sí solo la aceptación de una candidata posterior.

La firma y las descargas denegada y autorizada de F1-126 siguen requiriendo
su comprobación real desde la ficha y su conservación tras reinicio.
