# Registro canónico de fuentes de autoridad

Fecha de corte: **17 de julio de 2026**.

Estado: arquitectura del primer incremento de `NUC-006`. El adaptador en
memoria será únicamente un doble de contrato. La capacidad permanecerá en
**NO-GO productivo** hasta disponer de persistencia PostgreSQL, comprobación
real de actos y firmas, auditoría probatoria y validación institucional.

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
   cada revisión.
2. Una versión publicada no cambia de contenido.
3. Una corrección material crea una versión sucesora que identifica a la
   anterior.
4. Suspender o derogar conserva el contenido publicado y añade el acto que
   justifica la transición.
5. Retirar un fichero del almacenamiento no borra la identidad documental ni
   su historia; conservación y acceso se gobiernan por capacidades distintas.
6. Una fuente posterior no altera huellas de catálogos, flujos o convocatorias
   históricos.

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
HTTP. Antes de publicar o cambiar el estado solicitará a un puerto de
comprobación una evidencia ligada a:

- acción exacta;
- ID, versión y huella del contenido afectado;
- acto, documento y representación examinados;
- huella de la representación;
- órgano declarado;
- firmas o sellos verificados;
- comprobador, instante y resultado.

El puerto podrá conectarse en el futuro a AutoFirmaV3, @firma, un portafirmas,
un servicio de sello o una comprobación humana asistida. El núcleo no da por
válida la competencia del órgano por reconocer su nombre: exige una respuesta
positiva de la política institucional correspondiente.

El adaptador en memoria solo probará el contrato. Sus evidencias no serán
válidas en producción.

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

Como defensa mínima, quien publica no puede ser quien creó o editó por última
vez el borrador. Las incompatibilidades institucionales adicionales se
resolverán mediante RBAC/ABAC y políticas de segregación, no mediante listas de
personas en el código.

## 8. Datos y seguridad

El agregado no necesita DNI, nombre, correo, cuenta bancaria, afiliación,
salud, nómina ni datos de expedientes personales. Actor, órgano, documento y
actos se enlazan mediante referencias opacas.

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

Adaptadores previstos:

- memoria, únicamente para pruebas de contrato;
- PostgreSQL, primera persistencia productiva;
- archivo/documentos comunes para recuperar representaciones;
- comprobación de firma y acto mediante conectores sustituibles.

La composición pertenece a `internal/app/bootstrap`. El adaptador HTTP solo
traducirá DTO, límites y errores después de que el caso de uso esté cerrado.

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

- vectores dorados de huella canónica;
- estabilidad ante permutación de ámbitos y preceptos;
- rechazo de duplicados, UTF-8 inválido, controles, comodines y límites;
- periodos de vigencia y efectos inválidos;
- tiempos no UTC o con precisión no admitida;
- edición de una versión ya publicada;
- secuencia de versiones rota;
- transición sin acto comprobado o para otra huella;
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
- comprobación real de firmas, sellos y vigencia;
- catálogo institucional de órganos, materias y ámbitos;
- procedimiento de alta, corrección, suspensión y derogación validado por
  Secretaría, RRHH, Sistemas y Seguridad según la materia;
- copias, restauración y ensayo de recuperación;
- pruebas de aceptación con fuentes provinciales reales no sensibles.

