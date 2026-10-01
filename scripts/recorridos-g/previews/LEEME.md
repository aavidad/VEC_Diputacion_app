# Recorrido local de las previews de Dietas

Este guion abre **informes** y **catálogo de tarifas** con datos sintéticos. Sirve para revisar su presentación y sus límites. No consulta el servidor VEC, no autentica a nadie y no acredita publicación ni un cálculo de tarifas aprobado.

Necesita una **copia fuente sintética controlada** que contenga las dos páginas y sus archivos estáticos. El servidor propio expone los archivos públicos de `web/static` y únicamente estos datos: `data/demo/dietas/informes.json`, `data/demo/dietas/catalogo-rrhh.json` y `data/catalogos/dietas/informes-ejemplo-v1.json`. Debe usar una copia que reúna los cortes de informes y catálogo; una de las ramas por separado no basta. No pase como fuente un repositorio con secretos o datos privados.

Primero compruebe las dependencias sin abrir Chrome ni red:

```sh
node scripts/recorridos-g/previews/recorrer.mjs --source /RUTA/COPIA_SINTETICA --modo plan
```

Si falta una preview o un JSON, termina con código `2` e identifica cada ruta ausente. Si el plan pasa, use Chrome del sistema y Playwright instalado localmente:

```sh
VEC_PLAYWRIGHT_MODULE=/RUTA/PLAYWRIGHT/index.mjs \
  node scripts/recorridos-g/previews/recorrer.mjs \
  --source /RUTA/COPIA_SINTETICA --modo chrome
```

Comprueba ambas páginas en español e inglés a 1440 y 390 px y con zoom nativo al 200 %. Revisa el aviso sintético, filtros, una propuesta de 42,50 €, límites de exportar, imprimir y publicar, un fallo transitorio y su reintento en catálogo, foco por teclado, scroll del documento y ausencia de errores JavaScript, cookies, almacenamiento web, descargas y peticiones externas. La prueba de reintento devuelve un `503` **solo desde el navegador a la fixture**, sin tocar el archivo ni un servidor VEC. El resultado informa qué pasó; una dependencia ausente nunca cuenta como recorrido verde.

El servidor escucha en `127.0.0.1` con un puerto efímero. Deniega métodos distintos de GET, rutas `/api`, datos fuera de esos tres JSON y enlaces simbólicos. El navegador bloquea WebSocket, otros orígenes y cualquier ruta fuera de la lista. Se cierra y elimina el perfil temporal de Chrome al terminar.

Prueba focal del guion:

```sh
node --test scripts/recorridos-g/previews/recorrer.test.mjs
```

Este recorrido es una revisión de las previews. El montaje real, los permisos, las respuestas del backend y la persistencia requieren sus comprobaciones propias cuando estén instalados.
