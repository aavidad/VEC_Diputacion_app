# Simulador local de baremo

Abre los editores de Bolsa y Concursos. Bolsa ofrece experiencia y otros méritos;
Concursos valora grado, trabajo por nivel, antigüedad, permanencia, cursos y títulos
sobre un puesto y servicios sintéticos.
El cálculo usa los mismos servicios Go que `vec-baremador`. No consulta personas
ni registra baremaciones, activa reglas o modifica una convocatoria.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-baremador-web --web-dir web/static
```

El programa muestra la dirección que debe abrirse en Chrome. Escucha solo en
`127.0.0.1`; por defecto el sistema asigna un puerto libre. Para elegirlo, añadir
`--puerto 49153`. La ruta de los recursos la fija el operador con `--web-dir`.
Solo se cargan los recursos del editor enumerados en `http.go`; el servidor no
publica el directorio completo. Los catálogos del editor se cargan para los
idiomas declarados en `textos/idiomas.json`.

`GET /ejemplos` devuelve las reglas de cada ejemplo. `POST /simular` recibe
`modo`, `ejemplo_ref` y el objeto `reglas`. El servidor fija los datos sintéticos
de entrada, valida las reglas y calcula su representación canónica en Go.
Devuelve el resultado con su huella. Un bloqueo conserva sus motivos y no tiene
total; una regla inválida devuelve 422. El POST exige el origen y el Host exactos
de la dirección mostrada al arrancar, con `Content-Type: application/json`.

En Concursos, `GET /api/provision/v1/configuracion-local` devuelve configuración e
instantánea sintética de cada caso. `POST /api/provision/v1/simulaciones` recibe
`ejemplo_ref` y `configuracion`; liga la entrada al ejemplo embebido, sin admitir
una entrada enviada por el navegador. Usa `application.Simular` de Provisión y
responde con `provision.simulacion.v1`: desglose, total y huellas. La fecha de
corte de Concursos es exclusiva; la de Bolsa conserva su contrato inclusivo.

RRHH puede cambiar coeficientes, tablas de nivel, topes y fechas, comparar los
resultados y guardar un borrador JSON. Al cargar un borrador de Concursos, Go
valida antes de sustituirlo. Un archivo rechazado conserva los cambios previos.
La activación institucional permanece deshabilitada; no hay firma ni SQL.

No admite rutas de ficheros, URLs, datos personales ni credenciales. No emite cookies,
CORS ni caché. Cada solicitud tiene un máximo de 256 KiB, 128 elementos por
colección, 32 niveles y 8192 valores para Bolsa. El adaptador de Concursos limita
a 24 niveles y 10000 elementos por colección; su límite JSON de 2 MiB queda
subordinado al límite HTTP común de 256 KiB. La importación web de Concursos
también limita a 256 KiB. Se permiten dos simulaciones simultáneas en total.
El servidor limita la lectura de cabeceras a dos segundos, la lectura completa
a cinco y la escritura a diez. Se detiene con Ctrl+C.

## Ensayo local de procesos selectivos

La tercera dirección que muestra el programa abre el ensayo de oposición,
concurso y concurso-oposición. RRHH puede revisar los requisitos, las fases y
las reglas de un ejemplo sintético, cambiar la configuración y ver cómo se
obtiene cada resultado. Las notas editables pertenecen a las solicitudes sintéticas del ejemplo.
El ejercicio no admite nuevas solicitudes, nombres, requisitos, méritos ni documentos
procedentes del navegador. Tampoco registra candidaturas ni calificaciones.

`GET /api/seleccion/v1/ensayos` devuelve tres ejemplos sintéticos, uno por
modalidad. `POST /api/seleccion/v1/simulaciones` recibe `ejemplo_ref` y
`configuracion` y, opcionalmente, `notas_prueba`: una lista de
`solicitud_ref`, `fase_ref` y `puntos_micropuntos`. Las referencias deben pertenecer
al ejemplo y la fase debe ser una prueba. La nota es un entero de micropuntos
entre cero y el máximo configurado, o `null` para dejarla pendiente. Omitir la
lista conserva las notas originales. Cada nota editada identifica su procedencia
en el desglose; no cambia los requisitos ni los méritos.

El servidor aporta los demás hechos sintéticos del ejemplo y usa el
caso de uso de Selección con el baremador común. La respuesta es un cálculo
reproducible, sin acto de admisión, aprobación del tribunal ni traspaso a Bolsa
o Personal. Los textos de la pantalla proceden de catálogos en español e inglés.

Comprobaciones focales:

```sh
go test -race ./cmd/vec-baremador-web ./internal/modules/bolsa/adapters/simuladorlocal ./internal/modules/provision/... ./internal/modules/seleccion/...
go vet ./cmd/vec-baremador-web ./internal/modules/bolsa/adapters/simuladorlocal ./internal/modules/provision/... ./internal/modules/seleccion/...
```

Las pruebas comparan la salida HTTP con el servicio común y comprueban origen,
Host, método, campos desconocidos, exceso de tamaño y escapes del directorio de
recursos. Esta herramienta local no se monta en el portal ni acredita permisos
institucionales, aprobación de bases o una valoración oficial.

## Preparación de un concurso interno

La segunda dirección que muestra el programa abre
`/portal-empleado/modulos/provision/`. Permite revisar dos puestos sintéticos,
seleccionarlos, cambiar su orden de preferencia y simular la puntuación de cada
uno con el motor común. La convocatoria conserva referencias y versiones de RPT,
bases, reglas e instantánea. Una referencia declarada no acredita una vacante ni
la condición de empleado.

En «Convocatoria» se pueden ajustar fechas, coeficientes y topes del ejercicio.
Los campos inválidos conservan lo escrito y señalan qué corregir. «Valoración»
separa los requisitos de acceso de los puntos; una fuente ausente permanece
pendiente y no tiene total. La preparación sólo dura mientras está abierta la
página. Presentación, propuesta oficial, reclamación y resolución permanecen
deshabilitadas hasta conectar sus autoridades y persistencia.

`GET /api/provision/v1/procesos-locales` proyecta oferta, configuración y
preferencias del ejemplo. `POST /api/provision/v1/procesos-locales/simulaciones`
admite únicamente `ejemplo_ref`, `configuracion` y `preferencias`. El servidor
fija los hechos sintéticos y devuelve el resultado del mismo caso de uso que
consume `vec-simular-provision`, incluida la huella de reproducción local.
Esta huella no es una firma ni un justificante de presentación.

Los textos están disponibles en español e inglés mediante el selector común.
No se monta una API institucional ni se añaden conexiones, permisos o SQL.

## Ensayo de adjudicación global

La pestaña «Adjudicación» compara tres solicitudes sintéticas para dos puestos.
Muestra preferencias y puntuaciones de partida. La política del ensayo identifica
bases, versión, método y cadena de desempates; se pueden reordenar sus criterios y
cambiar su sentido. El método disponible es experimental y no constituye una
regla aprobada por RRHH.

«Simular adjudicación» llama al mismo caso de uso que
`vec-simular-adjudicacion`. Ninguna persona obtiene dos puestos y un puesto
individual no se oferta dos veces. Un empate sin resolver deja el conjunto
pendiente, sin asignaciones. La propuesta reproducida no reserva vacantes ni
firma, publica o ejecuta una resolución.

`GET /api/provision/v1/adjudicaciones-locales` devuelve la configuración y el
resumen sintético. El POST de `/simulaciones` recibe sólo `ejemplo_ref` y
`configuracion`; los resultados de valoración y las solicitudes se fijan en el
servidor. El [contrato del CLI](../vec-simular-adjudicacion/README.md) explica
las entradas completas y los límites del método.

## Reclamación, revisión y borrador de resolución

La pestaña «Reclamación y revisión» ofrece tres ejercicios del mismo provisional:
reclamación pendiente, mantener la puntuación y rectificar un dato sintético.
Cada decisión motivada añade otra versión. La anterior conserva su puntuación,
instantánea y huella; la rectificación recalcula con el motor común. La pantalla
muestra la cronología y las dependencias que siguen pendientes.

El resultado es siempre un borrador de ensayo. No registra reclamaciones,
modifica Personal/RUM, firma, publica ni dicta una resolución oficial.
`GET /api/provision/v1/ciclos-locales` proporciona los casos del catálogo.
`POST /api/provision/v1/ciclos-locales/simulaciones` acepta sólo `ejemplo_ref` y
`caso_ref`; los hechos, evidencias y decisiones sintéticas los fija el servidor.

La futura persistencia queda descrita en
[el contrato de Provisión](../../deploy/postgresql/provision/README.md), con la
reserva `provision 000001`. No contiene SQL ejecutable ni altera el núcleo.
