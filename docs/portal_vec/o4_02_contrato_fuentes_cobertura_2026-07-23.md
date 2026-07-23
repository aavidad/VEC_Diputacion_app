# O4-02: contrato de fuentes de cobertura

Fecha: 23 de julio de 2026.

Estado: cinco rondas de revisión emitieron `NO-GO` con pruebas de concepto
reproducibles. Las correcciones están implementadas en rama aislada; una nueva
revisión independiente y la integración siguen pendientes. Este documento no
concede un `GO` de integración, piloto o producción.

## NO-GO independiente y corrección

La primera revisión de esta reimplementación demostró:

1. un consumidor podía ignorar el contexto, esperar a que venciera la ventana
   y devolver un recibo fechado antes; el núcleo no revalidaba al regresar;
2. una credencial válida de una fuente podía contestar por otra definición
   publicada, porque la autoridad no estaba ligada al conector gobernado;
3. el constructor nominal de confirmación permitía afirmar que el TCB había
   verificado un HMAC sin aportar una firma de su clave autenticada.

La corrección añade, respectivamente:

1. comprobación incondicional del contexto y reloj final después del
   consumidor, más revalidación de ventana y catálogo;
2. igualdad exacta entre el `BackendRef` firmado de la fuente y
   `DefinicionFuenteRef` de la comprobación publicada;
3. prueba canónica Ed25519 firmada con la misma clave cuya posesión acreditó el
   verificador ante el TCB.

Las pruebas de regresión reproducen los tres ataques. El estado continúa en
`NO-GO` hasta que un agente distinto revise las correcciones.

La revisión posterior encontró dos defectos adicionales:

1. un replay exacto consumido en `t+2` dejaba de ser válido si el verificador
   emitía una confirmación nueva en `t+3`, porque el recibo durable se comparaba
   con el instante de esa confirmación posterior;
2. la coordinación completa del caso de uso residía en `ports`, en contra de
   la arquitectura hexagonal acordada para el repositorio.

La corrección conserva en el recibo la confirmación TCB firmada que autorizó el
primer efecto. El instante de consumo debe ser igual o posterior a esa
verificación original, igual o posterior a `Atestacion.EmitidaEn` y
estrictamente anterior a `Atestacion.ValidaHasta`. No se liga a una
confirmación obtenida durante un reintento. La firma original se valida con la
clave institucional y contra el mismo material de respuesta. La aplicación
sigue exigiendo que el recibo no proceda del futuro y revalida contexto, reloj,
firma, catálogo y ventana al regresar del consumidor. De este modo:

- primera consulta en `t+2`: crea un único efecto y un recibo en `t+2`;
- replay exacto en `t+3`: obtiene confirmación nueva en `t+3`, recupera el
  recibo original de `t+2` y no duplica el efecto;
- primera entrega retrodatada antes de la emisión: se rechaza;
- primera entrega posterior a la emisión pero anterior a la confirmación TCB:
  también se rechaza;
- misma clave durable con otra huella: se rechaza como conflicto;
- reloj regresivo, recibo futuro o fin exclusivo de TTL: se rechazan.

La coordinación se ha trasladado a `application.ServicioConsultaCobertura`.
`ports` conserva únicamente solicitudes, resultados probatorios, órdenes,
recibos e interfaces neutrales. Ningún puerto conoce la secuencia del caso de
uso, el timeout global ni la política de errores de disponibilidad. La
revisión independiente de esta segunda corrección sigue pendiente; por ello
el estado permanece en `NO-GO`.

La tercera revisión reprodujo una duplicación durable y una respuesta ambigua
posterior al `COMMIT`:

1. la unicidad por `(autoridad, generación, recibo)` permitía que la misma
   `PeticionRef` generase dos efectos si la fuente emitía un recibo nuevo;
2. una cancelación o timeout inmediatamente posteriores al `COMMIT` convertían
   un efecto confirmado en error, induciendo un reintento potencialmente
   duplicador.

La identidad durable se redefine así:

- clave primaria de efecto: `(OrganizacionRef, PeticionRef)`;
- huella de petición: canon cerrado con organización, expediente, versión,
  catálogo, vía, comprobación, procedencia, categoría, periodo e instante;
