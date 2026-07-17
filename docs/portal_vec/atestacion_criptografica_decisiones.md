# Atestacion criptografica de decisiones de autorizacion

Estado: **remediacion tecnica V4 y representaciones canonicas VEC-AD-2 y
VEC-AD-D-1 validadas; NO-GO productivo**. El corte separa el nucleo, el conector
PostgreSQL, el emisor aislado y el consumo atomico del efecto, y ha superado su
matriz tecnica en un arbol limpio. Esto no autoriza el despliegue: una decision
registrada o una huella canonica, por si solas, nunca conceden un efecto.

Fecha de corte: 17 de julio de 2026.

Las secciones 1 a 10 conservan el modelo de amenaza y el diseno previo que dio
origen a la remediacion. Ante cualquier diferencia sobre verificacion COSE,
credenciales o consumo PostgreSQL, el corte V4 siguiente es el contrato vigente.

## Corte V4: autoridad COSE y capacidad SQL separadas

La V4 no afirma que PostgreSQL verifique COSE. Define dos autoridades tecnicas
distintas y usa una capacidad HMAC de vida corta como prueba verificable por la
base de que el proceso aislado ya valido el COSE completo. Esta frontera solo es
valida si la separacion operacional descrita aqui existe de verdad.

```text
Proceso web/ejecutor                       Proceso emisor aislado
credencial ..._ejecutor_atestado           credencial ..._emisor_capacidad
sin raiz privada ni secreto HMAC            sin repositorio ni permiso de efecto
            |                                           ^
            +-------- socket Unix local ----------------+
            |       solicitud opaca / capacidad
            v
PostgreSQL: ejecutar_plan_atestado(bytea x7, jsonb capacidad)
```

### Frontera hexagonal y composicion

El caso de uso `application.EjecutorDocumentalAtestadoV4` depende
exclusivamente de `ports.ConectorEjecucionDocumentalAtestadaV4`. El puerto
recibe la solicitud vinculada y el sobre criptografico como valores opacos y
solo devuelve una confirmacion operativa tambien opaca. No expone `pgx`, SQL,
socket, DSN, claves, raices ni repositorios al nucleo.

La primitiva comun de inspeccion y verificacion COSE pertenece a
`internal/vec/adapters/seguridad/verificacioncose`: no importa PostgreSQL, no
elige raices y nunca devuelve autoridad. La configuracion de confianza, reloj,
revocacion, emision HMAC, cliente de socket Unix y transaccion SQL permanecen
en `internal/vec/adapters/postgres/confianzadocumental`. El corte ofrece el
constructor que una raiz de composicion productiva debera inyectar en el caso
de uso, pero todavia no lo conecta a HTTP, CLI ni MCP;
`cmd/vec-emisor-capacidad-v4` si compone directamente la parte emisora del
mismo adaptador y no importa la fachada del ejecutor. Un conector Oracle futuro
implementara el mismo puerto y sus mismas obligaciones atomicas sin modificar
`application` ni el dominio. La seleccion del motor pertenece al ensamblado y
a la configuracion de despliegue.

Esta inversion no permite sustituir el conector por una implementacion menos
segura. Cualquier motor debe revalidar la autoridad y confirmar consumo,
efecto, auditoria y outbox en una unica frontera transaccional; su homologacion
y las pruebas de contrato son puertas obligatorias.

El despliegue productivo debera arrancar el proceso emisor con una cuenta LOGIN
que solo herede el rol `NOLOGIN`
`vec_ejecucion_documental_v4_emisor_capacidad`. Mediante la funcion
`obtener_material_emisor_capacidad()` cargara, bajo una misma instantanea, la
configuracion actual, las raices publicas activas y una clave HMAC versionada.
No tendra privilegios directos sobre tablas ni podra ejecutar el efecto.
Recibira por un socket Unix una solicitud opaca, volvera a verificar el COSE
Sign1 real y solo entonces emitira la capacidad.

El proceso web productivo debera arrancar con otra cuenta LOGIN que solo herede
`vec_ejecucion_documental_v4_ejecutor_atestado`. El cliente implementado es
concreto para socket Unix: no acepta desde composicion una interfaz emisora,
una clave, una raiz ni un repositorio arbitrario. La ACL SQL ya impide a esta
identidad leer confianza o el secreto HMAC y solo le permite ejecutar la
funcion atomica. Queda pendiente acreditar que el socket no se publica por TCP,
que su directorio, propietario, grupo, modo, espacio de montajes y ciclo de vida
pertenecen al proceso emisor, y que ambos procesos no comparten fichero de
entorno, cuenta de servicio, contenedor, volumen de secretos ni volcados.

El contrato de capacidad tiene exactamente 20 propiedades:

```text
esquema, clave_id, clave_version, emisor_id, audiencia, nonce,
emitida_en, expira_en, huella_metadatos_sha256, huella_payload_sha256,
huella_sobre_sha256, huella_evidencia_sha256, huella_preimagen_sha256,
huella_decision_sha256, huella_efecto_sha256, revision_confianza,
huella_configuracion_sha256, raiz_clave_id, huella_raiz_sha256,
mac_sha256
```

`esquema` vale `vec.documentos.capacidad-ejecucion.v4`, `audiencia` vale
`vec_ejecucion_documental_v4.ejecutar_plan_atestado`, `nonce` son 32 bytes
aleatorios codificados como 64 hexadecimales minusculos y la vigencia maxima es
15 segundos. `clave_version` es un numero JSON positivo. Los otros valores se
autentican en el orden anterior, excluido `mac_sha256`; cada valor se encuadra
como `longitud_en_bytes_UTF-8 + ':' + valor + LF`. El MAC es HMAC-SHA-256.

Las siete huellas comprometen los bytes exactos de metadatos, payload
`VEC-AD-1`, sobre COSE Sign1, evidencia canonica, preimagen del recurso,
decision canonica y orden de efecto. Metadatos y efecto entran en SQL como
`bytea`: la funcion comprueba limites, las siete huellas, configuracion, raiz,
clave, reloj y HMAC **antes** de ejecutar `convert_from(..., 'UTF8')::jsonb`.
Por ello una representacion JSON alternativa, un byte adicional o una
sustitucion despues de verificar COSE no conserva la autoridad.

La clave de capacidad se gobierna como versiones append-only de al menos 32
bytes. Configuracion, raices y clave HMAC poseen punteros actuales con guardas
monotonas: no admiten `DELETE`, `TRUNCATE`, retroceso ni resurreccion. Una
revision de configuracion revocada es terminal; una SPKI o secreto HMAC
revocados no reaparecen bajo otro alias. La funcion de efecto bloquea los tres
punteros actuales, por lo que una revocacion concurrente queda ordenada antes o
despues del efecto, nunca entre su validacion y escritura.

En una sola transaccion y un solo `COMMIT`, PostgreSQL:

1. autentica la capacidad y revalida su caducidad justo antes del consumo;
2. interpreta y coteja los artefactos, incluidos los 30 campos de decision y
   los 25 del vinculo autenticacion-actor;
3. revalida decision, asignacion, rol, catalogo, politicas, sesion y contexto;
4. consume de forma unica `(clave_id, clave_version, nonce)` y la decision;
5. inserta atestacion, orden documental real, auditoria encadenada y outbox.

Cualquier rechazo o conflicto revierte tambien el nonce. No quedan funciones
SQL parciales de registrar, reservar o consumir, y ninguno de los dos roles
runtime posee `SELECT`, `INSERT`, `UPDATE`, `DELETE` o `TRUNCATE` sobre tablas.
`pgcrypto` es un prerrequisito gobernado y su uso queda limitado al HMAC de esta
frontera; no se presenta como verificador de la firma COSE. La matriz tecnica
demuestra que los roles runtime no heredan ejecucion general sobre las funciones
de la extension y solo alcanzan el envoltorio cerrado necesario para operar.

### Puerta operacional obligatoria

La V4 es **NO-GO** si una cuenta, proceso, pod, contenedor, unidad de sistema,
operador runtime o gestor de secretos puede obtener simultaneamente la
credencial emisora y la ejecutora. Tambien es NO-GO si el socket admite acceso
de la identidad ejecutora fuera del protocolo cerrado, si el secreto aparece
en logs/volcados o si se concede al ejecutor la funcion de material emisor.
La encapsulacion Go no sustituye esta segregacion. El ejecutable
`deploy/postgresql/ejecucion_documental_v4/probar_integracion.sh` demuestra con
identidades LOGIN reales la denegacion de HMAC invalido, replay, concurrencia,
atomicidad, revocacion concurrente y privilegios efectivos. La apertura
productiva exige ademas
copia/restauracion, despliegue segregado, custodia HSM/KMS o gestor de secretos
homologado y revision de Sistemas y Seguridad.

### Puertas SQL y de desmontaje

La remediacion tecnica acredita conjuntamente:

1. **Tipos sin privilegios implicitos.** La migracion revoca `USAGE` a
   `PUBLIC` y runtime, configura los privilegios por defecto y una guarda DDL
   propiedad del DBA cierra tambien tipos fila implicitos de tablas futuras.
   La prueba crea una tabla posterior al `up` y consulta privilegios efectivos.
