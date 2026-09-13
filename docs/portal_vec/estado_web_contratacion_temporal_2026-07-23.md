# Estado de la web de contratación temporal para RRHH

## Contador vigente de pantallas — 13 de septiembre de 2026

**14 de 19 terminadas en desarrollo (74 %); 4 parciales; 1 pendiente de pantalla real.**
Hay **16 superficies reales visibles (84 %)**, incluida la subsanación ya recorrida.
La subsanación está recorrida: guardado `201` (v6 → v7), reinicio de aplicación
y PostgreSQL y recuperación con el mismo recibo de 11 campos y una sola reserva.
La configuración corregida cumple el formato de publicación PostgreSQL y el
cargador exige el perfil V2 existente. No se han modificado permisos ni migraciones.
La línea de fases y el resumen de propuesta ya están publicados; su publicación
no incrementa por sí sola las pantallas terminadas. Visible no significa terminada.
El denominador queda fijo: 17 referencias de RRHH y 2 tareas adicionales ya
incluidas en el alcance (incorporación y seguimiento). No son 19 URL distintas:
varias se presentan como paneles del expediente. Los seis documentos no suman
seis pantallas. No incrementar el contador por commits, pruebas o cambios de CSS.

La continuidad después de subsanar está publicada como código: nueva fiscalización
(CT93), selección (CT94), aviso/respuesta (CT95) y propuesta (CT96, `539247a6`).
El formulario admite la versión fiscalizada y los documentos actuales usan la nueva
propuesta (`8927bf61`). Las cuatro UP se ejecutaron una vez en la principal el 13 de septiembre a las
06:26 UTC. La diferencia `timezone`/`TimeZone` del inventario quedó reconciliada:
siete cuerpos y catálogo completo coincidentes, historial y permisos conservados.
La misma aplicación se restableció a las 06:34 UTC y el producto `93e77d1d`
ya está activado con las cuatro migraciones. No reaplicar CT93–96.
Fiscalización real201 v7→v8 y recuperación tras reinicio: mismo recibo de12campos; consulta RRHH200 con ocho actuaciones.
La consulta documental conserva ahora la propuesta posterior a subsanación
cuando su historia añade resolución y anotación administrativa; comprobada en
código con propuesta v9 desde v10/v11, pendiente de recorrido principal.
La continuación tras renuncia de ese nuevo recorrido y SMTP siguen pendientes.

El porcentaje que falta es una **estimación inicial de trabajo por pantalla**,
no una medición de horas ni una certificación. Se revisará al cerrar cada tarea.
Cero significa recorrido funcional de desarrollo documentado, no producción ni
validación final de RRHH.
Estas estimaciones mezclaban pantalla e integración; no deben presentarse como
porcentajes exactos de web terminada. En las dependencias externas se distingue:

- Firma (7): falta pantalla y conexión; el circuito admitido sigue por concretar.
- Llamamiento (10): bandeja y gestión web se pueden completar independientemente
  del SMTP corporativo, cuyo código queda conservado y su conexión pendiente.
- GINPIX (16): ficha y resumen web ya recorridos; falta el envío externo.

El aplazamiento de una conexión externa no aplaza el trabajo independiente de su pantalla. Firma, correo y conectores conservan sus dependencias;
el operador permite terminar cualquier pieza de correo necesaria para avanzar.

