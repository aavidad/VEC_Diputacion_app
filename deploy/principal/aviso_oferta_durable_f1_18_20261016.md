# Aviso durable de ofertas por correo

Tarea F1-18, primer corte. B129 instalada una vez en la copia propia; prueba SQL focal con rollback conforme, recorrido de correo aún pendiente. Usa el correo del alta de Bolsa y deja constancia de esa fuente. La consulta del correo verificado de Usuarios y la elegibilidad de personas en activo quedan para sus contratos posteriores.

## Activación en la copia

1. Partir de una copia PostgreSQL 18 de la principal con propietarios y ACL completos. B117 debe estar instalada. Detener los escritores propios; no modificar material ni servicios de otro equipo. B129 coteja los cuerpos instalados de publicación y proyección.
2. Instalar una sola vez `lista_sql_codexs_aviso_oferta_f1_18_20261016.txt`. No hay DOWN. Las ofertas anteriores conservan su comportamiento; solo las nuevas fijan destinatarios. El binario nuevo y B129 se activan juntos: un publicador anterior no puede confirmar una intención nueva sin su vínculo de despacho.
3. Conservar las conexiones y el material HMAC versionado existentes. No se crean secretos ni perfiles. Cada oferta conserva la generación de su token para recuperarla después de una rotación; retirarla prematuramente cierra el despacho.
4. CONFIG NUEVA: `VEC_BOLSA_AVISO_OFERTA_ENLACE`, URL HTTPS de Mi Bolsa, con ruta `/area-personal/`, sin credenciales. Solo se selecciona en la copia privada. La plantilla está en `config/bolsa_aviso_oferta_v1.json`, con textos por idioma; no contiene nombres, posición ni datos del expediente. El transporte SMTP y su confianza reutilizan la configuración ya existente.
5. Reiniciar con vigilancia el binario exacto. Sin el enlace o con un enlace inválido, la app arranca y conserva las intenciones pendientes; la pantalla informa de la configuración ausente. Restaurar el enlace y reiniciar permite continuar la misma publicación. Una vuelta atrás mantiene un binario compatible con B129; no se revierte la migración ni se borra historia.

## Recorrido único de este corte

Desde una bolsa constituida, abrir «Gestionar ofertas». Con el transporte de correo propio detenido, publicar una oferta: recibo, destinatarios fijos y avisos pendientes. Conservar abierta esa confirmación; el navegador mantiene la intención solo en memoria, sin cookies ni almacenamiento web.

Reiniciar aplicación y PostgreSQL, poner en marcha el buzón local propio y pulsar «Reintentar avisos pendientes». Se envían los mismos destinatarios con el mismo identificador de mensaje y la fecha de publicación. La confirmación conserva oferta, recibo, fecha y plazo. Repetir no vuelve a enviar los aceptados. Una persona que queda disponible después ve la oferta en Mi Bolsa, pero no se añade a la instantánea de correo.

Un rechazo permanente queda fallido. Si la conexión termina después de transmitir el mensaje y no se conoce el resultado, queda sin confirmación y no se reenvía automáticamente, tampoco tras reiniciar. Este corte no ofrece el reintento expreso de ese resultado incierto; RRHH conserva su seguimiento por teléfono. «Enviado» no acredita entrega ni notificación administrativa.

Comprobar en Chrome a 1440 y 390 píxeles, en castellano e inglés: estados, recibo, recuperación, configuración ausente y ninguna información de destinatarios expuesta en el resumen. La clave original se pierde al cerrar o recargar la página; el resumen persiste en el servidor. La recuperación durable desde el expediente pertenece a F1-16.

## Controles y evidencia pendiente

- La instantánea y la vinculación se confirman con la publicación autorizada. Una vinculación posterior, de otro actor o con token distinto se rechaza. La prueba SQL B129 comprueba estos rechazos y el conjunto de destinatarios.
- La reclamación guarda la marca incierta antes de la red; la finalización coteja intento y token. La prueba focal del despacho cubre los cuatro resultados, caída tras enviar y recuperación sin duplicar. Una reclamación no confirmada nunca envía correo.
- El envío usa un presupuesto técnico por petición; el resto conserva su intención. No hay un proceso de fondo ni otra bandeja de salida. El aviso no modifica las reglas, el plazo ni la declaración de notificación de la oferta.
- Quedan las dos revisiones independientes de la cabeza final, Semgrep y gosec locales, instalación única de B129, recorrido completo en el buzón propio y puerta de calidad en CI. Este documento no acredita un envío real ni una candidata instalada.

El ensayo SQL de `20c56b29c` instaló B129 y pasó las comprobaciones de instantánea, token ajeno, vínculo exacto, reclamación antes de la red, resultado y rotación del turno tras un transitorio. La primera prueba revirtió por el nombre de una restricción sin su esquema en el guion; al cualificarlo pasó y revirtió todos sus datos. La migración no se reaplicó.

La copia causal contiene S6 de #1155, que cierra la publicación antigua de reglas durante el arranque. El código anterior de la principal todavía usa esa vía; el ensayo conjunto requiere el código de #1155, cuya integración corresponde a dirección. No se restituyen permisos ni se omiten guardas para arrancar. Este bloqueo de composición no acredita un envío de correo.
