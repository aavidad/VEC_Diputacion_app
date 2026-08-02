# Decisión F0-D1: capacidad de fuente corporativa

Fecha: 1 de agosto de 2026.
Estado: **decisión técnica cerrada; implementación y producción NO-GO**.

## Resultado

F0 añadirá a `vec_autorizacion_atestada_v3` un perfil nominal y de un solo uso
para los cuatro efectos C2.3. La fuente posee el hecho; V3 gobierna confianza
y consumo; ContextoActor posee la proyección y su prueba.

Esta decisión concreta el contrato
[C2.3](coordinacion_c2_3_publicacion_revocacion_corporativa_2026-07-31.md)
sin alterar las decisiones de
[contexto corporativo](decision_contexto_corporativo_rrhh_ct_000047c2_2026-07-30.md)
ni de
[organización y vínculo](decision_c2_2_organizacion_y_vinculo_corporativo_2026-07-30.md).

No se reutiliza ni modifica el canon humano de 37 campos. Sus tablas exigen
decisión PDP, motivo y ContextoActor humano, y poseen una FK a
`vec_autorizacion`; adaptarlas con nulos o excepciones crearía una segunda vía
de autoridad. F0 reutiliza solamente las primitivas y el gobierno comunes:

- `texto_json_go`, `encuadrar_mac` y `bytea_igual_constante`;
- `clave_capacidad_version`, `puntero_clave_emision` y sus revocaciones;
- configuración, raíz, puntero y revocaciones de confianza;
- `checkpoint_gobierno` y su serialización frente a revocaciones.

## Catálogo aprobado de fuente

Una capacidad con MAC válido no basta para afirmar qué fuente puede producir
un efecto. `000007` añadirá una relación append-only
`fuente_corporativa_contexto_actor_v1` con una fila por:

```text
(fuente_ref, fuente_version, audiencia_consumo)
```

La fila liga de manera inseparable:

- fuente y versión;
- acción y tipo de efecto del cruce nominal de su audiencia;
- emisor, clave, versión, revisión y huella de gobierno;
- configuración, secuencia y huella;
- raíz, versión, huella SPKI, suite y audiencia de despliegue;
- ventana `[valida_desde,valida_hasta)`, acto aprobatorio y fecha de registro.

La fila es inmutable. Una `fuente_version` puede habilitar un subconjunto de
las cuatro audiencias, pero cambia cuando cambia cualquier binding de una
audiencia habilitada. Dos versiones de la misma fuente y audiencia solo pueden
solaparse mediante ventanas finitas; la anterior pierde autoridad por fin de
ventana o revocación append-only, nunca por selección implícita de la última.

`000007` no añade claves alternativas. El catálogo referencia únicamente las
claves ya existentes:

```text
clave_capacidad_version(clave_id,version)
configuracion_confianza_version(revision)
raiz_confianza_version(clave_id,version)
configuracion_raiz(configuracion_revision,raiz_clave_id,raiz_version)
```

Tras bloquear esas filas, el consumidor deriva y cruza emisor, audiencia,
revisión y huella de gobierno, secuencia y huella de configuración, y huella
SPKI, suite y audiencia de despliegue. Esta ligadura transitiva sobre filas
inmutables impide que una fuente afirme su propia confianza sin duplicar
restricciones `UNIQUE`.

`revocacion_fuente_corporativa_contexto_actor_v1` será append-only y tendrá
como clave la misma terna. Conserva instante, motivo catalogado y acto. Alta y
revocación de fuente avanzan causalmente la fila existente de
`checkpoint_gobierno` mediante un disparador `BEFORE INSERT`; incrementan su
revisión y conservan los máximos de configuración y raíz. No se crea otro
reloj, contador, raíz, almacén de claves ni cadena de auditoría.

No habrá CRUD de fuente en el runtime. El alta inicial será un acto de gobierno
revisado. El mecanismo operativo definitivo de aprobación y revocación es un
bloqueo productivo de Sistemas, DBA, RRHH y DPD.

## Identificadores y límites comunes

`fuente_ref` es opaca, no semántica y no contiene identificadores directos.
Tiene entre 3 y 160 octetos, usa UTF-8 bajo colación `C` y cumple:

```text
^[A-Za-z0-9][A-Za-z0-9_.:/_-]{2,159}$
```

`fuente_version` es un entero decimal entre 1 y
`9007199254740991`. Las versiones del gobierno V3 conservan ese mismo rango.
`operacion_ref` cumple el contrato ContextoActor `oca_` más 24 a 128
caracteres `[A-Za-z0-9_-]`. `evento_fuente_ref` y `efecto_ref` son opacas,
ASCII técnicas y tienen entre 3 y 160 octetos con el mismo alfabeto de
`fuente_ref`. `efecto_ref` es la referencia opaca exacta del recurso/efecto
C2.3; no se añade otro campo, alias o selector de recurso.

Toda huella es SHA-256 hexadecimal minúscula, 64 caracteres y distinta de
cero. Todo instante es UTC finito, `timestamptz(6)`, y su representación
canónica es `YYYY-MM-DDTHH:MM:SS.ffffffZ`. No se aceptan offsets, precisión
distinta, años fuera de 1..9999, infinito, nulos, texto vacío ni Unicode
visualmente equivalente.

## Manifiesto de evento de fuente

El intermediario aprobado verifica el evento largo y produce un manifiesto
minimizado. El manifiesto es un objeto JSON plano, sin identificadores directos
ni payload maestro, con exactamente estos trece campos y este orden:

```text
esquema, version, fuente_ref, fuente_version,
evento_fuente_ref, huella_evento_fuente_sha256,
evento_fuente_emitido_en,
audiencia_consumo, accion, tipo_efecto,
operacion_ref, efecto_ref, huella_efecto_sha256
```

`esquema` es
`vec.contexto-actor.fuente-corporativa.manifiesto.v1` y `version` es el número
JSON `1`. `version` y `fuente_version` son números JSON decimales; los demás
valores son cadenas. `evento_fuente_emitido_en` no puede ser posterior a la emisión
de la capacidad. El manifiesto ocupa entre 128 y 16.384 octetos.

