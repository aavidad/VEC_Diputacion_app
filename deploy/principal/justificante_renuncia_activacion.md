# Activación del justificante de renuncia de Bolsa

Esta guía describe una entrega pendiente de integración. Su alcance es
únicamente el PDF de renuncia justificada que la persona aspirante aporta desde
Mi Bolsa con recibo y que una técnica de RRHH con permiso vigente consulta y
descarga desde la ficha del candidato. El seguimiento muestra la actuación.
La persona aspirante conserva el original: Mi Bolsa no ofrece una descarga
posterior de su PDF. El límite es de 16 MiB en el cliente; la API rechaza con
413 un cuerpo demasiado grande. El documento debe superar el antivirus antes
de quedar disponible.

## Estado de la entrega

El código y las migraciones de esta rama no acreditan una instalación. La PR
recoge las revisiones independientes, las huellas SQL y las pruebas de arranque,
recorrido en Chrome y recuperación después de reiniciar. La precondición de
admisiones descrita más abajo sigue pendiente de su productor gobernado.
Ningún paso de esta guía autoriza a probar ficheros sueltos en cidonia.

## Preparar una copia y fijar el orden

Dirección congela el árbol fuente y ensaya la misma candidata que instalará.
Comprueba el estado de migraciones en la copia y usa la lista
`deploy/principal/lista_sql_codexo_justificante_limpio_20261015.txt` para
documentar cada archivo, orden y huella antes de ejecutarlo. No se reaplica
una migración ya instalada.

La lista incluye primero los prerrequisitos de gobierno ADMIN ya integrados,
AD266 y AD282. Permiten preparar el catálogo y el acto del rol con claves y
confianza de desarrollo de 90 días. Se cotejan sus poshuellas antes de omitir
una UP ya instalada. El plan técnico de aplicación conserva la ventana breve
exigida por su protocolo; no es la vigencia del permiso publicado.

El orden de dependencia es AD277 → AUT95 → D20 → M7 → creación de los LOGIN
documentales sin membresías → D22 → membresías gobernadas → B119 →
CT222. AUT99, AD281 y D23 corrigen la autoridad externa de los actos propios y se intercalan según las
precondiciones y el orden exacto de la lista SQL. AD284 se instala después de
D23: solo el aspirante consume las claves y la auditoría de la superficie externa;
RRHH conserva las internas. AUT100 incorpora únicamente el LOGIN y grupo
documentales exactos al revalidador de sesión externa que necesita AUT99. Antes de
instalar se cotejan los números reservados, los roles y las funciones presentes
en la copia. La creación de LOGIN técnicos y sus credenciales queda en el
circuito de DBA; los secretos no entran en Git.

La base documental conserva metadatos, historia y recibos; el almacén local
custodia los bytes. B119 vincula el documento con la respuesta de renuncia.
La descarga de RRHH exige una decisión de autorización para esa operación,
consumida por su función nominal, y deja constancia de la lectura. CT222 acota
la frontera con el expediente de Contratación temporal. AUT99, AD281 y D23 se han ensayado en una copia y tienen revisión independiente.
AD284 corrige el consumo de la confianza externa y AUT100 la sesión nominal.
D24 conserva el límite de 256 caracteres mediante una validación compatible
con PostgreSQL. D25 usa la candidatura como ámbito de la huella de preparación,
manteniendo a Persona como principal y titular. B123 desambigua la clave primaria
de esa preparación y conserva la recuperación de la misma operación. D25 y B123 van
después de D24, y se instalan una sola vez.
B124 y D26 alinean la lectura con el ámbito técnico; B125 cierra las dos
lecturas preliminares al LOGIN de Bolsa. D27 exige una autorización documental
vigente antes de consultar la relación, y coteja el ámbito real en la misma
transacción. Se instala la lista completa antes de habilitar la lectura; las
versiones intermedias no constituyen una candidata de activación.

AD287 registra la consulta propia previa al perfil en la auditoría común. CA46
limita los perfiles a la cuenta autenticada; IS21 revalida esa sesión y AUT101
selecciona un único perfil propio cuyo rol publicado permite la descarga con
garantía alta. Sin perfil válido o con varios candidatos, deniega. El
certificado debe estar registrado en el inventario interno común de cuentas.
El fichero documental no elige la identidad ni el perfil. La selección se liga
a esa petición y no modifica las fronteras de Contratación temporal.
El formulario envía el PDF por POST idempotente; al reintentar
conserva el mismo archivo y la misma clave. Solo registra la renuncia después
de recibir un documento admitido.

## Precondición pendiente de activación

Falta el productor gobernado de admisiones de identidad externa. La tabla
`vec_identidad_externa_v1.llamante_autorizacion` necesita las admisiones nominales
de los consumidores, pero el producto aún no ofrece el acto para publicarlas.
Esta entrega no incluye INSERT de esas filas ni acredita su aprobación.
Los ajustes SQL de datos ficticios realizados en la copia solo permiten ensayar
el recorrido; no completan la activación. Dirección mantiene esa tarea separada.

## Configuración y gobierno

Se preparan fuera de Git los materiales privados que lee la composición
externa de justificantes y el almacén: `externo/documentos/justificantes-bolsa.json`,
sus claves, la conexión documental, el directorio de custodia y el servicio de
antivirus con su confianza TLS. Se usan ficheros privados con permisos `0600`
y LOGIN distintos según la función. Los valores concretos se derivan del
inventario de la copia; no se copian credenciales ni rutas de otra instalación.

Para la lectura interna se prepara
`identidad/bolsa-justificantes-rrhh.json` dentro del material privado de
desarrollo, con **solo** estas tres claves:

```json
{
  "version": 1,
  "superficie": "interna_corporativa",
  "dsn_documentos": "<DSN privado del LOGIN documental de RRHH>"
}
```

