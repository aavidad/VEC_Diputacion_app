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
psql -X -v ON_ERROR_STOP=1 -f deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql
psql -X -v ON_ERROR_STOP=1 -f deploy/postgresql/contexto_actor_v1/migraciones/000003_organizacion_corporativa_v1.up.sql
```

### Acreditación cerrada de uso del recibo V2

La migración aditiva `000002` añade la función cerrada
`acreditar_uso_registro_contexto_actor_v2(...)` y la infraestructura privada
de generación y triggers que acredita su lectura concurrente. La función recibe el subconjunto de
referencias, versiones y huellas comprometido por un consumidor, busca el
`rca_` exacto y reconstruye desde las filas actuales los bytes canónicos V2 y
el manifiesto de procedencia. La igualdad es byte a byte: JSON equivalente,
una huella autoconsistente sobre otros bytes, una versión avanzada o cualquier
estado no vigente devuelven `NULL`. El único resultado positivo es el
`clock_timestamp()` autoritativo tomado después de todos los locks. La función
exige una transacción `SERIALIZABLE` de escritura.

La migración instala una fila global de generación, cerrada al runtime, y dos
triggers de sentencia sobre cada una de las cinco tablas `*_actual`. El trigger
`BEFORE` toma un advisory exclusivo global para fijar un orden libre de
interbloqueos; el trigger `AFTER` avanza la fila MVCC en cada
`INSERT`/`UPDATE`/`DELETE`. Un tercer trigger rechaza `TRUNCATE` sin tomar el
advisory; mantenerlo separado evita el ciclo entre el `AccessExclusive` que
PostgreSQL adquiere antes del trigger y una acreditación concurrente. La acreditación toma el advisory compartido,
bloquea y relee cuenta, perfil, vínculos de contexto, persona y vínculos de
módulo, y solo entonces lee la generación `FOR SHARE`. El advisory ordena;
no constituye la prueba de frescura. Si una mutación comprometió después del
snapshot `SERIALIZABLE`, la versión nueva de la generación es invisible y
PostgreSQL fuerza `40001`. Si la acreditación obtuvo primero el lock, la
mutación espera y queda serializada después de su `COMMIT`. El reloj posterior
a la espera cierra también la expiración concurrente. El coste deliberado es
serializar globalmente los cambios de punteros mientras se acredita su uso.

La función es `SECURITY DEFINER`, pertenece exclusivamente a
`vec_contexto_actor_v1_propietario`, fija `search_path = pg_catalog` y no
expone tablas ni tipos fila. `PUBLIC` y el runtime de contexto no pueden
ejecutarla. Esta migración aislada no crea, adopta ni concede privilegios a
roles de autorización. La concesión futura debe realizarse nominalmente desde
la migración de composición, solo en la misma base, con `USAGE` del esquema y
`EXECUTE` sobre esta firma exacta. Separar contexto y autorización en bases
distintas sigue siendo NO-GO.

Un registrador consumidor debe llamarla antes de tomar sus propios locks y
repetir la llamada después de tomarlos, usando el segundo instante para todas
sus ventanas. La primera llamada conserva los locks de contexto hasta el final
de la transacción; la segunda ya no espera y detecta una expiración ocurrida
mientras se adquirían locks del consumidor.

### Organización corporativa versionada C2.2-A

La migración `000003` incorpora la historia de solo adición y el puntero
actual de organizaciones opacas `org_`. Cada versión conserva procedencia,
estado y vigencia; no contiene denominaciones, CIF, códigos visibles,
jerarquías ni otros datos semánticos. El puntero expresa la última versión
conocida y no concede permiso. Historia y puntero quedan bajo RLS forzada,
sin acceso funcional para runtime, selector, PDP o Contratación temporal.

El alta y la retirada toman primero la barrera compartida de acreditación y
después la barrera organizativa exclusiva. `000003` no publica ni revoca
organizaciones, no selecciona actores y no autoriza consultas RRHH. Su
instalación aislada tampoco cierra C2.2-A ni la composición productiva: aún
requiere el vínculo corporativo, selección, PDP, raíz, TLS viva y E2E.

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

Este corte entrega el adaptador, el almacén y la función cerrada que acredita
un `rca_`, pero no los compone en el bootstrap productivo. R-13 sigue siendo
una barrera técnica: el vínculo autenticación-actor V2 ya está cerrado como
capacidad Go, pero faltan la decisión PDP V3, su wrapper transaccional y la
concesión nominal que lo conecte con esta única autoridad. No se permite
réplica, dual-write ni outbox: introducirían
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
hostiles por `PUBLIC`, `MAINTAIN`, ACL de parámetros, membresía adicional, la
acreditación exacta de `rca_`, aislamiento, solo lectura, bytes forjados,
versiones, vigencia, generación MVCC, inserción fantasma y revocación
concurrente, down aditivo
con opt-in y typed nil/unitarias:

```sh
deploy/postgresql/contexto_actor_v1/probar_integracion.sh
```

`VEC_CONTEXTO_ACTOR_OMITIR_GO=1` ejecuta únicamente la batería PostgreSQL del
módulo; sirve para validar migraciones y concurrencia cuando el árbol Go está
siendo evolucionado por separado. El valor por defecto continúa ejecutando
también todas las pruebas Go existentes.

La imagen puede sustituirse mediante `VEC_POSTGRES_TEST_IMAGE`; por defecto se
usa una referencia PostgreSQL 18 fijada por digest.

La matriz autónoma de organización corporativa se puede reproducir con:

```sh
deploy/postgresql/contexto_actor_v1/\
probar_organizacion_corporativa_v1_pg18_4.sh
```

`probar_integracion.sh` la invoca directa y exactamente una vez, como última
matriz focal después de sus pruebas propias. El runner acredita la instalación,
estructura, ACL, RLS, límites, historia, generación, retirada, carreras y la
ejecución de los bytes completos del `down` mediante una conexión `pgx`
dedicada, incluida su cancelación y saneamiento.

La retirada base está denegada por defecto. `000001_contexto_actor_v1.down.sql`
solo acepta una instalación exacta de `000001`, completamente vacía y sin
consumidores. Reacredita propietario, roles, membresías, ACL, objetos,
propiedades y ACL de columna, índices, funciones, restricciones, todos los
triggers —incluidos los internos—, tipos, publicaciones, herencias,
suscripciones y privilegios predeterminados mediante un manifiesto PostgreSQL
18.4. También deniega publicaciones `FOR ALL TABLES`, ajustes de los roles o
de todos los roles por base e inventaría `pg_shdepend` con direcciones
semánticas, incluida la base por nombre y sin fijar OID. El contrato de roles
incluye límite de conexiones, caducidad y solo el estado ausente/presente de
la credencial, nunca su valor. Cualquier fila, `000002`, migración posterior,
objeto desconocido, deriva o dependencia externa aborta la transacción y
conserva íntegro el esquema.

La confirmación explícita no autoriza a borrar evidencia. La secuencia
operativa, únicamente para una instalación vacía acreditada, es:

```sh
PGOPTIONS="-c vec.confirmar_retirada_organizacion_corporativa_v1=RETIRAR_ORGANIZACION_CORPORATIVA_V1" \
psql -X -v ON_ERROR_STOP=1 \
  -f deploy/postgresql/contexto_actor_v1/migraciones/000003_organizacion_corporativa_v1.down.sql
