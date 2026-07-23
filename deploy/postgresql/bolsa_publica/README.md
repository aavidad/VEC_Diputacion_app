# PostgreSQL: proyección pública de Bolsa

Este despliegue crea la fuente autoritativa que consume `cmd/vec-publico`. Se
instala en una base físicamente separada de RRHH. El proceso expuesto a
Internet no recibe credenciales de la base interna ni puede alcanzar personas,
candidaturas, expedientes, nóminas, auditoría interna o KMS.

La entrada del publicador es JSON acotado, pero el almacenamiento es relacional
y normalizado. Cada objeto y cada hijo tienen una lista exacta de claves; un
campo desconocido, un número no entero o un exceso de cardinalidad se rechazan
antes del DML. Las diez vistas de lectura enumeran columnas y nunca usan
`SELECT *`.

Esta frontera evita que una columna o clave personal nueva se publique por
accidente. No demuestra que un texto libre sea anónimo: un DNI escrito dentro
de `descripcion` seguiría siendo texto válido. El publicador interno debe
clasificar el dato, aplicar minimización y DLP antes de construir la proyección.
Debe invocar la función con parámetros enlazados y no registrar el JSON.

## Identidades y privilegios

`roles_up.sql` crea cinco roles técnicos `NOLOGIN` y una identidad operativa
dedicada:

- `vec_bolsa_publica_propietario`: propietario de tablas, vistas y del trigger
  de invalidación;
- `vec_bolsa_publica_publicacion_propietario`: propietario distinto de las
  funciones `SECURITY DEFINER`;
- `vec_bolsa_publica_migrador`: puede asumir exclusivamente los dos
  propietarios durante una migración;
- `vec_bolsa_publica_consulta`: `USAGE` del esquema de lectura y `SELECT` sobre
  las diez vistas allowlist;
- `vec_bolsa_publica_publicador`: `USAGE` del esquema de publicación y
  ninguna capacidad de escritura por sí solo;
- `vec_bolsa_publica_publicador_login`: único `LOGIN` que recibe `EXECUTE`
  directo sobre `publicar_proyeccion_v2(jsonb,text)`. Se crea con contraseña
  nula y no puede usarse hasta que el gestor de secretos la aprovisione.

Los dos propietarios son distintos, no pueden iniciar sesión y no se conceden
al publicador. La función propietaria puede insertar todas las tablas y borrar
solo las cuatro raíces necesarias para el reemplazo completo. No puede hacer
`SELECT`, `UPDATE` ni borrar el historial de anclas. El publicador no tiene DML
directo, no ejecuta la función auxiliar y no puede `SET ROLE`.

La cuenta lectora se crea fuera del repositorio, con secreto gestionado y una
sola membresía directa:

```sql
GRANT vec_bolsa_publica_consulta TO vec_publico_login
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;

ALTER ROLE vec_bolsa_publica_publicador_login
    PASSWORD '<secreto entregado sin registrarlo>';
```

Los valores `ALTER ROLE ... SET` de un grupo no deben tomarse como sustituto de
la configuración del `LOGIN`. El adaptador Go fija y comprueba en cada conexión
de lectura: solo lectura, `REPEATABLE READ`, `search_path` cerrado, zona UTC,
`statement_timeout=10s`, `lock_timeout=2s` e inactividad transaccional de 10 s.
El LOGIN publicador tiene, antes de recibir una sentencia,
`statement_timeout=60s`, `lock_timeout=5s`,
`idle_in_transaction_session_timeout=5s`, `transaction_timeout=2min`,
`search_path` cerrado, nombre de aplicación fijo y parámetros de error
redactados. La función y el adaptador cotejan tanto los valores efectivos como
`pg_roles.rolconfig`; cualquier deriva falla antes de tomar el candado.

No se concederá `EXECUTE` al grupo `NOLOGIN`, a otra cuenta ni a un rol
compartido. La función comprueba además el nombre exacto de `session_user`.
Así una membresía accidental del grupo no abre la escritura.

## Integridad y publicación

El manifiesto canónico vincula toda la proyección visible:

- revisión e instante de la fuente;
- catálogos genéricos y todas sus entradas;
- referencia exacta del catálogo profesional actual y todos los snapshots
  históricos necesarios, cada uno con sus huellas gobernada y de proyección;
- conjunto ordenado de identificadores de convocatorias;
- huella completa y huella propia de resumen de cada convocatoria.

La huella SHA-256 del manifiesto se entrega al proceso por una fuente externa,
nunca se aprende de PostgreSQL. En el arranque, el adaptador recorre la
proyección de forma acotada, recalcula el manifiesto y crea una caché inmutable
de huellas. Cada lectura abre una transacción `REPEATABLE READ, READ ONLY`,
compara el testigo de base con el externo y contrasta el material devuelto. No
toma un advisory lock: MVCC deja que una lectura iniciada antes del `COMMIT`
termine coherentemente sobre A y que la siguiente vea B. Los listados calculan
la huella de resumen; no reconstruyen todos los detalles para validar una
página.

El contrato profesional V2 separa `categorias.actual` de
`categorias.snapshots`. La referencia actual contiene ID, versión, huella
gobernada y huella de proyección. Cada convocatoria conserva esas mismas cuatro
piezas y su huella canónica las vincula. El directorio y las facetas usan solo
entradas vigentes del snapshot actual; listado y detalle pueden resolver una
convocatoria antigua contra su snapshot exacto aunque la categoría haya
caducado. Se admiten de 1 a 64 snapshots, de 1 a 1.024 entradas por snapshot y
4.096 entradas en total. El conjunto vigente actual puede estar vacío.

`publicar_proyeccion_v2` es la única frontera de escritura operativa. Valida el
JSON completo, toma el candado exclusivo y reemplaza datos y testigo en una
sola transacción. Todo DML fuera de esa función pone el testigo a 64 ceros; ese
valor está reservado y se rechaza como configuración o ancla. Los lectores
fallan cerrados desde la siguiente consulta.

`manifiesto_consumido` es append-only para la función y su clave es única. Una
ancla aceptada no puede volver a usarse, ni en A→B→A ni después de invalidar A a
cero. Operativamente, también se descarta el manifiesto de cualquier intento
fallido o dudoso y se genera uno nuevo, aunque la transacción rechazada no haya
dejado filas.

El advisory lock exclusivo de escritura es transaccional: permanece hasta
`COMMIT` o `ROLLBACK`. Publicadores y migraciones usan la misma clave; las
migraciones toman primero su candado de versión y después el de publicación.
Los lectores no compiten por ese lock. La única ruta operativa admitida es
`postgrespublico.PublicarProyeccion`: conexión de un solo uso, parámetros
enlazados y una única sentencia autocommit. No se permite envolverla en una
transacción del llamante ni publicar desde `psql`.

Los `SET` declarados en una función no arrancan el temporizador de la sentencia
que ya la está ejecutando y se restauran al retornar. Por eso los límites
obligatorios viven en el LOGIN antes de invocar. En PostgreSQL 18,
`transaction_timeout` acota la transacción completa y el timeout de inactividad
termina antes una sesión abandonada, aborta sus cambios y libera el advisory
lock.

Riesgo residual: quien comprometa las credenciales del LOGIN puede enviar otra
sentencia que cambie un GUC después de retornar la función dentro de una
transacción explícita. PL/pgSQL no puede gobernar el estado futuro de esa
transacción llamante. La mitigación es arquitectónica: secreto accesible solo
al publicador autocommit, ACL directa a esa identidad, sin consola/SQL genérico,
conexión de un solo uso y `transaction_timeout` de rol. Un acceso SQL arbitrario
con esas credenciales es un incidente de credenciales, no una modalidad
soportada.

### Runbook de publicador bloqueado

1. Detener el publicador y retirar el secreto; no copiar parámetros ni JSON a
   tickets, terminales compartidos o logs.
2. Desde una sesión DBA, localizar exclusivamente PID, estado, edad y nombre de
   aplicación en `pg_stat_activity`, y el candado advisory en `pg_locks`.
3. Si supera los límites anteriores, ejecutar `pg_terminate_backend(pid)`.
   Confirmar que desaparecen sesión y candado y que la revisión visible no
   cambió parcialmente.
