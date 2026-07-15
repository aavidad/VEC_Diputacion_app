# Almacen documental seguro, cifrado e intercambiable

Fecha de referencia: 14 de julio de 2026.

## Estado real

El nucleo dispone del puerto transversal `AlmacenObjetos`: lectura y escritura
en flujo, capacidades declaradas, referencias opacas, cuarentena, promocion sin
sobrescribir el original, retencion, inmovilizacion y eliminacion controlada.
La fachada `AlmacenContenidoDocumento` permite que la generacion documental ya
utilice ese puerto sin conocer el proveedor. Cada operacion exige una accion
tecnica exacta y queda ligada a autorizacion, finalidad, clasificacion,
correlacion, carga, sujeto seudonimizado mediante HMAC, recurso, modulo y
huella HMAC de la solicitud. El conector debe devolver esos mismos vinculos en
la evidencia tecnica que el caso de uso incorpora a la auditoria; omitir,
alterar o cruzar uno solo deniega el resultado completo.

Tambien estan aplicados el agregado `CargaDocumental`, el puerto transaccional
`RepositorioCargasDocumentales` y el caso de uso `ServicioCargaDocumental`.
La cadena exige cuatro concesiones de negocio distintas y exactas —preparar,
confirmar, analizar y promover— y, dentro del puerto, acciones tecnicas
separadas para preparar, abandonar, confirmar, leer, analizar y promover. Una
no habilita otra. Una identidad de servicio autorizada realiza el analisis sin
atribuirlo falsamente al ciudadano. Reserva/idempotencia, cambio de estado,
auditoria y outbox se confirman mediante contratos tipados. Un resultado
parcial, de error o no concluyente queda retenido y nunca activa la promocion.

Existe un registro de fabricas para seleccionar por configuracion cualquiera de
los conectores instalados, sin condicionales de producto en el nucleo. El
adaptador de memoria declara que no cifra ni ofrece carga directa y solo sirve
para pruebas. Se ha incorporado ademas un adaptador S3 compatible basado en el
SDK libre de AWS para Go v2. Configura endpoint, region, direccionamiento por
ruta o nombre virtual, CA corporativa, lista CIDR de destino, credenciales,
buckets y cifrado sin
filtrar tipos del SDK al nucleo. Su perfil fuerte solo arranca tras una sonda
real y destructiva de versionado, SHA-256, retencion COMPLIANCE, bloqueo legal,
promocion y preservacion del original. Esto es implementacion verificable, no
homologacion del producto o instalacion elegidos.

El contrato de recibos de carga directa y su adaptador criptografico local ya
estan endurecidos. Separan cuatro claves HMAC —seudonimizacion, indice de
recibo, vinculo inmutable y atestacion de consumo— que no pueden compartir
identificador ni material. El secreto contiene 256 bits aleatorios y solo se
persiste su indice HMAC. El repositorio elige tanto `RegistradoEn` durante el
alta como `ConsumidoEn` durante la escritura condicional de consumo; el reloj
del proceso no puede proponer ni presentarse como evidencia durable. Tras el
alta, el adaptador incorpora `RegistradoEn` al recibo mediante un HMAC de
dominio separado y el consumo exige que coincida con el registro persistido.
El formato versionado `rcd2` aplica esta propiedad; un recibo de una version
anterior o desconocida se rechaza, sin compatibilidad permisiva. No existe una
version productiva expuesta que requiera migrar recibos en vuelo.

Cada carga y sesion forman un grupo HMAC: emitir otro recibo sustituye
atomicamente al activo anterior del grupo, aunque hayan cambiado otros
invariantes. El comprobante resultante es privado, no serializable y solo
permite construir una confirmacion cuando un verificador comprueba la
atestacion sobre el contexto completo y exacto. La misma raiz criptografica
debe actuar como emisor, consumidor y verificador; el constructor rechaza una
combinacion de instancias o una dependencia con nulo tipado.

