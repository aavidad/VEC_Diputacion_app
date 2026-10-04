# Plan de Formación — 4 de octubre de 2026

VEC consultará Formación corporativa y derivará a su trámite vigente. El recorrido
completo comprende plan, solicitudes, ejecución y certificados; cualquier ampliación
para gestionar esos trámites exige una decisión interna expresa y un corte posterior.
La prioridad y las responsabilidades de Personal y de las autoridades comunes siguen
vigentes. Este encargo entrega documentación, sin código, SQL ni instalación.

Base inspeccionada: `origin/main@77a4e7470cac5e0a02adce40dbdedd1c7a15b6db`.
Requisitos: [ficha de Formación candidata](https://github.com/aavidad/VEC_Diputacion_app/blob/1f9b76833/docs/estudio_requisitos/ficha_formacion_2026-10-04.md),
en PR separada. Este plan desarrolla exclusivamente H02–H04 y H15–H17 del
[plan de Carrera, Formación y RUM](carrera_formacion.md); no sustituye su seguimiento,
reasigna Personal B ni vuelve a producir las piezas integradas.

## Fuentes y decisiones aplicables

La [página institucional](https://www.dipgra.es/servicios/empleo-y-formacion/formacion-para-el-empleo-publico/)
y el [catálogo de acciones](https://www.dipgra.es/servicios/empleo-y-formacion/formacion-para-el-empleo-publico/plan-agrupado-de-formacion/)
permiten estudiar el plan y abrir sus convocatorias. El
[Portal de Transparencia](https://www.dipgra.es/contenidos/normativa-recursos-humanos/)
publica el Reglamento provincial. Los artículos y obligaciones están citados en la
ficha, junto con TREBEP 14.g/54.8, ET 23 y la resolución de una acción de 2026.
No se generan criterios provinciales ni plazos desde ejemplos o políticas sintéticas.
La publicación del curso de consolidación de grado es una fuente para Carrera;
Formación no reconoce ni inscribe el grado.

Una página pública, una publicación oficial y una API admitida son capacidades
diferentes. Los enlaces públicos no acreditan lectura de solicitudes personales,
recepción de asistencia ni certificados. Se conserva la duda
[119](../../dudas.md) y la 138 registrada en la candidata documental de Provisión
`2cb8eb3e4` para concretar la interfaz técnica. Esa PR incorpora el único delta
compartido de dudas; esta rama no copia ni modifica sus preguntas.
Las dudas 39/120/121 conservan sus ámbitos; las reglas ya publicadas no requieren
otra pregunta. Antes de implantar se comprobarán las versiones y modificaciones
aplicables y la dirección oficial de cada convocatoria.

## Inventario en la base inspeccionada

Se consultó primero el índice local `codebase-memory-mcp`, proyecto
`home-alberto-Trabajo-VEC_Diputacion_app-.worktrees-codexe-original-autorizacion-20261003`.
Ese índice corresponde a otra instantánea: las rutas y el contenido del inventario se
cotejaron después en el SHA de este plan. No se atribuye actualidad al índice por sí solo.

| Pieza | Rutas actuales | Qué puede reutilizarse y qué falta |
| --- | --- | --- |
| Preparación FOR-001 | `internal/modules/formacion/domain/plan.go`; `application/preparar.go`; `adapters/jsonio/preparacion.go`; `cmd/vec-formacion-preparar/` | Comprueba fechas, cantidades y referencias; informa pendientes. El alcance aceptado es sintético. #312 integrada por `431619c8b`; no repetir preparador, CLI o ejemplos. Falta fuente corporativa admitida. |
| Visor de Formación | `web/static/portal-empleado/modulos/formacion/{cliente,modelo,vista,entrada}.js`; `index.html`; `web/static/textos/{es,en}/formacion*.json` | Filtro, detalle y descarga del JSON original. El cliente carga `escenario.json`, con credenciales omitidas. No es lector corporativo ni portal nominal montado. Los enlaces están en `internal/modules/formacion/adapters/jsonio/fuentes.json`. |
| Carrera | `internal/modules/carrera/{domain,application,ports}/`; `ports/antecedentes.go`; `ports/proceso_selectivo.go`; `web/static/portal-empleado/modulos/carrera/` | #313 integrada por `81a4e66ff`; H05/H06/H11 tienen piezas integradas. H07 #388 aparcada; H08 conservada fuera de esta base. Sus preparadores sintéticos no consumen cursos ni reconocen grado. No duplicar estos contratos ni el lector de Personal B. |
| RUM | `internal/modules/meritos/domain/hecho.go`; `application/{servicio,consulta_servicio}.go`; `ports/{registro,consulta,hechos}.go`; `adapters/{postgres,http}/`; `web/static/portal-empleado/modulos/meritos/consulta-montaje.js` | RUM01 y cambios RUM02–03 están en la base; RUM04 #435 integrada por `46de1e00d`. El servicio `Verificar` mantiene acreditación pendiente. `LectorHechosPreparacion` es sintético, no lector nominal RUM05. El montaje nominal RUM04 depende del proveedor/emisor K y la composición de auditoría L; no se declara cerrado por su integración. |
| Baremador común | `cmd/vec-baremador/`; `cmd/vec-baremador-web/`; `internal/modules/bolsa/application/simulacionbaremo/`; `internal/modules/seleccion/ports/baremador.go` | Ya simula méritos formativos según reglas exactas y bases. Reutilizar el motor y sus contratos; Formación entrega hechos, nunca puntos. La simulación no acredita valoración administrativa. |
| Documentos | `internal/vec/documentos/application/servicio.go`; `adapters/{httpinterno,postgres,autorizacion}/`; `internal/vec/ports/documentos.go`; `internal/vec/application/documentos.go` | Registro, custodia y lecturas comunes con autorización. Falta el contrato de certificado formativo y la competencia/fuente exactas. Tener adaptadores comunes no acredita emisión, firma legal o descarga de un certificado de Formación. |
| Autorización y auditoría | `internal/vec/ports/{auditoria_intento_nominal,auditoria_frontera_ruta_exacta}.go`; `internal/vec/adapters/administracionperfiles/postgres/`; `internal/vec/adapters/contextoactor/postgres/` | Autoridades comunes, con perfiles y acciones fijas. Formación aún no tiene su consumidor nominal. No trasladar un actor técnico de ADMIN ni publicar permisos al recibir una petición. |

En esta base no existe `internal/modules/formacion/ports/` ni un adaptador corporativo
de Formación. Tampoco se acredita su composición en la aplicación real por la existencia
de una página estática. Los SQL presentes de Méritos pertenecen a su entrega existente;
este plan no los reserva de nuevo, reaplica o declara instalados en la principal.

## Huecos y propietarios

| Capacidad | Resultado pendiente | Dueño y dependencia |
| --- | --- | --- |
| FOR-001 | Catálogo y detalle desde fuente admitida, con versión/procedencia y derivación vigente. | A consume Formación corporativa; H02–H04. Necesidades, presupuesto, aprobación y publicación continúan donde exista competencia. |
| FOR-002 | Consulta propia de solicitud, estado, justificante, decisiones e historia. | A consume la fuente corporativa y el contexto autorizado común; H15. Sin fuente, solo enlace de continuación. |
| FOR-003 | Ejecución, asistencia, evaluación, superación y certificado original, cada hecho por separado. | A integra la fuente; Formación conserva expedición y Documentos la custodia/lectura; H16. |
| Formación → RUM | Acreditación competente, entrega única y reconciliación con recibo. | A conserva RUM; H17. Requiere cerrar acreditación y RUM05/06 donde proceda, sin reconstruir RUM01–04. |
| Personal / ficha integral | Identificación y vínculo, proyección de cursos por puerto. | B conserva Personal. A produce el puerto mínimo de Formación; B decide su consumidor. |
| Carrera y baremador | Consumo autorizado de hechos según bases, fecha y finalidad. | Sus dueños conservan reconocimiento y valoración. Una asistencia o un diploma no crea puntos ni grado. |

Toda lectura personal debe usar el perfil activo fijo, recurso, ámbito, finalidad y campos
exactos del núcleo. La auditoría común registra identidad, perfil, acción, recurso opaco,
finalidad, instante, permitido/denegado/error, correlación y proceso/canal, también al
consultar o descargar. Es de solo adición; un efecto y su auditoría se confirman en la
misma transacción. La frontera técnica común resuelve los rechazos sin identidad cuando
corresponda, sin fabricar actor. Caducidad, revocación o fallo de dependencia no permiten
servir datos ni reutilizar una autorización pasada.

## Minitareas y salidas por PR

Son cortes futuros propuestos. Antes de asignarlos, dirección confirma responsable,
archivos exclusivos y SHA base actual. Los nombres de rutas nuevas son previsión,
no interfaces aprobadas. Los archivos del núcleo, montaje y manifiestos requieren
turno de su custodio; ningún corte autoriza tomarlos de otro equipo.

| Corte y base | Salida usable por PR | Archivos propios previstos | Dependencias | Horas |
| --- | --- | --- | --- | ---: |
| F00 · inventario | Revalidar SHA, PR integradas, contratos y FIN; señalar solo deuda restante. | Este plan, durante su turno documental. | Origin/main y seguimiento vigente; no repetir #312/#313/RUM. | 1–2 |
| F01 · H02 | Acuerdo de fuente/versiones, destino por acción, campos y funciones conservadas. Derivación informativa si solo hay URL. | Contrato documentado junto al adaptador; actualización de `fuentes.json` si procede. | RRHH/Formación/Sistemas; dudas 119/138; aprobación de lectura separada de API. | 2–4 |
| F02 · B1 | Consumidor nominal y auditoría común con una consulta/CLI focal que permite o deniega el recurso exacto. | Nuevos `formacion/ports/{autorizacion,auditoria}.go`; consumidor propio; composición por custodio. | F01 y ABI/configuración auténticos de K/L; perfiles fijos, ningún permiso por petición. | 6–10 |
| F03 · B2/B3 | Correspondencia curso/edición/persona/organización versionada, con conflicto o ausencia explicados en consulta. | Contrato mínimo en `formacion/ports/`; adaptador de correspondencia propio. | F01/F02 antes de consultar datos personales; B produce contexto/relación y M organización. El acuerdo de códigos/DTO puede avanzar sin datos. No escribir sus tablas. | 2–4 |
| F04 · H03/H04 | Puerto, adaptador admitido y consumidor CLI de catálogo/detalle; vacío/error identificables. | `formacion/ports/catalogo.go`, `adapters/corporativo/`; CLI propia que reutiliza preparación. | F01; lectura pública expresamente admitida o F02/F03 para datos restringidos. No inferir API. | 4–6 |
| F05 · H04 | Catálogo/detalle en vista VEC y continuación oficial con estados claros ES/EN. | Vista/cliente propios de Formación, catálogos ES/EN; montaje cedido por custodio. | F04; registro real, frontera y revisión independiente de usabilidad. | 2–6 |
| F06 · H15, solicitud | Consultar solicitud propia y justificante originales desde la fuente, con permiso y recibo de acceso. | `formacion/ports/solicitudes.go`; adaptador y caso de uso propios. | F01–F03; fuente admite lectura personal; SQL F14 si necesita estado durable. | 3–5 |
| F07 · H15, historia | Vista propia distingue decisiones, suplencia/renuncia y comunicación; recuperar sin otra inscripción. | Cliente/vista propios de Formación y catálogos; consulta de historia por puerto. | F05/F06; fuente versionada. Escrituras siguen derivadas al circuito corporativo. | 3–5 |
| F08 · H16, ejecución | Consulta autorizada de sesiones/asistencia/evaluación con evidencia y rectificaciones. | `formacion/ports/ejecucion.go`; consumidor de la fuente y vista propia. | F01–F03/F05; hechos confirmados por Formación. No inferir asistencia de acceso. | 4–7 |
| F09 · H16, certificado | Consultar certificado original, tipo, procedencia, expedición, firma/registro y estado pendiente. | `formacion/ports/certificados.go`; adaptador a fuente/Documentos y caso de uso. | F08; Documentos confirma contrato, competencia y custodia; sin emitir PDF sustituto. | 4–7 |
| F10 · B5 | Descargar el original custodiado con autorización y auditoría de esa descarga concreta. | Consumidor documental de Formación; cliente/botón/textos propios; cambios comunes por dueño. | F02/F09; lector documental autorizado y revisión sensible de frontera. | 4–6 |
| F11 · H17, acreditación | Recibir un hecho con certificado/evidencia cotejados y revisión competente; discrepancia queda pendiente. | Consumidor en `formacion/ports/meritos.go`; contrato RUM y caso de uso por A. | F03/F09/F10; circuito positivo de acreditación RUM, fuente y firma admitidas. | 4–7 |
| F12 · H17, reconciliación | Entrega Formación→RUM, recibo durable y reintento; duplicado/conflicto recuperan historia correcta. | Entrega/reconciliación y pruebas propias de Formación; receptor RUM existente ampliado por su dueño. | F11; RUM nominal y conformidad de fuentes; F14 si hay outbox/estado nuevo. | 4–7 |
| F13 · SQL futuro | Candidata en borrador de persistencia estrictamente necesaria, con reserva, preimagen y dependencias; ensayo reproducible preparado. | SQL nuevo de Formación reservado; adaptador propio; orden causal por custodio. | F01–F03 y contrato de efectos F11/F12; no duplicar catálogo corporativo. | 4–6 |
| F14 · ensayo/instalación | Ensayo en clon principal, dos revisiones exactas y kit; instalación por dirección, sin reaplicar historia. | Adaptador/pruebas/acta propios; kit documental de instalación. | F13; reserva y ensayo aprobados; fuente/configuración nominal. Un borrador no está instalado. | 4–8 |
| F15 · recorrido/entrega | Chrome→API→autorización→fuente/PG→recibo, negativos y reinicio; manual explica qué se sigue haciendo fuera. | Pruebas focales/recorrido y manual del módulo; seguimiento único por dirección. | F05–F12 y F14 cuando haya persistencia; revisión de usabilidad/sensible y fuente admitida. | 4–6 |

F04 aporta puerto y consumidor en el mismo corte; F05 hace alcanzable esa capacidad.
F06–F12 solo se abren con sus dependencias nominales. Si la fuente no admite una lectura,
se entrega la derivación comprobada y se conserva el hueco correspondiente; no se
crean rutas que solo devuelvan indisponibilidad ni un segundo preparador sintético.

F13/F14 son una provisión de esfuerzo para estado de integración, recibos u outbox
necesarios. Si la consulta no precisa SQL propio, se eliminan esas dos tareas y se
recalcula la horquilla. No almacenar el catálogo corporativo como una segunda autoridad.
Antes de escribir una migración se reserva un número nuevo en `RESERVAS_MIGRACIONES.md`
fuera de Git y su orden causal. SQL permanece en borrador hasta el ensayo en el clon
local de la principal y dos revisiones independientes; MCP `postgres-clon-local` para
lecturas/EXPLAIN. Dirección instala. Nunca DOWN sobre historia ni SQL en cidonia por A.

## Dependencias y paralelismo

Un equipo empieza por F00/F01. Después prepara F02 mientras acuerda los códigos y DTO de F03 sin consultar datos personales. La lectura de F03 espera F02; F13 se prepara sobre los contratos admitidos.
F04/F05 hacen utilizable el catálogo; F06/F07 y F08/F09/F10 pueden separarse por archivos
una vez disponible la lectura nominal. F11/F12 esperan la fuente documental y RUM.
La preparación SQL puede avanzar, pero F14 espera contratos finales y revisiones.
El recorrido final reúne únicamente capacidades instaladas y fuentes admitidas.

Con dos equipos, uno conserva catálogo/solicitudes y el otro ejecución/documentos/
entrega RUM. Ambos consumen B1/B2/B3/B5; un responsable mantiene cada contrato común.
No hay dos escritores sobre `vista.js`, catálogos o montaje: antes de solapar se separan
archivos y consumidores. La acreditación RUM, la auditoría, la instalación y la revisión
limitan el paralelismo. Personal B conserva su producción; no forma un tercer equipo
implícito ni se cuenta aquí su trabajo.

## Estimación y trabajo externo

Las 16 filas suman **55–96 horas técnicas**, con revisión y comprobaciones focales:
**7–12 jornadas de un equipo** de ocho horas, redondeadas al día completo.
Con dos equipos se estiman **5–9 jornadas**, condicionadas a contratos y fuentes ya
admitidos. La secuencia orientativa de fases es 3–6 h (F00/F01), 6–10 h (bases),
6–12 h (catálogo/montaje y ensayo cuando pueda solaparse), 12–20 h (solicitudes y
certificados en paralelo), 8–14 h (entrega RUM) y 4–6 h (recorrido): 39–68 h de
camino de trabajo, redondeadas a 5–9 jornadas. Si una dependencia impide ese solape,
se usa la horquilla de un equipo.

Las minitareas heredadas mantienen sus intervalos: H02 2–4 h, H03/H04 6–12 h,
H15 6–10 h, H16 8–14 h y H17 8–14 h. Aquí se añaden las bases de integración,
descarga, posible SQL y recorrido; no se suma otra vez el plan completo de Carrera/RUM.

| Trabajo externo, fuera del total técnico | Dedicación orientativa | Condición |
| --- | ---: | --- |
| RRHH / Formación | 4–8 h de trabajo efectivo | Confirmar fuente/funciones, versiones y responsables; resolver 119 y validar el recorrido. Su espera no tiene plazo comprometido. |
| Sistemas / plataforma corporativa | 6–12 h de trabajo efectivo | Interfaz o exportación admitida, identificadores, entorno de prueba, destino por acción y límites. Sin integración autorizada, solo derivación. |
| K/L y Documentos | Estimación por sus propietarios | Proveedor/emisor, perfiles, proceso/canal, auditoría, lectura/firma/custodia. No contar sus desarrollos dentro de F02/F09. |
| Personal B / Organización M | Estimación por sus propietarios | Vínculo y organización; consumidor de ficha integral por B si se encarga. |
| Revisión funcional y aprobación de ampliación | Sin fecha comprometida | Solo necesaria para la ampliación que se decida; consulta/derivación no concede gestión. |

Las horas son esfuerzo, no fechas de publicación ni promesa de que las dependencias
existan. Se revisarán al cerrar F01 y al fijar los contratos RUM/Documentos.

## Comprobación y entrega

Este documento pasa enlaces locales y `git diff --check`. No necesita campañas Go,
SQL o navegador de producto. Su PR es independiente de la ficha y no acredita una
capacidad nueva instalada, recorrible o aprobada.

Cada corte de implementación aplicará las skills backend/interfaz y pruebas que
correspondan, `usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec`,
`impeccable`/`VEC-PRIORIDAD.md` antes de pantallas y revisión independiente de
usabilidad. Textos con `humanizer`/`VEC-USO.md`; SQL con `revisar-sql-vec` y
`ensayar-sql`; revisión focal `security-audit` y Semgrep local sobre cambios sensibles.
Go usará gopls; búsqueda de código primero por índice. Revisar la candidata exacta
con `revisar-cambios-vec`, documentar con `documentar-entregar-vec` y entregar por
`pr-vec`; solo dirección integra y despliega.

La validación futura cubrirá autorización ajena/caducada/revocada, fuente caída,
consulta/descarga auditadas, i18n ES/EN, teclado/foco, PC y móvil con Chrome del
sistema. Efectos nuevos requieren PostgreSQL real, recibo y recuperación tras reinicio.
No repetir puertas verdes de piezas integradas; probar el delta y las dependencias
que hayan cambiado. Las dudas pendientes no detienen los contratos o documentos
independientes, pero tampoco autorizan a simular una fuente o una competencia real.
