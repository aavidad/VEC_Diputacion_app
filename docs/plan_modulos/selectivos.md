# Procesos selectivos: inventario y siguiente trabajo

**Aparcado hasta cerrar Bolsa y CT (orden de Alberto, 07/10/2026).** Los objetivos vigentes están en [OBJETIVOS.md](OBJETIVOS.md).

## Cotejo local del tribunal al preparar un acta — 7 de octubre de 2026

`vec-selectivos-preparar-acta -tribunal-salida` comprueba la salida exacta del
preparador de tribunal: identidad, versión, fase y SHA256 de los bytes recibidos.
Rechaza un material alterado o una huella distinta. El modo sin ese argumento
conserva su comportamiento.

Los mensajes en castellano e inglés distinguen el cotejo local de las
comprobaciones pendientes de procedencia, vigencia, designación, habilitación
y firma. Las pruebas focales y la revisión independiente de `9a3e0db13ec9`
pasaron. El siguiente corte S5 necesita fuente y autoridad institucionales;
esta preparación no constituye un acta firmada.

## Selector de idioma común — 7 de octubre de 2026

La entrada de Selección usa `montarSelectorIdioma`, de la interfaz común,
en lugar de construir sus opciones y navegación por separado. Conserva filtros
y ancla al cambiar de idioma. Cada opción declara su idioma y el selector muestra
el catálogo que se ha cargado, incluido el respaldo cuando falla una traducción.

Pruebas focales de Selección e idioma: 16 correctas. La revisión en Chrome con
una respuesta de ensayo sintética comprueba escritorio, móvil, teclado, cambio
de idioma y un catálogo inglés no disponible. No acredita acceso nominal a la
API ni modifica el cierre de S2–S8.

Sigue pendiente adaptar los visores que cambian idioma sin perder el archivo
abierto. Requieren un contrato común que permita repintar sin navegar. La carga
del idioma activo y el reintento de catálogos corresponden al lector común de V;
Personal adaptará después su carga para evitar esperas al importar el portal.

## Cierre del 2 de octubre de 2026: estado vigente

Este apartado prevalece sobre los estados y órdenes de arranque del 1 de
octubre conservados debajo. Las estimaciones siguen siendo las iniciales;
no se han recalculado como trabajo restante. A conserva Selectivos, Carrera,
Formación y RUM; B conserva Personal y G el baremador común. Las referencias
históricas a H describen el reparto anterior.

