# CT-O2-R3B — candidatura técnica durable y confirmación PostgreSQL

Fecha del corte: 31 de agosto de 2026.

## Resultados cronológicos de las puertas dinámicas

Estado: **NO-GO dinámico y estático; R2 obtuvo `GO ESTÁTICO` independiente con
`P0=0`, `P1=0` y `P2=0`, pero las dos ejecuciones focales posteriores
terminaron en el oráculo del preflight y en la primera resolución Go,
respectivamente. R4 recibió un quinto `NO-GO` estático, con `P0=0`, `P1=2` y
`P2=0`. R5 recibió después un `NO-GO` estático independiente porque sus dos
regresiones multidimensionales no aislaban la dimensión como única causa de
rechazo. La ejecución dinámica de R6 falló por una ambigüedad PL/pgSQL y la
auditoría identificó una segunda colisión del mismo parámetro de salida. R7B
corrigió ambas ambigüedades, pero la sexta ejecución dinámica terminó en el
escenario de rotación porque la propia prueba rotaba antes de estabilizar la
raíz bajo `[2,1]`. R8 corrigió solo ese orden no vacuo; su ejecución dinámica
posterior alcanzó la confirmación pública y falló al exigir igualdad entre la
capacidad breve AD-3 y la concesión V3 más amplia. R9 corrige únicamente esa
relación temporal por contención y queda pendiente de revisión independiente;
no acredita PostgreSQL ni autoriza otra ejecución**.

### Primera ejecución pre-final: NO-GO de bootstrap

La ejecución pre-final autorizada del runner terminó con código `3` antes de
instalar migraciones. El bootstrap
`deploy/postgresql/contexto_actor_v1/roles_up.sql` rechazó en su línea 41 la
base desechable porque conservaba privilegios iniciales de `PUBLIC`:

```text
ERROR: la base dedicada debe llegar sin privilegios de PUBLIC
```

En esa ejecución no llegaron a ejecutarse `000046`, `000047`, las pruebas Go
contra PostgreSQL ni ninguna confirmación V2. La limpieza automática retiró el
contenedor, red, volumen y directorio temporal etiquetados
`ct-o2-r3b-pg-20260831`. La comprobación posterior obtuvo cero identificadores
para esos cuatro tipos de residuo. El contenedor ajeno quedó antes y después en
el mismo estado verificable:

```text
d6217278d8718a4e51c4be9523dde15406559989abe46f3a5e496624d6aa4aeb|running|/contagrx-t224-pg-focal|sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382
```

El candidato preservado incorporó después esa barrera en el runner: revoca los
privilegios predeterminados de `PUBLIC` sobre la base y el esquema `public`
antes de instalar la autoridad de contexto. El fallo pre-final permanece como
`NO-GO`; la corrección posterior no lo reinterpreta ni lo convierte en una
ejecución superada.

### Repetición excepcional final sobre `a0cedc2`: NO-GO de `000047`

Dirección autorizó una única repetición excepcional sobre el hash exacto
`a0cedc2e8fce0ab36f0d4fb4ff30dc43e6345596`. Se ejecutó exactamente una vez el
31 de agosto de 2026, desde `19:39:02Z` hasta `19:39:12Z`, duró 10 segundos,
terminó con `RC=3` y produjo un log con SHA-256
`f988ce0cde679aae1ea39102b346d5210a3c450b9659fc0c1a6ee7de1e678193`.

La repetición alcanzó las migraciones CT `000001` a `000005`, `000046` y
VEC-AD-3. Falló en
`000047_componentes/010_candidatura_y_aliases.sql`, línea 73, al resolver
`pg_catalog.greatest(timestamp with time zone, timestamp with time zone)`. No
alcanzó el resto de `000047`, las pruebas Go contra PostgreSQL ni ninguna
confirmación. Terminó con cero residuos propios y el cerrojo liberado; el
contenedor ajeno `d6217278...` permaneció intacto y en estado `running`.

Este segundo resultado también es `NO-GO`. No oculta ni sustituye el fallo
pre-final por `PUBLIC`, no acredita PostgreSQL y no autoriza otra ejecución.
R1 retira exclusivamente la cualificación inválida de la construcción especial
`GREATEST`, conservando la semántica de máximo temporal. Queda pendiente de una
nueva revisión independiente y de autorización dinámica expresa; no se declara
`GO`.

