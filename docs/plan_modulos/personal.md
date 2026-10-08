# Personal: continuación acotada

Propuesta para consenso de Dirección, Astra y Claude. El módulo se detiene tras acordar este plan; solo se rematan las piezas ya casi terminadas. Ninguna tarea de la cola autoriza código nuevo por sí sola. La prioridad de Contratación temporal y la cola H6 B205/B206/B204/B209 siguen separadas.

## Punto de partida

Base del inventario: `0a62a3ea68e58fbf890885a2f59cc80343107e18`, consultada por Git; main puede avanzar después. «Mi ficha» consulta las relaciones y los servicios propios con el empleado canónico de ContextoActor y una concesión V3; RRHH dispone de lista, detalle y actos de relaciones, ocupaciones, situaciones y servicios con historia bitemporal. RPT publica organización propia. Documentos conserva su autoridad común. Es una consulta parcial: faltan una historia propia uniforme, documentos vinculados a hechos de Personal, la fuente canónica de datos personales y un cauce general de rectificación. La competencia y la fuente del empleado para Dietas (`AUT-27`) siguen pendientes de decisión.

Las piezas abiertas se verifican por su hash exacto antes de afirmar integración:

| Pieza | Estado comunicado y remate permitido |
| --- | --- |
| PR #298, `trabajo/codexb-personal-ficha-recuperacion-20261001` @ `4b7138324` | Implementada; dos revisiones sensibles del producto `ab3442f77` y 65 pruebas focales. El sucesor cambia solo dos fixtures de ausencia en el coordinador: 26/26 pruebas. Producto idéntico; calidad local por fases cerrada (2483 Node y controles restantes verdes), base de tamaño1254 justificada; CI pendiente. |
| PR #299, `trabajo/codexb-personal-traza-rrhh-20261001` @ `94d7e3395` | Implementada; dos `GO` sensibles, `GO` de UX y 61 pruebas focales. Producto revisado en `437cbfb18`, CSS corregido en `0ab7fff8a` y los mismos dos fixtures incorporados en el sucesor: 26/26. Sin solapamientos a 1440/390 px y con reflujo CSS al 200 %. No se ha acreditado zoom nativo. La secuencia local HTTP con fixture dio 503 → 200 al reintentar → 200 al actualizar → 403 tras revocación simulada; no acredita mTLS, PostgreSQL ni navegador con backend real. Calidad local por fases cerrada (2484 Node y controles restantes verdes): la única intermitencia heredada de bootstrap pasó aislada en carrera. CI pendiente. |
| PR #300, `trabajo/codexb-personal-expediente-borrador-20261001` @ `3baa47910` | Solo JSON, comentarios de SQL 29 y apéndice; `GO` de preparación. Sin ensayo SQL ni activación. Comprobar CI y fusión documental sin atribuirle persistencia operativa. |

`Personal 000025` se reservó antes del borrador para la proyección histórica propia y el vínculo documental. Queda después de M3 y del contrato nominal acordado con D; no es una migración de competencia de Dietas. El borrador #300 conserva referencias de su primera propuesta a ocupaciones de Personal; debe alinearse con la propiedad ORG/RPT de M antes de producir SQL ejecutable. La reserva no aprueba la fuente, los permisos ni la migración. No editar el hito 6 ni activar SQL por este plan.

## Requisitos PER y frontera con Contratación temporal

