# Consulta histórica de firmas V2 en Contratación temporal

El cliente consulta `POST /api/vec/contratacion-temporal/firmas-documento/recuperaciones-v2` mediante el transporte HTTP interno existente. Recibe del consumidor CT un selector de siete campos ya resuelto por una fuente confiable. La vista no pide a RRHH la clave de operación, el paso, la vía ni la huella del catálogo.

Antes de mostrar una firma, el cliente comprueba el esquema y las claves de la respuesta, cruza cada recuperación con su firma y revisión, y calcula SHA-256 sobre los bytes UTF-8 del canon histórico recibido. Después descarta el canon bruto. La vista muestra fecha y estado histórico; deja recibo, referencias y huellas en un detalle técnico. Al cancelar, perder autorización o cambiar de selector elimina los resultados visibles. No ofrece descarga ni copia.

La hoja de vista permanece **sin montaje productivo** hasta que una fuente CT entregue ese selector a partir del expediente y de la firma original. Tampoco hay todavía un recorrido nominal de extremo a extremo: la fuente de actor y la autorización histórica del servidor siguen pendientes. `testdata/recuperacion-firmas-v2-demo.html` usa datos sintéticos y un transporte simulado; su JSON lo generó la proyección Go del handler de #590. La demo verifica presentación y limpieza de datos, no consulta una firma real.
