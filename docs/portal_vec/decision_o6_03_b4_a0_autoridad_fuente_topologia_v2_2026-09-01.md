# CT-LITE-O6-03-B4-A0 — autoridad, fuente y topología V2

Fecha: 1 de septiembre de 2026.

## Estado, capability, invariante y write-set

Este candidato parte de la base exacta y limpia
`5cca11d09d463ed22b0debc48876ae1497aa2948`, que integra
`CT-LITE-O6-03-B4-A-DOC-R1`. Es una decisión exclusivamente documental: no
implementa código ni SQL, no cierra `O6-03`, no cambia métricas y no autoriza
despliegue, producción ni datos reales.

Capability futura: `AltaDurableLlamamientoAbiertoV2`, propiedad exclusiva de
la aplicación de Bolsa.

Invariante:

> Una única transacción `SERIALIZABLE READ WRITE`, ejecutada por el único
> LOGIN nominal fijado aquí, relee una fuente V2 previa y durable de Bolsa,
> consume exactamente una autorización VEC V2 y confirma hecho de creación,
> agregado abierto, historia inicial, auditoría, outbox y recibo; todo
> `COMMIT` o todo `ROLLBACK`.

Write-set único de este corte:

```text
docs/portal_vec/decision_o6_03_b4_a0_autoridad_fuente_topologia_v2_2026-09-01.md
```

No se crea una segunda línea de implementación. Este documento continúa el
DAG integrado B4-R1 -> B4-A-R1 y conserva A1 bloqueada hasta obtener dos `GO`
independientes sobre su hash exacto e integración por Dirección.

## Preflight local acreditado antes de editar

- rama canónica exclusiva `trabajo/ct-o6-b4-a0-doc-20260901`;
- `HEAD` exacto `5cca11d09d463ed22b0debc48876ae1497aa2948` y árbol limpio;
- producto y referencia local de
  `origin/integracion/ct-producto-ligero-20260821` exactos en
  `67dee4d89c84c2417c3b323b375c05cf65429bfc`;
- el destino no existía, no tenía historia en ninguna referencia y no
  aparecía en otro worktree;
- Go local `go1.25.11 linux/amd64`; no se necesitó ejecutar código Go; y
- cero `fetch`, `pull`, rebase, reset, copia manual entre ramas, PostgreSQL,
  Docker, E2E, despliegue, producción o credenciales.

## Autoridades consumidas sin reinterpretación

Esta decisión consume:

- B2, `LlamamientoAbierto` y `NuevoLlamamientoAbierto`, como única autoridad
  de construcción del valor abierto;
- PRE-CAP y B3 exclusivamente para las terminales futuras de B4-B; no
  autorizan el alta;
- B4-R1, que obliga a una unidad Bolsa V2 aislada de V1 y colocalizada con la
  fachada VEC V2 en la misma base y transacción;
- B4-A-R1, que exige cerrar aquí autorización de alta, fuente bloqueable,
  identidad runtime y recuperación conjunta;
- O6-01 solo como contrato externo: su recibo no es fuente interna de Bolsa;
  y
- la fachada existente
  `vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada`;
  la reconciliación vigente por decisión se conserva inmutable, pero no se usa
  para demostrar ausencia B4-A porque exige localizadores que pueden faltar.

V1 queda histórica e inmutable. Ningún objeto, dato, función, recibo o
adaptador V1 participa en B4-A.

## Decisión de autorización exacta del alta

Toda alta nueva exige una decisión positiva V2 de solicitud ligada. No existe
alta exenta, autorización condicional ni fallback. La política nominal fija:

