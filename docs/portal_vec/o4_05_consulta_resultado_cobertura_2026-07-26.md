# O4-05: consulta protegida del resultado de cobertura

**Fecha:** 26 de julio de 2026

**Estado:** núcleo, contrato PostgreSQL, adaptador Go, HTTP, cliente web y
registro modular verificados con `GO` independiente. Permanecen pendientes la
composición raíz con dependencias reales y el E2E contra PostgreSQL; las piezas
aisladas no autorizan producción

## Problema

Una decisión de cobertura puede alcanzar `COMMIT` y perder su respuesta. En
ese caso el cliente no sabe si existe recibo y no puede repetir el efecto. La
cancelación, un corte de red o un `503 operacion_pendiente` deben conservar
ese bloqueo hasta consultar el resultado en el primario.

## Decisión arquitectónica

La recuperación será otro caso de uso, no un reenvío del efecto:

- recibe únicamente la clave de idempotencia y la referencia del expediente;
- obtiene identidad, perfil y organización de la frontera confiable;
- exige una autorización de lectura con acción y finalidad propias;
- deriva dentro del núcleo solamente los alias HMAC del ámbito estable;
- busca por esos alias y coteja organización y expediente en PostgreSQL;
- exige que el lector fuerte O4-04E reconstruya y valide el terminal histórico;
- publica una unión cerrada: `confirmado` o `no_observable`.

El caso de uso de consulta dependerá de una interfaz estrecha que no expone
reservar, reapropiar ni confirmar. Así se evita que un error de composición o
de adaptación convierta una lectura en efecto.

La vía, el motivo, la identidad semántica, la versión funcional y la
predecesora no forman parte de la consulta. Esos datos pertenecen a la
intención histórica ya confirmada. Volver a resolverlos desde el catálogo
vigente rompería la recuperación al publicar, retirar o sustituir una entrada.
El cliente tampoco puede aportar su versión histórica como autoridad.

## Infraestructura reutilizada

Cobertura ya dispone de:

- una preimagen de ámbito estable ligada a clave de idempotencia,
  organización, expediente, actor y perfil;
- alias HMAC multigeneración que convergen en una raíz durable;
- reserva, confirmación y terminal históricos;
- un lector interno O4-04E que verifica reserva, recibo, auditoría, outbox y
  la prueba funcional concedida o denegada;
- restauración y validación nominal del recibo.

No hace falta inventar otra tabla, guardar el motivo en claro ni crear otro
algoritmo de idempotencia. Sí hace falta una función exterior de solo lectura
que una los alias existentes con el lector fuerte O4-04E. Una ausencia se
proyectará de forma uniforme como `no_observable`: nunca prueba que el efecto
no sucediera y nunca habilita su repetición.

La función tendrá un rol nominativo de lectura sin privilegios de confirmación,
gobierno, migración ni acceso directo a tablas. `PUBLIC` y los roles de efecto
no podrán ejecutarla.

## Modelo de confianza y nominalidad

La unión de aplicación `confirmado | no_observable` será opaca, tendrá sus
constructores restringidos al núcleo y cruzará un puerto sellado que los
adaptadores ordinarios no podrán implementar directamente. El adaptador
PostgreSQL solo podrá devolver una observación cruda y no confiable dentro de
una sesión técnica de lectura. El núcleo cotejará la rama exacta, la consulta,
los alias HMAC, las coordenadas, la reserva, el recibo y los instantes antes de
elevar esa observación a la unión nominal.

Esta nominalidad impide la fabricación directa por error de composición, pero
no es una atestación del origen. El ejecutor PostgreSQL, el driver y su
conexión, el rol nominativo, el primario y la función `SECURITY DEFINER` forman
un TCB auditado. Una corrupción de cualquiera de esos componentes podría
inventar también la observación cruda. Resistir ese supuesto exigiría una
atestación asimétrica independiente sobre la consulta, la rama, la evidencia y
el instante, firmada por un componente que leyera el primario y cuya clave
privada no estuviera disponible para el proceso adaptador. Un HMAC calculado
dentro del mismo proceso no aportaría esa garantía.

