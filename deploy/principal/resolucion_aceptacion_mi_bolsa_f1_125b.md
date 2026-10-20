# Resolver una aceptación de Mi Bolsa desde la ficha

Segundo corte de F1-125. Reutiliza el lector del primer corte, el verificador
nominal de Bolsa y el acto existente de revisión de respuesta de RRHH.

## Activación

Aplicar CT249 una sola vez después de CT250 mediante la lista SQL principal.
No añade configuración, acciones ni concesiones. Reiniciar con la configuración
existente. La aceptación debe proceder del llamamiento nativo del expediente,
con una única participación y su respuesta firme; una emisión manual o una
marca de origen por sí sola no habilitan la resolución.

## Recorrido

Desde el caso 4.0, completar la emisión y el contacto efectivo. Antonio acepta
en Mi Bolsa y guarda el justificante. RRHH abre la ficha, revisa respuesta y
plazo y confirma la resolución. El recibo de resolución conserva sus propias
referencias, distintas del justificante de Mi Bolsa. No exige un correo para
una respuesta registrada en el portal.

Reabrir la ficha tras reiniciar aplicación y PostgreSQL y cotejar la referencia
y la fecha de la resolución. Ante una respuesta incierta, reintentar conserva
la misma petición; una resolución confirmada no se sustituye por otra.

## Límites y comprobaciones

El acto consume la autorización en la misma transacción que el asiento y el
recibo, también al recuperar una petición. Bolsa acredita la emisión, la
participación y la aceptación originales. Una restricción impide resolver dos
veces el mismo llamamiento entre las vías de correo y portal. El plazo queda
como revisión de RRHH; no se presenta como un cálculo automático.

La lectura recupera solo referencia de resolución, recibo, fecha y respuesta.
La propuesta de formalización que consume esa pareja corresponde a F1-127.
El ensayo de este corte debe acreditar el recibo HTTP 201 y su recuperación;
esta guía no acredita por sí misma que el recorrido se haya completado.
