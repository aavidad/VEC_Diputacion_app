# Capacidad documental transversal del Portal VEC

Fecha de referencia: 15-07-2026.

## Necesidad de Seleccion Externa

La peticion de RRHH exige generar automaticamente Word y PDF para contratos,
nombramientos, tomas de posesion, ceses, cambios, informes, resoluciones y
otros tipos configurables. Cada salida debe quedar unida al candidato, bolsa,
llamamiento y expediente, y formar parte del historico auditable.

PDF y DOCX son el primer corte solicitado, no una frontera del sistema. La
capacidad debe admitir JSON, XML, CSV, TXT, ODT u otro formato mediante perfiles
gobernados y conectores homologados, sin recompilar el nucleo.

Esta capacidad pertenece al nucleo. Bolsa, Personal, Nominas, Dietas y Cronos
la consumen por el mismo puerto; ningun modulo genera documentos por su cuenta.

## Decisiones de diseno

1. Word se genera como `.docx`, formato Open XML editable. No se genera el
   binario historico `.doc`: es menos interoperable, dificil de inspeccionar y
   peor base para automatizacion segura.
2. Una plantilla se identifica por `plantilla_id + version`. Una version
   publicada es inmutable. Los actos no usan implicitamente «la ultima».
3. La plantilla declara campos, formatos, permiso y nivel de garantia. Se
   rechazan campos adicionales, obligatorios ausentes y marcadores invalidos.
4. La fusion es literal: un valor aportado nunca se vuelve a interpretar como
   plantilla, XML, HTML, expresion o codigo.
5. El contenido se identifica por documento y SHA-256; el almacen devuelve una
   referencia opaca. Una URL temporal de S3 no es la identidad documental.
6. La huella de los datos de fusion es HMAC-SHA-256 con identificador de clave,
   no SHA-256 simple, para no facilitar ataques de diccionario sobre DNI u
   otros valores de baja entropia.
7. Metadatos, auditoria y evento de salida se confirman atomica y conjuntamente.
   El objeto binario se guarda antes de forma idempotente y direccionada por
   contenido; una tarea controlada puede retirar objetos huerfanos.
8. DOCX es un documento de trabajo editable. El PDF generado tampoco adquiere
   por si solo valor de acto: firma, sello de tiempo, registro, CSV y custodia
   son estados y puertos posteriores.
9. Un acto genera un unico `DocumentoLogico`. DOCX y PDF son representaciones
   tecnicas con huellas propias, pero comparten version, estado administrativo,
   plantilla exacta, relaciones y huella de fuente.
10. Toda generacion multiformato exige una clave idempotente. La reserva se
    aisla por principal y fija una huella HMAC de la peticion: un reintento
    devuelve el resultado ya confirmado y reutilizar la clave con otros datos
    se rechaza.
11. La referencia de autorizacion nunca procede de la orden del cliente. El
    servicio obtiene una decision interna RBAC+ABAC y comprueba de nuevo actor,
    perfil, accion, recurso, finalidad, correlacion, garantia y vigencia.
12. El catalogo lista plantillas solo para un modulo exacto. Modulo ausente,
    vacio, no canonico o con comodines deniega; una consulta transversal sera
    otro caso de uso autorizado, nunca el resultado de omitir el filtro.
13. Identidad de formato, perfil de representacion, revision de catalogo y
    conector instalado son conceptos distintos. Una plantilla publicada fija
    referencias y versiones exactas; MIME y extension no se infieren de una
    lista compilada ni de datos enviados por el cliente.
14. La aplicacion puede gobernar perfiles que un conector instalado ya entiende.
    Incorporar un motor, parser o capacidad criptografica nueva requiere instalar
    y homologar el conector, pero no modificar el nucleo. Ninguna fila del
    catalogo puede contener codigo, comandos, ejecutables o URL de descarga.
