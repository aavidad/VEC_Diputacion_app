# Pagos, tasas y conciliacion

Estado: especificacion de arquitectura aceptada; contratos de dominio y
puertos y primer caso de uso seguro para crear una orden desde una liquidacion
autoritativa, endurecidos y probados, pero **no exponibles**. Conector
corporativo, proceso remoto recuperable, persistencia productiva y pruebas con
la entidad recaudadora pendientes. La interfaz actual es una maqueta y no
acredita pagos.

Fecha de corte: 15 de julio de 2026.

## 0. Puerta de seguridad antes de habilitar pagos

La auditoria adversaria del contrato inicial ha mantenido el modulo cerrado.
No se conectara a HTTP, CLI, MCP ni a una pasarela hasta demostrar, mediante
pruebas negativas y de recuperacion, todos estos puntos:

- la orden nace de una liquidacion autoritativa, congelada y sellada; importe,
  moneda, sujeto y concepto no proceden de un alta libre;
- autorizacion ligada a accion, recurso, finalidad, sesion, garantia y campos
  positivos exactos; campo u obligacion no reconocidos deniegan;
- toda evidencia de proveedor contiene una atestacion opaca emitida por el
  verificador registrado, ligada a los bytes originales, clave, configuracion,
  conector y version;
- el origen de redireccion se obtiene del catalogo publicado y la carga POST,
  operacion y orden quedan unidas; el conector no elige un destino distinto;
- existe intencion durable previa, idempotencia remota verificable, custodia de
  notificaciones y recuperacion aun cuando falle la respuesta antes de conocer
  la referencia externa;
- cada intento de devolucion y conciliacion es una entidad hija inmutable; los
  resultados tardios se conservan y pueden bloquear, nunca se descartan;
- estado, hecho, auditoria y evento de salida usan un mapeo cerrado y se
  persisten atomicamente; el evento no puede afirmar un estado diferente;
- el mensaje probatorio original se promueve al almacen cifrado e inmutable y
  la cadena se ancla con una clave o servicio externo a la base de negocio.

La primera auditoria adversaria del nucleo ha cerrado ademas estas condiciones:

- cada accion, transicion, campo, finalidad y procedencia local o remota se
  comprueba mediante lista positiva exacta; ausencia, valor desconocido o
  combinacion no declarada deniegan;
- la atestacion especifica del cobro debe coincidir exactamente en referencia
  de autenticacion, sesion, metodo y garantia con el vinculo de autenticacion y
  actor firmado por la decision central; otra sesion del mismo sujeto tampoco
  se puede mezclar;
- referencias de persona, perfil, representacion, sesion, orden y devolucion
  son opacas y se validan sin recortar ni corregir la entrada;
- alta, peticion y devolucion usan dominios HMAC separados, y las reservas de
  alta y devolucion no son intercambiables;
- el origen publicado se recalcula y queda ligado al comando, operacion,
  orden, caducidad y tipos MIME permitidos;
- los hechos conservan decision de autorizacion, atestaciones de autenticacion
  y evidencia, y la auditoria se deriva del hecho en vez de aceptar identidad,
  accion o resultado libres;
- la idempotencia declarada por la pasarela es una capacidad obligatoria y una
  devolucion no puede reutilizar una reserva destinada a otra operacion.

Estas garantias cierran el contrato, no el despliegue. Antes de exponer una
ruta deben superarse conjuntamente seis puertas productivas:

1. identidad, PDP y atestaciones criptograficas reales en la composicion;
2. pasarela real con firma o mTLS, audiencia, proteccion contra repeticion e
   idempotencia verificadas;
3. intencion durable o saga registrada antes de todo efecto externo;
4. repositorio con unicidad, control optimista y transaccion atomica de orden,
   auditoria y outbox, incluida custodia de la entrega y notificaciones de un solo
   uso;
5. conciliacion productiva y procedimiento operativo de incidencias;
6. auditoria firmada e inmutable y claves en KMS/HSM con rotacion.

Hasta cerrar esas seis puertas, cualquier fallo ambiguo conserva estado
pendiente o incidencia. Nunca se interpreta como pago, rechazo definitivo o
devolucion segura por conveniencia operativa, y no se montan rutas HTTP, CLI o
MCP.

