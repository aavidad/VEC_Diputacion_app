# Recuperar una preparación local de solicitud

Selección posee este consumidor. Sus archivos nuevos están en el grafo público
de Bolsa para reutilizar `modelo.js` y `contrato-v2.js` sin mezclar superficies
públicas y privadas. Los archivos originales de Bolsa permanecen intactos.

La vista abre un archivo `vec.bolsa.preparacion-local.v1`, muestra convocatoria,
versión, requisitos, plazos, documentos mencionados y metadatos de archivos.
Todo procede del archivo y sigue sin cotejar. No consulta una ficha de persona,
no exige vínculo de empleado ni añade persona o empleado al resumen.

Recorrido local:

1. En la preparación de Bolsa, llegar al resumen y pulsar «Descargar borrador JSON».
2. Abrir `/bolsa/preparacion/recuperacion/?lang=es` o `?lang=en`.
3. Seleccionar ese JSON y revisar sus datos y lo pendiente. El TXT no es compatible.
4. Descargar el mismo resumen: se conservan exactamente los bytes importados.
5. Cerrar el archivo y volver a elegirlo para recuperar el mismo contenido.

El archivo solo vive en memoria de la pestaña. No se envía a un servidor ni se
guarda en almacenamiento web. Al cerrar, sustituirlo o abandonar la página se
retira la vista anterior y se revoca la URL de descarga. La importación no abre
las rutas de documentos ni los archivos que aparecen en los metadatos.

El lector rechaza claves duplicadas o desconocidas, tipos incorrectos, un estado
distinto de `sin_presentar` y requisitos distintos de `pendiente`. Reutiliza los
validadores existentes de identificadores, límites, nombres de archivo y rutas
documentales. Los arrays vacíos y la descripción opcional de los plazos siguen
siendo compatibles con el productor. El límite de 2 MiB del archivo es técnico; no es una regla de inscripción ni
admisión. No se imponen máximos adicionales a textos o listas de requisitos,
plazos y documentos. Los metadatos de archivos se comprueban con los límites
actuales del catálogo de preparación de Bolsa (8 archivos y 10 MiB por archivo).
El formato v1 no conserva una versión de esos límites: un resumen histórico
que exceda la configuración actual queda fuera del alcance de este lector.

La versión y huella aportadas no acreditan fuente, publicación ni vigencia. La
identidad, representación, formulario admitido y documentos custodiados siguen
pendientes. Las tasas o exenciones dependen de las bases y de su circuito; aquí
no se calculan ni se declaran obligatorias. No hay firma, registro, pago,
presentación, justificante ni inscripción durable por recuperar este archivo.

Esta unión incluye la exportación JSON adicional del productor revisado en
`8dd9b4cd7`, junto a su descarga TXT anterior. Los cinco archivos del productor
conservan exactamente los blobs revisados. La ayuda del recuperador enlaza al listado público, conserva el idioma y
permite elegir la convocatoria que lleva a su preparación y botón JSON. No usa
un identificador procedente del archivo ni abre una preparación sin selector. La preparación, sus guardas y los derechos siguen intactos.
La prueba focal usa una respuesta pública sintética declarada: la pantalla
productora genera el JSON real, el consumidor lo importa y lo reexporta con
los mismos bytes. No acredita una fuente administrativa ni datos de producción.

Comprobación focal:

```sh
node --test web/static/bolsa/preparacion/recuperacion/material.test.mjs \
  web/static/bolsa/preparacion/modelo.test.mjs \
  web/static/bolsa/preparacion/entrada.test.mjs
```

El dataset de `testdata/detalle-publico-sintetico.json` es una fixture del contrato
público. Su marca de fuente representa el formato de prueba, no una fuente real.
Solo las pruebas llaman al productor con esa fixture; el runtime importa el
archivo y nunca fabrica un detalle para eludir la guarda de demostración.

Fuentes del corte: `docs/plan_modulos/selectivos.md`, S3, y
`docs/estudio_requisitos/convoca_inscripcion_externa_contrato_2026-10-01.md`.
La preparación procede de #263; #267 verifica sus recorridos y no implementa
otra solicitud. No se presta `RegistrarSolicitud` del núcleo heredado, que
calcula baremación y guarda, ni el diario de borradores de convocatorias.
