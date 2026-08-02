# Enmienda F0-H0b: límite del runner funcional

Fecha: 2 de agosto de 2026.

Estado: **propuesta documental; exige doble `GO` documental independiente
antes de reanudar código**.

## Motivo medido

La estructura aislada H0b conserva un runner de 550 líneas. Antes de activar
el flujo funcional se hicieron tres análisis de capacidad de solo lectura: el
productor y dos revisores independientes. Ninguno modificó código.

| Análisis | Comprobación | Resultado |
| --- | --- | --- |
| Productor | Presupuesto de materialización, sesiones, causalidad y finalización. | El máximo 550 no permite añadir el flujo completo sin retirar controles o mover responsabilidades. |
| Revisor 1 | Fronteras D2c, D2d, capturador, runner y auxiliar H0b. | Rechazó como ahorro mover snapshot, rutas, huellas, derivación de raíces o bootstrap general al auxiliar H0b. |
| Revisor 2 | Presupuesto independiente de los checkpoints 3 y 4. | Confirmó que el margen debe incluir el *seam* de fallo y la matriz de inyección, no solo el camino nominal. |

Las tres lecturas convergen en estas magnitudes:

- runner actual: 550 líneas;
- coste bruto de activación funcional y cierre probatorio: entre 72 y 92
  líneas nuevas;
- ahorro potencial por reutilización o generalización: hasta 30 líneas, no
  garantizado ni acreditado en el extremo superior;
- mínimo con todo el ahorro potencial: `550 + 72 - 30 = 592` líneas;
- máximo conservador sin descontar ahorro: `550 + 92 = 642` líneas;
- rango conservador reproducible: **592–642 líneas**.

El límite 550 solo podría alcanzarse mediante compactación artificial,
eliminación de pruebas o traslado de operaciones fuera de su propietario.
Las tres opciones quedan prohibidas. Esta medición no es un `GO` de código ni
acredita H0b; justifica únicamente cambiar un límite local del arnés especial.

## Decisión de límites

Para el runner especial
`probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh` se fijan:

- presupuesto o umbral local de revisión por excepción medida: **640 líneas
  o menos**;
- límite duro: **650 líneas o menos**.

El auxiliar privado
`arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh` tiene:

- objetivo de revisión: **460 líneas o menos**;
- límite duro: **menos de 800 líneas**.

El umbral local de 640 no sustituye el objetivo general de diseño de 500
líneas de [DEC-051](registro_decisiones.md#dec-051--limite-de-tamano-de-los-ficheros-de-codigo).
El tope local de 650 permanece bajo el límite duro global de 800. Ninguno de
los umbrales autoriza minificar. El tope 650 ya incluye los checkpoints 3 y 4
completos: activación, finalizador retornable, *seam* de fallo y matriz de
inyección. Si el conjunto correcto no cabe, se detiene y se aprueba otra
enmienda; nunca se excede 650 por conveniencia.

No se añade un cuarto auxiliar. Tampoco se borran pruebas, regresiones,
comprobaciones de huella, acreditaciones de línea base ni mensajes necesarios
para distinguir fallos.

## Fronteras inmutables

El cambio de límite no cambia propietarios:

- D2c conserva análisis SQL, clasificación, rutas, clausuras, inventarios,
  snapshot, manifiestos y huellas;
- D2d conserva propiedad, descubrimiento y retirada del contenedor;
- el capturador conserva la lectura única y la copia por descriptor;
- los tres artefactos anteriores permanecen byte a byte inmutables;
- el runner conserva Docker, `psql`, sesiones, credenciales efímeras, orden
  exterior, estados causales y el finalizador único;
- el auxiliar H0b conserva únicamente plantillas, fixture R0, catálogos,
  listas positivas y oráculos retornables; no recibe contenedor, sesiones, red,
  secretos, reintentos ni decisiones de ejecución.

No se acepta como ahorro trasladar al auxiliar H0b la acreditación del
snapshot, la derivación de `/repo`, el bootstrap general o el orden de
limpieza. Tampoco se duplica esa autoridad en funciones nuevas.

## Checkpoints pendientes

El checkpoint 3 activa el flujo funcional, todavía sin matriz de inyección:

1. H0a y línea base;
2. autoprueba pura H0b;
3. subensayo sin R0, `42501` exacto, rollback y línea base;
4. creación y acreditación R0;
5. integración virtual C2 nominal y con error posterior;
6. un único finalizador sin cortocircuito desde el `COMMIT` R0;
7. ausencia R0 y retorno exacto de base de datos y ambas raíces.

El checkpoint 4 añade el *seam* y la matriz de inyección después de cada
frontera posterior al `COMMIT` y dentro de cada acción del finalizador. El
límite duro de 650 se evalúa después de ese checkpoint, no antes.

## Write-set futuro

Los checkpoints 3 y 4 pueden modificar exclusivamente:

```text
deploy/postgresql/autorizacion_atestada_v3/
  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh
```

La SHA-256 literal del auxiliar H0b se actualiza en el runner en el mismo
commit. D2c, D2d y el capturador no entran en el write-set. Cualquier tercera
ruta exige una decisión nueva antes de editar.

## Cierre y revisión

Antes de reanudar checkpoint 3, esta enmienda requiere dos revisiones
documentales independientes con `GO`. Los tres análisis de capacidad previos
son evidencia de motivación, no sustituyen esos dos dictámenes.

Después, cada checkpoint mantiene Bash, ShellCheck, límites,
`git diff --check`, Gitleaks, PostgreSQL 18.4, ausencia de residuos y revisión
independiente. El checkpoint 4 no puede apoyarse solo en el verde nominal del
checkpoint 3.

Esta enmienda no cierra H0b, C2 ni F0. El desglose permanece en `10/23`,
O4-05 en `3/5`, Contratación en `24/46`, Bolsa productiva en `1/14` y
producción en `NO-GO`.
