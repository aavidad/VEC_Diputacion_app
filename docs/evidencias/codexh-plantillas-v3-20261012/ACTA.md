# Ensayo local de plantillas CT v3, 12 de octubre de 2026

Código de operación revisado: `79448436a6c7b31282f8b9e999f85a840ef1eea0`.
GO independiente SQL/seguridad, P0/P1/P2=0. El primer candidato permitió
combinar argumentos de comprobación/escritura; se corrigió la aridad de shell
y binario y se probó su rechazo antes de conectar.

PostgreSQL 18.4, copia privada de la base sintética final y apagada de K
(`p46_revision/pgdata`), contenedor H `--rm --restart=no --memory 2g` con datos
en disco. El tar inicial HZ21 no superó el preflight del binario actual; se
conservó como evidencia y no se reconstruyó la base ni se hizo DOWN. La copia
final conserva la publicación base v1 con huella `5559604b7ac9aa1810f52859f38527ec42d3ded8b91b91c8f58b75772b06e263`.
Su recibo de provisión es distinto del de cidonia; el guion compara contenido,
no una identidad artificial del contenedor o un recibo específico.

`--comprobar`: transacción SERIALIZABLE READ ONLY, 14 ms. Comparación exacta
del hash de todas las filas de historia y recuentos de eventos/auditoría antes
y después: idénticos. Rechaza cualquier catálogo distinto del de #1065.

`--aplicar PUBLICA-PLANTILLAS-V3`: 14 ms, v3/revisión1, 10 documentos más
etiquetas. Dos filas de historia totales (v1 y v3), un evento y dos auditorías
de provisión. La fila v1 completa conserva su hash previo. Es una operación
de datos ficticios autorizada, sin consumo V3, otra identidad o firma inventados.

Chrome del sistema (`/usr/bin/google-chrome`) mediante playwright-cli,
certificado RRHH sintético y aplicación compilada de la rama, HTTPS local:
lista → filtro `2026/CT-000132` → ficha en nombramiento v7 → Resolución → PDF.
Antes de publicar: HTTP 200, catálogo v1, 50.727 bytes y las marcas antiguas.
Después de publicar y actualizar la ficha: HTTP 200, catálogo v3, 51.804 bytes,
sin «borrador de desarrollo», «Datos sintéticos» ni «modelo de ejemplo».
La interfaz rechaza mezclar la lista v1 ya abierta con una descarga v3; se
actualiza la ficha como indica LEEME. No se cambió ese control.

Reinicio completo de aplicación/PostgreSQL y nueva descarga desde el mismo
botón: HTTP 200, PDF idéntico byte a byte, mismas cabeceras de catálogo y
procedencia. Replay después de reinicio: mismo recibo/auditoría, sin otras
filas ni eventos. Tiempos servidor del PDF v3: 171,9 ms y 220,4 ms; base 43,0
y 42,9 ms, 19 consultas fijas. La base contiene 74 peticiones; estas medidas
son muestras de descarga, no un p95 con miles de expedientes.

La captura [ficha-resolucion.png](ficha-resolucion.png) muestra el botón usado
(y el estado «Borrador sin firmar»); el texto limpio no equivale a firma.
El clon devolvió incidencias previas en otras capacidades opcionales de la
ficha/portal; no se atribuye a este corte un recorrido completo sin errores.
El POST de descarga documental dio 200 en todos los casos finales.

Prueba focal Go, vet, gopls y bash -n verdes. Semgrep local `p/golang`
`--metrics=off`: 0 hallazgos/errores. Gosec del único paquete nuevo: 0
hallazgos/errores. No se ejecutó la puerta completa local; la pasa la CI de
la PR. La publicación real en cidonia sigue a cargo de dirección.

Recibo local: `recibo:8b9459ef-3bd9-4c95-ace2-311083b1f059`. Auditoría local: `provision:72170b76-ff56-4100-93c9-3513e37460ec`.

Huella del catálogo v3: `ac1c917bfade9610b5141e7a1f15e01e7911bcb520e2a586fccce6564d73ecef`.

SHA256 del PDF: `01be43c7df8a63f5eaa093f93022df8ea44ebb9f76b368e8429e73e00bdb16c9`.
