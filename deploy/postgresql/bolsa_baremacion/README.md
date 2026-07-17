# Persistencia PostgreSQL de baremaciones

Este paquete es el adaptador durable del puerto
`bolsa/ports.RepositorioBaremaciones`. Persiste una versión inmutable de la
baremación, su auditoría tipada y un único evento outbox en la misma transacción
serializable. No expone tablas a la aplicación.

## Garantías cerradas

- PostgreSQL 18.4 o posterior dentro de la misma versión mayor. La prueba fija
  por digest la imagen `postgres:18.4-bookworm`.
- Roles técnicos `NOLOGIN`, sin `SUPERUSER`, `BYPASSRLS`, creación de roles,
  bases ni replicación.
- RLS habilitado y forzado en todas las tablas; solo el propietario exacto
  puede verlas. Los runtimes reciben únicamente `USAGE` del esquema y
  `EXECUTE` de sus funciones cerradas.
- Los privilegios por defecto globales del propietario cierran `PUBLIC` para
  funciones y tipos. Una guarda DDL propiedad del DBA retira además `USAGE`
  sobre tipos fila implícitos, incluidos los creados por tablas futuras. Los
  runtimes no pueden invocar helpers ni usar tipos internos actuales o futuros.
- `SECURITY DEFINER` con `search_path = pg_catalog, pg_temp`; ningún nombre se
  resuelve desde esquemas controlables por el cliente.
- Aislamiento `SERIALIZABLE`, OCC exacto y bloqueo lógico por agregado,
  idempotencia o consumidor según la operación.
- Versiones, decisiones consumidas, reservas, tombstones, auditoría, eventos,
  intentos de entrega y cursores son append-only. Los punteros actuales solo
  avanzan mediante triggers monotónicos.
- El token de reserva y el token de arrendamiento nunca se almacenan en claro:
  se conserva únicamente SHA-256. Los tokens se entregan como parámetros
  binarios; el registro de parámetros del servidor y del proxy debe permanecer
  desactivado.
- Cada decisión de autorización solo puede producir un efecto exacto. Las
  lecturas sensibles también consumen una decisión y revalidan en vivo sesión,
  actor, asignación, rol, revisión y catálogo.
- Una versión, auditoría y evento se encadenan criptográficamente y se crean en
  el mismo `COMMIT`. El adaptador Go reconstruye y valida toda salida antes de
  confirmar; una respuesta incoherente provoca `ROLLBACK`.

La política es negar por defecto. No existen rutas de compatibilidad, estados
abiertos ni acceso directo de emergencia desde las cuentas de servicio.

## Dependencias e instalación

Las migraciones se aplican con una identidad DBA controlada. `roles_up.sql`
exige expresamente un superusuario: `CREATEROLE`, incluso siendo propietario
de la base, no constituye autoridad suficiente. El orden es:

1. `deploy/postgresql/autorizacion/roles_up.sql`
2. `deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql`
3. `deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql`
4. `roles_up.sql`
5. `migraciones_autorizacion/000001_revalidacion_bolsa_baremacion.up.sql`
6. `migraciones/000001_bolsa_baremacion.up.sql`
7. `migraciones/000002_operaciones_baremacion.up.sql`
8. `migraciones/000003_abandono_y_lecturas.up.sql`
9. `migraciones/000004_entrega_outbox.up.sql`
10. `migraciones/000005_manifiesto_probatorio_v3.up.sql`

### Ventana obligatoria para instalar `000005`

`000005` **no es una migración en caliente**. Antes de ejecutarla, Sistemas
debe detener el pool y los procesos de la aplicación y cerrar en la red toda
posibilidad de reconexión del ejecutor. El chequeo de sesiones y los locks SQL
son defensas adicionales; no eliminan la carrera si un pool puede volver a
conectarse después del chequeo.

Con el tráfico ya aislado se verifica que no queda ninguna sesión cuyo `LOGIN`
sea miembro de `vec_bolsa_baremacion_ejecutor`. Solo la sesión DBA de esa
ventana recibe el literal exacto:

```bash
PGOPTIONS='-c vec.confirmar_mantenimiento_bolsa_baremacion_v3=INSTALAR_MIGRACION_BOLSA_BAREMACION_V3_SIN_TRAFICO' \
  psql --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
  --file migraciones/000005_manifiesto_probatorio_v3.up.sql
```

La migración toma `ACCESS EXCLUSIVE` sobre las tablas mutables V1 antes de
examinar su historia. Los resultados cerrados y comprobados por el arnés son:

- sin el literal: `SQLSTATE 55000`, sin cambios parciales;
- con una sesión ejecutora presente: `SQLSTATE 55000`, sin cambios parciales;
- con historia de versiones no reconstruible: `SQLSTATE 55000`;
- con esquema compatible y cero sesiones ejecutoras observables: instalación
  completa. El arnés no puede acreditar el aislamiento de red; cerrar la
  conectividad y detener todos los pools sigue siendo una precondición
  operacional de Sistemas.

Después del `COMMIT` se retira el literal, se validan objetos, ACL y RLS y solo
entonces se reabre la conectividad. El literal no debe incorporarse al entorno
permanente de ningún servicio.

Las identidades `LOGIN` se crean fuera del repositorio y reciben un solo grupo:

| Grupo | Uso permitido |
|---|---|
| `vec_bolsa_baremacion_migrador` | Ejecutar migraciones controladas mediante `SET ROLE` al propietario. Nunca es una cuenta de runtime. |
| `vec_bolsa_baremacion_ejecutor` | Invocar exclusivamente las fachadas cerradas vigentes de baremación; no recibe acceso a tablas ni helpers. |
| `vec_bolsa_baremacion_lector_outbox` | Reclamar y finalizar entregas de un consumidor previamente registrado. |
| `vec_bolsa_baremacion_registrador_atestacion` | Reserva sin privilegios; permanece cerrada hasta integrar el registrador COSE auditado. |

No se debe conceder más de uno de los grupos de runtime a una misma identidad.
Las cadenas de conexión del ejecutor y del lector outbox son distintas. El
`session_user` debe ser siempre el `LOGIN` técnico real; no se admite un pool
compartido que cambie identidades mediante una variable de sesión.

El registrador reservado tampoco recibe `CONNECT` sobre la base. Una cuenta
`LOGIN` que solo pertenezca a ese grupo no puede abrir sesión. Concederle
conectividad o superficie SQL requiere una migración posterior, revisada junto
con el registrador criptográfico real; no se habilita mediante operación
manual.

## Frontera criptográfica

El adaptador Go exige un `VerificadorSellosBaremacion`. El ensamblado productivo
debe aportar una implementación auditada que use un gestor de claves/HSM,
rechace identificadores desconocidos y compare HMAC en tiempo constante. Un
doble que siempre acepte solo es válido en pruebas y no constituye una frontera
de seguridad.

PostgreSQL no confía únicamente en ese control local. Cada operación requiere
además una atestación COSE durable y vigente de la decisión PDP, y vuelve a
consultar el estado autoritativo de autorización. Este paquete no concede aún
ninguna función para registrar esas atestaciones: el rol registrador está
deliberadamente sin privilegios. Hasta conectar el verificador/registrador COSE
auditado, toda operación funcional falla cerrada con autorización obsoleta. No
se deben insertar atestaciones manualmente en producción.

### Límite conocido de recuperación idempotente

Este corte no cumple todavía la decisión transversal **DEC-045** y permanece
cerrado a producción. En `reservar_cambio`, la ventana
`solicitada_en`/`expira_en` y la autorización se revalidan antes de buscar la
reserva durable. Cuando la busca, la coincidencia exige además la misma
referencia y huella de decisión y los mismos instantes de solicitud y
caducidad.

Por tanto, una confirmación ya persistida solo puede recuperarse mientras siga
vigente la ventana y la autorización originales. Tras su caducidad o
revocación, el reintento no puede devolver la respuesta confirmada aunque la
historia durable exista. Esto es un **NO-GO para prometer recuperación
idempotente**; un corredor verde de ACL o PostgreSQL no convierte el adaptador
ni `000002` en productivos.

