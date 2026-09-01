# CT-LITE-O6-03-B4-R1-DOC — confirmación terminal durable V2

Fecha: 1 de septiembre de 2026.

## Estado, base y alcance

Esta corrección R1 parte de la base exacta y limpia
`0896322d67aac4cef1b559bc81f35fabd81230ad`, cuyo único padre es el producto
local `640610a4f806b0848682bbe844ff9d672c2777a6`. Es un candidato documental: no
implementa producto, no cierra `O6-03`, no cambia métricas y no adquiere
autoridad hasta que dos revisores independientes emitan `GO` sobre su hash
exacto y Dirección lo integre.

La capability decidida es
`ConfirmacionTerminalLlamamientoDuraderaV2`, propiedad de la aplicación de
Bolsa. Su responsabilidad exclusiva será confirmar de forma durable una
transición terminal de `LlamamientoAbierto` previamente dado de alta y
persistido por Bolsa.

Write-set único de este corte:

```text
docs/portal_vec/decision_o6_03_b4_confirmacion_terminal_durable_v2_2026-09-01.md
```

El fichero conserva modo `0644` y debe permanecer por debajo de 800 líneas. R1
fija una única autoridad y topología lógica: una unidad V2 aislada y propia de
Bolsa, con objetos, roles, funciones y adaptador V2 separados, dentro del mismo
dominio transaccional PostgreSQL que la fachada VEC V2 de revalidación y
consumo. B4-A fijará la ubicación, los nombres físicos y los números de
migración concretos. V1 queda histórica e inmutable: no se amplía, reutiliza,
convierte ni participa en la ejecución de B4.

## Preflight local acreditado antes de editar

- rama exclusiva `trabajo/ct-o6-b4-doc-20260901`, `HEAD` exacto y árbol
  limpio en `0896322d67aac4cef1b559bc81f35fabd81230ad`;
- Go `go1.26.5 linux/amd64`;
- rama de producto local y referencia local de
  `origin/integracion/ct-producto-ligero-20260821` en
  `640610a4f806b0848682bbe844ff9d672c2777a6`, sin ejecutar `fetch`, `pull` ni
  otra operación de red;
- el candidato original `0896322d67aac4cef1b559bc81f35fabd81230ad` tiene
  como padre único ese producto y añade únicamente este documento;
- B2 integrado mediante `adc8ec6c2f03fc9beee20eacd3e94e4b56ca441b`,
  PRE-CAP-DOC mediante `58800913b32e22b0f77eb8d62900d95c452e98fa`,
  PRE-CAP revisado mediante `2171c932c5f0c9b9080aa1f9dfda0e26581805f3`
  y B3 mediante `640610a4f806b0848682bbe844ff9d672c2777a6`;
- los cuatro merges son ancestros del producto y el `merge-tree` inicial entre
  producto y candidato original no presenta conflictos; produjo el árbol
  `c3782da7a957272e04e1776c96eccd08263f0593`;
- el documento de partida tenía 365 líneas y modo `0644`; las autoridades
  documentales leídas también tenían modo `0644` y menos de 800 líneas.

No se ejecutaron red, despliegue, credenciales, PostgreSQL, Docker, E2E,
producción ni puertas globales o pesadas.

## Autoridades e inventario que no se duplican

### Piezas B2, PRE-CAP y B3 integradas

`internal/modules/bolsa/domain/llamamiento_abierto.go` define el valor puro e
inmutable B2. Contiene exactamente las referencias de llamamiento, bolsa,
necesidad y propuesta, una versión positiva, el estado técnico `abierto` y
una única terminal. Su CAS y replay exacto son reglas de dominio.

PRE-CAP vive en
`internal/modules/bolsa/application/orden_terminal_llamamiento_autorizada_v2.go`.
Emite `OrdenTerminalLlamamientoAutorizadaV2`, efímera, opaca, no serializable
y ligada a decisión, solicitud, recurso, correlación, motivo, perfil, vínculo,
vigencia y política V2 exactos. No consume la autorización ni acredita un
efecto durable.

