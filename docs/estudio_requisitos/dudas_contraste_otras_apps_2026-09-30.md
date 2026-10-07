# Contraste público de las dudas de RRHH

Fecha: 30/09/2026. Fuente de las preguntas: [`dudas.md`](../../dudas.md),
preguntas 1 a 39.

Este contraste ayuda a decidir cómo configurar VEC mientras RRHH confirma su
procedimiento. Las referencias describen prácticas observables en otras
administraciones o productos; no aprueban una regla para la Diputación. Cuando
una fuente pública de la propia Diputación concreta un extremo, se señala para
que RRHH confirme su vigencia y su aplicación al circuito consultado.

Las propuestas de abajo son valores o comportamientos iniciales **configurables**,
con versión, responsable y fecha de efecto. Una propuesta sin confirmación no
debe producir por sí sola una exclusión, una adjudicación, una firma, un plazo
legal ni otro efecto administrativo. La aplicación conserva el dato y la regla
que se aplicaron a cada expediente; los cambios posteriores dejan historia y
auditoría. Las autoridades de identidad, Persona, módulos y publicaciones
siguen separadas según las especificaciones de VEC.

Para cada duda se indican de tres a cinco referencias públicas directas. Algunas
aparecen en varias dudas porque documentan capacidades distintas. «Contraste»
resume lo que permiten comprobar; «Propuesta para VEC» es una inferencia de
diseño; «Por confirmar» conserva la decisión de RRHH, Sistemas o Informática.
No se han consultado datos personales ni fuentes privadas.

El [reglamento provincial publicado el 16/01/2026](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1768521622300-final-680c0e98.pdf)
tiene una [corrección de 27/01/2026](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1769472066115-final-42c9b1bc.pdf)
que rectifica la fecha del pleno, y una [modificación de 14/05/2026](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1778713220300-final-241a2c51.pdf)
que añade la posible adhesión de organismos autónomos. Se han contrastado las
tres publicaciones antes de describir plazos, reposición y ámbito de bolsas.

## 1. Plazo de respuesta a los llamamientos

**Contraste:** el llamamiento publicado y el directo tienen reglas distintas. Conviene configurar el hecho de inicio y el cómputo por vía, conservando la publicación o contacto que los acredita.