| Requisito | Existente y hueco | Autoridad y continuación |
| --- | --- | --- |
| PER-001: Registro de Personal | B2 registra relaciones, situaciones y servicios con procedencia; ficha propia parcial. Faltan lectura histórica propia uniforme, documentos y fuentes para antigüedad/grados. | B produce el expediente laboral y antecedentes acreditados; H calcula y gobierna Carrera y el Registro Único de Méritos. No calcular trienios o grado desde fichajes ni inferir reconocimiento del tiempo declarado. |
| PER-002: Alta y toma de posesión | Existe plan/intención de incorporación CT→Personal, preparado en Personal23; no acredita toma de posesión eficaz ni altas corporativas. | CT tramita el expediente temporal y aporta el acto admitido; B confirma y registra la relación/incorporación por el puerto existente. M resuelve ocupación RPT; Usuarios/Sistemas identidad corporativa; Nómina y comunicaciones conservan sus acuses propios. B no duplica el procedimiento CT. |
| PER-003: Contrato, nombramiento y cese | B2 registra actos y CT conserva contrato/nombramiento/cese temporal; falta cerrar su conciliación como hechos laborales admitidos. | CT conserva documentos, propuesta y trámite temporal. B registra la relación y su fin con acto, fuente y versiones; publica el antecedente mínimo hacia Bolsa/Nómina. M conserva reserva y ocupación RPT. Nunca llamar cese eficaz a un borrador o ejecutar liquidaciones desde Personal. |
| PER-004: Situaciones | Existe catálogo versionado y registro B2 de situación; faltan régimen, vigencia y efectos admitidos. | B gobierna el registro de la situación laboral por régimen y acto. M aplica su efecto autorizado en ocupación/reserva; Cronos/Nómina/H reciben contratos o eventos propios. Las reglas de derechos, jornadas y efectos no se deducen del nombre de la situación. |
| PER-005: Rectificación | Hay propuesta acotada para asignación de Dietas; falta circuito general de datos oficiales de Personal. | B produce propuesta, evidencia, decisión competente y revisión nueva. I solo consume. No reutilizar una rectificación de Dietas como permiso general de edición. |

## Propiedad de código y contratos con otros módulos

La ubicación actual en `internal/modules/personal/` no convierte Organización/RPT en trabajo duplicado de B. Según Dirección17:40:

- **M:** `domain/organizacion.go`, `domain/organizacion_historica.go`, `domain/importacion_organizacion_historica.go`, `domain/rpt_publica.go`; `ports/organizacion_historica.go`, `ports/edicion_organizacion.go`, `ports/importacion_organizacion_historica.go`, `ports/rpt_publica.go`; las aplicaciones `consulta_organizacion_historica.go`, `importacion_organizacion_historica.go`, `consulta_rpt_publica.go`; adaptadores `organizacionpublica/`, `rptpublica/`, PostgreSQL `organizacion*.go` y sus pruebas. M conserva ORG-001, RPT-005/006 y su estimación de ocupación, reserva y vacantes. La fuente de responsables, delegaciones y suplencias es M; D aplica su autorización; I y B consumen según competencia.
- **B:** `domain/relacion_empleado.go`, `domain/ficha_propia.go`, `domain/registro_empleado_b2*.go`, `domain/plan_incorporacion_ct.go`; sus puertos y aplicaciones homónimos, `application/consulta_relacion_empleado.go`; adaptadores PostgreSQL `relacion_empleado.go`, `ficha_propia.go`, `registro_empleado_b2_*.go`, `plan_incorporacion_ct*.go`, consumidores CT y pantallas de ficha/registro B2. B produce relación laboral, actos/documentos propios y servicios reconocidos. I solo los consume. Los tipos existentes de ocupación B2 se conservan como antecedente/contrato de lectura, sin abrir otro gobierno de RPT.
- Los archivos mixtos, catálogos comunes y la composición no se editan a la vez. Se acuerda una tarea acotada con un único escritor y el turno de compartidos; no se modifican SQL ya instaladas. En un corte posterior puede separarse `internal/modules/organizacion/` y sus puertos, conservando contratos y datos; esa reorganización no entra en esta estimación ni autoriza mover código ahora.
- **CER-001:** B produce un puerto autorizado de servicios reconocidos con periodos, acto, fuente, corte y cobertura. J conserva plantillas, revisión, firma, CSV, registro y entrega de certificados; el menú es de J y Personal solo enlaza. Un listado B2 sintético no se transforma en certificado oficial. Trabajo propio B: **8–16 horas**, incluido abajo; J no vuelve a contarlo como producción de datos.
- **H05:** B produce antecedentes acreditados de relación, servicios, situaciones y puesto/nivel recibidos de M, con versiones y corte. H calcula Carrera y conserva el Registro Único de Méritos; A y el baremador consumen ese registro por puerto. B no calcula un grado desde una referencia de puesto ni mantiene otro expediente de méritos. Trabajo propio B: **6–12 horas**, incluido abajo.
- **RPT-005/006:** B aporta únicamente la relación de la persona por puerto, con identidad opaca, estado, vigencia y versión. M decide ocupación, reserva y vacantes con su fuente. Este puerto B cuesta **3–5 horas**; no incluye ni duplica el cálculo o mantenimiento RPT estimado por M.

