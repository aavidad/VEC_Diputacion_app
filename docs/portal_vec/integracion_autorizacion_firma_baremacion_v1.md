# Integración de autorización en la firma de baremaciones V1

## Ficha de control

- Fecha: 16 de julio de 2026.
- Revisión base estudiada: `b6fdc5d`.
- Estado: diseño adoptable; **NO-GO productivo**.
- Ámbito: autorización fresca por efecto en la futura saga durable de firma de
  baremaciones.
- Exclusiones: este documento no acredita conectores productivos, despliegue,
  KMS/HSM, PostgreSQL u Oracle, Autofirma, cumplimiento del ENS ni certificación
  jurídica u organizativa alguna.

Este documento fija cómo reutilizar el sistema transversal de autorización sin
convertir una decisión, una huella o un DTO en autoridad. La puerta productiva
permanecerá cerrada hasta probar la composición real y el consumo transaccional
descritos en los criterios de aceptación.

## 1. Resultado ejecutivo

El puerto transversal `puertosvec.Autorizador` debe reutilizarse sin ampliar su
interfaz. Es el PDP que evalúa una solicitud exacta y devuelve una decisión
breve, pero no ejecuta ni confirma el efecto de negocio:

```go
type Autorizador interface {
	Exigir(
		context.Context,
		domain.SolicitudAutorizacion,
	) (domain.DecisionAutorizacion, error)
}
```

No basta con inyectarlo directamente en una fachada de firma. La integración
segura requiere un servicio privado que, para cada efecto:

1. relea el flujo y resuelva una única intención autoritativa;
2. resuelva de nuevo el actor y revalide la sesión;
3. construya el recurso exacto desde datos del servidor;
4. solicite una decisión nueva al `Autorizador`;
5. coteje positivamente todos los campos de la decisión;
6. produzca evidencia y atestación verificables;
7. consuma la autorización mediante CAS en la misma frontera que confirma el
   checkpoint, la auditoría y el outbox.

La decisión nunca se conserva para autorizar otro paso. HTTP, CLI y MCP solo
reciben el caso de uso de mínimo privilegio y resultados minimizados.

## 2. Piezas existentes reutilizables