2. **Membresias cerradas al desmontar.** `roles_down.sql` obtiene la lista real
   de miembros y falla antes de cambiar estado si aparece cualquier miembro no
   previsto. Nunca revoca silenciosamente una membresia desconocida para poder
   terminar con exito.
3. **Superficie minima de `pgcrypto`.** Se inventarian las funciones y
   privilegios efectivos de la extension, se retira ejecucion general a
   `PUBLIC` cuando sea aplicable en la base dedicada y se concede solo la
   llamada minima a traves del envoltorio propietario. Cualquier cambio global
   en una base compartida requiere plan aprobado para no romper otros sistemas.
4. **Desmontaje destructivo con consentimiento y alcance cerrado.** El `down`
   exige siempre una opcion explicita, incluso sin filas. Elimina objetos
   conocidos con `RESTRICT`; una dependencia externa u objeto futuro no
   soportado aborta y conserva el estado. No usa `CASCADE`.

La validacion limpia del 15 de julio de 2026 incluye `go test ./...`, pruebas de
carrera V4, `go vet`, compilacion, `govulncheck` 1.6.0 y el ciclo completo con
PostgreSQL 18.4. El resultado tecnico verde no levanta las puertas operacionales
de credenciales, socket, secretos, copia, restauracion y aprobacion independiente.

## 1. Problema que debe cerrar

El CAS durable actual demuestra que asignacion, rol, control de vigencia y
catalogo siguen siendo los observados por el PDP. Todavia no demuestra que el
PDP concedio la tupla funcional que recibe la funcion SQL. Una credencial de
registro comprometida podria conservar las huellas vigentes y modificar
accion, recurso, contexto, finalidad, campos u obligaciones.

La solucion no es duplicar RBAC+ABAC en SQL. El PDP emitira una atestacion
criptografica de la decision completa; PostgreSQL verificara su origen e
integridad y el repositorio del efecto la consumira en su misma transaccion.

## 2. Invariantes obligatorios

- Denegacion por defecto: formato, suite, clave, firma, reloj, contexto o
  dependencia ausentes, desconocidos o ambiguos deniegan.
- La credencial `vec_autorizacion_registro` no posee ni puede usar la clave
  privada de atestacion.
- La clave privada no se exporta del HSM/KMS o servicio criptografico aprobado.
- PostgreSQL conserva exclusivamente claves publicas y su gobierno.
- Se firma la decision completa y una audiencia de despliegue; no solo su
  referencia o una huella aportada por el llamador.
- La verificacion se repite al consumir y ocurre dentro de la transaccion que
  escribe agregado, hecho, auditoria y outbox.
- Una mutacion administrativa usa la decision una sola vez. Una lectura solo
  puede reutilizarla si una politica positiva separada lo define y registra
  cada uso.
- No existe degradacion automatica a SHA-256 sin clave, HMAC local, algoritmo
  antiguo ni modo «sin verificar».
- La concesion debe quedar ligada a la autenticacion observada y a una sesion o
  asercion concreta. La garantia minima exigida no sustituye al metodo, garantia
  real, referencia y huella probatoria de autenticacion.
- La sesion se revalida contra su fuente autoritativa y se cruza con un
  `ContextoActor` ya resuelto. Ningun DTO, literal de bloque, principal, rol o
  permiso declarado puede fabricar esa capacidad.
- La cronologia exigida es `autenticacion_verificada_en <= sesion_emitida_en <=
  sesion_revalidada_en < sesion_valida_hasta`. Cuenta, metodo y garantia deben
  coincidir exactamente con el documento de actor.
- Una cuenta ordinaria declara `cuenta_ordinaria_ref == cuenta_ref`. Una cuenta
  privilegiada declara referencias distintas y solo es valida en
  `administracion_privilegiada`; esa superficie nunca acepta una cuenta
  ordinaria. Anonimo, `demo`, superficie o metodo desconocidos deniegan.

## 3. Perfil criptografico candidato

El primer perfil a evaluar por Sistemas y Seguridad es:

Esta propuesta para verificacion independiente en PostgreSQL no cambia ni
renombra retroactivamente el perfil experimental V4 descrito arriba. Si se
aprueba, su identificador, vectores y migracion se publicaran como una version
de protocolo nueva o como una compatibilidad demostrada formalmente; nunca se
reinterpretaran evidencias ya emitidas.

| Elemento | Valor candidato |
| --- | --- |
| Suite | `VEC-AD-ED25519-1` |
| Firma | Ed25519 separada, 64 bytes |
| Clave publica | 32 bytes |
| Huella de clave | SHA-256 minusculo de la clave publica |
| Clave privada | No exportable; fuera de PostgreSQL y de la credencial de registro |
| Verificacion SQL | `pgsodium.crypto_sign_verify_detached` mediante envoltorio cerrado |

