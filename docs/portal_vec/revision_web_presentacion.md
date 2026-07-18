# Captura y revisión de la presentación web

`scripts/capturar_presentacion_web.py` recorre por defecto el lanzador, el
portal público, las 14 vistas del área aspirante, las 16 vistas internas de
RRHH y siete estados de interacción DEMO, incluido el perfil técnico con
permisos restringidos. Repite el recorrido en 1440×1000,
1024×900 y 390×844.

## Preparación y uso

Arranque primero el servicio de presentación local. El script presupone
`http://127.0.0.1:8081`, pero la URL es configurable:

```bash
python3 -m pip install playwright
python3 -m playwright install chromium
python3 scripts/capturar_presentacion_web.py
python3 scripts/capturar_presentacion_web.py --url-base http://127.0.0.1:8081
```

Opciones útiles:

- `--salida RUTA`: cambia `var/revision-web`, que ya está bajo el `var/`
  ignorado por Git.
- `--superficie area-aspirante`: limita una ejecución de diagnóstico; puede
  repetirse. Sin esta opción siempre se cubre el manifiesto completo.
- `--timeout-ms 12000`: cambia la espera acotada. Se usa
  `domcontentloaded`; no se espera la inactividad total de red.
- `--tolerante`: conserva capturas y hallazgos, pero devuelve código cero.
  El modo predeterminado es estricto y devuelve un código distinto de cero si
  cualquier escenario presenta un fallo.
- `--con-interfaz`: muestra Chromium mientras se ejecuta la revisión.
- `--ejecutable-navegador RUTA`: usa un Chromium ya instalado cuando el puesto
  no puede descargar el navegador administrado por Playwright.

La salida contiene:

- `var/revision-web/capturas/<tamaño>/vista/...`: capturas de página completa;
- `var/revision-web/capturas/<tamaño>/flujo/...`: capturas del viewport para
  distinguir los estados de interacción;
- `var/revision-web/resultados.json`: resultado estructurado y métricas;
- `var/revision-web/informe.md`: índice legible con enlaces a las capturas.

## Qué se comprueba

Cada escenario nace en un contexto de navegador nuevo. El revisor no inyecta
cookies ni un `storage_state` y marca como fallo cualquier cookie,
`localStorage` o `sessionStorage` creado por la página. También comprueba:

- respuestas HTTP de error y recursos de red fallidos;
- errores de consola y excepciones JavaScript no controladas;
- estado de carga y título de cada vista;
- presencia, apertura móvil, opciones y estado actual de los menús;
- banner visible y explícitamente sintético en las superficies privadas;
- identificadores HTML duplicados;
- controles visibles sin nombre accesible;
- desbordamiento horizontal del documento.

Los flujos privados solo se ejecutan si el banner DEMO está visible y la URL
contiene `presentacion=rrhh`. Las confirmaciones y recibos se producen en los
adaptadores efímeros de presentación; el capturador no invoca conectores reales.

## Pruebas sin navegador

El manifiesto y los helpers se verifican sin instalar Playwright:

```bash
python3 -m unittest scripts.tests.test_capturar_presentacion_web
```
