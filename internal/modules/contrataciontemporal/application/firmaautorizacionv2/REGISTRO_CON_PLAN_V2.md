# Registro de firma V2 con plan fijado

El decorador recibe la capacidad interior emitida para CT172 y relee el plan publicado desde la fuente configurada del servidor. Comprueba que el descriptor elegido tiene los mismos bytes que el descriptor ligado a esa capacidad. Calcula la huella de los bytes reales de la decisión interior, prepara el envoltorio de cuatro campos y solicita una capacidad exterior nueva a la autoridad nominal común.

El paquete de transporte conserva ambas exportaciones V3 y el pin. El adaptador PostgreSQL llama una sola vez a CT176 dentro de una transacción serializable. CT176 consume ambas decisiones, registra o recupera el efecto CT172, comprueba el plan y conserva la hija histórica. Su respuesta mantiene las **14 claves originales del recibo CT172**: no devuelve metadatos nuevos de plan. El adaptador coteja ese recibo antes del commit; no reconstruye procedencia desde una autorización nueva ni devuelve recibo si el commit es incierto.

La lectura de firmas de 44 campos sigue delegada al repositorio anterior. La composición raíz y el montaje HTTP quedan pendientes. Los dobles de estas pruebas acreditan el contrato Go y el cierre de la transacción, no una migración instalada ni una firma real.
