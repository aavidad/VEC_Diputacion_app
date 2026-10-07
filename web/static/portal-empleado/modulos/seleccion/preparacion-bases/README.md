# Revisar el material preparado de las bases

Esta pantalla consulta una preparación conservada y abre la salida local del
preparador de Selección. Permite revisar contenido, requisitos, fechas,
documentos propuestos, referencias y pendientes. Ambas vías mantienen las
bases pendientes de aprobación, firma y publicación.

Para consultar, indique la referencia de la preparación y pulse «Consultar
preparación». La última revisión usa esa referencia; una revisión concreta
requiere también su número y huella originales. La sesión y los permisos los
resuelve el servidor común. La pantalla no pide identidad, ámbito ni permisos.

La consulta usa `POST /api/vec/seleccion/preparacion-bases/consultar`. Sólo muestra
una respuesta `200` con estado `obtenida`, selector coincidente y evidencias
válidas. Los pendientes proceden de la respuesta de Bolsa: no se evalúan aquí
ni se elimina el pendiente de las reglas de baremación. El detalle separa el
recibo de la preparación conservada del acceso efectuado por esta consulta.

Cambiar un dato, iniciar otra consulta, cancelar, cerrar o abandonar la pantalla
retira el material anterior y cancela la petición pendiente. Una respuesta tardía
no repone ese material. Una denegación, ausencia, conflicto, fallo de servicio o
respuesta incompatible deja la pantalla sin resultados ni descarga habilitada.

Los bytes originales de la consulta permanecen sólo en memoria para validar y
mostrar la respuesta. No hay descarga de la respuesta consultada: esa exportación
requiere su propia operación nominal auditada. «Descargar original» se limita al
archivo local abierto y conserva sus bytes exactos.

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
ni se convierten en enlaces. Esta pantalla no guarda versiones ni concede permisos nuevos. La consulta usa
el preparador durable existente de Bolsa; no firma, aprueba ni publica bases.
Este visor cubre una parte de preparación de S2; S2 sigue abierto.

Pruebas focales:

```sh
node --test web/static/portal-empleado/modulos/seleccion/preparacion-bases/modelo.test.mjs \
  web/static/portal-empleado/modulos/seleccion/preparacion-bases/consulta.test.mjs
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

El recorrido `recorrido_consulta_chrome.py` comprueba el consumidor en Chrome del
sistema con respuestas sintéticas interceptadas: actual/exacta, idioma,
retirada ante errores y respuesta tardía, y ausencia de descarga de la consulta.
No demuestra sesión nominal, autorización instalada, PostgreSQL ni reinicio.
El driver nominal de S2 permanece en `scripts/recorridos/seleccion/preparacion-bases/`;
su fuente de identidad y las demás dependencias deben estar disponibles antes
de acreditar ese recorrido.
