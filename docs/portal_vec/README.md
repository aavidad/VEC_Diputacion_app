# Contratos, estudios e historia técnica del portal VEC

Esta carpeta conserva 326 documentos anteriores a este índice, revisado el
20/10/2026 desde `origin/main@3ef7dee0b`. Incluye contratos de capacidades,
estudios de seguridad, propuestas y actas de cortes anteriores. La fecha o la
palabra «vinculante» dentro de un documento no acreditan su estado actual.
No se reescriben esos documentos ni se descarta su evidencia histórica.

## Referencias vigentes

| Para consultar | Referencia actual |
| --- | --- |
| Alcance y prioridades de producto | [PRD](../producto/PRD.md) |
| Arquitectura, composición, seguridad y operación | [TRD](../producto/TRD.md) |
| Recorrido único y variantes, perfiles y pantallas | [FLUJO_APP](../producto/FLUJO_APP.md), especialmente §4.0 |
| Sistema visual, accesibilidad e idiomas | [DISENO_UI_UX](../producto/DISENO_UI_UX.md) |
| Datos, propietarios, roles, transacciones, migraciones y API | [ESQUEMA_BACKEND](../producto/ESQUEMA_BACKEND.md) |
| Orden de construcción y condiciones de cierre | [PLAN_IMPLEMENTACION](../producto/PLAN_IMPLEMENTACION.md) |
| Estado de cada tarea | [TABLERO_TAREAS](../producto/TABLERO_TAREAS.md) |
| Instrucciones para trabajar | [AGENTS.md](../../AGENTS.md) |

El código de `origin/main` y las órdenes vigentes mandan ante una diferencia.
Código integrado, instalado y recorrido son estados distintos. Se trabaja con
copias locales y datos ficticios; cidonia permanece congelada. Un contrato
antiguo no autoriza reinstalar una migración ni activar un servicio.

## Detalle que se sigue consultando

| Documento | Qué conserva útil | Qué consultar para el estado actual |
| --- | --- | --- |
| [Contrato de módulos](contrato_modulos_vec.md) | Navegación hacia las reglas actuales y secuencia de activación y compatibilidad bajo «Lo que sigue vigente» | TRD §3 y §4; ESQUEMA_BACKEND §3 y §9. Las propuestas de Kerberos exclusivo, Lit, Module Federation y GrxFirma para entrar están superadas |
| [Seguridad de persistencia PostgreSQL](seguridad_persistencia_postgresql.md) | Citas técnicas y pruebas de aceptación detalladas de su §9 | ESQUEMA_BACKEND §3 a §6, §9 y §13. Su corte de julio y sus afirmaciones de composición productiva no describen el despliegue actual |
| [Contrato común de auditoría](contrato_auditoria_comun.md) | Contratos de entrada y consulta, atomicidad y límites del corte del 03/10 | TRD §7, ESQUEMA_BACKEND §5 a §7 y `internal/vec/auditoria/contrato.go`. El contrato genérico no acredita por sí solo una transacción ni una entrega externa |
| [Aislamiento modular](aislamiento_modular_y_dependencias_2026-07-25.md) | Detalle de diseño al que remite el contrato de módulos | TRD §2 y §3, y el montaje concreto del código |
| [Catálogos configurables](catalogos_configurables.md) | Antecedente de versionado, huellas y gobierno del 14/07 | ESQUEMA_BACKEND y código del catálogo. Su lista de pendientes de producción no es un estado vigente |

Las reglas generales ya absorbidas están inventariadas en TRD §18 y
ESQUEMA_BACKEND §13. Algunos nombres citados allí corresponden a documentos
retirados y se recuperan en la historia de Git; no se recrean como otra norma.
Por ejemplo, el antiguo documento de arquitectura técnica ya no está en esta
carpeta. El [contrato de módulos](contrato_modulos_vec.md) conserva una nota
que explica dónde vive cada parte.

## Cómo leer los demás documentos

Un contrato específico sigue sirviendo para entender su operación cuando
coincide con el código actual y sus pruebas. Las propuestas, auditorías y actas
fechadas conservan el resultado de aquel corte; no acreditan que sus pendientes
sigan abiertos ni que una capacidad esté publicada hoy. Antes de reutilizar
una instrucción, contraste el documento de producto de su área, la ruta de
código y la tarea del tablero. No extrapole un ensayo parcial al caso completo.

Para desarrollar use el [manual del programador](../manual_programador/README.md).
Los prototipos y capturas se localizan desde [Diseño](../diseno/README.md).
Los contratos HTTP acotados viven en `docs/api/`; la composición efectiva y
sus selectores determinan qué rutas están disponibles.

El [estudio técnico de septiembre](../historico/comunicacion_202609.md)
conserva el contexto de decisiones anteriores. Sus turnos, ramas y encargos
son históricos y no rigen la operación actual.