## Orden al reanudar

1. Verificar CI, hashes y estado de fusión de #298, #299 y #300; resolver solo sus remates concretos. Dirección integra en la rama canónica tras revisión del candidato final. Registrar por separado código fusionado, SQL ensayado/instalado y recorrido acreditado.
2. Obtener de RRHH y Sistemas el contrato exacto de fuente y competencia de Personal, incluido `AUT-27` para Dietas, antes de otra implementación. Resolver las dudas existentes 27/28/29 (fuente, campos y roles), 34 (perfil) y 39 (sistemas corporativos) en su seguimiento propio. Una cuenta, certificado o cargo no acredita relación ni concede acceso. No duplicar identidad, RPT o Documentos.
3. Solo con ese contrato, encargar una pieza visible por PR. Cada pieza llevará un dueño de archivos exclusivo, consumidor real, autorización V3 por acción/ámbito/finalidad/campos, fuente y fechas expuestas, historia conservada y comprobación focal. Si toca SQL, reservar antes el número, ensayar en clon y obtener revisión SQL independiente y dos revisiones sensibles del hash final; si toca pantalla, aplicar las skills visuales y revisión independiente de usabilidad. Semgrep local sobre lo cambiado antes del PR.

## Garantías de todos los cortes

Una sesión actúa con un único perfil activo. Autorización conserva perfiles fijos y asignaciones nominales; su provisión usa huella y CAS, nunca permisos publicados por petición. Cada lectura consume la concesión vigente y registra la auditoría en la misma transacción. En escrituras, estado, versión, auditoría y outbox aplicable se confirman juntos; la historia y la auditoría son de solo adición. No se crea un outbox nuevo para una consulta. Personal se sirve solo en el proceso interno; Cronos exterior y cualquier carpeta agregada no amplían su exposición ni sus campos.

## Cola propuesta, sujeta a las decisiones anteriores

