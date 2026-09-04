# Recorrido manual de contratación temporal O2-07

Esta guía permite probar los cuatro primeros pasos del flujo de Recursos Humanos con la
aplicación real: navegador con certificado de cliente → API interna →
autorización de servidor → PostgreSQL → recibo. No usa el adaptador DEMO.

## Entorno preservado para el recorrido manual

La evidencia sintética del 4 de septiembre de 2026 sigue disponible en el
servidor. No es necesario recrearla ni borrarla para recorrer otro expediente:

- repositorio: `/srv/fabrica/proyectos/VEC_Diputacion_app`;
- producto: `.worktrees/ct-producto-ligero-20260821`;
- PostgreSQL exclusivo del navegador: contenedor
  `vec-ct-o2-07-browser-20260904-tls`, puerto local remoto `55433`;
- PostgreSQL reservado para pruebas Go: contenedor
  `vec-ct-o2-07-e2e-20260904-tls`, puerto local remoto `55432`;
- material HTTPS y certificado de cliente de VEC:
  `/root/.local/state/vec-diputacion/desarrollo`.

Regla obligatoria: el servidor usado por el navegador solo apunta a `55433`.
Las pruebas Go solo apuntan a `55432`. Nunca se usan a la vez contra la misma
instancia.

## 1. Comprobar la instancia aislada

En una terminal del servidor:

```bash
ssh root@cidonia.cloud
cd /srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/ct-producto-ligero-20260821
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

El árbol debe estar limpio. El contenedor debe estar `running`, usar la imagen
fijada por digest, montar `vec-ct-o2-07-browser-20260904-data` en
`/var/lib/postgresql` y publicar únicamente `127.0.0.1:55433`. Tras la evidencia
automatizada, los tres recuentos deben ser iguales y mayores que cero; el valor
crece con cada solicitud sintética nueva.
La última consulta debe devolver exclusivamente
`true|vec_autorizacion_registro` y la siguiente
`true|vec_contratacion_temporal_gobernador`.

Antes de arrancar VEC, compruebe que PostgreSQL conserva exactamente las
definiciones de las dos migraciones de Asignación incluidas en el producto. La
comprobación es de solo lectura y se detiene ante cualquier diferencia:

```bash
set -euo pipefail

fuente_autorizacion=deploy/postgresql/autorizacion_atestada_v3/migraciones/000008_consumidor_asignacion_v3_atestada.up.sql
fuente_asignacion=deploy/postgresql/contratacion_temporal/migraciones/000050_asignacion_durable_v3_v4.up.sql

test "$(sha256sum "$fuente_autorizacion" | awk '{print $1}')" = \
  42f7dfef32464f1a0ce2f3bcb9af035d800b7ceb302917d0652161408488451c
test "$(sha256sum "$fuente_asignacion" | awk '{print $1}')" = \
  cba96bd9a281e583b9f14edc661d3fc3b26b2a4898515c907c4d5f56cdf89e89

huella_funcion() {
  docker exec vec-ct-o2-07-browser-20260904-tls \
    psql -X -U postgres -d postgres -Atqc \
    "SELECT pg_catalog.encode(public.digest(pg_catalog.convert_to(pg_catalog.pg_get_functiondef('$1'::regprocedure),'UTF8'),'sha256'),'hex');"
}

test "$(huella_funcion 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  e97203f7b0ae8efad70b7c1168e9c35b109157d3b191d46521960c10e69e67cb
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

printf '%s\n' 'OK: migraciones y funciones de Asignación coinciden'
```

Si alguna comparación falla, no abra el navegador ni reaplique migraciones a
ciegas: el código y la instancia no están alineados.

## 2. Arrancar VEC con PostgreSQL real

En la misma terminal remota, desde el worktree de producto:

```bash
directorio_pg=$(mktemp -d /tmp/vec-o2-07-pg.XXXXXX)
docker cp \
  vec-ct-o2-07-browser-20260904-tls:/var/lib/postgresql/18/docker/o207-ca.crt \
  "$directorio_pg/ca.crt"
chmod 600 "$directorio_pg/ca.crt"

docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
GRANT vec_contratacion_temporal_gobernador TO vec_ad3_o207_gobierno;
SQL
docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres \
  < deploy/postgresql/contratacion_temporal/migraciones/000035a_compatibilidad_barreras_decision_cobertura.up.sql

export VEC_CT_DATABASE_URL="postgresql://vec_ct_o207_runtime@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_GOBIERNO_DATABASE_URL="postgresql://vec_ad3_o207_gobierno@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_CONFIRMADOR_DATABASE_URL="postgresql://vec_ct_o207_confirmador@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_LECTOR_RESULTADO_DATABASE_URL="postgresql://vec_ct_o207_lector@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL="postgresql://vec_autorizacion_o207_registro@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"

