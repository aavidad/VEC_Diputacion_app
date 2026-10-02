# Revisar el material preparado de las bases

Esta pantalla abre la salida del preparador integrado de Selección. Permite
revisar contenido, requisitos, fechas, documentos propuestos, referencias y
pendientes. La descarga conserva exactamente los bytes del archivo elegido.

Desde la raíz del repositorio, prepare un archivo de prueba y arranque el visor:

```sh
go run ./cmd/vec-selectivos-preparar-bases \
  --catalogos-dir web/static/textos --idioma es \
  < cmd/vec-selectivos-preparar-bases/testdata/material-completo.json \
  > /tmp/vec-bases-preparadas.json

python scripts/servir_preparacion_rrhh.py --modulo seleccion-bases-preparacion
```

Abra la dirección de loopback que imprime el servidor y seleccione
`/tmp/vec-bases-preparadas.json`. Los pendientes aparecen antes del contenido.
«Descargar original» devuelve el archivo sin cambios; «Cerrar archivo» lo
retira de la pantalla. El selector cambia entre castellano e inglés sin perder
el material abierto. La ayuda está en «?».

El visor admite un único JSON UTF-8, de hasta 4 MiB y 1.024 elementos por lista.
Estos límites del visor son independientes del límite de entrada de 1 MiB del
CLI: la salida incluye la propuesta, los mensajes y, cuando procede, contenido
canónico. Un material excesivo se rechaza sin mostrar un resultado parcial.
El formato exige el estado pendiente y los pendientes de referencias,
documentos, firma/custodia, aprobación y publicación. No verifica que el
archivo provenga del CLI ni que sus datos sean ciertos.

El archivo permanece sólo en memoria del navegador. Recargar exige abrirlo de
nuevo. Las rutas de documentos y sus huellas son texto propuesto: no se consultan
ni se convierten en enlaces. No hay guardado institucional, permisos nuevos,
firma, aprobación, publicación ni integración con el gobierno de Bolsa.
Este visor cubre una parte de preparación de S2; S2 sigue abierto.

Pruebas focales:

```sh
node --test web/static/portal-empleado/modulos/seleccion/preparacion-bases/modelo.test.mjs
VEC_S2_SCRATCH=/directorio/scratch \
  python web/static/portal-empleado/modulos/seleccion/preparacion-bases/recorrido_chrome.py
python -m unittest scripts.tests.test_servir_preparacion_rrhh
```

El recorrido necesita Playwright y el Chrome del sistema. Ejecútelo dentro de
un sandbox sin red externa, con sólo loopback, fuente y herramientas de sólo
lectura, entorno vacío y directorio temporal propio. Los fixtures de `testdata/`
son salidas de los ejemplos del CLI; el servidor nunca los sirve. La prueba
abre ambos, descarga el original, cambia de idioma y comprueba escritorio,
móvil, teclado, rechazo de JSON malformado, falso estado aprobado y contenido
que intenta ejecutar HTML o navegar a otra red.
