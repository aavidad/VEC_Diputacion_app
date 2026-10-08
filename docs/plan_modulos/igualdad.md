# Plan de Igualdad: 4 de octubre de 2026

**Aparcado hasta cerrar Bolsa y CT (orden de Alberto, 07/10/2026).** Los objetivos vigentes están en [OBJETIVOS.md](OBJETIVOS.md).

La unidad competente seguirá medidas de un plan de empleo acreditado, responsables,
evidencias y evaluación anual. Los indicadores se consumirán desde sus propietarios
con metodología y revisión de calidad; publicación y remisión conservarán controles
y evidencias propios. El panel general no alojará denuncias ni pruebas de acoso.
Este encargo entrega documentación, sin código, SQL ni instalación.

Base inspeccionada: `origin/main@009472bd760e76cb2951433236711262f10648be`.
Requisitos principales: [ficha de Igualdad](../estudio_requisitos/ficha_igualdad_2026-10-04.md),
IG1–IG12 y IGU-001–IGU-002 del [catálogo](../estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md#8-acción-social-prl-igualdad-y-relaciones-laborales).
Personal B conserva población/relaciones/servicios, cada módulo sus hechos, K
identidad/perfiles y L auditoría/documentos/firma comunes. El plan no reasigna sus
archivos ni sustituye el seguimiento general.

## Fuentes y decisiones aplicables

La ficha recoge el Plan provincial aprobado el 27/01/2022 y su duración de cuatro
años. Se conserva como antecedente: el PDF publicado no acredita prórroga ni acto
operativo posterior a enero de 2026. Un anuncio sobre el Consejo de Igualdad tampoco
prueba otro plan de empleo. Mientras se acredita el acto posterior pueden avanzar
inventario, contratos y estudio bajo legislación actual; no se activan nuevas
decisiones atribuyendo vigencia al antecedente.

La DA 7.ª del TREBEP y el registro público de planes de Administraciones tienen ámbito
propio. El registro retributivo del RD 902/2020 corresponde a relaciones laborales,
con sus modalidades de acceso; no se extiende automáticamente a funcionarios ni
permite ver salarios nominales ajenos. La ficha diferencia plan de empleo, política
institucional, registro retributivo y protocolo reservado de acoso. Igualdad revisa
resultados/reglas y entrega recomendaciones aprobadas; no altera admisión, baremo,
puesto o salario ni declara automáticamente discriminación.

Las normas oficiales y los artículos de igualdad/no discriminación/protección de
datos constan en la ficha, consultados el 04/10/2026. Cada regla activada conservará
acto/versión/fecha de efectos. Se mantiene [dudas.md, 132](../../dudas.md), desarrolla
la 39 y coordina la 84: acto operativo, responsables/comisión, metodología/población,
circuito de remisión y publicación. No se vuelve a preguntar duración de 2022 ni ámbito
laboral ya publicados. Una obligación de remitir no acredita API o acuse recibidos.

## Inventario en la base inspeccionada

La búsqueda empezó en `codebase-memory-mcp`, proyecto
`home-alberto-Trabajo-VEC_Diputacion_app-.worktrees-codexe-original-autorizacion-20261003`.
Se cotejaron resultados con `git ls-tree` y lectura de archivos en esta base. El
índice anterior no acredita por sí solo rutas actuales o capacidad nominal.

| Pieza | Rutas actuales | Reutilización y límite |
| --- | --- | --- |
| Personal B / Organización | `internal/modules/personal/ports/{ficha_propia,historia_relaciones_propia,organizacion_historica,relacion_para_rpt}.go`; `internal/vec/ports/catalogos_rpt_lectura.go` | Contratos autorizados y versiones/cobertura del vínculo/organización. B y Organización entregarán población mínima por régimen/entidad/periodo; no se obtiene ficha integral para construir indicadores. |
| Usuarios / contexto K | `internal/modules/usuarios/application/correos.go`; `internal/vec/ports/contexto_actor.go`; `internal/vec/adapters/contextoactor/postgres/`; `internal/vec/adapters/administracionperfiles/postgres/` | Identidad/contacto y perfiles comunes, separados de representación laboral o mandato de comisión. Falta consumidor de Igualdad y concesiones fijas; no crear login ni permisos por solicitud. |
| Auditoría L | `internal/vec/ports/auditoria_intento_nominal.go`; `auditoria_frontera_ruta_exacta.go`; `internal/vec/adapters/postgres/auditoria_intento_nominal.go` | Denegado/error nominal tras cerrar el intento original, con acuse, y frontera técnica común. Faltan acciones/rutas, proceso/canal del módulo por su custodio. Un informe de seguimiento no sustituye auditoría de accesos. |
| Documentos y firma | `internal/vec/documentos/application/servicio.go`; `ports/{firma,custodia_firmado}.go`; `internal/vec/ports/documentos.go` | Registro/custodia/descarga y verificación comunes. Tipos/políticas y consumidor de plan/evaluación están pendientes; las piezas específicas CT no aprueban un plan ni acreditan remisión al registro público. |
| Datos estadísticos | `web/static/textos/{es,en}/datos-personales.json`; `internal/modules/personal/ports/ficha_propia.go` | Hay textos de finalidad estadística; no prueban un motor de indicadores, fundamento ni disponibilidad de categorías. No inferir sexo, identidad, discapacidad u otras categorías desde nombre o permisos. |
| Selección / Bolsa / Provisión | `internal/modules/seleccion/ports/`; `internal/modules/bolsa/`; [plan de Provisión](provision.md) | Mantienen bases y decisiones propias. Falta proyección agregada aprobada para Igualdad y receptor de recomendaciones. `internal/modules/bolsa/domain/calculoexperiencia/puntuacion_igualdad.go` trata igualdad de cálculo; no es módulo de política de igualdad. |
| Formación / Cronos | `internal/modules/formacion/application/preparar.go`; `adapters/jsonio/preparacion.go`; `internal/modules/cronos/ports/` | Formación tiene preparación sintética, sin asistencia/certificación corporativas; Cronos mantiene sus hechos autorizados. No hay por su existencia agregados de participación o conciliación admitidos para este fin. |
| Nóminas | `web/static/portal-empleado/modulos/nominas/vista.js`; `internal/modules/personal/domain/payroll.go`; [plan publicado](https://github.com/aavidad/VEC_Diputacion_app/blob/5c2f146bd15761d225cda7297b4fc7b8f34034a2/docs/plan_modulos/nominas.md) | Vista inyectable sin fuente nominal y sumador preliminar de conceptos. No constituyen fuente salarial ni registro retributivo fiable. Nóminas conservará percepciones individuales y emitirá agregados por contrato propio. |

No hay backend `internal/modules/igualdad/` ni pantalla propia en esta base. Las
ubicaciones nuevas son previsión; una coincidencia de nombre o un catálogo visible
no acredita registro/montaje. Las piezas integradas no acreditan instalación, fuentes
estadísticas conectadas o recorrido de Igualdad. No se reservan ni reaplican sus SQL.

## Huecos y propietarios

| Capacidad | Resultado pendiente | Dueño y dependencia |
| --- | --- | --- |
| IG1–IG3/IG11 | Acto/versiones, medidas/evidencias y evaluación anual con correcciones enlazadas. | A conserva Igualdad; unidad/RRHH/comisión/órgano acreditan competencia y acto; L/Documentos conserva originales. |
| IG4/IG12 | Indicadores reproducibles por población/periodo, ausencia/no comparable/validado. | Cada propietario emite contrato mínimo; Igualdad conserva metodología, revisión y resultado. |
| IG5 | Revisión e informe aprobado a Selección/Provisión/Formación. | Igualdad informa; sus dueños aplican cambios por su procedimiento, sin alteración automática. |
| IG6 | Registro retributivo de población laboral y acceso legalmente aplicable. | Nóminas prepara agregados; B delimita régimen/población; representación exige mandato vigente en autoridad común. |
| IG7 | Resumen publicable controlado y evidencia de remisión. | Unidad/órgano aprueban; publicación/registro externo mantienen sus acuses; DPD controla reidentificación. |
| IG8–IG10 | Información/derivación reservadas y permisos de consulta/edición/descarga independientes. | El circuito competente de acoso conserva expediente; A consume K/L sin trasladar víctimas/pruebas al panel. |

K provisiona perfiles fijos por huella/CAS, un perfil activo y concesión positiva
exacta por acción, recurso, entidad/plan/medida/población/periodo, finalidad, campos y
vigencia. Unidad, responsable, comisión, órgano, gestor de Nóminas, representación,
empleado y auditoría tienen capacidades separadas. El cargo, menú, cuenta o perfil
candidato no conceden acceso. Gestión interna, proyección pública aprobada y
modalidades de acceso del empleado quedan separadas; exposición exterior requiere
política propia, sin abrir salarios nominales o expedientes reservados.

Todas las lecturas, descargas, ediciones, aprobaciones, publicaciones y remisiones
usan auditoría común nominal: actor acreditado, perfil, acción, recurso opaco,
finalidad, instante, permitido/denegado/error, proceso/canal, correlación y versión.
Lectura y auditoría se confirman antes de devolver datos; consumo de autorización,
efecto, estado e historia en la misma transacción. Denegado/error se registra al
cerrar el intento original con el contrato común; sin identidad, frontera técnica
sin actor inventado. Fallo de auditoría impide revelar datos. Víctimas, denuncias,
salarios nominales, familia, salud y documentos completos no van a traza/logs.

## Minitareas y salidas por PR

Dirección fija SHA vigente, propietario y archivos exclusivos por corte. Las rutas
nuevas son previsión. Montaje/manifiestos y contratos K/L/B de otros propietarios
requieren su turno. Cada puerto tendrá consumidor útil sin inventar fuente.

| Corte y base | Salida usable por PR | Archivos propios previstos | Dependencias | Horas |
| --- | --- | --- | --- | ---: |
| I00 · inventario | Revalidar piezas, deuda y estado de acto/plan. | Este plan durante su turno. | SHA/FIN vigentes. | 1–2 |
| I01 · fuente | Versión/estado, ámbito, competencia, medidas y evaluación admitidos. | Contrato documentado del módulo. | Duda 132; acto auténtico posterior o lectura histórica de 2022; fuentes oficiales. | 3–5 |
| I02 · B1 | Consumidor nominal y auditoría común por plan/medida/acción exactos. | Nuevos `igualdad/ports/{autorizacion,auditoria}.go`; consumidor. | I01; ABI K/L y perfiles fijos por huella/CAS; proceso/canal. | 6–10 |
| I03 · B2/B3 | Población/entidad/régimen/periodo y responsables por puertos mínimos. | Puertos/consumidores de población del módulo. | B/Organización y K acreditan datos/mandatos; I02 antes de lectura. DTO puede acordarse antes. | 3–5 |
| I04 · IG1 | Puerto/fuente/consumidor del plan y versiones con vigencia/estado explícitos. | Dominio/aplicación/adaptador propios. | I01–I03; 2022 se conserva histórico si falta acto posterior. | 4–6 |
| I05 · IG1/IG9 | Vista del plan/medidas y documentos autorizados con ES/EN. | Entrada/cliente/vista propios; catálogos ES/EN. | I04; montaje por custodio y revisión de usabilidad. | 4–6 |
| I06 · IG2 | Responsable aporta avance/evidencia; ejecución y evaluación separadas. | Casos de uso/formulario propios. | I02/I05; acto/medidas/responsables vigentes, sin activar decisiones sobre antecedente. | 5–8 |
| I07 · SQL necesario | Borrador reservado de versión/historia/idempotencia del seguimiento. | SQL/adaptador propios. | I01–I06; no duplicar fuentes ni almacenar denuncias. | 4–6 |
| I08 · ensayo | Clon principal, revisiones exactas y kit; instalación por dirección. | Pruebas/kit propios. | I07; preimagen, número/orden causal reservados. | 4–8 |
| I09 · IG3/IG11 | Evaluación anual, revisión competente y corrección enlazada con original. | Aplicación/formularios y consumidor documental propios. | I06/I08; metodología/órgano y documento admitidos. | 5–8 |
| I10 · B5/IG9 | Consulta y descarga independiente de originales autorizados del plan/evaluación. | Consumidor documental y cliente propios. | I02/I09; Documentos/firma según acto, sin deducir firma de login. | 4–6 |
| I11 · recorrido inicial | Responsable/comisión→medida/evidencia/evaluación; negativos/reinicio/manual. | Pruebas focales/manual propios. | I05–I10 montados, acto acreditado y revisiones pertinentes. | 4–6 |
| I12 · IG4/IG5 | Indicadores de una fuente/población aprobadas y reporte a su dueño. | Puerto/consumidor e informe propios. | Ampliación encargada; B/Selección/Provisión/Formación/Cronos entregan hechos mínimos por su turno. | 6–10 |
| I13 · IG12 | Calidad/metodología reproducibles; ausente/no comparable/validado y revisión humana. | Casos de uso/datos metodológicos propios. | I12; fuente/versiones/denominador y control DPD aprobados. | 4–7 |
| I14 · IG6 | Consultar agregado retributivo laboral con acceso según modalidad/mandato. | Puerto/consumidor y vista propios. | Nóminas emite agregado, B delimita población; K acredita representación; no usar sumador preliminar. | 6–10 |
| I15 · IG7 | Proyección publicable aprobada con control de celdas/diferencias/reidentificación. | Informe/proyección y consumidor de publicación propios. | I13/I14; política aprobada, sin umbral universal inventado. | 4–6 |
| I16 · IG1/IG7 | Preparación/remisión al registro de planes de Administraciones y conciliación de acuse. | Puerto/adaptador/consumidor y estado propios. | Órgano/destino/formato/operador admitidos; L/Registro/firma cuando procedan. | 5–8 |
| I17 · IG8 | Información y derivación al circuito competente reservado, sin denuncias en panel. | Cliente/textos/contrato mínimo de derivación propios. | Protocolo/estado vigente y receptor de acoso/Canal/Disciplina según competencia; no acceso por defecto. | 3–5 |
| I18 · entrega ampliada | Recorridos de agregados/accesos/remisión/publicación y manual de límites. | Pruebas focales/manual propios. | I12–I17; acuses y revisión de calidad/reidentificación acreditados. | 4–6 |

I00–I11 cubren seguimiento del plan acreditado. Sin acto operativo, I04/I05 pueden
entregar lectura histórica y el estudio preparatorio; I06/I09 esperan la versión
habilitante para efectos nuevos. I12–I18 amplían indicadores/remisión/derivación:
no crean cálculo de Nóminas, sistema clínico o expediente de acoso. Los productores
de agregados se estiman por sus dueños; no se cuentan como hechos ya disponibles.

I07/I08 solo proceden si la fuente no conserva todo el estado admitido; eliminarlos
y recalcular cuando se reutilice. Toda migración futura reserva número/orden causal
en `RESERVAS_MIGRACIONES.md` fuera de Git; preimagen, borrador, ensayo en clon de
principal y dos revisiones exactas antes de instalar por dirección. MCP
`postgres-clon-local` solo lectura/EXPLAIN; nunca reaplicar historia, DOWN destructivo
ni SQL en cidonia por este carril. La ampliación revalida su estado durable.

## Dependencias y paralelismo

I00/I01 preceden a fuentes/montaje. I02 se solapa con acuerdo de DTO I03, cuya lectura
espera permiso. Dos equipos separan archivos de plan/vista (I04/I05) y seguimiento/evidencia/
persistencia (I06–I08), preparando contratos, documentación y pruebas. La entrega
conserva la cadena I04→I05→I06→I07→I08; no se presupone solape de todas sus horas. I09/I10
reúnen esas bases; I11 espera montaje/instalación. Después se separan fuentes/
metodología (I12/I13) y retribución/remisión (I14/I16), por contratos y población
aprobados. I15 espera control DPD y revisión humana; I17 puede documentarse antes.
Un escritor por vista, idioma o contrato; K/L/B/Nóminas mantienen colas y esfuerzo
propios. La espera del acto posterior no detiene estos acuerdos independientes.

## Configuración y estimación

Plan/acto, población, medidas, responsables, calendario, metodología/indicadores,
fuentes, agregación/supresión, evaluación, remisión, perfiles y conservación son
configuración gobernada: norma/acto, órgano, versión, huella y vigencia desde/hasta.
Cada informe conserva definición/entradas por periodo. Textos/categorías usarán datos
por idioma. RAT, fundamento/finalidad, datos mínimos, acceso especializado a microdatos
si se justifica, riesgos/EIPD, bloqueo/archivo y expurgo esperan DPD/Archivo antes de
datos reales. Seudonimización sigue siendo dato personal; historia no implica
retención ilimitada. No inferir categorías protegidas ni publicar celdas pequeñas sin
política, incluida comparación entre informes y periodos.

I00–I11 suman **47–76 h**, **6–10 jornadas de un equipo** de ocho horas; con dos
equipos se conserva **6–10 jornadas** cuando predomina esa secuencia. I12–I18 añaden **32–52 h**: conjunto **79–128 h**,
**10–16 jornadas de un equipo** o **8–13 jornadas con dos equipos**. Días completos
redondeados con margen de integración/revisión y dependencias listas. Al fijar I01 y
las fuentes de agregados se revisarán intervalos; la espera externa no tiene fecha
comprometida ni se incluye como esfuerzo de implementación.

| Trabajo externo, fuera del total | Dedicación orientativa | Condición |
| --- | ---: | --- |
| Unidad de Igualdad / RRHH / comisión | 6–10 h | Resolver la 132, acto/mandato, medidas, evaluación y recorrido. |
| Sistemas / registro/publicación corporativos | 6–12 h | Interfaz, entorno, destino/formato y acuses; no se presupone API. |
| K/L, Documentos y Personal B / Organización | Estimación de sus dueños | Identidad/perfiles, auditoría/documentos/firma, población y datos de servicio; aquí consumidores. |
| Dueños de Selección / Provisión / Formación / Cronos / Nóminas | Estimación por contrato | Emitir agregados mínimos con finalidad/metodología; fuera del esfuerzo del módulo. |
| DPD / Archivo | 4–8 h | Microdatos si necesarios, reidentificación, acceso/publicidad y conservación; aprobación sin plazo comprometido. |
| Órgano aprobador / responsable de remisión | 3–6 h | Actos, publicación y conciliación de recepción; no se equipara fichero guardado con remisión. |

## Comprobación y entrega

La PR documental verifica inventario por SHA, enlaces locales, sumas y
`git diff --check`; revisión independiente documental y una CI de PR. No ejecuta
Go/SQL/navegador ni acredita vigencia del plan, remisión o instalación nuevas.

Implementación con `programar-backend-vec`, `persistir-autorizar-vec`,
`programar-interfaz-vec`, `probar-recorridos-vec` según delta. Pantallas con
`usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec` e
`impeccable`/`VEC-PRIORIDAD.md` antes de programar y revisión independiente de
usabilidad. Textos con `humanizer`/`VEC-USO.md`; SQL con
`revisar-sql-vec`/`ensayar-sql`; `security-audit` focal y Semgrep local en cambios
sensibles, sin enviar código fuera. gopls e índice para Go/código.
`revisar-cambios-vec`, `documentar-entregar-vec` y `pr-vec` sobre candidata exacta;
dirección integra/despliega.

Aceptar exige ámbito/medida/periodo ajenos, canal incorrecto, caducidad/revocación,
descarga separada, fuente/auditoría caída sin datos, corrección/historia/recibo y
reintento/reinicio sin duplicados. PostgreSQL real en efectos; Chrome del sistema
PC/móvil con teclado/foco e i18n. Calidad/reidentificación se revisa antes de publicar;
el registro retributivo prueba modalidades laborales sin salarios individuales ajenos.
La derivación no expone víctimas/pruebas y la remisión necesita acuse del destino.
No repetir campañas integradas sin delta/fallo. Las dudas permiten contratos y estudio
independientes; no prorrogan el Plan de 2022 ni convierten un agregado en conclusión jurídica.
