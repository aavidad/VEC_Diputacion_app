# Publicación oficial de las bases desde la web (F1-95)

Continúa F1-61. Allí la publicación oficial de las bases solo la podía anotar
el propietario de la base de datos (Informática o la semilla de las copias).
Con este corte la anota RRHH desde el portal, con su permiso y su recibo. Lo
que se guarda es lo mismo: medio, número de boletín, referencia del anuncio,
fecha, enlace opcional y, si es una corrección, la anotación a la que
sustituye.

Son dos entregas:

- **A (este cambio).** Fachada SQL BC15, ruta Go, pantalla y fuente de la
  acción. Se puede instalar ya. Mientras el núcleo de autorización no tenga el
  consumidor de esta acción, la ruta deniega y no escribe nada.
- **B (borrador para el equipo de permisos).** AD301 añade al núcleo el perfil
  `publicacion_oficial_bases_rrhh`, el consumidor
  `consumir_publicacion_oficial_bases_v1`, la audiencia en la lista cerrada de
  claves de capacidad y el origen del pool de Bolsa. Solo con A y B instaladas,
  y el rol de RRHH publicado con la concesión, la anotación se acepta.

## Acción

| Dato | Valor |
| --- | --- |
| Acción | `bolsa.convocatoria.publicacion_oficial.declarar` |
| Recurso | `version_convocatoria`, referencia `cv1_<sha256 del id>_v<secuencia>` |
| Finalidad | `declarar_publicacion_oficial_bases` |
| Garantía | alta, con certificado, superficie interna |
| Ámbitos | `unidad_ref` y `ambito_ref` de la política de gestión publicada para esa versión |
| Campos | ninguno; obligación `auditar` |
| Audiencia | `vec.bolsa.publicacion_oficial_bases.v1` |

El navegador no envía ámbitos, declarante ni motivo. El ámbito sale de
`comprobar_ambito_gestion_inscripcion_v1` sobre la política que la versión
tiene sellada. Quién declara (`principal_id`) y el motivo salen de la decisión
consumida. El atributo `material_sha256` del recurso es el SHA-256 de la
petición normalizada por PostgreSQL (`p::text`), como en F1-09.

## SQL de la entrega A

`deploy/postgresql/bolsa_convocatorias/migraciones/000015_declaracion_publicacion_oficial_rrhh.up.sql`
(BC15) va después de BC14, al final de
`deploy/principal/lista_sql_main_tras_hz37.txt`. Solo UP; no se reaplica.

- `consultar_publicaciones_oficiales_bases_rrhh_v1(text, integer)`: lista las
  versiones publicadas y vigentes que tienen ámbito de gestión, con su
  anotación vigente si existe; o una sola versión si se pasa su referencia.
  No devuelve el identificador interno de la convocatoria.
- `declarar_publicacion_oficial_bases_rrhh_v1(jsonb, …material V3…)`: exige la
  sesión del pool de Bolsa (miembro del ejecutor y de nada más) y una
  transacción serializable. Comprueba que la versión y su huella sigan
  vigentes, consume la decisión con el consumidor del núcleo, comprueba que
  la decisión es de esta acción, este recurso y estos ámbitos, y llama a
  `declarar_publicacion_oficial_bases_v1` de BC14 en la misma transacción.
  Guarda el enlace entre la anotación y la decisión en
  `publicacion_oficial_bases_autorizacion` (solo adición, sin concesiones).
- Sin el consumidor, o sin permiso para ejecutarlo, responde `55000` antes de
  escribir. La aplicación lo muestra como «no disponible» (503).
- El ejecutor de Bolsa recibe `USAGE` del esquema de convocatorias. La
  postimagen comprueba que con él solo alcanza estas dos funciones y ninguna
  tabla.

Prueba focal: `pruebas_sql/bc15_declaracion_publicacion_oficial_rrhh.sql`,
incluida en `deploy/postgresql/fase1/ensayar.py --pruebas`. Sin AD301 espera
`55000`; con AD301, `42501` para un material que no es una decisión.

## Aplicación

Ruta exacta `/api/vec/bolsa/convocatorias/publicaciones-oficiales`:

- `GET`: versiones que el perfil activo puede declarar. Se evalúa la concesión
  sobre el ámbito de cada versión sin emitir decisión, y la consulta queda
  auditada. Sin la concesión responde 403.
- `POST` con `Idempotency-Key`: 201 al anotar, 200 al repetir el mismo envío.
  400 `solicitud_invalida`, 403 `acceso_denegado`, 409 `version_no_vigente` o
  `publicacion_en_conflicto`, 503 `publicacion_oficial_no_habilitada` o
  `resultado_no_confirmado` (reintentar con la misma clave).

CONFIG NUEVA: ninguna. El proveedor de material se crea con
`VEC_BOLSA_BORRADORES_ENABLED`, como el de F1-09. Si la clave de capacidad no
se puede publicar porque el núcleo aún no admite la audiencia, la ruta queda
montada y responde 503; el resto de Bolsa sigue igual.

## Permiso de RRHH

1. Admitir la fuente `deploy/principal/fuentes/bolsa_publicacion_oficial_bases_rrhh_v1.json`
   (huella `86e4db8765b261085c4970580afa5c9fb03ca93e8ec0c0730bf680bf62364062`,
   canon serializado con los tipos Go de dominio, vigencia de 90 días).
2. Publicar una versión nueva del rol de RRHH de Bolsa con esa concesión, por
   propuesta y cierre de dos administradores distintos, conservando las
   concesiones y ámbitos que ya tiene. La composición admite la concesión solo
   con su forma exacta y una vez.
3. La asignación del perfil debe cubrir el `unidad_ref` y `ambito_ref` de la
   política de cada convocatoria. Una versión de otro ámbito no aparece en la
   lista y su anotación se deniega (403).

## Límites

- La semilla de las copias (`declarar_ensayo_copia.sql`, prefijo `COPIA`)
  sigue anotando como propietario. No pasa por esta vía ni tiene enlace con
  una decisión; si su clave se reutiliza desde la web, se responde 409.
- No se comprueba contra el boletín que el anuncio exista: lo afirma quien
  declara, y queda su nombre en la anotación.
