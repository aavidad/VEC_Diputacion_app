# Reintentar un aviso sin confirmación

Tarea F1-18, segundo corte, ensayado en la copia local el 18/10. Depende del
primer corte #1180 y del contrato de dirección del 16/10 a las 06:09. Dirección
integra ambos cortes, en ese orden.

## Activación

Partir de la misma copia PostgreSQL 18 con B129 y B131 instaladas y sus permisos completos. Instalar una sola vez B132 mediante `lista_sql_codexs_reintento_aviso_f1_18_20261016.txt`; no volver a aplicar B129 ni B131. El binario conserva el funcionamiento anterior si falta B132, pero el reintento expreso requiere esa migración. No hay nueva configuración: reutiliza el enlace HTTPS de Mi Bolsa, el material versionado y el transporte de correo del primer corte.

Reiniciar con vigilancia. Sin enlace o con uno inválido, la app sigue arrancando y el reintento queda cerrado, con el motivo del catálogo. Restaurarlo y reiniciar permite consultar el estado antes de actuar. La vuelta atrás mantiene un binario compatible con B129/B131/B132 y conserva su historia.

## Recorrido

En «Gestionar ofertas», abrir una oferta con un envío sin confirmación. «Reintentar envíos sin confirmación» presenta la oferta, el número de avisos y una advertencia: el mensaje puede haberse recibido y podría duplicarse. Cancelar no envía nada. Confirmar registra una operación propia, con nueva clave y la revisión de avisos que se consultó.

La autorización queda ligada a esa entrada y se consume con el cambio de estado. Solo se rearman avisos sin confirmación; se conserva el identificador de cada mensaje y el recibo, la fecha y el plazo originales de la oferta. El acto de reintento tiene además su propio recibo y fecha. Una revisión obsoleta o un envío todavía en curso se rechazan sin efectos. Repetir la misma clave y entrada recupera el acto sin rearmar de nuevo.

Mientras una publicación esté pendiente de confirmación, el reintento manual permanece bloqueado. La prueba focal «una publicación incierta bloquea la confirmación manual hasta recuperar su recibo» comprueba que no se emite ese segundo acto y que la recuperación conserva la clave y el cuerpo originales.

La intención permanece solo en memoria del navegador. Ante una respuesta incierta se recupera esa misma operación; no se crea otra clave automáticamente. Un nuevo resultado sin confirmación exige una nueva confirmación expresa. El correo aceptado por el servicio no acredita entrega ni notificación administrativa.

## Ensayo del 18 de octubre

Fuente `1c2588362`, receta interna de L `fef11a8c3`, sin portal externo. B132 se
instaló una sola vez después de las 80 SQL anteriores. Su prueba focal ejecutó
BEGIN, comprobó revisión, protección de la tabla y cambio idéntico, y terminó
en ROLLBACK. No se fabricaron decisiones, firmas ni recibos. La guía del primer
corte recoge la preparación de la copia y los límites de la receta.

Chrome publicó una oferta de dos plazas con doce destinatarios. Once carecían
de contacto de alta y quedaron fallidos. Para el restante se había registrado
un correo ficticio por la API nominal de datos de contacto, con recibo. El
buzón TLS propio recibió el mensaje y cortó antes de contestar al DATA:
la pantalla mostró un envío sin confirmación. No se reenvió al consultar.

«Reintentar envíos sin confirmación» mostró la oferta, una persona y la
advertencia de posible duplicado. Cancelar no emitió POST ni correo. Al
confirmar, el reintento registró un recibo propio y volvió a enviar el mensaje.
Durante los ocho segundos que el buzón demoró su respuesta, la misma petición
con la misma clave recuperó el acto con 200; otra clave recibió 409
`aviso_envio_en_curso`, sin crear otro acto ni envío.

Se perdió deliberadamente la respuesta HTTP real 201 después de que la app
terminara el reintento. La pantalla mantuvo la intención y ofreció recuperar
el resultado. Tras reiniciar aplicación y PostgreSQL, el botón recuperó con
200 la misma clave, entrada, recibo y fecha del reintento. El resultado final
fue un enviado, once fallidos y ningún pendiente o envío sin confirmación.
Se conservaron también recibo, fecha y vencimiento originales de la oferta.
El buzón recibió exactamente dos mensajes de ese aviso: el primero sin
respuesta y el reintento confirmado, con idénticos Message-ID y bytes. La
recuperación posterior no añadió mensajes.

Se comprobó la pantalla final en castellano e inglés, a 1440 y 390 píxeles:
sin desbordamiento horizontal, sin scroll de documento en escritorio y sin
errores JavaScript. No hubo cookies ni almacenamiento web. Las primeras
capturas del cambio de idioma se descartaron por tomarse durante la carga;
las definitivas esperan el formulario visible. TLS y certificado de cliente
se verificaron, sin omitir errores. Los fallos previos de la ficha importada
y los cortes de conexión durante los reinicios quedan fuera de esa afirmación.

Con enlace ausente e inválido se comprobó el arranque y la conservación de
pendientes antes del segundo envío. Restaurar el enlace permitió continuar.
Las denegaciones de reintento por configuración, revisión obsoleta y permiso
siguen cubiertas por las pruebas focales de código; no se atribuyen al recorrido
real de esta copia.

Las dos revisiones sensibles anteriores y sus comprobaciones focales siguen
vigentes: Go y SQL propios no cambiaron. Semgrep no encontró hallazgos en lo
cambiado; GoSec conserva la cobertura parcial documentada hasta F0-29. La
puerta de calidad de `1c2588362` terminó verde. Los guiones, capturas y
respuestas están guardados fuera de Git. No se acredita correo corporativo,
entrega al destinatario, portal externo ni ficha del candidato importado.
