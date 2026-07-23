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
reautentica.

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
- propuesta vinculada.

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
huella de respuesta, y tiempos. Los datos funcionales completos están
incluidos por sus materiales canónicos.

La implementación del adaptador real debe custodiar las claves fuera del
proceso, mediante HSM/KMS o servicio institucional equivalente. Este contrato
no incorpora secretos ni una implementación criptográfica de producción.

## Persistencia y reautenticación

`EvidenciaDurableIntegracionBolsa` es serializable y minimizada. Se conserva
junto al comando/recibo/evento y contiene solo las referencias, huellas,
generación, sello y tiempos necesarios para una verificación posterior.

Tras un reinicio:

1. se reconstruyen los bytes canónicos desde los artefactos conservados;
2. se exige coincidencia exacta con la evidencia persistida;
3. se comprueba la autoridad y el anillo configurados localmente;
4. se verifica de nuevo el HMAC;
5. se crea un comprobante opaco nuevo.

La caducidad del transporte y la autenticidad durable son controles distintos.
Una respuesta caducada no puede usarse como respuesta fresca, aunque su
evidencia histórica todavía pueda reautenticarse.

El comprobante opaco no se persiste ni se acepta por transporte.

## Idempotencia y concurrencia

La identidad de inbox es autoridad más referencia de evento. Un replay con
bytes idénticos conserva el mismo acuse. Reutilizar la misma identidad con
otra carga es una colisión y se rechaza.

Cada evento porta secuencia anterior, secuencia siguiente y versión esperada.
El adaptador de persistencia deberá aplicar el control CAS y escribir estado,
auditoría, inbox y outbox en una única transacción. O6-01 define el contrato;
la transacción real corresponde a la tarea de persistencia posterior.

## Minimización

El contrato no transporta DNI, nombre, correo, teléfono ni dirección. La
selección se representa mediante una referencia seudonimizada y una política
de retención versionada. Esa referencia no debe entrar en logs ni telemetría.

Las versiones numéricas se limitan al entero seguro común de JSON para evitar
interpretaciones distintas entre Go, web, escritorio, CLI y MCP. También se
limitan la cardinalidad y la vigencia antes de invocar conectores.

## Pruebas incluidas

La batería focal cubre:

- imposibilidad de fabricar o serializar capacidades opacas;
- alteración y caducidad del contexto firmado;
- cancelación de fronteras;
- autoridad Bolsa fija y verificador sin capacidad de firma;
- rotación v2 con v1 retenida y rechazo al retirar v1;
- persistencia JSON y reautenticación tras reinicio y caducidad;
- matriz adversarial de organización, expediente, versión, correlación,
  finalidad, acción y recurso;
- referencias y huellas exactas de petición y recibo en eventos;
- alteración del material, replay, colisión, secuencia y acuse CAS;
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
- persistencia PostgreSQL e inbox/outbox transaccionales;
- composición con HSM/KMS y rotación operativa;
- API, CLI, MCP, web o escritorio;
- selección funcional, exclusiones, desempate y evidencia de reglas;
- comunicaciones, aceptación, renuncia y siguiente candidato;
- documentos, firmas múltiples y formalización;
- pruebas E2E, despliegue y autorización de producción.

Estas capacidades pertenecen a O6-02 a O6-05 y no deben declararse terminadas
por el cierre de este contrato.
