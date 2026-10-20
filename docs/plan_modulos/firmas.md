# Firmas de Contratación temporal: plan vigente

Corte del 20 de octubre de 2026 sobre `origin/main@3ef7dee0b`.
La firma se realiza con GrxFirma desde Chrome y se verifica antes de registrar
su resultado. Se ensaya en una copia local; cidonia está congelada. Los
certificados y firmas de desarrollo no dan eficacia administrativa.

## Integrado y comprobado

- La composición de firma y la preparación de originales están integradas;
  ya no es correcto afirmar que solo se ejecutan en pruebas.
- El [ensayo del informe](../estudio_requisitos/firma_copia_f1_32_20261019.md)
  acredita las firmas de técnica y Jefatura, el paso a fiscalización y su
  recuperación tras reiniciar. El resultado conserva `firma_eficaz:false`.
- Las [firmas anteriores](../../deploy/principal/firma_historica_ficha_f1_36c_20261016.md)
  se consultan desde la ficha, con el original permitido por el cargo.
- El [guion conjunto de firma](../../deploy/principal/candidata_hz39/firma/LEEME.md)
  conserva el ensayo de cinco firmas del informe y la resolución en dos
  expedientes. Esa evidencia no cierra un expediente único del caso 4.0.

## Qué falta para la aceptación

El [caso único](../producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación)
exige informe con técnica y Jefatura antes de Intervención; después, resolución
con Jefatura, Diputación y Secretaría General. Cada paso usa una persona
ficticia distinta y conserva su justificante. La recuperación debe acreditar
el mismo original y recibo después de reiniciar aplicación y PostgreSQL.
Los otros cuatro documentos del caso se descargan sin firma, identificados
como pendientes; no se simulan sus firmas.

Sigue pendiente completar el caso entero y preparar de forma reproducible
sus etapas de firma desde Git. Las PR de activación por cargos y fuentes del
preparador abiertas el 20/10 no se presentan aquí como integradas. Sus estados
se consultan en el [tablero](../producto/TABLERO_TAREAS.md).

## Preparación y reglas institucionales

La [receta local](../../deploy/principal/LEEME_copia_local.md#11-firma-compuesta-orden-firma)
indica plan publicado, cargos, asignaciones, admisiones, plantillas y
verificador. Requiere GrxFirma 0.0.106 o posterior para cofirmar el informe.
El reinicio recupera el material y los servicios de la misma copia; no repite
publicaciones ni sustituye recibos.

Delegaciones, suplencias y circuito institucional de portafirmas siguen
pendientes de RRHH y Secretaría General en [dudas.md](../../dudas.md).
El conector externo no se da por activo por existir su código. El
[plan general](../producto/PLAN_IMPLEMENTACION.md) fija el cierre y el
[inventario anterior](../historico/plan_modulos/firmas_anterior_20261020.md)
conserva los cortes y ramas de sus fechas.
