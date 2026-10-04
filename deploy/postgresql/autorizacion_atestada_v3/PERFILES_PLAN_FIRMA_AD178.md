# AD178: recuperación R5 y gobierno del plan nominal

AD178 está reservada y sigue siendo un borrador bloqueado. Prepara la recuperación
R5 de 48 campos y el gobierno del plan con el contrato cerrado por E a las 01:51
del 4 de octubre. No define el tercer lector ni crea perfiles humanos, LOGIN,
grupos técnicos, membresías o concesiones nominales.

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
en `pruebas_sql/ad178_referencia_post173.json` y en las dos guardas SQL del
delta/fachada. La consulta de 44 campos conserva su acción, audiencia y cuerpo.

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
Consumo y auditoría deben pertenecer al `xmin` de la transacción actual, según
AD167. La atestación se bloquea y coteja sin añadir un requisito nuevo de `xmin`.
El vencimiento se comprueba después de los bloqueos y antes del retorno.

Esta ampliación sigue en preparación. No ejecuta una captura ni una lectura
histórica por sí sola. Las seis guardas de instalación conservan sus `NULL` y
siguen abortando antes del DDL. El contrato AUT41 se ha leído en `f2d6f5010` y
el material Go en `0fa5bea482`; faltan la unión causal, el ensayo nominal y las
revisiones del SQL final. No se ha instalado ni ejecutado este comprobador.

## Delta y cuarentena de instalación

La migración añade una sola condición de recuperación junto al bloque de consulta
R5 V2, una condición de gobierno del plan y dos audiencias al CHECK de
`clave_capacidad_version`. No altera la
clasificación runtime, firma V2, cálculo del consumo, replay del núcleo,
configuración de origen AD172, eslabón nominal AD173 ni filas históricas.

Las seis huellas finales de definición/fuente/CHECK, antes y después del delta,
siguen a NULL. La primera guarda aborta con SQLSTATE `55000`:

```text
AD178: PARO clave=pre_post_aprobadas actual=NULL
esperado=POST175_176_medida_y_POST178_revisada
```

El PARO ocurre antes del DDL y las concesiones. Después de medir la base final,
la migración compara ambas preimágenes del núcleo, propietario, configuración,
ACL y dependencias, exige una única ancla, coteja las postimágenes y revierte el
delta en memoria para demostrar la conservación del resto del cuerpo. El CHECK
de audiencias también requiere pre/postimagen exacta; no se acepta por nombre.

La referencia offline procede de la captura real POST173 conservada por dirección:
definición `6c22fdbb165a00c4f37cb2f7dbb7add4e939e5b0134c0c599b9519bfe3b86db9`,
fuente `bbb932ef29375e88645fb524e6aae5fd3059d51cb0dfd9d4952ffd509470534c`.
Una inserción y su reversión exacta dejan los 110 clasificadores anteriores y
añaden únicamente recuperación y gobierno: 112 en total. La referencia
reconstruida produce
definición `7456c1d7ee9242c4739ccc740aaf67d03d22467070290d4ca6f6b73381d19e15`
y fuente `9246accc3493d0b13c185165db8c600241c558029d569a5af8bf35ec9fbbe1e2`.
Esas huellas son referencia de texto, no pre/postimágenes de instalación sobre
las futuras AD175/176. No se han trasladado a las guardas de la migración.

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
únicamente el ejecutor CT según el contrato de E. Esta pieza no modifica AD177
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

El orden causal es AD173 → AD174/175/176 reancladas → delta L → AD177 → CC7.
AD177 exige la postimagen del delta L, pese a tener un número menor. El borrador
actual de AD178 prepara los dos clasificadores, pero no habilita el circuito
por sí solo: permanecen bloqueadas sus guardas, las dependencias E/K y el ensayo. La lista causal entregada advierte de esta cuarentena.

## Aceptación preparada y límites

`pruebas_sql/ad178_recuperacion_contrato.sql` prepara la comprobación de ABI/ACL y
cinco rechazos: ausencia de cada campo nuevo y uso de sólo 44 campos. Usa entradas
sintéticas negativas; no fabrica un consumo V3 válido ni escribe historia. Se
ejecutará únicamente en el clon después de completar y revisar AD178.

`pruebas_sql/ad178_gobierno_predicado.sql` prepara cinco rechazos del predicado
real instalado: superficie interna, acción ajena, audiencia distinta, recurso con
comodín y campos de otra capacidad. Comprueba primero su coincidencia exacta y
única con la fuente del núcleo. Evalúa entradas sintéticas, sin consumir V3 ni
acreditar actor, grupo, material exacto o CAS. No se ha ejecutado en PostgreSQL.

Antes de integración faltan la base causal final y sus seis huellas, el contrato
de lectura, las concesiones categóricas de gobierno y fachada histórica nominal
de K, la fachada AD177 actualizada de E, el fixture firmado vigente, ensayo
PostgreSQL y dos revisiones independientes del hash final.
El ensayo debe conservar historia/ACL/firma V2/consulta44, comprobar denegación
ante cualquier coordenada distinta, el asiento común y rollback sin efectos.

No se ejecutaron Go, PostgreSQL, SQL en principal/cidonia, DOWN ni reaplicaciones.
La comprobación hecha hasta ahora es la reconstrucción offline de texto sobre
POST173; no acredita instalación, consumo firmado ni recuperación histórica.
Semgrep local comprobó las tres SQL propias con tres reglas: cero hallazgos.
Las dos listas de 48 campos coinciden con el contrato Go de E y
`git diff --check` está limpio.
