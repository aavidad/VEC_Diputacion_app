# Contratación temporal: plan vigente

Corte del 20 de octubre de 2026, sobre `origin/main@3ef7dee0b`.
La fase 1 conecta la petición del centro con Bolsa, firma, Personal y cese.
Su aceptación se rige por el [caso 4.0](../producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación),
en una copia local con datos ficticios. Cidonia permanece congelada.

## Qué está integrado

La ficha permite tramitar peticiones ratificadas, análisis con crédito,
cobertura y unidad, informe, fiscalización, llamamiento, formalización,
documentos e incorporación. La receta de copia local prepara también Personal
y cese. Estas capacidades integradas no acreditan que toda la secuencia esté
recorrida en el mismo expediente.

Hay evidencias parciales de [documentos desde la ficha](../historico/plan_modulos/contratacion_temporal_anterior_20261020.md),
[firma y fiscalización](../estudio_requisitos/firma_copia_f1_32_20261019.md),
[adjudicación de dos plazas](../../deploy/principal/aceptacion_dos_plazas_f1_55_20261018.md)
y [cese sin vínculo previo](../estudio_requisitos/recorrido_f1_30_cese_sin_vinculo_20261020.md).
Cada una conserva su alcance; ninguna cierra el caso único.

## Orden de continuación

1. Petición del centro, ratificación por otra persona, alta con número MOAD,
   análisis con crédito, vía de cobertura y unidad.
2. Informe firmado por técnica y Jefatura; fiscalización, incluida la variante
   de reparo y subsanación.
3. Llamamiento desde la bolsa constituida después de fiscalización favorable;
   comunicación, respuesta del aspirante y resolución de la aceptación.
4. Propuesta y resolución; firma de la resolución por Jefatura, Diputación
   y Secretaría General, en el orden del catálogo. Los otros cuatro documentos
   del caso siguen descargables sin firma, identificados como pendientes.
5. Acreditación de incorporación por el centro, alta y ocupación en Personal,
   descarga y confirmación de grabación manual en GINPIX.
6. Cese, vuelta a Bolsa y cierre. Después se reinician aplicación y PostgreSQL
   y se comprueban recibos, fechas, documentos y efectos sin duplicados.

La variante de vacante y la de varias plazas conservan sus propios datos y
recibos. La continuación desde una oferta adjudicada sigue pendiente; no debe
presentarse una oferta como un nombramiento.

## Cierre, activación y límites

El hito es un expediente completo del
caso 4.0 con reinicio antes del 6 de noviembre. Si no se llega, se recalcula
el final de fase 1, estimado para la segunda quincena de noviembre. Se mantienen
las condiciones del [plan general](../producto/PLAN_IMPLEMENTACION.md#5-fases-y-criterios-de-cierre):
funciones necesarias y auditoría activadas, botones útiles sin rutas ausentes
ni fallos de disponibilidad y tiempos de lectura medidos bajo el objetivo.

La [receta local](../../deploy/principal/LEEME_copia_local.md) fija la preparación;
la [guía del recorrido](../../GUIA_RECORRIDO_ALBERTO.md), el uso desde pantalla.
GINPIX es una confirmación de grabación manual y el correo es local. Las
firmas de desarrollo no dan eficacia administrativa. Lo pendiente de RRHH
continúa como valor provisional configurable y pregunta abierta en
[dudas.md](../../dudas.md).

El [tablero](../producto/TABLERO_TAREAS.md) conserva los responsables y estados.
El [plan de firmas](firmas.md) conserva sus dependencias. El
[inventario anterior](../historico/plan_modulos/contratacion_temporal_anterior_20261020.md)
queda fechado como histórico.
