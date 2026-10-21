# F1-27: guardar la propuesta tras la aceptación de Mi Bolsa

En J, el expediente alcanzó la versión 6 y RRHH confirmó la aceptación de
Mi Bolsa con su recibo. «Guardar la propuesta» no enviaba ninguna petición:
el envío exigía una comunicación de CT seleccionada, aunque el antecedente
era la resolución del portal. Al seleccionar una comunicación, se borraba
la preparación de la propuesta y desaparecía el botón.

La propuesta del portal conserva ahora su resolución, su recibo y la
preparación de documentos al consultar una comunicación de CT. Se muestra
una sola vez. Su envío no depende de la selección ni de la consulta de una
respuesta por correo de CT. La excepción solo alcanza a la propuesta del
portal con aceptación resuelta y versión admitida. Las demás operaciones
conservan sus guardas. El servidor sigue validando la petición y su permiso;
un rechazo se muestra sin afirmar que la propuesta se guardó.

Se reutilizan el formulario, el contrato HTTP, los catálogos y las pruebas de
F1-127 y de recuperación de comunicaciones. No hay textos visibles nuevos.
El defecto del acceso desde CT que abre el asistente de otro llamamiento y
la ocultación del contacto tras recuperar una comunicación siguen pendientes;
no se corrigen en este corte.

## Activación y comprobación

CONFIG NUEVA: ninguna. SQL: ninguno. Actos y datos nuevos: ninguno.
Solo cambian dos archivos JavaScript servidos por la aplicación. No cambia
Go ni se necesita otro binario para ejecutar esta lógica. El servidor conserva
los archivos con marca de versión en una caché de proceso
(`internal/app/server/estaticos_version.go`), por lo que una recarga del
navegador puede seguir recibiendo los archivos anteriores. Es necesario
reiniciar la aplicación con los archivos de la revisión integrada y comprobar
la huella web servida; la base no requiere cambios.

J conserva su binario y revisión originales hasta que dirección concrete la
actualización de sus archivos servidos y el reinicio. No se cambiarán los
metadatos del gestor para saltar su comprobación de revisión y huella. La
vuelta atrás consiste en arrancar el mismo binario con los archivos web
anteriores, conservando el mismo expediente y sus recibos.

Después de activar: abrir el expediente de J, comprobar la aceptación ya
resuelta, guardar la propuesta y conservar su recibo. Seleccionar la
comunicación debe mantener un único formulario. El recorrido en Chrome aún
está pendiente; esta nota no acredita el paso 12 ni el reinicio de J.

Las 73 pruebas focales cubren ambos caminos, la conservación del antecedente
del portal frente a campos ajenos del formulario, confirmación y rechazo del
servidor, independencia ante lecturas CT rechazadas o no disponibles, ausencia de propuesta sin resolución y bloqueo de un nuevo envío
tras confirmar. La base de tamaño del formulario pasa de 1.388 a 1.394 líneas
reales por conservar esa preparación y distinguir su antecedente.