`evento_fuente_ref` identifica una aserción atómica de exactamente una
audiencia y un efecto. Un evento de negocio multiefecto se descompone en
referencias atómicas distintas, firmadas y trazables en la fuente. Reutilizar
el par `(fuente_ref,evento_fuente_ref)` con otro perfil, `fuente_version`,
ligadura de confianza o efecto se rechaza; un sobre multiefecto nunca es replay.

El sobre COSE Sign1 contiene exactamente esos bytes como payload. La mecánica
estricta Ed25519/COSE puede reutilizar
`internal/vec/adapters/seguridad/verificacioncose`; el intermediario no copia
su implementación. El manifiesto, el COSE, la evidencia de verificación y la
SPKI pública se ligan por huellas dentro de la capacidad.
`huella_evento_fuente_sha256` es la huella del evento maestro verificado.
`huella_manifiesto_fuente_sha256` es distinta y se calcula sobre el manifiesto
minimizado firmado.

## Capacidad canónica

La capacidad es otro objeto JSON plano con exactamente estos 33 campos y este
orden:

```text
esquema, version, fuente_ref, fuente_version,
evento_fuente_ref, huella_evento_fuente_sha256,
evento_fuente_emitido_en,
huella_manifiesto_fuente_sha256,
huella_sobre_cose_sign1_sha256,
huella_prueba_confianza_sha256,
audiencia_consumo, accion, tipo_efecto,
operacion_ref, efecto_ref, huella_efecto_sha256,
clave_id, clave_version, revision_gobierno,
huella_gobierno_sha256, emisor_id,
configuracion_revision, configuracion_secuencia,
huella_configuracion_sha256,
raiz_clave_id, raiz_version, huella_raiz_spki_sha256,
audiencia_despliegue, suite, nonce, emitida_en, expira_en,
mac_sha256
```

`esquema` es
`vec.contexto-actor.fuente-corporativa.capacidad.v1`; `version` es el número
JSON `1`. Exactamente `version`, `fuente_version`, `clave_version`,
`revision_gobierno`, `configuracion_secuencia` y `raiz_version` son números
JSON decimales. `configuracion_revision` es la cadena que referencia
`configuracion_confianza_version.revision` de `000001`; los demás campos son
cadenas. No se admiten claves desconocidas o repetidas, nulos, objetos
anidados, arrays, fracciones, exponentes, BOM ni espacios fuera de cadenas.

Un ayudante nuevo y nominal reconstruye ambos cánones con `texto_json_go` y
exige igualdad byte a byte con la entrada. No se generalizan ni alteran
`capacidad_cruda_prevalida`, `capacidad_canonica` o `preimagen_mac` del perfil
humano.

La preimagen MAC usa el dominio exacto
`VEC-CONTEXTO-ACTOR-FUENTE-CORPORATIVA-V1`, seguido de los primeros 32 valores
en el orden anterior, cada uno encuadrado por `encuadrar_mac`. El MAC es
HMAC-SHA-256 con la clave V3 aprobada y se compara en tiempo constante. No se
acepta una preimagen, canon o huella calculados por el llamante.

La capacidad ocupa entre 512 y 32.768 octetos. `nonce` es una huella aleatoria
de 64 hexadecimales no nula. Su vigencia es `[emitida_en,expira_en)`, positiva
y de cinco segundos como máximo, contenida en la vigencia de clave,
configuración, raíz y fuente. `suite` es
`VEC-AD-3-COSE-EDDSA-1` y debe coincidir con la raíz catalogada.
La suite solo fija la mecánica criptográfica: el aislamiento de fuente exige
además audiencia de despliegue, raíz y fila de catálogo específicas.

## Cruce nominal exhaustivo

| Audiencia de consumo | Acción | Tipo de efecto | Rol de sesión |
| --- | --- | --- | --- |
| `vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1` | `contexto_actor.organizacion_corporativa.publicar` | `organizacion_corporativa.alta` | publicador |
| `vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1` | `contexto_actor.organizacion_corporativa.revocar` | `organizacion_corporativa.revocacion` | revocador |
| `vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1` | `contexto_actor.vinculo_corporativo.publicar` | `vinculo_corporativo.alta` | publicador |
| `vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1` | `contexto_actor.vinculo_corporativo.revocar` | `vinculo_corporativo.revocacion` | revocador |

`000007` amplía el `CHECK` de `clave_capacidad_version.audiencia_consumo` de
las tres audiencias existentes a esas siete y ninguna más. Audiencia, acción,
tipo, sesión técnica, catálogo, manifiesto, capacidad y efecto esperado deben
coincidir en una sola fila de la tabla. No existe un selector libre ni una
respuesta que enumere alternativas.

## Fachada y resultado

La firma final F0 será:

```sql
vec_autorizacion_atestada_v3.
consumir_fuente_corporativa_contexto_actor_v1_atestada(
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
```

C2 la crea con ACL exclusivamente propietaria; C3 es el único corte que la
expone a las fachadas finales. Su ABI, locks, replay y errores quedan fijados
en la [decisión C2](decision_f0_c2_consumidor_nominal_2026-08-02.md).

M6/M7 derivan únicamente audiencia, acción, tipo, operación, `efecto_ref` y
huella de efecto esperados desde constantes y filas bloqueadas. F0 extrae
fuente, versión, evento y huella de evento desde capacidad y manifiesto,
calcula las huellas de capacidad y manifiesto, cruza todo con el catálogo
duradero y lo devuelve. Ningún login, DTO ni inbox previo aporta o escoge
autoridad.

Devuelve las doce columnas allí enumeradas, sin instante del evento ni nonce.
`capacidad_ref` se deriva como `cfc_` más el SHA-256 de la capacidad; nunca
llega del cliente.

El manifiesto ocupa 128..16.384 octetos; COSE, 128..65.536; evidencia,
32..262.144; SPKI, exactamente 44; y capacidad, 512..32.768. La frontera de
transporte limita el mensaje antes de PostgreSQL y la función vuelve a aplicar
los límites antes de cualquier conversión, copia, hash o reserva adicional.

## Persistencia y minimización

