# Reintentar un aviso sin confirmación

Tarea F1-18, segundo corte, en preparación. Depende del primer corte #1180 y del contrato de dirección del 16/10 a las 06:09. No acredita instalación ni envío.

## Activación prevista

Partir de la misma copia PostgreSQL 18 con B129 y B131 instaladas y sus permisos completos. Instalar una sola vez B132 mediante `lista_sql_codexs_reintento_aviso_f1_18_20261016.txt`; no volver a aplicar B129 ni B131. El binario conserva el funcionamiento anterior si falta B132, pero el reintento expreso requiere esa migración. No hay nueva configuración: reutiliza el enlace HTTPS de Mi Bolsa, el material versionado y el transporte de correo del primer corte.

Reiniciar con vigilancia. Sin enlace o con uno inválido, la app sigue arrancando y el reintento queda cerrado, con el motivo del catálogo. Restaurarlo y reiniciar permite consultar el estado antes de actuar. La vuelta atrás mantiene un binario compatible con B129/B131/B132 y conserva su historia.

## Recorrido previsto

En «Gestionar ofertas», abrir una oferta con un envío sin confirmación. «Reintentar envíos sin confirmación» presenta la oferta, el número de avisos y una advertencia: el mensaje puede haberse recibido y podría duplicarse. Cancelar no envía nada. Confirmar registra una operación propia, con nueva clave y la revisión de avisos que se consultó.

La autorización queda ligada a esa entrada y se consume con el cambio de estado. Solo se rearman avisos sin confirmación; se conserva el identificador de cada mensaje y el recibo, la fecha y el plazo originales de la oferta. El acto de reintento tiene además su propio recibo y fecha. Una revisión obsoleta o un envío todavía en curso se rechazan sin efectos. Repetir la misma clave y entrada recupera el acto sin rearmar de nuevo.

La intención permanece solo en memoria del navegador. Ante una respuesta incierta se recupera esa misma operación; no se crea otra clave automáticamente. Un nuevo resultado sin confirmación exige una nueva confirmación expresa. El correo aceptado por el servicio no acredita entrega ni notificación administrativa.

## Comprobaciones pendientes

- Dos revisiones independientes de la cabeza final; pruebas focales de autorización, revisión, idempotencia, envío en curso y conservación del identificador de mensaje.
- Instalación única de B132 en la copia propia y prueba SQL con rollback, sin fabricar decisiones, firmas o recibos.
- Chrome en castellano e inglés, a 1440 y 390 píxeles: cancelar, confirmar, recibos, resultado incierto y recuperación tras reiniciar aplicación y base de datos.
- Configuración ausente e inválida: arranque y consultas disponibles, reintento cerrado sin rearmar.
- Semgrep y gosec locales sobre lo cambiado, revisión de usabilidad y puerta completa de CI.
