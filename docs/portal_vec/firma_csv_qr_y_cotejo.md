# Firma multiple, CSV, QR y servicio de cotejo

Fecha de referencia: 16 de julio de 2026.

## Decision

Los documentos administrativos emitidos por el portal podran incorporar una
franja de verificacion con:

- codigo seguro de verificacion (CSV) legible y apto para transcripcion;
- direccion estable del servicio de cotejo de la sede electronica;
- codigo QR que facilite abrir ese servicio;
- identificacion del organo emisor y, cuando proceda, apariencia de las firmas
  o del sello electronico.

El QR sera el codigo optico principal. Un codigo de barras lineal, si algun
sistema heredado lo exige, codificara el mismo CSV y no creara un segundo
identificador. El QR ya es un codigo de barras bidimensional y admite una URL
sin perder la posibilidad de imprimir el CSV como texto.

El QR no prueba por si mismo autenticidad, integridad ni firma. Es solo un
medio de acceso. La evidencia la forman los bytes exactos almacenados, su
huella, el CSV vinculado a esa version, las firmas o sellos electronicos, los
sellos de tiempo, la validacion y la auditoria.

## Estado de AutofirmaV2

La implementacion local ya aporta una base reutilizable:

- genera un QR PNG con `github.com/skip2/go-qrcode` y lo integra en la imagen
  de la apariencia visible PAdES;
- permite texto, logo, resumen de firmantes, posicion, rotacion y seleccion de
  paginas;
- dispone de flujo guiado de firma inicial y cofirmas sucesivas;
- evita duplicar el sello visible y el QR en las cofirmas posteriores;
- dispone de un verificador PAdES que recorre las firmas encontradas y de
  pruebas de preservacion de firmas anteriores.

Sus limites actuales para este portal son importantes:

- el contenido del QR es un texto o URL aportado mediante opciones, no un CSV
  gobernado por la Administracion;
- no enlaza el codigo con una version inmutable en un almacen central;
- no ofrece el servicio de cotejo, su autorizacion ni la politica de
  conservacion;
- no archiva todas las revisiones, evidencias y resultados de validacion;
- el caso de uso de cofirma encadena revisiones en memoria y devuelve solo la
  ultima; no invoca el verificador criptografico despues de cada paso;
- un error al construir la imagen visible o el QR puede degradarse hoy a una
  firma sin esa apariencia en vez de fallar cerrado;
- no valida que el contenido del QR sea una URL HTTPS de la sede, ni impone una
  longitud y un dominio aprobados;
- la dependencia QR usada no publica versiones y el revisionado fijado es de
  2020; puede mantenerse para compatibilidad durante el prototipo, pero debe
  pasar revision de dependencias o sustituirse en AutofirmaV3 por una biblioteca
  mantenida antes de produccion.

Por tanto, no se reimplementara la firma de escritorio dentro del portal. Se
reutilizara Autofirma mediante un conector y se le entregara un encargo de firma
preparado por el portal. Autofirma devuelve los bytes firmados y la evidencia
de la operacion; el portal los valida, almacena y relaciona.

Autofirma no debe convertirse en el archivo central. Puede seguir usando
temporales locales con permisos restrictivos y borrado al terminar. Es el
portal quien conserva de forma cifrada cada entrada, salida y evidencia.

## Cambios necesarios en AutofirmaV3

El fork puede mantener AutofirmaV2 intacta y cerrar estas capacidades mediante
un contrato versionado:

1. Operacion de firma de una unica revision preparada que reciba identificador
   opaco de encargo, nonce, caducidad, SHA-256 de entrada, politica de firma y
   apariencia autorizada.
2. Validacion estricta del origen y proteccion contra repeticion. El portal no
   entrega credenciales generales ni rutas internas al cliente.
3. QR gobernado: URL HTTPS, dominio de sede permitido, longitud maxima y CSV
   opaco. Si la politica lo marca obligatorio, cualquier fallo cancela la firma.
4. Recibo de salida con huellas de entrada y salida, formato, nivel, huella del
   certificado publico, instante y resultado. El portal lo vuelve a comprobar;
   no lo acepta como prueba por mera declaracion del cliente.
