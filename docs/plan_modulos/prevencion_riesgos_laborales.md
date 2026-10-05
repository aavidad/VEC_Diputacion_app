# Plan de Prevención de riesgos laborales: 4 de octubre de 2026

VEC permitirá seguir evaluaciones preventivas y medidas de un centro: responsable,
plazo, evidencia de ejecución y revisión por el técnico competente. La persona
consultará instrucciones aplicables. La clínica conserva custodia sanitaria separada;
cualquier conclusión administrativa de aptitud tendrá acceso reservado y contrato
mínimo. Este encargo entrega documentación, sin código, SQL ni instalación.

Base inspeccionada: `origin/main@009472bd760e76cb2951433236711262f10648be`.
Requisitos principales: [ficha de PRL](../estudio_requisitos/ficha_prevencion_riesgos_laborales_2026-10-04.md),
PR1–PR12 y PRL-001–PRL-003 del [catálogo](../estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md#8-acción-social-prl-igualdad-y-relaciones-laborales).
Personal B conserva vínculo/servicios y su consumidor; Organización/RPT conserva
centros/puestos; K identidad/perfiles; L auditoría/documentos/firma comunes.
El plan no reasigna sus archivos ni abre una base clínica de RRHH.

## Fuentes y decisiones aplicables

La ficha conserva LPRL, especialmente art. 22, RD 39/1997, RD 843/2011 y el
procedimiento provincial de movilidad por salud publicado en BOP 08/09/2025.
Su apartado 5 exige informe SPSL favorable sin datos clínicos; el ámbito de
carrera/laboral fijo no se extiende automáticamente a otros vínculos. El índice
público de normativa no acredita composición ni circuito actuales del servicio.

La vigilancia es generalmente voluntaria, con excepciones documentadas según su
fuente y personal sanitario competente. Consentir el reconocimiento no concede
acceso a historia clínica. VEC no ofrecerá a RRHH una casilla para abrir diagnósticos,
pruebas o antecedentes. Aptitud/limitación también puede revelar salud; incluso el
intercambio mínimo administrativo necesita finalidad, campos, receptor y permiso
exactos. Provisión/Personal mantienen la decisión posterior sobre puesto o movilidad.

Se conserva [dudas.md, 131](../../dudas.md), desarrolla la 39 y coordina las 84/89 para
centro/evaluación iniciales, responsables, interfaz sanitaria y conservación/ENS.
Las garantías clínicas y el procedimiento publicados no necesitan otra pregunta.
Prevención, DPD/Archivo e Informática acuerdan los contratos no publicados; no se
inventa proveedor, excepción sanitaria, plazo clínico ni comunicación oficial.

## Inventario en la base inspeccionada

La búsqueda empezó en `codebase-memory-mcp`, proyecto
`home-alberto-Trabajo-VEC_Diputacion_app-.worktrees-codexe-original-autorizacion-20261003`.
Se cotejaron resultados con `git ls-tree` y lectura de archivos de esta base.
El índice tiene otra instantánea; una coincidencia de palabra no prueba un módulo.

| Pieza | Rutas actuales | Reutilización y límite |
| --- | --- | --- |
| Relación de Personal B | `internal/modules/personal/ports/{relacion_empleado,relacion_para_rpt,ficha_propia}.go`; `adapters/composicion/LECTOR_RELACION_RPT.md` | Contratos de vínculo y lectura nominal. El lector RPT usa autorización/auditoría comunes y no decide ocupación. PRL debe acordar su contrato mínimo con B; no copiar ficha ni usar la audiencia RPT como permiso preventivo. |
| Organización y RPT | `internal/modules/personal/ports/{estructura_organizativa,organizacion_historica,rpt_publica}.go`; `internal/vec/ports/catalogos_rpt_lectura.go`; `internal/vec/adapters/postgres/catalogos_rpt_lectura.go` | Fuentes/contratos de unidades, puestos y versiones, con cobertura y procedencia. Su dueño entrega centros/puestos pertinentes; no fabricar otro maestro ni inferir que vacío significa ausencia de riesgos. |
| Usuarios / contexto K | `internal/modules/usuarios/application/correos.go`; `internal/vec/ports/contexto_actor.go`; `internal/vec/adapters/contextoactor/postgres/`; `internal/vec/adapters/administracionperfiles/postgres/` | Acceso VEC y contexto común, separado de empleo y competencia sanitaria. K conserva perfil/emisor; falta consumidor PRL y acreditación de facultades exactas. |
| Auditoría L | `internal/vec/ports/auditoria_intento_nominal.go`; `auditoria_frontera_ruta_exacta.go`; `internal/vec/adapters/postgres/auditoria_intento_nominal.go` | Intentos nominales denegados/error con acuse y frontera técnica separada. Faltan acciones/rutas y configuración proceso/canal de PRL por su custodio; no registrar salud en claro. |
| Documentos y firma | `internal/vec/documentos/application/servicio.go`; `ports/{firma,custodia_firmado}.go`; `internal/vec/ports/documentos.go` | Registro/custodia/descarga y verificación comunes. Faltan tipos/políticas preventivos y consumidor; las piezas específicas CT no son emisor sanitario. Custodia general no habilita a importar clínica. |
| Formación | `internal/modules/formacion/application/preparar.go`; `adapters/jsonio/preparacion.go`; `web/static/portal-empleado/modulos/formacion/`; [plan](formacion.md) | Preparador y visor sintéticos integrados; `Preparar` no inscribe ni certifica. Reutilizar autoridad de Formación para formación preventiva; asistencia y certificado esperan fuente/contrato nominal admitidos. |
| Cronos | `internal/modules/cronos/ports/solicitud_permiso.go`; `application/solicitud_permiso.go`; `ports/justificacion.go` | Conserva permisos/ausencias por su propio circuito. No es fuente de evaluación médica ni debe recibir diagnóstico o cita clínica detallada. Una incidencia no activa aptitud. |
| Provisión | [plan de Provisión](provision.md); `internal/modules/personal/ports/relacion_para_rpt.go` | Plan y contrato de relación existentes; falta consumidor del informe mínimo de movilidad. Ninguna pieza decide movilidad por salud al conectarse a PRL. |

No hay backend `internal/modules/prevencion_riesgos_laborales/` o `internal/modules/prl/`,
ni pantalla propia, en esta base. Las rutas futuras no se consideran registradas.
Las piezas comunes integradas no acreditan instalación preventiva, proveedor sanitario
conectado o recorrido. Ninguna migración existente se reserva o reaplica por este plan.

## Huecos y propietarios

| Capacidad | Resultado pendiente | Dueño y dependencia |
| --- | --- | --- |
| PR1–PR2/PR10 | Evaluación por centro/puesto y seguimiento versionado de medidas. | A conserva PRL; Prevención gobierna fuente/metodología; Organización/RPT y B aportan contexto por puerto. |
| PR3 | Instrucciones propias y formación/resultado acreditados. | PRL mantiene instrucción; Formación mantiene acción/asistencia/certificación. |
| PR4–PR5 | CAE e investigación de accidente con bloques y destinatarios separados. | PRL mantiene hechos preventivos; contratistas/mutua/autoridad conservan recepción oficial. |
| PR6–PR8 | Derivación sanitaria y conclusión mínima para actuación administrativa o movilidad. | Servicio sanitario conserva clínica y dictamen; PRL solo contrato mínimo reservado; Personal/Provisión decide su acto. |
| PR9–PR11 | Lectura/descarga exactas, auditoría y exportación controlada. | A consume K/L y Documentos; permisos sanitarios no se heredan de prevención. |
| PR12 | Agregados con metodología y control de identificación indirecta. | Prevención/DPD aprueban población/agregación; no buscador de diagnósticos. |

Técnico preventivo, responsable de centro, persona, delegación/comité, receptor
administrativo y sanitario tienen competencias distintas. K provisiona perfiles
fijos por huella/CAS, un perfil activo y concesión positiva por acción, recurso,
entidad/centro/puesto/expediente, finalidad, campos y vigencia. Soporte/jefatura/
administración/candidato no son universales. El sanitario conserva su canal y
acreditación profesional; el autoservicio no abre gestión ni clínica ajena. Exposición
exterior necesita política propia de capacidad, sin heredar la del portal público.

Toda consulta, descarga, cambio, exportación e intercambio usa auditoría común
nominal: identidad acreditada, perfil, acción, recurso opaco, finalidad, instante,
permitido/denegado/error, proceso/canal, correlación y versión. Lectura y auditoría
se confirman antes de revelar datos; consumo, efecto, estado e historia en la misma
transacción. Denegaciones/errores se registran al cerrar el intento original con
acuse común; sin identidad se aplica frontera técnica sin actor inventado. Fallo de
auditoría impide datos. No se registra aptitud en claro, diagnóstico, familia o
contenido documental. Actuaciones sanitarias por VEC/contrato auditado dejan traza
común mínima; el proveedor mantiene su auditoría clínica externa, sin replicarla.

## Minitareas y salidas por PR

Dirección confirma base actual y archivos exclusivos por corte. Ubicaciones nuevas
son previsión. Montaje, manifiestos y contratos K/L/B/Organización esperan turno de
sus dueños; un puerto se entrega con consumidor útil, sin maqueta sustitutiva.

| Corte y base | Salida usable por PR | Archivos propios previstos | Dependencias | Horas |
| --- | --- | --- | --- | ---: |
| P00 · inventario | Revalidar deuda y contratos, sin repetir Personal/Formación. | Este plan durante su turno. | SHA/FIN vigentes. | 1–2 |
| P01 · fuente | Evaluación/centro inicial, estados, responsables, evidencia y metodología admitidos. | Contrato documentado preventivo. | Duda131; Prevención/Sistemas; minimización y RAT. | 3–5 |
| P02 · B1 | Consumidor nominal con auditoría que permite/deniega centro/acción exactos. | Nuevos `prevencion_riesgos_laborales/ports/{autorizacion,auditoria}.go`; consumidor. | P01; ABI K/L, perfiles fijos por huella/CAS y proceso/canal. | 6–10 |
| P03 · B2/B3 | Correspondencia centro/puesto/relación/periodo con cobertura y conflicto explicados. | Puertos/consumidores mínimos del módulo. | B y Organización/RPT producen sus datos; P02 antes de lectura personal. DTO puede acordarse antes. | 3–5 |
| P04 · PR1 | Puerto, fuente autorizada y consumidor focal de evaluación/versiones. | Dominio/aplicación/adaptador propios. | P01–P03; no elaborar dictamen desde cuestionario libre. | 4–6 |
| P05 · PR1/PR9 | Vista por centro/puesto con estados claros y documentos accesibles según permiso. | Entrada/cliente/vista propios y catálogos ES/EN. | P04; montaje por custodio; revisión de usabilidad. | 4–6 |
| P06 · PR2 | Planificar medidas: aprobación, responsable, plazo y evidencia separados. | Casos de uso/formulario preventivos. | P02/P04/P05; política y competencia admitidas. | 5–8 |
| P07 · SQL necesario | Borrador reservado de versiones/historia/idempotencia preventivas. | SQL y adaptador propios del módulo. | P01–P06; no guardar clínica ni duplicar organización. | 4–6 |
| P08 · ensayo | Clon principal, revisiones exactas y kit; instalación por dirección. | Pruebas y kit propios. | P07; preimagen, número/orden causal reservados. | 4–8 |
| P09 · PR2/PR10 | Responsable aporta ejecución/evidencia; técnico revisa eficacia y rectifica con historia. | Aplicación/cliente/formularios propios. | P06/P08; la marca de ejecutada no equivale a revisión eficaz. | 5–8 |
| P10 · B5/PR3 | Consulta de instrucciones y descarga preventiva independiente/auditada. | Puerto/consumidor documental y vista propios. | P02/P05/P09; Documentos y destinatarios pertinentes. | 4–6 |
| P11 · recorrido inicial | Técnico/responsable/persona→medida/evidencia/revisión; negativos y reinicio. | Pruebas focales/manual del módulo. | P05–P10 montados; sin clínica ni aptitud automática. | 4–6 |
| P12 · PR3 posterior | Consultar asistencia/certificación preventiva de Formación, conservando pendientes. | Puerto/consumidor y vista propios. | Formación nominal admite fuente/contrato; no inferir asistencia de acceso. | 4–6 |
| P13 · PR4 | CAE: intercambio mínimo de riesgos/instrucciones por actividad y acuse. | Puerto/consumidor CAE y estado propio. | Ampliación encargada; contratista/destino/representación y política autorizados. | 6–10 |
| P14 · PR5 | Investigación preventiva restringida y preparación/acuse de comunicación cuando admitida. | Expediente/caso de uso/consumidor preventivos. | Fuente/circuito de accidente y mutua/autoridad; separar asistencia sanitaria. | 6–10 |
| P15 · PR6/PR7 | Derivación sanitaria y recepción reservada del dictamen administrativo mínimo con vigencia. | Puerto/consumidor de conclusión propia; sin clínica. | Servicio sanitario/DPD/L admiten contrato y acreditación; clínico/consentimiento quedan en sanitario. | 6–10 |
| P16 · PR8 | Entrega mínima a Personal/Provisión y recuperación del recibo de su actuación. | Consumidor de movilidad/medidas del módulo. | P15; procedimiento de 2025 y vínculo aplicable; B/Provisión conserva decisión. | 4–7 |
| P17 · PR12 | Agregados preventivos revisados por periodo/población, sin identificar indirectamente. | Puerto/caso de uso/informe propios. | Metodología, supresión y finalidad aprobadas; sin umbral inventado. | 4–6 |
| P18 · entrega ampliada | Recorrido de contratos admitidos, auditoría/negativos y manual de fronteras. | Pruebas focales y manual propios. | P12–P17; cada capacidad exige montaje/fuente propios. | 4–6 |

P00–P11 cierran riesgo/medidas del centro inicial. P12–P18 son ampliación propuesta;
P15 integra solo derivación y dictamen mínimo, sin presupuestar sistema clínico ni
sustitución del servicio sanitario. Si no hay interfaz sanitaria, la derivación oficial
queda comprobada y la recepción pendiente; no se fabrica aptitud favorable.

P07/P08 son condicionales a estado durable propio: eliminarlos y recalcular si la
fuente ya conserva el estado admitido. Cualquier migración reserva número/orden causal
en `RESERVAS_MIGRACIONES.md` fuera de Git, conserva preimagen, queda en borrador
hasta ensayo en clon principal y dos revisiones exactas. MCP `postgres-clon-local`
solo lectura/EXPLAIN; dirección instala. No reaplicar historia, DOWN destructivo ni
SQL en cidonia por este carril. Nuevos efectos posteriores revalidan su persistencia.

## Dependencias y paralelismo

P00/P01 preceden a las bases nominales; P02 puede solaparse con acuerdos de DTO P03,
cuya lectura espera permisos. Dos equipos pueden separar archivos de evaluación/vista (P04/P05) y
medidas/evidencia/persistencia (P06–P08), preparando DTO, documentos y pruebas
mientras esperan. La entrega conserva la cadena P04→P05→P06→P07→P08;
ese reparto no acredita un solape de todas sus horas.
P09 reúne esas bases; instrucciones/documentos P10 pueden avanzar por archivos propios.
P11 espera montaje/instalación. P12, P13/P14 y P15/P16 son carriles separables con
fuentes admitidas; cada contrato/vista tiene un escritor. La organización sanitaria,
K/L/B y Formación conservan sus responsables y no son esfuerzo incluido aquí.

## Configuración y estimación

Organización preventiva, centro/puesto, riesgos/medidas, responsables, revisiones,
evidencias, protocolos, perfiles y conservación se gobiernan con norma/acto, órgano,
versión, huella y vigencia. El sanitario conserva protocolos/criterios de vigilancia;
RRHH no configura diagnósticos o excepciones clínicas individuales. Textos/categorías
usan datos por idioma. RAT, habilitación por finalidad, información, custodia física
y lógica separada, riesgos/EIPD, bloqueo/archivo y expurgo requieren DPD/Archivo y
servicio sanitario antes de datos reales. La clínica no entra en búsqueda general,
copias o índices de Persona/Personal/Cronos/Nóminas. Historia no fija retención ilimitada.

P00–P11 suman **47–76 h**, **6–10 jornadas de un equipo** de ocho horas; con dos
equipos se conserva la cota de **6–10 jornadas** si predomina esa secuencia. P12–P18 añaden **34–55 h**: conjunto **81–131 h**,
**11–17 jornadas de un equipo** o **9–14 jornadas con dos equipos**. Jornadas
completas redondeadas, con integración/revisión y dependencias listas. Se reestima
cada ampliación tras el acuerdo de fuente; espera externa y atención sanitaria no
son duración técnica ni fecha prometida.

| Trabajo externo, fuera del total | Dedicación orientativa | Condición |
| --- | ---: | --- |
| Servicio de Prevención / responsable de centro | 6–10 h | Resolver la 131, evaluación/método/evidencia, competencias y recorrido inicial. |
| Sistemas / fuente preventiva | 6–12 h | Interfaz, identificadores, entorno y canales; sin presunción de proveedor. |
| K/L, Documentos, Personal B / Organización | Estimación de sus dueños | Contexto/perfiles, auditoría/documentos y fuente laboral/puesto; aquí consumidores propios. |
| DPD / Archivo / sanitario | 6–12 h | Tratamiento, fronteras, custodia/conservación y contrato mínimo de aptitud; aprobación sin plazo comprometido. |
| Formación / Provisión / contratista / mutua / autoridades | Estimación por corte | Asistencia, movilidad y comunicaciones conservan sus responsables; clínica fuera del total. |

## Comprobación y entrega

La PR documental verifica inventario por SHA, enlaces locales, sumas y
`git diff --check`; revisión independiente documental y una CI de PR. No ejecuta
Go/SQL/navegador ni acredita instalación preventiva o conformidad sanitaria.

Implementación con `programar-backend-vec`, `persistir-autorizar-vec`,
`programar-interfaz-vec`, `probar-recorridos-vec` según delta; pantallas con
`usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec` e
`impeccable`/`VEC-PRIORIDAD.md` antes de programar y revisión independiente de
usabilidad. Textos con `humanizer`/`VEC-USO.md`; SQL con
`revisar-sql-vec`/`ensayar-sql`; `security-audit` focal y Semgrep local en cambios
sensibles, sin enviar código fuera. gopls e índice para Go/código.
`revisar-cambios-vec`, `documentar-entregar-vec` y `pr-vec` sobre candidata exacta;
dirección integra y despliega.

Aceptación: ámbito/centro/persona ajenos, canal incorrecto, caducidad/revocación,
descarga independiente, fuente/auditoría caída sin datos, estados separados,
evidencia/revisión/rectificación conservadas y reintento/reinicio sin duplicados.
PostgreSQL real en efectos y Chrome del sistema PC/móvil con teclado/foco e i18n.
El contrato sanitario prueba que diagnóstico/pruebas no alcanzan RRHH, búsqueda,
logs o agregador; aptitud ausente queda pendiente. No repetir campañas integradas
sin delta/fallo ni usar una medida como cambio automático de puesto, jornada o pago.
Las dudas permiten contratos/documentos independientes, sin inventar clínica o competencia.