- huella de resultado: dominio propio, huella de petición, clave y valor de la
  comprobación y definición gobernada de la fuente;
- clave probatoria adicional: `(autoridad, generación, recibo)`.

Una fuente puede renovar generación, recibo, HMAC, ventana o instante de
evaluación y el verificador puede renovar firma e instante. Si petición y
resultado semántico coinciden, el consumidor devuelve el recibo durable
original y registra, como máximo, otra evidencia del mismo efecto. Cambiar
cualquier coordenada de la petición o el valor funcional del resultado es
conflicto. Reutilizar la misma clave probatoria con otra huella de respuesta,
o ligarla a otra petición, también es conflicto.

Tras el consumidor, un recibo durable válido y coherente con el reloj confirma
el `COMMIT` aunque el contexto se haya cancelado o agotado al regresar. No se
transforma en error porque ello ocultaría un efecto ya producido. Si no hay
recibo verificable se falla cerrado; el reintento con la misma `PeticionRef`
recupera el recibo original sin crear otro efecto. Un reloj regresivo o un
recibo fechado en el futuro continúan rechazándose. La tercera corrección
permanece en `NO-GO` hasta revisión ajena.

La cuarta revisión encontró dos defectos adicionales:

1. el reloj se comparaba únicamente antes y después del consumidor; tras
   observar `t+5` al validar el catálogo, un retroceso a `t+2` podía reabrir
   una respuesta cuyo fin exclusivo era `t+5`;
2. `ConfianzaAutoridadesFuenteAnalisis` implementaba dentro de `ports` la
   coordinación concreta de desafío, presentación y verificación.

La corrección mantiene un suelo temporal por operación. Todas las lecturas del
reloj del caso de uso —autenticación, catálogo, fuente, verificador,
preconsumo y salida— deben ser UTC canónico y mayores o iguales que cualquier
lectura anterior. Un retroceso falla cerrado antes de `ConsumirCobertura`; no
se ajusta el instante silenciosamente ni se reabre una evidencia caducada.

`ports` conserva la identidad inmutable y el contrato
`AutenticadorAutoridadesFuenteAnalisis`. La coordinación concreta se ha
trasladado a `adapters/seguridad`, donde el adaptador intercambiable crea el
desafío, solicita la presentación y verifica la confianza fijada por
composición. El caso de uso continúa siendo neutral respecto de web,
escritorio, CLI y MCP. Esta cuarta corrección permanece en `NO-GO` hasta una
nueva revisión independiente.

La quinta revisión detectó cuatro límites relacionados:

1. la aplicación obtenía el instante antes de una presentación y el adaptador
   lo reutilizaba después de aquella operación lenta;
2. el recibo histórico se verificaba con la clave de la operación actual, por
   lo que una rotación legítima K1→K2 rompía el replay;
3. el contrato de autenticación ocultaba la secuencia completa
   desafío→presentación→verificación;
4. varias pruebas dependían de timeouts de 5 ms y del planificador del equipo.

La aplicación crea ahora cada desafío, obtiene la presentación y solo entonces
lee el reloj autoritativo y solicita la verificación local. Esto se repite para
fuente, verificador y publicador: ninguna presentación reutiliza un instante
previo. El fin de credencial y raíz continúa siendo exclusivo y la
verificación conserva el horizonte máximo completo.

El recibo conserva la evidencia pública K1 original: datos de la credencial
institucional, firma de raíz, desafío, prueba de posesión, rol, instante,
identidad y clave pública. No contiene claves privadas, HMAC, tokens ni
secretos. La firma K1 se liga a la huella de petición y respuesta, atestación,
confirmación, instante y efecto durable. En el replay:

- la autoridad actual K2 se autentica primero con el reloj actual;
- la evidencia K1 se restaura y verifica por separado contra la confianza
  institucional;
- el recibo original se comprueba con K1, nunca con K2;
- una autoridad actual revocada, caducada o no confiable falla antes del
  consumidor y no puede usar el recibo histórico como autorización nueva;
