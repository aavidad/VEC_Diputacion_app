# Datos del recorrido de la candidata HZ39

Estos guiones preparan el puesto, la plaza, la participación de Antonio y la denominación del cargo de Manuel del [caso de aceptación](../../../../docs/producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación):

| Dato | Valor |
| --- | --- |
| Centro | 520, Transformación Digital en el catálogo; Servicios Digitales y Asesoría Tecnológica a Municipios en el recorrido |
| Categoría | Administrativo, C1, código 9 de la RPT |
| Puesto | 9/01, revisión 1 |
| Plaza | 2026/0033, revisión 1 |
| Bolsa | `bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk`, versión 1, ya constituida |
| Aspirante | Antonio Reyes Álvarez; puesto 5 del acta y turno vigente 1, porque los cuatro anteriores no están disponibles |

Las referencias del organismo del recorrido están en [seleccion.json](seleccion.json).
El guion añade únicamente el Administrativo que faltaba en Personal: un
tipo de puesto, su dotación, un puesto, una plaza y su vínculo. Reutiliza la
plantilla publicada y su cobertura de ocupaciones. Conserva la RPT y los 32 puestos
anteriores. La cobertura existente está vigente hasta el 1 de enero
de 2027.

## Preparación y ejecución

Dirección prepara la candidata en una copia local de la principal, con HZ37
y las SQL nuevas de la candidata instaladas en su orden. Deben existir la
RPT publicada de 2026, Personal40, el centro 520, la bolsa constituida y los
catálogos publicados `regimen:funcionario_interino` v1 y `modalidad:vacante` v1.

P conserva la preparación de la identidad de Antonio con los guiones existentes en
[inscripcion_recorrido_20261013](../../inscripcion_recorrido_20261013/LEEME.md).
[antonio_disponible.sql](antonio_disponible.sql) parte de la fila del acta
vigente que lleva el nombre de Antonio, la del puesto 5, «Antonio Reyes
Álvarez». Sus referencias de candidatura y participación están en
`seleccion.json`. Las cuatro personas de los puestos 1 a 4 pasan a **No disponible** con una
clave fija, así que el primer turno es el de Antonio. El guion solo las marca
si siguen disponibles; si alguna ya cambió de situación (por ejemplo, porque el
recorrido la llamó o la contrató), se para sin tocar nada. No comprueba
llamamientos u ofertas abiertos a esas personas: se ejecuta antes de llamar a
nadie. Se conservan la bolsa
sustituida y el orden del acta. La fuente de Bolsa que usa el llamamiento CT
debe devolverlo como primer candidato disponible, con orden de acta 5; si no
lo hace, la transacción se revierte.
Este guion no crea otra persona, candidatura, bolsa ni inscripción, ni mueve
vínculos de personas.

El recorrido principal empieza en la petición del centro, con Antonio ya en la
bolsa. Los pasos e1 a e5 son una demostración aparte, no bloqueante, como indica
FLUJO §4.0. No se necesita ejecutarlos para preparar el llamamiento del paso 9.
La persona del alta en Personal se obtiene
del vínculo de Bolsa confirmado al aceptar, nunca mediante una búsqueda por
nombre.

Desde la raíz del repositorio, con la conexión de la copia local ya configurada
y como superusuario:

```sh
psql -X -v ON_ERROR_STOP=1 \
  -v organismo=org_0e4b1d56ba2e24fb8e4808cbc7abb660 \
  -f deploy/principal/candidata_hz39/datos/organizacion_520_administrativo.sql
psql -X -v ON_ERROR_STOP=1 \
  -f deploy/principal/candidata_hz39/datos/antonio_disponible.sql
psql -X -v ON_ERROR_STOP=1 \
  -f deploy/principal/candidata_hz39/datos/cargo_manuel.sql
```

La lista de ejecución es
[lista_sql_codexk_datos_recorrido_20261014.txt](../../lista_sql_codexk_datos_recorrido_20261014.txt).
Las credenciales se mantienen fuera de Git y de los argumentos del comando.
Si se usa otro organismo, se pasa su referencia mediante `-v organismo`;
las referencias de puesto, plaza y plantilla se calculan para ese organismo
y serán distintas de las de `seleccion.json`.

Si se aplicó la receta anterior con una plantilla adicional, se restaura la
copia de partida antes de usar esta versión; el guion rechaza esa carga
incompatible y conserva su historia. No se borra ni se reescribe lo instalado.

Cada guion se ejecuta en una transacción. La receta de Personal añade cinco
filas de historia. La de Bolsa añade la situación de Antonio
y la traza que registra el disparador existente. La segunda
ejecución reconoce las mismas referencias y termina sin cambios. Repetirla
después del recorrido conserva también sus actuaciones posteriores; no vuelve a
poner disponible a Antonio ni restablece su turno. Una carga distinta con esas
referencias produce un error y revierte la transacción. No se cambian funciones, permisos permanentes ni disparadores.
Es una carga de datos; no se reserva un número de migración ni se ejecuta DOWN.

## Actuaciones que se hacen en pantalla

