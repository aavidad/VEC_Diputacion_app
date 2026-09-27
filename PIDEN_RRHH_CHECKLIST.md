# Petición de RRHH: comprobación de las cuatro fotografías

Fuente: cuatro fotografías recibidas el 27/09/2026, conservadas fuera de Git en
`/home/alberto/Trabajo/piden_VEC/`. Coinciden con la transcripción anterior de
[`Peticion.pdf`](docs/estudio_requisitos/peticion_rrhh_transcripcion_y_lectura.md).
Esta lista descompone sus ejemplos en operaciones comprobables. **HECHO** exige
una ruta web conectada a un caso de uso real; **PARCIAL** indica la parte exacta
que existe; **FALTA** no se sustituye por una vista DEMO. La evidencia de código
no acredita por sí sola instalación ni entrega corporativa. Estado contrastado
con `origin/main@7247682c` el 27/09/2026; el entorno servido solo se afirma
cuando `ESTADO_PROYECTO.md` documenta su comprobación. La ayuda visible se
abre desde «?» conforme al shell vigente, sin convertirla en efecto de negocio.

## Fotografía 1: histórico, estados y transparencia

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 1.01 | Consultar contratos realizados en orden cronológico. | Portal personal «Mi Bolsa» y ficha RRHH de candidato. | **PARCIAL**: la ficha RRHH ya consulta por participación `portal-bolsas-contratos.js` sobre `application/contratos_participacion.go`; «Mi Bolsa» aún no ofrece ese histórico completo y la fuente anterior a VEC depende de Personal/GINPIX (duda 39). |
| 1.02 | Consultar llamamientos anteriores. | «Mi Bolsa» y ficha RRHH en `/portal-empleado/`. | **PARCIAL**: `application/mibolsa/` muestra el último resultado y `portal-panel-interno.js` consulta el histórico interno; «Mi Bolsa» aún no reúne todos los llamamientos antiguos y la identidad externa sigue siendo de desarrollo. |
| 1.03 | Consultar renuncias y su resolución. | Ficha RRHH de Bolsa y detalle de Contratación. | **PARCIAL**: `internal/modules/bolsa/application/operacion_situacion_participacion.go` conserva operaciones B8; `web/static/portal-empleado/portal-bolsas-operaciones.js` consulta su historia. No hay histórico personal unificado. |
| 1.04 | Registrar y consultar sanciones con acto, motivo y competencia. | Ficha RRHH de candidato, «Sanciones». | **PARCIAL**: `portal-bolsas-sanciones.js` y `application/sanciones_participacion.go` registran y consultan sanción, consecuencia, resolución y recurso con recibo. El catálogo de ejemplo no aprueba todas las penalizaciones de Granada; duda 62. |
| 1.05 | Consultar cambios de estado anteriores y valor nuevo. | Ficha RRHH de candidato, «Histórico» y «Cambios». | **PARCIAL**: `portal-bolsas-traza-valores.js` consulta antes/después de situación y contacto mediante la migración Bolsa 000034; falta línea única de toda actuación y todo documento. |
| 1.06 | Consultar correos enviados, intentos y resultado. | Ficha RRHH de candidato, pestaña «Histórico». | **PARCIAL**: `internal/modules/bolsa/application/contacto_participacion.go` y `portal-bolsas-api.js` consultan contactos; el envío B7 conserva intentos, pero no acredita siempre entrega corporativa ni reúne todos los correos. |
| 1.07 | Consultar documentos generados y su versión. | Expediente de Contratación y módulo Documentos. | **PARCIAL**: `cliente-http-informe-definitivo.js` descarga borradores PDF/DOCX del catálogo versionado y `modulos/documentos/` permite consultar piezas propias; falta un índice único de todos los documentos del candidato con firma legal acreditada. |
| 1.08 | Mostrar «Disponible». | Bolsa RRHH, ficha y lista pública. | **HECHO**: catálogo `internal/modules/bolsa/domain/situacion_participacion.go`, lista `portal-panel-interno.js` y consulta `web/static/bolsa/lista-bolsas-api.js`. |
| 1.09 | Mostrar «Trabajando». | Bolsa RRHH y lista pública. | **HECHO**: mismos contratos B2; `portal-bolsas-contrato.js` valida el estado y `portal-panel-interno.js` lo presenta. |
| 1.10 | Mostrar «No disponible». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 y cambio gobernado en `portal-bolsas-api.js`; pantalla de Bolsa. |
| 1.11 | Mostrar «Pendiente de incorporación». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 validada por `portal-bolsas-contrato.js` y renderizada por `portal-panel-interno.js`. |
| 1.12 | Mostrar «Renuncia». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 y operación B8 en `operacion_situacion_participacion.go`. |
| 1.13 | Mostrar «Excluido». | Bolsa RRHH y lista pública. | **HECHO**: situación B2 y operación B8, con justificante en `portal-bolsas-operaciones.js`. |
| 1.14 | Mostrar «Disponible desde» con fecha. | Bolsa RRHH, Mi Bolsa y lista pública. | **HECHO** para el turno: la situación conserva fecha y la consulta SQL de orden vigente (Bolsa 000037) considera disponible a la persona al vencer; `portal-bolsas-reglas-situacion.js` presenta fecha y procedencia. No se crea otra actuación al vencer. |
| 1.15 | Calcular indisponibilidad tras contrato por regla configurable; ejemplo +5 meses desde 31/07/2026. | Ficha de situación de Bolsa y catálogo de reglas. | **PARCIAL**: `adapters/reglas/reglas_situacion.go` calcula la fecha con `b14.reposicion_general` y `/api/vec/bolsa/reglas-situacion` la propone a RRHH; se aplica tras confirmación, sin cese laboral corporativo automático. El ejemplo del catálogo sigue pendiente de RRHH (duda 13). |
| 1.16 | Calcular indisponibilidad tras acumulación de tareas; ejemplo +9 meses. | Ficha de situación de Bolsa y catálogo de reglas. | **PARCIAL**: el mismo cálculo selecciona `b14.reposicion_acumulacion_tareas` por modalidad en `data/demo/reglas/bolsa_reglas.ejemplo.demo.json`; falta activar automáticamente el efecto desde cese acreditado y aprobar parámetros (dudas 13 y 39). |
| 1.17 | Portal personal con acceso seguro e identidad del titular. | Área personal, «Mi Bolsa». | **PARCIAL**: B11 ya está montado y desplegado en desarrollo (`internal/app/bootstrap/bolsa_mi_bolsa.go`); deriva candidato de certificado y sesión del servidor. Sigue siendo identidad sintética de desarrollo, sin proveedor institucional aprobado. DNI + clave fue sustituido por DNIe/certificado en `INSTRUCCIONES_DESATASCO.md`. |
| 1.18 | El titular ve solo sus bolsas, posición y estado. | Área personal, «Mi Bolsa». | **PARCIAL**: `application/mibolsa/portal.go` y la consulta propia `GET /api/vec/bolsa/mi-bolsa` están montadas en desarrollo con ámbito de candidato; falta identidad institucional y acreditación con datos reales autorizados. |
| 1.19 | El titular ve último llamamiento, contratos y fecha de disponibilidad. | Área personal, «Mi Bolsa». | **PARCIAL**: `application/mibolsa/` ofrece situación, fecha y último resultado de correo; el histórico completo de contratos solo está en la ficha RRHH y la identidad externa es de desarrollo. |
| 1.20 | Consulta pública de integrantes de una bolsa con posición y estado publicables. | `/bolsa/`, `/api/publico/bolsa/bolsas/{bolsa_ref}/lista`. | **PARCIAL**: `web/static/bolsa/lista-bolsas-api.js` y `internal/modules/bolsa/publico/httpapi/bolsas.go` exponen lista minimizada; campos y plazo de publicación requieren decisión DPD/RRHH (duda 17). |