La `huella_orden_sha256` sirve solo dentro del lector fuerte para seleccionar y
cotejar la historia terminal. No forma parte del recibo publicado ni de la
respuesta de aplicación. El núcleo no la presentará como una prueba
independiente que el cliente o el adaptador puedan declarar.

## Contrato de transporte previsto

La API usará la ruta
`POST /api/vec/contratacion-temporal/cobertura/resultados`, separada de las
rutas de efecto, para que la clave y la referencia no aparezcan en URL,
historial o registros de proxy. El cuerpo contendrá exclusivamente
`expediente_ref` y `clave_idempotencia`. Actor, perfil, organización, roles,
permisos y credenciales no serán declarables. Tampoco se admitirán versión,
tipo, vía, motivo, identidad semántica ni predecesora.

El adaptador distinguirá:

- recibo terminal válido: `200`;
- terminal no observable: `202`;
- autenticación ausente o caducada: `401`;
- acceso no concedido: `403`;
- historia durable divergente: `409`;
- dependencia o proyección no confiable: `503`.

PostgreSQL usa `23505` para señalar una divergencia durable que aplicación
proyectará como `409`.
`55000` queda reservado a corrupción, barrera de migración o estado del lector
no utilizable y se proyectará como `503`. Una respuesta JSON mal formada,
contradictoria o no confiable también producirá `503`; nunca se convertirá en
`no_observable`. No se analizarán mensajes SQL para decidir la respuesta ni se
filtrarán detalles internos.

No emitirá `Retry-After`, no hará sondeo automático y no aceptará cookies,
almacenamiento web ni cabeceras libres de identidad. Solo un `200` con recibo
validado libera el bloqueo de la operación; cualquier otro resultado conserva
el bloqueo y no provoca otro intento del efecto.

## Revisión adversaria del primer contrato

Dos revisiones independientes emitieron `NO-GO` antes de integrar el primer
contrato. Detectaron:

1. dependencia del catálogo vigente para reconstruir una operación histórica;
2. denegaciones del PDP clasificadas como indisponibilidad;
3. una vista del autorizador que podía filtrar referencias al formatearse;
4. una respuesta contradictoria `ausente + terminal` degradada a pendiente.

Los puntos 2 a 4 están corregidos y probados. El punto 1 queda resuelto por el
lector histórico por ámbito descrito en este documento. Núcleo, PostgreSQL,
rotación, ACL/RLS y revisión independiente están verdes, pero la capacidad no
se contabiliza como productiva hasta cerrar adaptador, composición, HTTP y
E2E.

La revisión del contrato sustituto añadió otra condición: el puerto de
aplicación no debe quedar implementable directamente por infraestructura. El
núcleo controla una sesión de una sola lectura, síncrona y no retenible, y
solo la observación cruda puede atravesar la frontera PostgreSQL. La revisión
independiente ha verificado esta condición.

Una revisión de seguridad posterior detectó que la solicitud interna podía
serializarse por codecs genéricos. El corte se detuvo, se corrigió y volvió a
revisarse. La solicitud de aplicación y las tres capacidades sensibles del
puerto rechazan JSON, XML, texto, binario, gob, CBOR y YAML, tanto en
codificación como en reconstrucción. El DTO de salida del adaptador conserva
su interoperabilidad.

## Evidencia verificada del corte aislado

- Núcleo nominal y TCB sellado: `87a1c37`.
- Puerto mínimo, no implementable directamente y no serializable:
  `ae42b2f`.
- Caso de uso, autorización doble, minimización y clasificación de errores:
  `70e2d98`.
- Migración `000035`, rol nominativo mínimo, ACL y pruebas PostgreSQL:
  `93674c7`.
- Adaptador Go PostgreSQL `SERIALIZABLE READ ONLY`, cardinalidad exacta y
  clasificación cerrada de errores: `e36310a`.
