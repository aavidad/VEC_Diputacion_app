# Matriz normativa de Recursos Humanos

Fecha de corte: **16 de julio de 2026**.

Estado: **estudio de ingeniería normativa y NO-GO jurídico**. No sustituye el
informe de Secretaría/Asesoría Jurídica, RRHH, Intervención, Archivo, DPD,
Prevención o Seguridad. Antes de que una regla produzca efectos se verificará
el texto oficial consolidado, su ámbito y la competencia del órgano que la
aprueba.

## 1. Objeto y criterio de aplicabilidad

La matriz identifica las fuentes que condicionan el portal integral de RRHH y
traduce sus exigencias a capacidades y controles comprobables. No pretende
reproducir todo el ordenamiento ni decidir controversias jurídicas.

Cada requisito se clasificará durante el levantamiento definitivo como:

- **directo**: aplicable a la Diputación por su propio ámbito;
- **condicionado**: aplicable en los términos de la legislación básica,
  autonomía local, régimen del colectivo o desarrollo correspondiente;
- **provincial**: acuerdo, convenio, reglamento, resolución, bases o acto de la
  Diputación;
- **referencia**: patrón externo útil, pero sin valor normativo propio para la
  Diputación;
- **pendiente**: no puede activarse hasta consolidar la fuente o resolver una
  duda de vigencia, ámbito o competencia.

Una pantalla de administración no puede otorgar validez jurídica a cualquier
valor introducido. El sistema permitirá preparar, simular y revisar políticas;
solo una versión formalmente aprobada, firmada y publicada cuando proceda podrá
quedar activa.

## 2. Empleo público, organización y selección

