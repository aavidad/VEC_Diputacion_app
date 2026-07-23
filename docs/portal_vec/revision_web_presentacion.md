# Captura y revisión de la presentación web

`scripts/capturar_presentacion_web.py` recorre por defecto el lanzador, el
portal público, las 14 vistas del área aspirante, las 20 vistas internas de
RRHH y 25 estados de interacción. Incluye los selectores abiertos desde el
área aspirante y la administración, los perfiles funcionario y técnico con
permisos distintos, y una ruta real de Dietas sobre el mapa OSM interno. Repite
ese recorrido en 1440×1000, 1024×900 y 390×844. Desde el 24 de julio añade una
matriz contractual separada de las 17 pantallas de contratación temporal
solicitadas por RRHH, capturada por viewport en 1536×1024, 1440×1000 y
1280×900. El detalle, las
secuencias y las dos brechas de paridad están en
[la matriz 1..17](matriz_revision_visual_17_pantallas_rrhh_2026-07-24.md).

La ejecución cerrada el 20 de
julio de 2026 obtuvo **183/183 escenarios correctos, 183 capturas y cero
hallazgos** sobre la imagen Docker reconstruida después del pulido de
escritorio. Además de la puerta automática, se inspeccionaron manualmente las
capturas afectadas en 1440×1000 y 1024×900.
Esa evidencia precede a la matriz 1..17: no demuestra que sus 51 recorridos
nuevos hayan pasado. La nueva evidencia se declarará solo después de
reconstruir la imagen, ejecutar Docker e inspeccionar 1536, 1440 y 1280.
Ninguna evidencia automática sustituye la aceptación humana de RRHH.

## Preparación y uso

No se instala Playwright ni Chromium en el anfitrión. El navegador, las
dependencias Python y el capturador se ejecutan en la imagen fijada del servicio
`revision-web-presentacion`; solo `var/` se monta con escritura para conservar
los resultados. Primero se levanta la composición completa:

```bash
scripts/arrancar_presentacion_rrhh.sh
```

Después se ejecuta la revisión estricta:

```bash
docker compose --profile presentacion --profile herramientas-presentacion run \
  --rm --no-deps revision-web-presentacion
```

Compose hace que el revisor efímero comparta el espacio de red de
`proxy-presentacion` (`network_mode: service:proxy-presentacion`) y consulta
`http://127.0.0.1:8080`. Así Chromium prueba un origen loopback fiable, dispone
de WebCrypto y recorre exactamente el mismo proxy sin publicar otro puerto ni
instalar un reenviador en el anfitrión. La CLI mantiene como valor
predeterminado `http://127.0.0.1:8081`.

Opciones útiles:

- `--salida RUTA`: cambia `var/revision-web`, que ya está bajo el `var/`
  ignorado por Git.
- `--superficie area-aspirante`: limita una ejecución de diagnóstico; puede
  repetirse. Sin esta opción siempre se cubre el manifiesto completo.
- `--solo-pantallas-rrhh`: ejecuta únicamente las 17 pantallas numeradas en
  viewport 1536, 1440 y 1280; es la puerta específica de comparación con el
  documento recibido.
- `--timeout-ms 12000`: cambia la espera acotada. Se usa
  `domcontentloaded`; no se espera la inactividad total de red.
- `--tolerante`: conserva capturas y hallazgos, pero devuelve código cero.
  El modo predeterminado es estricto y devuelve un código distinto de cero si
  cualquier escenario presenta un fallo.
- `--red-docker-interna`: queda como ayuda de diagnóstico para una red Docker
  privada. Un origen HTTP privado no es un contexto seguro de navegador y no
  acredita el flujo real de Dietas; la puerta oficial usa loopback y producción
  debe usar HTTPS.

Para pasar opciones distintas sin instalar herramientas locales, se puede
sustituir el comando del servicio. Por ejemplo, un diagnóstico acotado del área
aspirante es:

```bash
docker compose --profile presentacion --profile herramientas-presentacion run \
  --rm --no-deps revision-web-presentacion \
  python3 scripts/capturar_presentacion_web.py \
  --url-base http://127.0.0.1:8080 \
  --salida var/revision-web --superficie area-aspirante
```