El consumo crea en la misma escritura una intencion durable con referencia
opaca y huella HMAC exactas. El repositorio devuelve las fechas de alta,
consumo y expiracion realmente persistidas, y solo admite
`RegistradoEn <= ConsumidoEn < min(ExpiraEn, ValidaHasta)`, con diez minutos como
maximo absoluto. La atestacion cubre tambien esas fechas y la intencion. Los
errores internos del conector se traducen a una respuesta publica cerrada para
no filtrar productos, rutas, claves, indices ni estado. Al reemitir, el
resultado conserva de forma tipada el recibo sustituido, el mismo grupo HMAC,
su autorizacion y el instante exacto. Un predecesor de otro grupo se rechaza.

No se considerara apta para datos reales ninguna instalacion hasta disponer de
un adaptador persistente, cifrado, cuarentena, antivirus, copia restaurada,
gestion de claves y auditoria de acceso. La existencia del puerto no acredita
por si misma seguridad ni conformidad.

## Invariantes

1. Los binarios no se guardan en PostgreSQL ni en los contenedores web.
2. Ningun modulo conoce rutas, buckets, credenciales ni productos concretos.
3. Las referencias son opacas y no contienen DNI, nombre, correo, expediente ni
   otra informacion personal.
4. Todo objeto privado se cifra en transito y en reposo; las copias tambien.
   El perfil fuerte no hereda proxies, no sigue redirecciones y solo conecta
   con direcciones resueltas dentro de la lista CIDR gobernada. Un cambio DNS
   no puede desviar el conector fuera de la red autorizada.
5. El servidor autoriza cada operacion por actor, perfil, objeto, ambito y
   finalidad. Conocer una referencia no concede acceso.
6. Cuarentena y almacen admitido tienen cuentas, permisos y ubicaciones logicas
   separadas. Un objeto no analizado nunca se descarga, firma ni incorpora a un
   expediente.
7. Original, derivados, firmas y evidencias son objetos distintos e inmutables.
8. Toda lectura y mutacion genera auditoria probatoria sin copiar datos
   personales ni contenido al registro.
9. La politica de retencion, bloqueo legal, transferencia a archivo y
   eliminacion se aplica a metadatos y objetos de forma coordinada.
10. Cambiar de filesystem a S3 compatible, nube privada o gestor documental
    solo requiere otro adaptador y una migracion operativa verificada.

## Arquitectura de puertos y adaptadores

```text
casos de uso documentales
        |
        +-- GestorSesionesCarga
        +-- AlmacenObjetos
        +-- GestorClaves
        +-- AnalizadorAntivirus
        +-- Firmador / ValidadorFirma / SelladorTiempo
        +-- RepositorioMetadatos y Auditoria
                 |
        AlmacenCifrado (decorador obligatorio)
                 |
      +----------+-------------------+
      |                              |
filesystem dedicado          S3 compatible / gestor documental
(contingencia inicial)        (S3 corporativo o Ceph RGW evaluado)
```

El cifrado se compone como decorador para que la garantia no dependa de que el
proveedor elegido tenga una opcion con un nombre concreto. El adaptador puede
anadir cifrado nativo del almacenamiento como defensa adicional.

Puertos y conectores:

- `GestorCargaDirecta`: contrato aplicado; crea una sesion de un solo uso con
  limite, caducidad maxima de diez minutos, huella esperada y destino de
  cuarentena. Las instrucciones son opacas, no serializables ni registrables;
  solo un metodo deliberado las revela al adaptador HTTP, y el destino HTTPS
  debe pertenecer al origen publicado por el conector. Junto a ellas se emite
  un recibo opaco de confirmacion, ligado a la sesion y a todos los vinculos de
  la carga. Un consumidor confiable verifica MAC y lo marca como usado
  atomicamente con la fecha del repositorio antes de construir la solicitud de
  confirmacion. Una atestacion HMAC independiente liga consumo, contexto,
  autorizacion, accion, sesion, evidencia y fecha durable. Un recibo ausente,
  sustituido, cruzado, caducado o ya consumido deniega antes de invocar el
  almacen. Confirma o abandona la recepcion.
