# CT-LITE-O6-03-B4-A-DOC-R1 — alta durable de llamamiento abierto V2

Fecha: 1 de septiembre de 2026.

## Estado, autoridad y alcance

Esta revisión parte del candidato exacto
`eb62050e493e829669c9ff5081628cc739f47809`, cuyo padre único es el producto
`c8cc4312b063ddca7294dfe27dc673ad1a4676d0`, que integra la autoridad
`CT-LITE-O6-03-B4-R1`. Es un corte exclusivamente documental: no implementa
producto, no cierra `O6-03`, no cambia métricas y no autoriza código, SQL,
integración, despliegue ni producción.

La capability futura es `AltaDurableLlamamientoAbiertoV2`, propiedad exclusiva
de la aplicación de Bolsa. Su responsabilidad única seguirá siendo crear de
forma durable un `LlamamientoAbierto` en estado técnico `abierto`, pero no
podrá implementarse hasta que A0 cierre la autoridad V2 exacta del alta, su
fuente durable bloqueable y la topología de identidad común Bolsa/VEC.

Invariante del corte:

> Una transacción `SERIALIZABLE READ WRITE`, conforme al orden y las fuentes
> que fije A0, consume la autorización VEC V2 y materializa hecho local de
> creación + agregado abierto + historia inicial + auditoría + outbox + recibo:
> todo `COMMIT` o todo `ROLLBACK`.

Write-set único:

```text
docs/portal_vec/decision_o6_03_b4_a_alta_durable_llamamiento_abierto_v2_2026-09-01.md
```

El siguiente corte, únicamente después de doble revisión independiente con
`GO` del hash exacto e integración de este R1 por Dirección, es
`CT-LITE-O6-03-B4-A0-AUTORIDAD-FUENTE-TOPOLOGIA-DOC`.

## Autoridades leídas y precedencia

Esta decisión consume, sin modificar ni reinterpretar:

- B2: `LlamamientoAbierto`, `DatosLlamamientoAbierto` y
  `NuevoLlamamientoAbierto` en
  `internal/modules/bolsa/domain/llamamiento_abierto.go`;
- PRE-CAP: `OrdenTerminalLlamamientoAutorizadaV2` en aplicación de Bolsa,
  limitada a aceptación, renuncia y expiración y sin autoridad para el alta;
- B3: `TransicionarLlamamientoConOrdenTerminalAutorizadaV2` como única
  transición terminal ejecutable;
- B4-R1 integrado, que decide una unidad V2 propia de Bolsa, aislada de V1 y
  colocada en el mismo dominio transaccional PostgreSQL que la fachada VEC V2;
- O6-01, cuyo `ReciboSolicitudLlamamientoBolsa` es un recibo externo de
  integración y no una fuente de verdad interna de Bolsa; y
- las autoridades VEC V2 de solicitud ligada, evidencia opaca, fachada de uso
  y registro/consumo atestado PostgreSQL.

Ante cualquier contradicción, B4-R1 y estas fronteras prevalecen sobre una
inferencia desde nombres, recibos de transporte o piezas V1. Este documento no
presenta `deploy/postgresql/autorizacion_atestada_v2` como autorizado para
producción: su propia autoridad conserva `NO-GO` hasta cerrar broker, HSM/KMS,
ancla anti-restauración, composición atómica y aprobaciones operativas.

## Dictámenes `NO-GO` incorporados y cierre R1

Este R1 registra los dos dictámenes emitidos sobre el candidato
`eb62050e493e829669c9ff5081628cc739f47809`:

- `NO-GO` local de Dirección, `P0`: no existe una autorización V2 exacta para
  crear el llamamiento abierto. PRE-CAP solo cubre aceptación, renuncia y
  expiración; una invocación condicional no autoriza el alta;
- `NO-GO` local de Dirección, `P1`: no existe una propuesta o hecho durable V2
  identificado y bloqueable cuya fuente pueda releerse. Una capacidad privada
  en memoria no sustituye la fuente durable exigida por B4-R1; y
- `NO-GO` remoto independiente, `P1`: el rol reservado
  `vec_bolsa_llamamientos_v2_ejecutor` no compone con la fachada VEC vigente.
  Esa fachada valida `session_user` y exige una única membresía directa
  `vec_autorizacion_atestada_v2_consumidor`; el LOGIN miembro de ambos roles
  falla y `SET ROLE` no cambia esa identidad. Además, la recuperación propuesta
  no reconciliaba el consumo VEC.

