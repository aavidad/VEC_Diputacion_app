# F1-150b: peticiones simultáneas del perfil de desarrollo

## Causa y decisión

Las rutas dinámicas de Contratación temporal preparan el rol y los ámbitos de
cada petición sobre una misma asignación de perfil. Dos peticiones pueden
preparar la siguiente versión con contenidos distintos. La publicación rechaza
correctamente una de ellas. Publicar otra versión como reintento tampoco sirve:
la decisión de la primera petición dejaría de referirse a la asignación vigente
que exige `vec_autorizacion.revalidar_decision_contexto_actor_v3_viva` al consumir.

Se reutiliza la frontera HTTP de desarrollo, después de verificar la identidad
y la ventana del certificado. Un turno por perfil sintético canónico abarca el
manejador completo: preparar, publicar, decidir y consumir. Lo comparte la
autoridad de todas las rutas; incluye selección, comunicaciones y respuestas,
formalización, seguimiento, cese, cancelación y el segundo técnico de RRHH.
También ordena las rutas de esa identidad que usan perfiles dedicados. Esta
elección conservadora evita inventariar por separado cada publicación interna.

Los perfiles de personas diferentes trabajan por separado. La espera respeta la
cancelación y el plazo de la petición y nunca supera 30 segundos. Ese límite se
aplica a esperar turno; no libera a un titular que aún esté ejecutando su
operación. El turno se retira al retornar el manejador, también si hay un error
o un pánico. Las entradas sin titular ni esperas se eliminan del mapa.

El contexto conserva el turno al atravesar una frontera interior del mismo
coordinador y perfil. La coordinación solo vale en la composición de desarrollo
con un proceso. No acredita coordinación entre procesos ni sustituye los
bloqueos, la vigencia, la auditoría o las guardas de PostgreSQL.

Un perfil distinto por ruta necesitaría nuevas publicaciones y cambios en los
contratos de sesión y ámbito. El turno en memoria resuelve la carrera actual sin
añadir SQL ni permisos. El estudio y la elección se hicieron dentro de los
primeros 30 minutos del encargo.

## Activación

CONFIG NUEVA: ninguna. SQL a instalar: ninguno.

Tras integrar, compilar el servidor y reiniciar solo la aplicación con la receta
de su copia. Se conservan PostgreSQL, certificados, recibos y configuración.
Con desarrollo desactivado esta frontera no se compone; no se añade ningún
requisito de arranque. Para volver atrás, usar el binario anterior mediante la
misma receta. No cambiar las copias de otro equipo.

La pantalla consumidora es la ficha de Contratación temporal, que consulta y
tramita el expediente mediante varias rutas. El recorrido conectado queda a
cargo de dirección y del equipo que conserva su copia.

## Validación

Cuatro pruebas unitarias verdes: cancelación, plazo de la petición, máximo de
30 segundos sin plazo del cliente, exclusión y retirada de entradas. La prueba
de frontera confirma que la petición cancelada no ejecuta el manejador.

Ensayo en PostgreSQL 18.4 desechable, sin red, con esquema HZ37 sin datos,
preparación versionada y las 147 migraciones de la lista de main. Se añadió
únicamente el dato del catálogo vacío de políticas que requieren las semillas
de los helpers existentes (revisión 1, huella canónica de `[]`). El primer intento
sin ese dato se detuvo antes de poder reproducir la carrera y no cuenta como
evidencia del fallo.

La misma prueba de frontera con el cerrojo retirado falla para el técnico
original y el segundo con «asignación sustituida antes de revalidar». Con el
cerrojo, las dos variantes pasan y cada perfil conserva dos publicaciones
sucesivas. Pasan también las dos pruebas existentes de publicación simultánea:
la repetición exacta se acepta y una publicación incompatible se rechaza.

La prueba verifica la asignación exacta en PostgreSQL dentro del manejador;
no emite una decisión ni ejecuta un acto de negocio completo. Las pruebas de
esta tarea no acreditan un recorrido en Chrome.

`go test -race -p 2 ./internal/app/bootstrap -run '^TestSerializacionPerfilCT'
-count=1` pasa (31,252 s; las dos variantes PostgreSQL se ejecutaron por separado
con la base desechable). `go vet -p 2 ./internal/app/bootstrap`, comprobación de
tamaños y `git diff --check` pasan. Semgrep local: cuatro reglas sobre los tres
Go de producción modificados, cero avisos. GoSec completa el paquete: 81 avisos
en archivos ajenos al cambio y ninguno en los modificados.

La concurrencia PostgreSQL usa el filtro
`^(TestSerializacionPerfilCTConservaAsignacionHastaRevalidarPostgreSQL18|TestAutoridadPublicacionConcurrente.*PostgreSQL18)$`,
con `VEC_PERFIL_DINAMICO_RRHH_PG_DESECHABLE=1` y los dos DSN de prueba existentes
dirigidos al contenedor desechable. Los binarios anterior y actual se compilaron
con la misma prueba; el anterior solo retiraba del revalidador la toma del turno.
El contenedor se eliminó al terminar. Sin nueva copia ni arranque de aplicación.

La PR permanece en borrador hasta las revisiones independientes de autorización;
dirección organiza la segunda.

La primera revisión detectó que los dos retornos tempranos nuevos usaban texto
plano. Se corrigieron reutilizando el sobre de error de catálogo existente
(`servicio_no_disponible`, JSON y cabeceras sin caché); la prueba focal comprueba
su contrato. Se conserva como deuda anterior la secuencia A–B–A: la caché de
instantáneas por solicitud puede recuperar una versión antigua después de otra
operación. La guarda sigue rechazándola; este corte resuelve el solapamiento.
