# Retomar el aviso de un llamamiento

Estado: preparado, pendiente de revisión y ensayo final.

La ficha consultaba solo comunicaciones ya anotadas. Si Bolsa había confirmado
la selección y todavía faltaba el aviso, recargar borraba su estado del navegador:
la ficha decía que no había llamamiento y ofrecía seleccionar otra persona.

CT231 añade a la lista existente una fila de selección confirmada pendiente de
aviso. Utiliza ocho campos que esa consulta ya permite leer: organización,
expediente, llamamiento, estado, fecha, versión y tipo y recibo del antecedente.
La versión uno pertenece a la selección real; no representa una comunicación.
No se crean referencias de comunicación, recibos ni respuestas para rellenarla.

La ficha muestra el justificante existente y recupera el formulario de aviso
con esas referencias. Mientras existe una selección pendiente, no ofrece una
nueva selección. Un error, una paginación incompleta o varias selecciones sin
comunicación cierran el siguiente paso. El navegador no usa su almacenamiento
como autoridad ni hace un POST de selección para recuperar el estado.

## Activación

Configuración, permisos y actos de administración nuevos: ninguno. La acción,
finalidad, once campos autorizados y consumidor de autorización existentes se
conservan. Los tres campos exclusivos de comunicación se omiten en la fila nueva.

1. Congelar código, configuración y la copia local de la candidata. Para el
   llamamiento nativo, CT230 y las correcciones de referencias y fechas de
   #1144 deben estar en la composición ensayada. Este corte parte de main y no
   copia esos cambios en su PR.
2. Comprobar sesión, portal y funciones anteriores con la configuración existente
   y con la consulta desactivada. Su ausencia debe cerrar solo el paso afectado.
3. Detener la aplicación e instalar CT231 una sola vez con su lista propia.
   La migración coteja el cuerpo instalado real, propietario y permisos. Una
   diferencia detiene la instalación; no se parchea una migración instalada.
4. Ejecutar las pruebas focales y arrancar la candidata sin instrumentación.
   Abrir la ficha con una selección confirmada sin aviso. Debe mostrar una fila,
   su justificante real y el formulario de comunicación; no otra selección.
5. Recargar y reiniciar aplicación y PostgreSQL. Volver a la ficha: mismos
   llamamiento y justificante, sin segunda selección. Comprobar también la lista
   anterior de comunicaciones y su paginación.
6. Usar el formulario existente de aviso. Tras guardarlo, su comunicación debe
   sustituir a la fila pendiente; no se fabrica una comunicación desde la lectura.
   El registro de aviso no acredita por sí solo entrega de correo.

Comprobar denegación de otro ámbito, campos o filas mezclados, cursor incoherente
y selección ambigua. Se conserva la sesión ligada al certificado y la auditoría
nominal de la consulta. El lector solo consulta tablas de Contratación.

La vuelta atrás usa la candidata anterior y la copia de ensayo conservada.
No hay DOWN, reaplicación, despliegue ni cambios de servicios compartidos.

## Evidencia

Pendientes fuente final, dos revisiones independientes, prueba SQL en la copia,
recorrido en Chrome y reinicio. El defecto se reprodujo en la copia propia con
un llamamiento O6 confirmado y cero comunicaciones, conservando su recibo real.
Este corte no acredita el recorrido completo de FLUJO_APP §4.0.