scripts/arrancar_vec_desarrollo.sh --puerto 8443
```

El lanzador genera o valida las credenciales fuera de Git, construye un binario
temporal y escucha solo en `127.0.0.1:8443` con TLS 1.3 y certificado de cliente
obligatorio. Se deja en primer plano. No es un despliegue.

## 3. Abrir el túnel y preparar el certificado del navegador

En una segunda terminal del equipo desde el que se abrirá el navegador:

```bash
directorio_navegador=$(mktemp -d /tmp/vec-o2-07-navegador.XXXXXX)
chmod 700 "$directorio_navegador"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo/ca/ca.crt \
  "$directorio_navegador/ca.crt"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo/mtls/cliente.p12 \
  "$directorio_navegador/cliente.p12"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo/mtls/cliente.p12.password \
  "$directorio_navegador/cliente.p12.password"
chmod 600 "$directorio_navegador"/*
ssh -N -L 18443:127.0.0.1:8443 root@cidonia.cloud
```

No muestre ni copie la contraseña en la consola. Impórtela directamente desde
`cliente.p12.password` cuando el navegador la pida.

1. Antes de importar el certificado personal, abra
   `https://localhost:18443/portal-empleado/` en un perfil temporal: el acceso
   debe quedar bloqueado por falta de certificado de cliente. Eso demuestra la
   denegación predeterminada.
2. Importe `ca.crt` como autoridad de confianza para sitios web.
3. Importe `cliente.p12` como certificado personal usando su fichero de
   contraseña, cierre el perfil anterior y abra uno nuevo.
4. Acceda a `https://localhost:18443/portal-empleado/` y elija
   **Contratación temporal** → **Nueva petición**.

Al acabar, cierre el perfil temporal. Elimine el directorio temporal mediante
el mecanismo de papelera o borrado seguro aprobado en su equipo; contiene una
credencial de desarrollo.

## 4. Registrar una solicitud, analizarla, decidir la cobertura, asignarla y ver los cuatro recibos

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

## 5. Demostrar que sobrevive al reinicio

1. En la primera terminal pulse `Ctrl-C`. Solo debe terminar VEC; no detenga
   PostgreSQL.
2. Sin cambiar las cinco variables exportadas ni el worktree, arranque otra
   vez:

```bash
scripts/arrancar_vec_desarrollo.sh --puerto 8443
```

3. Repita las cuatro consultas SQL del apartado anterior. Deben devolver las mismas
   referencias después del reinicio.
4. Consulte el resultado de cobertura, sin repetir la decisión, con la clave
   anotada en las herramientas de red:

```bash
expediente_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_EXPEDIENTE'
clave_idempotencia='PEGUE_AQUI_LA_CLAVE_DE_LA_DECISION'
cuerpo_consulta=$(EXPEDIENTE_REF="$expediente_ref" \
  CLAVE_IDEMPOTENCIA="$clave_idempotencia" python3 -c \
  'import json, os; print(json.dumps({"expediente_ref": os.environ["EXPEDIENTE_REF"], "clave_idempotencia": os.environ["CLAVE_IDEMPOTENCIA"]}, separators=(",", ":")))')
curl --silent --show-error --fail-with-body \
  --cacert /root/.local/state/vec-diputacion/desarrollo/ca/ca.crt \
  --cert /root/.local/state/vec-diputacion/desarrollo/mtls/cliente.crt \
  --key /root/.local/state/vec-diputacion/desarrollo/mtls/cliente.key \
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

La consulta protegida anterior recupera el resultado de cobertura. Alta y
Análisis todavía no tienen una vista que recargue sus recibos cerrados después
de desmontar la página; su persistencia se comprueba con las consultas SQL.

## Resultado y siguiente paso funcional

De los ocho pasos solicitados por Recursos Humanos, este corte permite recorrer
manualmente 1, **Solicitud**, 2, **Análisis**, 3, **Bolsa**, y 4,
**Asignación**. La secuencia restante es 5, **Informe jurídico y
Fiscalización**, 6, **Llamamiento**, 7, **Nombramiento** con sus seis
documentos, incluida la Diligencia, y 8, **Incorporación, GINPIX y
Seguimiento**.

El siguiente corte es **Fiscalización**: generar el informe jurídico, registrar
un resultado favorable, favorable con observaciones o desfavorable y, en este
último caso, devolver automáticamente el expediente a la Unidad conservando el
histórico completo. No se simula Fiscalización ni se abre Llamamiento en esta
guía.

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