- `AlmacenObjetos`: contrato, registro de conectores y adaptador de pruebas
  aplicados; escritura idempotente, apertura de lectura limitada,
  promocion controlada, inmovilizacion, retencion y eliminacion verificable.
- `GestorClaves`: crea una clave de datos por objeto y la envuelve con una clave
  maestra versionada en KMS/HSM. Solo persiste el sobre cifrado y la referencia
  de version, nunca la clave en claro.
- `AnalizadorContenido`: puerto aplicado para un conector ICAP, API autenticada
  o trabajador local aislado. Recibe por flujo la version exacta del objeto en
  cuarentena y devuelve motor, version, firmas, MIME detectado, bytes
  analizados, resultado cerrado y evidencia. Un error o resultado no
  concluyente nunca equivale a limpio y no decide por si mismo un estado
  administrativo. MCP se reserva para herramientas de IA; no es el transporte
  del control antivirus.
- `Firmador`, `ValidadorFirma` y `SelladorTiempo`: conectores separados para
  AutoFirma, plataforma corporativa, HSM, validacion y TSA.

Los puertos se expresan en capacidades, no en operaciones exclusivas de S3. Un
adaptador declara y demuestra si soporta versionado, bloqueo, retencion,
promocion atomica, checksum de extremo a extremo y enlaces temporales. El
arranque falla cerrado si faltan capacidades obligatorias para el perfil de
despliegue.

## Carga sin colapsar la aplicacion

Flujo estable aplicado en el nucleo:

```text
reservada -> preparada -> cuarentena -> analisis
                                      |
                                      +-> error/no concluyente/retenida
                                      +-> malicioso/sospechoso/retenida
                                      +-> limpio -> autorizacion independiente
                                                   -> promocion -> admitida
```

Con S3 compatible, el navegador carga directamente a una URL prefirmada de
corta duracion. La concesion permite exclusivamente escribir un objeto opaco,
con tamano maximo, checksum y condiciones fijadas por el servidor. No contiene
credenciales generales ni permiso para listar, leer, sobrescribir o elegir el
destino. La confirmacion posterior exige el recibo de un uso emitido por el
servidor, coteja el objeto con la sesion y liga la evidencia de consumo a la
evidencia de recepcion. Si el resultado remoto es ambiguo, el recibo no se
reutiliza: una reconciliacion idempotente futura debera determinar el estado.

El adaptador S3 actual crea primero un destino temporal opaco y nunca lo trata
como objeto canonico. Tras `HEAD` de la version, tamano, MIME, SHA-256, cifrado
y metadatos exactos, lo transmite en flujo limitado a una escritura canonica
condicional. No acumula el documento en memoria y elimina la version temporal
solo despues de verificar el canonico; tampoco declara exito hasta comprobar
su ausencia. Esta fase evita que el trafico del navegador atraviese el servidor
web, aunque la canonizacion consume ancho de banda entre la aplicacion y RGW.
La copia enteramente interna mediante `CopyObject` quedara como optimizacion de
conector cuando la version concreta de Ceph demuestre, mediante sonda, la
precondicion de destino `If-None-Match`, retencion atomica y SHA-256. La
documentacion oficial actual de Ceph no acredita conjuntamente esas tres
garantias, por lo que no se anuncian por configuracion ni se finge soporte.

Las referencias canonicas se derivan con HMAC y no incluyen la clave aportada
por el usuario. Un marcador S3 condicional e inmovilizado reserva de forma
global la huella exacta de idempotencia antes de cualquier objeto de negocio;
la misma clave no puede producir efectos distintos en otra zona u operacion.
Los reintentos concurrentes convergen en una sola version canonica. La politica
de conservacion de estos marcadores tecnicos, que solo contienen huellas y
referencias opacas, debe aprobarse con el calendario de conservacion y revisarse
antes de datos reales.

