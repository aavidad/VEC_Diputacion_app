# Una selección confirmada por expediente

F1-69 prepara CT238 para impedir dos selecciones confirmadas en la misma
organización y expediente. Se aplica tanto al llamamiento de una persona como
a la oferta de varias plazas: CT220 conserva una sola selección original para
todas ellas. Las continuaciones recuperan esa selección. CT234 crea referencias
propias para los pasos posteriores y, en este primer corte, no crea otra
selección por cada persona.

## Activación de la candidata

1. Partir de una copia local de la principal con PostgreSQL 18 y datos
   ficticios. Usar `copia_local.py crear --modulos --rrhh-original` con el
   material original aprobado y los preparadores propios de la receta común.
   Pasar explícitamente `--lista-sql` con la lista única: 85 migraciones
   de main y después CT238, 86 en total. Instalarla una vez;
   no reaplicar SQL con historia.
2. CT238 bloquea las escrituras de la tabla mientras comprueba que cada
   selección confirmada tiene organización y expediente y que no hay dos para
   el mismo par. Si encuentra una anomalía, se detiene sin borrar historia ni
   crear el índice. Se investiga el origen antes de preparar otra candidata.
3. Comprobar en la copia la selección de una persona, la oferta de varias
   plazas, la recuperación de una selección confirmada pendiente de aviso y
   el rechazo de una segunda confirmación para el mismo expediente. Usar la
   ruta y la pantalla actuales. Comprobar también arranque con y sin la
   configuración existente, reinicio y conservación de la selección original.

La prueba `ct238_seleccion_confirmada_unica.sql` usa una selección singular
confirmada de la copia HZ37. Si falta, se prepara por el recorrido habitual.
El ensayo focal revirtió sus filas; no acredita actos ni consumos de autorización.

El índice único de CT238 cubre solo las filas confirmadas de O6 y usa la
organización y el expediente guardados en la solicitud. También sirve a la
consulta CT231. No añade estados, permisos, funciones ni un lector nuevo.

## Ensayo local y límites

La copia correcta de la cabeza `b7bbc7bdb` se creó en 77,11 segundos con
`--lista-sql` e instaló una vez las 82 migraciones. La prueba SQL focal pasó
con ROLLBACK. Una primera copia sin esa opción no instaló CT238; se descartó
y no se usa como prueba.

La prueba física de concurrencia insertó una selección singular sin confirmar
la transacción. Otra conexión intentó registrar una oferta para la misma
organización y expediente, esperó a la primera y, tras confirmarla, recibió
`23505` del índice CT238. Los recuentos quedaron en seis filas antes y después;
no quedó ninguna fila de prueba. Esto acredita la unicidad del índice, no
una autorización ni un acto O6 con recibo real.

En aquella copia, con los módulos configurados, proceso vivo (`/livez`),
portal, cuadro de contratación y Bolsas respondieron 200. Sin módulos,
`/livez` y portal respondieron 200 y Bolsas 503. El reinicio conservó esos
resultados. La consulta detallada respondió 503 y motivó repetir el ensayo
con la receta común de RRHH, sin cambiar CT238.

La nueva copia `1a48d9086`, desde main con las 85 migraciones y CT238, se creó
en 84,6 segundos con los certificados y la identidad originales aprobados.
Faltaban los dos grupos vacíos de Méritos: se aplicaron sus definiciones
canónicas, sin miembros ni concesiones, tras dos revisiones y un ensayo con
ROLLBACK. Las copias nuevas ya los reciben del guion común de main.

Con el certificado RRHH, la misma consulta detallada respondió 200 y recuperó
el recibo original de O6. Tras reiniciar, devolvió el mismo cuerpo y recibo.
Chrome abrió la ficha, consultó las comunicaciones y mostró la fila del
llamamiento, que se pudo seleccionar sin enviar otra petición de escritura.
El recibo se cotejó en la respuesta y en la base: esa ficha de una selección
ya comunicada no lo imprime. Las seis selecciones existentes ya estaban
comunicadas; no se creó otro aviso para este ensayo.

`/livez` respondió 200 y `/healthz` 503 en esta copia de montaje parcial;
no se declara disponibilidad de todos los módulos. Dos revisiones de la
fuente de CT238 son favorables. La integración y la candidata de dirección
siguen pendientes.

Vuelta atrás: conservar la candidata y la copia anteriores. Si la instalación
se detiene, resolver la causa sobre datos ficticios y preparar otra copia. No
ejecutar DOWN ni reaplicar SQL. Los datos ficticios se pueden recargar solo en
la copia propia; se conservan las guardas y el trabajo de los demás equipos.
