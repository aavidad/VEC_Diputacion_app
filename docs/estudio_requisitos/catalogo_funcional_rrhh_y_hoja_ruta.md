# Catálogo funcional y hoja de ruta del portal integral de RRHH

Fecha de corte: **16 de julio de 2026**.

Estado: catálogo de alcance para análisis y planificación. No es una autorización
para tratar datos reales ni para activar reglas jurídicas no validadas.

## 1. Finalidad

Este catálogo convierte el análisis integral en capacidades identificables y
ordenadas. Permite presentar el producto, dividir el trabajo entre equipos y
evitar que «portal completo de RRHH» se interprete como un bloque monolítico.

Las fases indican orden de entrega, no importancia jurídica:

- **F0 — fundamento:** contratos y controles que condicionan todo lo demás;
- **F1 — Núcleo y Bolsa inicial:** primer producto utilizable y demostrable;
- **F2 — Selección y Bolsa completa:** operación integral del proceso;
- **F3 — Personal, organización y RPT:** expediente interno y planificación;
- **F4 — Portal del empleado y tiempo:** autoservicio, formación y Cronos;
- **F5 — gestión económica y materias reservadas:** nómina, acción social, PRL,
  igualdad y relaciones laborales;
- **FT — transversal:** capacidad que evoluciona en todas las fases.

Una capacidad solo pasa a producción si supera sus puertas jurídica,
funcional, seguridad, datos, accesibilidad, operación y prueba.

## 2. Núcleo compartido