`pgcrypto` no sirve como verificador de esta firma: su documentacion declara
que las funciones OpenPGP no soportan firma. PostgreSQL 18.4 y `pgcrypto` no se
deben presentar como una capacidad que no tienen:
<https://www.postgresql.org/docs/18/pgcrypto.html>.

`pgsodium` expone verificacion separada con clave publica. La version estable
que debe evaluarse a la fecha de corte es 3.1.11, publicada en PGXN; no se
instalara «latest» ni sin revision:
<https://pgxn.org/dist/pgsodium/3.1.11/>.

Este perfil es candidato, no una aprobacion criptografica. Antes de adoptarlo
se exige compatibilidad probada con PostgreSQL 18, revision de mantenimiento y
cadena de suministro, licencia, SBOM, hash del paquete, pruebas propias y
dictamen de Sistemas/Seguridad. Si el HSM corporativo o la politica
criptografica aplicable no admiten Ed25519, la puerta queda cerrada hasta
aprobar otra suite asimetrica y un verificador PostgreSQL auditado. RSA-PSS no
se declarara disponible sin una extension concreta que lo verifique.

No se usaran las funciones de generacion o firma de `pgsodium`: PostgreSQL solo
verificara con material publico. Tras cada instalacion o actualizacion se
revocara `EXECUTE` a `PUBLIC`, se inventariaran privilegios efectivos y solo el
propietario `NOLOGIN` ejecutara la funcion mediante un envoltorio cualificado y
con `search_path` cerrado.

## 4. Cabecera previa, envoltorio y representacion canonica

Antes de serializar o solicitar una firma, la configuracion confiable selecciona
una unica cabecera exacta:

```text
formato_version = 1
suite = VEC-AD-ED25519-1
clave_id = identificador opaco de la clave publica
audiencia = vec-diputacion/<entorno>/<base>/<esquema>
```

Version, suite, clave y audiencia forman parte del mensaje firmado. La firma se
anade despues y el envoltorio transporta cabecera y firma sin modificar la
cabecera. El firmante no elige ni devuelve a posteriori una suite o clave con la
que reconstruir el mensaje: recibe la seleccion exacta previa y solo puede
firmarla o fallar. Cambiar cualquiera de esos valores invalida la firma.

La audiencia es configuracion publicada y exacta; no se infiere del host
recibido ni se admite un comodin. Los tres identificadores son ASCII visible,
sin espacios, controles ni asteriscos. Que el serializador acepte un
identificador de suite no aprueba su algoritmo: las suites productivas siguen
necesitando la decision indicada en la seccion 3.

El mensaje de trabajo `VEC-AD-1` es binario, no la serializacion JSON del cliente:

1. literal ASCII
   `VEC-AUTORIZACION-ATESTACION-V1-AUTENTICACION-ACTOR` y byte cero;
2. version `uint16` big-endian;
3. suite, clave y audiencia como `uint32 longitud + UTF-8`;
4. las 31 propiedades de decision siguientes, en este orden fijo; el bloque
   anidado se expande en sus 25 datos, tambien en orden fijo;
5. fin de mensaje con longitud total `uint64`.

```text
decision_ref, concedida, codigo, principal_id, perfil_activo_ref,
accion, recurso_ref, modulo_id, tipo_recurso,
contexto_recurso_huella_sha256, finalidad, correlacion_ref,
vinculo_autenticacion_actor,
asignacion_ref, asignacion_huella_sha256, version_rol_ref,
version_rol_huella_sha256, control_vigencia_version_rol_ref,
control_vigencia_version_rol_revision,
control_vigencia_version_rol_huella_sha256,
revision_catalogo_politicas, catalogo_politicas_huella_sha256,
politicas_evaluadas_refs, politicas_evaluadas_huellas_sha256,
politicas_refs, politicas_huellas_sha256, garantia_minima,
campos_permitidos, obligaciones, emitida_en, valida_hasta
```

El bloque `vinculo_autenticacion_actor` se emite exactamente asi:

```text
bloque_version, autenticacion_ref, autenticacion_huella_sha256,
asercion_ref, sesion_ref, control_sesion_ref, control_sesion_revision,
control_sesion_huella_sha256, cuenta_ref, cuenta_ordinaria_ref,
principal_id, perfil_activo_ref, cuenta_privilegiada, superficie,
metodo_observado, garantia_observada,
politica_garantia_ref, politica_garantia_huella_sha256,
autenticacion_verificada_en, sesion_emitida_en, sesion_valida_hasta,
sesion_revalidada_en, contexto_actor_ref, contexto_actor_version,
contexto_actor_huella_sha256
```

