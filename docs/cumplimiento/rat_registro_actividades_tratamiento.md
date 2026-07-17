# Registro de actividades de tratamiento (RAT) — módulo de Bolsas

**Borrador para el Delegado de Protección de Datos. Art. 30 RGPD y art. 31
LOPDGDD.** Fecha: 17 de julio de 2026. No aprobado.

Responsable del tratamiento: **Diputación Provincial de Granada**
`[órgano concreto: pendiente de la mesa]`.
Delegado de Protección de Datos: `[designación vigente de la Diputación]`.
No se prevén transferencias internacionales en ninguna actividad.

---

## A1 — Gestión de bolsas de empleo temporal (aspirantes)

| Campo | Contenido |
| --- | --- |
| Fines | Constitución y gestión de bolsas: inscripción, baremación de méritos, orden de prelación, alegaciones y publicación de listados |
| Base jurídica | Art. 6.1.c y 6.1.e RGPD (TREBEP y normativa de selección); art. 9.2.b para la condición de discapacidad en turnos de reserva |
| Interesados | Personas aspirantes a las bolsas |
| Categorías de datos | Identificativos (nombre, DNI, contacto), académicos y profesionales (titulaciones, experiencia, méritos), puntuaciones; condición de discapacidad solo para el turno de reserva |
| Destinatarios | Órganos de selección y RRHH; publicación de listados con datos minimizados conforme a la normativa de transparencia (LO 3/2018, disp. ad. 7ª: identificación por iniciales/dígitos parciales) |
| Plazos de supresión | Propuesta: vigencia de la bolsa más los plazos de recurso y las tablas de valoración documental aplicables `[pendiente de la mesa]`; expurgo programado (T14) |
| Medidas | Ver declaración ENS: minimización en pantalla (DNI enmascarado), RBAC+ABAC con RLS, auditoría encadenada, cifrado de flujos sensibles |

## A2 — Llamamientos y contratación temporal

| Campo | Contenido |
| --- | --- |
| Fines | Cobertura de necesidades: propuesta calculada de personas según orden y reglas, comunicación fehaciente, aceptación/renuncia, contrato, cese y reincorporación |
| Base jurídica | Art. 6.1.c y 6.1.e RGPD; art. 6.1.b en la fase contractual |
| Interesados | Integrantes de bolsas llamados; personal temporal contratado |
| Categorías de datos | Identificativos y de contacto, posición y puntuación, disponibilidad y sus causas, condiciones del contrato |
| Destinatarios | RRHH, unidad de destino, Seguridad Social y AEAT (obligación legal), intervención |
| Plazos | Los del expediente de personal y sus tablas de valoración `[pendiente de la mesa]` |
| Medidas | La interfaz de revisión no recibe identidades (propuesta por secuencia y puntuación); decisión final humana y firmada; trazabilidad completa del acto |

## A3 — Portal del empleado (expediente propio)

| Campo | Contenido |
| --- | --- |
| Fines | Acceso de cada empleado/a a su propio expediente, certificados y comunicaciones cuando se habiliten los módulos |
| Base jurídica | Art. 6.1.c y 6.1.e RGPD |
| Interesados | Personal de la Diputación |
| Categorías de datos | Expediente de personal: datos identificativos, administrativos, retributivos cuando proceda |
| Destinatarios | La propia persona interesada; ningún expediente de terceros se proyecta (frontera de privacidad verificada en el diseño del workspace) |
| Plazos | Los del expediente de personal |
| Medidas | Proyección resuelta en servidor por persona, ámbito y finalidad; endpoint cerrado (503) hasta componerse con identidad real |

## A4 — Seguridad, trazabilidad y auditoría

| Campo | Contenido |
| --- | --- |
| Fines | Registro probatorio de actos y accesos; detección de uso indebido; cumplimiento ENS |
| Base jurídica | Art. 6.1.c y 6.1.e RGPD (obligaciones ENS); interés público en la integridad de los procesos selectivos |
| Interesados | Personal usuario del portal; aspirantes cuyos expedientes se consultan |
| Categorías de datos | Identidad del actor, rol, acto realizado, expediente afectado, finalidad declarada, momento, versión de reglas aplicada |
| Destinatarios | Órganos de control interno; jueces y tribunales cuando proceda |
| Plazos | Propuesta: los de prescripción de las responsabilidades que documentan `[pendiente de la mesa]` |
| Medidas | Cadena de auditoría firmada; registro de accesos con finalidad (T13); custodia durable (T12) |

## A5 — Consulta pública de convocatorias

| Campo | Contenido |
| --- | --- |
| Fines | Publicidad de convocatorias y estado de bolsas a la ciudadanía |
| Base jurídica | Art. 6.1.e RGPD; obligaciones de publicidad activa |
| Datos | **Sin datos personales**: la superficie pública sirve datos minimizados y agregados; verificado en `internal/modules/bolsa/adapters/httppublico` |
| Medidas | Fuente sintética por defecto; salida minimizada en servidor |

---

**Nota de exactitud**: las actividades A1–A3 describen capacidades del
sistema; su alta efectiva en el RAT corporativo debe producirse cuando cada
módulo entre en servicio con datos reales, momento que hoy no se ha
producido (no hay producción ni datos reales cargados).
