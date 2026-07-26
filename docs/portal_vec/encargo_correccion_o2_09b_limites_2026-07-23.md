# Encargo O2-09B — corrección de límites web

Fecha: 23 de julio de 2026.

## Base inmutable

- worktree: `.worktrees/ct-o2-09-web`;
- rama: `agent/ct-o2-09-web`;
- candidato revisado: `1323b4b382b28de21d2d8036346f13657e755242`;
- dictamen:
  [`revisiones/o2_09a_revision_independiente_2026-07-23.md`](revisiones/o2_09a_revision_independiente_2026-07-23.md).

No se modifica integración, no se registra una ruta y no se incorpora un
adaptador DEMO.

## Resultado exigido

Alinear la validación de la vista con los límites públicos de O2-08B:

1. periodo máximo de cien años civiles;
2. importe RC máximo de `922337203685477` céntimos.

Los límites deben tener un único nombre explícito en el módulo web, mensajes en
el catálogo i18n y regresiones de borde. No se copian reglas mediante números
mágicos dispersos.

## Pruebas mínimas

- exactamente cien años: aceptado;
- cien años y un día: rechazado antes de invocar el ejecutor;
- exactamente `922337203685477` céntimos: aceptado sin pérdida;
- `922337203685478` céntimos: rechazado;
- un importe decimal que supere el máximo aunque siga siendo entero seguro de
  JavaScript: rechazado;
- comando cerrado, doble envío, recibo y pruebas actuales: sin regresión;
- ningún `fetch`, cookie, almacenamiento web ni autoridad nueva.

La conversión monetaria no puede depender de una multiplicación de coma
flotante que pierda precisión. Se admite analizar la parte entera y decimal
como dígitos y comparar de forma exacta.

## Puertas

```text
node --test web/static/portal-empleado/modulos/contratacion-temporal/contratacion-temporal.test.mjs
node --check web/static/portal-empleado/modulos/contratacion-temporal/*.js
git diff --check
```

Además:

- repetir la suite al menos cincuenta veces;
- revisar límites de tamaño;
- buscar cookies, almacenamiento, credenciales y rutas fijadas;
- Gitleaks sobre los commits;
- dejar worktree limpio y SHA exacto.

El productor no emite GO. Dirección o un agente distinto revisará el candidato.
