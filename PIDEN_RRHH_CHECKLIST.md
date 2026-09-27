# Petición de RRHH: comprobación de las cuatro fotografías

Fuente: cuatro fotografías recibidas el 27/09/2026, conservadas fuera de Git en
`/home/alberto/Trabajo/piden_VEC/`. Coinciden con la transcripción anterior de
[`Peticion.pdf`](docs/estudio_requisitos/peticion_rrhh_transcripcion_y_lectura.md).
Esta lista descompone sus ejemplos en operaciones comprobables. **HECHO** exige
una ruta web conectada a un caso de uso real; **PARCIAL** indica la parte exacta
que existe; **FALTA** no se sustituye por una vista DEMO. La evidencia de código
no acredita por sí sola instalación ni entrega corporativa. Estado contrastado
con `main@3796cf010` el 27/09/2026.

## Fotografía 1: histórico, estados y transparencia

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 1.01 | Consultar contratos realizados en orden cronológico. | Portal personal «Mi Bolsa» y ficha RRHH de candidato. | **PARCIAL**: `internal/app/bootstrap/bolsa_mi_bolsa.go` monta la consulta propia; no presenta un histórico completo de contratos. Depende de la fuente de relaciones de Personal/GINPIX; duda 39. |
| 1.02 | Consultar llamamientos anteriores. | «Mi Bolsa» y ficha RRHH en `/portal-empleado/`. | **PARCIAL**: `internal/modules/bolsa/application/mibolsa/` y `web/static/portal-empleado/portal-panel-interno.js` muestran último llamamiento y antecedentes disponibles; falta acceso externo institucional B11. |
| 1.03 | Consultar renuncias y su resolución. | Ficha RRHH de Bolsa y detalle de Contratación. | **PARCIAL**: `internal/modules/bolsa/application/operacion_situacion_participacion.go` conserva operaciones B8; `web/static/portal-empleado/portal-bolsas-operaciones.js` consulta su historia. No hay histórico personal unificado. |
| 1.04 | Registrar y consultar sanciones con acto, motivo y competencia. | Ficha RRHH de candidato. | **FALTA**: `internal/modules/bolsa/domain/situacion_participacion.go` contiene situaciones, no una sanción ni su procedimiento. Falta fuente jurídica y competencia; duda 40. |
| 1.05 | Consultar cambios de estado anteriores y valor nuevo. | Ficha RRHH de candidato, pestaña «Histórico». | **PARCIAL**: `portal-bolsas-operaciones.js` presenta operaciones B8; `deploy/postgresql/bolsa_llamamientos/migraciones/000012_situacion_participacion.up.sql` conserva situaciones. Falta línea única de todos los cambios. |
| 1.06 | Consultar correos enviados, intentos y resultado. | Ficha RRHH de candidato, pestaña «Histórico». | **PARCIAL**: `internal/modules/bolsa/application/contacto_participacion.go` y `portal-bolsas-api.js` consultan contactos; el envío B7 conserva intentos, pero no acredita siempre entrega corporativa ni reúne todos los correos. |
| 1.07 | Consultar documentos generados y su versión. | Expediente de Contratación y ficha RRHH. | **PARCIAL**: `cliente-http-informe-definitivo.js` descarga seis borradores PDF/DOCX; falta índice durable completo de documentos generados y firmados. |
| 1.08 | Mostrar «Disponible». | Bolsa RRHH, ficha y lista pública. | **HECHO**: catálogo `internal/modules/bolsa/domain/situacion_participacion.go`, lista `portal-panel-interno.js` y consulta `web/static/bolsa/lista-bolsas-api.js`. |
| 1.09 | Mostrar «Trabajando». | Bolsa RRHH y lista pública. | **HECHO**: mismos contratos B2; `portal-bolsas-contrato.js` valida el estado y `portal-panel-interno.js` lo presenta. |
| 1.10 | Mostrar «No disponible». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 y cambio gobernado en `portal-bolsas-api.js`; pantalla de Bolsa. |
| 1.11 | Mostrar «Pendiente de incorporación». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 validada por `portal-bolsas-contrato.js` y renderizada por `portal-panel-interno.js`. |
| 1.12 | Mostrar «Renuncia». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 y operación B8 en `operacion_situacion_participacion.go`. |
| 1.13 | Mostrar «Excluido». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 y operación B8, con justificante en `portal-bolsas-operaciones.js`. |
| 1.14 | Mostrar «Disponible desde» con fecha. | Bolsa RRHH y lista pública. | **PARCIAL**: `portal-bolsas-api.js` envía `fecha_disponible` y `situacion_participacion.go` la conserva; no hay transición automática al vencer. |
| 1.15 | Calcular indisponibilidad tras contrato por regla configurable; ejemplo +5 meses desde 31/07/2026. | Ficha de situación de Bolsa y catálogo de reglas. | **FALTA**: B2 solo permite fecha registrada; ningún cálculo automático desde fin de contrato. RRHH debe fijar calendario, excepciones y versión: duda 13. No se aplica el ejemplo como norma. |
| 1.16 | Calcular indisponibilidad tras acumulación de tareas; ejemplo +9 meses. | Ficha de situación de Bolsa y catálogo de reglas. | **FALTA**: misma ausencia que 1.15; tampoco existe evento de cese corporativo fiable. Dudas 13 y 39. |
| 1.17 | Portal personal con acceso seguro e identidad del titular. | `/portal-empleado/`, «Mi Bolsa». | **PARCIAL**: `internal/app/bootstrap/bolsa_mi_bolsa.go` resuelve candidato desde contexto; B11 no tiene acceso externo institucional. DNI + clave de la foto fue sustituido por DNIe/certificado en `INSTRUCCIONES_DESATASCO.md`; nunca se usará DNI como contraseña. |
| 1.18 | El titular ve solo sus bolsas, posición y estado. | «Mi Bolsa». | **PARCIAL**: caso de uso `internal/modules/bolsa/application/mibolsa/`, API `/api/vec/bolsa/mi-bolsa`; falta acceso externo B11 y prueba navegador con esa identidad. |
| 1.19 | El titular ve último llamamiento, contratos y fecha de disponibilidad. | «Mi Bolsa». | **PARCIAL**: `mibolsa/` consulta datos propios; histórico contractual e identidad externa sin cerrar, como 1.01 y 1.17. |
| 1.20 | Consulta pública de integrantes de una bolsa con posición y estado publicables. | `/bolsa/`, `/api/publico/bolsa/bolsas/{bolsa_ref}/lista`. | **PARCIAL**: `web/static/bolsa/lista-bolsas-api.js` y `internal/modules/bolsa/publico/httpapi/bolsas.go` exponen lista minimizada; campos y plazo de publicación requieren decisión DPD/RRHH (duda 17). |

