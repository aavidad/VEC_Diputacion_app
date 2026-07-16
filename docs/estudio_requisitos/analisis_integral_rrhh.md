# Análisis integral del portal de Recursos Humanos

Fecha de corte del estudio: **16 de julio de 2026**.

Estado: **especificación de referencia en elaboración y NO-GO productivo** hasta
que RRHH, Secretaría/Asesoría Jurídica, Intervención, Archivo, DPD, Seguridad y
Sistemas validen las materias de su competencia.

## 1. Decisión ejecutiva

El producto será una plataforma integral de Recursos Humanos de la Diputación,
no una aplicación monolítica de nómina ni una suma de formularios aislados.
Compartirá identidad, persona, autorización, auditoría, documentos, firma,
calendarios, comunicaciones y procedencia de los datos, pero cada materia
conservará su propio dominio, reglas, permisos y ciclo de vida.

Esta separación permite cumplir simultáneamente cuatro necesidades:

1. que los datos acreditados de una persona se reutilicen sin volver a pedirlos;
2. que Bolsa, Personal, RPT, Cronos, Nómina o Prevención no reinterpreten el
   mismo hecho de formas contradictorias;
3. que un permiso sobre una finalidad no abra datos de otra;
4. que una nueva norma, convenio, base o módulo se incorpore sin recompilar el
   núcleo ni alterar expedientes cerrados.

La aplicación no decidirá derechos por semejanza, por una constante de código
o por una práctica observada en otra Administración. Toda decisión funcional
deberá poder responder:

- qué norma, convenio, acuerdo, base o resolución estaba vigente;
- a qué clase de personal, organismo, puesto, centro y periodo se aplicaba;
- qué hechos oficiales y documentos se utilizaron;
- qué regla versionada produjo el resultado;
- quién lo revisó, aprobó o firmó;
- cómo se rectificó sin borrar el resultado anterior.

## 2. Por qué el análisis completo es necesario ahora

No es necesario desarrollar ahora Nómina, Prevención, Acción Social o la
totalidad de Cronos. Sí es necesario fijar sus fronteras y los hechos que
intercambiarán antes de seguir ampliando el núcleo y Bolsa.

Hay reglas futuras que afectan ya a Bolsa. Por ejemplo, la disponibilidad tras
un cese temporal se aplica a la persona en todas las bolsas, no solo a una
participación concreta. Personal debe acreditar el nombramiento, contrato y
cese; Bolsa aplica la política de llamamiento vigente. Si el núcleo guardase
una constante de cinco o nueve meses, o si se copiase la restricción en cada
bolsa, sería necesario refactorizar y reconciliar múltiples historiales.

En cambio, el cálculo de una gratificación por turnicidad, la consolidación de
grado o una ayuda social no forman parte de Bolsa. Ahora solo se documentan sus
propietarios, contratos y requisitos de trazabilidad. Sus reglas se implantarán
cuando se aborde el módulo correspondiente y después de validar toda la cadena
normativa provincial aplicable.

## 3. Jerarquía de fuentes y puerta previa al código

El orden de trabajo obligatorio es:

1. norma de la Unión Europea y legislación estatal aplicable;
2. legislación de Andalucía y normativa local supletoria o de desarrollo;
3. BOP, acuerdos, convenio, reglamentos, resoluciones, circulares, RPT,
   plantilla, bases y procedimientos de la Diputación;
4. instrucciones operativas y funcionamiento real contrastado con la unidad;
5. prácticas de otras Administraciones y productos existentes, únicamente
   como referencia para mejorar el diseño.

Una fuente posterior no se presume texto consolidado de todas las anteriores.
Cada ficha de regla conservará fuente, versión, fecha de publicación, vigencia,
fecha de efectos, ámbito, órgano competente, modificaciones y estado de
validación. Una contradicción produce una incidencia jurídica o un resultado
`no determinable`; nunca se resuelve eligiendo silenciosamente el texto más
conveniente.

Las prácticas de otra Administración pueden aportar mejores pantallas,
explicaciones, estados o automatización, pero no crear requisitos, derechos,
baremos, plazos, importes ni consecuencias disciplinarias en la Diputación.

## 4. Mandato institucional de la Diputación

### 4.1 Estrategia de modernización de 2025

La Estrategia integral aprobada por la Junta de Gobierno el 18 de febrero de
2025 constituye una fuente institucional prioritaria. Su diagnóstico identifica
más de 175 categorías o especialidades, solapamientos, bolsas insuficientes,
certificados de servicios como cuello de botella, gestión reactiva de las
necesidades y comunicaciones informales.

