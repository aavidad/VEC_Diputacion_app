# Petición de RRHH: comprobación de las cuatro fotografías

Fuente: cuatro fotografías recibidas el 27/09/2026, conservadas fuera de Git en
`/home/alberto/Trabajo/piden_VEC/`. Coinciden con la transcripción anterior de
[`Peticion.pdf`](docs/estudio_requisitos/peticion_rrhh_transcripcion_y_lectura.md).
Esta lista descompone sus ejemplos en operaciones comprobables. **HECHO** indica
una capacidad ya cubierta dentro del alcance que precisa cada fila; «HECHO
(diseño VEC)» describe código y contrato, no un recorrido nuevo en navegador.
**PARCIAL** identifica lo que existe y lo que falta; **BLOQUEADO** señala una
dependencia externa. No se exige copiar tablas, DNI + clave ni la redacción de
las fotos. La evidencia de código no acredita por sí sola instalación, recorrido
en navegador ni entrega corporativa. Estado contrastado el 28/09/2026 con
`origin/main@eb2061f33`: 60 requisitos, 48 HECHO preexistentes, 11 PARCIAL y
uno BLOQUEADO (2.12). Ningún PARCIAL se eleva a HECHO por pruebas aisladas.
La ayuda visible se abre desde «?» conforme al shell vigente, sin convertirla
en efecto de negocio.
El bloque 5 procede del seguimiento del Departamento de 25/09/2026 (texto y
cinco capturas) y se cuenta aparte de la fotografía anterior: añade siete
requisitos, ninguno HECHO; cinco PARCIAL y dos PENDIENTE. Total de esta lista:
67 requisitos, 48 HECHO, 16 PARCIAL, dos PENDIENTE y uno BLOQUEADO. No se altera
la evidencia ni el cómputo original de los puntos 1.01–4.15.

