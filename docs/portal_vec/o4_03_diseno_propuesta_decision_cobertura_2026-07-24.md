# O4-03 — diseño de propuesta y decisión de vía de cobertura

Fecha: 24 de julio de 2026.

Estado: implementación en curso. O4-03 continúa abierta.

## Objetivo

Construir una propuesta automática reproducible a partir del catálogo
gobernado y de comprobaciones atestadas, y permitir que RRHH adopte o
rectifique una decisión humana motivada. La propuesta informa; nunca produce
por sí sola un efecto jurídico ni modifica el expediente.

O4-04 persistirá después la decisión, su autorización, auditoría, outbox y
recibo en una sola transacción. O4-05 conectará API y pantalla.

## Brechas confirmadas

O4-01 identifica vías, orden, obligatoriedad y procedencia, pero su canon V1 no
declara qué resultados habilitan cada vía ni cómo tratar una ausencia. Ese
canon ya publicado no se modifica.

O4-02 autentica y consume cada observación, pero su método público devuelve
solo `ComprobacionCobertura`. Para decidir con prueba suficiente se debe
conservar también petición, catálogo, atestación, verificación, orden de
consumo y recibo durable.

`DecisionViaCobertura` todavía no liga catálogo, propuesta, conjunto de
evidencias, política ni actuación. Tampoco existe rectificación segregada.

## Microcortes

| Corte | Resultado verificable |
| --- | --- |
| O4-03A | Política versionada y propuesta canónica, sin alterar O4-01 V1. |
| O4-03B | Evidencia completa y redactada de O4-02, con huella de conjunto. |
| O4-03C | Caso de uso de proponer, decidir y rectificar con autorización VEC V3, idempotencia y CAS contractual. |
| O4-03D | Pruebas adversariales, carrera y revisión independiente. |

## Avance verificable

| Pieza | Estado | Evidencia |
| --- | --- | --- |
| Política y propuesta canónica | Cerrada | `e066497` |
| Evidencia pendiente por vía | Cerrada | `aab240f` |
| Ligadura y preparación global C1 sin consumo | Cerrada | `2c0603e`, `a24f9c8`, `8b7779a` |
| Decisión y rectificación C2 | Cerrada | `589f85f` |
| Motivo funcional publicado y resolución server-side acotada | Cerrada | `b9d595d`, `cd3b544` |
| Identidad semántica de presentación | Cerrada | `2c18fc7` |
| Preparación VEC positiva sin registro previo | Cerrada | `14e332e` |
| Candidata VEC compuesta positiva o negativa | Cerrada | `194afd1` |
| Reserva idempotente cercada | Cerrada | `5b1c838` |
| Referencia y correlación VEC reservadas | Cerrada | `2778cb4`, `8294fad` |
| Identidad durable del análisis O3 | Cerrada | `f36c41f` |
| Proyección PostgreSQL del motivo | Cerrada | `d45c980` |
| Lector PostgreSQL del análisis O3 | Cerrada | `89d8112`; lectura autoritativa, acotada y verificada contra el canon O3 |
| Gobierno server-side de la operación | Cerrada | `c9dbb43`; catálogo, política, reloj, unidad, transición y motivo VEC no proceden del canal |
| Identidad VEC y análisis congelados en reserva | Cerrada | `17d4ea9`; recomposición ligada a referencia y huella exactas |
| Orden opaca final O4-04 | Cerrada | `5be6c14`; unión nominal: solo la concesión transporta efectos C1/C2 |
| Orquestador C3 | En implementación | Presentación pura autorizada cerrada en `3837f2b`; decidir y rectificar siguen pendientes |
| Transacción PostgreSQL única O4-04 | Pendiente | Confirmación y reconciliación nominales preparadas en `9238d53`; faltan sesión TCB `SERIALIZABLE`, revalidación y SQL |

«Cerrada» en esta tabla significa código probado, revisado y publicado en la
rama de trabajo. No equivale a capacidad E2E ni a autorización de producción.

No se crearán ficheros nuevos en `*/ports` mientras siga vigente DEC-092. Los
contratos nuevos nacerán en un subpaquete funcional de Contratación Temporal;
solo se ampliarán de forma compatible los contratos O4-02 ya existentes cuando
sea imprescindible conservar su evidencia.

## Política gobernada

La política de decisión debe quedar ligada a:

- organización y finalidad;
- identidad exacta del catálogo: referencia, versión y huella;
- referencia, versión, huella y vigencia de la propia política;
- por cada vía, comprobaciones previstas y resultados habilitantes;
- tratamiento explícito de la ausencia;
- prioridad para producir una recomendación determinista;
- motivo gobernado cuando la decisión humana se aparte de la recomendación.

La vía y sus condiciones son datos administrables. El núcleo solo conoce
estados técnicos cerrados y no contiene listas compiladas de Bolsa, SAE,
convocatoria u otras opciones futuras.

## Estados técnicos de propuesta