| dimensión | valor exacto |
| --- | --- |
| esquema de decisión | `vec.autorizacion.decision.reforzada.v2.solicitud-ligada` |
| esquema de solicitud | `vec.autorizacion.solicitud.v2.efectiva-minimizada` |
| acción | `bolsa.llamamiento.abrir` |
| finalidad | `gestion_alta_llamamiento_abierto` |
| perfil | `PerfilProteccionUsoAutorizacionInternoAlto` |
| garantía efectiva y mínima | alta |
| superficies admitidas | interna corporativa o administración privilegiada |
| módulo | `bolsa` |
| tipo de recurso | `llamamiento_abierto` |
| referencia de recurso | `LlamamientoRef` exacta |
| campos permitidos | colección exactamente vacía |
| obligaciones | colección exactamente vacía |

Una colección de campos vacía significa efecto atómico sin selección por
campo, nunca comodín. Se rechazan claves adicionales, valores corregidos,
normalización, perfiles distintos y cualquier `*`.

El recurso exacto conserva:

```text
Referencia = LlamamientoRef
ModuloID   = bolsa
Tipo       = llamamiento_abierto

Ambitos, exactamente 3:
  bolsa_ref     = BolsaRef
  necesidad_ref = NecesidadRef
  propuesta_ref = PropuestaRef

Atributos, exactamente 4:
  fuente_ref       = FuenteRef
  version_inicial  = 1
  estado_inicial   = abierto
  operacion_ref    = OperacionRef
```

La huella de contexto es exclusivamente
`RecursoAutorizable.HuellaContextoAutorizacionSHA256()` sobre ese recurso.
Aplicación debe cotejarla con solicitud, decisión, evidencia y capacidad; SQL
no vuelve a construirla.

El `sujeto_ref` no identifica a una persona ni se improvisa al emitir la
capacidad. Es el `SujetoRef` opaco persistido en la fuente previa por D2 y tiene
exactamente esta forma:

```text
hmac-sha256:bolsa.llamamiento-abierto.v2:<64 hex minúsculos>
```

D2 es su único productor. Lo obtiene mediante un puerto de pseudonimización
gobernado que aplica HMAC-SHA256 con clave secreta no exportable a la preimagen
encuadrada de `vec.bolsa.llamamiento-abierto.sujeto.v2`, `FuenteRef` y
`LlamamientoRef`, en ese orden. Ni A1, SQL ni VEC generan o corrigen el valor.
Solicitud, decisión, evidencia y capacidad deben ligarlo por igualdad exacta a
la fuente bloqueada; una divergencia deniega el efecto. El contrato D2 fija el
identificador y versión positiva de la clave, su rotación y los negativos de
clave ausente, salida mal formada o ámbito distinto, sin persistir la clave.

La decisión queda ligada además, por igualdad exacta, a actor confiable,
cuenta, perfil, sesión, método, garantía, vínculo vigente, correlación V2,
política aplicada, solicitud y su huella, referencia y huella de motivo,
instantes, emisor, audiencia y ausencia de comodines. Ninguno procede de
HTTP, JSON, cabeceras, cookies, parámetros, configuración libre o la fuente
durable.

## Motivo gobernado: dependencia previa cerrada

El alta usará una entrada dedicada y publicada del catálogo
`motivos_autorizacion_llamamiento`. Su clave debe ser opaca conforme al perfil
`motivo_<128 bits hex minúsculos>` y la referencia completa conserva
`CatalogoID`, versión positiva, huella SHA-256 del catálogo y `EntradaClave`.

No se fija una clave legible ni una huella ficticia en código o en este
documento. La autoridad catalogal debe publicar la entrada específica
«alta durable de llamamiento abierto V2» y entregar su referencia exacta. La
misma versión y huella se revalidan dentro de la transacción. Retirada,
caducidad, ausencia o divergencia deniegan un efecto nuevo.

La tarea previa se denomina
`CT-LITE-O6-03-B4-A0-D1-MOTIVO-ALTA-V2`. Debe quedar integrada antes de
emitir una solicitud real o ejecutar la prueba PostgreSQL de A5. No autoriza
por sí sola A1, SQL ni producción.

## Capacidad HMAC de registro y consumo atestado

