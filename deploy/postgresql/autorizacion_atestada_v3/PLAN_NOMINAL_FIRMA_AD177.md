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

## Grupo técnico dedicado (AD200)

Dirección decidió el 5 de octubre de 2026 que el gobierno del plan no lo ejecute
el runtime CT, sino un grupo técnico propio con una sola pertenencia, como el
lote de Administración (AD190). AD200 hace tres cosas, en una transacción:

- crea el grupo NOLOGIN `vec_plan_firma_gobierno_ejecutor`, con `CONNECT` sobre
  la base y `USAGE` sobre el esquema;
- añade al núcleo una rama de sesión para `gobierno_plan_nominal_firma_ct` que
  exige `login_gobierno_plan_firma_valido_v1()` y saca ese perfil de la
  clasificación genérica del runtime CT; la rama de contrato de AD178 no cambia;
- retira `EXECUTE` de `registrar_y_confirmar_gobierno_plan_firma_v1` al runtime
  CT y lo concede al grupo nuevo.

El LOGIN válido no tiene atributos privilegiados ni configuración propia,
pertenece sólo a ese grupo (con INHERIT, sin SET ni ADMIN) y no usa `SET ROLE`.
El DBA lo crea fuera de las migraciones, por ejemplo:

```sql
CREATE ROLE <login_gobierno_plan> LOGIN INHERIT PASSWORD '<secreto fuera de Git>';
GRANT vec_plan_firma_gobierno_ejecutor TO <login_gobierno_plan> WITH INHERIT TRUE, SET FALSE;
```

El núcleo exige además el origen del consumo (AD172): `resolver_origen_consumo_v1`
busca `login_nombre=session_user` en `configuracion_origen_consumos_v1`. El DBA
inserta, también fuera de Git, una fila por operación para el LOGIN nuevo con la
audiencia `vec_catalogos_configurables.plan_nominal_firma.gobierno.v1`, las
operaciones `vec.catalogos.crear`, `vec.catalogos.actualizar`,
`vec.catalogos.publicar` y `vec.catalogos.retirar`, el proceso `vec-admin` y el
canal `administracion_privilegiada`. Sin esas cuatro filas todo consumo real se
deniega con `42501 origen de consumo no acreditado`. Las filas que pudiera haber a
nombre del runtime CT no se pueden modificar ni borrar (la tabla es de solo
adición); quedan inertes, porque el runtime CT ya no ejecuta la fachada y la
rama de sesión lo rechaza.

Pertenecer al grupo no acredita a la persona: cada llamada sigue necesitando
una decisión V3 nominal de la audiencia de gobierno, emitida en la superficie
`administracion_privilegiada` con cuenta privilegiada y el perfil de Aplicación
que AUT51 amplió.

Preimagen medida en clon sobre main con AD190, CA35, AUT44 y la lista de Bolsa
contacto (AD197/B78, #727), que entra antes: núcleo
`c4d11c9e7a39726df85f25bc040ef0cb33a387032d8d147ba3420d5fa24b6e24`
(fuente `e689c573…`). Postimagen en ese clon: `546f341d…` (fuente `d02bb7f3…`).
El CHECK de audiencias no cambia. Una segunda aplicación se detiene en la
precondición. `pruebas_sql/ad200_grupo_gobierno_plan_firma.sql`, dentro de un
ROLLBACK, comprueba que un LOGIN exclusivo del grupo pasa la rama de sesión, que
uno con otra pertenencia se rechaza en ella y que el runtime CT ya no tiene
`EXECUTE` sobre la fachada.
`pruebas_sql/ad177_ad178_post_ad193.sql` es anterior a AD190: mide el núcleo
previo y espera el `EXECUTE` del runtime CT, así que no se ejecuta tras AD190 ni
tras AD200.

## Ámbitos del recurso de gobierno (AUT52, CC9 y AD201)

AD177 y CC7 calculaban la huella de contexto del recurso de gobierno sin
ámbitos. El PDP común exige que el recurso tenga exactamente las dimensiones de
la asignación del actor, y la del administrador con Rol7 tiene organización y
unidad, así que toda decisión de gobierno se denegaba (`ambito_no_autorizado`).
Dirección aprobó el 5 de octubre de 2026 que el recurso lleve esos dos ámbitos.

- AUT52, `vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(versión,
  asignación, persona, organización, unidad)`: la asignación actual de
  Aplicación (criterio de versión de AUT48) está activa, vigente, es de esa
  persona y tiene esa organización y esa unidad. Sólo para el propietario AD.
- CC9, `confirmar_gobierno_plan_nominal_firma_v2(material, organización,
  unidad, consumo)`: la confirmación de CC7 con la huella con ámbitos; mismas
  tablas, replay y recibos. Sólo para el propietario AD.
- AD201, `registrar_y_confirmar_gobierno_plan_firma_v2(material, organización,
  unidad, …)`: las comprobaciones de la v1 con la huella con ámbitos; exige que
  la decisión consumida sea la recibida, acredita persona, organización y
  unidad con AUT52 y confirma con CC9. La v1 deja de ser ejecutable por el grupo
  dedicado.

En Go, `plannominal.RecursoGobiernoPlanFirma(material, AmbitoGobiernoPlanFirma)`
construye el mismo recurso. El material del kit no cambia.

Pruebas en ROLLBACK: `pruebas_sql/ad201_gobierno_plan_firma_ambitos.sql` (con
ámbitos llega al núcleo; huella antigua, organización mal formada y v1 se
rechazan), `catalogos_configurables/pruebas_sql/plan_nominal_firma_ambitos_cc9_positivo_clon.sql`
(el recorrido positivo de CC7 con la huella con ámbitos; la v1 y otra unidad no
ligan la decisión) y `autorizacion/pruebas_sql/aut52_ambitos_gobierno_plan_firma.sql`
(contra la asignación real del clon). La prueba de AD200 es anterior a AD201.
