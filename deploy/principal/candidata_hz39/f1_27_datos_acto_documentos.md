# F1-27 — Número y fecha del acto en los documentos

Este corte aporta los campos `resolucion_numero` y `resolucion_fecha` al mapa
común de PDF y Word. Proceden de la solicitud validada antes de registrar la
resolución y, después, del recibo confirmado recuperado por la lectura nominal
de Contratación. No proceden del navegador al descargar el documento.

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
3. Antes del acto, el borrador puede mostrar esos campos vacíos: el número y
   la fecha se introducen en el formulario de registro. El renderizado del
   acto usa exactamente esa solicitud validada para calcular su huella.
4. Recuperar el recibo y generar PDF y Word. El original v7 nuevo debe tener
   los mismos bytes antes y después del registro. Su preparación rechaza un
   PDF que no coincida con la huella conservada por el acto.

No se cambia ninguna configuración de arranque. Las plantillas que no usan
los campos nuevos conservan su texto; este corte no publica una plantilla ni
acredita el arranque o las descargas de una copia actualizada.

## Límites del recorrido

La lectura existente de preparación admite las versiones 7 y 8. Este corte
completa los documentos en ese tramo; no amplía la lectura SQL para actos
posteriores. Las demás versiones conservan su representación anterior. La
custodia sigue recuperando los bytes de los originales ya guardados.

No hay identidad de la persona en este corte. La lectura nominal propia de
Bolsa y el dato del centro son los otros dos cortes encargados por dirección.
No se preparan ni firman documentos con esos contenidos pendientes.

La resolución ficticia ya registrada en J conserva su huella anterior.
Dirección autorizó rehacer ese dato de la copia cuando los dos cortes de
código estén integrados. No se implementa una rectificación del producto ni
se sobrescriben huellas o publicaciones antiguas.

## Verificación

Las pruebas focales comparan el PDF original antes y después del acto,
comprueban el contenido combinado de PDF y Word y rechazan un original con
huella distinta. También comprueban el cruce de expediente, cancelación,
fallo de lectura y ausencia de acto. No acreditan firmas ni actos en Chrome.
El recorrido de J y su reinicio final siguen pendientes.

Pruebas locales: focales Go con detector de carreras verdes en puertos,
representación, generación publicada y composición. Semgrep local: cuatro
reglas y doce archivos, un aviso en una línea idéntica a la base, ninguno
nuevo. GoSec oficial completó los cuatro paquetes: 114 avisos en archivos
idénticos a la base, ninguno en los cambios. La revisión independiente y
la puerta completa de CI se registran en la PR.