Los tres hallazgos quedan cerrados en el alcance documental de este R1 mediante
una única barrera A0 previa: se retira el paso directo a A1, se declara inválida
cualquier autoridad en memoria o condicional y se exige doble `GO` e integración
de A0 antes de contrato, código o SQL. Este cierre no afirma que ya existan la
autoridad, la fuente, la identidad o la atomicidad: A0 debe decidirlas y recibirá
`NO-GO` si alguna permanece abierta.

## Responsabilidad única de B4-A

B4-A crea el agregado abierto y nada más. Para una operación nueva debe:

1. releer bajo bloqueo la fuente durable V2 exacta que A0 autorice y acreditar
   el vínculo con la propuesta confirmada;
2. derivar en aplicación los datos de apertura y construir el valor mediante
   `NuevoLlamamientoAbierto`;
3. consumir obligatoriamente la autorización VEC V2 exacta de toda alta nueva;
4. confirmar en una sola transacción el hecho local de creación, la proyección
   abierta, su primera historia, auditoría, outbox y recibo; y
5. devolver exclusivamente el recibo minimizado confirmado.

B4-A no ejecuta una transición terminal. No acepta
`OrdenTerminalLlamamientoAutorizadaV2`, no llama a
`TransicionarLlamamientoConOrdenTerminalAutorizadaV2`, no invoca B3 y no crea
aceptación, renuncia ni expiración. Tampoco selecciona candidato ni prepara el
siguiente llamamiento.

B4-B conservará
`TransicionarLlamamientoConOrdenTerminalAutorizadaV2` como la única transición
terminal. B4-B dependerá de un alta B4-A ya confirmada, reconstruirá el abierto
con `NuevoLlamamientoAbierto` y nunca insertará oportunistamente el agregado si
falta.

## Origen exclusivo y límite de la solicitud

`SolicitudAltaDurableLlamamientoAbiertoV2` será un transporte opaco interno de
aplicación, nunca la autoridad del alta. Solo podrá construirse dentro de la
operación que relea y bloquee la fuente durable V2 exacta fijada por A0 y
consuma la autorización VEC V2 en el mismo `COMMIT`. Una capacidad privada en
memoria, aunque conserve procedencia o compromisos, no sustituye esa relectura;
una mera igualdad de referencias tampoco.

Queda prohibido construir, reconstruir, rehidratar o promover la solicitud
desde cualquiera de estas fuentes:

- `ReciboSolicitudLlamamientoBolsa` o cualquier otro recibo O6-01;
- HTTP, JSON, XML, texto, binario, Gob, CBOR, YAML, cabeceras, cookies,
  parámetros o almacenamiento del navegador;
- tablas, vistas, outbox, inbox o bytes de Contratación temporal;
- tablas o funciones V1 de Bolsa;
- una capacidad en memoria, un recibo externo o una confirmación externa;
- una lista, mapa o estructura de referencias escalares aportada por un
  adaptador; o
- una proyección persistida cuyo productor, objeto bloqueable y vínculo con la
  propuesta confirmada no hayan sido fijados por A0.

El recibo O6-01 permite a Contratación temporal verificar su intercambio. No
concede a Bolsa autoridad de creación, no puede convertirse en el hecho local
y no autoriza a leer tablas de Contratación temporal.

## Corte obligatorio CT-LITE-O6-03-B4-A0-AUTORIDAD-FUENTE-TOPOLOGIA-DOC

A0 es una única decisión documental de propiedad coordinada Bolsa/VEC/DBA. No
implementará código ni SQL. Este R1 no anticipa ni diseña sus valores: A0 debe
fijarlos de forma literal, sin usar V1, y obtener doble `GO` independiente e
integración por Dirección antes de desbloquear A1.

A0 debe cerrar conjuntamente:

1. acción, finalidad, perfil, tipo y recurso, huellas, motivo, correlación,
   emisor, audiencia, vigencia y consumo V2 exactos de la autorización de alta;
2. la función VEC nominal exacta y su consumo obligatorio para toda alta nueva,
   dentro del mismo `COMMIT` que crea el abierto;
