# CT-O2-R3B — candidatura técnica durable y confirmación PostgreSQL

Fecha del corte: 31 de agosto de 2026.

## Resultados cronológicos de las puertas dinámicas

Estado: **NO-GO; R2 pendiente de revisión independiente y de cualquier nueva
autorización dinámica**.

### Primera ejecución pre-final: NO-GO de bootstrap

La ejecución pre-final autorizada del runner terminó con código `3` antes de
instalar migraciones. El bootstrap
`deploy/postgresql/contexto_actor_v1/roles_up.sql` rechazó en su línea 41 la
base desechable porque conservaba privilegios iniciales de `PUBLIC`:

```text
ERROR: la base dedicada debe llegar sin privilegios de PUBLIC
```

En esa ejecución no llegaron a ejecutarse `000046`, `000047`, las pruebas Go
contra PostgreSQL ni ninguna confirmación V2. La limpieza automática retiró el
contenedor, red, volumen y directorio temporal etiquetados
`ct-o2-r3b-pg-20260831`. La comprobación posterior obtuvo cero identificadores
para esos cuatro tipos de residuo. El contenedor ajeno quedó antes y después en
el mismo estado verificable:

```text
d6217278d8718a4e51c4be9523dde15406559989abe46f3a5e496624d6aa4aeb|running|/contagrx-t224-pg-focal|sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382
```

El candidato preservado incorporó después esa barrera en el runner: revoca los
privilegios predeterminados de `PUBLIC` sobre la base y el esquema `public`
antes de instalar la autoridad de contexto. El fallo pre-final permanece como
`NO-GO`; la corrección posterior no lo reinterpreta ni lo convierte en una
ejecución superada.

### Repetición excepcional final sobre `a0cedc2`: NO-GO de `000047`

Dirección autorizó una única repetición excepcional sobre el hash exacto
`a0cedc2e8fce0ab36f0d4fb4ff30dc43e6345596`. Se ejecutó exactamente una vez el
31 de agosto de 2026, desde `19:39:02Z` hasta `19:39:12Z`, duró 10 segundos,
terminó con `RC=3` y produjo un log con SHA-256
`f988ce0cde679aae1ea39102b346d5210a3c450b9659fc0c1a6ee7de1e678193`.

La repetición alcanzó las migraciones CT `000001` a `000005`, `000046` y
VEC-AD-3. Falló en
`000047_componentes/010_candidatura_y_aliases.sql`, línea 73, al resolver
`pg_catalog.greatest(timestamp with time zone, timestamp with time zone)`. No
alcanzó el resto de `000047`, las pruebas Go contra PostgreSQL ni ninguna
confirmación. Terminó con cero residuos propios y el cerrojo liberado; el
contenedor ajeno `d6217278...` permaneció intacto y en estado `running`.

Este segundo resultado también es `NO-GO`. No oculta ni sustituye el fallo
pre-final por `PUBLIC`, no acredita PostgreSQL y no autoriza otra ejecución.
R1 retira exclusivamente la cualificación inválida de la construcción especial
`GREATEST`, conservando la semántica de máximo temporal. Queda pendiente de una
nueva revisión independiente y de autorización dinámica expresa; no se declara
`GO`.

### Revisión estática de `a0cedc2`: NO-GO, `P0=0`, `P1=2`, `P2=0`

La revisión estática independiente del candidato `a0cedc2` emitió `NO-GO` con
dos hallazgos P1 independientes:

1. los triggers de `candidatura_alta_tecnica` y `candidatura_alta_alias`
   rechazaban `UPDATE` y `DELETE`, pero no `TRUNCATE` de propietario; y
2. el runner solo seleccionaba el resolutor Go y después invocaba V2
   directamente, sin construir `NuevaTransaccionAltasPostgreSQLCandidata` ni
   atravesar `ConfirmarAltaCandidata`, por lo que no acreditaba las doce
   entradas ni el escaneo real de las ocho columnas.

R1, commit `df4b0bddacb755c50a1e16b2dac936f2f46affa1`, se mantuvo estrecho:
solo corrigió `pg_catalog.greatest` a `GREATEST` y conservó íntegros los dos
`NO-GO` dinámicos anteriores. R2 añade la defensa canónica `BEFORE TRUNCATE`
por sentencia, regresión de conservación exacta de filas y el recorrido del
adaptador público con proveedor neutral, éxito y replay desde dos pools. R2 no
ha ejecutado PostgreSQL ni el runner y permanece pendiente de revisión estática
independiente y, solo después, de una autorización dinámica nueva. No se
declara `GO`.

## Capacidad e invariante

Este corte estabiliza o recupera, antes de autorización, una única
`CandidaturaAlta` técnica mediante pares HMAC de una generación activa y hasta
tres retenidas. La candidatura es no autoritativa: resolverla no crea reserva
administrativa, expediente, actuación, auditoría, outbox ni consumo VEC.

El único efecto sigue en `confirmar_alta_atestada_v2`. La función acredita la
candidatura exacta —coordenadas, identidad, instante y pares generacionales— y,
en la misma transacción serializable, delega en el motor privado
`confirmar_alta_atestada_v1`. No se fabrica una `PreparacionAlta`.

## Fronteras

- `resolver_candidatura_alta_tecnica_v1` recibe diez entradas: dos matrices
  HMAC, organización, actor, perfil, cuatro referencias propuestas e instante
  propuesto. Devuelve once columnas y solo los estados `estabilizada` o
  `recuperada`.