## Fotografía 1: histórico, estados y transparencia

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 1.01 | Consultar contratos realizados en orden cronológico. | Portal personal «Mi Bolsa» y ficha RRHH de candidato. | **HECHO (diseño VEC)**: `GET /api/vec/bolsa/mi-bolsa/historial?pagina=N` y Bolsa 000044 muestran contratos propios recibidos de CT en orden cronológico; `mi-bolsa-historial.js` los presenta con certificado y autorización AD3-90. PostgreSQL 18 efímero comprobó UP/DOWN/ACL tras CT124/Bolsa42 (con concesión temporal de USAGE T13 requerida por la cadena de ensayo); el historial anterior a VEC depende de fuente corporativa (duda 39). |
| 1.02 | Consultar llamamientos anteriores. | «Mi Bolsa» y ficha RRHH en `/portal-empleado/`. | **HECHO (diseño VEC)**: el mismo histórico propio paginado consulta llamamientos y resultados B7 por candidatura propia; `application/mibolsa/historial.go`, `adapters/httppersonal/historial.go` y `mi-bolsa-historial.js` limitan a la persona autenticada. No se inventan contactos previos a VEC. |
| 1.03 | Consultar renuncias y su resolución. | Ficha RRHH de Bolsa y detalle de Contratación. | **HECHO (diseño VEC)**: Bolsa 000044 proyecta renuncias B30 propias y su estado de respuesta, además de la ficha RRHH B8; la web distingue respuesta firme de propuesta pendiente, sin presentarla como resolución administrativa CT. |
| 1.04 | Registrar y consultar sanciones con acto, motivo y competencia. | Ficha RRHH de candidato, «Sanciones». | **HECHO (diseño VEC)**: `portal-bolsas-sanciones.js` y `application/sanciones_participacion.go` registran acto, consecuencia, recurso y su historial desplegable con recibo/actor; dos revisiones independientes GO del hash productor `f919849d` confirmaron privacidad y contrato. El catálogo de ejemplo espera aprobación de RRHH (duda 62). |
| 1.05 | Consultar cambios de estado anteriores y valor nuevo. | Ficha RRHH de candidato, «Histórico» y «Cambios». | **HECHO (diseño VEC)**: `portal-bolsas-traza-valores.js` muestra antes/después de situación y contacto desde Bolsa 000034; la historia se agrupa por actuación y ámbito, sin exigir una tabla única mutable. |
| 1.06 | Consultar correos enviados, intentos y resultado. | Ficha RRHH de candidato, pestaña «Histórico». | **HECHO (diseño VEC)**: `application/contacto_participacion.go`, B7 y `portal-bolsas-api.js` muestran cada intento/resultado y su recibo; «enviado» no se confunde con entrega o notificación acreditada. |
| 1.07 | Consultar documentos generados y su versión. | Expediente de Contratación y módulo Documentos. | **HECHO (diseño VEC)**: `cliente-http-informe-definitivo.js` recupera DOCX/PDF del expediente y `modulos/documentos/` consulta piezas versionadas por su autoridad; no se duplica una lista personal universal. |
| 1.08 | Mostrar «Disponible». | Bolsa RRHH, ficha y lista pública. | **HECHO**: catálogo `internal/modules/bolsa/domain/situacion_participacion.go`, lista `portal-panel-interno.js` y consulta `web/static/bolsa/lista-bolsas-api.js`. |
| 1.09 | Mostrar «Trabajando». | Bolsa RRHH y lista pública. | **HECHO**: mismos contratos B2; `portal-bolsas-contrato.js` valida el estado y `portal-panel-interno.js` lo presenta. |
| 1.10 | Mostrar «No disponible». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 y cambio gobernado en `portal-bolsas-api.js`; pantalla de Bolsa. |
| 1.11 | Mostrar «Pendiente de incorporación». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 validada por `portal-bolsas-contrato.js` y renderizada por `portal-panel-interno.js`. |
| 1.12 | Mostrar «Renuncia». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 y operación B8 en `operacion_situacion_participacion.go`. |
| 1.13 | Mostrar «Excluido». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 y operación B8, con justificante en `portal-bolsas-operaciones.js`. |
| 1.14 | Mostrar «Disponible desde» con fecha. | Bolsa RRHH, Mi Bolsa y lista pública. | **HECHO** para el turno: la situación conserva fecha y la consulta SQL de orden vigente (Bolsa 000037) considera disponible a la persona al vencer; `portal-bolsas-reglas-situacion.js` presenta fecha y procedencia. No se crea otra actuación al vencer. |
| 1.15 | Calcular indisponibilidad tras contrato por regla configurable; ejemplo +5 meses desde 31/07/2026. | Ficha de situación de Bolsa y catálogo de reglas. | **PARCIAL**: Bolsa 000045/000050/000053 y CT129 calculan +5 meses por catálogo versionado y proyectan la situación efectiva; el relevo Go CT→Bolsa está montado en `internal/app/bootstrap/contratacion_temporal_seguimiento_cese_desarrollo.go`. `scripts/pruebas_rrhh_integradas/ejecutar.sh pg18` pasó en PostgreSQL 18 con recibo, fin de mes y proyección Mi Bolsa. Falta recorrido navegador→cese→Bolsa con reinicio en servidor aislado (duda 64). |
| 1.16 | Calcular indisponibilidad tras acumulación de tareas; ejemplo +9 meses. | Ficha de situación de Bolsa y catálogo de reglas. | **PARCIAL**: Bolsa 000045 y CT129 mapean modalidad a regla +9 meses, sin aplicar +5 a modalidad desconocida; la proyección de Bolsa 000050/000053 y el relevo Go están integrados. El mismo ensayo PostgreSQL 18 comprobó fin de mes, replay, relaciones simultáneas y lectura Mi Bolsa; falta recorrido navegador y reinicio (duda 64). |
| 1.17 | Portal personal con acceso seguro e identidad del titular. | Área personal, «Mi Bolsa». | **HECHO (diseño VEC)**: `internal/app/bootstrap/bolsa_mi_bolsa.go` deriva el candidato del certificado y la sesión del servidor; se conserva certificado, nunca DNI + clave. El proveedor institucional para datos reales sigue siendo una dependencia de producción, no un defecto de esta pantalla. |
| 1.18 | El titular ve solo sus bolsas, posición y estado. | Área personal, «Mi Bolsa». | **HECHO (diseño VEC)**: `application/mibolsa/portal.go` y `GET /api/vec/bolsa/mi-bolsa` aplican ámbito V3 del candidato y presentan sus participaciones, orden y situación, sin aceptar identidad libre del navegador. |
| 1.19 | El titular ve último llamamiento, contratos y fecha de disponibilidad. | Área personal, «Mi Bolsa». | **HECHO (diseño VEC)**: `application/mibolsa/portal.go` muestra situación, fecha y último contacto; la ruta propia `/api/vec/bolsa/mi-bolsa/historial` añade contratos y llamamientos paginados con autorización AD3-90, consumidos por `mi-bolsa-historial.js`. |
| 1.20 | Consulta pública de integrantes de una bolsa con posición y estado publicables. | `/bolsa/`, `/api/publico/bolsa/bolsas/{bolsa_ref}/lista`. | **HECHO (diseño VEC)**: `web/static/bolsa/lista-bolsas-api.js` y `internal/modules/bolsa/publico/httpapi/bolsas.go` muestran orden y estado con identidad minimizada; la foto no justifica publicar nombres o DNI completos. |