## Fotografía 2: propuesta y estructura de datos

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 2.01 | Gestionar bolsas vigentes y candidaturas. | `/portal-empleado/`, cuadro y ficha Bolsa. | **HECHO** para la gestión interna básica: `internal/app/bootstrap/bolsa_rrhh_desarrollo.go` monta `/api/vec/bolsa/bolsas`; `portal-bolsas-api.js` consume lista y ficha. |
| 2.02 | Proponer llamamientos según orden y Reglamento. | «Nuevo llamamiento» de Bolsa. | **PARCIAL**: `portal-panel-interno.js` presenta cuatro pasos y `internal/modules/bolsa/domain/llamamientos.go` conserva propuesta; política final de orden y respuestas pendiente de dudas 1–3, 13–14. No adjudica automáticamente. |
| 2.03 | Gestionar contratos. | Expediente de Contratación temporal. | **PARCIAL**: `web/static/portal-empleado/modulos/contratacion-temporal/` recorre propuesta, borradores e incorporación; falta firma y fuente contractual efectiva. |
| 2.04 | Registrar cese. | Expediente de Contratación, seguimiento. | **PARCIAL**: existen seguimiento y cierre de expediente; un cese laboral efectivo requiere fuente y autoridad de Personal/GINPIX (duda 39). |
| 2.05 | Registrar reincorporación y reflejarla en Bolsa. | Expediente de Contratación y ficha Bolsa. | **FALTA**: no hay operación completa de reincorporación laboral con efecto de Bolsa acreditado; duda 13 y fuente Personal/GINPIX. |
| 2.06 | Configurar y versionar reglas sin reprogramar cambios normativos. | Administración de Bolsa. | **PARCIAL**: `internal/modules/bolsa/application/gobiernoreglasbaremo/` gobierna baremos; faltan reglas temporales de disponibilidad y llamamiento con simulación/publicación, dudas 13–14. |
| 2.07 | Portal seguro del candidato. | «Mi Bolsa». | **PARCIAL**: ver 1.17. |
| 2.08 | Cuadro de mando de responsables y dirección. | Cuadro Bolsa en `/portal-empleado/`. | **HECHO** para indicadores operativos: `portal-panel-interno.js` consume `/api/vec/bolsa/estadisticas` montada en `bolsa_rrhh_desarrollo.go`. No sustituye informes oficiales. |
| 2.09 | Estadísticas y explotación por bolsa y situación. | «Estadísticas» de Bolsa. | **PARCIAL**: `portal-bolsas-api.js` valida `vec.bolsa.rrhh.estadisticas.v1`; `portal-panel-interno.js` presenta conteos. Faltan exportación y series históricas autorizadas. |
| 2.10 | Generar documentos Word y PDF. | Detalle de Contratación, descargas. | **PARCIAL**: seis borradores DOCX/PDF desde `cliente-http-informe-definitivo.js`; faltan tipos de foto 4 y documento final firmado. |
| 2.11 | Integrar correo electrónico. | «Nuevo llamamiento» y contactos. | **PARCIAL**: B7 admite relay de desarrollo, según `portal-panel-interno.js`; SMTP corporativo y resultado de entrega faltan (duda 10). |
| 2.12 | Integrar SMS y, si procede, mensajería instantánea. | «Nuevo llamamiento» y contactos. | **FALTA**: B7 indica canal correo; no hay conector institucional SMS/WhatsApp aprobado. Dudas 3 y 35. |
| 2.13 | Auditar acciones con trazabilidad. | Ficha RRHH e historial autorizado. | **PARCIAL**: operaciones B2/B8 y Contratación conservan auditoría/recibos; falta vista y cobertura integral de accesos y de todo el circuito. `ESPECIFICACIONES_AGENTES.md` E06. |
| 2.14 | Bolsa: id, nombre/categoría, vigencia, resolución aprobatoria y orden. | Lista/ficha Bolsa. | **PARCIAL**: `internal/modules/bolsa/domain/llamamientos.go` conserva identidad/vigencia; `portal-panel-interno.js` muestra categoría, vigencia y orden; resolución aprobatoria en todas las bolsas importadas requiere fuente Convoca (duda 16). |
| 2.15 | Candidato: pertenencia a bolsa, DNI protegido, nombre, apellidos, correo y dos teléfonos. | Ficha RRHH de candidato. | **PARCIAL**: pertenencia y contacto cifrado en `internal/modules/bolsa/application/datos_contacto_participacion.go`; DNI no es clave ni público. Falta acreditar totalidad de campos tras importación y fuente canónica de persona. |
| 2.16 | Situación: bolsa, candidato, posición, estado, disponibilidad y observaciones. | Ficha RRHH de candidato. | **PARCIAL**: `situacion_participacion.go`, `portal-panel-interno.js` y `portal-bolsas-api.js` cubren posición/estado/fecha; observaciones son actuaciones limitadas, no texto público universal. |

