# Contrato de auditoría común

Este documento fija qué contratos existen en `main`@`936aac665084b57b0d71adad15672e85e0efbb6a` y qué debe completar el equipo L. Sirve a K para registrar usuarios, perfiles y revocaciones, y a los demás módulos para integrar sus operaciones. La cobertura comprobada por módulo está en el [inventario del 03/10](../estudio_requisitos/cobertura_auditoria_2026-10-03.md).

La coordinación quedó publicada en `CANAL_CLAUDE_CODEX.md`, entrada «2026-10-03 01:30 CEST — De: Codex-L — CONTRATO AUDITORÍA PARA K Y LOS MÓDULOS». El canal queda fuera de Git. Este documento conserva su contenido técnico verificable y marca las ampliaciones pendientes; escribir el contrato no habilita rutas ni instala migraciones.

## 1. Propiedad y reglas

La auditoría pertenece a la autoridad común. El módulo conserva su estado, versiones, recibos e historia; su adaptador ejecuta el efecto y consume la autorización nominal. Las piezas nuevas registran la auditoría común dentro de esa misma transacción. Las tablas históricas propias se conservan y se enlazan por referencia; no se crea otra auditoría de módulo para sustituir la común.

Cada operación de una persona requiere identidad acreditada, perfil activo, acción, recurso opaco, finalidad, resultado, instante, proceso, canal y correlación. La orden del 03/10 incluye consultas y descargas de datos personales. K suministra identidad, perfil, asignación y versiones desde sus autoridades; los nombres visibles proceden de una fuente nominal autorizada. Un cuerpo HTTP, el nombre del cargo, el menú o la titularidad no conceden permiso ni acreditan quién actuó.

La autenticación que falla antes de identificar a una persona conserva un intento sin actor acreditado. La frontera no construye una persona a partir del certificado sin verificar, una cabecera o la carga de la petición. Los eventos excluyen DNI, correos, datos de salud, certificados, secretos, documentos, payloads y textos libres de error.

## 2. Modelo y puertos vigentes

El modelo es [`domain.AuditEntry`](../../internal/vec/domain/types.go). El puerto antiguo está en [`ports.AuditStore`](../../internal/vec/ports/ports.go):

```go
type AuditStore interface {
    AppendAudit(context.Context, domain.AuditEntry) (domain.AuditEntry, error)
    ListAudit(context.Context, string) ([]domain.AuditEntry, error)
}
```

Estas firmas no reciben una transacción ni acreditan una concesión ligada al efecto. El adaptador durable [`bolsa/adapters/postgres/registroaccesos/registro.go`](../../internal/modules/bolsa/adapters/postgres/registroaccesos/registro.go) deniega expresamente `AppendAudit` y `ListAudit`. Su contrato específico de consulta administrativa sí confirma permiso, lectura y auditoría en una transacción. El adaptador en memoria implementa las firmas genéricas para sus capacidades; su disponibilidad no prueba persistencia PostgreSQL.

Por tanto, K y las nuevas verticales deben reutilizar las fachadas nominales V3 o un contrato común que L complete con esa ligadura. No se habilita el `AppendAudit` genérico como permiso global ni se lo llama después de un efecto ya confirmado para declarar atomicidad. A la fecha de corte todavía no existe en estas firmas un puerto universal que haga esa integración por todos los módulos.

La consulta compartida utiliza [`auditoria.FuenteAuditoria`](../../internal/vec/auditoria/contrato.go):

```go
type FuenteAuditoria interface {
    ConsultarAuditoria(context.Context, ConsultaAutorizada) (PaginaFuente, error)
}
```

`ConsultaAutorizada` transporta filtro y material nominal V3, solicitud, decisión, confirmación y contexto de actor. [`Servicio.Consultar`](../../internal/vec/auditoria/servicio.go) emite y comprueba el material, liga cursor/filtros y valida la respuesta de la fuente. Cada propietario consulta su almacén por un adaptador propio; L no consulta tablas ajenas.

Actualmente `Filtro` solo admite `ct` o `bolsa`, exige expediente y fechas, admite actor opcional y usa paginación ligada a la concesión. Ampliar fuentes, campos o filtros requiere un contrato y permiso propios; aceptar una fuente nueva en HTTP no le concede acceso.

