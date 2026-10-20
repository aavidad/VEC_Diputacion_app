# Especificaciones para agentes de VEC

Este documento, pedido por Alberto el 12 de septiembre de 2026, reunía trece
reglas (E01 a E13) para todo agente que trabaje en VEC. Esas reglas viven ahora
en los seis documentos de producto de [docs/producto/](docs/producto/). La
tabla dice dónde está cada una. La última columna dice si quedó algo fuera.

La versión completa anterior está en la historia de Git de este fichero.

Índice comprobado el 20/10/2026. Las reglas de trabajo local, revisión y
entrega se consultan en [AGENTS.md](AGENTS.md); la arquitectura real, en
[TRD](docs/producto/TRD.md) y el [manual del programador](docs/manual_programador/README.md).
Este índice conserva los trece identificadores como referencias técnicas;
no constituye otro plan ni un canal de coordinación.

| Regla | Dónde vive ahora | Lo que solo queda aquí |
| --- | --- | --- |
| E01. Fuentes y precedencia | [PRD §1](docs/producto/PRD.md#1-los-seis-documentos-de-producto) y [PRD §6, «De dónde salen las reglas»](docs/producto/PRD.md#de-dónde-salen-las-reglas); órdenes de trabajo en [AGENTS.md](AGENTS.md) | Nada. Que no se edita el Word original de RRHH está en [PLAN_IMPLEMENTACION §8](docs/producto/PLAN_IMPLEMENTACION.md#8-reglas-que-siguen-vigentes-de-los-relevos-anteriores) |
| E02. Fidelidad a RRHH y propiedad de los datos | [PRD §7](docs/producto/PRD.md#7-modelo-de-datos-que-no-se-simplifica), [PRD §9.2](docs/producto/PRD.md#92-contratación-temporal), [PRD §9.4](docs/producto/PRD.md#94-inscripción-requisitos-méritos-y-baremación); pantallas en [DISENO_UI_UX §13.2 y §13.3](docs/producto/DISENO_UI_UX.md#13-pantallas-del-área-personal-aspirantes-y-empleados) | Nada |
| E03. Arquitectura hexagonal y ampliación por módulos | [TRD §2](docs/producto/TRD.md#2-arquitectura-hexagonal) y [TRD §3](docs/producto/TRD.md#3-módulos); misma persona y ficha del empleado en [PRD §6](docs/producto/PRD.md#6-principios-de-producto), [PRD §7](docs/producto/PRD.md#7-modelo-de-datos-que-no-se-simplifica) y [PRD §9.8](docs/producto/PRD.md#98-personal-y-organizaciónrpt) | Nada. Qué módulos se abren fuera de la red está como pendiente en [TRD §17](docs/producto/TRD.md#17-lo-que-falta-para-producción) |
| E04. Roles, ámbitos y autorización | [TRD §6](docs/producto/TRD.md#6-autorización), [FLUJO_APP §6](docs/producto/FLUJO_APP.md#6-perfiles-ámbitos-y-autorización), [ESQUEMA_BACKEND §7.1](docs/producto/ESQUEMA_BACKEND.md#71-decisión-v3) | Nada |
| E05. Administración y fronteras | [TRD §5](docs/producto/TRD.md#5-identidad-y-sesión), [FLUJO_APP §2.3](docs/producto/FLUJO_APP.md#23-administración-en-subdominio), [PRD §9.6](docs/producto/PRD.md#96-identidad-perfiles-y-administración) | Nada |
| E06. Auditoría y efectos duraderos | [TRD §7](docs/producto/TRD.md#7-efectos-duraderos-auditoría-outbox-idempotencia-y-versión), [ESQUEMA_BACKEND §5 a §7](docs/producto/ESQUEMA_BACKEND.md#5-contrato-de-transacción), [PRD §9.7](docs/producto/PRD.md#97-auditoría) | Nada |
| E07. Privacidad, secretos y almacenamiento | [TRD §5, «Sin cookies»](docs/producto/TRD.md#sin-cookies-ni-almacenamiento-del-navegador), [TRD §8](docs/producto/TRD.md#8-protección-de-datos), [TRD §11](docs/producto/TRD.md#11-seguridad) | Nada |
| E08. Idiomas, accesibilidad y presentación | [DISENO_UI_UX §2](docs/producto/DISENO_UI_UX.md#2-principios), [§8](docs/producto/DISENO_UI_UX.md#8-accesibilidad) y [§9](docs/producto/DISENO_UI_UX.md#9-idiomas); parte técnica en [TRD §12](docs/producto/TRD.md#12-textos-idiomas-y-accesibilidad-parte-técnica) | Nada |
| E09. Correo corporativo | [TRD §16](docs/producto/TRD.md#16-integraciones-externas) y [PRD §11, «Correo»](docs/producto/PRD.md#correo) | Nada |
| E10. SQL, criptografía y revisión sensible | [ESQUEMA_BACKEND §9.5 y §9.6](docs/producto/ESQUEMA_BACKEND.md#95-down-reaplicar-y-evolución), [PLAN_IMPLEMENTACION §10.3](docs/producto/PLAN_IMPLEMENTACION.md#103-ciclo-de-entrega) | Nada |
| E11. Dirección y trabajo en paralelo | [PLAN_IMPLEMENTACION §10.1](docs/producto/PLAN_IMPLEMENTACION.md#101-encargo-y-trabajo-paralelo) | Nada. La cuota está en [AGENTS.md](AGENTS.md), «Cuota» |
| E12. Aceptación y publicación | [TRD §14](docs/producto/TRD.md#14-pruebas-e-integración-continua), [PLAN_IMPLEMENTACION §10.3](docs/producto/PLAN_IMPLEMENTACION.md#103-ciclo-de-entrega) | Nada |
| E13. Qué debe decir cada encargo | [PLAN_IMPLEMENTACION §10.1 y §10.4](docs/producto/PLAN_IMPLEMENTACION.md#104-formato-de-la-entrega) | Nada. La plantilla del encargo y el repaso cada 30 minutos están en PLAN_IMPLEMENTACION §10.1 |
