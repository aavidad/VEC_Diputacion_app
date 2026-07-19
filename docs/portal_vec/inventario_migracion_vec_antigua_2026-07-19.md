# Inventario de migración desde la VEC antigua

Fecha de corte: **19 de julio de 2026**
Ámbito de este corte: carcasa común del Portal del Empleado y, con mayor
detalle, el módulo de Dietas.
Estado: documento de contraste técnico; no acredita puesta en producción ni
aceptación funcional de RRHH.

## Conclusión ejecutiva

El portal nuevo **no es todavía funcionalmente equivalente** a la VEC antigua
en Dietas, pero ya recupera su recorrido principal de desplazamiento: catálogo
provincial, ruta multiparada de ida y vuelta, cálculo con alternativas,
justificación de desvíos, ajustes por tramo, mapa/croquis, fechas y horas,
vehículo, tarifa y desglose económico. También permite consultar comisiones,
guardar el recorrido completo en un borrador demostrativo, enviarlo a
validación y descargar recibos PDF con QR y un resumen anual. Siguen pendientes
los justificantes, calendario, espacio gestor y persistencia/auditoría/firma
productivas.

La demostración de rutas es deliberadamente sintética, reproducible, volátil y
no liquidable. El adaptador productivo mismo-origen hacia el catálogo gobernado
y OSRM interno está implementado, pero no se activará hasta que Sistemas
publique la proyección de datos autorizada y el despliegue interno de teselas y
motor de rutas. Por ello Dietas debe describirse como **recorrido funcional de
presentación con arquitectura productiva preparada**, no como módulo en
producción ni migrado al 100 %.

## Significado de los estados

| Estado | Criterio |
| --- | --- |
| **Migrado** | Existe en la nueva superficie modular, es visible y tiene contrato o prueba enfocada. Puede seguir usando un adaptador sintético si está inequívocamente rotulado como demostración. |
| **Verificado en este corte** | La evidencia automatizada y visual indicada se ejecutó de nuevo sobre el árbol actual y terminó sin hallazgos. No equivale a aceptación de usuario ni a puesta en producción. |
| **En curso hoy** | Hay trabajo observable en el corte de trabajo del 19 de julio, pero todavía no existe evidencia integral verde. |
| **Pendiente real** | La capacidad sigue siendo necesaria, pero no está conectada y visible en el módulo nuevo. |
| **Retirado intencionadamente** | No se reutiliza la implementación antigua por seguridad, arquitectura o calidad. La necesidad de negocio puede seguir pendiente de una sustitución correcta. |

## Inventario general

| Capacidad de la VEC antigua | Estado en el portal nuevo | Evidencia o decisión |
| --- | --- | --- |
| Carcasa institucional, logotipo, navegación y tema común | **Migrado** | Las superficies nuevas comparten recursos del portal y el logotipo institucional. El código monolítico antiguo no gobierna la presentación. |
| Identidad común para Bolsa, Cronos y Dietas | **Migrado en presentación** | La composición entrega a los módulos un mismo contexto de actor y capacidades; no equivale todavía a identidad productiva de Sistemas. |
| Módulo Cronos visible con fichajes, permisos, historial y recibos | **Migrado como recorrido inicial** | Tiene presentador, vista, capacidades y puerto documental propios. No se declara aquí equivalencia completa con toda la VEC antigua. |
| Módulo Dietas visible | **Migrado como recorrido inicial** | Se monta desde la carcasa nueva y utiliza vista, presentador y adaptador separados. |
| Aplicación monolítica `web/static/app.js` como interfaz activa | **Retirado intencionadamente** | El artefacto de presentación elimina expresamente la aplicación antigua en `Dockerfile:35-41`; tampoco figura en `web/produccion.manifest`. Se conserva sólo como referencia de migración. |
| Estado de negocio guardado en `localStorage` | **Retirado intencionadamente** | La antigua VEC guardaba hojas DEMO con `DIETAS_SHEETS_STORAGE_KEY` en `web/static/app.js:746-798`. La presentación nueva usa memoria volátil y producción deberá usar puertos autorizados. |
| Datos o efectos ficticios como sustitución silenciosa de una API real | **Retirado intencionadamente** | Los adaptadores de presentación sólo se cargan con el selector explícito de demostración; las operaciones se rotulan sin efectos administrativos. |
| Revisión visual automatizada de las vistas actuales | **Verificado en este corte** | 102/102 escenarios correctos en escritorio, portátil y móvil, incluido el recorrido de Dietas; 54 vistas y 48 flujos, sin hallazgos. |

## Inventario detallado de Dietas