- una revocación que afectaba a K1 durante su ventana histórica invalida el
  recibo, mientras una rotación posterior no reescribe el efecto ya probado.

Las pruebas de timeout usan un contexto controlado y activan
`DeadlineExceeded` en el punto exacto de cada dependencia. No esperan
milisegundos reales ni debilitan el límite productivo de cinco segundos. La
quinta corrección sigue en `NO-GO` hasta revisión independiente.

La quinta revisión independiente encontró una ruta residual con el mismo
defecto temporal: las operaciones de validación presupuestaria y cálculo de
coste podían entregar al verificador el instante leído antes de obtener la
presentación. El contraejemplo es determinista: inicio `t0`, presentación en
`t0+2 s`, credencial válida hasta `t0+6 s` y horizonte obligatorio de cinco
segundos. Verificar en `t0` la aceptaría; verificar en `t0+2 s` debe
rechazarla.

La sexta corrección se ha reconstruido sobre la base convergente exacta
`ba80e8f766e1054f35db15b47c2f4f13ea6b2221`, que ya integra O3-02 y conserva
la antigua orquestación de `ports` únicamente como contraste de prueba. El
candidato:

- traslada O4-02 sin reintroducir casos de uso multipuerto en `ports`;
- mantiene toda presentación productiva en `application`;
- crea el desafío con un instante inicial, pero exige otra lectura
  autoritativa después de recibir cada presentación;
- valida credencial, raíz, revocación y horizonte contra ese instante
  posterior;
- elimina la variante de `ValidarPresentacion` que no obligaba a aportar el
  instante posterior;
- revalida el conjunto de autoridades con un instante final que no puede ser
  anterior al comienzo de la preparación;
- prueba el contraejemplo tanto en validación RC como en cálculo de coste y
  demuestra que no se consulta la fuente ni se consume ningún efecto.

Esta sexta corrección no implica `GO`: requiere revisión independiente nueva.

## Alcance

O4-02 define la frontera hexagonal para comprobar una condición de una vía de
cobertura de una contratación temporal. La vía, la comprobación y su
procedencia no están compiladas en el núcleo: pertenecen a una publicación
versionada, vigente e inmutable del catálogo de cobertura.

El contrato aporta:

- petición mínima y representación binaria canónica;
- respuesta funcional completa, inmutable y autenticada;
- autenticación institucional de fuente, verificador y publicador;
- restauración de la publicación y prueba exacta de pertenencia;
- consumo durable único, replay exacto e idempotente y conflicto explícito;
- cancelación prioritaria y errores públicos sin causas privadas;
- límites comunes para web, escritorio, API, CLI y MCP.

La capa `application` autentica autoridades y coordina las llamadas. La capa
`ports` define contratos mínimos y validaciones locales de cada artefacto. Los
adaptadores implementarán esos contratos sin trasladar política funcional a
HTTP, SQL, escritorio o mensajería.

No existe una variante web del caso de uso. El contrato no contiene HTTP,
cookies, sesiones, almacenamiento del navegador ni cabeceras de identidad. Los
distintos canales deberán invocar la misma aplicación y no pueden aportar
raíces, credenciales de sistema, confirmaciones ni recibos.

## Flujo obligatorio

Una consulta válida sigue este orden:

1. validar petición, dependencias, organización y timeout total;
2. canonizar la petición;
3. autenticar fuente, verificador y publicador mediante tres desafíos nuevos;
4. probar que las tres autoridades tienen identidad, backend y clave distintos
   y que el backend de fuente coincide con la definición publicada;
5. recuperar del publicador la publicación completa del catálogo;
6. restaurar su canon y huella y comprobar vigencia y pertenencia exacta;
7. consultar la fuente definida por el catálogo;
8. comprobar coordenadas, cronología y atestación de la respuesta;
9. verificar el HMAC y la firma canónica del verificador institucional;
10. volver a comprobar ventana y catálogo;
11. consumir de forma durable la respuesta;
12. al regresar del consumidor, validar reloj y recibo durable; un recibo
    verificable confirma el efecto aun ante cancelación competitiva; sin él,
    aplicar contexto, ventana, catálogo y error público antes de fallar.

