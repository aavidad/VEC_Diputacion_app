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
de la versión anterior al arreglo de pantalla. Tras reiniciar aplicación y
PostgreSQL se conservaron una única respuesta, su fecha y su justificante; Chrome
volvió a mostrar «Acepto» y «Respuesta firme». El cotejo independiente de esos
resultados consta en la [entrega parcial de F1-27](https://github.com/aavidad/VEC_Diputacion_app/pull/1308).
La prueba del formulario corregido y su recuperación deben constar aparte,
sobre su revisión exacta.

## Ensayo del formulario corregido, 20/10

Código ensayado: `5812c4d2e`, con main `3c643ce34` integrado. La copia se creó
en 101,17 segundos con la lista SQL completa y los módulos internos de la receta
común. No se añadió configuración propia de esta entrega.

| Actuación desde Chrome | Resultado |
| --- | --- |
| Emitir el llamamiento para la primera persona disponible | 201, una emisión |
| Registrar «Contactado» desde el formulario genérico | El servidor devuelve 201; el navegador recibe un 503 simulado después del registro real |
| Pulsar «Reintentar el mismo registro» | 200 con el mismo cuerpo, clave, justificante y fecha; una sola llamada registrada |
| Aceptar desde Mi Bolsa con el certificado de la persona | 201 con `estado=firme`; una sola respuesta |
| Reiniciar aplicación y PostgreSQL y consultar | Salud, Mi Bolsa, contactos y catálogo de Contratación 200; mismos justificantes y fechas |

El contacto se guardó a las `09:55:40.348819Z` y la aceptación a las
`09:56:32.849405Z`. El recuento tras reiniciar conservó una emisión, un contacto
telefónico y una respuesta; existía además el contacto del correo automático.
El formulario no envió una fecha del navegador. Tras la respuesta incierta
conservó los cuatro campos bloqueados y permitió recuperar el mismo registro.

La consulta conserva `modo=firme` y `en_revision=true`. Son dos datos distintos:
la respuesta es firme; el indicador de revisión comprueba que todavía no hay una
actuación posterior que cambie la situación de la participación. No se ha
fabricado ese cambio ni un vínculo con Contratación en esta copia. La resolución
de la aceptación desde la ficha queda en F1-125; no se acredita con este ensayo.

Chrome mostró el formulario y su botón completos a 1440 y 390 píxeles, sin
desborde de la página. El montaje DOM con los recursos reales pasó 14 casos:
errores diferenciados, recuperación del mismo envío y bloqueo cuando falta la
consulta telefónica. Pasaron 38 pruebas focales de la interfaz y una comprobación
focal adicional del estilo común. La copia y sus procesos se retiraron después
de conservar los resultados, sin repetir actuaciones para recuperar recibos.

La revisión posterior detectó también el éxito HTTP cuyo cuerpo llega truncado
o sin justificante válido. Se trata como resultado incierto y conserva el mismo
comando y clave. Dos pruebas focales seleccionadas comprobaron JSON rechazado,
justificante ausente o vacío y el reintento con la ficha repintada; las otras 33
quedaron fuera del filtro. No se repitió el recorrido real de pérdida del 503.
