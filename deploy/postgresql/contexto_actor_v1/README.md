# Registro físico de contexto V1, contrato durable V2

Este módulo implementa el registro autoritativo consumido por
`ResolutorRegistroContextoActorV2`. No autentica ni autoriza. Tampoco acepta
DNI, certificado, correo, nombre, roles, permisos o claims para descubrir una
persona o un perfil: las únicas entradas de resolución son las referencias
opacas exactas `cta_` y `prf_` ya acreditadas por la frontera de identidad.

## Modelo y garantías

- La cuenta aparece exclusivamente como `proyeccion_cuenta_*`: no es una
  segunda autoridad de identidad. Persona, perfil, vínculo
  cuenta-perfil-persona y vínculos a módulos también tienen historias
  versionadas append-only y punteros actuales separados.
- Cada versión resoluble referencia una procedencia inmutable exacta
  `(procedencia_ref, procedencia_version, procedencia_huella_sha256,
  procedencia_autoridad)`. Las revisiones son monotónicas y dos huellas no
  pueden compartir la misma referencia y versión.
- Las ventanas son `[vigente_desde, vigente_hasta)` y solo el estado `activo`
  produce contexto.
- Una operación bloquea en orden determinista solo la cuenta y el perfil
  exactos, las coincidencias `vca_`, las personas derivadas y sus vínculos; los
  relee y solo después observa `clock_timestamp()`. Actores independientes no
  quedan serializados globalmente.
- Cero o más de una vinculación exacta de cuenta y perfil producen la misma
  denegación. No hay `LIMIT 1`, perfil predeterminado ni precedencia.
- El snapshot incluye todos los vínculos actuales de la persona. Una
  revocación, expiración, tipo repetido o referencia repetida deniega el
  conjunto completo.
- PostgreSQL construye los bytes exactos de
  `RepresentacionCanonicaVinculadaV2`, incluyendo `cuenta_version`, persiste
  esos bytes y su SHA-256, y los devuelve dentro de la misma transacción
  `SERIALIZABLE` de escritura. El canon histórico V1 no se reinterpreta: solo
  admite `CuentaVersion == 0` y rechaza una capacidad V2 para impedir downgrade
  silencioso.
- La misma confirmación incluye un manifiesto de procedencia JSON cerrado y
  canónico, sus bytes exactos, su SHA-256 y la autoridad efectiva. Cuenta,
  persona, perfil, vínculo de contexto y vínculos a módulos incluyen su
  referencia, versión y procedencia; el adaptador exige que componentes y
  versiones coincidan exactamente con el `ContextoActor` confirmado.
- Las versiones Go siguen siendo `uint64`; PostgreSQL las representa mediante
  `numeric(20,0)` y restringe cada valor al intervalo
  `1..18446744073709551615`, sin estrecharlas a `bigint`.
- Los sufijos de `oca_` y `rca_` tienen al menos 24 caracteres. Las referencias
  de componentes (`cta_`, `per_`, `prf_`, `vca_`, `vin_`, `prc_` y referencias
  de módulo) conservan el contrato de al menos 22 caracteres.
- `operacion_ref` (`oca_`) es idempotente. Una repetición con la misma solicitud
  recupera el `rca_` ya confirmado; cualquier campo diferente colisiona
  cerrada. El `registro_contexto_ref` inicial se genera con otros 192 bits
  CSPRNG.
- Ante un COMMIT incierto, el adaptador consulta exclusivamente la misma
  operación y coteja recibo, bytes, huella e instante. La reconciliación usa
  `READ COMMITTED`, espera el mismo advisory lock de `operacion_ref` y solo
  después consulta con un snapshot renovado; así no concluye ausencia mientras
  finaliza un COMMIT concurrente. Si confirma ausencia, hace como máximo un
  reintento con los mismos `oca_` y `rca_`.
- `ServicioContextoActor.ResolverRegistrado` devuelve la confirmación completa:
  actor, `rca_`, canon, huellas, manifiesto y autoridad. La API heredada que
  devuelve solo `ContextoActor` falla cerrada en modo productivo.

## Roles y despliegue

Requiere PostgreSQL 18 y una base dedicada. Antes del bootstrap, el DBA debe
retirar de `PUBLIC` todos los privilegios de esa base y del esquema `public`;
`roles_up.sql` valida la precondición y no modifica ACL globales que el down no
podría reconstruir:

```sql
REVOKE ALL PRIVILEGES ON DATABASE mi_base_vec FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
```

Después, como superusuario:

```sh
psql -X -v ON_ERROR_STOP=1 -f deploy/postgresql/contexto_actor_v1/roles_up.sql
psql -X -v ON_ERROR_STOP=1 -f deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql
```