## Fotografía 2: propuesta y estructura de datos

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 2.01 | Gestionar bolsas vigentes y candidaturas. | `/portal-empleado/`, cuadro y ficha Bolsa. | **HECHO** para la gestión interna básica: `internal/app/bootstrap/bolsa_rrhh_desarrollo.go` monta `/api/vec/bolsa/bolsas`; `portal-bolsas-api.js` consume lista y ficha. |
| 2.02 | Proponer llamamientos según orden y Reglamento. | «Nuevo llamamiento» de Bolsa. | **PARCIAL**: `portal-panel-interno.js` recorre cuatro pasos y consulta el orden vigente de Bolsa, con propuesta/recibo durables; el catálogo de ejemplo gobierna excepciones y respuesta, pero la política final y la adjudicación automática dependen de RRHH (dudas 1–3, 13–14). |
| 2.03 | Gestionar contratos. | Expediente de Contratación temporal. | **PARCIAL**: `web/static/portal-empleado/modulos/contratacion-temporal/` recorre propuesta, borradores e incorporación; falta firma y fuente contractual efectiva. |
| 2.04 | Registrar cese. | Expediente de Contratación, «Seguimiento y cese». | **HECHO** como registro de desarrollo: `seguimiento-cese.js` consume `/api/vec/contratacion-temporal/ceses`, caso de uso `application/cese_cierre_modificacion.go`, PostgreSQL, recibo e historia; el recorrido en copia de ensayo con reinicio consta en `ESTADO_PROYECTO.md` (26/09). No acredita baja laboral corporativa. |
| 2.05 | Registrar reincorporación y reflejarla en Bolsa. | Expediente de Contratación y ficha Bolsa. | **FALTA**: no hay operación completa de reincorporación laboral con efecto de Bolsa acreditado; duda 13 y fuente Personal/GINPIX. |
| 2.06 | Configurar y versionar reglas sin reprogramar cambios normativos. | «Reglas vigentes» de Bolsa; configuración externa. | **PARCIAL**: `internal/vec/reglas/` y `data/demo/reglas/bolsa_reglas.ejemplo.demo.json` resuelven reglas versionadas de situación, reposición y avisos; falta administración web gobernada para revisar/publicar versiones definitivas (dudas 13–14, 33). |
| 2.07 | Portal seguro del candidato. | «Mi Bolsa». | **PARCIAL**: ver 1.17. |
| 2.08 | Cuadro de mando de responsables y dirección. | Cuadro Bolsa en `/portal-empleado/`. | **HECHO** para indicadores operativos: `portal-panel-interno.js` consume `/api/vec/bolsa/estadisticas` montada en `bolsa_rrhh_desarrollo.go`. No sustituye informes oficiales. |
| 2.09 | Estadísticas y explotación por bolsa y situación. | «Estadísticas» de Bolsa. | **PARCIAL**: `portal-bolsas-api.js` valida `vec.bolsa.rrhh.estadisticas.v1`; `portal-panel-interno.js` presenta conteos. Faltan exportación y series históricas autorizadas. |
| 2.10 | Generar documentos Word y PDF. | Detalle de Contratación, descargas. | **PARCIAL**: `cliente-http-informe-definitivo.js` ofrece DOCX/PDF de los tipos del catálogo `data/demo/plantillas/ct_plantillas_documentos.ejemplo.demo.json`; son borradores, sin firma legal ni custodia final. |
| 2.11 | Integrar correo electrónico. | «Nuevo llamamiento» y contactos. | **PARCIAL**: `application/emision_llamamiento.go` y B7 envían al relay de desarrollo y conservan resultado por contacto; faltan SMTP corporativo y prueba de entrega institucional (dudas 10 y 45). |
| 2.12 | Integrar SMS y, si procede, mensajería instantánea. | «Nuevo llamamiento» y contactos. | **FALTA**: B7 indica canal correo; no hay conector institucional SMS/WhatsApp aprobado. Dudas 3 y 35. |
| 2.13 | Auditar acciones con trazabilidad. | Ficha RRHH, «Cambios», y auditoría autorizada. | **PARCIAL**: `portal-bolsas-traza-valores.js`, operaciones B8/B24 y Contratación muestran recibos e historia con actor/instante; falta probar cobertura completa de todas las acciones y accesos conforme a E06. |
| 2.14 | Bolsa: id, nombre/categoría, vigencia, resolución aprobatoria y orden. | Lista/ficha Bolsa. | **PARCIAL**: `internal/modules/bolsa/domain/llamamientos.go` conserva identidad/vigencia; `portal-panel-interno.js` muestra categoría, vigencia y orden; resolución aprobatoria en todas las bolsas importadas requiere fuente Convoca (duda 16). |
| 2.15 | Candidato: pertenencia a bolsa, DNI protegido, nombre, apellidos, correo y dos teléfonos. | Ficha RRHH de candidato. | **PARCIAL**: participación y contacto cifrado con dos teléfonos en `application/datos_contacto_participacion.go`; DNI queda enmascarado en público. La importación Convoca y fuente canónica de contacto requieren validación (dudas 16 y 45). |
| 2.16 | Situación: bolsa, candidato, posición, estado, disponibilidad y observaciones. | Ficha RRHH de candidato. | **PARCIAL**: `situacion_participacion.go`, `portal-panel-interno.js` y `portal-bolsas-api.js` cubren posición/estado/fecha; observaciones son actuaciones limitadas, no texto público universal. |

