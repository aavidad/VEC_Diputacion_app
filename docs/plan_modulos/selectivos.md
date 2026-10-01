# Procesos selectivos: inventario y siguiente trabajo

Estado comprobado en `origin/main@0a62a3ea6` el 1/10/2026. Este plan describe
la continuación de oposición, concurso y concurso-oposición. El ensayo A que se
preparaba hoy está conservado en ramas remotas, todavía fuera de `main`.

## Qué existe en `main`

| Pieza | Código y entrega | Estado real |
| --- | --- | --- |
| Convocatorias | `internal/modules/bolsa/domain/convocatoria_gobernada.go`, `internal/modules/bolsa/application/gobiernoconvocatorias/`, `internal/modules/bolsa/ports/convocatorias_gobierno_autorizacion.go` y migraciones `deploy/postgresql/bolsa_convocatorias/` | Hay dominio y contratos de gobierno, versión y consulta exacta. La lectura exacta sigue V1 y su `EXECUTE` de runtime está cerrado; no constituye una ficha RRHH V3 operativa. |
| Información pública y solicitud preparada | `internal/modules/bolsa/adapters/postgrespublico/convocatorias.go`, `web/static/area-personal/vistas/inicio-convocatorias.js`; #263 y #267 fusionadas | Consulta pública y preparación de solicitud visibles. La preparación no es presentación registrada con firma y justificante. |
| Baremación | `internal/modules/bolsa/domain/reglasbaremo/`, `cmd/vec-baremador/`, `cmd/vec-baremador-web/` y `web/static/portal-empleado/modulos/bolsa/baremo/`; #238, #242 y #245 fusionadas | Cálculo y edición de ejemplos; #250 añadió el ensayo de Concursos. No son calificaciones aprobadas de un proceso selectivo. |
| Convoca heredado | `internal/modules/bolsa/domain/importacionconvoca/` y `internal/modules/bolsa/adapters/xlsconvoca/`; #284 corrige su error visible | Importación y preparación reutilizables. Convoca no sustituye el circuito nuevo de admisión, tribunal y aprobados. |
| Selección propia | No existe `internal/modules/seleccion/` en `main` | No hay proceso selectivo completo ni traspaso aprobado a Bolsa o Personal. |

La inscripción externa se atribuye a Bolsa en
`docs/estudio_requisitos/convoca_inscripcion_externa_contrato_2026-10-01.md`.
Selección dirigirá el proceso y sus decisiones mediante puertos; reutilizará
esa inscripción y el baremador existentes. Bolsa recibirá una lista aprobada y
Personal conservará la relación y ocupación que procedan. No se duplicarán sus
tablas ni se deducirá un alta de una puntuación.

## Huecos frente al catálogo

| Capacidad | Falta para el recorrido solicitado |
| --- | --- |
| SEL-001 | Unir necesidad, crédito, plaza, OEP, convocatoria y ejecución por referencias y versiones de RPT/Personal. |
| SEL-002 y SEL-003 | Ficha RRHH de bases exactas autorizada por V3; gobierno, firma y publicación de cada versión con anexos, plazos y retirada trazable. |
| SEL-004 y SEL-005 | Solicitud recuperable y presentada con representación, documentos, firma, registro y recibo; tasa, exención y conciliación según bases. |
| SEL-006 | Admisión motivada por requisito, subsanación y listas provisional y definitiva aprobadas. |
| SEL-007 | Designación, recusación y sustitución de tribunal, acceso limitado por proceso y fase, actas y firma. |
| SEL-008 | Ejercicios, mínimos, anonimato cuando proceda, incidencias, calificación por fase y publicación aprobada. |
| SEL-009 | Alegación o recurso con traslado, resolución y rectificación que conserve cada acto anterior. |
| BAR-001 a BAR-006 | Aplicar el baremador común a bases versionadas y méritos acreditados; revisión y resultado definitivo con desglose y firma. El cálculo de ejemplo no resuelve estas decisiones. |
| RUM-001 a RUM-003 | Consumir hechos y evidencias por contrato, distinguiendo requisito, mérito, validez documental y previsión permitida por las bases. |

## Minitareas en orden

Cada fila produce una PR separada con un recorrido visible. Los archivos son
write-sets previstos; antes de programar se comprueba su propietario y el turno
de padres, manifiestos, importadores y `dudas.md`.