5. Una invocacion por persona firmante en circuitos distribuidos. El portal
   valida y almacena la revision antes de habilitar la siguiente. La cofirma
   multiple local queda como comodidad para varios certificados disponibles en
   un mismo equipo, no como portafirmas multiusuario.
6. Opcion de devolver tambien cada revision de la cofirma local o, como minimo,
   un callback por paso para que el portal pueda conservarla y validarla.
7. No reserializar el PDF firmado fuera de una actualizacion incremental PAdES
   permitida; conservar exactamente los bytes recibidos y producidos.
8. Pruebas de lectura del QR impreso, firma multiple real, error obligatorio de
   apariencia, DSS estable, PDF/A, permisos DocMDP y modificaciones maliciosas.
9. Sustituir o encapsular la biblioteca QR no versionada tras una evaluacion de
   mantenimiento, licencia, reproducibilidad y pruebas de conformidad.

La validacion administrativa principal se realizara en el portal con el
conector local DSS. Esto evita que un cliente comprometido pueda declarar como
valida su propia salida.

## Invariantes

1. Un CSV identifica una unica version emitida de un documento y queda
   vinculado a sus bytes exactos y a su SHA-256.
2. Ningun CSV, QR o URL contiene DNI, nombre, expediente, correo, unidad ni
   identificadores secuenciales.
3. Conocer un identificador interno o una referencia de almacenamiento nunca
   permite cotejar ni descargar un documento.
4. El original, el PDF preparado, cada revision firmada, la version emitida y
   los informes de validacion son objetos distintos e inmutables.
5. No se dibuja, reimprime, normaliza ni estampa una apariencia sobre un PDF
   despues de firmado. Una mutacion posterior de contenido crea otra version.
6. Todas las firmas requeridas se determinan antes de emitir. Una firma tardia
   sobre un documento ya emitido inicia una nueva version administrativa y un
   nuevo CSV.
7. Las cofirmas PAdES se incorporan como revisiones incrementales y cada paso
   se valida antes de aceptar el siguiente.
8. Las revisiones intermedias del circuito de firma se conservan para prueba,
   pero no se publican como documentos emitidos ni activan el CSV reservado.
9. El servicio de cotejo devuelve exactamente la version emitida, no una
   regeneracion desde plantilla o base de datos.
10. El QR nunca sustituye al CSV y la URL escritos: el documento debe seguir
    siendo verificable sin camara y con tecnologias de apoyo.

## Flujo de emision y firma multiple

```text
documento logico aprobado
        |
        v
reservar CSV opaco y circuito de firmantes
        |
        v
renderizar PDF final con CSV + URL + QR
        |
        v
validar PDF/PDF-A y guardar version preparada
        |
        v
firma 1 -> validar -> guardar revision 1
        |
        v
firma 2 -> validar -> guardar revision 2
        |
        v
... firmas requeridas ...
        |
        v
sello de organo / tiempo / aumento LT-LTA cuando proceda
        |
        v
validacion final + registro + activacion CSV + notificacion
```

El CSV se reserva antes de renderizar para que aparezca dentro de los bytes que
se van a firmar, pero solo se activa cuando el circuito termina y la version
final ha superado validacion, registro y controles. Si el circuito se abandona,
el codigo queda inutilizado y nunca se recicla.

La franja se prepara antes de la primera firma. Las apariencias visibles de
firmas posteriores se crean dentro de sus propias revisiones PAdES
incrementales. No se ejecuta un proceso de estampado general despues de firmar.

## Circuitos configurables por puesto y competencia

Los firmantes no se fijan en codigo como una lista de personas o cargos. Cada
tipo documental y cada hito del proceso referencia una
`PoliticaCircuitoFirma` publicada, versionada e inmutable que declara:

- tipo de documento, proceso, fase y acto que se firma;
- puestos funcionales requeridos, por ejemplo Secretaria, Intervencion,
  titular de la Delegacion, Presidencia, jefatura o personal tecnico de RRHH;
- clase de intervencion de cada puesto: autoria, informe, visto bueno,
  fiscalizacion, propuesta, resolucion, firma o sello de organo;
- firma secuencial, paralela o por grupos, junto con orden, cardinalidad y
  quorum cuando exista un organo colegiado;
- incompatibilidades, separacion entre preparacion, revision y firma, y nivel
  de autenticacion exigido;