## Fotografía 2: propuesta y estructura de datos

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 2.01 | Gestionar bolsas vigentes y candidaturas. | `/portal-empleado/`, cuadro y ficha Bolsa. | **HECHO** para la gestión interna básica: `internal/app/bootstrap/bolsa_rrhh_desarrollo.go` monta `/api/vec/bolsa/bolsas`; `portal-bolsas-api.js` consume lista y ficha. |
| 2.02 | Proponer llamamientos según orden y Reglamento. | «Nuevo llamamiento» de Bolsa. | **HECHO (diseño VEC)**: `portal-panel-interno.js` recorre cuatro pasos y propone personas según orden vigente, con confirmación humana, recibo e historia; el control humano explica excepciones en vez de adjudicar sin regla aprobada. Los parámetros de ejemplo siguen pendientes de RRHH (dudas 13–14). |
| 2.03 | Gestionar contratos. | Expediente de Contratación temporal. | **HECHO (diseño VEC)**: el expediente coordina propuesta, formalización, incorporación, modificación y cese (`seguimiento-cese.js`) con recibos; la firma y alta corporativa son efectos externos separados, no otra tabla de contratos. |
| 2.04 | Registrar cese. | Expediente de Contratación, «Seguimiento y cese». | **HECHO** como registro de desarrollo: `seguimiento-cese.js` consume `/api/vec/contratacion-temporal/ceses`, caso de uso `application/cese_cierre_modificacion.go`, PostgreSQL, recibo e historia; el recorrido en copia de ensayo con reinicio consta en `ESTADO_PROYECTO.md` (26/09). No acredita baja laboral corporativa. |
| 2.05 | Registrar reincorporación y reflejarla en Bolsa. | Expediente de Contratación y ficha Bolsa. | **PARCIAL**: CT130/AD3-92, CT134/AD3-98 y Bolsa 000046 registran el retorno con autorización V3, lectura previa de un solo uso y recibo; el montaje CT→Bolsa está publicado en el corte 2 `da48a409b`. B55/AD3-101 y la ficha web están reunidos en la rama C3, con consulta nominal V3 y auditoría del GET y su denegación; PG18.4 efímero acreditó ACL, carrera, RLS y reinicio de base. `scripts/verificar_calidad.sh` pasó sobre C3, pero falta el ensayo del clon por Dirección y el recorrido mTLS/navegador con aplicación y recuperación; C3 aún no está en `main` (duda 65). |
| 2.06 | Configurar y versionar reglas sin reprogramar cambios normativos. | «Reglas vigentes» de Bolsa; configuración externa. | **HECHO (diseño VEC)**: `internal/vec/reglas/` resuelve catálogos versionados de situación, reposición y avisos desde `data/demo/reglas/bolsa_reglas.ejemplo.demo.json`; cambiar una regla publicada no exige modificar Go. Los valores de ejemplo esperan ratificación de RRHH. |
| 2.07 | Portal seguro del candidato. | «Mi Bolsa». | **HECHO (diseño VEC)**: mismo acceso por certificado y contexto del servidor de 1.17 (`internal/app/bootstrap/bolsa_mi_bolsa.go`), más seguro que DNI + clave de la foto. |
| 2.08 | Cuadro de mando de responsables y dirección. | Cuadro Bolsa en `/portal-empleado/`. | **HECHO** para indicadores operativos: `portal-panel-interno.js` consume `/api/vec/bolsa/estadisticas` montada en `bolsa_rrhh_desarrollo.go`. No sustituye informes oficiales. |
| 2.09 | Estadísticas y explotación por bolsa y situación. | «Estadísticas» de Bolsa. | **HECHO (diseño VEC)**: `portal-bolsas-api.js` valida agregados `vec.bolsa.rrhh.estadisticas.v1` y `portal-panel-interno.js` permite consultar cifras por bolsa/estado; la foto no exige exportar datos personales. |
| 2.10 | Generar documentos Word y PDF. | Detalle de Contratación, descargas. | **HECHO (diseño VEC)**: `cliente-http-informe-definitivo.js` descarga DOCX/PDF generados desde catálogo versionado; se rotulan borradores hasta firma y custodia oficiales, que son efectos distintos. |
| 2.11 | Integrar correo electrónico. | «Nuevo llamamiento» y contactos. | **HECHO (diseño VEC)**: `application/emision_llamamiento.go` envía por adaptador SMTP configurable, registra resultado y recibo por persona; en presentación usa relay de desarrollo, no se afirma entrega corporativa (dudas 10 y 45). |
| 2.12 | Integrar SMS y, si procede, mensajería instantánea. | «Nuevo llamamiento» y contactos. | **BLOQUEADO**: B7 tiene correo real y registro de intento manual de otros canales, pero falta conector corporativo SMS y política/canal autorizado; WhatsApp depende además de viabilidad y normativa (dudas 3 y 35). No se simula un envío. |
| 2.13 | Auditar acciones con trazabilidad. | Ficha RRHH, «Cambios», y auditoría autorizada. | **PARCIAL**: `internal/vec/auditoria/`, AD3-91, CT132, Bolsa48/B56 y CT136 dan la consulta con permiso propio, finalidad catalogada, cursor y denegaciones registradas; la ficha de la petición y la de la participación abren «Auditoría» (`modulos/auditoria/vista.js`). Desde el 29/09 cada fila se lee sin códigos: fecha y hora con segundos, persona, acción, «Qué cambió», «Motivo» y «Relacionado con»; referencias, recibo y huellas quedan en «Ver detalle técnico» (pruebas `vista.test.mjs`, capturas 1440/390 px sin desbordamiento). Falta la consulta positiva con certificado en navegador tras reiniciar la aplicación, que solo puede hacerse en la principal, y las decisiones de la duda 67. |
| 2.14 | Bolsa: id, nombre/categoría, vigencia, resolución aprobatoria y orden. | Lista/ficha Bolsa. | **HECHO (diseño VEC)**: `domain/llamamientos.go` conserva referencia, categoría, vigencia y resolución con huella; `InstantaneaOrdenBolsa` versiona posiciones en vez de una columna mutable universal. |
| 2.15 | Candidato: pertenencia a bolsa, DNI protegido, nombre, apellidos, correo y dos teléfonos. | Ficha RRHH de candidato. | **HECHO (diseño VEC)**: Persona común + participación por bolsa; `application/datos_contacto_participacion.go` protege correo/dos teléfonos y `portal-bolsas-contrato.js` exige documento enmascarado. No se duplica el DNI en cada bolsa. |
| 2.16 | Situación: bolsa, candidato, posición, estado, disponibilidad y observaciones. | Ficha RRHH de candidato. | **HECHO (diseño VEC)**: `situacion_participacion.go`, orden versionado y `portal-panel-interno.js` muestran posición, estado y fecha; observaciones se conservan como actuaciones autorizadas, evitando un texto libre público universal. |