B3 vive en
`internal/modules/bolsa/application/transicion_terminal_llamamiento_autorizada_v2.go`.
Reacredita la orden y aplica B2 en memoria. No persiste, no consume la decisión
y no emite recibo. B4 reutilizará esta derivación; queda prohibido copiar sus
reglas de terminal, CAS o replay en SQL.

### Selección existente

Bolsa ya posee dominio, aplicación, puertos y adaptadores de propuesta de
primer llamamiento. El despliegue `bolsa_llamamientos` conserva tablas de
bolsa, necesidad, política, instantánea, evaluación, propuesta, consumo,
auditoría y outbox. Su función `guardar_propuesta_v1` y el adaptador Go V1
son historia inmutable y están deliberadamente cerrados para el nuevo comando
completo; no son una persistencia V2 de `LlamamientoAbierto`, no se modifican
ni se invocan desde B4 y no aportan objetos, roles, funciones o adaptador a V2.

Contratación temporal ya posee `seleccion_llamamiento` en `ports`,
`application` y HTTP, además del adaptador PostgreSQL y la tabla de ejecuciones
de selección. Estas piezas coordinan idempotencia, propuesta y resultado
incierto de selección; no son la autoridad del estado terminal de Bolsa y B4
no las generaliza ni las copia.

### Comunicación y formalización existentes

Contratación temporal ya define:

- `TransaccionComunicacionLlamamiento`, su servicio y su HTTP, incluidos
  registro de entrega, respuesta local e intención outbox de siguiente
  candidato; y
- `TransaccionPropuestaFormalizacion`, su servicio y su HTTP, con snapshots de
  tipo, plantilla, anexos, política y plan de firma.

No existe implementación productiva de esas dos transacciones ni SQL que las
materialice. Sus contratos tampoco son una tabla terminal de Bolsa. B4 no
selecciona, comunica, calcula plazos, prepara siguiente candidato, formaliza,
renderiza, firma ni crea documentos.

### Ausencia exacta que abre B4

En la base no existe un puerto, adaptador ni tabla de `LlamamientoAbierto`.
Fuera de pruebas, el tipo aparece solo en B2, PRE-CAP y B3. Las tablas
`propuesta` de Bolsa y `ejecucion_seleccion_llamamiento_o6` de Contratación
temporal no representan ese agregado y no pueden tratarse como su fuente de
verdad por coincidencia de referencias.

## Decisión cerrada de autoridad y topología V2

B4 tendrá una sola unidad V2, aislada del despliegue V1 y propiedad exclusiva
de Bolsa. «Aislada» significa separación de autoridad, objetos y privilegios,
no otra base de datos: la unidad V2 deberá instalarse en la misma base primaria
y dominio transaccional PostgreSQL que expone la fachada VEC V2. Estado de
Bolsa y consumo VEC se ejecutarán por una misma conexión y una única
transacción local, con un solo desenlace `COMMIT` o `ROLLBACK`.

La separación mínima que B4-A debe concretar físicamente es:

- objetos V2 de estado, historia, auditoría, outbox y recibos propios de Bolsa,
  sin vistas, triggers, claves foráneas, lecturas ni escrituras sobre V1;
- roles propietarios y de ejecución V2 exclusivos, sin heredar autoridad de
  los roles V1 ni conceder DML directo al adaptador;
- funciones V2 de alta, confirmación, replay y recuperación separadas de las
  funciones V1, con fachadas nominales y privilegio mínimo; y
- un adaptador PostgreSQL V2 de Bolsa separado del adaptador Go V1, capaz de
  invocar las funciones V2 de Bolsa y la fachada VEC V2 dentro de la misma
  transacción, sin leer ni escribir tablas de VEC.

La fachada VEC V2 sigue siendo la única autoridad para releer, revalidar y
consumir la autorización. La colocalización no transfiere propiedad de tablas:
Bolsa llama a esa fachada nominal y VEC no adquiere autoridad sobre el estado
de Bolsa. B4-A fijará nombres, esquemas, migraciones, roles y rutas de fichero;
no volverá a decidir entre aislamiento y convivencia ni podrá elegir V1.

