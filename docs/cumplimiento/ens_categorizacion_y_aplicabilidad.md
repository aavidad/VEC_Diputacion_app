# ENS: propuesta de categorización y declaración de aplicabilidad

**Borrador técnico para el Comité de Seguridad. RD 311/2022.**
Fecha: 17 de julio de 2026. No aprobado.

## 1. Alcance del sistema

Sistema de información **Portal VEC — Ventanilla Electrónica del Empleado
Público** de la Diputación de Granada, con su módulo inicial de **Gestión de
Bolsas de trabajo** (elaboración, llamamientos, contratos, reglas, consulta,
documentos, comunicaciones y auditoría), su zona pública de consulta
minimizada y su futura app de escritorio para personal técnico (clientes
equivalentes por API, DEC-053). Incluye la persistencia PostgreSQL, la
generación documental y los conectores previstos (Autofirma, Notific@, SIR).

Quedan fuera del alcance: los sistemas corporativos origen (RPT, nómina,
directorio) que actúan como fuentes autoritativas, y la plataforma de
correo/mensajería, que se integran mediante conectores con recibo.

## 2. Información tratada y servicios

- Datos identificativos de aspirantes y personal (nombre, DNI enmascarado en
  pantalla, contacto), méritos, experiencia, titulaciones y puntuaciones.
- Posibles **categorías especiales** (art. 9 RGPD): condición de discapacidad
  en turnos de reserva de bolsas. Su presencia condiciona la valoración de
  confidencialidad.
- Actos administrativos con efectos jurídicos: bases selladas, propuestas de
  llamamiento, contratos, resoluciones firmadas, notificaciones.
- Evidencias probatorias: auditoría encadenada, recibos, versiones de reglas.

## 3. Propuesta de valoración por dimensiones (Anexo I)

| Dimensión | Nivel propuesto | Justificación |
| --- | --- | --- |
| Confidencialidad (C) | **Alto** | Expedientes de RRHH con posible dato de discapacidad (art. 9); su revelación causaría perjuicio grave a los interesados y a la entidad. Si la mesa acota el dato de discapacidad a un tratamiento separado, puede razonarse Medio. `[pendiente de la mesa]` |
| Integridad (I) | **Alto** | La alteración del orden de una bolsa o de un baremo determina derechos de acceso al empleo público y vicia actos administrativos. |
| Trazabilidad (T) | **Alto** | El valor probatorio de llamamientos y resoluciones exige reconstruir quién hizo qué, cuándo y con qué versión de reglas. |
| Autenticidad (A) | **Medio** | Firma cualificada y sellado previstos; la suplantación se mitiga con identidad fuerte y atestación de decisiones. |
| Disponibilidad (D) | **Medio** | Una indisponibilidad de horas es asumible con procedimientos alternativos; los plazos administrativos dan margen. |

**Categoría global propuesta: ALTA** (por C, I y T). La decisión formal de
categorización corresponde al responsable del sistema a propuesta del Comité
de Seguridad (art. 40 y Anexo I RD 311/2022).

## 4. Declaración de aplicabilidad (extracto de medidas relevantes)

Estado de las medidas del Anexo II con mayor peso para este sistema. La
declaración completa medida a medida debe cerrarla el Comité con la
categoría aprobada.

### Marco organizativo (org)

| Medida | Estado | Evidencia / pendiente |
| --- | --- | --- |
| org.1 Política de seguridad | Pendiente | Debe aprobarla el órgano competente; insumo: este paquete |
| org.2 Normativa de seguridad | Parcial | Directrices técnicas vigentes en `docs/portal_vec/auditoria_diseno_y_seguridad_2026-07-16.md` y registro de decisiones |
| org.3 Procedimientos | Parcial | Puerta de calidad y flujo de dirección documentados; faltan procedimientos de explotación |
| org.4 Proceso de autorización | Diseñado | Autorización por caso de uso en el código; falta el circuito organizativo |

### Marco operacional (op)

| Medida | Estado | Evidencia / pendiente |
| --- | --- | --- |
| op.acc.1-2 Identificación y requisitos de acceso | Implantada en diseño | RBAC+ABAC de lista positiva cerrada, sin comodines (`internal/vec`); RLS en PostgreSQL |
| op.acc.5 Mecanismo de autenticación | **Pendiente (S-03)** | Certificado/Kerberos+mTLS diseñados (DEC-053); dependen de Sistemas; sin producción hasta su visto bueno |
| op.acc.6 Acceso local/remoto | Implantada | Allowlist CIDR sobre el socket, no falsificable desde cliente |
| op.exp.8 Registro de actividad | Parcial | Auditoría encadenada firmada probada; durabilidad en PostgreSQL pendiente (**T12**) |
| op.exp.10 Protección de los registros | Parcial | Cadena de firmas; custodia durable e inalterable pendiente (**T12**) |
| op.exp.4-6 Mantenimiento y vulnerabilidades | Implantada | CI con `govulncheck`, `-race` y puerta canónica en cada push; secret scanning y push protection en el repositorio |
| op.cont Continuidad | **Pendiente** | Copias ensayadas y prueba de recuperación (**T12**); plan de continuidad organizativo |
| op.mon Monitorización | Pendiente | Observabilidad centralizada sin desplegar |

### Medidas de protección (mp)

| Medida | Estado | Evidencia / pendiente |
| --- | --- | --- |
| mp.com.2 Protección de la confidencialidad (tránsito) | **Pendiente** | TLS productivo y proxy final dependen de Sistemas; perfil local ya cerrado a loopback |
| mp.info.3 Cifrado (reposo) | Parcial | Sobres AES-256-GCM en flujos sensibles; cifrado de columnas personales en PostgreSQL pendiente (**T16**) |
| mp.info.6 Limpieza y borrado | Pendiente | Ciclo de conservación y expurgo programado (**T14**) conforme al estudio de archivo |
| mp.si Soportes | Pendiente | Procedimiento organizativo de soportes y copias |
| mp.sw.1 Desarrollo seguro | Implantada | Hexagonal con fallo cerrado, tests 0,85/1, tope de ficheros DEC-051, revisión de dirección, datos reales nunca en desarrollo (sintéticos con test que lo vigila) |
| mp.info.9 Copias de seguridad | **Pendiente** | Ensayo de copia y recuperación (**T12**) |

## 5. Plan de adecuación propuesto

1. Aprobar categorización y política (mesa/Comité).
2. Cerrar T12 (durabilidad probatoria y copias) y T13–T16 (medidas
   programables del paquete) — encoladas al agente.
3. TLS productivo, identidad real y KMS/HSM con Sistemas (S-03).
4. Declaración de aplicabilidad completa, auditoría de conformidad y
   publicación de la declaración de conformidad.
