# Dos plazas con aceptación previa

Tarea F1-55. Dirección fijó el alcance el 18/10: oferta de dos plazas,
aceptación previa de dos personas, adjudicación por el orden de la bolsa y
recuperación tras perder una respuesta y reiniciar. La retirada por cancelación
queda en F1-17. No responder a la oferta no produce un acto ni causa una baja.

## Preparar dos aspirantes

Crear la copia con `deploy/principal/copia_local.py`, `--modulos`, la lista
común completa y el kit interno de preparación. Usar conexiones propias y
conservar la configuración anterior antes de cambiar el material. No hay
migraciones propias de F1-55; la candidata del ensayo contiene las 81 de main.

Elegir dos candidaturas de la misma bolsa constituida, con participaciones
distintas, disponibles y sin vínculo vigente con otra persona. Cada una tiene
su directorio privado, certificado, cuenta, persona, perfil y vínculo. Preparar
las carpetas con la CA de la copia y modos privados, sin guardar material en Git.

Configurar las variables existentes `VEC_PRINCIPAL_POSTGRES_CONTAINER`,
`VEC_CANDIDATE_BOLSA_REF` y `VEC_CANDIDATE_BASE_URL` con el contenedor, bolsa y
URL HTTPS propios. El preparador admite además `VEC_CANDIDATE_DATABASE` y
`VEC_CANDIDATE_DATABASE_USER` cuando no se use su valor predeterminado.
Con las rutas privadas y las referencias elegidas en variables:

```sh
VEC_DEVELOPMENT_MATERIAL_DIR="$material_antonio" \
  deploy/principal/preparar_candidato_desarrollo.sh --motor docker \
  --identidad antonio-reyes --nombre 'Antonio Reyes Álvarez' \
  --candidato-ref "$candidato_antonio"

VEC_DEVELOPMENT_MATERIAL_DIR="$material_elena" \
  deploy/principal/preparar_candidato_desarrollo.sh --motor docker \
  --identidad elena-martin --nombre 'Elena Martín Ruiz' \
  --candidato-ref "$candidato_elena"
```

El motor predeterminado es Podman.
Los certificados nuevos duran 90 días; los existentes deben durar entre 60 y
90. Un cambio de candidatura que no coincida con la intención o el manifiesto
se rechaza antes de confirmar datos. La selección conserva el filtro de bolsa,
la pertenencia al acta y el ensayo con ROLLBACK antes del COMMIT.

La app utiliza el directorio activo mediante `VEC_DEVELOPMENT_MATERIAL_DIR`.
Para cambiarlo: parar la app, conservar el entorno anterior, seleccionar el
material propio, reiniciar y comprobar salud y consulta de Mi Bolsa. En este
ensayo se atendió a las dos personas por turnos. Desde cada certificado se
comprobó su posición propia antes de aceptar.

La autoridad de RRHH procede del arranque de desarrollo B-BACK que documenta
la receta interna de la copia. No se crearon asignaciones manuales ni se
cambiaron huellas. Dirección confirmó esta preparación el 18/10 a las 10:15.
No acredita la asignación de perfiles desde Administración.

## Recorrido

1. RRHH abre «Gestionar ofertas» desde Llamamientos y publica una oferta de dos
   plazas con política vigente de aceptación previa. Conserva recibo y plazo.
2. Acepta desde Mi Bolsa la persona situada en segundo lugar; después acepta
   la primera. Cada aceptación entrega su recibo.
3. Tras vencer el plazo, RRHH comprueba las propuestas por plaza: deben seguir
   el orden de la bolsa, aunque las respuestas hayan llegado al revés.
4. RRHH revisa y confirma la primera adjudicación. Se corta su respuesta al
   navegador después de que la API la guarde. Se conserva la misma pantalla,
   cuerpo y clave de la intención; tras reiniciar, se confirma de nuevo y se
   recupera el mismo acto. Después se confirma la segunda plaza.
5. Se reinician app y PostgreSQL, se recarga la oferta y se cotejan sus recibos,
   fechas, aceptaciones e historial. La tercera persona, que no contestó,
   conserva su situación sin una respuesta ni una baja por este motivo.

## Evidencia del 18/10

Fuente congelada de producto `1a29dfefa`, con F1-18 de main integrada.
Copia creada en 83,32 s, actualizada con respaldo frío y las 81 migraciones
cotejadas; B129, B131 y B132 se instalaron una sola vez al incorporarlas.
La compilación se hizo sin red. Binario SHA-256:
`4330466f70a07c697237dceedbd21054b5aec15b9723a9cc3f53c23a156d0dee`.

La oferta y las aceptaciones se registraron primero con `8d8a29e17`; la
copia continuó con `1a29dfefa` conservando esos hechos. Las dos identidades
y sus certificados de 90 días se prepararon con el guion real; repetirlo recuperó las mismas proyecciones. RRHH publicó la política y
la oferta desde Chrome; la política del ensayo fijó una hora de plazo. Aceptó primero la posición 2 y después la 1, ambas
con 201. Tras reiniciar, cada aspirante recuperó con 200 su mismo recibo y
fecha. Antes del vencimiento no se ofrecía adjudicar.

Con el selector de Mi Bolsa apagado, salud, portal, cuadro de contratación y
bolsas respondieron 200; la consulta propia dio 403. Al restaurarlo, la app
arrancó y conservó los datos anteriores.

Al vencer a las 10:47 de Madrid, la pantalla propuso la posición 1 para la
primera plaza y la 2 para la segunda. La primera adjudicación dio 201 y se
omitió su respuesta al navegador. Después de reiniciar app y PostgreSQL,
Confirmar recuperó con 200 el mismo cuerpo, clave, recibo y fecha. Solo había
un acto y una salida. Cancelar la revisión de la segunda no emitió POST;
confirmarla dio 201 y dejó las dos plazas cubiertas, con un acto por plaza.
La tercera posición siguió disponible y sin una respuesta a la oferta.

El corte añade a cada plaza un detalle «Recibo de adjudicación» con su fecha
y el botón de copiar. Reutiliza el historial de la consulta existente y el
componente de justificantes; no pide más datos ni registra actos al abrirlo.
La comprobación en Chrome de ese detalle y el reinicio final están pendientes.

Chrome del sistema con TLS y certificados comprobados, en castellano e
inglés. Anchos de 1440, 1024 y 390 px sin desbordamiento horizontal y sin
scroll de documento en escritorio. Cero errores JavaScript, cookies y
almacenamiento web. Se comprobó el foco inicial visible y un reflujo
equivalente al 200 %; no es una auditoría completa de accesibilidad.

Bash, Shellcheck y tres pruebas focales conformes. Las pruebas cubren las dos
identidades, recuperación, referencia distinta, entradas inválidas y los
límites de 59, 60, 90, 91 y 397 días. Dos revisiones independientes favorables
al código de `1a29dfefa`. Semgrep local: cinco reglas, dos archivos y cero
hallazgos; conserva el aviso de análisis parcial de Bash, revisado también a
mano y con Shellcheck. GoSec no aplica porque no cambia Go.

Los guiones, respuestas y capturas quedan fuera de Git. La consulta de personas
importadas sigue pendiente de la clave de su staging; no se necesitó para
aceptar ni para abrir ofertas desde Llamamientos. La consulta de preferencias
no forma parte de este ensayo. En inglés permanece el literal de estado de
la bolsa `vigente`, anterior a este corte. No se acredita portal externo,
correo corporativo, cancelación ni continuación hasta formalización en CT.