| Estado | Significado | Puede decidirse |
| --- | --- | --- |
| `viable` | Todas las condiciones exigidas se acreditan y al menos una vía es viable. | Sí, sobre una vía viable. |
| `incompleta` | Falta una comprobación cuyo tratamiento no permite ausencia. | No. |
| `conflictiva` | Existen observaciones incompatibles para la misma condición o procedencia. | No. |
| `sin_via` | El conjunto es completo y coherente, pero ninguna vía resulta habilitada. | No por el flujo ordinario. |

`no_aplica` es un resultado acreditado; no equivale a falta de datos.
Indisponibilidad, timeout y error privado tampoco equivalen a ausencia,
resultado negativo ni éxito.

## Propuesta

La propuesta canónica contiene, como mínimo:

- organización, expediente y versión analizada;
- referencia y huella del análisis durable de origen;
- identidades exactas de catálogo y política;
- huella global ordenada de los conjuntos de evidencias O4-02 de todas las
  vías, ligada al análisis durable y sin PII;
- evaluación de cada vía;
- vía recomendada cuando el estado es `viable`;
- referencia y huella reproducible de la propuesta;
- instante de generación y vigencia.

O4-02 liga cada petición, respuesta y orden a una vía concreta. Por tanto, una
comprobación compartida se consulta una vez por cada identidad O4-02 ligada a
vía; no se reutiliza una orden entre vías. Dentro de una misma vía, el replay
exacto se normaliza y una renovación con el mismo resultado funcional conserva
un único representante determinista. Claves con resultados funcionales
distintos producen conflicto.

La preparación global exige exactamente un conjunto por cada vía publicada,
rechaza vías omitidas, añadidas o duplicadas y liga:

- organización, expediente y versión;
- análisis durable y su huella;
- catálogo, política, finalidad, categoría y periodo;
- cada par ordenado `vía + huella del conjunto`;
- los resultados minimizados exactos utilizados por la propuesta.

Cambiar un byte de catálogo, política, análisis, resultado o evidencia cambia
la huella o invalida la preparación y la propuesta.

## Decisión humana

El cliente solo aporta la intención, referencias de autenticación, expediente,
versión, clave idempotente, huella semántica de la propuesta mostrada, vía
elegida y clave funcional del motivo cuando proceda. Actor, perfil, roles,
unidad, catálogo, política, resultados, evidencias, recibos y autoridad se
resuelven en el servidor.

La preparación C1 y sus órdenes son capacidades opacas de muy corta duración:
no se serializan, no viajan al cliente y no se almacenan para que una persona
decida más tarde. La consulta de presentación entrega una huella semántica que
excluye referencias transitorias e instantes, pero incluye análisis, catálogo,
política, resultados, evaluaciones y recomendación. Al confirmar, el servidor
recompone evidencia fresca y una propuesta exacta nueva. Si su huella
semántica no coincide con la mostrada, devuelve conflicto y exige
reconfirmación. La decisión final se liga siempre a esa propuesta exacta y a
sus órdenes frescas, no a la huella semántica aislada.

La decisión ordinaria exige:

- propuesta vigente y estado `viable`;
- vía elegida presente y viable;
- propuesta automática sin autoridad decisoria y actor humano resuelto por
  VEC; una futura propuesta humana requerirá una política de procedencia y
  segregación nueva, no implícita;
- motivación obligatoria y motivo gobernado si se aparta de la recomendación;
- autorización VEC V3 exacta para
  `contratacion_temporal.cobertura.decidir`;
- CAS sobre la versión esperada;
- una orden opaca que O4-04 pueda confirmar sin reinterpretar la política.

Existen dos motivos distintos y no intercambiables. El motivo funcional C2
explica una desviación o rectificación y liga entrada publicada con su clave
i18n. El motivo de autorización VEC justifica ante el PDP la operación exacta
y es obligatorio incluso cuando la decisión coincide con la recomendación.
Ambos se resuelven por autoridades del servidor; compartir una entrada solo
sería válido si una configuración publicada demostrase expresamente esa
equivalencia.

Antes de evaluar VEC, O4-03C consulta el replay y obtiene de O4-04 una reserva
durable exclusiva para la pareja ámbito idempotente + huella semántica. La
reserva bloquea un único propietario mediante un secreto efímero, una
caducidad y una revisión de cercado monotónica. También congela el agregado y
versión anteriores y todas las referencias candidatas, incluidas correlación
y decisión VEC. Una colisión semántica se rechaza y un replay ya confirmado
devuelve el recibo sin volver a evaluar ni autorizar. Una propiedad caducada
solo puede recuperarse con una revisión de cercado superior; el propietario
anterior ya no puede confirmar.

Solo el dueño de la reserva recompone la evidencia fresca, recupera la
correlación reservada, fija el instante de efecto y pide a VEC una evaluación
candidata sin registro
durable positivo. La orden opaca conserva solicitud, decisión candidata,
orden de registro VEC, propuesta exacta, evidencias y transición C2.

