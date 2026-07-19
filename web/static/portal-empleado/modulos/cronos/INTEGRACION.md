# Integración de Cronos en el Portal del Empleado

La vista, el contrato y el presentador son definitivos. No conocen `fetch`,
cookies, almacenamiento del navegador ni usuarios propios. La identidad procede
de la misma sesión interna que ya utiliza Bolsa y todas las operaciones fallan
cerradas si falta una capacidad explícita.

## Cableado del portal

1. Cargar `cronos.css` desde la plantilla común.
2. Cuando la vista principal sea `cronos`, entregar sin clonar el mismo
   `ContextoActor` validado que consumen Bolsa y Dietas, junto con las
   capacidades de Cronos como argumento separado.
3. Obtener el envelope `vec.cronos.area-personal.v1` desde el adaptador de API.
4. Crear `crearPresentadorCronos({ contextoActor, capacidades, datos, ejecutor,
   descargarRecibo, mensajes })`, escribir
   `presentador.renderizar()` en `#espacio-trabajo` e instalar la delegación con
   `presentador.instalarEventos({ raiz, alCambiar, anunciar })`.
5. Desmontar los eventos con la función devuelta al abandonar el módulo.

El `ejecutor` recibe `(comando, { identidad, datos })`. Los comandos son:

- `{ tipo: "registrar_fichaje", movimiento: "entrada" | "salida" |
  "inicio_pausa" | "fin_pausa" }`.
- `{ tipo: "solicitar_permiso", permiso_id, desde, hasta, cantidad, motivo,
  documento_ref }`.

La navegación interna usa `data-cronos-destino` y desplazamiento local; no
escribe el `hash`, que pertenece al router del portal. El catálogo `i18n.js`
contiene todo el texto de interfaz. Estados y movimientos llegan como códigos
canónicos y se traducen exclusivamente al renderizar.

Los fichajes, eventos y recibos transportan siempre `instante` ISO-8601 UTC. La
vista los localiza con `Intl.DateTimeFormat` en `Europe/Madrid` y muestra la
zona; una fecha/hora ya formateada nunca es fuente de verdad.

Las solicitudes conservan `desde`/`hasta` como fechas civiles ISO `AAAA-MM-DD`.
Las cantidades usan la unidad base del dominio: días enteros para `dia` y
minutos enteros para `minuto`; la vista transforma los minutos a `HH:MM h`.

El servidor debe resolver la relación entre `actor_ref` y persona empleada. El
navegador nunca envía un identificador de otra persona.

## Presentación descartable

Solo estos archivos son de presentación y deben excluirse de la imagen de
producción:

- `datos-presentacion.js`
- `adaptador-presentacion.js`

Se sustituyen, respectivamente, por la consulta a la API interna y su puerto de
comandos. La vista no cambia. Las operaciones de presentación son volátiles y
los recibos comienzan por `DEMO-`.

## Documentos

Cada botón «Descargar recibo» prepara mediante `prepararDescriptorRecibo` el
mismo contrato institucional que Dietas y lo entrega a la dependencia
`descargarRecibo(descriptor)`. El descriptor contiene referencia opaca, logo,
marca DEMO cuando procede y QR de cotejo sin identidad ni otros datos sensibles.
La composición transforma el descriptor en PDF; Cronos no genera el PDF ni el
QR ni abre una descarga por su cuenta.
