# AD3-132: privilegio TEMP de la base VEC

Esta reparación va después de las 45 SQL funcionales del H6 y antes de
arrancar la aplicación nueva. Retira sólo `TEMPORARY` de `PUBLIC` en la base
`postgres` identificada por la preimagen VEC. Conserva `CONNECT`, `CREATE`, el
propietario, los demás privilegios y la historia. No tiene `DOWN`.

La CLI recibe el plan del canario interno, el manifiesto SQL construido desde
un único commit y el lock verificado del paquete. Comprueba los SHA256 de las
45 SQL, de la migración, de la propia CLI, del inventario y de las anclas
CT145/AD125/CT152/CT153. Consulta PostgreSQL para comprobar que esas anclas tienen
las definiciones, propietarios, ACL y configuración fijados tras instalar el
lote. Una lista recortada o un archivo cambiado detienen la operación.

```sh
python3 deploy/postgresql/autorizacion_atestada_v3/aplicar_000132_temp_public.py preview \
  --engine podman --container "$PG_ID" --plan "$PLAN_H6" \
  --plan-receipt "$RECIBO_PLAN_H6" \
  --release-manifest "$PAQUETE/h6-sql-release.json" \
  --release-lock "$LOCK_H6" --output "$APROBACION_PRIVADA"
```

`preview` crea un JSON privado con `approved: false`. Incluye todos los LOGIN
con CONNECT, su actividad y membresías, el plan, la ACL y las huellas del
release y de la base. Cualquier LOGIN conectable fuera del plan y del dueño
DBA queda en `unexplained_connect_logins`: `trial` y `apply` se detienen aunque
alguien cambie el campo `approved`. Un nombre, una membresía, USAGE de un
esquema o una referencia de ticket no acreditan la provisión de otro servicio.
Si hay un extra, Alberto debe localizar su configuración o provisión real y
someter ese origen a revisión antes de continuar. No se añade una excepción
manual a esta reparación.

El recibo del plan lo emite el guion fijado del kit al promover el archivo
producido por el canario sin red. Liga el SHA del plan, material, paquete,
imagen, arranque y contenedor. En principal deriva del recibo del clon y liga
su propio contenedor. Es un control de consistencia del proceso local; no es
una firma frente a quien ya puede operar PostgreSQL como DBA.

Cuando el inventario no contiene extras y el DBA aprueba la preimagen exacta,
fija `approval_ref` y `approved: true` en el mismo archivo 0600. Se usan los
mismos argumentos con `trial --approval "$APROBACION_PRIVADA"` y después con
`apply --approval "$APROBACION_PRIVADA"`. `trial` ejecuta la transacción con
`ROLLBACK` y exige preimagen idéntica. `apply` revalida manifiesto, código,
lista, plan, anclas, roles y ACL, y pasa a `psql` los bytes SQL ya comprobados.

Una concesión TEMP directa o heredada a cualquier LOGIN de aplicación detiene
la operación y revierte la transacción. Tras `apply`, se comprueban de nuevo
CONNECT, roles, propietario, ACL y anclas. Si `psql` falla durante la aplicación,
el resultado se declara incierto aunque se observe una postimagen compatible:
el DBA debe conciliarla antes de reintentar o arrancar. No se reaplica la
migración por inferencia.

Las migraciones que usan tablas temporales al instalarse deben preceder
AD3-132 y ejecutarse con el DBA autorizado. Si alguna operación de aplicación
necesita TEMP en el futuro, se estudia su rol técnico concreto; no se devuelve
el privilegio a `PUBLIC`.
