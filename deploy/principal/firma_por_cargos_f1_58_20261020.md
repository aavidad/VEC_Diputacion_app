# F1-58: firma y consulta por cargos

Estado provisional: informe con dos firmas verificadas y resolución con dos
de sus tres firmas. Los PDF y sus registros se consultan desde la ficha. Faltan
Secretaría, los cuatro documentos pendientes de firma y el cotejo tras reiniciar.
La revisión de X y la comprobación automática siguen pendientes.

## Piezas que se reutilizan

La ampliación llamada «AUT98» en el consenso del 16 de octubre se integró como
AUT103. AUT98 estaba reservada por otro equipo. No debe crearse otra migración
para repetir esa ampliación.

| Necesidad | Implementación existente |
| --- | --- |
| Publicar cargos con control de versión | AUT53, AUT87 y AUT91; `vec-publicar-cargos-firma` |
| Añadir la consulta de firmas anteriores | AUT103; operación `consultar_detalle_selectores` |
| Recuperar la firma original con permiso propio | AUT103, AD286 y CT228; operación `recuperar_r5` |
| Limitar los selectores a la unidad autorizada | AUT102 y CT229 |
| Mostrar el detalle y sus selectores | Adaptador PostgreSQL de consultas RRHH, CT225 y CT229 |

Hay dos versiones distintas que no deben confundirse: el esquema del plan de
publicación (`vec.admin.cargos-firma.plan.v2`) y la versión de cada rol
(`rol:ct_cargo_…_consulta_dev:v2`). El plan nominal de los cinco pasos de firma
es otro catálogo, publicado con sus propios recibos.

El lector ampliado de CT229 admite los cuatro roles de consulta v2. El
adaptador elige esa lectura cuando la decisión concede exactamente
`selectores_recuperacion_firmas`. La llamada pasa por CT225 y llega al motor
de detalle sin pasar por CT216. El lector ordinario de CT216 conserva su
contrato. AUT93 y CT219 comprueban el perfil de firma, separado del de consulta.

## Preparación

Crear una copia propia siguiendo `LEEME_copia_local.md`, con `--modulos`,
`--rrhh-original` y `lista_sql_main_tras_hz37.txt`. Copiar los preparadores
cedidos a material privado propio y ajustar las rutas, los puertos y el nombre
de la instancia antes de ejecutarlos. Nunca usar conexiones de otra copia.

Preparar la firma con la orden `firma` de esa fábrica y un manifiesto privado.
Las etapas y sus requisitos están en el apartado 11 de la misma guía y en
`candidata_hz39/firma/LEEME.md`. Mantener los recibos de los actos realmente
ejecutados en esta copia. Las asignaciones cargadas como datos ficticios se
identifican como tales; no constituyen actos de Administración.

## Publicar los sucesores

1. Exportar las cuatro versiones de consulta publicadas y sus recibos de origen.
   Conservar una copia de sus documentos y huellas antes de preparar sucesores.
2. Preparar con el CLI existente un fichero de cargos de esquema v2. Para cada
   rol, usar versión 2 y la huella exacta de su versión 1 en
   `version_anterior_sha256`. Conservar sus competencias y las operaciones de
   cuadro y descarga; sustituir únicamente `consultar_detalle` por
   `consultar_detalle_selectores`. Ambas no pueden coexistir en el mismo rol.
3. Generar el plan canónico con `vec-publicar-cargos-firma preparar`. Aprobar
   sus bytes exactos y aplicarlo con el LOGIN segregado de cargos y su
   configuración de aprobación. Exigir un recibo permitido; una respuesta
   incierta se concilia con la misma operación antes de repetir.
4. Comprobar las cuatro versiones v2, sus controles y el registro como perfiles
   asignables. La versión anterior y su recibo conservan sus bytes originales.
