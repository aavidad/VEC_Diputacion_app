# Registro canónico de fuentes de autoridad

Fecha de corte: **17 de julio de 2026**.

Estado: arquitectura adoptada y dominio del primer incremento de `NUC-006`
implementado y verificado en la rama. Incluye persistencia canónica V1 del
agregado, pero todavía no sus puertos ni un repositorio. El futuro adaptador en
memoria será únicamente un doble de contrato. La capacidad permanecerá en
**NO-GO productivo** hasta disponer de caso de uso, persistencia PostgreSQL,
verificación criptográfica y de competencia, segregación institucional,
auditoría probatoria con anclaje externo inmutable y validación institucional.

## 1. Problema que resuelve

Los catálogos, flujos, convocatorias y decisiones de baremación ya pueden
conservar referencias como `FuenteRef`, `AprobacionRef` o
`FuentesNormativasRefs`. Esas cadenas permiten relacionar información, pero no
demuestran por sí solas:

- qué documento y representación se examinaron;
- si el contenido coincide con su huella;
- qué publicación oficial, acto y órgano les dan cobertura;
- qué artículos o apartados se materializan;
- desde cuándo está vigente la fuente y desde cuándo produce efectos;
- si fue modificada, suspendida o derogada;
- quién comprobó el acto y con qué evidencia.

El registro de fuentes de autoridad cubre esa brecha. No decide cómo puntuar,
cuántos días corresponden ni quién tiene derecho a una prestación. Conserva
la autoridad exacta que una política funcional deberá citar.

## 2. Alcance del primer incremento

La unidad de gobierno es una `FuenteAutoridadVersionada`: una instantánea
identificada por ID, versión y huella canónica. Como mínimo contiene:

- materia y ámbitos institucionales mediante claves gobernadas;
- documento lógico y representación concreta examinada;
- huella SHA-256 del contenido;
- publicación oficial, acto y órgano competente;
- preceptos concretos, sin interpretar su texto;
- vigencia jurídica y periodo de efectos por separado;
- fecha de conocimiento o incorporación al sistema;
- versión sustituida y relaciones de suspensión o derogación;
- evidencias estructuradas de los actos que cambian su estado.

Quedan expresamente fuera de este corte:

- una DSL genérica o código introducido desde la interfaz;
- la interpretación automática de normas;
- parámetros de baremación, calendarios, permisos, turnos o retribuciones;
- extracción automática desde PDF u OCR con efectos jurídicos;
- búsqueda o almacenamiento de datos personales;
- sustitución retroactiva de referencias históricas ya publicadas.

Las futuras familias de políticas serán tipadas por dominio y citarán una o
varias referencias exactas de este registro.

## 3. Conceptos separados

| Concepto | Significado |
| --- | --- |
| Documento lógico | Identidad administrativa del documento, independiente de sus bytes y representaciones |
| Representación | PDF, PDF/A, XML, escaneo u otra materialización concreta examinada |
| Publicación | Referencia al BOP, BOJA, BOE, sede, tablón o medio oficial aplicable |
| Acto | Acuerdo, resolución, decreto, certificación u otra actuación que publica, suspende o deroga |
| Fuente | Versión canónica de la autoridad registrada por el sistema |
| Precepto | Artículo, apartado, anexo o sección concreta citada, sin interpretación automática |
| Vigencia | Periodo durante el que la fuente forma parte del ordenamiento o marco convencional aplicable |
| Efectos | Periodo al que la actuación administrativa declara que se aplican sus consecuencias |
| Conocimiento | Instante en que el sistema incorporó la información, necesario para reconstrucción bitemporal |
| Política funcional | Regla tipada futura que utiliza la fuente; no forma parte de este agregado |

La fecha de publicación, la entrada en vigor, la fecha de efectos y el instante
de conocimiento no se deducen unas de otras.

## 4. Identidad e inmutabilidad

Una referencia consumible debe fijar simultáneamente:

```text
fuente_id
fuente_version
fuente_huella_sha256
preceptos_exactos
```

No existe el selector implícito «última versión». Un expediente iniciado con
una referencia conserva esa referencia salvo que un acto posterior ordene una
revisión explícita.