| Materia | Fuente y alcance | Regla o control funcional | Evidencia mínima | Decisión técnica o validación pendiente |
| --- | --- | --- | --- | --- |
| Clases de personal | [TREBEP, arts. 8-13](https://www.boe.es/eli/es/rdlg/2015/10/30/5/con); [LBRL, arts. 89-104](https://www.boe.es/eli/es/l/1985/04/02/7/con); [Estatuto de los Trabajadores](https://www.boe.es/eli/es/rdlg/2015/10/23/2/con) | Distinguir funcionario de carrera e interino, laboral fijo, indefinido y temporal, eventual y, con régimen propio, dirección pública | relación, régimen, entidad, grupo/subgrupo o categoría, resolución o contrato, causas y periodos | `Persona` no contendrá un único «tipo de trabajador»; tendrá relaciones jurídicas históricas. Cargo electo, alto cargo y función directiva tampoco se confundirán con el vínculo |
| Función pública andaluza | [Ley andaluza 5/2023, art. 2](https://www.boe.es/eli/es-an/l/2023/06/07/5/con) | Aplicación al sector local en los términos compatibles con la legislación básica y la autonomía local | artículo concreto y título de aplicación local | elaborar aplicabilidad por precepto. El [Decreto 51/2025](https://www.juntadeandalucia.es/boja/2025/40/4) regula principalmente la Administración General de la Junta: es referencia salvo remisión o fundamento local comprobado |
| Negociación colectiva | TREBEP arts. 31-38; ET arts. 82-92; acuerdos y convenio provinciales | Las condiciones sujetas a negociación no pueden nacer de una configuración unilateral | mesa, acta, texto acordado, órgano aprobador, firma, publicación, vigencia y modificaciones | flujo borrador → negociación/informes → aprobación → publicación → activación; impedir activar una política sin la cadena exigida |
| Planificación y OEP | TREBEP arts. 69-70; LBRL arts. 90-91 | Relacionar necesidades, dotaciones, plantilla, OEP, convocatoria y ejecución; plazo general de tres años para ejecutar la OEP | plaza, dotación, ejercicio, acuerdo, publicación, convocatoria y seguimiento | una OEP no es una RPT, una convocatoria ni una bolsa; serán entidades relacionadas y versionadas |
| Principios de acceso | [Constitución, arts. 14, 23.2 y 103.3](https://www.boe.es/eli/es/c/1978/12/27/%281%29/con); TREBEP arts. 55-62; [RD 896/1991, arts. 2-9](https://www.boe.es/eli/es/rd/1991/06/07/896/con) | Igualdad, mérito, capacidad, publicidad, transparencia, imparcialidad, profesionalidad e independencia del órgano de selección | bases, requisitos a la fecha de corte, anuncios, composición y recusaciones, pruebas, actas, puntuaciones, alegaciones y recursos | expediente probatorio completo y proyección pública minimizada; ningún asistente de IA sustituirá al órgano competente |
| Reserva por discapacidad | TREBEP art. 59 | Reserva mínima del 7 % en la OEP, al menos 2 % para discapacidad intelectual, en los términos del precepto | cálculo de plazas, cupos, adaptaciones, resultados y remanentes | configuración por OEP/convocatoria y accesos especialmente protegidos a la documentación; no inferir diagnósticos |
| Funcionario interino | TREBEP art. 10 y disposición adicional 17.ª | Separar vacante, sustitución, programa temporal y acumulación de tareas; controlar límites y causa efectiva | causa, plaza o persona sustituida, inicio, fin previsto, programa, periodos previos, convocatoria y cese | políticas por causa y vigencia. Confirmar la norma habilitante local para cualquier ampliación de programa; no reutilizar una duración universal |
| Laboral temporal | ET arts. 8, 12 y 15; TREBEP art. 11.3 | Causalidad laboral, jornada y contrato, además de igualdad, mérito, capacidad y publicidad en la selección | contrato, causa, modalidad, jornada, convenio, puesto, periodos y transformaciones | no aplicar automáticamente reglas de interinidad funcionarial a contratos laborales |
| Bolsa provincial 2026 | [Reglamento BOP de 16-01-2026](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1768521622300-final-680c0e98.pdf), arts. 1-12; [modificación de 14-05-2026](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1778713220300-final-241a2c51.pdf) | Prioridad de bolsas procedentes de OEP; bolsas específicas por inexistencia, agotamiento o caducidad; vigencia general de cinco años; autobaremo corregible; llamamientos y adhesión de organismos | origen, categoría, vigencia, orden, publicaciones, estados, intentos de contacto, respuesta, renuncia, justificación y excepción | modelar ya con prueba íntegra del llamamiento. Los errores de numeración interna detectados en el texto deben aclararse antes de generar referencias automáticas |
| Indisponibilidad tras cese | Reglamento provincial 2026, art. 9 | Indisponibilidad global de cinco meses tras finalizar una relación temporal; nueve meses en acumulación de tareas; excepciones regladas para ciertos ofrecimientos de vacante | acto y causa de cese, periodos, fecha calculada, bolsas afectadas y excepción motivada | política global por persona, versionada. No codificar la formulación coloquial «trabajó nueve meses y descansa X» ni copiar el estado a cada bolsa |
| Bases y autobaremo | TREBEP arts. 55-61; Reglamento provincial, art. 5; bases específicas | Experiencia, jornada, títulos, cursos, equivalencias, topes, incompatibilidades y fecha de corte dependen de cada versión de bases; el autobaremo no es el resultado oficial | bases firmadas, mérito, evidencia, periodo, fracción de jornada, cálculo, decisión técnica, firma, alegación y rectificación | motor exacto, determinista y explicable; aceptación/rechazo por unidad de mérito y rectificaciones inmutables; nunca coeficientes globales ocultos |
| Promoción interna | TREBEP art. 18, en particular 18.2 | Requiere, entre otras condiciones, al menos dos años de servicio activo en el subgrupo inferior para la vía funcionarial regulada | vínculo, subgrupo de origen/destino, antigüedad computable, convocatoria, pruebas y resolución | no confundir con grado personal, carrera horizontal, nivel del puesto ni progresión laboral |
| Progresión laboral provincial | [Convenio laboral 2006, art. 9](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/CONVENIO-2006.pdf) y modificaciones | El texto contempla para determinada progresión económica tres años continuos o cuatro interrumpidos y curso/prueba | colectivo, categoría, antigüedad, interrupciones, convocatoria, curso, prueba y acto de reconocimiento | obtener texto consolidado y cadena de modificaciones. No convertirlo en una regla general de «examen cada tres años» |
| Carrera y evaluación | TREBEP arts. 16-20 | Carrera horizontal/vertical, promoción y evaluación son instituciones diferentes | sistema aplicable, colectivo, tramo, méritos, evaluación, órgano y resolución | dominios y reglas separados; una detección de elegibilidad solo inicia o propone el expediente |
| RPT y plantilla | TREBEP art. 74; LBRL art. 90; [TRRL, arts. 126-177](https://www.boe.es/eli/es/rdlg/1986/04/18/781/con) | RPT ordena puestos y plantilla recoge plazas dotadas; no son sinónimos ni acreditan por sí solas la ocupación | documento, acuerdo, publicación, versión, unidad, plaza/dotación, puesto, grupo, escala, provisión, nivel, complementos y vigencia | modelo bitemporal y no destructivo; la persona/ocupación procede del Registro de Personal y actos, no de una deducción de la fila RPT |
| Vacantes históricas | normas de personal, RPT, plantilla, presupuesto y actos de cada periodo | Sin ocupante, vacante presupuestada, reservada, cubierta temporalmente y disponible para cobertura son estados diferentes | dotación, ocupación, reserva, situación, crédito, procesos y fechas | consulta derivada por fecha; prohibido el booleano genérico `libre`; RRHH debe validar estados y causas |
| Provisión y movilidad | TREBEP arts. 78-84; LBRL art. 101; acuerdos provinciales 2025-2026 | Concurso, libre designación, adscripción provisional y movilidades tienen requisitos, permanencia y baremos propios | convocatoria, área funcional, baremo, permanencia, solicitud, valoración, acta y resolución | incorporar el texto definitivo de baremo, permanencia, movilidad provisional y áreas funcionales antes del motor |
| Habilitación nacional | [RD 128/2018](https://www.boe.es/eli/es/rd/2018/03/16/128/con) | Secretaría, Intervención y Tesorería tienen reserva y provisión específicas | subescala, clasificación, puesto, provisión y registro | no tratarlos como puestos ordinarios configurables |

## 3. Jornada, permisos, turnos y teletrabajo

| Materia | Fuente y alcance | Regla o control funcional | Evidencia mínima | Decisión técnica o validación pendiente |
| --- | --- | --- | --- | --- |
| Jornada y turnos | TREBEP arts. 47-51; ET arts. 34-38; [Reglamento provincial de tiempo de trabajo](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/REGLAMENTO-TIEMPO-DE-TRABAJO.pdf) | Cálculo por entidad, vínculo, centro, calendario, ciclo y periodo | adscripción, jornada teórica, cuadrante, turno, descansos, marcajes, incidencias, autorizaciones y saldo | obtener reglamento consolidado y normas organizativas específicas, en particular centros sociales, antes de activar coeficientes |
| Festivos y compensación | Reglamento provincial, especialmente arts. 2-5 y 11; ET art. 37.2 para laborales | Relacionar festivo aplicable, turno previsto, trabajo real y derecho reconocido a descanso o pago | fuente del festivo, planificación, marcajes, autorización, saldo generado, disfrute o liquidación | el texto localizado contiene compensaciones específicas; no fijarlas hasta consolidar modificaciones de 2010, 2015, 2019 y reglas de centro |
| Horas extraordinarias | ET art. 35 para laborales; reglamento provincial art. 11; circular WCRONOS | Autorización, hecho real, causa, validación y forma de compensación | intervalo, marcajes, petición, autorizadores, política y liquidación | libro de tiempo inmutable; no trasladar el límite laboral de 80 horas a todo funcionario sin fuente propia |
| Control horario | ET art. 34.9 para laborales; régimen del empleo público; [circular WCRONOS](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/2018-circular_anotacion-realizacion-de-horas_ext_wcronos.pdf) | Conservar marcaje original y toda corrección | instante, dispositivo/origen, operador, motivo, solicitante, superior, validador y asiento compensatorio | nunca sobrescribir el fichaje original; acceso interno y minimizado |
| Permisos y vacaciones | TREBEP arts. 48-51; ET arts. 37-38; acuerdo/convenio y resoluciones provinciales | Derechos y justificantes pueden variar por colectivo, causa y fecha | hecho causante, documentación mínima, parentesco si procede, saldo, sustitución, decisión y recurso | catálogo administrable, pero cada entrada exige fuente, colectivo, vigencia, órgano y circuito; no usar una lista única para laborales y funcionarios |
| Vacaciones al cese en revisión | TREBEP art. 50; Reglamento provincial art. 17.4; [inicio de revisión de oficio de 2026](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/AREAS-Participacion-Publica-Normativa-Documentos-en-exposicion-publica-RRHH/2026_PES_01_002036_Iniciacion-de-procedimiento-de-revision-de-oficio.pdf) | El Pleno inició la posible declaración de nulidad del precepto provincial que reconocía vacaciones completas al personal fijo que cesa, con independencia de la fecha | acuerdo de inicio, alegaciones, dictamen, resolución final, publicación y fecha de efectos | no se localizó resolución final a la fecha de corte; política `en_revision`, no activable ni extrapolable a cálculos históricos sin el acto aplicable |
| Teletrabajo | TREBEP art. 47 bis; [instrucciones provinciales de 26-04-2024](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/Resolucion_instrucciones_teletrabajo_web.pdf) | Voluntario, reversible, autorizado, compatible con presencialidad y sometido a objetivos/condiciones | solicitud, tareas, plan, medios, días, objetivos, evaluación, revisiones y revocación | la DA 2.ª de la [Ley 10/2021](https://www.boe.es/eli/es/l/2021/07/09/10/con) excluye al personal laboral de AAPP de su régimen general; aplicar la fuente pública correspondiente |
| Geolocalización laboral | LOPDGDD art. 90; ET arts. 20.3 y 20 bis; RGPD arts. 5, 6, 13 y 35 | Necesidad, proporcionalidad, información expresa y control limitado a la finalidad laboral | dispositivo/vehículo, finalidad, franjas, información entregada, accesos, conservación, evaluación de impacto e incidencias | enclave interno; limitar o desactivar fuera del uso laboral; nunca exponer al portal externo, bot, analítica general o jefaturas sin competencia |
| Calendario histórico | ET arts. 34.6 y 37.2; disposiciones anuales estatales, autonómicas y locales | Conservar el calendario aplicable por territorio, centro, colectivo y año | publicación oficial, ámbito, día, traslado, versión y corrección | no reconstruir un año pasado con el calendario actual; separar día hábil administrativo, festivo, apertura y jornada de persona |

## 4. Nómina, Seguridad Social y fiscalidad

El análisis provincial detallado de jornadas especiales se documenta en
[Turnos, festivos, disponibilidades y compensaciones](turnos_festivos_y_compensaciones.md).

| Materia | Fuente y alcance | Regla o control funcional | Evidencia mínima | Decisión técnica o validación pendiente |
| --- | --- | --- | --- | --- |
| Retribuciones públicas | TREBEP arts. 21-30; LBRL art. 93; [RD 861/1986](https://www.boe.es/eli/es/rd/1986/04/25/861/con); presupuestos y acuerdos anuales | Conceptos, tablas y efectos por colectivo, puesto, jornada, periodo y acto | concepto, fuente, tabla, puesto, jornada, incidencia, cálculo, aprobación, pago y retroactividad | dominio de nómina separado y bitemporal; importes y topes versionados, nunca constantes; una nómina cerrada se rectifica, no se recalcula destructivamente |
| Seguridad Social | [LGSS](https://www.boe.es/eli/es/rdlg/2015/10/30/8/con); normas anuales, incluida [Orden PJC/297/2026](https://www.boe.es/eli/es/o/2026/03/30/pjc297/con) | Bases, tipos, topes, liquidaciones, altas, bajas y comunicaciones según periodo | versión de tablas, afiliación, relación, bases, tramos, liquidación, fichero, respuesta y conciliación | adaptador con TGSS y tablas gobernadas; validar el régimen aplicable a cada relación |
| IRPF | [Ley 35/2006](https://www.boe.es/eli/es/l/2006/11/28/35/con); [RD 439/2007](https://www.boe.es/eli/es/rd/2007/03/30/439/con) y cambios vigentes | Retención calculada con datos y tablas del periodo | declaración, circunstancias comunicadas, algoritmo/versión, tipo, regularización y certificado | compartimento fiscal; mínimo acceso y auditoría reforzada; no propagar datos familiares a otros módulos |
| Conciliación | reglas presupuestarias, contables, fiscales y de Seguridad Social aplicables | Separar cálculo, aprobación, fiscalización, contabilización, pago y presentación | lote, total, diferencias, aprobadores, firma, asientos, pagos, respuestas externas y correcciones | la primera sustitución de nómina exigirá ciclos paralelos, comparables y firmados antes de cortar el sistema anterior |

## 5. Prevención, igualdad y materias especialmente protegidas

El diseño detallado de Nómina, Acción Social, Formación, Salud Laboral,
Igualdad, Acoso, Disciplina, Incompatibilidades y Representación se recoge en
[Materias económicas, reservadas y relaciones laborales](materias_reservadas_economicas_y_relaciones_laborales.md).

| Materia | Fuente y alcance | Regla o control funcional | Evidencia mínima | Decisión técnica o validación pendiente |
| --- | --- | --- | --- | --- |
| Prevención de riesgos | [Ley 31/1995, arts. 14-31](https://www.boe.es/eli/es/l/1995/11/08/31/con) | Evaluar puestos, aplicar medidas, formar y vigilar la salud en los límites legales | evaluación del puesto, riesgo, medida, formación, revisión y únicamente conclusión de aptitud para RRHH | datos clínicos separados física y lógicamente; no guardar diagnósticos en Personal, RPT, Cronos ni logs generales |
| Vigilancia de la salud | Ley 31/1995, art. 22 | RRHH recibe solo conclusiones sobre aptitud o necesidad de medidas preventivas | consentimiento o habilitación, profesional sanitario, prueba, conclusión y destinatarios | bóveda sanitaria con accesos propios, conservación propia y sin búsqueda general |
| Movilidad por salud | [Procedimiento provincial de 2025](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/2025_BOP_Procedimiento_movilidad.pdf) | Usar conclusión de compatibilidad y puestos disponibles sin revelar la historia clínica | solicitud, conclusión mínima, puestos compatibles, orden, revisión y resolución | PRL emite un atributo minimizado y firmado; Personal/RPT no acceden al expediente sanitario |
| Igualdad | [LO 3/2007](https://www.boe.es/eli/es/lo/2007/03/22/3/con); TREBEP DA 7.ª; [Ley andaluza 12/2007](https://www.boe.es/eli/es-an/l/2007/11/26/12/con); [Ley 15/2022](https://www.boe.es/eli/es/l/2022/07/12/15/con) | Plan negociado, evaluación anual, registro, diagnóstico, impacto, indicadores, igualdad retributiva y no discriminación múltiple | metodología, negociación, medidas, responsables, plazos, evaluación, remisión y seguimiento | proyecciones agregadas/seudonimizadas; confirmar nuevo plan provincial: el [Plan de Igualdad 2022](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/Plan-de-Igualdad_2022.pdf) declaró cuatro años desde 27-01-2022 y no se ha localizado sustitución pública |
| No discriminación y diversidad | [Ley 15/2022](https://www.boe.es/eli/es/l/2022/07/12/15/con); [Ley 4/2023](https://www.boe.es/eli/es/l/2023/02/28/4/con); RDL 1/2013 | Evitar requisitos, formularios, decisiones y algoritmos discriminatorios; aplicar ajustes razonables | justificación de requisitos, adaptaciones, incidencias, revisión de impacto y resolución | revisión de bases y reglas antes de publicar; campos de identidad inclusivos; nunca preguntar salud sin necesidad y fundamento |
| Acoso y violencia | normativa estatal, plan/protocolos provinciales y convenio aplicables | Canal protegido, medidas cautelares, investigación competente y protección frente a represalias | comunicación, acceso, custodia, medidas, instructores, plazos y resolución | expediente aislado; no aparecerá en el expediente general ni será visible para la jefatura salvo actuación expresamente atribuida |
| Afiliación sindical | RGPD art. 9; LOPDGDD; libertad sindical y normas de representación | Uso únicamente para finalidades habilitadas, representación y descuentos autorizados | procedencia, finalidad, acceso, cesión y conservación | categoría especial separada; no inferir ni exponer en analítica nominal |
| Crédito y representación sindical | LOLS; TREBEP arts. 31-46; ET arts. 61-81; [acuerdo provincial de 2020](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/Acuerdo-regulacion-creditos-horarios-sindicales.pdf) | Mandatos, órganos, suplencias, créditos, cesiones, liberaciones, preavisos y consumo con historia | mandato, unidad, acto, crédito, cesión, periodo, ausencia autorizada y corrección | el acuerdo de 2020 se vinculó a aquel mandato; no codificar nombres, horas o liberaciones como vigentes en 2026 |

## 6. Procedimiento administrativo, documentos y archivo

| Materia | Fuente y alcance | Regla o control funcional | Evidencia mínima | Decisión técnica o validación pendiente |
| --- | --- | --- | --- | --- |
| Interesado, identidad, representación y registro | [Ley 39/2015, arts. 4-17](https://www.boe.es/eli/es/l/2015/10/01/39/con) | Identificar, acreditar representación, registrar y emitir recibo con fecha/hora y documentos | identidad, representación, asiento, anexos, sello temporal, recibo y canal | ventanilla conectada al registro oficial; la cuenta de portal no sustituye el asiento ni la representación |
| Notificaciones | Ley 39/2015 arts. 40-46; [RD 203/2021](https://www.boe.es/eli/es/rd/2021/03/30/203/con) | Puesta a disposición, acceso/rechazo y contenido probables | acto, destinatario, dirección, puesta a disposición, acceso/rechazo, avisos y errores | correo, SMS o Telegram son avisos; no sustituyen la notificación ni deben contener información sensible |
| Actuación administrativa automatizada | [Ley 40/2015, arts. 41-44](https://www.boe.es/eli/es/l/2015/10/01/40/con) | Identificar órganos responsables de definición, supervisión, auditoría e impugnación | habilitación, programa/regla, versión, entradas, salida, órgano, sello y recurso | si no existe habilitación, el cálculo será propuesta humana; un motor determinista no queda exento de estas garantías |
| Firma, sello y cofirma | Ley 39/2015 arts. 9-12 y 26; Ley 40/2015 arts. 42-43; [eIDAS](https://eur-lex.europa.eu/eli/reg/2014/910/); [Ley 6/2020](https://www.boe.es/eli/es/l/2020/11/11/6/con) | Distinguir firma personal, sello de órgano, sello de tiempo, CSV y validación a largo plazo | firmante, certificado, política, formato, validación, secuencia, instante y conservación | AutoFirma será adaptador; QR, código de barras, imagen o marca de origen no son firma |
| Documento electrónico | Ley 39/2015 arts. 26-28 y 53; [ENI](https://www.boe.es/eli/es/rd/2010/01/08/4/con); [NTI de Documento Electrónico](https://www.boe.es/eli/es/res/2011/07/19/%283%29/con) | Original lógico, formato, metadatos, huella, firmas, copias y derecho de acceso | contenido, formato, metadatos, hash, productor, firma, derivaciones y estado | generadores de PDF, PDF/A, ODT, DOCX, texto, CSV, JSON u otros detrás de puertos; declarar el original jurídico y formatos de preservación |
| Expediente y archivo | Ley 39/2015 art. 17; [NTI de Expediente Electrónico](https://www.boe.es/eli/es/res/2011/07/19/%284%29/con); [Ley andaluza 7/2011, art. 49](https://www.boe.es/eli/es-an/l/2011/11/03/7/con) | Índice, documentos, firmas, relaciones, transferencia, conservación, bloqueo y eliminación autorizada | serie, expediente, índice, documentos, metadatos, firmas, tabla de valoración y actuaciones | un almacenamiento S3 compatible no es por sí solo archivo electrónico; necesita gobierno archivístico y preservación |
| Digitalización | [NTI de Digitalización](https://www.boe.es/eli/es/res/2011/07/19/%282%29/con) | Diferenciar escaneo, digitalización garantizada y copia auténtica | operador, dispositivo, calidad, metadatos, cotejo, firma/sello y original | no etiquetar cualquier PDF escaneado como copia auténtica |
| Intercambio registral | [SICRES 4.0](https://www.boe.es/eli/es/res/2021/07/22/%282%29/con), ENI y RD 203/2021 | Intercambiar asiento, origen/destino, anexos, recibos, estados y errores | mensaje, versión, identificadores, anexos, acuse, estado y conciliación | puerto SIR/SICRES; MCP no sustituye un protocolo administrativo oficial |
| Incompatibilidades | [Ley 53/1984](https://www.boe.es/eli/es/l/1984/12/26/53/con); [RD 598/1985](https://www.boe.es/eli/es/rd/1985/04/30/598/con) | Tramitar actividad principal/secundaria, informes y resolución | actividades, jornadas, retribuciones, declaración, informes, decisión y publicidad legal | expediente propio y proyección pública separada; validar alcance concreto del procedimiento reglamentario en entidad local |
| Régimen disciplinario | TREBEP arts. 93-98; Ley andaluza 5/2023 arts. 165-177; ET y convenio para personal laboral | Legalidad, tipicidad, proporcionalidad, prescripción, audiencia, antirrepresalias y separación instructor/resolutor | hecho, vínculo, norma, fechas, medidas, instructor, prueba, audiencia, propuesta, resolución, recurso, inscripción y cancelación | el convenio provincial de 2006 menciona seis meses y legislación derogada, mientras la ley andaluza contempla hasta doce; **NO-GO** hasta criterio jurídico por vínculo; nunca inferencias automáticas desde fichajes |

## 7. Protección de datos, seguridad, transparencia y acceso

| Materia | Fuente y alcance | Regla o control funcional | Evidencia mínima | Decisión técnica o validación pendiente |
| --- | --- | --- | --- | --- |
| Protección de datos | [RGPD, arts. 5, 6, 9, 12-22 y 24-39](https://eur-lex.europa.eu/legal-content/ES/TXT/?uri=CELEX:02016R0679-20160504); [LOPDGDD](https://www.boe.es/eli/es/lo/2018/12/05/3/con) | Licitud, finalidad, minimización, exactitud, conservación, seguridad, responsabilidad proactiva, derechos, RAT, EIPD y DPD | finalidad, base, atributo, procedencia, acceso, cesión, conservación, derechos, riesgo e incidente | compartir `persona_id` no autoriza compartir datos; vistas por finalidad. En RRHH no se usará consentimiento genérico para sustituir obligación legal o interés público |
| Registro provincial de actividades | [RAT de la Diputación](https://www.dipgra.es/servicios/areas/transparencia/portal-de-transparencia/g-actividades-de-tratamiento-de-datos-personales/registro-de-actividades-de-tratamiento/) | Inventariar finalidades de personal, nómina, permisos, selección, presencia, geolocalización, formación, acción social y PRL | ficha de actividad, bases, colectivos, datos, destinatarios, conservación y medidas | el RAT actual prueba el alcance, no una autorización indiferenciada; actualizarlo y valorar separación de actividades antes de datos reales |
| ENS | [RD 311/2022, arts. 5-13, 17, 24-31 y 38-41](https://www.boe.es/eli/es/rd/2022/05/03/311/con) | Categorización por impacto, riesgos, responsables, inventario, autorización, protección, incidentes, continuidad y auditoría | declaración de aplicabilidad, categoría, riesgos, responsables, medidas, evidencias, incidentes y auditorías | no declarar informalmente «seguridad máxima». Aspiración ALTA y compartimentos de alto impacto deben formalizarse; auditoría al menos bienal y tras cambios relevantes |
| Auditoría | ENS; RGPD arts. 5.2 y 24; Ley 40/2015 | Separar trazabilidad jurídica, log de seguridad y telemetría técnica | actor, sujeto, acción, finalidad, expediente, regla, antes/después, tiempo fiable, origen y resultado | almacenamiento resistente a manipulación; acceso restringido; no copiar documentos, diagnósticos o secretos en logs |
| Transparencia | [Ley 19/2013, arts. 5-22](https://www.boe.es/eli/es/l/2013/12/09/19/con); [Ley andaluza 1/2014](https://www.boe.es/eli/es-an/l/2014/06/24/1/con) | Publicar información exigible con límites de protección de datos y demás derechos | contenido, fundamento, versión, aprobación, fecha, anonimización y retirada | proyecciones públicas independientes; nunca abrir el expediente personal para cumplir transparencia |
| Accesibilidad | [RD 1112/2018](https://www.boe.es/eli/es/rd/2018/09/07/1112/con); [RDL 1/2013](https://www.boe.es/eli/es/rdlg/2013/11/29/1/con); [Ley 6/2022](https://www.boe.es/eli/es/l/2022/03/31/6) | Accesibilidad de web, móvil, formularios y documentos; declaración, revisión y canal de comunicación | evaluación, declaración, incidencias, respuesta y correcciones | temas y acentos no podrán romper semántica, teclado, lector de pantalla o contraste. Audio complementa, no sustituye, la accesibilidad |
| Inteligencia artificial en empleo | [Reglamento UE 2024/1689, arts. 4, 26, 27, 50 y anexo III](https://eur-lex.europa.eu/eli/reg/2024/1689/oj), con aplicación escalonada | Determinados usos para selección, promoción, asignación, evaluación o vigilancia laboral pueden ser de alto riesgo y exigen obligaciones reforzadas | inventario, finalidad, clasificación, datos, validación, supervisión, registros, evaluación de derechos y comunicación | inicialmente el bot solo consultará información pública. Prohibido decidir admisión, puntuación, orden, promoción, disciplina, nómina o salud mediante el bot; revisar régimen aplicable antes de cualquier IA futura |

## 8. Fuentes provinciales que gobiernan el diseño

El [portal oficial de normativa de RRHH](https://www.dipgra.es/servicios/areas/transparencia/portal-de-transparencia/a-informacion-institucional-y-organizativa/a3-personal/normativa-de-recursos-humanos/)
recoge, entre otros, acuerdo de funcionarios, convenio laboral y modificaciones,
planificación, selección y bolsas, provisión, acción social, formación, tiempo de
trabajo, seguridad y salud e igualdad.

Estas fuentes no se cargarán como una colección plana de PDF. Para cada materia
se construirá una cadena normativa que indique:

- documento original, publicación y huella;
- acto y órgano aprobador;
- modificaciones, suspensiones, reanudaciones y derogaciones;
- ámbito personal, territorial, organizativo y temporal;
- fecha de vigencia y fecha de efectos;
- artículos o apartados materializados;
- estado de consolidación y responsable de validarlo.

Son especialmente prioritarios:

- la [Estrategia de modernización de 2025](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/2025_19_2_CERTIFICADO_NA_4_JUNTA_GOBIERNO_18_02_2025.pdf),
  que exige inventario, simplificación, bolsas suficientes, automatización de
  certificados, planificación e indicadores;
- el Reglamento de selección temporal y bolsas de 2026 y su modificación;
- RPT, plantilla y sus modificaciones históricas;
- acuerdo de funcionarios y convenio laboral con toda su cadena de cambios;
- reglamento de tiempo de trabajo y reglas específicas de centros;
- baremos y acuerdos de provisión/movilidad 2025-2026;
- acción social, formación, prevención, protocolos y plan de igualdad vigente.

## 9. Registro de políticas exigido al producto

Toda regla administrable conservará como mínimo:

```text
identificador y materia
entidad y colectivo
centro, unidad, plaza o puesto cuando proceda
fuente jurídica o convencional y artículo
documento original, publicación y huella
versión y versión sustituida
vigencia desde/hasta
efectos desde/hasta
estado: borrador, informada, aprobada, publicada, suspendida o derogada
órgano competente, responsables y firmas
parámetros tipados y unidades exactas
casos de prueba y resultado esperado
explicación del cálculo
impacto sobre expedientes abiertos y cerrados
```

Las expresiones serán declarativas, tipadas y acotadas; no se admitirá código,
SQL ni scripts introducidos desde la interfaz. Las políticas publicadas serán
inmutables. Una corrección creará otra versión y declarará expresamente desde
cuándo afecta y qué expedientes deben revisarse.

## 10. Reglas que no se programarán todavía

Hasta obtener consolidación y validación institucional no se fijarán:

- una regla general de tres años para «subir de nivel»;
- coeficientes universales de noche, turno, festivo u hora extraordinaria;
- jornada y ciclos de todos los centros sociales;
- un catálogo idéntico de permisos para laborales y funcionarios;
- permanencias comunes a toda provisión o promoción;
- un baremo general extrapolado a todas las convocatorias;
- importes retributivos, cotizaciones, retenciones o complementos;
- una causa binaria de plaza o puesto «libre»;
- acceso de jefatura basado solo en una relación informal;
- conservación indefinida de expedientes, documentos o registros;
- decisiones de empleo, puntuación o vigilancia mediante IA.

## 11. Documentación institucional pendiente

1. Texto consolidado del acuerdo de funcionarios y todas sus modificaciones.
2. Texto consolidado del convenio laboral de 2006 y cambios posteriores.
3. Reglamento de tiempo de trabajo consolidado y normas de centros sociales.
   Debe incluir la resolución del procedimiento de revisión de oficio del
   artículo 17.4 iniciado en febrero de 2026.
4. Baremo de provisión, permanencia, movilidad provisional y áreas funcionales.
5. RPT, plantilla y modificaciones históricas estructuradas, con sus acuerdos.
6. Catálogo de permisos, jornadas, turnos, complementos y códigos de nómina por
   colectivo y periodo.
7. Plan de Igualdad posterior a enero de 2026 o resolución sobre su vigencia.
8. Protocolos vigentes de acoso, prevención, movilidad por salud y emergencias.
9. Delegaciones de competencia, firma y suplencia.
10. Tablas de valoración, conservación, transferencia y expurgo del Archivo.
11. Política de seguridad, categorización ENS, análisis de riesgos y EIPD.
12. Diccionario e interfaces de Registro de Personal, GINPIX, CONVOCA,
    WCRONOS, Portale/Nómina, MOAD, AD, contabilidad y sistemas externos.

## 12. Decisión para Núcleo y Bolsa

Se puede continuar Núcleo y Bolsa sin desarrollar todavía el resto de RRHH si
se conservan desde ahora:

- persona canónica con vistas separadas por finalidad;
- relaciones jurídicas y organizativas históricas;
- vigencia y tiempo de conocimiento;
- registro de políticas y autoridades;
- documentos, expedientes, firma, notificación y auditoría compartidos;
- rectificaciones por nuevos actos, nunca borrado del anterior;
- autorización positiva por rol, entidad, unidad, relación, expediente y
  finalidad;
- contratos neutrales entre RPT, Personal, Bolsa, Cronos, Nómina, PRL y
  Archivo;
- ausencia de coeficientes o reglas de esos módulos dentro del núcleo.

Así se evita una refactorización estructural futura sin anticipar reglas que
todavía no han sido consolidadas o aprobadas.