5. En la copia, suceder las asignaciones de consulta conservando persona, cuenta,
   perfil, ámbitos y fecha final. Publicar un rol no modifica por sí solo
   las asignaciones. El lote ordinario requiere referencias nuevas al otorgar;
   no permite sustituir en silencio el rol de un perfil ya existente.
   Dirección autorizó el 20 de octubre cargar esta sucesión como semilla ficticia
   con procedencia `identidad:desarrollo:no-autoritativa`. No acredita actos de
   Administración; F1-106 retirará esta excepción.
6. Cotejar `PerfilConsultaCTRef` de las cuatro personas en `identidad/firma-vec.json`
   con los perfiles resultantes. Conservar los perfiles de firma, Documentos y
   circuito. Regenerar las admisiones si cambian sus referencias o huellas y
   reiniciar de forma vigilada.

La preparación y la publicación usan el mismo CLI existente. `MATERIAL` debe
ser una carpeta privada propia con rutas absolutas; `REPOSITORIO` y `BINARIO`
identifican la base de código y su ejecutable comprobados. La conexión y la
aprobación pertenecen a esa copia, con permisos de archivo 0600. No trasladar
los acuses ni las huellas de otra copia.

```sh
"$BINARIO" preparar --cargos "$MATERIAL/cargos-v2.json" \
  --plan "$MATERIAL/plan-v2.json" --caduca 2h --idioma es \
  --textos "$REPOSITORIO/web/static/textos/es/admin-cargos-firma.json"
"$BINARIO" aplicar --plan "$MATERIAL/plan-v2.json" \
  --conexion "$MATERIAL/conexion-v2.json" \
  --aprobacion "$MATERIAL/aprobacion-v2.json" \
  --acuse "$MATERIAL/acuse-v2.json" --timeout 30s --idioma es \
  --textos "$REPOSITORIO/web/static/textos/es/admin-cargos-firma.json"
```

El fichero de aprobación contiene `huella_plan_sha256`, calculada sobre los
bytes del plan. Su provisión se coteja en la conexión segregada antes de aplicar.
El nombre del acuse debe estar libre. Si el resultado es incierto, conservar
el plan y la operación; conciliar el recibo antes de volver a enviar.

Los perfiles de consulta conservan sus tres dimensiones: organización, clase
de ámbito y referencia de ámbito. La unidad histórica se comprueba con la
asignación de firma de la misma persona; no se sustituye ese control por un
permiso de consulta.

Conservar una fecha de emisión que no sea posterior al inicio de vigencia. El
validador de dominio rechaza esa combinación antes de usar la asignación.

No prolongar una asignación más allá de la retirada de su certificado. La autoridad, las admisiones y las asignaciones
nuevas de desarrollo duran de 60 a 90 días. Dirección aclaró el 20 de octubre
que se conservan los certificados X509 del kit de 397/400 días; esos certificados
no prolongan la autoridad nominal ni la asignación. La caducidad breve del plan de
publicación es el plazo para aprobarlo y aplicarlo, distinto de la vigencia de
los cargos publicados.

## Origen de la consulta histórica

La recuperación y la descarga son lecturas distintas. La recuperación exige
la fila técnica de origen AD172 que ya figura en
`origen_consumos_ad172/ternas.tsv`: audiencia
`vec_contratacion_temporal.firmas_r5.recuperar.v2`, operación
`contratacion_temporal.documento.firmas_r5_v2.recuperar`, canal
`interna_corporativa`, proceso `vec-server` y grupo
`vec_contratacion_temporal_ejecutor`. El LOGIN debe ser el runtime real de CT.

El apartado «Qué queda fuera» de `origen_consumos_ad172/README.md` explica el
cotejo separado de esta fila. Preparar una operación privada que seleccione
solo esa terna y exija el identificador de la copia y la huella exacta del
núcleo instalado. Cotejar en ese texto el perfil, grupo, audiencia y operación;
no ampliar la lista global por revisar una sola rama. Dos revisiones
independientes deben comprobar esa candidata antes de escribir.

En este ensayo, el paquete privado selecciona el bloque `recuperacionh`:

```sh
VEC_ORIGEN_MOTOR=docker \
VEC_ORIGEN_PG_CONTENEDOR="$CONTENEDOR_PROPIO" \
VEC_ORIGEN_BLOQUES=recuperacionh \
  bash "$MATERIAL_ORIGEN/ejecutar.sh" --ensayo
VEC_ORIGEN_MOTOR=docker \
VEC_ORIGEN_PG_CONTENEDOR="$CONTENEDOR_PROPIO" \
VEC_ORIGEN_BLOQUES=recuperacionh \
VEC_ORIGEN_AD172_APLICAR=SI-REVISADO \
  bash "$MATERIAL_ORIGEN/ejecutar.sh" --aplicar
```

El ensayo debe terminar en `ROLLBACK` con inventario idéntico. La aplicación
puede añadir solo esa fila y debe conservar los LOGIN y las filas anteriores.
No crea funciones, concesiones humanas ni membresías. No ignorar una huella
distinta ni corregir una fila inmutable mediante borrado.

## Comprobación desde la ficha

Usar GrxFirma 0.0.106 o posterior tanto para firmar como para cofirmar; registrar
la versión y la huella del ejecutable utilizado. No reutilizar un PDF firmado
con la 0.0.105. El verificador propio debe estar activo y VEC solo muestra la
firma después de una verificación positiva.

La receta de firma por línea de comandos usa el PDF exacto de la comprobación
previa de VEC. Cotejar su huella antes de firmar; nunca regenerar otro PDF para
el registro de ese paso. Para el primer paso, usar `firmar`; para los siguientes,
`cofirmar` sobre la entrada que entrega VEC con las firmas anteriores.

```sh
"$GRXFIRMA" -modo-cli -contenedor-p12 "$CERTIFICADO_P12" \
  -contrasena-stdin -entrada "$PDF_ENTRADA" -salida "$PDF_FIRMADO" \
  -formato pades -accion "$ACCION" < "$CONTRASENA_PRIVADA"
```

El certificado y el archivo de contraseña son privados; la contraseña entra
por la entrada estándar. El CLI no registra la firma. Enviar su resultado a la ruta nominal existente
`firmas-documento/registro-vec`, con el certificado de la misma persona y los
campos `expediente_ref`, `version_expediente`, `documento`, `paso_orden`,
`original_ref`, `original_version`, `firmado_base64` y `clave_idempotencia`.
Las referencias y versiones proceden de la comprobación previa de ese paso.
Exigir el recibo del registro y verificación positiva; una respuesta incierta
se concilia antes de repetir. Consultar y descargar después desde la ficha.

Esta vía distingue firma CLI, registro nominal y recuperación en Chrome. No
acredita por sí sola completar el lanzamiento automático del firmador desde
el botón del navegador.

Seguir el caso de `docs/producto/FLUJO_APP.md`, apartado 4.0: Técnico y Jefatura
firman el informe antes de Intervención; Jefatura, Diputación y Secretaría
firman la resolución, en ese orden. Son cuatro personas distintas. Los otros
cuatro documentos conservan su firma pendiente.

Antes de la primera firma de la resolución, el Técnico pulsa «Preparar
resolución para firma» y descarga el PDF preparado. Su cargo permite crear
y custodiar el original; no firma la resolución. Jefatura usa después su
propio certificado para leer ese original y aportar la primera firma. No
añadir permisos de creación al cargo de Jefatura para saltarse este paso.

Comprobar el lector ordinario con un perfil vigente que no se haya sucedido,
y el ampliado con los cuatro perfiles de consulta sucedidos. Estos devuelven
juntos los selectores y su recibo adicional. La versión v1 queda en la historia;
la sucesión no permite seleccionarla como perfil activo.
Pulsar «Consultar firma» vuelve a autorizar la lectura y devuelve el recibo del
registro original. Descargar los PDF firmados desde la ficha y guardar sus
huellas fuera de Git.

Reiniciar aplicación, PostgreSQL y verificador. Volver a consultar y descargar:
mismas firmas, fechas, recibos y bytes, sin duplicados. La retirada del perfil
o una unidad distinta deben denegar la recuperación. Conservar por separado
el resultado del ensayo SQL, las respuestas HTTP y las capturas de Chrome.