`000007` crea dos historias propias:

1. `atestacion_fuente_corporativa_contexto_actor_v1`, con PK
   `capacidad_ref` —que ya incorpora SHA-256— y unicidad durable exacta
   `(fuente_ref,evento_fuente_ref)`; conserva la `fuente_version` consumida;
2. `consumo_fuente_corporativa_contexto_actor_v1`, 1:1 por capacidad, con
   `nonce` y `operacion_ref` únicos, conserva canon, huella e instante.

Cualquier coincidencia de capacidad, evento estable, nonce u operación con
artefactos o coordenadas no idénticos es una colisión cerrada, no un replay.

El par forma la prueba local V3; no se añade una segunda cadena de auditoría.
La prueba de efecto de ContextoActor pertenece a M5 y solo retiene las
referencias y huellas devueltas, nunca capacidad, MAC, COSE, SPKI, evidencia o
payload de fuente.

Todas las relaciones son `PERMANENT`, append-only, con RLS activada y forzada,
y rechazan `UPDATE`, `DELETE` y `TRUNCATE`. `PERMANENT` descarta tablas
temporales o `UNLOGGED`; no impone conservación indefinida. No se guardan
identificadores directos, secretos HMAC ni el payload maestro. Las referencias
de evento y efecto de vínculo son datos personales seudonimizados: exigen
cifrado de soporte, registro de acceso y conservación, bloqueo y expurgo
gobernados. Los logs solo reciben códigos y correlaciones opacas.

El canon de consumo es JSON UTF-8 estricto, reconstruido con `texto_json_go`,
con este orden exacto:

```text
esquema, version, capacidad_ref, fuente_ref, fuente_version,
evento_fuente_ref, huella_evento_fuente_sha256, evento_fuente_emitido_en,
huella_manifiesto_fuente_sha256, huella_sobre_cose_sign1_sha256,
huella_prueba_confianza_sha256, audiencia_consumo, accion, tipo_efecto,
operacion_ref, efecto_ref, huella_efecto_sha256, clave_id, clave_version,
revision_gobierno, huella_gobierno_sha256, emisor_id,
configuracion_revision, configuracion_secuencia, huella_configuracion_sha256,
raiz_clave_id, raiz_version, huella_raiz_spki_sha256,
audiencia_despliegue, suite, nonce, emitida_en, expira_en, mac_sha256,
consumida_en
```

`esquema` es `vec.contexto-actor.fuente-corporativa.consumo.v1` y `version`
es `1`. Los seis campos numéricos son exactamente los mismos de la capacidad;
`configuracion_revision` y los demás son cadenas. `consumo_huella_sha256` es
SHA-256 de esos bytes. Canon y huella se persisten; vectores SQL/Go prueban
cada campo, orden y límite.

## ACL y roles

La función es `SECURITY DEFINER`, propietaria de
`vec_autorizacion_atestada_v3_propietario`, con
`search_path=pg_catalog` y `lock_timeout=2s`. C2 la deja privada. Solo C3
concede a `vec_contexto_actor_v1_propietario` `USAGE` de esquema y `EXECUTE`
de esa firma para anidarla en M6/M7. No recibe tablas, secuencias, tipos ni
otras funciones V3; el inventario rechaza cualquier consumidor adicional.

R0, después de F0/C3, creará estos grupos `NOLOGIN NOINHERIT`:

```text
vec_contexto_actor_v1_publicador_corporativo
vec_contexto_actor_v1_revocador_corporativo
vec_contexto_actor_v1_despachador_corporativo
```

La [decisión C2](decision_f0_c2_consumidor_nominal_2026-08-02.md) fija la forma
completa de los grupos, el `LOGIN` y su arista. C2 se instala sin R0, pero
reacredita `session_user` en cada llamada y deniega hasta que R0 y Sistemas
completen esa topología. Publicación exige solo el grupo publicador;
revocación, solo el revocador. Cruce, membresía adicional o despachadora y
`SET ROLE` deniegan.

C2 queda privada y C3 concede solo al propietario ContextoActor. Los grupos y
los `LOGIN` nunca obtienen `EXECUTE` directo sobre F0. Propietario,
configuración y ACL se reacreditan; `RLS` es únicamente defensa adicional.

## Transacción, locks y replay

F0 solo funciona dentro de la misma transacción exterior
`SERIALIZABLE READ WRITE` de C2.3, en UTC, con `statement_timeout` entre 1 y
10 segundos, `transaction_timeout` e
`idle_in_transaction_session_timeout` entre 1 y 15 segundos. Nunca abre ni
confirma una transacción.

La [decisión C2](decision_f0_c2_consumidor_nominal_2026-08-02.md) fija los
advisory exactos y el orden numérico. Tras checkpoint y advisory, C2 busca
primero historia. El replay exacto devuelve la prueba persistida sin llamar a
C1 ni revalidar el estado actual. Solo el camino nuevo ejecuta C1, A4 y B2.

Las altas y revocaciones de fuente, clave, configuración o raíz bloquean
primero el mismo checkpoint. Por ello no pueden ganar entre la revalidación y
el `COMMIT`.

El primer consumo devuelve `consumo_nuevo=true`; el replay histórico exacto,
`false`. Reutilizar operación, capacidad, evento o nonce con otra preimagen
colisiona cerrada. Las cuatro fachadas exigen `true` al crear un efecto nuevo.

Si falla cualquier escritura posterior de C2.3, PostgreSQL revierte también
atestación y consumo. Tras `COMMIT` incierto se reconcilia primero la prueba
local de C2.3 bajo sus mismos locks: si es exacta, el efecto confirmó y no se
llama a F0; si no existe, el consumo tampoco puede existir y se repite con los
mismos bytes. No se usa cola, compensación ni recibo tardío.

## Migración y retirada

El paquete de implementación admite estos artefactos acotados:

```text
deploy/postgresql/autorizacion_atestada_v3/migraciones/
  000007_fuente_corporativa_contexto_actor_v1.up.sql
  000007_fuente_corporativa_contexto_actor_v1.down.sql
  000007_componentes/
    010_validadores.sql
    020_canon_manifiesto.sql
    030_canon_capacidad_mac.sql
    040_canon_consumo.sql
    050_catalogo_fuente_checkpoint.sql
    060_atestacion_consumo.sql
    070_acreditar_material_fuente.sql
    080_consumidor_nominal.sql
    090_acl_audiencias_centinela.sql
    810_acreditar_retirada.sql
    820_retirar_objetos.sql
    830_restaurar_audiencias.sql
deploy/postgresql/autorizacion_atestada_v3/
  README.md
  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  arnes_fuente_corporativa_contexto_actor_v1.sh
  operaciones_runner_fuente_corporativa_contexto_actor_v1.sh
  capturar_snapshot_fuente_corporativa_contexto_actor_v1.go
  fuente_corporativa_contexto_actor_v1.sql
  000007_componentes/
    010_validadores.sql
    020_canon_manifiesto.sql
    030_canon_capacidad_mac.sql
    040_canon_consumo.sql
    050_catalogo_fuente_checkpoint.sql
    060_atestacion_consumo.sql
    070_acreditar_material_fuente.sql
    080_consumidor_nominal.sql
    090_acl_audiencias_centinela.sql
    100_estructura_acl.sql
    110_consumo_replay_rollback.sql
    810_acreditar_retirada.sql
    820_retirar_objetos.sql
    830_restaurar_audiencias.sql
    900_concurrencia_consumo_revocacion.sh
    910_retirada_dependencias_componentes.sh
    920_regresion_consumidores_v3.sql
internal/vec/adapters/seguridad/confianzaatestacion/
  capacidad_fuente_corporativa_v1_vector_test.go
internal/vec/adapters/seguridad/confianzaatestacion/testdata/
  manifiesto_fuente_corporativa_v1.json
  capacidad_fuente_corporativa_v1.json
  consumo_fuente_corporativa_v1.json
```

El README pertenece al corte final I0 de composición; D1 y D2 no lo modifican.
Cada artefacto enumerado, incluidos componentes, envoltorios, runner, prueba
focal y oráculo Go, queda por debajo de 800 líneas. Los componentes pueden
confirmarse dormidos y probarse directamente en PostgreSQL 18.4, pero no son
migraciones instalables ni los invoca ningún runner o migración anterior. El
oráculo Go termina en `_test.go`: es solo una prueba independiente de vectores,
sin API, autoridad ni ruta productiva. El capturador Go separado también es
exclusivamente probatorio y no se enlaza a ningún binario del producto.

Los tres JSON son fixtures canónicos compartidos y completamente sintéticos:
no contienen datos personales, credenciales, claves ni secretos. El oráculo Go
los decodifica a estructuras cerradas y los vuelve a serializar con
`encoding/json`; la prueba SQL reconstruye sus cánones desde campos tipados.
Ambos resultados deben igualar byte a byte los JSON y sus SHA-256, sin
literales canónicos duplicados. El material HMAC sintético se inyecta o deriva
solo en el runner y el oráculo y nunca se guarda en esos JSON.

## Grafo cerrado de implementación

No se admiten nodos, comodines ni rutas implícitas. En la tabla, `M` es
`deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes` y
`T` es
`deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes`.
La ruta indicada es el write-set completo de cada nodo.

| Nodo | Write-set exacto | Dependencia | Criterio focal |
|---|---|---|---|
| H0 | `deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/arnes_fuente_corporativa_contexto_actor_v1.sh`, `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/operaciones_runner_fuente_corporativa_contexto_actor_v1.sh` y `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/capturar_snapshot_fuente_corporativa_contexto_actor_v1.go` | D2d | PostgreSQL 18.4 por digest, `max_prepared_transactions=0`, `000001..000006`, línea base, snapshot exacto, propiedad y limpieza de recursos, analizador positivo/negativo y clasificación SQLSTATE |
| H0a | `deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh` | H0 | [Contrato H0a][decision-h0a] |
| H0b | `deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh` y `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/arnes_fuente_corporativa_contexto_actor_v1.sh` | H0a | [Contrato H0b][decision-h0b] |
| V0 | `internal/vec/adapters/seguridad/confianzaatestacion/capacidad_fuente_corporativa_v1_vector_test.go`, `internal/vec/adapters/seguridad/confianzaatestacion/testdata/manifiesto_fuente_corporativa_v1.json`, `internal/vec/adapters/seguridad/confianzaatestacion/testdata/capacidad_fuente_corporativa_v1.json` e `internal/vec/adapters/seguridad/confianzaatestacion/testdata/consumo_fuente_corporativa_v1.json` | D2b | `encoding/json`, igualdad byte a byte y cero código productivo |
| A1 | `M/010_validadores.sql` y `T/010_validadores.sql` | H0a | UTF-8, identificadores, números, instantes y límites |
| A2 | `M/020_canon_manifiesto.sql` y `T/020_canon_manifiesto.sql` | A1+V0 | 13 campos, adversariales y vector dorado |
| A3 | `M/030_canon_capacidad_mac.sql` y `T/030_canon_capacidad_mac.sql` | A1+V0 | 33 campos, seis números, preimagen, HMAC y vector dorado |
| A4 | `M/040_canon_consumo.sql` y `T/040_canon_consumo.sql` | A1+V0 | canon y huella sin aceptar canon del cliente |
| B1 | `M/050_catalogo_fuente_checkpoint.sql` y `T/050_catalogo_fuente_checkpoint.sql` | A1 | FK, revocación append-only, RLS, checkpoint y audiencias 3→7 |
| B2 | `M/060_atestacion_consumo.sql` y `T/060_atestacion_consumo.sql` | B1 | dos historias, unicidades, inmutabilidad y minimización |
| C1 | `M/070_acreditar_material_fuente.sql` y `T/070_acreditar_material_fuente.sql` | A2+A3+B1 | función privada, locks y cruces fuente–clave–configuración–raíz |
| C2 | `M/080_consumidor_nominal.sql` y `T/080_consumidor_nominal.sql` | H0b+C1+A4+B2 | [Contrato exacto][decision-c2] |
| C3 | `M/090_acl_audiencias_centinela.sql` y `T/090_acl_audiencias_centinela.sql` | C2 | [Contrato C3][decision-c3] |
| R1 | `M/810_acreditar_retirada.sql` y `T/810_acreditar_retirada.sql` | C3 | inventario, deriva, datos y dependencias |
| R2a | `M/820_retirar_objetos.sql` y `T/820_retirar_objetos.sql` | R1 | retirada inversa con `RESTRICT` y centinela, todavía con siete audiencias |
| R2b | `M/830_restaurar_audiencias.sql` y `T/830_restaurar_audiencias.sql` | R2a | audiencias 7→3 y base exacta |
| T1 | `T/100_estructura_acl.sql` | C3 | estructura y ACL completas |
| T2 | `T/110_consumo_replay_rollback.sql` | C3 | consumo, replay y rollback completos |
| P0 | `deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_fuente_corporativa_contexto_actor_v1.up.sql` y `deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_fuente_corporativa_contexto_actor_v1.down.sql` | H0+H0a+H0b+V0+A1+A2+A3+A4+B1+B2+C1+C2+C3+R1+R2a+R2b+T1+T2 | única composición instalable, transacción y txid únicos, orden cerrado y entrada solo por runner |
| Q1 | `T/900_concurrencia_consumo_revocacion.sh` | P0 | carreras, modos reales de FK al crear/retirar, `pg_locks`, contención, cero upgrades por componente, clasificación, reintento completo, backoff y agotamiento |
| Q2 | `T/910_retirada_dependencias_componentes.sh` | P0 | retirada, dependencias y componentes adversos |
| Q3 | `T/920_regresion_consumidores_v3.sql` | P0 | regresión de consumidores V3 existentes |
| I0 | `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/fuente_corporativa_contexto_actor_v1.sql`, `deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh` y `deploy/postgresql/autorizacion_atestada_v3/README.md` | Q1+Q2+Q3 | composición final, tres pasadas limpias y documentación operativa |

