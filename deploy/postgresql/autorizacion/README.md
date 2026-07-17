# PostgreSQL del nucleo de autorizacion

Estado: adaptadores durables V1/V2, proyeccion de motivos y registro nominal de
concesiones ligadas a solicitud V2 implementados y probados, **no habilitados
en la composicion productiva**. Fecha de corte: 17 de julio de 2026.

Implementa `ports.FuenteAutorizacion`,
`ports.RegistroDecisionesAutorizacion`,
`ports.RegistroDecisionesAutorizacionSolicitudLigadaV2` y la consulta
historica de `ports.ValidadorReferenciaMotivoAutorizacionV2` con `pgx` v5.10.0.
No conecta HTTP, CLI, MCP ni modulos de negocio y la aplicacion no ejecuta
migraciones al arrancar.

## Contenido

- `roles_up.sql`: bootstrap DBA de una sola ejecucion para grupos tecnicos
  `NOLOGIN`, sin contrasenas.
- `roles_v2_up.sql` y `roles_v2_down.sql`: evolucion DBA segregada para los
  grupos `NOLOGIN` de proyeccion y evaluacion historica. No los hace miembros
  del propietario y no modifica los roles V1; la retirada inventaria sus
  atributos, ajustes, ACL y los tres campos de todas las membresias.
- `migraciones/000001_autorizacion.up.sql`: esquema, invariantes, RLS,
  funciones cerradas y privilegios.
- `migraciones/000001_autorizacion.down.sql`: reversion destructiva del
  esquema, solo con copia y aprobacion.
- `../ejecucion_documental_v4/migraciones_autorizacion/000002_*`: evolucion
  del documento de decision de 30 a 31 claves, validacion cerrada del bloque
  autenticacion-actor y fuentes locales versionadas para revalidarlo.
- `migraciones/000003_proyeccion_motivos_autorizacion_v2.*.sql`: cabeceras
  publicadas inmutables, entradas temporales, retiradas append-only, eventos y
  checkpoint monotono del catalogo maestro de motivos V2.
- `migraciones/000004_registro_decisiones_solicitud_ligada_v2.*.sql`: registro
  V2 separado de V1, documentos semanticos cerrados de decision y motivo, CAS
  de identidad, rol, politicas y motivo actual, RLS, ACL runtime cerrada y
  conservacion obligatoria del historico.
- `roles_down.sql`: retirada final gobernada de grupos V1; inventaria roles y
  los tres campos OID de cada membresia antes de mutar, y falla ante cualquier
  relacion, atributo o dependencia inesperados.
- `probar_integracion.sh`: PostgreSQL efimero aislado, migracion ascendente,
  pruebas SQL y del adaptador Go con identidades reales separadas, migracion
  descendente y retirada de roles.

La prueba usa por defecto PostgreSQL 18.4 Bookworm con el indice OCI fijado a
`sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296`.
No busca, para ni modifica contenedores preexistentes: crea un nombre unico y
solo elimina ese contenedor mediante `trap`.

```bash
deploy/postgresql/autorizacion/probar_integracion.sh
```

También se pueden ejecutar las pruebas contra una instancia preparada:

```bash
export VEC_POSTGRES_TEST_FUENTE_DSN='postgresql://fuente:...@host/base?sslmode=verify-full'
export VEC_POSTGRES_TEST_REGISTRO_DSN='postgresql://registro:...@host/base?sslmode=verify-full'
export VEC_POSTGRES_TEST_ADMIN_DSN='postgresql://administrador_pruebas:...@host/base?sslmode=verify-full'
go test ./internal/vec/adapters/postgres -run TestIntegracionAutorizacionPostgreSQL -count=1
```

La consulta historica V2 usa tres variables distintas para demostrar tanto la
capacidad positiva del evaluador como la ausencia de esa capacidad en el
proyector y en la fuente V1:

```bash
export VEC_POSTGRES_TEST_MOTIVOS_EVALUADOR_DSN='postgresql://evaluador:...@host/base?sslmode=verify-full'
export VEC_POSTGRES_TEST_MOTIVOS_PROYECTOR_DSN='postgresql://proyector:...@host/base?sslmode=verify-full'
export VEC_POSTGRES_TEST_MOTIVOS_FUENTE_V1_DSN='postgresql://fuente_v1:...@host/base?sslmode=verify-full'
go test ./internal/vec/adapters/postgres -run '^TestIntegracionMotivosAutorizacionV2PostgreSQL$' -count=1
```

