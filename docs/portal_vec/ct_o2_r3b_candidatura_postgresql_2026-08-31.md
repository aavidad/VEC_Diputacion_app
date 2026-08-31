# CT-O2-R3B — candidatura técnica durable y confirmación PostgreSQL

Fecha del corte: 31 de agosto de 2026.

## Resultado de la única puerta dinámica

Estado: **NO-GO dinámico; candidato R3B-B no confirmado**.

La única ejecución autorizada del runner terminó con código `3` antes de
instalar migraciones. El bootstrap
`deploy/postgresql/contexto_actor_v1/roles_up.sql` rechazó en su línea 41 la
base desechable porque conservaba privilegios iniciales de `PUBLIC`:

```text
ERROR: la base dedicada debe llegar sin privilegios de PUBLIC
```

No se repitió la ejecución. No llegaron a ejecutarse `000046`, `000047`, las
pruebas Go contra PostgreSQL ni ninguna confirmación V2. La limpieza automática
retiró el contenedor, red, volumen y directorio temporal etiquetados
`ct-o2-r3b-pg-20260831`. La comprobación posterior obtuvo cero identificadores
para esos cuatro tipos de residuo. El contenedor ajeno quedó antes y después en
el mismo estado verificable:

```text
d6217278d8718a4e51c4be9523dde15406559989abe46f3a5e496624d6aa4aeb|running|/contagrx-t224-pg-focal|sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382
```

El siguiente intento, si dirección autoriza una nueva ejecución, debe preparar
la base dedicada con la barrera de privilegios de `PUBLIC` que exige Contexto
de actor antes de cargar `roles_up.sql`. Esta evidencia no autoriza repetir la
puerta ni convertir el resto de pruebas estáticas en un GO PostgreSQL.

El candidato preservado incorpora ya esa barrera en el runner y la mantiene
sin ejecutar: revoca los privilegios predeterminados de `PUBLIC` sobre la base
y el esquema `public` antes de instalar la autoridad de contexto. La única
invocación autorizada ya terminó con el fallo descrito y no se fabrica una
segunda ejecución.

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
append-only.

## Evidencia focal

El runner `probar_o2_r3b_candidatura_postgresql18_4.sh` usa exclusivamente la
imagen local PostgreSQL 18.4 fijada por digest y recursos etiquetados propios.
Instala `000047` después de `000046` y acredita:

- backfill e instante original;
- replay entre pools, concurrencia y rotación con alias;
- conflictos, colisión, rollback y ausencia de efectos al resolver;
- confirmación V2 real con el canon O2-05 y replay exacto;
- rechazo de mutación de coordenada y revocación viva sin efectos;
- ACL/RLS, `DOWN` protegido, retirada explícita y reinstalación; y
- conservación del contenedor ajeno y ausencia de residuos propios.

Las pruebas unitarias Go cubren además reintentos transaccionales, cancelación
posterior al COMMIT, envío seguro, `ErrTxCommitRollback`, primera y segunda
ambigüedad, y adulteración de fila/recibo/hash.

## Límite y siguiente corte

R3B no modifica aplicación, HTTP, rutas, composición ni frontend. Tampoco
cierra O2-06 ni declara la aplicación arrancable. R3C debe migrar
`ServicioRegistroSolicitud` al contrato candidato y componer el proveedor
concreto de material de confirmación bajo revisión independiente.
