# Recorridos pendientes de manual

## Comprobación del 30/09/2026

Se han observado ocho recorridos con Chrome en el clon sintético de
`main@8fc0b534dbdaa0a5b34d835810c4593ad823780b`, con 38 instalaciones SQL.
Las capturas reales de escritorio y móvil se conservan fuera de Git para revisión.
Dirección aún no ha confirmado un proceso completo que permita publicar su manual.

| Recorrido | Resultado observado | Punto pendiente |
| --- | --- | --- |
| Accesos | RRHH, centro y ratificador pasan sus comprobaciones positivas y de denegación. Intervención entra al portal. | Completar las comprobaciones de Intervención y disponer del portal externo con su propia autoridad de acceso. |
| Centro → RRHH | Petición y ratificación registradas por personas distintas, con recibos de versiones 1 y 2. La consulta posterior conserva la petición ratificada. | La entrega a RRHH responde 503: su POST todavía carece de frontera nominal en esta fuente. No hay alta confirmada. |
| Análisis e informe | El cuadro abre y consulta los expedientes con respuesta 200. | No hay expediente nuevo procedente de la entrega. El sembrador oficial se detiene antes de escribir porque faltan centros y categorías de sus casos. No se han ejecutado las actuaciones. |
| Intervención | La entrada y el formulario de acceso manual se ven en escritorio y móvil. Las consultas de RRHH se deniegan a este perfil. | Faltan casos nuevos en informe jurídico para recorrer fiscalización favorable, reparo y subsanación. |
| Llamamiento y respuesta | Se han observado las entradas de RRHH y del candidato. | La lista de bolsas responde 503 por una conexión omitida en el preparador del clon. Se corregirá antes de valorar el proceso. El portal externo aún no está habilitado. |
| Propuesta, documentos y firma | Una propuesta histórica en versión 7 permite descargar los seis borradores. Sus tamaños y huellas coinciden a 1440 y 390 píxeles. | La consulta de seguimiento responde 404 y el circuito de firma 503. No se ha solicitado una firma ni continuado hasta el cierre. |
| Bolsa y ofertas | El portal interno abre. | La misma conexión omitida impide cargar las bolsas. No se han publicado ofertas ni ejecutado respuestas, avisos o envío masivo. |
| Área personal | RRHH consulta sus preferencias internas y las recupera sin cambios tras recargar. Correos e imagen responden como lecturas. | Esto no acredita el Área personal del candidato. Faltan su entrada, los cambios confirmados, correo verificado, «Mi bolsa» y «Mi ficha». |

El primer 400 observado en el cuadro era un efecto del transporte del guion;
Chrome nativo devuelve 200. Tampoco se atribuye al producto el 503 de Bolsa
causado por el preparador. Los recibos del centro se conservan para recuperar
la misma petición; no se ha comprobado todavía su recuperación tras reinicio.

La actualización del clon a una fuente posterior se registrará por separado.
Estos resultados no se atribuyen a otro binario ni a SQL instaladas después.

## Evidencia anterior: 29/09/2026

Revisión del 29/09/2026 sobre `origin/main@3b910a170`. Aún no está acreditado de principio a fin ninguno de los procesos completos pedidos para esta carpeta. Las dos copias locales consultadas contienen expedientes y propuestas de ejemplo, pero no registran firmas de Contratación temporal.

En esta revisión no se abrió Chrome contra una copia con todas las novedades instaladas. La tabla recoge el último paso demostrado en las comprobaciones anteriores.

