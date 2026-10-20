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
- **B (ratificación del equipo de permisos).** AD301 añade al núcleo el perfil
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

El motivo es `motivos_publicacion_oficial_bases_bolsa` v1. La aplicación publica
ese catálogo al arrancar, como hace carga Convoca; si no puede, la ruta sigue
montada y responde 503 sin tocar el resto de Bolsa.

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

## Entrega B: AD301 en el núcleo y BC17 en Convocatorias

`deploy/postgresql/autorizacion_atestada_v3/migraciones/000301_perfil_publicacion_oficial_bases.up.sql`
va después de BC15. Solo UP; no se reaplica ni se revierte. Sigue el patrón de
AD291: bloqueo `vec_autorizacion_atestada_v3:nucleo`, preimagen y postimagen
por huella y definición final explícita de la función central, sin
`replace()` ni `EXECUTE`.

| Objeto | Preimagen | Postimagen |
| --- | --- | --- |
| `consumir_decision_mutacion_v3_interna` | `6e07b8c0…` (AD291) | `49d6559e7aa6e5575c630531cf5c2142a67677e5cf7d3d38da3f04cab7253035` |
| `clave_capacidad_version_audiencia_consumo_check` | `411555a7…` (AD291) | `b8a3f4abaebb452c245e2e33e191f776cea16a3e393ca11ae6ddf8c1901f10c8` |

La diferencia con la preimagen está entera en `ad301_central.diff`:

1. Rama de sesión del perfil `publicacion_oficial_bases_rrhh`: copia literal
   de la de `comprobacion_ofrecimiento_bolsa`. La sesión tiene que ser el
   LOGIN `vec_bolsa_llamamientos_desarrollo`, sin `SET ROLE`, miembro solo del
   ejecutor de Bolsa y sin ajustes por rol.
2. El perfil sale del caso genérico (una condición más en la lista de
   exclusiones).
3. Predicado del perfil: operación y acción
   `bolsa.convocatoria.publicacion_oficial.declarar`, audiencia
   `vec.bolsa.publicacion_oficial_bases.v1`, módulo `bolsa`, recurso
   `version_convocatoria` con referencia `cv1_<sha256>_v<n>` igual al efecto,
   huella del contexto igual a la del efecto, finalidad
   `declarar_publicacion_oficial_bases`, garantía alta, certificado,
   superficie interna, sin campos y con la obligación `auditar`. La huella
   firmada del recurso liga los ámbitos `ambito_ref` y `unidad_ref` y el
   material de la petición; BC17 la compara con la calculada por el servidor.

Además crea `consumir_publicacion_oficial_bases_v1` (patrón AD290), que solo
puede ejecutar `vec_bolsa_convocatorias_propietario`. Ese propietario recibe
`USAGE` del esquema del núcleo; la preimagen exige que ninguna función del
esquema esté abierta a `PUBLIC` y la postimagen, que con ese `USAGE` solo
alcance el consumidor y ninguna tabla. La audiencia entra en la restricción
de claves de capacidad y el origen se copia de la fila
`situacion_participacion` del mismo pool, con la misma comprobación de
membresía que AD291.

Prueba focal: `pruebas_sql/ad301_publicacion_oficial_bases.sql` (revierte
todo). Comprueba huellas, las tres ramas, el ACL del consumidor, el alcance
del propietario de convocatorias, la restricción y el origen, y que un
material ajeno se deniega con `42501` antes de llegar al núcleo.

### Cotejo del recurso y recuperación (BC17)

La decisión V3 no contiene un campo `ambitos`. Conserva la huella firmada del
recurso completo, con sus ámbitos y atributos. BC17 obtiene la política de
la versión vigente, calcula esa huella con los dos ámbitos y la huella de la
petición, y la compara tanto con la decisión como con el consumo. Una versión
de otro ámbito sigue denegada por el permiso vigente y por ese cotejo.

BC17 reconstruye solo la fachada de BC15, con su misma firma y permisos.
Exige la preimagen `59e961bc…` y deja `6f0fd898…`. El enlace de autorización
conserva el perfil del acto original: recuperar la clave desde otro perfil
provoca conflicto (`23505`, HTTP 409). Una decisión nueva del mismo perfil
puede recuperar el recibo original, sin otra declaración.

Orden de instalación de B: AD301 y después
`bolsa_convocatorias/migraciones/000017_recuperacion_publicacion_mismo_perfil.up.sql`,
al final de `lista_sql_main_tras_hz37.txt`, sin reaplicar BC15.

### Activación y comprobación

1. Crear una copia propia con la orden completa de `LEEME_copia_local.md`,
   módulos, identidad original de RRHH, lista SQL de esta candidata y los
   dos administradores de la copia. Sin la nueva SQL la aplicación arranca;
   solo el acto de publicación queda cerrado.
2. Admitir en el catálogo completo la fuente
   `fuentes/bolsa_publicacion_oficial_bases_rrhh_v1.json` por AUT58/AUT61.
   Publicar la versión del rol RRHH con propuesta de un administrador y
   cierre del otro, como en `habilitacion_rol_bolsa_rrhh_generico_20261013.md`.
   Conservar las concesiones, ámbitos y asignaciones anteriores.
3. Reiniciar la aplicación tras publicar el rol, para adoptar su instantánea.
   En Bolsas → Publicación de las bases, declarar o corregir el anuncio y
   conservar el recibo. Repetir la misma petición y clave: HTTP 200, mismo
   recibo y fecha, una sola declaración y un solo enlace de autorización.
4. Una convocatoria con ámbito ajeno no debe aparecer en la lista; enviar su
   declaración debe responder 403 sin otra anotación. Reiniciar PostgreSQL y
   aplicación y recuperar la misma petición. No preparar otra clave para
   conciliar una respuesta perdida.

No hay configuración de producto nueva. Se reutiliza el montaje B103 de
Administración; su fichero de entorno se cambia con el proceso parado y se
conserva la versión anterior para volver atrás.
