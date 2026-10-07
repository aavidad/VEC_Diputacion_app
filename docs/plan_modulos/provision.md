# Provisión: inventario y plan de continuación

## Bases públicas transcritas — 7 de octubre de 2026

`vec-provision-bases` consulta una transcripción versionada del concurso
2026/PPT_01/000026 y de la libre designación 2025/PPT_01/000474. Puede comprobar
la huella de los PDF locales y señalar diferencias de expediente, CVE, corte y
máximos del baremo frente a un borrador. La libre designación conserva su
circuito de idoneidad y carece de ranking automático.

El contraste es parcial: no coteja los códigos de puesto del borrador con los
nueve códigos publicados, porque sus referencias opacas aún no tienen una
correspondencia acreditada. Declara ese pendiente aunque no encuentre otras
diferencias. No reconoce vacantes ni aprueba bases. La revisión independiente
de `498f64306319` y las pruebas focales pasaron; PV02 requiere completar las
fórmulas, las excepciones y el vínculo de la oferta con la fuente RPT.

Fecha: 4 de octubre de 2026. Base inspeccionada:
`origin/main@77a4e7470cac5e0a02adce40dbdedd1c7a15b6db`.

RRHH puede ensayar preferencias, valoración, adjudicación y revisión en el
servidor local del Baremador. Falta convertir esas operaciones en un expediente
institucional registrado, autorizado y recuperable. El plan cubre concursos de
provisión o traslados, libre designación y comisiones de servicios.