## Fotografía 3: control, comunicaciones y avisos

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 3.01 | Contar bolsas activas. | Cuadro Bolsa. | **HECHO**: `/api/vec/bolsa/estadisticas`, validación y presentación en `portal-bolsas-api.js` y `portal-panel-interno.js`. |
| 3.02 | Contar candidatos por bolsa. | Cuadro Bolsa. | **HECHO**: `portal-bolsas-contrato.js` valida `por_bolsa[].total`; vista `portal-panel-interno.js`. |
| 3.03 | Contar disponibles, trabajando, excluidos y no disponibles. | Cuadro Bolsa. | **HECHO**: `portal-bolsas-contrato.js` valida `por_estado`; vista `portal-panel-interno.js`. |
| 3.04 | Seleccionar destinatarios por estado y por orden para un envío. | «Nuevo llamamiento», paso 2. | **HECHO (diseño VEC)** para ofertas: `portal-bolsas-api.js` selecciona en orden a quienes cumplen estado y elegibilidad, pagina la lista y limita 100; no envía ofertas a excluidos o a quienes prestan servicios sin excepción. Una campaña informativa a otros estados exige finalidad y permiso propios (duda 35). |
| 3.05 | Enviar correo masivo y personalizado a cada destinatario. | «Nuevo llamamiento», pasos 3–4. | **HECHO (diseño VEC)**: `portal-bolsas-correo.js` usa plantilla versionada, marcadores y vista previa; `application/correo_llamamiento.go` personaliza por persona y B7 envía por SMTP configurable. El relay de presentación no acredita entrega corporativa (dudas 10 y 45). |
| 3.06 | Comunicar oferta, modalidad, duración y vía web de solicitud; adjudicar por orden. | «Nuevo llamamiento» y «Mi Bolsa». | **PARCIAL**: B7/Mi Bolsa comunican y reciben respuestas; Bolsa 000047/000054 y AD3-93/97 aplican política versionada de 48 horas naturales, material V3 y propuesta por orden con confirmación RRHH. Corte 2 monta la capacidad backend y UI `rrhh-plazos-*`; PostgreSQL 18 pasó horario de verano, replay y orden. `application/oferta_publicada.go` aún admite resolver por una sola identidad: falta imponer la segunda validación configurable acordada para adjudicar, además del recorrido HTTP/navegador V3 real (duda 66). |
| 3.07 | Continuar con llamamiento directo si la oferta no se cubre. | Contratación, «Llamamiento». | **PARCIAL**: Bolsa 000047/000054 y corte 2 montan política versionada de oferta no cubierta y vencimiento para proponer siguiente elegible con confirmación RRHH; PostgreSQL 18 pasó no elegibles→llamamiento directo y recuperación de respuesta. `domain/intentos_contacto.go` aplica dos intentos separados al menos dos horas, pero aún permite iniciar el segundo ciclo el mismo día y el horario de ejemplo solo advierte: falta el valor configurable acordado de siguiente día laborable y control 09:00–14:00, más recorrido web de oferta vencida→siguiente candidato. No se finge aviso corporativo externo (duda 66). |
| 3.08 | Avisar de un candidato disponible saltado en el orden. | Cuadro Bolsa, avisos. | **HECHO** como aviso interno: `internal/modules/bolsa/application/avisos.go`, `/api/vec/bolsa/avisos`, `portal-bolsas-avisos.js`. No produce resolución automática. |
| 3.09 | Avisar un mes antes de tres años trabajando sin interrupción. | Cuadro Bolsa, avisos. | **HECHO (diseño VEC)**: `application/avisos.go` calcula el aviso desde historia disponible y `ports/politica_avisos.go` gobierna umbral/antelación; `portal-bolsas-avisos.js` lo muestra sin fabricar años anteriores a VEC. Historial corporativo previo requiere fuente de Personal (duda 39). |