## Fotografía 3: control, comunicaciones y avisos

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 3.01 | Contar bolsas activas. | Cuadro Bolsa. | **HECHO**: `/api/vec/bolsa/estadisticas`, validación y presentación en `portal-bolsas-api.js` y `portal-panel-interno.js`. |
| 3.02 | Contar candidatos por bolsa. | Cuadro Bolsa. | **HECHO**: `portal-bolsas-contrato.js` valida `por_bolsa[].total`; vista `portal-panel-interno.js`. |
| 3.03 | Contar disponibles, trabajando, excluidos y no disponibles. | Cuadro Bolsa. | **HECHO**: `portal-bolsas-contrato.js` valida `por_estado`; vista `portal-panel-interno.js`. |
| 3.04 | Seleccionar destinatarios por estado y por orden para un envío. | «Nuevo llamamiento», paso 2. | **PARCIAL**: `portal-panel-interno.js` filtra disponibles y «disponible desde» y selecciona hasta 100; no es una campaña general para cualquier combinación de estados. |
| 3.05 | Enviar correo masivo y personalizado a cada destinatario. | «Nuevo llamamiento», pasos 3–4. | **PARCIAL**: `portal-bolsas-correo.js` usa plantilla versionada, marcadores y vista previa; `application/emision_llamamiento.go` compone cada correo antes de reservar. Funciona con relay de desarrollo; SMTP corporativo y entrega acreditada siguen pendientes (dudas 10, 35 y 45). |
| 3.06 | Comunicar oferta, modalidad, duración y vía web de solicitud; adjudicar por orden. | «Nuevo llamamiento» y «Mi Bolsa». | **PARCIAL**: B7 recoge necesidad, modalidad, centro y fecha; `application/mibolsa/disposicion.go` permite respuesta propia a oferta, pero la adjudicación automática y plazos definitivos no están aprobados (dudas 1–3 y 14). |
| 3.07 | Continuar con llamamiento directo si la oferta no se cubre. | Contratación, «Llamamiento». | **PARCIAL**: continuación tras renuncia manual sintética en Contratación; falta regla general de plazo/no cobertura y envío directo corporativo. |
| 3.08 | Avisar de un candidato disponible saltado en el orden. | Cuadro Bolsa, avisos. | **HECHO** como aviso interno: `internal/modules/bolsa/application/avisos.go`, `/api/vec/bolsa/avisos`, `portal-bolsas-avisos.js`. No produce resolución automática. |
| 3.09 | Avisar un mes antes de tres años trabajando sin interrupción. | Cuadro Bolsa, avisos. | **PARCIAL**: `application/avisos.go` calcula periodos según `ports/politica_avisos.go` y `portal-bolsas-avisos.js` los muestra; la fuente histórica de trabajo anterior a VEC y el umbral definitivo requieren RRHH/Personal (dudas 13 y 39). |