El registro V2 exige cuatro identidades de prueba separadas. El runner crea
una base efimera exclusiva, comprueba primero la reversion vacia y, tras
registrar evidencia, demuestra que la migracion descendente se niega a
borrarla:

```bash
export VEC_POSTGRES_TEST_REGISTRO_V2_FUENTE_DSN='postgresql://fuente:...@host/base?sslmode=verify-full'
export VEC_POSTGRES_TEST_REGISTRO_V2_REGISTRO_DSN='postgresql://registro:...@host/base?sslmode=verify-full'
export VEC_POSTGRES_TEST_REGISTRO_V2_EVALUADOR_DSN='postgresql://evaluador:...@host/base?sslmode=verify-full'
export VEC_POSTGRES_TEST_REGISTRO_V2_ADMIN_DSN='postgresql://administrador_pruebas:...@host/base?sslmode=verify-full'
go test ./internal/vec/adapters/postgres -run '^TestIntegracionRegistroDecisionSolicitudLigadaV2PostgreSQL$' -count=1
```

Esas variables solo existen para pruebas. No deben imprimirse, guardarse en el
repositorio ni reutilizarse en produccion. Fuera del runner, cada prueba se
omite limpiamente si no esta completo su propio grupo de variables.

## Aplicacion de migraciones

Se presupone una base dedicada. Un DBA ejecuta primero `roles_up.sql` y, antes
de aplicar `000003`, `roles_v2_up.sql`. Las migraciones se aplican por orden
`000001`, `000002`, `000003` y `000004`. Cada bootstrap toma un bloqueo
transaccional para serializar ejecuciones concurrentes y, antes de cualquier
mutacion, aborta si ya existe uno solo de sus nombres reservados. No adopta ni
corrige roles homonimos aunque sus atributos
parezcan validos: podrian conservar credenciales, ajustes, membresias,
privilegios o propiedad ajenos que un bootstrap no debe asumir ni revocar.
Antes de esas comprobaciones exige que `current_user` sea superusuario. Ser
propietario de la base y disponer de `CREATEROLE` no basta; el runner lo prueba
con una cuenta `LOGIN` real y demuestra que el rechazo no crea roles ni cambia
las ACL de base o del esquema publico, tanto en V1 como en V2.

Por diseño no es idempotente: una segunda ejecucion falla. La comprobacion de
una instalacion existente debe hacerse con una auditoria separada y de solo
lectura. Una reconstruccion requiere la migracion descendente, retirar primero
las cuentas `LOGIN` y dependencias bajo procedimiento aprobado, ejecutar
`roles_down.sql` y solo entonces aplicar de nuevo el bootstrap. Este limite
evita tocar objetos extranjeros, incluso privilegios o propiedad situados en
otras bases del mismo cluster.

`roles_down.sql` y `roles_v2_down.sql` bloquean, en este orden, `pg_authid`,
`pg_auth_members` y `pg_database` en modo `ACCESS EXCLUSIVE`, siempre despues
de su bloqueo advisory y antes de comprobar inventario. Los dos primeros
impiden que un `GRANT ROLE` concurrente conserve OID que van a desaparecer; el
tercero estabiliza propietario, otorgante y ACL de la base entre preflight y
revocacion. El runner abre el `GRANT` desde otra base y usa tres barreras
advisory de prueba, no ventanas temporizadas: un observador ya conectado
acredita que el PID del down posee exactamente ambos bloqueos de roles, espera
`pg_database` y bloquea directamente al `GRANT`. Ninguna sesion se cancela; el
`GRANT` falla de forma natural despues del `DROP ROLE`. Finalmente se revisan
`roleid`, `member` y `grantor` por los OID retirados y la ausencia de membresias
huerfanas en todo el cluster.

V1 fija bajo esos bloqueos los cuatro OID y los atributos exactos creados por
el bootstrap. Solo admite la relacion estructural propietario-migrador, con
`ADMIN FALSE`, `INHERIT FALSE`, `SET TRUE` y como otorgante el superusuario
bootstrap gobernado del cluster (OID interno estable 10, aunque se renombre).
PostgreSQL atribuye a ese principal las concesiones emitidas por cualquier
superusuario; comparar sin mas con `current_user` impediria una retirada
legitima hecha por otro DBA nominativo. Cualquier otra relacion en la que un
rol V1 sea grupo, miembro **u otorgante** aborta antes de revocar ACL o
membresias. Ademas, V1 exige la ACL completa producida por `roles_up`: todos
los privilegios del propietario real de la base, `CONNECT` y `CREATE` para el
propietario VEC, y solo `CONNECT` para los otros tres grupos; el otorgante de
cada entrada debe ser el propietario real de la base. Una concesion adicional,
como `TEMPORARY`, aborta antes de ejecutar `REVOKE`.