## Fotografía 3: control, comunicaciones y avisos

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 3.01 | Contar bolsas activas. | Cuadro Bolsa. | **HECHO**: `/api/vec/bolsa/estadisticas`, validación y presentación en `portal-bolsas-api.js` y `portal-panel-interno.js`. |
| 3.02 | Contar candidatos por bolsa. | Cuadro Bolsa. | **HECHO**: `portal-bolsas-contrato.js` valida `por_bolsa[].total`; vista `portal-panel-interno.js`. |
| 3.03 | Contar disponibles, trabajando, excluidos y no disponibles. | Cuadro Bolsa. | **HECHO**: `portal-bolsas-contrato.js` valida `por_estado`; vista `portal-panel-interno.js`. |
| 3.04 | Seleccionar destinatarios por estado y por orden para un envío. | «Nuevo llamamiento», paso 2. | **PARCIAL**: `portal-panel-interno.js` filtra disponibles y «disponible desde» y selecciona hasta 100; no es una campaña general para cualquier combinación de estados. |
| 3.05 | Enviar correo masivo y personalizado a cada destinatario. | «Nuevo llamamiento», pasos 3–4. | **PARCIAL**: B7 emite con asunto/cuerpo comunes (`portal-panel-interno.js`); «?» explica la personalización pendiente y el relay de desarrollo. Correo corporativo pendiente; dudas 10, 35 y 42. |
| 3.06 | Comunicar oferta, modalidad, duración y vía web de solicitud; adjudicar por orden. | «Nuevo llamamiento» y portal de candidato. | **PARCIAL**: B7 recoge referencia, modalidad, centro y fecha; la solicitud personal, duración gobernada y adjudicación automática no están conectadas. Dudas 1–3 y 14. |
| 3.07 | Continuar con llamamiento directo si la oferta no se cubre. | Contratación, «Llamamiento». | **PARCIAL**: continuación tras renuncia manual sintética en Contratación; falta regla general de plazo/no cobertura y envío directo corporativo. |
| 3.08 | Avisar de un candidato disponible saltado en el orden. | Cuadro Bolsa, avisos. | **HECHO** como aviso interno: `internal/modules/bolsa/application/avisos.go`, `/api/vec/bolsa/avisos`, `portal-bolsas-avisos.js`. No produce resolución automática. |
| 3.09 | Avisar un mes antes de tres años trabajando sin interrupción. | Cuadro Bolsa, avisos. | **PARCIAL**: `avisos.go` calcula sobre historia VEC y `portal-bolsas-avisos.js` muestra el límite; falta historia laboral anterior al 17/09/2026 y política validada (dudas 13, 39). |
| 3.10 | Consultar el límite provisional de los avisos desde «?», sin texto de ayuda ocupando el cuadro. | Cuadro Bolsa, «?». | **HECHO en esta rama**: `portal-bolsas-avisos.js` usa un control de divulgación cerrado por defecto; textos en el i18n común `portal-i18n.js` y estilo en `portal-componentes.css`. Prueba `portal-bolsas-avisos.test.mjs`. |
| 3.11 | Mantener las explicaciones del llamamiento tras «?» y mostrar el límite de transporte como estado. | «Nuevo llamamiento», pasos 2–3. | **HECHO en esta rama**: `portal-panel-interno.js` ofrece la ayuda cerrada y muestra «Relay de desarrollo»; `portal-i18n.js` gobierna los textos. `portal-bolsas.test.mjs` comprueba que la ayuda no se abre sola. |

