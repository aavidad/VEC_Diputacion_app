# Incorporación y cierre del nombramiento nativo

F1-134 continúa el mismo expediente del caso 4.0 de
[`FLUJO_APP.md`](../../docs/producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación),
desde la resolución de nombramiento. Los pasos y sus recibos siguen pendientes
de comprobar en la candidata conjunta; este documento no acredita su ejecución.

## Antecedente del plan de Personal

El lector del plan usaba la aceptación vigente del circuito de correo, que
exige un justificante. La resolución de Mi Bolsa conserva su respuesta y su
recibo originales, y exige que ese justificante esté vacío. CT255 añade a la
fuente existente de antecedentes B2 una selección específica para esa pareja
de resolución y recibo. Las dos fuentes conservan el criterio vigente de fecha
de confirmación y referencia para elegir el antecedente.

La propuesta debe pertenecer al expediente y a la organización consultados,
conservar el mismo llamamiento y el contenido nativo de aceptación. Las
referencias de respuesta y recibo tienen que coincidir con la resolución
publicada. CT212 mantiene el cotejo posterior con la fuente propietaria de
Bolsa. No se fabrica un justificante ni se traslada un recibo de otra copia.
El lector general de aceptación y el circuito de no incorporación no cambian.

## Activación

CONFIG NUEVA: ninguna. Preparar Personal con la orden existente del
[apartado 12 de la fábrica](LEEME_copia_local.md#12-personal-incorporación-alta-ginpix-manual-y-cese-f1-119),
usando la configuración aprobada y las conexiones propias. Conservar su regla
`cese_fecha_efecto`: no tiene valor por omisión ni se decide en el navegador.

Para preparar un nombramiento desde el principio, crear una copia con
`--modulos --rrhh-original --administracion`. Seguir el
[procedimiento de Organización histórica](organizacion_historica_f1_106_20261020.md)
y el orden de fuentes y firma del
[apartado 6 de la fábrica](LEEME_copia_local.md#6-organización-histórica-y-lectura-de-personal-en-el-servidor-combinado).
La copia con solo `--modulos` arranca, pero no compone esa consulta ni permite
emitir el llamamiento nativo. Los expedientes históricos sin necesidad RPT
no sustituyen el expediente nuevo firmado y fiscalizado del caso 4.0.

CT255 requiere CT249 y CT253. La composición de ensayo `9bd4428a1` conserva
primero la lista de main `24a96efad`, seguida de CT250, CT249, CT253 y CT255;
no incluye CT256. La fila de CT255 se mantiene al final de la lista única:
este corte se integra después de CT249/CT253; la candidata D la conserva
también después de CT256. La focal se añade al registro existente sin
sustituir las de esas dependencias.
No reaplicar migraciones ni presentar como activación los datos preparados
directamente en una copia. No crear por SQL la incorporación, relación,
ocupación, cese o cierre que deben registrar las pantallas.

## Recorrido que se debe comprobar

| Paso | Operación existente | Evidencia |
| --- | --- | --- |
| 17 | El centro acredita la incorporación | Recibo de centro, fecha y documento |
| 18 | RRHH prepara y confirma el plan B2 | Plan con intención durable; recibo de alta, relación y ocupación en Personal |
| 16a–16b | RRHH descarga la ficha y confirma la grabación manual en GINPIX | Ficha ligada al origen B2 y recibo local de número y fecha; sin transmisión automática |
| 19 | RRHH registra el cese | Recibo CT y fin de la relación y ocupación comprobado en la ficha nominal de Personal |
| 20 | Bolsa aplica su política de retorno | Estado Disponible o Disponible desde en Mi Bolsa o en la ficha del candidato |
| 21 | RRHH cierra con el cese CT y GINPIX confirmados | Expediente completado y recibo de cierre |

Si se pierde una respuesta, conservar la misma clave y recuperar la operación;
no crear otra para simular el reintento. Tras reiniciar aplicación y PostgreSQL,
comprobar los mismos recibos y fechas, y la ausencia de duplicados. La ficha de
Personal y el estado de Bolsa son lecturas de sus propietarios. El cierre no
añade una puerta nueva sobre sus recibos: esa ampliación queda en F1-135.

## Estado del corte

La función se ha preparado desde su definición instalada en una copia propia
de la principal. Sin el productor de resolución portal, la migración se ha
rechazado con la función original intacta. El preparador de roles acepta el SQL.
La composición `9bd4428a1` contiene el mismo SQL de CT255 que este corte.
Su ensayo aplicó 142 migraciones al esquema real vacío y pasó 26 pruebas SQL
antes y después de reiniciar PostgreSQL, más dos pruebas de concurrencia.
La focal de CT255 comprobó el contrato privado y la denegación del acceso
directo del ejecutor. Este ensayo no acredita los actos del recorrido.

La composición de fábrica `398499113`, con las etapas F1-131 de `80eb0c03`
todavía pendientes de integración, confirmó catorce actos administrativos,
el plan y cuatro admisiones en una instancia nueva. La preparación terminó
con 503 en `incorporacion-personal-b2/categorias-rpt/v1`. Se conservaron el
material y PostgreSQL en archivos fríos propios; la instancia fue retirada.
Ese rechazo se corrigió en F1-137, integrada en main. La comprobación posterior
del preparador obtuvo las consultas de Personal con respuesta 200 antes y
después del reinicio; fue un diagnóstico sobre su copia existente, no una
creación completa desde la receta corregida.

Las dos revisiones independientes del cuerpo de CT255 y sus permisos son
favorables sobre `77b8da9c0`. Dirección confirmó el 20/10 a las 19:35 que el
revisor comprobó el positivo de correo, el de Mi Bolsa y los rechazos en su
ensayo manual sobre el clon. El guion cedido después no es una prueba autónoma:
depende de datos previos y varios negativos requieren estados que el esquema
prohíbe. Por decisión de dirección, la focal versionada conserva su alcance de
contrato privado y denegación; no se añade ese guion ni se anulan restricciones.

La candidata conjunta D conserva CT255 después de CT253 y CT256. Su responsable
prepara el mismo caso nativo hasta el nombramiento; no se crea otro expediente
en paralelo para estos pasos. Faltan ese recorrido HTTP y en Chrome, el
reinicio con sus datos, la CI de la cabeza final y la integración de las
dependencias. CT255 se integra después de CT253 y de comprobar el positivo
nativo del plan de Personal en el paso 18 de D.
No hay todavía una prueba positiva de alta o cese sobre el nombramiento nativo.
