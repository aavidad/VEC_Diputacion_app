# O6-01 — Contrato de integración con Bolsa

Fecha: 23 de julio de 2026

Ámbito: `contrataciontemporal/ports`

Estado del entregable: contrato y pruebas; no implica integración, E2E ni
producción.

## Decisión

Contratación Temporal consume Bolsa exclusivamente mediante puertos,
referencias opacas, recibos y eventos. No comparte tablas, repositorios,
identidades personales ni una sesión web con Bolsa.

El contrato es neutral al transporte. Web, aplicación de escritorio, CLI y MCP
deben traducir su entrada al mismo caso de uso; la autorización no procede de
cookies, campos JSON, cabeceras libres ni estado del navegador.

## Flujos contratados

1. Consulta acotada de disponibilidad, considerada volátil.
2. Preparación de una fotografía durable y completa del orden de Bolsa.
3. Solicitud de propuesta de llamamiento a partir de la orden autenticada.
4. Recepción de un evento durable de cambio del llamamiento.
5. Registro idempotente del evento mediante inbox, secuencia y versión
   esperadas.

La consulta de disponibilidad no acredita una actuación administrativa. Los
recibos de orden, llamamiento y evento sí pueden conservarse junto a su
evidencia durable.

## Contexto de petición

`ContextoPeticionIntegracionBolsa` es una capacidad opaca:

- sus datos internos no son exportados;
- no admite serialización ni reconstrucción JSON;
- solo lo emite la frontera confiable después de ligar autoridad solicitante,
  autorización, acción, recurso, finalidad, organización, expediente, versión,
  correlación y vigencia;
- el registro firmado que cruza un adaptador se rehidrata solo después de
  verificar autoridad, generación de clave y HMAC;
- la frescura se comprueba para cada uso en línea.

El registro serializable no concede autoridad por sí mismo. Es material no
confiable hasta que `AutenticadorContextoPeticionIntegracionBolsa` lo
reautentica. La autenticación histórica del registro no vuelve a abrir sus
quince minutos de uso: `DatosEn` continúa rechazando un instante caducado.

## Ligadura orden, llamamiento y evento

Un llamamiento nuevo requiere el recibo de orden y el comprobante opaco
obtenido al autenticarlo. El contexto nuevo debe coincidir exactamente con la
orden en:

- organización;
- expediente;
- versión del expediente;
- correlación;
- finalidad.

Además, la acción y el recurso del contexto nuevo deben ser los indicados por
el recibo firmado: acción de llamamiento y referencia exacta de la orden.

El evento se liga a un `EnlaceEventoLlamamientoBolsa` construido desde el
comando, recibo y comprobante autenticados. El cotejo exige igualdad exacta de:

- organización, expediente, versión y correlación;
- finalidad, acción y recurso gobernados;
- referencia y huella SHA-256 de la petición;
- referencia y huella SHA-256 del recibo;
- necesidad, Bolsa, orden, política y propuesta;
- referencia de llamamiento y selección seudonimizada;
- política de retención;
- solicitud, caducidad, confirmación del recibo, emisión, caducidad probatoria
  y límite de retención.

Por tanto, un evento no puede declarar referencias y huellas autoconsistentes
pero ajenas a los artefactos locales.

## Evidencia criptográfica

El consumidor recibe un puerto de verificación HMAC, no un sellador. El
verificador local fija:

- la autoridad Bolsa esperada;
- la generación activa;
- hasta tres generaciones anteriores retenidas, en orden descendente.

La autoridad o clave declaradas por una respuesta nunca amplían esa
configuración. La rotación permite verificar, por ejemplo, evidencia v1 con v2
activa únicamente mientras v1 permanezca explícitamente retenida.

La firma cubre un sobre canónico que contiene tipo, autoridad, referencia de
clave, referencia de evidencia, referencia y huella de petición, referencia y
huella de respuesta, tiempos de transporte y límite de retención. Los datos
funcionales completos están incluidos por sus materiales canónicos.

La implementación del adaptador real debe custodiar las claves fuera del
proceso, mediante HSM/KMS o servicio institucional equivalente. Este contrato
no incorpora secretos ni una implementación criptográfica de producción.

## Persistencia y reautenticación

`EvidenciaDurableIntegracionBolsa` es serializable y minimizada. Se conserva
dentro de un artefacto probatorio cerrado y contiene solo las referencias,
huellas, generación, sello y tiempos necesarios para una verificación
posterior.

Los artefactos durables son:

- `ArtefactoProbatorioOrdenBolsa`;
- `ArtefactoProbatorioLlamamientoBolsa`;
- `ArtefactoProbatorioEventoBolsa`.

Cada uno declara esquema, versión y tipo; incorpora el HMAC de la evidencia y
una huella SHA-256 de todo su sobre. La decodificación probatoria se realiza
con `DecodificarArtefactoProbatorioOrdenBolsa`,
`DecodificarArtefactoProbatorioLlamamientoBolsa` o
`DecodificarArtefactoProbatorioEventoBolsa`, según el tipo. Estas fronteras
aplican antes de materializar el DTO:

- máximo de 1 MiB;
- profundidad máxima de 24 niveles;
- máximo de 256 miembros por objeto o elementos por colección;
- rechazo de claves duplicadas en cualquier nivel;
- rechazo de campos desconocidos y contenido posterior;
- esquema, versión y tipo exactos;
- recodificación canónica idéntica byte a byte.

No se debe sustituir esta frontera por un `json.Unmarshal` genérico: la
biblioteca estándar puede normalizar espacio exterior antes de invocar el
codec del tipo y, por tanto, no puede aplicar el límite al encuadre original.
La huella exterior, el HMAC y la recodificación canónica son controles
complementarios. Estos artefactos son datos probatorios, no implementan ningún
puerto de efecto.

