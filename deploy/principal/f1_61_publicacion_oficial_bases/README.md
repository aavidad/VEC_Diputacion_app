# Publicación oficial de las bases (F1-61)

La publicación de las bases es un acto externo a VEC: la hace el Boletín
Oficial de la Provincia, el tablón de edictos electrónico o el boletín que
indiquen las propias bases (LPAC art. 45; TREBEP arts. 55 y 61; RD 364/1995,
arts. 15 y 16; el Reglamento de bolsas, BOP n.º 9 de 16/01/2026, remite a las
bases). VEC no puede comprobarla por sí mismo y no la da por hecha. Lo que hace
es guardar una declaración: quién anota la publicación, en qué medio, con qué
número de boletín y anuncio, en qué fecha y, si existe, el enlace. La
inscripción de esa versión solo se abre cuando hay una declaración vigente.

`ConfirmarPublicacion` del puerto de gobierno de convocatorias no se toca ni se
compone. Si F1-67 no lo consume, se retira allí.

## SQL

`deploy/postgresql/bolsa_convocatorias/migraciones/000014_publicacion_oficial_bases.up.sql`
(BC14) va después de BC12. Su línea está al final de
`deploy/principal/lista_sql_main_tras_hz37.txt`. No se reaplica ni tiene DOWN.

- Tablas `publicacion_oficial_bases` y `publicacion_oficial_bases_evento`, de
  solo adición. Tienen disparadores que rechazan UPDATE, DELETE y TRUNCATE, RLS
  forzada para el propietario y ninguna concesión.
- `declarar_publicacion_oficial_bases_v1(jsonb)`: solo la ejecuta el
  propietario del esquema. Valida cada campo (medio cerrado, boletín salvo en
  el tablón, fecha real y no futura en hora de Madrid, enlace https sin
  comillas ni ángulos). Exige que
  la versión esté publicada, que la huella coincida y que sea la última. Con la
  misma clave y el mismo contenido devuelve el recibo guardado; con otro
  contenido responde 23505. Una corrección sustituye a la declaración vigente
  de esa misma versión y queda encadenada por huella.
- `publicacion_oficial_vigente_v1`: devuelve la fecha de la declaración vigente
  de una versión exacta.
- Reconstruye `comprobar_publicacion_inscripcion_v1` y
  `proyectar_abierta_inscripcion_v1`. La preimagen exigida es la poshuella de
  BC12; las poshuellas son `c74ea68b…` y `5315649e…`. Conservan OID, ACL,
  volatilidad y configuración.
- Regla de apertura: la declaración vigente tiene que ser de uno a cinco días
  naturales anterior al primer día del plazo, en hora de Madrid. El plazo
  empieza el día siguiente a la publicación (LPAC art. 30); el margen cubre un
  fin de semana y dos festivos. Sin declaración, o fuera del margen, el
  comprobador responde B9602 (la aplicación lo muestra como plazo cerrado) y
  la lista y la ficha dejan fuera la versión. Sin declaración, el comprobador
  responde antes de mirar modalidad, categoría o plazo, así que no revela nada
  de una convocatoria aún no publicada.

El plazo sigue saliendo de la versión sellada. En este corte no se recalcula a
partir de la fecha de publicación: si no encaja, RRHH publica una versión nueva
con el plazo correcto y anota la publicación en ella.

Dos límites conscientes:

- No se impide anotar el mismo anuncio en dos versiones. Una versión nueva que
  solo corrige el plazo en VEC sigue amparada por el mismo anuncio del boletín.
  Que la anotación corresponde a esas bases lo afirma quien declara.
- No se exige que la fecha sea posterior a la versión en VEC, por la misma
  razón: la versión que corrige el plazo puede ser posterior al boletín. El
  margen de cinco días ya impide abrir con fechas antiguas.

Las pruebas `bc9_*.sql` y `b96_abiertas_100x128_fixture.sql` son vectores
históricos del estado anterior a BC14. No las usa ningún ensayo ni la CI y,
si se ejecutan contra una base con BC14, sus versiones no se abren.

Prueba focal: `pruebas_sql/bc14_publicacion_oficial_bases.sql`, incluida en
`deploy/postgresql/fase1/ensayar.py --pruebas`. La de BC12 declara ahora la
publicación de sus versiones antes de comprobarlas.

## Activación en una copia

Después de BC14 ninguna convocatoria aparece abierta hasta que se declare su
publicación. `copia_local.py crear` aplica `declarar_ensayo_copia.sql` después
de la lista SQL y del kit de datos, que es quien carga las versiones
(`copia_local_publicacion_bases.py`, registro en `publicacion-bases.log`). Las
copias de presentación salen del mismo `crear`, así que también la reciben. El
kit privado no cambia.

La semilla declara, como propietario de `vec_bolsa_convocatorias`, una
publicación ficticia para cada convocatoria cuya versión vigente tenga hoy un
único plazo de inscripción abierto y aún no tenga declaración. La fecha es el
día anterior al primer día del plazo, el medio es el tablón sin boletín y el
anuncio y la clave llevan el prefijo COPIA. Así la copia conserva las
convocatorias que tenía abiertas antes de BC14. Repetirla no escribe nada y,
sin BC14 instalada, no hace nada. No se aplica en cidonia, en el servidor nuevo
ni en producción.

Comprobado en una copia propia: sin declaración la lista de abiertas estaba
vacía. Después de declarar salió la versión 2 con su recibo. Tras reiniciar
PostgreSQL y la aplicación seguía en la lista, y repetir la declaración devolvió
el mismo recibo, con una sola fila y un solo evento.

Al instalar BC14 en una base con convocatorias abiertas, todas desaparecen de la
lista hasta que se anote su publicación. Es lo previsto.

El recorrido de la aspirante por HTTP necesita la inscripción encendida, que
depende de F1-90.

## Lo que falta: el acto de RRHH desde la web

Hoy solo Informática, o los datos de la copia, pueden anotar la publicación.
Para que lo haga RRHH desde su pantalla hace falta:

1. Un consumidor de autorización, `consumir_publicacion_oficial_bases_v1`, con
   su perfil en el núcleo V3 y la acción
   `bolsa.convocatoria.publicacion_oficial.declarar` sobre el recurso
   `version_convocatoria`. Los ámbitos salen de la política. Exige reconstruir
   el núcleo central, que solo hace el equipo R y como mucho una vez por
   semana.
2. Una fachada `SECURITY DEFINER` del propietario de convocatorias que consuma
   esa autorización y llame a `declarar_publicacion_oficial_bases_v1` en la
   misma transacción. `declarada_por` y `motivo_ref` salen de la autorización
   consumida, nunca del JSON de la petición.
3. La ruta Go y la pantalla de RRHH. La pantalla pinta el enlace escapado y,
   si se quiere limitar a dominios oficiales, la lista va en catálogo. La ruta
   y la pantalla siguen siguiendo el patrón de la comprobación de
   ofrecimiento de F1-09 (puerto, caso de uso, adaptador PostgreSQL y
   composición con denegación por defecto).

## Aviso para F1-41

BC13 exige como preimagen de `comprobar_publicacion_inscripcion_v1` la
poshuella de BC12 (`510d5fb7…`). Si BC14 se fusiona antes, BC13 se niega a
instalarse. Quien fusione en segundo lugar cambia esa preimagen a `c74ea68b…`
y hace que la comprobación conjunta exija también la declaración vigente, con
la misma regla de uno a cinco días.