## Fotografía 4: plantillas y auditoría

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 4.01 | Generar contrato laboral mediante plantilla Word. | Expediente de Contratación, «Documentos». | **HECHO (diseño VEC)**: `componentes-expedientes.js` descarga DOCX/PDF por cliente autorizado y `data/demo/plantillas/ct_plantillas_documentos.ejemplo.demo.json` selecciona contrato laboral por modalidad; el archivo es borrador hasta firma oficial. |
| 4.02 | Generar nombramiento mediante plantilla Word. | Expediente de Contratación. | **HECHO (diseño VEC)**: mismo recorrido de descarga y plantilla de nombramiento en el catálogo versionado; la generación no pretende crear un nombramiento eficaz. |
| 4.03 | Generar toma de posesión. | Detalle de Contratación, documentos. | **HECHO (diseño VEC)**: `cliente-http-informe-definitivo.js` descarga borrador DOCX/PDF de toma de posesión condicionado al expediente; no acredita posesión realizada. |
| 4.04 | Generar cese. | Expediente de Contratación. | **HECHO (diseño VEC)**: plantilla de cese en `data/demo/plantillas/ct_plantillas_documentos.ejemplo.demo.json` solo se genera tras `registrar_cese`; documento borrador, no baja corporativa. |
| 4.05 | Generar modificación de nombramiento. | Expediente de Contratación. | **HECHO (diseño VEC)**: plantilla DOCX/PDF condicionada a `registrar_modificacion_nombramiento` en el mismo catálogo; eficacia administrativa requiere circuito oficial distinto. |
| 4.06 | Generar informes. | Detalle de Contratación, «Documentos». | **HECHO (diseño VEC)**: informe definitivo como borrador PDF/DOCX desde `cliente-http-informe-definitivo.js`; otros modelos se gobiernan por catálogo. |
| 4.07 | Generar resoluciones. | Detalle de Contratación, «Documentos». | **HECHO (diseño VEC)**: resolución como borrador PDF/DOCX desde el cliente autorizado y catálogo versionado; la firma admitida sigue separada de generar el fichero. |
| 4.08 | Configurar otros tipos de plantilla. | Configuración externa de Contratación; Administración documental. | **PARCIAL**: `informejuridico/plantillas_borrador.go`, CT131/CT133/CT135/CT137 y AD3-94/96/99/100 versionan alta, publicación, consulta y ámbito de organización exacta; `rrhh-plantillas-vista.js` y `cmd/vec-provision-plantillas` consumen catálogo. CT137/AD3-100, API, descarga genérica y recuperación con la misma clave están reunidos en la rama C3, aún fuera de `main`. PG18.4 efímero comprobó ACL, replay y reinicio; los clientes web cotejan versión, procedencia, huella y recibo, y la puerta completa de calidad pasó. Faltan el ensayo del clon por Dirección y el recorrido de alta/publicación/descarga por identidades V3 reales tras reinicio de aplicación y base (duda 68). |
| 4.09 | Combinar datos de expediente y coste aproximado por categoría. | Documento y ficha de Contratación. | **HECHO (diseño VEC)**: `contratacion_temporal_fuentes_analisis_coste_desarrollo.go` calcula por categoría/grupo/periodo/jornada y `campos_borrador.go` combina `{{coste}}` o «sin calcular» con su fuente; no inventa importe corporativo. |
| 4.10 | Obtener campos de GINPIX/SAVIA o alternativa autorizada. | Expediente de Contratación, documentos y ficha GINPIX. | **HECHO (diseño VEC)** para combinar campos: `campos_borrador.go` toma datos versionados del expediente VEC como alternativa a GINPIX y `ficha_ginpix_v2.go` prepara salida manual. La conexión automática con GINPIX sigue bloqueada por contrato/fuente externa (duda 39), sin impedir generar borradores. |
| 4.11 | Auditoría: autor e instante exacto de cada cambio. | Historial autorizado RRHH/auditoría. | **PARCIAL**: «Auditoría» muestra el instante con segundos en hora de Madrid (la fuente lo guarda en microsegundos) y «Cambios de datos» de la ficha también pasa a segundos. El autor se ve como «Proceso automático» si es un actor de sistema; para una persona sigue «Nombre no disponible», con su referencia en el detalle técnico, hasta que la duda 71 fije la fuente nominal autorizada. Permiso propio, paginación, denegación CT136 y ACL comprobadas en PG18. Falta la consulta positiva con certificado en servidor y navegador tras reiniciar; los datos anteriores a VEC solo si constan en la fuente (duda 67). |
| 4.12 | Auditoría: valor anterior, valor nuevo y motivo. | Ficha RRHH, «Cambios», e historial CT. | **PARCIAL**: la columna «Qué cambió» muestra «anterior → nuevo» con los rótulos del catálogo único de fases y estados (peticiones) y de las siete situaciones de Bolsa; correo, teléfonos y datos de contacto se indican como «cambiado (dato protegido)», sin valores. «Motivo» muestra los publicables («Constitución de la bolsa») y marca el resto como reservado; B56 ya está en `main` y liga motivo y valores por recibo en Bolsa. En peticiones las actuaciones no guardan hoy clave de motivo (0 de 99 en el clon del hito 1), así que sale «No consta». Falta la consulta positiva con certificado y la decisión de RRHH sobre qué motivos y campos pueden mostrarse (duda 67). |
| 4.13 | Auditoría: documento o expediente relacionado. | Historial autorizado. | **PARCIAL**: la columna «Relacionado con» dice «Este expediente» o «Esta participación», y añade «con justificante» cuando la fuente liga un recibo; recibo, referencia y fuente siguen en el detalle técnico. La consulta de auditoría no trae el número visible (solo la referencia), por eso no repite el número que ya muestra la ficha. Falta la consulta positiva con certificado y la recuperación tras reinicio en servidor (duda 67). |
| 4.14 | Auditoría: IP o equipo cuando la política lo permita. | Auditoría segregada. | **HECHO (diseño VEC)**: la condición de la foto es «si la política lo permite»; CT117 omite IP/equipo por minimización y conserva actor, instante y correlación. No se inventa permiso para captarlos (duda 36). |
| 4.15 | Impedir alteraciones sin rastro y conservar historia. | Todas las operaciones y consulta de auditoría. | **PARCIAL**: CT008 y Bolsa conservan historia y recibos de solo adición; AD3-91/CT132/Bolsa48-B56 y el montaje del corte 2 `da48a409b` añaden lectura segregada con ACL y reversión protegida, ensayadas en PostgreSQL 18. La vista nombra las nueve actuaciones de peticiones y las de Bolsa (pausar, reactivar, excluir, situación, dato), y una acción sin rótulo sale como «Registró una actuación», nunca oculta. Falta la consulta positiva con certificado, el reinicio de la aplicación y comprobar cada acción del recorrido CT/Bolsa contra su registro (E06; duda 67). |

## Bloque 5: seguimiento del Departamento (25/09/2026)

Fuente: `fotos/2026-09-25 Seguimiento app de gestión Departamento.docx`, texto y
cinco capturas. Las capturas muestran tarjetas con recuento y «Ver trámites»,
tabla de categoría, fase y estado, menú actual «Contratación temporal» y el
error de carga en «Nueva petición». La petición escrita añade las dos vías del
expediente y el envío a firmas. **PENDIENTE** indica que aún no hay comprobación
o capacidad suficiente en `origin/main@e2ca2c061`; **PARCIAL** distingue las
piezas existentes de la función solicitada. No se atribuyen a `main` ramas
productoras todavía sin integrar.

