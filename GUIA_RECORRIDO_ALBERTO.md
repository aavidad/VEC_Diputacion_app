# Arranque vigente en el equipo local — 5 de septiembre de 2026

Base publicada del corte actual: `4034157`. El cierre de bandeja,
detalle y análisis corresponde a
`b2effbaf09fd4ad8477bf42c56e4615ff52d0c62`, con el corrector SQL `13f7a92`
y la interfaz integrados por avance directo. El desarrollo utiliza la misma rama
`trabajo/ct-app-llamamiento-b4a-20260905`, en el worktree local
`.worktrees/ct-app-llamamiento-b4a-20260905`. Las instancias de desarrollo
remotas están detenidas y conservadas; no se programa en dos líneas paralelas.

**Corte 3 incluido en esta entrega; recuperación demostrada:** respuesta declarada
por RRHH registrada desde el navegador (`201`). Después del parche temporal y
del segundo reinicio de aplicación y PostgreSQL principal, dirección confirmó
`200/200/200` al recuperar selección, comunicación y respuesta con sus claves
originales. Se conservan el mismo justificante, recibo y fecha de registro.
El defecto de comparación de fechas está corregido en ambas bases y el
diagnóstico temporal retirado. El corte queda cerrado técnicamente.
La métrica sigue en **5 de 8 pasos completos más parte del sexto**.

**Corte actual: solicitar resolución, sin aceptación terminal.** Dirección
comprobó `200/200/200/409`: recupera los tres antecedentes originales y la cuarta
operación informa de validación de respuesta/plazo pendiente, sin duplicados.
Las piezas Bolsa Go y SQL AD3 `000015` / Bolsa `000004` están preparadas, no
activadas ni instaladas permanentemente en BD. Faltan validación de negocio
competente y proveedor de permiso nominal; no aplicar SQL para eludirlas.

Las dos bases y el material de desarrollo se han trasladado sin regenerar
identidades, claves ni expedientes. Las copias físicas se verificaron antes
de arrancar. En la comprobación del traslado, después de reiniciar aplicación
y PostgreSQL locales, la base secundaria mostró los mismos 21 expedientes y
abrió el mismo detalle, versión 1. La base principal conservó sus 50 altas
confirmadas y 24 asientos de tramitación.
Son conjuntos diferentes: no mezclarlos ni usar uno para rellenar el otro.

| Modalidad del lanzador | Portal local | Base y alcance actual |
| --- | --- | --- |
| `recorrido` — principal | `https://localhost:8443/portal-empleado/` | PostgreSQL 55433: cinco pasos y parte del sexto; bandeja, detalle y análisis de una solicitud existente comprobados y publicados. |
| `consultas` — secundaria | `https://localhost:8444/portal-empleado/` | PostgreSQL 55432: bandeja y detalle reales en el entorno aislado. No ejecutar pruebas simultáneas contra esta base mientras la usa la aplicación. |

