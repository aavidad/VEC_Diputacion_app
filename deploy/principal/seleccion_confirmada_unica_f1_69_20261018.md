# Una selección confirmada por expediente

F1-69 prepara CT238 para impedir dos selecciones confirmadas en la misma
organización y expediente. Se aplica tanto al llamamiento de una persona como
a la oferta de varias plazas: CT220 conserva una sola selección original para
todas ellas. Las continuaciones recuperan esa selección. CT234 crea referencias
propias para los pasos posteriores y, en este primer corte, no crea otra
selección por cada persona.

## Activación de la candidata

1. Partir de una copia local de la principal con PostgreSQL 18 y datos
   ficticios. Pasar explícitamente `--lista-sql` con la lista única: 81
   migraciones de main y después CT238, 82 en total. Instalarla una vez;
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

Con los módulos configurados, salud, portal, cuadro de contratación y Bolsas
respondieron 200. Sin esa configuración, salud y portal respondieron 200 y
Bolsas 503. Tras reiniciar aplicación, PostgreSQL y buzón local, las cuatro
consultas configuradas volvieron a responder 200. La consulta detallada de
comunicaciones respondió 503: falta la receta común de permisos de RRHH. No
se repitió la petición para ocultar ese límite.

Faltan el permiso y la receta de RRHH, el recorrido de selección y aviso con
autorización real y la integración. La CI de la cabeza pasó cinco puertas,
pero este corte no se declara listo ni cerrado.

Vuelta atrás: conservar la candidata y la copia anteriores. Si la instalación
se detiene, resolver la causa sobre datos ficticios y preparar otra copia. No
ejecutar DOWN ni reaplicar SQL. Los datos ficticios se pueden recargar solo en
la copia propia; se conservan las guardas y el trabajo de los demás equipos.