### Revisión estática de `a0cedc2`: NO-GO, `P0=0`, `P1=2`, `P2=0`

La revisión estática independiente del candidato `a0cedc2` emitió `NO-GO` con
dos hallazgos P1 independientes:

1. los triggers de `candidatura_alta_tecnica` y `candidatura_alta_alias`
   rechazaban `UPDATE` y `DELETE`, pero no `TRUNCATE` de propietario; y
2. el runner solo seleccionaba el resolutor Go y después invocaba V2
   directamente, sin construir `NuevaTransaccionAltasPostgreSQLCandidata` ni
   atravesar `ConfirmarAltaCandidata`, por lo que no acreditaba las doce
   entradas ni el escaneo real de las ocho columnas.

R1, commit `df4b0bddacb755c50a1e16b2dac936f2f46affa1`, se mantuvo estrecho:
solo corrigió `pg_catalog.greatest` a `GREATEST` y conservó íntegros los dos
`NO-GO` dinámicos anteriores. R2 añade la defensa canónica `BEFORE TRUNCATE`
por sentencia, regresión de conservación exacta de filas y el recorrido del
adaptador público con proveedor neutral, éxito y replay desde dos pools. R2 no
había ejecutado PostgreSQL ni el runner y permanecía pendiente de revisión
estática independiente y, solo después, de una autorización dinámica nueva. En
ese corte no se declaraba `GO`.

### Revisión estática de R2: GO ESTÁTICO, `P0=0`, `P1=0`, `P2=0`

La revisión independiente del hash exacto
`ba6f0d18ee0d344345af45d5ecfe51f92ec3fa25` emitió `GO ESTÁTICO`, con
`P0=0`, `P1=0` y `P2=0`. Este resultado cerró la revisión del código de R2,
pero no acreditó PostgreSQL ni sustituyó la puerta dinámica obligatoria.

### Tercer NO-GO dinámico focal: oráculo booleano del preflight

La ejecución focal posterior se realizó el `31-08-2026` bajo el
cerrojo exclusivo, con `VEC_CT_O2_R3B_BD_DESECHABLE=SI`, la imagen local
fijada por digest y `--pull=never`. El relanzamiento marcado terminó con
`RC=1` después de instalar PostgreSQL, contexto de actor, autorización V3 y
las migraciones CT `000001` a `000005`, `000046`, VEC-AD-3 y `000047`. El
preflight devolvió:

```text
180004|t|t|t
```

El valor acredita PostgreSQL `18.4` y la presencia de los tres objetos
consultados. PostgreSQL serializa los booleanos como `t`/`f`, mientras el
oráculo del runner comparaba el resultado con `180004|true|true|true`. El
único defecto demostrado es ese literal esperado de shell; no se cambia el
`SELECT` ni se relajan versión, cardinalidad o presencia de objetos.

La ejecución no alcanzó las pruebas Go ni la confirmación. Terminó con cero
contenedores, redes, volúmenes o temporales propios, el cerrojo liberado y la
fuente limpia. El contenedor ajeno permaneció idéntico antes y después:

```text
d6217278d8718a4e51c4be9523dde15406559989abe46f3a5e496624d6aa4aeb|running|/contagrx-t224-pg-focal|sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382
```

La captura completa del panel tiene SHA-256
`f2efd05e362cee4e66c1bc9fc6dc96c230bb9b1cca1ecfb2903f7e21fa7ea171`.
Esa captura incluye también el rechazo inicial con `RC=64` por ausencia de la
marca desechable antes del relanzamiento. El resultado permanece en `NO-GO`
dinámico y no autoriza otra ejecución.

### Cuarto NO-GO dinámico focal: expansión matricial cualificada

La ejecución focal preservada posterior a R3 superó el preflight exacto
`180004|t|t|t` y comenzó
`TestCandidaturaAltaPostgreSQL18DeExtremoATerminal`. Falló en
`confirmacion_alta_postgresql18_test.go`, línea 57, durante la primera llamada
al resolutor, con el error Go normalizado:

```text
contratacion temporal: persistencia no disponible
```

El log `/tmp/vec-o2-r3b-r3-dynamic-20260831.log` conserva 10 líneas y 816
bytes; su SHA-256 es
`9fb15e6e88ce974362b50a0f343bb86e8ef97ea9aec814ed7b4872300c10167c`.
La normalización del adaptador impidió conservar en ese log el `PgError` crudo:
no se afirma que se capturase. La causa se identificó después por inspección
estática de `000047_componentes/020_resolucion_candidatura.sql` y del
tratamiento del parser oficial de PostgreSQL: la transformación especial de
`unnest` con varios arrays solo se aplica al nombre no cualificado; una llamada
`pg_catalog.unnest` con dos o tres argumentos intenta resolver una función
multiargumento inexistente.

La limpieza dejó cero recursos propios y el contenedor ajeno permaneció
idéntico antes y después:

```text
d6217278d8718a4e51c4be9523dde15406559989abe46f3a5e496624d6aa4aeb|running|/contagrx-t224-pg-focal|sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382
```

La ejecución no alcanzó la concurrencia, rotación, confirmación V2, ACL ni
reinstalación. Este cuarto resultado conserva el estado `NO-GO`, no acredita
PostgreSQL completo y no autoriza una repetición adicional.

### Corrección R4

R4 conserva `search_path = pg_catalog` y sustituye cada expansión multi-array
por `ROWS FROM` con llamadas individuales y explícitamente cualificadas a
`pg_catalog.unnest(...)`. La expansión de dos arrays conserva
`WITH ORDINALITY`, los nombres `ambito`, `huella`, `orden` y su validación; las
dos expansiones de tres arrays conservan `generacion`, `ambito`, `huella`, su
orden posicional y las comprobaciones de conflicto e inserción.

R4 corrige también los dos oráculos ACL posteriores para esperar la salida
real de `psql`: `t|t|f` tanto en la prueba Go como tras reinstalar `000047` en
el runner. El preflight permanece deliberadamente en `180004|t|t|t`; no se
relajan su versión ni sus tres comprobaciones de presencia.

## Capability, invariantes y write-set de R4

Capability: recuperar la expansión posicional de los pares HMAC de la
candidatura técnica y conservar oráculos fieles al formato booleano de
PostgreSQL, sin ampliar la autoridad del resolutor ni de la confirmación.

Invariantes:

- la candidatura continúa siendo técnica, durable y no autoritativa;
- ámbito, huella y generación permanecen alineados por posición, con la
  ordinalidad de la validación de dos arrays intacta;
- `search_path = pg_catalog` y la cualificación explícita de cada función se
  mantienen, sin caída a resolución por esquemas del llamador;
- el runtime solo conserva `EXECUTE` sobre el resolutor y V2, nunca sobre V1;
- no se ejecuta PostgreSQL ni se reinterpreta ningún `NO-GO` histórico; y
- R4 no cierra O2-06, R3B, la vertical, el arranque ni producción.

Write-set exacto:

1. `deploy/postgresql/contratacion_temporal/migraciones/000047_componentes/020_resolucion_candidatura.sql`;
2. `deploy/postgresql/contratacion_temporal/probar_o2_r3b_candidatura_postgresql18_4.sh`;
3. `internal/modules/contrataciontemporal/adapters/postgres/confirmacion_alta_postgresql18_test.go`; y
4. `docs/portal_vec/ct_o2_r3b_candidatura_postgresql_2026-08-31.md`.

### Quinto NO-GO estático de R4: `P0=0`, `P1=2`, `P2=0`

La revisión independiente del hash exacto
`8e771a9031ed4715d1686315c957e2f8261e4924` emitió `NO-GO` estático por dos
hallazgos P1:

1. las siete entradas de texto podían ser `NULL` y eludir tanto las
   expresiones `!~` de entrada como las comparaciones `<>` del replay, porque
   ambas producían `NULL` en vez de rechazo; y
2. el resolutor no exigía matrices unidimensionales con límite inferior uno,
   por lo que una matriz con límite cero conservaba su orden en `unnest` pero
   desplazaba el par raíz leído mediante `[1]`.

