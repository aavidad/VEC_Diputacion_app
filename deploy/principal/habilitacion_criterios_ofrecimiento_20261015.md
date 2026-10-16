# Criterios de ofrecimiento de Bolsa: activación del primer corte

Esta entrega añade diagnóstico y detención de la selección. El acto para acreditar
la excepción de vacante pertenece al corte siguiente: esta entrega no ofrece un
puesto a quien trabaja por el mero hecho de marcar una casilla.

## Orden de instalación

La publicación de ofertas y el montaje de política de #1145 ya están en la
principal. B118 parte del cuerpo instalado tras B117. Para las migraciones de
autorización compartidas, dirección fija esta secuencia: la lista completa de
#1147, con AD277, AD281, AD284 y AD287; AD276 de este corte; AD278 de #1146;
AD283 de #1155; AD279; AD288 y, al final, AD273. AD282 pertenece a los prerrequisitos de gobierno
de esa lista. Este corte no sustituye la lista conjunta congelada por dirección.
AD276 coteja la restricción instalada tras AD287 y conserva tanto la confianza
externa como la consulta previa al perfil de RRHH. Una preimagen distinta
detiene la instalación, sin modificarla para saltar el control.
La lista propia de R es
`lista_sql_codexr_criterios_ofrecimiento_20261015.txt`: los prerrequisitos AD266
y AD282, CT221, B118 y AD276, además de los anteriores. AD266 ya está en
la principal de código y añade al lector
de gobierno la siguiente secuencia global. Se instala solo si está pendiente;
sin ella, el programa actual de preparación de permisos rechaza la activación.
No se reaplica ninguna migración sobre una base con historia.

La copia de ensayo se restaura con los roles y las ACL exactas de la principal,
incluidas las ACL de base y tipos. Los puertos, certificados y material privado
pertenecen a la copia. La instalación no crea LOGIN ni concede permisos al arrancar.

## Permisos

1. Admitir por el gobierno vigente el descriptor de
   `fuentes/bolsa_exclusiones_ofrecimiento_rrhh_v1.json`. Su acción es
   `bolsa.rrhh.exclusiones_llamamiento.consultar`, para un llamamiento y una bolsa
   exactos, con la finalidad de seguimiento y su lista cerrada de campos.
2. Admitir la versión 2 de `fuente:bolsa:origen_ct_rrhh`, desde
   `fuentes/bolsa_origen_ct_rrhh_v2.json`. Añade expresamente la copia de la regla aplicada y
   los criterios y los eventos utilizados. La versión 1 conserva su definición.
3. Dos personas administradoras distintas publican la versión del rol de RRHH
   y avanzan las asignaciones aprobadas por los actos existentes de propuesta y
   cierre. Se conservan el perfil, la persona, los ámbitos y las demás concesiones.
   El procedimiento es el de `habilitacion_b1_convoca_20261009.md`, con estas dos
   fuentes y sin volver a admitir la fuente de carga.
4. Releer la concesión vigente antes de continuar. Sin el permiso de motivos,
   el seguimiento conserva sus personas y llamadas y explica que no se pueden
   consultar los motivos. Sin la concesión ampliada de origen, la selección
   deniega: un permiso anterior no abre campos nuevos.

Las definiciones de fuente ya publicadas conservan su vigencia y su huella de
un año; no se reescriben. El material nuevo de desarrollo debe durar entre sesenta
y noventa días. AD282 permite preparar claves y confianza nuevas por noventa
días mediante el procedimiento existente; el plan técnico conserva su plazo
breve. Una publicación administrativa caducada exige otra publicación aprobada:
el arranque no la renueva automáticamente. No se alargan claves anteriores ni se
incluyen credenciales o material privado en el repositorio.

## Regla de la bolsa

En la política de ofertas, RRHH pulsa «Añadir criterios para vacantes». El botón
lee el catálogo versionado
`web/static/portal-empleado/modulos/bolsa/criterios-ofrecimiento/v1.json` y rellena
solo el borrador. RRHH revisa la fuente y los parámetros y publica una versión
por el acto existente; el navegador conserva y muestra su recibo.

El catálogo cita el art. 9, apartado cuarto (numerado 3 en el BOP n.º 9 de
16/01/2026). La causa de vacante se comprueba separadamente de la modalidad.
Los nueve meses y el máximo de treinta y seis meses son valores de esa versión.
La reducción del plazo exige hechos acreditados; ninguna preferencia personal
hace elegible a una persona ni se registra como renuncia.

