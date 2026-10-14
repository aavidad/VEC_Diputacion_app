# Dietas: resultado del recorrido local

Estado a 14 de octubre de 2026: **recorrido parcial comprobado; candidata no instalable**.

## Copia y activación

La prueba usa una copia propia de HZ37 en PostgreSQL 18.4 (puerto local
55453), con los objetos globales y los permisos restaurados. No se modificó
el servicio compartido.
AD81 y el archivo explícito final de AD268 quedaron instalados una vez. También
quedaron instaladas una vez las migraciones de Dietas 000009, 000010, 000011,
000014, 000015 y 000018. La corrección de 000018 permite consultar la lista
propia; su instalación terminó en `COMMIT`.

El núcleo económico usado en este ensayo se instaló mediante una pieza privada.
Ese hecho **no acredita** la migración común AD273 de C: su versión final no
está en Git ni se ha ensayado como parte de una candidata reproducible.
Dietas 000013 (`27544e2…`), 000017 (`37bd81…`), 000019 (`fc55070f…`) y los
catálogos de tarifas (`54ec707…`) quedaron instalados una vez en la copia.
Las versiones finales de 000013, 000017 y tarifas recibieron dos revisiones
favorables. La revisión de 000019 también recibió dos vistos buenos y su
instalación terminó en `COMMIT`; corrige solo las cinco usos de la variable local del recibo.

## Comprobación en la aplicación

La aplicación local sirvió la cadena de caché v4 sin errores JavaScript. En
Chrome, la creación de una comisión con la misma clave y el mismo contenido
originales, después de los errores previos, devolvió `201`. Tras recargar, la
lista y el detalle respondieron `200`. El borrador ficticio quedó como
`VEC-D-2026-000002`, recibo
`rcd_e295960a-071d-b2fd-b83b-5f9e3f6adc21`, versión 1. La ruta Granada a
Santa Fe calculó 13,7 km y el cálculo mostró la tarifa publicada.

Este recibo acredita el alta del borrador en la copia. La consulta de
asignación de Personal devolvió `403` y bloqueó la edición y el envío; el
diagnóstico confirmó una concesión de lectura propia ausente del rol ficticio. No se han recorrido
una edición ni un envío confirmados. Tampoco se han acreditado la subida y
custodia del justificante, las etapas posteriores, la liquidación económica,
la descarga PDF/Word ni la recuperación tras reiniciar.

Se corrigió la petición a OSRM: las alternativas son adicionales a la ruta
principal, mientras que Dietas establece el número total de rutas. La corrección recibió dos revisiones favorables y pasó las
comprobaciones focales, `race`, `vet`, Semgrep local y `gosec`. No se cambió el
servicio compartido para obtener este resultado.

## Pendiente para cerrar el recorrido

Completar la asignación ficticia por el cauce del producto y repetir edición y
envío. Después, comprobar justificante custodiado, revisión, autorización,
liquidación, fiscalización, PDF y Word, reintento exacto y recuperación tras
reiniciar aplicación y PostgreSQL. También faltan el arranque sin la nueva
configuración y con ella incompleta. El ensayo local descrito aquí no sustituye
la prueba de la candidata congelada y entera en una copia de cidonia.