[decision-h0a]: decision_f0_h0a_guardia_autoprueba_sintetica_2026-08-01.md
[decision-h0b]: decision_f0_h0b_r0_sintetico_c2_2026-08-02.md
[decision-c2]: decision_f0_c2_consumidor_nominal_2026-08-02.md
[decision-c3]: decision_f0_c3_acl_audiencias_centinela_2026-08-02.md

El DAG cerrado es:

```text
D2b -> V0, D2c
D2c -> D2d
D2d -> H0
H0 -> H0a -> H0b
H0a -> A1
A1 + V0 -> A2, A3, A4
A1 -> B1
B1 -> B2
A2 + A3 + B1 -> C1
H0b + C1 + A4 + B2 -> C2
C2 -> C3
C3 -> R1, T1, T2
R1 -> R2a
R2a -> R2b
H0 + H0a + H0b + V0 + A1 + A2 + A3 + A4 + B1 + B2 + C1 + C2 + C3 + R1 + R2a + R2b + T1 + T2 -> P0
P0 -> Q1, Q2, Q3
Q1 + Q2 + Q3 -> I0
```

H0 crea el runner base, sus auxiliares privados y el capturador del snapshot.
H0a y H0b son sus dos únicos escritores correctivos posteriores: H0a limita la
autoprueba sintética y H0b añade los dos subensayos R0 y actualiza la huella
literal del arnés. Después de H0b, solo I0 modifica el runner tras Q1–Q3; no
modifica auxiliares ni capturador. No existe otro write-set autorizado.

A1 queda integrado en `169a055` tras corregir un `NO-GO` de cobertura
catalogal y semántica nula. La
[revisión final](revisiones/revision_f0_a1_validadores_2026-08-01.md) obtuvo
doble `GO`, `P0=P1=P2=0`, y desbloquea A2, A3, A4 y B1 como write-sets
disjuntos. Todavía no existe una composición `000007` instalable.

### Corrección D2c: arnés privado y snapshot exacto

La primera implementación exploratoria demostró que concentrar parser,
clasificador, inventario, snapshot, etapas, bootstrap, reintentos y
orquestación final en un solo shell obligaba a elegir entre superar 800 líneas
o comprimir código de seguridad hasta volverlo difícil de revisar. D2c no
cambia el protocolo ni añade una capacidad: separa responsabilidades dentro
del mismo runner de prueba.

`arnes_fuente_corporativa_contexto_actor_v1.sh` es un helper privado y
exclusivamente de pruebas. No tiene modo autónomo, no es ejecutable operativo,
no abre contenedores ni sesiones y falla si se invoca en vez de cargarse. El
runner conoce su ruta fuente como literal no sustituible, la entrega al
capturador y hace `source` exclusivamente de la copia privada acreditada;
además contiene su SHA-256 esperado literal y exige que la copia lo iguale
antes de cargarla. Nunca carga el helper desde el árbol vivo. El helper
contiene las funciones nominales de análisis léxico, clasificación SQLSTATE,
clausuras y rutas literales de etapa, inventario, snapshot y huellas. No
contiene credenciales, datos, Docker, reintentos ni decisiones de ejecución.

#### Enmienda D2d: auxiliar operativo privado

La revisión adversarial de H0 detectó una ventana de propiedad alrededor de
`docker run`. La [decisión D2d](decision_f0_d2d_auxiliar_operativo_2026-08-01.md)
la cierra mediante un auxiliar privado de ciclo de vida, token e intención
previos, etiqueta, identificador y `cidfile` cruzados, y liberación solo tras
acreditar ausencia. D2d no cambia el protocolo F0 ni añade una capacidad
productiva.

El runner H0 debe quedar como máximo en 550 líneas para reservar a I0 al menos
249 líneas sin superar el límite estricto de 799. El auxiliar operativo debe
quedar por debajo de 200 líneas; el helper SQL y el capturador, por debajo de
800. Prima la legibilidad sobre la compactación artificial. H0 prueba los tres
shells con ShellCheck, analiza la copia acreditada
del capturador con `go vet`, la compila con `go build -race` y ejecuta ese
binario privado con `--autoprueba`; prueba además que ejecutar directamente
cualquiera de los dos auxiliares falla cerrado. I0 conserva en el runner el bootstrap, R0 e
identidades sintéticas,
P0, los reintentos, Q1–Q3, el oráculo V0, las tres pasadas y el cierre
operativo; por eso no puede gastar su reserva durante H0.