El corte implementado solo crea, dentro de aplicacion, una orden de pago propio
desde una unica liquidacion autoritativa exigible. Cruza actor, perfil, vinculo
V1, sesion y atestacion; exige la accion, finalidad y campos exactos y cero
obligaciones desconocidas; y entrega al repositorio una mutacion atomica de
agregado, auditoria y outbox mediante reserva idempotente. Cero o varias
liquidaciones, representacion no acreditada, estado no exigible, mezcla de
sesiones o resultado repetido discordante deniegan. No hay adaptador HTTP, CLI,
MCP, arranque ni repositorio productivo, por lo que esta pieza no abre la
pasarela.

El contrato de `ConfirmarCreacion` cierra expresamente la carrera entre la
decision del PDP y el efecto. Dentro de la misma transaccion y con el reloj de
esa transaccion debe volver a comprobar la decision V1 y su huella, su vigencia
y no revocacion, el vinculo exacto de sesion y contexto de actor, la asignacion,
la version y control de vigencia del rol, el catalogo completo de politicas y
la liquidacion autoritativa —revision, huella, estado y ventana temporal—. En
el mismo `COMMIT` consume la relacion unica `DecisionRef -> (OrdenRef,
HuellaEfectoSHA256)` y confirma orden, auditoria y outbox. Solo un reintento del
mismo efecto ya existente puede resolverse de forma idempotente; reutilizar la
decision para otra orden o mutacion deniega sin escribir.

Estas condiciones estan expresadas en el puerto y simuladas bajo un unico mutex
por el adaptador de pruebas, con casos de revocacion entre PDP y `COMMIT`. No
existe todavia un repositorio productivo que las garantice. Si la liquidacion o
cualquier control autoritativo reside en otra fuente, esta debe participar
mediante bloqueo, testigo de exclusion (`fence`) o CAS valido hasta el `COMMIT`; una segunda lectura
previa seguida de confianza en una copia no satisface el contrato.

## 1. Alcance y separacion

El nucleo tratara por separado dos problemas que no deben compartir un unico
conector generico:

- **cobros de tasas o precios publicos** a ciudadania, asociados a una
  liquidacion, solicitud o expediente;
- **pagos salientes** de nomina, dietas, devoluciones o tesoreria, que siguen
  circuitos contables, bancarios y de control interno distintos.

La primera capacidad de Bolsa sera el cobro de una tasa cuando las bases lo
exijan. La aplicacion no se convertira en adquirente ni almacenara datos de
tarjeta. Usara la pasarela corporativa, bancaria o recaudatoria que aprueben
Tesoreria, Sistemas y Seguridad mediante un adaptador intercambiable.

## 2. Regla de autoridad

El navegador nunca puede declarar que un pago esta realizado. Tampoco puede
elegir el importe, moneda, concepto, sujeto pasivo, liquidacion, bonificacion o
cuenta de destino.

La aplicacion calcula o consulta esos datos en fuentes autorizadas, congela una
orden y la envia al conector. El retorno del navegador solo sirve para mejorar
la experiencia. El pago adquiere efecto en VEC cuando:

1. llega una confirmacion servidor a servidor autenticada y no repetida, o el
   conector obtiene el mismo resultado mediante consulta;
2. coinciden orden, identificador de proveedor, importe, moneda, concepto y
   estado esperado;
3. la evidencia se persiste junto a auditoria y evento de salida;
4. la politica exige y completa, cuando proceda, la conciliacion con
   recaudacion o extracto.

Un justificante PDF subido por el interesado es una evidencia para revision,
no una confirmacion automatica de pago.

## 3. Estados y hechos inmutables

La orden conserva una proyeccion de estado, pero su historia se construye con
hechos de solo adicion. La linea base es:

```text
creada
  -> enviada_a_pasarela
  -> resultado_pendiente
  -> confirmada
  -> conciliada

alternativas: rechazada | cancelada | caducada | resultado_desconocido
posteriores: devolucion_solicitada -> devuelta -> devolucion_conciliada
```

`resultado_desconocido` no significa rechazado ni permite crear otra orden a
ciegas. Primero se consulta el identificador idempotente en el proveedor. Una
devolucion o anulacion no borra el pago: incorpora nuevos hechos, aprobaciones,
motivo y referencias externas.

Las exenciones, bonificaciones y supuestos de no sujecion son decisiones
administrativas versionadas, no pagos ficticios de importe cero.

## 4. Contrato del conector de cobro

El puerto del nucleo sera tipado y expresara capacidades, por ejemplo:

```go
type PasarelaCobro interface {
    Capacidades(context.Context) (CapacidadesPasarelaCobro, error)
    CrearOperacion(context.Context, SolicitudOperacionCobro) (InicioOperacionCobro, error)
    ConsultarOperacion(context.Context, ReferenciaOperacionCobro) (ResultadoOperacionCobro, error)
    ValidarNotificacion(context.Context, NotificacionCobro) (ResultadoOperacionCobro, error)
    SolicitarDevolucion(context.Context, SolicitudDevolucionCobro) (ResultadoDevolucionCobro, error)
    Conciliar(context.Context, SolicitudConciliacionCobro) (ResultadoConciliacionCobro, error)
}
```

No se acoplara el dominio a Redsys, a una entidad concreta, a un NRC ni a una
API determinada. El adaptador traduce el contrato aprobado. Un perfil de
capacidades permite exigir redireccion alojada, notificacion autenticada,
consulta, devolucion, conciliacion y justificante sin suponer que todos los
proveedores ofrecen lo mismo.

Los pagos salientes tendran otros puertos para Tesoreria, contabilidad, nomina
y banco. Autorizar un cobro no autoriza crear una remesa.

## 5. Datos y evidencia

La cantidad se representa con entero en la unidad menor y codigo de moneda;
nunca con `float64`. Cada orden conserva al menos:

- identificador interno opaco e idempotencia;
- expediente, solicitud, liquidacion y tarifa/version de origen;
- sujeto opaco y, si procede, representacion;
- importe exacto, moneda y concepto normalizado;
- proveedor y version del conector;
- identificador externo y referencias de operacion/conciliacion;
- instantes de creacion, caducidad, confirmacion y conciliacion;
- resultados recibidos, codigo normalizado y huellas de evidencias;
- justificante original, validacion y relacion con el expediente;
- historial de anulaciones, devoluciones e incidencias;
- autorizacion, finalidad, correlacion, auditoria y eventos.

Los objetos probatorios se guardan en el almacen documental cifrado e
intercambiable. Los secretos de autenticacion del proveedor se custodian fuera
de la base de datos de negocio y se rotan por finalidad.

No se guardaran PAN, CVV, PIN, criptogramas, cookies de la entidad ni campos
equivalentes. Los parametros sensibles tampoco apareceran en URL, analitica,
trazas, errores o soporte.

## 6. Flujo de Bolsa

```text
solicitud en borrador
 -> determinar tasa/exencion desde bases y catalogo versionados
 -> crear orden idempotente ligada al borrador
 -> redirigir a pagina alojada de la pasarela
 -> recibir retorno de usuario sin dar el pago por valido
 -> recibir/consultar resultado autenticado servidor a servidor
 -> validar coincidencias y persistir evidencia
 -> conciliar si la politica lo exige
 -> habilitar firma/presentacion
```

Si las bases permiten presentar y pagar por otro orden, ese flujo se configura
en la version de convocatoria. No se codifica como excepcion oculta. El cierre
de plazo define expresamente que ocurre con una operacion iniciada pero no
confirmada.

La solicitud presentada conserva la referencia a la version exacta del pago o
de la resolucion de exencion. Una devolucion posterior no reescribe el asiento
de presentacion y activa el procedimiento administrativo que corresponda.

## 7. Seguridad y permisos

Rige lista positiva cerrada, denegacion por defecto y minimo privilegio. Si una
capacidad no concede de forma expresa la accion, recurso, contexto, finalidad,
campos y obligaciones exactos, no existe permiso:

- el titular o representante suficiente solo inicia y consulta cobros de su
  propia solicitud;
- ningun dato procedente del navegador concede el estado `confirmada`;
- el personal de tramitacion consulta la evidencia necesaria, pero no altera
  importes ni fabrica recibos;
- conciliacion, incidencias y devoluciones son acciones distintas con roles y
  ambitos expresos;
- una devolucion o ajuste de alto impacto puede requerir dos personas
  diferentes;
- el administrador tecnico del conector no obtiene acceso funcional a todos
  los expedientes;
- cada webhook o consulta aplica autenticidad, audiencia, caducidad,
  idempotencia y defensa contra repeticion;
- origen de red por si solo no autentica una notificacion.

El adaptador se ejecutara con salida de red limitada al proveedor aprobado,
TLS autenticado cuando sea posible, secretos dedicados y limites de tasa. Una
caida o respuesta ambigua falla en cerrado: se mantiene pendiente y se
concilia, nunca se convierte en pagada por disponibilidad.

## 8. Idempotencia y consistencia

- La misma solicitud de creacion devuelve la misma orden si sus datos no han
  cambiado; reutilizar la clave con otro importe o expediente es conflicto.
- La confirmacion relee bajo la misma transaccion la decision V1 y los
  controles actuales de sesion, contexto de actor, asignacion, rol, catalogo y
  liquidacion. Caducidad, retirada, revocacion, cambio o lectura ambigua
  revierten toda la operacion.