La plataforma debe dar soporte comprobable a sus objetivos:

- inventariar escalas, subescalas, categorías, especialidades, personal activo,
  vacantes, funciones y competencias;
- conservar la correspondencia histórica cuando se agrupen o extingan
  categorías;
- normalizar protocolos, impresos, responsables, canales y plazos;
- automatizar certificados sin perder revisión, firma y expediente;
- planificar necesidades y crédito antes de que aparezca la urgencia;
- medir tiempos de procesos selectivos y solicitudes de personal;
- medir cobertura de categorías mediante bolsas;
- ampliar y flexibilizar bolsas conforme al reglamento vigente.

Fuente oficial: [Estrategia integral de modernización, simplificación y
optimización de la gestión del personal](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/2025_19_2_CERTIFICADO_NA_4_JUNTA_GOBIERNO_18_02_2025.pdf).

### 4.2 Registro de Actividades de Tratamiento

El apartado 2.17 del Registro de Actividades de Tratamiento provincial incluye
en la finalidad de Recursos Humanos:

- toma de posesión, contratos, altas y bajas;
- gestión económica, nóminas, trienios, dietas y anticipos;
- permisos, vacaciones, incompatibilidades y régimen disciplinario;
- órganos de representación;
- selección y provisión;
- control horario, geolocalización y eventual verificación biométrica;
- formación, planes de pensiones y acción social;
- prevención de riesgos y vigilancia de la salud.

También declara datos identificativos, académicos, laborales, económicos,
geolocalización, afiliación sindical y salud, además de comunicaciones a
entidades tributarias, financieras, aseguradoras, Seguridad Social y órganos
judiciales.

Esto acredita el alcance institucional, pero **no autoriza un acceso común e
indiferenciado**. El nuevo sistema deberá revisar y actualizar el RAT, las bases
jurídicas, destinatarios, conservación y medidas de seguridad de cada finalidad.
La biometría, la geolocalización, la vigilancia de la salud, las denuncias y la
afiliación sindical requieren compartimentos, perfiles y registros de acceso
específicos, además de la evaluación jurídica y de impacto que corresponda.