El valor de `superficie` coincide con el contrato Go vigente. El DSN corresponde al LOGIN técnico
`vec_interno_documentos_bolsa_rrhh`, miembro únicamente del grupo documental
previsto. El fichero no contiene cuenta, certificado, persona ni perfil
asignado para elegir qué técnica puede consultar. Cada petición identifica a
la persona y comprueba su permiso vigente por rol.

La confianza externa se prepara con `preparar-portal-externo` y su LOGIN
publicador nominal, según `deploy/principal/confianza_externa_hz38.md`.
El inventario de este corte conserva las audiencias anteriores y añade
`bolsa_justificantes`: en total son 28 claves. Ese DSN administrativo nunca se
monta en la web. En el proceso de desarrollo, el montaje interno registra sus
dos claves por el protocolo de gobierno existente y toma las coordenadas
publicadas; una configuración ausente o incorrecta deja solo esa descarga
cerrada.

La publicación de acciones, versiones de rol y membresías personales se hace
por los actos de Administración gobernados, con dos administradores distintos
cuando corresponda. Se conservan el catálogo y las concesiones existentes; se
añade la acción específica para consultar o descargar el justificante. No se
insertan ni modifican directamente filas de datos, asignaciones o permisos.
Los datos ficticios necesarios para el recorrido se preparan con la semilla o
el guion privado de datos, fuera de esta activación.

Administración arranca con el inventario `metadatos_v1` según
`administracion_arranque_f0_02_20261015.md`. Si HTTPS usa un puerto distinto de
443, `VEC_ADMIN_HOST` incluye ese puerto; la petición y su origen deben coincidir
con esa dirección. El nombre DNS del certificado y el campo `host_admin` de
la política conservan su formato sin puerto. Cada cambio de estos ficheros o
variables se comprueba con reinicio vigilado y la configuración anterior
disponible para volver atrás.

La fuente activa del corte es
`deploy/principal/fuentes/bolsa_justificantes_renuncia_v3.json`. Reúne las dos
acciones propias del aspirante, con su contenido anterior, y la versión 2 de
la descarga de RRHH, con las dimensiones `unidad_ref` y `ambito_ref`. Los
archivos v1 y RRHH v2 se conservan como antecedentes; no se publican juntos en
el paquete activo porque duplicarían la misma acción. El nuevo paquete conserva
las demás fuentes y el censo vigente de roles.

La publicación del catálogo se realiza por la herramienta nominal
`vec-catalogo-acciones-admitir`, con plan y configuración DBA protegida que
aprueban sus bytes exactos. Esa aprobación es técnica; no acredita una firma
personal o legal. Antes de aplicarlo se valida el catálogo con los tipos
oficiales Go y se comprueba que el canon coincide con los bytes del plan.

La versión del rol se publica por el acto genérico vigente desde AUT92,
según `deploy/principal/habilitacion_rol_bolsa_rrhh_generico_20261013.md`.
Una persona ADMIN propone una sola entrada, la descarga de RRHH de la fuente
v3, y otra persona ADMIN distinta confirma la propuesta en
`/api/admin/perfiles/v1/gobierno-version-bolsa-rrhh/cierres`. Se conservan la
versión esperada, las concesiones anteriores y los ámbitos de cada asignación.
La nueva concesión exige garantía **alta**, igual que las acciones propias.
El recibo, las dos decisiones de autorización y el avance de las asignaciones
se comprueban antes de habilitar la consulta. La concesión ficticia previa de
la copia no acredita este acto. No se cambia el perfil para sortear el ámbito
ni se crea un perfil documental por persona.

Bolsa comprueba la bolsa y la participación como atributos del recurso. La
preparación obtiene una propuesta de ámbito de la asignación nominal vigente,
consume su propia autorización y solo devuelve datos tras cotejar la relación
real. Una discrepancia revierte la transacción. Las decisiones posteriores de
lectura y entrega repiten esa preparación y comprueban documento y ámbito. Una
técnica de otro ámbito recibe una denegación, sin conocer si existe el PDF.

## Ensayo de la candidata completa

1. En la copia, instalar SQL en el orden revisado, preparar configuración y
   ejecutar los actos ADMIN. Guardar acuses, versiones y huellas sin secretos.
2. Arrancar sin el material propio de justificantes. Comprobar que VEC atiende
   las demás funciones y que solo estas rutas responden 503. El registro debe
   indicar etapa y clase segura, sin DSN, claves ni datos personales.
3. Añadir los ficheros privados y reiniciar de forma vigilada. Comprobar que la
   función queda disponible y que el resto conserva su estado.
4. En Chrome del sistema, entrar como aspirante ficticio, subir un PDF de hasta
   16 MiB desde Mi Bolsa, confirmar una renuncia justificada y conservar el
   recibo. Comprobar que no aparece una descarga propia del PDF. Probar el
   rechazo de un archivo mayor y el 413 de la API.
5. Entrar como Carmen, con el permiso vigente publicado en su rol. Desde la
   ficha del candidato, consultar y descargar el mismo PDF y comprobar el
   seguimiento y la auditoría de lectura. Intentar la misma descarga con una
   cuenta sin ese permiso y comprobar el 403. Esta negativa no acredita por sí
   sola el rechazo de una técnica con permiso en otro ámbito.
6. Reiniciar aplicación y PostgreSQL en la copia. Repetir la consulta del
   recibo, la lectura autorizada y la descarga. Verificar también el rechazo
   seguro de un PDF que el antivirus no admita.

Una incidencia se corrige en la candidata, se congela de nuevo y se repite el
ensayo completo. Solo Dirección instala en cidonia el contenido exacto que
superó estas comprobaciones, con reinicio vigilado y vuelta atrás preparada.