| Id. | Capacidad | Fase | Resultado exigible |
| --- | --- | --- | --- |
| NUC-001 | Persona canónica | F0 | Identificador opaco común, coincidencia asistida, procedencia por atributo, fusiones y separaciones trazadas; el DNI no será clave técnica ni campo de búsqueda general |
| NUC-002 | Identidades y cuentas | F0 | Varias identidades por persona, proveedores sustituibles, sesiones con perfil único y sin acumulación silenciosa de privilegios |
| NUC-003 | Representación | F1 | Poder, tutoría o representación con alcance, vigencia, evidencia y revocación; cada actuación identifica representado y representante |
| NUC-004 | Organismos, unidades y centros | F0 | Organización histórica, códigos opacos, jerarquías versionadas y relaciones de dirección o dependencia con fechas |
| NUC-005 | Relaciones jurídicas | F0 | Nombramiento, contrato, vínculo, régimen, causa, situación y periodo como hechos separados y bitemporales |
| NUC-006 | Registro de autoridades y políticas | F0 | Fuente, artículo, documento, huella, ámbito, vigencia, efectos, órgano, firmas, parámetros tipados y casos de prueba |
| NUC-007 | Catálogos gobernados | F0 | Valores configurables con código estable, versión, traducción, procedencia, vigencia y sustitución; nunca listas legales en el código |
| NUC-008 | Formularios gobernados | F1 | Esquemas tipados y versionados, validación en servidor, accesibilidad, borrador y migración explícita; sin scripts arbitrarios |
| NUC-009 | Flujo de expedientes | F1 | Estados, tareas, plazos, responsables, suplencias, bloqueos, doble control, rectificaciones y cierre reproducibles |
| NUC-010 | Documento lógico | F0 | Original, representaciones, metadatos, huellas, procedencia, clasificación y relaciones sin duplicar el contenido |
| NUC-011 | Almacenamiento seguro | F0 | Puerto sustituible, cifrado, cuarentena, análisis antimalware, versionado, retención, copia y restauración verificadas |
| NUC-012 | Firma, sello, CSV y validación | F1 | Firma humana y sello diferenciados, cofirmas, tiempo fiable, validación a largo plazo, QR/CSV de cotejo y conector de AutoFirma |
| NUC-013 | Generación documental | F1 | Modelo canónico y adaptadores PDF/PDF/A, ODT, DOCX, texto, CSV, JSON u otros; indicar original jurídico y representación accesible |
| NUC-014 | Registro administrativo | F1 | Puerto al registro/SIR-SICRES, asiento, recibo, anexos, conciliación e idempotencia; no simular registro con una tabla propia |
| NUC-015 | Notificación y avisos | F1 | Notificación administrativa trazable y avisos separados por correo, Telegram u otros conectores sin datos sensibles |
| NUC-016 | Autorización positiva | F0 | Capacidades por rol y atributos de entidad, unidad, relación, expediente, finalidad, estado, tiempo y delegación; denegar lo no concedido |
| NUC-017 | Segregación de funciones | F0 | Incompatibilidades y doble intervención en configuración, baremación, firma, pago, nómina, seguridad y expedientes reservados |
| NUC-018 | Auditoría probatoria | F0 | Actor, sujeto, finalidad, acción, expediente, regla, antes/después, tiempo fiable y resultado en registro resistente a manipulación |
| NUC-019 | Registros técnicos y seguridad | F0 | Separados de la trazabilidad jurídica, minimizados, correlacionables y enviados al sistema corporativo de supervisión |
| NUC-020 | Calendarios | F1 | Día natural, hábil, festivo, apertura y jornada separados, con fuentes y versiones históricas |
| NUC-021 | Comunicaciones | F1 | Preferencias, suscripciones, plantillas, canales, entrega y bajas, gobernados por finalidad y sin convertir avisos en actos |
| NUC-022 | Búsqueda | F1 | Índices por proyección y permiso, prevención de enumeración, filtros auditados y resultados minimizados |
| NUC-023 | Exportación e informes | FT | Vistas aprobadas, marca de procedencia, finalidad, límites, cifrado, caducidad y auditoría; no SQL libre ni acceso a todas las columnas |
| NUC-024 | Ayuda y accesibilidad | FT | Una fuente versionada genera web, manual, audio y ayuda contextual; WCAG/EN 301 549, lectura fácil y asistente limitado a su corpus |
| NUC-025 | API, CLI, MCP y escritorio | FT | Adaptadores de los mismos casos de uso y permisos; MCP o IA nunca eluden autorización, validación o registro |
| NUC-026 | Eventos y bandeja de salida | F0 | Contratos versionados, idempotentes, con correlación, instante efectivo, procedencia, integridad y reintento durable |
| NUC-027 | Importación y reconciliación | F1 | Preparación, simulación, incidencias, aprobación y reversión lógica; nunca escritura directa en tablas de otro sistema |
| NUC-028 | Conservación y archivo | FT | Serie, expediente, índice, transferencia, bloqueo, valoración y expurgo autorizados; almacenamiento y archivo no son sinónimos |

## 3. Procesos selectivos y Bolsa

