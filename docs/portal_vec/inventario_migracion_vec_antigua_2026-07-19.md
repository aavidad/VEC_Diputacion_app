# Inventario de migración desde la VEC antigua

Fecha de corte: **19 de julio de 2026**
Ámbito de este corte: carcasa común del Portal del Empleado y, con mayor
detalle, el módulo de Dietas.
Estado: documento de contraste técnico; no acredita puesta en producción ni
aceptación funcional de RRHH.

## Conclusión ejecutiva

El portal nuevo **no es todavía funcionalmente equivalente** a la VEC antigua
en Dietas. La nueva implementación sí proporciona un recorrido modular,
navegable y accionable para consultar comisiones, crear un borrador sencillo,
enviarlo a validación y descargar un recibo PDF con QR. Sin embargo, aún no ha
migrado el recorrido enriquecido de rutas, mapa, cálculo por horarios,
justificantes, calendario ni explotación anual.

Esta diferencia no impide presentar Bolsa, que es el alcance urgente. Sí
impide afirmar que Dietas está migrado al 100 % o que conserva ya todas las
capacidades de la aplicación antigua. Para la presentación, Dietas debe
describirse como **recorrido funcional inicial de presentación**.

## Significado de los estados

| Estado | Criterio |
| --- | --- |
| **Migrado** | Existe en la nueva superficie modular, es visible y tiene contrato o prueba enfocada. Puede seguir usando un adaptador sintético si está inequívocamente rotulado como demostración. |
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
| Revisión visual automatizada de las 36 vistas actuales | **En curso hoy** | El manifiesto histórico sólo cubría 32 vistas. En este corte se está adaptando la revisión al título actual, a los diez grupos plegables y al nuevo asistente de llamamientos. No se considerará cerrada hasta obtener una ejecución integral verde en 1440, 1024 y 390 píxeles. |

## Inventario detallado de Dietas

| Capacidad | VEC antigua | Portal nuevo | Estado | Prioridad para la presentación |
| --- | --- | --- | --- | --- |
| Listado, filtros y selección de expedientes | Disponible | Disponible en `modulos/dietas/vista.js:127-152,175-199` | **Migrado** | Cubierta. |
| Detalle de comisión, importes, etapas e historial | Disponible | Disponible en `modulos/dietas/vista.js:87-124` | **Migrado** | Cubierta. |
| Indicadores de expedientes, pendientes, kilómetros, devengado y pagado | Disponible | Disponible en `modulos/dietas/vista.js:185-193` | **Migrado** | Cubierta, siempre como dato DEMO. |
| Crear borrador de comisión | Formulario completo | Formulario básico con fecha, motivo, origen, destino, kilómetros e importes manuales en `modulos/dietas/vista.js:155-172` | **Migrado parcialmente** | Suficiente para enseñar el cascarón, no para afirmar cálculo oficial. |
| Enviar borrador a validación | Disponible | Acción conectada al adaptador de presentación en `modulos/dietas/vista.js:292-380` | **Migrado en presentación** | Cubierta con recibo y sin efectos reales. |
| Ruta con varias paradas | Disponible: alta de paradas y cálculo en `web/static/app.js:2752-3055` | Sólo origen, destino y kilómetros manuales | **Pendiente real** | Alta si se enseña Dietas como producto heredado. No bloquea la demo de Bolsa. |
| Cálculo por carretera mediante motor interno | Cliente antiguo hacia `/api/vec/dietas/road-route` en `web/static/app.js:6063-6076`; adaptador HTTP endurecido en `internal/vec/adapters/httpapi/dietas_road_route.go:66-147` | No importado ni invocado por el módulo nuevo | **Pendiente real** | Alta para una Dietas creíble; se puede excluir del guion urgente. |
| Mapa del recorrido | Panel y visor local en `web/static/app.js:5768-5784,6257-6408` | No hay panel ni mapa | **Pendiente real** | Alta visual si los asistentes conocen la VEC antigua. |
| Cálculo de dieta por fechas, horas, media/completa y noches | Disponible en `web/static/app.js:2566-2640` | Los conceptos se introducen manualmente; el adaptador sólo calcula kilómetros por tarifa y suma conceptos | **Pendiente real** | Alta funcional, no necesaria para mostrar Bolsa. |
| Calendario mensual y selección de días | Disponible en `web/static/app.js:863-1009,1152` | No disponible | **Pendiente real** | Media. |
| Adjuntar y previsualizar justificantes | Disponible mediante imagen/PDF en `web/static/app.js:2660-2750,2831-2846` | Sólo se muestra el contador de justificantes; no existe alta ni descarga | **Pendiente real** | Alta si se recorre la tramitación completa. |
| Historial mensual | Disponible | Tabla mensual disponible en `modulos/dietas/vista.js:145-152` | **Migrado** | Cubierta. |
| Estadística, gráfico y tabla anual | Disponible en `web/static/app.js:674-743,1294-1374` | No disponible | **Pendiente real** | Media o baja para la presentación urgente. |
| Informe anual PDF | Disponible en `web/static/app.js:1190-1229` | El PDF nuevo es un recibo de actuación, no el informe anual | **Pendiente real** | Media; no confundir ambos documentos. |
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
2. **Teselas locales o visor vectorial local.** Si se usa Leaflet, las teselas
   se sirven desde una ruta interna como `/tiles/osm/{z}/{x}/{y}.png`. Deben
   documentarse versión, fuente, licencia y atribución. No se considera
   conectado mientras los recursos Leaflet y el servicio de teselas no formen
   parte verificable del despliegue.
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
- Dietas puede demostrar listado, detalle, borrador manual, envío a validación
  y recibo PDF/QR.
- La ausencia de mapa, OSRM, calendario, justificantes y explotación anual debe
  quedar fuera del discurso de capacidades terminadas.

### No aceptable

- Presentar Dietas como migrado al 100 % o equivalente a la VEC antigua.
- Describir los kilómetros manuales de la demostración como cálculo oficial.
- Confundir el recibo PDF de una actuación con el informe anual de Dietas.
- Afirmar integración real con Nóminas, aprobación de jefatura, firma,
  registro, almacenamiento o auditoría durable.

## Evidencia del corte

- Las pruebas JavaScript enfocadas de las superficies revisadas terminaron con
  **200 casos correctos y cero fallos** el 19 de julio de 2026.
- Las 36 rutas actuales pudieron abrirse en una inspección de navegador sin
  errores JavaScript observados, pero esta inspección no sustituye el recorrido
  completo de cada control.
- La primera ejecución fresca del manifiesto histórico produjo hallazgos por
  desajuste del propio revisor: título antiguo del lanzador, menús que ahora
  están agrupados en acordeones y selector anterior del llamamiento. Ese
  resultado no demuestra 540 defectos del portal; demuestra que la evidencia
  automática había quedado obsoleta. La actualización de ese contrato de
  revisión figura **en curso hoy** y deberá ejecutarse de nuevo hasta verde.

Este inventario debe actualizarse cuando una capacidad cambie de estado. Ningún
elemento pasa a **Migrado** sólo porque exista código heredado, un adaptador
sintético o una pantalla sin conexión verificable.
