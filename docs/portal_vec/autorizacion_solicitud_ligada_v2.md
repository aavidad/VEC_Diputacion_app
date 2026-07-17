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

La correlacion V2 es `domain.ReferenciaCorrelacionAutorizacionV2`, una
capacidad nominal opaca cuyo valor cero es invalido. No existe un constructor
publico desde `string`. `GenerarReferenciaCorrelacionAutorizacionV2` invoca una
sola vez el puerto CSPRNG minimo, exige el perfil `correlacion_` seguido de 32
digitos hexadecimales en minuscula y encapsula el resultado. Rechaza contexto
nulo o cancelado, generadores nulos incluso tipados, errores del CSPRNG y
salidas fuera del perfil. La misma capacidad se reutiliza durante toda la
operacion: `Exigir` no genera ni sustituye correlaciones.

Los 128 bits se generan con `crypto/rand` en la frontera confiable; nunca se
derivan de DNI, correo, persona, expediente ni texto aportado por el usuario.
El puerto `GeneradorReferenciasAutorizacionV2` y su adaptador criptografico
separan este espacio del usado para claves de motivo. Solicitud nominal,
huella, PEP, fachada, interfaz `Exigidor` y orden de consulta interna conservan
la capacidad, no el texto. `ValorCanonico()` solo se usa al comprometer la
decision, construir auditoria o cruzar la frontera durable. JSON, XML, Gob,
binario, texto, CBOR y YAML estan bloqueados; `fmt` y `slog` siempre redactan el
valor. HTTP nunca acepta la correlacion enviada por el cliente: la acuna en el
ingreso confiable y pasa la capacidad resultante.

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

El adaptador durable aplica dos semanticas deliberadamente distintas. Para una
concesion relee el catalogo y toda la instantanea actual dentro de la misma
transaccion, coteja version, huella, entrada y vigencia, y solo entonces crea la
capacidad. Para una denegacion valida la estructura y la preimagen exacta del
motivo, comprueba su procedencia historica en el instante evaluado y la conserva
en un registro probatorio append-only, sin exigir que asignacion, rol o catalogo
sigan siendo los actuales. Una revocacion posterior no puede borrar la prueba
de que el PDP denego aquella solicitud. El adaptador de memoria ya separa ambos
caminos, coteja la preimagen y conserva tambien las referencias de motivo en
almacenes fisicamente independientes para concesiones y denegaciones, ligadas a
`DecisionRef`. PostgreSQL V2 requiere una migracion y un adaptador nuevos; el
serializador y las tablas V1 no se ampliaran en silencio.

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

Antes de reservar esas copias se ejecuta un preflight sin crecimiento de
memoria proporcional a la entrada: comprueba forma, conteos y presupuesto de
ambitos, atributos, roles y vinculos. Solo una entrada acotada se clona, y el
clon se valida de nuevo antes de usarse. La fachada y el PEP aplican el mismo
patron al recurso para que una llamada directa tampoco copie mapas fuera de
limite antes de denegar.

Solicitud nominal, datos internos, evidencias y orden de registro bloquean
JSON, XML, Gob, binario, texto, CBOR, YAML y formateo/log estructurado. La
representacion canonica solo se entrega de forma deliberada al adaptador que
debe cotejarla.

## Compatibilidad y siguiente corte

- `VEC-AD-1` sigue siendo exclusivamente historico y rechaza V2.
- `VEC-AD-2` representa exclusivamente concesiones V2. Las denegaciones usan
  el dominio y tipo nominal independiente `VEC-AD-D-1`; ninguno puede aceptar
  el resultado del otro. Ambos formatos estan implementados y probados sin
  modificar el vector V1. El sobre firmado, su verificador aislado, el gobierno
  de claves y el consumo atomico siguen siendo puertas separadas y cerradas.
- La credencial del registro no sustituye la procedencia del PDP. El uso
  productivo exige atestacion asimetrica verificable o un verificador aislado;
  una huella SHA-256 por si sola no autentica al emisor.
- PostgreSQL ya materializa y resuelve historicamente el catalogo de motivos
  mediante una identidad evaluadora minima. Aun debe materializar la orden
  nominal V2 y revalidar la vigencia actual del motivo dentro de la misma
  transaccion que registre y consuma una concesion.
- La migracion de los flujos documentales V3/V4 debe seleccionar de forma
  explicita evidencias y registros V2.
