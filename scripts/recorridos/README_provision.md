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

Los selectores y las rutas acordadas con la interfaz están en
`provision_expectativas.json`. La propiedad `escenarios` permite incorporar
adjudicación y ciclo cuando tengan contrato y controles propios. Cada entrada
necesita `id`, `selector`, `ruta`, `estado_http` y `esperado` (punteros JSON y
valores exactos); `repetir: true` exige una segunda respuesta idéntica.
Los `pasos` previos admiten `selector`, `accion` (`click`, `fill` o `selectOption`)
y `valor` cuando corresponda, para elegir un ejemplo y completar sus controles.
Los escenarios se eligen
explícitamente con `--escenarios ID,ID`. Un escenario pendiente o desconocido
falla antes de arrancar servicios. Una lista vacía no acredita estos recorridos.

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
Adjudicación y ciclo necesitan sus propios controles, expectativas y ejecución;
la lista de escenarios aún está vacía.
