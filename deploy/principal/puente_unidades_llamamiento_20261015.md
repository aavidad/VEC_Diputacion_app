# Unidad del puesto y unidad gestora del llamamiento

F1-21 corrige la lectura del puesto que hace la ficha de Contratación antes de
ofrecerlo desde una bolsa constituida. Antes enviaba a Personal la unidad que
gestiona el expediente. Ahora toma la unidad del puesto de la RPT publicada
validada y conserva la unidad gestora para la autorización de Contratación y
el origen del llamamiento en Bolsa. No cambia la pantalla ni los permisos.

Reutiliza el contrato de la carga RPT de Personal: el código de centro publicado
se corresponde con `unidad:centro:<codigo_en_minusculas>`. La petición debe
conservar la referencia y huella de esa publicación. La consulta histórica
comprueba además el puesto individual, su unidad, vigencia, unicidad y todas
las páginas del mismo corte. La referencia del puesto se coteja con el formato
hexadecimal canónico de la importación de Personal, conservando sus dígitos y
separadores. Un centro no mapeable o un puesto ajeno o ambiguo
se rechazan. No se amplía la consulta a todo el organismo.

## Activación

No hay SQL ni configuración nueva. Instalar primero la consulta histórica de
la PR #1150 con su material nominal y el cliente mTLS existente de Contratación.
Preparar el binario y conservar el anterior para la vuelta atrás. Instalar con
la aplicación detenida y vigilar el reinicio. Los datos del puesto y la unidad
se conservan: no se modifican para hacer coincidir puesto y unidad gestora.

## Comprobación

Usar la misma petición del caso de `FLUJO_APP.md`, apartado4.0, vinculada a la
necesidad de la RPT. La unidad gestora de RRHH y la unidad del puesto del centro
son distintas. Desde la ficha, abrir «Llamamiento y comunicación» y realizar la
selección por la operación existente. La lectura de Personal debe usar la
unidad del puesto y producir su recibo real; Contratación y Bolsa conservan la
unidad gestora y sus permisos. La continuidad o parada de la selección depende
del orden y de las reglas de la bolsa; este corte no modifica esas reglas.

Con la consulta histórica desactivada o su configuración ausente, la aplicación
sigue arrancando y la selección queda no disponible por la vía existente. Un
puesto ajeno o ambiguo y un centro no mapeable no crean un llamamiento. Conservar
recibos y comprobar el mismo expediente después de reiniciar.

Las pruebas focales Go cubren las dos unidades distintas y los rechazos. El
recorrido real, la revisión independiente y la CI de la cabeza final están
pendientes. El ensayo usa una copia local con datos ficticios.
