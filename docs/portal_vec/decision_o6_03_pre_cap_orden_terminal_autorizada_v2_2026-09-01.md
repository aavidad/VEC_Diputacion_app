# CT-LITE-O6-03-PRE-CAP-DOC — orden terminal autorizada V2

Fecha: 1 de septiembre de 2026.

## Estado, base y alcance

Este documento parte de la base exacta y limpia
`f1c7f8f957cf1bfa478414f0cf24702cd49768f2`. Es un candidato documental:
carece de autoridad para programar, integrar o contabilizar la capacidad hasta
que un revisor independiente emita `GO` sobre su hash exacto y Dirección lo
integre.

La capability decidida es `OrdenTerminalLlamamientoAutorizadaV2`, propiedad
exclusiva de `internal/modules/bolsa/application`. Su única responsabilidad es
transportar de forma efímera y opaca la ligadura ya autorizada para una de las
tres transiciones terminales del `LlamamientoAbierto` integrado en B2.

Invariante del corte: cero V1, persona, sujeto, candidato, selección, sucesor,
planificador, persistencia, auditoría, outbox, comunicación, efecto durable,
composición o producción. Este documento no programa PRE-CAP ni modifica B2.

Write-set de este candidato:

```text
docs/portal_vec/decision_o6_03_pre_cap_orden_terminal_autorizada_v2_2026-09-01.md
```

## Autoridad y emisión exclusiva

La orden solo puede nacer de una
`EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2` obtenida mediante
`ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2`, cuya composición
real es `FachadaUsoDecisionAutorizacionSolicitudLigadaV2`. Bolsa no llamará
directamente a `NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2`, no
aceptará una decisión copiable y no construirá otra fachada o PEP.

V1 queda prohibida sin compatibilidad implícita: no se admiten
`SolicitudAutorizacion`, `Autorizador`, `FachadaUsoDecisionAutorizacion`,
`ExigidorEvidenciaUsoDecisionAutorizacion` ni
`EvidenciaUsoDecisionAutorizacion`; tampoco habrá conversión, proyección,
fallback o constructor desde bytes entre V1 y V2.

Identidad, perfil, método, garantía, sesión, actor y vínculo de autenticación
proceden únicamente de la frontera confiable existente y se entregan a la
fachada V2 por el caso de uso. No son campos de un DTO terminal ni de
`OrdenTerminalLlamamientoAutorizadaV2`, no se aceptan por HTTP o transporte y
no se reconstruyen desde JSON, cabeceras, cookies o parámetros. La orden
retiene únicamente la evidencia V2 como capacidad opaca; no duplica ni expone
la identidad o el contexto que aquella acredita.

## Operaciones nominales cerradas

Cada terminal tiene una política distinta. No existe un selector libre que
permita combinar acción, finalidad o perfil:

| terminal | acción | finalidad | perfil |
| --- | --- | --- | --- |
| aceptacion | bolsa.llamamiento.aceptar | gestion_aceptacion_llamamiento | PerfilProteccionUsoAutorizacionOrdinario |
| renuncia | bolsa.llamamiento.renunciar | gestion_renuncia_llamamiento | PerfilProteccionUsoAutorizacionOrdinario |
| expiracion_gobernada | bolsa.llamamiento.expirar | gestion_expiracion_gobernada_llamamiento | PerfilProteccionUsoAutorizacionInternoAlto |

Las tres políticas fijan literalmente:

- módulo `bolsa`;
- tipo de recurso `llamamiento_abierto`;
- `CamposPermitidos` vacío;
- `Obligaciones` vacío.

Una lista vacía de campos significa operación atómica sin restricción por
campo, nunca «cualquier campo». La decisión devuelta debe contener exactamente
ambas colecciones vacías. Aceptación y renuncia solo admiten el perfil
ordinario sobre la superficie personal externa ya gobernada; expiración solo
admite el perfil interno alto, garantía efectiva alta, decisión con garantía
mínima alta y superficie corporativa interna o de administración
privilegiada. Un perfil no puede sustituir a otro aunque el resto coincida.

## Recurso exacto y canon de huella

La aplicación construye un único `domain.RecursoAutorizable` desde el
`LlamamientoAbierto` de B2 y la intención terminal ya validada:

```text
Referencia = LlamamientoRef
ModuloID   = bolsa
Tipo       = llamamiento_abierto

Ambitos, exactamente 3 claves:
  bolsa_ref     = BolsaRef
  necesidad_ref = NecesidadRef
  propuesta_ref = PropuestaRef

Atributos, exactamente 3 claves:
  version_esperada = decimal canónico de VersionEsperada
  estado_terminal  = aceptacion | renuncia | expiracion_gobernada
  operacion_ref    = OperacionRef
```

No se admiten claves ausentes, adicionales, repetidas, vacías, corregidas,
normalizadas ni con comodín. `version_esperada` se obtiene de
`strconv.FormatUint(VersionEsperada, 10)`: solo dígitos ASCII, sin signo,
espacios ni ceros iniciales; cero y los valores fuera del entero seguro común
de JSON son inválidos. Las seis referencias y valores conservan exactamente
la forma opaca endurecida del dominio de Bolsa.

La huella de contexto debe ser exactamente el resultado de
`RecursoAutorizable.HuellaContextoAutorizacionSHA256()`: hexadecimal minúsculo
del SHA-256 de los bytes UTF-8 producidos por `encoding/json` para este
documento, sin espacios y con claves de mapa en orden lexicográfico:

```json
{"ambitos":{"bolsa_ref":"<BolsaRef>","necesidad_ref":"<NecesidadRef>","propuesta_ref":"<PropuestaRef>"},"atributos":{"estado_terminal":"<estado_terminal>","operacion_ref":"<OperacionRef>","version_esperada":"<VersionEsperada-decimal>"}}
```

Los marcadores entre ángulos representan los valores exactos escapados por
`encoding/json`; no forman parte de la preimagen real. No hay normalización
Unicode implícita. `Referencia`, `ModuloID` y `Tipo` no entran en esa preimagen
porque la autoridad VEC los compromete y coteja por separado. La evidencia
debe contener esa misma huella en `ContextoRecursoHuellaSHA256`.

## Ligadura V2 que debe coincidir

Antes de emitir la orden, PRE-CAP exige igualdad exacta entre la intención, la
política nominal, el recurso construido y la decisión contenida en la
evidencia para:

- acción de la fila terminal;
- finalidad de la misma fila;
- referencia, módulo, tipo y huella completa del recurso;
- correlación V2 opaca y válida comprometida en la solicitud y la decisión;
- referencia y huella del motivo catalogado exacto;
- perfil y vínculo obtenidos de la frontera existente;
- garantía exigida por el perfil;
- vigencia de vínculo, sesión, decisión y evidencia;
- `CamposPermitidos` y `Obligaciones` exactamente vacíos;
- ausencia total de comodines en toda dimensión positiva.

Además se conservan los compromisos V2 de solicitud y motivo con sus esquemas
exactos; una huella estructural no sustituye la procedencia del PDP ni la
resolución positiva del catálogo.

`OperacionRef` es la identidad semántica del comando terminal de B2.
Correlación V2 identifica la evaluación/autorización. Son valores distintos:
deben pertenecer a espacios opacos separados, no pueden ser iguales, derivarse
uno de otro ni sustituirse en replay, auditoría futura o idempotencia.

## Tiempo, motivo y replay

La vigencia real de la decisión nunca supera cinco minutos:
`ValidaHasta - EmitidaEn <= domain.VigenciaMaximaDecisionAutorizacion`. La
ventana es semiabierta: `EmitidaEn <= ahora < ValidaHasta`, limitada además
por la sesión y el vínculo.

En el instante confiable de emisión de la orden se invocan obligatoriamente,
y en este orden defensivo:

1. `evidencia.ValidarEn(ahora)`;
2. `evidencia.ValidarMotivo(referenciaMotivo)`;
3. cotejo exacto de política, recurso, correlación, perfil y colecciones.

La referencia de motivo procede del catálogo gobernado ya resuelto; PRE-CAP
no admite texto libre. La existencia y vigencia reales del motivo continúan
siendo responsabilidad de la frontera catalogal y, en el futuro, de la
transacción durable.

Un replay solo puede devolver la misma orden lógica si repite exactamente la
misma ligadura: evidencia/decisión V2, acción, finalidad, perfil, referencia y
huella del recurso, correlación, motivo catalogado, `VersionEsperada`,
`estado_terminal` y `OperacionRef`. Cualquier divergencia o caducidad falla
cerrada y devuelve capacidad cero; nunca se corrige una entrada ni se genera
otra operación.