El emisor es únicamente la autoridad VEC existente; Bolsa no genera claves,
MAC, nonces, material COSE, raíz o evidencia. La capacidad conserva el
esquema literal
`vec.autorizacion.capacidad-registro-consumo-atestado.v2`, la audiencia
literal `vec_autorizacion_atestada_v2.registrar_y_consumir` y una vida máxima
de cinco segundos, siempre acotada además por decisión, sesión, vínculo,
confianza, configuración y raíz.

Los 39 campos de la capacidad existente permanecen cerrados:

```text
esquema, clave_id, clave_version, emisor_id, audiencia, nonce,
emitida_en, expira_en, registro_ref, consumo_ref, decision_ref,
huella_decision_sha256, huella_motivo_sha256,
huella_payload_vec_ad_2_sha256, huella_sobre_cose_sign1_sha256,
huella_evidencia_verificacion_sha256, principal_id, accion, finalidad,
sujeto_ref, recurso_ref, contexto_recurso_huella_sha256,
correlacion_ref, decision_valida_hasta, efecto_ref,
huella_efecto_sha256, verificada_en, revision_confianza,
huella_configuracion_sha256, configuracion_publicada_en,
configuracion_expira_en, raiz_clave_id, raiz_version,
huella_raiz_spki_sha256, raiz_valida_desde, raiz_valida_hasta, suite,
audiencia_despliegue, mac_sha256
```

Para B4-A, `accion`, `finalidad`, `principal_id`, `sujeto_ref`, `recurso_ref`,
huella de contexto, correlación, decisión, motivo y sus huellas deben ser
idénticos a la ligadura anterior y a la fuente bloqueada.

Todas las preimágenes de este corte usan una sola función de encuadre:
representación UTF-8 sin normalización precedida por su longitud en bytes como
entero binario sin signo de 64 bits en orden big-endian. La concatenación no
usa separadores, JSON ni representación dependiente del lenguaje. Cada valor
se valida antes del encuadre.

`HuellaAltaSHA256` es el hexadecimal minúsculo de SHA-256 sobre la concatenación
encuadrada, en este orden exacto:

```text
vec.bolsa.llamamiento-abierto.alta.v2
OperacionRef
FuenteRef
LlamamientoRef
BolsaRef
NecesidadRef
PropuestaRef
1
abierto
SujetoRef
HuellaFuenteAltaSHA256
```

Los literales `1` y `abierto` son esos bytes UTF-8. A1 no elige ni amplía esta
preimagen; la reproduce y contrasta con un vector fijado por A0. La capacidad
usa:

```text
efecto_ref = efecto:bolsa:llamamiento-abierto:alta-v2:<HuellaAltaSHA256>
huella_efecto_sha256 = HuellaAltaSHA256
```

El marcador entre ángulos se sustituye por los 64 caracteres hexadecimales
reales; no pertenece a la cadena.

La función nominal de efecto es exactamente:

```text
vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(
  bytea, bytea, bytea, bytea, bytea, bytea, jsonb
)
```

La función VEC vigente de cinco argumentos no sirve para B4-A cuando faltan
todas las piezas locales: exige decisión y nonce que no se pueden derivar de
`OperacionRef` y `HuellaAltaSHA256`. B4-A no la usa para demostrar ausencia.

La tarea previa VEC/DBA
`CT-LITE-O6-03-B4-A0-D4-RECONCILIACION-EFECTO-V2` añadirá, sin modificar la
función vigente, la fachada nominal exacta:

```text
vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(
  text, text
)
```

Sus argumentos son `efecto_ref` y `huella_efecto_sha256`. Para entradas válidas
devuelve exactamente una fila con estado cerrado `ausente`, `exacto` o
`colision`; solo `exacto` entrega las cinco columnas opacas vigentes de recibo.
`exacto` exige que exista precisamente el par. `colision` se devuelve si existe
el efecto con otra huella o si existe la huella única ligada a otro efecto.
`ausente` solo es posible tras demostrar que no existe ninguna de las dos
claves únicas. Entrada inválida, cardinalidad imposible, fallo interno o
indisponibilidad elevan error constante y se interpretan como indeterminado. La
fachada comprueba la misma
identidad runtime, es `SECURITY DEFINER`, fija `search_path=pg_catalog`, revoca
`PUBLIC`, no expone decisión, nonce o tablas y recibe pruebas y doble revisión
antes de D3 y A1.

