# AD178: recuperación R5 y gobierno del plan nominal

AD178 añade al núcleo dos perfiles técnicos fijos: la recuperación R5 de 48
campos y el gobierno del plan nominal de firma CT. No crea perfiles humanos,
LOGIN, grupos técnicos, membresías ni concesiones nominales, y no define el
tercer lector del plan. El borrador original es de Codex-L
(`trabajo/codexl-perfiles-plan-firma-20261004`, `2c699a007`); el 5 de octubre
se reancló sobre main posterior a AD195/AD196 dentro de la cadena de firmas.

## Contrato preparado

El clasificador fijo es `recuperacion_firmas_r5_ct_v2`. Usa el mismo runtime
técnico CT que `consulta_firmas_r5_ct_v2`: la comprobación existente de
`vec_contratacion_temporal_ejecutor` permanece intacta. La clasificación técnica
no concede a una persona permiso para recuperar un canon. El núcleo conserva su
cotejo de identidad, contexto, decisión nominal y autorización viva.

| Coordenada | Valor exacto |
| --- | --- |
| Acción | `contratacion_temporal.documento.firmas_r5_v2.recuperar` |
| Audiencia | `vec_contratacion_temporal.firmas_r5.recuperar.v2` |
| Módulo | `contratacion_temporal` |
| Tipo de recurso | `expediente_contratacion_temporal` |
| Finalidad | `gestionar_contratacion_temporal` |
| Superficie | `interna_corporativa` |
| Recurso | `ExpedienteRef`, igual a `efecto_ref` |
| Huella del efecto | Huella del contexto autorizado: ámbito `organizacion_ref` y atributo `material_sha256` |
| Obligaciones | Array vacío |

Los campos son exactamente `CamposRecuperacionFirmasV2()` de E, contrato
`d9dd19fd3`: los 44 de consulta R5 más `CanonNominal`, `CanonNominalRef`,
`CanonNominalSHA256` y `MaterialRootSHA256`, ordenados. La lista completa está
en las tres guardas SQL (núcleo, fachada y comprobador). La consulta de 44 campos conserva su acción, audiencia y cuerpo.

La nueva fachada es:

```text
vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(
  p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
  p_persona_version numeric, p_perfil_version numeric,
  p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
)
RETURNS TABLE (
  decision_ref text, efecto_ref text, huella_efecto_sha256 text,
  consumo_huella_sha256 text, auditoria_ref text,
  consumida_en timestamptz, consumo_nuevo boolean
)
```

El núcleo recibe once argumentos al anteponer el clasificador fijo. La fachada
exige una autorización nueva en una transacción SERIALIZABLE de escritura.
Un consumo anterior de la misma decisión se rechaza; una lectura repetida requiere
otra decisión nominal. Sólo el propietario AUT y el propietario técnico CT reciben
EXECUTE sobre la fachada. No se concede acceso directo al núcleo ni SELECT a sus
tablas. La ACL final se comprueba dentro de la migración.

La fachada consume y audita; no devuelve el canon histórico. K debe aportar la
lectura histórica autorizada que conserve los bytes originales. Recuperar una
firma anterior no exige ni inventa un cargo actual de su firmante. E conserva el
cotejo del canon recuperado contra la firma/PDF registrados y la raíz original.

## Comprobador privado para AUT41

El contrato de E añade
`comprobar_consumo_recuperacion_firmas_ct_v2(material_consulta bytea, consumo jsonb)`.
Devuelve únicamente organización, expediente, tipo documental, versión de
expediente, decisión, huella del consumo, auditoría y vencimiento de la decisión.
AUT recibe EXECUTE de la función y no recibe lectura de las tablas AD. CT liga
por sus propias filas el tipo documental con la referencia del PDF histórico.

El material conserva las diez claves de `MaterialConsultaFirmasR5V2.Canonico()`.
La unidad es `null`; documento y versión quedan ligados dentro de sus bytes.
No se añade un ámbito de unidad a la consulta. El SHA256 de los bytes originales
se incluye en el contexto canónico de organización; la huella del efecto es la
de ese contexto, como en CT172, y no el SHA256 del material aislado.