Cualquier ausencia o incoherencia falla cerrado. Una fuente no puede
autoverificar su respuesta ni autopublicar el catálogo que la autoriza.

## Petición canónica

`SolicitudConsultarCobertura` solo transporta:

| Grupo | Coordenadas |
| --- | --- |
| Operación | petición, organización, expediente y versión |
| Catálogo | referencia, versión y huella SHA-256 |
| Selección | vía y comprobación exigible completas |
| Procedencia | clave y definición gobernada del conector |
| Entrada funcional | categoría y periodo previsto |
| Tiempo | instante UTC canónico de solicitud |

La representación usa el dominio
`VEC-CT-FUENTE-COBERTURA-PETICION-V1`, longitudes explícitas y enteros en
orden de red. No depende de JSON, orden de mapas, locale, zona horaria ni
formato de base de datos.

La respuesta incorpora la huella SHA-256 de esta petición canónica. Así queda
ligada también a orden, obligatoriedad, procedencia y cualquier futura
coordenada canónica, no solo a una selección nominal.

## Respuesta autenticada

La preimagen con dominio
`VEC-CT-FUENTE-COBERTURA-RESPUESTA-V1` cubre:

- petición y su huella canónica;
- organización, expediente y versión;
- identidad exacta del catálogo;
- vía, procedencia, categoría y periodo;
- clave, resultado, fuente, recibo e instante de la comprobación;
- definición gobernada de la fuente;
- autoridad, generación y único recibo de atestación;
- ventana `EmitidaEn`–`ValidaHasta`.

No se admite `Detalle` libre en la respuesta del proveedor. Esto evita que un
conector propague nombres, DNI, observaciones o mensajes internos. La
preimagen, resultado, atestación y confirmaciones se formatean de forma
redactada para `fmt` y `slog`.

`FuenteRef` debe ser exactamente la autoridad de la atestación y el recibo
funcional debe ser exactamente el recibo atestado. No existen dos identidades
o dos recibos probatorios con significado ambiguo.

El sello tiene dominio propio:

```text
hmac-sha256:fuente-cobertura-respuesta/v<generación>:<64 hexadecimales>
```

La fuente sella; el núcleo no conoce el secreto; una autoridad separada
verifica y emite una confirmación ligada a la huella de todo el material,
fuente, generación, recibo, sello y ventana. Alterar cualquier coordenada
invalida la respuesta.

La confirmación no es una declaración nominal. Su preimagen canónica con
dominio `VEC-CT-CONFIRMACION-COBERTURA-V1` contiene:

- identidad del verificador;
- huella de la petición canónica;
- huella de la preimagen completa de respuesta;
- autoridad, generación, recibo y sello HMAC;
- ventana de respuesta e instante de verificación.

El verificador firma esta preimagen con la misma clave Ed25519 que usó para
probar posesión en su credencial institucional. El núcleo verifica la firma
con la clave pública obtenida de la confianza del servidor. Pasar 64 bytes al
constructor, firmar con otra clave o alterar una coordenada no concede
confianza.

## Autoridades y raíces institucionales

O4-02 reutiliza el motor común Ed25519 de O3-03. No crea otra biblioteca de
credenciales, roles, raíces o revocaciones. Añade exclusivamente estos roles
de protocolo:

- `fuente_cobertura`;
- `verificador_cobertura`;
- `publicador_catalogo_cobertura`.

Cada autoridad presenta una credencial institucional y firma un desafío que
incluye nonce CSPRNG de 256 bits, huella de petición, organización, audiencia
y rol. La confianza se fija en la composición del servidor. Nunca procede del
cliente.

La segregación exige diferencias simultáneas en:

- `AutoridadRef`;
- `BackendRef` canónico firmado;
- clave pública de prueba de posesión.

Dos wrappers o aliases del mismo backend no son dos autoridades. La
verificación conserva las reglas comunes de vigencia, rotación de raíces,
última emisión permitida y revocación por autoridad y serie.