V2 no posee una arista estructural y exige inventario de membresias vacio en
las tres coordenadas. Tambien exige roles `NOLOGIN` sin contrasena, caducidad ni
ajustes por rol/base y una unica ACL `CONNECT` por grupo, otorgada por el
propietario de la base; al inventariar la ACL considera al rol tanto como
destinatario como otorgante. El runner construye incluso una concesion de rol
cuyo `grantor` es el propio proyector V2 y prueba que se conserva al abortar.

Estos bloqueos de catalogos compartidos requieren superusuario, afectan al
cluster completo y solo se ejecutan en una ventana de mantenimiento que impida
altas o cambios de roles no coordinados.

Después, una identidad `LOGIN` nominativa y temporal, miembro de
`vec_autorizacion_migrador`, aplica las migraciones por orden. La aplicacion no
recibe esa identidad.

Las identidades de ejecucion son cuentas `LOGIN` distintas por despliegue y
solo heredan lo necesario:

- `vec_autorizacion_fuente`: ejecutar la lectura de instantanea exacta;
- `vec_autorizacion_registro`: ejecutar exclusivamente el CAS nominal V1; el
  CAS V2 permanece sin `EXECUTE` runtime hasta completar su puerta COSE;
- `vec_autorizacion_motivos_proyector`: proyectar una publicacion o retirada
  gobernada, sin lectura directa de tablas;
- `vec_autorizacion_motivos_evaluador`: resolver exclusivamente la referencia
  historica en el instante que evalua el PDP, sin proyectar ni usar la barrera
  actual.

