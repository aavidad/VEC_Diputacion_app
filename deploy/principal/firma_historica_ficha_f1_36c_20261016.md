# F1-36C: consulta de firmas anteriores desde la ficha

Guion local: C está en rama, sin ensayo conjunto ni recorrido en Chrome.
No añade SQL: usar A (AD286 → CT228 → AUT103) y B (AUT102 → CT229) solo
si faltan en la copia. No ejecutar DOWN ni repetir historia.

## Activación gobernada

Partir de los recibos y ámbitos propios de la copia. Publicar por el gobierno
de cargos la versión de consulta que permite el campo exacto; después hacer
los actos nominales de retirada y otorgamiento que corresponden a las cuatro
personas de consulta. Conservar los permisos de cuadro y descarga. Actualizar la selección de consulta en `identidad/firma-vec.json` con los
perfiles devueltos por la preparación, con reinicio vigilado. La
asignación propia de firma solo restringe la lectura ya autorizada de la
ficha; no añade perfiles ni cambia el actor. Comprobar vigencia, organización
y unidad históricas. Guardar materiales y conexiones fuera de Git.

La ficha ampliada debe traer juntos `selectores_recuperacion_firmas` y
`recibo_selectores_recuperacion_firmas`, incluso con lista vacía. La ausencia
de solo uno falla; la ficha heredada omite ambos. El máximo es 128 firmas,
sin selector comodín. El panel
solo ofrece una consulta al pulsar «Consultar firma» junto al documento; no
hace peticiones por cada fila al abrir la ficha. La recuperación vuelve a
autorizar y registra cada acceso. La vista enseña fecha, estado histórico y
recibo del registro original; no enseña identidad canónica ni huellas.

## Comprobación pendiente

En una copia sintética, consultar una firma con asignación vigente. Cotejar
recibo de ficha, anexo y recibo original. Una lista vacía no ofrece firmas.
Un campo aislado, otra unidad, perfil retirado o selector alterado deben
rechazarse sin perder la ficha.
Al cambiar de expediente, abortar y limpiar. Si faltan textos o cliente, el
error queda en el panel y la ficha sigue usable.

Arrancar con la ampliación publicada y sin ella; comprobar también la ficha
y los PDF anteriores. Repetir después de reiniciar aplicación y PostgreSQL.
Recorrer en Chrome, teclado, zoom y móvil, con revisión independiente de
usabilidad y pruebas focales del código final. C no cierra F1-36 sin el caso
de `docs/producto/FLUJO_APP.md` §4.0 y las puertas de dirección.

La integración posterior del límite de 2 MiB (#1163) debe conservar los
contratos Go y JavaScript de este corte y probarlos juntos en una candidata.
