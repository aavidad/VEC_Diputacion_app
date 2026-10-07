# Revisar un borrador para una sesión propuesta

El visor abre la salida local del preparador de sesión S5. Muestra el orden del
día y las propuestas de acuerdo, vinculadas por el punto de agenda. Conserva
los textos aportados como datos; cambiar el idioma de la interfaz no traduce
automáticamente esos textos.

Use sólo material sintético. El archivo no acredita una sesión convocada o
celebrada, asistencia, deliberaciones, votaciones ni acuerdos adoptados. El
formato inicial deja sin cotejar la preparación del tribunal y la fase. El
formato con `cotejo_local` declara que el CLI cotejó una salida local del
tribunal y calculó su huella. El visor comprueba la coherencia del formato,
pero no repite ese cotejo ni verifica la procedencia, vigencia o habilitación.
El borrador no designa miembros, aprueba, firma ni califica.

Genere una salida del CLI y abra el visor desde loopback:

```sh
go run ./cmd/vec-selectivos-preparar-acta \
  --catalogos-dir web/static/textos --idioma es \
  < cmd/vec-selectivos-preparar-acta/testdata/material.json \
  > /tmp/vec-sesion-propuesta.json
python scripts/servir_preparacion_rrhh.py --modulo selectivos-acta-visor
```

Para abrir un borrador que declara el cotejo local, prepare también la salida
del tribunal y use la opción `-tribunal-salida` del CLI de acta:

```sh
go run ./cmd/vec-selectivos-preparar-tribunal \
  --catalogos-dir web/static/textos --idioma es \
  < cmd/vec-selectivos-preparar-tribunal/testdata/material.json \
  > /tmp/vec-tribunal-propuesto.json
go run ./cmd/vec-selectivos-preparar-acta \
  --catalogos-dir web/static/textos --idioma es \
  --tribunal-salida /tmp/vec-tribunal-propuesto.json \
  < cmd/vec-selectivos-preparar-acta/testdata/material-cotejo.json \
  > /tmp/vec-sesion-cotejada-local.json
```

Seleccione la salida JSON en «Abrir borrador preparado». Revise el contexto,
los textos propuestos y «Qué falta comprobar y completar». El número de punto
relaciona cada acuerdo propuesto con su agenda. «Ver referencias, huella
aportada y contenido completo» abre los datos técnicos. La fecha es opcional:
si falta, se muestra «Sin fecha propuesta», sin asignar una fecha por defecto.
Una fecha aportada se presenta en horario de Madrid y permanece como propuesta.

«Descargar original sin verificar» conserva los bytes exactos; no acredita su
procedencia ni constituye un acta de una sesión celebrada. «Cerrar archivo»
retira el material y permite abrirlo de nuevo. Recargar exige volver a abrirlo.
El archivo permanece sólo en memoria; no hay guardado, publicación, cookies,
almacenamiento web ni HTTP de negocio. Las referencias no se convierten en
enlaces ni generan consultas. No hay botones para celebrar una sesión,
adoptar acuerdos, aprobar ni firmar.

El contrato consumido es la raíz `titulo`, `preparacion`, `limite`, `mensajes`
del CLI `vec-selectivos-preparar-acta`, con `cotejo_local` únicamente cuando
ambos avisos de antecedente y fase indican el cotejo local. Se rechazan mezclas
de los dos formatos. Se exige el estado `borrador_propuesto`
y material `preparacion_sintetica`. Las listas vacías y los textos ausentes
permanecen incompletos, con sus pendientes. Se comprueba la correspondencia
de los pendientes con sus mensajes y los enlaces locales entre agenda y
propuestas; esa comprobación no acredita decisiones ni antecedentes.

El visor admite hasta 4 MiB, 100 puntos de agenda y 100 propuestas de acuerdo.
Cada texto tiene un máximo de 4096 bytes UTF-8 y no admite caracteres de control.
La lectura exige UTF-8 válido, claves exactas y ausencia de duplicados o claves
de prototipo. Un error retira el resultado y deshabilita la descarga. Reimportar,
cerrar o abandonar la página invalida las lecturas anteriores; una lectura
tardía no repone un borrador retirado.

Pruebas focales:

```sh
node --test web/static/portal-empleado/modulos/seleccion/preparacion-acta/modelo.test.mjs
VEC_S5_ACTA_SCRATCH=/directorio/scratch \
  python web/static/portal-empleado/modulos/seleccion/preparacion-acta/recorrido_chrome.py
```

Ejecute las pruebas en un sandbox sin red externa, con fuente y herramientas
de sólo lectura, entorno vacío y directorio temporal propio. El recorrido usa
Playwright con Chrome del sistema y el servidor de recursos permitido. Las
capturas y la descarga quedan fuera de Git; los fixtures no se sirven por HTTP.
Estas pruebas comprueban un visor local. No acreditan tribunal vigente, sesión,
acta, autorización nominal, aprobación ni firma.
