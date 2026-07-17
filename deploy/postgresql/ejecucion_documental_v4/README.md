# Ejecucion documental atestada V4 en PostgreSQL

Estado: implementacion tecnica de referencia y pruebas de integracion; **NO-GO
productivo** sin la separacion operacional, custodia de secretos y aprobacion
de Seguridad descritas en
[`atestacion_criptografica_decisiones.md`](../../../docs/portal_vec/atestacion_criptografica_decisiones.md).

Este directorio contiene el bootstrap de roles, las migraciones propias de la
ejecucion documental, las extensiones transaccionales del nucleo de
autorizacion y un runner efimero sobre PostgreSQL 18.4. La aplicacion no aplica
estas migraciones durante el arranque.

`roles_up.sql` exige superusuario antes de crear estado. `CREATEROLE`, incluso
combinado con la propiedad de la base, no es una autoridad de bootstrap valida.

## Retirada de roles

`roles_down.sql` solo se ejecuta despues de retirar las migraciones y todas las
cuentas `LOGIN`. Es una operacion de superusuario para una ventana de
mantenimiento del cluster. El orden de exclusiones es obligatorio:

1. bloqueo advisory del modulo;
2. `pg_authid` en `ACCESS EXCLUSIVE`;
3. `pg_auth_members` en `ACCESS EXCLUSIVE`;
4. prevalidacion completa;
5. retirada explicita de ACL, membresia estructural y roles.

La prevalidacion fija los cuatro OID, todos sus atributos —incluidos contraseña,
caducidad y ajustes globales o por base— y la unica membresia admitida:
propietario a migrador con `ADMIN FALSE`, `INHERIT FALSE`, `SET TRUE` y el
superusuario bootstrap del cluster como otorgante. Tambien inventaria `roleid`,
`member` y `grantor`; cualquier relacion adicional aborta antes de mutar.
PostgreSQL atribuye al bootstrap (OID interno 10) las membresias concedidas por
cualquier superusuario, aunque la operacion la ejecute otra cuenta DBA.

La guarda DDL se identifica por OID, no solo por nombre. El esquema y la funcion
deben pertenecer al mismo DBA que ejecuta la retirada, conservar sus ACL
exactas, y el disparador debe mantener evento, estado, conjunto exacto de
etiquetas y funcion exactos. Tambien se fijan lenguaje, `SECURITY DEFINER`,
`search_path`, volatilidad, paralelismo, aridad y el inventario completo de
objetos y usos dentro de la guarda.

Antes del primer `DROP` o `REVOKE` se inventarian igualmente las cinco
concesiones exactas de base, `USAGE` sobre `public`, el unico `EXECUTE` de HMAC,
las ACL de las demas clases de objeto y los privilegios por defecto. Se admite
el inventario vacio de un bootstrap que aun no alcanzo la primera migracion o
las dos filas globales exactas creadas por ella; ningun estado intermedio. Una
concesion adicional provoca un fallo sin retirar autoridad ni borrar
dependencias ajenas.

No se usa `CASCADE`. `DROP ROLE` y la transaccion completa fallan cerrados ante
cualquier dependencia no prevista.

## Prueba real

```bash
deploy/postgresql/ejecucion_documental_v4/probar_integracion.sh
```

Ademas de migraciones, ACL, RLS y pruebas Go, el runner comprueba:

- rechazo sin estado de un propietario de base con `CREATEROLE` pero no
  superusuario;
- retirada y reinstalacion limpias de un bootstrap interrumpido antes de la
  primera migracion;
- membresias entrantes, salientes y un rol V4 presente solo como `grantor`;
- contraseña, caducidad y ajustes globales o por base manipulados;
- propietario, ACL, `proconfig`, etiquetas, objetos y usos de guarda alterados;
- ACL de base, esquema, funcion, tabla y privilegios por defecto adicionales;
- un ciclo completo creado y retirado por un DBA superusuario alternativo,
  separando su propiedad del `grantor` bootstrap.

Finalmente ejecuta una carrera real entre bases. Un `GRANT` espera al proceso
de retirada sin ser cancelado ni terminado, falla naturalmente tras
`DROP ROLE` con SQLSTATE `42704` y la identidad concreta del rol retirado, y se
verifican:

- los tres campos OID de `pg_auth_members`;
- la desaparicion exacta de los cuatro roles;
- la ausencia global de membresias huerfanas;
- que la retirada no reabra las funciones de `pgcrypto` a `PUBLIC`.

El observador se conecta antes de congelar `pg_database` y usa
`pg_stat_get_activity()` y `pg_lock_status()` directamente. La vista
`pg_stat_activity` consulta `pg_authid` y quedaria bloqueada por la propia
proteccion que se pretende demostrar.
