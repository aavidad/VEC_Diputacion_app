# Auditoría adversaria del flujo durable de firma de baremación V1

## Ficha de control

- Fecha de revisión: 16 de julio de 2026.
- Revisión base: `035032e`.
- Estado: **NO-GO**.
- Naturaleza: revisión adversaria de contratos nominales, aplicación, adaptador
  en memoria y documentación.
- Alcance de la conclusión: exclusivamente los siete ficheros enumerados en
  este documento.
- Exclusiones: no acredita conectores productivos, composición homologada,
  despliegue, KMS/HSM, base de datos, Autofirma ni cumplimiento del ENS.

Este informe no declara que el sistema cumpla el ENS ni otra certificación. Su
objeto es identificar qué debe cerrarse antes de someter el circuito real a
acreditación técnica y organizativa.

## Alcance exacto

La revisión se realizó sobre una instantánea limpia de `035032e` a la que se
añadieron únicamente estas rutas sin seguimiento:

1. [`flujo_firma_baremacion_durable.md`](./flujo_firma_baremacion_durable.md)
2. [`ports/flujo_firma_baremacion.go`](../../internal/modules/bolsa/ports/flujo_firma_baremacion.go)
3. [`ports/flujo_firma_baremacion_test.go`](../../internal/modules/bolsa/ports/flujo_firma_baremacion_test.go)
4. [`application/flujo_firma_baremacion_durable.go`](../../internal/modules/bolsa/application/flujo_firma_baremacion_durable.go)
5. [`application/flujo_firma_baremacion_durable_test.go`](../../internal/modules/bolsa/application/flujo_firma_baremacion_durable_test.go)
6. [`adapters/memory/flujo_firma_baremacion.go`](../../internal/modules/bolsa/adapters/memory/flujo_firma_baremacion.go)
7. [`adapters/memory/flujo_firma_baremacion_test.go`](../../internal/modules/bolsa/adapters/memory/flujo_firma_baremacion_test.go)

No se incorporaron a la instantánea los prototipos sin confirmar de
`application/baremacion.go` ni `application/baremacion_test.go`. Esa exclusión
es deliberada: uno de los objetivos era comprobar si el lote propuesto formaba
una unidad compilable y revisable por sí misma.

## Resultado ejecutivo

| Sublote | Resultado mecánico | Decisión |
| --- | --- | --- |
| Documentación | No aplica compilación | NO-GO: anticipa garantías todavía no impuestas por los contratos |
| Puertos | Pruebas, carrera y `go vet` correctos en la instantánea | NO-GO: el contrato necesita cambios de autoridad, contexto criptográfico, cercado y atestación |
| Adaptador en memoria | Pruebas, carrera y `go vet` correctos en la instantánea | NO-GO: implementa el contrato incompleto y necesita endurecimiento contra TOCTOU |
| Aplicación | No compila de forma aislada | NO-GO crítico: depende de prototipos externos y no exige autorización por acción y recurso |

El resultado verde de los puertos y del adaptador en memoria solo acredita que
esas unidades compilan y satisfacen sus pruebas actuales. No convierte su
semántica en suficiente para producción.

## Marco de decisiones aplicable

Esta auditoría aplica las siguientes decisiones transversales:

- [DEC-047 — Capacidades TCB privadas y conectores nominales](./registro_decisiones.md#dec-047--capacidades-tcb-privadas-y-conectores-nominales):
  una validación pública acredita forma y vínculo nominal, pero no autoridad,
  vigencia, completitud ni consumo. La promoción debe ocurrir en un servicio
  privado ensamblado por una raíz homologada.
- [DEC-048 — Ámbito y fórmula verificable de idempotencia de baremación](./registro_decisiones.md#dec-048--ambito-y-formula-verificable-de-idempotencia-de-baremacion):
  el índice debe derivarse de identidad autoritativa, ámbito completo y
  llaveros históricos; la clave del cliente no identifica al sujeto ni se
  conserva en claro.
- [DEC-049 — Identidad atómica y fingerprint estable frente a rotaciones](./registro_decisiones.md#dec-049--identidad-atomica-y-fingerprint-estable-frente-a-rotaciones):
  una única resolución de identidad debe alimentar productor y verificador, y
  la rotación de pruebas criptográficas no debe romper la igualdad semántica.

## Hallazgos críticos

### C-01. El lote de aplicación no es autocontenido

**Rutas:**

- `internal/modules/bolsa/application/flujo_firma_baremacion_durable.go`
- `internal/modules/bolsa/application/flujo_firma_baremacion_durable_test.go`

**Símbolos productivos ausentes de la base:**

- `FuenteSesionAutenticadaBaremacion`
- `SesionAutenticadaBaremacion`
- `SesionAutenticadaBaremacion.capacidades`
- `dependenciaBaremacionNula`
- `referenciaAplicacionBaremacionValida`
- `ErrDependenciaBaremacionRequerida`
- `ErrResultadoBaremacionNoConfiable`

**Símbolos de prueba ausentes de la base:**

- `relojBaremacionPrueba`
- `instanteBaremacionPrueba`
- `contextoYVinculoAutenticacionAplicacionPrueba`
- `sesionBaremacionAutenticacionAlternativaPrueba`
- `sesionAutenticadaBaremacionIdentidadPrueba`

Estos símbolos proceden de prototipos sin seguimiento que no pertenecen al
alcance. Incluir dichos prototipos completos para obtener una compilación verde
ampliaría de forma no controlada el lote y arrastraría decisiones todavía no
aceptadas.

**Corrección exigida:** extraer contratos y errores específicos y pequeños para
el flujo de firma. Los accesorios de prueba deben ser locales a sus pruebas.
No se debe incorporar el prototipo general como dependencia implícita.

### C-02. La fachada autentica, pero no autoriza la operación

**Ruta y símbolos:**

- `application/flujo_firma_baremacion_durable.go`
- `FachadaFirmaBaremacionDurable.derivarCredenciales`
- `FachadaFirmaBaremacionDurable.Preparar`
- `FachadaFirmaBaremacionDurable.Finalizar`

`derivarCredenciales` comprueba que existe una sesión y que su vínculo está
vigente, pero la fachada no exige una decisión de autorización exacta para la
acción, el recurso, la finalidad y el perfil activo. Si una frontera expusiera
el caso de uso tal como está, un actor meramente autenticado podría aportar
referencias de proceso, solicitud, baremación y decisión sin que la fachada
demostrara su permiso sobre esa combinación.

**Impacto:** riesgo de autorización horizontal o vertical incorrecta y de
legitimación posterior de una intención no autorizada mediante el sello del
estado.

**Corrección exigida:** inyectar el `puertosvec.Autorizador` ya adoptado, formar
una `dominiovec.SolicitudAutorizacion` exacta, verificar de forma cerrada la
decisión devuelta y registrar/consumir su evidencia según el paso. La decisión
debe ser fresca y no persistirse como capacidad reinyectable.

### C-03. El estado inicial es una carga opaca aportable por el llamador

**Ruta y símbolos:**

- `application/flujo_firma_baremacion_durable.go`
- `OrdenPrepararFlujoFirmaBaremacion.EstadoTrabajoInicial`
- `OrdenPrepararFlujoFirmaBaremacion.ProcesoRef`
- `OrdenPrepararFlujoFirmaBaremacion.SolicitudRef`
- `OrdenPrepararFlujoFirmaBaremacion.BaremacionMeritoRef`
- `OrdenPrepararFlujoFirmaBaremacion.DecisionRef`

La prohibición de aceptar capacidades desde HTTP aparece en comentarios y en
la documentación, pero el tipo exportado permite entregar una
`CargaProtegida` arbitraria. El contenido no tiene un esquema verificable que
impida incluir credenciales temporales, direcciones firmadas, cookies, tokens o
referencias cruzadas.

**Corrección exigida:** la orden externa debe contener como máximo una clave de
operación de alta entropía y una referencia opaca mínima. Un puerto interno
debe resolver de forma autoritativa la intención completa, comprobar que todas
las referencias pertenecen al mismo expediente y construir un estado tipado y
saneado dentro del servicio privado.

## Hallazgos altos

### A-01. No se reautentica ni reautoriza cada efecto

**Ruta y símbolos:**

- `application/flujo_firma_baremacion_durable.go`
- `FachadaFirmaBaremacionDurable.Finalizar`
- `FachadaFirmaBaremacionDurable.ejecutarPaso`
- `ports.SolicitudEjecutarPasoFirmaBaremacion`

`Finalizar` deriva las credenciales una vez y ejecuta después completar firma,
custodiar, retener, reservar y confirmar. La obligación de rederivar identidad
y autorización se delega a un comentario del ejecutor; no existe una capacidad
efímera ni una prueba autoritativa que el contrato obligue a comprobar.

Una sesión, asignación de perfil o permiso revocado durante un flujo largo
podría no observarse antes del siguiente efecto.

### A-02. El AEAD carece de vínculo contextual suficiente

**Rutas y símbolos:**

- `adapters/memory/flujo_firma_baremacion.go`
- `ProtectorEstadoFlujoFirmaBaremacion.ProtegerEstadoFlujoFirmaBaremacion`
- `ProtectorEstadoFlujoFirmaBaremacion.DesprotegerEstadoFlujoFirmaBaremacion`
- `aadEstadoFlujoFirma`
- `application.FachadaFirmaBaremacionDurable.Preparar`

El AAD se limita al esquema y a `claveRef`. No liga el sobre al flujo, actor,
proceso, solicitud, baremación, decisión, versión ni paso. Además, `Preparar`
cifra el estado antes de generar `FlujoRef`, por lo que el puerto actual no
puede imponer ese vínculo.

El HMAC exterior protege la representación persistida frente a alteraciones no
autorizadas, pero no sustituye la separación contextual del AEAD ni evita que
una ruta interna con capacidad de resellado legitime un sobre trasplantado.

**Corrección exigida:** generar primero la identidad del flujo y pasar al
protector una solicitud canónica con esquema, audiencia, referencia y revisión
de clave, flujo, vínculo de actor, decisión, versión y propósito. La operación
de descifrado debe reconstruir y exigir exactamente el mismo AAD.

### A-03. El cercado protege la escritura, pero no el efecto remoto

**Rutas y símbolos:**

- `ports.ArrendamientoFlujoFirmaBaremacion.SecuenciaCercado`
- `ports.SolicitudGuardarFlujoFirmaBaremacion`
- `ports.SolicitudEjecutarPasoFirmaBaremacion`
- `application.FachadaFirmaBaremacionDurable.ejecutarPaso`

El repositorio rechaza una escritura con propietario, secuencia o vencimiento
obsoletos. Sin embargo, la solicitud al ejecutor no contiene secuencia de
cercado, versión esperada ni vencimiento. Si el arrendamiento caduca durante un
efecto, el trabajador anterior puede continuar alcanzando almacenamiento,
firma o retención mientras otro trabajador adquiere un cercado superior.

**Corrección exigida:** ligar el fence y la versión a la reclamación durable del
efecto y a su recibo. El diario productivo debe impedir que un fence inferior
inicie o sustituya un resultado y debe seguir aplicando la idempotencia nativa
del proveedor cuando exista.

### A-04. El resultado del ejecutor no está atestado ni completamente ligado

**Ruta y símbolos:**

- `ports.ResultadoEjecutarPasoFirmaBaremacion.ValidarPara`
- `ports.ResultadoFinalFlujoFirmaBaremacion`
- `application.FachadaFirmaBaremacionDurable.completarPaso`

La validación liga principalmente paso y `EfectoRef`. No acredita mediante una
prueba independiente la clave idempotente, el fence, la versión de entrada, la
huella del estado de entrada, la huella del estado de salida, el proveedor ni
la autorización consumida. Tampoco se exige que la huella del punto de control
de confirmación coincida con `ResultadoFinal.HuellaResultadoSHA256`.

La fachada termina firmando como estado legítimo valores que solo han pasado
validaciones sintácticas del conector.

**Corrección exigida:** definir un recibo nominal canónico y una atestación
verificada por una frontera distinta. El recibo debe ligar solicitud completa,
entrada, salida, resultado, fence, política, instante y conector. Conforme a
[DEC-047](./registro_decisiones.md#dec-047--capacidades-tcb-privadas-y-conectores-nominales),
el tipo público seguirá siendo nominal hasta que el servicio privado lo
promueva y consuma mediante CAS.

### A-05. La atomicidad entre efecto, recibo y checkpoint no está acreditada

**Rutas y símbolos:**

- `application.FachadaFirmaBaremacionDurable.declararPaso`
- `application.FachadaFirmaBaremacionDurable.completarPaso`
- `application.FachadaFirmaBaremacionDurable.recuperarPasoPersistido`
- `ports.EjecutorPasosFirmaBaremacion`

El patrón declarar-ejecutar-completar es una base adecuada para recuperación,
pero el puerto no exige un diario durable ni una transacción que una el recibo
del efecto con el cambio de checkpoint. El documento reconoce esta limitación;
las pruebas actuales no la cierran.

Para proveedores sin idempotencia nativa no puede prometerse ejecución exacta
una vez. Debe emplearse un diario transaccional, outbox/inbox y reconciliación,
o mantener el flujo productivo cerrado.

### A-06. La idempotencia no define una rotación recuperable

**Ruta y símbolos:**

- `application.FachadaFirmaBaremacionDurable.derivarCredenciales`
- `application.FachadaFirmaBaremacionDurable.sellarPartes`
- `ports.SelladorSolicitudBaremacion`

El vínculo del actor y el índice se derivan de nuevo mediante un sellador
genérico. El contrato no fija una revisión de política, un llavero histórico ni
una búsqueda por todos los candidatos vigentes. Una rotación puede producir un
índice distinto para la misma intención y crear un flujo duplicado.

Debe aplicarse la fórmula y la resolución atómica de
[DEC-048](./registro_decisiones.md#dec-048--ambito-y-formula-verificable-de-idempotencia-de-baremacion)
y
[DEC-049](./registro_decisiones.md#dec-049--identidad-atomica-y-fingerprint-estable-frente-a-rotaciones).

### A-07. La prueba de reinicio conserva el diario en memoria

**Ruta y símbolos:**

- `application/flujo_firma_baremacion_durable_test.go`
- `TestFachadaFirmaBaremacionDurableReanudaTrasTimeoutYReinicioSinDuplicarEfectos`
- `ejecutorFlujoFirmaPrueba.resultados`

La prueba crea otra fachada y otro protector, pero reutiliza el mismo
`entorno.ejecutor` y, por tanto, el mismo mapa de resultados. Demuestra cambio
de fachada, no pérdida y restauración del diario de efectos. Un reinicio real
del ejecutor perdería esa deduplicación.

### A-08. No existe una política suficiente de propiedad y destrucción de datos

**Rutas y símbolos:**

- `ports.CargaProtegida.Revelar`
- `application.FachadaFirmaBaremacionDurable.Preparar`
- `application.FachadaFirmaBaremacionDurable.ejecutarPaso`
- `adapters/memory.ProtectorEstadoFlujoFirmaBaremacion`

El hash, el cifrado, el descifrado y la reconstrucción crean copias de texto
plano que permanecen hasta que el recolector de basura las alcance. No hay una
propiedad de un solo uso ni destrucción en éxito, error, cancelación y pánico.

La limpieza explícita reduce exposición accidental, aunque no constituye por
sí sola una frontera frente a código malicioso con los mismos privilegios. El
aislamiento fuerte corresponde al proceso privado previsto por DEC-047.

### A-09. El agregado y su DTO de persistencia pueden filtrar metadatos

**Ruta y símbolos:**

- `ports.ExpedienteFlujoFirmaBaremacion`
- `ports.DatosPersistenciaEstadoProtegidoFlujoFirmaBaremacion`
- `ports.SolicitudEjecutarPasoFirmaBaremacion`

El sobre protegido bloquea JSON, texto, binario y formateo directo, pero el
agregado y el DTO exponen referencias, HMAC completos y metadatos enlazables a
formateadores y registradores genéricos. El DTO también transporta el sobre
cifrado completo.

Debe existir redacción segura para `fmt` y `slog`, prohibición de codificadores
genéricos fuera del adaptador de persistencia y pruebas de ausencia de datos
personales o identificadores correlacionables en errores, métricas y logs.

## Hallazgos medios

### M-01. Ventana TOCTOU en el repositorio de memoria

**Ruta y símbolos:**

- `adapters/memory.RepositorioFlujosFirmaBaremacion.GuardarFlujoFirmaBaremacion`
- `adapters/memory.RepositorioFlujosFirmaBaremacion.verificarExpediente`

El repositorio verifica el sello antes de adquirir el mutex. La solicitud
contiene slices y punteros que pueden compartir memoria con el llamador. Una
mutación concurrente entre la verificación y el clon puede almacenar un estado
con sello ya obsoleto y causar corrupción detectable o denegación de servicio.

Debe realizarse un clon defensivo al entrar, verificar esa copia y no volver a
observar memoria propiedad del llamador.

### M-02. La proyección puede devolverse después de su vencimiento

**Ruta y símbolos:**

- `ports.ProyeccionLanzamientoFirmaBaremacion.Validar`
- `application.FachadaFirmaBaremacionDurable.Preparar`
- `application.FachadaFirmaBaremacionDurable.Consultar`

La validación limita la duración relativa de la proyección, pero la fachada no
comprueba que siga vigente respecto del reloj al devolverla. El futuro
resolutor interno debe aplicar sesión fresca, uso único, vencimiento y
protección anti-repetición.

### M-03. La matriz de estados es insuficiente

Los tests de puertos y memoria no cubren todas las regresiones, saltos,
reordenaciones, cruces de resultados, bordes temporales, alias mutables y
combinaciones de cercado. Una validación verde del camino feliz no acredita la
máquina durable completa.

## Separación entre contrato nominal y acreditación productiva

### Lo que puede acreditar un contrato nominal

- Forma canónica y límites de los datos.
- Orden nominal de los pasos.
- Presencia de referencias, huellas y recibos.
- Ausencia de serialización genérica en tipos expresamente protegidos.
- Rechazo local de combinaciones incoherentes.
- Semántica esperada de CAS, arrendamiento e idempotencia para adaptadores.

### Lo que no acredita un contrato nominal

- Que una identidad sea la persona autoritativa del expediente.
- Que el permiso esté vigente o corresponda al recurso exacto.
- Que el KMS/HSM haya usado las claves, dominios e históricos homologados.
- Que una fuente no haya omitido generaciones o filas de una matriz.
- Que un efecto remoto haya ocurrido una sola vez.
- Que el recibo proceda del proveedor declarado.
- Que el repositorio haya consumido capacidad, auditoría y outbox en una única
  transacción.
- Que la composición, red, secretos, operadores y despliegue cumplan el ENS.

### Acreditación productiva pendiente

El GO productivo exige ensamblar el servicio privado desde la raíz de
composición homologada, con identidad autoritativa, PDP, KMS/HSM, diario de
efectos y PostgreSQL u Oracle reales. Los adaptadores de entrada solo deben
recibir el caso de uso de mínimo privilegio; nunca repositorios, verificadores,
promotores o capacidades TCB.

Si el proceso web forma parte del modelo de amenaza, KMS, registro y trabajador
de efectos deben residir en un proceso interno separado, con identidad de
servicio, mTLS y red restringida, de acuerdo con DEC-047.

## División recomendada en cuatro commits

### Commit 1 — Puertos nominales seguros

Mensaje orientativo: `firma: definir contratos durables nominales V1`.

Contenido:

- Puertos y tests del flujo.
- Solicitud de protección AEAD con contexto canónico.
- Recibo nominal ligado a entrada, salida, fence, versión y resultado.
- Contrato de idempotencia compatible con históricos y revisión de política.
- Tipos redaccionados y cargas efímeras de un solo uso.
- Estado explícito de NO-GO productivo.

No debe incluir todavía la fachada ni afirmar autenticidad de los recibos.

### Commit 2 — Adaptador adversario en memoria

Mensaje orientativo: `firma: implementar adaptadores de memoria del contrato V1`.

Contenido:

- Repositorio con CAS, arrendamiento y cercado.
- Clon defensivo previo a cualquier verificación.
- AEAD con AAD contextual exacto.
- Diario de efectos de prueba separado de la instancia del ejecutor.
- Pruebas de carrera, expiración, trabajador obsoleto, reinicio y manipulación.

El adaptador debe declararse no productivo y no durable tras reinicio salvo que
la prueba inyecte deliberadamente un diario compartido.

### Commit 3 — Servicio de aplicación con autoridad fresca

Mensaje orientativo: `firma: coordinar saga con autoridad por efecto`.

Contenido:

- Fachada y tests de aplicación.
- `FuenteSesionAutenticadaBaremacion` sustituida por un
  `ResolutorSesionAutoritativaFlujoFirmaBaremacion` específico y pequeño.
- `SesionAutenticadaBaremacion` sustituida por una capacidad opaca específica
  del flujo, no serializable y emitida por el resolutor.
- Uso del `puertosvec.Autorizador` para cada paso y recurso.
- Fuente interna de intención que elimine `EstadoTrabajoInicial` del comando
  externo y resuelva de forma conjunta las referencias.
- Errores y validadores propios del flujo, sin depender del prototipo general.
- Fixtures de prueba locales.

`dependenciaBaremacionNula` debe reemplazarse por un helper local acotado;
`referenciaAplicacionBaremacionValida`, por constructores o tipos nominales del
puerto; y los errores genéricos, por errores específicos del flujo.

### Commit 4 — Memoria técnica y puerta productiva

Mensaje orientativo: `docs: fijar auditoría y puerta productiva de firma V1`.

Contenido:

- Este informe.
- Documento del flujo ajustado a las garantías realmente probadas.
- Matriz ejecutada y resultados reproducibles.
- Inventario de conectores productivos pendientes.
- Criterios de GO y responsables de las evidencias.

La documentación no debe anunciar un circuito productivo hasta superar la
puerta descrita a continuación.

## Matriz mínima de pruebas

| Área | Casos mínimos |
| --- | --- |
| Autoridad | Seis pasos por actor autorizado, actor denegado, perfil cambiado, sesión revocada, sesión ambigua y recurso ajeno |
| Intención | Cruce de proceso, solicitud, baremación, decisión, sujeto, acción y finalidad; inyección de token, cookie, dirección firmada y capacidad desde DTO |
| Idempotencia | Misma intención y clave; distinta persona; distinta acción; distinto contenido; colisión; históricos 1..8; rotaciones independientes; omisión autoconsistente |
| AEAD | Manipulación de nonce, cifrado, tag, clave, algoritmo y AAD; trasplante entre flujo, actor, decisión, versión y paso; rotación con descifrado histórico |
| Efectos | Caída antes y después de declarar, llamar al proveedor, registrar recibo y completar checkpoint; respuesta perdida tras confirmar |
| Reinicio | Nuevas instancias de fachada, protector y ejecutor; diario durable compartido; proceso completamente reiniciado |
| CAS y cercado | Vencimiento durante el efecto; dos nodos; fence inferior; propietario cruzado; versión cruzada; igualdad exacta en el instante de expiración |
| Recibos | Resultado A contra solicitud B; clave, fence, entrada o salida cambiados; atestación falsa; checkpoint y resultado final discordantes |
| Máquina de estados | Todas las transiciones permitidas; salto, regresión, reordenación, duplicación, omisión y tiempos no monótonos |
| PII y memoria | `fmt`, `slog`, JSON, texto y binario; errores y métricas; alias retenidos; destrucción en éxito, error, cancelación y pánico |
| Composición | Handlers sin acceso a TCB; importaciones prohibidas; `typed nil`; denegación por defecto; proceso interno separado cuando aplique |

## Criterios explícitos de GO

### GO para contratos nominales

Solo podrá declararse GO nominal cuando se cumplan conjuntamente estos puntos:

1. Cada commit propuesto es autocontenido y compila sobre su base sin depender
   de prototipos sin seguimiento.
2. Los comandos externos no contienen estado de trabajo, actor, perfil,
   autorización, sello, checkpoint, fence ni capacidad reinyectable.
3. Los tipos públicos se denominan y documentan como nominales y no pueden
   promoverse a autoridad desde handlers, CLI, MCP o tests sustituibles.
4. El AEAD liga el contexto completo y se prueba contra trasplantes.
5. El recibo nominal liga solicitud, entrada, salida, fence, versión y
   resultado, y los cruces A/B fallan cerrados.
6. La idempotencia aplica DEC-048 y DEC-049, incluidas rotación, históricos y
   una única resolución atómica de identidad.
7. La matriz local, carrera, análisis estático, redacción y guardas de
   arquitectura resultan verdes.
8. La documentación mantiene explícito el NO-GO productivo.

### GO para acreditación productiva

El GO nominal no abre producción. El GO productivo requiere además:

1. Servicio privado ensamblado exclusivamente por la raíz homologada y
   expuesto mediante un puerto de mínimo privilegio.
2. Identidad, sesión y autorización revalidadas inmediatamente antes de cada
   efecto, con decisión exacta, vigente y registrada.
3. KMS/HSM real con separación de dominios, revisión de política, llaveros
   históricos completos, audiencia y atestación verificadas de extremo a
   extremo.
4. Diario durable de efectos y repositorio PostgreSQL u Oracle con unicidad,
   CAS, fence, auditoría y outbox confirmados en las fronteras transaccionales
   declaradas.
5. Recuperación demostrada tras reinicio total y tras pérdida de respuesta en
   cada frontera; ningún doble unitario sustituye esta evidencia.
6. Ausencia comprobada de capacidades y datos personales innecesarios en HTTP,
   CLI, MCP, eventos, logs, métricas, volcados y errores.
7. Pruebas con dos nodos, arrendamiento vencido durante un efecto, rotación de
   claves, cancelación, carreras y reconciliación de huérfanos.
8. Revisión de seguridad, protección de datos, operación y sistemas sobre el
   despliegue efectivo, sin deducir cumplimiento normativo únicamente del
   código.

Mientras cualquiera de estos criterios siga pendiente, el estado del circuito
productivo continuará siendo **NO-GO**.