- reglas de suplencia, delegacion, abstencion y sustitucion, con la referencia
  administrativa que debe acreditarlas;
- formato, nivel de firma, sello de tiempo, registro y politica de acceso de la
  version final.

La expresion `puesto firmante` significa una posicion organizativa o una
funcion competente, no un rol tecnico RBAC ni una cuenta compartida. Un
`ResolutorCompetenciaFirmante` consulta el directorio organizativo, RPT y
delegaciones vigentes y produce una asignacion atestada para el expediente. En
el momento de firmar se verifica simultaneamente que:

1. el certificado identifica a la persona que actua;
2. esa persona ocupa, sustituye o ejerce validamente el puesto funcional;
3. la asignacion estaba vigente en el instante de firma;
4. el acto, expediente, documento, version y huella coinciden exactamente;
5. no hay abstencion, incompatibilidad o cambio de circuito pendiente.

Una firma no se autoriza solo porque la persona posea el rol `rrhh` o
`administrador`. Los roles permiten acceder a la tarea; la competencia para el
acto procede de la asignacion organizativa atestada. No se usan cuentas
genericas de Secretaria, Intervencion o RRHH para firmar.

Los suplentes o delegados previstos pueden ocupar un hueco sin recompilar la
aplicacion. La firma conserva puesto ejercido, titulo de actuacion y referencia
de suplencia o delegacion. Si despues de la primera firma se altera un requisito
del circuito que no estaba previsto como alternativa, se cancela la emision y
se genera una version nueva; no se modifica el PDF parcialmente firmado.

Todos los modulos podran materializar como documento firmable el resultado
exacto de un calculo o revision: entrada y reglas, desglose, incidencias,
decisiones, version, huellas y momento de corte. La politica determina que
hitos requieren documento firmado. No se firma una cifra regenerada ni «el
estado actual», sino la instantanea inmutable que produjo el resultado.

Entre otros, se contemplan:

- solicitud, declaracion y autobaremo presentados por la persona;
- informe tecnico de revision y acta de la comision;
- listados provisionales y definitivos;
- informe de fiscalizacion, propuesta y resolucion;
- certificado, notificacion, recibo de registro y justificante;
- rectificacion, revocacion, rehabilitacion y nueva publicacion.

Cuando el ordenamiento y la politica aprobada permitan una actuacion
administrativa automatizada, se utilizara sello electronico del organo y se
identificara el sistema responsable. No se simulara la firma de una persona.

## Descarga por las partes interesadas

Todo documento incorporado a un expediente tiene una politica de acceso
positiva y una o varias representaciones identificadas. La aplicacion permite
descargar a cada parte autorizada, segun proceda:

- los bytes exactos del fichero que aporto y su recibo de presentacion;
- el original recibido, cuando su relacion y finalidad concedan acceso;
- la representacion accesible de consulta;
- el PDF/PAdES final con todas las firmas, CSV, QR y sellos;
- informes, requerimientos, comunicaciones, notificaciones y resoluciones que
  deban ponerse a su disposicion;
- un paquete del expediente con indice y manifiesto de huellas cuando exista
  derecho de acceso o entrega formal.

La condicion de parte interesada no concede acceso indiscriminado a notas de
trabajo, documentos internos, secretos protegidos ni datos de otras personas.
La politica se evalua por documento y version atendiendo a expediente,
relacion, representacion, finalidad, clasificacion, tramite, estado y posibles
limites de acceso. Cuando proceda acceso parcial se genera previamente una
representacion disociada o testada, que es otro objeto con su propia huella y
trazabilidad; nunca se ocultan datos alterando el original firmado.

Las capacidades se conceden por separado: ver metadatos, previsualizar,
descargar original, descargar representacion firmada y exportar expediente.
Una capacidad no implica las demas. La persona interesada y su representante
acreditado se relacionan con el expediente mediante referencias versionadas y
vigentes; una mera URL, CSV, identificador o conocimiento del numero de
expediente no concede acceso.

La descarga se inicia a traves de la API autorizada. Tras releer politica,
relacion, estado y huella, el conector de almacenamiento emite una autorizacion
opaca, de un solo uso o vida muy corta, limitada a objeto, version, operacion y
dispositivo/sesion cuando proceda. El contenedor S3, ruta interna y claves de
objeto no se exponen. Cada intento, concesion, denegacion y descarga terminada
queda auditado sin registrar el contenido ni la URL temporal.