Reglas de historia:

1. Un borrador puede corregirse mediante control optimista y deja auditoría de
   cada revisión. Conserva la cadena de huella anterior/nueva y todos los
   actores que lo editaron, no solo el último.
2. Una versión publicada no cambia de contenido.
3. Una corrección material crea una versión sucesora que identifica a la
   anterior.
4. Suspender o derogar conserva el contenido publicado y añade el acto que
   justifica la transición.
5. Retirar un fichero del almacenamiento no borra la identidad documental ni
   su historia; conservación y acceso se gobiernan por capacidades distintas.
6. Una fuente posterior no altera huellas de catálogos, flujos o convocatorias
   históricos.
7. El constructor público solo puede crear la versión 1. Las sucesoras nacen
   desde una versión existente no borrador, no admiten contenido idéntico y
   requieren un instante de registro estrictamente posterior.
8. El linaje de una sucesora fija la referencia y huella de contenido de la
   predecesora, su revisión, estado, cabeza de historia y huella de estado. Por
   tanto, identifica la instantánea completa de la que nació, aunque la versión
   predecesora experimente después otra transición de estado.
9. La huella de estado usa el esquema explícito
   `vec.fuente_autoridad.estado.v1`; no depende de serializar directamente la
   disposición accidental de una estructura Go.
10. Alta, ediciones y transiciones forman una cadena incremental de historia.
   Cada eslabón incorpora la cabeza anterior y el compromiso que se firma para
   una transición incluye esa cabeza; reescribir un editor o motivo anterior
   invalida el acto posterior.
11. Contenido, estado, compromiso y mensaje de atestación se convierten a DTO
    V1 congelados compuestos solo por primitivas. Añadir campos al modelo vivo
    no cambia retroactivamente bytes ni huellas V1.
12. Versiones, revisiones, secuencias y versiones documentales son `uint64` en
    dominio y persistencia. La sucesión falla cerrada al alcanzar el máximo y
    la rehidratación rechaza desbordamientos; los límites funcionales de
    ediciones y transiciones impiden alcanzar ese extremo por acumulación.

## 5. Estados y transiciones

El primer contrato admite estados técnicos cerrados:

```text
borrador -> publicada -> suspendida
                      -> derogada
suspendida -> publicada, solo mediante acto posterior de levantamiento
suspendida -> derogada
```

Los nombres representan el estado del registro, no una interpretación jurídica
universal. Cada transición distinta del alta de borrador exige un acto
comprobado. La reincorporación desde suspensión conserva tanto el acto de
suspensión como el de levantamiento.

Una nueva versión y una transición de estado no son equivalentes:

- cambiar el documento, los preceptos, el órgano, los ámbitos o los periodos
  crea otra versión;
- suspender, levantar la suspensión o derogar conserva la versión de contenido
  e incorpora una evidencia de transición.

## 6. Comprobación de actos

El caso de uso no aceptará como autoridad una cadena recibida en el cuerpo
HTTP. Antes de publicar o cambiar el estado preparará una solicitud opaca con
un compromiso canónico exacto. El adaptador firmante recibe sus bytes, no una
segunda colección de parámetros que pudiera divergir. El compromiso fija:

- `SolicitudRef`, única dentro de la historia de esa versión;
- ID, versión y huella del contenido afectado;
- revisión previa, secuencia y cabeza de la historia previa;
- estado anterior y posterior y acción exacta;
- persona canónica y código de motivo;
- `PreparadaEn` y `ExpiraEn`, que delimitan la vigencia técnica de la
  operación; el instante final de registro no forma parte del compromiso porque
  lo aporta el servidor al confirmar.

La solicitud es durable en su frontera: `BytesCanonicos`/`MarshalJSON` emiten
el compromiso V1 exacto y `RehidratarSolicitudTransicionFuenteAutoridadV1`
solo reconstruye una solicitud si recibe esos mismos bytes, sin campos
desconocidos, duplicados, espacios ni representaciones alternativas. Esto
permite atravesar un reinicio o una espera de Portafirmas sin reconstruir
actor, motivo, revisión o acción desde el callback. El futuro caso de uso y su
repositorio deberán custodiar los bytes y el estado de la operación
`pendiente/completada/cancelada/expirada/obsoleta`; este corte de dominio aún
no implementa ese registro de pendientes.