## Activación y retirada

CONFIG NUEVA: ninguna variable ni contrato nuevo. Se activan la fila técnica
de recuperación y los selectores y recursos de firma existentes del apartado
11 de `LEEME_copia_local.md`.

SQL DE PRODUCTO NUEVA: ninguna. La lista completa de main instala las
dependencias existentes una sola vez en la copia nueva. Los datos ficticios
y sus correcciones se cargaron mediante SQL privado, solo en la copia propia.

Sin el complemento, comprobar que la ficha ordinaria sigue disponible. Con él,
comprobar las firmas y su consulta. Un permiso ausente o retirado no se resuelve
abriendo otro lector ni cambiando las guardas. Una publicación ya aplicada no
se revierte ni se reaplica para cambiar su contenido.

## Evidencia de esta copia

Base de código: `fb5200b68`, sin cambios Go, web ni SQL. La fábrica aplicó
las 131 migraciones de la lista completa y obtuvo respuestas 200 de salud,
portal, cuadro de Contratación y bolsas. Las pruebas existentes
`aut103_concesion_consulta_firma_historica.sql` y
`ct229_selectores_recuperacion_detalle_rrhh.sql` pasaron, ambas con `ROLLBACK`.

El CLI compilado de esa base publicó primero los 16 cargos y después los cuatro
sucesores de consulta v2, con sus huellas v1 exactas. Ambos recibos son
permitidos. El segundo envío del mismo plan v2 devolvió el mismo recibo con
`replay: true`; las cuatro huellas v1 coinciden con las exportadas antes.
Una preparación con las dos operaciones de detalle juntas se rechazó y no
escribió un plan.

La aprobación técnica de la copia es un dato declarado del ensayo. Estos recibos
no acreditan la publicación de roles por dos administradores de F1-106. Las
cuatro asignaciones sucesoras están aplicadas, con personas, cuentas y perfiles
cotejados contra la fuente de firmantes. El validador real de Go aceptó las
16 asignaciones de firma y auxiliares y las dos de Administración: 18 válidas.
La firma se ejecuta con GrxFirma CLI y se registra por la API nominal. Ya hay
cuatro registros verificados; el reinicio y la última firma siguen pendientes.
No se da la tarea por terminada.

Los catorce actos nominales dejaron sus recibos reales en la copia: cinco
cargos, cinco enlaces y cuatro certificados. La publicación de certificados
exige fechas UTC con seis decimales; conservar ese formato en su descriptor.
Una petición rechazada no se volvió a enviar hasta comprobar que no había
dejado una fila por su clave ni una cabeza por el certificado.

El plan de cinco pasos se creó con el primer administrador y se publicó con
el segundo. La configuración reúne cuatro firmantes y cuatro admisiones, y
apunta a los recursos propios bajo `material/identidad/firma-recursos`. La
orden `firma` de la fábrica publicó las plantillas v4, activó el verificador
GrxFirma 0.0.107 y reinició aplicación y PostgreSQL. El cuadro de Contratación
y las bolsas respondieron 200 después. Después se registraron las dos firmas
del informe y las dos primeras de la resolución; falta el último paso.

En el entorno privado de Administración, `VEC_ADMIN_HOST` debe incluir el
puerto local si es distinto de 443. La salud de Administración devuelve 204.
Verificar el gobierno con `-acuse acuse-aplicar.json` liga la comprobación al
acto exacto; si el acto ya tiene recibo permitido, no volver a publicarlo para
resolver una verificación pendiente.

La consulta histórica falló inicialmente por origen técnico ausente, mientras
la descarga devolvía el PDF correcto. El guion de origen pasó dos revisiones
independientes sobre el núcleo exacto `b8397a6e…36cc4f`; el ensayo revirtió sin
cambios y la aplicación añadió una sola fila. Después, «Consultar firma»
respondió 200 y conservó el recibo original. La verificación criptográfica es
positiva; `firma_eficaz=false` mantiene separada la confirmación de Firmadoc,
que no se ha producido en este ejercicio.