Tras un reinicio:

1. se decodifican los artefactos cerrados y se comprueba su huella;
2. se reautentica históricamente el registro de contexto, sin exigir que siga
   fresca la ventana de uso;
3. se reconstruyen internamente los bytes canónicos;
4. se exige coincidencia exacta con la evidencia persistida;
5. se comprueba la autoridad y el anillo configurados localmente;
6. se verifica de nuevo el HMAC y la retención;
7. se crea una capacidad histórica opaca y no serializable.

La caducidad del transporte, la autenticidad durable y la retención son
controles distintos. Una respuesta caducada no puede usarse como respuesta
fresca, aunque su evidencia histórica todavía pueda reautenticarse. Al llegar
al límite firmado `RetenerHasta` también termina la reautenticación histórica.

El comprobante y las capacidades históricas opacas no se persisten ni se
aceptan por transporte. Una orden histórica solo puede alimentar una petición
nueva con un contexto fresco; un llamamiento histórico solo puede derivar el
enlace probatorio; y un evento histórico solo puede producir el comando
idempotente de registro. No permiten repetir el efecto original.

Los constructores en línea que emiten un llamamiento o registran un evento
exigen que el comprobante haya sido verificado en el mismo instante confiable
de la operación. La capacidad histórica rehidratada no conserva como vigente
el instante de su reautenticación: al construir el comando recibe otra vez el
instante actual y vuelve a comprobar `RetenerHasta`.

Además, el comando de registro solo entrega sus datos mediante
`DatosParaEfectoEn`. El adaptador de inbox debe obtener ese instante de su
reloj confiable y abrir la capacidad dentro de la misma transacción CAS que
escribe inbox, estado, auditoría y outbox. Así, un comando preparado antes de
la caducidad tampoco puede surtir efecto después de ella.

## Idempotencia y concurrencia

La identidad de inbox es autoridad más referencia de evento. Un replay con
bytes idénticos conserva el mismo acuse. Reutilizar la misma identidad con
otra carga es una colisión y se rechaza.

Cada evento porta secuencia anterior, secuencia siguiente y versión esperada.
El adaptador de persistencia deberá aplicar el control CAS, revalidar la
retención con `DatosParaEfectoEn` y escribir estado, auditoría, inbox y outbox
en una única transacción. O6-01 define el contrato; la transacción real
corresponde a la tarea de persistencia posterior.

## Minimización

El contrato no transporta DNI, nombre, correo, teléfono ni dirección. La
selección usa el tipo nominal `SeudonimoSeleccionBolsa`, cuya única gramática
admitida es un HMAC-SHA-256 con dominio
`vec.contratacion-temporal.seleccion` y versión explícita. No admite una
cadena opaca libre, un DNI, un NIE, un correo, un nombre ni un identificador
directo. `String`, `GoString`, todos los verbos de `Format` y `LogValue`
devuelven una marca redactada; los codecs JSON/texto validados se reservan al
transporte autorizado. La política de retención continúa siendo versionada.

Las versiones numéricas se limitan al entero seguro común de JSON para evitar
interpretaciones distintas entre Go, web, escritorio, CLI y MCP. También se
limitan la cardinalidad y la vigencia antes de invocar conectores.

## Pruebas incluidas

La batería focal cubre:

- imposibilidad de fabricar o serializar capacidades opacas;
- alteración y caducidad del contexto firmado;
- cancelación anterior y producida dentro de `SellarDatos` o
  `VerificarDatos`, sin promover una capacidad después de la criptografía;
- autoridad Bolsa fija y verificador sin capacidad de firma;
- rotación v2 con v1 retenida y rechazo al retirar v1;
- persistencia JSON y reautenticación tras reinicio y caducidad;
- reinicio real simulado destruyendo todas las capacidades opacas, creando
  verificadores nuevos y llegando a registro a las veinticuatro horas;
- matriz adversarial de organización, expediente, versión, correlación,
  finalidad, acción, recurso, necesidad, Bolsa, orden, política, llamamiento,
  selección, retención y cronología;
- referencias y huellas exactas de petición y recibo en eventos;
- alteración del material, replay, colisión, secuencia y acuse CAS;
- uso después de `RetenerHasta`, tanto desde una capacidad rehidratada antes
  como en la apertura dentro de la frontera CAS;
- rechazo de DNI, NIE, correo, nombre, identificadores directos y dominios
  HMAC ajenos como seudónimo, además de redacción en `fmt` y `slog`;
- matriz de JSON con duplicados, contenido posterior, exceso de tamaño,
  profundidad, colección y codificación no canónica, junto a esquema, versión
  y tipo estrictos para los tres artefactos;
- neutralidad respecto a web, escritorio, CLI y MCP.

Puertas reproducidas en el entregable:

```text
go test ./internal/modules/contrataciontemporal/ports
go test -race ./internal/modules/contrataciontemporal/ports
go vet ./internal/modules/contrataciontemporal/ports
go test ./...
git diff --check
```

## Fuera de alcance de O6-01

- adaptador real contra el módulo Bolsa;
- persistencia PostgreSQL e inbox/outbox transaccionales; los artefactos de
  este contrato son el formato que deberá conservar ese adaptador;
- composición con HSM/KMS y rotación operativa;
- API, CLI, MCP, web o escritorio;
- selección funcional, exclusiones, desempate y evidencia de reglas;
- comunicaciones, aceptación, renuncia y siguiente candidato;
- documentos, firmas múltiples y formalización;
- pruebas E2E, despliegue y autorización de producción.

Estas capacidades pertenecen a O6-02 a O6-05 y no deben declararse terminadas
por el cierre de este contrato.