El mensaje completo de atestación anida ese compromiso y añade la evidencia
examinada: referencia de evidencia, acto, documento, representación, huella
documental, órgano, firmas o sellos, comprobador y tiempos del acto y de la
comprobación. Solo quedan fuera el identificador, la huella y la firma del
propio sobre criptográfico, porque incluirlos en el mensaje que firman crearía
una dependencia circular. El comprobador productivo deberá acreditar que ese
sobre firma precisamente la huella del mensaje.

La cronología separa dos relojes que no deben confundirse:

| Reloj | Instantes | Regla |
| --- | --- | --- |
| Jurídico | `ActoOcurridoEn`, vigencia y efectos | El acto puede ser histórico y anterior al alta o a la preparación; debe ser igual o anterior a su comprobación. Su eficacia se decide por la fuente y la política jurídica, no por el orden técnico. |
| Técnico | Última mutación, `PreparadaEn`, `ComprobadaEn`, `RegistradaEn`, `ExpiraEn` | Se exige `última mutación < PreparadaEn ≤ ComprobadaEn ≤ RegistradaEn < ExpiraEn`. `ExpiraEn` es un límite exclusivo y el servidor aporta `RegistradaEn`. |

Una solicitud aplicada contra otra revisión o cabeza de historia queda
obsoleta; una comprobación o confirmación igual o posterior a `ExpiraEn` queda
expirada. Una evidencia
de la primera suspensión no es reutilizable tras levantarla y volver a
suspender: cambian, como mínimo, solicitud, revisión, secuencia, historia y
ventana técnica y, por tanto, la huella del compromiso. Dentro de una versión
no pueden repetirse las referencias de solicitud, evidencia ni atestación. Un
mismo documento o firma sí puede intervenir legítimamente en actos distintos;
PostgreSQL impondrá la unicidad global solo a identificadores que sean
intrínsecamente únicos.

El puerto podrá conectarse en el futuro a AutoFirmaV3, @firma, un portafirmas,
un servicio de sello o una comprobación humana asistida. El núcleo no da por
válida la competencia del órgano por reconocer su nombre: exige una respuesta
positiva de la política institucional correspondiente.

`Validar` comprueba forma, referencias, huellas derivadas y coherencia del
mensaje, pero **no** valida criptografía, cadena de confianza, revocación,
sello de tiempo, procedencia ni competencia. El adaptador en memoria solo
probará el contrato. Sus evidencias no serán válidas en producción.

## 7. Gobierno y autorización

Toda mutación exige:

- `ContextoActor` resuelto por el servidor;
- vínculo de autenticación revalidado;
- garantía de autenticación alta;
- decisión positiva y exacta del PDP;
- finalidad, motivo y correlación no vacíos;
- control optimista de la huella anterior;
- consumo único de la decisión de autorización;
- agregado, auditoría y evento de bandeja de salida en una sola transacción.

Las acciones se conceden por separado: crear borrador, actualizar, publicar,
suspender, levantar suspensión y derogar. Ninguna de ellas se deriva del nombre
del rol ni de un permiso enviado por el navegador.

Como defensa mínima, quien publica no puede ser quien creó ni **ninguna** de
las personas que editaron el borrador. Las identidades son referencias
canónicas opacas `per_`, no cuentas, alias, DNI o nombres. Las
incompatibilidades institucionales adicionales se
resolverán mediante RBAC/ABAC y políticas de segregación, no mediante listas de
personas en el código.

## 8. Datos y seguridad

El agregado no necesita DNI, nombre, correo, cuenta bancaria, afiliación,
salud, nómina ni datos de expedientes personales. Actor, órgano, documento y
actos se enlazan mediante referencias opacas.

Los motivos no son notas libres: son claves de catálogo gobernadas. La
justificación humana y su documentación se custodian fuera del agregado, con
clasificación, acceso y conservación propios. Los textos humanos admitidos se
exigen en NFC y rechazan controles y caracteres de formato invisibles; las
referencias técnicas son ASCII y las huellas SHA-256 nulas no se aceptan.