Ninguna recibe `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `TRUNCATE` ni propiedad
sobre tablas. No son superusuario, no crean roles o bases, no replican y no
tienen `BYPASSRLS`. `PUBLIC` carece de uso del esquema y de ejecucion de sus
funciones; los privilegios predeterminados tambien quedan revocados.
El PDP no dispone de una funcion de lectura de decisiones. La futura lectura o
consumo pertenece al repositorio del efecto, con una operacion definidora
exacta, atestacion y consumo atomico; no al rol que registra.

RLS esta habilitada y forzada en todas las tablas. Solo existe una politica
positiva para el propietario `NOLOGIN`; las funciones `SECURITY DEFINER`
poseen `search_path` cerrado y exponen una operacion parametrizada cada una.
Versiones, controles y decisiones son append-only; los punteros solo avanzan y
el cambio de politicas exige actualizar el control de catalogo en la misma
transaccion.

## Proyeccion de motivos V2

La proyeccion solo conserva coordenadas opacas y probatorias: identificador y
version de catalogo, huella completa de la instantanea publicada, clave
`motivo_` seguida de 32 hexadecimales, intervalo `[desde,hasta)`, referencia y
huella del evento de origen. No copia etiquetas, descripciones, actores,
expedientes ni otros datos personales del catalogo maestro.

`publicar_motivos_autorizacion_v2` y
`retirar_motivos_autorizacion_v2` serializan el evento de origen mediante un
checkpoint bloqueado, exigen secuencias contiguas y solo consideran idempotente
un replay cuando evento y datos coinciden exactamente. Un evento o secuencia
reutilizados con otra carga fallan cerrados. La retirada se inserta aparte: no
reescribe ni recalcula la huella publicada que ya forma parte de decisiones.
La base comprueba forma, unicidad, secuencia y coincidencia de las huellas que
recibe, pero no puede reconstruir la huella completa del catalogo porque esta
proyeccion minimizada omite deliberadamente sus textos y gobierno. La
autenticidad del evento pertenece al repositorio maestro y a la identidad
exclusiva del proyector; no se presenta este SHA-256 aportado como firma.

La carga minimizada de entradas admite como maximo 10.000 elementos y 16 MiB,
igual que el catalogo de dominio. El tamaño se rechaza antes de enumerar claves
y los duplicados se detectan por agrupacion SQL; no se usa una busqueda lineal
creciente que convierta el limite alto en trabajo O(n²).
Todos los instantes se limitan a valores finitos de los años 1 a 9999. Los
intervalos JSON exigen UTC con seis decimales y un round-trip textual exacto;
formas que PostgreSQL normalizaria, como `24:00:00` o un segundo `60`, se
rechazan antes de publicar y no pueden divergir durante un replay.

Si el catalogo maestro esta en la misma base, su repositorio debe invocar la
funcion de proyeccion dentro de la **misma transaccion** que confirma la
publicacion o retirada. No existe un outbox asincrono que pueda presentarse como
barrera. Si el maestro reside en otra base no hay atomicidad distribuida: ese
despliegue permanece cerrado hasta aplicar invalidacion previa en dos fases o
un arrendamiento corto que deniegue al caducar.

Hay dos semanticas de resolucion deliberadamente distintas:

- `resolver_motivo_autorizacion_v2_historico` responde como era el catalogo en
  el instante de evaluacion, no bloquea retiradas y es la unica operacion del
  rol evaluador;
- `resolver_motivo_autorizacion_v2_actual` usa siempre `clock_timestamp()` de
  PostgreSQL y mantiene `FOR SHARE` sobre el checkpoint hasta el `COMMIT` del
  llamador. Es un helper privado, sin `EXECUTE` para roles runtime. El registro
  V2 toma directamente el mismo checkpoint y coteja el motivo con su unico
  instante final; los efectos deberan repetir esa barrera en su propia
  transaccion atomica. La retirada necesita `FOR UPDATE`, por lo que no puede
  adelantarse entre la barrera y el registro o efecto confirmado.

`ValidadorReferenciaMotivoPostgreSQLV2` solo invoca la primera funcion mediante
una consulta parametrizada y recibe un `pgxpool.Pool` ya administrado por la
composicion. No abre conexiones, no conserva el DSN y no conoce la funcion
actual ni las operaciones de proyeccion. Su pool debe pertenecer a una LOGIN
evaluadora exclusiva: el runner confirma sobre una publicacion real que esa
identidad resuelve la referencia exacta, rechaza huella, clave e instante no
coincidentes y no puede proyectar ni ejecutar la barrera actual. Tambien
confirma que las identidades proyectora y fuente V1 no pueden usar el adaptador.

El PDP usa la variante historica con el instante autoritativo de evaluacion que
despues queda comprometido como `emitida_en`. El registro V2 vuelve a comprobar
la referencia completa y, para una concesion, bloquea el checkpoint actual
dentro de su transaccion serializable. Una denegacion no produce efecto y solo
necesita conservar la prueba historica. `emitida_en`, una fecha aportada por
cliente o la consulta historica nunca sustituyen la barrera actual al conceder.
Registrar en una transaccion y confirmar el efecto en otra tampoco basta: el
repositorio del efecto debe revalidar y consumir de nuevo de forma atomica.

Las cinco tablas tienen RLS habilitada y forzada, una unica politica positiva
para el propietario exacto y cero ACL directas para proyector o evaluador. Las
funciones publicas son `SECURITY DEFINER` con `search_path` cerrado. El `down`
no usa `CASCADE` y se niega a continuar si existe una sola publicacion,
entrada, retirada, evento o avance del checkpoint. Antes de comprobar toma
`ACCESS EXCLUSIVE` sobre las cinco tablas, empezando por el checkpoint como los
flujos runtime: espera escritores previos y ve su confirmacion, mientras que
impide que un escritor nuevo atraviese el preflight.

## CAS implementado

`registrar_decision_si_vigente` bloquea y compara en una transaccion
serializable:

1. asignacion actual, principal, referencia y huella;
2. version y huella exactas del rol;
3. revision y huella del control de vigencia de esa version;
4. revision y huella del catalogo;
5. conjunto completo referencia/huella de politicas actuales;
6. forma canonica, listas, manifiestos, referencias, UTC microsegundo y
   vigencia contra `clock_timestamp()` de PostgreSQL.

La prueba aplica primero el contrato historico y despues la evolucion V2. Una
decision nueva incorpora exactamente el bloque `vinculo_autenticacion_actor`
de 25 campos, ligado a la persona, perfil, sesion, control, garantia y
ContextoActor. El registro comprueba su estructura y el disparador bloquea y
revalida los punteros locales actuales antes del `INSERT`; una revocacion
concurrente no puede atravesar esa transaccion.

El documento historico tiene una lista blanca cerrada de 30 claves y el actual
exactamente 31: las anteriores mas `vinculo_autenticacion_actor`. Todas son
obligatorias, aunque una lista o mapa sea vacio; no se admiten claves
adicionales. Booleanos, numeros, textos, arrays y objetos conservan su tipo JSON
exacto, por lo que valores como `"concedida":"yes"` no se convierten. Los
manifiestos solo contienen valores string SHA-256 y las revisiones son enteros
positivos del rango `uint64`.

`pg_column_size` limita cada decision a 512 KiB. Es un techo deliberadamente
amplio para una capacidad breve con hasta 512 entradas de catalogo, campos u
obligaciones habituales, pero impide usar el registro como almacen documental o
forzar filas TOAST patologicas. Si una instalacion alcanza el limite debe acortar
referencias o segmentar su catalogo; nunca elevarlo implicitamente desde datos de
entrada.

Cualquier ausencia, cambio, espera conflictiva o error falla cerrado. El
adaptador nunca devuelve SQL, DSN, nombres internos ni contenido personal en
sus errores. La tabla durable solo admite decisiones concedidas con codigo
`concedida`; el registro probatorio de denegaciones debe incorporarse como
puerto de auditoria separado y append-only.

La barrera durable rechaza cualquier asterisco, completo o parcial, en RBAC,
campos, obligaciones, finalidades y ambitos positivos. No existe
`global=["*"]`. Las politicas ABAC pueden conservar el comodin restrictivo
exacto porque solo reducen acceso; nunca conceden.

## Registro nominal V2 implementado

`registrar_decision_solicitud_ligada_v2_si_vigente(bytea,bytea)` recibe
los bytes que el adaptador Go produce de forma determinista para la decision V2
y la referencia opaca de motivo. PostgreSQL limita y conserva esos bytes, pero
**no demuestra su canon lexico**: `jsonb` normaliza espacios, orden, escapes y
claves duplicadas antes de que las funciones puedan inspeccionarlos. La barrera
actual demuestra estructura y semantica cerradas, no unicidad de preimagen.
Rechaza V1, claves semanticas desconocidas, correlaciones no opacas, listas o
manifiestos no ordenados o repetidos, huellas nulas y motivos cuya preimagen
recibida no coincida con el compromiso de la decision. El orden de listas y
manifiestos usa explicitamente collation `C`, equivalente al orden por bytes
UTF-8 de Go para este perfil ASCII.

La funcion materializa una copia comun solo para aplicar las invariantes ya
probadas de identidad y RBAC/ABAC. Esa copia no se inserta en
`decision_autorizacion`: la fila vive en
`decision_autorizacion_solicitud_ligada_v2`, con sus bytes y huellas propios.
Por tanto, una decision V2 no puede reaparecer por una consulta o funcion V1 ni
perder silenciosamente `solicitud_huella_sha256` o `motivo_huella_sha256`.

Antes del `INSERT`, la misma transaccion serializable bloquea y coteja:

1. asignacion, principal, perfil y huella actuales;
2. version, control y huellas del rol;
3. revision, huella y conjunto completo de politicas;
4. sesion, control de sesion y ContextoActor actuales;
5. checkpoint, publicacion, version, huella, entrada y ausencia de retirada del
   motivo;
6. un unico `clock_timestamp()` tomado despues de adquirir todos los locks y
   aplicado a decision, sesion, actor y motivo antes del `INSERT` inmediato.

La migracion no concede `EXECUTE` a ninguna identidad runtime. Fuente, registro,
evaluador, proyector, `PUBLIC` e identidades de negocio quedan fuera; tampoco
poseen privilegios de tabla. El runner abre `EXECUTE` a
`vec_autorizacion_registro` solo en su base efimera, ejecuta la integracion, lo
revoca y verifica el cierre de las cinco identidades antes de continuar. La
tabla es append-only, tiene RLS forzada y la migracion descendente toma un
bloqueo exclusivo y se niega a borrar una sola decision durable.

La integracion real demuestra preservacion de los bytes emitidos por Go,
separacion V1/V2, entradas hostiles, ACL, retirada entre evaluacion y CAS y
expiracion del motivo mientras el registro espera un lock. Cuando se invoque
tras abrir su puerta, este registro acreditara frescura al insertar, pero no
concedera un efecto: faltan la capacidad efimera autenticada, la revalidacion y
el consumo unico dentro de la transaccion final de Bolsa.

El registro es **at-most-once, no idempotente**. `decision_ref` y la huella de
los bytes son unicos: dos intentos concurrentes no crean dos filas. Un replay
inmediato alcanza la unicidad y devuelve version existente; si antes cambian o
expiran identidad, rol, politicas, motivo o decision, puede fallar cerrado como
instantanea obsoleta antes de consultar esa unicidad. No equivale a consumo
unico del efecto ni recupera automaticamente un `COMMIT` de resultado ambiguo.

## Puerta productiva que sigue cerrada

Los CAS demuestran frescura de la instantanea, pero no vuelven a ejecutar el
PDP. Eso es deliberado: duplicar RBAC/ABAC en SQL crearia dos semanticas y
romperia la portabilidad hexagonal. V1 solo conserva la huella del contexto;
V2 añade los compromisos exactos de solicitud y motivo, pero PostgreSQL no
puede deducir de esos SHA-256 que el PDP sea realmente su emisor.

Conceder ahora la funcion a `vec_autorizacion_registro` permitiria a una
credencial robada crear un documento nuevo coherente con la configuracion
vigente y calcular sus huellas. No podria convertirlo en V1, pero el registro no
autenticaria su procedencia. Por eso la migracion deja esa ACL cerrada.

Eliminar la funcion general de lectura evita que ese rol recupere decisiones
ajenas, pero no autentica por si solo la procedencia del documento recibido.

Hasta resolverlo se aplican estas prohibiciones:

- ningun despliegue concede `EXECUTE` runtime sobre el registro V2;
- el adaptador no se monta en composicion productiva;
- una decision registrada no autoriza por si sola ningun efecto de negocio.

Antes de abrir la puerta se exige consumir una atestacion criptografica
versionada de la decision completa, emitida con credencial y clave separadas y
verificada sobre los **bytes exactos antes de convertir a `jsonb`**. El parser
de frontera debe detectar claves duplicadas y rechazar whitespace, orden,
escapes, numeros, fechas o cualquier otra representacion alternativa al canon
publicado; verificar solo el arbol `jsonb` no basta. El perfil VEC-AD-2 y su
verificador COSE ya existen en Go; faltan su catalogo durable y la funcion final
de consumo. No se presupone que `pgcrypto` verifique Ed25519 ni se aplica un
HMAC alternativo silencioso. Suite, representacion canonica, gobierno de claves,
rotacion, revocacion y pruebas se especifican en
`docs/portal_vec/atestacion_criptografica_decisiones.md`.

## Punto de extension para consumo atomico

La segunda condicion pendiente es consumir/revalidar la decision dentro de la
misma transaccion que produce el efecto. El punto de extension no esta en HTTP
ni en este adaptador aislado, sino en cada repositorio PostgreSQL de negocio:

```text
BEGIN
  fijar contexto tecnico local
  verificar atestacion versionada
  bloquear decision por decision_ref
  comparar tupla funcional exacta y clock_timestamp() < valida_hasta
  revalidar asignacion + control de rol + catalogo completo
  insertar consumo unico decision_ref + operacion_ref + idempotencia
  escribir agregado + OCC + hecho + auditoria + outbox
COMMIT
```

La futura funcion definidora puede llamarse
`consumir_decision_para_efecto(decision_ref, operacion_ref, contexto, sello)`.
Debe devolver un resultado tipado, no detalles SQL, y afectar exactamente una
fila. Una decision mutadora es de un solo uso; la unicidad se impone en base.
Si el caso de lectura admite reutilizacion, necesita una politica explicita y
un registro append-only por cada uso, nunca un supuesto implicito.

El puerto/repositorio de negocio debe recibir la decision y el comando
completo para que no exista ventana entre comprobar y confirmar. No es seguro
llamar primero a este CAS, cerrar su transaccion y ejecutar despues el efecto.

## Fuentes y mantenimiento

- pgx v5.10.0, version estable comprobada el 15-07-2026:
  <https://pkg.go.dev/github.com/jackc/pgx/v5@v5.10.0>.
- PostgreSQL 18.4 corrige vulnerabilidades de versiones anteriores:
  <https://www.postgresql.org/about/news/postgresql-184-1710-1614-1518-and-1423-released-3297/>.

El resumen de imagen y la version de pgx se revisan con cada actualizacion de
seguridad; fijarlos hace reproducible una entrega, no autoriza a congelarlos.