3. una sola topología viable de LOGIN, `session_user` y membresías compatible
   simultáneamente con la fachada VEC y la fachada Bolsa. Si esa topología exige
   una enmienda VEC, A0 debe derivar una tarea previa propiedad de VEC/DBA,
   integrada antes de cualquier SQL de Bolsa;
4. el productor V2 exacto del hecho o propuesta durable, una representación sin
   PII ni huellas derivadas de PII, su objeto y clave bloqueables, su función de
   captura o consulta y el vínculo exacto con la propuesta confirmada;
5. si hecho, propuesta y abierto nacen en la misma transacción, el orden y la
   fuente previa que los autoriza; si existe una fuente durable previa, su
   productor y bloqueo;
6. el orden total de locks y relecturas. No son autoridad una capacidad en
   memoria, un recibo externo, HTTP, Contratación temporal, escalares ni V1;
7. que `Recuperar` reconcilie internamente las seis piezas locales y el consumo
   VEC, sin ampliar el recibo ni exponer PII; y
8. `NO-GO` obligatorio si cualquiera de autoridad, fuente, identidad o
   atomicidad sigue abierta.

A1 permanece bloqueado hasta el doble `GO` y la integración de A0. Una enmienda
VEC/DBA exigida por la topología deberá ser una dependencia integrada antes de
autorizar cualquier SQL de Bolsa.

## Contrato futuro reservado en aplicación

Después de A0, A1 conserva exactamente estos dos ficheros nuevos:

```text
internal/modules/bolsa/application/alta_durable_llamamiento_abierto_v2.go
internal/modules/bolsa/application/alta_durable_llamamiento_abierto_v2_test.go
```

El fichero productivo definirá únicamente:

- `SolicitudAltaDurableLlamamientoAbiertoV2`, capacidad opaca, no
  serializable, no reconstruible y redactada;
- `ConsultaRecuperacionAltaLlamamientoAbiertoV2`, localizador opaco de una
  operación y su compromiso sellado, sin autoridad de escritura;
- `ReciboAltaDurableLlamamientoAbiertoV2`, valor minimizado de lista positiva;
  y
- el puerto `AltaDurableLlamamientoAbiertoV2`, con los métodos nominales
  `Confirmar` y `Recuperar`.

La forma contractual reservada es:

```text
AltaDurableLlamamientoAbiertoV2
  Confirmar(context.Context, SolicitudAltaDurableLlamamientoAbiertoV2)
    -> ReciboAltaDurableLlamamientoAbiertoV2, error
  Recuperar(context.Context, ConsultaRecuperacionAltaLlamamientoAbiertoV2)
    -> ReciboAltaDurableLlamamientoAbiertoV2, error
```

La solicitud y la consulta tendrán campos privados y valor cero inválido. La
solicitud bloqueará codificación y decodificación JSON, XML, texto, binario,
Gob, CBOR y YAML. `String`, `GoString`, cualquier `Format` y `LogValue`
devolverán una marca constante redactada sin referencias, huellas, decisión,
motivo, identidad ni causa privada. No habrá constructor público desde bytes o
escalares ni getters de propuesta, decisión, identidad o evidencia.

La consulta solo podrá localizar el resultado histórico exacto. No podrá
crear el hecho, repetir el efecto, completar datos ausentes ni convertir una
ausencia en autorización.

## Recibo: lista positiva exclusiva

`ReciboAltaDurableLlamamientoAbiertoV2` contendrá exactamente estos campos y
ningún otro:

```text
ReciboRef
OperacionRef
LlamamientoRef
BolsaRef
NecesidadRef
PropuestaRef
Version
Estado = abierto
HistoriaRef
AuditoriaRef
EventoRef
HuellaAltaSHA256
ConfirmadaEn
```

La lista es cerrada. El recibo no contendrá candidato, sujeto, selección,
posición, contacto, documento, identidad, perfil, credenciales ni huellas de
PII. Tampoco expondrá decisión V2, motivo, evidencia, capacidad, nonce,
consumo, clave, detalle de auditoría, carga de outbox o diagnósticos.

`HuellaAltaSHA256` comprometerá el intento técnico canónico y no será una
huella de datos personales. Su preimagen excluirá identidad, candidato,
selección, posición, contacto, documento y cualquier atributo personal. La
representación exacta y su vector se congelarán en A1 antes de persistencia.