El dictamen no autorizó PostgreSQL, Docker ni otra ejecución dinámica. Sus
puertas offline fueron verdes, pero no convierten R4 ni R3B en capacidad
cerrada.

### Corrección R5

R5 separa la validación del resolutor en dos etapas. La primera, segura para
cualquier entrada, exige que las dos matrices existan, sean exactamente
unidimensionales, comiencen en uno, tengan la misma cardinalidad entre uno y
cuatro, y que los siete textos y el instante existan y respeten su canon. Solo
después se inspeccionan elementos nulos y pares HMAC. Toda nulidad o forma
matricial adversa cubierta por R5 devuelve `22023` y el mensaje opaco
`candidatura de alta invalida` antes de locks o escrituras.

Las comparaciones de generación, identidad de replay y los tres campos de
conflicto de alias pasan a `IS DISTINCT FROM`. `ROWS FROM`, las ACL, la ABI,
los locks y la política HMAC permanecen intactos.

La regresión focal llama directamente al SQL como runtime, con una transacción
serializable UTC independiente por caso, después de estabilizar la candidatura
inicial y antes de rotar la política. Reutiliza sus pares HMAC reales y cubre
las diez entradas nulas una a una, matrices multidimensionales, límites cero,
vacías, cardinalidad cinco, cardinalidades distintas y un elemento nulo en
cada matriz. Cada rechazo exige estado `22023`, mensaje opaco, rollback,
historia y recuentos invariantes y ningún efecto administrativo.

## Capability, invariantes y write-set de R5

Capability: rechazar exhaustiva y saneadamente toda entrada nula o matriz HMAC
no canónica antes de resolver o hacer replay, conservando ABI y autoridad.

Invariantes:

- las diez entradas requeridas son no nulas;
- ambas matrices son exactamente unidimensionales, con límite inferior uno,
  cardinalidad común entre uno y cuatro y elementos no nulos;
- identidad, generación y conflictos se comparan de forma null-safe;
- toda forma adversa falla con `22023` antes de locks o escrituras;
- historia y efectos permanecen intactos; y
- `ROWS FROM`, ACL, ABI, locks y política HMAC no cambian.

Write-set exacto:

1. `deploy/postgresql/contratacion_temporal/migraciones/000047_componentes/020_resolucion_candidatura.sql`;
2. `internal/modules/contrataciontemporal/adapters/postgres/confirmacion_alta_postgresql18_test.go`; y
3. `docs/portal_vec/ct_o2_r3b_candidatura_postgresql_2026-08-31.md`.

### NO-GO estático independiente de R5

La revisión independiente del hash exacto
`8097acbdd12fe03ef89719cad63d7fbb10d7ba3f` emitió `NO-GO`. Cada fixture
multidimensional construía `{{a,b},{a,b}}`: la matriz adversa tenía
cardinalidad cuatro mientras su contraparte canónica conservaba cardinalidad
dos. Aunque se eliminase la validación `array_ndims(...) <> 1`, el resolutor
seguiría rechazando por cardinalidades distintas. Por tanto, la regresión era
vacua para demostrar el control de dimensionalidad.

El dictamen no debilita ni reinterpreta los demás rechazos de R5 y
no autoriza PostgreSQL, Docker ni otra ejecución dinámica. Las puertas offline
verdes de R5 tampoco convierten el candidato en `GO`.

### Corrección R6

R6 anida una sola vez cada literal canónico de dos elementos. Así genera
`{{a,b}}`, una matriz bidimensional de cardinalidad dos, y conserva la otra
entrada como `{a,b}`, un array unidimensional también de cardinalidad dos.

Antes de cada una de las dos llamadas adversas, la prueba consulta
explícitamente `array_ndims`, `array_lower(...,1)` y `cardinality` para ambas
entradas. Exige `2/1/2` en la matriz adversa, `1/1/2` en la contraparte y
ausencia de elementos nulos en las dos. La llamada posterior reutiliza esos
mismos argumentos desde el rol runtime dentro de su propia transacción
serializable y sigue exigiendo `22023`, el mensaje opaco
`candidatura de alta invalida`, rollback e invariancia exacta de historia y
efectos.

## Capability, invariante y write-set de R6

