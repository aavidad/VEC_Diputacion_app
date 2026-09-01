# CT-LITE-O6-03-B4-A-DOC — alta durable de llamamiento abierto V2

Fecha: 1 de septiembre de 2026.

## Estado, autoridad y alcance

Este documento parte de la base exacta
`c8cc4312b063ddca7294dfe27dc673ad1a4676d0`, que integra la autoridad
`CT-LITE-O6-03-B4-R1`. Es un corte exclusivamente documental: no implementa
producto, no cierra `O6-03`, no cambia métricas y no autoriza integración,
despliegue ni producción.

La capability fijada es `AltaDurableLlamamientoAbiertoV2`, propiedad exclusiva
de la aplicación de Bolsa. Su responsabilidad única es crear de forma durable
un `LlamamientoAbierto` en estado técnico `abierto` a partir de un hecho local
de creación posterior a la validación autoritativa de la propuesta.

Invariante del corte:

> Una transacción `SERIALIZABLE READ WRITE` materializa hecho local de
> creación + agregado abierto + historia inicial + auditoría + outbox + recibo,
> todo `COMMIT` o todo `ROLLBACK`.

Write-set único:

```text
docs/portal_vec/decision_o6_03_b4_a_alta_durable_llamamiento_abierto_v2_2026-09-01.md
```

El siguiente corte, únicamente después de revisión independiente e integración
de este documento, es `CT-LITE-O6-03-B4-A1-CONTRATO-GO`.

## Autoridades leídas y precedencia

Esta decisión consume, sin modificar ni reinterpretar:

- B2: `LlamamientoAbierto`, `DatosLlamamientoAbierto` y
  `NuevoLlamamientoAbierto` en
  `internal/modules/bolsa/domain/llamamiento_abierto.go`;
- PRE-CAP: `OrdenTerminalLlamamientoAutorizadaV2` en aplicación de Bolsa;
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

## Responsabilidad única de B4-A

B4-A crea el agregado abierto y nada más. Para una operación nueva debe:

1. acreditar que la solicitud nació dentro del flujo local de Bolsa después
   de validar la propuesta autoritativa exacta;
2. derivar en aplicación los datos de apertura y construir el valor mediante
   `NuevoLlamamientoAbierto`;
3. confirmar en una sola transacción el hecho local de creación, la proyección
   abierta, su primera historia, auditoría, outbox y recibo; y
4. devolver exclusivamente el recibo minimizado confirmado.

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

## Nacimiento exclusivo de la solicitud

`SolicitudAltaDurableLlamamientoAbiertoV2` solo puede nacer en proceso, dentro
de la aplicación de Bolsa, desde el flujo local que acaba de validar la
propuesta autoritativa. La validación debe conservar como capacidad privada la
procedencia y los compromisos exactos necesarios para el efecto; una mera
igualdad de referencias no la sustituye.

Queda prohibido construir, reconstruir, rehidratar o promover la solicitud
desde cualquiera de estas fuentes:

- `ReciboSolicitudLlamamientoBolsa` o cualquier otro recibo O6-01;
- HTTP, JSON, XML, texto, binario, Gob, CBOR, YAML, cabeceras, cookies,
  parámetros o almacenamiento del navegador;
- tablas, vistas, outbox, inbox o bytes de Contratación temporal;
- tablas o funciones V1 de Bolsa;
- una lista, mapa o estructura de referencias escalares aportada por un
  adaptador; o
- una proyección persistida que no conserve la capacidad local autoritativa.

El recibo O6-01 permite a Contratación temporal verificar su intercambio. No
concede a Bolsa autoridad de creación, no puede convertirse en el hecho local
y no autoriza a leer tablas de Contratación temporal.

## Contrato futuro reservado en aplicación

El siguiente corte reserva exactamente estos dos ficheros nuevos:

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

`hecho_creacion_llamamiento_abierto_v2` es la fuente local append-only del
nacimiento. `llamamiento_abierto_v2` es la proyección actual que B4-B podrá
transicionar en un corte posterior. La primera fila de
`historia_llamamiento_v2`, la auditoría, el evento outbox y el recibo pertenecen
a la misma operación y no existen parcialmente.

V1 permanece histórica e inmutable. V2 no tendrá vistas, triggers, claves
foráneas, sinónimos, lecturas, escrituras, backfill, conversión o proyección
sobre `bolsa_llamamientos` V1. Ninguna función V2 invocará
`guardar_propuesta_v1`, `revalidar_decision_bolsa_llamamientos_v1` ni otro
objeto V1.

## Atomicidad local con la fachada VEC V2

La unidad V2 de Bolsa debe residir en la misma instancia y base PostgreSQL que
la fachada VEC V2. El adaptador futuro usará la misma conexión y la misma
transacción local `SERIALIZABLE READ WRITE` para:

1. endurecer la sesión, identidad efectiva y límites;
2. localizar replay o colisión por `OperacionRef` y huella sellada;
3. abrir una sola vez la solicitud opaca y construir en Go el agregado con
   `NuevoLlamamientoAbierto`;
4. invocar la fachada nominal VEC V2 para releer, revalidar y consumir la
   autorización exacta cuando corresponda al efecto;