| Capacidad | VEC antigua | Portal nuevo | Estado | Prioridad para la presentación |
| --- | --- | --- | --- | --- |
| Listado, filtros y selección de expedientes | Disponible | Disponible en `modulos/dietas/vista.js:127-152,175-199` | **Migrado** | Cubierta. |
| Detalle de comisión, importes, etapas e historial | Disponible | Disponible en `modulos/dietas/vista.js:87-124` | **Migrado** | Cubierta. |
| Indicadores de expedientes, pendientes, kilómetros, devengado y pagado | Disponible | Disponible en `modulos/dietas/vista.js:185-193` | **Migrado** | Cubierta, siempre como dato DEMO. |
| Crear borrador de comisión | Formulario completo | Formulario con fechas y horas, motivo, vehículo propio, tarifa, conceptos económicos y la ruta calculada completa | **Migrado en presentación** | Recorrido útil para la demostración; el servidor debe recalcular antes de liquidar. |
| Enviar borrador a validación | Disponible | Acción conectada al adaptador de presentación en `modulos/dietas/vista.js:292-380` | **Migrado en presentación** | Cubierta con recibo y sin efectos reales. |
| Ruta con varias paradas | Disponible: alta de paradas y cálculo | Hasta doce paradas, ida/vuelta, alternativas y selección justificada en `modulos/dietas/presentador-rutas.js` y `vista.js` | **Migrado en presentación** | Cubierta con cálculo sintético explícito y no liquidable. |
| Cálculo por carretera mediante motor interno | Cliente antiguo hacia `/api/vec/dietas/road-route`; backend endurecido existente | Adaptador productivo mismo-origen en `calculador-rutas-http.js` y adaptador aislado en `calculador-rutas-presentacion.js` | **Preparado; activación productiva pendiente** | La demo es ejecutable; producción requiere la proyección PDP y OSRM internos. |
| Mapa del recorrido | Panel y visor local | Croquis SVG siempre disponible y Leaflet 1.9.4 local preparado para teselas OSM internas | **Migrado en presentación** | Cubierta sin enviar coordenadas a terceros; las teselas se activan sólo tras desplegar el proxy interno. |
| Cálculo de dieta por fechas, horas, media/completa y noches | Disponible | Fechas y horas, kilómetros por tarifa, manutención, alojamiento y otros gastos; todavía no determina automáticamente media/completa/noches conforme a normativa | **Migrado parcialmente** | El recorrido visual y total están cubiertos; el motor normativo oficial sigue pendiente. |
| Calendario mensual y selección de días | Disponible en `web/static/app.js:863-1009,1152` | No disponible | **Pendiente real** | Media. |
| Adjuntar y previsualizar justificantes | Disponible mediante imagen/PDF en `web/static/app.js:2660-2750,2831-2846` | Sólo se muestra el contador de justificantes; no existe alta ni descarga | **Pendiente real** | Alta si se recorre la tramitación completa. |
| Historial mensual | Disponible | Tabla mensual disponible en `modulos/dietas/vista.js:145-152` | **Migrado** | Cubierta. |
| Estadística, gráfico y tabla anual | Disponible | Resumen anual agregado y actividad mensual disponibles; no se ha recuperado el gráfico histórico completo | **Migrado parcialmente** | Suficiente para la demo; gráfico avanzado pendiente. |
| Informe anual PDF | Disponible | Resumen anual PDF demostrativo y recibo de actuación diferenciados | **Migrado en presentación** | No confundir los dos contratos documentales ni presentarlos como firmados oficialmente. |
| Exportación anual Excel | Disponible como HTML con extensión `.xls` en `web/static/app.js:1174-1187` | No disponible | **Implementación antigua retirada; salida real pendiente** | Baja para el martes. Debe sustituirse por el conector documental/exportador común, no copiarse literalmente. |
| Recibo PDF institucional con logotipo y QR de comprobación | No era el mismo contrato documental | Disponible en `modulos/dietas/presentador.js:158-215` y probado en `modulos/dietas/dietas.test.mjs:219-239` | **Migrado y mejorado** | Cubierta. |
| Aprobación de jefatura y tramitación RRHH | La antigua VEC contenía acciones y estados de aprobación en `web/static/app.js:7203-7254` | La vista actual del empleado sólo envía a validación; no hay espacio de trabajo del gestor de Dietas | **Pendiente real** | Alta sólo si el guion promete el punto de vista gestor de Dietas. |
| Integración con nómina | La antigua demostración reflejaba importes de Dietas en pantallas de nómina | El módulo nuevo sólo muestra estado y referencia sintética de pago | **Pendiente real** | No prometer integración productiva. |
| Persistencia, auditoría durable, firma y registro de la comisión | La antigua VEC simulaba parte del flujo | El recorrido actual usa un adaptador volátil de presentación | **Pendiente real** | No bloquea una demo honesta; sí bloquea piloto o producción. |

## Criterio obligatorio para OSM, mapa y cálculo de rutas

La decisión recomendada mantiene la separación ya prevista en el dominio:

1. **OSRM interno y sin consultas externas ordinarias.** El navegador no debe
   enviar orígenes, destinos, horarios ni identidad a OpenStreetMap, Google,
   Mapbox u otro servicio público. El cálculo se realiza mediante un conector
   hacia un OSRM desplegado dentro de la infraestructura autorizada.