La preparacion ya persiste atomicamente el agregado, la auditoria, el outbox y
un `ManifiestoPreparacionCargaDirectaV1` canonico. El manifiesto fija MIME,
tamano, huella del contenido, clasificacion, sesion seudonimizada, conector,
recurso, `DecisionRef`, huella de decision V1, efecto y plan exactos. La
confirmacion lee agregado y manifiesto como una unica instantanea y exige una
decision nueva para confirmar junto con el recibo y su atestacion; nunca
reconstruye ni reutiliza la autoridad de preparar. La restauracion desde una
fila durable no completa ni normaliza datos: esquema desconocido, campo
ausente, alteracion o cruce con otra carga deniegan.

La reemision de instrucciones de una preparacion ya confirmada permanece
expresamente deshabilitada. Para habilitarla se debera demostrar, en una misma
frontera transaccional, que el manifiesto y el recibo durable pertenecen a la
preparacion exacta y que no existe un resultado previo o indeterminado. Hasta
entonces, ni una sesion coincidente ni una autorizacion nueva bastan para
reemitir.

El HMAC no resuelve una transaccion distribuida. Actualmente existe una
ventana inevitable entre consumir durablemente el recibo y conocer/persistir
el resultado del almacen remoto. La API de confirmacion es por ello **NO
EXPONIBLE**. El nucleo ya exige una referencia y huella de intencion, pero no
existe todavia el repositorio PostgreSQL ni el trabajador que la hagan
recuperable. El adaptador productivo debe implementar conjuntamente:

- una intencion durable previa y una maquina de estados recuperable;
- una clave idempotente que el conector remoto garantice para la operacion
  exacta;
- consulta o reconciliacion capaz de decidir el resultado tras cualquier
  caida o respuesta ambigua;
- persistencia atomica local de resultado, agregado, auditoria y outbox.

No se simula esta garantia con un booleano, un reintento ciego ni una MAC. Si
falla cualquiera de esas piezas, el objeto queda pendiente de reconciliacion y
nunca se presenta como recibido o admitido.

En la primera fase con volumen local, un proceso de carga separado transmite a
un fichero temporal con limites sin acumularlo en memoria. Aplica el mismo caso
de uso y estados, por lo que migrar a S3 no modifica el dominio ni la interfaz.

La promocion no altera el original. Si se usa desarme y reconstruccion de
contenido, el resultado se registra como representacion derivada y conserva la
relacion y ambas huellas.

La API HTTP de este flujo permanece cerrada hasta disponer del adaptador
persistente transaccional, el gestor de carga directa cifrado y el analizador
corporativo que superen sus pruebas de capacidades. El contrato y su adaptador
de pruebas no se presentan como aptitud productiva.

## Cifrado y claves

- TLS moderno y, entre servicios de confianza, mTLS cuando lo determine el
  diseno de red.
- Cifrado autenticado por objeto mediante sobre digital: clave de datos unica,
  algoritmo y parametros aprobados, contexto autenticado y clave envuelta por
  KMS/HSM.
- Separacion de claves por entorno, finalidad y clasificacion; versiones,
  rotacion, revocacion, copia y recuperacion documentadas.
- La aplicacion recibe autorizacion temporal para usar una clave, no para
  exportar claves maestras. El texto claro solo existe el tiempo imprescindible
  en memoria controlada.
- Verificacion de la etiqueta de autenticacion y del SHA-256 de los bytes
  originales al leer. Una discrepancia bloquea la entrega y genera incidente.
- Cifrado independiente de base, objetos, copias y auditoria. Comprometer una
  sola clave no debe abrir todos los dominios.

La seleccion final de algoritmos, productos certificados y parametros se hara
con el Responsable de Seguridad a partir de la categorizacion ENS y de las
guias CCN aplicables. No se fijan en codigo listas criptograficas que deban
evolucionar, pero el sistema nunca admite un algoritmo no aprobado por mera
configuracion de un usuario.