La referencia de decisión VEC procede de la reserva y se entrega mediante un
generador inmutable y exclusivo de esa operación. No se acepta desde el
cliente, no se deriva de la correlación y no existe un generador mutable
compartido entre solicitudes.

La preparación para registro compuesto no escribe ni concesiones ni
denegaciones: devuelve una candidata opaca que contiene exactamente una orden
positiva o negativa. Congelar solo la referencia no permitiría registrar antes
una denegación de forma segura, porque la huella VEC también compromete
solicitud, vínculo, instantánea de permisos y ventana temporal. Un reinicio
podría recomponer otra huella con la misma referencia y provocar una colisión.
Por ello ambas ramas se registran dentro de O4-04.

O4-04 registra la candidata VEC, positiva o negativa, dentro de la misma
transacción serializable que deja la reserva en estado terminal. Solo la rama
positiva consume todas las evidencias, aplica el CAS y persiste decisión,
historial y outbox; ambas conservan la auditoría y el resultado terminal
correspondiente. No llama al servicio `ExigirSolicitudLigadaV3`, no abre una
transacción VEC anidada y no devuelve una confirmación durable antes del
`COMMIT`. De este modo no existe el intervalo «decisión VEC confirmada,
reserva aún no actualizada» que dejaría trazas huérfanas o duplicables tras un
reinicio.

Un resultado ambiguo al confirmar se resuelve consultando primero la
reserva/recibo en el primario. Si ya es terminal se reproduce el mismo
resultado; si sigue reservada puede recomponerse porque no se aplicó ningún
efecto; si el primario no permite comprobarlo se devuelve indisponibilidad. No
se reevalúa a ciegas.

El motivo funcional ya tiene una proyección PostgreSQL publicada y
autoritaria. O4-04 bloqueará y comprobará dentro de su transacción el
catálogo, versión, huella, estado publicado, instante de publicación, entrada
actual y clave i18n exacta. La fuente de fichero o memoria puede servir para
desarrollo y lectura, pero no sustituye esa revalidación durable.

Los canales no confiables aportan únicamente la clave funcional y deben pasar
por `ResolverClave`, que selecciona en el servidor la versión publicada y
aplica límites antes de clonar tanto el histórico como la lectura exacta.
`Resolver` queda reservado a la revalidación interna de una referencia ya
derivada; no se expondrá directamente por HTTP, CLI, MCP ni escritorio.

## Rectificación

Rectificar crea una nueva versión y conserva la anterior. Exige propuesta nueva
o todavía vigente, motivo publicado, referencia a la decisión sustituida y un
actor distinto del decisor anterior. Se prohíbe cuando ya existe asignación o
un efecto posterior, salvo un procedimiento explícito de retroacción que queda
fuera de O4-03.

La autorización usa una acción distinta:
`contratacion_temporal.cobertura.rectificar`. El replay exacto devuelve el
mismo recibo; la misma clave con otra semántica es conflicto.

## Seguridad y minimización

- Denegación predeterminada y privilegio mínimo por operación.
- Ningún nombre, DNI, contacto, candidatura o posición viaja por estos
  contratos.
- Las referencias a Bolsa o procedimiento proceden de evidencia gobernada, no
  del cliente.
- Texto de proveedor y causas privadas no se guardan ni se devuelven.
- Los errores públicos distinguen solicitud inválida, denegación, conflicto,
  resultado no confiable e indisponibilidad sin filtrar detalles internos.
- Contexto, catálogo, política, preparación global, evidencias y evaluación
  candidata VEC se revalidan justo antes de fabricar la orden; O4-04 repetirá
  la revalidación, registrará el resultado VEC y, si es positivo, consumirá
  todas las capacidades dentro del mismo `COMMIT`.
- La atomicidad exige que autorización VEC, Contratación Temporal y la
  proyección de gobierno que se bloquea en O4-04 estén en el mismo clúster
  PostgreSQL. Una distribución futura entre bases requerirá otro protocolo
  explícito de consistencia; no se fingirá atomicidad con dos `COMMIT`.
- El motivo conservado por el dominio compromete catálogo, versión, huella,
  entrada y clave i18n, pero no acredita autoridad externa por sí solo. O4-03C
  debe reconsultar la publicación exacta y comprobar que la entrada está
  vigente y declara esa clave i18n; O4-04 debe repetirlo antes de cualquier
  efecto durable. Una referencia rellenada por el llamador nunca basta.
- Web, escritorio, CLI y MCP invocarán el mismo caso de uso, sin cookies como
  autoridad.

## Puerta de cierre O4-03

Se exige probar propuesta determinista, ausencia, contradicción, catálogo y
política adulterados, decisión alternativa motivada, rectificación segregada,
replay, colisión semántica, CAS, cancelación, concurrencia, redacción y copia
defensiva. Deben quedar verdes pruebas unitarias, carrera, `go vet`, pruebas
globales, límites, secretos y revisión independiente.

El cierre técnico no autoriza producción. Permanecen O4-04, O4-05 y las
conformidades formales de RRHH, Jurídico, DPD, ENS, ENI y EIPD.