Los 25 datos son obligatorios. `principal_id` fija la persona canonica y
`perfil_activo_ref` fija el unico perfil activo. Ambos deben coincidir byte a
byte con la decision y con el `ContextoActor`; la huella del contexto es una
barrera adicional, no un sustituto de esas igualdades explicitas.
`cuenta_privilegiada` es un booleano explicito,
no un campo omitible. Todas las referencias son opacas con gramatica cerrada,
las tres huellas de documentos son SHA-256 hexadecimal minusculo, revisiones y
versiones son positivas y los cuatro instantes son UTC a microsegundo. El valor
cero del bloque es invalido.

Codificacion de valores:

- booleano: un byte `0x00` o `0x01`;
- texto: UTF-8 valido, `uint32` de bytes y bytes exactos;
- identificador tecnico positivo: ASCII visible con gramatica cerrada, sin
  comodines, controles, marcas bidireccionales ni caracteres de ancho cero;
- entero: `uint64` big-endian positivo;
- instante: microsegundos UTC desde Unix en `int64` big-endian;
- lista: `uint32` de elementos y cada texto; las listas de conjunto deben llegar
  ya en orden ascendente estricto de bytes UTF-8, sin repetidos ni comodines
  positivos; el serializador rechaza una lista desordenada y no la corrige;
- mapa: `uint32` de pares, ordenados por bytes UTF-8 de clave, seguido de clave
  y huella SHA-256;
- longitud final: entero `uint64` big-endian igual al numero de bytes del mensaje
  completo, incluidos los ocho bytes que contienen esta propia longitud.

El mensaje completo no puede superar 512 KiB. El limite se aplica antes de cada
escritura, no despues de acumular un buffer mayor. Cada conversion a `uint32` se
comprueba antes de escribir y cualquier desbordamiento, dato no canonico o
longitud incoherente invalida el mensaje.

Dominio y serializador Go aceptan solo UTC, microsegundo exacto y anos
`0001..9999`. La frontera JSON productiva aceptara una unica forma textual
`YYYY-MM-DDTHH:MM:SS.ffffffZ`: seis decimales y `Z`, sin `24:00`, segundo 60 ni
representaciones equivalentes. PostgreSQL reconstruye el mensaje desde los
valores validados y verifica la firma; nunca confia en una huella canonica
aportada por el llamador.

La entrada SQL no puede recibir primero `jsonb`: PostgreSQL ya habria descartado
las claves JSON duplicadas que el contrato debe denegar. Recibira texto/`json`
crudo con deteccion de duplicados en raiz y mapas antes de convertir, o los
campos como parametros tipados. Campo ausente, adicional, duplicado, nulo o de
tipo distinto deniega. `pg_column_size(jsonb)` y los 512 KiB binarios son
limites independientes.

La implementacion SQL debe reproducir semantica de bytes UTF-8, no semantica de
caracteres o de la configuracion regional: longitudes con
`octet_length(convert_to(valor, 'UTF8'))` y orden/igualdad mediante `bytea` o
configuracion regional `C` explicita. Tambien se publican vectores para
`2^63-1`, `2^63` y `MaxUint64`; `int8send` no basta para la mitad superior de un
`uint64`.

La implementacion Go pura de esta fase es
`domain.SerializarMensajeAtestacionAutorizacionV1`: liga la cabecera previa y
los 31 campos de una concesion reforzada —incluidos los 25 datos del vinculo—,
y publica una huella SHA-256 para
vectores. No firma ni aprueba Ed25519, HSM, KMS o extension PostgreSQL alguna.

Las solicitudes ligadas usan una representacion nueva y no reinterpretan V1:
`domain.SerializarMensajeAtestacionAutorizacionV2`. `VEC-AD-2` emplea version
binaria 2 y el separador de dominio
`VEC-AUTORIZACION-ATESTACION-V2-SOLICITUD-LIGADA-MOTIVO-CATALOGADO`. Tras
`correlacion_ref` incorpora, en este orden, `esquema_huella_solicitud`,
`solicitud_huella_sha256`, `esquema_huella_motivo` y
`motivo_huella_sha256`. Despues de los restantes campos de la decision escribe
la referencia completa del motivo: identificador de catalogo, version como
`uint64`, huella de la instantanea publicada y clave opaca de entrada. Antes de
emitir bytes recalcula la huella de esa referencia y la coteja con la decision.