## Certificados solicitados y documentacion firmada

Se diferencian dos conceptos:

- Certificados de identificacion/firma del ciudadano: la clave privada
  permanece en DNIe, tarjeta, almacen del usuario o prestador. El portal solo
  conserva los datos publicos y evidencias estrictamente necesarios.
- Certificados administrativos emitidos por RRHH: se generan desde una
  plantilla y datos versionados, producen PDF/DOCX cuando corresponda y siguen
  el ciclo de revision, firma o sello, registro, CSV, notificacion y archivo.

AutoFirma se integra como adaptador de firma en cliente. Para firma o sello de
la Diputacion se usa un conector de servicio corporativo o HSM. La aplicacion no
guarda claves privadas de organo en ficheros, variables, base de datos ni
contenedores.

Por cada actuacion se conservan, sin sobrescritura:

- bytes exactos del original y su SHA-256;
- bytes exactos de la representacion firmada y su SHA-256;
- formato y politica de firma aplicados;
- identidad y certificado publico del firmante o sello, con minimizacion;
- resultado de validacion, cadena de confianza y estado de revocacion obtenido;
- OCSP/CRL, sello de tiempo y evidencia necesaria para validacion a largo plazo;
- metadatos ENI, CSV cuando proceda y relaciones con expediente y documento
  origen;
- actor, perfil, finalidad, autorizacion, fecha fiable y correlacion de todas
  las operaciones.

Los formatos concretos —PAdES, XAdES o CAdES y sus niveles de longevidad— se
seleccionan por politica de firma, tipo documental, intercambio y conservacion;
no por preferencia del programador. Renovar o aumentar una firma crea evidencia
derivada y no cambia los bytes historicos anteriores.

Los circuitos con varios firmantes se modelan desde el principio como una
cadena de revisiones inmutables. La franja visible con CSV, URL y QR se prepara
antes de la primera firma; no se estampa un PDF despues de firmado. El modelo de
versiones, activacion del CSV, cotejo y revision se especifica en
`docs/portal_vec/firma_csv_qr_y_cotejo.md`.

## Metadatos minimos por objeto

- identificador y version opacos;
- conector y clase de almacenamiento;
- clasificacion y finalidad;
- tamano declarado y comprobado;
- tipo declarado, tipo detectado y nombre original saneado como metadato
  privado, nunca como ruta;
- SHA-256 del contenido original y, donde proceda, HMAC de datos de baja
  entropia;
- algoritmo de cifrado, referencia de clave maestra y sobre cifrado;
- estado de carga, cuarentena, antivirus, firma, retencion y disponibilidad;
- origen, presentador, expediente, documento logico y derivacion mediante
  referencias internas;
- fechas fiables, conservacion, bloqueo legal y eliminacion prevista;
- referencias de auditoria, autorizacion y evidencia externa.

## Autorizacion, privacidad y auditoria

PostgreSQL decide que objeto corresponde al recurso autorizado; el cliente no
aporta una ruta. Cada descarga vuelve a evaluar RBAC+ABAC, titularidad, unidad,
clasificacion, finalidad y garantia de autenticacion. Los enlaces temporales se
emiten despues de esa decision, caducan en minutos, son de una operacion y se
registran; para sensibilidad alta puede obligarse a que la lectura atraviese
un servicio interno controlado en lugar de exponer un enlace directo.

El contexto entregado a conectores nunca contiene el identificador real de la
persona. El nucleo obtiene un seudonimo HMAC por carga con una clave exclusiva,
distinta de las usadas para sesion, idempotencia, recibos y huellas de
solicitud. Los documentos generados por la propia Administracion aplican la
misma regla y ligan cada escritura al expediente, modulo y solicitud completa.