| Orden y responsable | Archivos previstos y dependencia | Criterio de cierre |
| --- | --- | --- |
| S0 · A, ensayo ya iniciado | Reunir `internal/modules/seleccion/**`, `cmd/vec-baremador-web/seleccion.go`, montaje HTTP y `web/static/portal-empleado/modulos/seleccion/**` desde las tres ramas WIP citadas abajo; catálogos ES/EN propios. Sin SQL. | Tres modalidades con ejemplos retirables, reglas configurables y desglose del baremador común en Chrome; sin admisión ni nota oficial. Corregir claves duplicadas de ES, crear EN, pruebas focales y revisión de HTTP/UX antes de PR. |
| S1 · A y propietario de Bolsa | Nuevo puerto `seleccion/ports/convocatoria.go`, caso de uso de consulta, adaptador a Bolsa y ficha RRHH propia. Adaptar `bolsa/application/convocatorias_consulta_interna.go` y `bolsa/adapters/postgres/convocatorias_consulta.go`; SQL AD3 y Bolsa **solo en borrador** tras reservar números en `RESERVAS_MIGRACIONES.md` y fijar orden con D en `ORDEN_SQL_NUCLEO.md`. | Versión exacta, bases, requisitos, fases y huella visibles con lectura V3 nominal y auditoría en la transacción; 403 sin concesión, 503 si cae la dependencia, recuperación tras reiniciar. Ensayo en clon y dos revisiones sensibles antes de integrar. |
| S2 · A y Bolsa | Reutilizar `gobiernoconvocatorias` y contratos de Documentos/Firma; adaptadores y vista RRHH propios. Depende de S1 y actos/órganos de la pregunta 110. | Preparar una versión, aprobar/publicar mediante autoridad competente y conservarla cuando se rectifique; recibos, firma y fuente oficial comprobados. |
| S3 · A, Bolsa y Registro/Documentos | Consumir la preparación de solicitud de Bolsa por puerto; casos de uso y vistas propias del aspirante. Depende de S2, identidad común, representación, firma, registro y reglas de tasa SEL-005. | Persona interna o externa presenta una solicitud, recupera el mismo recibo y puede corregirla dentro del plazo de sus bases sin duplicar la inscripción. |
| S4 · A, RRHH y fuente de méritos | Casos de uso de requisitos/admisión, listas y subsanación; puerto RUM/Personal con datos mínimos. Depende de S3 y perfiles fijos. | RRHH motiva cada admisión o exclusión, publica listas provisional/definitiva tras aprobación y conserva subsanaciones e historia. |
| S5 · A, autoridad común y tribunal | Gobierno de miembros/abstenciones por proceso y fase, sesiones, actas; vistas propias. Depende de S2/S4 y respuesta 110. | Miembros habilitados ven solo su fase; recusación/sustitución deja historia; acta firmada antes de publicar calificaciones. |
| S6 · A y baremador de Bolsa | Casos de uso de ejercicios, notas por fase y méritos; consumidores del motor común. Depende de S4/S5, bases exactas y evidencia admitida. | Mínimos, fases y empates se explican; ninguna nota pendiente se transforma en aprobado. Calificación y publicación conservan versión, autor y acto. |
| S7 · A, Bolsa y Personal | Resultado aprobado, entrega idempotente por puertos a Bolsa o Personal; alegaciones y rectificaciones SEL-009. Depende de S6, acto y datos de la pregunta 111. | Lista de aprobados y recibo del receptor; reintento no duplica bolsa, plaza ni relación, y una rectificación enlaza el acto anterior. |
| S8 · A, RPT/Personal | Relación OEP–plaza–convocatoria SEL-001. Depende del modelo histórico RPT y autoridad de vacantes. | RRHH ve grado de ejecución con cada plaza y acto de cobertura, sin tratar una propuesta como vacante cubierta. |

La consulta exacta actual merece una corrección conjunta al hacer S1: el SQL
`bolsa_convocatorias/000002` devuelve `encontrada`, mientras el adaptador Go V1
espera `obtenida`. Abrir una ACL por sí sola no arregla esa discrepancia. S1
debe establecer un estado nominal consistente y auditar también la denegación
según el contrato autorizado. La migración V3 nueva no se instala por este plan.

## Decisiones pendientes

`dudas.md` ya recibió las preguntas **109** (bases/versiones por modalidad),
**110** (órganos, perfiles y actos de admisión, tribunal y resultado) y **111**
(acto y datos mínimos del traspaso). Se conservan las preguntas 90–93 sobre
conflictos de interés, acceso y publicación protegida. Una configuración de
órganos no concede permisos: los perfiles serán fijos y su provisión seguirá
huella y CAS.

## Por dónde empezar mañana y trabajo conservado

Primera tarea: **S0**, reconciliar las tres ramas WIP sobre el `main` vigente y
cerrar el ensayo antes de abrir PR de producto. Hoy no hay PR de Selección
abierta ni instalación SQL realizada por A.

| Rama remota | SHA y contenido | Estado al parar |
| --- | --- | --- |
| `trabajo/codexa-selectivos-motor-20261001` | `9f781d286`: dominio, aplicación, puerto, adaptador al baremador y tres ejemplos sintéticos | Pruebas Go focales verdes; falta composición con HTTP/web y revisión final. |
| `trabajo/codexa-selectivos-web-20261001` | `fcdfc852d`: formulario, cliente y vista local | Incompleta: falta catálogo EN, corregir claves duplicadas ES, tests y navegador. No servir todavía. |
| `trabajo/codexa-selectivos-inventario-20261001` | `e6f640cb1`: montaje HTTP/README/test WIP, apilado sobre Personal B; preguntas 109–111 en `749f3567e` | Falta reunir motor/web y comprobar el recorrido. No tiene PR. |

Los tres SHA están publicados para que el siguiente agente recupere el trabajo
sin reconstruirlo. La PR de este fichero documenta el plan; no integra esos
WIP ni declara Selectivos completo.