| Proceso | Hasta dónde llega la evidencia | Qué falta para terminarlo |
| --- | --- | --- |
| Acceso de RRHH | Entra al portal interno, recibe una petición ratificada, crea el expediente y recupera el recibo tras reiniciar. | Continuar ese expediente hasta la firma y el cierre con los perfiles fijos aún en curso. |
| Acceso del centro solicitante | Registra una petición y recupera su recibo tras reiniciar. La petición ya se ha entregado a RRHH en un ejercicio posterior. | Completar el expediente resultante. La adaptación de la entrega al perfil fijo sigue en curso. |
| Acceso del ratificador | Ratifica la petición con otra identidad y recupera el recibo. La entrega posterior a RRHH también está probada. | Continuar el expediente hasta su cierre con los permisos definitivos. |
| Acceso de Intervención | Un ejercicio anterior registró una fiscalización favorable y recuperó el recibo tras reiniciar. El perfil fijo nuevo no permite recuperar el justificante completo de esa actuación: una repetición histórica devuelve conflicto sin efectos, el detalle muestra la actuación y la auditoría conserva solo la referencia del recibo. | Demostrar los caminos favorable y de reparo con subsanación dentro de la cadena completa y con las firmas admitidas. Aclarar con RRHH si necesita descargar el justificante de cada actuación ya registrada. |
| Acceso del candidato y Área personal | Puede entrar y ver «Mi bolsa» con sus participaciones en la presentación sintética. | Terminar la respuesta y los demás trámites propios con las reglas y la identidad definitivas. La separación de procesos por portal sigue en varias solicitudes de cambio abiertas. |
| Contratación temporal | Los ejercicios de desarrollo alcanzan la propuesta de nombramiento y los seis borradores. Seguimiento y cese también se recorrieron en Chrome en una copia de ensayo, con recuperación tras reiniciar la aplicación y la base de datos. GrxFirma verificó firmas de prueba en la principal. | Unir el flujo desde el centro hasta la firma por el circuito admitido, verificar y custodiar el documento firmado, incorporar, seguir y cerrar el mismo expediente. La firma legal y el envío al portafirmas están pendientes. También faltan los caminos favorable y de reparo completos, sus recibos y la recuperación tras reinicio. Los cambios de perfiles fijos y custodia siguen abiertos. |
| Bolsa | Hay constitución y orden sintéticos, política de ofertas, publicación y respuesta en ensayos locales; un llamamiento llegó al buzón de pruebas. | Comprobar en un mismo clon y desde el navegador publicación, respuesta, avisos, envío masivo, adjudicación y cierre, con recibos recuperados tras reinicio. Faltan reglas aceptadas por RRHH y acreditar la entrega de correo corporativo. |
| Área personal: preferencias | «Mis preferencias» se guardó y recuperó en las dos entradas de prueba. | Incorporarlo a un recorrido completo del Área personal, con las demás funciones propias terminadas. |
| Área personal: correos, imagen, «Mi bolsa» y «Mi ficha» | Correos, imagen y «Mi bolsa» tienen cortes de código y pruebas parciales. La pantalla «Mi ficha» sigue en una solicitud de cambio abierta. | Probar las cuatro funciones juntas en el clon actualizado, desde la entrada hasta la confirmación y recuperación de cada cambio. Completar la ficha del aspirante y su trámite de solicitud. |
| Administración: reglas vigentes | Se pueden consultar reglas; la apertura de su detalle sigue en una solicitud de cambio abierta. | Comprobar lectura completa y las operaciones de gobierno autorizadas con historia y recibo. |
| Administración: catálogos, plazos y categorías de la RPT | Los catálogos actuales se consultan. Los plazos configurables están en estudio y ramas; la edición de categorías aún no tiene implementación. | Permitir y comprobar los cambios autorizados, su versión, historia, recibo y recuperación. Las categorías en uso solo podrán deshabilitarse. |

Para retomar un manual, dirección debe confirmar que el proceso correspondiente está terminado. Entonces se recorre con datos sintéticos en un clon de la principal actualizado, se capturan y numeran los controles reales a 1440 píxeles y, cuando importa, a 390 píxeles, y otra persona sigue el manual antes de abrir su solicitud de cambio.

Fuentes de este corte: [estado del proyecto](../../ESTADO_PROYECTO.md), [guía del recorrido centro a RRHH](../../GUIA_RECORRIDO_ALBERTO.md), [lista de comprobación de RRHH](../../PIDEN_RRHH_CHECKLIST.md), [guion de Bolsa](../presentacion/guion_demo_rrhh_2026-09.md), [estado de firma](../../deploy/firma/README.md) y [alcance del expediente](../portal_vec/expediente_contratacion_temporal_rrhh.md).
