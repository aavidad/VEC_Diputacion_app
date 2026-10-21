# F1-27 — Nombre del centro en los documentos

El etiquetador documental reconoce la clave canónica del centro de la petición
y obtiene su nombre de la misma publicación de Organización ya cargada para
el catálogo. Conserva también las etiquetas del alta directa.

`centro-520` sigue siendo la referencia de la petición. No se transforma en
`centro:rpt:520` ni se añade al catálogo de alta. Solo cambia su nombre en la
representación documental; no cambia el expediente, sus permisos o sus recibos.
Sin la publicación de Organización no se inventa una etiqueta.

CONFIG NUEVA: ninguna. SQL: ninguno. Datos o actos de activación: ninguno.
La revisión integrada se instala por la receta existente y su reinicio vigilado.
Los originales ya custodiados conservan sus bytes. El ensayo de documentos
nuevos de J espera también los otros cortes autorizados; no se afirma aquí
que esos documentos estén completos o firmados.

Las pruebas focales comprueban ambos nombres, una referencia desconocida,
la ausencia de configuración y que el etiquetado no cambie las referencias
admitidas por petición y alta ni publique un alias en el catálogo.

La focal de catálogo y entrega pasa con `go test -p 2` (0,072 segundos).
Semgrep local: cuatro reglas, dos archivos, sin hallazgos. GoSec oficial
completa el paquete: 81 avisos en archivos idénticos a la base y ninguno en
los archivos cambiados. La revisión independiente y CI figuran en la PR.