15. Un perfil puede incorporar metadatos institucionales de procedencia antes de
    la firma: entidad y organo, UUID documental, perfil/version, instante, URI
    de verificacion, si la politica de acceso la permite, y referencia opaca del
    manifiesto. Es una lista positiva y excluye datos personales, destinatario
    e informacion de infraestructura. El SHA-256 final nunca se incrusta en los
    mismos bytes que compromete: se calcula despues y se liga desde el manifiesto,
    auditoria y firma. Cuando el
    formato no tenga metadatos fiables se usa un manifiesto lateral firmado; no
    se ocultan marcas con espacios o caracteres invisibles.

## Ejecucion documental atestada V4

La fachada neutral `application.EjecutorDocumentalAtestadoV4` solo conoce el
puerto de salida `ports.ConectorEjecucionDocumentalAtestadaV4`. `pgx`, SQL, el
socket Unix, las claves y los repositorios quedan dentro de
`internal/vec/adapters/postgres/confianzadocumental`. El constructor del
adaptador esta disponible para que la futura raiz de composicion productiva lo
inyecte, pero este corte aun no lo expone por HTTP, CLI ni MCP; el ejecutable
`cmd/vec-emisor-capacidad-v4` compone exclusivamente el emisor. Oracle u otro
motor homologado puede implementar el mismo puerto sin modificar el nucleo.

El primer efecto PostgreSQL no genera todavia el fichero: crea una
`orden_generacion_documental` real en estado `pendiente_generacion`. La orden,
el consumo unico de decision, la atestacion, el nonce de capacidad, la auditoria
encadenada y el evento de outbox se confirman en el mismo `COMMIT`; no existe una
funcion publica que registre primero y consuma despues.

La autorizacion criptografica se divide entre un proceso emisor COSE y el
proceso ejecutor. Se comunican por socket Unix y usan cuentas distintas que
heredan, respectivamente, los roles `NOLOGIN`
`vec_ejecucion_documental_v4_emisor_capacidad` y
`vec_ejecucion_documental_v4_ejecutor_atestado`. El primero puede obtener
raices y material HMAC, pero no escribir efectos; el segundo solo puede invocar
la funcion atomica y no puede leer el secreto ni ninguna tabla. La capacidad
HMAC-SHA-256 dura como maximo 15 segundos, usa nonce aleatorio de un solo uso y
liga las huellas de los siete artefactos por sus bytes exactos.

Esta separacion es una condicion de seguridad, no una recomendacion de
despliegue. Si una identidad, proceso, contenedor o gestor de secretos reune las
dos credenciales, el modulo permanece **NO-GO**. El contrato, preimagen,
rotacion, revocacion y pruebas obligatorias se detallan en
[Atestacion criptografica de decisiones](atestacion_criptografica_decisiones.md).

La remediacion tecnica V4 ha acreditado el cierre de `TYPES` actuales y futuros,
el fallo de `roles_down.sql` ante membresias inesperadas, la superficie minima de
`pgcrypto` y el consentimiento destructivo obligatorio. El `down` usa `RESTRICT`
y conserva dependencias externas. El producto sigue NO-GO hasta acreditar en el
despliegue copia, restauracion, segregacion y auditoria operacional.

## Flujo de dentro hacia fuera

```text
Plantilla publicada exacta
        |
        v
Autorizacion + perfil + assurance + finalidad
        |
        v
Fusion estricta de datos --> HMAC de datos (sin valores en la traza)
        |
        v
Resolucion exacta de perfil/conector homologado
        |
        v
Puerto de renderizado --> adaptador de formato intercambiable
        |
        v
Metadatos institucionales permitidos --> validacion estructural
        |
        v
SHA-256 + almacenamiento opaco/S3 compatible
        |
        v
Transaccion: metadatos + auditoria + outbox
        |
        +--> revision --> firma AutoFirmaV3 --> sello de tiempo
        +--> registro --> CSV/justificante --> expediente ENI
```

## Metadatos minimos de cada generacion

- ID y version del documento.
- ID y version exacta de la plantilla.
- modulo, tipo documental y expediente.
- formato, MIME, nombre neutro y tamano.
- SHA-256 del contenido y HMAC de los datos de fusion.
- referencia opaca del contenido.
- estado documental y estado de antivirus.
- actor efectivo, perfil activo, representado cuando exista, metodo y nivel de
  autenticacion, decision de autorizacion, finalidad, motivo y correlacion.