| Id. | Capacidad | Fase | Resultado exigible |
| --- | --- | --- | --- |
| SEL-001 | Plan y OEP | F3 | Relación trazable entre necesidad, crédito, plaza, OEP, publicación, convocatoria y grado de ejecución |
| SEL-002 | Convocatoria gobernada | F1 | Bases, anexos, plazas, requisitos, fases, calendario, tasa, baremo, tribunal y circuito como versión firmada e inmutable |
| SEL-003 | Publicidad | F1 | Página pública accesible con versiones, plazos, documentos, avisos y fuente oficial; retirada sin borrar historia |
| SEL-004 | Solicitud | F1 | Borrador recuperable, representación, requisitos, anexos, firma, registro, justificante y modificación dentro de plazo |
| SEL-005 | Tasas | F1 | Cálculo, exención/bonificación, pasarela intercambiable, retorno idempotente, conciliación y devolución |
| SEL-006 | Admisión | F2 | Comprobación de requisitos, obtención de oficio, exclusión motivada, subsanación y listas provisional/definitiva |
| SEL-007 | Tribunal | F2 | Composición, recusación, sustitución, acceso aislado, sesiones, actas y firmas; sin acceso general a RRHH |
| SEL-008 | Pruebas | F2 | Sedes, llamamientos, anonimato cuando proceda, calificaciones, mínimos, incidencias, actas y publicaciones |
| SEL-009 | Alegaciones y recursos | F2 | Presentación, traslado, informe, resolución, notificación y efecto sobre resultados sin borrar el acto anterior |
| RUM-001 | Registro Único de Méritos | F1 | Experiencia, titulaciones, cursos y otras evidencias reutilizables con procedencia, vigencia y estados explícitos |
| RUM-002 | Obtención de oficio | F1 | Consulta autorizada a Personal y fuentes interoperables, recibo de evidencia y alternativa humana si la respuesta no es concluyente |
| RUM-003 | Validez documental | F1 | Autenticidad/integridad separada del reconocimiento del mérito y de su aplicabilidad en una convocatoria |
| BAR-001 | Editor de baremo | F1 | Secciones y reglas tipadas para periodos, jornada, experiencia, títulos, cursos, topes, solapes, equivalencias y desempates |
| BAR-002 | Simulación | F1 | Casos de prueba, muestras anonimizadas, comparación entre versiones y bloqueo ante resultados no determinables |
| BAR-003 | Autobaremación | F1 | Cálculo orientativo explicable desde méritos e instantánea de bases; la persona no escribe puntos ni se autovalida |
| BAR-004 | Revisión técnica | F1 | Decisión por mérito o unidad, aceptación/rechazo parcial, motivo, evidencia, revisor, doble control por riesgo y firma |
| BAR-005 | Rectificación | F1 | Revocación, rehabilitación o corrección mediante nuevo acto firmado, fecha de efectos y recálculo relacionado |
| BAR-006 | Resultado | F1 | Puntuación declarada, calculada, validada y definitiva, con desglose exacto, topes, orden y huella reproducible |
| BOL-001 | Constitución | F1 | Bolsa ligada al proceso/origen, categoría, ámbito, versión normativa, vigencia, lista y criterios de orden |
| BOL-002 | Preferencias | F2 | Centro, territorio, duración, jornada y clase de oferta según configuración permitida y fechas efectivas |
| BOL-003 | Disponibilidad | F1 | Estado voluntario o reglado, ámbito global o específico, causa, periodo, documentos y decisión |
| BOL-004 | Restricción tras cese | F1 | Política provincial versionada sobre hechos oficiales de cese; cálculo global por persona y excepciones motivadas |
| BOL-005 | Llamamiento | F1 | Propuesta reproducible, orden, exclusiones, intentos, canales, plazos, respuestas, renuncias y saltos firmados |
| BOL-006 | Posición privada | F1 | Consulta propia de orden, estado, última actuación y siguiente paso sin revelar datos de terceros |
| BOL-007 | Publicaciones | F1 | Listas y resultados minimizados, aprobados y versionados; preferencia por consulta individual cuando sea suficiente |
| BOL-008 | Incorporación | F2 | Propuesta, documentación, aptitud mínima, nombramiento/contrato, alta en Personal y disponibilidad resultante |
| BOL-009 | Indicadores | F2 | Cobertura por categoría, agotamiento, tiempos, intentos, renuncias y necesidades, con definiciones y fuentes |

## 4. Personal, organización, RPT y planificación

