# Criterios de ofrecimiento de Bolsa: activación del primer corte

Esta entrega añade diagnóstico y detención de la selección. El acto para acreditar
la excepción de vacante pertenece al corte siguiente: esta entrega no ofrece un
puesto a quien trabaja por el mero hecho de marcar una casilla.

## Orden de instalación

Dirección ha fijado el orden S → R: primero la publicación de ofertas y el montaje
de la política de la PR #1145, después este corte. B118 debe comprobar el cuerpo
instalado tras B117. La lista causal de R es
`lista_sql_codexr_criterios_ofrecimiento_20261015.txt`: CT221, B118 y AD276.
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

Las fuentes de este corte duran un año. El material ficticio del ensayo debe
conservar al menos sesenta días de vigencia. No se incluyen credenciales ni
material privado en el repositorio.

## Regla de la bolsa

En la política de ofertas, RRHH pulsa «Añadir criterios para vacantes». El botón
lee el catálogo público versionado
`web/static/portal-empleado/modulos/bolsa/criterios-ofrecimiento/v1.json` y rellena
solo el borrador. RRHH revisa la fuente y los parámetros y publica una versión
por el acto existente; el navegador conserva y muestra su recibo.

El catálogo cita el art. 9, apartado cuarto (numerado 3 en el BOP n.º 9 de
16/01/2026). La causa de vacante se comprueba separadamente de la modalidad.
Los nueve meses y el máximo de treinta y seis meses son valores de esa versión.
La reducción del plazo exige hechos acreditados; ninguna preferencia personal
hace elegible a una persona ni se registra como renuncia.

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

El ensayo inicial de `e52781edc` instaló CT221 → B118 → AD276 una vez en una copia
propia de HZ37 y pasó las tres pruebas SQL focales de la lista de cambios. El
arranque genuino tardó 1,80 s y Chrome obtuvo sesión y bolsas con respuesta 200.
Ese ensayo precede a B117: no acredita aún la candidata conjunta ni la publicación
de la política desde pantalla. La PR conserva ese trabajo pendiente.


Después se declaró el catálogo y su módulo en los manifiestos internos y se
comprobó su servicio HTTP real. La publicación nominal desde el cliente web
produjo una política versión 1 con recibo. La consulta posterior al reinicio
conservó versión, fecha, recibo y criterios. Esta comprobación usa la operación
real de publicación; sigue pendiente recorrerla desde la navegación que monta S.