## Fotografía 4: plantillas y auditoría

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 4.01 | Generar contrato laboral mediante plantilla Word. | Expediente de Contratación, «Documentos». | **FALTA**: `cliente-http-informe-definitivo.js` enumera seis borradores, no contrato laboral. Requiere fuente y plantilla aprobada, dudas 39 y 41. |
| 4.02 | Generar nombramiento mediante plantilla Word. | Expediente de Contratación. | **PARCIAL**: hay propuesta y borradores de resolución; ningún nombramiento firmado o eficaz. Falta Portafirmas y plantilla aprobada. |
| 4.03 | Generar toma de posesión. | Detalle de Contratación, documentos. | **PARCIAL**: `cliente-http-informe-definitivo.js` descarga borrador DOCX/PDF; no equivale a toma de posesión acreditada. |
| 4.04 | Generar cese. | Expediente de Contratación. | **FALTA**: la lista de seis borradores del cliente no incluye documento de cese. Requiere fuente laboral y plantilla aprobada (duda 41). |
| 4.05 | Generar modificación de nombramiento. | Expediente de Contratación. | **FALTA**: no consta entre los seis tipos de `cliente-http-informe-definitivo.js`; requiere acto y plantilla gobernados (duda 41). |
| 4.06 | Generar informes. | Detalle de Contratación, «Documentos». | **PARCIAL**: informe definitivo en PDF/DOCX como borrador, `cliente-http-informe-definitivo.js`. |
| 4.07 | Generar resoluciones. | Detalle de Contratación, «Documentos». | **PARCIAL**: resolución en PDF/DOCX como borrador, mismo cliente; falta firma admitida. |
| 4.08 | Configurar otros tipos de plantilla. | Administración documental. | **FALTA**: `web/static/portal-empleado/modulos/documentos/i18n.js` declara generación no conectada; falta gobierno de plantilla con versión/aprobación. Duda 33. |
| 4.09 | Combinar datos de expediente y coste aproximado por categoría. | Documento y ficha de Contratación. | **PARCIAL**: coste estimado del análisis se muestra con fuente o «sin calcular»; la descarga DOCX no acredita coste desde catálogo corporativo. Dudas 7 y 39. |
| 4.10 | Obtener campos de GINPIX/SAVIA o alternativa autorizada. | Expediente de Contratación, ficha GINPIX. | **PARCIAL**: `internal/modules/contrataciontemporal/adapters/httpinterno/ficha_ginpix_v2.go` prepara fichero de ejercicio; integración real con GINPIX y mapeo homologado faltan. Duda 39. |
| 4.11 | Auditoría: autor e instante exacto de cada cambio. | Historial autorizado RRHH/auditoría. | **PARCIAL**: B8 muestra actor y fecha (`portal-bolsas-operaciones.js`); no hay consulta integral de todas las acciones. |
| 4.12 | Auditoría: valor anterior, valor nuevo y motivo. | Ficha RRHH, «Histórico». | **PARCIAL**: B8 muestra instante `desde`, situación nueva y `motivo` en `portal-bolsas-operaciones.js`; **no** muestra el valor anterior ni cubre todas las operaciones. |
| 4.13 | Auditoría: documento o expediente relacionado. | Historial autorizado. | **PARCIAL**: operaciones B8 guardan referencia y huella de justificante; Contratación guarda referencias de expediente. Falta vista unificada autorizada. |
| 4.14 | Auditoría: IP o equipo cuando la política lo permita. | Auditoría segregada. | **FALTA** como campo general: E06/E07 exigen minimización; RRHH y seguridad deben aprobar finalidad, conservación y acceso (duda 36). |
| 4.15 | Impedir alteraciones sin rastro y conservar historia. | Todas las operaciones y consulta de auditoría. | **PARCIAL**: historia y recibos en B2/B8 y Contratación; el enunciado absoluto requiere cobertura de todos los efectos, ACL y prueba de recuperación en PostgreSQL. |

