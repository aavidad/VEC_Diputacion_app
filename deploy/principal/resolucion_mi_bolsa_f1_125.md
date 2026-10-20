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

La resolución comprobará sesión, permiso, expediente, selección, participación y
respuesta firme del portal. La comprobación y el efecto deberán confirmarse
en la misma transacción. La historia de las respuestas de correo se conserva.
La entrega incluirá el guion de activación y las revisiones independientes.

## Preparación del recorrido

Se reutilizan la fábrica de copias, la preparación de Antonio y la etapa de
firma descritas en `LEEME_copia_local.md`, apartados 2, 6, 8 y 11. No se activa
`--portal-externo` para este recorrido de Mi Bolsa. El expediente debe ser nuevo,
de vacante, con la necesidad de RPT guardada; los expedientes antiguos de la
semilla no acreditan esa relación.

Después de completar los pasos anteriores del caso 4.0, RRHH usa «Llamar a la
siguiente persona» desde la ficha. Antonio acepta desde Mi Bolsa. La respuesta
se consulta desde la participación de esa emisión vinculada al expediente;
no se elige por nombre ni por su posición entre las respuestas.

La verificación existente de Bolsa comprueba la emisión, la selección de la
bolsa constituida, la participación y el recibo de aceptación firme. RRHH debe
revisar expresamente la respuesta y el plazo. Esta vía no acredita un cálculo
automático del plazo: el registro distinguirá esa revisión de una comprobación
automática de vigencia.

El contrato y el guion de instalación se completarán con la cabeza ensayada.
La propuesta de formalización posterior aún exige el antecedente de correo;
ese límite se ha comunicado a dirección para fijar su continuidad.