| Id. | Capacidad | Fase | Resultado exigible |
| --- | --- | --- | --- |
| PER-001 | Registro de Personal | F3 | Expediente canónico de relaciones, actos, antigüedad, trienios, situaciones, grados, ocupaciones y procedencia |
| PER-002 | Alta y toma de posesión | F3 | Lista de comprobación, acto firmado, registro, identidad corporativa, ocupación, nómina y comunicaciones reconciliadas |
| PER-003 | Contrato/nombramiento y cese | F3 | Causa, periodo, documentos, competencia, efectos, reserva, liquidaciones y eventos para Bolsa/Nómina |
| PER-004 | Situaciones | F3 | Catálogo por régimen y periodo, acto, efectos sobre ocupación, reserva, servicios, jornada, nómina y derechos |
| PER-005 | Rectificación de datos | F3 | Dato oficial de solo lectura, propuesta, evidencia, revisión, decisión y sustitución histórica |
| ORG-001 | Estructura organizativa | F3 | Organismos, áreas, servicios, centros, unidades y jerarquías históricas, incluidas delegaciones y suplencias |
| RPT-001 | Categorías y especialidades | F3 | Cuerpo, escala, subescala, clase, especialidad y categoría laboral separados, con equivalencias históricas |
| RPT-002 | Plantilla y plazas | F3 | Dotaciones presupuestarias por ejercicio, creación, amortización, reserva y vínculo con OEP |
| RPT-003 | Puestos | F3 | Puesto tipo e individual, código opaco, unidad, requisitos, funciones, provisión, nivel, complementos y vigencia |
| RPT-004 | Versiones de RPT | F3 | Importación, acuerdo, BOP, diferencias, incidencias, aprobación y consulta bitemporal sin sobrescritura |
| RPT-005 | Ocupación y reserva | F3 | Persona-relación-plaza-puesto con clase, titularidad, fechas, acto y reserva, sin deducirla de nómina |
| RPT-006 | Vacantes históricas | F3 | Proyecciones separadas de dotación vacante, puesto sin ocupante y necesidad cubrible, con cobertura e incertidumbres |
| PLA-001 | Planificación | F3 | Necesidades, escenarios, coste, jubilaciones previsibles, vacantes, bolsas disponibles y crédito, sin decisiones automáticas |
| PLA-002 | Capítulo I | F5 | Escenarios versionados y conciliados con presupuesto y nómina; estimación claramente separada del acto aprobado |
| CER-001 | Certificados | F2 | Plantillas versionadas, datos de oficio, revisión, firma/sello, registro, CSV, entrega y conservación |

## 5. Provisión, carrera y formación

| Id. | Capacidad | Fase | Resultado exigible |
| --- | --- | --- | --- |
| PRO-001 | Catálogo de procedimientos | F3 | Concurso, libre designación, adscripción provisional, comisión y movilidad con fuente y requisitos propios |
| PRO-002 | Concurso y baremo | F3 | Convocatoria, puestos, requisitos, méritos, preferencias, valoración, propuesta, alegaciones y adjudicación reproducibles |
| PRO-003 | Movilidad por salud | F3 | Conclusión funcional de PRL, puestos compatibles, prioridad, revisión y resolución sin datos clínicos |
| CAR-001 | Grado funcionarial | F3 | Nivel de puesto, grado personal, periodos computables, vía de reconocimiento, límites, acto e inscripción separados |
| CAR-002 | Progresión laboral | F3 | Grupo/categoría y progresión según convenio consolidado, convocatoria, curso/prueba y resolución |
| CAR-003 | Promoción interna | F3 | Elegibilidad, antigüedad, grupo/subgrupo, convocatoria, pruebas y toma de posesión, sin confundirse con carrera |
| CAR-004 | Desempeño | F5 | Sistema negociado, objetivo y transparente; datos, revisores, alegaciones y efectos expresos, sin vigilancia algorítmica oculta |
| FOR-001 | Plan de formación | F4 | Necesidades, presupuesto, acciones, ediciones, plazas, prioridad y publicación |
| FOR-002 | Solicitud y selección | F4 | Solicitud, autorización, criterios, sustitución, lista y notificación |
| FOR-003 | Ejecución y certificado | F4 | Asistencia, aprovechamiento, evaluación, coste, certificado firmado e incorporación automática a méritos cuando proceda |

## 6. Portal del empleado, Cronos y dietas

