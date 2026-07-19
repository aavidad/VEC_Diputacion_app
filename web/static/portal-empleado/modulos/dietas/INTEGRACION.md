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

const modulo = await montarModuloDietas({
  raiz,
  contextoActor,
  capacidades,
  adaptador,
  descargarRecibo,
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
- `confirmarOperacion(descriptor)`: muestra la confirmación institucional y
  resuelve exclusivamente con `true` cuando la persona confirma.
- `descargarRecibo(descriptor)`: entrega al conector documental el descriptor
  `vec.documentos.recibo.dietas.v1`. El QR contiene solo una referencia opaca de
  cotejo, nunca datos personales.
- `anunciar(mensaje, nivel)`: comunica resultados o errores mediante el sistema
  común de avisos del portal.

`desmontar()` elimina manejadores y contenido al abandonar el módulo.

## Límite de la demostración

`datos-presentacion.js` y `adaptador-presentacion.js` son los únicos elementos
demostrativos. Usan exclusivamente datos sintéticos, memoria volátil y recibos
marcados sin efectos administrativos. La vista, el presentador, el contrato, el
catálogo i18n y los estilos son reutilizables en producción.

Para producción se sustituye el adaptador de presentación por uno autenticado
contra la API interna. No se modifican la vista ni sus rutas. El nuevo adaptador
debe garantizar, como mínimo:

1. autorización de cada consulta y comando por capacidad y ámbito;
2. titularidad del expediente o habilitación administrativa expresa;
3. validación en servidor de importes, rutas, estados y transiciones;
4. idempotencia o versión esperada en operaciones mutables;
5. recibo probatorio y auditoría antes de confirmar el éxito;
6. ausencia de localizaciones en respuestas sin `dietas.ruta.read`.

La descarga productiva debe usar el servicio documental común y el cotejo debe
resolverse mediante petición `POST`; la ruta estática con
`presentacion=rrhh` existe solo en DEMO.

## Verificación

```sh
node --test web/static/portal-empleado/modulos/dietas/dietas.test.mjs
```

Las pruebas cubren identidad compartida, mínimo privilegio, aislamiento de
rutas, estados canónicos, acciones volátiles, recibos verificables sin datos
personales, i18n, accesibilidad, diseño adaptable y separación de fixtures.