## Unidad física V2 aislada de V1

La única unidad futura de persistencia es:

```text
deploy/postgresql/bolsa_llamamientos_v2/
internal/modules/bolsa/adapters/postgresllamamientosv2/
```

Esquema PostgreSQL exclusivo:

```text
vec_bolsa_llamamientos_v2
```

Roles `NOLOGIN`, sin herencia funcional desde V1:

```text
vec_bolsa_llamamientos_v2_propietario
vec_bolsa_llamamientos_v2_migrador
vec_bolsa_llamamientos_v2_ejecutor
```

Estos nombres quedan reservados, pero no acreditan una topología runtime. En
particular, conceder a un LOGIN membresía directa del ejecutor de Bolsa y del
consumidor VEC es incompatible con la comprobación vigente sobre `session_user`;
`SET ROLE` no lo corrige. Solo A0 puede fijar una topología viable o exigir la
enmienda previa propiedad de VEC/DBA.

Funciones exteriores nominales:

```text
vec_bolsa_llamamientos_v2.confirmar_alta_llamamiento_abierto_v2
vec_bolsa_llamamientos_v2.recuperar_alta_llamamiento_abierto_v2
```

Migración inicial exacta:

```text
deploy/postgresql/bolsa_llamamientos_v2/migraciones/000001_alta_llamamiento_abierto_v2.up.sql
deploy/postgresql/bolsa_llamamientos_v2/migraciones/000001_alta_llamamiento_abierto_v2.down.sql
```

Objetos propios del esquema:

```text
hecho_creacion_llamamiento_abierto_v2
llamamiento_abierto_v2
historia_llamamiento_v2
auditoria_llamamiento_v2
outbox_llamamiento_v2
recibo_alta_llamamiento_v2
```

`hecho_creacion_llamamiento_abierto_v2` reserva la materialización local
append-only del nacimiento, pero su nombre no la convierte en fuente
autoritativa. A0 debe identificar su productor y fuente previa o decidir el
orden que evita circularidad si nace en la misma transacción.
`llamamiento_abierto_v2` es la proyección actual que B4-B podrá transicionar en
un corte posterior. La primera fila de `historia_llamamiento_v2`, la auditoría,
el evento outbox y el recibo pertenecen a la misma operación y no existen
parcialmente.

V1 permanece histórica e inmutable. V2 no tendrá vistas, triggers, claves
foráneas, sinónimos, lecturas, escrituras, backfill, conversión o proyección
sobre `bolsa_llamamientos` V1. Ninguna función V2 invocará
`guardar_propuesta_v1`, `revalidar_decision_bolsa_llamamientos_v1` ni otro
objeto V1.

## Atomicidad local con la fachada VEC V2

La unidad V2 de Bolsa debe residir en la misma instancia y base PostgreSQL que
la fachada VEC V2. El adaptador futuro usará la misma conexión y la misma
transacción local `SERIALIZABLE READ WRITE`. Antes de A1, A0 fijará el orden
total de endurecimiento de sesión, locks, relecturas, replay/colisión,
construcción con `NuevoLlamamientoAbierto`, consumo VEC, materialización de las
seis piezas y validación previa al único `COMMIT`; este R1 no autoriza ni
anticipa ese orden concreto.

Toda alta nueva debe invocar y consumir la autorización exacta mediante la
función nominal VEC que fije A0. No existe alta exenta ni consumo condicional.

Las funciones PostgreSQL no abren conexiones ni confirman transacciones por
su cuenta. El consumo VEC y las seis piezas de Bolsa comparten un solo desenlace.
El adaptador de Bolsa no lee ni escribe tablas VEC; VEC conserva su autoridad y
solo expone la fachada nominal que su propietario apruebe.

Si no puede garantizarse misma base, conexión y transacción local, B4-A y todo
B4 quedan en `NO-GO`. No existe alternativa mediante otra base, transacción
distribuida, `dblink`, llamada remota, consumo previo, marca previa,
compensación, doble escritura, saga ni consistencia eventual.

La presencia actual de `autorizacion_atestada_v2` no acredita por sí sola esta
composición ni autoriza producción. La concesión futura de `EXECUTE` a la
fachada necesaria pertenece a VEC/DBA, debe ser mínima y revisada, y no concede
DML ni lectura de sus tablas al rol de Bolsa.