| Pieza visible y propietaria | Archivos orientativos exclusivos | Dependencia y comprobación proporcionada |
| --- | --- | --- |
| **Dependencia de Dietas: competencia acreditada.** Comprobar primero las consultas ya existentes de relación propia y competencias de Personal. Completar solo el contrato nominal que falte con D y G. Entrega coordinada: el consumidor real de Dietas muestra si la relación y la competencia son válidas o por qué no están acreditadas, sin copiar la ficha. | Dueño Personal: `internal/modules/personal/{domain,application,ports,adapters}`. Dueño de integración posterior: puerto consumidor de Dietas; no editar ambos en un mismo encargo. | Fuente/competencia 27–29 y 39 y AUT27 de D; no reutilizar la reserva Personal000025 para esta dependencia. Probar dos relaciones, relación cesada, revocación, denegación y referencia ajena en aplicación/HTTP; PostgreSQL real solo tras migración autorizada. |
| **Datos de la persona en su ficha.** Mostrar los campos internos acordados de identidad y contacto por los puertos autorizados de Persona y Usuarios, con fuente y fecha; el registro de Personal conserva solo su vínculo laboral. No crear campos personales nuevos en tablas compartidas ni usar el perfil externo como fuente del interno. | Dueño Personal: proyección y vista propia; dueños Persona/Usuarios: sus contratos de lectura, en encargos dependientes. | Dudas 27/28/34 y fuente admitida. Probar minimización por perfil, dos identidades sintéticas, negativa cruzada entre procesos y retirada de datos al revocar. No repetir el alta de la persona ni inferir empleo de su certificado. |
| **Primer incremento lector: periodos propios y procedencia.** «Mi ficha» muestra una página de relaciones y servicios del titular, con fechas efectivas, corte de conocimiento, estado y fuente. Personal proyecta sus hechos con una autorización propia; no presta la consulta B2 de RRHH. No incluye la decisión de rectificación ni la historia administrativa de terceros. Una segunda pieza añade puestos y situaciones propios con su contrato específico. | Dueño Personal: `internal/modules/personal/{domain,application,ports,adapters/postgres}` y sus vistas propias bajo `web/static/portal-empleado/modulos/personal/`. | Depende de Personal000025 y de campos/periodos autorizados. Probar corte efectivo frente al de conocimiento, varias relaciones, revocación entre páginas, hecho ajeno y fuente incompleta. Validar la lectura de una rectificación ya registrada, sin escribirla desde este corte. Ensayo SQL cuando exista candidata ejecutable y recorrido real de lectura; ninguna capacidad activa por el borrador. |
| **Documentos de Personal.** Desde un acto o periodo se lista y descarga el original o representación permitida, con versión, estado y procedencia. Documentos custodia el fichero; Personal conserva solo el vínculo `hecho–documento–versión`. | Dueño Personal: puerto y adaptador de vínculo en `internal/modules/personal/{ports,application,adapters}` y vista propia. Dueño Documentos, en encargo dependiente distinto, solo si su contrato actual necesita ampliación. | Depende de la terna y de la clasificación acordadas. Probar original frente a representación, versión sustituida, acceso propio/RRHH, denegación de terceros y descarga real; adjuntar no concede permiso. |
| **Solicitud general de rectificación.** El titular propone la corrección de un hecho de Personal con evidencia; RRHH competente decide y crea nueva versión. Reutiliza el circuito y recibos comunes de VEC. | Dueño Personal: `internal/modules/personal/{domain,application,ports,adapters}` y pantalla propia; dueño del circuito común solo si aparece una carencia concreta, en tarea dependiente. | Depende de campos rectificables, decisor y fuente. Probar propuesta, denegación, decisión, idempotencia, versión concurrente y lectura de ambas versiones tras reinicio. No presentar la solicitud como corrección ya aprobada. |
| **Historia RRHH y competencias gobernadas.** RRHH consulta la secuencia laboral y actúa solo con concesión central vigente. M aporta responsables/delegaciones/suplencias por su puerto; D aplica el permiso y B no crea otra fuente de cargos. | Dueño Personal: proyección y pantalla RRHH en `internal/modules/personal/` y `web/static/portal-empleado/modulos/personal/`; Autorización gobierna la concesión en encargo propio si falta contrato. | Depende de 27–29 y 34, de la fuente vigente y de suplencias definidas. Probar actor competente, otro ámbito, revocación, delegación vencida y auditoría. No otorgar permiso por petición del navegador ni por nombre del cargo. |

### Contratos y actos todavía sin completar

| Pieza B | Archivos orientativos exclusivos | Resultado y dependencia |
| --- | --- | --- |
| Relación nominal para RPT-005/006 | `ports/relacion_para_rpt.go`, `application/relacion_para_rpt.go` y su prueba; adaptación de lectura propia de Personal tras contrato. | M recibe relación vigente/corte/versión o indisponibilidad; sin nombres, cargo inferido ni consulta a tablas ajenas. Revalidar concesión y relación revocada; fuente de ocupación/reserva/vacantes permanece M. |
| Servicios de oficio para CER-001 | `ports/servicios_para_certificados.go`, `application/servicios_para_certificados.go` y pruebas; adaptador durable de Personal posterior al consumidor nominal acordado con D. | J obtiene periodos realmente reconocidos, acto/fuente/versiones, corte y cobertura. Distinguir declarado/comprobado/reconocido, solapes y fuente incompleta; no sumar antigüedad por decisión de la vista. |
| Antecedentes para H05 | `ports/antecedentes_carrera.go`, `application/antecedentes_carrera.go` y pruebas; adaptación con fuente autorizada de relación y puerto RPT de M. | H recibe antecedentes mínimos comprobados o pendientes, con procedencia y validez, sin grado inferido ni duplicar Registro Único de Méritos. |
| PER-002: confirmar incorporación laboral | Aplicación/puerto existentes `plan_incorporacion_ct.go`, consumidor CT y pruebas focales; composición en encargo único separado. | Conciliar la intención existente con hecho/acto admitidos de Personal. CT y B conservan claves/recibos; M y otros dueños confirman sus efectos. Sin cuenta o toma de posesión fabricadas. |
| PER-003: registrar contrato/nombramiento y fin | `registro_empleado_b2_actos.go` y pruebas, puente CT existente por puerto. | Registrar antecedente laboral admitido y cese con versión esperada e idempotencia; señal mínima para los consumidores, sin reescribir documento CT ni liquidar Nómina. |
| PER-004: situación por régimen y acto | Catálogos/aplicación B2 existentes y pruebas del efecto admitido; puertos de eventos a consumidores en encargos de sus dueños. | Registrar la situación versionada y comunicar solo efectos aprobados. M/Nómina/Cronos/H aplican su autoridad; reglas de RRHH pendientes configurables, sin constantes legales inventadas. |

