# Diseño acoplado del adaptador y la reconciliación O2-06

Fecha: 23 de julio de 2026.

Base de acoplamiento:
`b53c6b5b399e0be9b380b6ed45b023c04d7e7f6f`.

Estado: contrato de implementación alineado con O2-05 integrado. Este
documento no modifica la función SQL, sus migraciones ni sus privilegios y no
habilita producción.

## Puerta de acoplamiento superada

El diseño O2-06A de `2c800fa`–`5ae0532` era deliberadamente previo a la firma
SQL. Su contrato previsto no coincide con O2-05 integrado:

| Superficie | Diseño O2-06A antiguo | O2-05 integrado y autoritativo |
| --- | --- | --- |
| Entradas de confirmación | 14 | 12 |
| Columnas de salida | 12 | 8 |
| Resultado nominal SQL | `confirmada`, `replay`, `denegada`, `idempotencia_conflictiva` | No existe columna de resultado |
| Procedencia y huellas separadas | Procedencia, recibo, auditoría y evento | Una única `recibo_huella_sha256` |
| Reconciliación | Función read-only prevista | Reejecución idempotente de la misma función |
| Candidatos en reconciliación | Podían regenerarse | Se conservan exactamente los bytes del primer intento |

La fuente de verdad es la migración integrada:

```text
deploy/postgresql/contratacion_temporal/migraciones/
000005_funcion_confirmar_alta_atestada.up.sql
```

No se adapta O2-05 al diseño antiguo. Se adapta O2-06 a la firma real.

## Firma congelada de doce entradas

El adaptador invoca exclusivamente:

```sql
vec_contratacion_temporal.confirmar_alta_atestada_v1(
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea,
    p_alta_canonica bytea,
    p_sellos_hmac_canonicos bytea
)
```

Mapeo exacto:

| Posición | Entrada PostgreSQL | Origen Go |
| --- | --- | --- |
| 1 | `p_capacidad_canonica` | Exportación cerrada de la capacidad VEC-AD-3 |
| 2 | `p_decision_canonica` | Representación canónica de la decisión V3 |
| 3 | `p_motivo_canonico` | Representación canónica del motivo catalogado |
| 4 | `p_contexto_actor_canonico` | Resultado durable V2 clonado |
| 5 | `p_persona_version` | Instantánea del contexto, decimal sin pérdida |
| 6 | `p_perfil_version` | Instantánea del contexto, decimal sin pérdida |
| 7 | `p_payload_vec_ad_3` | Material opaco del proveedor VEC |
| 8 | `p_sobre_cose_sign1` | Sobre COSE del proveedor VEC |
| 9 | `p_evidencia_verificacion` | Evidencia opaca del proveedor VEC |
| 10 | `p_raiz_publica_spki` | DER SPKI Ed25519 exacta |
| 11 | `p_alta_canonica` | Canon `efecto-alta.v2` construido desde la orden |
| 12 | `p_sellos_hmac_canonicos` | Canon `sellos-hmac.v1` de pares alineados |

Los doce valores se construyen una sola vez. Los reintentos por serialización
y la reconciliación reutilizan copias byte a byte; no vuelven a pedir
capacidad, referencias, reloj, decisión ni material criptográfico.

## Salida congelada de ocho columnas

La función devuelve exactamente una fila:

```text
expediente_ref
numero_visible
version
recibo_ref
auditoria_ref
evento_ref
confirmada_en
recibo_huella_sha256
```

El adaptador usa `Query`, cierra las filas y exige una sola. Cero filas, una
segunda fila, `NULL`, versión fuera de rango, instante no canónico, referencia
inválida o huella distinta del canon local son resultado no confiable. El
recibo se valida antes de `COMMIT` y de nuevo sobre una copia defensiva
después del resultado durable o reconciliado.

O2-05 no informa si la fila procede de alta nueva o replay. O2-06 no lo
infiere desde SQLSTATE, texto de error ni valores aleatorios: ambos casos
representan el mismo recibo durable exacto.

## Transacción y sesión

Cada intento usa una transacción nueva:

```text
SERIALIZABLE READ WRITE
search_path = pg_catalog
row_security = on
timezone = UTC
lock_timeout = 2s
statement_timeout = 15s
idle_in_transaction_session_timeout = 20s
```

El pool es exclusivo de O2-06. La cuenta solo necesita `CONNECT`, `USAGE` y
`EXECUTE` sobre la firma anterior. No recibe DML, lectura de tablas,
preparación histórica, consumidor VEC genérico, `SET ROLE` ni ejecución de
`reconciliar_agregado_alta_v1`.

## Reintentos determinados

Antes de invocar `Commit`, solo `40001` y `40P01` permiten reintentar. Cada
reintento:

1. revierte de forma acotada;
2. abre una transacción nueva;
3. reinstala todos los parámetros;
4. reutiliza los mismos doce valores;
5. respeta un máximo de tres intentos y el plazo del llamador.

`23505`, `42501`, `55P03`, `57014`, errores de forma, cancelación y cualquier
otro SQLSTATE no se reinterpretan como conflicto funcional ni se reintentan.
Los mensajes PostgreSQL/pgx nunca cruzan la frontera.

## Resultado indeterminado y reconciliación

Después de invocar `Commit`:

