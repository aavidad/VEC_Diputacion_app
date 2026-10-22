# Diseño y arquitectura de VEC

Referencia revisada el 20/10/2026 sobre `origin/main@3ef7dee0b`.
La arquitectura vigente está en los documentos de producto:

| Tema | Referencia |
| --- | --- |
| Capas, módulos, composición y arranque | [TRD §2 a §4](../producto/TRD.md#2-arquitectura-hexagonal) |
| Propietarios de datos, transacciones, permisos y migraciones | [ESQUEMA_BACKEND](../producto/ESQUEMA_BACKEND.md) |
| Perfiles y recorrido de aceptación | [FLUJO_APP](../producto/FLUJO_APP.md) |
| Estructura visual, accesibilidad e idiomas | [DISENO_UI_UX](../producto/DISENO_UI_UX.md) |
| Cómo desarrollar una capacidad | [Manual del programador](../manual_programador/README.md) |

El código se distribuye en `internal/modules`, núcleo común `internal/vec`,
utilidades `internal/shared` y composición `internal/app/bootstrap` e
`internal/app/composicion`. Persona es común; cada módulo conserva sus datos
y se comunica con otros mediante contratos autorizados. Una carpeta de módulo
no acredita una capacidad terminada. El trabajo se verifica en copias locales;
cidonia permanece congelada.

## Material de esta carpeta

- [Diseño del módulo Aspirantes del 29/09](aspirantes_modulo_2026-09-29.md):
  detalle específico de propiedad y datos externos. Se conserva como referencia
  de diseño; el estado de sus capacidades se consulta en el código y el tablero.
- `direccion-20260929/`: capturas y prototipos de la revisión visual del 29/09.
  Son material histórico de diseño. No se ejecutan como portal ni prueban
  que una operación actual esté conectada o disponible.

Los contratos y estudios de otras fechas se localizan desde el
[índice de portal_vec](../portal_vec/README.md). Ante una diferencia, mandan
el código actual y las órdenes vigentes; el diseño fechado conserva su contexto.
