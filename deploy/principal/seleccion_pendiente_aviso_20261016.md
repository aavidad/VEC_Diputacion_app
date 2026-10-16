# Retomar el aviso de un llamamiento

Estado: comprobado en copia local; pendiente de integración por dirección.

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
2. Comprobar sesión, portal y funciones anteriores antes de CT231. Este corte
   no añade configuración; la aplicación debe arrancar también con el esquema
   anterior. Un fallo de consulta debe cerrar solo el paso afectado.
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

Candidata conjunta congelada `48d21082c`: código de #1144, #1170 y este corte,
sin instrumentación ni sustituciones de funciones. El binario no cambió al
integrar el reintento web. SQL CT231 definitivo SHA256
`6789d0c51233bb7b8728d233dfd5dbe747f4535a13464fc869a404d20cb394e3`;
función resultante `2037b0497f2c5bec4db4940f4079d1c44976fb774cff67dad4ca3063defa80f9`.
Dos revisiones independientes favorables de SQL y seguridad; revisión de
usabilidad de código y capturas, y revisión focal del reintento.

En la copia propia PostgreSQL 18, CT231 se instaló una vez y su focal pasó.
Un primer intento había revertido toda la transacción por una firma de control
incorrecta; se corrigió la fuente y se restauró la copia fría antes del ensayo
final. La aplicación, sesión y portal devolvieron 200 antes y después de CT231,
y tras reiniciar aplicación y PostgreSQL.

Chrome del sistema, expediente 2026/94011:

- Tras recargar y reiniciar, la ficha recuperó una selección pendiente y su
  justificante original, con versión uno y sin botón para seleccionar otra.
- El primer GET recibió un 503 transitorio y el único reintento devolvió 200.
  Solo se reintenta la consulta inicial con ese error validado; no un rechazo
  de acceso, una página intermedia ni una respuesta inválida.
- Desde «Continuar con el aviso», la confirmación del formulario produjo un
  registro real de aviso y HTTP 201. Después del segundo reinicio, el GET
  devolvió una comunicación de once campos con las referencias originales de
  comunicación y recibo; la fila pendiente había desaparecido. La ficha abrió
  la sección de respuesta; no se probó anotar una respuesta ni se creó otra
  selección.
- Se conservaron un origen, una confirmación nativa y la huella del recibo y
  artefacto O6 `e623c8a192757fc29d7f419dfd5ddc04c7fdd3d441beed40cdc224988b41c8ac`.
- La sesión de Intervención abrió con 200; esa consulta devolvió 403 sin datos
  y dejó un asiento de denegación de frontera. Cuatro lecturas autorizadas
  comprobadas dejaron cuatro consumos y cuatro asientos durables de auditoría.
- Un expediente anterior de la misma organización devolvió dos páginas 200 de
  una comunicación cada una, con cursor en la primera y cierre en la segunda.
  La copia no contiene un expediente que mezcle ambos tipos de fila. El orden
  y cursor mixtos se comprobaron con valores declarados y ROLLBACK, sin crear
  comunicaciones, firmas ni recibos del producto.

Se revisaron castellano e inglés a 1440 y 390 píxeles, teclado y contenido al
200 %, sin desbordamiento horizontal. Al cambiar de idioma una consulta recibió
403 y mantuvo cerrado el paso; recargar en inglés permitió consultar con 200.
Ese rechazo transitorio queda señalado, sin reintentar ni eludir la denegación.
Las capturas móviles están desplazadas: no acreditan el enfoque del encabezado
bajo la barra fija ni la visibilidad del último botón al 200 %.

Focales Go de puertos y adaptador, 115 comprobaciones web, grafo y caché de la
composición final favorables. `gopls`, Semgrep local y gosec sobre lo cambiado:
sin hallazgos nuevos ni errores de carga; gosec conserva los 35 hallazgos
anteriores de los paquetes examinados. La CI ejecuta la puerta completa.

## Límites

Anotar el aviso no envía correo ni acredita recepción. Este corte protege la
recuperación desde la ficha; un POST manual con otra clave no tiene todavía una
unicidad global por expediente. La lista cierra ante varias pendientes.
No acredita el recorrido completo de FLUJO_APP §4.0 ni autoriza despliegues.
Las evidencias con datos, sesiones y material de la copia se conservan fuera de
Git; aquí solo constan resultados y huellas.