| Ruta | Símbolos principales | Uso permitido y límite |
| --- | --- | --- |
| `internal/vec/ports/autorizacion.go` | `Autorizador`, `FuenteAutorizacion`, `RegistroDecisionesAutorizacion`, `RegistroDenegacionesAutorizacion` | Contrato transversal del PDP y registro CAS. No consume por sí mismo la decisión con el efecto. |
| `internal/vec/domain/autorizacion.go` | `RecursoAutorizable`, `SolicitudAutorizacion`, `DecisionAutorizacion` | Vocabulario canónico de actor, acción, recurso, finalidad, correlación y evidencia de política. `DecisionAutorizacion` es evidencia breve, no permiso permanente. |
| `internal/vec/application/autorizacion.go` | `ServicioAutorizacion`, `ServicioAutorizacion.Exigir` | Implementación RBAC seguida de restricciones ABAC, denegación por defecto y registro de concesiones con revalidación de instantánea. |
| `internal/vec/ports/evidencia_uso_autorizacion.go` | `EvidenciaUsoDecisionAutorizacion`, `NuevaEvidenciaUsoDecisionAutorizacion`, `ValidarEn` | Capacidad opaca para trasladar una decisión al adaptador duradero. Su huella no es una firma y no demuestra procedencia del PDP. |
| `internal/vec/application/contexto_actor.go` | `ServicioContextoActor.Resolver` | Resuelve una única persona y perfil desde referencias opacas. Cero o varias coincidencias se deniegan. |
| `internal/vec/application/vinculo_autenticacion_actor.go` | `ServicioVinculoAutenticacionActorV1.Crear` | Revalida sesión, cuenta, superficie y garantía y crea el vínculo opaco que consume el PDP. Debe invocarse por efecto. |
| `internal/modules/bolsa/ports/llamamientos.go` | `CreadorVinculoAutenticacionActor`, `TransaccionPropuestasLlamamiento` | Modelo pequeño de revalidación y consumo de la autorización con un efecto. La futura firma no debe depender semánticamente del módulo de llamamientos; puede reutilizar el patrón o mover el contrato genérico. |
| `internal/modules/bolsa/application/llamamientos.go` | `ServicioLlamamientos.ProponerPrimerLlamamiento`, `decisionLlamamientoExacta` | Referencia actual más completa: resuelve recurso, revalida vínculo, coteja decisión, crea evidencia y la entrega a una transacción. |
| `internal/vec/application/catalogos.go` | `autorizarCatalogo`, `evidenciaUsoDecisionCatalogo` | Patrón adicional de autorización vinculada y consumo con el cambio de gobierno. |
| `internal/modules/bolsa/ports/autorizacion_baremacion.go` | `AccionOperacionBaremacion`, `ClaseRecursoOperacionBaremacion`, `ContextoOperacionBaremacion` | Catálogo cerrado de acciones, clases y campos positivos de baremación, ventana máxima de uso de 30 segundos y puentes al almacén. |
| `internal/modules/bolsa/ports/cobertura_manifiesto_baremacion.go` | `accionesManifiestoCanonicas` | Inventario canónico existente de las autorizaciones que deben aparecer en el manifiesto probatorio de una firma. |
| `internal/vec/application/atestacion_autorizacion.go` | `ServicioAtestacionesAutorizacionV1.Atestar` | Forma una atestación con cabecera y firmante fijados por composición. Sigue necesitando verificación independiente y consumo. |
| `internal/vec/ports/ejecucion_documental_atestada_v4.go` | `ConectorEjecucionDocumentalAtestadaV4` | Modelo de puerto que obliga al conector a reverificar y confirmar efecto, auditoría y outbox en una frontera transaccional. Es específico de documentos y no debe acoplarse directamente a Bolsa. |
| `internal/vec/adapters/postgres/confianzadocumental` | configuración fijada, servicio de confianza y consumo PostgreSQL V4 | Referencia para implementar DEC-047: raíces privadas, reloj interno, audiencia cerrada, COSE, revocación y fábrica de autoridad no parametrizable desde handlers. |

## 3. Alcance y límites del `Autorizador` actual

El puerto es reutilizable como PDP, pero deliberadamente no resuelve los
siguientes problemas:

- no prueba criptográficamente qué implementación emitió la decisión;
- no resuelve el recurso ni demuestra que sus atributos procedan del servidor;
- no liga por sí solo flujo, paso, efecto, fence, versión, entrada o plan;
- no impone uso único ni una decisión diferente para cada efecto;
- no consume la decisión con el cambio de negocio;
- no sustituye la relectura de sesión, perfil, asignación, políticas y estado en
  la frontera final;
- permite una vigencia general de hasta cinco minutos, mientras que Bolsa ya
  reduce la ventana efectiva a un máximo de treinta segundos;
- una implementación defectuosa o un doble podría devolver una decisión con
  `nil`; el caso de uso debe cotejarla completa y fallar cerrado.

### 3.1. Tratamiento del motivo

`SolicitudAutorizacion.Motivo` se valida sintácticamente, pero actualmente no
forma parte de `DecisionAutorizacion`, de su huella reforzada ni de su
atestación. Tampoco interviene en las restricciones ABAC actuales.

Por tanto, **`Motivo` no es una dimensión de seguridad mientras no quede
ligado criptográficamente**. No debe usarse para distinguir permisos, pasos,
recursos ni finalidades. Si una necesidad futura exige esa distinción, se debe
incorporar un código de motivo gobernado y su huella al recurso exacto, o
publicar una nueva versión del formato de decisión. No se incluirá texto libre
potencialmente sensible en recursos, huellas, logs o atestaciones.

## 4. Contrato privado propuesto

### 4.1. Superficie pública mínima

La frontera de entrada debe recibir únicamente referencias opacas mínimas y una
clave de operación tratada por el sistema de idempotencia semántica. No debe
aceptar:

- `DecisionAutorizacion` ni referencias de decisiones aportadas como permiso;
- `EvidenciaUsoDecisionAutorizacion` o atestaciones;
- sujeto, perfil, finalidad, clase o atributos de recurso declarados por el
  cliente;