4. Ejecutar en una transacción de diagnóstico
   `pg_try_advisory_xact_lock` para las claves de migración y publicación,
   hacer `ROLLBACK` y comprobar que ambas están libres.
5. Verificar `rolconfig`, membresía, ACL directa, TLS y versión PostgreSQL antes
   de reponer un secreto nuevo. Descartar el ancla del intento dudoso.
6. Reintentar mediante `PublicarProyeccion` con ancla nueva. Comprobar revisión
   y manifiesto desde un lector gobernado; después rehabilitar el servicio.

## Actualizaciones y disponibilidad

Publicar B sobre una base que servía A hace que los procesos configurados con A
fallen cerrados. No se cambia la variable de ancla en caliente. Para una primera
fase puede declararse una ventana de mantenimiento, publicar B y reiniciar el
proceso con su ancla externa.

El cambio sin parada requiere blue/green: segunda base de proyección, publicación
y validación de B, segundo proceso configurado con B, comprobación de salud y
cambio atómico del proxy. Después se drena A. No se reutiliza una misma base para
simular blue/green.

## Instalación

La base dedicada debe llegar sin privilegios de `PUBLIC` en la base ni `CREATE`
en el esquema `public`:

```sql
REVOKE CONNECT, TEMPORARY ON DATABASE vec_bolsa_publica FROM PUBLIC;
REVOKE CREATE ON DATABASE vec_bolsa_publica FROM PUBLIC;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
```

Orden de instalación:

1. `roles_up.sql` como DBA;
2. `migraciones/000001_proyeccion_publica.up.sql` con `current_user` igual a
   `vec_bolsa_publica_migrador`;
3. aprovisionamiento externo del LOGIN lector y asignación del secreto al
   `vec_bolsa_publica_publicador_login` ya creado, sin cambiar sus GUC, nombre
   ni ACL;
4. publicación interna de una proyección completa y de un manifiesto nuevo;
5. distribución de la huella externa al proceso público y arranque.

Configuración mínima de `cmd/vec-publico`:

```text
VEC_EXECUTION_PROFILE=produccion
VEC_AUTH_MODE=disabled
VEC_BOLSA_PUBLICA_DATABASE_URL=postgres://...?...sslmode=verify-full
VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256=<64 hex distintos de cero>
VEC_BOLSA_CATEGORIES_CATALOG_ID=categorias-profesionales
VEC_BOLSA_CATEGORIES_CATALOG_VERSION=<version actual explicita>
VEC_BOLSA_CATEGORIES_CATALOG_SHA256=<64 hex gobernados>
VEC_BOLSA_CATEGORIES_PUBLIC_PROJECTION_SHA256=<64 hex de la proyeccion>
```

El DSN se guarda en un tipo redactado. PostgreSQL debe verificar CA y nombre de
servidor (`verify-full`) con TLS 1.2 o posterior en el destino principal y todos
los fallbacks. La raíz productiva no importa ni puede seleccionar adaptadores de
fichero, memoria o presentación.

## Prueba reproducible

Con Docker y OpenSSL:

```bash
deploy/postgresql/bolsa_publica/probar_integracion.sh
```

El runner fija PostgreSQL 18.4 por digest y prueba TLS real, roles y propietarios
exactos, ausencia de DML y `SET ROLE`, contrato JSON recursivo, rechazo de PII en
claves desconocidas, límites, sentinel cero, historial A→B→A, arranque HTTP,
listado/facetas/detalle multiversión, categorías caducadas, doble huella exacta,
invalidación global, lectores A/B sin bloqueo, contención del pool, redacción
de errores y logs, terminación de un publicador que excede el timeout de
inactividad, liberación del candado, recuperación de publicación/migración,
serialización y reversión atómica. Certificados, credenciales y fixtures son
efímeros.

## Reversión

La migración `down` exige
`vec.confirmar_retirada_proyeccion_bolsa_publica` mientras quede cualquier dato,
incluidos fuente e historial. Primero se retiran los esquemas; después se
revoca y elimina externamente el LOGIN lector. `roles_down.sql` se niega a
borrar los grupos mientras conserven miembros ajenos y elimina por sí mismo el
LOGIN publicador dedicado.
