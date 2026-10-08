# Objetivos de VEC

Actualizado el 8 de octubre de 2026 sobre `origin/main@033fda6ed` y el estado de cidonia comprobado ese día (HZ8 instalado).

El objetivo de ahora es cerrar Bolsa y Contratación temporal (CT). Los demás módulos están aparcados hasta entonces (orden de Alberto, 07/10/2026). Después van Cronos y Dietas, por ese orden.

## Qué significa «Bolsa y CT cerradas»

Se dan por cerradas cuando se cumplen las cuatro condiciones siguientes en cidonia, con datos sintéticos. Cada una se comprueba allí, no en un clon ni en una prueba aislada.

1. **Recorrido completo de RRHH en el navegador, con reinicio.** Con Chrome y certificados sintéticos de personas distintas: petición del centro, ratificación, alta, análisis con crédito, oferta desde la bolsa con varias plazas, aceptación, informe, fiscalización con un reparo y su subsanación, resolución firmada por dos personas, GINPIX, paso a Personal, cese y vuelta de la persona a su bolsa. Después se reinician la aplicación y PostgreSQL y se repiten las consultas: mismos recibos, mismas fechas y ningún registro duplicado. Queda un acta con la respuesta HTTP y una captura de cada paso.
2. **Selectores encendidos.** Están activos los que necesita ese recorrido: `VEC_BOLSA_BORRADORES_ENABLED`, `VEC_BOLSA_PORTAL_CANDIDATO_ENABLED`, `VEC_DOCUMENTOS_ENABLED`, `VEC_BOLSA_CESE_CT_ENABLED`, `VEC_PERSONAL_B2_GOBIERNO_ENABLED`, `VEC_CT_INCORPORACION_ACREDITADA_ENABLED`, `VEC_FIRMA_VERIFICACION_ENABLED` y `VEC_RRHH_AUDITORIA_ENABLED`. El aviso de dependencias al arrancar (#745) no muestra ninguna ausencia.
3. **Ningún «no disponible» sin motivo.** Cada botón o enlace que ve RRHH funciona o explica en pantalla por qué no está y qué falta (por ejemplo, una respuesta pendiente de RRHH). El registro de la aplicación no tiene respuestas 404 ni 503 de rutas de Bolsa o CT durante el recorrido.
4. **Lecturas rápidas.** Con el volumen de la principal, las pantallas de lectura de Bolsa y CT (cuadro de CT, lista de expedientes, ficha, bolsas, candidatos y ficha del candidato) responden por debajo de 300 ms en el percentil 95, medido por HTTP en cidonia. La base común (sesión, módulos y preferencias) baja de 100 ms.

El correo sale hoy por un buzón de pruebas. Para la presentación es correcto y no impide el cierre; para el uso real hará falta el correo de la Diputación (punto 9).

## Estado de cidonia hoy

- Instalado: HZ8 (CT193, B87, canales del llamamiento, Categorías de la RPT en el portal y reglas de Bolsa v5).
- Encendidos: `VEC_BOLSA_PORTAL_CANDIDATO_ENABLED` y `VEC_DOCUMENTOS_ENABLED`.
- Apagados o ausentes: `VEC_BOLSA_CESE_CT_ENABLED` (ausente), `VEC_PERSONAL_B2_GOBIERNO_ENABLED=false`, `VEC_CT_INCORPORACION_ACREDITADA_ENABLED=false` y la auditoría de RRHH. La verificación de firma estaba apagada según el informe del 08/10; hay que comprobarla.
- PR fusionadas el 08/10: #895, #896, #897, #898, #899, #900, #901, #902 y #904. Las que lleven SQL o configuración nueva tienen que entrar en el siguiente despliegue.

## Lo que falta, en orden

1. **Encender en cidonia lo que ya existe.** Vuelta a la bolsa tras el cese (selector y LOGIN de #771), paso a Personal con su SQL pendiente (AD211/P22, AD175/P32, AD180/P34 y siguientes), incorporación acreditada, auditoría de RRHH y verificación de firma. Lo hace Claude.
2. **Fusionar las PR abiertas de Bolsa y CT**: #774 (cese de bolsa no constituida), #910 (ficha sin bolsa), #906 (nombres de centro), #905 (peticiones del centro), #907 (validación del alta en SQL), #540 (auditoría de consultas fallidas) y #909 (área personal).
3. **Arreglar ORQ-1**: el expediente queda bloqueado si la selección falla entre CT y Bolsa. No tiene dueño.
4. **Repetir el recorrido completo** con el main de ese momento y dejar el acta. Lo que falle se anota como tarea aparte.
5. **Carga de bolsas desde el Excel de CONVOCA en pantalla**: #751 → #752 → #759, con AD218 y B80 en cidonia.
6. **Firma de dos personas de punta a punta** (Codex-V, ver [firmas.md](firmas.md)) y su recorrido.
7. **Plazos de CT editables** (Codex-Y) y **tarjetas de plazo en Inicio** (CT192, Codex-S). Después se cierran #233, #237 y #240.
8. **Velocidad**: cuadro de CT por debajo de 300 ms (último p95: 363–372 ms), preferencias por debajo de 100 ms (Codex-V) y medición de todas las lecturas en cidonia.
9. **Correo de la Diputación configurable desde Administración**, cuando Informática dé los datos del servidor.
10. **Cifras globales de Bolsa pulsables**, con su lectura global paginada y auditada (equipo V).
11. **Limpieza menor**: retirar la ruta muerta del cierre administrativo y ajustar el catálogo de reglas a la duda 147.

El detalle de cada punto está en [bolsa.md](bolsa.md), [contratacion_temporal.md](contratacion_temporal.md) y [firmas.md](firmas.md).

## Lo que depende de RRHH o del DPD

Nada de esto impide cumplir las cuatro condiciones con los valores de ejemplo, que se cambian en catálogo sin tocar código.

| Quién | Qué | Dudas |
| --- | --- | --- |
| RRHH | Plantillas Word de contratos, informes y resoluciones | 124 |
| RRHH | Oferta al SAE: datos, canal, criterios de Mesa General y documento que acredita la selección | 72, 73, 126 |
| RRHH y Secretaría General | Delegaciones de firma y suplencias; perfiles fijos | 122, 128, 143 |
| RRHH e Informática | Firmadoc o AutoFirma; SICAL; GINPIX; servidor de correo | 74, 125, 127, 88 |
| RRHH | Toma de posesión eficaz y fecha de efectos del cese | 140 |
| RRHH | Plazo para aportar documentos tras aceptar; audiencia antes de la baja | 18, 62 |
| RRHH | Urgencia, coste sin fecha de fin y plazo de respuesta tras llamar por teléfono | 63, 146, 151 |
| RRHH | Que el correo valga como publicación a efectos del art. 8.1 del Reglamento | — |
| DPD | Campos y acceso de la lista pública de cada bolsa | 17, 83 |
| DPD | Uso del correo y teléfono que vienen de CONVOCA | 45 |
| DPD | Conservación de las actuaciones de aspirantes externos | 97 |

Se pueden retirar de `dudas.md` citando la norma: la 95 (RD 424/2017, arts. 10.2 y 12.4) y la 147 (Reglamento de bolsas, arts. 8 y 11). Las fuentes y propuestas están en la PR #908.

## Después de Bolsa y CT

1. **Cronos**, con su plan en [cronos.md](cronos.md).
2. **Dietas**, con su plan en [dietas.md](dietas.md).

Los demás módulos siguen aparcados. Sus planes se conservan con esa marca y se retoman uno a uno, cada uno después de las bases que necesita.