### Primera minitarea exacta de mañana

Solo tras comprobar las CI/mezclas #298→#299→#300, una **asignación expresa de Dirección** y el **contrato nominal acordado con J y D**: abrir **`trabajo/codexb-personal-servicios-cer-contrato-20261002`** sobre el main publicado que las incluya. Mientras falte ese contrato o encargo, se permite únicamente contrastar el documento, sin abrir implementación. Primer corte autorizado, **2–3 horas**, dentro de las 8–16 h de CER-001: contrastar con J y D el DTO y la operación positiva del puerto de servicios, reutilizando B2; escribir solo `ports/servicios_para_certificados.go`, su caso de uso neutral y pruebas de titular ajeno, servicio declarado y fuente incompleta. La autoridad se inyecta; sin consumidor nominal real el servicio deniega y no se registra HTTP ni se publica una capacidad. No convertir un doble de prueba en fuente de oficio ni entregar datos por ser un llamante interno. El siguiente corte conecta un consumidor real de J antes de anunciar capacidad. Si necesita SQL, reservar número y preparar únicamente borrador con D; no modificar el núcleo por este plan.

Formación, Registro Único de Méritos, nómina, cotización, Cronos y Dietas conservan sus propios datos y planes. La carpeta personal podrá reunir consultas mínimas de esas autoridades cuando existan contratos y fuentes admitidos; no se abre aquí un banco paralelo de cursos, méritos o pagos. M conserva estructura, plazas, puestos, ocupación, reserva y vacantes RPT. B aporta relación y actos laborales por puerto; su ficha consume la proyección mínima autorizada de M. No confundir reserva con vacante disponible ni contar ese gobierno otra vez en B.

## Estimación

Horas de trabajo de un equipo Codex con varios subagentes, revisiones y CI, como trabajamos hoy. Cada bloque se divide en PR de 1–3 horas; no es una migración grande ni una espera continua de pruebas globales.

| Minitarea | Horas de equipo |
| --- | --- |
| Cerrar #298–300, incorporar únicamente sus correcciones y comprobar CI | 1–3 |
| Contrastar fuente, campos y vínculos existentes; acordar el contrato nominal con D | 2–4 |
| Coordinar competencia de Personal y consumidor Dietas sobre los puertos existentes | 3–5 |
| Datos internos autorizados de la persona y contacto, sin duplicar sus autoridades | 3–5 |
| Relación laboral por puerto para RPT-005/006 (solo B; M estima RPT) | 3–5 |
| Dominio y puerto de historia propia de relaciones/servicios, con cortes separados | 2–4 |
| Proyección paginada y consumidor durable propios; SQL nuevo, ensayo y dos revisiones | 5–8 |
| Cliente y vista de historia propia, sin selector de empleado | 3–5 |
| Extender la lectura propia de puestos/situaciones con versiones RPT admitidas | 3–5 |
| Vínculo hecho–terna documental y listado por el servicio común | 3–5 |
| Descarga original autorizada desde la ficha, con versión y estado documentales | 2–4 |
| Solicitud de rectificación y decisión RRHH por etapas, conservando ambas versiones | 5–8 |
| Consulta de historia RRHH y competencia fija por ámbito; reutilizar los perfiles comunes | 3–5 |
| PER-002: conciliación de incorporación laboral por el puente CT existente | 4–8 |
| PER-003: antecedente laboral de contrato/nombramiento y cese | 4–8 |
| PER-004: registro versionado de situaciones y efectos admitidos | 4–8 |
| Fuente autorizada de servicios para CER-001; primer contrato de 2–3 h incluido | 8–16 |
| Antecedentes propios para H05, consumiendo fuente RPT de M | 6–12 |
| Recorrido interno completo con fuentes admitidas, reinicio y manual de uso final | 3–5 |

