# Resolver una aceptación recibida en Mi Bolsa

Tarea F1-125, asignada por dirección el 20/10 a las 11:56. Estado: diagnóstico
confirmado; implementación, instalación y recorrido pendientes.

La persona acepta desde Mi Bolsa y conserva el recibo de su respuesta firme.
RRHH debe ver ese recibo en la ficha del expediente y resolver la aceptación
sin aportar un correo. La respuesta recibida por correo conserva su formulario
y su justificante propios. No se crea un correo ni un justificante ficticio
para representar una respuesta del portal.

## Bloqueos encontrados

- El formulario y su contrato exigen referencia de correo, fecha de recepción
  y huella del archivo `.eml` antes de habilitar la resolución.
- El servidor exige el justificante de esa respuesta antes de comprobar la
  respuesta firme de Bolsa. Quitar los campos del formulario no basta.
- La tabla de resolución exige una referencia a la respuesta recibida por
  correo. El recibo de Mi Bolsa necesita un contrato de persistencia propio;
  no puede guardarse bajo el nombre de un correo ni saltarse esa relación.
- La lectura nativa de la respuesta usa el vínculo entre el expediente y la
  emisión de Bolsa. Hay que conservar su relación con la selección confirmada
  desde la ficha, la persona llamada y el recibo original.

Referencias de código: `contrato-llamamiento.js`, `formulario-llamamiento.js`,
`contratacion_temporal_comunicacion_llamamiento_desarrollo.go`,
`resultado_emision_ct_b98_composicion.go` y la tabla
`resolucion_manual_respuesta_rrhh` de Contratación temporal.

## Criterio de entrega

En una copia propia: abrir el llamamiento desde la ficha por la selección de
la bolsa constituida; Antonio acepta desde Mi Bolsa; RRHH consulta su respuesta
y resuelve desde Chrome con 201. Tras reiniciar aplicación y PostgreSQL,
conservar la misma respuesta, resolución, fechas y recibos, sin duplicados.

La resolución comprobará sesión, permiso, expediente, comunicación, persona y
respuesta firme del portal. La comprobación y el efecto deberán confirmarse
en la misma transacción. La historia de las respuestas de correo se conserva.
La entrega incluirá el guion de activación y las revisiones independientes.