## 3. Campos: situación real y ampliación pendiente

| Dato | Modelo vigente | Productor y límite |
| --- | --- | --- |
| Identidad nominal | `ActorID` | Autoridad de identidad/contexto; referencia estable opaca. K conserva la relación autorizada con la persona. |
| Perfil activo | `ActorProfile`; `ActorRoles` también existe | Usar perfil activo acreditado; una lista de roles no sustituye la asignación concreta. No todos los productores antiguos lo completan. |
| Asignación y versiones históricas | No son campos propios de `AuditEntry` | La versión de persona/perfil viaja en el material V3. Falta completar una proyección común verificable de asignación y versión; no introducir valores libres en `Metadata`. |
| Representación y autenticación | `RepresentedSubjectID`, `AuthMethod`, `AuthAssurance` | Conservar solo cuando el contexto acredita esas propiedades; no deducir representación de una referencia enviada. |
| Decisión, finalidad y motivo | `AuthorizationRef`, `Purpose`, `Reason`, `RuleRef` | Autoridad de autorización y catálogo de motivos. No escribir razones con contenido personal o mensajes completos de proveedor. |
| Acción y propietario | `Action`, `ModuleID` | Operación declarada por el caso de uso y su capacidad; no aceptar una acción libre para ampliar permisos. |
| Recurso y versión | `SubjectRef`, `ObjectVersion`; `ExpedienteRef` / `DocumentRef` cuando procedan | Referencias opacas y versión del recurso realmente consultado o cambiado. Una ruta no sustituye la referencia del recurso. |
| Resultado | `Result` | Hay productores con `no_confirmado` y otros con estados más específicos. Falta un catálogo transversal que distinga permitido, denegado y error conservando indeterminación del COMMIT. |
| Instante y correlación | `OccurredAt`, `CorrelationRef` | Reloj/contexto de confianza; instante UTC. Deben enlazar petición, autorización, registro y efectos. |
| Proceso y canal | Sin campos propios | Pendiente de L: referencias gobernadas suministradas por composición, incluidas tareas/CLI. No equivalen a IP, URL o User-Agent libre. |
| Huellas e integridad | `BeforeHash`, `AfterHash`, `Seq`, `IntegrityAlgorithm`, `PrevSignature`, `Signature` | El significado depende del esquema del productor. El nombre `Signature` no acredita firma asimétrica ni sello institucional. |
| Destinatario y doble control | Sin campos específicos | Cuando proceda, K/módulo aporta referencia opaca, vínculo y actores distintos de propuesta/aprobación. La relación durable y su proyección siguen pendientes de completar por operación. |

El modelo actual no impone obligatoriedad universal para todos esos campos. La consulta común tampoco los expone todos: su `Registro` conserva actor, acción, resultado, instante, expediente, recibo, motivo y huellas, con proyección minimizada de antes/después. Completar el modelo, su validación, sus productores y la proyección son trabajos distintos.

## 4. Registro durable vigente V3

La autoridad interna está en [AD3-1](../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.up.sql): `atestacion_decision_v3`, `consumo_decision_v3`, `auditoria_consumo_v3` y `control_cadena_auditoria`. [AD3-2](../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.up.sql), `registrar_y_consumir_decision_v3_atestada`, revalida el material antes de registrar el consumo y avanzar la cadena. Los consumidores específicos mantienen la audiencia y operación de cada capacidad.

La fila común de auditoría conserva `auditoria_ref`, secuencia, decisión, efecto, huella del efecto, huella anterior, huella nueva e instante. El actor/perfil/finalidad se ligan por el material de decisión/contexto relacionado; no son columnas nominales completas de esa fila. La cadena externa de [AD116](../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000116_consumo_candidato_externo.up.sql) conserva su separación.

Hasta AD207 la cabeza se bloqueaba con `FOR UPDATE`. La preimagen interna encadena secuencia, huella anterior, decisión, efecto, huella del efecto y huella de consumo. `registrada_en` no aparece como campo separado de esa preimagen. Completar el sobre nominal exige versionar qué bytes protege la huella, conservando las cadenas históricas.