Si no puede existir esta unidad V2 colocalizada y la fachada VEC V2 no puede
participar atómicamente en su transacción local, B4 completo es `NO-GO`. No hay
topología alternativa mediante otra base, transacción distribuida, consumo
previo, marca intermedia, compensación ni consistencia eventual.

## Decisión: B4 se divide obligatoriamente

La transición terminal no puede inventar el estado abierto que pretende
cerrar. B4 se divide en dos capacidades durables secuenciales:

1. `CT-LITE-O6-03-B4-A`: alta durable de `LlamamientoAbierto` en Bolsa;
2. `CT-LITE-O6-03-B4-B`: confirmación terminal durable V2 sobre un alta B4-A
   ya confirmada.

B4-B no hará un `INSERT` oportunista cuando falte la fila, no reconstruirá el
alta desde HTTP, Contratación temporal, PRE-CAP, una propuesta V1 o una
respuesta nominal, y no aceptará una fila creada por otra autoridad.

## B4-A — fuente y nacimiento durable del abierto

La fuente del alta es el hecho durable, propio de Bolsa, que crea el
llamamiento a partir de una propuesta autoritativa confirmada. El contrato
O6-01 contiene un recibo externo con las referencias necesarias para que
Contratación temporal verifique el intercambio, pero ese recibo de transporte
no puede convertirse en la fuente de verdad interna de Bolsa ni autoriza una
lectura de tablas de Contratación temporal.

B4-A deberá ocurrir en la unidad V2 y autoridad de Bolsa: releerá bajo bloqueo
el hecho de creación y la propuesta durable exacta y confirmará conjuntamente
el abierto y su recibo de alta. Si hoy no existe ese hecho local, B4-A deberá
materializarlo en la misma transacción que emita el recibo de creación; queda
prohibido inferirlo después desde una respuesta enviada a otro módulo.

El nacimiento inmoviliza, sin normalizar ni sustituir:

```text
LlamamientoRef
BolsaRef
NecesidadRef
PropuestaRef
Version = N, positiva y dentro del entero seguro común
Estado = abierto
```

El alta debe tener identidad idempotente propia. Un replay byte y
semánticamente exacto devuelve el mismo recibo; cualquier reutilización con
otra referencia, versión o compromiso es colisión. Su contrato posterior
deberá fijar también historia inicial, auditoría, outbox y recibo atómicos, sin
copiar candidato, selección seudonimizada, posición ni contacto.

## Puerto de salida de aplicación

El contrato de salida se definirá en `internal/modules/bolsa/application`, no
en `ports`: la orden PRE-CAP pertenece a aplicación y moverla a `ports`
crearía una dependencia inversa o un DTO reconstruible. El patrón nominal es:

```text
ConfirmacionTerminalLlamamientoDuraderaV2
  Confirmar(contexto, SolicitudConfirmacionTerminalLlamamientoDuraderaV2)
    -> ReciboConfirmacionTerminalLlamamientoDuraderaV2
  Recuperar(contexto, ConsultaRecuperacionTerminalLlamamientoDuraderaV2)
    -> ReciboConfirmacionTerminalLlamamientoDuraderaV2
```

`SolicitudConfirmacionTerminalLlamamientoDuraderaV2` nace únicamente desde
una `OrdenTerminalLlamamientoAutorizadaV2` válida. Retiene esa orden como
capacidad privada, bloquea JSON, XML, texto, binario, Gob, CBOR y YAML y no
ofrece constructor desde campos o bytes. Solo puede exponer al adaptador los
localizadores opacos mínimos de llamamiento y operación necesarios para
adquirir bloqueos; no expone decisión, actor, perfil, vínculo, motivo ni
evidencia a transporte, logs o formateo.

Después de la relectura bloqueada, la propia solicitud se abre una sola vez
con el agregado B2 reconstruido y el reloj transaccional. Esa apertura coteja
la orden, revalida V2 y devuelve una proyección sellada, no reconstruible, con
el material mínimo para confirmar. El adaptador no acepta los escalares de la
orden como parámetros independientes y no puede corregirlos.

