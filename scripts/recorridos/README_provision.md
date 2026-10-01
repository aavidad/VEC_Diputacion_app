# Recorrido del simulador interno de Provisión

El guion abre la interfaz de Provisión con Chrome del sistema y envía peticiones
al servidor Go local. Usa ejemplos sintéticos propios del módulo. Las APIs no se
interceptan y no se descargan herramientas, dependencias ni datos.

## Preparación y ejecución

La fuente debe reunir el servidor `cmd/vec-baremador-web`, los casos de uso y
`web/static/portal-empleado/modulos/provision/`. El guion puede vivir en otra
rama; `--source-root` identifica el árbol que se va a probar.

```bash
node --check scripts/probar_provision_navegador.mjs
node scripts/probar_provision_navegador.mjs --verificar-configuracion
node scripts/probar_provision_navegador.mjs --source-root /ruta/al/arbol/agrupado
```

Se puede usar un binario ya construido con esa misma fuente:

```bash
node scripts/probar_provision_navegador.mjs \
  --source-root /ruta/al/arbol/agrupado \
  --server-bin /ruta/al/binario/vec-baremador-web
```

`CHROME_BIN`, `PLAYWRIGHT_MODULE` y `GO_BIN` permiten elegir herramientas locales
ya instaladas. El valor predeterminado de Chrome es `/usr/bin/google-chrome`.
Si falta una herramienta o dependencia, la ejecución falla y deja constancia.
`--caso es:390` permite comprobar un fallo concreto sin repetir la matriz. El
acta identifica ese recorrido como focal; no acredita los demás idiomas o anchos.

El servidor usa un puerto libre en `127.0.0.1`, elegido por él mismo. No se acepta
una URL de un servidor existente. El guion cierra sus contextos de navegador,
Chrome y el servidor propio en `finally`; elimina el directorio temporal del
binario. También atiende `SIGINT`, `SIGTERM` y un límite total de ocho minutos.

## Comprobaciones

- Español e inglés a 1440 y 390 píxeles, con idioma del documento correcto.
- Reflujo de 720 píxeles CSS con escala 2 y alto de 450 píxeles. El zoom nativo de
  Chrome al 200 % queda pendiente de revisión manual.
- Recorrido con Tab, foco visible y sin obstrucción; movimiento de preferencias
  con Enter y conservación del foco.
- Dos simulaciones desde la interfaz: misma entrada, respuesta, desglose y SHA256
  al repetir. Total y desglose visibles coinciden con el servidor y el idioma.
  Las respuestas completas quedan en el acta.
- Configuración inválida: texto conservado, campo marcado, explicación visible,
  simulación bloqueada y recuperación al corregir el valor.
- Rechazo HTTP 400 de JSON incompleto y ausencia HTTP 404 del endpoint de
  solicitudes institucionales en este servidor local.
- Ningún error JavaScript, error inesperado de consola, llamada externa,
  credencial enviada, cookie o uso de almacenamiento web.
- Ausencia de desbordamiento horizontal de página; las tablas anchas deben
  desplazarse dentro de su contenedor. Los contenedores y las acciones deben
  caber en sus antecesores, sin recorte oculto por `overflow`.

Chrome puede escribir un diagnóstico de red por los rechazos provocados. El acta
los cuenta aparte; nunca los presenta como errores JavaScript.

Los selectores, rutas y resultados esperados están en
`provision_expectativas.json`. `esperado` relaciona punteros JSON con valores
exactos; `cantidades` exige el tamaño de una colección. Los `pasos` previos
admiten `selector`, `accion` (`click`, `fill` o `selectOption`) y `valor` cuando
corresponda.

## Adjudicación y ciclo

Los cuatro escenarios disponibles son `adjudicacion`, `ciclo-pendiente`,
`ciclo-mantener` y `ciclo-rectificar`. Este modo omite las pruebas y los POST de
preparación ya comprobados. La página carga su preparación inicial, pero el
recorrido actúa exclusivamente sobre las pestañas de ensayos.

```bash
node scripts/probar_provision_navegador.mjs \
  --source-root /ruta/al/arbol/con/ensayos \
  --server-bin /ruta/al/binario/vec-baremador-web \
  --escenarios adjudicacion,ciclo-pendiente,ciclo-mantener,ciclo-rectificar
```

Cada escenario compara la petición y los bytes de dos respuestas HTTP reales,
conserva el JSON recibido y su SHA256, coteja el resultado visible y abre los
detalles por teclado. Se comprueba el foco al terminar la respuesta, los recortes
de referencias y huellas, las tablas y los botones. La adjudicación verifica dos
asignaciones sin duplicados y la validación de política. El ciclo exige una o
dos versiones según el caso, la misma valoración inicial, enlaces de huellas y
los puntos exactos del ejemplo. Mantener conserva `28386027`; rectificar pasa a
`27906027`. Son micro-puntos de este fixture sintético.

