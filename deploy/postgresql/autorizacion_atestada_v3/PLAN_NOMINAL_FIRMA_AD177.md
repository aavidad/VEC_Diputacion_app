# Gobierno del plan nominal de firma

AD177 es un borrador. La instalación se rechaza mientras sus dos preimágenes
estén pendientes: definición del núcleo y CHECK de audiencias después del delta
de L. El CHECK de tipos de auditoría no sustituye el CHECK de audiencias.

CC7 conserva los datos y la publicación original del plan. AD177 comprueba un
consumo nominal nuevo y su auditoría común dentro de la misma transacción. CC7
no consulta tablas de Autorización. La revalidación privada del pin desde CT usa
un consumo de firma vigente, sin conceder permisos de gobierno al firmante.

Los perfiles técnicos de gobierno y lectura aún no existen en la cadena
post-H9/AD173. L conserva su implementación. Quedan por completar las fachadas
que consumen material V3 y confirman gobierno o consulta. Ningún LOGIN recibe
EXECUTE de los comprobadores privados de este borrador.

`comprobar_consumo_gobierno_plan_firma_v1(jsonb)` acepta los siete campos del
resultado de consumo y coteja las filas, la decisión, el efecto, la vigencia y
la transacción original. Devuelve las referencias y los datos nominales mínimos
derivados de esa decisión. Actor, perfil y operación no se aceptan del catálogo.

`comprobar_consumo_firma_plan_ct_v1(jsonb)` delega en la comprobación existente
AD167. Sólo concede ejecución a la autoridad de catálogos; conserva el cuerpo y
las ACL originales de AD167.

Orden previsto: delta de perfiles/audiencias de L → AD177 → CC7. Las fachadas
AD177 podrán referirse a CC7 mediante PL/pgSQL, pero no se invocan ni se exponen
antes de completar esa cadena. Falta el ensayo causal con PostgreSQL real,
las pruebas de concurrencia y recuperación, y dos revisiones del hash final.
