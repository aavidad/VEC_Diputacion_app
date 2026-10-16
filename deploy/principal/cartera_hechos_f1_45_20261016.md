# Cartera personal de méritos: hechos visibles (F1-45)

## Alcance y estado

La API `GET /api/vec/meritos/cartera` y la pantalla **Mis méritos** ya están en
`main` (#1081 y #1105). Este corte comprueba que la persona ve los hechos de su
propia cartera: tipo, denominación y disponibilidad, junto con el total, las
páginas siguientes y el recibo de lectura que entrega la API. El contrato está
en [`internal/modules/meritos/ports/cartera.go`](../../internal/modules/meritos/ports/cartera.go)
y el cliente y la vista en [`web/static/area-personal/`](../../web/static/area-personal/).

La valoración por categoría de Bolsa y los puntos esperan al motor común de
baremación. El total de esta consulta cuenta hechos; no es una puntuación. La
pantalla no acredita títulos ni decide si un mérito cuenta para una bolsa.

La API y la pantalla están integradas. El estado de activación, los resultados
reales y la versión ensayada se registran en la PR de F1-45. Este guion describe
la preparación y las comprobaciones; por sí solo no acredita una instalación.

El corte completa el exportador de alias existente para la cuenta propia de
Méritos y clasifica su fichero como material externo. Usa el mismo
seudonimizador de Usuarios y conserva el sujeto certificado. Cuenta y perfil
de Méritos son distintos de los de Usuarios y Bolsa; el fichero de configuración
no concede permisos.

## Preparación en la copia local

Reutilizar el [guion de habilitación existente](habilitacion_meritos_cartera_20261013.md)
y sus listas SQL de [cartera](lista_sql_codexm_meritos_basal_20261013.txt) y
[revisión de RRHH](lista_sql_codexm_meritos_revision_rrhh_20261013.txt). Antes
de instalar, cotejar qué migraciones ya figuran aplicadas y seguir el orden de
ese guion; ninguna se repite. La [fuente de permisos existente](fuentes/meritos_cartera_propia_v1.json)
define la lectura propia `meritos.cartera.consultar_propia` en ámbito
`persona_ref`, con garantía alta y auditoría. La habilitación gobernada publica
la fuente completa del guion, con las tres concesiones de consulta, retirada y
declaración. El ensayo de este corte se limita al GET. Antes de activar,
comprobar que los actos de Administración vigentes aceptan esta fuente, la
versión del rol y la asignación personal, y conservar sus acuses. El rol fijo
se publica por propuesta y cierre de Administración (AUT60/80), con dos
administradores distintos. La función `publicar_rol_usuarios_externo_v1` no
publica este rol. Identificarse no concede el rol.

Preparar fuera de Git los siete LOGIN nominales y sus conexiones separadas, el
material de identidad externa, los motivos y la confianza vigente que pide el
guion anterior. La composición abre también la conexión nominal de operación
al montar la cartera, aunque este ensayo solo lea. Usar una persona y hechos
**ficticios** vinculados por su referencia real de persona. Los ajustes de
datos ficticios de la copia propia pueden hacerse con SQL directo privado,
sin modificar el código ni el SQL del producto, y se declaran por separado. No
acreditan una publicación gobernada, una aprobación, un acto o un recibo. No
añadir esos ajustes ni credenciales a Git.

La fase de identidad reutiliza
[`vec-provisionar-candidato-externo`](../../cmd/vec-provisionar-candidato-externo/README.md):
admite un snapshot de población `usuarios`, sin vínculo candidato. Primero se
exportan los alias con `vec-server exportar-seudonimos-portal-externo` y el
material externo separado, guardando la salida en un fichero privado `0600`.
Se prepara ese fichero con `umask 077` antes de ejecutar el exportador.
Se selecciona la cuenta nueva exacta; no se copia el HMAC de la cuenta anterior.
La fuente de esta fase no contiene un bloque de identidad previo si se pasa
`--alias-archivo`. Con el entorno administrativo del README:

```sh
vec-provisionar-candidato-externo \
  --fuente "$FUENTE_M4_PRIVADA" --fase identidad \
  --dsn-archivo "$DSN_IDENTIDAD_PRIVADO" --alias-archivo "$ALIAS_M4_PRIVADO"
```

Del plan se conservan `huella_sha256` y `preimagen_sha256`. Se repite la misma
llamada añadiendo `--aprobar "$HUELLA_PLAN" --preimagen "$HUELLA_PREIMAGEN"`.
El CLI crea y consume la aprobación en su transacción y entrega el recibo.
Ante un fallo se coteja la preimagen antes de intentar otra operación.

El perfil se publica por la fachada existente
`vec_contexto_actor_v1.publicar_perfil_usuarios_externo_v1`
([CA15](../postgresql/contexto_actor_v1/migraciones/000015_contexto_externo_segregado.up.sql)), con su canal
administrativo autorizado, versión esperada y huella del material aprobado.
La asignación usa
`vec_autorizacion.publicar_asignacion_usuarios_externo_v1`
([AUT81](../postgresql/autorizacion/migraciones/000081_asignacion_meritos_cartera_externa.up.sql)): exige ese
perfil vigente y el rol gobernado, y devuelve versión y huella. Su LOGIN tiene
como única membresía directa `vec_autorizacion_publicador_usuarios_externo`,
con INHERIT y sin SET ni ADMIN; carece de privilegios elevados y de ajustes
propios de sesión. Las conexiones administrativas de publicación quedan fuera
de la web y no sustituyen ninguno de los siete pools de la cartera. No se
insertan filas de aprobación a mano para sustituir estas publicaciones.

El inventario inicial de la copia O del 16/10 encontró las migraciones de la cartera,
pero ningún hecho para la persona elegida ni consultas propias; tampoco encontró
publicadas las acciones, la versión y asignación del rol, los motivos ni los
siete LOGIN nominales. Los ficheros de conexión privados están preparados, pero
falta el material de identidad que permite cargarlos. Este inventario no
acredita que la función esté activada.

## Ensayo de aceptación

1. Congelar el mismo árbol y configuración que se usará en el recorrido. Arrancar
   primero sin el material de cartera y comprobar que las demás funciones
   atienden y que la consulta propia queda cerrada; registrar la respuesta real.
2. Aplicar solo las migraciones que falten y preparar el material privado.
   Cotejar el productor de cada acto de Administración requerido; publicar
   solo mediante el acto que corresponda y guardar su acuse. Si falta un
   productor, detener la activación de esa función. Reiniciar de forma
   vigilada y comprobar salud de la aplicación y disponibilidad de la consulta.
3. Entrar en **Mis méritos** con la persona ficticia autorizada. Cotejar la
   respuesta del GET con la pantalla: mismos hechos, tipos, disponibilidad y
   total; conservar el recibo de lectura de la API. Si hay más de una página,
   pulsar la acción de continuación y comprobar que no se repiten ni se pierden
   hechos. Una cartera vacía se muestra como vacía, sin inventar resultados.
4. Comprobar con otra persona ficticia que la lectura propia no revela la
   cartera ajena. Registrar la denegación real sin ampliar permisos para
   superar la prueba.
5. Reiniciar aplicación y base en la copia y repetir la lectura autorizada.
   Comparar los hechos y su disponibilidad con la lectura anterior; conservar
   los recibos y fechas de cada consulta sin atribuirles valoración o puntos.

La PR conserva las respuestas reales, los acuses y las pruebas de arranque,
consulta y recuperación que se hayan completado. Solo se afirma la paginación
si se ha recorrido con datos suficientes. Declarar, retirar, acreditar y
baremar son capacidades distintas de la lectura comprobada en este corte.
La instalación fuera de la copia local queda fuera de este ensayo.