Capability: demostrar de forma no vacua que el resolutor rechaza matrices
PostgreSQL multidimensionales aunque ambas entradas conserven la cardinalidad
válida e igual.

Invariante: cada una de las dos regresiones multidimensionales difiere de la
entrada estable únicamente en `array_ndims != 1`. Cardinalidad, límite
inferior, ausencia de nulos, contraparte canónica y las demás precondiciones
permanecen válidas; los diecinueve casos existentes se conservan y el SQL
productivo no cambia.

Write-set exacto:

1. `internal/modules/contrataciontemporal/adapters/postgres/confirmacion_alta_postgresql18_test.go`; y
2. `docs/portal_vec/ct_o2_r3b_candidatura_postgresql_2026-08-31.md`.

El corte productor R6 no modificó SQL ni el runner y no ejecutó PostgreSQL,
Docker, red, gates globales, integración o producción. La ejecución dinámica
posterior queda registrada por separado a continuación.

## Puertas offline de R6

El preflight local verificó antes de editar la ruta, rama, `HEAD` inicial
`8097acbdd12fe03ef89719cad63d7fbb10d7ba3f`, árbol limpio, Go 1.26.5,
formato, sintaxis Bash, ShellCheck, modos y límites iniciales.

Después de la corrección terminaron verdes `gofmt`, las pruebas focales
normales y con carrera de `ports` y del adaptador PostgreSQL con
`GOPROXY=off`, y `go vet` sobre esos dos paquetes. El runner, leído sin
ejecutarlo ni modificarlo, superó `bash -n` y ShellCheck. También quedaron
verdes `git diff --check`, los modos `644/644/755`, el límite de 800 líneas
del test —789 líneas— y Gitleaks sobre los dos ficheros del write-set. Para
Gitleaks se verificó antes el SHA-256 exigido
`c100de843d374f76143b03487de20fe341fb20cae8a71b6fdff896aec561391d`.

Estas puertas son exclusivamente estáticas y offline. No sustituyen la
revisión independiente ni autorizan PostgreSQL, Docker, red, gates globales,
integración o producción.

### Quinto NO-GO dinámico focal: ambigüedades PL/pgSQL de R6

La ejecución dinámica ya realizada sobre R6 falló en la resolución nominal de
la candidatura; R7B no la repite. El log preservado tiene SHA-256
`3b6d53d3be4687a2ec5e8db3911b3c774223227134de1355ee947fe6341c9d60`.
Con `plpgsql.variable_conflict=error`, la consulta de raíces referenciaba
`ambito_hmac` sin cualificar, nombre que también pertenece al parámetro `OUT`.
La auditoría completa de la función detectó la segunda colisión en
`ON CONFLICT (ambito_hmac)`.

R7B asigna alias a la tabla y cualifica todas las columnas de la consulta de
raíces; además, sustituye la inferencia de columna por `ON CONFLICT ON
CONSTRAINT candidatura_alta_alias_pkey DO NOTHING`. No cambia datos, semántica
ni ABI. Su capability es la resolución nominal compatible con
`plpgsql.variable_conflict=error`, y su invariante es que ninguna referencia
SQL de la función colisione con el parámetro `OUT ambito_hmac`. El write-set se
limita al SQL del resolutor y a esta acta. La validación dinámica posterior
queda registrada a continuación y no convierte el candidato en `GO`.

### Sexto NO-GO dinámico focal: rotación anticipada en la prueba

La ejecución dinámica posterior sobre el hash exacto
`2f995d734faf1f3725a8d5c2f571f6d5a2ca8797` confirmó que R7B corrigió las dos
ambigüedades PL/pgSQL: superó la resolución nominal, la matriz de nulidad y
forma, el replay entre procesos y la concurrencia. El único fallo apareció en
`rotacion_anade_alias_sin_mutar_raiz`, línea 123, con el error Go opaco:

```text
contratacion temporal: persistencia no disponible
```

El subtest invocaba `rotarPoliticaR3B` antes de crear la candidatura original
con generaciones `[2,1]`. Por ello el SQL recibía una política ya rotada
`[3,2,1]`, rechazaba correctamente con `22023` la solicitud inicial `[2,1]` y
el adaptador opacificaba el detalle. El fallo pertenece al orden de la prueba,
no demuestra un defecto nuevo del SQL, de su ABI, de las ACL ni de los locks.