`EstadoPersistibleV1` emite un único JSON canónico: fechas UTC RFC3339Nano,
listas vacías como `[]`, enteros sin dependencia de la arquitectura y orden
determinista. La rehidratación rechaza campos desconocidos o repetidos, listas
`null`, espacios o campos reordenados, fechas equivalentes escritas de otra
forma, enteros desbordados y cualquier valor que no vuelva a producir
exactamente los mismos bytes. `HuellaEstadoSHA256` es el SHA-256 de esos bytes.

Los bytes canónicos son la evidencia persistida y se almacenarán sin
transformación en `BYTEA` en PostgreSQL o `BLOB` en Oracle/otro conector. Una
columna `JSONB`, columnas indexadas o documentos JSON de consulta serán solo
proyecciones reconstruibles: nunca la fuente de verdad ni el material sobre el
que se sustituya o recalcule una firma. Al leer, el repositorio rehidratará y
comparará los bytes exactos antes de usar el agregado.

Antes de decodificar DTO se aplican el límite total de bytes y un recorrido
por tokens. Este *preflight* rechaza profundidad superior a 16, más de 64
campos por objeto, claves repetidas y arrays por encima de su límite: 128
ámbitos, 512 valores, 1.024 preceptos, 128 ediciones, 128 transiciones y 64
firmas; los arrays genéricos se limitan a 128. Así una entrada pequeña pero
patológica no fuerza millones de asignaciones antes de validar invariantes.

Las cadenas SHA-256 internas detectan alteraciones accidentales y ligan la
historia, pero un actor con escritura total sobre la base podría recomponer una
cadena completa. Producción deberá revalidar las atestaciones criptográficas y
anclar periódicamente la cabeza de historia en almacenamiento WORM o un sistema
externo sellado e independiente de las credenciales de escritura de VEC.

La auditoría y los eventos solo incluirán:

- referencias opacas;
- acción, finalidad y resultado;
- versión, estado y huellas;
- tiempos canónicos;
- correlación y evidencia de autorización.

No se copiarán títulos completos de documentos ni texto normativo en logs o
eventos. El contenido se recuperará desde el archivo lógico bajo autorización.

## 9. Puertos y adaptadores

El corte hexagonal prevé:

| Puerto | Responsabilidad |
| --- | --- |
| Consulta de fuentes | Obtener una versión exacta y listar su historia sin elegir una versión por defecto |
| Repositorio de gobierno | Confirmar cada transición con OCC, auditoría, outbox y consumo de autorización |
| Comprobador de actos | Verificar documento, huella, órgano y firmas sin introducirlos desde HTTP como autoridad |
| Reloj | Aportar instantes UTC canónicos desde composición |
| Autorizador | Exigir una decisión PDP vinculada al actor, acción, recurso y finalidad exactos |
| Operaciones pendientes | Custodiar solicitud canónica, caducidad, estado terminal e idempotencia durante Portafirmas o comprobaciones asíncronas |

Adaptadores previstos:

- memoria, únicamente para pruebas de contrato;
- PostgreSQL, primera persistencia productiva;
- archivo/documentos comunes para recuperar representaciones;
- comprobación de firma y acto mediante conectores sustituibles.

La composición pertenece a `internal/app/bootstrap`. El adaptador HTTP solo
traducirá DTO, límites y errores después de que el caso de uso esté cerrado.
El repositorio será también la autoridad de la cadena: comprobará
transaccionalmente que la predecesora existe, que no nacen dos sucesoras con la
misma versión y que una rehidratación no introduce historia sintética. Deberá
conservar además las revisiones de borrador de forma append-only: el agregado
encadena sus huellas, pero la recuperación probatoria del contenido intermedio
corresponde al repositorio. Estado canónico, solicitud pendiente, auditoría,
outbox y consumo de autorización se confirmarán atómicamente; un proceso
separado exportará las cabezas al anclaje WORM y reconciliará fallos sin dar
por confirmada una transición incompleta.

## 10. Integración gradual

