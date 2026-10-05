# ABI AD193: consumo confirmado v4

Candidata desde `origin/main@296f78373`, unida a `main@337ca757b` y reanclada
sobre AD184, AD185 y AD192 en ese orden. El ensayo estructural de esta revisión
terminó correctamente en una copia privada de POST192. El 5 de octubre se
ensayó con vec-admin real sobre el estado de la principal (sección final); ese
ensayo obligó a adaptar también las fachadas AD184/AD185.
La captura fría `POSTIMAGEN_FINAL_AD192_K.json` (SHA256
`de59d6a47401bc8e0aba65c3ab2034467cf4ab1576123cdd415c8d4d583d3919`)
conserva 6254 registros anteriores y no tiene AD193 instalada. Las preimágenes
esperadas son núcleo POST192, comprobador conservado desde POST173 y CHECK POST192
`0f6d15ebdc61ba5ff67903bde824db878a6396fa593029e98498946e2d8d1331`.
El núcleo capturado tiene `prosrc` SHA256
`b7eb48be035e854928c9685c916f9139166a197e3cae731510b3597f0fe40a45`
y definición SHA256 `536ea653143147e0cfb2d3277948530acb8d4d1643e44f4786462fe5ad1379fe`.
AD193 rechaza la preimagen antigua. Dirección revisa el hash final antes del
ensayo causal.

AD174/176/179/183/186/187/188/189 amplían el CHECK y añaden funciones propias.
AD184 y AD185 amplían el núcleo; AD192 añade otra familia al CHECK. AD193
conserva esas ramas y no reconstruye el comprobador fuera de su delta previsto.

## Esquema y autoridad