- Cliente web sin cookies, almacenamiento, reintento ni sondeo automático:
  `7d4e1ec`.
- Adaptador HTTP de consulta, cancelación prevalente y contrato cerrado:
  `42920ae`.
- Registro modular de la quinta ruta con fallo atómico y manejador de solo
  lectura separado: `5965223`.
- Pruebas focales Go, `go vet`, formato y detector de carreras: verdes.
- Callback concurrente, doble, omitido, retenido y tardío: falla cerrado.
- Codecs estándar y reales de CBOR/YAML: bloqueo verificado para valor,
  puntero y reconstrucción; el DTO de salida continúa serializable.
- PostgreSQL 18.4 desde cero, upgrade, down protegido, alias HMAC,
  aislamiento `SERIALIZABLE READ ONLY`, límites temporales y ACL: verde.

Los commits anteriores cierran y revisan las piezas aisladas de recuperación.
No acreditan todavía un recorrido productivo: la siguiente entrega debe
inyectar en la raíz el rol lector, el ejecutor PostgreSQL, el lector del
núcleo, el servicio de aplicación, el contexto corporativo, el PDP, el
sellador y el reloj, y probar el recorrido completo sin exponer una capacidad
de efecto.

## Pruebas adversarias exigidas

Además de los casos funcionales, el corte debe demostrar:

- rechazo de callback omitido, repetido, concurrente, asíncrono, retenido o
  ejecutado después del retorno;
- cierre de la sesión y reversión protegida ante error, cancelación, plazo,
  `panic`, resultado tardío o incumplimiento del ciclo de una sola lectura;
- conservación de la cancelación y del plazo como errores, nunca como
  `no_observable`;
- rechazo de las contradicciones `no_observable + terminal`, `confirmado` sin
  evidencia, dos ramas terminales, estado desconocido o resultado acompañado
  de error;
- rechazo de mutaciones de organización, expediente, generaciones HMAC,
  referencias, versión, cercado, recibo, rama e instantes;
- conservación de las referencias preasignadas en una denegación sin publicar
  campos propios de un efecto aplicado;
- clasificación estable de `23505` como conflicto `409`, y de `55000`,
  corrupción de JSON, indisponibilidad y proyección no confiable como `503`;
- una transacción `SERIALIZABLE READ ONLY` contra el primario, sin reintento
  automático, con rollback seguro si la observación no puede elevarse.

## Límite conocido

El navegador conserva la intención pendiente solo en memoria. Tras una
recarga no existe una fuente segura para reconstruirla, porque el producto no
usa cookies, `localStorage` ni `sessionStorage`. La recuperación posterior a
una recarga requiere una bandeja durable de operaciones propias, indexada por
identidad corporativa y referencias opacas del servidor. Esa bandeja será un
corte independiente; no se sustituirá con estado persistido en el navegador.

Alta de solicitud tampoco reutilizará su operación de preparación como
consulta: actualmente puede reservar y generar referencias. Necesita un
lector terminal de solo lectura propio y se abordará después de cerrar
Cobertura.

## Puertas de cierre

1. El caso de uso no posee ningún puerto de efecto.
2. La consulta no depende del catálogo ni acepta semántica histórica del
   cliente.
3. La autorización de lectura se revalida antes de acceder a persistencia.
4. Identidad, organización, perfil, expediente o clave cruzados no revelan el
   terminal.
5. Confirmado y no observable son estados nominales no fabricables
   directamente por adaptadores ordinarios; la autenticidad de la observación
   cruda depende del TCB auditado descrito en este documento.
6. Cancelación y plazo no se confunden con ausencia de terminal.
7. HTTP y web no hacen reintento ni sondeo automático.
8. Cambio o retirada del catálogo no impiden recuperar el terminal.
9. La prueba E2E pierde la respuesta, consulta tras reinicio y obtiene el mismo
   recibo sin repetir la decisión.