`ConsultaRecuperacionTerminalLlamamientoDuraderaV2` solo contiene la identidad
opaca de operación y la huella sellada del intento. Es un localizador de un
recibo histórico, nunca autoridad para crear, repetir o cambiar un efecto. Su
acceso se somete a la autorización de lectura interna que corresponda en un
corte posterior.

## B4-B — sección crítica y único COMMIT

Para un efecto nuevo, el adaptador abre una transacción `SERIALIZABLE READ
WRITE`, endurece sesión y tiempos y ejecuta este orden observable:

1. en la unidad V2 localiza por las referencias opacas derivadas de la
   solicitud y bloquea la fila B4-A, su historia actual y las autoridades
   necesarias en un orden estable;
2. relee la fuente durable de creación y reconstruye B2 mediante
   `NuevoLlamamientoAbierto`; no confía en una proyección Go anterior;
3. coteja por igualdad exacta `LlamamientoRef`, `BolsaRef`, `NecesidadRef`,
   `PropuestaRef` y `Version = N` entre alta, fuente, reconstrucción y orden;
4. exige estado `abierto`, ausencia de otra terminal y consistencia completa
   de historia, auditoría y recibo de alta;
5. después de adquirir todos los bloqueos obtiene un único `ahora_tx` del
   reloj confiable de la transacción; un instante capturado antes no sirve;
6. en `ahora_tx` relee y revalida la decisión V2, la solicitud y su huella, el
   motivo catalogado exacto y vigente, acción, finalidad, recurso y huella,
   correlación, perfil, garantía, sesión, vínculo actual, campos y obligaciones
   vacíos y ausencia de comodines;
7. aplica B3 a la orden cotejada y al B2 reconstruido. El resultado debe ser la
   terminal solicitada y exactamente `N -> N+1`;
8. consume la autorización V2 y aplica el CAS durable solo si la fila continúa
   abierta, con versión N y las cuatro referencias intactas;
9. escribe estado actual, historia de solo adición, consumo de autorización,
   auditoría, outbox y recibo inmutable; y
10. ejecuta un solo `COMMIT` para esos seis efectos.

La revalidación y el consumo V2 pertenecen a la autoridad VEC existente. El
adaptador V2 de Bolsa invocará su fachada transaccional autorizada en la misma
transacción PostgreSQL local que las funciones V2 de Bolsa; no leerá ni
escribirá tablas ajenas ni creará un ledger alternativo. Esa colocalización y
fachada atómica son una precondición ya decidida, no una opción de B4-A. Si no
pueden existir, B4 es `NO-GO` sin alternativa distribuida, marca previa,
compensación o consumo eventual.

SQL se limita a bloqueos, integridad, CAS y persistencia de las proyecciones
ya derivadas. No decide qué terminal corresponde, no interpreta reglas de
selección y no reproduce `TransicionarATerminal`; B3 sigue siendo la única
regla ejecutable de transición.

## Replay, respuesta perdida y cancelación

La identidad semántica es `OperacionRef`. El intento conserva una huella
canónica que compromete las cuatro referencias, N, terminal, operación,
acción, finalidad, recurso y huella, correlación, decisión y solicitud V2,
motivo y huella, perfil y vínculo. Solo la misma operación con la misma huella
puede devolver el recibo ya confirmado.

- Mismo `OperacionRef` y misma huella completa: se relee y verifica el paquete
  durable y se devuelve exactamente el mismo recibo, sin segunda historia,
  consumo, auditoría ni outbox.
- Mismo `OperacionRef` con cualquier diferencia: colisión cerrada, sin efecto.
- Fila terminal sin el paquete completo y coherente: corrupción o estado
  indeterminado; nunca se fabrica un recibo.
- Resultado de `COMMIT` perdido: no se repite a ciegas. `Recuperar` consulta
  por operación y huella, valida estado, historia, consumo, auditoría, outbox y
  recibo y distingue confirmado de ausencia demostrada e indeterminado.
- La recuperación de un recibo histórico no vuelve a consumir ni exige que la
  decisión siga vigente; acredita el efecto que se revalidó y consumió cuando
  se confirmó. Una decisión caducada nunca puede producir un efecto nuevo.
