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

Otros medios y otros gastos reciben descripción e importe. No se pide al
empleado escribir una referencia o huella documental. Cuando exista custodia
autorizada, el servidor podrá conservar su referencia y SHA-256 como pareja;
la descripción por sí sola no acredita un fichero.

## Comprobación

Las pruebas Node del directorio cubren clientes, ruta, mapa, formulario,
aceptación, recibos, cancelación y bandejas. La comprobación de PostgreSQL,
autorización V3 y navegador corresponde al corte integrado y se documenta
fuera de este archivo.