- El consumo de autorizacion es unico por
  `DecisionRef -> (OrdenRef, HuellaEfectoSHA256)`: una repeticion exacta no
  reescribe y cualquier otro efecto deniega.
- Una notificacion repetida con el mismo contenido no duplica auditoria,
  presentaciones ni efectos contables.
- Dos resultados incompatibles abren incidencia y bloquean el avance.
- La confirmacion de estado, evidencia, auditoria y outbox es atomica.
- La llamada remota no se mantiene dentro de una transaccion de base de datos;
  se coordina como proceso recuperable con consulta y conciliacion.
- Los trabajos reintentables usan identificadores externos estables; nunca
  repiten un cobro o devolucion sin consultar antes el resultado incierto.

## 9. Seleccion del adaptador

Antes de programar un proveedor concreto se inventariara la pasarela que ya use
la Diputacion o su organismo de recaudacion y se obtendran:

- contrato tecnico y entornos de prueba;
- mecanismo de firma/autenticacion y rotacion de claves;
- estados, codigos, caducidades e idempotencia reales;
- notificacion, consulta, cierre y conciliacion;
- modelo de justificante, devolucion y anulacion;
- disponibilidad, soporte y retencion;
- requisitos de red, certificados y proteccion de datos;
- pruebas oficiales y procedimiento de alta en produccion.

Solo despues se implementara el adaptador. Cambiar de proveedor no modifica el
nucleo, pero desarrollar, certificar y conciliar el nuevo conector sigue siendo
trabajo tecnico y operativo.

## 10. Contraste con fuentes oficiales

El Observatorio de Administracion Electronica describe la
[Pasarela de Pagos de la AGE](https://dataobsae.administracionelectronica.gob.es/cmobsae3/panel/Panel.action?selectedScope=A2)
como una plataforma para habilitar el pago telematico de tasas, instalable por
el organismo o consumible como servicio centralizado. Se evaluara junto a la
solucion ya implantada por la Diputacion, sin presumir que sea aplicable a una
entidad local concreta.

La documentacion oficial de
[Redsys sobre integracion por redireccion](https://pagosonline.redsys.es/desarrolladores-inicio/documentacion-tipos-de-integracion/desarrolladores-redireccion/)
separa el retorno de la sesion del navegador de la notificacion de resultado a
la plataforma. Este patron respalda la decision de no confiar en la pagina de
retorno; no supone haber seleccionado Redsys como proveedor.

A fecha de corte, PCI SSC identifica PCI DSS 4.0.1 como version activa y
aclara que incluso los entornos redirigidos deben determinar con su adquirente
y proveedor el alcance y la validacion que les corresponde:
[version 4.0.1](https://blog.pcisecuritystandards.org/just-published-pci-dss-v4-0-1)
y
[aclaracion sobre comercio electronico redirigido](https://blog.pcisecuritystandards.org/faq-clarifies-new-saq-a-eligibility-criteria-for-e-commerce-merchants).
Por ello se prefiere pagina alojada/redireccion y ausencia de datos de tarjeta,
pero no se declarara cumplimiento PCI por arquitectura o por afirmacion del
proveedor: Tesoreria, adquirente y Seguridad deben fijar el alcance real.

## 11. Criterios de aceptacion

- Manipular importe, moneda, concepto, orden o `estado=pagada` en el navegador
  no produce ningun efecto administrativo.
- El retorno correcto del navegador sin confirmacion de servidor sigue
  pendiente.
- Una notificacion invalida, repetida o para otra audiencia falla cerrada y se
  registra sin datos sensibles.
- Un pago confirmado por importe distinto bloquea la solicitud y abre
  incidencia.
- Una caida tras cobrar se recupera consultando la misma operacion, sin segundo
  cargo.
- El recibo, la solicitud y la conciliacion se pueden cotejar por referencias y
  huellas exactas.
- Un rol de tramitacion no puede devolver ni conciliar si esas acciones no estan
  concedidas expresamente.
- Revocar la decision, la sesion, el contexto de actor, la asignacion o el rol,
  o revisar el catalogo o la liquidacion entre PDP y `COMMIT`, impide crear la
  orden y no deja agregado, auditoria, outbox ni consumo parcial.
- Una `DecisionRef` consumida no puede crear una segunda orden; el reintento
  exacto del primer efecto no vuelve a escribirlo.
- El despliegue demuestra que VEC no recibe ni conserva datos de tarjeta.