- fence, versión, efecto, estado inicial o checkpoint elegidos por el cliente;
- URL firmada, token, cookie, credencial de objeto o capacidad KMS.

El servicio determinará el siguiente paso permitido desde el estado durable.
Si una operación interna incluye un paso esperado, este se usará únicamente
como precondición exacta y nunca para saltar la máquina de estados.

### 4.2. Dependencias privadas

El servicio de aplicación tendrá, como mínimo, estas dependencias obligatorias:

```go
type resolutorIntencionFirmaBaremacion interface {
	ResolverIntencionesFirmaBaremacion(
		context.Context,
		SolicitudResolverIntencionFirmaBaremacion,
	) ([]IntencionFirmaBaremacionNominal, error)
}

type creadorVinculoActor interface {
	Crear(
		context.Context,
		dominiovec.SolicitudRevalidacionAutenticacionActorV1,
		dominiovec.ContextoActor,
	) (dominiovec.VinculoAutenticacionActorV1, error)
}

type atestadorDecisionAutorizacion interface {
	Atestar(
		context.Context,
		dominiovec.DecisionAutorizacion,
	) (puertosvec.AtestacionAutorizacionV1, error)
}
```

También necesita `puertosvec.Autorizador`, reloj confiable, generador de
referencias, diario de efectos y un aplicador transaccional específico. La
interfaz del resolutor puede ser pública para que un adaptador hexagonal la
implemente, pero sus resultados son **nominales**: no conceden autoridad. El
servicio exige exactamente una coincidencia; no admite `LIMIT 1`, precedencia o
selección implícita.

### 4.3. Autoridad efímera

La promoción a autoridad ocurre dentro del servicio privado. El valor
comprobado no se exporta:

```go
type autoridadPasoFirmaBaremacion struct {
	decision   dominiovec.DecisionAutorizacion
	evidencia  puertosvec.EvidenciaUsoDecisionAutorizacion
	atestacion puertosvec.AtestacionAutorizacionV1
}
```

Este tipo:

- tiene valor cero inválido;
- no implementa serialización ni proyección de red;
- no se persiste, registra ni devuelve;
- se crea y consume dentro de una misma llamada;
- se descarta tanto en éxito como en error, cancelación o pánico;
- no puede reconstruirse desde referencias, huellas o recibos.

### 4.4. Aplicación transaccional

El puerto de salida del paso debe recibir una reclamación durable exacta, el
recibo nominal del efecto, la evidencia y la atestación. El adaptador
productivo debe, dentro de la misma transacción:

1. bloquear y releer flujo, checkpoint, versión y fence;
2. comprobar que el efecto está declarado y no ha sido sustituido;
3. releer la concesión registrada y la configuración vigente;
4. verificar la atestación contra raíces, suite, audiencia y revocación fijadas;
5. cotejar la decisión completa con la reclamación y el recibo;
6. consumir la decisión una sola vez;
7. confirmar checkpoint, auditoría y outbox;
8. hacer `COMMIT` o no producir ninguno de esos cambios.

Cuando el efecto remoto no pueda integrarse en la misma transacción, se usará
un diario durable con outbox/inbox, idempotencia del proveedor y
reconciliación. No se afirmará ejecución exactamente una vez sin una evidencia
material que la demuestre.

## 5. Recurso de autorización exacto

Todos los recursos usan `ModuloID = "bolsa"`. La referencia y el tipo dependen
del efecto. Los ámbitos y atributos se construyen desde la intención durable y
deben ligar, al menos:

- sujeto autoritativo mediante referencia opaca o seudónimo HMAC adecuado;
- flujo y paso esperado;
- efecto declarado;
- versión esperada y secuencia de cercado;
- huella del estado de entrada;
- huella del plan o solicitud canónica;
- índice idempotente HMAC;
- referencias de proceso, solicitud, baremación y decisión que deban pertenecer
  a la misma intención.

Los campos sensibles no se incluyen en claro. El recurso se clona antes de
autorizar y no se vuelve a leer desde memoria propiedad del llamador.

