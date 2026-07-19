# Integración del módulo Dietas

Dietas comparte la identidad resuelta por el núcleo del Portal del Empleado. El
módulo no autentica, no mantiene usuarios propios y no usa cookies ni
almacenamiento del navegador. Debe recibir la misma instancia validada de
`ContextoActor` que usan el resto de módulos internos.

## Composición

```js
const adaptador = crearAdaptadorDietasPresentacion({
  contextoActor,
  capacidades,
});

const calculadorRuta = crearCalculadorRutasDietasPresentacion({
  contextoActor,
  capacidades,
});

const modulo = await montarModuloDietas({
  raiz,
  contextoActor,
  capacidades,
  adaptador,
  calculadorRuta,
  descargarRecibo,
  visorRuta,
  confirmarOperacion,
  anunciar,
});
```

Los nombres de capacidades se exportan desde `contrato.js`. La ausencia de una
capacidad implica denegación; no se infieren permisos a partir del rol ni de la
interfaz.

## Puertos

- `adaptador.obtenerDatos()`: devuelve el panel conforme a
  `vec.dietas.portal.v1`, ya proyectado para el actor y sus capacidades.
- `adaptador.ejecutar(comando)`: acepta `crear_borrador` y
  `enviar_validacion`, y devuelve el nuevo panel. El adaptador productivo debe
  aplicar autorización y control de concurrencia también en servidor.
- `calculadorRuta.obtenerCatalogo()`: devuelve el catálogo territorial conforme
  a `vec.dietas.catalogo-rutas.v1`. El DTO público no contiene coordenadas.
- `calculadorRuta.calcular(solicitud)`: recibe
  `vec.dietas.solicitud-ruta.v1` y devuelve
  `vec.dietas.calculo-ruta.v1`, con entre una y tres alternativas y sus tramos.
  La vista nunca llama directamente a OSRM.
- `confirmarOperacion(descriptor)`: muestra la confirmación institucional y
  resuelve exclusivamente con `true` cuando la persona confirma.
- `descargarRecibo(descriptor)`: entrega al conector documental el descriptor
  `vec.documentos.recibo.dietas.v1` o el resumen anual
  `vec.documentos.resumen-anual.dietas.v1`. El QR contiene solo una referencia
  opaca de cotejo, nunca datos personales.
- `visorRuta.montar({ raiz, descriptor })`: recibe geometría ya autorizada y
  solo admite OpenStreetMap servido en la red interna mediante
  `/tiles/osm/{z}/{x}/{y}.png`. El navegador no llama a OSRM, geocodificadores,
  teselas públicas ni otros terceros. Sin la biblioteca Leaflet local conserva
  el croquis SVG sintético, sin atribuirlo a OpenStreetMap ni usarlo para
  liquidar kilómetros.
- `anunciar(mensaje, nivel)`: comunica resultados o errores mediante el sistema
  común de avisos del portal.

`desmontar()` elimina manejadores y contenido al abandonar el módulo.

## Límite de la demostración

`datos-presentacion.js`, `adaptador-presentacion.js` y
`calculador-rutas-presentacion.js` son los únicos elementos demostrativos. Usan
exclusivamente datos sintéticos, memoria volátil y recibos marcados sin efectos
administrativos. El cálculo DEMO simula el contrato del OSRM interno, queda
marcado como no liquidable y no usa red, cookies ni almacenamiento local. La
vista, los presentadores, el contrato, el catálogo i18n y los estilos son
reutilizables en producción.

Para producción se sustituye el adaptador de presentación por uno autenticado
contra la API interna. No se modifican la vista ni sus rutas. El nuevo adaptador
debe garantizar, como mínimo:

1. autorización de cada consulta y comando por capacidad y ámbito;
2. titularidad del expediente o habilitación administrativa expresa;
3. recálculo y validación en servidor de importes, rutas, alternativa elegida,
   motivos de desvío, ajustes por tramo, estados y transiciones;
4. idempotencia o versión esperada en operaciones mutables;
5. recibo probatorio y auditoría antes de confirmar el éxito;
6. ausencia de localizaciones en respuestas sin `dietas.ruta.read`.

El adaptador productivo del puerto de rutas debe delegar en la API interna (el
backend dispone del cálculo `POST /api/vec/dietas/road-route`), que aplica RBAC,
consulta OSRM dentro de la red corporativa y devuelve geometría ya proyectada
según `dietas.ruta.read`. El servidor debe conservar la versión del motor y la
referencia opaca del cálculo, recalcular antes de liquidar y auditar actor,
idempotencia, alternativa, justificación y ajustes. El cliente HTTP usa
`credentials: "omit"`: no envía cookies, certificados cliente ni credenciales
HTTP del navegador. En consecuencia, no usa `globalThis.fetch` implícitamente y
exige un cliente inyectado por el futuro conector de identidad nativo o por una
mediación corporativa autenticada sin cookies. Kerberos/SPNEGO o mTLS **no se
presuponen transportados por Fetch**. Hasta que Sistemas valide y suministre ese
conector, la composición productiva permanece bloqueada y falla cerrada. El
servidor vuelve a autorizar cada operación. Si la proyección territorial no
está disponible, falla sólo la herramienta de rutas: el listado y detalle de
Dietas permanecen operativos.

Leaflet 1.9.4 ya se sirve
desde el propio portal como dependencia fijada, con licencia, procedencia y
huellas verificables. Para activar el mapa real falta que Sistemas importe y
publique las teselas OSM internas en la ruta anterior y habilite expresamente
su uso en la composición productiva. Hasta entonces la vista muestra el
croquis SVG DEMO y no intenta descargar teselas. La atribución de OpenStreetMap
y OpenMapTiles solo se hace visible cuando el mapa real se monta.

La descarga productiva debe usar el servicio documental común de servidor:
generación, firma/sello cuando proceda, custodia, auditoría y cotejo por `POST`.
La ruta estática con `presentacion=rrhh` y el generador PDF del navegador existen
solo en DEMO.

## Sustitución de adaptadores

| Función | Presentación | Producción |
|---|---|---|
| Expedientes y comandos | `adaptador-presentacion.js`, memoria volátil | API interna autorizada, persistencia y recibo probatorio |
| Catálogo y ruta | catálogo provincial y geometría sintética no liquidable | catálogo autoritativo + API/OSRM internos; nunca cálculo en la vista |
| Mapa | croquis SVG; Leaflet local si está desplegado | Leaflet local + `/tiles/osm/`; sin CDN ni salida a Internet |
| PDF anual y recibos | generador común del navegador, marca DEMO | servicio documental de servidor, firma/custodia/cotejo |

La vista, el presentador, el contrato, i18n y los estilos no cambian al efectuar
estas sustituciones.

## Verificación

```sh
node --test web/static/portal-empleado/modulos/dietas/dietas.test.mjs
```

Las pruebas cubren identidad compartida, mínimo privilegio, aislamiento de
rutas, estados canónicos, acciones volátiles, recibos verificables sin datos
personales, i18n, accesibilidad, diseño adaptable y separación de fixtures.