Prioridad del bloque: **5.02 + 5.07**, luego **5.03**, después **5.08** y a
continuación **5.04–5.06**. **5.01** es transversal a todos esos recorridos. El cuadro
general del portal debe consumir vistas autorizadas de cada módulo sin duplicar
la autoridad de Bolsa ni de Contratación; resolver con Dirección su encaje con
el inicio común antes de implementar 5.03.

| Nº | Requisito concreto | Pantalla o ruta | Estado real y siguiente comprobación |
| --- | --- | --- | --- |
| 5.01 | Aplicación intuitiva para cualquier trabajador del Departamento, sin formación previa; rótulos claros, pocos pasos y ayuda desde «?». | Portal RRHH; recorridos CT y Bolsa. | **PARCIAL**: existen shell común, formularios por pasos y ayuda «?», pero falta revisión de usabilidad con el recorrido completo de ambos módulos y valoración de RRHH. Mantener PC prioritario, móvil usable, teclado y estados accesibles (E08). |
| 5.02 | Renombrar los textos visibles «Contratación temporal» a «Peticiones de personal temporal» en castellano e inglés. | Menú, portada, títulos y ayudas de la interfaz. | **PENDIENTE**: el menú de la captura aún dice «Contratación temporal»; comprobar y cambiar los catálogos i18n y textos visibles de `main` en ambas lenguas. No renombrar código, rutas, permisos ni referencias persistidas. |
| 5.03 | Abrir el portal RRHH en un cuadro de mandos general con «Expedientes en trámite», «Bolsas de trabajo» y «Ofertas al SAE»: tarjetas con recuento, «Ver trámites» y listas con categoría, fase y estado. | Primera pantalla del portal RRHH. | **PARCIAL, candidata no publicada** `trabajo/codexg-rrhh-503-montaje-20260928`: Inicio reúne consultas autorizadas de CT y Bolsa, recuentos, enlaces, estados de carga/error y ayuda ES/EN; SAE queda sin cifra ni enlace operativo porque falta fuente autorizada. Pasaron focales, grafo de caché y Chrome con datos sintéticos a 1440/390 px y reflow 200 %; faltan puerta global, publicación, recorrido con API/PostgreSQL y oferta SAE durable. No atribuir este corte a `main` ni a una lectura real de SAE. |
| 5.04 | Ofrecer en «Expediente» dos entradas, «Por bolsa de trabajo» y «Por oferta al SAE», cada una con relación de documentos y datos que se deben rellenar para esa vía. | Expediente RRHH y nueva petición. | **PARCIAL**: CT reconoce vías de cobertura gobernadas, incluida `oferta_sae`, y muestra formularios/avisos; falta la elección inicial de dos recorridos y la relación contextual de documentos y campos por vía desde catálogo versionado. No fijar por código una lista universal ni confundir elegir vía con seleccionar candidato. |
| 5.05 | Gestionar ofertas al Servicio Andaluz de Empleo como vía alternativa: oferta, candidatos remitidos y selección. | Expediente CT; ofertas SAE del cuadro. | **PARCIAL**: CT dispone de `oferta_sae` como vía de cobertura y puerto para fuentes, pero no consta un recorrido durable de oferta, remisión y selección ni conexión real con SAE. Construir el mínimo gobernado por catálogo y autorización; registrar en `dudas.md` el contrato y fuente externa pendientes antes de afirmar intercambio con SAE. |
| 5.06 | Enviar a Firmadoc y demás firmas y mostrar siempre la fase de firma en el expediente; AutoFirma para la firma del órgano. | Expediente CT, documentos y fase de firma. | **PARCIAL**: se generan borradores y el flujo prevé formalización/firma, pero falta un circuito acreditado de envío, estados, recepción, verificación y documento firmado. Definir adaptador Firmadoc apagado por defecto hasta conocer su API (duda en `dudas.md`); separar autenticación, borrador, firma del órgano con AutoFirma y eficacia del acto. |
| 5.07 | Comprobar y corregir el error «No se pudo cargar el cuadro. Reintente o contacte con soporte.» que aparece en «Nueva petición». | CT, pestaña «Nueva petición». | **PENDIENTE**: la captura acredita el fallo y `main` conserva el mensaje de error; no hay prueba de que la causa ya esté resuelta. Reproducir con la aplicación, API, autorización y base compatibles; corregir la causa y comprobar carga, reintento y ausencia de error en navegador. |
| 5.08 | «Mis preferencias» por persona: idioma, accesibilidad, tema, inicio, filas, avisos, correos verificados e imagen propia. | Solo desde el avatar de RRHH y área personal; sin entrada lateral. | **EN PREPARACIÓN LOCAL**: hay contrato de diseño y ramas de 5.08a, b y c, sin vertical publicada ni SQL instalado. Usuarios será propietario de preferencias y correos; Documentos custodiará la foto. Requiere V3 nominal, historia, auditoría, recibos, consumidor en ambos portales, clon PG18 con binario, navegador y revisiones sensibles. La pregunta DPD 73 está preparada en rama documental sin integrar; las elecciones de aviso no acreditan envío. |

