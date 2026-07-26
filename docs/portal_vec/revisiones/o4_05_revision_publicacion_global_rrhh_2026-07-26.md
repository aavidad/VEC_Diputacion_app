# O4-05: revisión de la publicación global estable RRHH

Fecha: 26 de julio de 2026

Ámbito: C2-C/`000037`, ordinal global de versiones para consultas RRHH

Estado: GO técnico independiente, integrado en `3cb17ca`; no autoriza
producción

## Problema cerrado

`expediente_version_integral.registrada_en` no podía actuar como corte global.
Su precisión admite empates y dos transacciones pueden confirmar en un orden
distinto al de sus relojes. Una secuencia consumida antes del `COMMIT` tampoco
resuelve el problema: un escritor lento podría reservar un ordinal menor y
confirmar después que otro.

La solución debía ser aditiva sobre la barrera global 16 y el control de
consultas v1. No podía modificar las migraciones `000006`–`000036`, añadir una
columna a la historia inmutable ni introducir cursor o fachada antes de tener
un corte fiable.

## Diseño revisado

La migración
[`000037_publicacion_global_rrhh.up.sql`](../../../deploy/postgresql/contratacion_temporal/migraciones/000037_publicacion_global_rrhh.up.sql)
crea dos tablas propias:

- `control_publicacion_rrhh`, singleton que conserva `corte_base` y
  `ultimo_corte`;
- `publicacion_version_rrhh`, proyección 1:1 con FK exacta
  `(expediente_ref, version)` hacia la historia integral y
  `corte_global` único, positivo y limitado a `2^53-1`.

La proyección extrae y coteja el resumen mínimo indexable: organización,
número visible, flujo y su versión/huella, fase, estado, centro, categoría,
modalidad opcional, unidad opcional de asignación, instantes y huella del
agregado. Referencia, versión, flujo, fase, estado y huella se contrastan con
las columnas autoritativas de la historia; organización y número visible se
contrastan con el alta.

Un bloque `analisis` presente debe ser un objeto completo con categoría y
modalidad. Un bloque `asignacion` presente debe aportar unidad. JSON `null`,
objetos vacíos, arrays, texto o campos ausentes se rechazan; la ausencia real
del bloque sí produce el campo opcional nulo.

## Backfill y orden posterior

El ascenso bloquea `expediente_version_integral` en modo que excluye nuevas
inserciones, valida que la cantidad cabe en el rango JSON seguro y realiza el
backfill por:

```text
expediente_ref COLLATE "C", version
```

Ese ordinal reproducible solo delimita `corte_base`; no afirma un orden
histórico de confirmación. La misma transacción instala después un disparador
`AFTER INSERT` `SECURITY DEFINER`.

Cada inserción nueva bloquea el singleton, asigna `ultimo_corte + 1`, inserta
la proyección y actualiza el control. El bloqueo se conserva hasta el final de
la transacción exterior. Un segundo escritor no obtiene ordinal hasta que el
primero confirma; un `ROLLBACK` revierte historia, proyección y singleton, y
permite reutilizar el ordinal sin hueco ni fantasma.

## Seguridad y reversión

Ambas tablas tienen propietario exacto, RLS habilitada y forzada y una única
política para `vec_contratacion_temporal_propietario`. `PUBLIC`, migrador,
ejecutor, gobernador, confirmadores, lectores y consultor RRHH carecen de
acceso directo. La proyección rechaza `UPDATE`, `DELETE` y `TRUNCATE`.

La reversión
[`000037_publicacion_global_rrhh.down.sql`](../../../deploy/postgresql/contratacion_temporal/migraciones/000037_publicacion_global_rrhh.down.sql)
exige barrera 17 y control de consultas v1. Bloquea la historia durante la
prevalidación y solo vuelve a 16 cuando:

- no existe publicación posterior a `corte_base`;
- singleton, fuente y proyección siguen siendo 1:1 y no divergen;
- no existen FK, vistas, disparadores, cursores ni fachadas futuras
  dependientes.

Los accesos C2-B preexistentes no bloquean la retirada porque no referencian
esta proyección. No se usa `CASCADE`, excepción destructiva ni modificación de
la historia.

## Evidencia reproducida

El runner
[`probar_o4_05_publicacion_global_rrhh_pg18_4.sh`](../../../deploy/postgresql/contratacion_temporal/probar_o4_05_publicacion_global_rrhh_pg18_4.sh)
terminó con:

```text
O4-05 C2-C PostgreSQL 18.4: GO técnico
```

La imagen está fijada por digest a PostgreSQL 18.4. El contenedor se ejecuta
sin red ni puertos, con almacenamiento efímero, instala dependencias reales y
las migraciones de Contratación temporal `000001`–`000036`, y no deja residuos.

La matriz verificó:

- instalación vacía, singleton `0/0` y ciclo `16→17→16` aun con accesos C2-B;
- dos `up` y dos `down` concurrentes con un único ganador;
- backfill no vacío, determinista y sin afirmar orden histórico;
- extracción base y completa, empates temporales y orden `COLLATE "C"`;
- rechazo de JSON hostil y divergencias de referencia, versión, organización,
  número, flujo, fase, estado y huella;
- primer ordinal posterior igual a `corte_base+1`;
- rollback sin fila fantasma, sin avance y con reutilización del ordinal;
- dos escritores, donde el segundo espera el `COMMIT` del primero;
- ocho escritores adicionales, cortes únicos y continuos `1..16`;
- propiedad, RLS forzada, ACL, índices y disparadores de inmutabilidad;
- rechazo seguro del down ante historia posterior, vista dependiente o
  proyección adulterada, sin cambio parcial;
- retirada final del bloque base y restauración exacta de la barrera 16.

`bash -n`, ShellCheck y `git diff --check` forman parte de las puertas del
corte. Los cinco ficheros de migración, pruebas y runner permanecen por debajo
del límite duro de 800 líneas.

## Límites

Este GO acredita únicamente la publicación global estable. Todavía no existen
cursor autenticado, fachadas exteriores de cuadro y detalle, consumo/lectura/
auditoría en una transacción, adaptador Go, matriz TLS viva, composición raíz
ni E2E HTTP. No cambia `19/46`, el 41 % oficial ni el `NO-GO` productivo.