## Opacidad, serialización y errores

`OrdenTerminalLlamamientoAutorizadaV2` es efímera, no serializable, no
persistible y no reconstruible. Tanto la orden como cualquier proyección
interna deliberada bloquean codificación y decodificación genéricas por
JSON, XML, Text, Binary, Gob, CBOR y YAML.

`String`, `GoString`, cualquier verbo o bandera de `Format` y `LogValue`
devuelven exclusivamente:

```text
[ORDEN-TERMINAL-LLAMAMIENTO-AUTORIZADA-V2-OPACA]
```

PRE-CAP declarará solo estos centinelas públicos, con textos constantes sin
referencias, hashes, motivos, identidad ni causas privadas:

- `ErrOrdenTerminalLlamamientoInvalida` para valor cero, construcción,
  lectura, vigencia, motivo, ligadura, replay o estructura inválidos;
- `ErrSerializacionOrdenTerminalLlamamientoProhibida` para cualquier intento
  de codec genérico.

Precedencia obligatoria:

1. una operación de codec devuelve siempre
   `ErrSerializacionOrdenTerminalLlamamientoProhibida`, incluso sobre un valor
   cero o corrupto, y no produce bytes ni estado parcial;
2. fuera de codecs, toda causa semántica, temporal o estructural se reduce a
   `ErrOrdenTerminalLlamamientoInvalida` y devuelve la orden o datos cero;
3. el formateo y el log redactado prevalecen sobre cualquier inspección y no
   revelan si la orden era válida.

Los errores no envuelven entradas o errores de proveedor que puedan actuar
como oráculo. El diagnóstico privado futuro se emitirá, si procede, por la
autoridad de observabilidad y nunca mediante la orden.

## Fronteras, DAG y write-sets posteriores

El grafo obligatorio es lineal:

```text
CT-LITE-O6-03-PRE-CAP-DOC
  -> CT-LITE-O6-03-PRE-CAP
    -> CT-LITE-O6-03-B3
```

Frontera PRE-CAP: implementará únicamente políticas nominales, obtención de
evidencia por el exigidor V2, cotejo y emisión de la orden opaca. No invocará
`LlamamientoAbierto.TransicionarATerminal` ni consumirá una decisión durable.
Su write-set reservado, nuevo y exclusivo, será:

```text
internal/modules/bolsa/application/orden_terminal_llamamiento_autorizada_v2.go
internal/modules/bolsa/application/orden_terminal_llamamiento_autorizada_v2_test.go
```

Frontera B3: consumirá una orden PRE-CAP válida y el valor puro B2 para derivar
la transición terminal en memoria. No volverá a decidir la política, fabricar
evidencia, modificar PRE-CAP ni editar los ficheros de B2. Su write-set
reservado, nuevo y disjunto, será:

```text
internal/modules/bolsa/application/transicion_terminal_llamamiento_autorizada_v2.go
internal/modules/bolsa/application/transicion_terminal_llamamiento_autorizada_v2_test.go
```

Cualquier evidencia documental o acta de revisión posterior pertenecerá a un
corte de Dirección separado; no amplía estos write-sets productivos.

PRE-CAP y B3 no consumen la autorización de manera durable y no acreditan
`COMMIT`. Un corte transaccional posterior y separado deberá releer y
revalidar decisión V2, motivo, vigencia, versión y ligadura, consumir la
autorización y escribir conjuntamente estado, historia/auditoría y outbox en
una única transacción. Solo su recibo durable podrá acreditar el efecto.

## Limitaciones y siguiente corte

Este candidato no aporta API, HTTP, PostgreSQL, persistencia, auditoría,
outbox, inbox, comunicación, entrega, aceptación efectiva, renuncia efectiva,
expiración efectiva, siguiente candidato, persona, sucesor, planificador,
composición raíz, web, E2E, despliegue o producción. No cierra O6-03, no cambia
métricas y no declara una vertical arrancable.

El siguiente corte posible, únicamente después de `GO` independiente e
integración de este documento, es `CT-LITE-O6-03-PRE-CAP` sobre su write-set
reservado. B3 permanece bloqueada hasta integrar PRE-CAP revisado.
