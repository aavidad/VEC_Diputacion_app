---
name: programar-interfaz-vec
description: Conecta pantallas administrativas VEC con contratos y recibos reales. Usar para formularios, estados de carga/error, bandejas, seguimiento, descargas o adaptación móvil.
---

# Programar interfaz VEC

Inspeccionar web/static/portal-empleado/modulos/contratacion-temporal y su transporte,
contratos, renderizado e i18n. Reutilizar esos componentes y el estilo administrativo.

1. Fijar con backend la API y el estado observable antes de escribir consumidor/UI.
2. Asignar por separado contrato, cliente y montaje si pueden avanzar sin solaparse.
3. Representar carga, vacío, error, conflicto y confirmación con acciones comprensibles.
   Mostrar confirmación sólo tras una respuesta válida y conservar su recibo.
4. Deshabilitar acciones todavía no conectadas con una explicación concreta. Mantener
   esa condición también en derivados, avance automático y ejecutor de comandos.
5. Cancelar peticiones al cambiar de expediente o desmontar. Evitar que respuestas
   tardías actualicen otro expediente. No introducir cookies ni almacenamiento web.
6. Descargar los bytes originales con nombre y tipo correctos. No volver a serializar
   un documento cuando se necesita conservar su huella exacta.
7. Comprobar el cambio visible en escritorio y móvil cuando afecte a disposición.
   Usar textos castellanos e i18n existente; revisar desbordamiento y acciones accesibles.

No mostrar un borrador como firmado ni una ficha descargada como enviada a GINPIX.
Entregar API utilizada, montaje, archivos y comprobación funcional. Usar
$orquestar-vec para repartir componentes dentro del ámbito recibido.
