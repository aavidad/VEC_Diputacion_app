# Carrera y Formación: continuación de Codex-H

## Cierre del 2 de octubre de 2026: estado vigente

Este apartado prevalece sobre los estados y la próxima acción del plan inicial
conservado debajo. Las horquillas son estimaciones originales, sin recálculo del
trabajo restante. A conserva Carrera, Formación y RUM; B conserva Personal y
G el baremador común. Las menciones a H corresponden al reparto histórico.

| Minitarea | Estado comprobado | Siguiente dependencia |
| --- | --- | --- |
| H01 · verificar entregas | Comprobado en este cierre. | Revalidar PR y FIN antes de retomar. |
| H02 · fuente de Formación | Preguntas 119–121 pendientes de RRHH sobre Formación, convenio y vías de grado. | Fuente, competencias y versiones admitidas. |
| H03 · puerto de catálogo | Pendiente. | Contrato confirmado en H02. |
| H04 · consulta de Formación | Pendiente. | Fuente/adaptador y consumidor de H03; mantener consulta y derivación. |
| H05 · antecedentes de Personal | [#341](https://github.com/aavidad/VEC_Diputacion_app/pull/341) fusionada, `08231a909`: puerto y ensayo sintético. La entrega de B [#376](https://github.com/aavidad/VEC_Diputacion_app/pull/376) aporta contrato. | Lector nominal real de B; el contrato no acredita una lectura montada. |
| H06 · política de grado | [#366](https://github.com/aavidad/VEC_Diputacion_app/pull/366) fusionada, `1b3d5dcb9`: política sintética. | Fuente provincial y aprobación; los valores de ensayo no conceden derechos. |
| H07 · preparación de expediente | [#388](https://github.com/aavidad/VEC_Diputacion_app/pull/388) abierta en borrador, `fcfbce9cf`; calidad CI en curso al comprobar. | Cerrar CI/revisión; H08 y lector B antes de datos personales. La corrección común #402 ya está fusionada y no cambia esta fuente. |
| H08 · autorización nominal | WIP conservado en `trabajo/codexa-carrera-h08-n2@39672b2ae64e820249f7317e0c8702eac15cbd86`, sin PR; dos revisiones favorables del corte preparado. | Lector B, montaje, HTTP y fuente real; esas revisiones no acreditan consumo montado. |
| H09 · persistencia | Pendiente. | H05–H08, reserva SQL, ensayo y revisiones antes de instalación por dirección. |
| H10 · resolución e inscripción | Pendiente. | Persistencia, autorización montada, documentos/firma y recibo de Personal. |
| H11 · cotejo de promoción | [#393](https://github.com/aavidad/VEC_Diputacion_app/pull/393) fusionada, `f48d0f3dc`: dictamen sintético. | Bases y hechos reales autorizados; el cotejo no decide admisión. |
| H12 · seguimiento de promoción | Inventario, sin código nuevo. | Estado autorizado de A y acto de Personal; H08 y H11. |
| H13 · política laboral | Pendiente. | Convenio consolidado y aprobación de RRHH. |
| H14 · expediente laboral | Pendiente. | H05/H08/H09 y política laboral aprobada. |
| H15 · solicitudes de Formación | Pendiente. | Fuente, competencias y permisos de H02–H04. |
| H16 · ejecución y certificado | Pendiente. | Hechos de Formación y custodia/firma de Documentos. |
| H17 · Formación → RUM | Pendiente. | H16 y RUM01–06; entrega única con recibo reconciliable. |

El nivel del puesto, el grado reconocido y la progresión laboral se mantienen
separados. Las consultas de ensayo y políticas sintéticas siguen pendientes
de fuente y aprobación para uso real.

| Minitarea RUM | Estado comprobado | Qué queda |
| --- | --- | --- |
| RUM01 · hechos y procedencia | [#339](https://github.com/aavidad/VEC_Diputacion_app/pull/339) fusionada, `c4c1856fd`: modelo y CLI. | Su consumo no convierte una declaración en acreditación. |
| RUM02 · cambios e historia | [#400](https://github.com/aavidad/VEC_Diputacion_app/pull/400) abierta, `770c288cf`, calidad CI en curso tras incorporar `03af`. Escritura, archivo y repetición probados en el clon con siete operaciones reales y reinicio. | Cierre de CI e integración; canal externo y verificación competente siguen cerrados. |
| RUM03 · persistencia y V3 | Misma #400: lector V3, convivencia con S1 y dos revisiones favorables del corte; comprobaciones locales verdes. AD3-142 ensayada e instalada solo en el clon. | Integración e instalación controlada en destino por dirección; no está instalada en la principal. |
| RUM04 · consulta propia | WIP publicado en `trabajo/codexa-rum04-p1-n1@f25aac133ea44e54aa1edb88e883bcb09f634ccf`. AD3-145/Méritos2 reservadas; lectura propia 200 y ausencia ajena comprobadas. Tras reinicio en otro proceso, misma huella y negocio conservado; recibos de acceso 4→6. | Dos revisiones, Chrome, adaptación tras AD3-143/144 y cierre del montaje. No hay cierre funcional acreditado. |
| RUM05 · lectura por otros procesos | Pendiente. | Lectura positiva RUM03 y contratos mínimos por finalidad de Selectivos/baremador. |
| RUM06 · conformidad de fuentes | Pendiente. | Correspondencia de Persona, hechos y evidencias; conflictos/reintentos y circuito Formación. |

AD3-145 se prepara con guardas POST-142; no se presenta como POST-144.
El orden reservado es 143 K → 144 G → 145 A → 146 E → 147 E → 148 E;
esta secuencia no afirma instalación. No reaplicar AD3-142 en el clon ni usar
un SQL reservado como capacidad disponible en la principal.

Para retomar, leer los FIN y verificar #388/#400 antes de continuar sus cortes.
Después cerrar la candidata RUM04 sobre su fuente final, sin reconstruir RUM01
ni duplicar Personal. Las preguntas 119–121 y las integraciones pendientes
siguen abiertas; este cierre no acredita producción ni reconocimiento de grado.

Alcance: CAR-001–003, FOR-001–003 y Registro Único de Méritos (RUM).
**CAR-004 Desempeño queda fuera de este encargo y de su estimación.**

Estado comprobado el 1 de octubre de 2026 sobre `origin/main@0a62a3ea6`.
La rama `main` de la raíz local es histórica; no utilizarla como base de producto.
Claude revisa, integra y despliega. Este plan no autoriza instalación ni uso de datos reales.

## Lo que existe

| Pieza en main | Ruta | Estado y límite |
| --- | --- | --- |
| Relaciones y servicios propios | `internal/modules/personal/domain/ficha_propia.go`; `web/static/portal-empleado/modulos/personal/cliente-http-ficha-propia.js` | Consulta compuesta de Personal. No aporta todavía grado ni subgrupo estructurado para Carrera. |
| Antecedentes y actos de Personal | `internal/modules/personal/domain/registro_empleado_b2_actos.go` | Registra relación, ocupación, servicio y situación. No registrar un grado usando otro tipo de acto. |
| Valoración del grado reconocido | `internal/modules/provision/domain/concursos_contrato.go`; `concursos_calculo.go` | Distingue nivel del puesto y grado personal. Puntuar un mérito no reconoce un grado. Propiedad de B; no duplicar. |
| Vista de méritos | `web/static/portal-empleado/modulos/meritos/vista.js` | Consulta por datos inyectados, sin fuente de Formación ni Registro Único de Méritos compuesto. |
| Méritos del candidato | `internal/candidate/domain/merit.go`; `application/service.go` | Ligados a convocatoria. `AddMerit` no sustituye al registro único ni a la acreditación de oficio. |
| Documentos comunes | `internal/vec/application/documentos.go`; `internal/vec/ports/documentos.go` | Reutilizar custodia y firma; un borrador o certificado de acceso no acredita firma documental. |

No hay módulos propios `carrera` ni `formacion` en esa base publicada.
Granada sí dispone de [plataforma y plan institucional de Formación](https://www.dipgra.es/servicios/empleo-y-formacion/formacion-para-el-empleo-publico/plan-agrupado-de-formacion/).
Orden de dirección 16:15: **VEC consulta y deriva** mientras se aclara qué funciones conserva esa plataforma.

## Candidatas de esta sesión

| Rama propia | Fuente congelada | Resultado |
| --- | --- | --- |
| `trabajo/codexh-formacion-plan-20261001` | `6e7b0138a95abd9ef24f592f1c36df68c389afbd` | CLI Go revisa un plan sintético, fechas, necesidades, ediciones, plazas y presupuesto; visor ES/EN con filtro, detalle, pendientes y descarga del JSON original. |
| `trabajo/codexh-carrera-preparacion-20261001` | `e39f32f48786174c6c5adc65a613e27fa5a0226a` | CLI Go revisa integridad de grado, progresión y promoción; conserva antecedentes, fuentes, periodos y referencias. Visor ES/EN y descarga. Todos los casos siguen pendientes. |

Fuentes verificadas en `origin`: Formación [PR #312](https://github.com/aavidad/VEC_Diputacion_app/pull/312) y Carrera [PR #313](https://github.com/aavidad/VEC_Diputacion_app/pull/313),
ambas en borrador y con CI en curso. Tienen pruebas focales, revisión independiente
y Chrome ES/EN 1440/390 con
filtros, descarga y zoom de presentación 200 %. Go/race/vet/build y cierre de web/i18n/manifiestos se comprobaron por fases;
2492 pruebas web y el sufijo literal de calidad pasaron en cada candidata. Revisión
independiente final GO y fuentes remotas verificadas. CI sigue en curso; no declarar
LISTA hasta su verde. El turno H se liberó sin publicar páginas o ejemplos.

Las CLI son herramientas de preparación y vista previa para RRHH, reutilizables en
el circuito de trabajo; no un motor desechable de demostración. El paquete de datos
actual sí es de ensayo y retirable: la versión actual solo acepta el alcance sintético.
El uso productivo con antecedentes reales requiere fuente y autorización positivas;
este corte no lo acredita. No producen inscripciones, solicitudes registradas,
selección, reconocimiento, firma ni incorporación a méritos. No añaden SQL o permisos.
Los ejemplos son retirables: están en JSON de entrada y en la proyección de cada visor;
no se instalan en bases ni se publican en manifiestos productivos.
Arranque común: `python3 scripts/servir_preparacion_rrhh.py --modulo formacion`
o `--modulo carrera`. Solo sirve recursos cerrados en loopback, sin credenciales.
Antes de retomar, comprobar las PR y su integración; nunca volver a producir estas piezas.

## Huecos frente al catálogo

| Capacidad | Trabajo pendiente |
| --- | --- |
| CAR-001 Grado funcionarial | Fuente de relación, nivel, grado reconocido y periodos; política provincial versionada; expediente autorizado, revisión, resolución e inscripción por Personal. La preparación actual no concede derechos. |
| CAR-002 Progresión laboral | Convenio consolidado y modificaciones, colectivo y categoría; convocatoria, curso/prueba, revisión y resolución. No convertir antigüedad en progresión automática. |
| CAR-003 Promoción interna | Requisitos de las bases y antecedentes de Personal; referencia al proceso de A, seguimiento de pruebas y acto/toma de posesión por B. No crear otro motor selectivo. |
| FOR-001 Plan de formación | Consulta del plan y ediciones de la fuente institucional, con versión y procedencia. Necesidades, presupuesto y publicación se gestionarán donde RRHH confirme su autoridad. |
| FOR-002 Solicitud y selección | Derivación canónica o adaptador corporativo; solicitud/estado/recibo originales, autorización competente, criterios, listas y sustituciones. No implementar otra inscripción o selección por defecto. |
| FOR-003 Ejecución y certificado | Fuente de asistencia, aprovechamiento, evaluación, coste y certificado firmado. Entrega idempotente de hechos acreditados a Méritos, sin atribuir puntos ni grado. |

## Orden de minitareas

Ahora solo se terminan las dos candidatas casi hechas y este plan. Las filas siguientes
requieren reanudación expresa de dirección; no se empieza más producto en esta sesión.

Cada fila se entrega con un consumidor o una comprobación concreta. Si no cabe,
dividir por el contrato indicado; avisar a dirección tras 30 minutos sin avance.

| Orden | Minitarea y propietario | Salida y condición para continuar | Archivos previstos | Dependencias |
| --- | --- | --- | --- | --- |
| H01 | H: verificar PR, SHA de main y limpieza de esta entrega | Inventario actualizado en este fichero; no repetir candidatas integradas. | Este plan y `CANAL_CLAUDE_CODEX.md` | Cierre y PR de las candidatas |
| H02 | H con RRHH: identificar fuente institucional de Formación | Decisión sobre catálogo, inscripción, selección y certificados; URL/identificadores/contrato confirmados. Sin respuesta, solo derivación informativa. | Este plan; `dudas.md` solo en turno H | RRHH; autoridad corporativa; turno H |
| H03 | H: puerto de consulta de catálogo corporativo | Plan/acción/edición, fecha, plazas, estado, fuente y versión; datos mínimos. Una fuente caída no genera ejemplos como sustituto. | `internal/modules/formacion/ports/catalogo.go` | H02 y contrato confirmado; no inferir API de una URL |
| H04 | H: adaptador de esa fuente y consumidor interno de solo lectura | Consulta o enlace canónico acordado, estados vacío/error/denegado, ES/EN. Sin autoridad de lectura propia, no devolver solicitudes de personas. | `formacion/adapters/corporativo/*`; `modulos/formacion/cliente-http.js`, `vista.js` y catálogos | H02–H03; lectura autorizada si hay datos propios; turno de montaje |
| H05 | B produce antecedentes; H acuerda y consume el contrato | Persona/empleado/referencia de relación, régimen, grupo/subgrupo, ocupaciones, nivel, grado y servicios con procedencia, versión y vigencia. Consulta por puerto, sin tablas cruzadas. | Puerto de Carrera `ports/antecedentes.go`; B decide sus archivos propios | Acuerdo B; contrato sin leer datos personales |
| H06 | H: política CAR-001 en catálogo de datos | Vías, periodos computables, límites y evidencias según fuente provincial aprobada. Configuración ausente o sin aprobación devuelve pendiente. | `data/catalogos/carrera/*`; `carrera/domain/politica.go` | RRHH valida fuente/versión; solo pruebas sintéticas hasta H08 |
| H07 | H: expediente de preparación CAR-001 con esas instantáneas | Faltantes, contradicciones, periodos y revisión explicada. Nivel del puesto separado del grado; no sumar periodos solapados. | `carrera/application/preparar.go`; `modulos/carrera/vista.js` y catálogos | H05–H06; H08 y lector autorizado B antes de consumir datos personales |
| H08 | H y autoridad central: perfil fijo y acciones Carrera | Consulta/preparación/decisión/descarga exactas, ámbito y finalidad. Provisión por huella y CAS; nunca publicación de permisos por petición. | `carrera/ports/autorizacion.go`; composición nominal; gobierno central según dueño | Autoridad central; lector nominal B; SQL reservado si corresponde |
| H09 | H: candidata de persistencia y SQL borrador | SQL solo en borrador reservado, con posición en `ORDEN_SQL_NUCLEO.md`; historia aditiva, versión y operación idempotente. Instalación solo por dirección. | `carrera/adapters/postgres/*`; SQL nuevo reservado; `ORDEN_SQL_NUCLEO.md` por custodio | H05–H08; ensayo/clon y revisión sensible; sin instalación implícita |
| H10 | H: revisión/resolución CAR-001; B: inscripción del acto | Documento y firma reales por sus puertos. Personal confirma inscripción con recibo; Carrera conserva la referencia y reconcilia fallos sin duplicar el acto. | `carrera/application/resolver.go`, puertos documentos/registro; B decide inscripción | H09 instalado por dirección; H08 montado; recorrido durable; firma real |
| H11 | H y A: cotejo CAR-003 frente a bases versionadas | A conserva convocatoria, admisión, pruebas y decisión selectiva. Carrera muestra cumple/no cumple/pendiente con motivo y fuente; no sustituye la admisión. | `carrera/ports/proceso_selectivo.go`; `application/cotejar_promocion.go` | A confirma contrato y autoridad; H05 y H08/lector B antes de datos personales |
| H12 | H y B: seguimiento de promoción | Estado desde A y toma de posesión desde B. Un resultado de prueba o una propuesta no crea una relación de servicio. | `carrera/application/seguimiento_promocion.go`; vista y catálogos propios | H11; H08; fuentes autorizadas A/B y actos reales de B |
| H13 | RRHH y H: política CAR-002 independiente | Convenio vigente consolidado, cadena de modificaciones, colectivo/categoría, convocatoria y curso/prueba admitidos. Sin fuente aprobada, preparación pendiente. | Catálogo Carrera de progresión; pregunta en `dudas.md` en turno | RRHH/convenio consolidado; H08 antes de datos personales |
| H14 | H: expediente CAR-002 sobre contratos H05/H08/H09 | Mismo patrón de integridad, historia y recibo; política propia, sin reutilizar requisitos funcionariales. Resolución e inscripción separadas. | `carrera/domain/progresion.go`; caso de uso/vista propios; SQL borrador si nuevo dato | H05, H08; H09 instalado y H13 aprobado antes de efecto real |
| H15 | H: consulta propia/derivación FOR-002 con la autoridad confirmada | Recuperar estado y justificante original; distinguir solicitud, autorización, selección y notificación. No confirmar éxito ante resultado incierto. | `formacion/ports/solicitudes.go`; adaptador corporativo y vista propia | H02–H04; autoridad nominal/contexto empleado B; permisos/privacidad revisados |
| H16 | H: hechos FOR-003 desde Formación y Documentos | Asistencia, superación y certificado son estados diferentes; validar procedencia y firma/custodia según contrato, sin emitir certificados ficticios. | `formacion/ports/ejecucion.go`, `certificados.go`; adaptadores fuente/documentos | H02; lectura nominal propia y de documentos; competencia/aprobación confirmadas |
| H17 | H: Formación entrega al RUM propio y reconcilia | Curso, horas, resultado, evidencia y vigencia se incorporan una vez cuando proceda. Conservar recibo y reintento; no copiar puntos de una convocatoria. | `formacion/ports/meritos.go`; entrega/reconciliación; H conserva el consumidor RUM | H16; RUM01–06 de H completos; permisos; SQL/eventos durables autorizados |

H03/H04 se entregan juntos con consumidor. Una URL pública permite derivar,
pero no demuestra que exista una API. H05 acuerda el contrato; H07 usa únicamente
sintéticos hasta H08 y el lector autorizado de B.

RUM01–06 se ejecutan antes de H17; RUM01 puede avanzar junto con el contrato H05.

H02–H04 pueden avanzar a la vez que H05–H07 en archivos exclusivos.
H11 se acuerda con A mientras RRHH valida H06; no invadir su proceso.
H13 no bloquea CAR-001/CAR-003 ni la derivación de Formación.

## Registro Único de Méritos: responsabilidad de H

Una persona aporta cada hecho una vez, con fuente, evidencia, vigencia y estado
(declarado, pendiente, acreditado o rechazado). Requisitos de acceso, previsión
admitida por unas bases y puntos de una convocatoria son decisiones separadas.
El RUM sirve a empleados y aspirantes sin exigir empleo al externo ni duplicar Persona.
Los accesos internos y externos permanecen separados por canal, perfil, campos y finalidad.

| Orden | Responsabilidad de H | Archivos previstos | Dependencias y cierre |
| --- | --- | --- | --- |
| RUM01 | Hecho y procedencia versionados, evidencia opaca, vigencia y estados | `internal/modules/meritos/{domain,ports}/*` | Contratos de Persona/Documentos; modelo sintético probado, sin asumir acreditación de `candidate.AddMerit`. |
| RUM02 | Declarar, verificar, rechazar y rectificar con historia de solo adición | `meritos/application/*`; puertos de autorización y auditoría | D autoriza nominalmente; diferencia declarante/verificador y fuente. Todo efecto conserva motivo, versión y recibo, sin borrar hechos anteriores. |
| RUM03 | Persistencia y gobierno de acciones/campos propios | `meritos/adapters/postgres/*`; SQL nuevo reservado y orden causal | Reserva/ORDEN_SQL_NUCLEO por custodio, dos revisiones y ensayo. Autorización+estado+auditoría+outbox en transacción; instalación por dirección y recuperación sin duplicados. |
| RUM04 | Consulta/aportación propia en portales separados y revisión competente | `modulos/meritos/{cliente-http,vista,i18n}.js`; textos ES/EN; consumidor externo propio | RUM02–03 instalados/compuestos; identidad común y permiso exacto. Reutilizar vista actual, extraer textos a catálogos; no compartir permisos entre empleado y candidato. |
| RUM05 | Puerto de hechos para Selectivos A y baremador existente | `meritos/ports/hechos.go`; adaptador lector y consumidores por sus dueños | RUM03 con lectura positiva, minimización y auditoría. Cada proceso conserva bases, hito de cumplimiento y valoración; nunca trasladar puntos al RUM. |
| RUM06 | Conformidad, reutilización y reconciliación de fuentes | Casos de uso/pruebas focales de `meritos`; contrato Formación→RUM | Correspondencia Persona/curso/evidencia/versiones, conflicto y reintento explicados. Conservar historia del candidato; ninguna importación masiva o acreditación automática sin circuito acordado. |

La entrega desde Formación (H17) conserva el certificado original en Documentos,
registra el hecho acreditado y devuelve recibo reconciliable. Curso inscrito,
asistencia, superación y certificado firmado siguen siendo hechos distintos.

## Decisiones y dependencias

En el turno H, numerar en `dudas.md` las preguntas ya enviadas al canal: funciones
que conserva Formación corporativa, convenio vigente para laborales y vías de
reconocimiento del grado. Añadir responsables, fuente/versiones y fecha de aprobación
a los catálogos; ninguna cifra de ensayo se convierte en norma oficial.

B conserva Personal, sus actos y provisión; A conserva procesos selectivos; el baremador conserva su autoridad y responsable
confirmado, sin reasignarlo por inferencia;
M conserva Organización/RPT. Documentos conserva custodia y firma. Dirección17:40 asignó el Registro Único de
Méritos a H; A y el baremador lo consumen por puerto. El modelo del candidato actual
no lo sustituye. H05 es consumidor: B produce y estima el lector de antecedentes
en su plan; aquí solo se cuenta el acuerdo/consumo de H, sin duplicar su trabajo.
Las operaciones entre módulos usan referencias opacas, eventos y recibos reconciliables.

Manifiestos, padres, importadores y dudas siguen la cola B → A → E → G → F → M → H → I → J,
con LIBERO y SHA. No iniciar una migración sin reserva y orden causal, ni usar un
borrador SQL como capacidad instalada. No escribir en cidonia ni en datos reales.

## Pruebas y primera acción de la próxima sesión

Primero comprobar `gh pr list`, los FIN de H y los hashes publicados de las dos candidatas.
Después ejecutar **H05**: acordar con B el DTO/puerto de antecedentes y preparar
solo el consumidor de H en la rama prevista `trabajo/codexh-antecedentes-carrera-20261002`
(todavía sin crear). B conserva el lector y su autorización. H02 se resuelve con RRHH
en paralelo; si B no entrega todavía, RUM01 puede avanzar con datos sintéticos.
No crear un catálogo corporativo paralelo ni otro motor de selección.

Pruebas focales de fechas, fuentes, solapes, versiones y pendientes; Node sobre la
interfaz afectada; Chrome del sistema a 1440/390 y zoom 200 %, ES/EN y teclado.
Gosec solo en paquetes cambiados y Semgrep local sin métricas. Revisión independiente;
dos en SQL, permisos, identidad, firma o datos personales. Calidad global una vez
por PR cuando corresponda; no repetir puertas verdes. Registrar impedimentos de entorno sin omitir controles obligatorios.

## Estimación

Horas de trabajo de un equipo Codex con subagentes, revisión y CI. Las filas largas
se reparten en cortes de PR de 1–3 horas, conservando el mismo contrato y propietario.
No son plazos de aprobación ni fechas de despliegue.

| Minitarea | Horas de un equipo |
| --- | --- |
| H01 Verificar entregas | 1–2 |
| H02 Acordar fuente Formación | 2–4 |
| H03 Puerto de catálogo | 2–4 |
| H04 Adaptador y consulta visible | 4–8 |
| H05 Puerto consumidor de antecedentes B | 1–2 |
| H06 Política CAR-001 | 4–8 |
| H07 Preparación de expediente | 5–9 |
| H08 Autorización nominal | 6–10 |
| H09 Candidata de persistencia | 8–14 |
| H10 Resolución e inscripción | 10–18 |
| H11 Cotejo de promoción | 5–9 |
| H12 Seguimiento de promoción | 6–10 |
| H13 Política laboral | 3–6 |
| H14 Expediente laboral | 8–14 |
| H15 Consulta/derivación de solicitudes | 6–10 |
| H16 Ejecución y certificado de fuente | 8–14 |
| H17 Entrega Formación→RUM | 8–14 |
| RUM01 Hechos y procedencia | 4–6 |
| RUM02 Declaración/verificación/rectificación | 8–12 |
| RUM03 Persistencia y autorización | 10–16 |
| RUM04 Consultas y aportación propias | 10–16 |
| RUM05 Lectura por otros procesos | 4–8 |
| RUM06 Conformidad y reconciliación | 2–4 |

Total técnico de H: **135–226 horas**, unas **17–29 jornadas de un equipo** de ocho horas.
Con dos equipos, Carrera y Formación/RUM pueden separarse: **12–21 jornadas**, porque
contratos de Personal, autorización, firma, RUM y revisión limitan el paralelismo.

Trabajo externo: Personal B y proceso A pueden añadir **1–3 jornadas cada uno** si
sus contratos están cerca; servidor/autoridades nominales, **2–4 jornadas**; Formación
corporativa/Documentos, **3–8 jornadas** de coordinación si hay fuente/adaptador disponible.
Parte puede solaparse: no sumar esas horquillas automáticamente. La espera de RRHH,
convenio y vías de reconocimiento no tiene fecha comprometida. Sin respuesta o API
admitida, solo se cierra consulta/derivación; no se declara terminado el circuito completo.
La estimación supone fuentes y competencias confirmadas, y se revisa al cerrar H02/H05.

## Consenso Astra

- Se compararon Granada, Alicante, EAPC, INAP y GVA. Granada ya tiene plataforma
  formativa: ausencia de módulo VEC no autoriza sustituirla. Dirección confirmó consulta y derivación.
- Se acordó separar grado, progresión laboral y promoción. No existe una regla universal
  de examen o ascenso por antigüedad; solo políticas provinciales aprobadas y versionadas.
- Las herramientas inmediatas se limitaron a preparación sintética exportable. El canal
  Go → JSON → visor local evita tocar el servidor de ensayos que pertenece a A.
- Se conservaron procesos de A, actos/registro de B y custodia/firma comunes. Un curso
  acreditado no equivale a puntos, y una resolución no equivale a inscripción confirmada.
- Ronda final: Astra dio GO al contenido `af90952f…` tras exigir archivos y dependencias
  por minitarea, autorización antes de antecedentes personales, SQL borrador e instalación
  separados, SHA/estado de candidatas y parada explícitos. La tabla se corrigió a cinco
  columnas sin cambiar su contenido. Falta el GO de dirección sobre este documento exacto;
  no abrir la PR del plan antes de recibirlo.

Astra validó también la estimación del documento `0c1a028d…`: 17 filas suman 99–168 h;
13–21 jornadas de un equipo y 9–15 con dos, condicionadas a fuentes y autoridades.
Se separa la espera externa y se revisa la horquilla al cerrar H02/H05.

Se aplican los cambios de dirección 17:40: CAR-004 fuera, RUM dueño H y H05 consumidor
con rama prevista. La estimación anterior de 17 tareas y 99–168 h queda sustituida por las
23 tareas y 135–226 h; Astra dio GO a esta ampliación y su cálculo en el documento `4017c75f…`.
Falta el GO final de dirección sobre este documento antes de abrir su PR.


## Retoma H07 — 7 de octubre de 2026

Se recuperan los diez archivos de la PR #388 sobre la base `01046e2e7`,
sin repetir H05 ni H06. La CLI combina la declaración, los antecedentes y
la política de grado; conserva sus diferencias, los periodos solapados y
la procedencia. La preparación mantiene pendiente el reconocimiento.

Pruebas focales Go y vet correctas, revisión independiente favorable,
Semgrep local (42 reglas, siete archivos) y gosec focal sin hallazgos.
Medición local del ejecutable: 64 casos, 30 ejecuciones; mediana 14,92 ms,
p95 16,64 ms y máximo 21,79 ms, incluidos arranque, archivos y salida JSON.
Esta medida corresponde a la CLI; no mide PostgreSQL ni una pantalla.

Faltan el lector nominal de Personal, la autorización H08, el registro del
expediente y el reconocimiento por quien tenga competencia. El corte
recupera una preparación existente; no completa Carrera ni habilita datos reales.