El permiso tecnico de almacen es una capacidad opaca V1, no una estructura que
pueda rellenar un modulo. Solo una fabrica positiva y especifica transforma una
decision vigente del PDP en uno de los pasos declarados del plan. La decision
debe estar ligada, mediante la huella de su recurso, a operacion, carga,
clasificacion, seudonimo HMAC, huella HMAC de solicitud, efecto y, cuando ya
exista, referencia y version exactas del objeto. Accion de negocio, campos,
finalidad, contexto de actor V1 y ausencia de obligaciones desconocidas se
cotejan sin aproximaciones. El valor cero, la serializacion, una accion nueva,
un campo adicional, un paso no declarado, una proyeccion de trazabilidad o una
autorizacion caducada deniegan.

Cada adaptador vuelve a validar la capacidad con su reloj inmediatamente antes
del efecto. Leer, analizar, escribir, promover y retener son pasos diferentes;
una concesion no se convierte en otra. Mientras no exista una fabrica positiva
y su caso de uso completo, listar, bloquear, levantar un bloqueo, eliminar o
realizar cualquier operacion adicional permanecen cerrados incluso para un
administrador. La implementacion productiva añadira el consumo durable unico
`DecisionRef -> (EfectoRef, huella del plan)` y el recibo idempotente de cada
paso dentro de la misma transaccion que la intencion y el outbox. Los efectos
remotos ambiguos quedan pendientes de reconciliacion; nunca se compensan
inventando permiso para borrar.

Se auditan creacion de sesion, emision y consumo de recibo, recepcion,
analisis, promocion, lectura,
descarga, exportacion, firma, validacion, sustitucion, retencion, inmovilizacion
y eliminacion. El log contiene referencias y huellas, no DNI, nombres de
fichero completos, URLs firmadas, tokens, claves ni contenido.

## Conservacion, copia y recuperacion

- Versionado y bloqueo contra sobrescritura para documentos firmados,
  expedientes cerrados y evidencias que lo requieran.
- Retencion derivada de serie documental y estado del procedimiento; no una
  duracion global codificada.
- Bloqueo legal separado de retencion ordinaria y trazado con doble control.
- Copias cifradas, inmutables cuando proceda, con claves y credenciales
  distintas y fuera del dominio de fallo principal.
- Conciliacion periodica entre PostgreSQL y objetos para detectar huerfanos,
  ausencias, tamano o huella incompatibles y retenciones incoherentes.
- Restauraciones completas ensayadas, incluida correspondencia entre
  metadatos, objetos, claves, firmas y auditoria. Una copia no restaurada no se
  considera evidencia suficiente de recuperacion.

## Primera fase y evolucion

Primera fase posible, enteramente local:

- PostgreSQL para metadatos y cola transaccional;
- volumen dedicado cifrado o S3 corporativo existente;
- proceso de carga y trabajador separados del servidor web;
- KMS/HSM corporativo; si no existe, su seleccion es condicion previa para
  datos de sensibilidad alta, no una clave embebida provisional;
- antivirus corporativo por ICAP/API o trabajador local aislado;
- copia externa cifrada y prueba de restauracion.

Evolucion sin reescribir el nucleo: S3 distribuido local, cabina, nube privada
o servicio de archivo, manteniendo referencias y puertos. La migracion copia,
descifra y vuelve a cifrar de forma controlada, verifica huellas, conserva las
referencias logicas y registra la procedencia.