La regla distingue el supuesto de aplicación del descuento y los tipos de
periodo descontable. El catálogo propone `programa` para el primer dato, como
criterio provisional pendiente de RRHH, y conserva `acumulacion_tareas` y
`sustitucion` para el segundo. La cita normativa se muestra aparte. Una política
anterior sin el dato del supuesto mantiene la comprobación pendiente; este
corte no activa el cálculo ni ofrece automáticamente a quien sigue trabajando.

Para una primera política se completan también plazo, adjudicación y plazas con
el catálogo vigente. La regla por sí sola no sustituye las demás partes de la
política. Un catálogo ausente o defectuoso impide únicamente añadir ese apartado.

## Comprobación de la candidata

- Arrancar sin criterios publicados: sesión, bolsas y funciones anteriores
  siguen disponibles. La ausencia de nuevas concesiones cierra sus funciones.
- Publicar una política con criterios desde la pantalla. Recuperar el mismo
  recibo y su versión; una clave con otro contenido debe dar conflicto.
- Ante una incorporación sin fecha o sin hechos suficientes, la selección se
  detiene antes de pasar a otra posición. El aviso identifica posición y versión
  y abre la bolsa; no afirma que se haya creado un llamamiento.
- En el seguimiento de un llamamiento ya creado, consultar los motivos del corte
  original, con versión de regla y hechos por tipo y fecha. Paginarlos. Una
  cuarentena posterior no borra la explicación de ese corte.
- Repetir las consultas después de reiniciar aplicación y PostgreSQL: mismos
  recibos y hechos históricos. Comprobar las denegaciones sin concesión y ante
  otra bolsa o llamamiento, con auditoría nominal y sin datos del recurso ajeno.
- Recorrer escritorio y móvil, en castellano e inglés. Un fallo del bloque de
  motivos conserva los candidatos y las llamadas del seguimiento.

## Evidencias de la candidata local

El montaje final `e065ac39b` incorpora el código de este corte y la corrección
literal de remonte de O #1147 (`4e675f68e`), solo en la candidata de ensayo. El
binario normal conserva SHA256
`6162f2dbff0c308639c605f7dfd5080b42ef90af3d93ce107c4df9d128b89d75`;
no hay instrumentación ni sustituciones de funciones. La corrección de O no se
duplica en #1144: su PR entra antes por el orden fijado por dirección.

El prefijo de ensayo R57 instaló una vez la lista de O hasta AD287 y, después,
B117, CT221, B118 y AD276 finales. Pasaron sus pruebas focales, incluidas
condiciones, hechos, ausencia de criterios y límites entre superficies. Las
fuentes CT221, B118 y AD276 tienen SHA256, respectivamente:

- `8a82e85afbb3df559eef8e92ef34abf1fe1cd6cd4e9204bb28d92b6f3a1de32f`.
- `466c5ce66b3c896d47ecf7eaeab22b670c046854d1954a5e9f375f07bda584ff`.
- `26391d46f93e9de817e8e603be8e38bad0fedf82cf5cadb741e14044272241c7`.

La copia R58 conserva ese esquema, ampliado después por AD278 y AD283, y los
datos de trabajo de R56. El traslado mantuvo las funciones, permisos y
restricciones del destino; las ACL de base y tipos se restauraron desde la
principal con dos revisiones. Una comprobación posterior de solo lectura cotejó
quince cuerpos y sus metadatos: getter CT221, nueve funciones B118, cuatro AD276
y el consumidor central tras AD283. Coinciden con esas fuentes. No se reaplicó
SQL instalado. La fuente anterior de AD276 usada en R56 tenía los mismos cuatro
cuerpos; la final cambia la guarda previa para respetar AD287→AD276.

Las dos concesiones de RRHH se publicaron por propuesta y cierre con dos
administradores distintos; se cotejaron asignación y concesiones conservadas.
El material ADMIN nuevo dura noventa días. Las definiciones de fuente anuales
anteriores quedan como excepción de esta copia, a cargo de R hasta su relevo
por la activación común; no se presentan como material nuevo.

La composición normal incorporó la separación de unidades de #1153 y las
correcciones de referencias, bytes y fechas de este corte. La selección original
del expediente 2026/94011 quedó confirmada mediante el cliente web real, con un
origen Bolsa y un justificante nativo conservados tras reiniciar. La corrección
SQL de ese justificante va separada en #1170 (CT230); la recuperación de su ficha
y la anotación del aviso, en #1178 (CT231). No se afirma que #1144 aislada incluya
esos dos cambios.