5.08 se entrega por recorridos verificables: **a)** preferencias guardadas en
servidor con catálogo cerrado, versión y lectura al arrancar; **b)** varios
correos propios con uno solo activo y verificado, código de un uso sin token en
URL y consulta del activo por puerto para CT/Bolsa; **c)** iniciales, icono o
foto tratada en servidor y custodiada en Documentos, visible únicamente en la
audiencia interna con permiso nominal. Ningún corte se contará cerrado por
mostrar solamente el formulario. Los límites de filas de cada API permanecen;
si son menores que la preferencia, la interfaz explicará el límite efectivo.

## Bloqueos que no deben presentarse como funciones terminadas

1. Reglas temporales y adjudicación: el cálculo de ejemplo ya funciona por catálogo, pero sus parámetros y efectos definitivos requieren RRHH (dudas 1–3, 13–14).
2. Identidad externa y publicación individual: Mi Bolsa está montado con identidad de desarrollo; proveedor institucional y política pública definitiva siguen pendientes (duda 17).
3. Correo corporativo, SMS, WhatsApp y GINPIX: faltan integraciones y credenciales/contratos autorizados (dudas 10, 35, 39 y 45).
4. Firma y actos eficaces: hay borradores, cese y sanción durables de desarrollo; su validez y modelos oficiales requieren autoridad competente y Portafirmas (dudas 7, 33, 39 y 62).
5. Los datos sintéticos solo sirven para verificar recorridos y no sustituyen historia laboral o entrega externa.

## Comprobación del corte (28/09/2026)

- `main@eb2061f33` contiene los PR #78 y #80 (código de cortes 1 y 2) y #79 y #81 (operaciones de cortes 1 y 2). El PR #82 de revocación Mi Bolsa (`91cc66097`) está abierto y sin fusionar. El corte 3 sigue en rama separada; sus piezas y el cambio documental `5f6d51d3d` no forman parte de `main` por esta actualización del checklist.
- El histórico propio AD3-90/Bolsa44 pasó PostgreSQL 18 efímero y dos revisiones, y entró en el corte 1. Esto acredita código, no instalación ni consulta de servidor.
- La puerta completa de los cortes 1 y 2 de código fue verde antes de sus PR; los ensayos PostgreSQL 18 de reglas, auditoría y reincorporación fueron aislados. En este corte del 28/09 faltaban clon, instalación de diez SQL, identidad V3 y navegador con recuperación. Dirección comunicó su instalación y arranque estable el 29/09; esa noticia no cierra por sí sola los puntos PARCIAL ni acredita una firma. En particular, 3.06 carece de segunda validación durable por otra persona y 3.07 no impone el siguiente día hábil ni el horario 09:00–14:00; ambos requieren contrato transaccional pendiente.
- Los datos de prueba son sintéticos. Este corte solo actualiza documentación: no instala migraciones, modifica servicios compartidos ni envía correos o SMS.

## SQL acumulada de cidonia: 39 instaladas — NO reaplicar (29/09/2026)