El log `/tmp/vec-o2-r3b-r7b-dynamic-20260901.log` tiene SHA-256
`92c660c7001c597a32e82922d247ac45a7ac92c672839a75962ec9cc4c45c581`. Su
metadato `/tmp/vec-o2-r3b-r7b-dynamic-20260901.meta` tiene SHA-256
`31f884f1585bf7877d3f6d7c1fca8347c7699e6b6107ab131c35cbce61832320` y
registra `RC=1`, ocho segundos, fuente limpia, cero residuos y cerrojo libre.
El contenedor ajeno permaneció idéntico antes y después:

```text
d6217278d8718a4e51c4be9523dde15406559989abe46f3a5e496624d6aa4aeb|running|/contagrx-t224-pg-focal|sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382
```

La metadata fue reconstruida desde los instantes del fichero y comprobaciones
inmediatas porque su escritura original no se alcanzó. Este resultado conserva
el estado `NO-GO`, no acredita la matriz PostgreSQL completa y no autoriza una
repetición.

## Capability, invariantes y write-set de R8

Capability: acreditar de forma no vacua la rotación HMAC real, estabilizando
primero la raíz bajo `[2,1]`, rotando después a `[3,2,1]` y recuperando la
misma candidatura al añadir solo el alias de la nueva generación activa.

Invariantes:

- la candidatura original se estabiliza antes de rotar;
- raíz, identidad, instante y propuesta originales no mutan;
- el replay posterior usa exactamente `[3,2,1]`;
- la propuesta alternativa se descarta;
- los aliases pasan exactamente de dos a tres;
- resolver no crea ningún efecto administrativo; y
- SQL, ABI, ACL, locks y runner permanecen intactos.

Write-set exacto:

1. `internal/modules/contrataciontemporal/adapters/postgres/confirmacion_alta_postgresql18_test.go`; y
2. `docs/portal_vec/ct_o2_r3b_candidatura_postgresql_2026-08-31.md`.

R8 mueve `rotarPoliticaR3B` hasta después de comprobar la candidatura original
y antes de construir el replay rotado. No ejecuta PostgreSQL ni modifica SQL o
runner.

### Séptimo NO-GO dinámico focal: igualdad indebida de ventanas en R8

La ejecución dinámica posterior sobre el hash exacto
`637fe26d38fe3ef6278be1bcf0bd5cea0ef12b1f` terminó con `RC=1` en
`TestConfirmacionAltaPublicaPostgreSQL18DesdeDosPools`, línea 575, al devolver
`prepararEntradasConfirmarAlta` el error opaco:

```text
contratacion temporal: persistencia no disponible
```

El log inmutable `/tmp/vec-o2-r3b-r8-dynamic-20260902.log` tiene SHA-256
`3a0d80dda00cd6582c629f7a8487a8d5ef3bead1b3539d40bef9c27aa216525a`.
Su metadato `/tmp/vec-o2-r3b-r8-dynamic-20260902.meta` tiene SHA-256
`261f3c61d37d744b0c01545b3696f1ab3d0b788b7ad43dad833929395fc3a1b2`
y registra quince segundos, fuente limpia, cero residuos y cerrojo libre. El
contenedor ajeno permaneció idéntico antes y después:

```text
d6217278d8718a4e51c4be9523dde15406559989abe46f3a5e496624d6aa4aeb|running|/contagrx-t224-pg-focal|sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382
```

La causa exacta está en el cotejo Go: exigía que la capacidad AD-3 de cinco
segundos tuviera los mismos extremos que la concesión V3 registrada de noventa
segundos. La garantía requerida es contención completa de la ventana breve,
no igualdad. Este resultado conserva `NO-GO`, no declara PostgreSQL verde y
no autoriza una nueva ejecución dinámica.

## Capability, invariantes y write-set de R9

Capability: aceptar una capacidad breve solo cuando toda su ventana está
contenida en la concesión registrada, sin ampliar ninguno de sus extremos.

Invariantes:

- la emisión de la capacidad no puede preceder ni a la decisión ni al registro
  durable de la concesión;
- la emisión sigue perteneciendo al intervalo half-open y debe ser
  estrictamente anterior a `ValidaHasta`;
- la expiración puede coincidir con `ValidaHasta`, pero nunca superarlo;
- `ValidarEstructura` y la duración máxima AD-3 de cinco segundos permanecen
  intactos; y
- SQL, runner, fixture PostgreSQL 18, `materialPublicoR3B` y evidencias
  dinámicas permanecen inmutables.

Write-set exacto:

1. `internal/modules/contrataciontemporal/adapters/postgres/confirmacion_alta.go`;
2. `internal/modules/contrataciontemporal/adapters/postgres/confirmacion_alta_test.go`; y
3. `docs/portal_vec/ct_o2_r3b_candidatura_postgresql_2026-08-31.md`.

Las regresiones estáticas cubren concesión de noventa segundos que contiene
capacidad de cinco, emisión anterior a la decisión, emisión anterior al
registro, emisión exactamente en el límite final y expiración posterior al
límite. R9 no ejecuta PostgreSQL, Docker ni el runner, no repite R8 y no
convierte ninguna puerta dinámica anterior en verde.

## Capacidad e invariante funcional

Este corte estabiliza o recupera, antes de autorización, una única
`CandidaturaAlta` técnica mediante pares HMAC de una generación activa y hasta
tres retenidas. La candidatura es no autoritativa: resolverla no crea reserva
administrativa, expediente, actuación, auditoría, outbox ni consumo VEC.

El único efecto sigue en `confirmar_alta_atestada_v2`. La función acredita la
candidatura exacta —coordenadas, identidad, instante y pares generacionales— y,
en la misma transacción serializable, delega en el motor privado
`confirmar_alta_atestada_v1`. No se fabrica una `PreparacionAlta`.

## Fronteras

- `resolver_candidatura_alta_tecnica_v1` recibe diez entradas: dos matrices
  HMAC, organización, actor, perfil, cuatro referencias propuestas e instante
  propuesto. Devuelve once columnas y solo los estados `estabilizada` o
  `recuperada`.
- `confirmar_alta_atestada_v2` conserva las doce entradas y las ocho salidas de
  O2-05. V1 deja de ser ejecutable por el runtime y permanece como motor
  `SECURITY DEFINER` privado.
- `ProveedorMaterialConfirmacionAlta` entrega una única instantánea neutral de
  las diez entradas VEC-AD-3. Su implementación concreta corresponde a R3C.
- `TransaccionAltasCandidata` recibe únicamente una
  `OrdenConfirmarAltaCandidata`; no acepta bytes del cliente ni una correlación
  libre.

## Persistencia, rotación y backfill

La migración `000047_candidatura_alta_durable_o2_r3b` crea una raíz inmutable y
una tabla append-only de pares de alias. Ámbito y huella se guardan alineados en
la misma fila y una generación solo puede pertenecer a una raíz. La clave HMAC
en claro nunca cruza SQL ni logs: PostgreSQL recibe únicamente sellos.

Un replay exacto devuelve las referencias, identidad, par raíz e instante
originales. Una generación nueva añade un alias sin mutar la raíz. Un cruce de
pares, otra huella o identidad para el mismo ámbito y una colisión de
referencias se rechazan; una colisión aleatoria no se interpreta como
idempotencia.

El backfill copia las identidades de reserva anteriores y conserva
`identidad_reserva_alta.creada_en` como `instante_efecto`. El `DOWN` ordinario
queda bloqueado si existe historia nueva. La retirada destructiva exige el
testigo explícito ya gobernado por las migraciones CT.

## Atomicidad y tratamiento de errores

El adaptador abre transacciones nuevas, serializables y read-write. Solo
`40001` y `40P01` se reintentan, como máximo tres intentos. Un COMMIT confirmado
prevalece sobre una cancelación posterior.