Antes de producción debe implantarse DEC-045: índice e intención estables que
excluyan sesión, autorización y tiempos efímeros, más una autorización actual
independiente para cada recuperación. No se reordena aquí el SQL ni se relaja
la autorización porque ese cambio afecta conjuntamente al puerto, al
adaptador, al modelo de amenazas y a la API.

## Archivo probatorio y prevalidación V3

`000005_manifiesto_probatorio_v3` añade el contrato portable entre Go y
PostgreSQL sin reescribir las versiones V1 congeladas. La instalación se
rechaza si ya existe una baremación con versión superior a uno: no inventa ni
reconstruye a posteriori evidencias que PostgreSQL nunca recibió.

Una incorporación de decisión usa tres fases separadas:

1. La reserva conserva su HMAC de petición, distinto del HMAC de confirmación.
2. Una autorización dedicada permite obtener y sellar la vista probatoria
   previa. Se consume una sola vez con un efecto exacto y admite únicamente el
   replay literal de esa operación.
3. La confirmación exige el recibo de prevalidación, vuelve a comprobar OCC,
   reserva, autorización, agregado y manifiesto, y persiste versión, auditoría,
   outbox, manifiesto y resultado en el mismo `COMMIT`.

El archivo de una versión N contiene exactamente los N-1 manifiestos que
explican sus decisiones posteriores al alta. Cada entrada conserva el JSON
estructurado y los tres artefactos binarios exactos: contenido canónico,
representación canónica y preimagen HMAC. PostgreSQL reconstruye y valida cada
byte antes de responder. La continuidad no admite huecos y el documento de
salida tiene un límite cerrado de 64 MiB, comprobado antes y después de
agregarlo.

Las tablas V3 son append-only, tienen RLS habilitado y forzado y solo aceptan
al propietario exacto. El ejecutor recibe seis fachadas `SECURITY DEFINER`:
reserva, prevalidación, confirmación y tres lecturas enriquecidas. Las
fachadas V1 equivalentes quedan revocadas; la operación de abandono V1 se
mantiene porque no elude el archivo probatorio.

El fixture dorado compartido
`internal/modules/bolsa/testdata/manifiesto_probatorio_v3_dorado.json` es
validado por Go y por PostgreSQL. Fija 18 autorizaciones, 16 evidencias y las
huellas y longitudes exactas de los tres artefactos; una divergencia en
cualquiera de los dos runtimes rompe el corredor.

Esta migración implanta en las fachadas V3 un protocolo que evita la
incompatibilidad HMAC de V1 y cierra la persistencia del archivo. No repara ni
reabre las fachadas V1, no elimina los bloqueos de producción descritos en
DEC-045 ni sustituye el verificador HSM/KMS y el registrador COSE auditados.
Los sellos de la prueba son fixtures, no una frontera criptográfica
productiva.

### Incompatibilidad funcional V1 confirmada

La prueba SQL aislada tampoco demuestra que el adaptador Go pueda completar el
flujo. La aplicación sella la reserva y la confirmación sobre preimágenes
distintas; por diseño producen HMAC diferentes. Sin embargo,
`confirmar_cambio` V1 exige que la confirmación repita exactamente la
`huella_solicitud_hmac` guardada en la reserva. La prueba
`pruebas_sql/integracion_v1.sql` no detecta la incompatibilidad porque inyecta
el mismo HMAC literal en ambas llamadas.

Además, el manifiesto probatorio V1 incorpora autorizaciones efímeras de
reserva y confirmación, por lo que no puede utilizarse como parte de una
intención estable. Los recibos de custodia y retención necesitan también una
representación y huella canónicas propias; la huella del PDF no acredita el
recibo técnico.

La apertura exige una prueba de extremo a extremo Go → adaptador → PostgreSQL
real. Debe cubrir una respuesta perdida después de `COMMIT`, reinicio, sesión y
autorización nuevas, revocación, concurrencia y recuperación conjunta de
versión, auditoría y outbox. Hasta entonces, las funciones V1 no reciben
tráfico productivo aunque la prueba SQL y las ACL terminen correctamente.

## Entrega outbox

