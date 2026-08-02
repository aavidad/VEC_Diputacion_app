# Revisión F0-H0b: estructura aislada

Fecha: 2 de agosto de 2026.

## Resultado

La estructura probatoria de H0b quedó integrada por *squash* limpio en
`ad8b170`. El corte conserva H0a, separa `/repo` y `/repo_h0b` a partir de un
único snapshot, carga un tercer auxiliar privado y devuelve D2c a su frontera
de análisis, inventario y huellas.

Este resultado es estructural. El orden exterior R0/H0b, las integraciones
virtuales C2, el finalizador único y su matriz de inyección de fallos continúan
dormidos y pertenecen al commit funcional 2.

## Cadena de revisión

El candidato `2d27303` obtuvo doble `NO-GO`, con recuento consolidado
`P0=0, P1=1, P2=0`: generaba M010/T010 sintéticos y comprobaba la SHA-256
después de copiarlos, pero no ejecutaba `validar_componentes_sql_f0` antes de
cada copia.

La corrección `ca46ba9` validó por separado M010 y T010 nominales tras
generarlos y antes de copiarlos. Tras reescribir T010 con el caso de error,
lo volvió a validar antes de la segunda copia. Dos revisiones finales dieron
`GO`, `P0=P1=P2=0`.

## Evidencia reproducida

- `bash -n`, ShellCheck y `git diff --check`: verdes;
- rechazo autónomo de los tres auxiliares shell con estado `64`;
- D2d y el capturador Go permanecen byte a byte inmutables;
- runner `550` líneas, auxiliar SQL `588` y auxiliar H0b `241`;
- H0 tres veces, A1 y C1 sobre PostgreSQL 18.4 fijado por digest: verdes;
- H0a nominal/error volvió a la línea base sin alterar `/repo_h0b`;
- cero contenedores y cero temporales residuales;
- Gitleaks del commit de corrección: sin hallazgos.

El árbol técnico `ad8b170` quedó publicado junto con esta evidencia en el
corte `11b237a`. Su ejecución CI `30728047031` terminó completamente verde;
no se hereda el verde de una base anterior como prueba del nuevo árbol.

## Alcance y estado

Quedan estructuralmente acreditados H0a, la captura viva única, los dos
manifiestos derivados, las dos raíces aisladas y el tercer auxiliar privado.
No quedan acreditados el flujo exterior R0/H0b ni el cierre funcional de H0b.

Las métricas permanecen en F0 `10/23`, O4-05 `3/5`, Contratación `24/46` y
Bolsa productiva `1/14`. Producción continúa en `NO-GO`.

Contratos aplicables: [enmienda de aislamiento](../enmienda_f0_h0b_auxiliar_privado_r0_2026-08-02.md)
y [decisión R0 sintético](../decision_f0_h0b_r0_sintetico_c2_2026-08-02.md).