Total del alcance de expediente aquí definido: **67–123 horas**, aproximadamente **9–16 días de un equipo** o **6–11 días con dos equipos**. Dos equipos pueden separar cliente/vistas de proyección/documentos; la cadena nominal y SQL sigue siendo secuencial. La cifra no incluye construir nómina, cotización, Formación o Carrera ni migrar todo el histórico corporativo.

No dependen del equipo: decisiones RRHH de las dudas 27–29/34/39, contrato nominal y orden SQL de D, interfaces y diccionario de Sistemas/RPT, acceso al servidor y validación de fuentes documentales. Si se entregan al iniciar cada bloque, reservar **2–5 días laborables adicionales** para intercambio y aceptación. Si no hay fecha de respuesta o la fuente no tiene interfaz utilizable, la espera es indeterminada y esta horquilla no fija una fecha de finalización. El ensayo SQL y el recorrido exigen los accesos acordados; una aprobación del plan no los sustituye.

## Consenso Astra

Astra dio GO al planteamiento inicial tras distinguir historia propia, rectificación y consulta RRHH, actualizar evidencias y separar provisión por CAS de consumo transaccional. Dirección17:40 cambió y fijó las propiedades: M conserva ORG/RPT-005/006; B aporta relación y produce servicios CER-001, antecedentes H05 y actos/documentos propios; J conserva certificados y H el Registro Único de Méritos. Este documento incorpora el mapa PER-001…005, archivos y primera rama, y recalcula solo las horas B (67–123 h). El acuerdo previo no se presenta como GO de estos cambios; La ronda focal de Astra aceptó el reparto y la estimación y pidió explicitar la asignación nueva de Dirección/J/D antes de programar y alinear el borrador000025 con M antes de SQL. Ambos puntos quedaron incorporados y Astra dio GO al plan completo en la ronda final. La actualización posterior de hashes de #298/#299 recoge solo los remates de tests/tamaño y su evidencia local, sin cambiar alcance ni estimación. Se pide GO de Claude.

Dirección deberá confirmar el orden y el primer encargo antes de reabrir código. El GO del plan no activa Personal000025 ni autoriza despliegue.

Fuentes: `AGENTS.md`; `ESPECIFICACIONES_AGENTES.md` E02–E08; `docs/estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md` (PER, RPT y EMP); `docs/estudio_requisitos/analisis_integral_rrhh.md` §§15–22; `docs/estudio_requisitos/modelo_historico_rpt_plazas_puestos_y_vacantes.md`. Los estados de PR proceden del encargo de Dirección y requieren comprobación en Git/CI.

## Contratos nominales preparados (1 de octubre de 2026)

Por encargo expreso de Dirección se preparan tres interfaces y sus DTO mínimos
en los puertos de Personal. Este corte aporta la definición necesaria para los
consumidores posteriores; no espera las reglas pendientes de RRHH ni activa una
fuente institucional.

| Contrato | Acción y audiencia propias | Consumidor posterior |
| --- | --- | --- |
| `LectorServiciosParaCertificadosV1` | `personal.servicios_certificados.consultar` / `vec_personal.servicios_certificados.v1` | CER: servicios con estado, periodos, acto y procedencia; J conserva emisión y firma. |
| `LectorAntecedentesCarreraV1` | `personal.antecedentes_carrera.consultar` / `vec_personal.antecedentes_carrera.v1` | H05: relaciones, servicios, situaciones y puesto/nivel procedentes de M; H conserva cálculo y Méritos. |
| `LectorRelacionParaRPTV1` | `personal.relacion_rpt.consultar` / `vec_personal.relacion_rpt.v1` | RPT: relación exacta y versión esperada; M conserva ocupación, reserva y vacantes. |

Las consultas conservan el actor efectivo acreditado, referencias opacas,
organismo y cortes separados de efectos y conocimiento. La respuesta distingue
cobertura completa, parcial y no acreditada, con certeza y versión de la fuente.
La estructura de evidencia B2 se reutiliza para la evidencia de cada consulta
nominal nueva; no permite reutilizar la autorización de una operación B2.

Los contratos exigen concesión central positiva y consumo junto a lectura y
auditoría en la transacción de la fuente. Quedan sin implementación, SQL, HTTP,
composición ni autorización operativa. Las fuentes actuales de CER y H05 siguen
siendo de ensayo. Los DTO no contienen nombre, DNI ni correo y no sustituyen
las proyecciones puras de dominio ni crean otra ficha de persona.