## Bloqueos que no deben presentarse como funciones terminadas

1. Reglas temporales y adjudicación: ejemplos de la foto no son una norma aprobada (dudas 1–3, 13–14).
2. Identidad externa y publicación individual: DNIe/certificado y política de datos públicos aún no acreditados (duda 17).
3. Correo corporativo, SMS, WhatsApp y GINPIX: faltan integración y credenciales/contratos autorizados (dudas 10, 35, 39).
4. Firma, contrato eficaz, cese y sanción: requieren competencia, acto, plantilla y fuente jurídica; un borrador no los acredita.
5. Los datos sintéticos solo sirven para verificar recorridos y no sustituyen historia laboral o entrega externa.

## Comprobación del corte de interfaz

- Rama `trabajo/piden-rrhh-20260927`, basada en `main@3796cf010`; aún no integrada ni desplegada.
- `scripts/verificar_calidad.sh`: verde (Go, carrera, `vet`, web, manifiestos, dependencias, vulnerabilidades y tamaño). Caché Go en `/dev/shm/go-build`.
- Capturas locales de los componentes de avisos y B7 a 1440/390 px con datos sintéticos en `/tmp/vec-avisos-preview-{1440,390}.png` y `/tmp/vec-b7-preview-{1440,390}.png`. Son previsualizaciones de componentes; no acreditan API, PostgreSQL ni navegador contra el runtime VEC.
- No se han instalado migraciones, modificado servicios compartidos ni enviado mensajes.
