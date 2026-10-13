# Datos del recorrido de la candidata HZ39

Este guion prepara el puesto y la plaza del [caso de aceptación](../../../../docs/producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación):

| Dato | Valor |
| --- | --- |
| Centro | 520, Transformación Digital en el catálogo; Servicios Digitales y Asesoría Tecnológica a Municipios en el recorrido |
| Categoría | Administrativo, C1, código 9 de la RPT |
| Puesto | 9/01, revisión 1 |
| Plaza | 2026/0033, revisión 1 |
| Bolsa | `bolsa:administrativo:2026-09-17`, versión 1, ya constituida |
| Aspirante | Antonio Reyes Álvarez; disponible tras incorporarlo en pantalla (paso e4) |

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
Antonio se inscribe y aporta sus méritos en pantalla; Carmen admite la solicitud
y lo incorpora a la bolsa en los pasos e1 a e4. Antes del llamamiento P debe
comprobar en Bolsas que Antonio está **Disponible** y tiene el turno previsto.
Este SQL no crea candidaturas, no cambia su orden ni altera su situación.
La persona del alta en Personal se obtiene
del vínculo de Bolsa confirmado al aceptar, nunca mediante una búsqueda por
nombre.

Desde la raíz del repositorio, con la conexión de la copia local ya configurada
y como superusuario:

```sh
psql -X -v ON_ERROR_STOP=1 \
  -v organismo=org_0e4b1d56ba2e24fb8e4808cbc7abb660 \
  -f deploy/principal/candidata_hz39/datos/organizacion_520_administrativo.sql
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

La carga se realiza en una transacción. La primera ejecución añade cinco
filas de historia. La segunda reconoce las mismas referencias y termina sin
cambios. Una carga distinta con esas referencias produce un error y revierte
la transacción. No se cambian funciones, permisos permanentes ni disparadores.
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

SHA256 de esta receta:
`9b4052414b3f8d14e2aec4cce4cf0a3d35e53fb1437fa8ea8d5148746ad1ac17`.

La carga original y su repetición se comprobaron en la copia K con PostgreSQL
18 y HZ37; esa comprobación no incluía la consulta de vacantes. La comprobación
focal de esta corrección se indica en la PR. El recorrido completo de Chrome y
el reinicio corresponden al ensayo único de P. Este paquete no cambia el
arranque ni añade configuración.
