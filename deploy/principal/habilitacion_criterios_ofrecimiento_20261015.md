# Criterios de ofrecimiento de Bolsa: activación del primer corte

Esta entrega añade diagnóstico y detención de la selección. El acto para acreditar
la excepción de vacante pertenece al corte siguiente: esta entrega no ofrece un
puesto a quien trabaja por el mero hecho de marcar una casilla.

## Orden de instalación

La publicación de ofertas y el montaje de política de #1145 ya están en la
principal. B118 parte del cuerpo instalado tras B117. Para las migraciones de
autorización compartidas, dirección fija esta secuencia: la lista completa de
#1147, con AD277, AD281, AD284 y AD287; AD276 de este corte; AD278 de #1146;
AD279; AD288 y, al final, AD273. AD282 pertenece a los prerrequisitos de gobierno
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

## Evidencias y límites actuales

Un ensayo anterior instaló CT221, B118 y AD276 una vez en una copia propia y
pasó las comprobaciones focales de permisos, ausencia de política, ambos modos
de inicio, criterios, hechos, instantáneas y separación de superficies. Ese
ensayo no acredita la cadena compartida posterior a AD277 ni la última
ampliación opcional del validador B118. Esta ampliación se comprobó por separado
en una función temporal, sin reaplicar la migración instalada.

En la copia se admitió el catálogo con auditoría y se publicaron las dos
concesiones de RRHH por propuesta y cierre de dos administradores distintos.
Se comprobó la nueva asignación y la conservación de las concesiones anteriores.
La política se publicó mediante el cliente web real y se recuperaron su versión,
fecha y recibo después de reiniciar. Falta repetir la publicación con la última
regla desde la navegación del portal y ensayar la cadena final completa.

La consulta histórica de #1150 ya está integrada. El puente de unidades de
#1153 se incorpora a esta candidata para ensayar el consumidor con el permiso
ampliado de origen. Un binario anterior que exige los 38 campos originales
rechaza correctamente la concesión ampliada de 40; ese rechazo no prueba la
selección conjunta. La PR conserva pendientes el recorrido de pausa y motivos,
los recibos del llamamiento y la recuperación tras reiniciar sobre la candidata
final. No está lista para instalarse.