## Fotografía 4: plantillas y auditoría

| Nº | Requisito concreto | Pantalla o ruta | Estado y prueba en el código / bloqueo |
| --- | --- | --- | --- |
| 4.01 | Generar contrato laboral mediante plantilla Word. | Expediente de Contratación, «Documentos». | **PARCIAL**: `cliente-http-informe-definitivo.js` y el catálogo `data/demo/plantillas/ct_plantillas_documentos.ejemplo.demo.json` ofrecen borrador DOCX/PDF de contrato laboral. Requiere modelo aprobado, firma y fuente laboral oficial (dudas 7, 33 y 39). |
| 4.02 | Generar nombramiento mediante plantilla Word. | Expediente de Contratación. | **PARCIAL**: el mismo cliente ofrece borrador DOCX/PDF de nombramiento; no acredita nombramiento firmado o eficaz. Falta Portafirmas oficial y plantilla aprobada. |
| 4.03 | Generar toma de posesión. | Detalle de Contratación, documentos. | **PARCIAL**: `cliente-http-informe-definitivo.js` descarga borrador DOCX/PDF; no equivale a toma de posesión acreditada. |
| 4.04 | Generar cese. | Expediente de Contratación. | **PARCIAL**: `cliente-http-informe-definitivo.js` permite borrador DOCX/PDF de cese tras la actuación CT115; falta modelo aprobado, firma y efecto laboral externo (dudas 7 y 39). |
| 4.05 | Generar modificación de nombramiento. | Expediente de Contratación. | **PARCIAL**: el cliente ofrece borrador DOCX/PDF ligado a la modificación; faltan modelo aprobado, firma y eficacia administrativa (dudas 7 y 33). |
| 4.06 | Generar informes. | Detalle de Contratación, «Documentos». | **PARCIAL**: informe definitivo en PDF/DOCX como borrador, `cliente-http-informe-definitivo.js`. |
| 4.07 | Generar resoluciones. | Detalle de Contratación, «Documentos». | **PARCIAL**: resolución en PDF/DOCX como borrador, mismo cliente; falta firma admitida. |
| 4.08 | Configurar otros tipos de plantilla. | Configuración externa de Contratación; Administración documental. | **PARCIAL**: `adapters/informejuridico/plantillas_borrador.go` valida catálogo versionado y `data/demo/plantillas/` aporta modelos de ejemplo; faltan editor web, revisión/publicación competente y nuevos tipos aprobados (dudas 7 y 33). |
| 4.09 | Combinar datos de expediente y coste aproximado por categoría. | Documento y ficha de Contratación. | **PARCIAL**: coste estimado del análisis se muestra con fuente o «sin calcular»; la descarga DOCX no acredita coste desde catálogo corporativo. Dudas 7 y 39. |
| 4.10 | Obtener campos de GINPIX/SAVIA o alternativa autorizada. | Expediente de Contratación, ficha GINPIX. | **PARCIAL**: `internal/modules/contrataciontemporal/adapters/httpinterno/ficha_ginpix_v2.go` prepara fichero de ejercicio; integración real con GINPIX y mapeo homologado faltan. Duda 39. |
| 4.11 | Auditoría: autor e instante exacto de cada cambio. | Historial autorizado RRHH/auditoría. | **PARCIAL**: B8 muestra actor y fecha (`portal-bolsas-operaciones.js`); no hay consulta integral de todas las acciones. |
| 4.12 | Auditoría: valor anterior, valor nuevo y motivo. | Ficha RRHH, «Cambios», e historial CT. | **PARCIAL**: `portal-bolsas-traza-valores.js` y `modulos/contratacion-temporal/vista-expedientes-cambios.js` muestran antes/después; B8 conserva motivo. Falta verificar cobertura de cada acción administrativa con la misma precisión. |
| 4.13 | Auditoría: documento o expediente relacionado. | Historial autorizado. | **PARCIAL**: operaciones B8 guardan referencia y huella de justificante; Contratación guarda referencias de expediente. Falta vista unificada autorizada. |
| 4.14 | Auditoría: IP o equipo cuando la política lo permita. | Auditoría segregada. | **FALTA** como campo general: E06/E07 exigen minimización; RRHH y seguridad deben aprobar finalidad, conservación y acceso (duda 36). |
| 4.15 | Impedir alteraciones sin rastro y conservar historia. | Todas las operaciones y consulta de auditoría. | **PARCIAL**: historia y recibos en B2/B8 y Contratación; el enunciado absoluto requiere cobertura de todos los efectos, ACL y prueba de recuperación en PostgreSQL. |