PGOPTIONS="-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2" \
psql -X -v ON_ERROR_STOP=1 \
  -f deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql
psql -X -v ON_ERROR_STOP=1 \
  -v confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
  -f deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.down.sql
psql -X -v ON_ERROR_STOP=1 \
  -v confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
  -f deploy/postgresql/contexto_actor_v1/roles_down.sql
```

La retirada de `000002` es una ventana de mantenimiento exclusiva. Después
de inmovilizar la tabla de control y las doce relaciones base, conserva
bloqueos `SHARE` sobre los catálogos que podrían cambiar funciones, tipos,
ACL, comentarios, dependencias, estadísticas extendidas, publicaciones o
suscripciones hasta el `COMMIT`.
Debe ejecutarse sin tráfico de migración; los límites de bloqueo y sentencia
hacen que cualquier contención falle cerrada.

#### Frontera operativa de la retirada

El contrato cubre el DDL, las ACL, los comentarios y las dependencias creados
o modificados mediante las operaciones soportadas por PostgreSQL 18. La
inmovilización de las trece relaciones y de los catálogos acreditados impide
que esos cambios atraviesen la ventana situada entre el inventario y los
`DROP`.

Un DML directo de superusuario sobre `pg_catalog` equivale a un compromiso
total de la base y queda fuera de este contrato. Existen defensas puntuales
para estados hostiles reproducidos por las pruebas, pero el manifiesto no se
presenta como un canon universal capaz de reconocer cualquier escritura
arbitraria sobre catálogos internos.

La contención de otro superusuario no abre una vía permisiva: si no se pueden
obtener los bloqueos dentro de los límites operativos, la retirada aborta y
PostgreSQL revierte la transacción completa. No se autoriza continuar, omitir
un objeto ni aceptar una huella parcial.

Debe retirarse antes la membresía del LOGIN de aplicación; `DROP ROLE` falla
cerrado si aún existen dependencias o membresías. `roles_down.sql` es
autónomo, transaccional y denegado por defecto: vuelve a comprobar atributos,
credencial sin leer su valor, comentarios, etiquetas, ajustes globales y por
base, membresías, propietarios, ACL y dependencias. Mantiene bloqueados los
catálogos compartidos desde esa comprobación hasta los `DROP ROLE`; así una
deriva nacida después del down de base no abre una carrera destructiva.

El down solo elimina objetos y grants enumerados del módulo con `RESTRICT`;
nunca usa `CASCADE` ni `DROP OWNED`. Restaura únicamente los valores nativos
de las dos ACL predeterminadas que creó `000001`, para que `roles_down.sql`
pueda retirar el propietario; no altera las ACL globales de base y `public`
que eran precondición del despliegue. La guarda
`vec_contexto_actor_v1:migracion:base:v1` queda reservada para que las
migraciones posteriores la tomen en modo compartido mientras crean o retiran
consumidores.

La matriz focal de retirada prueba confirmaciones, evidencia, `000002`,
objetos y ACL hostiles, publicaciones globales anteriores y posteriores,
ajustes por base, dependencias de rol dentro y fuera de la base, invocación
desnuda de `roles_down.sql`, propietario, manifiesto, reentrada, reinicio,
reconexión, carrera observable y
`up → down seguro → up` con OID nuevo:

```sh
deploy/postgresql/contexto_actor_v1/\
probar_retirada_segura_contexto_actor_v1_pg18_4.sh
```

La retirada de roles deriva la autoridad del otorgante histórico de la
membresía canónica, exige el mismo otorgante en las tres ACL `CONNECT` y no
depende del superusuario que ejecute el alta o la retirada. Su ciclo focal,
incluidos otorgantes divergentes, se reproduce con:

```sh
deploy/postgresql/contexto_actor_v1/\
probar_otorgantes_roles_contexto_actor_v1_pg18_4.sh
```
