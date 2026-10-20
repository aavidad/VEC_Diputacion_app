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

1. Preparar una copia propia con los productores privados de roles y datos
   del recorrido vigente. Los datos son ficticios.
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
