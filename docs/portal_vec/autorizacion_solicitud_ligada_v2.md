# Autorizacion ligada a solicitud V2

## Decision adoptada

Los efectos nuevos y durables usan un contrato V2 distinto del historico V1.
No existe conversion, proyeccion ni sustitucion automatica entre ambos. V1 se
mantiene legible para compatibilidad, pero sus servicios, PEP, evidencias y
registros rechazan decisiones V2. Los componentes V2 rechazan decisiones V1.

La decision V2 compromete mediante SHA-256 canonico versionado:

- la solicitud efectiva minimizada: identificador opaco de principal, perfil,
  accion, recurso completo y su contexto, finalidad, correlacion opaca,
  contexto de actor comprometido, vinculo de autenticacion y referencia de
  motivo catalogado;
- la referencia completa de motivo por separado: identificador y version del
  catalogo, huella del catalogo y clave de la entrada.

Estas huellas acreditan integridad estructural. No son firmas, no prueban por
si solas la procedencia del PDP o del catalogo y no deben presentarse como un
mecanismo de confidencialidad.

El esquema cerrado de la solicitud es
`vec.autorizacion.solicitud.v2.efectiva-minimizada`. No incorpora nombre
visible, correo, roles, permisos ni atributos declarados del principal. Esos
campos no intervienen en el PDP V2 y hashearlos no los anonimizaria. El metodo
y la garantia de autenticacion tampoco se duplican: ya quedan comprometidos en
el vinculo autoritativo. Si una politica futura necesitase atributos de
principal, exigiria un contrato V3 y una instantanea autoritativa, no ampliar
silenciosamente este DTO V2.

El puerto y la funcion de huella aceptan exclusivamente
`SolicitudAutorizacionLigadaV2`, una capacidad nominal opaca creada por
`NuevaSolicitudAutorizacionLigadaV2`. Su contrato de datos no contiene
`Principal` declarado ni un campo `Motivo`: identidad, perfil, metodo y
garantia se derivan del vinculo. El motor RBAC/ABAC historico recibe una
proyeccion minima creada dentro de la aplicacion; no existe conversion publica
V1/V2. Constructor y `Datos()` clonan defensivamente y bloquean serializacion,
decodificacion, formato y log estructurado.

`CorrelacionRef` usa el perfil `correlacion_` seguido de 32 digitos
hexadecimales en minuscula. Sus 128 bits se generan con `crypto/rand` en la
frontera; nunca se derivan de DNI, correo, persona, expediente ni texto
aportado por el usuario. La barrera final del servicio y la huella reutilizan
`domain.ReferenciaCorrelacionAutorizacionV2Valida`. El puerto
`GeneradorReferenciasAutorizacionV2` y su adaptador criptografico separan este
espacio del usado para claves de motivo. La futura frontera HTTP debe generar
la referencia mediante ese puerto y nunca aceptar su valor desde el cliente.

En este corte el generador ya existe, pero la fachada y el PEP V2 todavia
transportan la correlacion como `string`. Validar su forma impide introducir
un DNI o una etiqueta, pero no demuestra que el llamador haya usado el CSPRNG.
Antes de exponer V2, la correlacion se convertira en una capacidad nominal
opaca: se acuñara una sola vez en el ingreso confiable y la misma capacidad se
reutilizara durante toda la operacion. Hasta cerrar esa frontera, el flujo V2
permanece **NO-GO productivo**.

## Motivo catalogado y minimizacion

V2 no admite texto libre ni codigos de negocio legibles como motivo. El
contrato nominal solo contiene `ReferenciaMotivo`, cuya `EntradaClave` usa el
perfil especializado `motivo_` seguido de exactamente 32 digitos
hexadecimales en minuscula. Es un identificador opaco de 128 bits generado por
el servidor; la etiqueta y descripcion humanas permanecen exclusivamente en
el catalogo gobernado y no se copian en la solicitud, decision ni auditoria de
autorizacion. DNI, correo, nombre, numero de expediente u otros datos de
negocio quedan rechazados por construccion en este campo.