| Id. | Capacidad | Fase | Resultado exigible |
| --- | --- | --- | --- |
| EMP-001 | Carpeta personal | F3 | Vista propia de datos oficiales, actos, puesto, servicios, documentos, solicitudes y procedencia |
| EMP-002 | Bandeja y solicitudes | F3 | Formularios, borradores, estado, plazo, tarea, justificante, subsanación, decisión y notificación |
| EMP-003 | Espacio del responsable | F4 | Cobertura y tareas de su unidad con delegación temporal; nunca expediente completo, diagnóstico, nómina o méritos de Bolsa |
| CRO-001 | Perfil de jornada | F4 | Calendario, colectivo, centro, ciclo, horario, flexibilidad, reducciones y vigencia desde políticas aprobadas |
| CRO-002 | Cuadrantes y turnos | F4 | Planificación, publicación, cambios, cobertura, descanso, trazabilidad y reglas específicas por centro |
| CRO-003 | Marcajes | F4 | Hecho original inmutable, dispositivo, origen, calidad, corrección compensatoria y auditoría |
| CRO-004 | Incidencias | F4 | Detección explicable, alegación/corrección, aprobación y cierre; no convertir anomalía en sanción automática |
| CRO-005 | Permisos y vacaciones | F4 | Catálogo por régimen, hecho causante, saldo, justificación mínima, necesidades del servicio, resolución y recurso |
| CRO-006 | Teletrabajo | F4 | Solicitud, tareas, días, objetivos, medios, revisión, reversibilidad y revocación conforme a instrucciones vigentes |
| CRO-007 | Horas y compensaciones | F4 | Autorización, tiempo real, política, saldo, disfrute o liquidación, con contrato trazable hacia Nómina |
| CRO-008 | Geolocalización de vehículos | F4 | Finalidad, horario, vehículo, ruta, acceso, retención y EIPD; enclave Mulhacén sin salida al portal público |
| DIE-001 | Comisión de servicio | F4 | Orden, persona, objeto, ruta, fechas, medio, anticipo, autorizadores y documentos |
| DIE-002 | Itinerario y kilometraje | F4 | Fuente cartográfica/ruta versionada, excepciones motivadas y decimal exacto; GPS no acredita por sí solo el derecho |
| DIE-003 | Liquidación | F4 | Tarifas por vigencia, justificantes, cálculo, fiscalización, pago y rectificación; sin `float64` para importes |

## 7. Nómina y gestión económica

| Id. | Capacidad | Fase | Resultado exigible |
| --- | --- | --- | --- |
| NOM-001 | Conceptos y tablas | F5 | Definiciones, fórmulas tipadas, fuentes, colectivos, vigencia, unidades, redondeo y casos de prueba |
| NOM-002 | Devengo | F5 | Relación, puesto, jornada, incidencias, retroactividad y prorratas con procedencia |
| NOM-003 | Seguridad Social | F5 | Altas/bajas, bases, tramos, tipos, liquidaciones, ficheros, respuestas y conciliación por versión normativa |
| NOM-004 | IRPF | F5 | Declaraciones, algoritmo/tablas, regularizaciones y certificados en compartimento fiscal |
| NOM-005 | Cálculo y cierre | F5 | Pre-nómina, diferencias, doble revisión, cierre inmutable, firma y reapertura solo mediante rectificación formal |
| NOM-006 | Fiscalización y contabilidad | F5 | Lotes, documentos, reparos, aprobación, asientos, pagos y conciliación con Intervención/Tesorería |
| NOM-007 | Recibo y certificados | F5 | Acceso propio, representación accesible, firma/sello, CSV, conservación y revelación mínima |
| NOM-008 | Migración | F5 | Datos conciliados y varios ciclos paralelos con tolerancias aprobadas antes de retirar el sistema vigente |

## 8. Acción social, PRL, igualdad y relaciones laborales

