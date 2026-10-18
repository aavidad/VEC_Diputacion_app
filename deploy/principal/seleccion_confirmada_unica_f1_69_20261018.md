# Una selección confirmada por expediente

F1-69 prepara CT238 para impedir dos selecciones confirmadas en la misma
organización y expediente. Se aplica tanto al llamamiento de una persona como
a la oferta de varias plazas: CT220 conserva una sola selección original para
todas ellas. Las continuaciones recuperan esa selección. CT234 crea referencias
propias para los pasos posteriores y, en este primer corte, no crea otra
selección por cada persona.

## Activación prevista

1. Partir de una copia local de la principal con PostgreSQL 18 y datos
   ficticios. Dirección instala una vez la lista única de 78 migraciones de
   main y después CT238, en su orden. No se reaplican SQL con historia.
2. CT238 bloquea las escrituras de la tabla mientras comprueba que cada
   selección confirmada tiene organización y expediente y que no hay dos para
   el mismo par. Si encuentra una anomalía, se detiene sin borrar historia ni
   crear el índice. Se investiga el origen antes de preparar otra candidata.
3. Comprobar en la copia la selección de una persona, la oferta de varias
   plazas, la recuperación de una selección confirmada pendiente de aviso y
   el rechazo de una segunda confirmación para el mismo expediente. Usar la
   ruta y la pantalla actuales. Comprobar también arranque con y sin la
   configuración existente, reinicio y conservación de la selección original.

La prueba `ct238_seleccion_confirmada_unica.sql` utiliza una selección
singular confirmada de la copia HZ37. Si no existe, se prepara mediante el
recorrido habitual de selección antes de ejecutar la prueba. Sus filas de
comprobación se revierten; no acreditan actos ni consumos de autorización.

El índice único de CT238 cubre solo las filas confirmadas de O6 y usa la
organización y el expediente guardados en la solicitud. También sirve a la
consulta CT231. No añade estados, permisos, funciones ni un lector nuevo.

## Evidencia pendiente

Faltan el ensayo SQL sobre una copia nueva, las pruebas focales, la comprobación
de arranque con y sin configuración y el recorrido tras reiniciar. Hasta
entonces, esta guía no acredita instalación ni funcionamiento del control.

Vuelta atrás: conservar la candidata y la copia anteriores. Si la instalación
se detiene, resolver la causa sobre datos ficticios y preparar otra copia. No
ejecutar DOWN ni reaplicar SQL. Los datos ficticios se pueden recargar solo en
la copia propia; se conservan las guardas y el trabajo de los demás equipos.
