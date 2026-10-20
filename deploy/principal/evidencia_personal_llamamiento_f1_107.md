# F1-107: consultas de Personal que justifican el llamamiento

Al seleccionar una persona desde la ficha del expediente, la consulta de
Personal comprueba el puesto. El recibo de selección conserva ahora el recibo
y la auditoría de cada página consultada en `procedencia.consultas_personal`.
La lista queda cubierta por el sello del recibo y por la huella del artefacto
que ya guarda Contratación temporal. Ese sello técnico no es una firma de la
persona ni una firma del documento administrativo.

## Activación

CONFIG NUEVA: ninguna. No cambia perfiles, permisos, identidad del puesto ni
su huella. Reutiliza la consulta histórica de Personal y el llamamiento de
Bolsa desde la ficha de Contratación temporal.

1. Preparar una copia nueva desde main con los productores de roles y datos
   del recorrido vigente y Administración. Los datos son ficticios. La
   [guía de copia local, apartado 6](LEEME_copia_local.md) describe las
   admisiones y los actos necesarios: fuente de Organización histórica,
   fuente de origen de Bolsa en su versión 2, perfil de consulta publicado
   por dos administradores y ampliación del perfil de Bolsa de la técnica.
   El catálogo restaurado conserva la versión 1 de la fuente de origen;
   volver a añadir esa misma acción no actualiza sus campos.
2. Instalar la lista común `deploy/principal/lista_sql_main_tras_hz37.txt`,
   con CT245 al final. La migración exige las definiciones instaladas y sus
   permisos exactos; cualquier divergencia impide instalarla.
3. Publicar juntos el binario de esta candidata y sus archivos web, y
   reiniciar la aplicación. La configuración de Personal y del llamamiento
   conserva sus valores anteriores. Si no están activados, la aplicación
   debe seguir arrancando y la selección queda sin servicio.

No se reaplica CT245 ni se deshace la historia. Los recibos anteriores, sin
la lista de consultas, mantienen su representación y su verificación. Para
leer recibos nuevos hay que conservar una versión compatible con CT245.

## Comprobación

1. Con el perfil técnico de Recursos Humanos, abrir un expediente fiscalizado
   del recorrido 4.0 y confirmar la selección desde su ficha.
2. Comprobar en el recibo durable que `procedencia.consultas_personal` contiene
   las referencias originales de las páginas de Personal, en su orden.
   No debe contener nombres ni datos del puesto.
3. Recuperar la selección con su misma clave. Deben mantenerse el recibo,
   las referencias de Personal y el número de registros.
4. Reiniciar la aplicación y PostgreSQL de la copia. Recuperar otra vez la
   misma selección y cotejar las referencias originales.

Las pruebas focales de Go cubren la paginación completa, la ausencia de
referencias, el error de una página y la alteración de una referencia después
del sellado. La prueba SQL `ct245_consultas_personal_canon.sql`
comprueba el canon y las formas admitidas sin datos personales.

El ensayo SQL, las revisiones independientes y el resultado del recorrido se
registran en la PR. Las comprobaciones con material preparado para una prueba
de persistencia no sustituyen la selección realizada desde la pantalla.

## Ensayo del 20 de octubre: avance y comprobaciones pendientes

La unión de código `105f19576`, sobre main `bf055fc99`, conserva CT245 sin
cambios y sus dos revisiones anteriores. La revisión independiente de la
unión confirma que el proveedor nominal conserva el puesto y las referencias
de Personal. Pasan las pruebas focales de Go, con detector de carreras,
`go vet`, las pruebas del conductor SQL y Semgrep, sin hallazgos en seis
archivos. GoSec no señala archivos cambiados, pero su cobertura es parcial
por errores de carga de paquetes.

La copia nueva arranca en 94,18 segundos e instala las 136 migraciones.
La prueba SQL del canon pasa. Sin activar Organización histórica, las
consultas de Contratación y Bolsa responden 200. Administración arranca con
su conjunto 7 y exporta el montaje de roles desde su acuse confirmado.
Desde Chrome, el centro presenta una petición nueva de una persona para el
puesto 9 y otra persona la ratifica; ambas operaciones devuelven su recibo.

Quedan pendientes la actualización gobernada de la fuente de origen de
Bolsa, su ampliación de perfil y la publicación del perfil de Organización
histórica. La copia estándar necesitaba además la provisión existente de los perfiles
fijos de lectura de peticiones y firma del informe. Se aplicó contra sus
preimágenes exactas y se retiraron las variables temporales después de
reiniciar. La entrada de peticiones de Recursos Humanos volvió a responder
200 y el alta del expediente respondió 201.

El expediente nuevo `2026/95107` alcanza la versión 5 por pantalla: análisis,
vía de bolsa, asignación e informe jurídico preparado, con recibos propios.
La firma del informe, la fiscalización y la selección dependen de completar
las etapas de activación. Hasta entonces, la selección desde la ficha y su
recuperación después del reinicio no están acreditadas.
