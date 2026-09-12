# Estado de la web de contratación temporal para RRHH

## Contador vigente de pantallas — 13 de septiembre de 2026

**6 de 19 terminadas en desarrollo; 12 parciales; 1 pendiente de pantalla real.**
El contador conserva la evidencia de las **15 superficies reales (79 %)** anteriores.
La subsanación está programada, pendiente de activar; cuenta como parcial, no inexistente.
La línea de fases y el resumen de propuesta ya están publicados; su publicación
no incrementa por sí sola las pantallas terminadas. Visible no significa terminada.
El denominador queda fijo: 17 referencias de RRHH y 2 tareas adicionales ya
incluidas en el alcance (incorporación y seguimiento). No son 19 URL distintas:
varias se presentan como paneles del expediente. Los seis documentos no suman
seis pantallas. No incrementar el contador por commits, pruebas o cambios de CSS.

El porcentaje que falta es una **estimación inicial de trabajo por pantalla**,
no una medición de horas ni una certificación. Se revisará al cerrar cada tarea.
Cero significa recorrido funcional de desarrollo documentado, no producción ni
validación final de RRHH. Firma, correo y conectores conservan sus dependencias;
el operador permite terminar cualquier pieza de correo necesaria para avanzar.

**Disponibilidad transversal:** la aplicación está sana con el binario
`783c051a6ee1d81fb4925aed99615dbe3cabbc39ee3ed53c316e2620de9ea66f` desde las 22:54 UTC; PostgreSQL no se reinició en este corte. Los
cortes publicados `9a522d2f`, `e5dcda5f`, `dea52536`, `35bbdd14`, `8d94e7e5`
y `e8d3a2a6` se conservan. Después, `4fc058f5` acreditó un PDF y un DOCX reales
desde el expediente v9 con HTTP `200` e historia exacta; `d4cae9f2` publicó la
búsqueda y el control táctil; `861b4132` publicó el resumen GINPIX, con ficha y
seguimiento `GET 200`, misma huella y pantalla sin desbordamiento a 1440/390 px;
y `c3a30d8b` recuperó Cobertura desde la fase persistida de solicitud mediante
el helper `faseSolicitud`. Las
piezas de línea de fases y propuesta están publicadas en `da17727b` y
`1ce29f87`. Unidad y `fe493…v9` muestran ocho fases, con Gestión de bolsa y Nombramiento
respectivamente activas, sin marcar fases completadas. Historial abierto de
3 y 9 actuaciones; consultas `200`, sin desbordamiento ni errores JS a 1440/390.
La corrección móvil se publicó en `6ee37c8d`, el historial en `9c2376ce` y
rectificación en `75054c10`. El diagnóstico `762942e2` identifica el fallo de
cobertura en su preparador; todavía no declara resuelto el `503`.

