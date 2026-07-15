# Ejecucion documental atestada V3

Fecha de decision: 15-07-2026.

## Estado y finalidad

El ejecutor V2 demuestra el gobierno de perfiles, la revocacion, los limites y
el aislamiento defensivo de buffers, pero permanece en sombra. Una auditoria
adversaria encontro seis condiciones que impiden conectarlo a produccion:

1. pasar un descriptor a una interfaz no prueba que el binario ejecutado sea el
   que fue homologado;
2. una salida puede ser estructuralmente valida y no representar el contenido
   neutral solicitado;
3. un SHA-256 recalculable no autentica una evidencia restaurada;
4. varias relecturas reducen el TOCTOU, pero no sustituyen una confirmacion
   atomica con cercado;
5. una referencia generada sin reserva durable puede colisionar o reutilizarse;
6. el limite del fichero no equivale al consumo total de memoria de todas sus
   copias.

El objetivo de V3 es cerrar estas condiciones antes de sustituir el generador
PDF/DOCX. No abre HTTP, CLI, MCP ni tareas de IA y no cambia el estado
productivo del portal.

El primer contrato V3 quedo congelado y paso pruebas normales, con detector de
carreras y analisis estatico el 15 de julio de 2026. Una auditoria adversaria
posterior determino que sigue siendo un contrato en sombra y no una autorizacion
de produccion. La iteracion V4 debe cerrar, como minimo:

1. eliminar constructores publicos que convierten bytes o campos coherentes en
   tipos denominados verificados sin realizar criptografia;
2. sustituir el DTO de consumo por la capacidad opaca de autorizacion reforzada,
   con coincidencia exacta de actor, accion, recurso, finalidad, ambito, sesion,
   vigencia, obligaciones y plan;
3. ligar mediante capacidades inmutables la entrada realmente leida, la salida
   realmente escrita y lo declarado en cada recibo;
4. exigir recibo atestado del almacen de cuarentena y promocion cercada;
5. convertir reconciliacion, retos, inicio y confirmacion en operaciones CAS
   frescas, versionadas, de un solo uso y recuperables tras reinicio;
6. representar y conservar resultados negativos e indeterminados sin que
   puedan conceder confirmacion;
7. acreditar la cronologia completa: recibos, generacion, firma, verificacion y
   confirmacion durable en ese orden;
8. generar todas las referencias opacas en servidor, redactar capacidades y
   normalizar tiempos a UTC con precision persistible.

Hasta superar de nuevo la auditoria, V2 y V3 permanecen desconectados de los
flujos publicos y de los adaptadores productivos.

## Frontera hexagonal abierta por V4

La remediacion V4 coloca la orquestacion neutral en
`application.EjecutorDocumentalAtestadoV4`, que depende solo de
`ports.ConectorEjecucionDocumentalAtestadaV4`. La implementacion de PostgreSQL,
incluidos `pgx`, SQL, verificacion COSE, emision HMAC y socket Unix, queda en
`internal/vec/adapters/postgres/confianzadocumental`. La inyeccion esta
preparada, pero su raiz de composicion productiva y las superficies HTTP, CLI y
MCP siguen pendientes; `cmd/vec-emisor-capacidad-v4` compone directamente el
proceso emisor y no convierte al nucleo en propietario de su transporte.

El puerto expresa la obligacion de revalidar la autorizacion y confirmar efecto,
auditoria y outbox en una unica frontera transaccional. Un adaptador Oracle
podra cumplir ese contrato sin modificar el caso de uso. Cambiar de motor no
permite degradar atomicidad, autoridad, idempotencia ni denegacion por defecto.
La frontera y el adaptador PostgreSQL han superado su validacion tecnica limpia;
la homologacion de un adaptador Oracle exigira repetir el mismo contrato.

## Principio de confianza

La aplicacion no recibe renderizadores o verificadores ejecutables. Recibe un
despachador de componentes atestados, que es transporte hacia cargas de trabajo
aisladas. El despachador selecciona el trabajador exclusivamente por el
descriptor exacto y nunca por una ruta, comando, imagen o URL aportados por el
catalogo o el usuario.

Cada trabajador devuelve un recibo COSE Sign1 o envoltorio criptografico
equivalente. Un verificador independiente comprueba firma, cadena de confianza,
vigencia, revocacion, identidad de carga, clave y correspondencia exacta con el
reto y el plan. La aplicacion solo acepta el recibo verificado y tipado.

Los tres trabajos son distintos:

- renderizador: produce la representacion;
- validador estructural: comprueba formato, perfil y reglas tecnicas;
- verificador semantico: demuestra que la representacion corresponde al
  contenido neutral y no a otro documento valido.