| Minitarea | Entrega comprobada | Qué queda |
| --- | --- | --- |
| S0 · ensayo | [#343](https://github.com/aavidad/VEC_Diputacion_app/pull/343) fusionada; fuente `23671aaed`. | El ensayo sintético no acredita admisión ni calificación oficial. |
| S1 · bases exactas V3 | [#390](https://github.com/aavidad/VEC_Diputacion_app/pull/390) fusionada; fuente `df04de956`. Lectura exacta en PostgreSQL, Chrome y recuperación tras reinicio comprobadas. | Montaje institucional; no dar por instalada esa capacidad en la principal. |
| S2 · preparación de bases | [#389](https://github.com/aavidad/VEC_Diputacion_app/pull/389) fusionada; fuente `0395be5cf`. CLI de preparación, con material pendiente de aprobación. El inventario durable recoge la variante `disponible_para_preparacion`; se conserva acta privada. | Escritor durable y gestión de claves/composición raíz; aprobación competente, firma y publicación. El inventario no acredita implementación. |
| S3 · solicitud recuperable | [#403](https://github.com/aavidad/VEC_Diputacion_app/pull/403) fusionada; fuente `f87269037`, diez comprobaciones CI verdes. Produce JSON y recupera exactamente los mismos 1089 bytes. | Registro y presentación, representación, firma y tasa conforme a las bases. |
| S4 · admisión y subsanación | Preparación por CLI: revisión de requisitos (#533), aportaciones (#539), visor (#538), borrador de lista provisional con motivos y plazo de subsanación de catálogo configurable (`--salida lista-provisional`, S4-L1) borrador de la definitiva desde la provisional y la resolución de cada exclusión (`--salida lista-definitiva`, S4-L3), visor de ambas listas (S4-L2/L4) y revisión de la provisional que incorpora solicitudes omitidas (`--salida revision-provisional`, S4-L5). | Hechos autorizados de RUM/Personal; aprobación por el perfil competente, identidad y orden por apellidos al publicar, vencimiento con Calendarios, registro de los escritos de subsanación, rectificación de decisiones ya tomadas, comprobar automáticamente que la definitiva parte de la última revisión de la provisional (necesita el registro de revisiones publicadas; hasta entonces la definitiva lo deja como pendiente para RRHH) y publicación. |
| S5 · tribunal y actas | Pendiente. | Órganos y perfiles confirmados; composición, habilitación y actas firmadas. |
| S6 · fases y calificaciones | Preparación local con `vec-seleccion-calificaciones`: revisión de un ejercicio ligada a bases, configuración, fase, fuentes y antecedente, con huella del material normalizado. Las notas ausentes permanecen pendientes. Dos revisiones independientes de `92a66d9e4e03` y pruebas focales correctas. | Cotejar bases, admisión y anonimato, fuentes de corrección y acta S5; autor competente, CAS y registro durable, cálculo común de méritos, reclamación y publicación aprobada. La huella local no acredita esos actos. |
| S7 · aprobados y traspaso | Pendiente. | Resultado aprobado y recibo idempotente de Bolsa/Personal, con rectificaciones. |
| S8 · OEP y plazas | Pendiente. | Referencias y actos de RPT/Personal; cuadro de ejecución. |

Para retomar, leer el FIN de cierre y comprobar los hashes/estados remotos;
no reconstruir S0–S3. Primero resolver el siguiente corte independiente de
S2 o S3 con sus propietarios. S4 espera los hechos autorizados de RUM y los
perfiles/actos que correspondan. Ninguna de estas fusiones acredita producción,
aprobación de bases, solicitud registrada o firma legal.

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

H es dueño del Registro Único de Méritos. A consulta sus hechos y evidencias por
un puerto autorizado; B aporta los datos de Personal, entre ellos relaciones,
ocupaciones y servicios reconocidos. El baremador sigue siendo común. Selección
conserva las bases y la decisión aplicada en cada convocatoria, sin convertir
la puntuación en un dato permanente del mérito.

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
| S1 · A y propietario de Bolsa | Nuevo puerto `seleccion/ports/convocatoria.go`, caso de uso de consulta, adaptador a Bolsa y ficha RRHH propia. Adaptar `bolsa/application/convocatorias_consulta_interna.go` y `bolsa/adapters/postgres/convocatorias_consulta.go`; SQL AD3 y Bolsa **solo en borrador** tras reservar números en `RESERVAS_MIGRACIONES.md` y fijar orden con D en `ORDEN_SQL_NUCLEO.md`. Obtener de RPT/Personal las referencias mínimas de OEP, plaza y versión necesarias para las bases. | Versión exacta, bases, requisitos, fases y huella visibles con lectura V3 nominal y auditoría en la transacción; 403 sin concesión, 503 si cae la dependencia, recuperación tras reiniciar. Ensayo en clon y dos revisiones sensibles antes de integrar. |
| S2 · A y Bolsa | Reutilizar `gobiernoconvocatorias` y contratos de Documentos/Firma; adaptadores y vista RRHH propios. Depende de S1 y actos/órganos de la pregunta 110. Separar preparación de la versión y aprobación/publicación en PR recorribles. | Preparar una versión, aprobar/publicar mediante autoridad competente y conservarla cuando se rectifique; recibos, firma y fuente oficial comprobados. Las referencias de plaza/OEP exigidas por las bases están resueltas antes de aprobarlas. |
| S3 · A, Bolsa y Registro/Documentos | Consumir la preparación de solicitud de Bolsa por puerto; casos de uso y vistas propias del aspirante. Depende de S2, identidad común, representación, firma, registro y reglas de tasa SEL-005. Separar solicitud recuperable de presentación registrada. | Persona interna o externa presenta una solicitud, recupera el mismo recibo y puede corregirla dentro del plazo de sus bases sin duplicar la inscripción. |
| S4 · A, H y RRHH | Casos de uso de requisitos/admisión, listas, subsanación e impugnación de la admisión; puerto autorizado al Registro Único de Méritos de H y, para requisitos independientes de méritos, a los hechos de Personal de B. Depende de S3 y perfiles fijos. | RRHH motiva cada admisión o exclusión, publica listas provisional/definitiva tras aprobación y conserva subsanaciones, impugnaciones e historia. |
| S5 · A, autoridad común y tribunal | Gobierno de miembros/abstenciones por proceso y fase, sesiones, actas; vistas propias. Preparación del tribunal desde S2; habilitación de actuaciones tras S4 y respuesta 110. Separar composición y actas en PR recorribles. | Miembros habilitados ven solo su fase; recusación/sustitución deja historia; acta firmada antes de publicar calificaciones. |
| S6 · A, H y baremador común | Casos de uso de ejercicios, notas por fase, méritos y reclamación/rectificación de calificaciones; consumir hechos de H y el motor común existente. Depende de S4/S5, bases exactas y evidencia admitida. | Mínimos, fases y empates se explican; ninguna nota pendiente se transforma en aprobado. Calificación, reclamación y publicación conservan versión, autor y acto. |
| S7 · A, Bolsa y Personal | Resultado aprobado, entrega idempotente por puertos a Bolsa o Personal y rectificación del resultado o entrega. Depende de S6, acto y datos de la pregunta 111. | Lista de aprobados y recibo recuperable del receptor con operación estable; el reintento no duplica efectos. La entrega a Personal no constituye nombramiento ni ocupación: B aplica el acto competente. Cada rectificación enlaza los actos y efectos previos. |
| S8 · A, RPT/Personal | Cuadro OEP–plaza–convocatoria SEL-001, tras las referencias mínimas de S1. Depende del modelo histórico RPT y autoridad de vacantes. | RRHH ve grado de ejecución con cada plaza y acto de cobertura, sin tratar una propuesta como vacante cubierta. |

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

## Estimación

Horquillas provisionales de trabajo de A, incluidos implementación, pruebas
focales y preparación de cada PR. Presuponen que los contratos comunes de
identidad, firma, registro, representación y cobro que se necesiten ya son
utilizables. No incluyen el trabajo de H en Méritos, B en Personal, D en sus
autoridades comunes, otros propietarios ni las decisiones y esperas externas.
S2, S3 y S5 se dividirán en PR pequeñas según se confirme cada contrato.

| Minitarea | Horas de A | Principal incertidumbre |
| --- | ---: | --- |
| S0 · ensayo recuperado | 10–16 | Reconciliar las tres ramas WIP y comprobar Chrome. |
| S1 · consulta exacta y V3 | 20–32 | Contrato de Bolsa, referencias RPT y ensayo SQL. |
| S2 · bases, aprobación y publicación | 24–40 | Actos competentes, firma y fuente oficial. |
| S3 · solicitud y presentación | 32–52 | Representación, registro y tasa según las bases. |
| S4 · admisión y subsanación | 28–44 | Evidencias de H y decisión motivada. |
| S5 · tribunal y actas | 24–40 | Autoridad por proceso y firma de actas. |
| S6 · fases y calificaciones | 28–48 | Reglas de bases y reclamaciones. |
| S7 · aprobados y traspaso | 24–40 | Recibo del receptor y rectificaciones. |
| S8 · cuadro OEP y plazas | 16–28 | Referencias y actos de cobertura de RPT/Personal. |
| **Total de esfuerzo de A** | **206–340** | No es la duración total de Selectivos. |

Con jornadas de ocho horas, un equipo necesita **26–43 días de esfuerzo** si
las dependencias están disponibles. Dos equipos no reducen esa cifra a la mitad:
S0–S7 tienen un camino casi secuencial. Con S8 en paralelo, la planificación
orientativa es **24–39 días**. La preparación de S5 desde S2 podría acortar ese
camino, pero hay que separar sus horas de las actas posteriores a S4 antes de
calcular otra horquilla. La cifra de dos equipos no es una fecha de entrega.

RRHH debe facilitar o validar las bases por modalidad, reglas de tasa y
exención, órganos competentes, composición del tribunal, actos y datos del
traspaso (preguntas 109–111). Sistemas debe confirmar los contratos y entornos
de identidad, representación, firma, registro, publicación oficial, cobro y
conciliación que correspondan. Sus esfuerzos y plazos quedan **por estimar por
sus responsables**; una espera por estas respuestas aumenta el calendario sin
consumir las horas de A de la tabla. H y B estimarán por separado sus puertos y
datos propios.

## Consenso Astra

En la primera ronda, Astra revisó la propiedad de los datos y el orden de los
cortes. Se acordó que H mantiene el Registro Único de Méritos, B produce los
hechos de Personal y A consume ambos por puertos autorizados según su finalidad.
El baremador existente es común; A conserva las bases y decisiones de cada
convocatoria. Acreditar un hecho, cumplir un requisito y puntuarlo siguen
siendo decisiones distintas. Un mérito reutilizable se consulta a H; los hechos
de Personal que se necesiten como requisitos independientes proceden de B.

En la segunda ronda se mantuvo S8 como cuadro final, pero las referencias
mínimas de plaza, OEP y versión se exigirán antes de aprobar las bases. S4
incluye impugnaciones de admisión; S6, reclamaciones de calificaciones; S7,
rectificaciones del resultado y de la entrega. S5 separa preparación del
tribunal y actas. Astra dio GO a este reparto y pidió tratar la horquilla de
dos equipos como escenario condicionado. La entrega a Bolsa o Personal tendrá
un recibo recuperable; Personal decidirá los actos de relación y ocupación
que le correspondan.