- Cancelación observada antes de `COMMIT`: rollback total y cero recibo de
  éxito. Si el `COMMIT` puede haber ocurrido antes de observar la cancelación,
  el resultado es indeterminado y se recupera; la cancelación no deshace ni
  oculta un recibo durable confirmado.

El recibo mínimo liga recibo, llamamiento, bolsa, necesidad, propuesta,
operación, estado terminal, versión N, versión N+1, consumo V2, auditoría,
outbox, huella del intento e instante de confirmación. No incluye candidato,
persona, selección, posición, contacto, sucesor, texto de motivo, credencial o
evidencia V2 en claro.

## Normas, datos, finalidad, control y responsabilidad

| Eje | Decisión B4 |
| --- | --- |
| Normas y requisito | RGPD/LOPDGDD: minimización, exactitud e integridad; ENS: autenticidad, trazabilidad, privilegio mínimo, continuidad y recuperación; Leyes 39/2015 y 40/2015: acto, competencia, versión, motivo, recibo e historia; ENI y Ley andaluza 7/2011: evidencia y conservación gobernadas. La matriz de competencia y las aprobaciones siguen pendientes. |
| Datos | Solo referencias opacas, estado técnico, versiones, compromisos V2, motivo catalogado por referencia/huella e identificadores de consumo, auditoría, outbox y recibo. Cero identidad personal, candidato, contacto, selección, posición o documento. |
| Finalidad y autoridad | Confirmar en la unidad V2 de Bolsa una aceptación, renuncia o expiración gobernada ya autorizada. Cada terminal conserva exactamente la acción, finalidad y perfil fijados por PRE-CAP; Bolsa es autoridad del agregado y VEC de decisión, vínculo, motivo común y consumo. |
| Control preventivo | Denegación predeterminada, orden opaca, alta B4-A previa, relectura bajo bloqueo, reloj transaccional posterior, V2 sin fallback, motivo/vínculo vigentes, cotejo de cuatro referencias y N, B3 y CAS N→N+1. |
| Evidencia y prueba | Historia append-only, consumo único, auditoría encadenada, outbox y recibo en un COMMIT; huella de intento, replay exacto, recuperación y matriz PostgreSQL real posterior. Este documento solo aporta decisión e inventario. |
| Conservación y acceso | B4 no inventa plazos ni autoriza expurgo. Registros y recibos quedan bloqueados frente a borrado silencioso hasta política aprobada de su autoridad; una política documental VEC se consume donde corresponda, nunca se copia. Acceso por operación, finalidad y mínimo privilegio. |
| Responsables pendientes | Bolsa: propietario funcional/técnico del agregado y su unidad V2; VEC común: autorización, fachada V2 y consumo; autoridad catalogal: motivo; Sistemas/DBA y Seguridad: validar el aprovisionamiento de la topología fijada, reloj, roles, cifrado, copias y recuperación; RRHH/órgano competente: competencia funcional; DPD, Archivo, Secretaría/Jurídico y responsables ENS: tratamiento, conservación y aprobaciones. Esta tabla no atribuye competencias. |

## Seguridad, privacidad, i18n y accesibilidad

- V1 queda histórica, inmutable y fuera de ejecución: no se amplía, reutiliza,
  convierte, migra, proyecta ni consume mediante `guardar_propuesta_v1`,
  `revalidar_decision_bolsa_llamamientos_v1` o cualquier otra pieza V1.
- Identidad, perfil, vínculo, motivo y decisión no proceden de JSON, HTTP,
  cabeceras, cookies, almacenamiento web, parámetros o tablas de otro módulo.
- Orden, solicitud y proyecciones internas se redactan en `fmt` y `slog`; los
  errores públicos son centinelas constantes y no actúan como oráculo.
- Auditoría y outbox son minimizados y no duplican PII. Cifrado, claves,
  copias, restauración y observabilidad quedan en sus autoridades operativas.
- B4 no contiene interfaz ni texto visible. Un consumidor futuro usará claves
  i18n, formatos localizados y estados accesibles por texto, no solo por color;
  no se abre HTTP, web ni revisión visual en este corte.