La resolución queda en borrador, con firma, publicación y efecto oficial en
`false`; los botones oficiales permanecen deshabilitados. Los rechazos de JSON
incompleto y endpoint institucional pertenecen al modo de preparación, no se
repiten con los ensayos. Un escenario desconocido falla antes de arrancar.

## Evidencia y alcance

Cada ejecución crea un directorio privado bajo
`~/.local/state/vec-codexb-provision-20261001/run-*/`. Conserva capturas y
`resultado.json`, con commit y cambios de la fuente, huella del binario, versión
de Chrome, huellas del guion y expectativas, peticiones HTTP y estado de cada caso. Un fallo produce salida 1 y
una captura si el navegador ya estaba abierto. Los artefactos quedan fuera de
Git y deben revisarse antes de adjuntarlos a una PR.

Este recorrido acredita únicamente el simulador sintético local que se haya
ejecutado. No acredita el portal institucional, PostgreSQL, identidad,
autorización, firmas, publicaciones ni efectos administrativos. Las capturas
sirven para la revisión visual independiente; el guion no acredita por sí solo
conformidad WCAG ni contraste de todo el portal.

El 1 de octubre de 2026 pasaron los seis casos sobre la fuente limpia
`c7995ca353571d3e533574beffe09f98e83aca46`, con Chrome `149.0.7827.200` y el
binario construido desde ese árbol. El acta de esa ejecución está en
`run-AuivdP/resultado.json`, bajo el directorio de evidencia anterior. Se
comprobaron las preferencias, la validación visible, Tab después de editar,
los puntos y el desglose localizados, la repetición exacta y los rechazos previstos.
Los seis casos tuvieron cero errores JavaScript, llamadas externas, cookies y
uso de almacenamiento web. Chrome, su perfil y el servidor propio se cerraron.

La revisión visual encontró después recorte de tablas y acciones a 390 píxeles.
Ese acta conserva lo probado sobre HTTP, teclado y determinismo; no acredita
corrección visual del móvil. El guion incorpora ahora el control del recorte.

La corrección pasó cuatro recorridos focales sobre la fuente limpia
`2245df47124d54651f083bc811be55baeffd3019`, con el binario construido desde ese
árbol. Las tablas de valoración ocuparon 324 píxeles a 390, con desplazamiento
interno del contenido. Las acciones de simulación y corrección quedaron completas
dentro del panel. Las actas bajo el mismo directorio de evidencia son:

| Caso | Acta |
| --- | --- |
| Español, 390 px | `run-olfz1I/resultado.json` |
| Inglés, 390 px | `run-3RWIxk/resultado.json` |
| Español, reflujo 720 px | `run-OScxDB/resultado.json` |
| Inglés, reflujo 720 px | `run-p15TQM/resultado.json` |

El recorrido cubre la primera interfaz de preparación y simulación.
La matriz de adjudicación y los tres casos de ciclo pasó sobre
`52ee65857597f1bb50bcba9873a6f4d5512aee7b`, fuente limpia y binario coherente,
en `run-BHiEPN/resultado.json`: seis combinaciones ES/EN a 1440, 390 y reflujo
720 píxeles, con ocho POST de ensayos por combinación y ninguno de preparación.
El focal `run-oDO61e/resultado.json`, sobre la misma fuente a ES/390, comprobó la
aserción añadida de foco tras respuesta y los valores exactos del fixture.
Estas actas corresponden al montaje combinado; una rama aislada necesita
comprobar sus diferencias de montaje antes de atribuirle la misma evidencia.

El focal de adjudicación aislada pasó sobre la fuente limpia
`28f296a90067b7109d91cde2691de2cefff70d5a`, tras portar la restauración de foco.
Su acta es `run-6oqONX/resultado.json`, ES/390, con dos POST de adjudicación y
ninguno de preparación. El resultado tiene la misma huella que el ensayo
combinado; conserva foco, referencias abiertas, límites del panel y validación.

La rama de ciclo `0a634b8fe` conserva los mismos blobs que `52ee6585` en la
interfaz Provisión, sus catálogos ES/EN y los handlers de proceso, adjudicación,
ciclo y catálogo de ensayos examinados. Esa comparación de fuentes no es otra
ejecución del navegador; la matriz registrada se ejecutó sobre `52ee6585`.

El focal separado `native-zoom-C0Qmbq/resultado.json` pasó ocho vistas ES/EN
con zoom nativo de Chrome al 200 %: preferencias, valoración, adjudicación y ciclo.
Usó preparación `2245df471` y ensayos `7f4103218`, con foco y capturas CDP revisados.
Chrome midió 720 píxeles CSS en una ventana de 1440, DPR 2 y zoom CSS 1.
Se hicieron doce POST locales para las fases de métricas y capturas.
El guion conserva su prueba de reflujo; no automatiza este focal de zoom nativo.
La revisión fue favorable; no acredita contraste completo, lector de pantalla ni persistencia.