El despliegue crea tres roles `NOLOGIN`: propietario, migrador y runtime. El
operador crea un `LOGIN` dedicado sin atributos administrativos y le concede
solo `vec_contexto_actor_v1_runtime`, sin `ADMIN OPTION` y con `SET FALSE`:

```sql
GRANT vec_contexto_actor_v1_runtime TO mi_login_contexto_actor
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
```

El constructor Go acredita ese LOGIN. Las funciones también verifican en cada
llamada que siga siendo un LOGIN ordinario, sin `SET ROLE`, configuración de
rol ni membresías adicionales. Runtime recibe `USAGE` del esquema y `EXECUTE`
solo sobre acreditación, resolución y reconciliación; no recibe privilegios de
tabla, secuencia, tipos fila ni auxiliares. La acreditación evalúa además los
privilegios efectivos heredados de `PUBLIC` o membresías sobre toda la base y
rechaza cualquier esquema, relación, secuencia, función, tipo u objeto definido
por usuarios fuera del allowlist. En PostgreSQL 18 incluye expresamente
`MAINTAIN` sobre relaciones y las ACL globales de parámetros `SET` y
`ALTER SYSTEM`, consultadas mediante `pg_parameter_acl` y
`has_parameter_privilege`; también se rechazan cuando llegan por `PUBLIC` o por
la membresía runtime permitida. El permiso temporal `CREATE` del propietario
sobre la base existe solo dentro de la transacción que crea el esquema y se
revoca antes del commit.

Las inserciones y cambios de puntero pertenecen a un proceso corporativo de
gobierno que este corte no inventa. `pruebas_sql/fixtures_sinteticos.sql` es
solo material sintético de integración, marcado expresamente, y no es una
fuente corporativa. La función productiva rechaza toda procedencia cuya
autoridad no sea exactamente `autoridad_maestra_acreditada`; no existe una ruta
de desarrollo alternativa. El runner comienza con fixtures
`no_autoritativa`, comprueba que no producen recibo, y solo después añade una
revisión maestra sintética y aislada para probar el caso positivo.

### NO-GO de composición productiva (R-13)

Este corte entrega el adaptador y el almacén, pero no los compone en el
bootstrap productivo. R-13 sigue siendo una barrera técnica: falta el vínculo
autenticación-actor V2 y la acreditación transaccional del PDP V3 contra esta
única autoridad. No se permite réplica, dual-write ni outbox: introducirían
otra cardinalidad de huellas y dejarían de acreditar el mismo recibo. El
esquema de este registro y el del PDP deben residir en la misma base VEC para
construir ese puente transaccional; si no es posible, la composición permanece
NO-GO. `VinculoAutenticacionActorV1` conserva la huella histórica V1 y rechaza
el contexto V2, por lo que no es un puente válido. También siguen pendientes la
fuente maestra gobernada, su reconciliación, rectificación, deduplicación y
trazabilidad. Una fuente futura deberá materializar cada versión con autoridad
`autoridad_maestra_acreditada` y procedencia exacta
`(referencia, versión, huella, autoridad)`.

## Pruebas

El runner crea un PostgreSQL 18 efímero, aplica bootstrap/migración, acredita
un LOGIN runtime y prueba bytes canónicos con Go, ACL, cardinalidad cero y
ambigua, revocación, expiración durante una espera de locks, colisión
idempotente, rechazo productivo de `no_autoritativa`, adulteraciones de bytes,
huella, autoridad y versiones del manifiesto, límites `uint64`, un único
reintento, carrera real entre reconciliación y COMMIT, privilegios efectivos
hostiles por `PUBLIC`, `MAINTAIN`, ACL de parámetros, membresía adicional, down
con opt-in y typed nil/unitarias:

```sh
deploy/postgresql/contexto_actor_v1/probar_integracion.sh
```

La imagen puede sustituirse mediante `VEC_POSTGRES_TEST_IMAGE`; por defecto se
usa una referencia PostgreSQL 18 fijada por digest.

El down es destructivo y exige:

```sh
psql -X -v ON_ERROR_STOP=1 \
  -v confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
  -f deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.down.sql
psql -X -v ON_ERROR_STOP=1 \
  -v confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
  -f deploy/postgresql/contexto_actor_v1/roles_down.sql
```

Debe retirarse antes la membresía del LOGIN de aplicación; `DROP ROLE` falla
cerrado si aún existen dependencias o membresías. El down solo elimina objetos
y grants del módulo: no intenta restaurar las ACL globales que eran
precondición del despliegue.
