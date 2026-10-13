# Incorporar una solicitud admitida a la bolsa existente

B112 amplía la incorporación existente para aceptar las dos vías autorizadas:
constitución original B7 y carga CONVOCA confirmada B79. Se instala una vez,
después de B97; no altera recibos anteriores ni crea recibos CONVOCA.

RRHH abre la solicitud de Antonio en «Solicitudes y admisión», registra la
admisión y pulsa «Incorporar a la bolsa». Selecciona expresamente el acta
existente mediante el campo del formulario y confirma. La participación,
categoría y vigencia proceden de Bolsa; el candidato procede del vínculo
vigente de la misma persona. Una selección ajena se rechaza.

Para el recorrido conservado, la categoría es `categoria:rpt:administrativo` y
el acta que contiene la participación actual de Antonio es
`acta:importacion-convoca:d421e67449e4595c87e602d68a749e561ee0c2767b175fd93e8e57b117787ad6`.
Ese identificador es un dato operativo para dirección, no prueba de origen.
El prefijo del acta no decide qué circuito la constituyó.

B79 confirma lote, original, constitución, vínculos y recibo en una transacción,
y su recibo es inmutable. Si existe recibo B79 para el acta, B112 coteja todas
sus referencias con la constitución. Si no existe, exige la constitución B7
completa: bolsa vigente, canónicos y huellas, instantánea, entrada y vínculo
exacto de candidato. Conserva la política de asociación de la convocatoria,
la autorización V3, la versión esperada, auditoría, historia, evento y recibo
de incorporación en la misma transacción. Reintentar usa la misma clave y
material.

CONFIG NUEVA: ninguna. Lista SQL:
`lista_sql_codexo_inscripcion_incorporacion_b7_20261013.txt`.
La pantalla, cliente y acto B97 ya existen. Dirección prueba la incorporación
con el usuario RRHH después de instalar y conserva el recibo; no repetir el
alta del aspirante ni sustituir su identidad.