## Bloqueos que no deben presentarse como funciones terminadas

1. Reglas temporales y adjudicación: el cálculo de ejemplo ya funciona por catálogo, pero sus parámetros y efectos definitivos requieren RRHH (dudas 1–3, 13–14).
2. Identidad externa y publicación individual: Mi Bolsa está montado con identidad de desarrollo; proveedor institucional y política pública definitiva siguen pendientes (duda 17).
3. Correo corporativo, SMS, WhatsApp y GINPIX: faltan integraciones y credenciales/contratos autorizados (dudas 10, 35, 39 y 45).
4. Firma y actos eficaces: hay borradores, cese y sanción durables de desarrollo; su validez y modelos oficiales requieren autoridad competente y Portafirmas (dudas 7, 33, 39 y 62).
5. Los datos sintéticos solo sirven para verificar recorridos y no sustituyen historia laboral o entrega externa.

## Comprobación del corte

- Rama `trabajo/piden-rrhh-20260927`, rebasada sobre `origin/main@7247682c`; aún no integrada ni desplegada.
- `scripts/verificar_calidad.sh`: verde sobre la base vigente (Go, carrera, `vet`, web, manifiestos, dependencias, vulnerabilidades y tamaño), con caché Go en `/dev/shm/go-build`.
- Capturas locales del panel de avisos con datos sintéticos en `/tmp/vec-avisos-actual-1440.png` y `/tmp/vec-avisos-actual-390.png`; son previsualizaciones de componente, no un recorrido de API/PostgreSQL del runtime VEC.
- No se han instalado migraciones, modificado servicios compartidos ni enviado correos o SMS.
