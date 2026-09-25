# Dietas en el Portal del Empleado

La vista interna se monta con `montarVistaRecorridosDietas` de
`vista-recorridos.js`. Recibe clientes HTTP inyectados por la composición del
portal. La identidad, la relación de servicio y la competencia se resuelven y
revalidan en el servidor; el navegador no envía actor, perfil, persona ni unidad
como autoridad libre. Todos los clientes usan el mismo origen, no crean cookies
ni emplean almacenamiento web.

## Montaje

```js
montarVistaRecorridosDietas(raiz, {
  clienteBorradores,
  clienteAsignacion,
  clienteRectificacion,
  clienteCircuito,
  clienteRectificacionAdmin,
  clienteCatalogoCompetente, // solo cuando Personal publique catálogo autorizado
  calculadorRuta,
  visorRuta,
  relacionesAutorizadas,    // GET Personal /relaciones-dietas
  fechaReferenciaPersonal,  // fecha de esa misma respuesta
  estadoRelaciones,
  traducir,
  anunciar,
  registrarDesmontar,
});
```

El portal interno (`componerDietasInternas`) compone hoy `clienteBorradores`,
`clienteAsignacion`, `calculadorRuta`, `visorRuta` y las relaciones que
Personal acredita antes de montar. Si Personal deniega por falta de empleado
canónico (`empleado_no_disponible`) o por ambigüedad (`empleado_ambiguo`), la
vista lo dice y deja el alta y el envío cerrados. `clienteCircuito`,
`clienteRectificacion` y `clienteRectificacionAdmin` todavía no se componen:
sin ellos no aparecen los pasos del circuito ni la corrección de la asignación.

`clienteBorradores` usa `/api/vec/dietas/comisiones` para alta, consulta,
edición, borrado lógico y envío. POST crea cabecera v1; PUT conserva documento
v2 con vehículo propio, rutas independientes, ajustes motivados, otros conceptos
y aceptación de uno o todos los tramos del grupo acreditado por Personal. Cada
mutación se confirma exclusivamente con su recibo. `numero_documento` y
`fecha_apertura` proceden de PostgreSQL y son la identificación visible cuando
la proyección v2 los entrega.

`clienteAsignacion` consulta primero
`GET /api/vec/personal/relaciones-dietas`; la composición pasa el par
`relacion_ref`/`unidad_ref` y su `fecha_referencia` sin inferirlos de otros
datos. La vista consulta la asignación D7 propia mediante ese par. Si falta la
proyección o falla Personal, edición dependiente del grupo y envío quedan
cerrados. Los nombres de centro, unidad y validadores no se inventan a partir
de referencias; se presentan como no disponibles hasta que Personal proyecte
etiquetas autorizadas.

`clienteRectificacion` consulta y solicita rectificación textual D7c, sin
conceder al empleado edición directa de la asignación. La vista conserva el
recibo y refresca la asignación vigente tras confirmar o recuperar una
solicitud. La bandeja administrativa usa otro cliente y autorización propia;
confirmar una corrección requiere además el catálogo competente autorizado de
centros y personas. Sin ese catálogo, la confirmación queda deshabilitada.

`clienteCircuito` consulta bandejas por revisión, autorización, liquidación y
fiscalización y envía decisiones con motivo cuando se devuelve. Cada etapa
exige permiso V3 propio; mostrar una pestaña no concede acceso. Una comisión
fiscalizada no se presenta como pagada.

## Ruta y mapa

`crearCalculadorRutasDietasHTTP` consume `GET /api/vec/dietas/route-catalog`
y `POST /api/vec/dietas/road-route`. El formulario ofrece un itinerario
principal con hasta diez paradas y, en edición con vehículo propio, hasta ocho
rutas independientes con el mismo máximo por ruta. El componente
`vista-mapa-comision.js` valida la geometría OSRM interna de los códigos exactos
antes de permitir guardar y cancela peticiones al cambiar de ruta o desmontar.
`crearVisorRutaDietas({ permitirTeselas: true })` pide teselas únicamente a
`/tiles/osm/{z}/{x}/{y}.png` del mismo origen. El servidor recalcula el
documento antes de persistir importes y conserva la versión del grafo.

El país España usa la tabla nacional provisional del corte. Elegir otro país
muestra que falta cálculo gobernado y deshabilita guardar; no aplica la tarifa
española como sustitución. El nivel de detalle bajo, medio o alto modifica
solo el desglose visible, nunca el grupo, la tarifa o el resultado de la API.
El documento v2 muestra el total orientativo singular del grupo acreditado,
separando dietas, kilometraje y otros conceptos; no lo presenta como
liquidación.

Otros medios y otros gastos (D5) llevan tipo del catálogo versionado que
sirve `route-catalog` (`otros_gastos`), fecha dentro de la comisión,
descripción, importe y justificante por referencia y huella SHA-256. El
fichero queda en custodia de la persona: el navegador solo lo lee, si ella lo
elige, para calcular la huella, y no lo envía. PostgreSQL (Dietas 000009)
vuelve a exigir catálogo, fechas y justificante al guardar. Las líneas
guardadas antes de D5 se siguen leyendo, pero para volver a guardar hay que
completarlas.

Un documento devuelto por cualquier paso del circuito (D6) llega con
`devolucion` (etapa, motivo, versión devuelta y fecha), que PostgreSQL
(Dietas 000010) toma de la historia mientras está devuelto o en corrección.
La ficha lo muestra y ofrece «Corregir» y «Reenviar a revisión del
administrativo»; no ofrece eliminarlo, y PostgreSQL también lo impide. La
corrección y el reenvío usan las mismas rutas PUT y `POST …/enviar`, con
clave de idempotencia y versión esperada. A qué paso debe volver está
pendiente de RRHH (pregunta 51 de `dudas.md`).

Quien revisa un reenvío recibe en el documento del circuito la devolución
anterior (`devolucion`: etapa, motivo, versión y fecha, nunca quién la hizo)
y el detalle de la bandeja la muestra como «Reenvío» y «Motivo de la
devolución». Los textos libres (motivo de devolución, centro y unidad) no
admiten blancos de borde: el cliente recorta y valida con `texto-dietas.js`
el mismo conjunto que Go rechaza (lo que quitan `trim()` y
`strings.TrimSpace`). El fixture `testdata/documento_propio_v2.json` del
adaptador HTTP es el JSON real de Go que valida `contrato-documento-go.test.mjs`.

## Comprobación

Las pruebas Node del directorio cubren clientes, ruta, mapa, formulario,
aceptación, recibos, cancelación y bandejas. La comprobación de PostgreSQL,
autorización V3 y navegador corresponde al corte integrado y se documenta
fuera de este archivo.
