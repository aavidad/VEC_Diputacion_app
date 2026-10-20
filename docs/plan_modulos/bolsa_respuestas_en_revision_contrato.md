# Confirmar una aceptación directa pendiente — F1-71

Dirección confirmó el 16/10/2026 a las 01:39 este alcance: RRHH confirma una aceptación de llamamiento directo desde la ficha existente. Se reutiliza la lista nominal de candidatos; este corte no añade una bandeja ni otra lectura. La admisión y el rechazo de justificaciones quedan fuera de este corte. El código del justificante PDF de #1147 ya está integrado; su activación y el cotejo nominal de custodia conservan su aceptación propia. CAT15 permanece fuera de esta confirmación.

Corte de estado del 20/10/2026 sobre `origin/main@3ef7dee0b`: AD288 y B128
están integradas por #1176 y #1164. El
[guion existente](../../deploy/principal/aceptacion_desde_ficha_f1_71_20261018.md)
conserva el ensayo parcial del 18/10 y sus límites. La copia ensayada recibió
la cadena por etapas; no acredita una instalación desde cero de toda la lista
actual. Tampoco sustituye la resolución de la aceptación del caso único.

## Recorrido y contrato

La marca `aceptacion_pendiente` permite abrir «Confirmar aceptación» en la ficha. RRHH revisa la persona, el llamamiento y el paso a pendiente de incorporación, escribe el motivo y confirma. El recibo conserva la respuesta resuelta, la resolución, el motivo y la fecha originales. Regularizar y Excluir mantienen sus contratos.

El POST es `/api/vec/bolsa/bolsas/{bolsa}/candidatos/{participacion}/respuestas/{llamamiento}/confirmacion`. El cuerpo cerrado contiene `motivo` y `situacion_desde_esperada`; la cabecera `Idempotency-Key` conserva la operación. El navegador obtiene las referencias y la fecha de situación de la lectura nominal existente. Bolsa deriva la respuesta B30 única por llamamiento y participación y coteja su vínculo B17; el navegador no aporta identidad ni destino.

El acto usa `bolsa.respuesta_revision.resolver`, recurso de tipo `respuesta_revision_bolsa` y referencia de participación, finalidad `resolver_respuestas_bolsa_rrhh` y audiencia `vec_bolsa_llamamientos.respuesta_revision.resolver.v1`. Los ámbitos son `unidad_ref` y `ambito_ref`, resueltos desde la sesión confiable y cotejados con el gobierno de transiciones activo. El atributo `confirmacion_aceptacion_sha256` liga el canon completo a la autorización. El motivo administrativo se conserva como texto del acto; el motivo de autorización publicado lo selecciona la preparación de confianza.

El canon contiene, en orden: esquema `vec.bolsa.respuesta-revision.confirmar-aceptacion.v1`, bolsa, participación, llamamiento, fecha esperada UTC con seis decimales, motivo, clave, actor y destino fijo `pendiente_incorporacion`. La aplicación reutiliza el servicio de situación. La base consume V3 antes de leer la respuesta y, en esa misma transacción, coteja pertenencia, respuesta `acepta`, situación vigente, fecha esperada y transición publicada. El registro de una situación posterior cierra la marca pendiente de B100/B101.

Una resolución inmutable conserva respuesta, canon, clave, recibo y evento pendiente. B34 mantiene la traza del cambio. El reintento consume autorización vigente y devuelve el recibo original antes de cotejar la situación o política actuales. Otra intención con la misma clave, otra resolución de esa respuesta o una fecha esperada distinta devuelven 409. El adaptador verifica el recibo antes del COMMIT. Un error revierte consumo y efecto, y la frontera registra el intento por su vía nominal.

## Activación y límites

El [guion de activación](../../deploy/principal/aceptacion_desde_ficha_f1_71_20261018.md) describe publicación del permiso por dos administradores, origen de consumo, orden SQL y reinicio. AD288 corresponde a R; B128 conserva la escritura de Bolsa. Se reutiliza la política global de #1155 y su publicación normativa. No se activa CAT15 ni se interpreta una referencia documental como prueba de custodia.

La regla de confirmación es provisional y configurable mediante la política de situación mientras RRHH responde la duda 157. El [Reglamento publicado el 16 de enero de 2026](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1768521622300-final-680c0e98.pdf), artículos 10 y 11, se aplica al posterior corte de justificaciones según el procedimiento acreditado. El [cambio del 14 de mayo](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1778713220300-final-241a2c51.pdf) regula la adhesión de organismos autónomos; no aporta un vínculo entre esta respuesta y una oferta publicada.

El código y su consumidor están integrados. Cada copia necesita los actos de
activación y el material descritos en el guion; la presencia de código no los
crea al arrancar. El ensayo acreditado recupera el recibo tras reiniciar, con
sus límites declarados. No acredita instalación en un servidor ni el cierre
del [caso 4.0](../producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación).