El constructor general `NuevaAutorizacionOperacionBaremacion` rechaza
correctamente atributos en los recursos no relacionados con almacenamiento.
No se debe debilitar esa regla para alojar la saga. Se creará un constructor o
contexto específico para pasos de firma con una lista exacta y cerrada de
atributos.

## 6. Mapeo de checkpoints

La finalidad de autorización es siempre la `FinalidadClave` de la decisión
técnica leída del expediente —por ejemplo,
`baremacion_proceso_selectivo`—. No se toma del handler ni se reemplaza por la
finalidad criptográfica de un sello.

| Checkpoint durable | Acción de autorización | Clase y referencia exacta | Campos positivos existentes |
| --- | --- | --- | --- |
| Preparar firma | `AccionPrepararFirmaDecisionBaremacion` | `ClaseRecursoDecision` / `DecisionRef` | `sesion_firma`, `evidencia_preparacion` |
| Completar firma | `AccionConsultarFirmaDecisionBaremacion` | `ClaseRecursoSesionFirma` / `SesionFirmaRef` | `estado_firma`, `artefacto_firma`, `evidencia_consulta` |
| Custodiar firma | `AccionCustodiarDocumentoFirmadoBaremacion` | `ClaseRecursoDocumentoFirmado` / `DocumentoFirmadoRef` | `documento_firmado.custodia`, `evidencia_custodia` |
| Retener firma | `AccionRetenerDocumentoFirmadoBaremacion` | `ClaseRecursoDocumentoFirmado` / `DocumentoFirmadoRef` | `documento_firmado.retencion`, `evidencia_retencion` |
| Reservar cambio | `AccionReservarDecisionBaremacion` | `ClaseRecursoBaremacion` / `BaremacionMeritoRef` | `reserva.decision` |
| Confirmar cambio | `AccionConfirmarDecisionBaremacion` | `ClaseRecursoBaremacion` / `BaremacionMeritoRef` | `baremacion`, `decision`, `evidencia_transaccion` |

Custodia y retención deben conservar además los vínculos técnicos ya exigidos
por `autorizacion_baremacion.go`: operación, carga, clasificación, seudónimo
HMAC del sujeto, huella HMAC de solicitud y efecto. La retención incorpora
también referencia y versión exactas del objeto.

### 6.1. Subefectos de la finalización

Los seis checkpoints son estados de recuperación, no seis permisos globales.
La finalización contiene más fronteras de privilegio y cada una exige una
decisión fresca:

| Subefecto | Acción | Clase y referencia |
| --- | --- | --- |
| Consultar el estado de la sesión | `AccionConsultarFirmaDecisionBaremacion` | `ClaseRecursoSesionFirma` / `SesionFirmaRef` |
| Validar inicialmente el artefacto | `AccionValidarFirmaDecisionBaremacion` | `ClaseRecursoArtefactoFirma` / `FirmaRef` |
| Aplicar sello de tiempo, si la política lo exige | `AccionSellarTiempoDecisionBaremacion` | `ClaseRecursoArtefactoFirma` / `FirmaRef` |
| Aumentar longevidad, si la política lo exige | `AccionAumentarFirmaDecisionBaremacion` | `ClaseRecursoArtefactoFirma` / `FirmaRef` |
| Validar el artefacto aumentado | `AccionValidarFirmaDecisionBaremacion` | `ClaseRecursoArtefactoFirma` / nueva `FirmaRef` |
| Recuperar el binario institucional | `AccionRecuperarBinarioFirmadoBaremacion` | `ClaseRecursoDocumentoFirmado` / `DocumentoFirmadoRef` |
| Custodiar el documento | `AccionCustodiarDocumentoFirmadoBaremacion` | `ClaseRecursoDocumentoFirmado` / `DocumentoFirmadoRef` |
| Aplicar retención | `AccionRetenerDocumentoFirmadoBaremacion` | `ClaseRecursoDocumentoFirmado` / `DocumentoFirmadoRef` |
| Reservar la versión de negocio | `AccionReservarDecisionBaremacion` | `ClaseRecursoBaremacion` / `BaremacionMeritoRef` |
| Confirmar agregado y evidencia | `AccionConfirmarDecisionBaremacion` | `ClaseRecursoBaremacion` / `BaremacionMeritoRef` |

