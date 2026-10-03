# Provisión privada de cargos CT

AD160 añade una fachada privada a la autoridad central. Cargo significa el
`RolID` exacto de una versión publicada. La persona, su perfil real de CA,
organización, unidad y vigencia quedan en `asignacion_perfil`. El kit no crea
identidades, sesiones ni concesiones administrativas.

La fuente cedida de AD160 es `14044dbd007a36798076e7227c7f2387da423696`.
AD164 es su sucesora aditiva: comprueba la independencia entre proponente y
aprobador, conserva los planes y recibos anteriores y retira `EXECUTE` de las
cuatro fachadas privadas. Sus cuerpos, la función interna y la proyección del
adjunto rechazan con `42501` antes de leer, escribir o auditar datos. No modifica
AD160, CA24 ni AUT31.

El efecto SQL queda cerrado hasta integrar la auditoría común nominal y la
categoría Administrador Aplicación del catálogo central. La comprobación de
independencia por sí sola no concede acceso. Se conservan las autoridades
históricas que heredan al propietario: sus llamadas y su `SET ROLE` tampoco
abren los cuerpos publicados. Quien posee capacidad DDL del esquema puede
cambiar una función; el cierre no elimina esa autoridad. Dirección debe revisar y ensayar
la candidata exacta antes de instalarla; no acredita instalación principal.

## Autoridad necesaria

El operador debe tener una sesión administrativa auténtica, de garantía alta,
con certificado o DNIe, y una asignación vigente del RolID
`administracion_perfiles`. Las decisiones V3 han de estar registradas por el
canal central confiable. Cada operación exige una decisión exacta para:

| Operación | Acción | Tipo de recurso |
| --- | --- | --- |
| Preparar | `administracion.perfiles.proponer` | `perfil` |
| Aprobar | `administracion.perfiles.aprobar` | `propuesta_perfil` |
| Aplicar | `administracion.perfiles.otorgar` | `perfil` |
| Recuperar | `administracion.perfiles.recibo.consultar` | `recibo_perfil` |

La finalidad es `gestion_perfiles`; campos y obligaciones deben estar vacíos.
Los ámbitos son exactamente organización y unidad. La referencia es
`cargo_ct:<clave>`. El único atributo es la huella SHA256 del plan. El método
`Plan.Recurso` construye el recurso para el PDP central.

La plantilla Aplicación candidata de AUT24 todavía no está publicada en esta
base de código. AD160 y AD164 no la publican ni se conceden esa autoridad. Para el
recorrido se necesita publicar mediante el circuito central gobernado la
plantilla, la asignación y la sesión aprobadas, y obtener decisiones genuinas.
No basta insertar una fila, escribir un aprobador en JSON o pertenecer al grupo
PostgreSQL. Dirección debe fijar esa dependencia en su orden de integración.

Las plantillas de los cargos también deben estar publicadas de antemano, con
sus versiones y huellas. AD160 no altera sus concesiones. El catálogo RRHH
conserva la correspondencia entre paso y RolID; no se infiere de un nombre de
perfil ni se convierte un perfil opaco en cargo.

## Plan y efecto

El fichero privado contiene `plan` y `operador`. `plan` conserva versión 1,
clave de 32 dígitos hexadecimales, RolID, versión y huellas del rol/control,
huella de la preimagen completa del cargo y ámbito, registro CA auténtico del
destino y su huella, organización, unidad, asignación central canónica en
base64 y caducidad UTC. `operador` contiene decisión/motivo V3 canónicos en
base64 y versiones de persona/perfil. No lleva un aprobador textual.

`destino_sha256` es SHA256 UTF-8 de
`<huella_registro_CA>:<huella_manifiesto_procedencia_CA>`. CA acredita ese
registro contra sus punteros actuales. La preimagen del cargo usa un encuadre de longitudes UTF-8: prefijo
`vec.cargos.preimagen.v1:`, filas ordenadas por perfil, cada una con
`<longitud_perfil>:<perfil><longitud_asignacion>:<asignacion><huella>`;
si no hay filas, `ausente`. Se añaden RolID, organización y unidad, cada valor
precedido de su longitud y `:`. Se aplica SHA256 a esos bytes.
Se comprueba en
SQL; no se acepta como prueba por sí sola.

El circuito previsto prepara el plan y conserva una aprobación central ligada
a sus bytes. AD164 exige que el proponente y el aprobador sean personas
distintas y que ninguna sea la destinataria. Aplicar, recuperar y acreditar el
adjunto conservan las guardas para comprobar el origen almacenado cuando se
autorice una apertura futura. Un origen con una sola
persona se deniega sin reescribirlo. No se exige una tercera persona al aplicar.

