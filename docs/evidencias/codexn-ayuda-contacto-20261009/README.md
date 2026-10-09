# Ayuda sobre el contacto de un llamamiento

El cambio revisado en `194f5968bc7c5b511a320d227dad4bb08e8a01b8` explica en la ayuda «?» de Mi Bolsa y del seguimiento de RRHH cuándo empieza el plazo de respuesta. El envío del correo no lo inicia: el plazo empieza con el contacto efectivo registrado por RRHH. La explicación está en los catálogos de español e inglés.

## Recorrido real de Mi Bolsa

Las capturas [móvil en español](mi-es-390.png) y [escritorio en inglés](mi-en-1440.png) proceden de Chrome con la aplicación local y PostgreSQL 18 desechable. En ambos casos, `GET /api/vec/bolsa/mi-bolsa` y `GET /api/vec/bolsa/mi-bolsa/historial` devolvieron `200`. La ayuda se abrió en la pantalla real; no hubo errores JavaScript, desbordamiento horizontal, cookies ni almacenamiento web. Las consultas auxiliares de imagen y preferencias devolvieron `401` en esa sesión de prueba.

## Seguimiento de RRHH: vista aislada

Estas capturas muestran el componente con los estilos reales del portal, pero no un recorrido conectado a la API ni a PostgreSQL. La ficha de candidatos de la prueba local no permitió alcanzar el panel de seguimiento desde la navegación normal. Por eso las imágenes se identifican como **VISTA AISLADA**:

- [Sin llamamiento, 390 px](rrhh-vista-aislada-sin-390.png).
- [Cargando, 390 px](rrhh-vista-aislada-cargando-390.png).
- [Error, 390 px](rrhh-vista-aislada-error-390.png).
- [Estado normal en inglés, 1440 px](rrhh-vista-aislada-normal-en-1440.png).

En las cuatro vistas, la ayuda abierta queda dentro del panel, el foco permanece en «?» y no hay desbordamiento horizontal. No se observaron errores JavaScript. Estas capturas comprueban el recorte detectado por la revisión independiente; no acreditan que RRHH pueda abrir ese seguimiento en el escenario conectado.

## Comprobaciones y límite de rendimiento

La revisión independiente dio `GO` al commit indicado. La puerta completa `verificar_calidad.sh` terminó verde: 3669 pruebas JavaScript, comprobación de catálogos ES/EN, manifiestos y controles restantes. Semgrep local sobre el JavaScript cambiado no encontró hallazgos. Este corte no cambia Go, SQL ni configuración.

En el ensayo local se midieron **64,106 ms** para un `GET /api/vec/bolsa/bolsas`. Es una medición puntual de esa petición: no representa el tiempo de toda la pantalla ni un percentil de rendimiento.
