# Paquete de cumplimiento para las mesas de validación

**Estado: BORRADOR TÉCNICO PARA VALIDACIÓN. Ningún documento de este
directorio está aprobado ni surte efecto hasta su validación formal.**

Fecha de elaboración: 17 de julio de 2026.
Elaborado por: dirección técnica del proyecto VEC.

## Contenido

| Documento | Marco | Debe validarlo |
| --- | --- | --- |
| [Categorización ENS y declaración de aplicabilidad](ens_categorizacion_y_aplicabilidad.md) | ENS — RD 311/2022 | Comité de Seguridad, responsable del sistema, Secretaría |
| [Registro de actividades de tratamiento (RAT)](rat_registro_actividades_tratamiento.md) | RGPD art. 30, LOPDGDD | Delegado de Protección de Datos, responsable del tratamiento |
| [Evaluación de impacto (EIPD) del módulo de Bolsas](eipd_evaluacion_impacto_bolsas.md) | RGPD art. 35 | Delegado de Protección de Datos, Comité de Seguridad, Servicio de Selección |

Existe una versión imprimible conjunta en
`paquete_validacion_cumplimiento.pdf`, regenerable con `generar_paquete.py`.

## Método

Los tres documentos describen el sistema **tal y como está construido**, con
la evidencia técnica citada (paquete de código, decisión DEC o documento del
repositorio), y separan de forma expresa:

- lo ya implantado y verificable en el código;
- lo diseñado pero pendiente de componer (con su tarea Txx o dependencia);
- lo que solo las personas responsables pueden decidir o firmar.

Las propuestas de valoración (niveles ENS, plazos de conservación, juicio de
proporcionalidad) son exactamente eso, propuestas: los huecos marcados
`[pendiente de la mesa]` deben resolverse en la validación.

## Relación con el resto del repositorio

- [Auditoría de diseño y seguridad](../portal_vec/auditoria_diseno_y_seguridad_2026-07-16.md)
- [Cumplimiento y seguridad del portal](../portal_vec/cumplimiento_y_seguridad.md)
- [Informe del comité de seguridad](../comite_seguridad/LEEME.md)
- [Estudios de requisitos de RRHH](../estudio_requisitos/README.md) — los
  cuatro estudios con puerta NO-GO que acompañan a este paquete en las mesas.
- [Registro de decisiones](../portal_vec/registro_decisiones.md)