2. **Teselas locales o visor vectorial local.** Leaflet 1.9.4 ya se distribuye
   desde el propio portal, con licencia, procedencia y huellas verificables. Las
   teselas se sirven desde una ruta interna como
   `/tiles/osm/{z}/{x}/{y}.png`. Deben documentarse versión, fuente, licencia y
   atribución. No se consideran conectadas hasta que el servicio interno forme parte
   verificable del despliegue. Mientras tanto se usa el croquis SVG y no se
   producen peticiones de red fallidas ni salidas a Internet.
3. **Fallo cerrado.** Si OSRM o la matriz oficial no responden, se puede mostrar
   una representación informativa, pero no convertir una línea recta ni un
   valor sugerido en kilómetros liquidables.
4. **Trazabilidad del cálculo.** La comisión debe conservar itinerario,
   distancia aprobada, versión del grafo, versión de la tarifa, instante y
   motivo de cualquier corrección manual.
5. **Importación oficial antes de liquidar.** El estado actual declara
   `pendiente_importacion_completa` y
   `import_required_before_liquidation=true` en
   `internal/modules/dietas/routes.go:94-116`. Por tanto, las parejas semilla
   sirven para desarrollo y presentación, no para una liquidación real.
6. **No afirmar un Leaflet plenamente operativo en la VEC antigua.** Existe
   código para inicializar `window.L` y teselas locales, pero no se ha
   localizado en la superficie antigua un recurso que cargue la biblioteca.
   El código disponía de un visor SVG local como alternativa. Además, el enlace
   directo para abrir OSM se dejó deliberadamente sin destino para evitar la
   salida de coordenadas.

## Implementaciones antiguas que no deben copiarse literalmente

- Persistencia DEMO en `localStorage`.
- Ficheros adjuntos conservados como `DataURL` dentro del navegador. La
  capacidad debe volver mediante el puerto documental seguro, subida directa
  al almacén de objetos, cuarentena y análisis antivirus.
- Exportación Excel basada en HTML renombrado como `.xls`. Debe sustituirse por
  un formato real generado por el conector de salidas.
- Generación documental particular del módulo. PDF, QR, CSV, ODT, JSON y las
  demás salidas deben pasar por el puerto documental común.
- Llamadas desde el navegador a mapas, teselas o motores de rutas externos.
- Acoplamiento visual directo entre Dietas y Nóminas. La integración futura
  debe usar contratos, eventos o consultas autorizadas entre módulos.

## Criterio de aceptación para el martes

### Imprescindible

- Bolsa pública, área candidata y portal interno abren desde el lanzador.
- Los diez apartados de Bolsa y sus diecisiete rutas son navegables según el
  rol previsto.
- Llamamientos, ayuda, audio y recibos PDF/QR tienen al menos un recorrido
  ejecutable y claramente rotulado como demostración.
- No existen datos personales reales, efectos administrativos ni adaptadores
  sintéticos cargados de forma silenciosa.
- La revisión automatizada actualizada termina verde en escritorio, portátil y
  móvil, incluyendo `reglas`, `consulta`, Cronos y Dietas.

### Aceptable como alcance limitado

- Cronos y Dietas pueden mostrarse como recorridos iniciales para explicar el
  portal modular común.
- Dietas puede demostrar listado, detalle, ruta multiparada, alternativas,
  ajustes, mapa/croquis, borrador, envío a validación y recibos PDF/QR.
- El cálculo de rutas y las cifras de la demostración deben describirse como
  sintéticos y no liquidables. Calendario, justificantes y espacio gestor siguen
  fuera del discurso de capacidades terminadas.

### No aceptable

- Presentar Dietas como migrado al 100 % o equivalente a la VEC antigua.
- Describir los kilómetros manuales de la demostración como cálculo oficial.
- Confundir el recibo PDF de una actuación con el informe anual de Dietas.
- Afirmar integración real con Nóminas, aprobación de jefatura, firma,
  registro, almacenamiento o auditoría durable.

## Evidencia del corte

- La batería JavaScript completa de `web/static` terminó con **239 casos
  correctos y cero fallos** el 19 de julio de 2026.
- Las pruebas del capturador terminaron con **15 casos correctos y cero fallos**.
- La revisión visual estricta terminó con **102/102 escenarios correctos**:
  54 vistas, 48 flujos y 102 capturas en 1440×1000, 1024×900 y 390×844,
  sin desbordamiento horizontal, controles sin nombre, identificadores
  duplicados, cookies, almacenamiento web ni errores de navegador.
- El recorrido manual de Dietas calculó Granada–Motril–Granada, guardó el
  borrador, seleccionó el expediente resultante y terminó sin errores de consola
  ni peticiones fallidas.
- Las pruebas enfocadas del servidor Go, el manifiesto productivo y la
  construcción/verificación de los artefactos Docker de producción y
  presentación terminaron correctamente.

Este inventario debe actualizarse cuando una capacidad cambie de estado. Ningún
elemento pasa a **Migrado** sólo porque exista código heredado, un adaptador
sintético o una pantalla sin conexión verificable.