No se leen ni escriben tablas VEC y no se crea un ledger alternativo.

## Fuente durable canónica y productor único

La fuente autoritativa previa será exclusivamente la relación append-only:

```text
vec_bolsa_llamamientos_v2.fuente_alta_llamamiento_abierto_v2
```

Cada fila contiene exactamente:

```text
FuenteRef
LlamamientoRef
BolsaRef
NecesidadRef
PropuestaRef
HuellaPropuestaAltaSHA256
VersionInicial = 1
OperacionOrigenRef
SujetoRef
HuellaFuenteAltaSHA256
ConfirmadaEn
```

`HuellaFuenteAltaSHA256` es el hexadecimal minúsculo de SHA-256 sobre el mismo
encuadre binario ya definido y, en este orden exacto, el esquema literal
`vec.bolsa.llamamiento-abierto.fuente-alta.v2`, FuenteRef, LlamamientoRef,
BolsaRef, NecesidadRef, PropuestaRef, HuellaPropuestaAltaSHA256, el literal
`1`, OperacionOrigenRef, SujetoRef y ConfirmadaEn. El instante se representa
en UTC con el patrón ASCII exacto `YYYY-MM-DDTHH:MM:SS.ffffffZ`, siempre seis
dígitos de microsegundo. La preimagen excluye candidato, persona, selección,
posición, contacto, documento, texto libre y cualquier huella derivada de PII.

### Propuesta V2 previa: identidad cerrada aquí

La propuesta previa no queda delegada a D2 ni a una futura decisión. Su única
capacidad productora se denomina exactamente
`ConfirmarPropuestaAltaLlamamientoAbiertoV2`, pertenece a la aplicación de
Bolsa y tiene esta forma cerrada:

```text
ConfirmarPropuestaAltaLlamamientoAbiertoV2
  Confirmar(context.Context, SolicitudConfirmarPropuestaAltaLlamamientoAbiertoV2)
    -> ReciboPropuestaAltaLlamamientoAbiertoV2, error
```

Su solicitud es opaca, no serializable y no reconstruible desde escalares,
HTTP, memoria, recibos o adaptadores. Solo nace tras obtener una
`domain.PropuestaLlamamiento` canónica mediante la autoridad de dominio
`domain.ProponerPrimerLlamamiento`; el documento completo de dominio, sus
evaluaciones, participación y sujeto no se persisten ni se convierten en
huellas. Esta reutilización de dominio no invoca
`ServicioLlamamientos.ProponerPrimerLlamamiento`, `guardar_propuesta_v1`,
objetos o datos V1.

La capacidad confirma su único efecto durable dentro de una transacción
`SERIALIZABLE READ WRITE` en la relación append-only:

```text
vec_bolsa_llamamientos_v2.propuesta_alta_llamamiento_abierto_v2
```

No admite `UPDATE` ni `DELETE`. Ninguna otra capacidad, función o rol puede
insertarla.

Cada fila de propuesta contiene exactamente:

```text
PropuestaRef
LlamamientoRef
BolsaRef
NecesidadRef
VersionInicial = 1
OperacionPropuestaRef
EstadoPropuesta = confirmada
HuellaPropuestaAltaSHA256
ConfirmadaEn
```

`PropuestaRef` es la clave primaria. `OperacionPropuestaRef`,
`LlamamientoRef` y `HuellaPropuestaAltaSHA256` son claves únicas. La
representación no contiene candidato, persona, sujeto, selección, posición,
contacto, documento, texto libre ni huella de esos datos.
`HuellaPropuestaAltaSHA256` solo compromete, con el encuadre binario común ya
fijado y en este orden exacto:

```text
vec.bolsa.llamamiento-abierto.propuesta-alta.v2
PropuestaRef
LlamamientoRef
BolsaRef
NecesidadRef
1
OperacionPropuestaRef
confirmada
ConfirmadaEn
```

La única función bloqueable de captura es:

```text
vec_bolsa_llamamientos_v2.capturar_propuesta_alta_llamamiento_abierto_v2(text)
```

El argumento es `PropuestaRef`. La función es `SECURITY DEFINER`, fija
`search_path=pg_catalog`, devuelve exclusivamente las nueve columnas
anteriores y ejecuta `FOR UPDATE` sobre la propuesta exacta dentro de la
transacción llamante. Cero filas, más de una, entrada inválida, incoherencia,
fallo o indisponibilidad son denegación opaca. El LOGIN runtime no recibe
`EXECUTE`: solo la invocan internamente las fachadas `SECURITY DEFINER`
propietarias de D2-F y B4-A.

El único productor de la fuente será después la capacidad de Bolsa V2
`RegistrarFuenteAltaLlamamientoAbiertoV2`. Recibe una capacidad opaca sellada
que liga PropuestaRef, FuenteRef, LlamamientoRef y OperacionOrigenRef; no recibe
esos valores como escalares corregibles. Bloquea primero la propuesta mediante
la función anterior, reconcilia la fuente y solo entonces la inserta.

La fuente conserva una FK compuesta exacta hacia
`propuesta_alta_llamamiento_abierto_v2` que obliga a la igualdad byte a byte
de PropuestaRef, LlamamientoRef, BolsaRef, NecesidadRef, VersionInicial,
`OperacionOrigenRef = OperacionPropuestaRef` y
HuellaPropuestaAltaSHA256. `fuente.PropuestaRef` y
`fuente.LlamamientoRef` son además únicas, y
`fuente.ConfirmadaEn >= propuesta.ConfirmadaEn`. Cero, más de una o cualquier
divergencia deniegan el efecto.

Esta capacidad no deriva autoridad desde O6-01, Contratación temporal, HTTP,
memoria, V1 o un recibo externo. Tendrá contrato, persistencia, autorización,
pruebas y revisión propios antes de B4-A.

La tarea previa se denomina
`CT-LITE-O6-03-B4-A0-D2-FUENTE-ALTA-V2` y conserva una sola línea canónica
dividida causalmente: D2-P implementa y prueba contrato, persistencia y captura
de propuesta; tras su doble revisión e integración, D2-F implementa y prueba
`RegistrarFuenteAltaLlamamientoAbiertoV2`. Si D2-P no puede existir sin usar
V1 u otra autoridad, el resultado es `NO-GO`.

Una retirada futura se registra por adición, nunca modifica o borra la fuente.
Una fuente retirada no permite un alta nueva. La recuperación de un recibo ya
confirmado sí puede cotejar la fuente histórica retirada.

`hecho_creacion_llamamiento_abierto_v2` no duplica la fuente: es el consumo
único de esa fuente por B4-A. Debe tener una FK exacta a `FuenteRef` y una
unicidad que impida que una fuente produzca dos hechos o llamamientos. Las
cuatro referencias, `SujetoRef`, `VersionInicial` y
`HuellaFuenteAltaSHA256` deben coincidir byte a byte entre fuente, hecho,
agregado, historia y recibo. `HuellaPropuestaAltaSHA256` coincide además
entre propuesta, fuente, hecho, auditoría y recibo.

## Topología runtime única compatible con VEC

Se corrige el marcador NOLOGIN de B4-A-R1: el rol con nombre reservado
`vec_bolsa_llamamientos_v2_ejecutor` es el único LOGIN runtime. No se crea un
LOGIN adicional ni un segundo rol ejecutor.

Propiedades exactas:

| rol | propiedades |
| --- | --- |
| `vec_bolsa_llamamientos_v2_propietario` | `NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS` |
| `vec_bolsa_llamamientos_v2_migrador` | `NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS` |
| `vec_bolsa_llamamientos_v2_ejecutor` | `LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS` |

El LOGIN tiene exactamente una membresía directa:

```text
vec_bolsa_llamamientos_v2_ejecutor
  -> vec_autorizacion_atestada_v2_consumidor
     ADMIN FALSE, INHERIT TRUE, SET TRUE
```

No tiene otras membresías directas o transitivas. El rol consumidor VEC no es
LOGIN y no es miembro de otro rol. No se usa `SET ROLE`. Así
`session_user` permanece exactamente
`vec_bolsa_llamamientos_v2_ejecutor` y satisface la comprobación vigente de
una única membresía consumidora.

El LOGIN recibe de forma directa `USAGE` sobre
`vec_bolsa_llamamientos_v2` y `EXECUTE` únicamente sobre las fachadas Bolsa
nominales. No recibe DML, secuencias, tipos auxiliares, funciones internas,
propiedad ni membresía del propietario.

La membresía VEC es solo un marcador de identidad. D3 debe enmendar las ACL VEC
vigentes y revocar `EXECUTE` sobre registro/consumo y ambas reconciliaciones de
`PUBLIC` y de `vec_autorizacion_atestada_v2_consumidor`; por tanto el LOGIN no
hereda ninguna fachada VEC invocable. La misma migración inventariará
previamente todos los consumidores legítimos existentes, los migrará y
concederá cada función exacta únicamente al propietario `NOLOGIN` de su fachada
modular `SECURITY DEFINER`. El cambio y las concesiones sustitutas son atómicos.
Si aparece un consumidor directo que
no pueda migrarse sin interrupción o ampliar autoridad, D3 es `NO-GO`.

El propietario Bolsa recibe `USAGE` sobre
`vec_autorizacion_atestada_v2` y `EXECUTE` únicamente sobre la función de
registro y consumo y la nueva reconciliación por efecto. Esto permite que la
fachada Bolsa `SECURITY
DEFINER` componga VEC en la misma conexión mientras `session_user` conserva la
identidad runtime. No se concede DML ni acceso a tablas VEC.

Así, una llamada directa con `current_user` igual al LOGIN falla por ACL antes
de entrar en VEC; la llamada desde la fachada Bolsa ejecuta con el propietario
modular, pero conserva el `session_user` exacto que verifica VEC. Ningún
`SET ROLE`, concesión a rol heredado o wrapper alternativo puede recrear el
atajo. La migración inversa queda protegida y no puede restaurar `EXECUTE` al
rol consumidor mientras exista un LOGIN miembro.

`PUBLIC`, el consumidor VEC genérico y cualquier otro LOGIN quedan sin
`USAGE`, DML o `EXECUTE` sobre objetos de Bolsa V2. La membresía VEC del LOGIN
no transfiere privilegios Bolsa al rol consumidor. Las fachadas Bolsa deben
comprobar propiedades, membresía única, `session_user`, aislamiento,
transacción, TLS y límites antes de interpretar material.

La tarea previa
`CT-LITE-O6-03-B4-A0-D3-ROLES-ACL-V2` implementará roles y concesiones y
probará catálogo de membresías, invocación directa VEC denegada, propietarios
modulares positivos, ACL negativas, migración inversa protegida y ausencia de
autoridad V1. No se integra si exige otra membresía, `SET ROLE`, otro LOGIN o
deja una función VEC invocable por el rol consumidor.
D3 depende de D4 integrado para conceder la fachada exacta ya existente.

## Orden total de locks, relecturas y efecto

Para una operación nueva, el adaptador y las fachadas ejecutan este orden:

1. abrir una transacción local `SERIALIZABLE READ WRITE`, endurecer
   `search_path`, zona UTC, timeouts y solo después aceptar la solicitud;