El propio capturador arranca con una cadena de confianza cerrada. El runner
contiene como literal la ruta y el SHA-256 esperado e inmutable de su fuente;
con `umask 077` la copia a un directorio privado `0700` y a un destino nuevo
`0600` con exclusión, sin seguir enlaces, rechazando enlaces duros y
contrastando tipo, dispositivo, inode, tamaño y tiempos antes y después.
La lectura y copia de la fuente viva del capturador y de ambos auxiliares se
limitan antes de recorrer, copiar, hashear o reservar; una mutación creciente
solo puede consumir el máximo permitido más un byte de sonda. Solo si las
copias privadas igualan sus huellas literales se analizan, compilan o cargan. La
compilación usa exclusivamente el toolchain local y la biblioteca estándar,
con `GOTOOLCHAIN=local`, módulos en solo lectura y red de módulos desactivada;
nunca usa `go run` ni vuelve a leer la fuente viva. La autoprueba se ejecuta
desde ese binario privado compilado con detector de carreras. I0 preserva sin
cambios las tres huellas y los tres artefactos inmutables.

No se copia el repositorio completo al contenedor. Para cada ejecución, el
helper SQL resuelve un inventario cerrado de ficheros base y, cuando proceda, la
clausura literal de la etapa. El runner entrega exclusivamente ese inventario
al capturador Go; no usa `cp`, `tar`, glob ni descubrimiento dinámico sobre el
árbol vivo.

El capturador es una herramienta `package main` solo de prueba, sin API ni
código enlazado al producto. El runner entra una sola vez en la raíz física
del repositorio y la retiene como directorio de trabajo; el capturador solo
acepta `.` como raíz y la abre mediante `os.OpenRoot(".")`, por lo que no
reabre una ruta absoluta sustituible. Contrasta el directorio retenido antes y
después y solo admite rutas relativas locales, limpias, únicas y enumeradas.
Desciende un componente cada vez con `OpenRoot`, exige
mediante `Lstat` un directorio real y no simbólico, compara ese resultado con
el `fstat` del directorio abierto y conserva todos sus descriptores hasta el
final. Para el fichero exige un archivo regular no enlazado; lo abre bajo la
última raíz, conserva el descriptor, rechaza `st_nlink != 1` y acredita con
`fstat` antes y después el mismo dispositivo, inode, tipo, tamaño, `mtime`,
`ctime` y número de enlaces. Al terminar repite y coteja `Lstat` de cada
componente con los descriptores retenidos.
Copia y calcula SHA-256 en un solo flujo desde **ese mismo descriptor** hacia
un destino nuevo creado con exclusión. Repite `Lstat` al finalizar y exige que
la ruta siga resolviendo al mismo inode. Una mutación o sustitución antes o
durante la lectura falla; nunca se vuelve a abrir por nombre el contenido que
se validará.

Los componentes se analizan en ese snapshot, nunca en el árbol vivo. Se
genera un manifiesto ordenado `ruta relativa + SHA-256`, se transfiere
solamente el snapshot y se reconstruye el mismo manifiesto dentro del
contenedor. Psql no arranca hasta acreditar igualdad byte a byte y ejecuta
solo esos ficheros. Los dos auxiliares que cargará el runner se obtienen
previamente mediante el mismo capturador y se cargan desde el
descriptor/snapshot ya acreditado, no desde sus rutas originales. El
envoltorio efímero generado por el runner recibe la misma ligadura. Una ruta
ausente, adicional, duplicada, enlazada, fuera de la raíz, mutada o con huella
distinta falla antes de SQL y la limpieza sigue siendo obligatoria.

El capturador incorpora una autoprueba determinista, sin temporizadores, que
coordina mediante canales la sustitución antes y durante la lectura. Cubre
fichero normal, enlace simbólico, enlace duro, renombre/reemplazo, mutación del
mismo inode y cambio de un componente de directorio. H0 exige que todas las
carreras sean rechazadas y que el destino parcial no pueda ser consumido.

Las pruebas negativas del parser incluyen por separado `ABORT`, `ABORT WORK`,
`ABORT TRANSACTION` y variantes con espacios o comentarios, además de los
controles ya enumerados. El split no relaja `ON_ERROR_STOP`, `txid_current()`,
`ROLLBACK`, `max_prepared_transactions=0`, las huellas de catálogo y roles ni
la entrada única mediante el runner.

Los dos envoltorios `000007` son los únicos puntos instalables. No se publican
ni integran en la cadena de migraciones hasta que `up`, `down`, todos sus
componentes, el oráculo y el runner estén completos y hayan superado revisión
independiente. Cada envoltorio abre una única transacción y carga sus
componentes con `\ir`, por rutas relativas y en orden determinista. En todo
envoltorio instalable o de ensayo, la primera línea operativa es
`\set ON_ERROR_STOP on`; después fija `\set VERBOSITY sqlstate`, deshabilita
el autocommit con `\set AUTOCOMMIT off` y solo entonces ejecuta `BEGIN`. Antes
y después de cada `\ir`, también en los envoltorios `up` y `down`, acredita el mismo
`txid_current()`; todo el SQL incluido queda dentro de esa transacción. El
runner H0/I0 es el único punto de entrada operativo y refuerza el contrato con
`psql -X -v ON_ERROR_STOP=1 -f`; se prohíbe ejecutar directamente envoltorios
o componentes. Si falta, cambia de forma adversa, se reordena o falla cualquier
componente, psql detiene la lectura y se revierte el paquete entero, sin
instalación parcial. Ningún componente se considera migración instalable ni
capacidad completa por separado.