Se añaden dos columnas `transaccion_origen xid8`, una en `consumo_decision_v3`
y otra en `auditoria_consumo_v3`. Son internas y admiten `NULL`: sin DEFAULT,
backfill, parámetro de cliente ni NOT NULL global. El núcleo interno toma una
sola vez `pg_catalog.pg_current_xact_id()` y escribe explícitamente ese mismo
valor en ambos INSERT. El sello completo incluye la época; no se convierte a
`xid` ni se usa `xmin` como TopXID. PostgreSQL documenta `xid8` y las funciones
de identificación de transacciones en su
[manual de PostgreSQL 18](https://www.postgresql.org/docs/18/functions-info.html#FUNCTIONS-PG-SNAPSHOT).

La familia nueva es `consumo_confirmado_v4`, `version_consumo=4`. El CHECK
existente se conserva íntegro en la rama histórica con sello NULL. La rama v4
reutiliza el predicado v3 y exige sello positivo. Las familias históricas,
AD169/171 y las familias técnicas K conservan NULL y su formato anterior.
Los registros existentes no se reescriben.

## Preimagen del eslabón

Cada campo usa el encuadre vigente: longitud UTF-8 decimal, dos puntos, valor
UTF-8 y salto de línea. SHA256 se aplica a la concatenación, en este orden:

| Posición | Valor |
| --- | --- |
| 1 | `consumo_confirmado_v4` |
| 2 | `4` |
| 3 | secuencia decimal |
| 4 | anterior_sha256 |
| 5 | decision_ref |
| 6 | efecto_ref |
| 7 | huella_efecto_sha256 |
| 8 | consumo_huella_sha256 |
| 9 | proceso |
| 10 | canal |
| 11 | consumida_en UTC con seis decimales |
| 12 | registrada_en UTC con seis decimales |
| 13 | actor_ref |
| 14 | perfil_activo_ref |
| 15 | finalidad_ref |
| 16 | transaccion_origen decimal completo |

Los dos instantes proceden del mismo `v_ahora`. La proyección final v4 tiene
dos propiedades obligatorias:

| Propiedad | Origen | Tipo |
| --- | --- | --- |
| `transaccion_origen` | `u.transaccion_origen::text`, auditoría | STRING decimal canónica positiva uint64 |
| `transaccion_consumo_origen` | `a.transaccion_origen::text`, consumo | STRING decimal canónica positiva uint64 |

Ambos valores deben ser iguales. Sólo `transaccion_origen` se encuadra en la
posición 16. Se rechazan cero, signos, ceros iniciales, JSON number, NULL,
propiedades ausentes y valores superiores a `18446744073709551615`. Nunca se
pasan por float64 o JavaScript Number. Por ejemplo, `"9007199254740993"` y
`"18446744073709551615"` deben conservarse literalmente. El verificador mixto
y CLI son propiedad de dirección y deben admitir v4 antes de producirla.

`consumo_huella_sha256` conserva su preimagen. El recibo de siete propiedades
conserva nombres, valores y recuperación. AD193 no añade propiedades al recibo
ni al resultado del comprobador.

## Productores e INSERT existentes

El inventario de fuente distingue los cuerpos que escriben, no cuenta cada
migración que reconstruye el mismo cuerpo:

| Cuerpo | INSERT consumo/auditoría | Cambio |
| --- | --- | --- |
| `registrar_y_consumir_decision_v3_atestada`, origen AD002 | dos INSERT propios | sin ampliar; sello NULL |
| `consumir_consulta_rrhh_v3_interna`, origen AD003 | dos INSERT propios | sin ampliar; sello NULL |
| `consumir_decision_mutacion_v3_interna`, origen AD010 y cadena posterior | dos INSERT propios | ambos sellados; v4 |
| `consumir_decision_mutacion_v3_externa_interna`, copia AD116 | dos INSERT propios | sin columnas nuevas; formato anterior |
| `consumir_decision_mutacion_v3_usuarios_externa_interna`, copia AD118 | dos INSERT propios | sin columnas nuevas; formato anterior |

Tres fachadas leen la fila recién escrita por el núcleo y exigían la familia
v3 de forma literal. AD193 las reconstruye con la misma técnica reversible y
huellas medidas antes y después:

| Fachada | Antes | Después |
| --- | --- | --- |
| `registrar_y_consumir_usuarios_admin_v3_atestada` (AD185) | v3 | v4 con ambos sellos iguales al TopXID actual |
| `registrar_y_consumir_denominacion_persona_v3_atestada` (AD184) | v3 | v4 con ambos sellos iguales al TopXID actual |
| `cotejar_consumo_denominacion_persona_v3_atestada` (AD184) | v3 | v3 histórica sin sello, o v4 con los dos sellos iguales |

Ninguna otra función instalada en la principal nombra la familia v3; la prueba
SQL lo comprueba.

Los cuatro INSERT de AD002/AD003 quedan expresamente fuera de la ampliación.
Las fachadas que delegan en el núcleo interno reciben v4; las delegaciones
externas del propio núcleo siguen su formato anterior. El preflight de la copia
final debe contrastar este inventario con `pg_proc.prosrc`: el índice local
consultado no contiene los símbolos SQL actuales.

## Comprobador de firma

Se reconstruye `comprobar_consumo_firma_ct_v1(jsonb)` con la misma ABI y ACL.
En vez de `xmin`, exige que ambos sellos sean iguales al TopXID actual; NULL o
un sello de otra transacción deniegan. Conserva recibo nuevo, referencias,
huellas, fecha, decisión positiva, acción, audiencia, módulo, finalidad,
perfil, superficie, campos, obligaciones y vigencia. No elimina wrappers,
SAVEPOINT ni bloques EXCEPTION instalados. La transformación reversible
verifica definición exacta, `pg_proc` salvo `prosrc`, OID, propietario,
configuración, ACL y dependencias locales/compartidas antes y después.

## Preflight y aceptación pendiente

En la copia fría final, antes de aplicar AD193, se deben recoger:

```sql
SELECT p.oid::regprocedure AS funcion,
 encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') AS def_sha,
 encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') AS src_sha,
 to_jsonb(p)-'prosrc' AS metadatos
FROM pg_proc p
WHERE p.oid IN (
 to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
 to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)'));
SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') AS check_sha
FROM pg_constraint c
WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
```

La migración comprueba propietario, configuración y ACL exactos de ambas
funciones y rechaza bloques ausentes o repetidos. Las huellas se fijan como literales medidos antes de revisar; no se calculan
para aprobar el destino. Las postimágenes completas de definición y cuerpo
del núcleo y comprobador también se comprueban como literales medidos.

La prueba SQL incluida verifica columnas/ACL, huella posterior del núcleo, ramas
AD184/185/192 y un vector de 16 campos con XID
superior al entero seguro de JSON. No fabrica filas favorables. Lo que el ensayo del 05/10 cubrió y lo que no está en la última sección. Quedaban pendientes el ensayo causal con productores VEC reales y la
conformidad de K antes de LISTA.
El ensayo causal de dirección debe demostrar con productores VEC reales:

1. Consumo fresco y comprobador en la misma transacción: ambos sellos iguales,
   recibo de siete campos y eslabón v4 recomputado.
2. El mismo caso dentro de SAVEPOINT y de EXCEPTION, también tras RELEASE:
   acepta con sello TopXID aunque xmin sea SubXID.
3. Un recibo comprometido en otra transacción: deniega aunque se envíe
   consumo_nuevo=true y permanezca vigente.
4. Replay y recuperación: mismo recibo y una sola fila por tabla; el replay
   no se presenta como consumo recién creado.
5. Operaciones AD002/AD003/externas y familias mixtas previas: formato intacto,
   sin exigir sello no nulo ni reescribir huellas.
6. Rol no autorizado y dato de sello enviado como propiedad adicional del
   recibo: denegación. No hay entrada SQL pública para elegir el sello.

El sello permite comprobar la procedencia transaccional dentro de una instalación.
El recorrido causal pendiente debe demostrar su uso con los productores reales.
Restauraciones o importaciones necesitan procedencia explícita; XID no identifica
de forma global una transacción entre instalaciones. No reaplicar UP ni ejecutar
DOWN sobre historia conservada.

## Evidencia anterior y aceptación pendiente

La copia privada PostgreSQL 18 restaurada desde POST173 recibió una sola vez
AD174/176/179/183/186/187/188/189 y después AD193. Dos revisiones SQL/sensibles
independientes aprobaron el SQL exacto. UP y la prueba estructural terminaron
con código 0; los 6240 registros anteriores, la cabeza, OID, propietario, ACL
y configuración de las dos funciones se conservaron. Los sellos históricos
siguen NULL: no hubo backfill. La prueba de 16 encuadres y la serialización
de valores superiores a 2^53 y del máximo uint64 quedaron verdes.

Ese ensayo corresponde al candidato anterior, previo a AD184/185/192; no acredita
esta revisión ni instalación en la principal. La postimagen nueva permite calcular
fuera de SQL las tres sustituciones reversibles del núcleo: `pg_proc.prosrc`
`bb21afce73af87d532574c99da4f3ea8cd0534edc4910f14018d9ee55eb2a79b`
y `pg_get_functiondef` `f581dbf9aa01d454caa6906ece774f97cf16cca9e9b8910d0eb348e23ef8c34b`.
Las preimágenes del comprobador coinciden con las del candidato anterior. Esta medida aún no acredita consumo
fresco CT175/AUT41, SAVEPOINT o
EXCEPTION dentro de ese recorrido, replay o rechazo entre transacciones con
productores VEC reales. Falta además la conformidad de K antes de LISTA. La
migración no se ha instalado en la principal.

## Ensayo estructural POST192 del 04/10

La candidata `5d7e2c3d65c11a000c0ba6c1807bf09ce985866f` recibió dos revisiones
independientes de preparación, una SQL y otra sensible. El SQL con SHA256
`f226520ce551c6fa53c43cf9a00b4806d36c6f51adfb6ba9a7a808f2c3a0c51b` y la
prueba `2f3d4e8052bdef5825b99518c05bda78223c88660159e173735161c367da5b8f`
se ejecutaron en una copia del frío final de K, PostgreSQL 18, sin red ni acceso
a la principal. AD192 ya estaba instalada: no se reaplicó. Un único UP de AD193
y la prueba estructural terminaron con código 0.

Las postimágenes del núcleo medidas en PostgreSQL coinciden con las calculadas
arriba. La postimagen del comprobador es definición SHA256
`fe5e93bbece6225e72155029211e77b3d08518a3ac6ef8b350f994658cbcb3fe` y cuerpo
`3984c45db3ceb2ebbeead7246336294a8a2ffbb2d99f66cf6f6876fc07ab6d1f`.
Se conservaron los 6254 registros de auditoría, los 6240 consumos, la secuencia
y cabeza, OID, metadatos y ACL. Los datos se compararon excluyendo las columnas
nuevas; todos los sellos históricos siguen NULL. Las demás funciones, roles,
políticas, ACL de tablas, catálogo de audiencias y CHECK de outbox quedaron
idénticos. Las pruebas comprobaron las ramas AD184/185/192, el encuadre de 16
campos y la serialización de los valores superiores a 2^53 y del máximo uint64.

El ensayo estructural no acredita consumo nominal CT175/AUT41, SAVEPOINT,
EXCEPTION, rechazo entre transacciones o replay con productores reales. La PR
queda en borrador hasta ese recorrido y la conformidad de K; no se declara LISTA.

## Ensayo causal con vec-admin real del 05/10

Se partió de la copia fría H10-30 de K (SHA256
`ea3d00de2d52a941855dc7569c6e4ac52582a531e4118796ee160639a0d71fcd`), con
AD194, IS16, CA36 y AUT47 aplicadas una vez, como está la principal. Encima se
hizo el arranque 2+1 con el guion de K, el mantenimiento a v5, la raíz de
laboratorio y el gobierno de usuarios del día, y se arrancó vec-admin con
binarios de esta rama. PostgreSQL 18.4 en contenedor de 2 GB, datos en disco,
sin red exterior ni acceso a la principal.

AD193 aplica sobre AD194 sin cambios: AD194 sólo toca las dos funciones de
repetición de AD192 y las preimágenes medidas coinciden con los literales.
El número 194 es mayor, pero el orden de instalación no importa entre ellas.

Primer ensayo, con la AD193 anterior (`f226520c…`): la lista de usuarios daba
200 antes de AD193 y 403 justo después. PostgreSQL registraba
`AD185: acuse_o_autoridad_rechazados`, porque la fachada AD185 exigía
`consumo_confirmado_v3`. Las dos fachadas de denominación de AD184 tenían el
mismo literal. Por eso AD193 incluye ahora el bloque de fachadas.

Segundo ensayo, desde cero con la AD193 final (`bddba35f…`, prueba
`e5eb87f3…`), en caliente con vec-admin en marcha:

1. Antes de AD193, dos lecturas reales dejaron dos filas v3 sin sello.
2. UP y prueba con código 0. Una segunda ejecución para en
   `columna_transaccion_origen… actual=presente` sin efectos.
3. Después, el mismo proceso vec-admin siguió listando usuarios (200 para las
   dos personas). Cada lectura escribió una fila v4 con los dos sellos
   iguales; el `xmin` de las filas es un SubXID distinto del sello, porque el
   núcleo inserta dentro de su bloque EXCEPTION. La fachada AD185 aceptó el
   sello del TopXID en ese contexto.
4. `vec-auditoria-verificar` de esta rama verificó eslabones reales v1, v3 y
   v4 uno a uno. Rechazó el v4 con el sello cambiado, como número JSON, a cero
   o sin `transaccion_consumo_origen`. No hay eslabones rotos en toda la cadena
   y la cabeza coincide con la última huella.
5. El comprobador AD167 denegó (42501) un recibo v4 real confirmado en otra
   transacción aunque llevara `consumo_nuevo=true`, también dentro de
   SAVEPOINT y de un bloque EXCEPTION, y con una octava propiedad
   `transaccion_origen`. El rol lector de vec-admin no puede ni ver el esquema;
   `vec_autorizacion_propietario` no puede llamar al núcleo. Ninguna función VEC
   recibe el sello como parámetro.
6. Las repeticiones con acuse nuevo de fuentes, unidad, bootstrap, mantenimiento
   y gobierno de usuarios devolvieron el mismo recibo, sin filas nuevas en sus
   familias, que siguen sin sello.
7. Tras reiniciar PostgreSQL y vec-admin, la selección de perfil se conservó,
   las lecturas siguieron en 200 con filas v4 selladas y las repeticiones de
   las familias técnicas dieron el mismo recibo. Los 6240 registros previos
   siguen sin sello.

Límites. El comprobador AD167 sólo admite consumos de firma CT, y los
productores reales de esa firma (AD177, AUT41, CT175 y CT176, de Codex-E)
no están en main. Por eso el caso positivo del comprobador (consumo fresco de
firma, dentro de SAVEPOINT o EXCEPTION, aceptado) no se ha podido recorrer con
productores reales; se ha comprobado el mismo sello en la fachada AD185, que sí
es un productor real. La repetición de una lectura ADMIN no es posible por
diseño: la fachada exige una decisión nueva y el núcleo sólo se llama desde
las fachadas. Tampoco se ha probado la denominación de AD184, que este
arranque no usa.