### Servicios para Certificados: primer corte, autoservicio (5 de octubre de 2026)

`LectorServiciosParaCertificadosV1` ya tiene implementación y se añade
`LectorServiciosParaCertificadosV2`, con la misma consulta y los días
reconocidos que guarda Personal17. V1 sigue disponible y es la misma lectura
sin los días.

- Sólo autoservicio: el empleado tiene que ser el único empleado canónico de
  la persona que consulta. Se comprueba en Go con el ContextoActor y otra vez en
  SQL con la proyección de Personal16. Si se pide otro empleado, se deniega. La
  consulta de RRHH sobre otra persona necesita una competencia propia y queda
  para otro corte.
- Fuente: `vec_personal.consultar_servicios_certificados_propios_v1`
  (Personal36). Toma la última revisión conocida de cada servicio en el
  organismo pedido, vigente en la fecha del corte y ya comenzado entonces. El
  periodo y los días salen tal como constan, sin recalcular. Cobertura y certeza
  van como «no acreditada», porque la fuente no tiene eficacia administrativa.
  Con más de 200 servicios la consulta da error y no devuelve datos.
- Autorización: consumidor propio AD195 (`personal.servicios_certificados.consultar`,
  audiencia `vec_personal.servicios_certificados.v1`, finalidad
  `consultar_servicios_para_certificados`). Consumo, lectura y auditoría común
  van en la misma transacción. El recibo es la referencia de esa auditoría. Los
  rechazos se registran en la auditoría común de intentos.
- Código: `ports/servicios_para_certificados.go` (V2),
  `ports/lector_servicios_certificados.go`, `domain/lector_servicios_certificados.go`,
  `application/lector_servicios_certificados.go`,
  `adapters/postgres/lector_servicios_certificados.go` y
  `adapters/composicion/lector_servicios_certificados.go`
  (`ComponerLectorServiciosCertificados`).
- SQL en orden: `deploy/principal/lista_sql_claude_personal_servicios_certificados_20261005.txt`.
  Va después de AD193. AD195 mide el núcleo posterior a AD193: si otra
  migración lo reescribe antes, se detiene sin cambiar nada y hay que medirla
  de nuevo. Comprobación: `pruebas_sql/servicios_certificados_propios_000036.sql`.

Falta todavía montar la ruta del consumidor (Certificados), dar el permiso en
administración, configurar el origen técnico de consumo para el LOGIN de
Personal y pasar el traductor de Certificados a V2.

## Preparación propia de una revisión de servicios — 4 de octubre de 2026

La historia propia de servicios incorpora «Preparar revisión» en cada fila
recibida. Permite elegir un dato de esa revisión y describir propuesta, motivo
y evidencia declarada. El estado permanece «Preparación sin presentar».
Discutir el dato no determina que sea jurídicamente rectificable.

«Revisar borrador» vuelve a consultar la historia propia por su cliente
autorizado existente, con las mismas fechas y sin enviar referencias de persona,
empleado, propuesta, motivo ni evidencia. Comprueba las revisiones del servicio,
sus valores, fuente, acto y versiones. Si cambian o falla el acceso, retira la
historia anterior y el borrador. La procedencia del borrador revisado conserva
servicio, revisión, fuente, acto, corte y referencia de la consulta nueva.

La preparación vive únicamente en memoria y se limpia al descartarla, actualizar
la historia, cambiar fechas o desmontar la vista. No tiene descarga, portapapeles,
adjuntos, envío, registro, SQL ni permiso de escritura. Personal no utiliza el
circuito de rectificación de Dietas. El circuito de presentación y decisión
competente continúa pendiente; este corte no cierra PER-005.

Las 23 pruebas Node focales de preparación, vista e HTTP lector existente pasan
con Node 20.19.2. Incluyen selector ajeno, revisión sustituida, revocación,
dependencia caída, respuestas tardías, limpieza, texto hostil mediante textContent
y catálogos ES/EN. Semgrep local sobre cuatro archivos de implementación:
cuatro reglas, ningún hallazgo. No se ejecutaron Go, SQL ni servicios reales.
La revisión sensible y de usabilidad del candidato final, la cadena de caché y
los manifiestos corresponden a Dirección antes de integrar.

