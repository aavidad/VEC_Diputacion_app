# Gobierno del plan nominal de firma

AD177 es un borrador. La instalación se rechaza mientras sus dos preimágenes
estén pendientes: definición del núcleo y CHECK de audiencias después del delta
de L. El CHECK de tipos de auditoría no sustituye el CHECK de audiencias.

CC7 conserva los datos y la publicación original del plan. AD177 comprueba un
consumo nominal nuevo y su auditoría común dentro de la misma transacción. CC7
no consulta tablas de Autorización. La revalidación privada del pin desde CT usa
un consumo de firma vigente, sin conceder permisos de gobierno al firmante.

Los perfiles técnicos de gobierno y lectura aún no existen en la cadena
post-H9/AD173. L conserva su implementación. La fachada de gobierno está preparada;
falta completar la consulta nominal. Ningún LOGIN recibe
EXECUTE de los comprobadores privados de este borrador.

`comprobar_consumo_gobierno_plan_firma_v1(jsonb)` acepta los siete campos del
resultado de consumo y coteja las filas, la decisión, el efecto, la vigencia y
la transacción original. Devuelve las referencias y los datos nominales mínimos
derivados de esa decisión. Actor, perfil y operación no se aceptan del catálogo.

`comprobar_consumo_firma_plan_ct_v1(jsonb)` delega en la comprobación existente
AD167. Sólo concede ejecución a la autoridad de catálogos; conserva el cuerpo y
las ACL originales de AD167.

`registrar_y_confirmar_gobierno_plan_firma_v1` fija el perfil técnico de gobierno,
coteja acción, audiencia, recurso y huella del material, consume la decisión y
confirma el cambio por la fachada CC7 en una llamada SQL. Deriva actor, perfil,
finalidad, proceso y canal del registro común; verifica la caducidad al finalizar.
El identificador debe ser una clave documental común de al menos tres caracteres,
como exige el plan CT; no admite dos puntos. El material enlaza contenido y CAS
mediante `material_sha256`. El servicio común actual aún necesita su decorador
de material operativo para emitir esa autorización ligada al contenido.

La composición debe registrar denegados y errores mediante el puerto común de
intentos publicado por L, después del rollback del efecto. Este borrador no
contiene ese montaje ni una fuente nominal sustitutiva.

Orden previsto: delta de perfiles/audiencias de L → AD177 → CC7. Las fachadas
AD177 podrán referirse a CC7 mediante PL/pgSQL, pero no se invocan ni se exponen
antes de completar esa cadena. Falta el ensayo causal con PostgreSQL real,
las pruebas de concurrencia y recuperación, y dos revisiones del hash final.