El seguimiento histórico se consultó con HTTP 200 en castellano e inglés, a
1440 y 390 píxeles. Mostró las cuatro posiciones omitidas por disponibilidad
posterior, la versión uno de la regla y cero hechos contractuales, sin perder
los candidatos ni las llamadas. La consulta de puestos de Personal usó la
unidad del puesto y conservó la unidad gestora del expediente.

Desde la pantalla se guardó la primera política de Auxiliar de Servicios
Generales con HTTP 201. El botón añadió la propuesta vigente y RRHH conservó
sus valores: supuesto de descuento «Programa temporal», periodos descontables
«Acumulación de tareas» y «Sustitución», y cita del artículo 9, apartado cuarto.
Tras reiniciar aplicación y PostgreSQL, el GET devolvió HTTP 200 con la misma
versión, huella, contenido y recibo. La interfaz seguía indicando que la regla
estaba pendiente de aprobación: no se presenta como una excepción ya acreditada.
La bolsa estaba sin configurar antes de este acto; sesión, portal y funciones
anteriores arrancaron tanto antes como después de guardar los criterios.
El justificante reaparece al volver a la pantalla y se copia desde su botón común,
sin mostrar la referencia opaca. Chrome a 390 píxeles comprobó copia real con
teclado, cabecera legible y ausencia de desbordamiento horizontal.

El 503 que impedía abrir la política procedía de una comprobación por número de
versión del rol. Se sustituyó por la concesión exacta publicada para consultar o
publicar, manteniendo la decisión estricta y el consumo nominal. Dos revisiones
independientes y la focal de versión once, permiso ausente y concesiones ajenas
fueron favorables. Semgrep local no encontró hallazgos nuevos; gosec cargó los
385 archivos examinados sin errores y conservó los 78 avisos anteriores, ninguno
en las líneas nuevas.

## Límites del ensayo

La lectura contextual de Auxiliar de Servicios Generales incluye 42 posiciones.
Sus cuatro personas trabajando, sin hechos contractuales, quedan pendientes de
comprobación y sin turno elegible. Los datos antiguos también carecen de varios
vínculos de persona; no se deduce que estén disponibles por esa ausencia. Esa
comprobación fue un SELECT de diagnóstico, no una lectura desde el portal ni
una selección. La pantalla general de candidatos conserva su lectura general;
el motivo contextual se muestra en la selección y en el seguimiento de su corte.

El freno desde CT se comprobó en otra copia desechable, R59, del mismo caso
2026/94011. Se retiraron exclusivamente sus efectos ficticios de selección y
comunicación y se hizo vigente una fila de semilla «trabajando». Dos revisiones
aprobaron esa preparación antes de ejecutarla. Los doce conjuntos protegidos, incluidos expediente, solicitud, análisis,
decisiones y política, conservaron sus huellas. El guion no modificó firmas ni
asignaciones de rol; no se compararon mediante una huella previa propia.
No se fabricaron actos ni se cambiaron las decisiones para otro contexto. La historia central
anterior de esa copia queda referida a efectos retirados y no acredita el ensayo
nuevo. La copia R58 y sus justificantes confirmados permanecieron intactos; su
aplicación y base estuvieron detenidas durante el ensayo de R59 para impedir
mezclas de conexiones.

En la primera posición había una sola persona vinculada, cero hechos de contrato
y ninguna cuarentena. La política uno era válida. Desde la ficha, una clave nueva
generada por la interfaz devolvió HTTP 409 `comprobacion_ofrecimiento_pendiente`:
la pantalla identificó posición uno, comprobación por RRHH y versión uno, y aclaró
que no había creado el llamamiento. Los cuatro contadores comprobados del caso permanecieron a cero:
selección O6, origen y comunicación del expediente, y llamamientos de la bolsa
Administrativo. No son los recuentos globales de la copia. Los doce conjuntos protegidos
conservaron sus huellas. Las lecturas previas de CT, Bolsa y Personal dejaron
asientos durables; no se atribuye una decisión de selección al rechazo anterior
al acto. La correlación HTTP se genera aparte: la atribución de esos asientos es
por recurso, secuencia y ventana de tiempo, sin afirmar una igualdad inexistente.

El primer montaje reprodujo una cancelación del GET al remontar la misma vista.
Se reutilizó la corrección de O y su prueba, conservando la cancelación al salir.
En la candidata final, una sesión limpia cargó su versión de recursos correcta:
el POST mantuvo el 409 y «Ver la bolsa» abrió con teclado el listado real. Los
cuatro contadores siguieron a cero. No se convirtió en éxito el 401 de otra
consulta ni se cambió una guarda. Un cruce de bolsa y llamamiento había cerrado
la lectura sin datos con 503: no se cita como una denegación formal 403.
El cambio de idioma directo también tuvo un rechazo transitorio en el montaje
anterior; recargar permitió consultar. Estos límites quedan separados del freno
contextual comprobado.