| Id. | Capacidad | Fase | Resultado exigible |
| --- | --- | --- | --- |
| ASO-001 | Convocatoria de ayuda | F5 | Modalidad, colectivo, requisito, presupuesto, plazo, evidencia, comisión, cuantía y política versionados |
| ASO-002 | Solicitud y unidad familiar | F5 | Datos familiares mínimos, oposición/consulta, documentación, subsanación y compartimento propio |
| ASO-003 | Presupuesto y resolución | F5 | Reserva, prelación, propuesta de comisión, informe, resolución, nómina/pago, reintegro y cierre |
| PRL-001 | Riesgos y medidas | F5 | Puesto/centro, evaluación, medida, responsable, revisión, formación y evidencia |
| PRL-002 | Vigilancia de la salud | F5 | Bóveda clínica y profesionales propios; hacia Personal solo aptitud o restricción funcional mínima |
| PRL-003 | Accidente e investigación | F5 | Parte, asistencia, investigación, medidas, comunicaciones externas y acceso restringido |
| IGU-001 | Plan e indicadores | F5 | Diagnóstico, negociación, objetivos, medidas, responsables, calendario, indicadores y seguimiento |
| IGU-002 | Impacto en selección | F2 | Informe, composición equilibrada o justificación, lenguaje inclusivo, adaptaciones y revisión de bases |
| RES-001 | Protocolos reservados | F5 | Denuncia, protección, instrucción, prueba, medidas y resolución en expediente aislado y con barrera frente a represalias |
| RES-002 | Sistema interno de información | F5 | Enlace o derivación mínima al canal institucional existente; no crear un segundo canal ni copiar su expediente al portal |
| RLA-001 | Negociación | F5 | Mesa, representación, propuestas, actas, acuerdos, publicación, vigencia y materialización en políticas |
| RLA-002 | Crédito y actividad sindical | F4 | Representante, bolsa/saldo, solicitud y justificación mínima; afiliación separada del expediente ordinario |
| RLA-003 | Incompatibilidades | F5 | Declaración, actividad, jornada, retribución, informes, resolución y proyección pública legalmente exigible |
| RLA-004 | Disciplina | F5 | Hechos, tipificación, prescripción, medidas, instructor, audiencia, prueba, resolución y recurso con segregación |

## 9. Analítica, transparencia y control

| Id. | Capacidad | Fase | Resultado exigible |
| --- | --- | --- | --- |
| ANA-001 | Indicadores gobernados | FT | Definición, fórmula, fuente, periodo, responsable, calidad y versión; no métricas improvisadas sobre producción |
| ANA-002 | Cuadros de mando | FT | Modelo de lectura separado, datos agregados/seudonimizados y filtrado por ámbito |
| ANA-003 | Transparencia | FT | Proyección aprobada, minimizada, accesible, versionada y con política de retirada/conservación |
| ANA-004 | Informes regulatorios | F3 | Plantillas y adaptadores para ISPA u otras obligaciones, con revisión y firma |
| ANA-005 | Acceso y exportación | FT | Registro nominal de consulta/descarga, finalidad, volumen, campos, resultado y alertas de uso anómalo |

## 10. Flujos que cruzan módulos

### 10.1 De aspirante a empleado

```text
persona aspirante
→ solicitud registrada
→ resultado y bolsa
→ llamamiento
→ propuesta de nombramiento o contrato
→ comprobaciones finales
→ acto firmado
→ nueva relación en Registro de Personal
→ ocupación
→ identidad corporativa, Cronos y Nómina mediante eventos conciliados
```

La persona no se duplica. Sí nace una relación jurídica nueva. Bolsa no escribe
directamente en Personal, AD, Cronos o Nómina.

### 10.2 Servicio prestado reutilizado como mérito

```text
nombramiento/contrato y cese en Personal
→ periodo oficial de servicios
→ proyección minimizada al Registro Único de Méritos
→ instantánea de convocatoria
→ aplicación de sus bases
→ revisión y resultado
```