| N.º | Pantalla | Estado | Falta estimada | Responsable / dependencia | Pendiente concreto o evidencia |
|---:|---|---|---:|---|---|
| 1 | Inicio y cuadro de mando | Parcial | 25 % | Dirección y `qa_rail_unidad` | Rail e historial publicados y observados en ambos casos a 1440/390; no acredita cierre funcional del cuadro completo. |
| 2 | Nueva petición de personal | Terminada en desarrollo | 0 % | Dirección: conservación | Alta real y recibo persistido; guía de recorrido. |
| 3 | Análisis de RRHH | Parcial | 25 % | Director remoto: activar entrega local 64aeec4f | Formulario y catálogo configurable conectados; falta fuente de motivos válida e instalación/recorrido. Catálogo vacío sigue sin permitir POST. |
| 4 | Gestión de bolsa y comprobaciones | Parcial | 25 % | Dirección: integrar candidato de `corregir_preparador_cobertura` | Caso `b50fa…` v2 RC validada: detalle `200`; la propuesta devuelve `503` en el preparador; candidato de concurrencia pendiente de medición real. |
| 5 | Unidad y bandeja de trabajo | Parcial | 25 % | Dirección: continuar desde Unidad existente | Detalle v3 `200`, móvil corregido y referencia exacta. Solo ofrece Asignación; Informe exige asignación proyectada. Pendiente confirmar la asignación del ejercicio. |
| 6 | Informe jurídico automático | Terminada en desarrollo | 0 % | Dirección: conservación | Preparación real y recibo; documento de desarrollo sin firma. |
| 7 | Firma de Jefatura y envío a Intervención | Pendiente | 100 % | Dirección: circuito de firma admitido | No hay pantalla real equivalente; no sustituir firma o envío por simulación. |
| 8 | Fiscalización por Intervención | Parcial | 25 % | Dirección: continuidad desde Unidad | Conexión integrada. El caso Unidad debe confirmar asignación y preparar informe antes de fiscalizar; Intervención requiere identidad y resultado explícitos. |
| 9 | Subsanación de reparos | Parcial, programada sin activar | Por verificar | Director remoto: instalación de paquetes revisados | Formulario, API, permisos, persistencia y refresco del historial entregados en 4996034d/6257b6f6; nueva fiscalización tras corrección en 60e0c7a1. SQL CT92/AD3-38 y CT93 con doble revisión. Ninguna instalada según ACK remoto 13 septiembre 01:22 CEST; falta recorrido real. |
| 10 | Llamamiento de candidatura | Parcial | 50 % | Remoto: `ensayo_contacto_aislado`; Dirección: gobierno Usuarios/SMTP | Fuente revisada; ensayo aislado en curso, sin instalación principal. Falta composición/gobierno nominal Usuarios y operación positiva; SMTP pendiente. |
| 11 | Selección de candidatura | Terminada en desarrollo | 0 % | Dirección: conservación | Selección y continuación real de Bolsa con recibos conservados. |
| 12 | Resultado del llamamiento | Parcial | 25 % | Dirección: inicio y plazo gobernados | Aceptación y renuncia manuales recorribles; falta vencimiento con inicio y plazo acreditados. |
| 13 | Traslado de candidatura | Parcial | 25 % | Dirección y `qa_rail_unidad` | Resumen publicado; asset real `GET 200`. Falta recorrerlo con aceptación recuperada, sin equipararlo a envío. |
| 14 | Documentación para formalización | Parcial | 25 % | Remoto: Dirección integra y comprueba runtime | `4fc058f5`: un PDF y un DOCX de la propuesta v7 recuperados desde v9 con HTTP `200`; falta revisión funcional del conjunto. |
| 15 | Generación de datos GINPIX | Terminada en desarrollo | 0 % | Dirección: conservación | Ficha manual real recuperada; no representa envío externo. |
| 16 | Resumen final y envío a GINPIX | Parcial | 25 % | Apoyo local: resumen GINPIX | `861b4132`: ficha y seguimiento reales `GET 200`, misma huella y 1440/390 px; el envío externo sigue pendiente. |
| 17 | Generación documental de formalización | Parcial | 25 % | Remoto: Dirección integra y comprueba runtime | Seis pares PDF/DOCX disponibles; un PDF y un DOCX representativos respondieron `200` desde v9, con historia conservada. |
| 18 | Incorporación | Terminada en desarrollo | 0 % | Dirección: conservación | Incorporación y recuperación documentadas con mismo recibo. |
| 19 | Seguimiento y cierre administrativo | Terminada en desarrollo | 0 % | Dirección: conservación | Anotación/cierre y estado recuperados; no es cese ni cierre jurídico. |

Fuentes: matriz exacta histórica inferior; `GUIA_RECORRIDO_ALBERTO.md`
(recibos, incorporación, ficha, cierre y límites de fiscalización/cobertura);
`ESTADO_PROYECTO.md` (cola de entregas y dependencias); composición real de
`vista-expedientes.js`. El inventario no usa el adaptador DEMO como evidencia.
Responsables locales y remotos trabajan en ramas separadas; Dirección integra
y coordina los recorridos pendientes sobre la aplicación activa.
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
| 3 | Análisis de RRHH | Parcial | 25 % | Director remoto: activar entrega local 64aeec4f | Formulario y catálogo configurable conectados; falta fuente de motivos válida e instalación/recorrido. Catálogo vacío sigue sin permitir POST. |
| 4 | Gestión de bolsa y comprobaciones automáticas | `tarea-cobertura` |
| 5 | Unidad del Departamento y bandeja de trabajo | `tarea-asignacion` |
| 6 | Informe jurídico automático | `tarea-informe-juridico` |
| 7 | Firma de Jefatura y envío a Intervención | `tarea-envio-intervencion` |
| 8 | Fiscalización por Intervención | `tarea-fiscalizacion` |
| 9 | Subsanación de reparos | Parcial, programada sin activar | Por verificar | Director remoto: instalación de paquetes revisados | Formulario, API, permisos, persistencia y refresco del historial entregados en 4996034d/6257b6f6; nueva fiscalización tras corrección en 60e0c7a1. SQL CT92/AD3-38 y CT93 con doble revisión. Ninguna instalada según ACK remoto 13 septiembre 01:22 CEST; falta recorrido real. |
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
