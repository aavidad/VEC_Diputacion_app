# Estado de la web de contratación temporal para RRHH

Fecha de corte: 23 de julio de 2026.

## Resultado

La superficie de presentación está compuesta dentro del Portal del Empleado y
reutiliza el tema, la identidad interna y la navegación comunes. El código
neutral no contiene datos sintéticos ni decide autorizaciones; el fixture y el
adaptador volátil solo se cargan con `?presentacion=rrhh`.

Este corte cubre el cuadro operativo, el alta O2-09B, cinco expedientes
coherentes, ocho fases, diecisiete tareas, documentos y auditoría. Las acciones
usan capacidades exactas, vuelven a validarse en el adaptador y quedan
visiblemente deshabilitadas cuando el perfil no las posee.

## Paridad funcional de las 17 tareas

| N.º | Tarea | Evidencia principal de pantalla |
|---:|---|---|
| 1 | Solicitud del centro | Petición, RC y documentos |
| 2 | Análisis de RRHH | Modalidad, categoría, duración y RC |
| 3 | Vía de cobertura | Bolsa, SAE, convocatoria y decisión motivada |
| 4 | Asignación | Unidad, responsable y bandeja de trabajo |
| 5 | Informe jurídico | Fuentes, borrador, editor y generación |
| 6 | Envío a Intervención | Vista previa, firma y recibo de traslado |
| 7 | Fiscalización | Modalidad, resultado, observaciones y adjuntos |
| 8 | Subsanación | Reparos, correcciones, responsables y evidencias |
| 9 | Inicio de llamamiento | Configuración e historial de intentos |
| 10 | Selección | Orden de bolsa y candidatura concreta elegible |
| 11 | Resultado | Respuesta, acta e historial minimizado |
| 12 | Traslado | Índice documental y tarjeta minimizada |
| 13 | Informe definitivo | Candidatura, observaciones, historial y versión |
| 14 | Formalización | Subpasos, documentos, vista previa y firmas |
| 15 | Incorporación | Proyección autorizada y confirmación |
| 16 | GINPIX | Modelo canónico, resumen final e historial |
| 17 | Seguimiento y cese | Situación, prórroga, cese, conservación y cierre |

El cuadro añade «Mis tareas», distribución por fase y accesos rápidos. Los
controles de tareas históricas o no autorizadas se presentan en solo lectura.

## Evidencia automática

- JavaScript del portal: `272/272` pruebas superadas.
- Contratación temporal Go: siete paquetes superados.
- Validación de manifiestos: superada.
- Navegación Docker a `1440 × 1000`: 18 capturas, una del cuadro y una por cada
  tarea; cero errores de consola y cero desbordamientos horizontales
  (`scrollWidth = clientWidth = 1440`).
- Artefactos locales: `var/revision-web-ct-expedientes/`; no se versionan.
- Los ficheros del módulo permanecen por debajo de 800 líneas.

La revisión aplicó `admin-data-web`: densidad administrativa legible, columnas
operativas visibles, acciones cerca del contexto, estados expresos, navegación
persistente, mínimo privilegio y trazabilidad accesible.

## Sustitución del modo de presentación

La vista, presentador, contratos e i18n son reutilizables. La aplicación real
debe sustituir el adaptador de presentación por puertos de cuadro, expediente,
índice documental, auditoría de solo adición y comandos con versión,
idempotencia y recibo de servidor.

Los datos y efectos sintéticos están aislados en `datos-presentacion.js`,
`datos-presentacion-ampliacion.js` y `adaptador-presentacion.js`. Ninguno
aparece en `web/interno.manifest` ni en `web/produccion.manifest`.

## Continuación

Antes de revisar conjuntamente Dietas debe repetirse el perfil Docker con sus
artefactos OSRM/MBTiles. En esta revisión de contratación se usó un contenedor
auxiliar sin teselas porque esos artefactos no estaban disponibles en el
worktree; ello no afectó a estas dieciocho pantallas.