Inventario acumulado fijado sobre
`origin/main@089933415e18e5c5cf710e40df865e911320d33d` (merge #97).
Las filas funcionales anteriores conservan su fotografía histórica y no se
reinterpretan aquí; C3 ya entró en `main` mediante #88. Este apartado actualiza
solo el inventario SQL y su estado de instalación comunicado por Dirección.
Son 39 rutas únicas en orden causal: **29 instaladas el 28/09/2026**
(27 `UP` y dos deltas DBA de rol de los cortes 1/2) y **10 instaladas el
29/09/2026** (cinco `UP` de C3 y cinco `UP` de #97). El corte 2 repite la lista
del corte 1: se incluye **una sola vez**. Las listas de respuesta semántica,
consulta de recibo y contexto ya están reunidas en la lista de #97: CT138,
AD3-104, CT139, AD3-105 y CT140 aparecen **una sola vez**.

**Estado comunicado por Dirección:** las 29 rutas de los cortes 1/2 quedaron
**instaladas el 28/09/2026 en cidonia/principal; NO reaplicar ninguna**.
AD3-94 y AD3-95 se instalaron con las correcciones del PR #84; conservar esa
historia y su postimagen. **Dirección comunicó que las cinco rutas de C3 y las
cinco de #97 se instalaron el 29/09/2026 en cidonia/principal; NO reaplicar.**
El ensayo anterior de #97 fue solo en clon; el hito 1 posterior arrancó estable
sobre `main@65241ff1` más el ajuste #102 de audiencia CT140, según Dirección.
Esta anotación no acredita por sí sola navegador, firma legal ni producción.
Para futuras migraciones se volverán a comprobar preimagen, roles y postimagen
contra la historia real. **No reaplicar migraciones históricas ni ejecutar
`DOWN` sobre historia conservada.** AD3-104 no arrastra AD3-102/103; no se
incorporan migraciones ajenas por su numeración.
Bolsa 000049 y Pública 000003 siguen excluidas por `NO-GO` del paquete de
los primeros cortes.

### Cortes 1 y 2: 29 rutas instaladas el 28/09/2026 — NO reaplicar

Fuente: `deploy/principal/piden_rrhh_corte2_20260928/migraciones.txt`.

**Cada una de las 29 rutas siguientes está instalada en cidonia/principal desde
el 28/09/2026: NO reaplicar.** Son 27 `UP` y dos deltas DBA de rol.

```text
deploy/postgresql/autorizacion_atestada_v3/migraciones/000090_consumidor_historial_propio_bolsa.up.sql
deploy/postgresql/bolsa_llamamientos/migraciones/000044_historial_propio_candidato.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000091_consumidor_consulta_auditoria_rrhh.up.sql
deploy/postgresql/contratacion_temporal/migraciones/000132_consulta_auditoria_ct.up.sql
deploy/postgresql/bolsa_llamamientos/migraciones/000048_consulta_auditoria_participacion.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000092_consumidor_reincorporacion_titular_ct.up.sql
deploy/postgresql/contratacion_temporal/migraciones/000130_reincorporacion_titular.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000093_consumidor_politica_ofertas_bolsa.up.sql
deploy/postgresql/bolsa_llamamientos/migraciones/000047_politica_ofertas_ejemplo.up.sql
deploy/postgresql/bolsa_llamamientos/migraciones/000045_restriccion_global_cese.up.sql
deploy/postgresql/contratacion_temporal/migraciones/000129_verificacion_cese_bolsa.up.sql
deploy/postgresql/bolsa_llamamientos/migraciones/000046_reincorporacion_titular_ct.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000094_consumidor_catalogo_plantillas_ct.up.sql
deploy/postgresql/contratacion_temporal/migraciones/000131_catalogo_plantillas_documentos.up.sql
deploy/postgresql/bolsa_llamamientos/migraciones/000050_lectura_estado_cese.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000095_consumidor_politica_cese_bolsa.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000096_consumidor_catalogo_plantillas_documental_ct.up.sql
deploy/postgresql/contratacion_temporal/migraciones/000133_obtener_catalogo_plantillas_publicado_documental.up.sql
deploy/postgresql/bolsa_llamamientos/migraciones/000053_mi_bolsa_disponibilidad_maxima.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000097_consumidor_consulta_politica_ofertas_bolsa.up.sql
deploy/postgresql/bolsa_llamamientos/roles_calculador_politica_up.sql
deploy/postgresql/bolsa_llamamientos/migraciones/000051_consulta_politica_ofertas_v3.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000098_consumidor_lectura_reincorporacion_ct.up.sql
deploy/postgresql/contratacion_temporal/migraciones/000134_lectura_reincorporacion_titular.up.sql
deploy/postgresql/bolsa_llamamientos/migraciones/000054_plazo_ofertas_48_horas.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000099_ambito_organizacion_plantillas_ct.up.sql
deploy/postgresql/contratacion_temporal/migraciones/000135_ambito_organizacion_plantillas.up.sql
deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql
deploy/postgresql/contratacion_temporal/migraciones/000136_auditoria_frontera_auditoria_ruta_exacta.up.sql
```

### Corte C3: cinco UP instaladas el 29/09/2026 — NO reaplicar

Fuente: `deploy/principal/lista_sql_trabajo_codexg_rrhh_c3_20260928.txt`.

```text
deploy/postgresql/autorizacion_atestada_v3/migraciones/000100_documental_tres_ambitos_ct.up.sql
deploy/postgresql/contratacion_temporal/migraciones/000137_documental_tres_ambitos.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000101_consumidor_consulta_reincorporacion_titular_bolsa.up.sql
deploy/postgresql/bolsa_llamamientos/migraciones/000055_lectura_reincorporacion_titular_v3.up.sql
deploy/postgresql/bolsa_llamamientos/migraciones/000056_motivo_traza_auditoria_participacion.up.sql
```

### Corte #97: cinco UP instaladas el 29/09/2026 — NO reaplicar

Fuente: `deploy/principal/lista_sql_trabajo_codexg_respuesta_integracion_20260928.txt`.

```text
deploy/postgresql/contratacion_temporal/migraciones/000138_respuesta_recibida_semantica.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000104_consumidor_consulta_recibo_respuesta.up.sql
deploy/postgresql/contratacion_temporal/migraciones/000139_consulta_recibo_respuesta.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000105_consumidor_consulta_comunicaciones_expediente_ct.up.sql
deploy/postgresql/contratacion_temporal/migraciones/000140_consulta_comunicaciones_expediente.up.sql
```

### Huellas SHA-256 de los cinco SQL del merge #97

| Ruta exacta | SHA-256 |
| --- | --- |
| `deploy/postgresql/contratacion_temporal/migraciones/000138_respuesta_recibida_semantica.up.sql` | `b2a95713037ba35809cdd730a9e02098705c50a544d19df53d9e84bea013e221` |
| `deploy/postgresql/autorizacion_atestada_v3/migraciones/000104_consumidor_consulta_recibo_respuesta.up.sql` | `fdbe756324e1cf127cc4dc0df215ca4c843183baf07287b1fec77f1926f8a655` |
| `deploy/postgresql/contratacion_temporal/migraciones/000139_consulta_recibo_respuesta.up.sql` | `e61b0dadd24c56146dc3c100b7e9fd860e1d46a55742702c1a4e7399c84d9d9b` |
| `deploy/postgresql/autorizacion_atestada_v3/migraciones/000105_consumidor_consulta_comunicaciones_expediente_ct.up.sql` | `e20d365484c18ddddba55f132a7b210da13ac69d03f940ef80bacbe7533f583e` |
| `deploy/postgresql/contratacion_temporal/migraciones/000140_consulta_comunicaciones_expediente.up.sql` | `30dfe1a3b20a9157c3571338581353bc6ae97a9be887f90e335c37808ad664d5` |

Estas huellas corresponden a los ficheros `UP` del árbol #97. La instalación
de las diez rutas se recoge según la comunicación de Dirección del 29/09;
las huellas del repositorio no sustituyen el registro de instalación de la
base ni elevan los pasos de RRHH a 6/8.
