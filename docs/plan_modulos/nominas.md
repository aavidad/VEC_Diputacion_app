# Plan de Nóminas: 4 de octubre de 2026

La persona consultará sus recibos originales por entidad, relación y periodo, con
lectura y descarga autorizadas por separado. Podrá pedir revisión de una discrepancia.
El cálculo, cierre, cotización, fiscalidad y pago mantienen sus autoridades y quedan
para una ampliación posterior. Este encargo entrega documentación, sin código, SQL
ni instalación.

Base inspeccionada: `origin/main@009472bd760e76cb2951433236711262f10648be`.
Requisitos principales: [ficha de Nóminas](../estudio_requisitos/ficha_nominas_2026-10-04.md),
N1–N12 y NOM-001–NOM-008 del [catálogo](../estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md#7-nómina-y-gestión-económica).
Personal B mantiene relaciones y servicios; K identidad y perfiles; L las autoridades
comunes. El plan no reasigna sus archivos ni modifica el seguimiento general.

## Fuentes y decisiones aplicables

La ficha conserva las fuentes oficiales consultadas el 04/10/2026: TREBEP 21–30,
LBRL 90/93, RD 861/1986, ET 26–29, LGSS 139–142, Ley 35/2006 art. 99,
Ley 39/2015 y protección de datos. Desarrollar la consulta no fija fórmulas,
retenciones, tablas salariales ni un convenio. Cada regla futura necesitará acto,
versión, colectivo, periodo y aprobación competente. La fuente actual de recibos y
su interfaz siguen pendientes de confirmación; GINPIX no se presume su proveedor.

Se conserva [dudas.md, 129](../../dudas.md), que desarrolla la 39 y las 28–29;
Cronos y Dietas mantienen las 21/26 y conservación la 84. La 129 pide fuente,
entidades/relaciones/periodos iniciales y circuito de discrepancia. No se vuelve a
preguntar legislación publicada ni se inventa respuesta interna. Una URL corporativa
no acredita API, custodia, autorización o permiso para copiar recibos.

## Inventario en la base inspeccionada

La búsqueda empezó en `codebase-memory-mcp`, proyecto
`home-alberto-Trabajo-VEC_Diputacion_app-.worktrees-codexe-original-autorizacion-20261003`.
Sus resultados se cotejaron mediante `git ls-tree` y lectura de archivos de esta base;
el índice corresponde a otra instantánea y no acredita el estado actual por sí solo.

| Pieza | Rutas actuales | Reutilización y límite |
| --- | --- | --- |
| Vista de recibos | `web/static/portal-empleado/modulos/nominas/vista.js`; `nominas.css`; `nominas.test.mjs` | Ya admite una fuente inyectada con `consultar`/`descargar`, periodos y estados de carga, vacío, denegación y error. Sin fuente muestra `no_configurado`. Reutilizarla; faltan cliente nominal, fuente admitida y montaje real de esa lectura. |
| Presentación e idioma | `web/static/portal-empleado/modulos/nominas/datos-presentacion.js`; `i18n.js` | Atlas e importes de ejemplo son sintéticos; no son recibos ni derechos. Los mensajes están embebidos en JavaScript: deuda de migrarlos a catálogos de datos ES/EN antes de la integración nominal. No conectar el atlas como fuente económica. |
| Borrador económico histórico | `internal/modules/personal/domain/payroll.go`; `payroll_test.go` | `BuildPayrollDraft` suma conceptos aportados y separa deducciones/bases marcadas. No contiene reglas completas por colectivo, cierre, autorización o conciliación. Sigue siendo preparación de Personal; no presentarlo como motor oficial ni ampliarlo para alojar Nóminas. |
| Personal B | `internal/modules/personal/ports/{ficha_propia,historia_relaciones_propia,relacion_empleado}.go`; `adapters/composicion/FICHA_PROPIA.md`; `adapters/postgres/ficha_propia.go` | Referencias y consulta propia con autorización común. Reutilizar contratos mínimos mediante B; la ficha no concede permiso sobre recibos. Falta el contrato de relación/entidad/periodo para Nóminas. |
| Usuarios / contexto K | `internal/modules/usuarios/application/correos.go`; `internal/vec/ports/contexto_actor.go`; `internal/vec/adapters/contextoactor/postgres/`; `internal/vec/adapters/administracionperfiles/postgres/` | Cuenta/contacto y contexto nominal separados del vínculo laboral. K provee identidad, emisor y perfiles fijos; falta consumidor de Nóminas. No crear login, maestro de personas o permisos por petición. |
| Auditoría L | `internal/vec/ports/auditoria_intento_nominal.go`; `auditoria_frontera_ruta_exacta.go`; `internal/vec/adapters/postgres/auditoria_intento_nominal.go` | Registro común de denegado/error tras cerrar el intento original, con acuse. La frontera sin identidad tiene contrato propio. Falta configurar proceso/canal y las acciones/rutas exactas de Nóminas por sus custodios. |
| Documentos y firma | `internal/vec/documentos/application/servicio.go`; `ports/{firma,custodia_firmado}.go`; `adapters/postgres/custodia_firmado.go`; `internal/vec/ports/documentos.go` | Registro externo, custodia y descarga común; verificación de firma separada. Sus piezas específicas de CT no autorizan nómina. Falta tipo, política, custodio y consumidor del original; una referencia externa no acredita custodia de bytes. |
| Tiempo y gastos | `internal/modules/cronos/ports/`; `internal/modules/dietas/ports/`; `internal/modules/contrataciontemporal/` | Sus propietarios mantienen incidencias, liquidaciones y contratación. No existe por estas piezas un conector económico aprobado para Nóminas; una propuesta o aprobación no prueba alta efectiva ni pago. |

No hay paquete backend `internal/modules/nominas/` en esta base. La vista y el
borrador existentes tampoco acreditan lectura nominal de recibos. Los adaptadores
comunes y migraciones integrados son código conservado; este inventario no acredita
instalación, activación en principal ni recorrido de Nóminas.

## Huecos y propietarios

| Capacidad | Resultado pendiente | Dueño y dependencia |
| --- | --- | --- |
| N1–N2 | Correspondencia titular/relación/entidad y lista propia paginada. | A consume el puerto de Personal B y el proveedor K; RRHH/Sistemas confirman fuente. |
| N3–N5 | Lectura, descarga independiente y rectificación enlazada del original. | A conserva consumidor; L/Documentos acuerdan custodia, política y firma cuando proceda. |
| N6–N8 | Discrepancia mínima, ámbitos económicos y auditoría completa. | A conserva expediente de discrepancia; unidad competente responde; K/L mantienen sus autoridades. |
| N9 | Hechos económicos aprobados con procedencia, vigencia y versión. | B/Personal, Cronos, Dietas, CT y Acción social emiten sus contratos; Nóminas no lee sus tablas. |
| N10–N12 | Conectores externos, reglas, revisión/cierre y ciclos conciliados. | Nóminas mantiene cálculo; Seguridad Social, Contrat@, fiscalidad, Intervención y Tesorería conservan sus actos. Requiere ampliación y presupuesto propios. |

Un único perfil activo fijo y concesión central positiva deben fijar acción,
recurso, entidad/relación/periodo, finalidad, campos y vigencia. La titularidad,
jefatura, soporte, menú o perfil candidato no conceden acceso económico. Gestión
interna y autoservicio conservan canales propios; una exposición exterior del
empleado exige política expresa, sin abrir la gestión.

Todas las listas, lecturas, descargas y efectos usan auditoría común nominal:
actor acreditado, perfil, acción, recurso opaco, finalidad, instante, resultado
permitido/denegado/error, proceso/canal, correlación y versión pertinente. Una
lectura confirma su auditoría antes de revelar datos; efecto, consumo de permiso,
estado e historia se confirman en la misma transacción. Denegaciones/errores se
registran después del rollback mediante el contrato común; sin identidad se usa
la frontera técnica, sin inventar actor. Fallo de auditoría impide entregar datos.
No se registran importes, banco, retenciones, familia, salud ni documentos completos.

## Minitareas y salidas por PR

Cada corte parte del SHA vigente que dirección confirme y recibe archivos exclusivos.
Las rutas nuevas son previsión. Composición, manifiestos, K, L y B solo los modifica
su custodio en un corte dependiente; no se crean autoridades paralelas.

| Corte y base | Salida usable por PR | Archivos propios previstos | Dependencias | Horas |
| --- | --- | --- | --- | ---: |
| N00 · inventario e idioma | Revalidar piezas y deuda; pasar los textos de la vista existente a catálogos ES/EN y formatear sus fechas con el lector común. La vista sigue sin fuente de recibos conectada. | Este plan, `modulos/nominas/{i18n.js,vista.js,nominas.test.mjs}` y `textos/{es,en}/nominas.json`. | SHA/FIN vigentes e idioma común del portal. | 1–2 |
| N01 · fuente | Contrato de lectura, titular, versión, cobertura y continuación oficial. | Contrato junto al adaptador propio. | Duda129; RRHH/Sistemas; custodia y tratamiento. | 3–5 |
| N02 · B1 | Consumidor nominal que consulta o deniega un recurso exacto con auditoría. | Nuevos `nominas/ports/{autorizacion,auditoria}.go`; consumidor propio. | N01; ABI K/L y perfiles fijos provisionados por huella/CAS. | 6–10 |
| N03 · B2/B3 | Correspondencia persona/relación/entidad histórica, con conflicto explicado. | Puerto/consumidor de relación de Nóminas. | B entrega relación y datos de servicio; K antes de lectura personal. DTO puede acordarse antes. | 3–5 |
| N04 · N2 | Puerto, adaptador admitido y consumidor focal del listado paginado. | `nominas/ports/recibos.go`, aplicación y adaptador propio. | N01–N03; fuente nominal autorizada. | 5–8 |
| N05 · N2/N7 | Vista existente conectada con estados claros y textos ES/EN en datos. | Cliente/vista de Nóminas; nuevos catálogos ES/EN. | N04; montaje por custodio y revisión de usabilidad. | 4–6 |
| N06 · N3/N4 | Lectura original con referencia, versión, huella y permiso exacto. | Consumidor documental propio; casos de uso. | N02/N04; contrato L/Documentos y custodio admitidos. | 4–7 |
| N07 · B5/N3 | Descarga autorizada aparte, auditada y ligada al original. | Cliente y consumidor de descarga propios. | N06; no URL con identidad/token ni caché compartida. | 4–6 |
| N08 · SQL necesario | Borrador reservado de estado mínimo de integración/rectificaciones/discrepancia. | SQL y adaptador propios de Nóminas. | Contratos N01–N07; no duplicar recibos corporativos. | 4–6 |
| N09 · ensayo | Ensayo en clon principal, revisiones exactas y kit; instalación por dirección. | Pruebas/kit propios. | N08; reserva, preimagen y dependencias. | 4–8 |
| N10 · N5/N6 | Rectificación enlazada y discrepancia mínima recuperable por la unidad competente. | Casos de uso, cliente y catálogos propios. | N06–N09 cuando haya estado durable; circuito interno admitido. | 4–6 |
| N11 · recorrido | Titular→recibo/descarga/discrepancia, negativos, reinicio y manual. | Pruebas focales y manual del módulo. | N05–N10; revisión documental, sensible y de usabilidad. | 4–6 |
| N12 · N9 posterior | Contratos económicos por fuente y periodo, con consumidor de ensayo. | Puertos y consumidor propios; cada emisor por su dueño. | Ampliación encargada; hechos aprobados B/Cronos/Dietas/CT/Acción social. | 6–10 |
| N13 · N11 posterior | Inventario y preparación contrastable de reglas por colectivo, sin cierre oficial. | Preparación propia y datos versionados de ensayo. | Políticas aprobadas; revisar límite de `payroll.go` con B, sin mover su autoridad unilateralmente. | 8–14 |
| N14 · N10 posterior | Acuerdo técnico y preparación de lotes de un intercambio externo elegido. | Puerto, adaptador preparatorio y consumidor propio. | Fuente/operador/esquema admitidos; distinguir RED/SILTRA, Contrat@ e IRPF. | 8–12 |
| N15 · N12 posterior | Guion de ciclos paralelos, revisión, fiscalización, cierre y conciliación con criterios acordados. | Plan de validación económica junto al consumidor. | N12–N14; responsables de Nóminas/Intervención/Tesorería. | 6–10 |

N00–N11 cierran el primer alcance de consulta/discrepancia. N12–N15 preparan la
ampliación económica; no estiman la sustitución completa del cálculo, todos los
conectores productivos o banca. Tras esas salidas se desglosará esa implementación
con fuentes y volúmenes conocidos. Si N01 solo admite derivación, se entrega esa
continuación y permanecen pendientes las lecturas; no se fabrica un recibo sintético.

N08/N09 solo proceden si hace falta estado durable propio. Se eliminan y recalculan
si la fuente/Documentos ya conservan todo el estado requerido. Toda migración futura
reserva número y orden causal en `RESERVAS_MIGRACIONES.md` fuera de Git; queda en
borrador hasta ensayo en clon principal y dos revisiones independientes. MCP
`postgres-clon-local` solo lectura/EXPLAIN; dirección instala. No reaplicar historia,
DOWN destructivo ni SQL en cidonia desde este carril.

## Dependencias y paralelismo

N00/N01 preceden a las bases nominales. N02 y el acuerdo de DTO N03 pueden avanzar
a la vez; su lectura espera autorización real. Con dos equipos, uno conserva lista,
cliente y catálogos (N04/N05); otro original, descarga y estado propio (N06–N09).
N10 reúne contratos finales; N11 espera capacidades montadas y, cuando corresponda,
instaladas. Un solo escritor conserva vista, idioma y cada contrato compartido.
K/L/B no son un tercer equipo incluido en estas horas. N12–N15 pueden estudiarse
después sin abrir cálculo real mientras se usan los recibos de origen.

## Configuración y estimación

Fuente/versiones, entidades, relaciones, periodos, tipos/formato, rectificación,
custodia, perfiles y conservación serán datos gobernados: norma/acto, órgano,
versión, huella y vigencia desde/hasta. Cada consulta conserva la versión pertinente.
Las reglas económicas futuras requieren el mismo gobierno; no hay cuantías en código.
RAT, habilitación concreta, destinatarios, bloqueo/archivo y expurgo quedan acordados
con DPD/Archivo antes de datos reales. Historia inmutable no fija retención ilimitada.

N00–N11 suman **46–75 h**, equivalentes a **6–10 jornadas de un equipo** de ocho
horas. Con dos equipos: **5–8 jornadas**, condicionado al solape descrito y a bases
disponibles. N12–N15 añaden **28–46 h** de preparación económica: conjunto **74–121 h**,
**10–16 jornadas de un equipo**; con dos equipos **8–13 jornadas**. Se redondean
jornadas completas y se conserva margen de integración/revisión; la espera externa
no reduce el esfuerzo ni es una fecha comprometida.

| Trabajo externo, fuera del total | Dedicación orientativa | Condición |
| --- | ---: | --- |
| RRHH / unidad de Nóminas | 4–8 h | Resolver129, cobertura y circuito de discrepancia; reglas económicas requieren encargo posterior. |
| Sistemas / proveedor actual | 6–12 h | Interfaz, entorno, identificadores, integridad y continuidad del original. |
| K/L, Documentos y Personal B | Estimación de sus dueños | Identidad/perfiles, proceso/canal, custodia/firma y relación histórica; se cuenta aquí solo el consumidor. |
| DPD / Archivo | 3–6 h | Finalidades, campos, series, acceso, conservación y riesgos; aprobación sin plazo comprometido. |
| Intervención / Tesorería / organismos externos | Estimación posterior | Ciclos y acuses de la ampliación económica; no están dentro del primer corte. |

## Comprobación y entrega

La PR documental comprueba inventario por SHA, enlaces locales, sumas y
`git diff --check`; requiere revisión independiente documental y una CI de PR.
No acredita nueva instalación ni usa puertas Go, SQL o navegador sobre Markdown.

Implementación futura: `programar-backend-vec`, `persistir-autorizar-vec`,
`programar-interfaz-vec` y `probar-recorridos-vec` según delta. Antes de pantallas,
`usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec` e
`impeccable`/`VEC-PRIORIDAD.md`, con revisión independiente de usabilidad.
Textos con `humanizer`/`VEC-USO.md`; SQL con `revisar-sql-vec`/`ensayar-sql`.
Revisión focal `security-audit` y Semgrep local en cambios sensibles, sin subir código;
gopls para Go y búsqueda por índice. Candidata exacta con `revisar-cambios-vec`,
`documentar-entregar-vec` y `pr-vec`; dirección integra y despliega.

La aceptación cubre titular ajeno, perfil/canal incorrectos, permiso caducado o
revocado, descarga independiente, fuente caída, auditoría fallida, original/huella,
rectificación conservada y recuperación tras reinicio. PC/móvil, teclado/foco e i18n
se comprueban con Chrome del sistema cuando exista pantalla nominal. Efectos nuevos
usan PostgreSQL real y reintento sin duplicados. No se repiten campañas de piezas
integradas sin delta o fallo nuevo. Las dudas permiten avanzar documentos y contratos
independientes; no conceden fuente, competencia económica ni firma legal.
