# Estado de la web de contratación temporal para RRHH

## Contador vigente de pantallas — 13 de septiembre de 2026

**6 de 19 terminadas en desarrollo; 12 parciales; 1 pendiente de pantalla real.**
Hay superficie real publicada para **15 de 19 (79 %)**. Las superficies preparadas en código se distinguen de las activadas en la aplicación.
La subsanación cuenta como parcial programada, pendiente de instalación y recorrido;
no como pantalla todavía inexistente. Visible no significa terminada.
El denominador queda fijo: 17 referencias de RRHH y 2 tareas adicionales ya
incluidas en el alcance (incorporación y seguimiento). No son 19 URL distintas:
varias se presentan como paneles del expediente. Los seis documentos no suman
seis pantallas. No incrementar el contador por commits, pruebas o cambios de CSS.

El porcentaje que falta es una **estimación inicial de trabajo por pantalla**,
no una medición de horas ni una certificación. Se revisará al cerrar cada tarea.
Cero significa recorrido funcional de desarrollo documentado, no producción ni
validación final de RRHH. Firma, correo y conectores conservan sus dependencias;
el operador permite terminar cualquier pieza de correo necesaria para avanzar.

**Corte de integración comprobado:** producto remoto `e99c1c82` publica fases RRHH,
historial y ajustes móviles; asignación e informe se distinguen por evidencia real.
Subsanación `4996034d`, refresco `6257b6f6`, catálogo `64aeec4f` y refiscalización
`60e0c7a1` recibidos por el director remoto, que confirma que CT92/AD3-38/CT93 aún
no están instaladas. Estos paquetes no aumentan el número de recorridos cerrados.
Se conserva la recuperación documental PDF/DOCX y el historial anteriores.

| N.º | Pantalla | Estado | Falta estimada | Responsable / dependencia | Pendiente concreto o evidencia |
|---:|---|---|---:|---|---|
| 1 | Inicio y cuadro de mando | Parcial | 25 % | Director local; integración del parche entregado | Ámbito de indicadores; línea de progreso depende de definición de flujo publicada. |
| 2 | Nueva petición de personal | Terminada en desarrollo | 0 % | Sin trabajo pendiente | Alta real y recibo persistido; guía de recorrido. |
| 3 | Análisis de RRHH | Parcial | 25 % | Director remoto: activar entrega local 64aeec4f | Formulario y catálogo configurable conectados; falta fuente de motivos válida e instalación/recorrido. Catálogo vacío sigue sin permitir POST. |
| 4 | Gestión de bolsa y comprobaciones | Parcial | 25 % | Remoto: preparacion_continuidad | Reapertura integrada; comprobar recorrido de cobertura desde análisis existente. |
| 5 | Unidad y bandeja de trabajo | Parcial | 25 % | Remoto: preparacion_continuidad | Asignación persistida; falta recorrido de bandeja con lectores nominales. |
| 6 | Informe jurídico automático | Terminada en desarrollo | 0 % | Sin trabajo pendiente | Preparación real y recibo; documento de desarrollo sin firma. |
| 7 | Firma de Jefatura y envío a Intervención | Pendiente | 100 % | Espera circuito de firma admitido | No hay pantalla real equivalente; no sustituir firma/envío por simulación. |
| 8 | Fiscalización por Intervención | Parcial | 25 % | Remoto: preparacion_continuidad | Conexión normal integrada; comprobar formulario con expediente en su fase. |
| 9 | Subsanación de reparos | Parcial, programada sin activar | Por verificar | Director remoto: instalación de paquetes revisados | Formulario, API, permisos, persistencia y refresco del historial entregados en 4996034d/6257b6f6; nueva fiscalización tras corrección en 60e0c7a1. SQL CT92/AD3-38 y CT93 con doble revisión. Ninguna instalada según ACK remoto 13 septiembre 01:22 CEST; falta recorrido real. |
| 10 | Llamamiento de candidatura | Parcial | 50 % | Espera contacto/SMTP operativo | Selección y aviso local existen; envío efectivo depende de correo. Completar dependencia si impide avanzar, autorizado por operador. |
| 11 | Selección de candidatura | Terminada en desarrollo | 0 % | Sin trabajo pendiente | Selección y continuación real de Bolsa con recibos conservados. |
| 12 | Resultado del llamamiento | Parcial | 25 % | Espera política de plazo admitida | Aceptación y renuncia manuales recorribles; falta vencimiento con inicio/plazo acreditados. |
| 13 | Traslado de candidatura | Parcial | 25 % | Director local; integración UI entregada | Fuente Word pide generar documentos tras selección. Resumen de aceptación conectado a CT65; falta publicación/recorrido. No hace falta crear otro traslado. |
| 14 | Documentación para formalización | Parcial | 25 % | Remoto: director y correccion_arnes_ordinario | Recuperación histórica D/CT91 publicada: PDF/DOCX 200; falta cierre visual del conjunto documental. |
| 15 | Generación de datos GINPIX | Terminada en desarrollo | 0 % | Sin trabajo pendiente | Ficha manual real recuperada; no representa envío externo. |
| 16 | Resumen final y envío a GINPIX | Parcial | 25 % | Director local/remoto; integración de 450cea84 | Resumen real y ficha manual conectados en código; falta publicación/recorrido. La alternativa manual está admitida por Word, envío automático sigue separado. |
| 17 | Generación documental de formalización | Parcial | 25 % | Remoto: director y correccion_arnes_ordinario | Seis PDF/DOCX existentes; recuperación desde expediente avanzado corregida y publicada. Pendiente revisión visual final del conjunto. |
| 18 | Incorporación | Terminada en desarrollo | 0 % | Sin trabajo pendiente | Incorporación y recuperación documentadas con mismo recibo. |
| 19 | Seguimiento y cierre administrativo | Terminada en desarrollo | 0 % | Sin trabajo pendiente | Anotación/cierre y estado recuperados; no es cese ni cierre jurídico. |

Fuentes: matriz exacta histórica inferior; `GUIA_RECORRIDO_ALBERTO.md`
(recibos, incorporación, ficha, cierre y límites de fiscalización/cobertura);
`ESTADO_PROYECTO.md` (cola de entregas y dependencias); composición real de
`vista-expedientes.js`. El inventario no usa el adaptador DEMO como evidencia.
Responsables locales activos trabajan en ramas separadas; el director local
integra y coordina la comprobación de los recorridos pendientes con el remoto activo.
Los estados de responsables deben actualizarse al acabar o reasignar cada tarea.

## Corte histórico de julio (no es el contador vigente)

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
