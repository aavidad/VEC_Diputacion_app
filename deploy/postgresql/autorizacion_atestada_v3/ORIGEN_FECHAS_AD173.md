# Fechas y coordenadas nominales del consumo: AD173

AD173 añade la familia `consumo_confirmado_v3`, versión `3`, a la auditoría
común. Su eslabón compromete las fechas de registro y consumo, actor, perfil
activo y finalidad. Sólo cambia el bloque de auditoría de los consumos nuevos
que alcanzan el núcleo MUTACIÓN; las ramas delegadas conservan sus propios
núcleos. No actualiza las filas históricas ni recalcula sus huellas.

El actor procede de `principal_id`, el perfil de `perfil_activo_ref` y la
finalidad de `finalidad`, en la decisión canónica autenticada y revalidada.
El núcleo ya coteja actor y perfil con el contexto canónico. El proceso y el
canal mantienen las autoridades de AD172: configuración técnica exacta y
superficie acreditada. Ninguna coordenada nueva procede de una cabecera o un
parámetro de auditoría enviado por el cliente.

`registrada_en` y `consumida_en` proceden del mismo `v_ahora` del núcleo. La
primera se conserva en `auditoria_consumo_v3`; la segunda, en
`consumo_decision_v3`. La FK de decisión, efecto y huella del efecto permite
unirlas sin copiar ni completar fechas históricas. La proyección para verificar
el eslabón debe incluir ambas fechas.

## Huellas y ABI

La nueva `huella_sha256` corresponde al eslabón de la corriente común. Es
SHA256 del encuadre de estos quince campos, en este orden:

| Posición | Campo |
| --- | --- |
| 1 | `consumo_confirmado_v3` |
| 2 | `3` |
| 3 | Secuencia decimal |
| 4 | `anterior_sha256` |
| 5 | `decision_ref` |
| 6 | `efecto_ref` |
| 7 | `huella_efecto_sha256` |
| 8 | `consumo_huella_sha256` |
| 9 | `proceso` |
| 10 | `canal` |
| 11 | `registrada_en`, UTC6 |
| 12 | `consumida_en`, UTC6 |
| 13 | `actor_ref` |
| 14 | `perfil_activo_ref` |
| 15 | `finalidad_ref` |

`encuadrar_mac` usa longitud decimal en octetos UTF-8, `:`, valor y salto de
línea. Las fechas usan exactamente `YYYY-MM-DDTHH:MM:SS.ffffffZ`, seis decimales,
años `0001` a `9999`, instante finito y zona UTC. Se comprometen las dos
coordenadas aunque contengan el mismo instante. Cambiar cualquiera modifica la
huella. Una proyección con fechas distintas debe rechazarse antes del cotejo.

`consumo_huella_sha256` conserva su cálculo original sobre el material V3.
AD173 no añade fechas a esa huella ni al material firmado. Esta decisión mantiene
el cotejo de replay y la referencia `aud_v3_` seguida de sus primeros 32 hex.
La huella que añade las cinco coordenadas es `huella_sha256` del eslabón.

La firma SQL permanece:

```text
consumir_decision_mutacion_v3_interna(
  text, bytea, bytea, bytea, bytea, numeric, numeric,
  bytea, bytea, bytea, bytea
)
RETURNS TABLE (
  decision_ref text, efecto_ref text, huella_efecto_sha256 text,
  consumo_huella_sha256 text, auditoria_ref text,
  consumida_en timestamptz, consumo_nuevo boolean
)
```

El núcleo conserva OID, propietario, ACL, configuración y dependencias. Mantiene
la autorización viva, los bloqueos, la atestación y el consumo en la transacción
del efecto. El replay retorna por el bloque original y conserva el asiento
v1/v2/v3 que encuentre; no añade otro ni renueva su fecha o procedencia.

## Preimagen e instalación

La candidata requiere PostgreSQL 18 y una de las secuencias causales POST154 →
AD172 o POST168 → AD172. AD172 presupone AD169/AD171 y la genealogía de la variante
acreditadas. AD173 acepta estas parejas exactas del núcleo POST154:

| Representación | Antes, POST154 + AD172 | Después, AD173 |
| --- | --- | --- |
| `pg_get_functiondef` UTF-8 | `92b4245448ffb76aa0792788616e5a779ddd867612e4bd28ab106d95d5356446` | `777f6a6e94c57cfdd8516d082441c1662e303c4b11aa1f187a542a9a11eeb2d6` |
| `prosrc` UTF-8 | `54327be7e866b84d0cd1feff3a54ef6fc58ceec92daaa2277b150da984371fe9` | `528f35de95885283cc9987fc6f2ca8f09db59e61672b10f3c69134b1d91c23f6` |

La variante POST168 procede del núcleo real POST-H9, que conserva las extensiones
posteriores a AD168 y carece de RPT. No se añade RPT ni se aplica AD154 a esta base:

| Representación | Antes, POST168 + AD172 | Después, AD173 |
| --- | --- | --- |
| `pg_get_functiondef` UTF-8 | `32939b59240dab122779c10789c8fca3d5bcd4024b6341b477e61d82020d8c10` | `6c22fdbb165a00c4f37cb2f7dbb7add4e939e5b0134c0c599b9519bfe3b86db9` |
| `prosrc` UTF-8 | `24c003368a265ab7622e87b0c1949f0f13ba0b585a6624ff0729ecce88de066e` | `bbb932ef29375e88645fb524e6aae5fd3059d51cb0dfd9d4952ffd509470534c` |

Sólo POST154 exige la fachada RPT existente. El par definición/fuente completo
selecciona la variante; las anclas sirven únicamente para aplicar el delta
después del cotejo. Se rechaza cualquier otra pareja.

También exige el CHECK validado `auditoria_tipo_disjunto_v3` de AD172: SHA256 de
`pg_get_constraintdef(oid,false)` UTF-8
`55603c422c7c752909d558d8dd238e371719c1e9d0626272e955cde21240c282`,
capturado en el clon real. El nombre o estado validado por sí solos no bastan. Amplía
su condición conservada con la familia v3 y lo denomina
`auditoria_tipo_disjunto_v4`. Mantiene las familias v1/v2, intentos y eventos
existentes. La familia nueva exige las coordenadas nominales y deja nulas las
columnas ajenas; la fecha debe admitir la representación UTC6 descrita.

La reconstrucción sustituye exactamente dos bloques: declaraciones privadas y
preimagen/inserción de auditoría. La migración exige una coincidencia por bloque,
la postimagen esperada y la reversión exacta de ambos en memoria; comprueba que
los metadatos y dependencias restantes no cambian. La guarda rechaza reejecución.
No incluye DOWN.

La lista `deploy/principal/lista_sql_codexl_fecha_consumo_20261003.txt` contiene
sólo AD173. Una base POST174 u otra definición distinta requiere resolver
su genealogía antes del ensayo; no basta el nombre de la función o constraint.
Las dependencias presentes no se reaplican. La lista prepara el ensayo de
dirección y no autoriza una instalación en principal/cidonia.

## Prueba y fixture pendiente

`pruebas_sql/ad173_consumo_fecha_nominal.sql` comprueba ABI, postimagen, ACL,
RLS y disparadores conservados, FK exacta, familias v1/v2/v3, campos obligatorios,
mezclas inadmisibles, límites nominales y rango temporal. Evalúa registros
sintéticos con `jsonb_populate_record` contra el CHECK real, sin insertarlos en
la corriente. El vector UTF-8 y las alteraciones individuales comprueban el
encuadre real de PostgreSQL. El ROLLBACK final revierte la configuración local
de la prueba.

Los diez vectores de `pruebas_sql/ad173_vectores_eslabon.json` son compartidos
con el verificador Go. Incluyen un asiento válido, actor/perfil/finalidad
alterados, ambas fechas alteradas, fechas desiguales, los dos extremos UTC6 y
referencias nominales UTF-8. La prueba SQL también verifica cualquier asiento
v3 real ya presente: vínculo exacto, igualdad de fechas, coordenadas canónicas,
referencia del recibo y SHA256 de los quince campos.

La ejecución estructural no acredita un consumo firmado cuando encuentra cero
filas v3. Para exigirlo, dirección ejecuta en su clon:

```bash
psql -X -v ON_ERROR_STOP=1 -v ad173_exigir_consumo=1 \
  -f deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ad173_consumo_fecha_nominal.sql
```

Ese recorrido necesita un LOGIN técnico permitido por la fachada elegida y su
terna AD172, una decisión/contexto nominal sintéticos que cumplan las autoridades
CA26/IS13 vigentes, material V3 firmado y sus referencias de efecto reales del
clon. Se prepara por los cauces de esas autoridades; no se cambian fechas de
fuentes históricas ni se sustituye una sesión para superar un rechazo. El
material privado queda fuera de Git.

Dirección debe guardar antes del ensayo las filas históricas y cabeza de la
auditoría, los consumos y la fachada RPT; después compara sus huellas, ACL y
definición. Sobre el consumo nuevo debe demostrar replay con el mismo recibo y
fecha, rollback de negocio sin fila/cabeza/consumo adicionales y recuperación
tras reinicio sin duplicados. La prueba preparada no ejecuta esos recorridos.

## Estado de la candidata

Se comprobó fuera de PostgreSQL el SHA256 de los diez vectores y la reconstrucción
exacta POST149 → AD154 → AD172 → AD173, usando la captura real conservada de
POST149. Se verificó también el delta POST168 + AD172 → AD173 a partir de la
captura real del clon de dirección, y el SHA256 de su CHECK POST172. Las
postimágenes coinciden con las tablas anteriores. Esta comprobación
no acredita ejecución PL/pgSQL, instalación ni persistencia.

El ensayo PostgreSQL del hash final corresponde a dirección en su único clon
desechable. Faltan ese ensayo, el fixture firmado positivo, los recorridos de
replay/rollback/reinicio y dos revisiones independientes. AD173 sigue siendo
candidata; no está integrada ni instalada por este encargo.