2. adquirir advisory lock transaccional derivado de `OperacionRef`;
3. releer por operación el paquete local y clasificar replay o colisión antes
   de bloquear otra fuente;
4. adquirir advisory lock transaccional derivado de `PropuestaRef`;
5. capturar con `FOR UPDATE` la propuesta exacta y exigir estado confirmado,
   versión 1 y huella válida;
6. adquirir advisory lock transaccional derivado de `FuenteRef`;
7. bloquear con `FOR UPDATE` la fuente exacta y su posible retirada;
8. exigir el vínculo compuesto exacto entre propuesta y fuente, fuente
   existente, no retirada para efecto nuevo, productor V2
   acreditado, huella válida y coincidencia exacta de las cuatro referencias,
   `SujetoRef`, versiones y OperacionOrigenRef;
9. reconstruir B2 en Go mediante `NuevoLlamamientoAbierto`; SQL no decide el
   agregado;
10. releer y cotejar solicitud, decisión, motivo, vínculo, confianza, tiempos,
   recurso, capacidad y plan; obtener el instante transaccional después de los
   bloqueos;
11. invocar una sola vez la fachada VEC de registro y consumo y cotejar su
   resultado nominal;
12. insertar hecho, agregado abierto, historia inicial, auditoría, outbox y
    recibo, con unicidades y FK internas exactas;
13. releer defensivamente las seis piezas y el consumo VEC, comparar todos los
    compromisos y validar el recibo minimizado; y
14. ejecutar el único `COMMIT`.

Un `40001` o `40P01` anterior al intento de commit permite reintento acotado
en una transacción nueva, con la misma solicitud sellada y todas las
relecturas. Error o pérdida tras intentar commit produce resultado
indeterminado y solo abre `Recuperar`. La cancelación observada antes de
commit revierte todo; después del intento no deshace un resultado durable.

## Recuperación conjunta obligatoria

`Recuperar` usa una transacción de solo lectura serializable, la identidad
runtime exacta y los localizadores opacos `OperacionRef` y
`HuellaAltaSHA256`. Deriva de ellos el `efecto_ref` exacto y consulta la fachada
D4 con ese efecto y huella; no necesita decisión ni nonce local. No reconsume,
completa ni repara.

| estado observado | resultado |
| --- | --- |
| seis piezas locales exactas + consumo VEC exacto | recibo histórico exacto |
| cero piezas locales + estado VEC `ausente` tras comprobar que no existen ni efecto ni huella | ausencia demostrada |
| alguna pieza local sin las otras | corrupción/indeterminado |
| estado VEC `exacto` sin las seis piezas locales | corrupción/indeterminado |
| estado VEC `colision` | colisión/corrupción, nunca ausencia |
| seis piezas locales sin consumo VEC exacto | corrupción/indeterminado |
| cualquier huella, referencia, instante o vínculo divergente | colisión o corrupción, nunca recibo |
| VEC no disponible o no reconciliable | indeterminado, nunca ausencia |

La retirada posterior de la fuente o la caducidad posterior de la decisión no
invalidan un recibo histórico confirmado. Solo impiden un efecto nuevo. La
recuperación coteja VEC mediante la fachada D4, pero nunca vuelve a registrar o
consumir.

## Privacidad, seguridad y límites

- Fuente, hecho, agregado, historia, auditoría, outbox y recibo contienen solo
  referencias opacas, estado, versiones, instantes y huellas técnicas sin PII.
- Errores, `fmt`, `slog` y trazas son constantes redactadas; no muestran SQL,
  DSN, material VEC, referencias, huellas, motivo o causa privada.
- Cero cookies, almacenamiento web, identidad de cliente, V1, lectura de
  tablas ajenas, doble escritura, saga, compensación o consistencia eventual.
- Una llamada opera sobre una fuente, un agregado y una operación; límites de
  bytes, gramática y tiempo se aplican antes de reservar o bloquear.
- RLS forzada, políticas del propietario exacto, funciones `SECURITY DEFINER`
  con `search_path=pg_catalog`, revocación de `PUBLIC` y `down` protegido.
