# Estado de la web de contratación temporal para RRHH

Fecha de corte: 26 de julio de 2026.

## Resultado

La superficie de presentación está compuesta dentro del Portal del Empleado y
reutiliza el tema, la identidad interna y la navegación comunes. El código
neutral no contiene datos sintéticos ni decide autorizaciones; el fixture y el
adaptador volátil solo se cargan con `?presentacion=rrhh`.

Este corte cubre el cuadro operativo, el alta O2-09B, cinco expedientes
coherentes, ocho fases, dieciocho tareas, documentos y auditoría. Las acciones
usan capacidades exactas, vuelven a validarse en el adaptador y quedan
visiblemente deshabilitadas cuando el perfil no las posee.

La aplicación conserva dos tareas adicionales —incorporación y seguimiento—
que no aparecen como pantallas independientes en la referencia, pero forman
parte del objetivo funcional aprobado. Tener más capacidad no altera el
recorrido mínimo solicitado.

## Matriz exacta de las 17 pantallas de RRHH

| N.º | Pantalla de la referencia | Vista o tarea de la aplicación |
|---:|---|---|
| 1 | Inicio y cuadro de mando | Vista `cuadro` |
| 2 | Nueva petición de personal | Vista `alta` O2-09B |
| 3 | Análisis de RRHH | `tarea-analisis` |
| 4 | Gestión de bolsa y comprobaciones automáticas | `tarea-cobertura` |
| 5 | Unidad del Departamento y bandeja de trabajo | `tarea-asignacion` |
| 6 | Informe jurídico automático | `tarea-informe-juridico` |
| 7 | Firma de Jefatura y envío a Intervención | `tarea-envio-intervencion` |
| 8 | Fiscalización por Intervención | `tarea-fiscalizacion` |
| 9 | Subsanación de reparos | `tarea-subsanacion` |
| 10 | Llamamiento de la candidatura | `tarea-iniciar-llamamiento` |
| 11 | Selección de candidatura de la bolsa | `tarea-seleccion-candidato` |
| 12 | Resultado del llamamiento | `tarea-resultado-llamamiento` |
| 13 | Traslado de la candidatura | `tarea-traslado-intervencion` |
| 14 | Documentación para formalización | `tarea-informe-definitivo` |
| 15 | Generación de datos para GINPIX | `tarea-ginpix` |
| 16 | Resumen final y envío a GINPIX | `tarea-envio-ginpix` |
| 17 | Generación documental para formalización | `tarea-formalizacion` |

El cuadro añade «Mis tareas», distribución por fase y accesos rápidos. Los
controles de tareas históricas o no autorizadas se presentan en solo lectura.

## Evidencia automática

- JavaScript focal del corte O4-05: `56/56` pruebas superadas tras incorporar
  la consulta protegida del resultado de cobertura.
- JavaScript completo de la web: `383/383` pruebas superadas.
- Contrato del capturador RRHH: `5/5` pruebas superadas.
- Contratación temporal Go: siete paquetes superados, también con detector de
  carreras.
- Suite Go completa y `go vet ./...`: superados.
- Navegación Docker: `51/51` capturas correctas, correspondientes a las 17
  pantallas exactas en `1536 × 1024`, `1440 × 1000` y `1280 × 900`.
- Las capturas son del área visible, no de página completa; no presentan
  desbordamiento horizontal, foco accidental en el enlace de salto, cookies,
  almacenamiento web ni errores de consola.
- Artefactos locales:
  `var/revision-web-contratacion-rrhh/`; no se versionan.
- Los ficheros del módulo permanecen por debajo de 800 líneas.

La revisión aplicó `admin-data-web`: densidad administrativa legible, columnas
operativas visibles, acciones cerca del contexto, estados expresos, navegación
persistente, mínimo privilegio y trazabilidad accesible. Las referencias y la
huella técnica permanecen disponibles en un bloque plegado, sin desplazar los
datos administrativos prioritarios.

## Sustitución del modo de presentación

La vista, presentador, contratos e i18n son reutilizables. La aplicación real
debe sustituir el adaptador de presentación por puertos de cuadro, expediente,
índice documental, auditoría de solo adición y comandos con versión,
idempotencia y recibo de servidor.

El cliente HTTP productivo para alta, propuesta, decisión, rectificación y
recuperación propia ya existe y figura en los manifiestos. No se conecta
todavía a la fuente general: faltan proyecciones productivas para las demás
lecturas y la composición raíz con dependencias reales. Ante un resultado
indeterminado, los presentadores bloquean todo reenvío y mantienen el aviso
durante la navegación; solo un recibo confirmado y validado lo libera.

Los datos y efectos sintéticos están aislados en `datos-presentacion.js`,
`datos-presentacion-ampliacion.js` y `adaptador-presentacion.js`. Ninguno
aparece en `web/interno.manifest` ni en `web/produccion.manifest`.

## Continuación

Antes de revisar conjuntamente Dietas debe repetirse el perfil Docker con sus
artefactos OSRM/MBTiles. En esta revisión de contratación se usó un contenedor
auxiliar sin teselas porque esos artefactos no estaban disponibles en el
worktree; ello no afectó a estas dieciocho pantallas.