`evento_outbox` representa el hecho de dominio y es inmutable; su estado
permanece siempre `pendiente`. La entrega se modela por separado:

- `entrega_outbox_version` conserva reclamaciones, expiraciones, fallos y
  entregas por consumidor.
- `entrega_outbox_actual` apunta a la última revisión de cada entrega.
- `cursor_outbox_version` conserva cada avance confirmado y ordenado.
- `cursor_outbox_actual` apunta al último evento entregado.

El contrato es de entrega al menos una vez y en orden estricto por secuencia.
Un consumidor debe deduplicar en el destino mediante `referencia` y
`huella_registro_sha256`: si el destino aceptó el mensaje pero se perdió el
acuse, el evento puede volver a entregarse.

Antes de usar el lector se registra su `LOGIN` exacto mediante una migración
operativa revisada. Ejemplo para replay completo desde secuencia cero:

```sql
BEGIN;
SET LOCAL ROLE vec_bolsa_baremacion_propietario;
SET LOCAL search_path = pg_catalog;

INSERT INTO vec_bolsa_baremacion.consumidor_outbox_version (
    consumidor_ref, version, estado, rol_sesion, secuencia_inicial,
    registrada_en, acto_ref
) VALUES (
    'consumidor:integracion_rrhh', 1, 'activo',
    'vec_bolsa_outbox_rrhh_login', 0, clock_timestamp(),
    'acto:alta-consumidor:integracion_rrhh'
);
INSERT INTO vec_bolsa_baremacion.consumidor_outbox_actual (
    consumidor_ref, version, estado, actualizada_en
) VALUES (
    'consumidor:integracion_rrhh', 1, 'activo', clock_timestamp()
);
COMMIT;
```

El `LOGIN` indicado debe existir, tener `LOGIN` habilitado y pertenecer a
`vec_bolsa_baremacion_lector_outbox`. Para rotarlo se añade la siguiente
versión `activo` y se avanza el puntero; para revocarlo se añade la siguiente
versión `revocado`. Una revocación es terminal para esa referencia.

El trabajador genera un token aleatorio de al menos 32 bytes, llama a
`reclamar_evento_outbox` y conserva el token solo en memoria hasta invocar
`finalizar_entrega_outbox`. El arrendamiento admite entre 5 y 300 segundos. Un
fallo lleva una clave técnica acotada, nunca el cuerpo de una excepción ni
datos personales.

## Operación y protección de datos

- Cifrado de volumen, copias, WAL y réplicas; claves separadas del servidor y
  rotadas por el sistema corporativo. PostgreSQL no sustituye el cifrado de la
  infraestructura.
- TLS mutuo o autenticación fuerte entre aplicación y base de datos, incluso
  en red interna. No publicar el puerto de PostgreSQL fuera del segmento de
  datos.
- Copias con PITR y pruebas periódicas de restauración. La retención y el
  expurgo se gobiernan por expediente; los `down` nunca borran historia ni
  sustituyen un procedimiento aprobado de archivo o baja del entorno.
- Exportar las cabeceras de las cadenas de auditoría/eventos a un anclaje
  externo inmutable y alertar ante huecos, cambios de huella o secuencia.
- Alertar por `autorizacion_obsoleta`, colisiones repetidas, expiraciones de
  arrendamiento, fallos de entrega y esperas de bloqueo. No registrar pruebas
  COSE, agregados, tokens ni contenido personal en logs de aplicación.
- La base almacena metadatos y el agregado de baremación. Los documentos
  firmados permanecen en el almacén de objetos cifrado mediante su conector;
  aquí solo se conservan referencias y huellas.

## Pruebas reproducibles

Desde la raíz del repositorio:

```bash
./deploy/postgresql/bolsa_baremacion/probar_integracion.sh
```

La prueba usa PostgreSQL 18.4 real y comprueba:

- compilación y pruebas Go del adaptador, transacción y puerto;
- instalación completa y prueba funcional/adversaria dentro de una transacción
  que termina en `ROLLBACK`;
