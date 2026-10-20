# Consultar la aceptación de Mi Bolsa en la ficha

Primer corte de F1-125, separado por dirección el 20/10 a las 12:56.

La ficha muestra la aceptación firme, la fecha de emisión, la fecha de respuesta
y el justificante original de Mi Bolsa cuando acredita su origen en una
selección confirmada de Contratación. Este corte es de consulta; la resolución
y su recuperación corresponden al segundo corte de F1-125.

## Activación

Usar la fábrica de `LEEME_copia_local.md`, apartados 2 y 4, con puertos y
material propios. La lista SQL ordenada incorpora CT250 después del lector
CT201 vigente. No hay variables ni concesiones nuevas. El arranque conserva
las configuraciones existentes; sin la ampliación SQL, la marca se interpreta
como falsa y la consulta anterior sigue disponible.

Para el caso 4.0 se reutiliza la firma gobernada de los apartados 6 y 11 de
esa guía, con recibos propios. El expediente debe conservar su necesidad RPT.
RRHH llama desde la ficha a la siguiente persona de la bolsa constituida;
Antonio acepta desde Mi Bolsa y conserva el justificante. Si aún no está vinculado, elija «Llamamiento de este expediente» y pulse
«Vincular». Se conserva el recibo propio de ese vínculo. La emisión nativa
se coteja por el identificador original del expediente; las emisiones manuales
conservan la referencia visible. No se cambia la configuración de Bolsa.

## Comprobación

Abrir la ficha y comprobar las fechas y el mismo recibo de respuesta que en
Mi Bolsa. Tras reiniciar aplicación y PostgreSQL, deben conservarse. Una
emisión manual, una respuesta provisional o varias participaciones no se
presentan como aceptación de ese origen.

La consulta consume su autorización nominal antes de leer Bolsa. El origen se
acredita enlazando la selección propia, su emisión y recibo exactos y la única
participación que devuelve la función nominal de Bolsa. La versión fiscalizada
es la histórica de la selección; avanzar el expediente no altera ese origen.
La marca no autoriza una escritura ni sustituye la revisión de RRHH.