Los tres deben diferir por parejas en rol, descriptor, artefacto, homologacion,
dominio de aislamiento, carga de trabajo, proceso y clave de firma. Un mismo
objeto Go, proceso, binario, clave o dominio con nombres distintos no satisface
la independencia.

En una primera instalacion local pueden ser tres procesos o contenedores con
usuarios, redes, certificados y claves diferentes. El contrato permite moverlos
despues a otros servidores o a nube privada sin modificar el nucleo.

## Reserva, cercado e idempotencia

Antes de ejecutar un componente remoto se crea un manifiesto que compromete:

- consulta, perfil y revision exactos;
- publicacion y secuencia operativa vigente;
- digests de todos los componentes;
- HMAC del contenido neutral;
- limites y politica;
- plan, finalidad, recurso y decision de autorizacion.

El registro durable reserva de forma unica `ReservaRef`, `BorradorRef` y
`EfectoRef`. Solo despues se autoriza el recurso concreto. La activacion consume
`DecisionRef` una unica vez y emite un token opaco de cercado ligado a todas las
referencias, secuencias, huellas y digests.

```text
preparada
   |
   v
activa -- marcar inicio durable --> efecto_iniciado
   |                                  |          \
   |                                  v           v
   |                             confirmada   indeterminada
   v                                               |
abandonada_sin_efecto                              v
                                            reconciliacion
```

No se libera una referencia o una decision por caducidad, abandono o estado
indeterminado. Cada transicion usa comparacion y sustitucion sobre estado,
secuencia de cercado y huella del vinculo. Un token anterior o cruzado nunca
inicia, confirma o reconcilia otro efecto.

Si hay timeout, cancelacion o respuesta ambigua despues de marcar el inicio, no
se repite el renderizado. Se consulta el efecto exacto al broker:

- `aplicado_exacto`: puede confirmarse;
- `no_aplicado_atestado`: puede abandonarse de forma controlada;
- `desconocido` o `conflictivo`: permanece bloqueado para intervencion y
  reconciliacion.

El conector PostgreSQL V4 debe implementar las restricciones unicas sobre
idempotencia, borrador, efecto y decision. Confirmacion, evidencia, auditoria y
outbox se escriben en una sola transaccion. Oracle podra implementar
`ports.ConectorEjecucionDocumentalAtestadaV4` sin cambiar el caso de uso.

## Evidencia autenticada

La evidencia final compromete manifiesto, reserva, efecto, cercado, perfil,
publicacion, componentes, recibos, HMAC de entrada, objeto de salida, conector,
version, SHA-256, tamano y cronologia.

Se firma o protege con MAC mediante una clave exclusiva de evidencia gestionada
por KMS/HSM o por un gestor de secretos homologado. No se reutilizan las claves
de datos, idempotencia, seudonimizacion o auditoria. La restauracion es un caso
de uso con verificador criptografico; no existe restauracion productiva que se
limite a recalcular un SHA-256 publico.

El SHA-256 se conserva para integridad y direccionamiento por contenido. No se
presenta como prueba de autoria ni sustituye firma, sello, sello de tiempo,
registro, CSV, custodia o auditoria.

## Flujo y almacenamiento temporal

El corte productivo no acumulara el documento completo en varios segmentos de
memoria. Se
añadira un puerto de sesion temporal sobre el almacen de objetos existente, sin
debilitar `AlmacenObjetos`:

1. reservar una escritura temporal cifrada en zona de cuarentena;
2. entregar al trabajador una capacidad opaca, breve, de un solo objeto y una
   sola operacion;
3. escribir por flujo mientras el adaptador calcula tamano y SHA-256;
4. congelar la sesion y devolver referencia/version/hash/tamano;
5. abrir lecturas independientes, acotadas y de un solo uso para validadores;
6. promover atomica e idempotentemente el objeto solo tras la confirmacion V3;
7. expurgar temporales abandonados mediante TTL y reconciliacion auditable.

Un almacen S3 compatible local, filesystem cifrado o gestor documental puede
implementar el puerto. No se exponen bucket, ruta, credencial o URL al dominio.
Las capacidades temporales solo funcionan en la red interna de componentes,
con TLS mutuo, caducidad breve y sin permiso de listado.

La entrada canonica se sella por flujo y tiene un techo propio. Se configuran
ademas limite de salida, expansion de contenedores, memoria por trabajador,
cuota por principal, concurrencia global y tiempo de CPU. Los parsers que
necesiten acceso aleatorio usan temporal cifrado y acotado dentro del proceso
aislado, no memoria ilimitada del servidor web.

## Herramientas reutilizables

No se implementaran criptografia, almacenamiento distribuido ni identidad de
carga desde cero. Los adaptadores podran apoyarse, tras homologacion, en:

- PostgreSQL para unicidad, cercado, transacciones y outbox;
- Ceph RGW u otro S3 compatible mantenido para objetos cifrados, versionados y
  con retencion;
- [OpenBao](https://openbao.org/docs/), HSM o KMS institucional para claves y
  firmas;
- [SPIFFE/SPIRE](https://spiffe.io/docs/latest/spire-about/spire-concepts/) o
  PKI corporativa para identidad de cargas y TLS mutuo;
- bibliotecas mantenidas de COSE y validadores especificos de cada formato.

Para COSE en Go se preselecciona
[Veraison go-cose](https://github.com/veraison/go-cose), proyecto mantenido que
implementa COSE Sign1, incluye pruebas de conformidad y fuzzing y publica sus
revisiones de seguridad. Se encapsulara dentro del verificador local; no se
expondra como dependencia del dominio ni se aceptaran algoritmos por el mero
hecho de que la biblioteca los soporte.

La eleccion de producto pertenece al despliegue. El nucleo solo conoce puertos,
capacidades, recibos y evidencias.

### Puertas especificas del conector PostgreSQL V4

El adaptador validado acredita, con privilegios efectivos:

- revocacion de `USAGE` sobre tipos y arrays actuales, privilegios por defecto y
  guarda DDL para tipos fila implicitos de objetos futuros;
- fallo previo de `roles_down.sql` si existe cualquier miembro fuera de la
  lista positiva esperada, sin revocacion silenciosa;
- inventario de `pgcrypto`, retirada de ejecucion general y acceso solo a la
  operacion HMAC minima mediante el envoltorio propietario;
- prohibicion incondicional por defecto del `down`, retirada con `RESTRICT` y
  rechazo de dependencias externas u objetos futuros no soportados.

El producto sigue **NO-GO** aunque estas puertas tecnicas terminen correctamente
mientras no haya credenciales segregadas, ACL del directorio/socket, claves bajo
HSM/KMS o gestor homologado y operacion, copia y restauracion aprobadas.

### Revision de productos a fecha de decision

[MinIO](https://github.com/minio/minio) no se incorpora a una instalacion nueva:
su repositorio oficial fue archivado el 25 de abril de 2026 y la edicion
comunitaria quedo sin desarrollo activo. Puede seguir siendo un adaptador de
migracion para instalaciones existentes, nunca una dependencia del nucleo ni
la propuesta predeterminada.

La preseleccion para prueba de concepto queda asi:

- [Ceph Object Gateway](https://docs.ceph.com/en/latest/radosgw/) es la opcion
  madura para una plataforma con recursos y operacion de cluster suficientes;
- [SeaweedFS](https://github.com/seaweedfs/seaweedfs) y
  [Garage](https://garagehq.deuxfleurs.fr/) son candidatos mas ligeros, sujetos
  a pruebas de compatibilidad, cifrado, versionado, retencion, bloqueo de objeto,
  recuperacion y soporte antes de cualquier decision;
- RustFS continua en beta en 2026 y no se propone para documentos sensibles
  mientras sus funciones distribuidas, ciclo de vida y KMS sigan incompletas.

La decision final exige matriz ENS, prueba de restauracion, actualizaciones de
seguridad, licencia, comunidad/mantenimiento y capacidad real del equipo de
sistemas. Si ninguna opcion ligera supera la homologacion, la primera fase usa
filesystem cifrado y segregado detras del mismo puerto hasta disponer de Ceph o
del servicio corporativo de objetos.

## Puertas de aceptacion

- cien preparaciones concurrentes iguales producen una reserva y un efecto;
- reutilizar la clave con otro plan o reutilizar `DecisionRef` se deniega;
- token obsoleto, cruzado o de otra secuencia no alcanza ningun componente;
- una revocacion anterior al inicio impide ejecutar;
- una respuesta perdida se reconcilia sin segundo efecto;
- reto, rol, proceso, clave, digest, hash o resultado alterados invalidan el
  recibo;
- el mismo trabajador o clave en dos funciones se rechaza;
- una salida valida pero semanticamente distinta se rechaza;
- evidencia sin firma/MAC o con cualquier campo manipulado no se restaura;
- los limites de memoria, flujo, expansion, concurrencia y tiempo se prueban con
  adaptadores hostiles;
- sin registro durable, broker, verificador criptografico, reconciliador,
  almacenamiento temporal y claves separadas, el ensamblado falla cerrado.
- sin la separacion hexagonal comprobada, la matriz SQL anterior y la
  segregacion operacional real del emisor/ejecutor, el ensamblado V4 falla
  cerrado.

La separacion hexagonal y la matriz SQL V4 estan validadas. Las restantes
puertas describen el trabajo futuro del flujo documental completo y mantienen
el NO-GO productivo.
