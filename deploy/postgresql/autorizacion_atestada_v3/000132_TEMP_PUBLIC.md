# AD3-132: privilegio TEMP de la base VEC

Esta reparación se ejecuta después de las SQL funcionales del H6 y antes de
arrancar la aplicación nueva. Retira únicamente `TEMPORARY` de `PUBLIC` en la
base `postgres` identificada por la preimagen VEC. Conserva `CONNECT`, `CREATE`,
el propietario, las concesiones directas y la historia. No tiene `DOWN`: una
concesión nueva de `TEMPORARY` requeriría otro cambio aprobado.

La ruta de operación es `aplicar_000132_temp_public.py`. Se usa con el ID completo
del contenedor PostgreSQL aislado, un plan de conexiones obtenido del canario
interno antes del arranque, la lista causal SQL del H6 y el commit fuente fijado.
El CLI se conecta como el dueño DBA de la base. No recibe DSN ni contraseña.

```sh
python3 deploy/postgresql/autorizacion_atestada_v3/aplicar_000132_temp_public.py preview \
  --engine podman --container "$PG_ID" --plan "$PLAN_H6" \
  --sql-list "$LISTA_H6" --source-commit "$FUENTE_SHA" \
  --output "$APROBACION_PRIVADA"
```

`preview` crea un JSON privado con `approved: false`, inventario de todos los
LOGIN y CONNECT, sesiones activas, membresías, acceso a esquemas VEC, ACL de la
base y huellas de preimagen. El DBA verifica cada cuenta CONNECT que no figura
en el plan interno. Para una cuenta VEC adicional inactiva consigna finalidad,
provisión SQL de la fuente con SHA256 o referencia privada de provisión, y su
aprobación referenciada. Si la cuenta pertenece a otro servicio, carece de
procedencia comprobable o mantiene sesiones, se detiene el corte. Después el
DBA fija `approval_ref` y `approved: true` en ese mismo JSON. El archivo queda
fuera de Git y no contiene claves.

Con los mismos argumentos, se sustituye `preview --output ...` por
`trial --approval "$APROBACION_PRIVADA"` y después por
`apply --approval "$APROBACION_PRIVADA"`. `trial` ejecuta la transacción con
`ROLLBACK` y exige preimagen idéntica. `apply` vuelve a inventariar y compara
la aprobación exacta antes del efecto. Un cambio de ACL, propietario, roles,
membresías, plan, lista o fuente exige otra previsualización y aprobación.

La migración comprueba la base, dueño, esquemas, funciones, objetos públicos
conocidos, extensiones, roles y sesiones. Después de revocar, compara cada
privilegio de la ACL salvo `PUBLIC TEMPORARY`, confirma que los LOGIN de
aplicación no conservan TEMP efectivo y exige que CONNECT siga igual. Un fallo
de estas comprobaciones revierte toda la transacción. Tras `apply`, se repite
la inspección de LOGIN del kit antes del arranque y tras el reinicio.

Las migraciones que usan tablas temporales durante su instalación deben
preceder AD3-132 y ejecutarse como DBA. No se concede TEMP al proceso de
aplicación. Una necesidad futura de TEMP se estudia para el rol técnico y
operación concretos; no se devuelve el permiso a `PUBLIC`.