No se reutiliza una decisión entre validación inicial y final, ni entre
recuperación, custodia y retención, aunque compartan actor o referencias.

## 7. Piezas que no deben reutilizarse como autoridad

- Los prototipos sin confirmar `internal/modules/bolsa/application/baremacion.go`
  y `internal/modules/bolsa/application/flujo_firma_baremacion_durable.go`.
- Una `DecisionAutorizacion` persistida y reinyectada como permiso.
- Una referencia de autorización incluida por el cliente.
- `EvidenciaUsoDecisionAutorizacion` como prueba de procedencia del PDP.
- Una atestación cuya clave, suite o audiencia haya elegido la petición.
- Constructores públicos, dobles o comprobaciones nominales como promoción a
  autoridad productiva.
- El helper privado `exigirDecisionAutorizacionVinculada` copiado fuera del
  núcleo: se reutiliza su semántica, no se duplica una implementación que pueda
  divergir.
- `ContextoOperacionBaremacion` sin un recurso de saga que ligue fence, versión
  y plan.
- Una autorización general para toda la finalización.
- `FinalidadSelloReservaBaremacion` y
  `FinalidadSelloConfirmacionBaremacion` como finalidades del PDP: son dominios
  criptográficos distintos de la finalidad administrativa.
- Un repositorio o diario en memoria como evidencia de reinicio, durabilidad o
  seguridad productiva.

## 8. Composición conforme a DEC-047

La raíz homologada construye una sola vez:

1. fuente y servicio de contexto de actor;
2. revalidador de autenticación y servicio de vínculo;
3. `ServicioAutorizacion` con fuente, registros, reloj y generador reales;
4. firmante/atestador PDP con suite, clave y audiencia fijadas;
5. resolutor autoritativo del flujo;
6. diario y transacción de efectos;
7. adaptadores de firma, almacenamiento y persistencia homologados;
8. servicio privado de firma;
9. fachada pública de mínimo privilegio.

HTTP, CLI, MCP y los módulos funcionales solo reciben la fachada final. No
reciben conectores TCB, fuentes, registros, verificadores, promotores,
repositorios ni constructores de capacidades.

Los resultados de conectores públicos se denominan y tratan como `Crudo` o
`Nominal`. Su validación acredita forma y vínculo, no autenticidad ni
autoridad. La verificación y promoción se realiza dentro del servicio privado.

Si el modelo de amenaza incluye un handler o proceso web comprometido, la
opacidad de tipos Go no es una barrera suficiente. El PDP atestador, el
verificador/registro y el trabajador de efectos deben ejecutarse en procesos
internos separados, con identidades de servicio, mTLS y red restringida. El
protocolo entre procesos sigue siendo nominal y el consumidor verifica de
nuevo su atestación.

Las guardas de arquitectura deben impedir:

- importaciones de adaptadores TCB desde handlers;
- campos `Autorizador`, KMS, repositorio o verificador en adaptadores de entrada;
- constructores de composición fuera de `bootstrap` y pruebas autorizadas;
- capacidades, decisiones o atestaciones en peticiones y respuestas;
- serialización o formateo no redactado de autoridad interna.

## 9. Matriz adversaria mínima

