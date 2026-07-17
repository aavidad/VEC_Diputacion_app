# Confianza publica VEC-AD-2 en PostgreSQL

Estado: primer corte real, ejecutable y cerrado por defecto. El catalogo y el
cargador Go se prueban juntos contra PostgreSQL 18.4. Sigue siendo **NO-GO para
produccion** hasta disponer de ancla externa anti-restauracion, manifiestos de
gobierno autorizados criptograficamente y aprobacion operacional de Seguridad.

Este directorio almacena solo la lista positiva publica necesaria para verificar
atestaciones VEC-AD-2:

- suite exacta `VEC-AD-2-COSE-EDDSA-1` y algoritmo `EdDSA`;
- claves publicas Ed25519 en SPKI DER canonico;
- audiencia de despliegue gobernada;
- revisiones, ventanas, actos, revocaciones y punteros historicos;
- huellas SHA-256 y conjunto exacto de entre 1 y 64 raices.

No contiene ni admite claves privadas, semillas, HMAC, capacidades, contrasenas
de produccion, firmantes, registradores de panel o llamamientos. Tampoco concede
`EXECUTE` a esos modulos ni siembra confianza durante la migracion.

## Roles y superficie runtime

Los tres roles son grupos `NOLOGIN`, sin superusuario, `BYPASSRLS`, creacion de
roles/bases ni replicacion:

- `vec_confianza_atestacion_v2_propietario`: propietario aislado de objetos;
- `vec_confianza_atestacion_v2_migrador`: solo puede asumir al propietario para
  una migracion o acto de gobierno controlado;
- `vec_confianza_atestacion_v2_lector_autoridad`: recibe `CONNECT`, `USAGE` del
  esquema y `EXECUTE` de una unica funcion.

El LOGIN lector se aprovisiona fuera del repositorio. Debe ser no privilegiado,
tener una unica membresia directa al lector, con `ADMIN FALSE`, `INHERIT TRUE` y
`SET TRUE`. Cualquier atributo privilegiado, membresia adicional o capacidad de
delegar el rol produce SQLSTATE `42501`.

La unica salida es:

```sql
SELECT *
  FROM vec_confianza_atestacion_v2.obtener_confianza_actual();
```

Devuelve, en este orden, 16 columnas: revision, huella de configuracion,
publicacion, expiracion, estado y revocacion de configuracion; identificador,
algoritmo, suite, audiencia, SPKI, huella SPKI, ventana, estado y revocacion de
cada raiz. La funcion no recibe selectores del llamador, usa `SECURITY DEFINER`
con `search_path=pg_catalog`, bloquea los punteros y devuelve el conjunto exacto.
Una configuracion ausente, caducada, revocada, incompleta o sin al menos una raiz
activa y vigente no produce confianza.

Las raices revocadas no se filtran si la configuracion conserva otra raiz activa:
su estado forma parte del conjunto exacto. Una revocacion posterior a la
publicacion invalida la reconstruccion hasta publicar una revision que la
incorpore de forma coherente.

El lector no tiene privilegios sobre tablas, secuencias, tipos fila ni funciones
auxiliares. Todas las tablas usan `ENABLE/FORCE RLS` y una politica que exige a la
vez el rol y `current_user` propietario exacto. Los ACL por defecto y una guarda
DDL cierran tambien funciones y tipos futuros. `PUBLIC` no conserva `CONNECT`,
`CREATE`, `TEMPORARY` ni acceso al esquema.

## Instalacion

Se presupone una base VEC dedicada y endurecida. La aplicacion no ejecuta estas
migraciones durante el arranque.

```bash
psql -X -v ON_ERROR_STOP=1 -f deploy/postgresql/confianza_atestacion_v2/roles_up.sql
psql -X -v ON_ERROR_STOP=1 -f deploy/postgresql/confianza_atestacion_v2/migraciones/000001_catalogo_confianza_v2.up.sql
```

`roles_up.sql` exige superusuario y aborta antes de crear estado si `PUBLIC`
conserva privilegios en la base o en `public`. Las cuentas LOGIN y sus secretos
se gestionan mediante el sistema de secretos institucional, nunca en estos SQL.

## Gobierno y rotacion

Toda publicacion o revocacion debe hacerse en una transaccion `READ COMMITTED`.
La primera sentencia de gobierno, antes de cualquier lectura o DML, adquiere el
lock exclusivo:

```sql
BEGIN ISOLATION LEVEL READ COMMITTED;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_confianza_atestacion_v2:gobierno:v1', 0
    )
);
-- acto, versiones, miembros y punteros gobernados
COMMIT;
```

El orden de una publicacion completa es: acto, nuevas raices si existen,
configuracion, miembros exactos, punteros de raiz rotados y puntero de
configuracion. Todo se confirma en un unico `COMMIT`. Los punteros son
append-only, estrictamente monotonos y no aceptan fechas futuras. Tras activar
una revision, su membresia queda sellada. `UPDATE`, `DELETE` y `TRUNCATE` fallan.

La carga Go adquiere el lock compartido en una sentencia independiente y despues
lee con un snapshot `READ COMMITTED` fresco. Asi, una revocacion confirmada
mientras esperaba no puede quedar oculta por un snapshot anterior.

## Limites que mantienen el NO-GO

La monotonia evita rollback y resurreccion de una huella SPKI dentro de la base
viva. No detecta restaurar una copia completa antigua, promover un failover
rezagado o reescribir a la vez datos y punteros. Produccion necesita una ancla
externa monotona, verificacion de backup/WAL y procedimiento de recuperacion.

`acto_gobierno` relaciona clase, referencia y huella del documento administrativo,
pero todavia no es un manifiesto del evento firmado por una autoridad HSM/KMS.
El catalogo tampoco consume decisiones de negocio de Bolsa: esa revalidacion y
consumo unico deben ocurrir en la misma transaccion del efecto correspondiente.

## Retirada

El down es destructivo y falla incluso con el catalogo vacio salvo que el
operador aporte en esa unica sesion:

```text
-c vec.confirmar_destruccion_confianza_atestacion_v2=DESTRUIR_CONFIANZA_V2_IRREVERSIBLE
```

Despues se retiran todas las cuentas LOGIN miembro y se ejecuta
`roles_down.sql`. No se usa `CASCADE`; objetos, ACL o membresias no inventariados
hacen que la transaccion aborte.

## Prueba real

```bash
deploy/postgresql/confianza_atestacion_v2/probar_integracion.sh
```

El runner usa un contenedor efimero, contrasenas aleatorias y claves publicas de
vectores Ed25519. Comprueba migracion, ACL, RLS, tipos futuros, identidad exacta,
huella Go/PostgreSQL, revocacion concurrente, caducidad durante una espera,
anti-alias, anti-rollback, sellado, down protegido y reinstalacion limpia.