La petición del centro, ratificación, expediente, llamamiento, aceptación,
firmas, incorporación, alta en Personal, ocupación y cese se registran desde
las pantallas del recorrido. Ninguna de estas actuaciones se precarga aquí.

La Diputada ficticia es **Beatriz Navarro Salcedo**, definida en
[firmantes.json](../../../../scripts/operacion/firma_ct_tres_pasos/firmantes.json).
Su usuario sigue siendo `rrhh_diputacion` y su perfil de circuito,
`perfil:ct:diputacion_delegada_rrhh`. Al preparar los datos de los firmantes se
usa ese fichero como fuente; los certificados, claves, provisión nominal y
recibos quedan fuera de este paquete.

## Reutilización y comprobación

La receta reutiliza los INSERT de la carga que K entregó para el recorrido,
que proceden de
[organizacion_rpt_2026_v1.sql](../../../postgresql/personal/datos/organizacion_rpt_2026_v1.sql).
La revisión detectó que la plantilla adicional del primer guion impedía
consultar las vacantes: Personal necesita una sola plantilla publicada por
organismo y fecha. Esta versión vincula la plaza a la plantilla ya existente;
no crea otra plantilla ni cobertura.

Es un ajuste de los datos ficticios de la candidata. La plaza nueva conserva
la fuente y huella de su propia fila; la plantilla y cobertura anteriores
conservan las de su carga de 32 plazas. No se recalculan esas fuentes ni se
presenta la ampliación como una revisión oficial de plantilla. El guion se
usa únicamente en la base ficticia del recorrido, no con datos reales.

SHA256 de la receta de Personal:
`9b4052414b3f8d14e2aec4cce4cf0a3d35e53fb1437fa8ea8d5148746ad1ac17`.

La carga original y su repetición se comprobaron en la copia K con PostgreSQL
18 y HZ37; esa comprobación no incluía la consulta de vacantes. La comprobación
focal de esta corrección se indica en la PR. El recorrido completo de Chrome y
el reinicio corresponden al ensayo único de P. Este paquete no cambia el
arranque.

El ajuste de Antonio reutiliza `constitucion_entrada`, `vinculo_candidato`,
`situacion_participacion` y `leer_origen_llamamiento_ct_v1` existentes. No cambia
las funciones, permisos, disparadores ni reglas de orden. Las referencias de su
persona, candidato y participación se conservan en `seleccion.json`; el nombre
no se utiliza como clave.

Dirección cambió la referencia del caso a la bolsa vigente: la del 17/09 ya
estaba sustituida por el acta del 12/10. Esa sustitución y ambas actas se
conservan. No se añaden integrantes al acta ni se cambian canónicos o huellas.

Hasta F1-100, `seleccion.json` y este guion tomaban como de Antonio la fila 1
del acta, que es de Isabel Ramírez Vargas. La lista de la bolsa mostraba ese
nombre en la participación de Antonio, y su propia fila, la del puesto 5,
aparecía como de otra persona. El nombre visible sale del acta protegida, que
solo la aplicación puede descifrar; por eso `seleccion.json` y el guion fijan
las referencias de la fila que lleva su nombre, y el nombre no se puede
comprobar en SQL. En una copia preparada con la versión anterior, este guion
añade las cuatro situaciones y no deshace nada, pero la identidad de Antonio
de esa copia sigue ligada a la fila 1: hay que crear la copia de nuevo.

Desde F1-99 el guion de Bolsa ya no mueve el vínculo de la persona de origen
(`persona_ref` de `seleccion.json`) a la candidatura vigente. Esa persona
conserva su permiso de Mi Bolsa ligado a la candidatura de la bolsa sustituida,
y la aplicación no admite cambiar sus ámbitos: Mi Bolsa respondía 403. En su
lugar, `copia_local.py crear --modulos` llama a `preparar_candidato_desarrollo.sh
--candidato-ref` y crea una identidad nueva de Antonio, con su certificado,
ligada a `candidato_ref`, la candidatura del puesto 5. El permiso lo publica la aplicación al arrancar. El
apartado 2 de `LEEME_copia_local.md` explica el resto de la preparación.

Comprobación del ajuste de Antonio en PostgreSQL 18.4, sobre la copia actual de
HZ37, con la versión anterior del guion (que aún movía el vínculo): dos ejecuciones, mismo estado en las 11 tablas comparadas. Con la categoría
de la petición conservada en K, la fuente del llamamiento CT devuelve a Antonio
como disponible en el turno 1 de la bolsa vigente (19,278 ms). La segunda carga
tardó 78,59 ms, incluidos Docker y psql. El recorrido completo de pantalla y
reinicio sigue siendo el ensayo de P.

El guion `cargo_manuel.sql` añade una revisión del catálogo organizativo en
borrador y conserva las anteriores. La denominación del puesto de Manuel
queda «Subdirección de Informática». La fuente de esa carga es
`datos:recorrido:hz39`; no representa un acto administrativo ni cambia
cargos, adscripciones, personas, permisos o recibos. Conserva el canon de
texto del catálogo para que la pantalla pueda editarlo después con su huella.