El valor recibido aparece junto al valor propuesto, antes del motivo y la evidencia.
La comparación se conserva al elegir fechas, días, estado o clase, con los formatos existentes.
La prueba Node focal comprueba su ubicación y actualización; la revisión visual corresponde a Dirección.


## Ficha propia: reanclaje de autorización — 7 de octubre de 2026

AD211 recupera el consumidor nominal de AD74 para la ficha propia ya
implementada. Es una migración nueva: AD74 permanece intacta. La lista
`deploy/principal/lista_sql_codext_ficha_personal_20261007.txt` ordena
AD211 y Personal22 para una base donde ambas capacidades estén ausentes.
Si Personal22 o el consumidor ya están instalados, no se ejecuta esa lista.

Ensayo en clon propio PostgreSQL 18.4, en disco y con límite de 2 GB:
UP únicos correctos, ACL y negativa de material nulo correctas. El núcleo
conserva sus metadatos y la inversión textual de la extensión reproduce la preimagen;
los 6.240 consumos, registros de auditoría y cabeza anteriores permanecen idénticos.
Dos revisiones independientes favorables del código `e8e767e84`.

La preimagen incluye AD195/AD196 y después AD178/AD177, AD190, AD197,
AD198, AD200, AD199, AD208 y AD207, con sus dependencias de las listas de main.
Otra extensión del núcleo o del CHECK obliga a medir de nuevo antes de instalar.
No se ha instalado en la principal ni demostrado acceso nominal desde navegador:
faltan identidad, perfil y origen V3 propios para esa comprobación.
Tampoco se ha medido el rendimiento de esta lectura autorizada.
La siguiente dependencia es reanclar AD175/Personal32 (#555) y después
AD180/Personal34 (#577), conservando sus consumidores y recibos existentes.


## Exportación propia: retoma de #555 — 7 de octubre de 2026

AD175 se reancla sobre AD211 con su permiso de exportación separado del de
consulta. Personal32 conserva el corte en los recibos nuevos y rechaza los
antiguos sin corte; no rellena ni modifica sus datos históricos. Se acotan
todos los argumentos antes del primer parseo de JSON.

En un clon PostgreSQL 18.4 se aplica AD175 y la prueba de preservación instala
Personal32 una sola vez. Un recibo ficticio, insertado directamente para esta
prueba, mantiene su contenido y los dos cortes NULL; su modificación falla.
Metadatos de la consulta e inversión textual del parche correctos. ACL y
tres negativas reales de tamaños inválidos correctas con un LOGIN técnico
exclusivo. Dos revisiones favorables del código `7c9e7ecb2`.

El recibo de prueba no procede de una lectura nominal: falta ensayar una
exportación con identidad, permiso, origen y material firmados propios,
recuperarla tras reinicio y medir su latencia. Orden: AD211/Personal22,
después la lista de AD175/Personal32, y finalmente AD180/Personal34.
No se instala nada en la principal por esta retoma.


## Historia propia: retoma de #577 — 7 de octubre de 2026

AD180 se reancla sobre AD175 y conserva el permiso propio de lectura de
revisiones. Personal34 consulta el periodo efectivo y el corte de conocimiento,
con un límite de 200 revisiones, procedencia y cobertura explícita. Los
argumentos se acotan antes de convertirlos a JSON.

Ensayo del código `53c69babe` en un clon nuevo PostgreSQL 18.4: UP180 y
Personal34 únicos, ACL, estructura, rechazo sin consumo y tres negativas
reales de tamaño correctos. Una dependencia ausente se probó dentro de una
transacción revertida: el diagnóstico dio su clave, actual=false y esperado=true.
Dos revisiones independientes favorables. No se reaplica la versión ensayada
en la copia anterior.

La prueba nominal preparada no se ha ejecutado: falta el material firmado
propio y su configuración de identidad, perfil y origen. Siguen pendientes
recorrido de navegador, revocación concurrente, recuperación tras reinicio y
latencia de la lectura autorizada. Orden final: AD211/Personal22,
AD175/Personal32 y AD180/Personal34, sin repetir SQL instalada ni DOWN.
