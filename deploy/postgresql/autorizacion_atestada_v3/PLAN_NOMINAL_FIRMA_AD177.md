# Gobierno del plan nominal de firma

AD177 se instala después de AD178 y comprueba dos preimágenes medidas el 5 de
octubre de 2026 en el clon (copia fría H10-30 + AD194, IS16, CA36, AUT47,
AD193, AD195, Personal36, AD196, AUT49, AUT48 y AD178):

| Huella | Valor |
| --- | --- |
| `pg_get_functiondef` del núcleo tras AD178 | `2ccd704afe6140d604faa626631e9743edda8f785517d1136054c746cb9b1381` |
| CHECK de audiencias tras AD178 (`pg_get_constraintdef(oid,true)`) | `2e687cbf9055a1c1a94035a74bdbf80fcf37c91b0613a7a5325f5d55def56e64` |

Si cualquiera difiere, o si sus funciones ya existen, aborta con SQLSTATE
`55000` antes de crear nada. El CHECK de tipos de auditoría no sustituye el
CHECK de audiencias. Se aplica una vez, sin DOWN, con
`deploy/principal/lista_sql_claude_firmas_ad178_ad177_20261005.txt`.

El comprobador de gobierno usa el ABI de AD193 publicado por L en
`f37e177b6`: ambos sellos internos `transaccion_origen xid8` deben coincidir
con `pg_current_xact_id()` y la auditoría debe pertenecer a
`consumo_confirmado_v4`, versión 4. Los valores proceden de las filas propias
de la autoridad; el recibo conserva siete propiedades y no admite sellos
enviados por el cliente. Un sello NULL o de otra transacción deniega. La
precondición comprueba las dos columnas de tipo xid8 antes de crear funciones.
No utiliza xmin ni una conversión al xid de 32 bits.

AD193 preserva la ABI de `comprobar_consumo_firma_ct_v1(jsonb)`. AUT41,
CT175 y CT176 mantienen su delegación y el formato del recibo. Este ajuste
no modifica esas migraciones ni AD167 instalada. En el clon, AD177 se aplicó
una vez con código 0 y una segunda aplicación se detuvo en su precondición;
`pruebas_sql/ad177_ad178_post_ad193.sql` comprueba ACL, sello v4 y el rechazo
de un recibo real de otra transacción. Falta el ensayo causal con productores
reales; AD177 no está instalada en la principal.

CC7 conserva los datos y la publicación original del plan. AD177 comprueba un
consumo nominal nuevo y su auditoría común dentro de la misma transacción. CC7
no consulta tablas de Autorización. La revalidación privada del pin desde CT usa
un consumo de firma vigente, sin conceder permisos de gobierno al firmante.

AD178 añade el perfil técnico de gobierno al núcleo; el de lectura del plan
sigue sin contrato. La fachada de gobierno está preparada; falta completar la
consulta nominal. Ningún LOGIN recibe EXECUTE directo de los comprobadores
privados; los LOGIN que heredan un grupo propietario (por ejemplo el de
gobierno H9, que hereda el propietario AUT) sí pueden llamarlos, pero no pueden
producir un consumo de esta transacción, así que siempre deniegan. La fachada
exterior `registrar_y_confirmar_gobierno_plan_firma_v1` sólo tiene EXECUTE para
el grupo runtime CT, porque el núcleo exige que `session_user` pertenezca a él;
ese grupo no acredita a la persona. Hoy hay dos LOGIN en ese grupo. Antes de
activar el gobierno conviene decidir si se usa un grupo técnico de gobierno
dedicado con una sola pertenencia, como en las ramas de administración.

El comprobador de gobierno repite las condiciones de la rama del núcleo
(superficie `administracion_privilegiada`, `cuenta_privilegiada=true` y forma
`catalogo_id:version` del recurso) y exige la acreditación de categoría
Aplicación de AUT, de modo que no depende de que sólo esta fachada use el
perfil. El tipo de recurso es el genérico `catalogo_configurable`: CC7 debe
rechazar cualquier catálogo que no sea el plan nominal de firma, y K debe
limitar la extensión de `acreditar_perfil_aplicacion_nominal_v1` a estas
acciones sobre el plan.

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
mediante `material_sha256`.

La primera versión se invoca desde el kit privado: conserva el fichero de
material y su SHA esperado. Cada reintento obtiene una autorización nueva para
los mismos bytes y la misma clave, sin regenerar fechas, traza, evento o JSON.
El fichero aprobado no concede permisos. Se reutilizan las transiciones de
`CatalogoConfigurable`; la edición web y la recuperación semántica de una orden
reconstruida quedan para V2.

CC7 devuelve el actor, fecha, auditoría y recibo de outbox del efecto original,
incluso si la publicación fue retirada después. AD177 entrega el consumo del
acceso nuevo por separado; no sustituye esos campos históricos por datos del
reintento.

El gobierno usa `administracion_privilegiada`, como la fuente real de Aplicación.
La categoría y las concesiones se revalidan mediante la fachada AUT existente.
Actualmente sus acciones y versión siguen restringidas: K debe publicar la
extensión aprobada. Este borrador rechaza el gobierno hasta entonces; no infiere
la categoría del rol técnico, un nombre de cargo o un identificador propuesto.

La composición debe registrar denegados y errores mediante el puerto común de
intentos publicado por L, después del rollback del efecto. Este borrador no
contiene ese montaje ni una fuente nominal sustitutiva.

Orden: AD193 → AD195/AD196 → AD178 → AD177 → CC7. Las fachadas AD177 se refieren a CC7 por
PL/pgSQL. Al instalar AD177 la fachada de gobierno ya es ejecutable por el
runtime CT, pero deniega siempre mientras falten CC7 y la extensión de K. Falta el ensayo causal con PostgreSQL real,
las pruebas de concurrencia y recuperación, y dos revisiones del hash final.


CT176 consume dos autorizaciones de firma ligadas en la misma transacción. AD177 prepara la fachada exterior `consumir_plan_firma_ct_v2_atestada`, cuyo contexto liga el material y el envoltorio completo (`plan_firma_sha256`). El envoltorio conserva el descriptor de once claves, el pin publicado y la SHA de la decisión interior. La fachada compara actor, perfil, versión del rol, acción y recurso de ambas decisiones. No añade un perfil al núcleo ni modifica AD170.

Después del registro CT172, `recuperar_consumo_firma_plan_ct_v1` relee el consumo interior actual desde tablas propias AD, exige que los bytes de decisión sean idénticos a los conservados y lo comprueba mediante AD167. Devuelve sólo los siete campos del recibo; no presta un recibo histórico. CT176 revalida el pin mediante CC7 y conserva ambos vínculos antes de COMMIT. Las fachadas nuevas sólo reciben EXECUTE para el propietario CT; el LOGIN no puede invocarlas directamente.

Los datos compartidos se cotejan con el material y el descriptor. `esquema_contexto`, `mapeo_version` y `mapeo_fuente_ref` se verifican por la publicación íntegra fijada, sin afirmar un cotejo independiente de la derivación del selector. Ninguna de estas preparaciones acredita todavía ensayo, instalación ni firma nominal.
