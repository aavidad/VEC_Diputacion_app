# Decisión F0-C2: consumidor nominal privado

Fecha: 2 de agosto de 2026.

## Finalidad y corte

C2 compone A4, B2 y C1 para registrar y consumir una fuente corporativa en la
misma transacción exterior. No publica todavía una fachada ejecutable.

Su write-set completo es:

```text
M/080_consumidor_nominal.sql
T/080_consumidor_nominal.sql
```

`M` y `T` conservan el significado fijado en la
[decisión F0 principal](decision_f0_capacidad_fuente_corporativa_2026-08-01.md).
C2 no modifica otros componentes, roles, runners, envoltorios ni migraciones.

## Firma y resultado exactos

La función es:

```sql
vec_autorizacion_atestada_v3.
registrar_y_consumir_fuente_corporativa_contexto_actor_v1_atestada(
    p_audiencia_consumo_esperada text,
    p_accion_esperada text,
    p_tipo_efecto_esperado text,
    p_operacion_ref_esperada text,
    p_efecto_ref_esperada text,
    p_huella_efecto_sha256_esperada text,
    p_capacidad_canonica bytea,
    p_manifiesto_fuente_canonico bytea,
    p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
)
RETURNS TABLE (
    capacidad_ref text,
    fuente_ref text,
    fuente_version numeric,
    evento_fuente_ref text,
    huella_evento_fuente_sha256 text,
    huella_manifiesto_fuente_sha256 text,
    operacion_ref text,
    efecto_ref text,
    huella_efecto_sha256 text,
    consumo_huella_sha256 text,
    consumida_en timestamptz,
    consumo_nuevo boolean
)
```

El orden y los nombres son ABI. No devuelve `evento_fuente_emitido_en`,
`nonce`, COSE, evidencia, SPKI ni material HMAC.

La función es `VOLATILE`, no `STRICT`, `SECURITY DEFINER`,
`PARALLEL UNSAFE` y no leakproof. Pertenece a
`vec_autorizacion_atestada_v3_propietario` y fija exactamente
`search_path=pg_catalog` y `lock_timeout=2s`.

## Precondiciones y locks

Solo se admite dentro de `SERIALIZABLE READ WRITE`, en UTC, con los límites de
sesión cerrados por F0. C2 aplica los límites A1 antes de reservar memoria,
deriva los cánones necesarios y obtiene `capacidad_ref` y `operacion_ref` sin
aceptarlos del cliente.

Después de bloquear `checkpoint_gobierno FOR UPDATE`, calcula con semilla cero:

```text
hashtextextended(
  'vec_autorizacion_atestada_v3:f0:capacidad:v1:' || capacidad_ref, 0)
hashtextextended(
  'vec_autorizacion_atestada_v3:f0:operacion:v1:' || operacion_ref, 0)
```

Ordena los dos `bigint` de menor a mayor, elimina el duplicado si una colisión
de hash los hace iguales y toma cada `pg_advisory_xact_lock` una sola vez. No
se permite un orden capacidad→operación dependiente de la llamada.

## Replay histórico

Con los locks ya tomados, C2 busca exhaustivamente las cuatro coordenadas
durables: capacidad, `(fuente_ref, evento_fuente_ref)`, nonce y operación.

Solo existe replay cuando la historia reconstruida coincide exactamente en:

- coordenadas e identificadores esperados;
- huellas de los cánones reconstruidos de capacidad y manifiesto;
- canon de consumo reconstruido, byte a byte, y su huella persistida;
- huellas de COSE, evidencia, SPKI y efecto;
- audiencia, acción, tipo de efecto y referencias ligadas.

En replay devuelve exclusivamente los valores persistidos y
`consumo_nuevo=false`. No llama a C1, no lee de nuevo el reloj y no revalida
vigencias, punteros, catálogo ni revocaciones actuales. Es una consulta de un
hecho histórico confirmado, no una autorización para crear otro efecto.

Cualquier coincidencia parcial o reutilización con otra preimagen es una
colisión cerrada. Las fachadas posteriores deben exigir `consumo_nuevo=true`
para crear un efecto nuevo.

## Consumo nuevo

Si no existe historia, C2 llama al acreditador C1 con los mismos bytes y
coordenadas. Entrega el `acreditada_en` devuelto por C1 a A4 y lo persiste sin
otro reloj como `consumida_en`, tanto en el canon como en la relación de
consumo.

Después inserta la atestación B2 y su consumo 1:1 en la misma transacción. La
primera confirmación devuelve `consumo_nuevo=true`. Un rollback posterior
elimina ambas filas y cualquier efecto que las consuma.

C2 no usa `ON CONFLICT` permisivo ni reintenta dentro de la función. Los
SQLSTATE `23505` y `40001` se propagan sin máscara ni reclasificación. El
cliente autorizado reintenta la transacción exterior completa; una repetición
posterior puede entonces resolver como replay exacto y devolver `false`.

## ACL secuenciada

Al terminar C2, la función conserva ACL propietaria: se revoca de `PUBLIC`,
migrador, emisor, consumidor, runtime y propietarios ajenos. C2 no concede
`USAGE` de esquema ni `EXECUTE` a ContextoActor.

C3 es el único corte autorizado para conceder a
`vec_contexto_actor_v1_propietario` exactamente `USAGE` del esquema y
`EXECUTE` sobre esta firma. No concede tablas, secuencias, tipos ni otras
funciones V3. Así C2 no permite puentear R0 ni las fachadas M6/M7.

## Criterio de cierre de T080

T080 debe acreditar, como mínimo:

- ABI de once entradas y doce salidas, nombres, tipos y orden exactos;
- volatilidad, no estrictez, paralelismo, propietario, `proconfig` y ACL;
- ambos advisory, orden numérico y eliminación de duplicados;
- primera inserción y replay exacto después de caducar, revocar o rotar;
- ausencia de llamada C1 y de revalidación actual en replay;
- colisiones separadas de capacidad, evento, nonce y operación;
- igualdad exacta `acreditada_en=consumida_en` y canon/huella A4;
- doce resultados, `true` inicial y `false` histórico;
- rollback integral y propagación sin máscara de `23505` y `40001`.

Las carreras multisesión globales y el reintento con backoff pertenecen a Q1;
C2 no los simula como éxito local. El corte permanece privado y no cambia las
métricas funcionales ni el `NO-GO` de producción.
