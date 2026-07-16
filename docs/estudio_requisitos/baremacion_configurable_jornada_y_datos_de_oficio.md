# Estudio de baremacion configurable, jornada y datos obtenidos de oficio

## Estado del documento

| Campo | Valor |
| --- | --- |
| Fecha de contraste normativo | 16 de julio de 2026 |
| Naturaleza | Estudio funcional, juridico, de datos y de arquitectura previo a la implementacion. |
| Estado | Propuesta razonada pendiente de validacion por Seleccion, Personal, Secretaria/Asesoria Juridica, DPD y Sistemas. |
| Alcance | Convocatorias, bolsas, autobaremacion, revision administrativa y servicios prestados. |
| Acredita implementacion | **Parcial.** Esta implantado el fundamento neutral de valores exactos; no el motor gobernado, sus reglas, persistencia ni pantallas. |
| Puerta productiva | **NO-GO.** El baremo demostrativo actual no puede utilizarse para resolver una convocatoria real. |

Este documento no sustituye el informe juridico de la Diputacion ni interpreta
unas bases concretas. Su finalidad es impedir que el producto codifique como
regla universal lo que en Derecho depende de cada convocatoria.

### Estado tecnico del fundamento exacto

El paquete `internal/shared/baremacion` ya incorpora y prueba:

- puntuacion no negativa en micropuntos, con seis decimales exactos;
- racionales canonicos y acotados;
- fracciones de jornada en el intervalo `(0, 1]`, por ejemplo `1/3` sin
  aproximacion binaria;
- fechas civiles gregorianas sin hora ni zona;
- intervalos civiles semiabiertos `[desde, hasta)` para evitar dobles conteos;
- serializacion JSON canonica y rechazo cerrado de valores ambiguos, no
  canonicos o fuera de limites.

Son tipos neutrales: no contienen coeficientes, meses convencionales,
redondeos, topes ni reglas de una convocatoria. Su existencia no levanta el
`NO-GO` del motor. El tipo de puntuacion anterior de Bolsa sigue siendo un
contrato diferente; su migracion exigira un conversor explicito y versionado,
sin alias ni reinterpretacion silenciosa del JSON historico.

## Decision ejecutiva

Las bases publicadas deben ser la autoridad de cada proceso. No existe una
formula universal que permita afirmar que la experiencia vale siempre 0,1
puntos por mes, que toda jornada parcial se prorratea de la misma forma o que
una titulacion superior vale siempre un punto. La aplicacion debe permitir a
RRHH describir lo aprobado en las bases mediante reglas tipadas, validarlas con
casos de ejemplo y publicar una version inmutable y firmada.

La solucion propuesta se apoya en diez decisiones:

1. Cada convocatoria referencia una version exacta de bases y una version
   exacta de reglas de baremacion. Ningun calculo consulta «las reglas
   actuales».
2. RRHH configura las reglas desde el portal interno, pero no introduce codigo,
   SQL ni expresiones ejecutables. Utiliza un constructor de decisiones con
   operadores seguros y catalogos gobernados.
3. La version publicada queda congelada. Una correccion exige acto formal,
   documento rectificativo, nueva version enlazada y analisis de los calculos
   afectados; nunca se modifica silenciosamente una convocatoria abierta.
4. El calculador es determinista, usa enteros y punto fijo, conserva todos los
   pasos y puede reproducir historicamente el resultado byte a byte.
5. La jornada se representa por subperiodos y fraccion exacta, no con
   `float`. Las bases deciden si se prorratea, se computa de forma integra, se
   agregan jornadas concurrentes o se elige el periodo de mayor valor.
6. Las reducciones que legalmente o segun las bases deban computar como jornada
   completa llegan al motor como una atestacion minima. El motivo medico,
   familiar o de violencia no se expone al expediente de baremacion.
7. Los servicios de la Diputacion se incorporan de oficio desde el historial
   administrativo de Personal. La nomina ayuda a reconciliar, pero no es la
   unica fuente de autoridad.
8. La persona candidata ve el periodo obtenido de oficio y su calculo, no puede
   editar el dato administrativo y dispone de rectificacion, subsanacion o
   alegacion cuando discrepa.
9. El autobaremo ciudadano es una propuesta explicada. La puntuacion
   administrativa resulta de meritos y evidencias revisados, decisiones
   motivadas y firmadas y el mismo motor de reglas publicado.
10. Ninguna IA decide admision, validez de un merito ni puntuacion. La IA puede
    explicar informacion publica o ayudar a localizar una regla sin acceder a
    datos personales ni sustituir al organo competente.

## Que debe ser configurable y que no

### Configurable por convocatoria

- Secciones, subsecciones y tipos de merito.
- Coeficiente por dia, mes, ano, hora, credito, titulo, examen u otra unidad
  cerrada.
- Ambito de la experiencia: misma categoria, categoria equivalente, cuerpo,
  grupo, funciones, Diputacion, Administracion local, otra Administracion,
  sector publico, privado, extranjero, cooperacion o investigacion.
- Relaciones de equivalencia entre categorias y titulaciones, siempre con
  fundamento y version.
- Fecha de corte, tratamiento del periodo en curso y unidad de calendario.
- Regla de jornada, concurrencia, solape, interrupcion, excedencia y ausencia.
- Conversion de dias, meses, anos, horas y creditos.
- Precision intermedia, redondeo, truncamiento y tratamiento de restos.
- Requisitos de acreditacion, fuentes obtenidas de oficio y documentos
  alternativos admitidos.
- Exclusiones, incompatibilidades, duplicidades y eleccion del merito mas
  favorable.
- Minimos, maximos por regla, seccion y total.
- Criterios de desempate y su orden.
- Politica sobre el limite del autobaremo declarado, si las bases la establecen.

### Invariantes que la pantalla no puede rebajar

- Igualdad, merito, capacidad, publicidad, transparencia e imparcialidad.
- Relacion objetiva de los meritos con las funciones y requisitos de la plaza.
- Criterios y medios de acreditacion fijados antes de aplicarlos.
- Denegacion por defecto, competencia del actor, separacion de funciones y
  doble control.
- Versionado, firma, integridad, trazabilidad, motivacion y derecho de revision.
- Minimizacion, limitacion de finalidad, exactitud y seguridad de los datos.
- Limites tecnicos de complejidad, magnitud, precision y volumen.
- Prohibicion de ejecutar codigo o consultas introducidos desde el formulario.
- Inmutabilidad del expediente y de la version publicada.

## Marco juridico revisado

### Seleccion y bases

