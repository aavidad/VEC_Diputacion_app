# Publicación con auditoría de resultados

`NuevoPublicador` envuelve el puerto durable de denominación de Persona. Recibe
el registrador común de intentos y una configuración privada con proceso, canal,
motivos de denegación y error, y plazo de auditoría. No concede permisos ni emite
capacidades. La autoridad durable conserva el consumo V3, CAS, historia, puntero,
auditoría permitida y outbox en su propia transacción.

Un recibo confirmado debe corresponder a la Persona, procedencia, versión y
huella del sobre preparado. Si la autoridad deniega, falla o devuelve un recibo
incompatible, el publicador registra el resultado mediante la auditoría común.
El módulo y la finalidad son los del contrato de denominación; el recurso es la
Persona validada de la preparación. Una entrada que pretenda cambiarlos no llega
a la autoridad y se registra como error sobre esa Persona.
Conserva el contexto registrado V2 y su vínculo original; no atribuye una persona
cuando esa evidencia es inválida. Sólo una denegación explícita de la autoridad
se clasifica como denegada. Un COMMIT desconocido se registra como error
observado: no afirma que el efecto se haya deshecho y no repite la publicación.

El intento se registra después de retornar la autoridad durable. Si la petición
se cancela, usa el plazo configurado conservando los valores del contexto. Ante
un COMMIT incierto del registrador común, vuelve a presentar la misma orden una
vez. Sin acuse válido, devuelve indisponibilidad y ningún recibo. No guarda el
nombre, términos de búsqueda, datos del certificado ni mensajes SQL.

La autoridad recibe copias de los mapas de contexto y de los bytes del sobre.
Las pruebas sintéticas cubren confirmación, denegación, COMMIT desconocido,
recibo ajeno, alteración de copias, cancelación y recuperación de un intento
ambiguo. Son dobles de puertos: no acreditan persistencia ni autorización real.
El montaje necesita el adaptador CA32, las capacidades nominales AD184 y AUT42,
un emisor real y el registrador común. Esta pieza no los sustituye ni activa
una pantalla.