Antes de abrir ninguna vista, el capturador exige la cabecera técnica
`X-VEC-Modo-Presentacion: aislada-sintetica-v1`. Solo el handler validado de
`vec-presentacion` la emite; el servidor integrado, el público, el interno y
una composición de presentación sin sus dos guardas no la incluyen. La
ausencia o alteración de esta marca bloquea todo el recorrido.

La salida contiene:

- `var/revision-web/capturas/<tamaño>/vista/...`: capturas de página completa;
  si la altura excede el límite fiable de Chromium, el revisor recorre la
  página y conserva todas las partes como `*.parte-NN.png`;
- `var/revision-web/capturas/<tamaño>/flujo/...`: capturas del viewport para
  distinguir los estados de interacción, incluidos el menú general y el
  submenú completo de Bolsa abiertos en móvil;
- `var/revision-web/resultados.json`: resultado estructurado y métricas;
- `var/revision-web/informe.md`: índice legible con enlaces a las capturas.

## Qué se comprueba

Cada escenario nace en un contexto de navegador nuevo. El revisor no inyecta
cookies ni un `storage_state` y marca como fallo cualquier cookie,
`localStorage`, `sessionStorage`, base IndexedDB o Cache Storage creado por la
página. También comprueba:

- respuestas HTTP de error y recursos de red fallidos;
- una ventana de red estable y cualquier petición que siga pendiente al vencer
  la observación acotada;
- errores de consola y excepciones JavaScript no controladas;
- estado de carga y título de cada vista;
- presencia, apertura móvil, opciones y estado actual de los menús;
- banner visible y explícitamente sintético en las superficies privadas;
- identificadores HTML duplicados;
- controles visibles sin nombre accesible;
- desbordamiento horizontal del documento.
- presencia por fila, posición fija, geometría y contenido completo de las
  columnas Estado y Acciones en las tablas administrativas prioritarias de
  1024 px o más; el rediseño móvil de esas tablas queda fuera de esta entrega.

El flujo `rrhh-dietas-ruta-real` añade estas condiciones a la puerta actual:

- el cálculo debe proceder del mediador cartográfico interno y declarar la
  versión gobernada del grafo OSRM;
- el visor debe montar una única capa Leaflet con teselas del mismo origen;
- la revisión espera la carga real de OpenStreetMap, no una geometría SVG ni
  una distancia sintética;
- cualquier error de ruta o tesela permanece visible y hace fallar el escenario.

Los flujos privados solo se ejecutan si el banner DEMO está visible y la URL
contiene `presentacion=rrhh`, la respuesta conserva la cabecera técnica exacta
y el perfil requerido está declarado. Las confirmaciones y recibos se producen
en los adaptadores efímeros de presentación; el capturador no invoca conectores
administrativos reales. La excepción deliberada es la cartografía local de
Dietas, que no trata identidad ni produce actos. Además de los recorridos público y aspirante, se exige un recibo
`DEMO-REC-*` en operaciones representativas de bases, admisión, méritos,
baremo, importación, llamamientos, contratos, documentos, comunicaciones,
exportación, roles y alegaciones.

La descarga documental se prueba además a nivel de contrato: el certificado
es un PDF 1.4 binario, con MIME y nombre correctos, cierre válido, identidad
institucional, texto del certificado, referencia opaca y QR vectorial. El QR
fue renderizado desde el PDF final y decodificado con un lector independiente,
que recuperó exactamente la URL canónica de comprobación sin datos personales.

## Pruebas del manifiesto sin abrir Chromium

Los helpers también se comprueban dentro de la misma imagen fijada, sin instalar
Python ni Playwright en el anfitrión:

```bash
docker compose --profile herramientas-presentacion run --rm --no-deps \
  revision-web-presentacion \
  python3 -m unittest scripts.tests.test_capturar_presentacion_web
```

Las capturas quedan en `var/revision-web`, fuera del control de versiones. La
imagen de Playwright, igual que OSRM y TileServer GL, se ejecuta solo en Docker;
no se crean servicios, paquetes ni procesos persistentes en el anfitrión.