- [Reglamento Diputación, art. 8.1.b, p. 8](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/2026_Reglamento_seleccion_de_personal_y_bolsas_bop_16_01_2026.pdf): máximo de dos días desde publicación.
- [Ley 39/2015, art. 30](https://www.boe.es/eli/es/l/2015/10/01/39#art30): reglas de cómputo administrativo por horas y días.
- [SAP, Accept Online Offer](https://help.sap.com/docs/successfactors-recruiting/recruiting-with-integration-to-position-management-and-employee-central-test-script/accept-online-offer?locale=en-US&state=PRODUCTION&version=2405): respuesta desde el portal de ofertas.

**Propuesta para VEC:** mantener las 48 horas actuales únicamente como ejemplo sintético y preparar una regla de publicación de dos días con evento de publicación acreditado; RRHH debe confirmar si el cómputo es hábil antes de activarla. UTC para conservar el instante y Europe/Madrid para mostrarlo; duración, unidad, calendario y vía en versión explícita.

**Por confirmar:** confirmar el calendario, el cómputo y las reglas del contacto directo. «Dos días desde publicación» no equivale necesariamente a «48 horas desde apertura por RRHH».

## 2. Falta de respuesta y renuncia

**Contraste:** falta de contacto, falta de interés en una publicación y renuncia son hechos distintos. Los productos separan el rechazo del candidato del rechazo decidido por quien selecciona.

- [Reglamento Diputación, arts. 8, 10 y 11](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/2026_Reglamento_seleccion_de_personal_y_bolsas_bop_16_01_2026.pdf): intentos telefónicos, renuncias justificadas y baja definitiva.
- [SAP, Accept Online Offer](https://help.sap.com/docs/successfactors-recruiting/recruiting-with-integration-to-position-management-and-employee-central-test-script/accept-online-offer?locale=en-US&state=PRODUCTION&version=2405): declinar produce un estado propio del candidato.
- [Odoo, Refuse applicants](https://www.odoo.com/documentation/19.0/applications/hr/recruitment/refuse_applicant.html): rechazo con motivo y comunicación configurada.

**Propuesta para VEC:** al vencer, pasar a revisión RRHH; continuar al siguiente elegible tras confirmación y revalidación del orden, con penalización automática desactivada. Registrar cada intento y causa por separado, con revisión de la justificación y acto competente antes de cualquier baja.

**Por confirmar:** documentar la aplicación de los arts. 10 y 11, quién valida justificantes y resuelve exclusiones. La ausencia de respuesta a una publicación no debe equipararse al supuesto de dos procesos telefónicos fallidos.

## 3. Cómo responde el candidato

**Contraste:** un portal puede recoger aceptación o renuncia y admitir una gestión previa fuera del sistema. Cada canal necesita evidencia y un tratamiento explícito de la respuesta tardía.

- [SAP, Accept Online Offer](https://help.sap.com/docs/successfactors-recruiting/recruiting-with-integration-to-position-management-and-employee-central-test-script/accept-online-offer?locale=en-US&state=PRODUCTION&version=2405): aceptar o declinar en el portal.
- [SAP, test público Recruiting, apartados 4.6.6.4 y 4.6.6.7](https://help.sap.com/doc/51167a37852a42cbad9aafe7efbea0d6/2311/en-US/Recruiting_with_Integration_to_Position_Management_and_Employee_Central_Test_Script_3EG.pdf): oferta verbal fuera del software y posterior respuesta en línea.
- [Ley 39/2015, arts. 9 a 11](https://www.boe.es/eli/es/l/2015/10/01/39#art9): identificación y firma tienen requisitos diferenciados.

**Propuesta para VEC:** «Mi Bolsa» con certificado como canal inicial; RRHH podrá registrar una respuesta externa con canal, fecha real, actor y referencia de evidencia conservada por su autoridad. La validación del registro externo dependerá de una política aprobada; la respuesta tardía quedará pendiente de decisión y conservará su instante.

**Por confirmar:** canales admitidos, custodia del original y efecto de las respuestas tardías; determinar cuándo la actuación exige firma documental además de identificación.

## 4. Circuito de firma

**Contraste:** los portafirmas permiten firmas sucesivas o simultáneas y seguimiento por firmante. La integración debe recuperar el estado y documento firmados y tramitar el motivo del rechazo.

- [Port@firmas Junta, condiciones de integración](https://desarrollo.juntadeandalucia.es/recursos/activo/portafirmas): documentos por referencia, PDF/A y gestión de rechazos.
- [SAS, Portafirmas](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/ayudadigital/aplicaciones/otros-servicios/portafirmas): multifirma jerarquizada y seguimiento de firmantes.
- [Manual Port@firmas 3.7, §12](https://desarrollo.juntadeandalucia.es/sites/default/files/2025-12/manualUsuarioPortafirmasv3.7.pdf): rechazo con observaciones obligatorias.
- [Workday, Approval Chain Step](https://doc.workday.com/admin-guide/en-us/manage-workday/business-processes/business-process-step-types/klq1658852814436.html): aprobaciones secuenciales y devolución a pasos anteriores.

**Propuesta para VEC:** por defecto ningún circuito no publicado permite avanzar; catálogo por documento, modalidad y fase con orden, competencia, suplencia y evidencia exigida. Rechazo o devolución crean incidencia con motivo; un documento corregido genera nueva versión y nueva solicitud, conservando la anterior.

**Por confirmar:** lista de documentos, cargos competentes, orden, aprobaciones que permiten fiscalizar y firmas exigidas antes de formalizar. Informática debe dar el contrato del portafirmas de Diputación; la documentación de Junta acredita un patrón, no compatibilidad con ese sistema.

## 5. Reparos y subsanaciones

**Contraste:** conformidad, observación, reparo suspensivo y conformidad condicionada requieren consecuencias distintas. La devolución por un defecto debe indicar quién lo subsana y qué verificaciones quedan afectadas.

- [RD 424/2017, arts. 11, 12, 14 y 15](https://www.boe.es/buscar/act.php?id=BOE-A-2017-5192#a1-4): conformidad, subsanación, observaciones y discrepancia.
- [Haciendas Locales, arts. 216 y 217](https://www.boe.es/buscar/act.php?id=BOE-A-2004-4214#a216): efectos suspensivos y órgano que resuelve discrepancias.
- [Workday, Approval Chain Step](https://doc.workday.com/admin-guide/en-us/manage-workday/business-processes/business-process-step-types/klq1658852814436.html): devolución a una persona o paso anterior.

**Propuesta para VEC:** clasificación elegida por Intervención; un reparo suspensivo bloquea avance y devuelve a la unidad gestora competente, conservando versión y motivo. Si cambia el contenido informado o firmado, generar nueva versión y repetir los controles afectados antes de remitir de nuevo; el tratamiento de observaciones será configurable.

**Por confirmar:** responsables de subsanación, cuándo repetir informe/firma, fiscalización aplicable y circuito de discrepancias. RRHH no puede tratar cualquier reparo como una mera advertencia ni resolver una discrepancia por permiso funcional genérico.

## 6. Roles y responsabilidades

**Contraste:** las responsabilidades se asignan por acción y ámbito; aprobación, devolución, reasignación y cancelación son capacidades distintas. Las suplencias administrativas deben identificar al titular y a quien actúa.

- [Workday, Business Processes, sección Business Processes and Security](https://doc.workday.com/admin-guide/en-us/manage-workday/business-processes/business-process-framework-concepts/dan1370797198449.html): políticas distintas para iniciar, aprobar, corregir, cancelar y reasignar.
- [Ley 40/2015, art. 13](https://www.boe.es/buscar/act.php?id=BOE-A-2015-10566#a13): constancia de suplencia y persona que la ejerce.
- [SAP, test público Recruiting, §4.6.6.1 a §4.6.6.3](https://help.sap.com/doc/51167a37852a42cbad9aafe7efbea0d6/2311/en-US/Recruiting_with_Integration_to_Position_Management_and_Employee_Central_Test_Script_3EG.pdf): preparación por reclutador y aprobación por dos responsables.

**Propuesta para VEC:** matriz inicial configurable con centro solicitante, RRHH analista, unidad gestora, firmantes, Intervención y Personal separados; cada acción requiere asignación nominal, ámbito y vigencia. Separar preparación y publicación de reglas y plantillas; la adjudicación quedará en revisión humana hasta aprobar el control y las incompatibilidades aplicables.

**Por confirmar:** personas y unidades competentes, actuaciones que requieren dos personas, suplencias y quién puede reasignar o cancelar. El cargo escrito en un catálogo no concede acceso por sí solo.

## 7. Reglas y catálogos del análisis

**Contraste:** modalidades, causas y cobertura tienen fuentes propias; una necesidad declarada por el centro no decide automáticamente el nombramiento. Las comprobaciones de disponibilidad y el acto de agotamiento deben quedar diferenciados.

- [Circular Diputación 08/05/2026, p. 1 y formulario NIS](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/CIRCULAR-PETICIONES-DE-PERSONAL_2026.report.pdf): plantilla obligatoria y decisión del Servicio de Selección Temporal.
- [Reglamento Diputación, arts. 3 y 7](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/2026_Reglamento_seleccion_de_personal_y_bolsas_bop_16_01_2026.pdf): SAE, nueva bolsa y agotamiento por resolución expresa.
- [Workday, Business Processes](https://doc.workday.com/admin-guide/en-us/manage-workday/business-processes/business-process-framework-concepts/dan1370797198449.html): definiciones de proceso configurables por organización.

**Propuesta para VEC:** catálogos versionados por autoridad y vigencia; modalidad desconocida queda pendiente de clasificación y disponibilidad vacía genera alerta de comprobación. El agotamiento administrativo y la elección SAE/nueva convocatoria exigen decisión acreditada, sin convertir el resultado técnico en resolución.

**Por confirmar:** catálogos oficiales, mantenimiento y criterios negociados de selección SAE. La circular manda usar NIS y la bandeja MOADH «NECESIDADES DE SELECCIÓN TEMPORAL»: RRHH debe confirmar cómo VEC sustituye o coordina ese circuito.

## 8. RC y coste de personal

**Contraste:** coste previsto y certificación de crédito son controles distintos. Las aplicaciones de nómina calculan según estructuras salariales, reglas y contrato, con datos mantenidos por sus responsables.

- [RD 500/1990, arts. 31 y 32](https://www.boe.es/buscar/act.php?id=BOE-A-1990-9664#art31): reserva por cuantía y certificación expedida por Intervención.
- [RD 424/2017, art. 13.2](https://www.boe.es/buscar/act.php?id=BOE-A-2017-5192#a1-5): existencia y adecuación de crédito en fiscalización limitada.
- [Odoo, Salaries](https://www.odoo.com/documentation/19.0/fr/applications/hr/payroll/salaries.html): estructuras, reglas y parámetros de cálculo.
- [Circular Diputación, formulario NIS](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/CIRCULAR-PETICIONES-DE-PERSONAL_2026.report.pdf): financiación y RC o documento autorizado para programas.

**Propuesta para VEC:** importe estimado con desglose, jornada, periodo, tabla y versión; RC por referencia, importe, ejercicio, fuente y validación acreditada. Crédito insuficiente, modificado o no verificable deja pendiente el control económico y exige recalcular/revalidar cuando cambien fechas, jornada o conceptos.

**Por confirmar:** sistema económico, responsables, conceptos incluidos y tablas vigentes; confirmar el hito obligatorio y las alternativas autorizadas a RC. No aplicar un porcentaje inventado de Seguridad Social.

## 9. Documentos

**Contraste:** una plantilla debe definir datos, destinatarios y firmas; el documento generado conserva su propia versión. Los modelos oficiales de Diputación y las normas de documento electrónico dan referencias comprobables.

- [Circular Diputación, NIS normalizada](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/CIRCULAR-PETICIONES-DE-PERSONAL_2026.report.pdf): campos y formato de la solicitud vigente.
- [NTI Documento Electrónico, §§III a V](https://www.boe.es/buscar/act.php?id=BOE-A-2011-13169): contenido, firma y metadatos.
- [Odoo, Request a signature, Templates](https://www.odoo.com/documentation/19.0/applications/productivity/sign/request_signatures.html): campos, firmantes, acceso y copia nueva por solicitud.
- [Port@firmas Junta](https://desarrollo.juntadeandalucia.es/recursos/activo/portafirmas): documentos estables y PDF/A para su circuito.

**Propuesta para VEC:** catálogo versionado para los seis tipos existentes, por modalidad y fase, con campos permitidos y condiciones; propuesta por configurador y publicación por otra persona autorizada. Cada generación conserva modelo, datos, huella y versión; permanece borrador hasta completar el circuito oficial.

**Por confirmar:** plantillas de los seis documentos, anexos, contenido y firmas. La NIS pública no sustituye esas seis plantillas ni confirma su contenido.

## 10. Comunicaciones y contacto

**Contraste:** el contacto se rectifica con identidad y trazabilidad; envío, fallo de entrega y notificación administrativa son resultados diferentes. Una aceptación SMTP no acredita acceso de la persona.

- [AEPD, derecho de rectificación](https://www.aepd.es/preguntas-frecuentes/1-tus-derechos/2-tus-derechos-de-proteccion-de-datos/FAQ-0110-que-el-derecho-de-rectificacion): corrección de datos personales inexactos.
- [AEPD, formulario e instrucciones](https://www.aepd.es/documento/formulario-derecho-de-rectificacion.pdf): identificación proporcional y canales para rectificar.
- [Ley 39/2015, arts. 41 y 43](https://www.boe.es/eli/es/l/2015/10/01/39#art41): aviso por correo y acreditación de notificación separados.
- [Odoo, Common emailing issues and solutions](https://www.odoo.com/documentation/19.0/applications/general/email_communication/faq.html): cola, error y reintento de entrega.

**Propuesta para VEC:** titular autenticado propone/cambia el contacto con comprobación del nuevo correo; RRHH tramita discrepancias con permiso específico, motivo e historia. SMTP corporativo con estados pendiente, aceptado por servidor y fallo; fallos dejan incidencia recuperable y los mensajes se clasifican como aviso salvo circuito de notificación acreditado.

**Por confirmar:** quién aprueba correcciones, evidencia del correo y servicio administrativo de notificaciones; definir cuándo reintentar o usar otro canal. El correo sigue siendo el del alta VEC ya acordado.

## 11. Incorporación, GINPIX y cierre

**Contraste:** aceptación, formalización, incorporación y alta en el sistema de personal son hitos diferentes. SAP separa Recruiting, Onboarding y Employee Central y contempla expresamente la no incorporación.

- [SAP, Recruit-to-Hire Business Process](https://help.sap.com/docs/successfactors-employee-central/implementing-employee-central-core/recruit-to-hire-business-process): validación de datos, fecha de alta y no-show.
- [SAP, Features Of Manage Pending Hires](https://help.sap.com/docs/successfactors-onboarding/administering-onboarding-1-0/features-of-manage-pending-hires): borrador, aprobación pendiente y devolución.
- [Odoo, Contracts](https://www.odoo.com/documentation/19.0/applications/hr/payroll/contracts.html): contrato, jornada y remuneración gestionados en Personal/Nómina.
- [Workday, Business Process Framework, p. 2](https://www.workday.com/content/dam/web/en-us/documents/datasheets/workday-business-process-framework.pdf): pasos de integración externos dentro del proceso.

**Propuesta para VEC:** Personal confirma incorporación mediante acto y fecha efectiva; GINPIX tiene estados preparado, enviado, confirmado y rechazado, con referencia de envío, reintento sin duplicación y acuse verificado. Cierre administrativo por RRHH solo tras los controles configurados y las confirmaciones requeridas, manteniendo seguimiento y cese como actuaciones propias.

**Por confirmar:** documento de incorporación, campos e interfaz de GINPIX, acuse que acredita su aceptación y ubicación de esos hitos en las ocho fases. Ninguna fuente consultada proporciona el contrato de integración de la instalación GINPIX de Diputación.

## 12. Cambios e incidencias posteriores

**Contraste:** corregir, cancelar y deshacer una decisión requieren competencias y efectos distintos. La no incorporación y el cese deben conservar la causa y la fecha real y llegar al sistema que posee la relación de personal.

- [Workday, Business Processes, sección Security](https://doc.workday.com/admin-guide/en-us/manage-workday/business-processes/business-process-framework-concepts/dan1370797198449.html): acciones de corrección, cancelación y rescisión con permisos propios.
- [SAP, Recruit-to-Hire Business Process](https://help.sap.com/docs/successfactors-employee-central/implementing-employee-central-core/recruit-to-hire-business-process): proceso No-Show para quien no acude el primer día.
- [TREBEP, art. 10.3](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719#a10): finalización de interinidad por causa acreditada.
- [Odoo, Contracts](https://www.odoo.com/documentation/19.0/applications/hr/payroll/contracts.html): datos de jornada y retribución asociados al contrato.

**Propuesta para VEC:** incidencia o solicitud de cambio con motivo y versión nueva; cambios de fechas, jornada, modalidad o coste reabren por defecto los controles económicos, informes y firmas que dependían de esos datos. Cancelar antes del acto eficaz y revisar actos posteriores serán operaciones separadas; cese e incorporación corresponden a Personal y sus efectos en Bolsa se transmiten con recibo.

**Por confirmar:** matriz cambio/control, autoridad para cancelar o rectificar y procedimiento de no incorporación y cese por modalidad. No extrapolar las causas de cese del interino a todo contrato laboral.

## 13. Reglamento de bolsas y reglas de reposición

**Contraste:** las reglas de reposición cambian entre administraciones y según el régimen jurídico. Granada fija una carencia transversal tras el cese y contempla excepciones para ofrecer vacantes a ciertas personas que ya trabajan; Mérida distingue listas de larga y corta duración.

- [Reglamento de Granada, BOP 16/01/2026, artículos 8 y 9](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1768521622300-final-680c0e98.pdf): un mínimo de cinco meses desde la finalización, nueve para nombramientos por acumulación de tareas; excepción para vacantes y retorno por prelación.
- [Corrección del BOP de Granada, 27/01/2026](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1769472066115-final-42c9b1bc.pdf): corrige únicamente la fecha del pleno de aprobación, de 18 a 23/12/2025; no cambia esas reglas.
- [Manual oficial de bolsas de Mérida, apartados 5.1 y 5.2, páginas 8 y 9](https://descargas.merida.es/empleo-publico/bolsas-empleo-manual.pdf): misma posición para larga duración; corta duración puede mantener posición o pasar al final según duración y categoría. Sus seis meses son una regla local.
- [Estatuto de los Trabajadores, artículo 15.5](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11430#a15): controla más de dieciocho meses en veinticuatro mediante contratos por circunstancias de la producción, con reglas referidas tanto a la persona como al puesto.
- [TREBEP, artículo 10 y disposición adicional 17](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719#a10): límites diferentes para vacantes, programas, sustituciones y acumulación; la vacante tiene reglas y excepciones propias sobre los tres años.

**Propuesta para VEC:** conservar como valores provisionales el mínimo de cinco meses y los nueve meses por acumulación de tareas, con cese acreditado, cómputo configurable y modalidad desconocida pendiente. Incorporar una excepción expresa y revisada para vacantes; un contrato activo no puede ser un bloqueo absoluto. Separar la carencia de Bolsa del control de temporalidad de Personal, con avisos preventivos configurables por modalidad y puesto, sin declarar automática adquisición de condición fija ni cese.

**Por confirmar:** RRHH debe confirmar tipos de lista realmente usados, criterio de reposición, cómputo civil y excepciones, fuente de servicios/ceses, alcance a organismos autónomos y responsables del examen jurídico; el aviso genérico a tres años no cubre todos los límites.

## 14. Orden, adjudicación y llamamiento

**Contraste:** el orden de la bolsa y la disponibilidad gobiernan la propuesta; los canales e intentos dependen de la norma local. La ordenación de Granada no se reduce a ordenar por puntos.

- [Reglamento de Granada, artículos 6 y 8](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1768521622300-final-680c0e98.pdf): prioridad por ejercicios aprobados; publicación selecciona mejor posición disponible; directo exige dos llamadas separadas al menos dos horas y dos procesos para el mismo ofrecimiento.
- [Reglamento de la Bolsa Única de la Junta de Andalucía, BOJA 85/2026, apartado 11](https://www.juntadeandalucia.es/boja/2026/85/49): asignación por órgano competente según orden y preferencias; aceptación escrita y comunicaciones electrónicas. Su protocolo telefónico excepcional tiene reglas diferentes.
- [Manual oficial de Mérida, apartado 4.2, página 7](https://descargas.merida.es/empleo-publico/bolsas-empleo-manual.pdf): lista rotatoria, hasta tres llamadas entre 7:30 y 14:30. Demuestra que horario y número de intentos no se pueden trasladar a Granada.

**Propuesta para VEC:** proponer al primer aceptante elegible según una versión del orden y exigir confirmación de RRHH, conservando la revisión por una persona competente; registrar cada intento real y mandar a revisión los supuestos de exclusión. Los valores 09:00–14:00 y segundo ciclo el siguiente laborable quedan configurables y sin atribución normativa; la publicación y su plazo deben identificar el hecho inicial real, no una apertura interna sin publicación.

**Por confirmar:** RRHH debe fijar desempates residuales, autoridad de adjudicación y exclusión, doble validación, horario y tratamiento del segundo ciclo, y confirmar cuándo pasa de llamamiento directo al sistema de publicación; el reglamento prevé una transición condicionada a disponer del programa.

## 15. Numeración de expedientes

**Contraste:** el identificador de expediente, su número visible y el asiento registral cumplen funciones diferentes. Los anuncios de Diputación muestran un formato de procedimiento que no prueba cuál usa Contratación temporal.

- [NTI Expediente Electrónico, anexo I](https://www.boe.es/buscar/act.php?id=BOE-A-2011-13170#ai): identificador y metadatos para intercambio.
- [Ley 39/2015, arts. 16 y 70](https://www.boe.es/eli/es/l/2015/10/01/39#art16): registro de documentos y expediente administrativo.
- [Anuncio Diputación 14/05/2026, p. 1](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/2026_Mod_Reglamento_sel_per_temporal_Bolsa_Empleo-1778713220300-final-241a2c51.pdf): ejemplo público `2026/PES_01/001735`.

**Propuesta para VEC:** referencia opaca estable y número visible por autoridad externa cuando exista; el formato y el momento de asignación se configuran desde la autoridad acreditada. Un número emitido no se reutiliza; el asiento de registro y el identificador interoperable se conservan en campos separados.

**Por confirmar:** formato real de Contratación, sistema que lo asigna, momento, reglas anuales y tratamiento de peticiones anuladas. No asumir que `PES_01` corresponde a este procedimiento.

## 16. Exportación de CONVOCA para la carga inicial

**Contraste:** una migración necesita distinguir bolsa, persona y participación, conservar referencias de origen y conciliar el resultado antes de activar los datos. El catálogo público permite inventariar bolsas, pero no acredita el formato de la exportación privada de Granada.

- [Bolsas constituidas del Ayuntamiento de Sevilla](https://www.sevilla.org/servicios/empleo/servicio-de-recursos-humanos/bolsas-de-trabajo/ayuntamiento-de-sevilla/constituidas): publica categoría, fecha de vigencia y último empleado; advierte que personas anteriores pueden volver a estar disponibles.
- [Modificación del reglamento de Granada, BOP 14/05/2026](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1778713220300-final-241a2c51.pdf): añade adhesión expresa y publicada de organismos autónomos para categorías existentes; ese ámbito debe figurar en el inventario.
- [Odoo 17, exportación e importación, apartado «Import from another application»](https://www.odoo.com/documentation/17.0/applications/essentials/export_import_data.html#import-from-another-application): identifica registros y relaciones por External ID estable y evita duplicados al repetir cargas. Es un patrón de migración, no prueba del formato CONVOCA; contenido comprobado en el resultado indexado oficial, apertura directa con timeout.

**Propuesta para VEC:** preparar una carga en zona de revisión con fecha de corte, huella del archivo, identificadores de origen por entidad y tabla versionada de correspondencia de estados; bloquear filas ambiguas y activar solo tras conciliación de totales y posiciones aprobada por RRHH. Definir adaptadores por formato confirmado, usando muestras sintéticas y sin deducir el XLS de Bolsa de otras exportaciones.

**Por confirmar:** RRHH/Sistemas deben entregar inventario, exportación y diccionario reales, responsable generador, fecha de corte, criterio de actualización final, histórico migrable y autoridad de cada dato de contacto; la investigación pública no resuelve esos datos internos.

## 17. Consulta pública de las bolsas

**Contraste:** hay administraciones con listas consultables sin identificación y sistemas empresariales con estados propios del portal del candidato. Ocultar parte del DNI no determina por sí solo qué historia personal puede publicarse.

- [Manual oficial de Mérida, apartados 8 y 10](https://descargas.merida.es/empleo-publico/bolsas-empleo-manual.pdf): consulta sin certificado de orden y situación, con DNI enmascarado; referencia comparativa del mismo producto CONVOCA.
- [AEPD, orientación para la disposición adicional 7](https://www.aepd.es/documento/orientaciones-da7.pdf): criterio de cuatro cifras y enmascaramiento según tipo de documento; evita publicar fragmentos diferentes que permitan recomponer la identificación.
- [AEPD, consultas de Administraciones Públicas](https://www.aepd.es/areas-de-actuacion/administraciones-publicas/consultas-mas-relevantes-atendidas-a-traves-del-canal-del-dpd): exige valorar finalidad y minimización en procesos selectivos; distingue publicidad de notificación y recomienda retirada y límites de acceso según el caso.
- [SAP SuccessFactors, KBA pública 2080952](https://userapps.support.sap.com/sap/support/knowledge/en/2080952): diferencia etiqueta interna, etiqueta visible al candidato y texto del siguiente paso, con permisos de visibilidad configurables.

**Propuesta para VEC:** mantener una proyección pública independiente, con catálogo de campos permitidos y publicación desactivada hasta su aprobación; inicialmente bolsa, versión y documentación pública, y datos personales solo en el alcance aprobado. «Mi Bolsa» conserva historia propia autorizada y paginada, distinguiendo ausencia de datos de ausencia de contratos; nunca expone causas médicas o familiares al público.

**Por confirmar:** RRHH, con el DPD, debe aprobar finalidad, campos, identificador y estado publicables, actualización y retirada de las listas; también qué acciones y antecedentes puede consultar el titular y de qué fuente se obtiene la historia anterior a VEC.

## 18. Pausas, reactivaciones y bajas

**Contraste:** la pausa voluntaria, la revisión de una solicitud y la exclusión definitiva tienen consecuencias distintas. Mérida mantiene posición durante la pausa; la Junta exige solicitud telemática y reglas temporales propias.

- [Reglamento de Granada, artículos 10 y 11](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1768521622300-final-680c0e98.pdf): causas justificadas documentadas sin penalización; bajas definitivas por renuncia injustificada, incumplimiento de incorporación/documentación y ciertos intentos fallidos.
- [Manual oficial de Mérida, apartados 6, 7 y 9](https://descargas.merida.es/empleo-publico/bolsas-empleo-manual.pdf): RRHH registra suspensión/reactivación solicitada, conserva posición y establece documentación y plazos locales de formalización.
- [Bolsa Única de la Junta de Andalucía, apartados 10.4, 10.7 y 12](https://www.juntadeandalucia.es/boja/2026/85/49): suspensión mínima de cuatro meses, reactivación solicitada y exclusión por falsedad previa audiencia; causas justificadas específicas.
- [SAP SuccessFactors, KBA pública 2080952](https://userapps.support.sap.com/sap/support/knowledge/en/2080952): estados configurables, pasos requeridos y responsables habilitados; aporta un patrón de gobierno de transiciones, sin regular bolsas españolas.

**Propuesta para VEC:** tratar «en revisión» por defecto como estado de la solicitud/incidencia, con responsable, motivo y recibo, conservando el estado de disponibilidad anterior hasta una decisión salvo medida provisional expresamente aprobada. Pausar/reactivar será solicitud del titular con validación de RRHH, misma posición como propuesta configurable, evidencia mínima y transiciones versionadas; baja definitiva requiere decisión motivada y conserva historia.

**Por confirmar:** RRHH debe confirmar si «en revisión» afecta realmente a la elegibilidad, cuándo y cómo surte efectos la pausa, ámbito de bolsas, justificantes, autoridad y audiencia, catálogo de exclusiones y documentación/plazo tras aceptar; no trasladar las 24 horas de Mérida ni cuatro meses de la Junta a Granada.

## 19. Fuente de requisitos para reescribir Cronos

**Contraste:** el inventario funcional debe reunir normas y modificaciones, configuración, permisos y tareas de cada usuario. El texto provincial de 2010 tiene modificaciones posteriores: el acuerdo de 02/09/2026 declara expresamente la nulidad del artículo 17.4; ese precepto no puede copiarse sin contraste.

- [Diputación de Granada: Reglamento regulador del tiempo de trabajo, BOP 20/01/2010](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/REGLAMENTO-TIEMPO-DE-TRABAJO.pdf). Fuente primaria de jornada, calendario y permisos; debe leerse junto con sus modificaciones.
- [Diputación: acuerdo de 02/09/2026 sobre el artículo 17.4](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/Certificado_pleno_2_09_2026_punto_nA_12_Rev_art_17.4.pdf). Apartado tercero, página 3: declara la nulidad de pleno derecho del precepto por infracción respecto del personal funcionario del artículo 50.1 del TREBEP.
- [SAP SuccessFactors: implementación de Clock In Clock Out](https://help.sap.com/docs/successfactors-time-tracking/sap-best-practices-for-sap-successfactors-time-tracking/setup-clock-in-clock-out-in-time-tracking-5wq?locale=en-US&state=PRODUCTION&version=2211). Enumera tipos de evento, grupos, configuración, permisos y circuitos de entradas manuales; sirve de lista funcional de contraste.

**Propuesta para VEC:** preparar una ficha por operación, con norma y versión, pantallas, entradas, salida, permiso y casos sintéticos de aceptación. Por defecto, el material del legado se recibe sin credenciales ni datos personales; RRHH valida la ficha antes de activar reglas.

**Por confirmar:** acceso al PHP, pantallas y esquema saneados; persona conocedora del legado; aprobador de requisitos; relación consolidada de normas y modificaciones aplicables a cada colectivo.

## 20. Fuentes y canales de fichaje

**Contraste:** SAP distingue eventos de terminal, web y móvil y permite registrar manualmente un fichaje omitido con permisos propios; Odoo conserva el método de registro. Ninguna de estas fuentes acredita cuáles son los terminales ni la interfaz actual de Diputación.

- [SAP SuccessFactors: Clock In Clock Out](https://help.sap.com/docs/successfactors-time-tracking/sap-best-practices-for-sap-successfactors-time-tracking/setup-clock-in-clock-out-in-time-tracking-5wq?locale=en-US&state=PRODUCTION&version=2211). Documenta terminal/web/móvil y tipos de evento configurables.
- [SAP SuccessFactors: creación de eventos manuales](https://help.sap.com/docs/successfactors-employee-central/operating-time-management-in-sap-successfactors/creating-manual-time-events?locale=en-US). Requiere permisos de lectura/creación y población de empleados definida; contempla fallo del terminal y olvido.
- [Odoo 19: detalle de registros de asistencia](https://www.odoo.com/documentation/19.0/applications/hr/attendances/attendance_logs.html). Registra entrada/salida, tiempo trabajado y método —interfaz, quiosco o manual— y distingue incidencias que requieren corrección.
- [Diputación: instrucciones de teletrabajo de 26/04/2024](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/Resolucion_instrucciones_teletrabajo_web.pdf). Exige autorización expresa y mantiene jornada y horario; no especifica un protocolo técnico de fichaje remoto.

**Propuesta para VEC:** catálogo configurable de orígenes y movimientos; conservar evento original, identificador del origen, instante UTC y zona de presentación. Por defecto, duplicados recuperan el evento existente y discrepancias u omisiones abren una incidencia; la rectificación enlaza el original con motivo y autorización, sin inventar una salida.

**Por confirmar:** inventario de terminales e interfaces, fuente prevalente, precisión y zona de cada origen, reglas de desconexión y deduplicación, autoridad de correcciones y pruebas sintéticas.

## 21. Reglas y responsabilidades de Cronos

**Contraste:** las reglas provinciales diferencian colectivos, turnos y calendarios; Workday permite configurar validaciones que bloquean o solo avisan. Los movimientos de tiempo destinados a nómina necesitan un contrato propio.

- [Diputación: Reglamento del tiempo de trabajo](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/REGLAMENTO-TIEMPO-DE-TRABAJO.pdf). Distingue centros sociales, turnos, calendarios y conformidad de jefatura/resolución para determinadas solicitudes; texto histórico sujeto a modificaciones.
- [Diputación: nulidad del artículo 17.4, acuerdo 02/09/2026](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/Certificado_pleno_2_09_2026_punto_nA_12_Rev_art_17.4.pdf). Cambio expreso que obliga a comprobar la vigencia del catálogo de permisos y vacaciones.
- [Workday: creación de validaciones de entrada de tiempo](https://doc.workday.com/admin-guide/en-us/human-capital-management/time-tracking/time-entry-validations/dan1370796789009.html). Diferencia errores críticos y advertencias y permite condiciones configurables con mensajes comprensibles.
- [Odoo 19: entradas de trabajo para nómina](https://www.odoo.com/documentation/19.0/applications/hr/payroll/work_entries.html). Describe movimientos generados desde planificación, asistencia y ausencias y su consumo para determinados cálculos de nómina.

**Propuesta para VEC:** catálogo versionado por colectivo y centro, con unidad, vigencia, calendario común y autoridad de cada decisión; el saldo se explica mediante movimientos. Mantener por defecto la orden del operador de jefatura y después RRHH, sin autoaprobación ni doble intervención por una misma persona, y transmitir a nómina únicamente incidencias aprobadas cuando exista contrato.

**Por confirmar:** reglas vigentes y texto consolidado, competencias y suplencias nominales, validación de justificantes, cómputos y campos que debe consumir nómina. La anulación comprobada no decide estas configuraciones pendientes.

## 22. Fuente de requisitos para Dietas

**Contraste:** los procedimientos públicos distinguen autorización del desplazamiento, aportación de justificantes y tramitación económica. La documentación de Workday pide fijar configuración, seguridad e interacción con otros productos antes de implantar gastos.

- [UGR: ECO-02, liquidación de comisión de servicio después del viaje](https://filosofiayletras.ugr.es/facultad/documentos/gestion-departamental/procedimientos/comisiones-servicio). Identifica autorización previa, datos mínimos, documentos, subsanación y unidad de gasto.
- [UGR: Instrucción 1/2025 sobre el módulo de comisiones de servicio](https://gerencia.ugr.es/sites/webugr/gerencia/public/2025-01/Instruccion%201_2025%20M%C3%B3dulo%20de%20Comisiones%20de%20Servicio%20en%20RCF.pdf). Explica la relación entre Registro Contable de Facturas, UXXI, PERLICO y sede, y qué expedientes entran en el módulo.
- [Workday: decisiones de configuración de gestión de gastos](https://doc.workday.com/admin-guide/en-us/financial-management/expenses/expense-management/setup-considerations--expense-management.html). Trata procesos, requisitos de seguridad, limitaciones e integraciones antes de implementar.

**Propuesta para VEC:** inventario por perspectiva —solicitante, jefatura y RRHH— con formulario, norma, autoridad, justificante, transición y resultado esperado. Por defecto, permitir preparar borradores con reglas rotuladas como provisionales; la conexión de efectos económicos exige requisitos aprobados y contrato del destino.

**Por confirmar:** aplicación actual, impresos e instrucciones provinciales, responsables de autorización/tramitación económica y quién aprueba la ficha completa con RRHH e Intervención.

## 23. Tarifas y reglas de cálculo de Dietas

**Contraste:** las cuantías dependen de concepto, grupo y vigencia; la indemnización por kilómetro y su tratamiento fiscal son reglas distintas. La Orden HFP/793/2023 entró en vigor el 18/07/2023 y la HFP/792/2023 el 17/07/2023; no debe inferirse de ellas una tabla provincial única.

- [BOE: Real Decreto 462/2002, artículos 2, 10–12 y 18 y anexos](https://www.boe.es/buscar/act.php?id=BOE-A-2002-10337). Regula ámbito, grupos, cuantías, devengo y excepciones; remite para personal laboral al convenio o normativa específica.
- [BOE: Orden HFP/793/2023](https://www.boe.es/eli/es/o/2023/07/12/hfp793). Actualiza la indemnización prevista en el RD 462/2002: automóvil 0,26 €/km y motocicleta 0,106 €/km, con entrada en vigor al día siguiente de publicarse.
- [BOE: Orden HFP/792/2023](https://www.boe.es/buscar/doc.php?id=BOE-A-2023-16461). Fija la exclusión de gravamen para determinados gastos de locomoción; entrada en vigor el día de publicación.
- [Odoo 19: categorías de gastos](https://www.odoo.com/documentation/19.0/applications/finance/expenses/expense_categories.html). Separa reembolso de gasto real, precio por unidad y cantidad diaria; referencia de diseño, sin cuantías provinciales.

**Propuesta para VEC:** reglas de cálculo y fiscales separadas, con fuente, autoridad, versión, fechas, colectivo y moneda, usando valores exactos. Como propuesta provisional, aplicar la vigencia a cada línea/día de devengo y conservar su versión; una comisión que cruce tarifas debe señalar el cambio y quedar pendiente de revisión hasta que RRHH confirme ese criterio.

**Por confirmar:** fuente provincial vigente por colectivo, grupos y excepciones, autoridad de publicación y criterio temporal para cambios de tarifa. Las cifras estatales citadas no se dan por aprobadas para todos los empleados de Diputación.

## 24. Justificantes, rutas y excepciones de Dietas

**Contraste:** cada concepto conserva su justificante y las excepciones requieren evidencia y decisión competente. La UGR admite, en determinados casos, documentación alternativa que pruebe el viaje cuando se pierde la tarjeta de embarque; ese criterio pertenece a su procedimiento.

- [UGR: preguntas frecuentes de comisiones de servicio](https://ofcontrolinterno.ugr.es/informacion/documentos/preguntas-frecuentes/comisiones). Explica supuestos de billetes y tarjetas de embarque extraviadas con memoria y prueba alternativa.
- [UGR: Instrucción 1/2025, apartados 5–6](https://gerencia.ugr.es/sites/webugr/gerencia/public/2025-01/Instruccion%201_2025%20M%C3%B3dulo%20de%20Comisiones%20de%20Servicio%20en%20RCF.pdf). Establece copia auténtica, custodia del original y codificación documental en su circuito; su plazo no se propone como plazo VEC.
- [Odoo 19: registro de gastos](https://www.odoo.com/documentation/19.0/applications/finance/expenses/log_expenses.html). Vincula cada gasto con categoría, fecha, importe, quién lo pagó y recibo.

**Propuesta para VEC:** catálogo de justificantes por gasto, con referencia de custodia y huella, evidencia alternativa y aprobación expresa de la excepción. Mantener cartografía interna y distinguir kilómetros calculados, declarados y autorizados; por defecto, un desvío necesita motivo y evidencia, y una huella por sí sola no acredita custodia ni aceptación del documento.

**Por confirmar:** documentos y formatos admitidos, custodio y conservación, personas que aceptan sustituciones, regla provincial de ruta/distancia y excepciones autorizables.

## 25. Autorización, fiscalización y liquidación de Dietas

**Contraste:** aprobar un gasto, contabilizarlo y pagarlo son actos diferentes. El régimen local de control interno atribuye al órgano interventor control sobre gasto, obligaciones y pago; la secuencia concreta de Diputación requiere su procedimiento.

- [BOE: Real Decreto 424/2017, artículos 3 y 7](https://www.boe.es/buscar/act.php?id=BOE-A-2017-5192). Distingue las formas de control y las fases de intervención del gasto y pago.
- [UGR: ECO-02, comisión de servicio](https://filosofiayletras.ugr.es/facultad/documentos/gestion-departamental/procedimientos/comisiones-servicio). Documenta autorización previa, liquidación posterior, subsanación y cierre.
- [Odoo 19: procesamiento de gastos](https://www.odoo.com/documentation/19.0/applications/finance/expenses/approve_expenses.html). Define aprobación por usuario autorizado, rechazo con motivo y estados diferentes de contabilización y pago.

**Propuesta para VEC:** circuito configurable que parta del recorrido actual —administrativo, responsable, RRHH e Intervención— y registre cada transición con actor, ámbito, versión y recibo. Por defecto, impedir autoaprobación, devolver con motivo, corregir en nueva versión y reenviar al administrativo; una rectificación eficaz necesita otra actuación enlazada y autoridad expresa.

**Por confirmar:** competencias y suplencias, separación exigida en cada paso, alcance de fiscalización, documento de liquidación/cierre y quién puede anular, rectificar o decidir si la subsanación repite pasos.

## 26. Integración de Dietas con nómina y sistemas económicos

**Contraste:** un sistema puede reembolsar por nómina o por el circuito contable, y la aprobación no acredita pago. Las fuentes prueban esa separación y la necesidad de contratos; no describen la interfaz de GINPIX ni garantizan idempotencia de la integración provincial.

- [UGR: Instrucción 1/2025 del módulo de comisiones de servicio](https://gerencia.ugr.es/sites/webugr/gerencia/public/2025-01/Instruccion%201_2025%20M%C3%B3dulo%20de%20Comisiones%20de%20Servicio%20en%20RCF.pdf). Acredita integración entre aplicaciones de autorización, sede y registro contable, con alcance funcional explícito.
- [Odoo 19: reembolso a empleados](https://www.odoo.com/documentation/19.0/applications/finance/expenses/reimburse.html). Distingue incorporación a la siguiente nómina, contabilización y pago mediante otros medios; la inclusión en nómina mantiene su propio estado hasta procesarse.
- [Workday: configuración de gestión de gastos](https://doc.workday.com/admin-guide/en-us/financial-management/expenses/expense-management/setup-considerations--expense-management.html). Documenta seguridad de configuración, integraciones y prevención de pagos duplicados mediante conciliación de recibos/transacciones; no acredita una API idempotente de Diputación.

**Propuesta para VEC:** por defecto, desactivar envío hasta contratar el destino; separar liquidada, remitida, aceptada/rechazada y pago confirmado. Proponer clave estable por liquidación/versión, registro de envíos pendientes, consulta de estado antes de reintentar un resultado incierto y conciliación por referencia externa, incluidos anticipos y pagos parciales; las garantías requieren ensayo con el proveedor real.

**Por confirmar:** GINPIX u otro destino, responsable y momento de envío, esquema/interfaz, acuses, consulta y conciliación, cancelaciones/rectificaciones y juegos sintéticos para probar reintentos sin doble pago.

## 27. Registro de Personal y sistemas maestros

**Contraste:** el Registro Central de Personal de la AGE conserva actos de la vida administrativa y ofrece servicios distintos para anotación, consulta y generación documental. SAP Employee Central distingue identidad de persona, relaciones y fechas de efectos; estos ejemplos orientan el contrato, pero no identifican el maestro provincial.

- [MTDFP: Registro Central de Personal](https://digital.gob.es/funcion-publica/dgfp/registro-central-personal): inscripciones y actos; Anot@RCP, Consult@RCP y Edit@RCP.
- [TREBEP, artículo 71](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719#a71): registro de cada Administración e intercambio homogéneo con protección de datos.
- [SAP: Job Relationships](https://help.sap.com/docs/SAP_SUCCESSFACTORS_EMPLOYEE_CENTRAL/b14dd15ca58f43e0856184a740a4b212/d07e7c1b7ee14b71906a45bb56f20ffd.html): vínculos con jefaturas y gestores, con fechas de efecto desde el alta hasta la baja; no describe por sí sola la relación jurídica de empleo.
- [SAP: How Users are Uniquely Identified](https://help.sap.com/docs/successfactors-employee-central/implementing-employee-central-core/how-users-are-uniquely-identified-in-sap-successfactors): identificador de persona inmutable distinto de identificadores externos modificables.

**Propuesta para VEC:** configurar una fuente y un responsable por bloque, con referencias opacas de persona/relación/puesto, versión de contrato, fecha de efectos, actualización e historia; comenzar con lectura autorizada y adaptadores desactivados hasta su admisión. Personal consume el maestro confirmado y mantiene su autoridad funcional; una cuenta del directorio no acredita empleo.

**Por confirmar:** RRHH identifica Registro de Personal, RPT, ocupaciones, servicios y nómina; Sistemas aporta diccionario, identificadores, interfaces realmente disponibles, histórico migrable, reconciliación y muestras sintéticas. No se presupone que RCP estatal ni SAP sean sistemas de Diputación.

## 28. Datos y rectificaciones de Personal

**Contraste:** el autoservicio permite consultar información y presentar una corrección, mientras el responsable comprueba la evidencia y modifica el dato autorizado. Los certificados de la AGE se solicitan a órganos diferentes según materia y relación de servicio.

- [AEPD: derecho de rectificación](https://www.aepd.es/derechos-y-deberes/conoce-tus-derechos/derecho-de-rectificacion): identificación del dato inexacto, corrección solicitada y documentación justificativa cuando sea necesaria.
- [MTDFP: certificados en la AGE](https://digital.gob.es/funcion-publica/dgfp/regimen-juridico/servicio-informa/certificados-en-la-administracion-general-del-estado): solicitud por Funciona/SIGP y órgano competente según certificado y situación.
- [SAP: Employee Self-Service](https://help.sap.com/docs/successfactors-employee-central/implementing-employee-central-core/708a728e453c435581564ecb1819281a.html): las transacciones y campos editables dependen de permisos y requisitos de la organización.

**Propuesta para VEC:** mostrar cada bloque con fuente y vigencia; puesto, relación, situación, servicios reconocidos y antigüedad serán de consulta autorizada y solicitud de rectificación por defecto. La solicitud conservará dato discutido, evidencia, responsable, estado y resultado, y solo la confirmación del sistema de origen actualizará la proyección; los certificados y referencias de nómina se consultarán mediante su autoridad específica.

**Por confirmar:** RRHH aprueba campos consultables, circuito de revisión y órgano certificante por bloque; Sistemas define cómo enviar la solicitud y recibir su resolución. DPD y la unidad competente confirman el cauce del derecho de rectificación y su relación con la corrección de actos administrativos.

## 29. Roles y ámbitos de Personal

**Contraste:** SAP diferencia permisos de lectura y escritura por rol, entidad y campo; el ENS exige gestión expresa del acceso y segregación en tareas críticas. El alcance de jefatura necesita relación y competencia acreditadas, además del nombre del rol.

- [SAP: Administrator Permissions for Employee Central](https://help.sap.com/docs/successfactors-employee-central/implementing-employee-central-core/administrator-permissions-for-employee-central): lectura propia de información económica frente a mantenimiento autorizado de RRHH.
- [SAP: Using Role-Based Permissions, páginas 122 y siguientes](https://help.sap.com/doc/2e2b9bb483b4450e8fcbb70e984808e8/2411/en-US/SF_RBP_Adm.pdf): permisos por bloques/campos y distinción entre consulta actual, historia y edición.
- [ENS, anexo II, op.acc.2–4](https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191#aii): derechos exactos, gestión del acceso y concurrencia para tareas críticas según las medidas aplicables.

**Propuesta para VEC:** publicar una matriz por bloque y operación con perfil, unidad, relación, periodo, finalidad y campos exactos; partir de denegación y habilitar únicamente concesiones centrales aprobadas. Titular consulta lo propio, jefatura recibe datos operativos necesarios, RRHH y nómina actúan en su materia, auditor consulta trazas autorizadas y soporte solo metadatos; delegaciones y suplencias tendrán acto, alcance y caducidad, sin sumar perfiles.

**Por confirmar:** RRHH y cada dueño del dato aprueban la matriz funcional, doble control y firma; Sistemas/Seguridad fijan publicación, revocación y activación de suplencias. Esta matriz debe acordarse con la gobernanza global de la duda 31.

## 30. Fuente de identidades y ciclo de vida de usuarios

**Contraste:** AutenticA ilustra un repositorio horizontal procedente de fuentes primarias y altas delegadas, con atributos de unidad y puesto y autorización limitada. Un identificador inmutable evita enlazar cuentas por nombre o correo; debe distinguirse del identificador de la persona y de la relación laboral.

- [DATAOBSAE: AutenticA, descripción del indicador de usuarios](https://dataobsae.administracionelectronica.gob.es/cmobsae3/panel/Panel.action?selectedScope=15): procedencia del repositorio, altas delegadas, atributos y alcance limitado de autorización.
- [SAP: How Users are Uniquely Identified](https://help.sap.com/docs/successfactors-employee-central/implementing-employee-central-core/how-users-are-uniquely-identified-in-sap-successfactors): GUID inmutable de persona e identificador externo mutable.
- [ENS, anexo II, op.acc.1 y op.acc.4](https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191#aii): identificación singular y revisión de derechos al terminar la relación o cambiar funciones.

**Propuesta para VEC:** parametrizar un directorio autorizado y una vinculación verificable entre cuenta y persona, con eventos versionados de alta, cambio, suspensión y baja más conciliación de estado. Al recibir una baja se revoca el acceso correspondiente; el plazo máximo de propagación y antigüedad admitida de la información serán parámetros obligatorios antes de activar el conector, sin convertir atributos o grupos en permisos funcionales.

**Por confirmar:** Sistemas identifica directorio, propietario del identificador estable, interfaz, autenticación y responsable operativo; Seguridad fija plazo de revocación, tratamiento de caídas y verificación de cuentas administrativas separadas. RRHH confirma qué eventos laborales alimentan el directorio sin equiparar baja de cuenta y cese jurídico.

## 31. Gobierno de roles y asignaciones

**Contraste:** el ENS separa responsabilidades y limita derechos mediante autorización expresa; SAP aporta auditoría de cambios de roles, grupos y asignaciones. Su documentación también advierte que retirar permisos parciales puede dejar otra vía administrativa activa: conviene comprobar la revocación efectiva.

- [ENS, artículos 13/17/20 y anexo II op.acc.3–4](https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191): responsables diferenciados, mínimo privilegio y segregación aplicable.
- [SAP: Change Audit Use Cases](https://help.sap.com/docs/successfactors-platform/creating-change-audit-reports/change-audit-use-cases): informes de roles, grupos, asignaciones y administración de accesos delegados.
- [SAP KBA 2608722: revocación administrativa](https://userapps.support.sap.com/sap/support/knowledge/en/2608722): la vista pública documenta acceso administrativo que puede persistir tras retirar permisos RBP; el detalle completo requiere login.

**Propuesta para VEC:** configurar proponente, revisor, publicador y revocador por capacidad y dominio; roles/asignaciones sensibles exigirán dos personas distintas, motivo, simulación de accesos y versión nueva antes de publicar. La emergencia permanecerá deshabilitada hasta aprobar responsables, finalidad, duración y revisión; al habilitarse será temporal y auditada, sin administrador universal.

**Por confirmar:** RRHH, responsables de los datos, Sistemas y Seguridad nombran los órganos que aprueban cada capacidad, incompatibilidades, suplencias y revisión periódica. La política de emergencia debe identificar aprobador, alertas y revisor posterior; no basta otra confirmación de la misma persona.

## 32. Frontera de acceso a Administración

**Contraste:** FNMT diferencia certificado válido y revocado; @firma documenta entornos distintos para pruebas e integración y exige alta autorizada de aplicaciones. Validar el certificado acredita identidad según el mecanismo admitido, pero no concede permiso administrativo ni firma un acto.

- [FNMT: verificar estado](https://www.sede.fnmt.gob.es/certificados/administracion-publica/verificar-estado): comprobación de validez/revocación y clases de certificado, incluido DNIe.
- [Servicios de firma: FAQ de integración @firma, preguntas 1/6/9/10](https://sede.administracionespublicas.gob.es/ayuda/faq/Serviciosfirma): entornos de desarrollo/producción, admisión de aplicaciones, proveedores y kit de certificados de prueba.
- [ENS, anexo II op.acc y op.com](https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191#aii): control de acceso, autenticación y protección de comunicaciones según categorización/política.

**Propuesta para VEC:** conservar E05: pruebas en admin.cidonia.cloud con DNIe/FNMT válido, revocación comprobada y concesión nominal privada; producción en intranet. Emisor, audiencia, CA, orígenes, terminación de identidad y validador serán configuración privada obligatoria, y las pruebas usarán identidades sintéticas autorizadas para comprobar válido+autorizado, válido+no autorizado, revocado, caducado y fallo del validador, sin aceptar cabeceras libres ni persistir credenciales en la web.

**Por confirmar:** Sistemas/Seguridad confirma proveedor, cadena de confianza, terminación TLS/aserción, OCSP/CRL o servicio admitido, política de frescura y origen, responsables y entorno/kit aprobado. La FAQ demuestra que existe un kit público de referencia; no acredita el acceso local ni que sus certificados cubran todas las clases requeridas por VEC.

## 33. Configuración funcional compartida

**Contraste:** DIR3 mantiene un inventario común con responsabilidad distribuida; SAP separa objetos de configuración, permisos y auditoría, y fecha determinadas reglas de calendario. Estos mecanismos permiten localizar la fuente y conservar versiones utilizadas sin reescribir el pasado.

- [PAe: DIR3](https://administracionelectronica.gob.es/ctt/dir3): inventario común de unidades/oficinas y mantenimiento distribuido corresponsable; servicios de consulta/descarga.
- [SAP: Defining the Holiday Planned Working Time](https://help.sap.com/docs/successfactors-employee-central/implementing-time-management-in-sap-successfactors/defining-holiday-planned-working-time): reglas de tiempo en festivos con fecha efectiva.
- [SAP: Administrator Permissions for Employee Central](https://help.sap.com/docs/successfactors-employee-central/implementing-employee-central-core/administrator-permissions-for-employee-central): permisos separados sobre objetos, estructuras y configuración.
- [SAP: Change Audit](https://help.sap.com/docs/successfactors-platform/creating-change-audit-reports): cambios de configuración registrados con autor, momento y contenido del cambio.

**Propuesta para VEC:** inventariar calendario, unidad/centro, plantilla, regla, parámetro y conector con dueño, fuente, versión, vigencia, ámbito y circuito de publicación; dejar sin activar los elementos cuya fuente no esté confirmada. Preparación y publicación sensible serán funciones distintas; recuperar una versión generará una nueva publicación y análisis de impacto, preservando la versión de cada expediente y manteniendo en origen los maestros corporativos.

**Por confirmar:** RRHH y los propietarios funcionales deciden qué administra VEC y qué consulta; Sistemas concreta fuentes/interfaces y responsables de conectores. DIR3 puede aportar códigos de interoperabilidad, pero no se presume suficiente para toda la estructura, centros o competencias internas de Diputación.

## 34. Perfil del usuario y campos editables

**Contraste:** SAP distingue contacto, dirección, identidad y datos de pago, cada uno con permisos; su autoservicio permite configurar qué transacciones ofrece al titular. La AEPD exige exactitud y un cauce de rectificación, sin convertir el derecho en edición irrestricta de cualquier dato.

- [SAP: Personal Data](https://help.sap.com/docs/successfactors-employee-central/implementing-employee-central-core/personal-data): bloques distintos de datos personales, contacto, identidad y pago con requisitos/permisos propios.
- [SAP: Employee Self-Service](https://help.sap.com/docs/successfactors-employee-central/implementing-employee-central-core/708a728e453c435581564ecb1819281a.html): transacciones permitidas según configuración y autorización.
- [AEPD: principios](https://www.aepd.es/derechos-y-deberes/cumple-tus-deberes/principios): finalidad, minimización y exactitud.
- [AEPD: derecho de rectificación](https://www.aepd.es/derechos-y-deberes/conoce-tus-derechos/derecho-de-rectificacion): solicitud motivada sobre datos inexactos/incompletos.

**Propuesta para VEC:** una sola persona canónica, con autoridad y política de edición por campo; el correo obligatorio procede del alta propia de VEC conforme E09 y el resto queda de consulta/solicitud de corrección hasta aprobar el diccionario. La configuración podrá habilitar teléfono, domicilio y preferencias del titular con validación, verificación cuando proceda, historia y confirmación del origen; discrepancias quedarán pendientes sin sincronización bidireccional libre ni cambios de recibos históricos.

**Por confirmar:** RRHH/Sistemas confirman maestro por campo para empleado y aspirante, obligatoriedad, formatos, cambios verificables, sincronización y responsable de discrepancias. DPD valida necesidad y visibilidad; no se exige vínculo laboral para crear el perfil de aspirante.

## 35. Preferencias y canales de contacto

**Contraste:** AEAT ofrece suscripción, modificación y consulta de avisos informativos; DEHú permite gestionar datos de contacto. La Ley 39/2015 separa el aviso por correo/dispositivo de la práctica y acreditación de la notificación.

- [AEAT: suscripción a avisos informativos](https://sede.agenciatributaria.gob.es/Sede/procedimientoini/FZ04.shtml): alta/modificación, consulta y canales correo/móvil.
- [Ley 39/2015, artículos 41 y 43](https://www.boe.es/buscar/act.php?id=BOE-A-2015-10565#a41): aviso de puesta a disposición distinto de notificación y evidencia de esta.
- [Confederación Hidrográfica del Júcar: avisos y contacto DEHú](https://www.chj.es/preguntaweb/Pregunta/197): gestión de correos y teléfono para avisos sobre notificaciones disponibles.

**Propuesta para VEC:** configurar clases de comunicación y canales autorizados con preferencia por finalidad, fecha y versión; avisos opcionales exteriores estarán desactivados hasta elección del titular, mientras los administrativos seguirán la regla específica aprobada y no un interruptor global. Conservar fuente/versiones de la preferencia, destino aplicable, cambio autenticado y recibo técnico de despacho, usando el correo del alta VEC; un despacho no acreditará comparecencia, entrega legal o aceptación.

**Por confirmar:** RRHH/Secretaría clasifican comunicaciones obligatorias, informativas y optativas y sus efectos; Sistemas aprueba correo corporativo y los demás canales realmente disponibles. DPD confirma fundamento y evidencia por preferencia; los ejemplos de SMS no autorizan implantarlo localmente.

## 36. Auditoría y soporte de Administración y Usuarios

**Contraste:** el ENS prevé registros de actividad e incidentes y protección de las trazas; SAP permite informes separados sobre datos personales, permisos y configuración. La minimización y la finalidad también limitan la información que recibe soporte.

- [ENS, anexo II op.exp.8/op.exp.9 y op.acc.3](https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191#aii): trazabilidad, gestión de incidentes y segregación de auditoría.
- [SAP: Change Audit](https://help.sap.com/docs/successfactors-platform/creating-change-audit-reports): identifica registros modificados, autor e instante, incluso por API/importación.
- [SAP: Creating a Change Audit Report](https://help.sap.com/docs/successfactors-platform/implementing-and-managing-data-protection-and-privacy/creating-change-audit-report): permiso específico para generar informes y parámetros de selección.
- [AEPD: principios](https://www.aepd.es/derechos-y-deberes/cumple-tus-deberes/principios): acceso limitado a finalidad y conservación necesaria.

**Propuesta para VEC:** soporte consultará por defecto estado de servicios, código de error y correlación opaca; cada incidencia derivará a Sistemas, autoridad del dato o Seguridad sin suplantación ni expedientes/contactos en claro. La consulta/exportación de trazas requerirá concesión expresa, finalidad, periodo y campos minimizados; registros segregados de solo adición y conservación por clase aprobada, sin fijar un plazo universal ni trasladar los 48 h de descarga de informes SAP a la retención de VEC.

**Por confirmar:** Seguridad/DPD/Archivo fijan lectores, exportadores, conservación, custodia y atención de incidentes; RRHH define el circuito de discrepancias y Sistemas el diagnóstico y escalado. Debe distinguirse la bitácora técnica del expediente de soporte y del historial administrativo, con responsables y permisos propios.

## 37. Avisos personales de oportunidades

**Contraste:** evaluar requisitos de acceso contra hechos con fuente, estado y fecha, separando esa evaluación de la admisión y de la puntuación de méritos. La ausencia de un dato en un registro deja una comprobación pendiente; una previsión de terminar estudios no equivale a poseer el título ni a estar en condiciones de obtenerlo.

- [TREBEP, artículo 56.1.e y 56.3](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719#a56): exige poseer la titulación y permite requisitos específicos objetivos, proporcionados y generales; sostiene que el aviso debe comprobar requisitos de las bases, sin conceder excepciones individuales.
- [Convocatoria AGE/INAP de 18/12/2025, apartado 3.7](https://www.boe.es/diario_boe/txt.php?id=BOE-A-2025-26262): exige al fin del plazo poseer el título o estar en condiciones de obtenerlo; define esto último mediante certificado de todas las asignaturas o créditos superados y, cuando proceda, tasas abonadas. Ejemplo oficial concreto; no acredita una excepción para seguir estudiando después del plazo.
- [Ministerio de Ciencia: consulta al Registro Nacional de Titulados Universitarios Oficiales](https://www.ciencia.gob.es/Universidades/ConsultaTitulos.html): identifica el registro de títulos, la descarga de certificados y la autorización a terceros con CSV; incluye credenciales oficiales de homologación/equivalencia. El certificado o consulta autorizado puede aportar evidencia, siempre con el contrato y la finalidad aprobados.
- [Ministerio de Educación: consulta a registros de títulos no universitarios](https://www.educacionfpydeportes.gob.es/gl/servicios-al-ciudadano/catalogo/general/20/202363/ficha/202363.html): solo consulta títulos expedidos desde 1991 y excluye certificados de profesionalidad, pruebas de acceso y equivalencias, entre otros. Sostiene que «no encontrado» no permite deducir «no cumple».
- [Ministerio de Ciencia: preguntas frecuentes de consulta de títulos, apartados 5 y 6](https://www.ciencia.gob.es/dam/jcr%3A6610c941-8c9c-4f36-9aed-87b2aa2bdd10/RNTOU_PreguntasFrecuentes_acc.pdf): excluye títulos propios y títulos de especialista expedidos por Sanidad del RNTUO. Las fuentes y verificadores deben variar por tipo de título.

**Propuesta para VEC:** por defecto, requisitos estructurados y versionados con resultado `cumple`, `no_cumple` o `pendiente`, motivo, hito y evidencia; mostrar «No podemos comprobar [requisito]. Revisa las bases y aporta la documentación indicada. Este aviso no decide tu admisión». Al pulsar el aviso se abrirá la ficha pública de la oferta con bases, requisitos, plazo, documentos y vía de solicitud, para empleados y aspirantes externos. La inscripción con titulación pendiente permanece desactivada; solo una OPE cuyas bases y revisión competente admitan expresamente el supuesto puede configurarla, con hito, evidencia y revalidación obligatorios, y nunca se traslada a bolsas de incorporación inmediata.

**Por confirmar:** RRHH debe designar quién prepara, revisa y publica los requisitos y qué documento acredita cada título, equivalencia y hito; Sistemas debe confirmar los servicios autorizados de consulta. Hay que distinguir «título obtenido pendiente de expedición/acreditación» de «estudios aún sin terminar»: las fuentes consultadas no justifican una habilitación general de este segundo supuesto para OPE.

## 38. Jornada contratada y turnos

**Contraste:** conservar por separado jornada contratada, referencia de jornada completa, periodo de cómputo y distribución de turnos. La norma provincial distingue régimen general y centros/turnos con organización propia, por lo que una semana concreta del cuadrante no determina por sí sola la fracción del contrato.

- [Diputación: Resolución 29/12/2015 sobre jornada, horario, vacaciones y permisos, página 1, apartados 1, 2.1 y 2.2.d](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/RESOLUCION-2015-SOBRE-JORNADA-HORARIO-VACACIONES-Y-PERMISOS.pdf): fija 37 horas y media generales para personal no sometido a turnos, limita la aplicación a turnos a lo compatible con su régimen y contempla compensar diferencias semanales en cómputo mensual dentro del horario flexible. Debe confirmarse su vigencia y modificaciones antes de activarla en producción.
- [Diputación: Reglamento regulador del tiempo de trabajo de 2010, artículos 1.2, 2.4 y 3.2](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/REGLAMENTO-TIEMPO-DE-TRABAJO.pdf): remite la jornada de Centros Sociales a sus normas y la distribución de turnos a un cómputo anual fijado para 2003. Acredita la existencia de regímenes diferenciados; no permite reconstruir las horas anuales actuales sin sus acuerdos.
- [TREBEP, artículo 47](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719#a47): atribuye a las Administraciones la jornada general y las especiales y contempla tiempo completo/parcial para funcionarios. Sostiene un catálogo por ámbito, sin fijar 37,5 horas universalmente.
- [Estatuto de los Trabajadores, artículos 12 y 34](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11430#a12): el contrato parcial se compara con un trabajador a tiempo completo comparable, con reglas subsidiarias si no existe; la jornada ordinaria máxima se expresa como promedio anual y admite distribución irregular en sus condiciones legales. Se aplica al personal laboral, no como regla automática para todo vínculo.

**Propuesta para VEC:** conservar provisionalmente 37 h 30 min (2.250 minutos) como ejemplo del régimen general y calcular la fracción con minutos contratados/referencia del mismo ámbito y periodo; catálogo configurable por régimen, colectivo, categoría, centro, vigencia, periodo, fuente y versión, con cómputo sin redondeo intermedio. Mantener el rechazo actual de una media contractual superior al 100 % mientras RRHH no determine su tratamiento; los excesos de semanas concretas, su compensación y los turnos pertenecen a Cronos y no elevan automáticamente la jornada del contrato.

**Por confirmar:** RRHH debe confirmar jornadas completas actuales por ámbito, número de horas y periodo para turnos, acuerdos de Centros Sociales, equivalencia entre jornada anual y media semanal, redondeo de presentación y tratamiento de los supuestos superiores a la referencia. El acuerdo de 2015 aporta una base verificable para 37,5 horas generales, pero no acredita hoy una regla universal.

## 39. Procesos de RRHH y aplicaciones actuales

**Contraste:** inventariar procesos junto con su unidad responsable, aplicación, fuente maestra e intercambios, conservando una persona con varios vínculos y las fechas de sus efectos. La estrategia provincial ya pide inventario y normalización; la documentación SAP muestra como patrón técnico la conciliación entre persona, empleos y números de personal, con estados de integración verificables.

- [Diputación: Estrategia integral de modernización de 18/02/2025, apartados IV.A, IV.C y anexo I](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/2025_19_2_CERTIFICADO_NA_4_JUNTA_GOBIERNO_18_02_2025.pdf): pide inventario de escalas/categorías, funciones, competencias, vacantes y duplicidades, y protocolos con responsables, canales y pasos; su diagnóstico identifica certificados de servicios como cuello de botella. Fundamenta el inventario previo y una posible prioridad de certificados, sin darla por elegida por RRHH.
- [Diputación: Registro de Actividades de Tratamiento, apartado 2.17 y actualización de 29/11/2022](https://www.dipgra.es/servicios/areas/transparencia/portal-de-transparencia/g-actividades-de-tratamiento-de-datos-personales/registro-de-actividades-de-tratamiento/): publica el registro institucional y la versión del tratamiento de RRHH. Sirve para identificar finalidades y responsables que deben verificarse antes de conectar datos; no concede acceso común a todas las materias.
- [SAP SuccessFactors: Setting Up SAP SuccessFactors for Replication](https://learning.sap.com/courses/sap-successfactors-employee-central-integration-with-sap-hcm-for-sap-s-4hana/setting-up-sap-successfactors-for-replication): distingue persona/múltiples empleos de números de personal y periodos de validez del receptor; exige correspondencias y ofrece confirmaciones y estados de transferencia. Es un patrón de producto, sin acreditar versiones ni módulos contratados de GINPIX ni su integración con VEC.

**Propuesta para VEC:** inventario configurable y versionado por proceso, con propietario funcional/técnico, aplicación y versión/módulo contratado, volumen/tiempo/incidencias, norma o acuerdo, fuente maestra por atributo, identificadores e historia, interfaz autorizada, frecuencia, conciliación, acuses, permisos y conservación. Mantener como punto de partida los sistemas que hoy tengan autoridad acreditada y priorizar servicios/certificados como candidata de mejora tras Contratación por el problema documentado, pendiente de decisión RRHH; la sustitución de nómina requiere un encargo y validación propios.

Detalle del inventario propuesto (inferencia de diseño, no atribución de responsabilidades actuales):

| Proceso a inventariar | Fuente maestra/propietario que debe acreditar RRHH | Identificadores, historia y resultado que conviene comprobar |
| --- | --- | --- |
| Persona, identidad y contacto | Autoridad común de persona; origen corporativo de cada atributo pendiente de acuerdo | Persona estable, documentos anteriores, contacto verificado, origen y cambios; separar cuenta de persona y de vínculo |
| Registro de Personal y relaciones | Registro y unidad de Personal competentes, incluidos organismos autónomos | Organismo, relación, nombramiento/contrato/cese, situación, fechas de efecto y registro, varios vínculos sin duplicar persona |
| Plantilla, plazas, RPT y ocupaciones | Actos aprobados de plantilla/RPT y unidad de ordenación; Personal para ocupación efectiva | Plaza y puesto separados, versiones, correspondencias de categorías, vacantes y periodos de ocupación |
| Servicios, antigüedad y certificados | Personal para servicios propios y actos de reconocimiento; organismo emisor para servicios externos | Periodo, vínculo, jornada, solapes/rectificaciones, servicio declarado frente a reconocido, certificado firmado y asiento |
| Selección, Bolsa y contratación temporal | Bases y actos de Selección; Bolsa conserva participaciones/orden; Contratación conserva expediente | Convocatoria/bases/versiones, participación, llamamiento, expedientes y entrega confirmada a Personal |
| Jornada, presencia, permisos y teletrabajo | Acuerdos/calendarios por colectivo/centro; Personal para jornada contratada y Cronos para ejecución | Calendario/periodo, autorización, solicitud/resolución, fichajes y ajustes; no usar fichajes como antigüedad |
| Altas/bajas, IT y Seguridad Social | Personal para actos del vínculo; servicio gestor y TGSS/INSS para respuestas oficiales | CCC, NAF y referencias externas protegidas, fecha de efectos, envío/aceptación/rechazo y conciliación; IT con información administrativa mínima |
| Contratos SEPE y certificados de empresa | Unidad gestora y canales SEPE autorizados que declare la Diputación | Contrato/prórroga/certificado, referencia de comunicación, acuse, error y corrección; exportación no equivale a recepción |
| Nómina, cotización, IRPF y pago | Nómina y gestor vigente para cálculo; AEAT/TGSS y Tesorería/Contabilidad para sus confirmaciones | Periodo, concepto/regla/versiones, regularización, liquidación, declaración y pago/conciliación por separado |
| Carrera, formación y acción social | Acuerdo/convenio/convocatoria aplicable y unidad competente por materia | Grado o derecho reconocido, asistencia/certificado, aprobación y pago separados; vigencia por colectivo |
| Igualdad, acoso, disciplina y prevención | Unidades y custodios específicos que se designen; salud laboral bajo su custodia propia | Protocolo vigente, acceso por finalidad y mínima conclusión comunicable a RRHH, sin trasladar el expediente sensible completo |
| Documentos, firma y archivo | Registro, portafirmas y Archivo con sus autoridades existentes | Tipo/versión, firmante competente, original/CSV, asiento, serie documental, transferencia y custodia |

**Por confirmar:** cada unidad debe completar la fila de sus procesos y Sistemas los módulos/versiones/contratos/interfaces de GINPIX y demás aplicaciones; RRHH debe aprobar una fuente maestra por atributo y las correspondencias de identificadores con los organismos autónomos. Quedan por recibir acuerdos/convenios y protocolos vigentes, volúmenes, problemas y certificados más frecuentes; la prioridad siguiente y cualquier sustitución de nómina exigirán responsables de validación, conciliación paralela del histórico y cálculo/cotización/fiscalidad/pago, criterios de aceptación y recuperación acordados, sin umbrales inventados.