El registro se incorporará sin reinterpretar datos históricos:

1. Crear y probar el agregado y su referencia exacta.
2. Añadir puertos, caso de uso y doble en memoria.
3. Implantar PostgreSQL, ACL/RLS, procedimiento transaccional y pruebas SQL.
4. Conectar la comprobación institucional de actos y firmas.
5. Crear versiones nuevas de contratos consumidores que admitan la referencia
   exacta.
6. Migrar catálogos, flujos y convocatorias solo cuando cada referencia antigua
   pueda reconciliarse y aprobarse.
7. Mantener lectores de compatibilidad; nunca recalcular una huella publicada
   para sustituir `FuenteRef` por otro formato.

Los primeros consumidores previstos son catálogos gobernados, convocatorias y
versiones de baremo. Cronos, Nómina, carrera, turnicidad y materias reservadas
no se activarán por disponer de este registro: seguirán necesitando sus fuentes
consolidadas y políticas tipadas propias.

## 11. Pruebas exigidas

El incremento no se considerará cerrado sin:

- vectores dorados independientes para huella de contenido y estado V1;
- estabilidad ante permutación de ámbitos y preceptos;
- rechazo de duplicados, UTF-8 inválido, controles, comodines y límites;
- periodos de vigencia y efectos inválidos;
- tiempos no UTC o con precisión no admitida;
- edición de una versión ya publicada;
- secuencia de versiones rota;
- linaje de sucesora alterado en revisión, estado, historia o huella de estado;
- rechazo de sucesora idéntica y agotamiento del contador `uint64`;
- transición sin acto comprobado o para otra huella;
- reutilización de evidencia entre ciclos de suspensión;
- segregación frente al creador y cualquiera de los editores históricos;
- aceptación de actos jurídicos históricos y rechazo de una cronología
  técnica fuera de la ventana preparada;
- solicitud durable rehidratada tras reinicio, obsoleta, expirada y repetida;
- garantía de que todo agregado válido produce ambas huellas;
- validación lineal con contenido próximo a 4 MiB y 128 transiciones;
- igualdad exacta entre huella de estado y bytes persistibles V1;
- ida y vuelta estricta de borradores y agregados con edición, acto y firmas;
- vectores dorados de compromiso y mensaje atestado;
- causalidad del acto aun cuando se recalcule un mensaje estructuralmente
  válido;
- rechazo previo a la asignación de JSON profundo, duplicado o con arrays
  patológicos, más pruebas de fuzz y ejecución en arquitectura de 32 bits;
- reutilización de una decisión de autorización para otro efecto;
- conflictos OCC, idempotencia, cancelación y concurrencia;
- atomicidad entre agregado, auditoría y outbox;
- ausencia de datos personales en auditoría y eventos;
- pruebas de ACL/RLS y cuentas PostgreSQL de mínimo privilegio;
- comprobación de que los adaptadores alternativos cumplen el mismo contrato.

## 12. Condiciones para retirar el NO-GO

La capacidad podrá usarse como autoridad productiva únicamente cuando existan
y estén aprobados:

- migración PostgreSQL y procedimiento transaccional;
- identidades técnicas, ACL, RLS y rotación de secretos;
- auditoría resistente a manipulación y exportación al sistema corporativo;
- almacenamiento y recuperación segura del documento original;
- comprobación criptográfica real del mensaje atestado, certificados, cadena,
  revocación, firmas, sellos, tiempo y vigencia aplicables;
- resolución atestada de competencia del órgano y de la persona actuante,
  además de segregación institucional persistente y probada;
- custodia durable de solicitudes pendientes y recuperación tras reinicios,
  cancelaciones, expiraciones y callbacks repetidos;
- repositorio append-only y anclaje externo WORM o sellado equivalente de las
  cabezas de historia, con verificación y procedimiento de alarma;
- catálogo institucional de órganos, materias y ámbitos;
- procedimiento de alta, corrección, suspensión y derogación validado por
  Secretaría, RRHH, Sistemas y Seguridad según la materia;
- copias, restauración y ensayo de recuperación;
- pruebas de aceptación con fuentes provinciales reales no sensibles.