Cada minitarea dormida tiene una regla de cierre obligatoria. Parte de una
instancia PostgreSQL 18.4 efímera y nueva con `000001..000006` instaladas;
registra como línea base el catálogo, las tres audiencias y todos los bytes de
`checkpoint_gobierno`; abre un `BEGIN` exclusivo de ensayo; carga mediante
`\ir` solo la clausura de componentes requerida hasta esa etapa; ejecuta sus
aserciones focales; y termina siempre con `ROLLBACK`. Inmediatamente antes de
cada `\ir` guarda `txid_current()` y después acredita el mismo valor. El camino
nominal exige el `ROLLBACK` explícito y el camino de error fuerza el rollback
al cerrar la sesión; un cierre por desconexión no convierte el ensayo en
aprobado. Después, desde una sesión de control independiente, acredita cero
objetos F0, audiencia exactamente igual a sus tres valores iniciales,
`checkpoint_gobierno` idéntico byte a byte y ausencia de roles, sesiones u
objetos temporales creados por el ensayo. Acredita además que
`pg_prepared_xacts` permanece vacío; H0 arranca PostgreSQL con
`max_prepared_transactions=0`, por lo que ni una evasión puede dejar una
transacción preparada. Cualquier residuo impide cerrar la minitarea.

La función shell `validar_componentes_sql_f0` pertenece exclusivamente a H0 y
se ejecuta antes de abrir psql. Acredita primero tipo y tamaño con una sonda
acotada; rechaza el fichero vacío o mayor de un mebibyte antes de contar líneas
o ejecutar `awk`. Usa después un analizador léxico acotado y probado que
reconoce cadenas simples y `E''`, identificadores dobles, comentarios de línea,
comentarios de bloque anidados y cuerpos con delimitador dólar. Rechaza todo
metacomando psql al inicio lógico de línea y, solo en nivel SQL superior, las
sentencias `BEGIN`, `START TRANSACTION`, `COMMIT`, `END` transaccional,
`ROLLBACK`, `SAVEPOINT`, `RELEASE SAVEPOINT`, `PREPARE TRANSACTION`,
`COMMIT PREPARED`, `ROLLBACK PREPARED` y `ABORT`. H0 incluye casos positivos y
negativos del propio analizador: acepta esos textos dentro de cadenas,
comentarios y cuerpos PL/pgSQL, y rechaza sus variantes ejecutables y evasiones
por espacios o comentarios. Solo los envoltorios son dueños de la transacción.
La comprobación de `txid_current()` impide que una evasión del validador pueda
cerrar, sustituir o preparar la transacción sin ser detectada.

La clasificación de reintentos no interpreta mensajes humanos. El runner fija
`LC_ALL=C`, captura por separado la salida de error con verbosidad `sqlstate` y
la función shell `clasificar_sqlstate_psql_f0` solo acepta un fallo psql de
script que contiene exactamente un SQLSTATE admitido y ningún segundo error.
Emite un único registro `sqlstate=55P03`, `sqlstate=40P01` o
`sqlstate=invalido`; la política de reintento consume solo ese registro. H0
prueba positivos, negativos, salida vacía, múltiples SQLSTATE y salida
truncada. Cualquier ambigüedad se trata como error semántico no reintentable.

En `up` y `down`, tras abrir la transacción y fijar su configuración local, el
catálogo de locks es cerrado, exhaustivo y no se descubre dinámicamente. Se
intenta primero la barrera común
`vec_autorizacion_atestada_v3:migraciones:v1` y después la propia
`vec_autorizacion_atestada_v3:migracion:000007`, ambas sin espera mediante
`pg_try_advisory_xact_lock`; un resultado falso produce `55P03`. Después se
preadquieren, siempre con `NOWAIT`, sin inventario previo y en este orden
exacto, los modos finales que conservarán hasta rollback o commit:

1. `clave_capacidad_version` en `ACCESS EXCLUSIVE`;
2. `puntero_clave_emision` en `SHARE`,
   `configuracion_confianza_version`, `raiz_confianza_version` y
   `configuracion_raiz` en `SHARE ROW EXCLUSIVE`, y
   `puntero_configuracion_actual` en `SHARE`, en ese orden; los modos más
   fuertes son los exigidos al crear las FK desde F0;
3. `checkpoint_gobierno` en `SHARE`, sin lock de fila;
4. solo en `down`, `fuente_corporativa_contexto_actor_v1`,
   `revocacion_fuente_corporativa_contexto_actor_v1`,
   `atestacion_fuente_corporativa_contexto_actor_v1` y
   `consumo_fuente_corporativa_contexto_actor_v1`, en ese orden y en
   `ACCESS EXCLUSIVE`;
5. `revocacion_clave_capacidad`, `revocacion_configuracion` y
   `revocacion_raiz` en `ACCESS EXCLUSIVE`, exactamente en el orden de
   `000006` y como últimas relaciones del plan.

No hay ascensos ni adquisición posterior de un modo más fuerte sobre
relaciones preexistentes. Los locks automáticos posteriores solo pueden ser
iguales o más débiles que el ya preadquirido y no amplían su matriz de
conflictos; Q1 lo acredita por componente. No hay espera circular. Solo tras
adquirir el plan completo se inventaría el catálogo, se acredita la
forma exacta de `000001..000006` y la ausencia o presencia esperada de `000007`,
y se manipula el centinela. `checkpoint_gobierno` nunca se toma `FOR UPDATE`
durante el DDL: su `SHARE NOWAIT` es compatible con el `ROW SHARE` del
consumidor y bloquea actualizaciones; sus bytes se leen únicamente después de
cerrar con el plan completo todas las fuentes que pueden mutarlo. Esto corta el
ciclo histórico checkpoint→clave→revocación del consumidor V3 y serializa con
sus sesiones, con el DML revocación→checkpoint y con `000006.down`.

El DDL aprobado no referencia por FK ninguna otra relación existente. Si el
write-set cambia, D2b se reabre antes de implementar. Q1 acredita en PostgreSQL
18.4, mediante `pg_locks` y sesiones de contención, que cada modo preadquirido
es el final y que ningún componente ni el `CREATE`/`DROP` real de las FK provoca
después un modo más fuerte. La reproducción previa en PostgreSQL 18.4 de
`CREATE TABLE` hija con `REFERENCES` observó `AccessShareLock` y
`ShareRowExclusiveLock` sobre la tabla padre; por eso las tres relaciones
referenciadas se preadquieren directamente en `SHARE ROW EXCLUSIVE`, no en
`SHARE`.