| Fuente oficial | Consecuencia para el producto |
| --- | --- |
| [Constitucion Espanola, articulos 23.2 y 103.3](https://www.boe.es/buscar/act.php?id=BOE-A-1978-31229) | El acceso debe respetar igualdad, merito y capacidad. La tecnologia no puede introducir ventajas, criterios ocultos o excepciones personales. |
| [Doctrina constitucional recopilada por el BOE sobre el articulo 23.2](https://www.boe.es/legislacion/derechos_fundamentales.php?id_articulo=23.2&id_concepto=222) | Los criterios absolutos y relativos deben estar predeterminados. El motor debe ejecutar reglas fijadas con anterioridad, no completar silencios despues de ver candidaturas. |
| [TREBEP, Real Decreto Legislativo 5/2015, articulos 3, 10, 11, 37, 55 y 59 a 61](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719) | Exige publicidad, transparencia, imparcialidad, adecuacion entre pruebas y funciones y proporcionalidad de la valoracion de meritos en el concurso-oposicion. Obliga a resolver si los criterios generales requieren negociacion, sin confundirlos con la determinacion concreta excluida en los terminos del articulo 37. |
| [Directiva 97/81/CE sobre trabajo a tiempo parcial](https://eur-lex.europa.eu/legal-content/ES/TXT/?uri=CELEX%3A31997L0081) y [Ley Organica 3/2007 de igualdad](https://www.boe.es/buscar/act.php?id=BOE-A-2007-6115) | Una regla aparentemente neutra de jornada puede requerir justificacion objetiva y analisis de discriminacion indirecta. La revision juridica debe comprobar su necesidad, proporcionalidad e impacto, en particular sobre quienes ejercen derechos de conciliacion. |
| [Ley 7/1985, reguladora de las bases del regimen local, texto actualizado el 3 de junio de 2026](https://www.boe.es/buscar/act.php?id=BOE-A-1985-5392) | Enmarca la seleccion de personal de las entidades locales, remite a convocatoria publica y principios constitucionales y contempla el Registro de Personal. Refuerza que el dato maestro de servicios sea administrativo, no el importe de nomina. |
| [Real Decreto 896/1991, reglas basicas de seleccion local](https://www.boe.es/eli/es/rd/1991/06/07/896/con) | Las bases locales deben especificar, cuando proceda, los meritos, su valoracion y los medios para acreditarlos. La aplicacion debe conservar esos tres elementos juntos. |
| [Ley 5/2023, de la Funcion Publica de Andalucia, texto consolidado](https://www.boe.es/buscar/act.php?id=BOE-A-2023-16066) | Su articulo 3.1.d incluye al personal de las Administraciones locales andaluzas con respeto a la autonomia local y la legislacion basica. Refuerza principios, objetividad, adecuacion, publicidad y vinculacion de bases; los preceptos referidos expresamente a la Administracion General de la Junta sirven como referencia y no se trasladan automaticamente a la Diputacion. |

Las bases vinculan a la Administracion, al organo de seleccion y a las personas
participantes. Por ello, una mejora tecnica no autoriza a reinterpretar una base
ambigua. La pantalla debe bloquear la publicacion de reglas incompletas y
remitir la duda a RRHH y al informe juridico antes de abrir el plazo.

Los permisos, reducciones y excedencias protegidos por el TREBEP, el Estatuto
de los Trabajadores y la normativa de igualdad producen derechos en su propio
ambito, pero no existe una conversion informatica unica de todos ellos a puntos
de experiencia. La norma aplicable y las bases deben indicar el efecto. Tampoco
debe aceptarse sin analisis una reduccion proporcional que, aun formulada de
forma neutra, perjudique especialmente a un colectivo protegido. Por eso el
constructor registra la opcion, su fundamento y el informe que la valida; no
propone una opcion «normal» por defecto.

### Cautelas jurisprudenciales vigentes

- **No privilegiar automaticamente a la Administracion convocante.** La [STS
  4712/2025, ECLI:ES:TS:2025:4712](https://www.poderjudicial.es/search/sentencias/principio%20de%20merito%20y%20capacidad/21/AN)
  considero discriminatorio duplicar la puntuacion por haber trabajado en el
  SERGAS frente a servicios equivalentes de otros servicios de salud. Una
  diferencia limitada puede requerir motivacion objetiva y proporcional, pero
  no se presume valida. Por ello, una regla como «mas puntos por trabajar en la
  Diputacion» queda bloqueada hasta aportar informe juridico y justificacion
  funcional especificos.
- **La experiencia previa no puede cerrar materialmente el acceso.** La [STC
  86/2016](https://hj.tribunalconstitucional.es/HJ/es/Resolucion/Show/24932)
  admite valorar experiencia dentro de limites, no convertirla en una ventaja
  desproporcionada que haga ilusorio competir a terceros.
- **Una cifra no sustituye la motivacion.** La [STS 2511/2025,
  ECLI:ES:TS:2025:2511](https://www.poderjudicial.es/search/sentencias/principio%20de%20merito%20y%20capacidad/51/PUB)
  exige poder explicar los pasos seguidos al aplicar las bases. El expediente
  debe conservar hechos, regla, operaciones, evidencia y decision, ademas de la
  puntuacion.
- **Prorrata y proteccion no se confunden.** La STS (Social) 378/2026, de 15 de
  abril, ECLI:ES:TS:2026:1798, avala la proporcionalidad por servicios efectivos
  a tiempo parcial en el supuesto concreto de un proceso de estabilizacion y la
  distingue del computo individual de antiguedad. Se [identifica y reproduce en
  este acceso jurisprudencial](https://www.iberley.es/jurisprudencia/sentencia-social-tribunal-supremo-sala-lo-social-15-4-26-1840740849).
  Es una referencia actual relevante, no una formula universal para toda bolsa
  local, y no elimina la proteccion que corresponda a conciliacion u otras
  situaciones.

El producto debe permitir configurar lo que resulte juridicamente valido, no
convertir en opcion ordinaria una diferencia de trato de alto riesgo.

### Procedimiento, interoperabilidad y datos de oficio

| Fuente oficial | Consecuencia para el producto |
| --- | --- |
| [Ley 39/2015, articulos 28, 35, 53, 75 a 77, 82 y 88](https://www.boe.es/buscar/act.php?id=BOE-A-2015-10565) | Debe evitarse exigir documentos que ya obren en poder de una Administracion en los terminos legales. La consulta de oficio, la oposicion cuando proceda, la imposibilidad, prueba, audiencia, motivacion y resolucion deben quedar registradas. |
| [Ley 40/2015, articulo 155](https://www.boe.es/buscar/act.php?id=BOE-A-2015-10566) | El intercambio entre Administraciones debe limitarse a datos necesarios, con condiciones y garantias trazables. |
| [Esquema Nacional de Interoperabilidad, Real Decreto 4/2010](https://www.boe.es/buscar/act.php?id=BOE-A-2010-1331) y [NTI de protocolos de intermediacion](https://www.boe.es/buscar/act.php?id=BOE-A-2012-10049) | Los conectores de Personal, registro, titulaciones u otras Administraciones deben usar formatos, metadatos y evidencias interoperables, sin acoplar el dominio al proveedor. |
| [Reglamento de actuacion electronica, Real Decreto 203/2021, articulos 61 y 62](https://www.boe.es/buscar/act.php?id=BOE-A-2021-5032) | La transmision de datos se incorpora al procedimiento con valor y trazas administrativas. Deben constar titular, dato, finalidad/procedimiento, instante y, cuando corresponda, empleado que consulta. |

Obtener de oficio no equivale a copiar sin control. Cada dato incorporado debe
identificar fuente, instante, finalidad, resultado de la consulta, version y
huella de la instantanea usada para el calculo.

### Proteccion de datos, seguridad y decisiones automatizadas

| Fuente oficial | Consecuencia para el producto |
| --- | --- |
| [RGPD](https://eur-lex.europa.eu/legal-content/es/TXT/?uri=CELEX%3A32016R0679) y [LO 3/2018](https://www.boe.es/buscar/act.php?id=BOE-A-2018-16673) | Deben aplicarse licitud, finalidad, minimizacion, exactitud, conservacion limitada, privacidad desde el diseno, seguridad, transparencia y ejercicio de derechos. Por la evaluacion sistematica con efectos relevantes y los cruces previstos, la recomendacion del estudio es realizar una evaluacion de impacto relativa a la proteccion de datos (EIPD) antes de produccion, sujeta a la determinacion formal del DPD. |
| [AEPD: evaluacion de la intervencion humana en decisiones automatizadas](https://www.aepd.es/prensa-y-comunicacion/blog/evaluacion-de-la-intervencion-humana-en-las-decisiones-automatizadas) | Una revision nominal no basta si la persona revisora no tiene competencia, informacion, tiempo ni capacidad real para corregir. La intervencion administrativa debe ser efectiva y auditable. |
| [Ley 40/2015, articulos 41 y 42](https://www.boe.es/buscar/act.php?id=BOE-A-2015-10566) | Una actuacion administrativa automatizada final exige identificar los organos responsables de especificacion, programacion, mantenimiento, supervision, auditoria y recurso y su sistema de firma. La primera fase evita ese regimen: el motor calcula y explica; el organo competente decide. |
| [AEPD: guia de evaluacion de impacto para las AAPP](https://www.aepd.es/prensa-y-comunicacion/notas-de-prensa/la-aepd-publica-una-guia-para-ayudar-las-aapp-realizar-evaluaciones-de-impacto-desde-el-diseno) | El tratamiento masivo de candidatos, cruces de fuentes y datos especialmente sensibles exige analizar riesgos desde el diseno. |
| [Esquema Nacional de Seguridad, Real Decreto 311/2022](https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191) | Identidad, autorizacion, registro, custodia, disponibilidad e integridad deben responder a la categoria que formalmente determine la Administracion, no a una etiqueta comercial de «alta seguridad». |
| [Reglamento europeo de inteligencia artificial, Reglamento (UE) 2024/1689](https://eur-lex.europa.eu/eli/reg/2024/1689/oj/spa) | El calculador propuesto es aritmetica determinista, no un modelo de IA. Si en el futuro una IA influyera en seleccion o gestion de empleo, debera analizarse expresamente su clasificacion y obligaciones antes de usarla. |

No se enviaran al motor causas clinicas, familiares, sindicales o de violencia
que expliquen una reduccion. Personal emitira, cuando corresponda, un dato
derivado como `computo_jornada_integro_por_situacion_protegida`, con vigencia,
fuente y atestacion, pero sin revelar la causa a Bolsa.

La base juridica ordinaria del tratamiento publico se determinara conforme al
articulo 6.1.c/e y 6.3 RGPD y la norma aplicable; el consentimiento no sera un
atajo por defecto para obtener los servicios internos. La persona sera
informada y podra ejercer los derechos o la oposicion que procedan. Las listas
publicas se generaran con identificacion limitada conforme a la disposicion
adicional septima de la LOPDGDD y el criterio del DPD; nunca publicaran DNI
completo, jornada, causas protegidas, titulos ni justificantes.

## Comparacion de soluciones y bases reales

Los valores siguientes son ejemplos de bases oficiales, no reglas generales ni
una recomendacion automatica para la Diputacion.

| Referencia | Mecanismos observados | Requisito que aporta al producto |
| --- | --- | --- |
| [Bolsa Unica Comun de la Junta de Andalucia, Resolucion de 10 de diciembre de 2019](https://www.juntadeandalucia.es/boja/2019/240/18.html) | Puntuacion diaria distinta para experiencia en la misma categoria y funciones homologas; antiguedad publica; formacion por hora; titulaciones independientes; ejercicios superados; limites y reglas sobre el autobaremo declarado. | Unidades por dia, ambitos y equivalencias versionados, deduplicacion de cursos y politica configurable sobre el limite de lo alegado. |
| [Pacto de Bolsa Unica del Servicio Andaluz de Salud, BOJA de 9 de enero de 2026](https://www.juntadeandalucia.es/boja/2026/5/10) | Multiples coeficientes segun categoria, centro, sistema, pais y tipo de experiencia; formulas de conversion y precision; docencia, investigacion, formacion, oposicion y desempates. Los servicios SAS se muestran de oficio desde su sistema de personal. | Motor multidimensional, precision fijada por bases, reglas sectoriales y proyeccion de servicios internos sin pedir certificado al empleado. |
| [Ayuntamiento de Baza, Auxiliar de Biblioteca, BOP Granada 2025](https://bop-admin.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1741910455812-final-4c255791-2.pdf) | 0,55 puntos por mes completo en determinados servicios; jornada parcial proporcional; suma de dias dividida entre 30 y descarte de decimales; determinadas reducciones protegidas computan a tiempo completo; acreditacion publica combinada. | La politica de jornada protegida, el resto temporal y los documentos exigidos pertenecen a la regla, no a una formula global. |
| [INGESA, convocatoria de 2026](https://www.boe.es/diario_boe/txt.php?id=BOE-A-2026-10143) | Unidad diaria y conversiones; reglas especificas para reducciones, excedencias, guardias y periodos concurrentes; eleccion de la subcategoria de mayor puntuacion; contenido detallado del certificado de servicios. | Periodos semanticos, horas especiales, ausencia, solape y fuente probatoria deben modelarse expresamente. |
| [Junta de Andalucia, personal interino de Justicia, 2026](https://www.juntadeandalucia.es/boja/2026/38/24) | Valores mensuales diferentes para experiencia en el mismo cuerpo, cuerpo superior o inferior y otros ambitos. | Matriz configurable de relacion entre categoria de origen y categoria convocada. |
| [Osakidetza: curriculum y listas de contratacion](https://www.osakidetza.euskadi.eus/servicios-curriculum-vitae/webosk00-procon/es/) | El profesional visualiza experiencia existente en RRHH y meritos validados, y aporta lo que falta para su revision. | Separacion entre dato autoritativo interno, dato declarado y decision administrativa de validacion. |
| [Junta de Andalucia, personal laboral fijo, Resolucion de 18 de julio de 2025](https://www.juntadeandalucia.es/boja/2025/141/31) | En esa convocatoria las jornadas inferiores a la completa se valoran como completas; el autobaremo limita por apartado lo que la comision puede conceder. | La jornada y el caracter vinculante u orientativo del autobaremo son dos politicas independientes y expresas. |
| [Congreso de los Diputados, Enfermeria, convocatoria de 2026](https://www.boe.es/diario_boe/txt.php?id=BOE-A-2026-12330) | Jornada parcial proporcional; contratos parciales simultaneos acumulables hasta el 100 %; reduccion por guarda legal completa; fracciones inferiores al mes acumulables. | Se necesita agregar fracciones compatibles con techo y distinguir contrato parcial de reduccion protegida. |
| [Red Sanitaria Militar, convocatoria de 2025](https://www.boe.es/buscar/doc.php?id=BOE-A-2025-26153) | Parcial proporcional; solapes parciales acumulables hasta el 100 %; titular con reduccion protegida a jornada completa y sustituto por su porcentaje real; autobaremacion orientativa. | La condicion de titular/sustituto y el alcance del autobaremo cambian un resultado aun con las mismas fechas. |
| [Ayuntamiento de Canena, proceso publicado en 2022](https://bop.dipujaen.es/descargarws.dip?anioExpedienteEdicto=2022&boletinSuplemento=0&ejercicioBop=2022&fechaBoletin=2022-12-15&numeroEdicto=6235&tipo=bop) | En ese caso, una jornada igual o superior al 50 % se computa completa y una inferior se prorratea; los contratos simultaneos se resuelven por el mas favorable. | Debe existir una politica por umbral y no puede suponerse que todos los solapes se suman. |

La comparacion descarta un formulario limitado a «puntos por ano». Un baremo
profesional necesita expresar la semantica completa que transforma una evidencia
en unidades computables y estas en puntuacion.

## Modelo funcional de las reglas

### Agregado `ConjuntoReglasBaremo`

Cada version debe contener como minimo:

- identidad de convocatoria, expediente y version;
- referencia y huella del documento de bases publicado;
- fecha de corte de meritos;
- secciones ordenadas, maximos y minimos;
- reglas tipadas, incompatibilidades y prioridades;
- catalogos y tablas de equivalencia por identidad, version y huella;
- politica temporal, de jornada, solape, precision y redondeo;
- evidencias admitidas y fuentes obtenidas de oficio;
- cadena de desempates;
- ejemplos de prueba aprobados;
- autor, revisores, aprobadores, firmas, instantes y motivo;
- representacion canonica y huella criptografica.

Una regla no guardara solo `puntos_por_unidad`. Debe conservar la condicion de
aplicacion, la unidad reconocida, el modo de obtenerla, los limites y la
explicacion que vera la persona interesada.

### Familias de meritos que debe soportar el constructor

El catalogo inicial debe incluir, sin convertirlo en una lista cerrada de
codigo:

- experiencia y antiguedad;
- titulaciones academicas oficiales, equivalencias y titulaciones extranjeras
  homologadas o reconocidas;
- formacion recibida e impartida, horas, creditos, aprovechamiento, vigencia y
  entidad organizadora u homologadora;
- ejercicios o fases de oposicion superados;
- idiomas y acreditaciones por nivel;
- docencia, tutoria, investigacion, publicaciones y actividad cientifica cuando
  proceda;
- permisos, carnets, habilitaciones y certificados profesionales;
- otros meritos expresamente previstos por una norma o las bases.

La incorporacion futura de una familia requerira un nuevo tipo de regla o
conector si aporta una semantica nueva. Crear desde administracion un valor mas
dentro de una familia ya soportada no exige recompilar.

### Experiencia: ejes de clasificacion

Una regla de experiencia puede combinar los ejes que las bases utilicen:

| Eje | Ejemplos de valores gobernados |
| --- | --- |
| Empleador | Diputacion, entidad local, Junta, otra Administracion, sector publico, privado, extranjero. |
| Relacion | Funcionario de carrera o interino, laboral fijo o temporal, estatutario, autonomo u otra relacion admitida. |
| Trabajo | Misma categoria, equivalente, superior, inferior, mismo grupo, mismas funciones o funciones relacionadas. |
| Centro o ambito | Centro concreto, servicio de salud, dificil cobertura, investigacion, cooperacion u otro previsto. |
| Periodo | Inicio, fin inclusivo o exclusivo segun regla, fecha de corte, interrupciones y tramos especiales. |
| Dedicacion | Completa, fraccion contractual, variable, discontinua, horas o guardias. |
| Situacion computable | Servicio activo, ausencia o reduccion protegida atestada, excedencia admitida u otra situacion prevista. |
| Evidencia | Historial interno, certificado de servicios, vida laboral, contrato, nombramiento u otras combinaciones exigidas. |

Las equivalencias de categoria o funciones no se deducen por similitud textual.
Se publican en una tabla aprobada o se resuelven mediante decision motivada del
organo competente, con efecto solo donde las bases lo permitan.

## Modelo exacto de periodo y jornada

### Datos minimos de un subperiodo

Un periodo de servicio se divide cada vez que cambia la categoria, empleador,
relacion, situacion administrativa o dedicacion. Cada subperiodo conserva:

- referencia opaca y version de fuente;
- fecha y, solo si las bases lo requieren, hora de inicio y fin;
- categoria, puesto, grupo y funciones normalizadas;
- empleador, unidad y naturaleza publica o privada;
- tipo de relacion y situacion computable;
- fraccion de jornada exacta como numerador/denominador o puntos basicos, donde
  10.000 representa el 100 %;
- horas cuando la base use una conversion horaria;
- atestacion de computo integro sin revelar una causa protegida;
- incidencias, solapes y rectificaciones enlazadas;
- fuente, fecha de consulta y huella del dato obtenido.

No se usa una media anual si hubo cambios: seis meses al 100 % y seis al 50 %
son dos tramos. Tampoco se infiere la jornada a partir del importe de una
nomina, pues atrasos, complementos, embargos, ausencias o regularizaciones
distorsionan esa deduccion.

### Politicas de jornada que el motor debe poder expresar

| Politica | Efecto |
| --- | --- |
| Proporcional | Multiplica la unidad temporal por la fraccion de jornada del tramo. |
| Integra | Computa el periodo como 100 % aunque el contrato sea parcial, solo si las bases lo ordenan. |
| Integra desde umbral | Computa al 100 % desde el porcentaje publicado y prorratea por debajo; el umbral forma parte de la version de reglas. |
| Protegida integra | Prorratea ordinariamente, pero una atestacion valida hace computar el tramo al 100 %. |
| Por horas | Convierte horas acreditadas a dias, meses o anos con el divisor exacto de las bases. |
| Variable | Usa tramos o una relacion de horas certificada; no admite un porcentaje libre sin fuente. |
| Concurrente acumulable | Suma fracciones de periodos compatibles hasta el limite fijado, normalmente 100 %. |
| Concurrente no acumulable | Elige una sola regla o el periodo mas favorable, con desempate determinista. |
| Sustitucion de reduccion | Computa a la persona sustituta por la dedicacion realmente certificada, aunque el titular tenga computo integro protegido. |

«Reduccion del 50 %» y «contrato al 50 %» no son necesariamente equivalentes.
La primera puede estar protegida y computar integramente segun la base; el
segundo puede prorratearse. Bolsa solo necesita la clasificacion derivada y no
el motivo personal que la origina.

### Tiempo, solapes y redondeo

El constructor debe obligar a elegir expresamente:

1. si las fechas son inclusivas y como se trata el dia final;
2. calendario natural o unidad convencional;
3. conversion, por ejemplo dias/30, dias/365 o dias/365x12;
4. momento y modo de redondeo: cada periodo, cada regla, seccion o total;
5. precision intermedia y precision final;
6. conservacion, acumulacion o descarte de restos;
7. tratamiento de periodos simultaneos: rechazar doble computo, sumar hasta un
   limite o elegir la puntuacion mayor;
8. tratamiento del periodo en curso y fecha maxima computable;
9. topes temporales y de puntuacion.

Estas opciones deben mostrarse tambien en lenguaje claro. No basta con guardar
una expresion matematica que solo entienda el programador.

### Formacion, titulaciones y otros meritos

El constructor no puede limitar estos meritos a descripcion y puntos. Debe
ofrecer opciones tipadas para:

- formacion recibida o impartida, asistencia o aprovechamiento;
- tarifa lineal por hora/credito o bandas de duracion;
- equivalencias separadas de creditos tradicionales, CFC y ECTS; se han
  observado conversiones distintas, por lo que no existira un valor global;
- duracion minima, numero maximo de actividades, vigencia o depreciacion por
  antiguedad;
- materias incluidas, transversales, excluidas y relacion con la categoria;
- entidades organizadoras, impartidoras, homologadoras o financiadoras
  admitidas;
- cursos repetidos: rechazar, elegir mayor carga, mayor puntuacion,
  aprovechamiento o permitir una actualizacion por cambio normativo;
- titulacion oficial, nivel MECES/EQF, RUCT, homologacion o equivalencia
  extranjera;
- titulo igual, superior o adicional al requisito, independencia entre titulos,
  dobles grados, rama y relacion con el puesto;
- computar uno, todos, solo el superior o solo el mas favorable;
- idiomas por MCERL, entidad acreditadora, nivel superior por idioma, vigencia y
  topes;
- oposiciones por ejercicio, fase, nota, oferta y cuerpo; docencia,
  investigacion, publicaciones, proyectos, comunicaciones, carrera profesional,
  habilitaciones, voluntariado u otros meritos cuando las bases los prevean.

La relacion «misma rama», «funciones homologas» o «curso relacionado» no se
resuelve comparando texto ni mediante IA. Procede de catalogos y tablas de
equivalencia aprobados, o de una decision motivada y revisable.

### Alcance del autobaremo declarado

La convocatoria debe seleccionar y publicar uno de estos comportamientos, sin
valor oculto por defecto:

| Modo | Comportamiento |
| --- | --- |
| Orientativo | La declaracion ayuda a presentar, pero no limita la puntuacion correcta que determine el organo. |
| Limite por apartado | La revision puede reducir, pero no superar lo autoasignado en cada apartado, con las recolocaciones que las bases permitan. |
| Limite total | La suma declarada limita el total, solo si las bases lo establecen de forma valida. |
| Propuesta calculada | La persona selecciona hechos y meritos y el sistema propone la cifra; la responsabilidad sobre la evidencia y la revision administrativa permanecen separadas. |

El formulario debe advertir del efecto antes de firmar y presentar. El modo
elegido no altera el derecho a subsanar o alegar que resulte aplicable.

## Obtencion de servicios internos de oficio

### Fuente correcta

El dato autoritativo debe ser el historial de servicios de Personal: altas,
nombramientos o contratos, categoria y puesto, relacion juridica, fechas,
jornada y resoluciones que alteren su computo. En el modelo del portal se
publica como periodos normalizados mediante referencias como
`service_period_ref`.

La jerarquia propuesta es:

1. expediente e historial administrativo de Personal;
2. nombramiento, contrato, toma de posesion, cese y resoluciones de jornada;
3. certificado de servicios prestados emitido por el organo competente;
4. nomina, afiliacion y vida laboral como reconciliacion o evidencia auxiliar;
5. fichaje/Cronos solo como comprobacion excepcional de incidencias, nunca como
   sustituto de la relacion administrativa ni como medida ordinaria de merito;
6. declaracion y documentos de la persona para periodos que la Diputacion no
   pueda obtener de oficio.

La nomina no debe convertirse en el maestro de experiencia. Puede detectar que
falta un periodo o que existe una incoherencia, pero el importe abonado no
acredita por si solo categoria, funciones, relacion, jornada ni dias
computables.

Si el producto corporativo de «nomina» contiene tambien las tablas maestras de
Personal, el conector puede leer de ese mismo producto. En ese caso debe usar
altas, contratos/nombramientos, ceses, categoria y jornada, no inferirlos de la
cuantia de los recibos. De este modo se cumple la obtencion automatica pedida
sin convertir una consecuencia economica en prueba unica del servicio.

### Conector y flujo propuestos

El nucleo consumira un puerto de servicios prestados, independiente de Oracle,
PostgreSQL, Active Directory, Peoplenet u otro producto. El adaptador interno:

1. recibe persona, finalidad, convocatoria y fecha de corte autorizadas;
2. consulta la fuente de Personal dentro de la red corporativa;
3. normaliza periodos y separa los cambios de jornada o situacion;
4. valida coherencia temporal, categorias y solapes;
5. devuelve una instantanea minimizada, versionada, firmada o atestada;
6. registra fuente, resultado, huella, autorizacion e instante;
7. permite reconciliar la instantanea con nominas sin enviar importes a Bolsa;
8. conserva un mecanismo de indisponibilidad, rectificacion y aportacion
   alternativa conforme al procedimiento aplicable.

Un fallo, ausencia o demora del conector nunca se transforma en «cero dias» o
«cero puntos». Produce un estado de dato no disponible, conserva la incidencia
y abre la via de comprobacion o aportacion que corresponda.

Una discrepancia material entre Personal y nomina queda como incidencia de
calidad y bloquea el baremo administrativo definitivo del periodo afectado hasta
que RRHH emita una reconciliacion o rectificacion trazable. No bloquea los
periodos independientes que sean validos.

El calculo de una solicitud queda ligado a esa instantanea. Una rectificacion
posterior crea otra version y un nuevo calculo; no reescribe el resultado que
sirvio para un listado anterior.

### Experiencia de la persona candidata

La VEC presentara tres bloques diferenciados:

- **Obtenido de oficio:** periodos internos de solo lectura, fuente, fecha,
  categoria, jornada computable y resultado desglosado.
- **Declarado por la persona:** experiencia externa o interna ausente, con
  evidencias y estado de verificacion.
- **Incidencias:** discrepancia, solicitud de rectificacion, imposibilidad de
  consulta, subsanacion y alegacion con sus plazos.

La persona podra excluir de su propuesta un periodo que considere incorrecto,
pero no editar un registro autoritativo de Personal. La correccion se tramita
en el sistema responsable y se incorpora mediante nueva version.

## Formulario interno para RRHH

### Navegacion propuesta

El formulario debe ser un asistente de administracion denso y revisable, no una
unica pagina de campos libres:

1. **Identificacion y fundamento:** convocatoria, expediente, categoria,
   version de bases, documento firmado, fecha de corte, organo competente y
   decision sobre negociacion de criterios generales cuando proceda.
2. **Estructura:** secciones, orden, maximos, minimos y peso en la puntuacion
   total.
3. **Reglas:** tipo de merito, condiciones, unidad, coeficiente, limites y texto
   explicativo.
4. **Experiencia y jornada:** matriz de procedencias/categorias, conversion
   temporal, fracciones, reducciones protegidas, solapes y restos.
5. **Acreditacion:** fuente de oficio, documentos necesarios, combinaciones
   validas, subsanabilidad y causas de rechazo.
6. **Incompatibilidades:** titulos requisito, cursos duplicados, experiencias
   concurrentes y eleccion mas favorable.
7. **Desempates:** cadena ordenada, sentido y dato de respaldo.
8. **Simulacion:** calculadora de casos anonimos, tabla de resultados esperados,
   avisos y comparacion con la version anterior.
9. **Revision:** informe legible que reproduce todas las reglas y diferencias,
   con aprobacion tecnica y juridica por actores distintos.
10. **Firma y publicacion:** representacion canonica, huella, documento de
    reglas, firmas y enlace uno a uno con las bases publicadas.

### Controles imprescindibles

- Vista de arbol y tabla de todas las reglas, con busqueda y filtros.
- Previsualizacion de la explicacion que vera la persona candidata.
- Deteccion de huecos, contradicciones, reglas inalcanzables y topes
  incompatibles.
- Alertas si una categoria de la convocatoria no tiene regla o equivalencia.
- Bloqueo de una prima por empleador propio sin informe juridico y motivacion
  funcional proporcionada, y de una regla de jornada sin analisis de igualdad.
- Unidades y coeficientes tipados; nunca numeros sin unidad.
- Diferencia semantica entre versiones, no solo diferencia de JSON.
- Importacion y exportacion en formato abierto para revision, sin que importar
  equivalga a publicar.
- Simulacion masiva con datos sinteticos y casos limite, nunca con candidatos
  reales en entornos de desarrollo.
- Pruebas obligatorias de 100 %, 50 %, 33,33 %, porcentaje arbitrario, cambio a
  mitad de periodo, ano bisiesto, solapes que suman menos, igual o mas del 100 %
  y titular/sustituto de una reduccion.
- Cuatro ojos: autor y aprobador distintos; perfil tecnico no sustituye al
  organo de seleccion ni al informe juridico.

### Ciclo de gobierno

`Borrador -> validado tecnicamente -> revisado juridicamente -> aprobado ->
firmado -> publicado -> sustituido/retirado`

Solo el borrador se edita. Cualquier vuelta desde una revision genera una nueva
revision del borrador. La publicacion exige todas las firmas y dependencias
vigentes en una unica operacion. Una fe de erratas o rectificacion crea una
sucesora y registra que solicitudes, calculos o listados deben revisarse.

## Autobaremacion y revision administrativa

### Reutilizacion en convocatorias futuras

El portal conserva un Registro Unificado de Meritos (RUM) de la persona y el
historial inmutable de sus presentaciones. La reutilizacion distingue cuatro
objetos que no deben confundirse:

| Objeto | Reutilizacion |
| --- | --- |
| Hecho o merito | Un titulo, curso o periodo puede proponerse en futuras convocatorias mientras siga vigente y resulte pertinente. |
| Evidencia | El documento o consulta de oficio se referencia sin volver a subir bytes; se comprueban integridad, firma, vigencia, revocacion y permiso de uso. |
| Validacion del hecho | La comprobacion de autenticidad o contenido puede aprovecharse si conserva alcance, fuente y vigencia; el nuevo organo puede revisarla cuando proceda. |
| Valoracion y autobaremo | Son exclusivos de una solicitud, convocatoria y version de reglas. Se conservan historicamente, pero nunca se copian como puntuacion de otro proceso. |

Al iniciar una nueva solicitud, el sistema:

1. propone los meritos existentes que podrian encajar;
2. refresca los servicios obtenidos de oficio hasta la nueva fecha de corte;
3. avisa de documentos caducados, revocados, sustituidos o insuficientes;
4. aplica exclusivamente las nuevas bases;
5. permite añadir, retirar o corregir la seleccion sin alterar expedientes
   anteriores;
6. exige revision, firma y registro de una nueva instantanea.

Una persona no debe volver a aportar el mismo fichero valido, pero tampoco se
la inscribe ni se presentan meritos automaticamente sin el tramite y la firma
que correspondan. La conservacion del RUM y de cada expediente seguira tablas
de retencion, bloqueo y supresion aprobadas; «reutilizable» no significa
conservacion indefinida ni acceso general entre modulos.

### Puntuacion que vera la persona en cada convocatoria

Si una persona participa en dos procesos con bases distintas, el mismo merito
puede producir puntuaciones distintas. La web no guarda una «puntuacion de la
persona» universal: calcula un resultado para la pareja exacta
`solicitud + version de reglas de convocatoria`.

La persona selecciona o declara hechos, periodos y evidencias; no escribe los
puntos. El sistema combina los datos obtenidos de oficio y los aportados,
aplica fecha de corte, jornada, solapes, conversiones, coeficientes y topes de
las bases y muestra el calculo completo. Una modificacion de los meritos obliga
a recalcular antes de firmar.

La interfaz separara sin ambiguedad:

| Resultado | Momento | Valor administrativo |
| --- | --- | --- |
| Simulacion | Mientras se prepara la solicitud. | Orientativa, mutable y no presentada. |
| Autobaremo presentado | Instantanea que la persona revisa, firma y registra. | Declaracion/propuesta provisional con el alcance vinculante u orientativo fijado en las bases. |
| Baremo administrativo provisional | Tras revisar RRHH u organo de seleccion requisitos, meritos y evidencias. | Actuacion provisional motivada, publicada con plazo de alegaciones cuando proceda. |
| Baremo definitivo | Tras resolver alegaciones y cerrar las decisiones vigentes. | Puntuacion oficial del proceso, firmada y enlazada con el listado o resolucion correspondiente. |

Los cuatro resultados usan el mismo calculador y la misma version publicada de
reglas. Las diferencias proceden de la instantanea presentada y de las
decisiones de validacion o rectificacion, no de cambiar coeficientes a mitad del
proceso. Cada pantalla mostrara su etiqueta, version, fecha, estado de firma y
si cabe alegacion o recurso.

Para cada merito, la explicacion debe mostrar:

- dato de origen y evidencia;
- regla exacta y version;
- unidades brutas;
- unidades elegibles tras fecha de corte y solapes;
- fraccion o politica de jornada aplicada;
- conversion temporal y restos;
- coeficiente;
- puntuacion antes de topes;
- tope aplicado y puntuacion resultante;
- estado: declarado, obtenido de oficio, pendiente, subsanable, validado,
  rechazado, rectificado o impugnado;
- motivo y decision administrativa vigente.

Como referencia de interaccion, el [manual oficial de autobaremo del
SAS/VEC](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/profesionales/ventanilla-electronica-de-profesionales/como-cumplimento-una-solicitud-de-autobaremo)
muestra un arbol jerarquico, asociacion de meritos, exclusion del titulo usado
como requisito, calculo explicado, bloqueo mientras se recalcula y firma de la
presentacion. Su [manual de
alegaciones](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/profesionales/ventanilla-electronica-de-profesionales/como-realizo-alegaciones-mi-baremo-provisional)
aporta el patron de motivo catalogado, exposicion, documentos, presentacion y
estado de tratamiento. Se reutilizan esos patrones, no su baremo ni sus datos.

El tecnico no escribe una puntuacion final arbitraria. Acepta, rechaza, acepta
parcialmente o pide subsanacion sobre hechos y evidencias, con motivo y firma.
El motor recalcula. Una excepcion verdaderamente necesaria requiere una
decision tipada, motivada, firmada, enlazada con la regla y visible en
auditoria; nunca un campo oculto de «ajuste».

Una inspeccion posterior puede sustituir una decision de validacion o rechazo
mediante otra decision firmada. Se conservan ambas, la causalidad, el actor, la
base juridica y todos los listados afectados.

## Casos de prueba que deben aprobar las bases antes de publicarse

Los cuatro primeros casos usan una regla hipotetica de 0,1 puntos por mes y se
incluyen solo para validar el producto:

| Caso | Entrada | Politica | Resultado esperado |
| --- | --- | --- | --- |
| J-01 | 12 meses al 100 % | Proporcional | 1,2 puntos. |
| J-02 | 12 meses al 50 % | Proporcional | 0,6 puntos. |
| J-03 | 12 meses al 50 % con atestacion protegida | Protegida integra | 1,2 puntos. |
| J-04 | 6 meses al 100 % y 6 al 50 % | Proporcional por tramos | 0,9 puntos. |
| J-05 | Dos trabajos simultaneos al 50 % | Acumulable hasta 100 % | El resultado coincide con un periodo al 100 %, sin doble computo superior al limite. |
| J-06 | Dos periodos simultaneos que encajan en reglas distintas | No acumulable, mayor valor | Se usa una sola vez la alternativa con mas puntuacion y se explica la descartada. |
| J-07 | Jornada del 50 % y umbral publicado del 50 % | Integra desde umbral | Se computa al 100 %; al 49,99 % se aplica la proporcion exacta. |
| J-08 | Titular y sustituto durante una reduccion protegida | Protegida integra/sustitucion real | El titular recibe el computo integro atestado y el sustituto su porcentaje certificado, sin compartir la causa. |
| T-01 | 61 dias acreditados | Meses completos por division entera entre 30 | 2 meses; el resto de 1 dia se trata como indiquen las bases. |
| T-02 | 61 dias acreditados | `dias / 365 x 12`, precision y redondeo publicados | El simulador debe reproducir el valor aprobado y mostrar cada operacion. |
| T-03 | Periodo abierto que supera la fecha de corte | Corte de convocatoria | Solo se computa hasta la fecha de corte. |
| M-01 | Titulo usado como requisito y otro titulo independiente superior | Incompatibilidad de requisito | El requisito no puntua; el segundo solo puntua si encaja en la regla publicada. |
| M-02 | Mismo curso presentado dos veces | Deduplicacion | Se aplica una sola vez la variante que las bases determinen. |
| C-01 | Suma de reglas superior al maximo de seccion | Tope | Se conserva el bruto, se muestra el exceso y se aplica el maximo exacto. |
| R-01 | Rectificacion de jornada tras listado provisional | Nueva instantanea | Nuevo calculo causal; el anterior permanece reproducible y se identifica el listado afectado. |

Cada convocatoria añadira ejemplos obtenidos de sus propias bases. El informe
de validacion guardara entradas, salida esperada, salida real, version del motor
y firmas de quienes los aprobaron.

## Arquitectura objetivo

El corte respeta la arquitectura hexagonal:

- `ConjuntoReglasBaremo` gobierna una version semantica e inmutable.
- Un compilador valida las reglas tipadas y produce un plan de calculo canonico;
  no ejecuta codigo aportado por usuarios.
- Un calculador puro recibe plan, meritos e instantaneas y devuelve un desglose
  determinista.
- Puertos separados obtienen servicios de Personal, catalogos, documentos,
  firmas, autorizacion, auditoria y persistencia.
- Adaptadores distintos pueden conectar PostgreSQL u Oracle y el sistema de
  Personal disponible sin alterar el dominio.
- API, web, CLI y MCP invocan los mismos casos de uso. MCP no obtiene una via
  privilegiada ni acceso a datos personales por ser usado por una IA.

Las puntuaciones se expresan en unidades enteras de precision definida, como
micropuntos. Las fracciones de jornada usan enteros o racionales exactos. No se
admite `float64` en decisiones administrativas reproducibles.

El lenguaje de reglas sera limitado y declarativo: comparaciones tipadas,
pertenencia a catalogos, tablas de decision, conversiones, multiplicacion por
coeficiente, suma, minimo, maximo, truncamiento y redondeos enumerados. Cualquier
nueva operacion semantica pasa por desarrollo, revision y pruebas; no se habilita
un interprete general desde administracion.

## Seguridad, trazabilidad y separacion de funciones

- El editor vive exclusivamente en el portal interno y red corporativa, con
  certificado, Kerberos y sesion vinculados conforme a la politica aprobada.
- Los candidatos solo reciben la proyeccion publica de reglas y su propio
  expediente autorizado.
- Toda lectura o cambio sensible exige accion, recurso, finalidad y campos
  positivos exactos; la ausencia de permiso deniega.
- Autor, revisor tecnico, revisor juridico, aprobador y publicador quedan
  diferenciados. La matriz final debe impedir autocontrol donde se exijan dos
  personas.
- Publicacion, rectificacion y decision de merito generan recibo, evento,
  auditoria y outbox en la misma transaccion.
- Bases, reglas, instantaneas, evidencias, calculos y listados se enlazan por
  referencias y huellas; no por nombres mutables.
- La bitacora es append-only, sellada, exportable y sometida a politica de
  conservacion. Los documentos permanecen en el almacen cifrado por conector.
- Las vistas de soporte y auditoria minimizan DNI, causas protegidas y
  documentos; el acceso excepcional queda justificado y revisado.
- La disponibilidad del sistema de Personal no puede abrir un atajo inseguro.
  Se registra la incidencia y se aplica la via alternativa prevista en las
  bases y el procedimiento.

## Situacion real del repositorio y huecos

El proyecto ya contiene decisiones utiles, pero este requisito no esta
terminado:

- `internal/shared/baremacion` aporta ya puntuacion, racional, fraccion de
  jornada, fecha e intervalo civil exactos, con limites defensivos y pruebas;
- `internal/modules/bolsa/domain/baremacion.go` dispone de puntuacion entera,
  referencias y decisiones append-only aprovechables.
- `docs/portal_vec/dominio_y_autobaremacion.md` ya separa autobaremo ciudadano,
  baremo administrativo, revision y rectificacion.
- `docs/portal_vec/estudio_pantallas_profesionales.md` ya reserva las pantallas
  `personal.servicios`, `bolsa.convocatorias`, `bolsa.meritos` y
  `bolsa.autobaremo`.
- `docs/portal_vec/nominas_personal_publico.md` ya establece que Bolsa consume
  servicios normalizados por referencia.

Sin embargo:

- `internal/candidate/domain/baremo.go` y
  `internal/candidate/usecases/baremo_rules.go` son legado con secciones y
  coeficientes fijos y aritmetica no apta para este objetivo.
- La pantalla actual de autobaremo y las reglas 0,2/0,1 del workspace son una
  demostracion, no el producto final.
- No existe aun el agregado gobernado completo de reglas ni el constructor de
  RRHH.
- No existe aun un modelo productivo de periodos de servicio/jornada ni el
  conector autoritativo con Personal.
- No estan implementadas las politicas configurables de jornada, reduccion,
  solape, conversion, restos y redondeo aqui definidas.

Por tanto, no debe mostrarse el prototipo como baremacion lista ni conectarse a
una convocatoria real.

## Plan de implantacion propuesto

1. **Cierre funcional y juridico:** Seleccion aporta dos o tres bases reales,
   Personal identifica la fuente autoritativa y Asesoria/Secretaria valida las
   interpretaciones y ejemplos.
2. **Dominio de reglas:** completar, sobre los tipos exactos ya implantados, el
   agregado, unidades, topes, incompatibilidades, canonico y calculador puro.
3. **Pruebas de conformidad:** transcribir ejemplos oficiales y crear casos
   golden, propiedades, limites, concurrencia y reproduccion historica.
4. **Servicios de oficio:** modelo de Personal, conector de solo lectura,
   normalizacion, reconciliacion, atestacion y rectificacion.
5. **Gobierno interno:** borradores, comparador, doble revision, firma,
   publicacion y auditoria.
6. **VEC candidata:** datos de oficio, experiencia externa, explicacion,
   autobaremo, subsanacion y alegaciones accesibles.
7. **Revision administrativa:** decisiones firmadas, recalculo, inspeccion,
   rectificacion y efecto sobre listados.
8. **Adaptadores reales y seguridad:** PostgreSQL inicial, posibilidad Oracle,
   almacen documental, identidad, firma, registro, notificacion, ENS y
   observabilidad.
9. **Piloto paralelo:** calcular una convocatoria cerrada o un conjunto
   anonimizado en paralelo con el procedimiento oficial y reconciliar todas las
   diferencias antes de abrir produccion.

## Decisiones que RRHH y el area juridica deben cerrar por convocatoria

- Unidad y formula exactas de cada merito.
- Fecha de corte y tratamiento del periodo en curso.
- Jornada parcial, reducciones protegidas, horas especiales y discontinuidad.
- Solapes y eleccion del periodo o regla mas favorable.
- Restos, precision y redondeo en cada fase.
- Categorias, funciones, empleadores y equivalencias admitidas.
- Fuente de oficio y evidencia alternativa si la consulta falla.
- Meritos requisito, incompatibilidades y duplicidades.
- Topes, minimos y desempates.
- Alcance vinculante del autobaremo declarado.
- Actores competentes para revisar, aprobar, rectificar y publicar.
- Plazos y efectos de subsanacion, alegacion y rectificacion.

El sistema puede forzar que estas decisiones se contesten y comprobar su
coherencia; no debe inventar la respuesta juridica.

## Criterios de aceptacion del estudio

Antes de comenzar la implementacion funcional deben cumplirse estas puertas:

- RRHH confirma que el inventario representa las bases que utiliza.
- Personal identifica la fuente real de servicios, su calidad y sus campos de
  jornada, sin depender solo de importes de nomina.
- Asesoria/Secretaria valida la interpretacion de jornada, reducciones,
  solapes, redondeos y autobaremo para la primera convocatoria.
- El DPD determina registro de actividad, informacion, plazos, accesos y
  evaluacion de impacto.
- Seguridad/Sistemas determina categoria ENS, separacion interna/externa,
  conectores, claves, custodia y continuidad.
- Se seleccionan bases piloto y se aprueba una bateria de resultados esperados.
- Se mantiene **NO-GO** para datos y decisiones reales hasta completar codigo,
  pruebas, revision independiente y adaptadores productivos.
