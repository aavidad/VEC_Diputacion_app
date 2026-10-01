# Recuperar una preparación local de solicitud

Selección posee este consumidor. Sus archivos nuevos están en el grafo público
de Bolsa para reutilizar `modelo.js` y `contrato-v2.js` sin mezclar superficies
públicas y privadas. Los archivos originales de Bolsa permanecen intactos.

La vista abre un archivo `vec.bolsa.preparacion-local.v1`, muestra convocatoria,
versión, requisitos, plazos, documentos mencionados y metadatos de archivos.
Todo procede del archivo y sigue sin cotejar. No consulta una ficha de persona,
no exige vínculo de empleado ni añade persona o empleado al resumen.

Recorrido local:

1. Abrir `/bolsa/preparacion/recuperacion/?lang=es` o `?lang=en`.
2. Seleccionar un resumen JSON de preparación. Revisar sus datos y lo pendiente.
3. Descargar el mismo resumen: se conservan exactamente los bytes importados.
4. Cerrar el archivo y volver a elegirlo para recuperar el mismo contenido.

El archivo solo vive en memoria de la pestaña. No se envía a un servidor ni se
guarda en almacenamiento web. Al cerrar, sustituirlo o abandonar la página se
retira la vista anterior y se revoca la URL de descarga. La importación no abre
las rutas de documentos ni los archivos que aparecen en los metadatos.

El lector rechaza claves duplicadas o desconocidas, tipos incorrectos, un estado
distinto de `sin_presentar` y requisitos distintos de `pendiente`. Reutiliza los
validadores existentes de identificadores, límites, nombres de archivo y rutas
documentales. Los arrays vacíos y la descripción opcional de los plazos siguen
siendo compatibles con el productor. El límite de 2 MiB y las cardinalidades del
lector son límites técnicos propios; no son reglas de inscripción ni admisión.

La versión y huella aportadas no acreditan fuente, publicación ni vigencia. La
identidad, representación, formulario admitido y documentos custodiados siguen
pendientes. Las tasas o exenciones dependen de las bases y de su circuito; aquí
no se calculan ni se declaran obligatorias. No hay firma, registro, pago,
presentación, justificante ni inscripción durable por recuperar este archivo.

Dependencia pendiente: la pantalla actual de Bolsa descarga texto localizado;
no exporta todavía su objeto JSON interno. La exportación JSON adicional se ha
encargado al propietario de esa pantalla. Este corte prueba el consumidor con
un archivo sintético generado en pruebas mediante el `crearResumen` existente.
No acredita el recorrido completo desde esa descarga hasta la recuperación.

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