5. invocar `confirmar_alta_llamamiento_abierto_v2` para materializar las seis
   piezas locales; y
6. validar el recibo antes de un único `COMMIT`.

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
los compromisos técnicos autoritativos de la solicitud, sin PII.

- mismo `OperacionRef` y misma carga/huella exactas: devuelve byte y
  semánticamente el mismo recibo confirmado, con los mismos trece campos y sin
  nuevas filas de hecho, agregado, historia, auditoría u outbox;
- mismo `OperacionRef` con cualquier carga o huella distinta: colisión cerrada
  y rollback total;
- recibo existente sin paquete local completo y coherente: corrupción o
  resultado indeterminado; nunca se fabrica o completa un recibo;
- `40001` o `40P01` antes de intentar `COMMIT`: permiten reintento acotado en
  una transacción nueva, siempre con la misma solicitud sellada;
- error de `COMMIT`, pérdida de respuesta o conexión tras intentarlo: resultado
  indeterminado; queda prohibido repetir a ciegas y se exige `Recuperar`; y
- `Recuperar` es de solo lectura, coteja operación, huella y las seis piezas
  locales, y devuelve el recibo histórico exacto o una ausencia demostrada.

Una autorización caducada no puede crear un efecto nuevo. Recuperar un recibo
histórico confirmado no vuelve a consumir la autorización ni exige que siga
vigente: solo acredita el `COMMIT` anterior.

## Matriz de errores futura

| Condición | Resultado público | Efecto durable | Reintento |
| --- | --- | --- | --- |
| Solicitud/consulta cero, corrupta o no local | centinela de entrada inválida, opaco | ninguno | no |
| Codec o intento de reconstrucción | centinela de serialización prohibida | ninguno | no |
| Propuesta o capacidad autoritativa ausente, retirada o divergente | denegación opaca | ninguno | no |
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
| `..._ejecutor` | `USAGE` de esquema y `EXECUTE` solo en las dos funciones nominales | `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `TRUNCATE`, `REFERENCES`, secuencias, funciones internas, `SET ROLE` |
| `PUBLIC` | ninguno | esquema, tipos, tablas, secuencias y funciones |

Todas las tablas tendrán `ENABLE ROW LEVEL SECURITY` y `FORCE ROW LEVEL
SECURITY`, política exclusiva del propietario exacto e inmutabilidad donde
corresponda. El ejecutor no recibirá DML directo ni una política que lo haga
equivalente al propietario. Las funciones serán `SECURITY DEFINER`, fijarán
`search_path = pg_catalog`, comprobarán identidad efectiva y no expondrán
oráculos de existencia.

El rol V2 no heredará roles V1. Cualquier `EXECUTE` necesario sobre la fachada
VEC V2 lo concede su autoridad sobre esa función exacta; no transfiere
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

El corte futuro se detiene en `NO-GO` si:

1. la solicitud puede nacer fuera del flujo local autoritativo;
2. se usa O6-01, HTTP, bytes, escalares, tablas CT o V1 como fuente;
3. `NuevoLlamamientoAbierto` deja de ser la única construcción B2;
4. B4-A invoca B3 o realiza una terminal;
5. no existe aislamiento físico V2 con los nombres fijados;
6. no puede compartirse base, conexión y transacción local con VEC V2;
7. el ejecutor requiere DML directo, `SET ROLE` o acceso a tablas ajenas;
8. replay, colisión, cancelación o `COMMIT` ambiguo no fallan cerrados;
9. alguna de las seis piezas puede sobrevivir sin las otras; o
10. se pretende declarar `autorizacion_atestada_v2`, B4-A u O6-03 aptos para
    producción.

## Secuencia obligatoria de futuros cortes

```text
CT-LITE-O6-03-B4-A-DOC (este candidato)
  -> revisión independiente del hash exacto
    -> integración por Dirección
      -> CT-LITE-O6-03-B4-A1-CONTRATO-GO
        -> A2 roles y topología V2
          -> A3 migración 000001 y funciones nominales
            -> A4 adaptador postgresllamamientosv2
              -> A5 PostgreSQL real: replay, recuperación, ACL/RLS y rollback
                -> revisión independiente e integración B4-A
                  -> documento/contrato B4-B
```

A1 modifica solo los dos ficheros de aplicación reservados. Crea el contrato
opaco y sus pruebas unitarias; no añade PostgreSQL, adaptador, composición,
HTTP, V1 ni ningún fichero fuera de ese par. A2–A5 tendrán write-sets nuevos y
disjuntos o se ejecutarán en secuencia. Ningún corte se integra sin revisión
independiente de su hash exacto.

## Matriz de pruebas futuras

A1 deberá cubrir valor cero, nacimiento exclusivamente local, imposibilidad de
constructor escalar, copias defensivas, redacción, bloqueo de todos los codecs,
lista positiva exacta del recibo, consulta sin autoridad y errores opacos.

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

El siguiente corte exacto es `CT-LITE-O6-03-B4-A1-CONTRATO-GO`, limitado a
los dos ficheros futuros de aplicación ya reservados y bloqueado hasta que este
hash reciba revisión independiente e integración por Dirección.