Ambas bases tienen instalada la dependencia de consultas. Dirección confirmó
en 55433 la publicación del catálogo sintético original y sus dos vínculos:
tres resultados positivos en una transacción, secuencia de catálogo `8`.
No copiar filas entre bases ni repetir esa instalación para arrancar.
Las migraciones de registro de respuesta `000056` de Contratación temporal y
`000014` de autorización también están instaladas en **ambas bases**: no
reaplicarlas. Dirección aplicó literalmente el bloque `DO $fechas$` de la
migración `000014` en ambas bases, con sus tres comprobaciones incorporadas
correctas. El diagnóstico temporal de la función de respuesta también se retiró
de ambas bases. La huella SHA256 final del núcleo de autorización comunicada
por dirección es
`42f67b75786e996c56309350389801091cf749adb85a2e7b6d40ee49c399fb62`.
No repetir el bloque temporal para arrancar: también está aplicado.
Las huellas anteriores del historial no son referencias operativas actuales.
Cada modalidad requiere **once conexiones PostgreSQL separadas**, todas a su
propia base. Sus variables y funciones están en el
[manual de sistemas, apartado 2](docs/manual_sistemas/README.md#2-configuración-local-once-conexiones-por-instancia).

El operador conserva fuera de Git el lanzador `arrancar-local.sh`, las bases
y los certificados. La bitácora local identifica su ruta exacta. Defina
`VEC_ESTADO_LOCAL` con ese directorio privado existente, sin generar otro:

```bash
: "${VEC_ESTADO_LOCAL:?Indique el directorio privado conservado en la bitácora local}"
test -f "$VEC_ESTADO_LOCAL/arrancar-local.sh"
bash "$VEC_ESTADO_LOCAL/arrancar-local.sh" recorrido
# Solo si se necesita la instancia secundaria, en otra terminal:
bash "$VEC_ESTADO_LOCAL/arrancar-local.sh" consultas
```

No arranque otra copia si el puerto ya está ocupado. Ctrl-C detiene esa
aplicación, no borra la base. El lanzador reutiliza y comprueba el material
existente; las instrucciones de importación del certificado se muestran al
arrancar. Es desarrollo sintético, no producción ni identidad corporativa.

### Abrir una solicitud existente y registrar su análisis

Recorrido confirmado por dirección el 5 de septiembre en **8443**, incluido
en el código publicado: bandeja de **50 expedientes** → solicitud existente versión `1` →
formulario real de análisis → respuesta `201` → recibo versión `2`.
Se conservaron las 50 altas: esta actuación no registra una solicitud nueva.

1. Abra el portal principal con el certificado de RRHH y entre en
   **Contratación temporal**. Use la bandeja, no **Nueva petición**.
2. Localice una fila en fase **Solicitud**, en curso, versión `1`, y ábrala
   con los controles de esa fila. La aplicación lleva su referencia al
   detalle: no hace falta copiar identificadores ni llamar a la API a mano.
3. Compruebe el expediente y su versión. Debe aparecer el formulario real
   **Análisis por Recursos Humanos** para esa solicitud, sin reconstruir el alta.
4. Complete sus campos con los catálogos y datos sintéticos válidos del
   formulario. Pulse **Registrar análisis** una sola vez.
5. Compruebe que el recibo conserva el expediente y devuelve versión `2`.
   En la red, lista y detalle usan `/cuadro/consultas` y
   `/expedientes/consultas`; el análisis usa `/analisis/registros`, con
   respuesta `201`. Todas pertenecen a `/api/vec/contratacion-temporal`.

En la comprobación se usó la solicitud identificada por el tramo
`b50fa719…` de su referencia; el recibo fue
`rec_ct_an_c4c3b1fc86cc0d5531e655b26bd68096`, versión `2`.
La referencia abreviada solo identifica la evidencia: no se pega en formularios.
Esa solicitud ya está analizada; no repetirla como si continuara en versión `1`.
Abrir otra solicitud para analizarla constituye otra actuación, no una consulta.

Si el formulario no aparece, hay una actualización pendiente o se muestra un
error, no fuerce la versión ni cree otra alta para eludirlo. Una pantalla
cargada no sustituye al recibo real. Se mantiene el límite de **cinco pasos
completos y parte del sexto**, con aviso local, no correo corporativo.

**Reinicio confirmado por dirección:** la base principal sigue mostrando
50 expedientes y el detalle conserva la versión `2`. Una lectura independiente
confirmó un único recibo, asiento y terminal del análisis en esa versión.
La comprobación conserva el efecto único, sin otra alta.
La entrega de bandeja y análisis ya está integrada y publicada en `b2effbaf`;
no queda código de ese corte pendiente de integrar. Publicar el código no
reactiva las instancias remotas ni autoriza producción.

### Corte 3: registrar la declaración de una respuesta recibida por RRHH

El recorrido real en la principal devolvió `201`: RRHH declaró una
**aceptación recibida**, con referencia del correo, huella SHA256, identidad
de quien registra y justificante persistentes. La lectura de la base confirmó
**una respuesta, un asiento de historial y un evento de salida**.
Es una declaración registrada, **no una aceptación terminal**: no cambia el
estado de Bolsa, no avanza el expediente y la comunicación sigue en versión `2`.
No verifica origen, firma ni custodia del correo; tampoco acredita envío o
entrega del aviso local. La cuarta operación permite solicitar la resolución,
pero sigue pendiente la aceptación válida con su autorización y comprobaciones
propias; no convierte esta declaración automáticamente en aceptación.

Para recuperar este caso use los datos originales, sin crear otra solicitud
ni cambiar claves. El acceso requiere el certificado y contexto autorizado
de RRHH; conocer las referencias no sustituye el permiso.

1. Abra el llamamiento del expediente
   `expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001`.
   Recupere la selección con versión `6` y clave
   `90d52c16-a63d-4ef1-bcf7-62c7c455f9aa`, sin preparar una selección nueva.
2. Recupere su comunicación local con la clave
   `d1b5428f-2188-4b8f-98b7-42f82ad88c2a`. El recibo aporta las referencias
   de llamamiento y comunicación al formulario de respuesta; compruebe
   versión de comunicación `2`. No invente ni cambie esas referencias.
3. En el formulario de respuesta, mantenga los valores de la tabla y cargue
   el mismo [correo sintético de ejemplo](docs/manual_rrhh/ejemplos/respuesta_sintetica.eml),
   sin editarlo ni cambiar sus saltos de línea. La huella se calcula en el
   navegador: el archivo `.eml` (máximo 2 MiB) **no se sube ni queda custodiado
   por la aplicación**. Se envían su referencia y huella declaradas, no su
   contenido ni nombre de archivo. La fecha del formulario se expresa en UTC.

| Dato de la respuesta original | Valor que debe conservarse |
| --- | --- |
| Clave de operación | `c3e0f431-b274-48fd-a2e8-4b1e6d220056` |
| Respuesta declarada | `aceptacion` |
| Referencia del correo | `correo:sintetico:respuesta-20260905-0056` |
| Fecha de recepción declarada, UTC | `2026-09-05T10:00:00Z` |
| SHA256 del archivo | `7984edfd3ba13c87b0c04160dbfa8b338b356ead70d80df04066e67e4ed419b9` |

Desde el directorio de esta guía puede comprobar los bytes del ejemplo:

```bash
sha256sum docs/manual_rrhh/ejemplos/respuesta_sintetica.eml
```

El registro inicial se realizó en
`/api/vec/contratacion-temporal/llamamientos/respuestas/registro`, con estado
`registrada_por_rrhh`. Conservó el justificante
`84727d1d-31ef-4fde-92c8-d3a8e2953931`, el recibo
`9e14599d-2edc-42aa-afde-170420c838aa` y la fecha de registro
`2026-09-05T18:09:06.065542Z`.

**Defecto corregido y recuperación confirmada:** los rechazos intermitentes
`403` procedían de comparar fechas como texto (seis decimales frente a una
representación que omite ceros finales), aunque expresasen el mismo instante;
no del reinicio. Ambas bases comparan ahora instantes mediante conversión a
`timestamptz`, sin cambiar bytes firmados ni permisos. El diagnóstico temporal
está retirado.

Después de aplicar esa corrección y del segundo reinicio de aplicación y
PostgreSQL principal, dirección confirmó desde el navegador las tres
recuperaciones `200/200/200`: selección, comunicación y respuesta. Con las
mismas claves y material, la respuesta devuelve `replay_registrada_por_rrhh`
y conserva el justificante `84727d1d-31ef-4fde-92c8-d3a8e2953931`, el recibo
`9e14599d-2edc-42aa-afde-170420c838aa` y la fecha original
`2026-09-05T18:09:06.065542Z`. También se confirmó ausencia de errores de
JavaScript, cookies, almacenamiento web y desbordamiento horizontal en móvil.
También se comprobó desde el navegador el rechazo `409` por conflicto,
sin generar un duplicado. El corte está incluido en esta entrega, cerrado
técnicamente; no equivale a cerrar todo el paso 6.
Ante un rechazo posterior, no encadene reintentos ni genere otra clave para
sortearlo: conservar la operación original sigue siendo obligatorio.

### Corte 4 en curso: solicitar resolución de aceptación

Tras recuperar la declaración `aceptacion` anterior, el mismo formulario muestra
**4. Solicitar resolución de aceptación** (`data-ct-llamamiento-form="resolucion"`).
Organización, expediente, llamamiento y comunicación proceden del recibo original;
la versión esperada es `2` y `prueba_respuesta_ref` es su `justificante_ref`, no
el recibo ni una referencia inventada. Solo la clave es editable: debe ser propia
de resolución, distinta de las tres anteriores; si ya hizo el intento, conserve
su clave. No se carga otro `.eml` ni se introducen identidad o plazo en pantalla.

Pulse **Revisar y solicitar resolución de aceptación** y confirme expresamente.
`POST /api/vec/contratacion-temporal/llamamientos/resoluciones` devuelve actualmente
`409 validacion_respuesta_pendiente`: **«Pendiente de validar respuesta y plazo
por RRHH. No se ha confirmado la aceptación.»**. Es una condición conocida sin
efecto, no un resultado indeterminado; se conserva el intento para volver a
solicitarlo manualmente con los mismos datos, nunca automáticamente ni con otra
clave. No genera recibo de aceptación ni permite continuar al nombramiento.

El navegador real confirmó `200/200/200/409`, sin duplicados ni cambios de estado.
La dinámica SQL aislada comprobó almacenamiento con un doble de autorización
estrictamente privado y transaccional, sin datos ni migraciones persistidos;
no acredita criptografía ni aceptación funcional E2E.
Las preguntas pendientes no detienen programación independiente; no conceden
plazo ni autoridad. Véase el [plan vigente](ESTADO_PROYECTO.md).

## Recorrido remoto del 4 de septiembre — historial conservado

Las secciones posteriores conservan los ejemplos funcionales y la evidencia
anterior. Sus comandos SSH, rutas remotas y apéndice de recreación no son
instrucciones de arranque actuales. Para operar aquí use el bloque precedente;
no recree bases ni arranque el desarrollo remoto.

Esta guía permite probar los cinco primeros pasos completos del flujo de
Recursos Humanos, incluida la Fiscalización y su devolución a la unidad, con
la aplicación real: navegador con certificado de cliente → API interna →
autorización de servidor → PostgreSQL → recibo. No usa el adaptador DEMO.

## Entorno preservado para el recorrido manual

La evidencia sintética del 4 de septiembre de 2026 sigue disponible en el
servidor. No es necesario recrearla ni borrarla para recorrer otro expediente:

- repositorio: `/srv/fabrica/proyectos/VEC_Diputacion_app`;
- producto: `.worktrees/ct-producto-ligero-20260821`;
- única línea de trabajo del paso 6:
  `.worktrees/ct-app-llamamiento-b4a-20260905`, base `83fc7631`;
- PostgreSQL exclusivo del navegador: contenedor
  `vec-ct-o2-07-browser-20260904-tls`, puerto local remoto `55433`;
- PostgreSQL reservado para pruebas Go: contenedor
  `vec-ct-o2-07-e2e-20260904-tls`, puerto local remoto `55432`;
- material HTTPS y certificados separados de RRHH e Intervención:
  `/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904`.

Regla obligatoria: el servidor usado por el navegador solo apunta a `55433`.
Las pruebas Go solo apuntan a `55432`. Nunca se usan a la vez contra la misma
instancia.

Ambas instancias ya tienen autorización `000011/000012/000013`, Bolsa `000003`
y Contratación temporal `000053/000054/000055`. No recrear, borrar ni reaplicar
migraciones o concesiones. El apéndice de recreación no se usa en este hito.

## 1. Comprobar la instancia aislada

En una terminal del servidor:

```bash
ssh root@cidonia.cloud
cd /srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/ct-app-llamamiento-b4a-20260905
git status --short --branch
docker start vec-ct-o2-07-browser-20260904-tls >/dev/null
docker inspect vec-ct-o2-07-browser-20260904-tls \
  --format 'estado={{.State.Status}} imagen={{.Config.Image}} {{range .Mounts}}volumen={{.Name}} destino={{.Destination}}{{end}} puertos={{json .HostConfig.PortBindings}}'
docker exec vec-ct-o2-07-browser-20260904-tls \
  psql -X -U postgres -d postgres -Atqc \
  "SELECT concat_ws('/',
    (SELECT count(*) FROM vec_contratacion_temporal.candidatura_alta_tecnica c
      JOIN vec_contratacion_temporal.confirmacion_agregado_alta a USING(expediente_ref)),
    (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta),
    (SELECT count(*) FROM vec_contratacion_temporal.confirmacion_agregado_alta));"
docker exec vec-ct-o2-07-browser-20260904-tls \
  psql -X -U postgres -d postgres -Atqc \
  "SELECT login.rolcanlogin::text||'|'||string_agg(grupo.rolname,',' ORDER BY grupo.rolname)
     FROM pg_roles login
     JOIN pg_auth_members miembro ON miembro.member=login.oid
     JOIN pg_roles grupo ON grupo.oid=miembro.roleid
    WHERE login.rolname='vec_autorizacion_o207_registro'
    GROUP BY login.rolcanlogin;"
docker exec vec-ct-o2-07-browser-20260904-tls \
  psql -X -U postgres -d postgres -Atqc \
  "SELECT login.rolcanlogin::text||'|'||string_agg(grupo.rolname,',' ORDER BY grupo.rolname)
     FROM pg_roles login
     JOIN pg_auth_members miembro ON miembro.member=login.oid
     JOIN pg_roles grupo ON grupo.oid=miembro.roleid
    WHERE login.rolname='vec_ad3_o207_gobierno'
      AND grupo.rolname='vec_contratacion_temporal_gobernador'
    GROUP BY login.rolcanlogin;"
```

El producto debe seguir limpio; la candidata contiene el trabajo compartido
del paso 6 y no se limpia ni se sustituye. El contenedor debe estar `running`, usar la imagen
fijada por digest, montar `vec-ct-o2-07-browser-20260904-data` en
`/var/lib/postgresql` y publicar únicamente `127.0.0.1:55433`. Tras la evidencia
automatizada, los tres recuentos deben ser iguales y mayores que cero; el valor
crece con cada solicitud sintética nueva.
La última consulta debe devolver exclusivamente
`true|vec_autorizacion_registro` y la siguiente
`true|vec_contratacion_temporal_gobernador`.

Antes de arrancar VEC, compruebe que PostgreSQL conserva exactamente las
definiciones de las migraciones de Asignación, informe jurídico y
Fiscalización incluidas en el producto. La comprobación es de solo lectura y
se detiene ante cualquier diferencia:

El núcleo común incluye ahora los cambios de `000011/000012/000013`.
Dirección confirma esta huella instalada en ambas bases (`55432` y `55433`):
`5079499ad2c05ead1afc6e36b3505249b30f7ba98b913c1e5b4e1c942b8d2b57`.
Es la definición ampliada que contrasta el comando siguiente; no debe
compararse con la huella anterior a la recuperación.

```bash
set -euo pipefail

fuente_autorizacion=deploy/postgresql/autorizacion_atestada_v3/migraciones/000008_consumidor_asignacion_v3_atestada.up.sql
fuente_asignacion=deploy/postgresql/contratacion_temporal/migraciones/000050_asignacion_durable_v3_v4.up.sql
fuente_autorizacion_informe=deploy/postgresql/autorizacion_atestada_v3/migraciones/000009_consumidor_informe_juridico_v3_atestada.up.sql
fuente_informe=deploy/postgresql/contratacion_temporal/migraciones/000051_informe_juridico_durable_v4_v5.up.sql
fuente_autorizacion_fiscalizacion=deploy/postgresql/autorizacion_atestada_v3/migraciones/000010_consumidor_fiscalizacion_v3_atestada.up.sql
fuente_fiscalizacion=deploy/postgresql/contratacion_temporal/migraciones/000052_fiscalizacion_durable_v5_v6.up.sql

test "$(sha256sum "$fuente_autorizacion" | awk '{print $1}')" = \
  42f7dfef32464f1a0ce2f3bcb9af035d800b7ceb302917d0652161408488451c
test "$(sha256sum "$fuente_asignacion" | awk '{print $1}')" = \
  cba96bd9a281e583b9f14edc661d3fc3b26b2a4898515c907c4d5f56cdf89e89
test "$(sha256sum "$fuente_autorizacion_informe" | awk '{print $1}')" = \
  32542f535e58f06668987be79ff6f23c66efc04957da964a2986cd79920934b5
test "$(sha256sum "$fuente_informe" | awk '{print $1}')" = \
  4fdbc64117f0f619ce7fb3758ccc63fde08cb21fd5f6ffbfe4cd6fb138be0840
test "$(sha256sum "$fuente_autorizacion_fiscalizacion" | awk '{print $1}')" = \
  a9f79e4486cdf7cf475d66d0f20015170d0c5627c8368fd786b53e0d086845df
test "$(sha256sum "$fuente_fiscalizacion" | awk '{print $1}')" = \
  c3258be1075381d5ce1d077a3ddc25c2e38d2961b87a5c637f0906fe2746ce61

huella_funcion() {
  docker exec vec-ct-o2-07-browser-20260904-tls \
    psql -X -U postgres -d postgres -Atqc \
    "SELECT pg_catalog.encode(public.digest(pg_catalog.convert_to(pg_catalog.pg_get_functiondef('$1'::regprocedure),'UTF8'),'sha256'),'hex');"
}

test "$(huella_funcion 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  5079499ad2c05ead1afc6e36b3505249b30f7ba98b913c1e5b4e1c942b8d2b57
test "$(huella_funcion 'vec_autorizacion_atestada_v3.registrar_y_consumir_asignacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  a384447d25ef86cab6bbdb99d32a2d478232a972a3c70e6290ea470c4917ce53
test "$(huella_funcion 'vec_contratacion_temporal.asignacion_claves_exactas_v1(jsonb,text[])')" = \
  e9dbba7cb7cb82f287a9edfb8ac43b862ada1d7ad74a196bece82735b28aa2e1
test "$(huella_funcion 'vec_contratacion_temporal.preparar_asignacion_v1(jsonb)')" = \
  977fef59d076ab551718970686ada6ee86c4bc702e5ebe679839098082badf65
test "$(huella_funcion 'vec_contratacion_temporal.consultar_asignacion_v1(jsonb)')" = \
  152a7e7ad94c6cf654aab22ffe74354b46eb18fd956a7aa5249793c54c0a84c8
test "$(huella_funcion 'vec_contratacion_temporal.confirmar_asignacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  f20c58b07740b8e2b5907d1d9e017d649b641812de0ce6ded41c64583ab02276
test "$(huella_funcion 'vec_autorizacion_atestada_v3.registrar_y_consumir_informe_juridico_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  c880864503d228663ea1058ebb4644d09aa191e2944f8ee646e2d2a606f41c8a
test "$(huella_funcion 'vec_contratacion_temporal.preparar_informe_juridico_v1(jsonb)')" = \
  a7f5d8ba0e8fa185a5ba46208e7bf608ff4f5c9d6a5bcbd05be2d4a64d4e4e44
test "$(huella_funcion 'vec_contratacion_temporal.recibo_informe_juridico_v1(text)')" = \
  1cbc9c91c9c712cb30a223e3666dbd7034d87f34c504407914644c93c1ed97ae
test "$(huella_funcion 'vec_contratacion_temporal.confirmar_informe_juridico_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  c71c053abf0f2f98fb01b7534d55f5e5c56e4ede92e81822911c36b7fe1da39e
test "$(huella_funcion 'vec_autorizacion_atestada_v3.registrar_y_consumir_fiscalizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  f96438ebd4a445c53240d0741f520fc52073fb4087820afe21b5618519610147
test "$(huella_funcion 'vec_contratacion_temporal.preparar_fiscalizacion_v1(jsonb)')" = \
  34a33a97425f02e59b66c05186c496a419c545e2ab7734b958a15f7be0cfb056
test "$(huella_funcion 'vec_contratacion_temporal.recibo_fiscalizacion_v1(text)')" = \
  27fc25b3d1fb12431db2e29ccad52f25d56344ea5ec5540a63e916c918dfaa36
test "$(huella_funcion 'vec_contratacion_temporal.confirmar_fiscalizacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  27e369c38267d18d3ed258eca1a986c5835717d9c9630740d2291033fddeb147

printf '%s\n' 'OK: Asignación, informe y Fiscalización coinciden'
```

Si alguna comparación falla, no abra el navegador ni reaplique migraciones a
ciegas: el código y la instancia no están alineados.

## 2. Arrancar VEC con PostgreSQL real

En la misma terminal remota, desde la candidata indicada, con las seis
conexiones separadas a `55433`:

```bash
directorio_pg=$(mktemp -d /tmp/vec-o2-07-pg.XXXXXX)
docker cp \
  vec-ct-o2-07-browser-20260904-tls:/var/lib/postgresql/18/docker/o207-ca.crt \
  "$directorio_pg/ca.crt"
chmod 600 "$directorio_pg/ca.crt"

export VEC_CT_DATABASE_URL="postgresql://vec_ct_o207_runtime@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_GOBIERNO_DATABASE_URL="postgresql://vec_ad3_o207_gobierno@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_CONFIRMADOR_DATABASE_URL="postgresql://vec_ct_o207_confirmador@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_LECTOR_RESULTADO_DATABASE_URL="postgresql://vec_ct_o207_lector@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL="postgresql://vec_autorizacion_o207_registro@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL="postgresql://vec_bolsa_llamamientos_desarrollo@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"

material_vec=/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904
scripts/arrancar_vec_desarrollo.sh --puerto 8443 \
  --directorio-material "$material_vec"
```

El lanzador genera o valida las credenciales fuera de Git, construye un binario
temporal y escucha solo en `127.0.0.1:8443` con TLS 1.3 y certificado de cliente
obligatorio. Se deja en primer plano. No es un despliegue.

## 3. Abrir el túnel y preparar el certificado del navegador

En una segunda terminal del equipo desde el que se abrirá el navegador:

```bash
directorio_navegador=$(mktemp -d /tmp/vec-o2-07-navegador.XXXXXX)
chmod 700 "$directorio_navegador"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/ca/ca.crt \
  "$directorio_navegador/ca.crt"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/cliente.p12 \
  "$directorio_navegador/cliente.p12"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/cliente.p12.password \
  "$directorio_navegador/cliente.p12.password"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/intervencion.p12 \
  "$directorio_navegador/intervencion.p12"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/intervencion.p12.password \
  "$directorio_navegador/intervencion.p12.password"
chmod 600 "$directorio_navegador"/*
ssh -N -L 18443:127.0.0.1:8443 root@cidonia.cloud
```

No muestre ni copie las contraseñas en la consola. Impórtelas directamente
desde sus ficheros cuando el navegador las pida. Use perfiles temporales
separados: `cliente.p12` representa RRHH e `intervencion.p12` representa
Intervención; ninguno sustituye al otro.

1. Antes de importar el certificado personal, abra
   `https://localhost:18443/portal-empleado/` en un perfil temporal: el acceso
   debe quedar bloqueado por falta de certificado de cliente. Eso demuestra la
   denegación predeterminada.
2. Importe `ca.crt` como autoridad de confianza para sitios web.
3. Importe `cliente.p12` como certificado personal usando su fichero de
   contraseña, cierre el perfil anterior y abra uno nuevo para RRHH.
4. Acceda a `https://localhost:18443/portal-empleado/` y elija
   **Contratación temporal** → **Nueva petición**.

Al acabar, cierre el perfil temporal. Elimine el directorio temporal mediante
el mecanismo de papelera o borrado seguro aprobado en su equipo; contiene una
credencial de desarrollo.

## 4. Registrar una solicitud, analizarla, decidir la cobertura, asignarla, informar y fiscalizar

1. Seleccione un centro, una persona de contacto referenciada, una categoría,
   un grupo o subgrupo y un motivo de los catálogos mostrados.
2. Escriba un detalle sintético inequívocamente nuevo, por ejemplo
   `Prueba manual Alberto 2026-09-04 07:15`.
3. Indique fechas válidas y marque que no existe retención de crédito para este
   dato sintético.
4. Pulse **Revisar solicitud** y luego **Confirmar y registrar** una sola vez.
5. En las herramientas de red del navegador debe aparecer un único
   `POST /api/vec/contratacion-temporal/solicitudes` con estado `201`.
6. La pantalla debe mostrar **Solicitud registrada** y un recibo real con
   referencia de expediente, número visible, versión, referencia de recibo y
   fecha de confirmación. Anote esas referencias.

Si se vuelve a enviar exactamente el mismo formulario con una clave nueva, la
aplicación lo rechaza con `409` para impedir un duplicado semántico. No repita
la solicitud: revise el expediente existente. Cambie el detalle solo cuando
se trate realmente de otra petición.

Compruebe el último recibo persistido desde una tercera terminal remota:

```bash
ssh root@cidonia.cloud \
  "docker exec vec-ct-o2-07-browser-20260904-tls psql -X -U postgres -d postgres -P pager=off -c \"SELECT numero_visible, expediente_ref, recibo_ref, version_expediente, confirmada_en FROM vec_contratacion_temporal.confirmacion_agregado_alta ORDER BY confirmada_en DESC LIMIT 1;\""
```

Los identificadores de PostgreSQL deben coincidir con los del recibo visible.

Después del recibo de Alta aparece el formulario **Análisis por Recursos
Humanos**. Para repetir el recorrido acreditado:

1. Seleccione **Sustitución**, **Categoría C2**, **Grupo C2** y **Necesidad
   temporal**.
2. Use las fechas `2027-01-01` y `2027-03-31`, jornada `10000` y la entrada
   **Retención de crédito sintética 001**.
3. Pulse **Registrar análisis** una sola vez. En la red debe aparecer un único
   `POST /api/vec/contratacion-temporal/analisis/registros` con estado `201`.
4. El segundo recibo debe mostrar la misma referencia de expediente, versión
   resultante `2`, una referencia `rec_ct_an_…` y fecha de confirmación.

Compruebe el último Análisis persistido:

```bash
ssh root@cidonia.cloud \
  "docker exec vec-ct-o2-07-browser-20260904-tls psql -X -U postgres -d postgres -P pager=off -c \"SELECT recibo_json->>'expediente_ref' AS expediente_ref, recibo_json->>'version_resultante' AS version_resultante, recibo_json->>'recibo_ref' AS recibo_ref, confirmada_en FROM vec_contratacion_temporal.confirmacion_operacion_analisis ORDER BY confirmada_en DESC LIMIT 1;\""
```

Los tres datos deben coincidir con el segundo recibo visible. Las cinco
modalidades disponibles —Sustitución, Vacante, Acumulación de tareas, Programa
y Relevo— pasan por el mismo servicio real; el recorrido acreditado usa
Sustitución.

Después del recibo de Análisis aparece **Decidir la vía de cobertura**:

1. Espere a que la propuesta muestre estado viable y vía recomendada
   **Bolsa vigente**. En la red debe aparecer un único
   `POST /api/vec/contratacion-temporal/cobertura/propuesta` con estado `200`.
2. Pulse **Confirmar vía de cobertura** una sola vez y acepte el diálogo de
   confirmación. Anote la `clave_idempotencia` del cuerpo de la petición en las
   herramientas de red; se utilizará solo para consultar el resultado.
3. Debe aparecer un único
   `POST /api/vec/contratacion-temporal/cobertura/decisiones` con estado `201`.
4. El tercer recibo debe mostrar el mismo expediente, versión resultante `3`,
   una referencia `recibo:ct:cobertura:…`, la referencia de la decisión y la
   fecha de confirmación.

Compruebe decisión, autorización y recibo persistidos sustituyendo las dos
referencias por las que muestra la pantalla:

```bash
expediente_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_EXPEDIENTE'
recibo_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_TERCER_RECIBO'
docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -v expediente_ref="$expediente_ref" \
  -v recibo_ref="$recibo_ref" -U postgres -d postgres -P pager=off <<'SQL'
BEGIN TRANSACTION READ ONLY;
SELECT d.expediente_ref, d.version_expediente, d.recibo_ref,
       d.decision_ref, d.via_elegida, d.persistida_en,
       count(a.decision_ref) AS concesiones_autorizacion
  FROM vec_contratacion_temporal.decision_cobertura_gobernada_durable d
  JOIN vec_autorizacion.decision_concedida_contexto_actor_v3 a
    ON a.decision_ref=d.decision_vec_ref
 WHERE d.expediente_ref=:'expediente_ref'
   AND d.recibo_ref=:'recibo_ref'
 GROUP BY d.decision_ref;
COMMIT;
SQL
```

Debe devolver una sola fila, versión `3`, vía `bolsa_vigente`, las mismas
referencias visibles y `concesiones_autorizacion = 1`.

Después del recibo de Cobertura aparece **Asignar expediente a la unidad
responsable**:

1. Compruebe que la unidad mostrada es `unidad:desarrollo:rrhh` y la persona
   responsable seleccionada es `persona:responsable-sintetica-001`.
2. Marque **He comprobado el expediente, la unidad y la referencia
   responsable**.
3. Pulse **Confirmar asignación** una sola vez y acepte el diálogo de
   confirmación.
4. En la red debe aparecer un único
   `POST /api/vec/contratacion-temporal/asignaciones` con estado `201`. Conserve
   su cuerpo exacto de cinco campos para la comprobación posterior al reinicio.
5. El cuarto recibo debe mostrar el mismo expediente, versión resultante `4`,
   una referencia de recibo y la fecha de confirmación.

Compruebe la Asignación persistida sustituyendo la referencia por la visible:

```bash
expediente_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_EXPEDIENTE'
docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -v expediente_ref="$expediente_ref" \
  -U postgres -d postgres -P pager=off <<'SQL'
BEGIN TRANSACTION READ ONLY;
SELECT r.expediente_ref,
       t.recibo_json->>'version_resultante' AS version_resultante,
       t.recibo_json->>'recibo_ref' AS recibo_ref,
       r.unidad_ref, r.responsable_ref, r.estado,
       count(*) OVER () AS terminales_del_expediente
  FROM vec_contratacion_temporal.reserva_asignacion r
  JOIN vec_contratacion_temporal.terminal_asignacion t
    USING (ambito_hmac)
 WHERE r.expediente_ref=:'expediente_ref';
COMMIT;
SQL
```

Debe devolver una sola fila: versión `4`, estado `confirmada`, la misma
referencia de recibo, la unidad y la persona responsable mostradas y
`terminales_del_expediente = 1`.

Después del recibo de Asignación aparece **Preparar informe jurídico**:

1. Compruebe que la pantalla muestra el mismo expediente y la versión asignada
   `4`.
2. Marque **He comprobado el expediente y entiendo que se generará un
   documento de desarrollo sin firma**.
3. Pulse **Confirmar y preparar informe** una sola vez y acepte el diálogo.
4. En la red debe aparecer un único
   `POST /api/vec/contratacion-temporal/informes-juridicos/preparaciones` con
   estado `201`. Conserve el cuerpo exacto de tres campos para comprobar la
   repetición después del reinicio.
5. La pantalla debe mostrar el quinto recibo, la versión resultante `5` y el
   contenido del documento encabezado por
   **DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA**.

Compruebe informe, documento y autorización persistidos:

```bash
expediente_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_EXPEDIENTE'
docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -v expediente_ref="$expediente_ref" \
  -U postgres -d postgres -P pager=off <<'SQL'
BEGIN TRANSACTION READ ONLY;
SELECT r.expediente_ref, r.estado,
       t.documento_ref, t.decision_ref, t.confirmada_en,
       a.version AS version_resultante,
       v.agregado_json->>'fase_actual' AS fase_actual,
       count(DISTINCT d.documento_ref) AS documentos,
       count(DISTINCT c.decision_ref) AS consumos_autorizacion
  FROM vec_contratacion_temporal.reserva_informe_juridico r
  JOIN vec_contratacion_temporal.terminal_informe_juridico t
    USING (ambito_hmac)
  JOIN vec_contratacion_temporal.documento_informe_juridico_desarrollo d
    USING (ambito_hmac)
  JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 c
    ON c.decision_ref=t.decision_ref
  JOIN vec_contratacion_temporal.expediente_integral_actual a
    USING (expediente_ref)
  JOIN vec_contratacion_temporal.expediente_version_integral v
    USING (expediente_ref, version)
 WHERE r.expediente_ref=:'expediente_ref'
 GROUP BY r.expediente_ref, r.estado, t.documento_ref, t.decision_ref,
          t.confirmada_en, a.version, v.agregado_json;
COMMIT;
SQL
```

Debe devolver una fila en estado `confirmada`, versión `5`, fase
`informe_juridico`, un documento y un consumo de autorización. Las referencias
de documento y recibo de la pantalla deben corresponder al mismo resultado.

Después del informe aparece **Registrar resultado de Fiscalización**. Esta
operación pertenece a Intervención, no a RRHH. En el perfil que vaya a enviarla
debe estar instalado `intervencion.p12`; una petición hecha con `cliente.p12`
queda denegada sin escritura.

En el entorno de desarrollo, si el navegador ya ha fijado el certificado de
RRHH para ese origen, abra un perfil temporal separado que contenga solo la CA
y `intervencion.p12`. Abra el portal normalmente y entre en
**Contratación temporal**. La pantalla separada de Intervención no contiene las
funciones de alta o análisis reservadas a RRHH. Pegue la referencia del
expediente mostrada en el recibo anterior, conserve la versión remitida `5` y
pulse **Abrir fiscalización**. Aparecerá el formulario real **Registrar
resultado de Fiscalización**. No abra la consola ni llame directamente a la
API.

1. Seleccione **Favorable**, **Favorable con observaciones** o
   **Desfavorable**. Los dos últimos exigen observaciones.
2. Para comprobar la vuelta a la unidad elija **Desfavorable**, escriba una
   observación inequívocamente sintética y pulse **Registrar resultado** una
   sola vez.
3. En la red debe aparecer un único
   `POST /api/vec/contratacion-temporal/fiscalizaciones/resultados` con estado
   `201`. Conserve su cuerpo exacto de cinco campos para el reinicio.
4. El sexto recibo visible debe indicar versión `6`, fase
   `subsanacion_unidad`, estado `incidencia`, una referencia de auditoría y el
   retorno a `unidad:desarrollo:rrhh` con
   `persona:responsable-sintetica-001`.

Compruebe la historia y el retorno persistidos:

```bash
expediente_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_EXPEDIENTE'
docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -v expediente_ref="$expediente_ref" \
  -U postgres -d postgres -P pager=off <<'SQL'
BEGIN TRANSACTION READ ONLY;
SELECT a.version, v.fase_clave, v.estado,
       r.resultado, r.estado AS estado_reserva,
       u.estado AS estado_retorno, u.unidad_ref, u.responsable_ref,
       t.auditoria_ref,
       count(DISTINCT c.decision_ref) AS consumos_autorizacion
  FROM vec_contratacion_temporal.expediente_integral_actual a
  JOIN vec_contratacion_temporal.expediente_version_integral v
    USING (expediente_ref, version)
  JOIN vec_contratacion_temporal.reserva_fiscalizacion r
    USING (expediente_ref)
  JOIN vec_contratacion_temporal.terminal_fiscalizacion t
    USING (ambito_hmac)
  JOIN vec_contratacion_temporal.retorno_fiscalizacion_unidad u
    USING (ambito_hmac)
  JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 c
    ON c.decision_ref=t.decision_ref
 WHERE a.expediente_ref=:'expediente_ref'
 GROUP BY a.version, v.fase_clave, v.estado, r.resultado, r.estado,
          u.estado, u.unidad_ref, u.responsable_ref, t.auditoria_ref;
COMMIT;
SQL
```

Debe devolver una fila: versión `6`, fase `subsanacion_unidad`, estado
`incidencia`, reserva `confirmada`, retorno `pendiente` y un único consumo de
autorización.

## 5. Demostrar que sobrevive al reinicio

1. En la primera terminal pulse `Ctrl-C`. Solo debe terminar VEC; no detenga
   PostgreSQL.
2. Sin cambiar las seis variables exportadas ni el worktree, arranque otra
   vez:

```bash
scripts/arrancar_vec_desarrollo.sh --puerto 8443 \
  --directorio-material /root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904
```

3. Repita las seis consultas SQL del apartado anterior. Deben devolver las
   mismas referencias después del reinicio.
4. Consulte el resultado de cobertura, sin repetir la decisión, con la clave
   anotada en las herramientas de red:

```bash
expediente_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_EXPEDIENTE'
clave_idempotencia='PEGUE_AQUI_LA_CLAVE_DE_LA_DECISION'
cuerpo_consulta=$(EXPEDIENTE_REF="$expediente_ref" \
  CLAVE_IDEMPOTENCIA="$clave_idempotencia" python3 -c \
  'import json, os; print(json.dumps({"expediente_ref": os.environ["EXPEDIENTE_REF"], "clave_idempotencia": os.environ["CLAVE_IDEMPOTENCIA"]}, separators=(",", ":")))')
curl --silent --show-error --fail-with-body \
  --cacert /root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/ca/ca.crt \
  --cert /root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/cliente.crt \
  --key /root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/cliente.key \
  -H 'Accept: application/json' \
  -H 'Content-Type: application/json; charset=utf-8' \
  --data-binary "$cuerpo_consulta" \
  https://localhost:8443/api/vec/contratacion-temporal/cobertura/resultados
```

La respuesta debe indicar `confirmado` e incluir exactamente el mismo tercer
recibo. Esta consulta no repite ni modifica la decisión.

5. En la consola del mismo navegador, repita el cuerpo exacto de cinco campos
   que conservó de Asignación, sin generar otra clave:

```javascript
const cuerpoAsignacion = PEGUE_AQUI_EL_OBJETO_JSON_DE_CINCO_CAMPOS;
await fetch("/api/vec/contratacion-temporal/asignaciones", {
  method: "POST",
  credentials: "same-origin",
  headers: {
    Accept: "application/json",
    "Content-Type": "application/json; charset=utf-8",
  },
  body: JSON.stringify(cuerpoAsignacion),
}).then(async (respuesta) => ({ estado: respuesta.status, cuerpo: await respuesta.json() }));
```

Debe responder otra vez `201`, con el mismo expediente, versión `4` y
referencia de recibo. Repita la consulta SQL de Asignación: debe seguir habiendo
un único terminal para ese expediente. El recibo interno contiene trazabilidad
adicional que no expone la API; compare semánticamente los campos públicos, no
el texto JSON literal. Esta repetición recupera el resultado y no crea una
segunda asignación.

6. En la consola del navegador repita también el cuerpo exacto de tres campos
   conservado del informe, sin generar otra clave:

```javascript
const cuerpoInforme = PEGUE_AQUI_EL_OBJETO_JSON_DE_TRES_CAMPOS;
await fetch("/api/vec/contratacion-temporal/informes-juridicos/preparaciones", {
  method: "POST",
  credentials: "same-origin",
  headers: {
    Accept: "application/json",
    "Content-Type": "application/json; charset=utf-8",
  },
  body: JSON.stringify(cuerpoInforme),
}).then(async (respuesta) => ({ estado: respuesta.status, cuerpo: await respuesta.json() }));
```

Debe responder otra vez `201` con el mismo informe, documento, recibo, huella
e instante de confirmación. La consulta SQL debe seguir mostrando un único
documento y un único consumo de autorización para ese informe.

7. En el perfil temporal de Intervención, repita el cuerpo exacto de cinco
   campos conservado de Fiscalización, sin generar otra clave:

```javascript
const cuerpoFiscalizacion = PEGUE_AQUI_EL_OBJETO_JSON_DE_CINCO_CAMPOS;
await fetch("/api/vec/contratacion-temporal/fiscalizaciones/resultados", {
  method: "POST",
  credentials: "same-origin",
  headers: {
    Accept: "application/json",
    "Content-Type": "application/json; charset=utf-8",
  },
  body: JSON.stringify(cuerpoFiscalizacion),
}).then(async (respuesta) => ({ estado: respuesta.status, cuerpo: await respuesta.json() }));
```

Debe responder otra vez `201` con exactamente el mismo recibo, auditoría,
evento, instante y retorno. La consulta SQL de Fiscalización debe seguir
mostrando una sola reserva, un solo terminal, un solo retorno y un solo consumo
de autorización.

La consulta protegida anterior recupera el resultado de cobertura. Alta y
Análisis todavía no tienen una vista que recargue sus recibos cerrados después
de desmontar la página; su persistencia se comprueba con las consultas SQL.

## 6. Llamamiento y aviso local — recorrido real recuperado tras reinicio

**Cinco pasos completos más este tramo del sexto recorrible.** Dirección
comprobó en navegador real selección `200` y comunicación local `201`.
Tras reiniciar VEC a las `00:46:33 UTC` del 5 de septiembre de 2026, abrió
un contexto nuevo de navegador y repitió las mismas claves: `200/200`,
mismo JSON de selección y mismo recibo de comunicación, ahora con estado
`replay_registrada_localmente`. Se conservaron las claves del servidor,
PostgreSQL y el directorio de material; no se depende de almacenamiento web.

Datos sintéticos ya registrados, para recuperar sin crear otro expediente:

- Expediente: `expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001`;
  versión de entrada `6`, fiscalización favorable.
- Clave de selección: `90d52c16-a63d-4ef1-bcf7-62c7c455f9aa`.
  Recibo: `recibo:pldjefhkpgifphgejphkpgcmkjlkphjgmfiedehlnpbphnoefikkcfmjancmdhce`;
  fecha conservada: `2026-09-05T00:45:25.244696Z`.
- Clave de comunicación: `d1b5428f-2188-4b8f-98b7-42f82ad88c2a`.
  Recibo: `recibo:68d32388-b423-4483-b7b1-2fcd091624a8`;
  fecha conservada: `2026-09-05T00:45:25.506363Z`.
  Intención de aviso: `outbox:2e3b64dc-ec3c-4076-bbdb-f036090aaa64`.

### Recorrerlo a mano

1. Arranque la candidata y abra el túnel con los comandos de los apartados
   2 y 3. Use el certificado de RRHH y abra
   **Contratación temporal → Nueva petición → Llamamiento y comunicación**.
   No registre otra solicitud ni repita la fiscalización.
2. Introduzca el expediente anterior, versión `6` y su clave de selección.
   Pulse **Revisar e iniciar llamamiento** y confirme. La ruta
   `POST /api/vec/contratacion-temporal/llamamientos/seleccion` devuelve
   `200` y el recibo anterior. No pulse **Preparar clave nueva**.
3. En **Registrar comunicación**, organización, llamamiento, versión `1` y
   recibo antecedente se rellenan desde la selección real. No invente ni
   cambie esas referencias. Introduzca la clave de comunicación anterior,
   pulse **Revisar y registrar comunicación** y confirme.
4. La repetición devuelve `200`, **Registro local recuperado · Sin entrega
   acreditada**, el mismo recibo, fecha e intención. La primera escritura
   devolvió `201`. La versión local resultante `2` no es la del expediente.
5. Para comprobar otro reinicio, termine solo VEC con `Ctrl-C` en su terminal,
   conserve las seis conexiones del apartado 2 y arranque desde la misma
   candidata, sin regenerar material ni detener PostgreSQL:

```bash
scripts/arrancar_vec_desarrollo.sh --puerto 8443 \
  --directorio-material /root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904
```

Abra un contexto nuevo de navegador con el certificado de RRHH y repita
los puntos 2–4 con exactamente los mismos datos. Compare los recibos y
fechas anteriores; no espere un recibo nuevo.

### Comprobar lo persistido, sin modificarlo

La revisión independiente contrastó en lectura real una orden original
(`2026-09-05T00:07:16.220132Z`), una propuesta, un llamamiento y un terminal
de selección. La recuperación conserva una historia y un evento pendiente,
con contador de exclusión `1 → 2`. Comunicación, historia y evento pendiente
también tienen una fila cada uno. El intento interrumpido se recuperó con su
misma clave, sin borrar la orden ni duplicar esos efectos.

```bash
expediente_ref='expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -v expediente_ref="$expediente_ref" \
  -U postgres -d postgres -P pager=off <<'SQL'
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT count(*) OVER () AS ejecuciones, s.situacion,
       s.recibo_json->>'recibo_ref' AS recibo_seleccion,
       s.recibo_json->>'llamamiento_ref' AS llamamiento,
       c.estado, c.recibo_json->>'ReciboRef' AS recibo_comunicacion,
       c.registrada_en, c.recibo_json->>'IntencionEnvioRef' AS intencion,
       (SELECT count(*) FROM vec_contratacion_temporal.historia_comunicacion_llamamiento_local h
         WHERE h.comunicacion_ref=c.comunicacion_ref) AS historia,
       (SELECT count(*) FROM vec_contratacion_temporal.outbox_comunicacion_llamamiento_local o
         WHERE o.comunicacion_ref=c.comunicacion_ref AND o.estado='pendiente') AS outbox_pendiente
  FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 s
  LEFT JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c
    ON c.seleccion_clave=s.clave_idempotencia
 WHERE s.solicitud_json->>'expediente_ref'=:'expediente_ref';
ROLLBACK;
SQL
```

Para el expediente indicado debe aparecer una ejecución `confirmada`, los
recibos anteriores, comunicación `registrada_localmente`, una historia y un
evento pendiente de comunicación. Ese evento por sí solo no acredita el
fichero. Se verificó también el aviso: **803 bytes**, directorio `0700`,
archivo `0600`, SHA256
`d0173929c18b437d44b84fcab632ef60659b3246bb98586a09cc0fc506a48c17`.

En la terminal remota, compruebe tamaño, permisos y huella sin mostrar contenido:

```bash
material_vec=/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904
intencion_envio_ref='outbox:2e3b64dc-ec3c-4076-bbdb-f036090aaa64'
huella_intencion=$(printf '%s' "$intencion_envio_ref" | sha256sum | cut -d ' ' -f 1)
aviso_local="$material_vec/comunicaciones/aviso-$huella_intencion.json"
stat -c '%a %s %n' "$material_vec/comunicaciones" "$aviso_local"
sha256sum "$aviso_local"
```

**Es selección más aviso LOCAL persistido, no correo enviado, entrega,
aceptación ni apertura de plazo.** La fuente de Bolsa es sintética firmada;
los recibos y efectos descritos sí están guardados de verdad.

## Resultado y siguiente paso funcional

De los ocho pasos solicitados por Recursos Humanos, este corte permite recorrer
manualmente 1, **Solicitud**, 2, **Análisis**, 3, **Bolsa**, 4,
**Asignación** y 5, **Informe jurídico y Fiscalización**. El contador funcional
queda en **5 de 8**. Fiscalización acepta los tres resultados reales y el
desfavorable devuelve automáticamente el expediente a la Unidad conservando
el histórico completo.

El apartado 6 añade **selección y aviso local recorribles**, con los mismos
recibos recuperados tras reiniciar. El resultado es **5 pasos completos más
un tramo del sexto**, no el 100 % del llamamiento corporativo. Faltan envío y
entrega reales, aceptación o renuncia y gestión del plazo; no están
acreditados por un aviso local. Después quedan 7, **Nombramiento**, con sus
seis documentos, incluida la Diligencia, y 8, **Incorporación, GINPIX y
Seguimiento**.

La evidencia corresponde a la línea de trabajo indicada. Es un recorrido de
desarrollo: ni su integración ni su publicación en GitHub autorizan producción
o sustituyen la aceptación de Recursos Humanos.

## Apéndice: recreación segura de la instancia aislada

No ejecute este apéndice durante la repetición pendiente de Dirección. Los
recursos actuales no llevan etiqueta de propiedad porque preceden a esta guía;
el script se negará a borrarlos. Su retirada inicial exige inspección y
confirmación humana explícita de los nombres, digest, volumen, montaje, puerto,
ausencia de clientes y ausencia de un proceso `vec-server`.

Después de esa retirada inicial autorizada, el siguiente bloque crea recursos
propios y solo reemplaza recursos que conserven exactamente su etiqueta:

```bash
set -Eeuo pipefail

fuente=vec-ct-o2-07-e2e-20260904-tls
volumen_fuente=vec-ct-o2-07-e2e-20260904-a-data
destino=vec-ct-o2-07-browser-20260904-tls
volumen_destino=vec-ct-o2-07-browser-20260904-data
imagen='postgres@sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382'
propietario='o2-07-browser-guide-v1'
clave_etiqueta='es.dipgra.vec.propietario'

if pgrep -af '/vec-server'; then
  printf '%s\n' 'ERROR: detenga VEC antes de clonar PostgreSQL' >&2
  exit 1
fi

test "$(docker inspect "$fuente" --format '{{.State.Status}}')" = running
test "$(docker inspect "$fuente" --format '{{.Config.Image}}')" = "$imagen"
test "$(docker inspect "$fuente" --format '{{range .Mounts}}{{if eq .Destination "/var/lib/postgresql"}}{{.Name}}{{end}}{{end}}')" = "$volumen_fuente"
test "$(docker inspect "$fuente" --format '{{(index (index .HostConfig.PortBindings "5432/tcp") 0).HostIp}}:{{(index (index .HostConfig.PortBindings "5432/tcp") 0).HostPort}}')" = '127.0.0.1:55432'
docker image inspect "$imagen" >/dev/null

estado() {
  docker exec "$1" psql -X -U postgres -d postgres -Atqc "SELECT concat_ws('/',
    (SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),
    (SELECT COALESCE(max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),
    (SELECT revision FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id),
    (SELECT secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id));"
}

lector() {
  docker exec "$1" psql -X -U postgres -d postgres -Atqc "SELECT count(*)
    FROM pg_catalog.pg_shdepend
    WHERE refclassid='pg_catalog.pg_authid'::pg_catalog.regclass
      AND refobjid='vec_contratacion_temporal_lector_resultado_cobertura'::pg_catalog.regrole;"
}

test "$(docker exec "$fuente" psql -X -U postgres -d postgres -Atqc "SELECT count(*) FROM pg_stat_activity WHERE backend_type='client backend' AND pid <> pg_backend_pid();")" = 0
test "$(estado "$fuente")" = '23/23/23/13'
test "$(lector "$fuente")" = 3

if docker container inspect "$destino" >/dev/null 2>&1; then
  test "$(docker inspect "$destino" --format "{{index .Config.Labels \"$clave_etiqueta\"}}")" = "$propietario" || {
    printf '%s\n' 'ERROR: contenedor existente sin propiedad exacta; no se toca' >&2
    exit 1
  }
  docker stop "$destino"
  docker rm "$destino"
fi

if docker volume inspect "$volumen_destino" >/dev/null 2>&1; then
  test "$(docker volume inspect "$volumen_destino" --format "{{index .Labels \"$clave_etiqueta\"}}")" = "$propietario" || {
    printf '%s\n' 'ERROR: volumen existente sin propiedad exacta; no se toca' >&2
    exit 1
  }
  docker volume rm "$volumen_destino"
fi

docker volume create --label "$clave_etiqueta=$propietario" "$volumen_destino"
fuente_detenida=false
restaurar_fuente() {
  if test "$fuente_detenida" = true; then
    docker start "$fuente" >/dev/null
  fi
}
trap restaurar_fuente EXIT

docker stop "$fuente"
fuente_detenida=true
docker run --rm --pull=never --network none --read-only \
  --label "$clave_etiqueta=$propietario" \
  --mount "type=volume,src=$volumen_fuente,dst=/origen,readonly" \
  --mount "type=volume,src=$volumen_destino,dst=/destino" \
  --entrypoint /bin/sh "$imagen" -ceu '
    test -f /origen/PG_VERSION
    test ! -e /destino/PG_VERSION
    cp -a /origen/. /destino/
    sync
  '
docker start "$fuente" >/dev/null
fuente_detenida=false
trap - EXIT

test "$(estado "$fuente")" = '23/23/23/13'
test "$(lector "$fuente")" = 3

docker run -d --name "$destino" --pull=never --restart=no \
  --label "$clave_etiqueta=$propietario" \
  --publish 127.0.0.1:55433:5432 \
  --mount "type=volume,src=$volumen_destino,dst=/var/lib/postgresql" \
  "$imagen"

for intento in $(seq 1 100); do
  if docker exec "$destino" pg_isready -U postgres -d postgres >/dev/null 2>&1; then
    break
  fi
  test "$intento" -lt 100
  sleep 0.1
done

docker exec -i "$destino" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
DO $provision$
BEGIN
  IF NOT EXISTS (
    SELECT FROM pg_catalog.pg_roles
    WHERE rolname='vec_autorizacion_o207_registro'
  ) THEN
    CREATE ROLE vec_autorizacion_o207_registro LOGIN;
  END IF;
END
$provision$;
GRANT vec_autorizacion_registro TO vec_autorizacion_o207_registro;
GRANT vec_contratacion_temporal_gobernador TO vec_ad3_o207_gobierno;
SQL

test "$(docker exec "$destino" psql -X -U postgres -d postgres -Atqc "SELECT login.rolcanlogin::text||'|'||string_agg(grupo.rolname,',' ORDER BY grupo.rolname) FROM pg_roles login JOIN pg_auth_members miembro ON miembro.member=login.oid JOIN pg_roles grupo ON grupo.oid=miembro.roleid WHERE login.rolname='vec_autorizacion_o207_registro' GROUP BY login.rolcanlogin;")" = 'true|vec_autorizacion_registro'
test "$(docker exec "$destino" psql -X -U postgres -d postgres -Atqc "SELECT login.rolcanlogin::text||'|'||string_agg(grupo.rolname,',' ORDER BY grupo.rolname) FROM pg_roles login JOIN pg_auth_members miembro ON miembro.member=login.oid JOIN pg_roles grupo ON grupo.oid=miembro.roleid WHERE login.rolname='vec_ad3_o207_gobierno' AND grupo.rolname='vec_contratacion_temporal_gobernador' GROUP BY login.rolcanlogin;")" = 'true|vec_contratacion_temporal_gobernador'

docker exec -i "$destino" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
BEGIN;
SET LOCAL session_replication_role = replica;
TRUNCATE TABLE
  vec_autorizacion_atestada_v3.clave_capacidad_version,
  vec_autorizacion_atestada_v3.puntero_clave_emision,
  vec_autorizacion_atestada_v3.revocacion_clave_capacidad,
  vec_autorizacion_atestada_v3.configuracion_confianza_version,
  vec_autorizacion_atestada_v3.raiz_confianza_version,
  vec_autorizacion_atestada_v3.configuracion_raiz,
  vec_autorizacion_atestada_v3.puntero_configuracion_actual,
  vec_autorizacion_atestada_v3.revocacion_configuracion,
  vec_autorizacion_atestada_v3.revocacion_raiz,
  vec_autorizacion_atestada_v3.atestacion_decision_v3,
  vec_autorizacion_atestada_v3.consumo_decision_v3,
  vec_autorizacion_atestada_v3.auditoria_consumo_v3;
UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno
SET revision=0, configuracion_secuencia_minima=0, raiz_version_minima=0,
    actualizada_en=clock_timestamp()
WHERE control_id;
UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
SET secuencia=0, cabeza_sha256=pg_catalog.repeat('0',64),
    actualizada_en=clock_timestamp()
WHERE control_id;
COMMIT;
SQL

test "$(estado "$destino")" = '0/0/0/0'
test "$(lector "$destino")" = 3
test "$(estado "$fuente")" = '23/23/23/13'
test "$(lector "$fuente")" = 3
```

El bloque no elimina datos de la fuente. Detiene la fuente solo durante la copia
en frío, la vuelve a arrancar y reinicia únicamente el gobierno de autorización
en el clon. La aplicación inicializa ese gobierno al arrancar contra `55433`.
