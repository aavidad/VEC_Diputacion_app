# Ficha de requisitos de VEC Nóminas: 4 de octubre de 2026

Estado: estudio funcional para validación de RRHH. El primer corte comprende la consulta de recibos propios procedentes de la fuente actual que confirme la Diputación. Esta ficha no acredita una aplicación de nómina operativa ni autoriza sustituir su cálculo, cierre o pago.

Sigue el formato de [Cronos](ficha_cronos_2026-09-23.md) y concreta [el catálogo, §7](catalogo_funcional_rrhh_y_hoja_ruta.md#7-nómina-y-gestión-económica), [la matriz normativa, §4](matriz_normativa_rrhh_2026.md#4-nómina-seguridad-social-y-fiscalidad) y [las materias económicas reservadas, §3](materias_reservadas_economicas_y_relaciones_laborales.md#3-nómina-seguridad-social-y-fiscalidad). Son antecedentes del estudio; las fuentes públicas verificadas figuran al final. Se mantienen persona común, relaciones históricas y autoridades separadas de E02–E07 de `ESPECIFICACIONES_AGENTES.md`.

## Qué debe hacer

La persona consulta sus recibos por entidad, relación y periodo, abre el documento de origen y puede solicitar que se revise una discrepancia. El documento conserva procedencia y versión. RRHH mantiene el circuito actual de elaboración y responde desde la unidad competente. Publicar el recibo, calcular una nómina y ordenar el pago son operaciones diferentes.

El desarrollo posterior podrá abordar conceptos, devengos, atrasos, regularizaciones, cotización, fiscalidad y conciliación. Requiere inventario, reglas aprobadas por colectivo y ciclos paralelos contrastados. La consulta inicial no aplica fórmulas ni transforma un borrador en recibo oficial. La fecha o el sello del documento deben proceder de su fuente; identificarse en VEC no firma la nómina.

## Requisitos para VEC

| Id | Requisito | Resultado comprobable |
| --- | --- | --- |
| N1 | Identidad común y relación acreditada por Personal. | El servidor resuelve persona, entidad y relaciones; el navegador no elige un titular ajeno. |
| N2 | Listado propio por periodo y relación, con paginación y procedencia. | Solo devuelve recibos autorizados; diferencia ausencia de documentos, denegación y fuente no disponible. |
| N3 | Consulta y descarga como capacidades independientes. | Poder ver el listado no autoriza descargar; cada documento exige permiso positivo vigente. |
| N4 | Documento de origen con referencia opaca, versión, fecha, formato y huella. | Se conserva el original recibido; no se presenta una reconstrucción como documento emitido por el pagador. |
| N5 | Correcciones enlazadas y acceso histórico. | Una rectificación identifica el documento sustituido y su fuente; nunca sobrescribe un recibo cerrado. |
| N6 | Canal de discrepancia hacia la unidad responsable. | La consulta puede referenciar recibo y periodo sin copiar salario, banco o circunstancias familiares a mensajes generales. |
| N7 | Datos económicos segregados. | Jefatura y soporte no acceden a recibos ni a retenciones por sus perfiles ordinarios. |
| N8 | Auditoría común de lista, lectura, descarga y discrepancia. | Actor y perfil nominales, finalidad, recurso opaco, instante, resultado y origen quedan acreditados. |
| N9 | Contratos futuros de entrada desde Personal, Cronos, Dietas y Contratación. | Cada hecho lleva fuente, versión, vigencia y aprobación; no se leen tablas ajenas. |
| N10 | Adaptadores futuros independientes para Seguridad Social, Contrat@ e IRPF. | Separan preparación, envío, acuse y aceptación; un fichero local no acredita recepción externa. |
| N11 | Reglas y cierre económico gobernados, fuera del primer corte. | Régimen funcionario/laboral, conceptos, redondeos y cambios retroactivos requieren políticas aprobadas. |
| N12 | Sustitución futura con revisión, fiscalización y conciliación. | Se valida por ciclos y responsables; cierre inmutable y rectificación enlazada. |

N1–N8 desarrollan NOM-007 del catálogo. N9–N12 conservan NOM-001–NOM-006 y NOM-008 como alcance posterior; no los cuentan como implementados.

## Perfiles y ámbitos positivos

Los nombres siguientes describen competencias propuestas. Sus asignaciones centrales deberán aprobarse con perfiles fijos, provisión por huella y comparación de la versión esperada (CAS). Una petición de consulta nunca publica permisos. Se usa un único perfil activo y se deniega cualquier ámbito o campo no concedido.

| Perfil | Capacidad positiva | Ámbito y campos |
| --- | --- | --- |
| Persona titular | Listar y abrir recibos propios; descarga con permiso separado. | Persona, relación, entidad y periodos admitidos, también de relaciones finalizadas si existe concesión vigente. |
| Gestión de nómina | Incorporar referencia y resolver discrepancia. | Entidades y series asignadas; acceso nominal solo cuando lo requiera el caso de uso autorizado. |
| Revisión de nómina | Revisar futura preparación y cierre. | Lote y periodo concretos; separación respecto a quien cambia reglas o prepara el lote. |
| Intervención / Tesorería | Fiscalizar / ordenar y conciliar, en operaciones propias. | Expediente o lote y datos económicos mínimos; no implica navegar todos los recibos. |
| Operación de conector | Consultar estado técnico y reintentar el intercambio autorizado. | Fuente y lote asignados; sin acceso general a documentos personales. |
| Auditoría | Consultar trazabilidad y exportar con concesión independiente. | Finalidad, periodo y referencias delimitados; no obtiene automáticamente el contenido de la nómina. |

Administrador funcional, administrador técnico y jefatura no son perfiles universales. La representación, si se admite, requiere vínculo vigente y alcance expreso en el núcleo común; no entra por defecto en el primer corte.

La gestión de RRHH usa el canal interno autorizado. El autoservicio del empleado conserva su frontera de canal y recursos propios. Ser la misma Persona en Bolsa o en el portal público no concede acceso a recibos: el perfil de candidato y el canal exterior de candidaturas quedan excluidos. Un acceso exterior para empleados necesita una política expresa de esa capacidad, sin trasladar al exterior las operaciones internas de gestión.

## Datos mínimos y privacidad

El índice necesita referencias de persona y relación, entidad pagadora, periodo, tipo documental, identificador de origen, versión, estado comunicado por la fuente y referencia de custodia. El contenido salarial permanece en el documento o servicio autorizado. No se incorpora a Persona un maestro paralelo de salario, banco, embargo o familia.

La base del tratamiento debe documentarse por finalidad: obligación legal o misión pública en RGPD, art. 6.1.c/e, con su habilitación concreta. Artículos 5, 25 y 32: minimización y seguridad. Si un recibo revela salud o afiliación, deben justificarse además las condiciones del art. 9; nunca basta un consentimiento genérico. [RGPD publicado en BOE](https://www.boe.es/doue/2016/119/L00001-00088.pdf).

Antes de datos reales: responsable del tratamiento, RAT, destinatarios, información al titular, custodia y política de conservación aprobados con DPD y Archivo. La historia de solo adición protege las rectificaciones; no establece retención ilimitada. La aplicación separa uso activo, bloqueo y archivo; el expurgo autorizado conserva la evidencia mínima exigible. [LOPDGDD, arts. 8, 28 y 32](https://www.boe.es/buscar/act.php?id=BOE-A-2018-16673).

Cada consulta, descarga, cambio, denegación o error deja auditoría nominal en la autoridad común: actor, perfil activo, acción, recurso opaco, finalidad, instante, resultado y origen (proceso y canal), con correlación y versión cuando proceda. El efecto y su auditoría se escriben en la misma transacción. Las lecturas se auditan antes de entregar datos; si falla, no se devuelve el documento. La auditoría es de solo adición, segregada de la gestión funcional. No incluye importes, diagnósticos, familia, documentos completos ni credenciales. Los logs técnicos tampoco los contienen.

## Relaciones internas por puertos

| Propietario | Contrato mínimo propuesto | Límite |
| --- | --- | --- |
| Núcleo común | Identidad, contexto, autorización y auditoría. | Mismo acceso VEC, sin login ni permisos propios del módulo. |
| Personal | Persona/relación, entidad, ocupación, jornada y periodos históricos autorizados. | Personal mantiene la relación; una cuenta no prueba empleo. |
| Cronos | Incidencia con efecto económico ya autorizado, periodo y versión. | No recibe diagnósticos ni decide automáticamente una deducción o sanción. |
| Dietas | Liquidación aprobada, referencia, importe y estado de conciliación. | La aprobación no se interpreta como pago ni se paga dos veces. |
| Contratación temporal | Referencia a alta/cese confirmado por su autoridad y enlace a Personal. | Una propuesta de contratación no activa por sí sola nómina. |
| Documentos / Archivo | Lectura autorizada del original y metadatos de preservación. | La referencia no sustituye custodia ni firma. |
| Acción social | Ayuda autorizada o reintegro con referencia, periodo e importe. | Solo efecto económico; no recibe el expediente familiar o sanitario. |

Los contratos se versionan con finalidad, idempotencia y referencia de origen; cada módulo mantiene sus datos. El agregador Personal consulta por puertos y conserva restricciones de campos y canal.

## Dependencias externas

La fuente actual de recibos se conectará por un adaptador de lectura que Informática y RRHH autoricen. GINPIX u otra aplicación solo se configurará cuando se confirme su responsabilidad e interfaz. Debe reconciliar titulares y relaciones, comprobar integridad y evitar cachés compartidas o enlaces con datos personales en URL.

Seguridad Social (RED/SILTRA), comunicación de contratos (Contrat@) e IRPF serán adaptadores separados, cuando llegue su corte. Mantendrán esquema, versión, lote, huella, autorización de operador, envío, acuse, rechazo y conciliación. No se implementan ahora cálculo, presentación ni pago. Contabilidad, Tesorería y banca mantienen su autoridad sobre sus actos; sus respuestas nunca sustituyen el recibo salarial de origen.

## Normativa y fuentes públicas verificadas

Consulta: 04/10/2026. Los textos consolidados permiten estudiar los preceptos; para activar una regla se conservarán acto auténtico, versión y fecha de efectos.

| Fuente oficial y artículo | Consecuencia para el alcance |
| --- | --- |
| [TREBEP, arts. 21–30](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719#a21) | Distinguir regímenes y conceptos; la consulta no calcula retribuciones. |
| [LBRL, arts. 90 y 93](https://www.boe.es/buscar/act.php?id=BOE-A-1985-5392#a90) y [RD 861/1986, arts. 1–7](https://www.boe.es/buscar/act.php?id=BOE-A-1986-10798#a1) | Registro de Personal como fundamento; retribuciones locales con su régimen propio. |
| [ET, arts. 26–29 y 8.3](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11430#a29) | Recibo salarial del personal laboral y comunicación de contratos; ámbitos distintos. |
| [LGSS, arts. 139–142](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11724#a139) | Afiliación, altas/bajas y cotización tienen obligaciones e intercambios específicos. |
| [Ley 35/2006, art. 99](https://www.boe.es/buscar/act.php?id=BOE-A-2006-20764#a99) | Retenciones y pagos a cuenta, fuera del corte de consulta. |
| [Ley 39/2015, arts. 17, 26 y 53](https://www.boe.es/buscar/act.php?id=BOE-A-2015-10565#a17) | Archivo, documentos y acceso dentro de sus procedimientos. |
| RGPD y LOPDGDD, preceptos enlazados arriba. | Finalidad, acceso limitado y conservación con fundamento. |

## Qué debe ser configurable

Fuente y versión del conector, entidades/relaciones cubiertas, periodos disponibles, metadatos y tipos de recibo, política de rectificación, formatos, custodia, perfiles y conservación. Cada configuración guarda fuente pública o acto interno competente, versión, huella, vigencia desde/hasta y órgano aprobador. Un cambio conserva la configuración usada para cada consulta y documento. Fórmulas, topes o tablas futuras exigen el mismo gobierno por periodo; no se fijan en código. Textos visibles, estados y ayudas usarán catálogos por idioma.

## Primer corte usable y decisiones de RRHH

N1–N5, N7–N8: la persona entra con su identidad VEC, elige entre sus relaciones autorizadas, lista un periodo y consulta o descarga el recibo original. Debe poder reconocer emisor, periodo y eventual rectificación. Si la fuente no está conectada se indica y se mantiene deshabilitada la descarga.

Aceptación futura: fuente real autorizada y ensayo sintético, navegador hasta documento y auditoría común, denegación de titular ajeno, permiso independiente de descarga, fallo del conector sin filtración, rectificación conservada y recuperación del mismo documento tras reinicio. Son criterios pendientes de implementación; esta entrega solo verifica documentación y fuentes.

RRHH elige unidad responsable, fuente, relaciones y periodos iniciales y circuito de discrepancia. Informática acuerda la interfaz; DPD/Archivo fijan tratamiento y conservación. Las reglas económicas y la sustitución del cálculo se estudian en un corte posterior.

Pregunta interna consolidada: **dudas.md, 129 (Nóminas)**, desarrolla la 39 y las 28–29 sobre Personal; las 21 y 26 ya cubren efectos de Cronos y liquidaciones de Dietas, y la 84 cubre conservación por series. Evita pedir de nuevo normas públicas o parámetros de un futuro cálculo.
