# Preparar la revisión de admisión

El visor abre la salida local del preparador S4. Muestra requisitos propuestos,
motivos y acciones sugeridas. Cada requisito permanece pendiente de revisión.
Los títulos proceden de la propuesta aportada: el visor no los extrae de S3 ni
interpreta datos de Méritos o Personal.

Use sólo material sintético. La lista aún debe cotejarse con todos los requisitos
de las bases exactas. Puede estar incompleta. Un soporte declarado, rechazado,
acreditado según el paquete o fuera de la vigencia indicada no resuelve el
requisito ni cambia el Registro Único de Méritos.

Prepare una salida del CLI y abra el visor desde loopback:

```sh
go run ./cmd/vec-selectivos-preparar-admision \
  --catalogos-dir web/static/textos --idioma es \
  < cmd/vec-selectivos-preparar-admision/testdata/material.json \
  > /tmp/vec-admision-preparada.json
python scripts/servir_preparacion_rrhh.py --modulo selectivos-admision-visor
```

Seleccione la salida JSON en «Abrir material preparado». Revise los motivos y
«Qué preparar ahora». «Ver soportes y referencias» abre los datos aportados por
el paquete. «Descargar original sin verificar» conserva los bytes exactos; esa
descarga no acredita la procedencia del archivo ni un acto de admisión.
«Cerrar archivo» retira el material y permite abrirlo otra vez.

La versión propuesta de las bases se muestra como contexto. El vínculo opcional
con S3 conserva la huella de los bytes originales y metadatos de un resumen
local `sin_presentar`. Coincidir dentro del archivo no acredita una convocatoria
ni la correspondencia entre solicitud y requisito. Sin ese contexto, el visor
muestra «Sin solicitud vinculada». Identidad y registro permanecen pendientes.

El transporte consumido es `vec.seleccion.admision-preparacion.v1`, raíz JSON
directa del CLI. Se exigen `preparacion_sintetica`,
`pendiente_revision_competente`, `propuesto_no_cotejado`, requisitos `pendiente`
y `persistido`/`admision_oficial` en `false`. La lista vacía es válida y pide
definir requisitos. El archivo admite hasta 4 MiB, 64 requisitos y 32 soportes
o referencias de hechos por requisito. Las claves son exactas; se rechazan
duplicados, claves de prototipo, referencias incompatibles y falsas decisiones.
Los títulos tienen un máximo de 256 bytes UTF-8, sin controles ni espacios
exteriores. Las referencias se muestran como texto, sin enlaces ni peticiones.
El límite de salida del visor permite la expansión de soportes repetidos entre
requisitos; es independiente del límite de entrada de 1 MiB del CLI.

El archivo permanece sólo en memoria. Abrir otro, cerrar, abandonar la página o
recibir un error retira resultados y descarga. Una lectura tardía no los repone.
Cambiar de idioma conserva el archivo válido y usa el traductor común. No hay
HTTP de negocio, cookies, almacenamiento web, guardado ni publicación. La
pantalla no ofrece controles para admitir, excluir o aprobar listas. Los plazos
de subsanación siguen pendientes del circuito y las bases aplicables.

Pruebas focales:

```sh
node --test web/static/portal-empleado/modulos/seleccion/preparacion-admision/modelo.test.mjs
VEC_S4_SCRATCH=/directorio/scratch \
  python web/static/portal-empleado/modulos/seleccion/preparacion-admision/recorrido_chrome.py
```

Ejecute las pruebas en un sandbox sin red externa, fuente y herramientas de sólo
lectura, entorno vacío y directorio temporal propio. El recorrido usa Playwright
con Chrome del sistema y el servidor de recursos permitido. Las capturas y la
descarga quedan fuera de Git. Los fixtures no se sirven por HTTP. Estas pruebas
comprueban un visor local; no acreditan sesión nominal, lectura autorizada de
RUM/Personal, registro, aprobación, publicación ni persistencia.
