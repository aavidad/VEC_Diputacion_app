---
name: documentos-producto-vec
description: Lectura y mantenimiento de los seis documentos de producto de VEC (PRD, TRD, flujo de la app, diseño UI/UX, esquema del backend y plan de implementación). Usar al empezar cualquier tarea de VEC, al encargar trabajo a otro agente y al cerrar una entrega que cambie alcance, flujo, pantallas, esquema o plan.
---

# Documentos de producto de VEC

Los seis documentos de `docs/producto/`, más su tablero de tareas, son la referencia común de todos los
agentes. Se leen antes de programar, revisar o encargar trabajo.

| Documento | Responde a | Leer siempre que… |
| --- | --- | --- |
| `docs/producto/PRD.md` | Qué es VEC, para quién, qué módulos y con qué prioridad | empieces cualquier tarea |
| `docs/producto/TRD.md` | Cómo está construido: pila, arquitectura, seguridad, despliegue | toques código, configuración o arranque |
| `docs/producto/FLUJO_APP.md` | Quién hace qué y en qué orden, por perfil y por recorrido | toques una pantalla, una ruta o un paso del proceso |
| `docs/producto/DISENO_UI_UX.md` | Cómo se ven y se usan las pantallas | toques web/static o textos visibles |
| `docs/producto/ESQUEMA_BACKEND.md` | Esquemas, tablas, roles, migraciones y API | toques SQL, persistencia, autorización o API |
| `docs/producto/PLAN_IMPLEMENTACION.md` | Qué falta, en qué orden y con qué criterio de cierre | elijas o encargues la siguiente tarea |
| `docs/producto/TABLERO_TAREAS.md` | Qué falta, quién lo hace y en qué estado | elijas, encargues o cites una tarea (`Tarea: Fx-yy`) |

## Al empezar una tarea

1. Lee `PRD.md` y `PLAN_IMPLEMENTACION.md` enteros; después, los demás según la
   tabla. Si el documento y el código no coinciden, manda el código y las
   órdenes vigentes de `AGENTS.md`: anota la diferencia en tu entrega.
2. Comprueba que la tarea está en el plan o la pide dirección. No amplíes alcance
   por iniciativa propia.
3. Al encargar trabajo a otro agente, nombra los documentos que debe leer y los
   apartados concretos que afectan a su tarea.

## Al cerrar una entrega

Actualiza en la misma PR el documento afectado cuando cambies:

- alcance, perfiles o prioridad → `PRD.md`;
- pila, arranque, configuración, seguridad o despliegue → `TRD.md`;
- un paso, una pantalla o quién actúa → `FLUJO_APP.md`;
- un patrón visual o una regla de pantalla → `DISENO_UI_UX.md`;
- un esquema, una tabla principal, un rol PostgreSQL o una ruta de API → `ESQUEMA_BACKEND.md`;
- el cumplimiento de un criterio de fase → `PLAN_IMPLEMENTACION.md`;
- el estado de una tarea → `TABLERO_TAREAS.md` (solo dirección; el equipo lo
  pide en su entrega).

Escribe en castellano llano (skill `humanizer`, `VEC-USO.md`), describe lo que
existe y lo que falta sin adornos y cita rutas del repositorio. No pegues
secretos, rutas privadas ni datos personales. Un cambio pequeño basta: no
reescribas un documento entero para una línea.