## Paradas duras

B4-A o B4-B se detienen sin candidato integrable si ocurre cualquiera:

1. falta doble revisión independiente e integración de este documento;
2. no puede instalarse la unidad V2 propia de Bolsa, con objetos, roles,
   funciones y adaptador separados de V1, en el mismo dominio transaccional
   PostgreSQL que la fachada VEC V2;
3. B4-A no acredita fuente local de Bolsa y nacimiento durable del abierto;
4. la fachada VEC V2 no puede revalidar y consumir dentro del mismo `COMMIT`
   local sin acceso cruzado a tablas;
5. no pueden releerse bajo bloqueo las cuatro referencias, N, motivo, vínculo,
   decisión o historia;
6. el reloj no es confiable, retrocede o se captura antes de los bloqueos;
7. PRE-CAP, B3 o el cotejo exacto fallan, aparece un comodín o se intenta usar
   V1;
8. hay colisión de operación, CAS fallido, fila terminal incompleta, recibo
   incoherente o desenlace de commit no reconciliado;
9. una cancelación pre-COMMIT no revierte el conjunto completo; o
10. se pretende añadir candidato, contacto, sucesor, HTTP, composición, datos
    reales, despliegue o producción.

## DAG y cortes posteriores

```text
CT-LITE-O6-03-B4-R1-DOC (este candidato)
  -> doble revisión independiente del hash exacto
    -> integración por Dirección
      -> CT-LITE-O6-03-B4-A-DOC/contrato
        -> B4-A implementación + pruebas PostgreSQL + revisión + integración
          -> CT-LITE-O6-03-B4-B-DOC/contrato
            -> puerto/sellos de aplicación B4-B
              -> persistencia V2 y adaptador B4-B
                -> replay/recuperación PostgreSQL
                  -> revisión independiente + integración
```

B4-B depende de B4-A integrado. Los subcortes de aplicación, persistencia,
adaptador y recuperación tendrán write-sets disjuntos o se ejecutarán en
secuencia. B4-A-DOC fijará las rutas, nombres y migraciones físicos de la única
unidad V2 ya decidida; no reabrirá autoridad, topología ni participación de V1.

## Pruebas documentales y matriz futura

Puertas ligeras de este candidato:

```text
base R1, rama y estado limpios
Go 1.26.5
inventario rg de tipos, puertos, adaptadores y SQL
genealogía y ancestros B2 -> PRE-CAP -> B3
modo 0644, write-set único y menos de 800 líneas
git diff --check
Gitleaks focal exacto sobre el fichero
merge-tree de solo lectura contra producto 640610a4f806b0848682bbe844ff9d672c2777a6
```

No proceden `gofmt`, `go test`, carrera, `go vet`, PostgreSQL, Docker, E2E,
HTTP, accesibilidad visual ni puertas globales: el corte solo añade Markdown.

Los futuros B4-A/B4-B deberán probar en PostgreSQL efímero real, al menos,
alta y replay, fuente adulterada, ausencia de alta, cuatro referencias y cada
una alterada, N y CAS concurrente, reconstrucción B2, tres terminales mediante
B3, V2 válida/caducada/revocada, motivo y vínculo retirados durante bloqueos,
reloj posterior, consumo único, seis efectos atómicos, cancelación antes de
commit, respuesta perdida, reinicio, recuperación, colisión semántica,
serialización prohibida, ACL/RLS, timeouts, rollback, reversión protegida y
cero PII en tablas, recibos, outbox, auditoría, errores y logs.

## Limitaciones y siguiente corte

Este candidato no crea puerto, adaptador, tabla, migración, recibo, alta ni
efecto terminal. Tampoco aporta comunicación, aceptación efectiva, renuncia
efectiva, expiración efectiva, siguiente candidato, formalización, API,
composición, web, E2E, despliegue o producción.

El siguiente corte es exclusivamente la doble revisión independiente del hash
exacto de este documento. Solo tras ambos `GO` e integración por Dirección se
abre directamente `CT-LITE-O6-03-B4-A-DOC/contrato`, sujeto a la topología V2
aislada y colocalizada fijada aquí.
