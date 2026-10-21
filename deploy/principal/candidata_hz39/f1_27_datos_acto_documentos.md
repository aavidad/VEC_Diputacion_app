# F1-27 — Número y fecha del acto en los documentos

Este corte aporta los campos `resolucion_numero` y `resolucion_fecha` al mapa
común de PDF y Word. Al registrar la resolución salen de la solicitud ya
validada. Después salen del recibo del acto, que se lee con la consulta del
acto de Contratación (CT261, fusionada en main) en cualquier versión posterior
a la 8. Nunca salen del navegador al descargar el documento.

La propuesta ya aporta `propuesta_actuacion` y `propuesta_fecha`. No tiene un
número oficial propio; ni su secuencia ni su referencia se presentan como tal.
Los textos y la combinación de campos siguen en el catálogo de plantillas.

## Activación

CONFIG NUEVA: ninguna. SQL a instalar: ninguno. Concesiones nuevas: ninguna.

1. Instalar la revisión integrada por dirección mediante la receta existente.
2. Publicar por la operación administrativa de plantillas una nueva versión
   que combine `{{resolucion_numero}}` y `{{resolucion_fecha}}`, en lugar de
   dejar el número como texto fijo. Conservar las publicaciones anteriores.
   El catálogo de formalización y las descargas deben usar esa publicación.
3. Antes del acto, la descarga del borrador deja esos campos vacíos y no
   consulta nada: el número y la fecha se escriben en el formulario de
   registro. El renderizado del acto usa esa misma solicitud validada para
   calcular su huella.
4. Recuperar el recibo y generar PDF y Word. El original firmable de la
   resolución exige que el acto exista y que su PDF tenga exactamente la
   huella que guardó el acto; si no, no se prepara.

No se cambia ninguna configuración de arranque. Las plantillas que no usan
los campos nuevos conservan su texto; este corte no publica una plantilla ni
acredita el arranque o las descargas de una copia actualizada.

## Límites del recorrido

Solo se lee el acto cuando hace falta: si la plantilla publicada combina el
número o la fecha, o si se prepara el original de la resolución. Las
plantillas anteriores de los demás documentos no añaden ninguna lectura. El
original de la resolución se prepara desde la versión 7 que fijó el acto y
coteja su huella también con plantillas anteriores; si la plantilla o los
nombres cambiaron después del acto, el original no se prepara. La custodia
sigue devolviendo los bytes de los originales ya guardados.

No hay identidad de la persona en este corte. La lectura nominal propia de
Bolsa y el dato del centro son los otros dos cortes encargados por dirección.
No se preparan ni firman documentos con esos contenidos pendientes.

La resolución ficticia ya registrada en J conserva su huella anterior.
Dirección autorizó rehacer ese dato de la copia cuando los dos cortes de
código estén integrados. No se implementa una rectificación del producto ni
se sobrescriben huellas o publicaciones antiguas. Con este corte, el original de esa
resolución de J no se prepararía si su PDF ya no coincide con la huella del
acto (por ejemplo, por el nombre del centro de #1433); hay que rehacer los
pasos 13 y 14 como se autorizó.

## Verificación

Las pruebas focales comprueban:
- el PDF original es igual antes y después del acto;
- PDF y Word llevan el número y la fecha;
- se rechaza un original con otra huella o de otra versión;
- el original de la resolución sin acto no se prepara;
- tras la versión 9 se lee el acto, y la descarga previa al acto no lo lee;
- las plantillas anteriores no añaden lecturas;
- también el cruce de expediente, la cancelación y el fallo de lectura.

No acreditan firmas ni actos en Chrome. El recorrido de J y su reinicio final
siguen pendientes.

Pruebas locales: go test, -race y go vet de puertos, representación,
generación publicada, aplicación y composición, en verde. Semgrep local sin
hallazgos nuevos. GoSec completo no carga con Go 1.26 y queda omitido.
