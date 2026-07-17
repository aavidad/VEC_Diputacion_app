# Evaluación de impacto en protección de datos (EIPD) — módulo de Bolsas

**Borrador para consulta al Delegado de Protección de Datos. Art. 35 RGPD.**
Fecha: 17 de julio de 2026. No aprobado. Metodología: guía de la AEPD.

## 1. Necesidad de la evaluación

La EIPD procede porque el tratamiento reúne varios criterios del art. 35.3
y de las listas de la AEPD:

- **evaluación sistemática** de aspectos personales (baremación de méritos)
  que produce efectos jurídicos (orden de acceso al empleo público);
- **gran escala** a nivel provincial (cientos de integrantes por bolsa);
- posible tratamiento de **categorías especiales** (discapacidad en turnos
  de reserva);
- colectivo en situación de asimetría (aspirantes frente a administración).

## 2. Descripción sistemática del tratamiento

Ciclo: inscripción → baremación conforme a bases selladas → listados →
llamamiento (propuesta calculada en servidor → revisión humana → acto
firmado → comunicación fehaciente) → contrato/cese → archivo.

Flujos técnicos relevantes, tal y como están construidos:

- Las **bases y reglas** de cada bolsa se publican como versiones inmutables
  con huella criptográfica; todo cálculo cita su versión exacta.
- La **propuesta de llamamiento** se calcula íntegramente en servidor y se
  presenta a la persona técnica **sin identidades** (secuencia, resultado,
  puntuación, regla y fundamento); la relación con las personas queda
  custodiada en servidor bajo autorización.
- La **zona pública** solo sirve datos minimizados; los listados usan
  identificación reducida conforme a la disposición adicional 7ª LOPDGDD.
- Los accesos internos se autorizan por caso de uso (lista positiva, RLS) y
  quedan trazados con finalidad.

## 3. Decisiones automatizadas (art. 22 RGPD)

El sistema **no adopta decisiones basadas únicamente en tratamiento
automatizado**: la propuesta del motor es un insumo; la selección la revisa
y la firma una persona con competencia, y el acto resultante es recurrible
por las vías administrativas ordinarias. Este diseño debe preservarse: la
prohibición de "aceptar propuestas sin revisión" es un requisito, no una
opción de interfaz.

## 4. Necesidad y proporcionalidad

- **Idoneidad**: la gestión electrónica con reglas selladas reduce el error
  y la arbitrariedad frente al procedimiento manual.
- **Minimización**: verificada por diseño (identidades fuera de las
  pantallas de revisión; público minimizado; demo sintética separada por
  doble llave con test que impide su fuga al modo normal).
- **Limitación de finalidad**: cada acceso declara finalidad y queda
  registrado (T13); los catálogos impiden usos fuera de los previstos.
- **Exactitud**: fuentes autoritativas declaradas y versionadas; cálculo
  con aritmética exacta (sin flotantes) y redondeos configurados por bases.

## 5. Análisis de riesgos para derechos y libertades

| Riesgo | Mitigación implantada | Residual y plan |
| --- | --- | --- |
| Acceso indebido a expedientes por personal interno | RBAC+ABAC lista positiva, RLS, finalidad declarada, auditoría encadenada | Medio-bajo; baja a bajo con registro de accesos durable (T12/T13) |
| Exfiltración del censo de una bolsa | Las consultas globales no descargan censos (contrato del panel lo rechaza); propuestas sin identidad | Bajo |
| Alteración del orden o del baremo | Versiones selladas con huella, atestación de decisiones, cadena firmada | Bajo; auditoría durable pendiente (T12) |
| Suplantación de identidad interna | Diseñada identidad fuerte (certificado/Kerberos+mTLS); allowlist de red | **Medio mientras no se componga (S-03)**: sin producción hasta el visto bueno de Sistemas |
| Pérdida de trazabilidad probatoria | Cadena firmada en memoria | **Medio**: T12 la lleva a PostgreSQL con copias ensayadas |
| Reidentificación en demostraciones | Datos 100 % sintéticos, doble llave servidor+URL, test anti-fuga | Bajo |
| Brecha de confidencialidad en tránsito | Perfil actual solo loopback/red interna | TLS productivo con Sistemas antes de piloto |
| Conservación excesiva | Estudio de archivo elaborado | Expurgo programado (T14) y plazos a fijar por la mesa |

## 6. Medidas previstas y plan

Programables (encoladas al agente): T12 durabilidad probatoria y copias
ensayadas; T13 registro de accesos con finalidad; T14 conservación y
expurgo; T15 atención de derechos (acceso, rectificación, supresión,
oposición, limitación) con evidencia; T16 cifrado en reposo de columnas
personales con gestión de claves.

Organizativas (mesas): designación de roles ENS, política de seguridad,
plazos de conservación, procedimiento de brechas (notificación AEPD 72 h,
CCN-CERT), formación del personal usuario.

## 7. Conclusión provisional

Con las medidas implantadas, el tratamiento es **viable en demostración con
datos sintéticos** (situación actual). Para un **piloto con datos reales**
se consideran condiciones previas: T12, T13 y TLS productivo, más la
validación de este documento por el DPD. Para **producción**: identidad
real (S-03), medidas T14–T16, categorización ENS aprobada y esta EIPD
formalizada con el dictamen del DPD.

`[Dictamen del DPD: pendiente]` · `[Aprobación del responsable: pendiente]`
