# F1-117: diagnóstico del rechazo de firma de Jefatura

Trabajo en curso sobre `main@fb5200b68`, el 20 de octubre de 2026. Dirección
priorizó esta tarea a las 09:44 y aparcó F1-62. No hay cambio de código ni de
permisos. No se ha identificado todavía la guarda que devuelve el rechazo.

## Qué acreditan los registros recibidos

Dirección cedió registros técnicos de su copia de F1-103, sin claves ni
material de identidad. Las horas de esta tabla son las del registro, con
desplazamiento `+02:00`:

| Hora | Respuesta | Duración | Consultas | Observación |
| --- | --- | --- | --- | --- |
| 06:57:47 | 201 | 0,59 s | 161 | Registro de firma confirmado |
| 07:00:11 | 503 | 0,33 s | 96 | Contexto cancelado; la telemetría lo clasifica como cancelación del cliente |
| 07:00:12 | 403 | 0,65 s | 182 | PostgreSQL devuelve `42501`; la ruta está ocultada |
| 07:08:49 | 201 | 0,56 s | 161 | Registro de firma confirmado |
| 07:11:12 | 403 | 0,64 s | 182 | PostgreSQL devuelve `42501`; la ruta está ocultada |
| 07:13:35 | 201 | 0,68 s | 185 | Registro de firma confirmado |
| 07:17:06 | 409 | 0,09 s | 30 | Conflicto con `42501`; la ruta está ocultada |
| 07:18:04 | 409 | 0,11 s | 30 | Conflicto con `42501`; la ruta está ocultada |

Los registros permiten distinguir la cancelación, las denegaciones y los
conflictos. No contienen el nombre de la función ni la línea de la excepción
de PostgreSQL. Tampoco ligan los eventos con ruta ocultada a un expediente o
a un cuerpo HTTP. Por tanto, no bastan para atribuir todos estos eventos al
registro de firma ni para asegurar que los intentos comparados son iguales.

Los dos rechazos 403 terminan en menos de un segundo. El registro no
identifica su ruta ni la comprobación que falló.

## Comprobaciones propias

La copia propia se creó desde la revisión indicada, con las fuentes del
lanzador estándar cedido por dirección y la lista SQL completa de `main`.
La creación tardó **101,13 segundos**. No se usaron actos ni servicios de
otra copia.

- Chrome del sistema, versión 155.0.8059.39, con certificado de la copia:
  sesión, preferencias, imagen, cuadro de Contratación y catálogo de alta
  respondieron 200. Esto acredita esas lecturas, no una firma.
- `go test -p 6 -race ./internal/app/bootstrap -run
  '^TestFirmaVecRevalidacionesConservanElMismoContextoAtestado$' -count=1`:
  pasó. Comprueba que las revalidaciones conservan el contexto atestado de
  la petición, también con accesos concurrentes; no sustituye al ensayo SQL.

En esta copia se ha activado `log_error_verbosity=verbose` y
`log_parameter_max_length_on_error=0`. La consulta de `pg_settings` confirmó
ambos valores. El registro conserva el contexto SQL de los errores y omite
los valores de parámetros; ese contexto puede localizar la función y la
línea si se reproduce el rechazo. No modifica funciones ni permisos.

## Qué falta para terminar

Preparar las siete etapas privadas de firma mediante
`deploy/principal/copia_local.py firma`, con material y recibos nuevos de la
copia. La orden versionada ejecuta un manifiesto privado; sus fuentes de
preparación no están en Git. Se ha pedido su cesión a material común para
adaptarlas sin copiar un kit ya ejecutado.

Después hay que firmar como Jefatura en un expediente nuevo desde Chrome,
capturar el cuerpo y la correlación y cotejar la guarda SQL si falla. Si no
se reproduce, conservar esa evidencia y la recuperación tras reiniciar,
sin atribuir una corrección de código. La tarea sigue pendiente de esa
comprobación y de revisión independiente.
