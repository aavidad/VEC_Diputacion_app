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

La única función exterior F0 será:

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
```

M6/M7 derivan únicamente audiencia, acción, tipo, operación, `efecto_ref` y
huella de efecto esperados desde constantes y filas bloqueadas. F0 extrae
fuente, versión, evento y huella de evento desde capacidad y manifiesto,
calcula las huellas de capacidad y manifiesto, cruza todo con el catálogo
duradero y lo devuelve. Ningún login, DTO ni inbox previo aporta o escoge
autoridad.

Devuelve únicamente `capacidad_ref`, fuente y versión, evento, huellas del
evento y manifiesto, operación, efecto y huella, `consumo_huella_sha256`,
`consumida_en` y `consumo_nuevo`. `capacidad_ref` se deriva como `cfc_` más el
SHA-256 de la capacidad; nunca llega del cliente.

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
`search_path=pg_catalog` y `lock_timeout=2s`. Solo
`vec_contexto_actor_v1_propietario` recibe `USAGE` de esquema y `EXECUTE` de
esa firma para anidarla exclusivamente en las fachadas finales M6/M7. No
recibe acceso a tablas, secuencias, tipos internos ni otras funciones V3; el
inventario de dependencias rechaza cualquier consumidor adicional.

R0 fijará estos grupos `NOLOGIN`:

```text
vec_contexto_actor_v1_publicador_corporativo
vec_contexto_actor_v1_revocador_corporativo
vec_contexto_actor_v1_despachador_corporativo
```

La función reacredita `session_user` aunque la invoque anidada la fachada
propietaria de ContextoActor. Publicación exige solo membresía publicadora;
revocación, solo revocadora. Membresía cruzada, adicional o despachadora
deniega. La ausencia de R0 también deniega. Los LOGIN no obtienen `EXECUTE`
directo y se provisionan fuera de Git.

Se revoca de `PUBLIC`, migrador, emisor, consumidor y runtime V3, runtime de
ContextoActor, los tres grupos R0 y cualquier login técnico. La función exige
propietario y configuración exactos en cada llamada; `RLS` es solo defensa
adicional.

## Transacción, locks y replay

F0 solo funciona dentro de la misma transacción exterior
`SERIALIZABLE READ WRITE` de C2.3, en UTC, con `statement_timeout` entre 1 y
10 segundos, `transaction_timeout` e
`idle_in_transaction_session_timeout` entre 1 y 15 segundos. Nunca abre ni
confirma una transacción.

La fachada C2.3 toma antes sus barreras A→B→C→D, advisory locks, E y filas en
el orden ya decidido. F0 continúa así:

1. bloquea `checkpoint_gobierno FOR UPDATE`;
2. toma advisory locks de capacidad y operación en orden canónico;
3. busca replay o colisión;
4. bloquea `FOR SHARE` catálogo de fuente, clave/puntero, configuración y su
   puntero, raíz y asociación configuración–raíz;
5. lee el reloj, cruza todos los campos y revalida vigencias y revocaciones;
6. verifica HMAC y ligaduras de manifiesto, COSE, evidencia y SPKI;
7. vuelve a leer el reloj, revalida el estado bajo los mismos locks, inserta
   atestación y consumo y devuelve la prueba.

Las altas y revocaciones de fuente, clave, configuración o raíz bloquean
primero el mismo checkpoint. Por ello no pueden ganar entre la revalidación y
el `COMMIT`.

El primer consumo devuelve `consumo_nuevo=true`. Un replay byte a byte exacto
devuelve la misma prueba con `false`; reutilizar operación, capacidad, evento
o nonce con otra preimagen colisiona cerrada. Las cuatro fachadas de efecto
exigen `true` al crear un efecto nuevo.

Si falla cualquier escritura posterior de C2.3, PostgreSQL revierte también
atestación y consumo. Tras `COMMIT` incierto se reconcilia primero la prueba
local de C2.3 bajo sus mismos locks: si es exacta, el efecto confirmó y no se
llama a F0; si no existe, el consumo tampoco puede existir y se repite con los
mismos bytes. No se usa cola, compensación ni recibo tardío.

## Migración y retirada

Los únicos artefactos previstos son:

```text
deploy/postgresql/autorizacion_atestada_v3/migraciones/
  000007_fuente_corporativa_contexto_actor_v1.up.sql
  000007_fuente_corporativa_contexto_actor_v1.down.sql
deploy/postgresql/autorizacion_atestada_v3/
  README.md
  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  fuente_corporativa_contexto_actor_v1.sql
```

El README pertenece al write-set de implementación F0-C; D1 no lo modifica.

Antes de escribir se vuelve a acreditar que `000007` está libre y que
`000001..000006` tienen su forma exacta. `up` toma la barrera común nueva
`vec_autorizacion_atestada_v3:migraciones:v1`, después la barrera propia
`...:migracion:000007`, y bloquea directamente en modo final las relaciones
que altera; no asciende un lock. Migraciones futuras adoptarán la barrera
común. Las anteriores son precondiciones ya instaladas, no se ejecutan en
paralelo con `000007`.

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
Elimina primero el centinela con `RESTRICT`, después F0 en orden inverso y
restaura el `CHECK` exacto de las tres audiencias anteriores. Nunca usa
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
- `up→down→up`, retirada bloqueada por historia y restauración del `CHECK`;
- `000006.down` con F0 falla y conserva todo; centinela habilitado, alterado o
  ausente falla cerrado; `000007.down` vacío y después `000006.down` funcionan;
- catálogo acredita un único `pg_depend` normal; borrar un trigger normal y
  después la función con `RESTRICT` falla y el rollback restaura el trigger;
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
