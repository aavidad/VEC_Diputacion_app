# F1-27 A0: consulta del acto de resolución conservado

El expediente puede avanzar después de la resolución sin perder el recibo del
acto registrado en su versión 8. Este corte permite leer ese recibo desde la
versión actual para recuperar el número, la fecha y la referencia de la
resolución. La lectura usa el permiso, el registro de acceso y la auditoría del
detalle de Contratación temporal. Conserva el documento y su huella originales;
no registra otra resolución ni modifica el expediente.

La migración CT261 amplía la lectura instalada de CT69 a las versiones
posteriores. Comprueba la versión 8, su actuación y su recibo frente al
expediente actual. El uso anterior de preparación continúa limitado a las
versiones 7 y 8 por su contrato Go. No hay configuración, actos de
administración ni datos nuevos. No se añaden lecturas ni configuración al arranque. Sin CT261, la lectura
del acto posterior devuelve un fallo de su función. No se ha ensayado aquí
el arranque de una candidata con este corte ni su recorrido en Chrome.

Dirección debe incluir CT261 en la lista SQL de la candidata, ensayarla en la
copia local de la principal, instalar el mismo orden ensayado y reiniciar con
vigilancia. El ensayo debe comprobar una resolución existente en versiones 8,
9 y posterior, un expediente sin acto y un expediente ajeno. El recibo leído
debe conservar la solicitud y la huella del documento original. La prueba Go
focal no acredita instalación ni recorrido en navegador.

Hoy no existe un acto propio que anule o rectifique esta resolución en el
esquema de Contratación temporal. El lector rechaza una historia o un recibo
divergentes; no presenta ese rechazo como prueba de anulación. Cuando exista
un acto de anulación o rectificación, el lector deberá denegar la resolución
afectada. Esa ligadura queda para la tarea aparte de dichos actos. La copia J
y sus recibos quedan intactos.

Ensayo local: CT261 se instala con código 0 en PostgreSQL 18.4 restaurado de
un volcado coherente de J, con sus roles y permisos de base conservados. La
comprobación SQL del contrato y de las historias presentes también termina
con código 0. Es una copia privada sin red, limitada a 2 GiB; J sigue intacta.
Esta prueba no ejecuta una lectura nominal real de versión 9 o posterior.
Las focales Go cubren ese contrato con dobles; el consumo genuino, el
recorrido en Chrome y las revisiones independientes siguen pendientes.