`pgconn.SafeToRetry` y `pgx.ErrTxCommitRollback` no provocan reconciliación.
`08007` o un error de transporte con envío posible permiten una única
reconciliación independiente, limitada a cinco segundos, con las mismas doce
entradas. Una segunda ambigüedad devuelve `ErrResultadoAltaIndeterminado`.
Una fila, recibo o `recibo_huella_sha256` divergente se revierte y devuelve
`ErrResultadoAltaNoConfiable`. Ningún error expone SQL, nombres de tablas o
coordenadas internas.

## Canon y ACL

Go construye byte a byte `vec.contratacion-temporal.efecto-alta.v2` y
`vec.contratacion-temporal.sellos-hmac.v1`, con UTC canónico a microsegundos.
El hash del efecto debe coincidir con el atributo autorizado y el resumen de la
capacidad debe ligar decisión, correlación, operación, efecto, audiencia y
ventana antes de abrir la transacción; su intervalo breve debe quedar contenido
en la concesión registrada.

El runtime carece de privilegios sobre las tablas nuevas, sobre
`preparar_alta_v2` y sobre `confirmar_alta_atestada_v1`. Para este flujo solo
recibe `EXECUTE` sobre el resolutor y V2. Las funciones auxiliares y las tablas
siguen cerradas a `PUBLIC`. Ambas tablas activan y fuerzan RLS; su única
política pertenece al rol propietario y los triggers conservan el historial
append-only frente a `UPDATE`, `DELETE` y `TRUNCATE`, también cuando actúa el
propietario.

## Evidencia focal

El runner `probar_o2_r3b_candidatura_postgresql18_4.sh` usa exclusivamente la
imagen local PostgreSQL 18.4 fijada por digest y recursos etiquetados propios.
Instala `000047` después de `000046`. La primera ejecución dinámica posterior
a R2 se detuvo en el oráculo del preflight; la siguiente, ya sobre R3, superó
ese preflight y se detuvo en la primera resolución Go. La ejecución posterior
de R6 volvió a detenerse en la resolución nominal por la primera ambigüedad
PL/pgSQL descrita arriba. R7B corrigió ambas ambigüedades y su ejecución superó
nominal, nulidad y forma, replay y concurrencia, pero se detuvo en el orden
anticipado de la rotación. R4 y R5 no se ejecutaron contra PostgreSQL; ninguna
ejecución alcanzó la matriz completa que una autorización dinámica futura
tendría que acreditar:

- backfill e instante original;
- replay entre pools, concurrencia y rotación con alias;
- rechazo de `UPDATE`, `DELETE` y `TRUNCATE` por propietario, con recuentos y
  huellas de ambas tablas exactamente conservados;
- conflictos, colisión, rollback y ausencia de efectos al resolver;
- construcción del adaptador Go público con proveedor neutral, doce entradas,
  escaneo real de ocho columnas, confirmación y replay exacto desde dos pools;
- cotejo del recibo completo y de `recibo_huella_sha256` mediante la función
  privada Go, sin exponer esa huella por el puerto;
- rechazo de mutación de coordenada y revocación viva sin efectos;
- ACL/RLS, `DOWN` protegido, retirada explícita y reinstalación; y
- conservación del contenedor ajeno y ausencia de residuos propios.

Las pruebas unitarias Go cubren además `40001` y `40P01` en transacciones
nuevas con máximo de tres intentos, cancelación posterior al COMMIT,
`SafeToRetry`, `pgx.ErrTxCommitRollback`, `08007`, transporte posiblemente
enviado, primera y segunda ambigüedad, y adulteración de fila/recibo/hash. Para
cada clase se comprueban inicios, commits y reconciliaciones; la rama
`SafeToRetry` inyecta ahora el fallo en `Commit`, no antes de alcanzarlo.

## Límite y siguiente corte

R3B no modifica aplicación, HTTP, rutas, composición ni frontend. Tampoco
cierra O2-06 ni declara la aplicación arrancable. El siguiente corte es la
revisión independiente del hash exacto de R9. Este corte no autoriza una nueva
validación dinámica; cualquier repetición requerirá una autorización posterior
expresa y separada. No se declara PostgreSQL verde y la integración continúa
prohibida. R3C permanece bloqueada hasta resolver este `NO-GO`; después deberá
migrar `ServicioRegistroSolicitud` al contrato candidato y componer el
proveedor concreto de material de confirmación bajo revisión independiente.