- `confirmar_alta_atestada_v2` conserva las doce entradas y las ocho salidas de
  O2-05. V1 deja de ser ejecutable por el runtime y permanece como motor
  `SECURITY DEFINER` privado.
- `ProveedorMaterialConfirmacionAlta` entrega una única instantánea neutral de
  las diez entradas VEC-AD-3. Su implementación concreta corresponde a R3C.
- `TransaccionAltasCandidata` recibe únicamente una
  `OrdenConfirmarAltaCandidata`; no acepta bytes del cliente ni una correlación
  libre.

## Persistencia, rotación y backfill

La migración `000047_candidatura_alta_durable_o2_r3b` crea una raíz inmutable y
una tabla append-only de pares de alias. Ámbito y huella se guardan alineados en
la misma fila y una generación solo puede pertenecer a una raíz. La clave HMAC
en claro nunca cruza SQL ni logs: PostgreSQL recibe únicamente sellos.

Un replay exacto devuelve las referencias, identidad, par raíz e instante
originales. Una generación nueva añade un alias sin mutar la raíz. Un cruce de
pares, otra huella o identidad para el mismo ámbito y una colisión de
referencias se rechazan; una colisión aleatoria no se interpreta como
idempotencia.

El backfill copia las identidades de reserva anteriores y conserva
`identidad_reserva_alta.creada_en` como `instante_efecto`. El `DOWN` ordinario
queda bloqueado si existe historia nueva. La retirada destructiva exige el
testigo explícito ya gobernado por las migraciones CT.

## Atomicidad y tratamiento de errores

El adaptador abre transacciones nuevas, serializables y read-write. Solo
`40001` y `40P01` se reintentan, como máximo tres intentos. Un COMMIT confirmado
prevalece sobre una cancelación posterior.

`pgconn.SafeToRetry` y `pgx.ErrTxCommitRollback` no provocan reconciliación.
`08007` o un error de transporte con envío posible permiten una única
reconciliación independiente, limitada a cinco segundos, con las mismas doce
entradas. Una segunda ambigüedad devuelve `ErrResultadoAltaIndeterminado`.
Una fila, recibo o `recibo_huella_sha256` divergente se revierte y devuelve
`ErrResultadoAltaNoConfiable`. Ningún error expone SQL, nombres de tablas o
coordenadas internas.

## Canon y ACL

Go construye byte a byte `vec.contratacion-temporal.efecto-alta.v2` y
`vec.contratacion-temporal.sellos-hmac.v1`, con UTC canónico a microsegundos.
El hash del efecto debe coincidir con el atributo autorizado y el resumen de la
capacidad debe ligar decisión, correlación, operación, efecto, audiencia y
ventana exactos antes de abrir la transacción.

El runtime carece de privilegios sobre las tablas nuevas, sobre
`preparar_alta_v2` y sobre `confirmar_alta_atestada_v1`. Para este flujo solo
recibe `EXECUTE` sobre el resolutor y V2. Las funciones auxiliares y las tablas
siguen cerradas a `PUBLIC`. Ambas tablas activan y fuerzan RLS; su única
política pertenece al rol propietario y los triggers conservan el historial
append-only frente a `UPDATE`, `DELETE` y `TRUNCATE`, también cuando actúa el
propietario.

## Evidencia focal

El runner `probar_o2_r3b_candidatura_postgresql18_4.sh` usa exclusivamente la
imagen local PostgreSQL 18.4 fijada por digest y recursos etiquetados propios.
Instala `000047` después de `000046`. La matriz que deberá acreditar cuando
exista una autorización dinámica nueva —R2 no la ha ejecutado— incluye:

- backfill e instante original;
- replay entre pools, concurrencia y rotación con alias;
- rechazo de `UPDATE`, `DELETE` y `TRUNCATE` por propietario, con recuentos y
  huellas de ambas tablas exactamente conservados;
- conflictos, colisión, rollback y ausencia de efectos al resolver;
- construcción del adaptador Go público con proveedor neutral, doce entradas,
  escaneo real de ocho columnas, confirmación y replay exacto desde dos pools;
- cotejo del recibo completo y de `recibo_huella_sha256` mediante la función
  privada Go, sin exponer esa huella por el puerto;
- rechazo de mutación de coordenada y revocación viva sin efectos;
- ACL/RLS, `DOWN` protegido, retirada explícita y reinstalación; y
- conservación del contenedor ajeno y ausencia de residuos propios.

Las pruebas unitarias Go cubren además `40001` y `40P01` en transacciones
nuevas con máximo de tres intentos, cancelación posterior al COMMIT,
`SafeToRetry`, `pgx.ErrTxCommitRollback`, `08007`, transporte posiblemente
enviado, primera y segunda ambigüedad, y adulteración de fila/recibo/hash. Para
cada clase se comprueban inicios, commits y reconciliaciones; la rama
`SafeToRetry` inyecta ahora el fallo en `Commit`, no antes de alcanzarlo.

## Límite y siguiente corte

R3B no modifica aplicación, HTTP, rutas, composición ni frontend. Tampoco
cierra O2-06 ni declara la aplicación arrancable. El siguiente corte es la
revisión independiente del hash exacto de R2 y, solo con una autorización
posterior expresa, una nueva validación dinámica. R3C permanece bloqueada hasta
resolver ese `NO-GO`; después deberá migrar `ServicioRegistroSolicitud` al contrato
candidato y componer el proveedor concreto de material de confirmación bajo
revisión independiente.