**Disponibilidad transversal:** binario de `148d075e` con fuente sintética exacta
para el caso de cobertura activo; etiquetas `9d5d41eb`, bandeja móvil `253c4d63`
y distinción de períodos `25722b17` servidas y comprobadas. No hubo nueva migración
ni reinicio PostgreSQL por estas mejoras; la recuperación de refiscalización
conserva su evidencia anterior independiente.
Históricamente, la aplicación estuvo sana con el binario
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
| 1 | Inicio y cuadro de mando | Terminada en desarrollo | 0 % | Dirección: conservación | 52 filas autorizadas y filtros vacío/completados HTTP200; acciones ligadas a la página. CSS253c4d63 probado a1440/390, sin desbordamiento global y tabla con scroll interno. |
| 2 | Nueva petición de personal | Terminada en desarrollo | 0 % | Dirección: conservación | Alta real y recibo persistido; guía de recorrido. |
| 3 | Análisis de RRHH | Parcial | 25 % | Dirección: catálogo de motivos para rectificación | Registro normal201 v1→v2, cinco modalidades localizadas y consulta posterior200 con RCvalidada/dos actuaciones. Rectificación continúa sin fuente de motivos admitida, bloqueada por defecto. |
| 4 | Gestión de bolsa y comprobaciones | Terminada en desarrollo | 0 % | Dirección: conservación | Fuente sintética del período exacto: propuesta200, decisión única201 v2→v3, resultado200 con recibo idéntico y detalle200 con asignación ofrecida.1440/390 sin JS ni desbordamiento; no fuente corporativa. |
| 5 | Unidad y bandeja de trabajo | Terminada en desarrollo | 0 % | Dirección: conservación | Asignación sintética201 v3→v4, recibo conservado tras refresco. Reapertura200 confirma unidad y ofrece informe, sin otra asignación;1440/390 sin errores JS. |
| 6 | Informe jurídico automático | Terminada en desarrollo | 0 % | Dirección: conservación | Preparación real y recibo; documento de desarrollo sin firma. |
| 7 | Firma de Jefatura y envío a Intervención | Pendiente | 100 % | Dirección: circuito de firma admitido | No hay pantalla real equivalente; no sustituir firma o envío por simulación. |
| 8 | Fiscalización por Intervención | Terminada en desarrollo | 0 % | Dirección: conservación | Intervención201 favorable con observaciones v7→v8; mismo recibo de12campos tras reinicio y replay original201, historia8 por consulta200. Acceso manual sin inferir antecedentes ni firma. |
| 9 | Subsanación de reparos | Terminada en desarrollo | 0 % | Director remoto: nueva fiscalización, sin reaplicar CT93–96 | Guardado `201` v6 → v7 y recuperación tras reiniciar aplicación/PostgreSQL: mismo recibo de 11 campos, una reserva y siete actuaciones. Conserva incidencia; 1440/390 px sin desbordamiento ni errores JS. AD3-38/CT92 instaladas una vez, no reaplicar. |
| 10 | Llamamiento de candidatura | Parcial | 50 % | Director remoto: recorrido; conexión SMTP aplazada | Contacto: ensayo aislado de DDL, ACL y reinicio terminado; faltan operación positiva y gobierno de Usuarios. CT94/95 aplicadas y reconciliadas, recorrido pendiente; SMTP aplazado. |
| 11 | Selección de candidatura | Terminada en desarrollo | 0 % | Dirección: conservación | Selección y continuación real de Bolsa con recibos conservados. |
| 12 | Resultado del llamamiento | Parcial | 25 % | Dirección: inicio y plazo gobernados | Aceptación y renuncia manuales recorribles; falta vencimiento con inicio y plazo acreditados. |
| 13 | Traslado de candidatura | Parcial | 25 % | Director remoto: recorrido de entrega integrada | Resumen publicado; asset real `GET 200`. Falta recorrerlo con aceptación recuperada, sin equipararlo a envío. |
| 14 | Documentación para formalización | Terminada en desarrollo | 0 % | Dirección: conservación | Seis borradores agrupados con doce acciones PDF/DOCX y estados propios. DOCX representativo200 con SHA7141 conservado; revisión1440/390 sin errores JS. |
| 15 | Generación de datos GINPIX | Terminada en desarrollo | 0 % | Dirección: conservación | Ficha manual real recuperada; no representa envío externo. |
| 16 | Resumen final y envío a GINPIX | Terminada en desarrollo | 0 % | Dirección: conservación | Incorporación original y seguimientoGET200; resumen1440/390 distingue exportación manual y transmisión externa pendiente. No se ha enviado a GINPIX. |
| 17 | Generación documental de formalización | Terminada en desarrollo | 0 % | Dirección: conservación | Seis pares PDF/DOCX; descarga real con huella conservada. Cancelación/reintento en navegador con transporte retenido controladamente; el PDF cancelado no llegó al servidor. |
| 18 | Incorporación | Terminada en desarrollo | 0 % | Dirección: conservación | Incorporación y recuperación documentadas con mismo recibo. |
| 19 | Seguimiento y cierre administrativo | Terminada en desarrollo | 0 % | Dirección: conservación | Anotación/cierre y estado recuperados; no es cese ni cierre jurídico. |
Fuentes: matriz exacta histórica inferior; `GUIA_RECORRIDO_ALBERTO.md`
(recibos, incorporación, ficha, cierre y límites de fiscalización/cobertura);
`ESTADO_PROYECTO.md` (cola de entregas y dependencias); composición real de
`vista-expedientes.js`. El inventario no usa el adaptador DEMO como evidencia.
Responsables locales y remotos trabajan en ramas separadas; Dirección integra
y coordina los recorridos pendientes sobre la aplicación activa.
Los estados de responsables deben actualizarse al acabar o reasignar cada tarea.

Entrega publicada `f19f7d47`: integra las tres tandas web con el producto remoto.
Incluye cinco modalidades de análisis, continuidad por recibo de Unidad y
fiscalización, seis documentos agrupados con cancelación, resumen de traslado,
reintento de consulta de cobertura, resumen del resultado y separación entre
ficha manual y transmisión GINPIX. Comprobación focal de integración: 128/128.
Activada y recorrida en los alcances cerrados arriba; el resto conserva su estado parcial.

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
| 3 | Análisis de RRHH | Parcial | 25 % | Dirección: catálogo de motivos para rectificación | Registro normal201 v1→v2, cinco modalidades localizadas y consulta posterior200 con RCvalidada/dos actuaciones. Rectificación continúa sin fuente de motivos admitida, bloqueada por defecto. |
| 4 | Gestión de bolsa y comprobaciones automáticas | `tarea-cobertura` |
| 5 | Unidad del Departamento y bandeja de trabajo | `tarea-asignacion` |
| 6 | Informe jurídico automático | `tarea-informe-juridico` |
| 7 | Firma de Jefatura y envío a Intervención | `tarea-envio-intervencion` |
| 8 | Fiscalización por Intervención | `tarea-fiscalizacion` |
| 9 | Subsanación de reparos | `tarea-subsanacion`, formulario del expediente |
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
