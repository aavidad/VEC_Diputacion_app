# Recorrido de I — F1-27

En curso, 21 de octubre de 2026. Fuente de ejecución:
`e39c95ab7605aaf8be3ba74e9077603574cdf422`, main con F1-148 y CT259.
Dirección indicó a las 08:09 crear otra copia para comprobar la segunda firma
de Jefatura. El expediente 2026/95032 de H no se reintenta.

La creación usa la [receta completa](../recorrido_copia/LEEME.md), con una
fuente limpia y subida, sin añadir ramas pendientes. Antes de arrancarla se
paró H con su gestor para conservar una sola copia viva. Sus datos, la
solicitud rechazada y el PDF firmado permanecen guardados. G se conservó
parada hasta comprobar las dos firmas de I.

| Tramo | Estado |
| --- | --- |
| Creación completa y carga nativa de Bolsa | Completada en 294,47 segundos; 145 migraciones |
| Petición, ratificación y alta (1–3) | 200, 200 y 201; expediente 2026/95033 |
| Análisis, vía, asignación e informe (4–7) | Actuaciones 201 hasta versión 5 |
| Dos firmas del informe y fiscalización (7b–8) | Dos firmas 201 y fiscalización favorable 201; versión 6 |
| Llamamiento, contactos y aceptación (9–11) | Desde las 09:00 de Madrid |
| Propuesta, resolución, documentos y tres firmas (12–15) | Pendiente |
| Incorporación, Personal, GINPIX, cese, vuelta a Bolsa y cierre (17–21) | Pendiente |
| Reinicio y recuperación de lecturas y recibos | Pendiente |

Las dos firmas del informe se confirmaron desde Chrome con GrxFirma y los
certificados propios de técnico y Jefatura. Hubo un envío de registro por
firmante; ambos devolvieron 201, verificación técnica válida y
`firma_eficaz=false`. La cofirma de Jefatura completa el paso 2 de 2 del
informe. Ese resultado acredita el recorrido técnico de I; no recupera el
instante interno exacto del envío rechazado en H ni sustituye Firmadoc.

El PDF firmado se descargó desde la ficha: 49.315 bytes, SHA-256
`4ac25b3327f66738e0901d448a43522dff0e140368b78378db9b611739f9ef1e`.
Su huella coincide con la devuelta por la segunda firma. Intervención
registró después el resultado favorable, con recibo y versión resultante 6.
No se ha seleccionado ni contactado a la persona candidata en I.
La prueba SQL y las revisiones de F1-148 cubren además las precisiones de
fecha histórica que provocaban el defecto de contrato.
La franja de llamadas de 09:00 a 14:00 se conserva. I tendrá referencias,
actos y recibos propios; no trasladará los de H ni de otras copias.
Los materiales privados permanecen fuera de Git. Esta hoja no cierra el
recorrido ni las tareas F1-27 y F1-28.

## Lecturas al retomar

La ficha recupera la fiscalización favorable y el informe con sus dos firmas.
La sesión nominal de técnico muestra el PDF firmado; a 390 px no hay
desbordamiento horizontal (375 px de contenido). Las capturas permanecen
en el material privado del ensayo. No se ha repetido ninguna actuación.

Al abrir la ficha con la sesión de gestión, la consulta de comunicaciones
devolvió 503. El registro sitúa el rechazo en la guarda de versión de
`autoridad_postgresql_desarrollo.go:542`, envuelta por
`contratacion_temporal_autoridad_postgresql_desarrollo.go:763`. El inventario
independiente confirma que una lista vacía debe poder consultarse; la
ausencia de selección no explica el error. La preparación y la publicación
del perfil leen en transacciones distintas, pero todavía no se ha demostrado
qué petición adelantó la versión. Se comunicó a dirección para F1-33.
La guarda se conserva y el ensayo continúa con el paso 9 desde las 09:00.

Tras retomar I se retiró G con su gestor original, conservando los registros
privados y su hoja de evidencia. H sigue parada con el envío rechazado.

A las 08:52 se abrió la ficha una sola vez, siguiendo la respuesta de dirección
de las 08:46. La consulta de comunicaciones devolvió 200 y una lista vacía.
El 503 anterior queda registrado como F1-150. No se repitieron actuaciones
ni se cambió la configuración. La selección y el contacto siguen pendientes
de la franja de llamadas.