La [ficha de requisitos P1–P16](https://github.com/aavidad/VEC_Diputacion_app/blob/2cb8eb3e4/docs/estudio_requisitos/ficha_provision_2026-10-04.md)
se entrega en una rama documental independiente. Este plan parte de main y no
apila esa rama. El inventario no acredita instalación ni aceptación de RRHH;
no modifica los contadores del seguimiento único.

## Qué existe en la base exacta

El módulo tiene 43 archivos versionados: 19 en dominio, 7 en aplicación,
5 en puertos y 12 en el adaptador de simulación. La hoja web tiene 11 archivos.
El índice local de concursos del 01/10 sirvió para localizar piezas; los paths
y el contenido se contrastaron en la base anterior, sin atribuir al índice
antiguo el estado de main.

| Capacidad | Archivos actuales | Estado y hueco concreto |
| --- | --- | --- |
| Valoración de seis familias | `internal/modules/provision/domain/concursos_contrato.go`, `concursos_calculo.go`, `concursos_meritos.go`, `concursos_periodos.go`, `concursos_cursos_titulos.go`, `concursos_validacion.go`; `application/concursos_simular.go`; `ports/concursos_simulador.go`. | Grado, trabajo desempeñado, antigüedad, permanencia, cursos y títulos. Reglas tipadas y desglose; faltan reglas contrastadas de un proceso institucional y hechos autorizados. |
| Aritmética y comparación | `internal/shared/baremacion/{puntos,racional,fecha_civil,intervalo_civil,jornada,redondeo,tope}.go`; `domain/concursos_diferencia.go`; `cmd/vec-baremador/{main,concursos_diferencia}.go`. | Reutilizable por Bolsa y Provisión. No crear otro motor ni convertir una comparación de ejemplo en aprobación de bases. |
| Proceso y solicitud multipuesto | `domain/proceso_{contrato,validacion,copia}.go`; `application/proceso_simular.go`; `ports/proceso.go`; `adapters/simulacion/proceso_json.go`; `cmd/vec-simular-provision/`. | `PrepararProceso` y `SimularProceso` conservan oferta, referencias, versiones, requisitos y preferencias. No guardan ni presentan la solicitud. |
| Adjudicación global | `domain/adjudicacion_{contrato,simular,validacion}.go`; `application/adjudicacion_simular.go`; `ports/adjudicacion_simulador.go`; `adapters/simulacion/adjudicacion_json.go`; `cmd/vec-simular-adjudicacion/`. | Método experimental de aceptación diferida, universo sintético, un puesto por persona y empate pendiente. Faltan conformidad con bases, resultas, sorteo acreditado y efectos jurídicos. |
| Reclamación y revisión | `domain/ciclo_{contrato,validacion,copias}.go`; `application/ciclo_ensayar.go`; `ports/ciclo_ensayo.go`; `adapters/simulacion/ciclo_json.go`; `cmd/vec-ensayar-provision/`. | Provisional de un puesto, mantener/rectificar y nueva instantánea. No enlaza una solicitud registrada con el universo adjudicado. Resolución siempre borrador. |
| Fuentes y guardado futuro | `ports/proceso.go`, `ports/ciclo_persistencia.go`; `deploy/postgresql/provision/README.md`. | Personal/RPT, instantáneas Personal/RUM/RPT, repositorios, firma y publicación son puertos pendientes. No hay adaptador PostgreSQL ni SQL ejecutable del módulo. |
| Servidor de ensayos | `cmd/vec-baremador-web/{provision,provision_adjudicacion_http,provision_ciclo_http}.go`, `provision_ensayos_catalogo.go`, `provision_ensayos.json`, `provision_presentacion.json`, `README.md`. | Loopback y hechos sintéticos fijados en el servidor; no autenticación institucional, V3 nominal, registro o firma. |
| Hoja y clientes | `web/static/portal-empleado/modulos/provision/{entrada,montaje,modelo,vista,cliente-local,ensayos-cliente,ensayos-modelo,ensayos-vista}.js`, `index.html`, `provision.test.mjs`, `ensayos.test.mjs`. | Preparación, valoración, adjudicación y revisión reutilizables; clientes exclusivos del ensayo. Acciones oficiales deshabilitadas. |
| Textos y recorrido | `web/static/textos/{es,en}/provision.json`; `scripts/probar_provision_navegador.mjs`, `scripts/recorridos/{README_provision.md,provision_expectativas.json}`. | Guion Chrome del sistema disponible para el servidor local. No se ejecutó en esta edición documental ni acredita persistencia institucional. |

`cmd/vec-provision-plantillas` aprovisiona plantillas de Contratación temporal.
Los archivos de bootstrap que contienen «provision» aprovisionan otras
capacidades; sus nombres no prueban un montaje del módulo Provisión.
No existe `internal/modules/provision/manifest.go` en esta base. El constructor
de hoja local tampoco acredita registro del módulo en el servidor institucional.

El [recorrido del 01/10](../estudio_requisitos/provision_recorrido_y_persistencia_2026-10-01.md)
conserva una base anterior y menciona candidatas de aquel corte. La presencia
actual de proceso, adjudicación y ciclo se acredita por los archivos de main,
sin reutilizar esos estados históricos de PR ni atribuirles instalación.

## Autoridades y contratos que se consumen

| Responsable | Dato o capacidad única | Provisión recibe |
| --- | --- | --- |
| Personal B | Relación de servicio, situaciones, servicios, grado reconocido y actos de ocupación y reserva, ligados a la Persona común. | Referencias, versión, vigencia, fuente y campos mínimos autorizados. El nivel del puesto no acredita grado; el certificado no acredita relación de empleo. |
| Organización/RPT M | Puestos, requisitos, estructura y proyección histórica de cobertura/reserva a partir de los actos de Personal. | Oferta/versiones y actos por puertos. No reconstruir RPT ni declarar vacante por ausencia de un dato. |
| RUM A | Méritos, evidencias, acreditación y discrepancias. | Instantánea autorizada al corte, conservando estados y procedencia. La puntuación de Provisión no se escribe como mérito universal. |
| Baremador común (A en el reparto vigente) | Aritmética y familias de reglas compartidas. | Servicio común y versión de motor; cualquier familia nueva se acuerda con su responsable. Las reglas del procedimiento permanecen en Provisión. El reparto histórico con G no reasigna su mantenimiento actual. |
| Núcleo y administración comunes (identidad K y auditoría L) | Contexto nominal, perfiles fijos, concesiones, auditoría y composición. | Permiso por acción/recurso/ámbito/finalidad/campos, vigente y revocable. Sin PDP, login ni auditoría paralelos. |
| Documentos, firma, registro y publicación | Custodia, documento firmado, asiento y evidencia de publicación o entrega. | Referencias y recibos verificables por puertos propios. Borrador, huella y autenticación no acreditan firma ni publicación. |

Reutilizar `FuentePersonalProceso`, `FuenteRPTProceso`, `FuenteInstantaneasCiclo`,
`AutorizadorProceso`, `RepositorioBorradorProceso` y `RepositorioCicloProvision`;
ajustar sus DTO con el consumidor real si faltan datos. No leer tablas ajenas ni
copiar fuentes completas. Los puertos de firma y publicación existentes siguen
pendientes; se conectarán a la autoridad documental admitida.

La fuente ausente o discrepante conserva el estado pendiente. La consulta de
un recurso, su descarga y su modificación requieren permisos distintos. La
auditoría común registra éxito, denegación y error, incluidos consulta, cálculo,
recuperación y descarga, con identidad nominal y perfil activo, acción, recurso opaco, finalidad,
instante, resultado permitido/denegado/error, proceso, canal y correlación.
La traza es de sólo adición y se minimizan los datos registrados.
Cada escritura consume autorización en la transacción del efecto, versión,
idempotencia, historia y recibo; el outbox corresponde sólo al efecto que lo
necesite. Un borrador simple no exige una cola de comunicaciones.

## Orden de las minitareas y salida de cada PR

Las horas incluyen implementación, revisión, pruebas focales y corrección de CI.
Cada fila termina en una función que pueda usar RRHH o la persona participante,
con guía breve. Si precisa varios commits, se mantiene una PR por resultado;
no se entrega un adaptador sin consumidor. Los efectos institucionales esperan
sus dependencias; los ensayos de conformidad pueden avanzar con datos sintéticos.

| Orden | Minitarea y requisitos | Dependencia previa | Salida visible o usable de la PR | Horas de un equipo |
| --- | --- | --- | --- | ---: |
| PV01 | Elegir y transcribir la versión pública del primer concurso, oferta y rectificaciones (P1–P3). | RRHH elige proceso; las fuentes públicas ya permiten preparación. | RRHH abre la ficha de bases exactas y ve qué reglas están transcritas o pendientes, sin botón de aprobación ficticio. | 6–10 |
| PV02 | Contrastar fechas, seis familias y desempates en el ensayo existente (P4, P6, P8). | PV01; responsable del baremador común valida extensiones del motor. | Comparador reproducible con casos sintéticos de las bases; explica cualquier diferencia y conserva la configuración revisada. | 8–16 |
| PV03 | Consumir perfiles fijos y auditoría común en el recorrido de lectura (P14). | K/L aportan acciones nominales y auditoría; contexto/relación coherentes. | Persona/RRHH acceden sólo a su ámbito; se registra lectura permitida, denegada y fallida. Soporte tiene correlación sin datos privados. | 12–20 |
| PV04 | Conectar instantáneas autorizadas de Personal/RPT/RUM (P3, P4, P13). | PV03; puertos nominales de B, M y A disponibles. | Detalle de fuentes al corte, con versión y motivos pendientes; sin fuentes inventadas ni uso del permiso de otro módulo. | 12–20 |
| PV05 | Montar catálogo y detalle institucional exactos (P1–P4). | PV01, PV03–PV04; fuente de convocatoria publicada. | Lista paginada y ficha con requisitos, puesto, bases, fechas y discrepancias; denegación/revocación y dependencia caída visibles. | 16–24 |
| PV06 | Preparar y revisar convocatoria versionada (P2, P3, P16). | PV01–PV04; repositorio y acciones propias del núcleo común. | RRHH guarda y recupera una preparación; otro perfil revisa. Versiones previas consultables; aprobación/publicación separadas. | 20–32 |
| PV07 | Solicitud multipuesto registrada y recuperable (P5, P16). | PV05–PV06; registro y firma de solicitud si las bases la exigen. | Persona ordena preferencias, confirma, obtiene recibo y lo recupera tras reinicio; repetición exacta sin duplicado. | 24–40 |
| PV08 | Admisión, causas y subsanación (P4, P7). | PV07; perfiles de tramitación y decisión confirmados. | RRHH consulta pendientes y emite actuación motivada; participante consulta y aporta lo requerido por el canal autorizado. | 16–28 |
| PV09 | Valoración institucional con desglose (P6). | PV02, PV03 y PV08; fuentes acreditadas. | Comisión/RRHH consulta puntos por puesto y procedencia; provisional versionado con recibo. Un dato ausente no obtiene total oficial. | 16–28 |
| PV10 | Presentación de alegación al provisional exacto (P9). | PV09; evento de publicación y plazo gobernados. | Persona selecciona la valoración publicada, presenta evidencia y recupera su recibo; rechazo causal de referencias cruzadas. | 20–32 |
| PV11 | Decisión motivada y recálculo conservando historia (P9). | PV10; perfil competente y segunda validación cuando proceda. | Comisión mantiene o rectifica; se ven ambas versiones y el cambio explicado. La fuente corregida vuelve a su autoridad, sin escritura cruzada. | 16–28 |
| PV12 | Adjudicación global conforme a las bases (P8). | PV02, PV08–PV11; universo cerrado y política validada. | RRHH obtiene propuesta global por preferencias con desglose, puestos desiertos y empates pendientes; sorteo sólo con evidencia del acto. | 20–36 |
| PV13 | Resultas, opciones y cambios del universo (P8–P10). | PV12; reglas concretas de bases y actos disponibles. | Propuesta nueva tras la actuación admitida, con explicación causal; la anterior permanece. Ninguna renuncia sintética modifica una adjudicación eficaz. | 16–28 |
| PV14 | Acta y borrador de resolución en Documentos (P10, P15). | PV11–PV13; plantilla admitida y custodia. | Órgano competente consulta el borrador y antecedentes; descarga autorizada, auditada y rotulada como borrador. | 16–28 |
| PV15 | Firma, publicación y comunicaciones con evidencia (P10, P15). | PV14; conectores admitidos y perfiles de firma/publicación. | Resolución enlazada con firma validada y recibo de publicación; reintento recupera la operación. Un envío sin confirmación mantiene incertidumbre. | 16–28 |
| PV16 | Trasladar resolución a Personal y RPT (P10, P13). | PV15; B/M ofrecen operaciones nominales idempotentes. | RRHH ve recepción y estado de cese/toma de posesión; se recupera cada recibo y se concilian fallos sin repetir altas u ocupaciones. | 16–28 |
| PV17 | Circuito propio de libre designación (P11). | PV03–PV07; bases de LD, perfiles y circuitos documentales disponibles. | Solicitud, expediente de idoneidad y propuesta motivada en pantalla; resolución sólo por su autoridad, sin sustituirla por ranking automático. | 24–40 |
| PV18 | Solicitud y resolución de comisión de servicios (P12). | PV03–PV07; informe de origen y perfil de propuesta del destino; firma/publicación disponibles para resolver. | Origen y destino tramitan sus pasos; participante consulta informe, propuesta y resolución; reserva por Personal con recibo y reflejo en la proyección de RPT. | 24–40 |
| PV19 | Prórroga, vencimiento y cese de comisión (P12, P13). | PV18; actos/reglas y puertos de Personal/RPT. | RRHH ve fechas y pendientes, registra la actuación autorizada y recupera su recibo; el reloj no produce un cese jurídico inventado. | 16–28 |
| PV20 | Activación controlada y aceptación del conjunto (P1–P16). | Cortes anteriores integrados, dependencias instaladas por su responsable. | Guía y recorrido humano de las tres modalidades con negativos y reinicio; acta de lo aceptado y límites reales. | 10–18 |

PV17 y PV18 reutilizan el registro, fuentes y recuperación del recorrido común.
Sus propuestas y resoluciones conservan cada modalidad; si firma/publicación
no están disponibles, la salida se limita a propuesta preparada y la PR no
cierra la resolución. PV20 incluye verificación técnica; la aceptación formal
corresponde a RRHH y el despliegue a Dirección/Sistemas.

## Archivos y reparto sin solapamiento

Provisión conserva `internal/modules/provision/`, sus CLI de ensayo, la hoja
`web/static/portal-empleado/modulos/provision/` y sus catálogos de texto. El
Baremador común (A en el reparto vigente) conserva `internal/shared/baremacion/`, `cmd/vec-baremador/` y los
archivos compartidos de `cmd/vec-baremador-web/`. Se acuerda con el responsable del baremador un único
escritor de cada handler de Provisión y del README compartido; no se modifica
el editor de Bolsa para abrir otro procedimiento.

B conserva los archivos de Personal; M los de Organización/RPT; A los de RUM.
Provisión implementa consumidores por puerto. El director principal mantiene
registro del módulo, raíces de composición, autoridad de rutas, shell, menú e
importadores/caché comunes. Se solicita su montaje concreto cuando la función
esté lista; no se añaden nuevas vías `*_desarrollo.go`.

Antes de programar cada fila, reservar en el canal sus archivos exactos,
contratos compartidos y escritor de los padres. Con dos equipos, el primero
mantiene proceso/solicitud/admisión y el segundo valoración/revisión y luego
las modalidades. Los archivos de contrato compartidos se ceden por turno.
Los nombres nuevos de archivos se fijarán al abrir cada corte después de
buscar consumidores; este plan no autoriza una segunda implementación.

## Persistencia, permisos y prueba de cada corte

El contrato de [persistencia pendiente](../../deploy/postgresql/provision/README.md)
menciona `provision000001` ya reservada. El custodio SQL debe comprobar titular, alcance,
preimagen y cola H6/núcleo vigente antes de retomarla. Cualquier número nuevo
se reserva fuera de Git en `RESERVAS_MIGRACIONES.md` antes de crear SQL; no se
reutiliza una reserva ajena. En este encargo no se crea ni ejecuta SQL.

Cada futuro corte durable lleva el consumidor y su recorrido real. Revisar la
candidata SQL exacta por dos revisores independientes y ensayarla en el clon
local de la principal; usar MCP `postgres-clon-local` sólo para consultas,
EXPLAIN e índices. La instalación corresponde a Dirección, separada del ensayo.
No reaplicar UP/DOWN de migraciones instaladas ni revertir historia.

La definición de salida exige positivo, denegación, error de dependencia,
entrada inválida, versión obsoleta y, cuando escriba, idempotencia y recuperación
tras reiniciar aplicación/PostgreSQL. Para adjudicación probar preferencias,
empates, sorteo, exclusiones, resultas y rectificación del universo contra las
bases elegidas; el orden de transporte o el nombre no desempatan.

Aplicar `programar-backend-vec` y `persistir-autorizar-vec` en implementación;
`programar-interfaz-vec`, `usabilidad-vec`, `aspecto-vec`,
`disenar-sistema-visual-vec` e `impeccable/VEC-PRIORIDAD.md` antes de pantallas;
`probar-recorridos-vec` y Playwright con Chrome del sistema para el recorrido.
Textos en catálogos español/inglés y revisión `humanizer/VEC-USO.md`.
Revisión de usabilidad independiente antes de integrar; escritorio con paneles
y scroll interno, móvil usable, teclado/foco y estados vacío/carga/error.

Para SQL usar `revisar-sql-vec`/`ensayar-sql`; para cambios sensibles aplicar
`security-audit` focal y Semgrep local sin subir código. Identidad, permisos,
SQL y fronteras de datos personales requieren dos revisiones independientes
sobre el hash final. `revisar-cambios-vec`, `documentar-entregar-vec` y `pr-vec`
cierran cada entrega; sólo el director principal integra. La revisión de este
plan documental no sustituye ninguna de esas comprobaciones de producto.

## Estimación y supuestos

| Alcance | Horas de un equipo |
| --- | ---: |
| PV01–PV05: bases, fuentes, permisos y consulta | 54–90 |
| PV06–PV11: preparación, solicitud, admisión, valoración y revisión | 112–188 |
| PV12–PV16: adjudicación, documentos y enlace con Personal/RPT | 84–148 |
| PV17–PV19: libre designación y comisión completa | 64–108 |
| PV20: activación y recorrido conjunto | 10–18 |
| Total técnico restante estimado | 324–552 |

A ocho horas por jornada, **41–69 días de un equipo**. Con dos equipos y
archivos disjuntos, prever **29–53 días**: tras el recorrido común pueden
solaparse revisión/adjudicación y los circuitos propios de libre designación
y comisión. La cadena PV03 → PV04 → PV06–PV16 → PV20 suma 230–394 horas,
unas 29–50 jornadas; el techo de 53 reserva margen para coordinar ambos equipos.
Las fuentes, autorización, registro y publicación siguen en el camino crítico;
duplicar equipos no reduce esas esperas a la mitad.

La horquilla incluye consumidores de Provisión, pruebas, revisión y ajustes de
las piezas existentes. Supone disponibles las autoridades y contratos comunes.
El desarrollo de RPT/Personal/RUM, un conector nuevo de firma/registro o cambios
sustanciales del motor se estiman en su módulo propietario y obligan a revisar
la fecha del recorrido dependiente; no están incluidos dos veces aquí.

| Aportación externa | Supuesto de planificación | Efecto si falta |
| --- | --- | --- |
| RRHH/Secretaría | 3–7 jornadas de dedicación repartidas: elegir proceso y versión, validar transcripción, perfiles y actos, circuito y prueba manual. Las bases públicas ya se estudian sin esa espera. | Se continúa conformidad y preparación sintética; se pospone el efecto concreto sin inventar decisión, órgano o delegación. |
| Sistemas/K/L y responsables de las fuentes | 4–8 jornadas repartidas si ya existen entornos y conectores: acceso nominal, fuentes autorizadas, custodia, ensayo/instalación y activación. | No se monta la capacidad dependiente; fallo cerrado. Un conector o modelo de fuente nuevo requiere otra estimación. |

Estas dedicaciones pueden solaparse con desarrollo. No se suman automáticamente
al plazo técnico y no cuantifican la espera para obtener una respuesta o acceso.
No hay fecha comprometida ni aprobación del plan por RRHH.

## Dudas internas y primera continuación

Reutilizar [dudas.md](../../dudas.md), preguntas 27–29, 36, 60–61, 122 y 128
para Personal, auditoría, conservación y perfiles/delegaciones. Dirección
ha registrado la pregunta 137 sobre primer proceso, responsables,
fuentes internas y circuito de registro/firma/publicación. No volver a pedir
coeficientes, requisitos o plazos que ya figuran en las bases públicas.

Primero PV01 y PV02: elegir el antecedente público, conservar sus versiones y
contrastar el motor actual, en especial el corte exclusivo y los desempates.
En paralelo se comprueban contratos y disponibilidades de PV03–PV04 con B,
M, A y K/L. No comenzar otra persistencia antes de conocer esas dependencias.

Fuentes de continuidad:
[estudio del Baremador](../estudio_requisitos/integracion_baremador_concursos_provision.md),
[recorrido/persistencia](../estudio_requisitos/provision_recorrido_y_persistencia_2026-10-01.md),
[contrato de módulos](../portal_vec/contrato_modulos_vec.md),
[plan Personal](personal.md), [plan Organización/RPT](organizacion_rpt.md) y
[plan Carrera, Formación y RUM](carrera_formacion.md).
No se retiran las ramas de ensayo ni se cambian sus recibos por este documento.

## Baremo del concurso 2026: cálculo configurable — 8 de octubre de 2026

El CLI existente `vec-baremador --modo concursos` incorpora una configuración
versionada del concurso 2026/PPT_01/000026 y una entrada de ejemplo. El comando
y las huellas de ambos archivos están en `cmd/vec-provision-bases/README.md`.

La experiencia agrupa meses reconocidos por nivel antes de aplicar las
fracciones y usa su ventana propia de diez años. Esa ventana no recorta la
antigüedad ni la permanencia. La permanencia provisional aplica el corrector
configurado y lo informa separado de la jornada. El comparador de versiones
incluye cobertura, ventanas y correctores. No convierte días en meses cuando
la entrada no permite acreditar esa unidad.

El ejemplo calcula experiencia, grado, antigüedad, permanencia definitiva y
formación. Titulaciones queda pendiente por la diferencia entre las bases
6E.4 y 6E.6.d sobre másteres. La permanencia que mezcla periodos definitivos y
provisionales queda pendiente de confirmar el reparto del redondeo. Mientras
falte cualquiera de esas partidas, el resultado conserva el desglose y
`total: null`; no presenta una suma parcial como puntuación completa.

La pregunta 150 de `dudas.md` recoge ambas decisiones para RRHH. Los valores
están en el catálogo de ejemplo y pueden cambiar sin recompilar el motor.
Este corte no admite solicitudes, adjudica puestos ni sustituye la revisión
del órgano competente; siguen pendientes fuentes nominales, registro y el
circuito institucional de concurso y libre designación.

Medición del CLI compilado: 200 ejecuciones de la misma entrada, p95 de
2,79 ms y media de 2,58 ms, con resultado idéntico. Incluye arranque del
proceso, lectura y comprobación SHA de los dos archivos, cálculo y salida
JSON. No mide HTTP, PostgreSQL ni una carga institucional de empleados.