La vista previa no sustituye la descarga probatoria. Para documentos aportados
se conserva el original exacto aunque exista una conversion segura; para
documentos generados se conserva tanto el formato fuente gobernado como sus
representaciones. PDF/PAdES sera la representacion humana firmada habitual,
pero CSV, JSON, ODT, XML u otros formatos pueden entregarse tambien en original
o mediante firma separada/contenedor interoperable cuando la politica lo
requiera.

## CSV y contenido del QR

El formato definitivo debe aprobarse en la norma o resolucion interna que
regule el sistema. Como base tecnica:

- al menos 128 bits aleatorios generados criptograficamente;
- representacion legible, sin caracteres ambiguos y agrupada para
  transcripcion;
- digito o caracteres de control para detectar errores de tecleo;
- no derivado de DNI, numero de expediente, fecha ni contador;
- unicidad garantizada tambien mediante restriccion en base de datos;
- almacenamiento del resumen criptografico del CSV, protegido con una clave
  del gestor de claves, en vez de indexar el valor como texto recuperable;
- vinculacion atomica con identificador documental, version, SHA-256, organo,
  estado, politica de acceso y periodo de disponibilidad.

El QR contendra una URL HTTPS estable del servicio de cotejo y el CSV opaco. Se
preferira transportar el codigo en el fragmento de la URL para que navegadores,
proxies y servidores no lo incluyan automaticamente en rutas y registros. La
pagina de cotejo lo toma localmente y lo envia mediante `POST`. Siempre se
ofrecera tambien un formulario donde escribir el CSV; esta tecnica es una
medida de minimizacion, no un requisito normativo.

No se usaran acortadores externos, codigos QR dinamicos de terceros, analitica
publicitaria ni dominios distintos de la sede. El QR sera negro sobre blanco,
con margen de silencio, tamano de impresion y correccion de errores ensayados;
no se colocara un logotipo encima de sus modulos.

## Cotejo publico, protegido e interno

La posesion del CSV no debe confundirse con publicidad universal. La politica
se fija por tipo y clasificacion documental:

- **publico**: muestra y descarga directa de los bytes emitidos;
- **protegido**: confirma autenticidad e integridad con datos minimos y exige
  identificacion, titularidad o una garantia adicional para ver o descargar el
  contenido;
- **interno o alto**: cotejo solo en el entorno autorizado o mediante area
  autenticada; no se publica el documento en Internet por incorporar un QR.

Los documentos estrictamente internos para los que no proceda un servicio de
cotejo en sede utilizaran firma PAdES o sello electronico y controles internos,
sin asignarles por defecto un CSV publico.

La pantalla de cotejo debe mostrar, sin revelar mas datos de los necesarios:

- resultado claro: valido, retirado, sustituido, no emitido o no encontrado;
- organo emisor, tipo documental y fecha de emision cuando puedan mostrarse;
- SHA-256 de la version recuperada;
- firmas y sellos, con estado, instante fiable y politica de validacion;
- relacion con una version posterior cuando una politica juridica permita
  informar de la sustitucion;
- descarga de los bytes exactos solo si la politica de acceso lo permite.

La consulta protegida aplica una proyeccion positiva y cerrada emitida por el
PDP. `estado` y `codigo_ref` son campos base obligatorios: una decision sin
ambos, con lista vacia o con cualquier campo desconocido se deniega completa y
se audita, sin construir una respuesta `disponible`. Los campos opcionales no
se serializan si no aparecen con su nombre canonico exacto en la decision; no
se recortan, traducen ni normalizan nombres futuros. El indicador
`permite_descarga` tambien es opcional: solo se revela cuando ese campo ha sido
concedido y solo puede valer verdadero cuando concurren la politica documental
y la capacidad exacta `descarga`. La consulta publica conserva su contrato y
su lista de campos propios, separados de esta proyeccion autenticada.