La forma opaca evita que un catalogo publicado por error convierta la clave en
un canal de datos personales, pero no demuestra por si sola aleatoriedad,
procedencia, existencia ni vigencia. La generacion segura y unicidad pertenecen
al servicio de gobierno del catalogo; una referencia estructural rellenada por
el llamador tampoco es autoridad suficiente.

Por ello el servicio V2 exige un `ValidadorReferenciaMotivoAutorizacionV2`. La
implementacion de catalogos:

1. limita la resolucion al catalogo de motivos configurado;
2. obtiene la version exacta y toma un clon canonico defensivo;
3. exige estado publicado y fecha de publicacion no futura;
4. coteja la huella completa del catalogo;
5. exige que la entrada exacta exista y este vigente.

Se rechazan la referencia cero, las claves que no cumplan el perfil opaco, la
huella centinela de 64 ceros y versiones no representables en `GOARCH=386`.
El catalogo es gobierno de configuracion, no un campo aportado directamente
por HTTP. Los puertos que necesiten validar la forma completa reutilizan
`domain.ReferenciaMotivoAutorizacionV2Valida`; la resolucion positiva contra el
catalogo sigue siendo obligatoria antes de evaluar o registrar.

## Registro durable

Los puertos V2 reciben una
`OrdenRegistroDecisionAutorizacionSolicitudLigadaV2` opaca, nunca una decision
aislada. La orden liga y entrega defensivamente:

- la decision V2 completa;
- la referencia de motivo exacta cuya huella figura en la decision.

El adaptador durable debe releer el catalogo dentro de la misma transaccion del
registro, cotejar version, huella, entrada y vigencia, y solo entonces insertar
la concesion o denegacion. El adaptador de memoria ya coteja la preimagen y la
conserva junto a `DecisionRef`. PostgreSQL V2 requiere una migracion y un
adaptador nuevos; el serializador y las tablas V1 no se ampliaran en silencio.

La proyeccion PostgreSQL de motivos no puede depender solo de un consumidor
asíncrono: un retraso al retirar una entrada dejaria una ventana de aceptacion.
Si catalogo maestro y autorizacion comparten base, publicacion o retirada y
proyeccion se confirmaran en la misma transaccion. Si algun dia se separan,
una retirada se hara en dos fases, invalidando primero la proyeccion de
autorizacion; alternativamente se exigira un arrendamiento breve que deniegue
al caducar. Nunca se retirara primero el maestro y se propagara despues.

## Inmutabilidad y fugas

El constructor nominal V2 clona antes de exponer la capacidad todos los mapas
y slices de recurso y contexto de actor. `Datos()` devuelve otra copia. El
vinculo de autenticacion se comparte unicamente porque es una capacidad opaca e
inmutable creada por la fabrica del dominio. Esto evita divergencias TOCTOU
entre evaluacion, huellas y registro.

Solicitud nominal, datos internos, evidencias y orden de registro bloquean
JSON, XML, Gob, binario, texto, CBOR, YAML y formateo/log estructurado. La
representacion canonica solo se entrega de forma deliberada al adaptador que
debe cotejarla.

## Compatibilidad y siguiente corte

- `VEC-AD-1` sigue siendo exclusivamente historico y rechaza V2.
- Una atestacion nueva exigira `VEC-AD-2`; no se modificara el vector V1.
- La credencial del registro no sustituye la procedencia del PDP. El uso
  productivo exige atestacion asimetrica verificable o un verificador aislado;
  una huella SHA-256 por si sola no autentica al emisor.
- La correlacion V2 debe pasar de valor validado a capacidad nominal acuñada
  por el generador confiable.
- PostgreSQL V2 debe materializar la orden nominal, la referencia de motivo y
  su revalidacion transaccional.
- La migracion de los flujos documentales V3/V4 debe seleccionar de forma
  explicita evidencias y registros V2.