Desde [AD207](../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000207_cadena_auditoria_sellado_diferido.up.sql) ningún escritor bloquea `control_cadena_auditoria`. Sigue el patrón de los registros de transparencia de certificados ([RFC 9162](https://www.rfc-editor.org/rfc/rfc9162.html), §4, y el secuenciador de [Trillian](https://pkg.go.dev/github.com/google/trillian/log)): la entrada se acepta con un acuse, un secuenciador la incorpora por lotes con número contiguo dentro de un plazo máximo, y la cabeza se ancla fuera con el sello periódico de AD186.

Cada asiento toma un número de una secuencia (identifica el asiento en la cola; puede tener huecos por ROLLBACK) y calcula su huella con la misma preimagen de su tipo, con 64 «f» en el lugar del anterior. Un disparador lo deja en `pendiente_sellado_auditoria_v5`. El sellador (grupo `vec_auditoria_encadenador`, LOGIN propio fuera de Git, bucle de `vec-server` configurado con `VEC_AUDITORIA_SELLADO_DATABASE_URL`) toma los pendientes por lotes, les da posición contigua y escribe en `eslabon_auditoria_v5`, de solo adición:

```text
eslabon(p) = sha256(F("vec.auditoria.eslabon.v5") F(cadena) F(p) F(eslabon(p-1))
                    F(secuencia) F(auditoria_ref) F(tipo_registro) F(huella_sha256)
                    F(registrada_en) F(sellado_en))
```

`F` es `encuadrar_mac`, las fechas van en UTC con microsegundos y `cadena` vale `interna` o `externa` (la externa no tiene tipo y usa el texto vacío). La fila de control queda congelada con la última secuencia y la cabeza de la cadena anterior, que no cambia; `eslabon(N)` del corte es esa cabeza. La cadena externa tiene su propia secuencia, cola y tabla de eslabones.

La cadena de accesos RRHH de Contratación temporal ([CT183](../../deploy/postgresql/contratacion_temporal/migraciones/000183_cadena_accesos_rrhh_sellado_diferido.up.sql)) sigue el mismo patrón con tablas propias de CT (`eslabon_acceso_rrhh_v1`, `pendiente_sellado_acceso_rrhh_v1`, `sellado_acceso_rrhh_v1`), dominio `vec.ct.acceso_rrhh.eslabon.v1` y el mismo sellador, que puede ejecutar `sellar_cadena_accesos_rrhh_v1`. La huella de cada acceso sigue siendo `sha256(anterior || prueba_canonica)`, con el marcador como anterior; `verificar_cadena_accesos_rrhh_v1()` la recalcula para toda la cadena.

El plazo máximo de incorporación vive en `sellado_auditoria_v5` (10 s). El sellador renueva allí su latido solo si en la cola no queda nada más antiguo que el plazo. Si el latido caduca, el disparador rechaza los asientos nuevos y la operación auditada no se confirma. Mientras espera su eslabón, un asiento está en la misma tabla, con los mismos disparadores de solo adición. Como en un registro de transparencia, quien conserva un acuse (`auditoria_ref`, número, huella y fecha) puede exigir después que aparezca sellado; un asiento retirado de la cola por el propietario antes de sellarse solo se descubre así o porque su número falta, que por sí solo no distingue un ROLLBACK.

## 5. Confirmación de operaciones y lecturas

La siguiente secuencia es requisito de integración. Los ejemplos citados confirman la transacción antes de devolver datos o recibos; los campos y productores pendientes se indican en el apartado 3. Cada operación nueva debe demostrar la secuencia con su propio recorrido.

1. La frontera acredita identidad y perfil y fija correlación/proceso/canal desde contexto de confianza.
2. El caso de uso pide la concesión exacta para acción, recurso, ámbito, finalidad, versión, campos y obligaciones.
3. El adaptador inicia la transacción y revalida el material V3 y el contexto aplicables antes de consumir la decisión. No basta la comprobación anterior a abrir la transacción.
4. La fachada nominal confirma consumo, efecto, auditoría y outbox cuando la operación lo requiere. Para una lectura personal, el consumo y el registro de acceso se confirman con la lectura autorizada.
5. Solo después del COMMIT confirmado se devuelven datos personales o recibo de éxito. El adaptador no publica filas recuperadas antes de esa confirmación.

Ejemplos trazados: [`Fuente.ConsultarAuditoria` de CT](../../internal/modules/contrataciontemporal/adapters/auditoriaconsulta/fuente.go) confirma antes de devolver la página; [`Documentos.Repositorio.transaccion`](../../internal/vec/documentos/adapters/postgres/repositorio.go) llama una fachada y confirma antes de retornar; [`Dietas.ConsumirAccesoRutasDietas`](../../internal/modules/dietas/adapters/postgres/acceso_rutas.go) exige consumo nuevo y auditoría y confirma antes de devolver el recibo.

Un replay conserva el efecto y su recibo originales, pero una nueva consulta o recuperación nominal puede necesitar una nueva decisión y auditoría de acceso. Una clave de idempotencia no elimina la trazabilidad de cada persona que consulta. Se siguen las fachadas del propietario; no se añade otra alta ni se reescribe el recibo histórico.

## 6. Denegaciones, errores y resultado indeterminado

Una operación denegada no aplica el efecto. Si el efecto se revierte, su auditoría transaccional también puede revertirse: la constancia del intento se registra por una operación separada con datos minimizados y resultado de fallo/denegación. Ese registro no debe confundirse con confirmar la operación original.

La frontera temprana vigente recibe solo correlación, motivo cerrado, superficie, ruta y actor opcional. [`bootstrap/auditoria_consulta.go`](../../internal/app/bootstrap/auditoria_consulta.go) registra 401/403 después del manejador; si falla, genera `slog.Error`. Esta conducta documenta el hueco actual: la denegación permanece, pero el log técnico no prueba que se haya persistido el intento en la auditoría funcional común. La política de indisponibilidad y el registrador nominal de intentos son pendientes de L; un error del registrador nunca concede permiso.

Una respuesta perdida del COMMIT puede dejar un efecto confirmado. [`Cronos.ejecutarTransaccionMarcaje`](../../internal/modules/cronos/adapters/postgres/transaccion_marcaje.go) distingue `fallo_confirmado` de `resultado_indeterminado`; `Rollback` sobre una transacción cerrada no prueba reversión. La recuperación debe comprobar el recibo durable antes de reintentar. No se escribe «error sin efecto» cuando la base no lo acredita.

Consultar un documento y obtener permiso de descarga son operaciones identificables; registrar la autorización no demuestra que el cliente recibiera todos los bytes. El resultado de transmisión, si se recoge, se añade como evento relacionado y nunca cambia el registro histórico del acceso.

## 7. Consulta, exportación y conservación pendientes

La consulta común de CT/Bolsa y su permiso propio se mantienen. La ampliación para administración debe añadir fuente y proyección de cada propietario, filtros autorizados por persona/recurso/acción/fechas y auditoría de la propia consulta. K aporta la sesión ADMIN y el perfil exacto; L no decide que cualquier administrador pueda leer todos los registros.

La exportación judicial necesita su acción, finalidad y ámbito, conjunto/versiones exactos, manifiesto de huellas, firma y sello admitidos, instrucciones de comprobación y registro nominal de solicitud/resultado/entrega. Es una capacidad pendiente: no se publica en las rutas actuales de `FuenteAuditoria`. La TSA HMAC de [`tsa_desarrollo.go`](../../internal/app/bootstrap/tsa_desarrollo.go) está rotulada no autoritativa; no es un sello RFC 3161 ni acredita por sí misma valor jurídico de una exportación.

Conservación por serie, resolución y bloqueo judicial se enlaza al [catálogo común de conservación](../../internal/vec/adapters/conservacion/catalogo.go) y a la duda 84. Los plazos provisionales no autorizan eliminación. La retención/rotación de registros técnicos se configura separadamente y no altera la evidencia funcional.

## 8. Contraste público y consenso de encaje

El contraste se limita a patrones de diseño. No se instalaron estos productos ni se enviaron datos o código a ellos.

| Fuente pública revisada | Patrón útil y encaje acordado |
| --- | --- |
| [AWS CloudTrail: integridad de logs](https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-log-file-validation-intro.html) | Huellas por archivo y manifiestos firmados enlazados. VEC reutiliza su cadena durable y necesita un anclaje confiable para detectar supresión del final; no adopta el servicio AWS. |
| [Kubernetes: auditoría](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/) | Nivel de metadatos con usuario, instante, recurso y verbo sin cuerpo. VEC conserva datos nominales mínimos y resultados; no guarda cargas personales para ampliar trazabilidad. |
| [OpenTelemetry: modelo de logs](https://opentelemetry.io/docs/specs/otel/logs/data-model/) | Instante de evento/observación, severidad, recurso, ámbito de emisión y contexto de traza. VEC puede enlazar correlación petición→BD→efectos en su emisor común, sin convertir la telemetría en permiso o auditoría durable del efecto. |
| [systemd: journald.conf](https://github.com/systemd/systemd/blob/main/man/journald.conf.xml) | Límites de espacio y tiempo de retención configurables. VEC separa recogida/rotación técnica de conservación de evidencia funcional. No se fijan plazos de ejemplo como obligación legal. |

El [ENS, RD 311/2022](https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191), artículo 24 y anexo II `op.exp.8`/`op.exp.9`, fundamenta registro de actividad e incidentes, revisión, protección y conservación conforme a política. No establece en esos apartados un plazo universal que pueda inventarse para todas las series. El [protocolo RFC 3161](https://www.rfc-editor.org/rfc/rfc3161) define solicitud/respuesta y token de sello de tiempo; se exige verificar el token y la confianza aplicable antes de atribuirle esa función.

El consenso con Astra publicado en el canal mantiene la autoridad común y su cadena, añade verificador/anclaje y reutiliza el recolector local de JSONL catalogado. El encaje acordado sigue pendiente de implementar por piezas. Este documento y el inventario no declaran captura universal, certificación ENS ni firma/sello legal.

## 9. Paquete verificable de desarrollo — 04/10

El manifiesto `vec.auditoria.exportacion.desarrollo.v1` liga tamaño y SHA256
exactos del archivo de cadena, cobertura, aviso de fechas históricas, referencias
de captura y acuse, fecha declarada y política/proveedores versionados. El
proveedor conserva KMS/TSA de desarrollo con un dominio de firma exclusivo de
exportación. El recibo checkpoint anterior no cambia.

El contrato nuevo está en
[`ports/auditoria_exportacion.go`](../../internal/vec/ports/auditoria_exportacion.go):

```go
type FuenteCapturaExportacionAuditoria interface {
    CapturarAuditoriaParaExportacion(context.Context) (CapturaParaExportacionAuditoria, error)
}
```

La composición vincula cada instancia de fuente a la solicitud y al contexto
nominal acreditados. El adaptador futuro deberá confirmar en una sola transacción
la autorización, el rango coherente y el registro común de captura antes de
devolver bytes. Fija la cabeza previa; el evento de captura queda fuera de su
propio rango. La aplicación valida el archivo con el verificador común y firma
tras ese COMMIT. Ausencia, fallo, cancelación o captura inválida impiden el recibo
y la salida de datos.

La CLI existente incorpora `verificar-exportacion` para comprobar un paquete,
el archivo y la raíz/pin externos. Separa firma, integridad de archivo y cadena;
siempre informa `origen_extraccion: no_acreditado`, TSA no verificada offline,
sin tiempo independiente ni firma legal. Una huella de acuse firmada no
autentica su autoridad. Las [instrucciones](../../cmd/vec-auditoria-checkpoint/README.md)
explican esos límites y la comprobación de alteraciones.

No hay emisor CLI que acepte una captura JSON libre ni adaptador de captura
activado. Faltan las acciones, proyección y ámbito propios de Aplicación,
la fachada nominal durable y la auditoría de recuperación/entrega. La consulta
común actual de CT/Bolsa no concede esos permisos; Sistemas sólo consulta
registros técnicos. Continúan pendientes la captura judicial, periodicidad
y conservación por serie.