| Área | Pruebas obligatorias |
| --- | --- |
| Resolución | Cero, una y varias coincidencias; recurso ajeno; cruce de proceso, solicitud, sujeto, baremación y decisión; no usar `LIMIT 1`. |
| Identidad | Actor ajeno, perfil distinto, sesión ambigua, revocada o caducada; cuenta privilegiada no permitida; cambio de perfil entre pasos. |
| Frescura | Una decisión nueva por efecto; referencias de decisión distintas; caducidad exacta; reloj atrasado; revocación entre PDP y consumo. |
| Alcance | Mutación individual de acción, módulo, tipo, referencia, ámbitos, atributos, finalidad, correlación, campos y obligaciones. |
| Flujo | Cruce de flujo, paso, efecto, versión, fence, estado de entrada, plan e índice idempotente; salto, regresión, omisión y reordenación. |
| Firma | Resultado A contra solicitud B; sesión, firmante, política, artefacto o huella cruzados; sello o aumento no exigidos y exigidos ausentes. |
| Persistencia | Relectura de decisión y configuración; decisión consumida; COMMIT perdido; respuesta perdida; reinicio; outbox e inbox reconciliados. |
| Concurrencia | Dos trabajadores, fence inferior, arrendamiento vencido durante el efecto, versión competida y dos reintentos simultáneos. |
| Atestación | Firma falsa, payload cambiado, clave desconocida o revocada, suite, audiencia o entorno distintos, configuración de confianza caducada. |
| Obligaciones | Toda obligación no implementada se deniega; cumplimiento ausente, extra, duplicado o de otra decisión. |
| Motivo | Cambiar el texto de `Motivo` no puede ampliar autoridad; ningún test debe tratarlo como vínculo de seguridad mientras no exista una nueva versión ligada. |
| Cancelación | Cancelación antes y después de resolver, autorizar, atestar, declarar, ejecutar y confirmar; ningún estado parcial se legitima. |
| Redacción | `fmt`, `slog`, JSON, texto, binario, errores, métricas, auditoría y outbox sin capacidad, secreto, PII o identificador no previsto. |
| Arquitectura | AST/importaciones, dependencias nulas y nulas tipadas, handlers sin TCB y composición exclusiva desde raíz homologada. |

## 10. Criterios de aceptación

### 10.1. Aceptación del contrato nominal

El contrato podrá considerarse apto para integración interna cuando:

- exista una tabla cerrada y probada de paso, acción, clase, referencia, campos
  y atributos;
- el valor cero y toda combinación incompleta se denieguen;
- la intención se resuelva autoritativamente con cardinalidad exacta uno;
- actor y vínculo se obtengan de nuevo para cada efecto;
- la decisión se coteje por todos sus campos y se limite a treinta segundos;
- campos y obligaciones se consuman positivamente o se denieguen;
- la autoridad interna no se pueda serializar, persistir o devolver;
- cada solicitud y recibo liguen flujo, paso, efecto, versión, fence, entrada,
  salida, plan e idempotencia;
- las pruebas adversarias nominales, de carrera y de arquitectura sean verdes;
- la documentación siga declarando NO-GO productivo.

### 10.2. Puerta productiva

El GO productivo exige, de forma acumulativa:

- ensamblado real desde la fachada hasta identidad, PDP, KMS/HSM y persistencia;
- PostgreSQL u Oracle reales con relectura, uso único y CAS en la transacción;
- atestación verificable con claves, audiencia, entorno, revocación y rotación
  homologados;
- diario durable, outbox/inbox y recuperación probados tras reinicio completo;
- idempotencia o reconciliación demostrada para Autofirma, almacenamiento,
  retención y cualquier efecto remoto;
- pruebas de cancelación, caída en cada ventana, respuestas perdidas, dos nodos
  y fence obsoleto;
- evidencias de que handlers y DTO no reciben capacidades TCB;
- gestión operativa de claves, copias, restauración, observabilidad y respuesta
  a incidentes revisada por los responsables correspondientes;
- revisión independiente de seguridad y protección de datos sobre el despliegue
  concreto.

Superar pruebas unitarias o de memoria no satisface esta puerta. Este documento
no declara cumplimiento del ENS ni sustituye una evaluación formal del sistema
implantado.

## 11. Decisión de implementación

La opción recomendada es mantener `Autorizador` pequeño y estable y añadir la
orquestación específica en un servicio privado de firma. Modificar el PDP para
que conozca la saga acoplaría el núcleo transversal al módulo Bolsa y no
resolvería la procedencia, el consumo CAS ni el aislamiento frente a un handler
comprometido.

La siguiente unidad de trabajo debe limitarse a:

1. contratos nominales específicos de intención, reclamación y recibo del paso;
2. recurso exacto de saga y constructor de contexto de autorización;
3. servicio privado con decisión fresca y autoridad efímera;
4. adaptador de memoria exclusivamente adversario;
5. composición productiva y conectores reales en una fase posterior y separada.