La apertura futura debe unir consumo, auditoría común, CAS, asignación y recibo
en una transacción serializable. Las tablas locales `cargo_ct_consumo` y
`cargo_ct_auditoria` de AD160 no sustituyen esa autoridad común. Hasta cerrar
esa dependencia, ninguna fachada SQL queda disponible para el ejecutor.

Este primer corte permite alta o nueva versión para la misma persona, perfil y
asignación. Deniega otro titular activo del mismo cargo y ámbito. Un traslado a
otra persona necesita revocar primero la asignación previa por su circuito
central; el kit no la sobrescribe ni añade una segunda titularidad.

Cuando se habilite el circuito completo, recuperar revalidará una autorización
actual antes de devolver el recibo original.
Mantiene recibo, fecha y versión. Una clave con bytes diferentes se deniega.
Una nueva decisión de acceso añade una traza sin repetir la publicación.

El plan puede incluir `vinculo_certificado_canonico`, una cadena JSON
canónica del descriptor CA. Esa cadena forma parte de la huella aprobada.
Cuando existe, el efecto llama a la fuente CA nominal dentro de la misma
transacción; sólo un retorno `true` permite confirmar. La persona del descriptor
debe coincidir con el destino. El callback acredita además la cuenta y sus
versiones contra el registro CA real. Si falta su migración o su ACL/propietario
no coinciden, la operación se deniega y no publica la asignación.

La proyección central privada es
`acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text) → jsonb`, sólo para el
propietario CA. Devuelve persona, referencia del registro destino y cadena del
adjunto tras revalidar plan, aprobación y puntero de asignación. El callback es
`vec_contexto_actor_v1.registrar_vinculo_certificado_aprobado_ct_v1(bytea,text,text,text,text) → boolean`.
Su aprobación y recibo proceden de la autoridad central. No los acepta del JSON
del operador. La fuente CA24 se entrega en una candidata separada y necesita
su ensayo/revisión antes del recorrido con certificado.

Las tablas nuevas son `cargo_ct_plan`, `cargo_ct_aprobacion`,
`cargo_ct_consumo`, `cargo_ct_auditoria` y `cargo_ct_recibo`, en
`vec_autorizacion`. Conservan historia de solo adición y RLS forzada. El rol
NOLOGIN `vec_autorizacion_cargos_ct_ejecutor` recibía en AD160 acceso a las cuatro
fachadas. AD164 retira ese acceso, también para quienes hereden el grupo. Una
cuenta LOGIN privada no puede obtener permisos adicionales por pertenecer a él.

## Preparación fuera de línea

El comando `recurso` de la CLI existente valida el plan y construye el recurso
para el PDP sin conectar a PostgreSQL ni conceder permisos. Es la capacidad
disponible de este artefacto mientras las fachadas permanecen cerradas.

Compilar con Go 1.26.5. Los ficheros de fuente y conexión deben estar fuera de
Git, sin enlaces y con permisos restringidos. Nunca poner el DSN en argumentos.
El comando `recurso` no conecta a PostgreSQL. Para aprobar y recuperar, obtener
una decisión nueva del PDP con el tipo de recurso indicado arriba y conservar
sin cambios los bytes del plan.

```sh
vec-cargos-ct --operacion recurso --recurso-operacion preparar --fuente <plan-privado>
```

Las operaciones `preparar`, `aprobar`, `aplicar` y `recuperar` con conexión SQL
se deniegan después de AD164. No debe concederse `EXECUTE` para superar el
cierre. Su habilitación exige otra candidata con el contrato común real y sus
revisiones; la CLI no hace reintentos automáticos.

La lista causal de esta candidata es
`deploy/principal/lista_sql_codexe_cargos_fachada_20261003.txt`. Sus objetos
mínimos son AUT7, CA2 e IS3, junto con la autoridad central inicial. No cambiar
el orden por el número de una migración que no aporte objetos utilizados.

La prueba SQL de AD164 comprueba cierre de concesiones directas, rechazo de
los seis cuerpos desde el propietario heredado, ausencia de auditoría local,
independencia y CAS
con registros sintéticos del propietario que se revierten por completo. Esos
registros no acreditan sesiones ni decisiones PDP. La prueba SQL de AD160
corresponde al checkpoint previo a AD164 y no debe repetirse después del cierre.
Quedan por demostrar en PostgreSQL real: alta, CAS negativo, revocación
concurrente, aprobación caducada, fallo de escritura de auditoría, replay y
recuperación tras reinicio. Ninguna prueba Go sustituye esas garantías.

El índice MCP no está expuesto en esta sesión; se consultó su CLI antes de leer
los archivos de las autoridades. Este worktree aún no figura en el índice, por
lo que la inspección final usa sus archivos exactos.