Nómina puede conciliar, pero no será la única fuente de servicios. Una falta de
respuesta no se convierte en cero meses.

### 10.3 Permiso con efecto económico

```text
solicitud
→ responsable por necesidad del servicio
→ RRHH cuando proceda
→ resolución
→ incidencia aprobada de jornada
→ concepto o ausencia para Nómina
→ conciliación
→ recibo y auditoría
```

Solo se intercambia el efecto necesario, no el diagnóstico o documento que
justifica el permiso.

## 11. Orden de implantación recomendado

### F0 — cerrar el fundamento

1. Modelo canónico de persona, cuenta, relación, organización y autoridad.
2. Política versionada, aritmética exacta y tiempo civil.
3. autorización positiva, segregación, auditoría y eventos.
4. contrato documental, almacenamiento, firma y archivo lógico.
5. separación efectiva de portales externo e interno.

### F1 — primera entrega útil de Núcleo y Bolsa

1. convocatoria y bases gobernadas;
2. portal público y área del aspirante;
3. solicitud, tasa, firma, registro y justificante;
4. méritos reutilizables y datos de oficio;
5. editor/simulador de baremo, autobaremo y revisión firmada;
6. listas, bolsa, disponibilidad, carencia y llamamiento;
7. documentos, certificados básicos, avisos y consulta propia;
8. datos ficticios y entorno de demostración sin atajos desechables.

### F2 — completar Selección y Bolsa

Órgano de selección, pruebas, admisión, subsanación, alegaciones, recursos,
incorporación, indicadores e integraciones corporativas reales.

### F3 — construir la columna interna

Registro de Personal, RPT/plantilla histórica, ocupaciones, vacantes,
planificación, provisión, carrera, certificados y portal propio del empleado.

### F4 — tiempo y operación del empleado

Formación, espacio de responsables, Cronos, centros/turnos, teletrabajo, horas,
dietas y geolocalización, después de textos consolidados, negociación y EIPD.

### F5 — materias económicas y reservadas

Nómina, acción social, PRL, igualdad, protocolos, incompatibilidades, disciplina
y relaciones laborales, con equipos y compartimentos especializados.

## 12. Puertas de entrada y salida de cada capacidad

Antes de iniciar código funcional:

- fuente provincial y normativa identificadas;
- vigencia, ámbito y modificaciones comprobados;
- propietario del dato y órgano competente designados;
- procedimiento real y excepciones levantados;
- roles, segregación, conservación y riesgos definidos;
- casos de aceptación públicos o anonimizados aprobados.

Para considerar una capacidad terminada:

- dominio y puertos no dependen del adaptador elegido;
- autorización positiva probada, incluidas denegaciones y separación de zonas;
- reglas, fuentes y versiones aparecen en resultados y auditoría;
- concurrencia, reintento, idempotencia y fallo parcial están probados;
- interfaz y documentos son accesibles;
- manual funcional, técnico, operación, seguridad y ayuda se actualizaron;
- migración, copia, restauración, observabilidad y reversión se ensayaron;
- responsable funcional, seguridad y órgano jurídico competente aceptaron su
  parte de la evidencia.

## 13. Alcance inmediato y protección frente a refactorizaciones

Ahora se desarrollan Núcleo y Bolsa. El resto no debe simularse con reglas
arbitrarias para aparentar cobertura. Se conservarán los prototipos de Personal,
Cronos y Dietas solo como material de descubrimiento hasta su migración.

La protección frente a una futura refactorización no consiste en programar
anticipadamente todos los módulos. Consiste en fijar límites, identificadores,
hechos, versiones, autoridad, tiempo y contratos neutrales. Con esas piezas,
los módulos posteriores se conectarán de forma aditiva y el núcleo no tendrá
que conocer sus coeficientes, tablas, pantallas o proveedores.