El runner reintenta el envoltorio completo un máximo de tres intentos, en un
proceso psql, sesión y transacción nuevos cada vez, con pausas acotadas de 100
y 400 milisegundos. Solo `55P03` o `40P01` permiten reintento. Nunca se
reintenta un componente aislado, un error semántico, una precondición fallida,
un artefacto ausente o mutado ni otro SQLSTATE. El agotamiento deja el mismo
estado byte a byte que la línea base y finaliza en fallo.

`up` crea además en `revocacion_raiz` el trigger reservado
`dependencia_f0_fuente_corporativa_v1`, dirigido por `tgfoid` a
`serializar_revocacion_consultas_rrhh_v3()`, y lo deja con `tgenabled='D'`.
Es un centinela catalogal real: nunca se ejecuta ni incrementa dos veces el
checkpoint, pero crea exactamente un `pg_depend` normal (`deptype='n'`) a la
función. `000006.down.sql` detecta esa dependencia extra en su prevalidación.
Aunque se intentara borrar primero un trigger normal,
`DROP FUNCTION ... RESTRICT` fallaría y el rollback lo conservaría. Consumidor
y `down` exigen nombre, tabla, función, dependencia y estado deshabilitado
exactos. El artefacto histórico `000006` permanece sin cambios.

`down` solo retira una instalación vacía y exacta. Deniega ante catálogo o
revocaciones de fuente, atestaciones, consumos, claves/punteros de cualquiera
de las cuatro audiencias, dependencias C2.3, funciones posteriores, grants,
propietarios, ACL, políticas, disparadores, OID u objetos no inventariados.
Elimina primero el centinela con `RESTRICT`; después retira, en este orden,
`consumo_fuente_corporativa_contexto_actor_v1`,
`atestacion_fuente_corporativa_contexto_actor_v1`,
`revocacion_fuente_corporativa_contexto_actor_v1` y
`fuente_corporativa_contexto_actor_v1`; por último restaura el `CHECK` exacto
de las tres audiencias anteriores. Nunca usa
`CASCADE`. Después, `000006.down.sql` puede acreditar y retirar su instalación
vacía. Cualquier rechazo revierte la retirada completa y conserva evidencia.

## Prueba de cierre de implementación

El runner focal V3 no cuenta como cuarto runner ContextoActor. Usará
PostgreSQL 18.4 efímero, sin puertos ni secretos, e instalará R0 sintético solo
para probar la futura llamada anidada. Debe cubrir:

- los tres cánones, vectores SQL/Go, límites y los cuatro cruces nominales;
- aislamiento distinto de `SERIALIZABLE READ WRITE`, zona no UTC y límites de sesión ausentes, nulos, cero o superiores a los máximos;
- fuente ausente, cruzada, caducada o revocada;
- clave, configuración, raíz, audiencia, manifiesto, COSE y efecto cruzados;
- llamada directa denegada y llamada anidada por el rol exacto aceptada;
- roles cruzados, adicionales, despachador, `PUBLIC`, DML y `SET ROLE`;
- consumo nuevo, replay exacto y colisiones por capacidad, evento estable,
  nonce u operación;
- rechazo de sobre multiefecto y aceptación de eventos atómicos separados;
- rollback después del consumo y reconciliación de `COMMIT` incierto;
- carreras consumo–revocación en ambos órdenes y checkpoint causal;
- carreras de `up` y `down` de `000007`, en ambos órdenes, contra un consumidor
  humano V3, DML en cada revocación con avance de checkpoint y `000006.down`;
  cada carrera termina en éxito serializado o `55P03` sin residuo, nunca en
  espera circular; cualquier `40P01` hace fallar esta demostración;
- inyección separada de `40P01` acredita su clasificación defensiva y el
  reintento del envoltorio completo en sesión y transacción nuevas;
- agotamiento de los tres intentos, backoff acotado, nueva sesión/transacción
  por intento, ausencia de reintento semántico y línea base byte a byte intacta;
- `up→down→up`, retirada bloqueada por historia y restauración del `CHECK`;
- `000006.down` con F0 falla y conserva todo; centinela habilitado, alterado o
  ausente falla cerrado; `000007.down` vacío y después `000006.down` funcionan;
- catálogo acredita una única dependencia normal propia del centinela; tras
  retirar los tres triggers históricos, `DROP FUNCTION ... RESTRICT` falla y
  el rollback restaura los cuatro triggers;
- ensayo aislado de cada componente dormido dentro de su envoltorio
  transaccional efímero y ejecución de `up`/`down` exclusivamente mediante el
  runner; para cada componente se prueba ausencia, mutación adversa y orden
  incorrecto, verificando rollback total y ausencia de objetos, filas,
  privilegios o locks persistentes del intento fallido;
- control transaccional y metacomandos adversos son rechazados; incluso ante
  evasión simulada, `max_prepared_transactions=0` y `pg_prepared_xacts` vacío
  impiden conservar una transacción preparada;
- regresión íntegra de las tres audiencias y consumidores V3 existentes.

Se exigen tres ejecuciones limpias, ShellCheck, `git diff --check`, límites de
líneas, Gitleaks y revisión independiente P0/P1/P2 antes de confirmar la
implementación.

## Bloqueos productivos

El código puede construirse con una fuente sintética gobernada tras obtener
GO de esta decisión. Producción permanece NO-GO hasta cerrar:

- fuente maestra, versiones, semántica, retención y actos aprobados;
- intermediario, emisor, COSE, HSM/KMS y rotación/revocación operativas;
- raíces, configuración, claves y coordinación global de secuencias;
- identidades técnicas, R0, TLS/mTLS y segregación de red aprobados;
- EIPD, categorización ENS, análisis de riesgos y procedimiento de incidentes;
- conformidad formal de RRHH, Sistemas, DBA, Seguridad y DPD.

Ninguna ausencia se sustituye por configuración libre, memoria, cabeceras,
cookies, datos DEMO o una decisión PDP.