## Frontera SQL y autoridad de reglas

SQL se limita a identidad runtime, límites, bloqueos, unicidad, integridad
física, inserciones append-only, recuperación y atomicidad. No:

- ejecuta ni copia el CAS de B2;
- reimplementa `NuevoLlamamientoAbierto` ni
  `LlamamientoAbierto.TransicionarATerminal`;
- invoca, copia o aproxima B3;
- decide estados terminales, selección, candidato, orden o siguiente persona;
- interpreta reglas de Bolsa, PRE-CAP, política o motivo; ni
- lee o escribe V1 o tablas de otro módulo.

La única construcción de B4-A es `NuevoLlamamientoAbierto` en Go. La única
transición futura seguirá siendo
`TransicionarLlamamientoConOrdenTerminalAutorizadaV2` en B4-B. Restricciones
SQL como no nulo, gramática técnica, unicidad, FK interna y `Estado=abierto`
protegen la representación persistida; no se presentan como una segunda
autoridad de negocio.

## Idempotencia, replay y resultado ambiguo

`OperacionRef` identifica semánticamente el alta. `HuellaAltaSHA256` liga de
forma canónica la operación, las cuatro referencias, versión, estado abierto y
los compromisos técnicos que A0 ligue a la fuente durable y a la autorización
V2 consumida, sin PII.

- mismo `OperacionRef` y misma carga/huella exactas: devuelve byte y
  semánticamente el mismo recibo confirmado, con los mismos trece campos y sin
  nuevas filas de hecho, agregado, historia, auditoría u outbox;
- mismo `OperacionRef` con cualquier carga o huella distinta: colisión cerrada
  y rollback total;
- recibo existente sin paquete local completo y coherente: corrupción o
  resultado indeterminado; nunca se fabrica o completa un recibo;
- `40001` o `40P01` antes de intentar `COMMIT`: permiten reintento acotado en
  una transacción nueva, siempre con la misma solicitud sellada y con nueva
  relectura durable y revalidación/consumo VEC; la solicitud sola no autoriza;
- error de `COMMIT`, pérdida de respuesta o conexión tras intentarlo: resultado
  indeterminado; queda prohibido repetir a ciegas y se exige `Recuperar`; y
- `Recuperar` es de solo lectura, coteja operación, huella, las seis piezas
  locales y el consumo VEC correspondiente, y devuelve el recibo histórico
  exacto o una ausencia demostrada sin ampliar sus trece campos.

Una autorización caducada no puede crear un efecto nuevo. Recuperar un recibo
histórico confirmado no vuelve a consumir la autorización ni exige que siga
vigente: solo acredita el `COMMIT` anterior.

## Matriz de errores futura

| Condición | Resultado público | Efecto durable | Reintento |
| --- | --- | --- | --- |
| Solicitud/consulta cero, corrupta o no local | centinela de entrada inválida, opaco | ninguno | no |
| Codec o intento de reconstrucción | centinela de serialización prohibida | ninguno | no |
| Fuente durable, propuesta o autorización V2 ausente, retirada o divergente | denegación opaca | ninguno | no |
| VEC V2 no disponible o no revalidable | denegación/indisponibilidad opaca | ninguno | no se interpreta como autorización |
| Misma operación y misma huella | mismo recibo | ninguno nuevo | replay resuelto |
| Misma operación con carga distinta | centinela de colisión | ninguno | no |
| Agregado o historia preexistentes incoherentes | centinela de integridad/indeterminado | ninguno nuevo | solo investigación/recuperación |
| `40001`/`40P01` antes de `COMMIT` | error temporal opaco | rollback total | sí, nuevo intento acotado |
| Cancelación observada antes de `COMMIT` | cancelación opaca | rollback total | no automático |
| Desenlace de `COMMIT` ambiguo | resultado indeterminado | desconocido hasta recuperar | solo `Recuperar` |
| Recuperación exacta confirmada | recibo histórico exacto | ninguno | finaliza |
| Recuperación con ausencia demostrada | alta no confirmada | ninguno | una nueva operación exige nueva autorización |
| ACL, RLS, identidad o topología inválidas | denegación opaca | ninguno | no |

Los nombres de centinela se congelarán en A1. No envolverán SQL, DSN,
referencias, huellas, decisión, motivo, identidad ni errores privados.