El contrato V2 congela de forma exhaustiva nombre Go, etiqueta JSON, orden y
visibilidad de los 35 campos actuales de `DecisionAutorizacion`: si el tipo
crece o cambia sin ampliar conscientemente el formato, la serializacion falla
cerrada. Conserva el techo exacto de 512 KiB y hay vectores que cubren 512
KiB menos un byte, el limite exacto y el limite mas un byte. El vector pequeno
de interoperabilidad tiene 2.326 bytes y SHA-256
`b095845f68d24df46361f110fa3dbfce82202d8021a87749ad054ef398289eab`.
Este corte solo fija bytes canonicos y su huella: todavia no firma, no verifica
procedencia, no selecciona una suite productiva y no sustituye la revalidacion
atomica del catalogo, sesion, actor y efecto.

`domain.ParsearMensajeAtestacionAutorizacionV2NoAutoritativo` aporta el lector
estricto pareado. Limita el mensaje y los conteos antes de reservar memoria,
lee los 35 campos, los 25 datos del vinculo y las cuatro coordenadas del motivo,
comprueba todos sus cruces y exige reserializacion byte a byte identica. Su
resultado no contiene `DecisionAutorizacion` ni reconstruye
`VinculoAutenticacionActorV1`: solo permite consultar cabecera nominal,
`decision_ref` y las huellas de solicitud y motivo. Formateo y codecs generales
quedan redactados o prohibidos. Superar este parser no acredita una firma.

Una denegacion no se etiqueta ni se reinterpreta como `VEC-AD-2`. El contrato
nominal `domain.SerializarMensajeAtestacionDenegacionAutorizacionV1` publica el
formato separado `VEC-AD-D-1`, con versionado y separador de dominio propios.
Conserva los mismos 35 campos cerrados y las cuatro coordenadas completas del
motivo, pero exige `concedida=false` y un codigo distinto de `concedida`.
`VEC-AD-2` sigue exigiendo exactamente el resultado contrario. Ambos comparten
solo las primitivas de escritura y la validacion estructural, de modo que un
cambio futuro de `DecisionAutorizacion` bloquea los dos formatos sin convertir
una prueba negativa en capacidad ejecutable.

`VEC-AD-D-1` mantiene el limite exacto de 512 KiB y prueba los bordes limite
menos uno, limite y limite mas uno. Su vector pequeno de interoperabilidad tiene
2.371 bytes y SHA-256
`ff44e2eeab73f9c9e1c8563d006880bf63224396b545ab94bf184da186ef0380`.
Esta huella solo acredita integridad reproducible: la procedencia de la
denegacion permanece cerrada hasta que exista el sobre y el verificador aislado.
El parser nominal separado
`domain.ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo` aplica
las mismas barreras y rechaza expresamente una concesion o el dominio VEC-AD-2.
Una denegacion temprana puede carecer aun de `garantia_minima`; si ese campo ya
existe solo admite un nivel del vocabulario gobernado.

El verificador comun `adapters/seguridad/verificacioncose` es reutilizable por
el conector PostgreSQL actual y por futuros conectores Oracle u otros. Exige
CBOR determinista, solo `alg` y `kid` protegidos, cero cabeceras no protegidas,
payload y AAD exactos, EdDSA o ES256 de lista positiva y forma low-S para
ES256. Copia sobre, `kid` y clave publica, aplica limites antes de interpretar
y bloquea formateo y codecs. Su resultado de inspeccion sigue siendo nominal:
el consumidor privado debe cotejar catalogo, audiencia, suite, entorno,
vigencia y revocacion antes de promoverlo a una capacidad efimera. Para el
perfil PDP actual ES256 continua sin estar aprobado aunque la primitiva comun
sepa verificarlo.

El formato Go `VEC-AD-1` actual ya no acepta el antiguo mensaje de 30 campos:
se cambio el separador de dominio y el vector fijo. No existe lector, fallback
ni conversion automatica desde aquel borrador. La V4 aporta revalidacion de
identidad,
verificador aislado y migracion PostgreSQL sin reinterpretar V1. Cualquier
cambio incompatible del esquema exigira una version nueva.

`VinculoAutenticacionActorV1` es opaco: otro paquete no puede rellenarlo con un
literal. Permite serializar sus datos para la firma, pero rechaza su
reconstruccion desde JSON o texto. La unica fabrica publica invoca un
`RevalidadorAutenticacionActorV1`, exige referencias de solicitud opacas y
cruza el resultado con `ContextoActor`, incluyendo cuenta, metodo, garantia,
perfil y huella del documento completo. PostgreSQL permanece deliberadamente
cerrado hasta disponer de un rehidratador que vuelva a comprobar sesion,
cuentas y documento de actor en la transaccion de registro/consumo; leer los 25
campos almacenados no reconstruye por si solo la capacidad.

Habra una sola implementacion de referencia documentada y dos verificadores
independientes: Go y PostgreSQL. No duplican el PDP; solo codifican tipos. Un
paquete de vectores binarios versionados fijara entrada, artefacto binario
independiente, SHA-256, clave
publica y firma esperada. Cualquier diferencia bloquea la entrega.