La custodia del secreto CSV aplica la misma regla. Proteger una clave nueva,
recuperarla y eliminar una referencia huerfana son tres capacidades opacas e
incompatibles, con decisiones PDP distintas y vinculadas al recurso y a la
huella exactos. La decision de reservar un CSV no autoriza ninguna de ellas.
El flujo interactivo no puede eliminar custodia: esa operacion exige un worker
interno expresamente autorizado, cuenta privilegiada en la superficie
administrativa, intencion durable y reconciliacion. Mientras esa composicion
no exista, el huerfano se aisla y registra para reconciliacion; nunca se borra
reutilizando el permiso del usuario.

Se aplican limitacion de intentos, deteccion de abuso y, cuando sea necesario,
prueba antirobot accesible. Las respuestas, tiempos y codigos de error no deben
permitir enumerar documentos. El CSV, las URL completas y los documentos no se
copian a logs, metricas ni trazas.

## Revision por RRHH y personas autorizadas

El visor interno no usa la misma autorizacion que el cotejo publico. Evalua
RBAC y ABAC por expediente, unidad, finalidad, clasificacion y relacion con la
persona. Su ficha documental incluye:

- original aportado y todas las representaciones derivadas;
- estado de cuarentena y antivirus;
- version, tamano, MIME detectado y huella;
- cadena de revisiones y derivaciones;
- circuito previsto, firmas recibidas y firmas pendientes;
- validacion criptografica, confianza, revocacion, sellos de tiempo y nivel
  PAdES;
- CSV reservado o activo, registro y notificacion;
- retencion, bloqueo legal y accesos auditados;
- comparacion visual y estructural entre revisiones PDF.

Los ficheros no confiables se visualizan desde un origen web aislado, sin
credenciales de sesion y con una politica de contenido restrictiva. Para PDF
firmados se valida tambien la diferencia de paginas, anotaciones, objetos y
apariencia entre revisiones. La vista previa segura es un derivado; la descarga
probatoria entrega los bytes exactos autorizados.

## Componentes abiertos que se reutilizan

- **AutofirmaV2/AutofirmaV3**: firma en cliente, seleccion del certificado y
  apariencia QR ya disponible.
- **DSS de la Comision Europea**: servicio aislado detras del puerto de firma y
  validacion para PAdES, firmas multiples, aumento T/LT/LTA, OCSP/CRL, listas de
  confianza y deteccion de modificaciones maliciosas en PDF. Se usara una
  version estable; a esta fecha es DSS 6.4, no la candidata 6.5.RC1.
- **Apache PDFBox a traves de DSS**: manejo y apariencia de PDF donde resulte
  mas conforme que el motor actual.
- **PostgreSQL**: metadatos, circuito, vinculo del CSV, autorizacion y outbox.
- **almacen de objetos S3 compatible**: bytes inmutables de todas las
  revisiones y evidencias, mediante el conector `AlmacenObjetos`.

La demostracion publica de DSS no se utilizara con documentos reales. Los
ficheros se procesaran en una instancia local aislada o en el servicio
corporativo aprobado.

## Puertos del nucleo

El dominio no depende de Autofirma, DSS, PDFBox ni una biblioteca QR concreta:

- `GeneradorCodigoCotejo`: reserva, activa, retira y resuelve un CSV;
- `PreparadorAparienciaDocumento`: incorpora la franja antes de firmar;
- `CatalogoPoliticasCircuitoFirma`: obtiene la politica publicada exacta;
- `ResolutorCompetenciaFirmante`: resuelve puesto, titularidad, suplencia o
  delegacion vigentes sin acoplar el nucleo al directorio corporativo;
- `RepositorioCircuitosFirma`: conserva asignaciones, pasos, rechazos,
  revisiones y recibos del circuito;
- `FirmadorDocumento`: inicia o completa una revision de firma;
- `ValidadorFirmaDocumento`: produce un informe firmado o sellado de
  validacion;
- `SelladorTiempo`: obtiene evidencia temporal;
- `AlmacenObjetos`: conserva cada revision y abre los bytes exactos;
- `RepositorioDocumental`: conserva relaciones, estados y huellas;
- `AutorizadorDescargaDocumento`: concede capacidades por relacion, finalidad,
  clasificacion, representacion y version exactas;
- `EmisorDescargaTemporal`: entrega una autorizacion opaca y acotada para los
  bytes exactos sin revelar la topologia del almacen;
- `RegistroElectronico`: registra la version final;
- `Auditoria`: registra toda reserva, firma, validacion, cotejo, lectura,
  retirada y sustitucion.