| Resultado observado | Tratamiento |
| --- | --- |
| `nil` | Confirmación durable |
| `40001` o `40P01` | Rollback determinado; puede continuar la política ordinaria |
| `pgx.ErrTxCommitRollback` | Rollback determinado; no se reconcilia |
| `pgconn.SafeToRetry(err) == true` | Fallo garantizado antes de enviar datos; no se reintenta ni reconcilia |
| `08007` | Resolución de transacción desconocida; se reconcilia |
| Otro error de transporte/timeout/EOF | Resultado indeterminado; se reconcilia |

La reconciliación no abre una superficie read-only. Usa un contexto interno
acotado, una conexión y transacción nuevas y reejecuta una sola vez la misma
función con los mismos doce valores. Esto es seguro porque O2-05 consume la
capacidad y confirma el agregado en un único `COMMIT`:

- si el primer `COMMIT` ocurrió, la rama durable de replay devuelve el mismo
  recibo;
- si no ocurrió, la misma capacidad y el mismo efecto pueden confirmar una
  única vez;
- ninguna de las dos rutas crea un segundo agregado.

La reconciliación no regenera candidatos ni capacidad y no consulta memoria,
txid, WAL/LSN o reloj del cliente. Si su `COMMIT` vuelve a ser ambiguo, si la
base no responde, si el recibo diverge o si no se acredita una fila exacta,
la salida es `ErrResultadoAltaIndeterminado`. Nunca se afirma éxito ni
rollback.

Un error marcado por pgx como seguro para reintentar no entra en esta ruta:
esa marca acredita que ningún dato llegó al servidor. Reejecutarlo con un
contexto nuevo después de una cancelación podría crear un efecto que el
solicitante ya no espera.

## Reinicio

La recuperación tras reinicio se apoya en los mismos materiales durables que
ya estabilizan la orden de alta. El adaptador no mantiene caché necesaria
para la corrección. Una nueva invocación semántica obtiene del flujo de
aplicación el mismo candidato/recibo o el replay durable; dentro de una
reconciliación iniciada el lote de doce parámetros permanece inmutable.

Antes de proyectar y autorizar el efecto, aplicación entrega al puerto
`ResolutorCandidaturaAlta` las colecciones HMAC alineadas y una propuesta de
referencias acuñada por servidor. El puerto devuelve siempre la candidatura
técnica asociada al mismo ámbito y huella:

- el primer intento estabiliza la propuesta sin crear expediente ni conceder
  autoridad;
- un segundo servicio o proceso recupera exactamente reserva, expediente,
  número visible y recibo;
- la misma clave semántica con otra huella se rechaza;
- tras rotación puede recuperarse un par retenido, pero nunca cruzar
  generaciones de ámbito y huella.

La interfaz es neutral a PostgreSQL y no restaura
`preparar_alta_v2`. La creación administrativa continúa limitada a
`TransaccionAltas`, después de la decisión positiva V3.

No se acepta reconstruir autoridad desde cookies, cabeceras libres, JSON del
cliente o almacenamiento web. Web, escritorio, CLI y MCP consumen el mismo
caso de uso.

## Errores públicos y saneado

| Error neutral | Significado |
| --- | --- |
| `ErrOrdenAltaInvalida` | La orden o sus ligaduras no permiten intentar el efecto |
| `ErrAutorizacionDenegada` | La autoridad o el material fueron rechazados |
| `ErrPersistenciaNoDisponible` | Fallo determinado o dependencia no disponible |
| `ErrResultadoAltaIndeterminado` | Se intentó `COMMIT` y la reconciliación no pudo acreditar el resultado |
| `ErrResultadoRegistroNoConfiable` | La fila o el recibo no superaron validación |

No se incluyen en errores, logs o recibos: SQLSTATE salvo clase interna
permitida, texto pgx/PostgreSQL, DSN, capacidad, HMAC, claves, payload, COSE,
identidad, documentos ni datos personales. Las referencias y huellas del
recibo son seudónimos protegidos, no datos anónimos.

## Pruebas obligatorias

- unitarias: 12 parámetros en orden, canon del efecto y sellos, 8 columnas,
  0/1/>1 filas, recibo y huella adulterados;
- transacción: éxito, `40001`, `40P01`, errores no reintentables, cancelación
  antes de `COMMIT`, `ErrTxCommitRollback` directo y envuelto;
- indeterminado: respuesta perdida/timeout después de `COMMIT`, segunda
  transacción exacta, segundo fallo ambiguo y divergencia de recibos;
- PostgreSQL 18 efímero: éxito, replay, concurrencia, caída antes de
  `COMMIT`, respuesta perdida, conexión/proceso nuevos, ACL y recibo
  adulterado;
- focales, carrera, `go vet`, globales, calidad, tamaños, `diff --check` y
  barrido de secretos.

La habilidad `admin-data-web` refuerza aquí que una acción administrativa
jurídicamente relevante nunca se presenta como confirmada sin recibo,
instante, referencias y evidencia íntegra; el estado indeterminado permanece
visible como tal.

## Límites

O2-06 no compone la ruta real, no registra HTTP, no toca web y no habilita
producción. O2-07 deberá inyectar el pool y proveedor VEC reales y fallar
cerrado si falta cualquiera. HSM/KMS, custodia, ancla anti-restauración y las
puertas CT-CUM continúan fuera de este commit.
