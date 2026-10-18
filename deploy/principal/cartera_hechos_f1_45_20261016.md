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

La consulta exige una cuenta y un perfil de Méritos propios, distintos de los
de Usuarios y Bolsa, con el mismo certificado y la misma persona. Hoy no hay
productor gobernado que los dé de alta para el aspirante. El exportador de
alias tampoco enumera esa cuenta; este corte no lo cambia.

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

### Cuenta, perfil y asignación de Méritos: semilla de copia

Decisión de dirección del 16/10, 06:09: en la copia local, la cuenta y el
perfil de Méritos y su asignación al rol se cargan como **semilla de copia
declarada**, igual que las filas del rol externo del aspirante (decisión de
las 03:48). Reglas:

- Se cargan por SQL privado en la copia, fuera de Git, con la persona y el
  certificado ficticios de Usuarios. No se reutiliza la cuenta de Usuarios ni
  se copia su HMAC.
- No es una aprobación real ni lleva recibo. No se inventan actos ni filas de
  aprobación para que parezca una publicación de Administración.
- No se construye ahora el productor de esa cuenta ni se amplía el exportador.
- Responsable: Codex-O. Se retira al integrar F0-21 y la tarea de permisos por
  datos de la fase 2, que deben dar de alta esta cuenta por la vía gobernada.

Las vías de producto que existen para cuando haya productor son la fachada
`vec_contexto_actor_v1.publicar_perfil_usuarios_externo_v1`
([CA15](../postgresql/contexto_actor_v1/migraciones/000015_contexto_externo_segregado.up.sql))
y `vec_autorizacion.publicar_asignacion_usuarios_externo_v1`
([AUT81](../postgresql/autorizacion/migraciones/000081_asignacion_meritos_cartera_externa.up.sql)).
AUT81 no tiene todavía llamador en Go.

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
   solo mediante el acto que corresponda y guardar su acuse. La cuenta, el
   perfil y la asignación de Méritos van como semilla de copia (apartado
   anterior); si falta otro productor, detener la activación. Reiniciar de forma
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
