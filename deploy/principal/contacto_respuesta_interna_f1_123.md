# Contacto y respuesta en Mi Bolsa interna

Tarea F1-123. El correo enviado avisa del llamamiento; no acredita un contacto
efectivo. Con las reglas vigentes, RRHH registra la llamada y su resultado antes
de que la persona pueda responder desde Mi Bolsa. Se reutilizan las operaciones
de contacto, respuesta y consulta del vínculo de Contratación existentes.

## Activación

Instalar los recursos web de esta revisión con la aplicación correspondiente,
siguiendo `LEEME_copia_local.md`. No hay migraciones, variables nuevas ni cambios
del catálogo de reglas. La copia debe tener el portal interno y los permisos de
RRHH y de la persona candidata ya activados. La entrada del aspirante externo
es otro recorrido.

La consulta de intentos anuncia si se puede registrar el teléfono con fecha del
servidor mediante `registro_telefono`. La pantalla debe respetar ese contrato.
No se convierte un correo enviado en contacto efectivo ni se guarda una respuesta
en nombre del aspirante para desbloquear el recorrido.

## Recorrido

1. RRHH emite el llamamiento de una bolsa constituida y conserva su justificante.
   Comprueba la persona, la participación y el vínculo con el expediente.
2. Tras la llamada ficticia declarada en el ejercicio, abre la ficha de la persona
   y registra el resultado «Contactado» para ese llamamiento. Conserva el
   justificante de contacto y la fecha guardada por VEC.
3. La persona entra con su propio certificado en Mi Bolsa interna. Comprueba el
   llamamiento abierto y su fecha límite, pulsa «Aceptar» y conserva el justificante.
4. RRHH vuelve a Contratación y comprueba que el vínculo de la misma emisión
   muestra la aceptación confirmada. Esto no acredita por sí solo la resolución
   ni la propuesta de nombramiento: sus antecedentes deben estar completos.
5. Reiniciar aplicación y PostgreSQL con la misma configuración. Consultar los
   mismos contactos, respuesta y vínculo, comprobando referencias, fechas y
   justificantes. La recuperación no requiere registrar otro contacto ni responder
   otra vez.

## Comprobación del fallo

El formulario genérico enviaba una fecha del navegador y entraba por una operación
antigua incompatible con uno de los resultados de control del catálogo vigente.
El registro telefónico con hora del servidor ya admitía esas reglas. Además, el
cliente presentaba todos los errores HTTP 409 como conflictos de clave, aunque
el código recibido fuese `contacto_no_valido`. El formulario estaba fuera del
panel de la ficha y su botón podía quedar cubierto por el resumen lateral.

En la copia de F1-27 basada en `fb5200b68`, el 20/10 se observó el rechazo 409
del formulario genérico. El formulario telefónico específico devolvió 201;
Mi Bolsa interna permitió responder con 201 y estado firme, y el expediente
2026/93001 mostró la aceptación confirmada de la misma emisión. Son resultados
de la versión anterior al arreglo de pantalla. La prueba del formulario corregido
y su recuperación deben constar aparte, sobre su revisión exacta.