Las capacidades se seleccionan por politica versionada. Agregar otro firmador,
validador, formato optico o repositorio no cambia el nucleo ni los modulos.

## Pruebas de aceptacion

1. Escanear el QR y transcribir manualmente el CSV conduce a la misma version.
2. Cambiar un byte impide validar y nunca devuelve una regeneracion parecida.
3. Un codigo inexistente, retirado o reservado no permite inferir si hay una
   persona o expediente relacionado.
4. Dos documentos y dos versiones emitidas nunca comparten CSV.
5. Un circuito fallido no recicla su codigo.
6. Cada cofirma conserva validas las firmas anteriores y queda como revision
   inmutable relacionada.
7. Cualquier estampado no previsto despues de firmar se rechaza.
8. Se detectan cambios de paginas, anotaciones, objetos y apariencia entre
   revisiones, incluidos casos de ataque de sombra PDF.
9. La verificacion de cada firma reproduce confianza, revocacion y tiempo con
   la evidencia archivada.
10. El cotejo protegido no entrega contenido a quien solo posee una referencia
    interna, una URL caducada o un CSV sin la garantia adicional requerida.
11. El visor de RRHH no ejecuta JavaScript embebido, enlaces, adjuntos ni otro
    contenido activo procedente del fichero revisado.
12. CSV, URL, datos personales y contenido no aparecen en logs o trazas.
13. El documento impreso conserva legibles CSV y URL aunque el QR este danado.
14. La restauracion recupera la misma version, huella, firmas, CSV y auditoria.
15. Una decision protegida con campos vacios, sin `estado` o `codigo_ref`, o
    con un campo desconocido no produce una respuesta util; las proyecciones
    minima y completa solo contienen los campos expresamente concedidos.
16. Cambiar la persona titular de un puesto no cambia la politica; el resolutor
    acredita la asignacion vigente y rechaza certificados, suplencias o
    delegaciones que no correspondan al instante y acto exactos.
17. Un rol tecnico suficiente sin competencia organizativa no puede firmar; una
    competencia organizativa sin permiso de acceso tampoco puede abrir la tarea.
18. Una firma, rechazo o sustitucion no prevista despues de la primera firma no
    muta la revision existente y obliga a emitir una version nueva.
19. La persona aportante recupera los bytes exactos y el recibo; la parte
    interesada recupera el PDF/PAdES que le corresponde; una persona ajena no
    obtiene ni metadatos enumerables ni URL de objeto.
20. Una entrega parcial descarga una representacion testada distinta y conserva
    inmutables y relacionadas la fuente, la transformacion y ambas huellas.
21. La URL temporal caducada, reutilizada, dirigida a otra version o presentada
    fuera de su contexto no permite descargar.

## Base normativa y tecnica

- Ley 40/2015, articulos 41 y 42, actuacion administrativa automatizada, sello
  electronico y CSV:
  <https://www.boe.es/buscar/act.php?id=BOE-A-2015-10566>.
- Real Decreto 203/2021, articulo 21: unicidad, vinculacion, cotejo directo y
  gratuito, integracion preferente del CSV y URL en todas las paginas, nueva
  identificacion ante modificaciones, acceso y conservacion:
  <https://www.boe.es/buscar/act.php?id=BOE-A-2021-5032>.
- Norma Tecnica de Interoperabilidad de Documento Electronico: contenido,
  firma, metadatos, identificador y CSV:
  <https://www.boe.es/buscar/act.php?id=BOE-A-2011-13169>.
- Digital Signature Service de la Comision Europea, version estable y
  documentacion de PAdES, firmas multiples, validacion y ataques de sombra:
  <https://ec.europa.eu/digital-building-blocks/sites/display/DIGITAL/Digital+Signature+Service+-++DSS>
  y
  <https://ec.europa.eu/digital-building-blocks/DSS/webapp-demo/doc/dss-documentation.html>.

Antes de activar CSV en produccion, Secretaria, Archivo, Proteccion de Datos y
Seguridad deben aprobar su ambito, organos responsables, sede de cotejo,
politica de acceso, plazo de disponibilidad y conservacion. Asesoria Juridica
debe determinar el instrumento formal aplicable a la Diputacion. Esta
especificacion no sustituye esa aprobacion.