Revision tecnologica de 15-07-2026: MinIO no se propone para una instalacion
nueva porque [su repositorio oficial](https://github.com/minio/minio) fue
archivado el 25-04-2026. Ceph RGW queda como candidato maduro cuando existan
recursos de cluster; SeaweedFS y Garage solo pasan a prueba de concepto ligera
y deben demostrar cifrado, versionado, retencion/bloqueo, recuperacion,
actualizaciones de seguridad y operacion ENS. Mientras no haya un S3 homologado,
la primera fase usa el volumen dedicado cifrado detras del mismo puerto. Esta
decision no cambia el dominio ni bloquea una migracion posterior.

## Pruebas de aceptacion

1. Cambiar el adaptador de almacenamiento sin recompilar dominio ni modulos.
2. Demostrar que ningun bucket, volumen u objeto es publico ni listable.
3. Rechazar sesion caducada, exceso de tamano, checksum distinto, tipo real no
   permitido, malware, objeto repetido o promocion sin evidencia.
4. Impedir descarga de pendiente, cuarentena, rechazado o retirado.
5. Verificar que una referencia o URL filtrada no permite ampliar operacion,
   objeto, plazo ni identidad.
6. Rotar la clave maestra sin modificar los bytes ni perder objetos historicos.
7. Detectar alteracion del cifrado o contenido antes de entregar texto claro.
8. Probar autorizacion negativa por titularidad, rol, unidad, finalidad y nivel
   de autenticacion.
9. Validar firma y revocacion con resultado reproducible y evidencia temporal.
10. Restaurar metadatos, objetos, claves, auditoria y firmas en un entorno
    aislado y comparar todas las huellas.
11. Probar retencion, bloqueo legal, doble control y eliminacion coordinada.
12. Confirmar que logs, metricas, trazas y copias no contienen DNI, tokens,
    claves, URLs prefirmadas ni contenido documental.
13. Rechazar recibos de otra carga, otra sesion, caducados o reutilizados y
    demostrar que el segundo intento no alcanza el conector de objetos.
14. Rechazar acciones tecnicas vacias, desconocidas o cruzadas —por ejemplo,
    intentar analizar con una concesion de lectura— sin efectos laterales.
15. Demostrar que las horas de alta y consumo proceden de sus transacciones
    durables, que un desfase del reloj del proceso no amplia el plazo y que
    alterar cualquiera de ellas invalida el recibo o la atestacion.
16. Demostrar que una reemision para la misma carga y sesion desactiva el
    recibo anterior incluso si cambian otros campos, que el predecesor acredita
    el mismo grupo HMAC y que solo uno de varios consumos concurrentes puede
    prosperar.
17. Simular una caida en cada punto entre intencion, llamada remota y
    confirmacion local; recuperar el resultado sin reutilizar recibos ni crear
    dos objetos.
18. Rechazar constructores con dependencias de recibo nulas, nulas tipadas o
    con raices criptograficas distintas, sin iniciar ninguna operacion.
19. Demostrar que emision, consumo, expiracion, limite de autorizacion,
    referencia y huella de intencion estan autenticados y que alterar uno solo
    impide alcanzar el conector.
20. Traducir fallos internos del almacen a errores publicos estables y probar
    que la respuesta no contiene producto, ruta, indice, HMAC, token ni causa
    interna.
21. Reemitir con una autorizacion nueva sin reutilizar la anterior y conservar
    de forma tipada el recibo predecesor, su autorizacion y el instante de
    sustitucion para auditoria.

## Base normativa de diseno

- Esquema Nacional de Seguridad, Real Decreto 311/2022: confidencialidad,
  integridad, trazabilidad, autenticidad, disponibilidad y conservacion;
  criptografia y gestion de claves con algoritmos y parametros autorizados por
  el CCN: <https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191>.
- Esquema Nacional de Interoperabilidad, Real Decreto 4/2010, texto
  consolidado: <https://www.boe.es/buscar/act.php?id=BOE-A-2010-1331>.
- NTI de Documento Electronico: contenido, firma y metadatos como componentes
  relacionados: <https://www.boe.es/buscar/act.php?id=BOE-A-2011-13169>.
- NTI de Politica de Firma y Sello Electronicos y de Certificados de la
  Administracion: <https://www.boe.es/buscar/act.php?id=BOE-A-2016-10146>.
- Ley 6/2020 de servicios electronicos de confianza:
  <https://www.boe.es/buscar/doc.php?id=BOE-A-2020-14046>.

La categorizacion ENS, el analisis de riesgos, la politica de firma, el cuadro
de clasificacion y el calendario de conservacion deben aprobarlos los roles
competentes. Este documento es una especificacion tecnica y no sustituye sus
decisiones ni una auditoria o certificacion de conformidad.
