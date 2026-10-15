# F0-27: arranque cuando Contratación temporal no se compone

Esta entrega no añade configuración ni SQL. No requiere actos de administración ni cambios de datos.

En la copia local de la principal, arrancar la candidata con la configuración completa de Contratación temporal y comprobar el caso de `docs/producto/FLUJO_APP.md` §4.0. Reiniciar el proceso y repetir la consulta del expediente y la sesión. Después, en una segunda copia con material ficticio, omitir la configuración de CT y arrancar el mismo binario. No modificar la configuración de la principal para este ensayo.

Con CT disponible, sus rutas deben conservar las respuestas y los permisos anteriores. Sin CT, `/livez` responde 200, `/healthz` responde 503, `/portal-empleado/` se sirve y `/api/vec/session` responde a una persona con certificado de desarrollo. La consulta pública de convocatorias y las consultas públicas de Personal siguen disponibles si sus propias fuentes son válidas. Las rutas de CT y las capacidades que consumen su autoridad responden 503 con `servicio_no_disponible`. Una petición sin certificado no supera mTLS.

El registro técnico anota `contratacion_temporal_no_disponible`, el componente y una clase de causa cerrada. No incluye textos libres del error, rutas privadas ni conexiones. La falta del borrador de llamamiento se clasifica como `borrador_llamamiento_no_disponible`.

La raíz compartida impide montar por separado Bolsa RRHH, Documentos, Dietas, Cronos, Personal empleado y Usuarios cuando falla CT. Sus rutas dependientes quedan cerradas en este estado. El ensayo completo con y sin CT, incluida la recuperación tras reiniciar, corresponde a la candidata integrada por dirección.