- fecha UTC, metadatos ENI minimos, referencias de firma, registro y CSV.
- revision de catalogo, perfil inmutable, conector y digest exactos;
- marca institucional aplicada y manifiesto lateral, cuando el perfil lo exija.

Los nombres, DNI, direcciones, observaciones y demas valores fusionados no se
copian a logs, eventos ni auditoria. Permanecen en el documento protegido y en
los datos de negocio cuyo acceso este autorizado.

## Estado de implementacion

Capacidades inventariadas por los cortes anteriores del nucleo:

- dominio de plantilla versionada, campos y estados;
- gobierno de borrador y publicacion con doble control y version inmutable;
- fusion estricta y reproducible;
- formatos PDF y DOCX por puerto;
- agregado `DocumentoLogico` separado de sus representaciones DOCX/PDF;
- vinculacion tipada y extensible con persona, proceso, llamamiento, contrato,
  expediente u otras entidades de futuros modulos;
- generacion multiformato en una operacion y estado administrativo independiente
  del formato;
- reserva idempotente concurrente, conflicto por cambio de peticion, abandono,
  caducidad y devolucion del resultado confirmado;
- confirmacion atomica del documento logico, todas sus representaciones,
  auditoria encadenada y evento de outbox en el adaptador de memoria;
- autorizacion interna por puerto RBAC+ABAC; roles y permisos declarados por el
  cliente son solo informativos y no conceden acceso;
- DOCX determinista, sin macros, medios ni relaciones externas;
- PDF determinista con fuente Unicode incrustada y sin acciones activas;
- validacion defensiva de la salida DOCX/PDF antes de almacenarla y limites de
  expansion y tamano total;
- sellado HMAC de datos y SHA-256 de contenido;
- clave HMAC separada para la peticion idempotente, de modo que la rotacion de
  la clave de datos no cambie un reintento en curso;
- referencia de contenido direccionada por huella en el adaptador de memoria;
- aislamiento de datos personales y material de referencia del contexto Docker.
- contrato V2 en sombra de formatos gobernados: identidad sintactica, perfil
  historico inmutable/versionado, conformidad exacta (esquema, dialecto,
  canonicalizacion, reglas y politica), limite de bytes y estado operativo
  actual separado y revocable;
- componentes atestados por funcion —renderizado, validacion estructural,
  marcado, extraccion y verificacion semantica— con identidad, version, huella
  de artefacto, homologacion, dominio de confianza y techo propios. El caso de
  uso no devuelve interfaces ejecutables;
- requisitos de regresion extraidos del ejecutor V2 retirado: cardinalidad
  exacta, relecturas contra TOCTOU, menor techo efectivo, escritor acotado,
  aislamiento de copias y ligadura del contenido al artefacto. Su auditoria
  anadio la exigencia de evidencia autenticada y durable. V3/V4 deben satisfacer
  ambos grupos sin reintroducir aquel camino de ejecucion competidor;
- el contrato V1 de seleccion de conector y marcado permanece compilable solo
  durante la migracion, pero falla cerrado y no alcanza ejecutores;
- metadato institucional tipado, previo a firma, no personal y sin referencia
  circular. La especificacion de politica, extractor y verificador V2 esta
  definida como contrato reutilizable; su integracion en V4 sigue pendiente;
- remediacion hexagonal V4: puerto neutral, caso de uso sin dependencia de
  `pgx`, conector PostgreSQL separado, emisor compuesto desde `cmd` y resultado
  opaco. La validacion tecnica del corte esta superada; siguen pendientes su
  cableado productivo y la acreditacion operacional.

Todavia no integrado en el flujo productivo:

- persistencia y gobierno durable del catalogo, evidencias y conectores;
- broker productivo que verifique criptograficamente las atestaciones de cada
  componente y su segregacion, sin confiar en declaraciones del ejecutable;