Además, el `BackendRef` de la fuente debe coincidir exactamente con
`DefinicionFuenteRef` de la comprobación que pertenece al catálogo restaurado.
No hay listas compiladas: una nueva procedencia se habilita publicando su
definición y emitiendo una credencial institucional para ese mismo ámbito. Una
credencial de SAE no puede contestar una comprobación definida para Bolsa.

## Catálogo dinámico gobernado

El publicador no devuelve un booleano. Devuelve la publicación completa. El
núcleo:

1. recalcula canon y huella con
   `RestaurarCatalogoViasCobertura`;
2. exige identidad exacta referencia–versión–huella;
3. impide usar una publicación posterior a la petición;
4. exige vigencia tanto al solicitar como al comprobar;
5. busca la vía por clave;
6. exige igualdad completa de clave, orden, obligatoriedad, procedencia y
   definición de fuente.

Una vía nueva funciona mediante una publicación nueva, sin `switch`,
allowlist compilada ni recompilación del núcleo.

## Consumo durable e idempotencia

`ConsumidorCobertura` recibe una orden inmutable ligada a:

- petición, organización, expediente y versión;
- huellas SHA-256 de petición semántica y resultado funcional;
- autoridad, generación y recibo de respuesta;
- huella SHA-256 de la respuesta completa;
- atestación y confirmación del verificador;
- publicación confirmada del catálogo.

El adaptador durable deberá imponer, en una misma transacción:

1. unicidad de efecto por `(organización, petición)`;
2. igualdad de huella de petición y huella de resultado al reintentar;
3. unicidad probatoria por `(autoridad, generación, recibo)`.

Para la misma identidad de efecto:

- mismas huellas semánticas: devuelve exactamente el recibo original;
- recibo o firma nuevos con igual semántica: son evidencia adicional, no otro
  efecto;
- otra petición semántica u otro resultado: devuelve
  `ErrRespuestaCoberturaYaConsumida`;
- misma clave probatoria con otra huella de respuesta: devuelve conflicto;
- respuesta caducada antes de llamar al consumidor: la aplicación la rechaza.

El recibo persistido conserva también la confirmación TCB original. En un
replay no se sustituye por la confirmación recién obtenida: se verifica de
nuevo su firma, su correspondencia con la misma respuesta y que el consumo no
sea anterior a aquella autorización.

Los dobles adversariales prueban replay concurrente, unicidad y conflicto. No
se declara que exista todavía un adaptador productivo. La implementación
durable, su transacción con expediente/auditoría y las pruebas de reinicio
corresponden a la tarea de persistencia y composición.

Al regresar de `ConsumirCobertura`, la aplicación consulta siempre contexto y
reloj. Primero rechaza un reloj final anterior al instante preconsumo. Después
valida el recibo contra la evidencia original firmada y las huellas semánticas
de la orden actual. Un recibo verificable, no futuro, confirma el efecto aunque
el contexto haya vencido justo después del `COMMIT`. Sin recibo verificable,
la cancelación o el timeout producen fallo cerrado y el reintento recupera por
`PeticionRef`. La validez temporal del recibo se prueba contra la confirmación
que autorizó el primer efecto y la ventana de su atestación, no contra una
confirmación nueva obtenida al reintentar.

## Límites e interoperabilidad

| Regla | Límite |
| --- | --- |
| Timeout total | Más de cero y máximo 5 segundos |
| Vida de respuesta | Más de cero y máximo 5 segundos |
| Fin de ventana | Exclusivo |
| Periodo | 100 años exactos como máximo |
| Versión de expediente | 1 a 2^53−1 |
| Versión de catálogo | 1 a 2^53−1 |
| Instantes | UTC canónico, precisión máxima de microsegundo |
| Preimagen | Máximo 64 KiB |

El límite 2^53−1 permite transportar versiones de forma exacta también por
clientes JSON/JavaScript. Los periodos son fechas civiles UTC y no duraciones
aproximadas.

Tras cada dependencia se comprueba primero el contexto, salvo al resolver el
resultado durable del consumidor: un recibo válido prevalece sobre una
cancelación inmediatamente posterior al `COMMIT`. Solo se exponen
`context.Canceled` o `context.DeadlineExceeded`; una causa privada nunca es
alcanzable con `errors.Is`, `errors.As` o `Unwrap`.