## Matriz de rollback y consistencia

| Punto de fallo | Postcondición obligatoria |
| --- | --- |
| Antes de abrir la solicitud | cero llamada VEC y cero escritura |
| Construcción B2 rechazada | cero llamada de efecto y cero escritura |
| Revalidación/consumo VEC rechazado | rollback de toda la transacción |
| Inserción del hecho local | ninguna de las seis piezas sobrevive |
| Inserción del agregado abierto | hecho, agregado y resto revierten |
| Historia inicial | cero historia parcial y cero puntero huérfano |
| Auditoría | cero alta sin auditoría |
| Outbox | cero alta sin evento durable |
| Recibo | cero efecto presentado sin recibo |
| Validación defensiva del recibo | rollback total |
| Cancelación pre-`COMMIT` | rollback total y cero recibo de éxito |
| Respuesta perdida postintento | no repetir; recuperar en otra conexión |

Las futuras pruebas inyectarán un fallo en cada escritura y verificarán por
conteo e identidad que no queda estado parcial.

## ACL, RLS y privilegio mínimo

| Autoridad | Privilegios permitidos | Privilegios prohibidos |
| --- | --- | --- |
| `..._propietario` | propiedad exacta de esquema, objetos y funciones V2 | LOGIN, uso funcional ordinario, autoridad V1 |
| `..._migrador` | asumir propietario solo durante migración gobernada | LOGIN, ejecución funcional, DML ordinario |
| `..._ejecutor` | reserva de `USAGE` de esquema y `EXECUTE` solo en las dos funciones nominales, supeditada a A0 | `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `TRUNCATE`, `REFERENCES`, secuencias, funciones internas, `SET ROLE` |
| `PUBLIC` | ninguno | esquema, tipos, tablas, secuencias y funciones |

Todas las tablas tendrán `ENABLE ROW LEVEL SECURITY` y `FORCE ROW LEVEL
SECURITY`, política exclusiva del propietario exacto e inmutabilidad donde
corresponda. El ejecutor no recibirá DML directo ni una política que lo haga
equivalente al propietario. Las funciones serán `SECURITY DEFINER`, fijarán
`search_path = pg_catalog`, comprobarán identidad efectiva y no expondrán
oráculos de existencia.

El rol V2 no heredará roles V1. A0 decidirá la única identidad runtime viable;
este documento no concede membresías ni presupone que `EXECUTE` baste para
superar la validación de `session_user`. Cualquier privilegio sobre la fachada
VEC V2 lo concede su autoridad sobre una función exacta y no transfiere
propiedad ni acceso a tablas VEC.

## Privacidad, seguridad, i18n y accesibilidad

| Superficie | Permitido | Prohibido |
| --- | --- | --- |
| Hecho/agregado/historia | referencias opacas, versión, estado técnico y compromisos mínimos | candidato, sujeto, selección, posición, contacto, documento e identidad |
| Auditoría/outbox | referencias técnicas minimizadas y huella del alta sin PII | payload de propuesta, credenciales, perfil, evidencia V2 o texto libre |
| Recibo | lista positiva de trece campos | cualquier campo adicional o huella de PII |
| Errores/logs/trazas | códigos constantes, correlación operativa gobernada y redacción | entradas, SQL, DSN, secretos, referencias personales o material de autorización |
| Copias/retención | política aprobada por sus autoridades | borrado silencioso o plazo inventado por B4-A |

No se usan datos reales. Cifrado, claves, backup, restauración, observabilidad,
retención y destrucción requieren sus autoridades y aprobaciones. Este corte no
atribuye competencia ni acredita RGPD, ENS, ENI o producción.

B4-A no tiene HTTP, interfaz ni texto visible. Un consumidor posterior deberá
usar claves i18n, formatos localizados y estados accesibles por texto además de
color. No procede revisión visual en este corte.

## Límites y paradas duras

Los siguientes límites se congelarán en A1/SQL antes de reservar memoria o
adquirir autoridad:

- una sola operación y un solo agregado por llamada;
- referencias y huellas con gramática y longitud máximas cerradas;
- versión positiva dentro del entero seguro común;
- instantes UTC canónicos con precisión acordada con PostgreSQL;
- carga opaca con presupuesto fijo, sin colecciones libres, mapas abiertos ni
  texto funcional; y
- tiempos de sentencia, lock, inactividad y cancelación obligatorios.

El corte A0 y todos los posteriores se detienen en `NO-GO` si:

1. autoridad, fuente, identidad o atomicidad siguen abiertas;
2. la solicitud puede nacer sin relectura y bloqueo de fuente durable V2;
3. se usa O6-01, HTTP, bytes, escalares, tablas CT o V1 como fuente;
4. `NuevoLlamamientoAbierto` deja de ser la única construcción B2;
5. B4-A invoca B3 o realiza una terminal;
6. no existe aislamiento físico V2 con los nombres fijados;
7. no puede compartirse base, conexión y transacción local con VEC V2;
8. la topología exige dos membresías incompatibles, `SET ROLE`, DML directo o
   acceso a tablas ajenas;
9. replay, colisión, cancelación o `COMMIT` ambiguo no fallan cerrados;
10. alguna pieza local puede sobrevivir sin las otras o sin consumo VEC; o
11. se pretende declarar `autorizacion_atestada_v2`, B4-A u O6-03 aptos para
    producción.

## Secuencia obligatoria de futuros cortes

```text
CT-LITE-O6-03-B4-A-DOC-R1 (este candidato)
  -> doble revisión independiente del hash exacto: GO + GO
    -> integración por Dirección
      -> CT-LITE-O6-03-B4-A0-AUTORIDAD-FUENTE-TOPOLOGIA-DOC
        -> doble revisión independiente del hash exacto: GO + GO
          -> integración por Dirección
            -> CT-LITE-O6-03-B4-A1-CONTRATO-GO
              -> A2 roles y topología V2
                -> A3 migración 000001 y funciones nominales
                  -> A4 adaptador postgresllamamientosv2
                    -> A5 PostgreSQL real: replay, recuperación, ACL/RLS y rollback
                      -> revisión independiente e integración B4-A
                        -> documento/contrato B4-B
