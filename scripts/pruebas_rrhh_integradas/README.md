# Recorrido sintético integrado RRHH

`./scripts/pruebas_rrhh_integradas/ejecutar.sh pg18` reutiliza cuatro ensayos
existentes, con PostgreSQL 18 efímero, para cese y cálculo +5/+9, proyección en
Mi Bolsa, reincorporación y la política de ofertas de 48 horas. No usa una base
compartida ni restaura volcado alguno.

`./scripts/pruebas_rrhh_integradas/ejecutar.sh navegador ORIGEN CERT CLAVE`
recorre la lectura propia de Mi Bolsa con Chromium y certificado sintético. El
origen debe ser HTTPS loopback; el servidor debe haber sido arrancado por el
director con la composición real, los DSN sintéticos, las migraciones y el
material de candidato ya preparados. Comprueba en 1440 y 390 px la navegación,
GET Mi Bolsa, GET del histórico propio, respuestas 200, ausencia de cookies,
almacenamiento web, errores JavaScript y desbordamiento horizontal.

Este corte no escribe por SQL, no inicia ni reinicia VEC y no emite operaciones
de cese, reincorporación ni oferta desde el navegador. Esas operaciones no tienen
una ruta UI compuesta común en `fc78cbd9b`; se validan aquí por sus ensayos PG18.
Los endpoints de reincorporación, política de ofertas, auditoría y plantillas que
están en candidatos de integración se añadirán al recorrido sólo tras recibir su
hash con montaje, permisos y datos sintéticos verificables.