## Evidencia ejecutable

Las pruebas incluyen:

- HMAC real sobre la preimagen completa;
- mutación campo a campo, huella de petición, sello y metadatos;
- raíz, rol, audiencia, organización, credencial y desafío manipulados;
- autoridad, backend o clave reutilizados;
- publicación adulterada, suplantada o caducada;
- vía futura añadida exclusivamente por catálogo;
- replay exacto concurrente, conflicto y expiración exclusiva;
- replay exacto con confirmación nueva en `t+3` y recibo original de `t+2`;
- nuevo recibo de fuente para la misma `PeticionRef` sin segundo efecto;
- semántica o resultado distintos para la misma `PeticionRef` en conflicto;
- reutilización incompatible de una clave probatoria en conflicto;
- concurrencia con recibos de fuente distintos y un único efecto;
- timeout y cancelación competitivos posteriores al `COMMIT`;
- respuesta perdida tras `COMMIT` y recuperación por `PeticionRef`;
- rechazo de un primer recibo anterior a la emisión de la fuente;
- timeout total, cancelación prioritaria y nulos tipados;
- consumidor que ignora contexto, reloj vencido y cancelación competitiva;
- retroceso del reloj y recibo fechado en el futuro;
- retroceso de `t+5` a `t+2` entre catálogo y fuente, con consumo cero;
- presentación que avanza de `t0` a `t0+2 s`, credencial hasta `t0+6 s` y
  horizonte de cinco segundos, rechazada antes de consultar RC o coste;
- autenticador criptográfico concreto fuera de `ports`;
- credencial SAE intentando contestar la definición de Bolsa;
- confirmación sin clave privada TCB y firma Ed25519 alterada;
- límites 100 años, 2^53−1 y cinco segundos;
- minimización y formatos redactados sin PII.

Puertas previstas:

```text
go test ./internal/modules/contrataciontemporal/ports \
  ./internal/modules/contrataciontemporal/application \
  ./internal/modules/contrataciontemporal/adapters/seguridad -count=1
go test -race ./internal/modules/contrataciontemporal/ports \
  ./internal/modules/contrataciontemporal/application \
  ./internal/modules/contrataciontemporal/adapters/seguridad -count=1
go vet ./internal/modules/contrataciontemporal/ports \
  ./internal/modules/contrataciontemporal/application \
  ./internal/modules/contrataciontemporal/adapters/seguridad
go test ./internal/modules/contrataciontemporal/application \
  -run '^TestServicioConsultaCoberturaReintentoExactoConRelojAvanzado$' \
  -count=20
go test ./...
go vet ./...
scripts/comprobar_tamano_ficheros.sh
git diff --check
scripts/verificar_secretos_git.sh 'BASE..HEAD'
```

## Decisiones pendientes de Sistemas

No se han localizado conectores productivos autorizados para Bolsa, SAE u
otras fuentes de cobertura en este repositorio. Por tanto, O4-02 no lee tablas
ajenas ni simula un adaptador de producción.

Antes de habilitar datos reales, Sistemas, RRHH y seguridad deben acordar:

1. API, vista o servicio institucional de cada procedencia;
2. identidad de servicio y `BackendRef` canónico;
3. provisión, rotación y revocación de raíces y credenciales;
4. custodia y rotación del secreto de respuesta;
5. despliegue separado del verificador criptográfico;
6. publicación append-only del catálogo y autoridad publicadora;
7. consumidor durable y transacción con expediente, auditoría y outbox;
8. timeout, reintentos, circuit breaker y observabilidad sin PII;
9. pruebas E2E de caída, expiración, replay, restauración y revocación.

Los adaptadores reales deberán ser finos: traducir el puerto acordado y nunca
consultar directamente tablas propiedad de Bolsa, SAE u otro módulo.

La implementación contribuye a minimización, integridad, exactitud y
trazabilidad, pero no certifica por sí sola RGPD, LOPDGDD, ENS, ENI ni la
legalidad del procedimiento.