- verificador semantico atestado, distinto tanto del renderizador como del
  validador estructural, que pruebe la equivalencia entre contenido neutral y
  representacion generada;
- reserva durable e idempotente de la referencia de borrador, anclaje
  autenticado de la evidencia y reconciliacion de efectos ambiguos;
- ejecucion V4 por flujo hacia almacenamiento acotado; no existe un ejecutor
  documental productivo y no se permite reintroducir un borrador integro en
  memoria para documentos grandes;
- cuotas por peticion y por principal, limite de concurrencia y presupuesto de
  memoria total; el limite nominal del fichero no equivale al consumo agregado
  de sus copias y serializaciones;
- adaptadores JSON, XML, CSV, TXT, ODT y demas formatos;
- migracion en sombra y posterior sustitucion del selector PDF/DOCX heredado;
- contratos gobernados de marca institucional, extraccion independiente,
  equivalencia semantica y manifiestos laterales firmados, pendientes de
  integrar en V4 con adaptadores reales por perfil;
- ensamblado productivo que compruebe que las atestaciones proceden del broker
  autorizado y que el HMAC usa una clave exclusiva gestionada por KMS/HSM.

Controles tecnicos V4 ya verificados, sin que levanten el NO-GO productivo:

- ACL de tipos cerrada para objetos actuales, tipos explicitos futuros y tipos
  fila implicitos futuros mediante una guarda DDL, con privilegios efectivos;
- desmontaje de roles con lista positiva de miembros y fallo previo ante
  membresias inesperadas;
- superficie `pgcrypto` minima, sin ejecucion general heredada por los roles
  runtime;
- `down` destructivo bloqueado siempre salvo opcion expresa, sin `CASCADE` y con
  rechazo de dependencias externas mediante `RESTRICT`.

Pendiente antes de produccion:

- gestion web gobernada de borrador, prueba, aprobacion, publicacion, retirada y
  nueva version de plantillas;
- reglas gobernadas por plantilla/flujo para exigir relaciones y cardinalidades
  adicionales; el primer corte exige un expediente principal;
- adaptadores productivos PostgreSQL para catalogo, reserva y expediente, y
  almacenamiento S3 compatible local;
- implementacion PostgreSQL de la reserva idempotente con indice unico,
  transaccion, arrendamiento renovable y recuperacion tras caidas;
- politica de rotacion y periodo de convivencia de claves de idempotencia; la
  clave debe permanecer verificable durante toda la ventana de reintentos;
- PDF/A y PDF/UA o perfil equivalente que supere validacion de accesibilidad y
  preservacion; el PDF actual es una salida funcional, no la certificacion final;
- integracion con AutoFirmaV3, validacion de firma, sello cualificado de tiempo,
  registro electronico, CSV y justificante;
- metadatos y exportacion completa de expediente ENI;
- politicas de conservacion, bloqueo, expurgo y legal hold;
- antivirus para documentos aportados (no aplica al contenido generado por el
  propio motor) y controles DLP cuando procedan;
- pruebas de contrato de los repositorios documentales PostgreSQL/S3,
  concurrencia, caidas y restauracion;
- despliegue y prueba de segregacion real del emisor Unix V4: reunir la
  credencial emisora y ejecutora mantiene obligatoriamente el estado NO-GO;
- custodia y rotacion de claves mediante HSM/KMS o gestor de secretos homologado,
  ACL del directorio/socket y procedimientos operativos aprobados.

## Criterio de aceptacion inicial de RRHH

Dada una plantilla publicada y una version concreta, un tecnico autorizado
puede generar un unico documento logico con DOCX y PDF para un expediente. El
sistema devuelve identidad, huellas y estado; conserva ambas representaciones;
registra una unica traza completa sin valores personales y un evento enlazado;
un reintento no duplica nada, y modificar plantilla, datos, relaciones o salidas
solicitadas con la misma clave se rechaza.

El criterio juridico final añade firma valida, sello de tiempo, asiento de
registro, CSV, metadatos ENI, custodia y validacion PDF accesible.