- Cifrado, claves, backup, restauración, retención y observabilidad continúan
  bajo sus autoridades. No se usan datos reales ni se declara conformidad.

Este corte no tiene interfaz. Un consumidor posterior usará i18n y estados
accesibles por texto además de color; no procede revisión visual ahora.

## Paradas duras

El DAG se detiene en `NO-GO` si:

1. se intenta usar otra fuente, rama o implementación equivalente;
2. el motivo dedicado no está publicado y vigente;
3. el productor de fuente no puede partir de una propuesta Bolsa V2
   autoritativa sin V1, HTTP, CT, memoria o recibos externos;
4. el LOGIN tiene cero, dos o más membresías, usa `SET ROLE`, recibe DML o
   puede ejecutar directamente una fachada VEC;
5. propietario Bolsa y fachada VEC no pueden operar en la misma base,
   conexión y transacción local;
6. una alta nueva puede omitir el consumo VEC o usar otro esquema, acción,
   finalidad, perfil, sujeto, recurso, encuadre, efecto o huella;
7. una fuente puede consumirse dos veces o el hecho duplica su autoridad;
8. el orden de locks cambia, el reloj retrocede o se captura antes de los
   bloqueos;
9. replay, colisión, cancelación o commit ambiguo no fallan cerrados;
10. recuperación usa la función vigente de cinco argumentos, interpreta
    indisponibilidad como ausencia o intenta reparar; o
11. se pretende abrir HTTP, composición raíz, E2E, despliegue o producción.

## DAG, pruebas y siguiente corte

```text
CT-LITE-O6-03-B4-A0 (este candidato)
  -> doble revisión independiente del hash exacto: GO + GO
    -> integración por Dirección
      -> D2-P propuesta confirmada V2
        -> D2-F contrato y productor de fuente V2
      -> D1 motivo dedicado publicado
      -> D4 reconciliación VEC por efecto único
        -> D3 roles, LOGIN y ACL exactas
      -> D1 + D2-P + D2-F + D3 + D4 integrados
        -> CT-LITE-O6-03-B4-A1-CONTRATO-GO
          -> A2/A3 persistencia y fachadas nominales
            -> A4 adaptador postgresllamamientosv2
              -> A5 PostgreSQL real y revisión independiente
```

D1, D2-P y D4 pueden analizarse en paralelo solo si sus write-sets son
disjuntos; se integran uno a uno. D2-F depende de D2-P y D3 depende de D4. A1
no empieza hasta que D1, D2-P, D2-F, D3 y D4 estén integrados. Ningún corte
copia código entre ramas ni crea otra ruta B4-A.

Puertas ligeras de este candidato: autoridad completa, inventario de ramas y
worktrees, destino sin historia, modo `0644`, menos de 800 líneas, UTF-8,
`git diff --check`, Gitleaks focal, patch-id único y merge-tree limpio contra
producto. No proceden Go, carrera, `go vet`, PostgreSQL, Docker, E2E ni puertas
globales porque solo se añade Markdown.

Los cortes posteriores probarán en PostgreSQL efímero real fuente y retirada,
replay/colisión por efecto y por huella, membresía exacta, invocación VEC
directa denegada, propietarios modulares positivos, ACL/RLS, cada referencia alterada,
autorización válida/caducada/revocada, motivo/vínculo retirados durante locks,
consumo único, fallo en cada una de las seis escrituras, cancelación,
respuesta perdida, recuperación tras reinicio, `down` protegido y ausencia de
PII en tablas, WAL de prueba, recibos, outbox, auditoría, errores y logs.

El siguiente corte seguro es la doble revisión independiente de este hash
exacto. Tras integrarlo, el primer corte de implementación permitido es D1,
D2-P o D4 dentro de sus autoridades; D2-F espera a D2-P, D3 espera a D4 y A1
continúa bloqueada hasta integrar los cinco.