- cuentas `LOGIN` reales y separadas para ejecutor, lector y registrador;
- ausencia de acceso directo, helpers y superficies cruzadas para runtime;
- cierre de funciones y tipos actuales y futuros, incluidos tipos fila;
- ausencia de `CONNECT` para el registrador reservado;
- reserva, confirmación atómica, reintento exacto, lectura sensible y fallo
  cerrado sin atestación;
- decisiones canónicas con una clave menos o más;
- entrega, fallo, reintento, cursor y evento original inmutable;
- rechazo sin opt-in de cada `down`, incluso sobre esquema vacío;
- rechazo con opt-in de cada `down` si existe cualquier historia, sin cambios
  parciales de funciones, restricciones ni ACL;
- rechazo del `down` base ante una dependencia externa, conservando ambos
  objetos gracias a `RESTRICT`;
- desmontaje completo de una segunda instalación realmente vacía, sin vaciar
  ni alterar la primera instancia que conserva historia;
- prevalidación de `roles_down.sql` frente a membresías entrantes, salientes y
  opciones estructurales, además de atributos de rol, propietario, ACL,
  `search_path` y etiquetas de la guarda DDL;
- carrera real entre bases: el `GRANT` espera al `roles_down`, falla de forma
  natural después de `DROP ROLE` y no queda ninguna arista relacionada con los
  OID retirados ni ninguna membresía huérfana en el clúster;
- dos sesiones PostgreSQL reales compitiendo por el mismo evento: exactamente
  una obtiene el arrendamiento;
- ausencia del token de arrendamiento en la historia durable.
- paridad Go/PostgreSQL del fixture dorado V3, byte por byte;
- perfil textual común con Go: tipos JSON exactos, referencias ASCII visibles
  sin comodín, HMAC acotado y RFC3339Nano UTC canónico;
- inventario de catálogo cerrado: ninguna función para `PUBLIC`, cinco tablas
  V3 con RLS forzado y política única, cero ACL directa de objetos para roles
  runtime y conjuntos exactos de fachadas ejecutables;
- 4096 autorizaciones y 4096 evidencias persistidas y cotejadas por los
  triggers set-based, más construcción del archivo dentro de la puerta global
  de 15 segundos;
- agregación lineal sintética de 4096 fragmentos y 64 MiB con huella conocida;
- rechazo con `SQLSTATE 23503/23514` de hijos huérfanos, divergentes,
  secuencias fuera de rango y cabeceras incompletas al confirmar;
- reserva, prevalidación consumible, confirmación y archivo N-1 atómicos;
- extremo a extremo Go → PostgreSQL en tres procesos: prevalidación durable,
  fallo KMS y reintento confirmado mientras la capacidad sigue solo en memoria;
  después, dos reinicios y dos recuperaciones idempotentes del resultado ya
  confirmado, sin serializar la capacidad;
- dos conexiones `SERIALIZABLE` detenidas por una tercera barrera compitiendo
  por la misma baremación: un `COMMIT`, un `SQLSTATE 40001` y reintento cerrado
  como `en_curso`, con una sola reserva y un solo consumo;
- RLS forzado con un `LOGIN` real, incluso ante un `GRANT SELECT` accidental;
- rechazo íntegro del `down` V3 con historia y reversión de una segunda
  instalación V3 realmente vacía.

El tramo V3 también puede ejecutarse de forma aislada:

```bash
./deploy/postgresql/bolsa_baremacion/probar_integracion_v3.sh
```

La imagen puede sustituirse de forma explícita con
`VEC_POSTGRES_TEST_IMAGE`, manteniendo PostgreSQL 18.4 o una revisión aprobada.

## Reversión

Se ejecutan los `down` en orden inverso. `000005` falla antes de mutar si existe
cualquier manifiesto, hijo, prevalidación, resultado o consumo V3. Los `down`
anteriores conservan su inventario completo del esquema. Las relaciones se
bloquean con `ACCESS EXCLUSIVE` antes de comprobarlas. El opt-in nunca autoriza
a borrar historia y estos ficheros no son herramientas de expurgo.

La reversión `000005` exige su literal propio, incluso vacía:

```bash
PGOPTIONS='-c vec.confirmar_reversion_bolsa_baremacion_v3=REVERTIR_MIGRACION_BOLSA_BAREMACION_V3' \
  psql --set ON_ERROR_STOP=1 \
  --file migraciones/000005_manifiesto_probatorio_v3.down.sql
```

Las reversiones `000004`, `000003`, `000002` y la frontera de autorización
exigen, en su propia sesión, el literal común:

```bash
PGOPTIONS='-c vec.confirmar_reversion_bolsa_baremacion=REVERTIR_MIGRACION_BOLSA_BAREMACION_V1' \
  psql --set ON_ERROR_STOP=1 --file RUTA_DEL_DOWN.sql
```

Una reversión vacía de `000005` restaura el inventario y la superficie SQL V1,
no un flujo funcionalmente equivalente a V3. V1 conserva la incompatibilidad
HMAC y las limitaciones de lectura descritas arriba, por lo que este `down` es
una operación de retirada técnica y nunca una autorización para devolverle
tráfico.

La retirada base `000001` exige además su confirmación destructiva distinta,
incluso vacía:

```bash
PGOPTIONS='-c vec.confirmar_destruccion_bolsa_baremacion=DESTRUIR_HISTORIA_BOLSA_BAREMACION_IRREVERSIBLE' \
  psql --set ON_ERROR_STOP=1 \
  --file migraciones/000001_bolsa_baremacion.down.sql
```

El nombre histórico del literal no cambia su alcance: si existe una sola fila,
la retirada se rechaza. El base solo elimina tablas y funciones de clases
conocidas y termina con `DROP SCHEMA ... RESTRICT`; nunca usa `CASCADE`. Una
vista, función u otra dependencia externa hace abortar toda la transacción y se
conserva.

Tras retirar esquema y frontera se ejecuta `roles_down.sql`. Este prevalida
íntegramente la guarda DDL, las opciones de los cinco roles y las tres
coordenadas de todas sus membresías antes de revocar o eliminar nada. La
retirada exige superusuario y una ventana de mantenimiento sin administración
concurrente de roles: después de su bloqueo advisory toma `ACCESS EXCLUSIVE`
primero sobre `pg_authid`, después sobre `pg_auth_members`, `pg_database` y
`pg_default_acl`, en ese orden, hasta el `COMMIT`.
Así un `GRANT` iniciado desde cualquier otra base no puede conservar OID que el
down vaya a retirar.

El inventario se hace sobre los cinco OID y las tres coordenadas `roleid`,
`member` y `grantor`; solo admite una fila: el enlace estructural
propietario→migrador, con `ADMIN FALSE`, `INHERIT FALSE`, `SET TRUE` y otorgado
por el superusuario bootstrap gobernado del clúster (OID interno estable 10).
PostgreSQL atribuye a ese principal las concesiones hechas por cualquier
superusuario, aunque la guarda pertenezca a otro DBA nominativo. Contraseña,
caducidad y cualquier ajuste global o por base también deben permanecer
ausentes.

Las ACL de base se inventarían por otorgante, destinatario, privilegio y
opción de concesión: solo existen `CONNECT`/`CREATE` para el propietario y
`CONNECT` para migrador, ejecutor y lector. El registrador reservado carece de
conexión. Tras retirar el esquema pueden no existir defaults si el bootstrap
se abortó antes de migrar, o quedar exactamente los dos globales cerrados por
`000001`, uno para funciones y otro para tipos; cualquier estado parcial, fila
o destinatario adicional aborta antes de que el down revoque nada.

Cualquier otra arista, atributo o procedencia aborta antes de mutar. El arnés
conecta de antemano tres sesiones reales, demuestra con `pg_blocking_pids` que
el `GRANT` espera al down, deja que ambos terminen sin cancelarlos y comprueba
después que no quedan roles, aristas por ninguno de sus cinco OID ni huérfanos
globales. Una segunda instalación ejecutada por un DBA alternativo prueba que
propiedad administrativa y otorgante bootstrap se validan por separado.

Una instancia con historia se conserva intacta. Su baja requiere un
procedimiento independiente y aprobado de archivo, verificación, retención y
destrucción; nunca se consigue convirtiendo un `down` en un borrado.