Falta el acto propio para que RRHH acredite los hechos (corte siguiente) y la
confirmación de la duda 165 sobre el descuento. Las preferencias de modalidad
corresponden a su corte propio y nunca excluyen del llamamiento directo.
Este ensayo no acredita todo el recorrido de FLUJO_APP §4.0, AD279, AD288 ni la
subida a cidonia. Dirección integra y congela su candidata conjunta.

## Comprobación después de integrar la principal con los justificantes

La cabeza de integración `35f23ed7f` conserva las tres SQL de este corte con
las mismas huellas. La composición propia `70a29941d` incorpora además CT230
y CT231 y usa el binario normal SHA256
`6d26fc8910751b60f7f70a8b393772e9f410514ebda5aacf67f9affc1659e685`.
Su cohorte web es `20261016-r1-seleccion-pendiente-v8`; conserva la corrección
de remonte y las funciones de justificantes integradas por la principal.

Desde Chrome nuevo, la misma solicitud y clave recuperaron el resultado
original, también después de reiniciar aplicación y PostgreSQL. Se mantuvieron
un origen, una confirmación y los bytes del recibo y artefacto. La política
conservó versión, huella, contenido y recibo; portal, sesión y comprobación de actividad respondieron
200 una vez abierto el servicio. El primer sondeo inmediato al arranque no
había encontrado todavía el puerto y se conserva como tal, no como un 200.

En R59, sin repetir la preparación de datos ni instalar SQL, otro intento desde
la ficha devolvió 409 en posición uno y política uno. «Ver la bolsa» abrió el
listado con teclado. Los cuatro contadores del caso siguieron a cero y los doce
conjuntos protegidos conservaron sus huellas. R58 permaneció detenida durante
esa comprobación y volvió a arrancar después de detener R59. La resolución de
conflictos web tiene revisión independiente favorable y pruebas focales; este
contraste no amplía el alcance al recorrido completo ni a las dependencias
pendientes del núcleo de autorización.

## Corrección tras la revisión SQL independiente (16/10)

La revisión SQL de #1144 dio NO-GO. Cambios en CT221 y B118; AD276 no cambia
y la huella del consumidor central sigue en `cfcacf14…5960`:

- La consulta de motivos exige que la unidad de la orden sea la unidad gestora
  del llamamiento, leída con bloqueo compartido, y que el ámbito sea el del
  conjunto de gestión de RRHH, igual que B124. Si no coinciden, deniega con el
  mismo código 42501 y sin datos.
- La reserva solo coteja los eventos de cada posición cuando la lectura los
  trae. Si la lectura no los trae, la reserva exige que la política no tenga
  criterios.
- Un contrato que no coincide con su origen ya no lanza un error que atasca el
  relevo de CT. Queda en el histórico sin participación ni bolsa y entra en la
  cuarentena existente. Si llega repetido, sigue en cuarentena, y la persona
  de ese llamamiento queda pendiente de comprobación.
- Se quitan las comprobaciones de `current_user` que no podían fallar y la
  lectura del orden contextual fija `row_security` y `statement_timeout`.

Las huellas SHA256 nuevas son estas: CT221
`3b26d68056f1b2a2bfd0bf2689598441934c8bf3d2962592d850a0549adf0db9` y B118
`f36c7a84ee88b55b6aa0a262833a36c7a75294cd62f694a748fefea08d9a6710`. AD276
conserva `26391d46…41c7`. Las de los ensayos R57 a R59 corresponden a las
fuentes anteriores.

Se ensayó en una copia propia de HZ37 con lo que faltaba de main (incluida la
lista de #1147) y después CT221, B118 y AD276, una sola vez. Las cinco pruebas
SQL de la PR pasan. La del fixture añade tres comprobaciones: otra unidad, otro
ámbito o una orden sin unidad se deniegan, y la unidad propia llega hasta el
acuse nominal. También comprueba que el contrato divergente queda en
cuarentena. Al reaplicar, CT221, B118 y AD276 se rechazan por preimagen.
La prueba usa `vec_bolsa_llamamientos_desarrollo`, miembro de
`vec_bolsa_llamamientos_ejecutor`, como dato de la copia.