La función relee y bloquea consumo, auditoría y atestación. Comprueba los siete
valores del recibo contra las filas, las huellas de decisión y capacidad,
acción/audiencia de recuperación, los 48 campos exactos y obligaciones vacías.
Desde AD193 no usa `xmin`: exige que el sello interno `transaccion_origen` (xid8)
de consumo y de auditoría sea igual a `pg_current_xact_id()` y que la auditoría
sea de la familia `consumo_confirmado_v4`, versión 4. Un recibo de otra
transacción, un sello NULL (historia anterior a AD193) o un sello enviado por el
cliente deniegan. El sello es el TopXID, así que un bloque `EXCEPTION` o un
`SAVEPOINT` del llamador ya no cambia el resultado, como pasaba con `xmin`.
El vencimiento se comprueba después de los bloqueos y antes del retorno.

## Delta e instalación

La migración añade una condición de recuperación y otra de gobierno justo antes
del bloque de consulta R5 V2 (ancla única), y amplía el CHECK de audiencias de
`clave_capacidad_version` con el mismo patrón que AD185: envuelve el CHECK
vigente y añade `OR audiencia_consumo IN (recuperación, gobierno)`. No cambia la
clasificación runtime (los dos perfiles caen en el runtime CT existente), firma
V2, cálculo del consumo, replay, origen AD172, eslabón AD173 ni filas históricas.

Huellas medidas el 5 de octubre de 2026 en el clon de la principal
(copia fría H10-30 + AD194, IS16, CA36, AUT47, AD193, AD195, Personal36, AD196,
AUT49 y AUT48, en ese orden; es lo que tendrá la principal al aplicar las listas
de main):

| Huella | Antes | Después |
| --- | --- | --- |
| `pg_get_functiondef` del núcleo | `728dde660bd784951e6685402a625f62dedc6d08cff35d3e259a1d9af471737a` | `2ccd704afe6140d604faa626631e9743edda8f785517d1136054c746cb9b1381` |
| `prosrc` del núcleo | `717eba51bc117748907f46dbf9ad1341b53a1aeb1896745b9561eebc3f6d189c` | `4729b6666065a8a3582443d803bf8535940f0eae650b27a315aca260f3183b8b` |
| CHECK de audiencias (`pg_get_constraintdef(oid,false)`) | `26497f113bb8468042bffa3fffaf846ce9da5d6db289f0d3b48a0f011703d5f0` | `e76428d2ecd1c79da88a827cf33138f5c75026ed2138858c67f93420e932eae7` |

Las preimágenes son las del núcleo y el CHECK que dejan AD195 (consumidor de
servicios certificados de Personal) y AD196 en main. Cualquier otra migración
que reescriba el núcleo antes que AD178 obliga a remedir. La
migración compara las seis huellas, propietario, configuración, ACL y
dependencias del núcleo; revierte el delta en memoria para demostrar que el resto
del cuerpo no cambia; y aborta con SQLSTATE `55000` ante cualquier diferencia o
si ya está instalada. Se aplica una vez, sin DOWN, con
`deploy/principal/lista_sql_claude_firmas_ad178_ad177_20261005.txt`.


## Gobierno preparado y lectura pendiente

El clasificador fijo `gobierno_plan_nominal_firma_ct` exige las cuatro acciones
`vec.catalogos.crear/actualizar/publicar/retirar`, audiencia
`vec_catalogos_configurables.plan_nominal_firma.gobierno.v1`, módulo CT, recurso
`catalogo_configurable`, finalidad `gestionar_contratacion_temporal`, campos y
obligaciones vacíos. Usa el grupo técnico existente
`vec_contratacion_temporal_ejecutor` mediante la clasificación CT original; la
migración no cambia su membresía ni publica EXECUTE del núcleo a ese grupo.

La superficie de gobierno es `administracion_privilegiada`, con
`cuenta_privilegiada=true`; recuperación sigue en `interna_corporativa`. El actor
requiere la categoría fija Aplicación acreditada por K: versión de rol y su SHA,
`categoria_administrativa=aplicacion`, `tipo_perfil=fijo_sistema`. Esa metadata
pertenece a AUT33, no a una regla por nombre de cargo. L no lee ni concede acceso
a sus tablas. K mantiene la fuente de autenticación y revalidación de sesión,
certificado/DNIe, garantía y concesiones positivas específicas.