Fuente oficial: [Registro de Actividades de Tratamiento de la Diputación](https://www.dipgra.es/servicios/areas/transparencia/portal-de-transparencia/g-actividades-de-tratamiento-de-datos-personales/registro-de-actividades-de-tratamiento/).

### 4.3 Sistemas actuales que hay que inventariar, no duplicar

La información pública y la documentación local acreditan al menos:

- GINPIX 7/SAVIA y CONVOCA para personal, selección y bolsas;
- WCRONOS para control de presencia;
- Portale y el anterior Nominae como portales del personal;
- MOAD, sede electrónica, registro, firma y verificación mediante CSV;
- Active Directory/Kerberos para identidad corporativa;
- correo corporativo y los canales de comunicación existentes.

Antes de integrar datos reales se elaborará para cada sistema una ficha con:

- propietario funcional y técnico;
- dato maestro y datos derivados;
- interfaz disponible, límites, versión y soporte;
- identificadores y reglas de reconciliación;
- latencia, horario de actualización y comportamiento ante fallo;
- base jurídica y finalidad de cada intercambio;
- contrato de servicio, auditoría y plan de retirada.

Ningún módulo leerá directamente tablas ajenas ni escribirá en GINPIX, Nómina,
WCRONOS o Active Directory. Todo intercambio pasará por un puerto y un adaptador
aprobado, con instantánea, procedencia, idempotencia y conciliación.

## 5. Modelo conceptual que no se puede simplificar

### 5.1 Persona, cuenta y relación de servicio

Se separarán:

- persona física canónica;
- identificadores legales y corporativos protegidos;
- cuentas e identidades de acceso;
- representación de otra persona;
- perfil de aspirante;
- relación de servicio;
- régimen jurídico;
- vínculo y causa temporal;
- situación administrativa o laboral;
- cargo, mandato o función directiva;
- adscripción y ocupación de plaza o puesto.

La plantilla provincial de 2026 distingue personal funcionario, laboral y
eventual, además de personal procedente de otras Administraciones. Esa
procedencia no es una cuarta clase de empleo. La aplicación deberá representar,
como mínimo, funcionario de carrera e interino; laboral fijo, indefinido y
temporal; eventual; y las condiciones separadas de personal directivo, alto
cargo o cargo electo.

No se usará un único campo `tipo_personal`. El vínculo real no se deduce de la
columna de adscripción de la RPT: se acredita mediante Registro de Personal,
nombramiento, contrato y, solo como reconciliación, nómina.

### 5.2 Categoría, plaza, puesto y ocupación

Son conceptos diferentes:

- **categoría, cuerpo, escala, subescala o especialidad:** clasificación
  profesional de la persona o de la plaza;
- **plaza:** dotación de plantilla con cobertura presupuestaria;
- **puesto tipo:** definición común de funciones y requisitos;
- **puesto individual:** unidad organizativa identificada en la RPT;
- **dotación:** número o unidad presupuestada;
- **ocupación:** relación temporal entre una persona y una plaza o puesto;
- **reserva:** derecho o condición que impide tratar una ausencia como vacante
  libremente cubrible.

La historia de plantilla, RPT, modificaciones, ocupaciones y reservas será
bitemporal: fecha de efectos administrativos y fecha en que el sistema conoció
el hecho. Una vacante presupuestaria, un puesto sin ocupante y una necesidad
cubrible son proyecciones diferentes.

### 5.3 Nivel del puesto, grado y categoría laboral

Se separarán siempre:

- nivel de complemento de destino del puesto;
- grado personal consolidado de una persona funcionaria;
- grado o nivel laboral reconocido conforme al convenio y sus modificaciones;
- grupo/subgrupo funcionarial;
- grupo profesional laboral;
- nivel o categoría económica y efectos retributivos.

No existe un examen periódico universal de «subida de nivel». La consolidación
por desempeño y la vía mediante curso y prueba tienen requisitos, límites,
incompatibilidades y actos de reconocimiento propios. El sistema podrá detectar
personas potencialmente elegibles y preparar un expediente, pero el grado solo
cambiará por resolución competente e inscripción en el Registro de Personal.

El análisis provincial completo se mantiene en
[modelo histórico de RPT, plazas, puestos y vacantes](modelo_historico_rpt_plazas_puestos_y_vacantes.md).

## 6. Contextos funcionales propuestos

| Contexto | Es dueño de | No puede hacer |
| --- | --- | --- |
| Identidad y acceso | cuentas, autenticación, sesiones, perfiles activos, capacidades y delegaciones de acceso | convertir un grupo de AD o un menú en permiso funcional |
| Personas y representación | identidad canónica, contacto, preferencias, representación y procedencia | declarar una relación laboral o una plaza por observación de una cuenta |
| Organización, plantilla y RPT | organismos, unidades, centros, categorías, plazas, puestos, dotaciones y sus versiones | decidir por sí solo quién ocupa o puede cubrir un puesto |
| Registro de Personal | relaciones de servicio, nombramientos, contratos, ceses, situaciones, ocupaciones, antigüedad y grados reconocidos | calcular nómina o imponer reglas de Bolsa |
| Planificación de efectivos | necesidades, escenarios, coste estimado, crédito, plantilla, OEP y cobertura | crear una contratación o alterar presupuesto sin expediente |
| Selección y OPE | convocatorias, solicitudes, admisión, pruebas, tribunales, listados, alegaciones y nombramiento propuesto | gestionar la disponibilidad ordinaria de una bolsa ya constituida |
| Bolsa | constitución, orden, disponibilidad, preferencias, llamamientos, renuncias, carencias y propuesta de incorporación | inventar contratos, ceses o servicios prestados |
| Registro Único de Méritos | hechos académicos y profesionales, evidencias, verificaciones, vigencia y reutilización | trasladar puntos de una convocatoria a otra |
| Baremación | reglas publicadas, cálculos, revisión, rectificación, firma y explicación | admitir código, SQL o fórmulas libres introducidas por RRHH |
| Provisión y movilidad | concursos, libre designación, movilidad provisional o por salud, preferencias y adjudicación | reutilizar sin más el ranking de Bolsa |
| Carrera y desempeño | grado, carrera horizontal/vertical, promoción interna, objetivos y evaluación | modificar un grado o retribución sin resolución y registro |
| Calendarios | días naturales, hábiles, festivos, apertura de centro y versiones de calendario | decidir la jornada individual de una persona |
| Cronos | cuadrantes, jornada prevista, fichajes, teletrabajo, ausencias, permisos y saldos de tiempo | valorar importes de nómina o revelar causas médicas a una jefatura |
| Nómina y retribuciones | conceptos, tablas, incidencias, cálculo, atrasos, cierre, cotización, IRPF, pagos y recibos | reabrir o sobrescribir una nómina cerrada |
| Dietas y gastos | comisión de servicio, itinerario, anticipo, justificantes, liquidación y aprobación | usar la posición GPS de Cronos como justificación automática del gasto |
| Formación | planes, necesidades, acciones, plazas, selección, asistencia, evaluación y certificación | reconocer un grado o mérito fuera del procedimiento aplicable |
| Acción social y pensiones | convocatorias, modalidades, requisitos, presupuesto, comisión, ayudas, anticipos y aportaciones | exponer datos familiares o de salud a módulos no autorizados |
| Prevención y salud laboral | riesgos, aptitud, vigilancia, accidentes, adaptación y medidas preventivas | entregar diagnósticos clínicos a RRHH o responsables de unidad |
| Igualdad y protocolos reservados | planes, indicadores, impacto, medidas y procedimientos confidenciales | mezclar denuncias o víctimas con el expediente ordinario accesible |
| Relaciones laborales | negociación, órganos de representación, crédito sindical, incompatibilidades y disciplina | convertir afiliación sindical o una denuncia en atributo general del empleado |
| Expediente y archivo | documentos, expedientes, índices, series, acceso, conservación y transferencia | conceder acceso por compartir almacenamiento físico |
| Comunicaciones y notificaciones | plantillas, destinatarios, canales, entrega, comparecencia y recibos | considerar Telegram o correo una notificación administrativa por defecto |
| Analítica y transparencia | indicadores, cuadros de mando y publicaciones minimizadas | consultar tablas operativas sin una proyección y finalidad aprobadas |

## 7. Principios para las reglas configurables

La variabilidad ordinaria se resolverá con políticas declarativas, tipadas,
versionadas y publicadas desde la aplicación. Esto no significa aceptar
cualquier fórmula escrita por un operador.

Una política deberá declarar:

- identificador, versión, estado y ámbito;
- fuentes jurídicas y fecha de efectos;
- colectivos, puestos, centros y situaciones incluidos o excluidos;
- hechos de entrada y autoridad de cada hecho;
- operadores permitidos, unidades, redondeos, topes y prioridades;
- calendario y tratamiento de intervalos;
- documentos y revisiones requeridos;
- casos de prueba aprobados;
- impacto de igualdad y privacidad cuando proceda;
- quién prepara, revisa, publica, suspende o sustituye la política.

El ciclo será:

```text
borrador → simulada → revisada funcionalmente → revisada jurídicamente
→ aprobada/firmada → publicada → vigente → sustituida o anulada
```

Los expedientes conservarán la versión exacta que utilizaron. Corregir una
regla no recalcula silenciosamente actos firmes; genera un análisis de impacto,
una rectificación o el procedimiento que determine la unidad competente.

## 8. Intercambio de hechos entre módulos

Los módulos no compartirán entidades internas. Intercambiarán hechos mínimos,
opacos y firmados o atestados cuando produzcan efectos. Ejemplos:

- Personal publica un periodo de servicios, su régimen, jornada y fuente;
- Bolsa consume el cese y la clase de nombramiento para evaluar una restricción
  global de llamamiento;
- Cronos publica una incidencia de tiempo ya aprobada, sin diagnóstico;
- Nómina consume la incidencia y devuelve referencia de liquidación, no todos
  los conceptos al resto del sistema;
- Formación publica asistencia y superación; Carrera decide si esa evidencia
  cumple su convocatoria;
- Prevención comunica `apto`, `apto_con_limitaciones` o `no_apto` y las medidas
  funcionales necesarias, no la historia clínica;
- RPT publica los requisitos versionados de un puesto; Provisión evalúa la
  convocatoria exacta;
- Calendarios devuelve un cálculo reproducible con sus fuentes; Cronos aplica
  además el cuadrante de la persona.

Todo mensaje con efectos contendrá identificador, versión de esquema,
correlación, instante efectivo, instante de emisión, productor, finalidad,
referencias de fuente, idempotencia e integridad. Un fallo de integración no se
interpreta como cero, ausencia, vacante, permiso o cumplimiento.

## 9. Superficies de acceso

Se mantienen físicamente y lógicamente separadas:

1. portal público sin identidad para información publicada;
2. área personal exterior de aspirantes y representantes;
3. autoservicio interno del personal;
4. espacio interno de responsables de unidad;
5. tramitación interna especializada de RRHH;
6. administración funcional;
7. administración técnica, seguridad y auditoría privilegiada.

Una persona empleada que participa en una bolsa entra en el perfil de aspirante.
Un técnico de RRHH que consulta su propia nómina entra en el perfil de empleado.
Los privilegios no se suman en una misma sesión. Las superficies internas usan
red corporativa o VPN autorizada, audiencia y sesión propias, Kerberos/AD y la
autenticación reforzada aprobada; administración privilegiada usa además cuenta
nominativa separada, elevación temporal y puesto o bastión gestionado.

La pertenencia a una unidad no permite consultar todo su expediente. Una
jefatura obtiene únicamente los datos necesarios para organizar el servicio y
resolver la tarea: disponibilidad, saldo suficiente y efecto operativo, no
diagnósticos, afiliación, nómina, méritos de Bolsa o denuncias.

## 10. Clasificación inicial de la información

| Clase | Ejemplos | Frontera mínima |
| --- | --- | --- |
| Pública aprobada | convocatorias, bases, RPT publicada, calendarios generales, resultados autorizados | proyección específica, retirada y conservación aprobadas |
| Personal ordinaria | contacto, puesto, solicitudes propias, formación, servicios | titular o competencia y finalidad exactas |
| Económica reservada | nóminas, cuentas, retenciones, embargos, anticipos, ayudas | enclave interno, cifrado, acceso nominativo y exportación controlada |
| Laboral especialmente sensible | fichajes, ausencias, geolocalización, productividad, disciplina | red interna, ámbito organizativo y registro reforzado |
| Categoría especial o equivalente por riesgo | salud, discapacidad, afiliación sindical, violencia, adaptación | compartimento independiente y revelación mínima |
| Confidencial de investigación | acoso, denuncia, testigos, medidas cautelares | equipo autorizado ad hoc, seudónimos y barrera frente al expediente ordinario |
| Seguridad y privilegio | roles, claves, sesiones, auditoría, incidencias | plano de administración separado y registros inmutables |

La clasificación exacta y la categorización ENS se aprobarán formalmente. El
objetivo de diseño sigue siendo soportar ENS categoría ALTA; no se declarará
conformidad hasta disponer de categorización, análisis de riesgos, implantación,
auditoría y evidencias.

## 11. Hallazgos sobre el código actual

Los paquetes actuales de Personal, Cronos y Dietas son demostraciones útiles
para descubrir necesidades, pero no son autoridad funcional ni están listos
para datos reales.

| Hallazgo | Riesgo | Decisión |
| --- | --- | --- |
| `DefaultLeavePolicies` fija cupos y límites en Go | una modificación legal exige compilar y puede aplicar reglas obsoletas | sustituir por políticas publicadas y conservar solo casos sintéticos de prueba |
| `DailyReductionForAge` fija 63/64 años y una/dos horas | ignora régimen, vigencia, situación y posible modificación normativa | retirar del cálculo productivo; consumir una política y hechos efectivos |
| `Workday` recibe la edad | dato derivado mutable y no evidencia la fecha de nacimiento ni la regla | usar referencia de elegibilidad emitida por Personal, minimizada y fechada |
| `EmploymentRegime` mezcla relación, vínculo y cargo | decisiones erróneas para fijo, indefinido, temporal, eventual o directivo | reemplazo aditivo por el modelo separado del apartado 5.1 |
| `RPTPosition` mezcla fila, dotación, puesto, categoría y estado | no puede reconstruir historia ni vacantes | migración aditiva al modelo bitemporal ya especificado |
| `CategoryRule.PointsPerMonth` usa `float64` | resultados no exactos ni reproducibles | usar valores exactos compartidos y reglas versionadas |
| `PayrollDraft` solo suma conceptos y deducciones | no resuelve devengo, cotización, IRPF, atrasos, cierre o conciliación | conservar como demostración; diseñar el libro de nómina antes de ampliarlo |
| Dietas usa `float64` para kilómetros e importes | redondeos y ausencia de procedencia de ruta/tarifa | decimal exacto para dinero y distancia gobernada con fuente/versiones |
| permisos de manifiesto son demasiado amplios | `manage` no expresa acción, persona, unidad, finalidad o estado | mantener manifiestos, descomponer capacidades por caso de uso y añadir ABAC contextual |

Las pruebas actuales acreditan coherencia interna del prototipo, no corrección
jurídica. Estos paquetes quedan en **NO-GO productivo**. No se borran mientras
se diseñe la migración y no se conectarán a datos personales reales.

## 12. Qué se conserva sin obligar a refactorizar Bolsa

Se conservan:

- arquitectura hexagonal y composición externa;
- identificadores opacos y persona canónica;
- puertos pequeños e intercambiables;
- autorización positiva y denegación por defecto;
- auditoría, documentos, firma y calendarios transversales;
- aritmética exacta y periodos civiles;
- catálogos y políticas con versiones;
- bandeja y sistema de diseño comunes;
- eventos y proyecciones minimizadas.

Bolsa no importará tipos internos de Nómina, Cronos o Personal. Solo consumirá
contratos neutrales de hechos necesarios. La futura regla de carencia global se
representará una sola vez por persona y periodo, con causa y fuente, y será
evaluada por la política de llamamiento; no se copiará como estado mutable a
cada bolsa.

El desarrollo inmediato de núcleo y Bolsa puede continuar cuando respete estas
fronteras. Los módulos completos de RRHH se incorporarán de forma aditiva. El
núcleo no necesitará conocer sus menús, tablas, reglas o proveedores.

## 13. Validaciones institucionales pendientes

Antes de programar cada bloque se solicitará, como mínimo:

- texto consolidado o cadena completa del Acuerdo de funcionarios y Convenio
  laboral, con criterio sobre vigencia de cada modificación;
- inventario y diccionario del Registro de Personal, GINPIX, WCRONOS, Portale,
  Nómina, MOAD, AD y contabilidad;
- responsables y autoridad de persona, relación, plaza, puesto, ocupación,
  jornada, servicios, grado y concepto retributivo;
- reglamentos específicos de turnos y centros sociales;
- resolución y efectos de la revisión de oficio del artículo 17.4 del
  Reglamento de tiempo de trabajo iniciada en febrero de 2026;
- catálogo actual de permisos, justificantes, efectos, plazos y aprobadores;
- reglas de permanencia, provisión, promoción y carencias por proceso;
- circuito real de altas, nombramientos, contratos, ceses y toma de posesión;
- tablas, interfaces y controles del cálculo de nómina;
- políticas de conservación y tablas de valoración documental;
- RAT actualizado, análisis de riesgos, EIPD y criterio del DPD sobre datos de
  alto riesgo;
- categorización ENS, red y sistemas donde vivirá cada compartimento;
- vigencia o sustitución del Plan de Igualdad aprobado en 2022, cuyo plazo
  declarado de cuatro años aparenta haber finalizado en enero de 2026;
- catálogo de órganos, puestos competentes, delegaciones, suplencias y firmas.

## 14. Especificaciones relacionadas

- [Matriz normativa de Recursos Humanos](matriz_normativa_rrhh_2026.md)
- [Catálogo funcional y hoja de ruta](catalogo_funcional_rrhh_y_hoja_ruta.md)
- [Petición del Servicio de Selección Externa](peticion_rrhh_transcripcion_y_lectura.md)
- [Baremación configurable y datos de oficio](baremacion_configurable_jornada_y_datos_de_oficio.md)
- [RPT, plazas, puestos y vacantes](modelo_historico_rpt_plazas_puestos_y_vacantes.md)
- [Calendario hábil y laboral histórico](calendario_habil_laboral_historico.md)
- [Turnos, festivos y compensaciones](turnos_festivos_y_compensaciones.md)
- [Archivo documental relacionado](archivo_documental_rrhh_relacionado.md)
- [Seguridad y despliegue de Cronos](seguridad_y_despliegue_cronos.md)
- [Acceso interno de técnicos y administración](acceso_interno_tecnicos_administracion.md)
- [Comparativa de portales públicos](../referencias_portales_aapp/comparativa_y_composicion_recomendada.md)
- [Comparativa de sistemas integrales de RRHH](../referencias_portales_aapp/comparativa_sistemas_integrales_rrhh.md)
- [Informe para el Comité de Seguridad](../comite_seguridad/informe_validacion_arquitectura_seguridad.md)

La matriz normativa detallada y la comparación ampliada de aplicaciones forman
parte de esta memoria. Las especificaciones por módulo seguirán completándola
sin convertirla en una norma jurídica. La aprobación final corresponde a los
órganos y unidades competentes.