## 5. Puerto de firma y separacion de credenciales

El nucleo dependera de un puerto `FirmanteAtestacionesAutorizacion`. Su
adaptador productivo invoca un HSM/KMS o servicio local mediante mTLS y una
identidad exclusiva del PDP. El puerto recibe cabecera preseleccionada y mensaje
canonico coherentes y devuelve solo la firma —y, si el proveedor lo exige,
metadatos probatorios que deben confirmar exactamente la misma clave—. Nunca se
usa una suite o clave devuelta para reconstruir despues el mensaje ya firmado.

Los contratos Go y sus orquestadores cerrados estan implementados por version.
V1 usa `ports.FirmanteAtestacionesAutorizacionV1` y
`application.ServicioAtestacionesAutorizacionV1`. V2 no es un alias ni una
ampliacion silenciosa: usa `ports.FirmanteAtestacionesAutorizacionV2` y
`application.ServicioAtestacionesAutorizacionV2`. Su solicitud conserva una
copia del mensaje VEC-AD-2 exacto y liga por separado huella del mensaje,
decision, solicitud original y motivo catalogado. El resultado y la atestacion
son tambien tipos V2 distintos; no admiten una firma procedente de otra
solicitud. Mensaje, firma y referencias se redactan en formateo y logs, y los
codecs genericos quedan bloqueados para obligar al adaptador durable a usar un
esquema explicito.

Este contrato V2 sigue siendo nominal. El firmante devuelve evidencia opaca,
pero el servicio no verifica por si mismo COSE, procedencia institucional,
vigencia o revocacion de la clave y no crea autoridad ejecutable. El diseno V4
documental incorpora verificador aislado, catalogo durable y consumo
PostgreSQL; el perfil equivalente de confianza y consumo atomico para VEC-AD-2
permanece pendiente. Tampoco se acredita aun un adaptador HSM/KMS homologado
para el firmante PDP. Por ambas razones la puerta productiva continua cerrada.

El envoltorio durable tendra esquema cerrado: version, suite, `clave_id`,
audiencia, huella del mensaje, firma binaria, referencia opaca de operacion y
fecha informativa del proveedor. En PostgreSQL la firma sera `bytea`; en una
frontera JSON sera Base64 RFC 4648 estandar con relleno y se rechazara si
decodificar y recodificar no produce exactamente el original. El limite
estructural generico no valida criptografia: la frontera de cada suite exige su
longitud exacta —64 bytes para la candidata Ed25519— y un verificador real.
`firmada_en` no forma parte hoy del mensaje y se trata como metadato no fiable,
salvo que el proveedor la ateste expresamente.

El firmante:

- solo acepta la audiencia y suite configuradas;
- registra peticion, clave, resultado y correlacion sin datos personales;
- aplica limites, plazo breve y antirrepeticion operativa;
- nunca devuelve ni registra clave privada;
- no comparte token, certificado, proceso auxiliar ni rol PostgreSQL con el
  registrador;
- falla cerrado si el HSM no confirma la operacion.

Un firmante software con clave de fichero solo se admite en pruebas aisladas;
no puede activarse mediante una variable accidental en una imagen productiva.

## 6. Gobierno y rotacion de claves publicas

PostgreSQL tendra un catalogo append-only de versiones de clave. `clave_id` es
globalmente unico, inmutable y no puede reutilizarse nunca para otro material:

```text
clave_id, suite, clave_publica, huella_sha256, audiencia,
valida_desde, valida_hasta, estado, revision,
acto_alta_ref, acto_retirada_ref, creada_en
```

Estados permitidos: `publicada`, `retirada` y `revocada`. No se borra ni se
reactiva una version. Alta, retirada y revocacion requieren funcion, rol y acto
expresos; el registrador no puede gestionarlas.

Durante una rotacion puede verificarse la clave anterior hasta su retirada,
pero solo una clave firma decisiones nuevas. La seleccion del firmante es
configuracion explicita; nunca «la primera activa». Revocar una clave invalida
inmediatamente toda decision no consumida firmada por ella. La comprobacion usa
`clock_timestamp()` y el estado durable, no el reloj del cliente.

Las copias incluyen catalogo, actos y firmas, nunca la clave privada. La
recuperacion se considera fallida si no puede reconstruir que clave verificaba
cada efecto.

## 7. Registro y consumo atomico

`registrar_decision_si_vigente` evolucionara para recibir el documento y el
envoltorio. En la misma transaccion:

1. valida las 31 claves de decision, los 25 datos del vinculo y sus tipos;
2. exige formato, suite y audiencia exactos;
3. bloquea y revalida sesion, cuentas, control de sesion y documento de actor;
4. exige que persona y perfil coincidan con la decision y que
   `valida_hasta` no supere sesion, contexto de actor, asignacion ni politicas;
5. bloquea la version de clave y comprueba estado y vigencia;
6. reconstruye `VEC-AD-1` y verifica la firma;
7. ejecuta el CAS ya existente de asignacion, rol y catalogo;
8. inserta decision, envoltorio y huella de clave de forma append-only.

Registrar no basta. Cada repositorio PostgreSQL de negocio implementa una
operacion cerrada equivalente a:

```text
BEGIN
  validar identidad tecnica y contexto local
  bloquear decision, clave y controles vigentes
  verificar otra vez firma, audiencia, caducidad y tupla funcional esperada
  insertar consumo unico(decision_ref, operacion_ref, huella_efecto)
  escribir agregado + OCC + hecho + auditoria + outbox
COMMIT
```

No se valida en una transaccion y se escribe en otra. Un reintento busca primero
un consumo ya confirmado y devuelve exactamente su resultado sin repetir el
efecto. Solo una operacion aun no consumida revalida firma, caducidad y
revocacion antes de escribir. Una misma decision con otra operacion o huella de
efecto se deniega. Conflicto, cero filas, mas de una fila o revocacion provocan
`ROLLBACK`.

## 8. Errores, auditoria y privacidad

Los consumidores reciben codigos cerrados: `atestacion_invalida`,
`clave_no_vigente`, `audiencia_invalida`, `decision_caducada`,
`decision_consumida` o `instantanea_obsoleta`. No reciben algoritmo interno,
SQL, clave, firma, DSN, politica ni dato personal.

La auditoria conserva referencia de decision, operacion, clave, suite, huella
del mensaje, resultado y correlacion. No copia el mensaje completo a logs. Las
denegaciones necesitan su propio puerto append-only; una denegacion no se
registra como capacidad ejecutable.

## 9. Matriz minima de pruebas

- vector canonico comun Go/PostgreSQL y verificacion de firma conocida;
- mutacion individual de los 31 campos historicos de `VEC-AD-1` y de los 35
  campos de `VEC-AD-2`/`VEC-AD-D-1`, incluidos en ambos casos los 25 datos del
  vinculo autenticacion-actor;
- ausencia de contexto de actor, sesion o control; mezcla de sesion, cuenta,
  perfil, persona o superficie; `demo`, anonimo y cuenta privilegiada fuera de
  administracion;
- cronologia invertida, precision submicrosegundo, ano fuera de `0001..9999` y
  decision que sobreviva a sesion o contexto de actor;
- cambio de orden, elemento, clave o valor en cada lista y mapa;
- campo ausente, adicional, nulo, tipo distinto, duplicado o sobredimensionado;
- cambio de version, suite, clave, audiencia o firma;
- firma truncada, base64 no canonico, clave de longitud incorrecta;
- clave inexistente, futura, retirada, revocada o de otra audiencia;
- decision futura, caducada o con precision temporal no canonica;
- replay, reintento idempotente y dos consumos concurrentes;
- revocacion o cambio de catalogo entre registro y consumo;
- credencial de registro sin acceso al firmante ni al catalogo de claves;
- `PUBLIC`, fuente, registro y modulo sin ejecucion directa del verificador;
- extension ausente, version/hash inesperados o privilegios contaminados;
- copia y restauracion conservando la prueba completa.

Toda mutacion debe fallar antes del efecto. Las pruebas se ejecutan con las
identidades tecnicas reales y PostgreSQL efimero, no con el propietario.

## 10. Implantacion por fases

1. Aprobar suite, extension y modelo de amenazas con Sistemas y Seguridad.
2. Fijar dependencia, hash, SBOM, compilacion reproducible y privilegios de
   `pgsodium`; si falla cualquier comprobacion, no migrar.
3. Implementar `VEC-AD-1` en Go y PostgreSQL con vectores versionados.
4. Crear catalogo de claves y procedimientos de alta, rotacion, retirada y
   revocacion; ensayar recuperacion.
5. Implementar el puerto de firma con HSM/KMS y separar identidades.
6. Ampliar registro CAS y demostrar mutacion campo a campo.
7. Migrar un unico efecto de bajo riesgo a consumo atomico; despues extender
   por repositorio, nunca con una funcion generica que acepte SQL o nombres.
8. Superar revision de codigo, amenaza, carga, fallo, copia y operacion.
9. Solo entonces solicitar la apertura productiva mediante decision formal.

Hasta terminar el paso 9, la conducta correcta es `503` o no montar el
adaptador. La disponibilidad nunca convierte una verificacion ausente en
permiso.