El recurso de gobierno es `catalogo_id:version`. El ID cumple
`^[a-z][a-z0-9._-]{2,127}$`; la versión es decimal positiva. El predicado no acepta
comodines ni referencias de recurso libres. La fachada exterior AD177, propia
de E, verifica los límites de versión, material, actor y pin/CAS; recibe EXECUTE
únicamente el grupo runtime CT (AD177 retira el EXECUTE que el borrador daba al propietario de catálogos). Esta pieza no modifica AD177
ni añade una fachada de gobierno alternativa.

La huella de contexto es SHA256 UTF-8 de esta representación canónica:

```text
{"ambitos":{},"atributos":{"estado":"ESTADO","material_sha256":"SHA256_BYTES_EXACTOS_MATERIAL","revision":"REVISION"}}
```

Los valores proceden del material y catálogo validados por AD177/CC7. En
particular, `material_sha256` se calcula sobre los bytes exactos del fichero de
material, no sobre una reserialización JSON. El contexto SHA se liga como
`c.huella_efecto_sha256 = d.contexto_recurso_huella_sha256` en el núcleo.
L no recibe ese fichero ni recomputa sus tres atributos: la evidencia de esa
raíz y del CAS pertenece a AD177/CC7. La huella del contexto tampoco sustituye a
`MaterialRootSHA256` del contrato de recuperación R5.

`consulta_plan_nominal_firma_ct` sigue pendiente de acción, audiencia, finalidad,
recurso y campos exactos de E. Tampoco tiene una rama aceptada. La revalidación
privada del pin en COMMIT CT de CC7 consume la firma vigente; no convierte al
firmante en administrador ni acredita una lectura administrativa nueva.

El orden causal es AD193 → AD195/AD196 → AD178 → AD177 → CC7. AD177 exige la postimagen de
AD178, pese a tener un número menor. AD178 prepara los dos clasificadores, pero
no habilita el circuito por sí solo: falta la fuente nominal de K (incluida la
extensión de `acreditar_perfil_aplicacion_nominal_v1` a las acciones
`vec.catalogos.*`), CC7 y los consumidores AUT41/CT175/CT176.

## Ensayo y límites

En el clon descrito arriba se aplicaron AD178 y AD177 una sola vez, en ese
orden, con código 0. Una segunda aplicación de cada una se detiene en su
precondición sin cambios. Pasan las tres pruebas:

- `pruebas_sql/ad178_recuperacion_contrato.sql`: ABI/ACL de la fachada y cinco
  rechazos (cada campo nuevo ausente y sólo 44 campos).
- `pruebas_sql/ad178_gobierno_predicado.sql`: el predicado de gobierno aparece
  una vez en el núcleo y rechaza superficie interna, acción ajena, audiencia
  distinta, recurso con comodín y campos de otra capacidad.
- `pruebas_sql/ad177_ad178_post_ad193.sql`: huellas finales del núcleo y del
  CHECK, comprobadores con sello v4 y sin `xmin`, ningún LOGIN con EXECUTE
  directo fuera de los grupos propietarios, runtime CT sólo en la fachada de
  gobierno, sin sello en filas históricas, y rechazo de un recibo real de otra
  transacción en los dos comprobadores nuevos.

La prueba de AD193 (`pruebas/000193_transaccion_origen_consumo_v4.sql`) fija la
postimagen del núcleo de AD193 y deja de cumplirse después de AD178 por diseño.

- `pruebas_sql/ad177_ad178_positivo_sintetico_clon.sql` (sólo clon desechable,
  como superusuario): dentro de una transacción que termina en ROLLBACK retira
  CHECK y disparadores de las tres tablas, siembra consumo, auditoría v4 y
  atestación sintéticos sellados con la transacción actual y comprueba que el
  comprobador de recuperación acepta y devuelve sus ocho claves, que un
  `decision_ref` de 600 caracteres se rechaza por forma, que un sello NULL se
  rechaza en la relectura de filas, y que el comprobador de gobierno pasa sus
  comprobaciones y se detiene en la categoría Aplicación de AUT. Esta prueba
  habría detectado el fallo de la expresión regular `{2,511}` (PostgreSQL limita
  las repeticiones a 255), que convertía todo en denegación.

Fuera de esa prueba sintética no hay consumo positivo. Falta el recorrido
causal con productores reales (CT175 → AD178 → núcleo → AUT41 → comprobador, y
gobierno AD177 → CC7), que depende de la fuente nominal de K y de CC7, AUT41,
CT175 y CT176. Nada de esto se ha instalado en la principal ni en cidonia.
