# Cumplimiento: tareas organizativas para el DPD

Las reglas técnicas de seguridad y protección de datos están ahora en el TRD:
[§5 Identidad y sesión](../producto/TRD.md#5-identidad-y-sesión),
[§6 Autorización](../producto/TRD.md#6-autorización),
[§7 Efectos duraderos](../producto/TRD.md#7-efectos-duraderos-auditoría-outbox-idempotencia-y-versión),
[§8 Protección de datos](../producto/TRD.md#8-protección-de-datos),
[§9 Documentos, firma y expediente electrónico](../producto/TRD.md#9-documentos-firma-y-expediente-electrónico),
[§11 Seguridad](../producto/TRD.md#11-seguridad) y
[§17 Lo que falta para producción](../producto/TRD.md#17-lo-que-falta-para-producción).
Las reglas de producto están en
[PRD §11, «Protección de datos»](../producto/PRD.md#protección-de-datos) y la
accesibilidad en [DISENO_UI_UX §8](../producto/DISENO_UI_UX.md#8-accesibilidad).

Aquí queda solo lo que no es código: lo que tienen que preparar o aprobar el
DPD (delegado de protección de datos), el responsable de seguridad y RRHH antes
de usar datos reales. La versión anterior, con el contexto del prototipo, está
en la historia de Git de este fichero.

## Normas que se aplican

| Norma | Qué exige al portal |
| --- | --- |
| RGPD (Reglamento UE 2016/679) y LOPDGDD (Ley Orgánica 3/2018) | Licitud, minimización, información, derechos, seguridad, privacidad desde el diseño, encargados y brechas |
| ENS (Real Decreto 311/2022) | Política de seguridad, categorización, análisis de riesgos, medidas, auditoría y declaración o certificación de conformidad |
| Ley 39/2015 | Derechos en el procedimiento, identificación y firma, registro, plazos, notificaciones, expediente y recursos |
| Real Decreto 203/2021 | Sede, identificación, firma, registro, copias, archivo, notificación y expediente electrónicos |
| ENI (Real Decreto 4/2010) y sus normas técnicas | Metadatos, formatos, firma, copia auténtica, expediente, intercambio y conservación |
| Real Decreto 1112/2018 (accesibilidad) | Declaración de accesibilidad y canal de reclamaciones |

## Protección de datos

- Inventariar los tratamientos (bolsa, baremación, comunicaciones,
  alegaciones, archivo y auditoría) y mantener el registro de actividades al
  día para cada finalidad nueva.
- Fijar la base jurídica de cada tratamiento: obligación legal o misión de
  interés público; el consentimiento solo cuando sea libre de verdad.
- Publicar la información por capas: responsable, DPD, fines, legitimación,
  destinatarios, conservación, derechos y dónde reclamar.
- Firmar contratos de encargado con cada proveedor que trate datos
  (alojamiento, correo, firma, custodia, observabilidad o soporte).
- Hacer la EIPD (evaluación de impacto) si hay tratamiento masivo, decisiones
  automatizadas con efectos importantes o categorías especiales.
- Definir el circuito de derechos (acceso, rectificación, oposición,
  limitación y supresión), con plazo controlado.
- Definir la conservación por fase: expediente activo, recursos, archivo y
  bloqueo.
- Tener un procedimiento de brechas: detección, clasificación, contención,
  aviso a la autoridad y, cuando toque, a las personas afectadas.

## Seguridad (ENS)

- Categorizar el sistema por disponibilidad, autenticidad, integridad,
  confidencialidad y trazabilidad.
- Aprobar la política de seguridad, los responsables, el procedimiento de
  autorización y la aceptación de riesgos.
- Mantener vivo el análisis de riesgos y repasarlo con cada cambio importante.
- Revisar los permisos de forma periódica y dar de baja los accesos al cambiar
  de puesto.
- Ensayar la restauración de copias y tener plan de continuidad.
- Gestionar vulnerabilidades y valorar una prueba de intrusión según el riesgo.
- Documentar la declaración o certificación de conformidad y sus evidencias.

## Documento y expediente electrónico

- Aprobar la política de firma y certificados: quién firma, qué se firma, en
  qué formato, cómo se valida, el sello de tiempo y la conservación.
- Fijar los catálogos comunes (órgano, procedimiento, tipo documental, estado,
  idioma, formato) y los formatos admitidos para conservar.
- Las notificaciones deben dejar constancia de la puesta a disposición, el
  acceso o rechazo, el vencimiento del plazo y el contenido.
- Las copias auténticas conservan órgano, responsable, fecha, formato y
  relación con el original.
- El expediente debe poder leerse en el futuro aunque cambien reglas, formatos
  o proveedores.

## Transparencia

- Publicar los criterios de baremo, el desempate, el calendario, la forma de
  subsanar y los recursos.
- Los listados públicos identifican lo justo para defender derechos, sin datos
  completos innecesarios.
- Publicar la declaración de accesibilidad y el canal de reclamaciones.