```

A0 tiene propiedad coordinada Bolsa/VEC/DBA y es la única barrera documental
añadida antes de A1. A1 modifica solo los dos ficheros de aplicación reservados.
Crea el contrato opaco y sus pruebas unitarias; no añade PostgreSQL, adaptador,
composición, HTTP, V1 ni ningún fichero fuera de ese par. A2–A5 tendrán
write-sets nuevos y disjuntos o se ejecutarán en secuencia. Ningún corte se
integra sin la revisión independiente exigida de su hash exacto.

## Matriz de pruebas futuras

A1 deberá cubrir valor cero, construcción restringida al mecanismo interno que
A0 ligue a la relectura durable sin convertir la solicitud en autoridad,
imposibilidad de constructor escalar, copias defensivas, redacción, bloqueo de
todos los codecs, lista positiva exacta del recibo, consulta sin autoridad y
errores opacos.

Los cortes PostgreSQL deberán cubrir, en una instancia efímera real: alta
nominal, replay exacto desde otro proceso, colisión de operación, propuesta o
hecho alterados, cada referencia alterada, versión/estado inválidos, dos o más
sesiones concurrentes, VEC V2 válida/caducada/revocada, indisponibilidad VEC,
fallo en cada escritura, cancelación pre-`COMMIT`, respuesta perdida,
recuperación tras reinicio, ACL/RLS negativas, timeouts, `down` protegido,
reinstalación y ausencia de PII en tablas, WAL de prueba, recibos, outbox,
auditoría, errores y logs.

## Limitaciones y siguiente corte

Este documento no crea tipos Go, puertos, adaptadores, roles, SQL, tablas,
funciones, migraciones, recibos efectivos, composición ni pruebas dinámicas.
No acredita un alta, un `COMMIT`, comunicación, aceptación, renuncia,
expiración, siguiente candidato, API, web, E2E, despliegue o producción.

El siguiente corte exacto es
`CT-LITE-O6-03-B4-A0-AUTORIDAD-FUENTE-TOPOLOGIA-DOC`, bloqueado hasta que el
hash exacto de este R1 reciba doble `GO` independiente e integración por
Dirección. A1 continúa bloqueado hasta que A0 reciba su doble `GO` y quede
integrado; cualquier enmienda VEC/DBA que A0 derive deberá preceder al SQL de
Bolsa.
