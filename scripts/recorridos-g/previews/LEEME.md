# Recorrido local de las previews de Dietas

Este guion abre **informes** y **catálogo de tarifas** con datos sintéticos. Sirve para revisar su presentación y sus límites. No consulta el servidor VEC, no autentica a nadie y no acredita publicación ni un cálculo de tarifas aprobado.

Necesita una **copia fuente sintética controlada** que contenga los cuatro archivos de `scripts/recorridos-g/previews/paginas/dietas/`, los estáticos de `web/static` y tres JSON: `data/demo/dietas/informes.json`, `data/demo/dietas/catalogo-rrhh.json` y `data/catalogos/dietas/informes-ejemplo-v1.json`. Las páginas de prueba viven fuera de `web/static`; el producto no las sirve. No pase como fuente un repositorio con secretos o datos privados.

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

Comprueba ambas páginas en español e inglés a 1440 y 390 px y con zoom nativo al 200 %. En informes, el navegador recibe una variante sintética de configuración y datos enlazados en versión 2: fecha de liquidación, estado liquidado y concepto de manutención. El filtro debe mostrar exactamente `DI-002` y `DI-006`, con **42,50 € de importe incluido** según el idioma. La variante vive solo en la interceptación del navegador: no modifica los tres JSON originales.

En catálogo, prepara por separado una propuesta local de 42,50 € y comprueba que no se publica. También revisa el aviso sintético, filtros, exportar e imprimir cerrados, foco por teclado, desbordamiento y scroll. En escritorio de al menos 1024 píxeles CSS, la ventana no debe desplazarse: el contenido largo usa el panel principal. En móvil y al 200 %, si el contenido supera el espacio disponible, debe existir un desplazamiento alcanzable. Comprueba cero errores JavaScript, cookies, almacenamiento web, descargas y peticiones externas.

La prueba de reintento devuelve un único `503` **solo desde el navegador al JSON de catálogo**, sin tocar el archivo ni un servidor VEC. Cualquier otro `503` corta el recorrido. El resultado informa qué pasó; una dependencia ausente nunca cuenta como recorrido verde.

El servidor escucha en `127.0.0.1` con un puerto efímero. Solo sirve cuatro rutas exactas de preview, los estáticos de `web/static` y los tres JSON indicados. Deniega otros métodos, rutas `/api` y enlaces simbólicos. El navegador bloquea WebSocket, otros orígenes y cualquier ruta fuera de la lista. Se cierra y elimina el perfil temporal de Chrome al terminar.

Ejecute Chrome dentro de un aislamiento de sistema operativo: fuente y herramientas de solo lectura, red externa cerrada, bucle local aislado, un único espacio temporal escribible y límites de CPU, memoria, procesos y tiempo. El modo `plan` no necesita navegador ni red.

Prueba focal del guion:

```sh
node --test scripts/recorridos-g/previews/recorrer.test.mjs
```

Este recorrido es una revisión de las previews. El montaje real, los permisos, las respuestas del backend y la persistencia requieren sus comprobaciones propias cuando estén instalados.
