#!/usr/bin/env bash
# Infraestructura desechable del recorrido RRHH. Se carga con `source`.
# Nunca consulta el PostgreSQL principal ni archivos de configuración privados.

rrhh_e2e_error() { printf 'RRHH E2E PG: %s\n' "$*" >&2; return 1; }

rrhh_e2e_diagnosticar_pg() {
    local nombre=${VEC_E2E_PG_CONTAINER:-}
    [[ -n $nombre ]] || return 0
    printf 'RRHH E2E PG: estado Docker: ' >&2
    docker inspect --format '{{.State.Status}} exit={{.State.ExitCode}}' "$nombre" >&2 2>/dev/null ||
        printf 'contenedor no inspeccionable\n' >&2
    tail -n 30 "${VEC_E2E_WORK:?}/pg_docker.log" 2>/dev/null |
        sed -E 's/([Pp][Aa][Ss][Ss][Ww][Oo][Rr][Dd]|[Tt][Oo][Kk][Ee][Nn]|[Ss][Ee][Cc][Rr][Ee][Tt])=[^[:space:]]+/\1=[redactado]/g' |
        grep -Ei 'error|fatal|permission denied|could not|initdb|database system is ready|no space|invalid|unsupported|refus' |
        tail -n 12 >&2 || true
}

rrhh_e2e_iniciar_pg() {
    [[ ${VEC_E2E_ROOT:-} == /* && -d ${VEC_E2E_ROOT:-} ]] ||
        { rrhh_e2e_error 'VEC_E2E_ROOT debe ser un checkout absoluto'; return 1; }
    [[ ${VEC_E2E_WORK:-} == /dev/shm/vec-e2e-* && -d ${VEC_E2E_WORK:-} && ! -L ${VEC_E2E_WORK:-} ]] ||
        { rrhh_e2e_error 'VEC_E2E_WORK debe ser un directorio real /dev/shm/vec-e2e-*'; return 1; }
    { command -v docker >/dev/null && command -v openssl >/dev/null; } ||
        { rrhh_e2e_error 'se requieren Docker y openssl'; return 1; }
    [[ -z ${VEC_E2E_PG_CONTAINER:-} ]] ||
        { rrhh_e2e_error 'ya existe un contenedor asignado a este recorrido'; return 1; }
    local imagen=${VEC_E2E_PG_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
    local nombre clave puerto intento publicar mascara
    nombre="vec-rrhh-e2e-${BASHPID}-${RANDOM}"
    publicar='127.0.0.1::5432'
    if [[ -n ${VEC_E2E_PG_PORT:-} ]]; then
        [[ $VEC_E2E_PG_PORT =~ ^[0-9]+$ && $VEC_E2E_PG_PORT -ge 1024 && $VEC_E2E_PG_PORT -le 65535 ]] ||
            { rrhh_e2e_error 'VEC_E2E_PG_PORT inválido'; return 1; }
        publicar="127.0.0.1:${VEC_E2E_PG_PORT}:5432"
    fi
    clave=$(openssl rand -hex 32) || return 1
    mascara=$(umask)
    umask 077
    docker run --rm --pull never --name "$nombre" \
        --publish "$publicar" \
        --shm-size 2g --env PGDATA=/dev/shm/vec-rrhh-e2e-pgdata \
        --mount "type=bind,src=$VEC_E2E_ROOT/deploy/postgresql,dst=/repo/deploy/postgresql,readonly" \
        --env "POSTGRES_PASSWORD=$clave" --env POSTGRES_INITDB_ARGS='--auth-host=scram-sha-256' \
        "$imagen" >"$VEC_E2E_WORK/pg_docker.log" 2>&1 &
    VEC_E2E_PG_RUN_PID=$!
    umask "$mascara"
    VEC_E2E_PG_CONTAINER=$nombre
    puerto=
    for ((intento=0; intento<30; intento++)); do
        puerto=$(docker port "$nombre" 5432/tcp 2>/dev/null) && break
        kill -0 "$VEC_E2E_PG_RUN_PID" 2>/dev/null || break
        sleep 1
    done
    if [[ -z $puerto ]]; then
        rrhh_e2e_diagnosticar_pg
        rrhh_e2e_detener_pg
        rrhh_e2e_error 'Docker no pudo iniciar PostgreSQL 18 aislado'
        return 1
    fi
    puerto=${puerto##*:}
    [[ $puerto =~ ^[0-9]+$ && $puerto -gt 0 ]] ||
        { rrhh_e2e_detener_pg; rrhh_e2e_error 'Docker no asignó un puerto loopback'; return 1; }
    if [[ -n ${VEC_E2E_PG_PORT:-} && ${VEC_E2E_PG_PORT} != "$puerto" ]]; then
        rrhh_e2e_detener_pg
        rrhh_e2e_error 'VEC_E2E_PG_PORT no coincide con el puerto aleatorio de Docker'
        return 1
    fi
    VEC_E2E_PG_PORT=$puerto
    VEC_E2E_PG_ADMIN_DSN="postgresql://postgres:$clave@127.0.0.1:$puerto/postgres?sslmode=disable"
    export VEC_E2E_PG_CONTAINER VEC_E2E_PG_PORT VEC_E2E_PG_ADMIN_DSN
    for ((intento=0; intento<90; intento++)); do
        if docker exec "$nombre" pg_isready -q -U postgres -d postgres 2>/dev/null; then
            return 0
        fi
        sleep 1
    done
    rrhh_e2e_diagnosticar_pg
    rrhh_e2e_detener_pg
    rrhh_e2e_error 'PostgreSQL 18 no quedó listo'
}

rrhh_e2e_detener_pg() {
    local nombre=${VEC_E2E_PG_CONTAINER:-}
    [[ -z $nombre ]] || docker rm -f -- "$nombre" >/dev/null 2>&1 || true
    [[ -z ${VEC_E2E_PG_RUN_PID:-} ]] || wait "$VEC_E2E_PG_RUN_PID" 2>/dev/null || true
    unset VEC_E2E_PG_CONTAINER VEC_E2E_PG_ADMIN_DSN VEC_E2E_PG_PORT VEC_E2E_PG_RUN_PID
    # PGDATA vive exclusivamente en /dev/shm del contenedor; --rm lo elimina.
    # El runner retira después el log acotado en VEC_E2E_WORK.
}

rrhh_e2e_sql() {
    local ruta=$1
    [[ $ruta == deploy/postgresql/* && $ruta != *..* && -f ${VEC_E2E_ROOT:?}/$ruta ]] ||
        { rrhh_e2e_error "archivo SQL ausente o fuera del checkout: $ruta"; return 1; }
    if ! docker exec -i "${VEC_E2E_PG_CONTAINER:?}" psql -X -q -v ON_ERROR_STOP=1 \
        -U postgres -d postgres -f "/repo/$ruta" >/dev/null; then
        rrhh_e2e_error "falló la precondición o instalación de $ruta"
        return 1
    fi
}

rrhh_e2e_unica_migracion() {
    local familia=$1 numero=$2 fichero ruta
    local -a encontrados=()
    shopt -s nullglob
    encontrados=("$VEC_E2E_ROOT/deploy/postgresql/$familia/migraciones/${numero}_"*.up.sql)
    shopt -u nullglob
    ((${#encontrados[@]} <= 1)) || { rrhh_e2e_error "$familia $numero es ambiguo"; return 1; }
    ((${#encontrados[@]} == 0)) && return 0
    fichero=${encontrados[0]}
    ruta=${fichero#"$VEC_E2E_ROOT/"}
    rrhh_e2e_sql "$ruta"
}

rrhh_e2e_cola_sql() {
    local ruta=${1#"$VEC_E2E_ROOT/"}
    local modulo=$2
    # Todos los UP canónicos de esta cola son transacciones. Un error cierra
    # psql y revierte ese intento; se conserva el texto del último bloqueo.
    docker exec -i "${VEC_E2E_PG_CONTAINER:?}" psql -X -q -v ON_ERROR_STOP=1 \
        -U postgres -d postgres -f "/repo/$ruta" > /dev/null \
        2> "$VEC_E2E_WORK/pg_error_$modulo"
}

rrhh_e2e_instalar_sql() {
    [[ -n ${VEC_E2E_PG_CONTAINER:-} && -n ${VEC_E2E_PG_ADMIN_DSN:-} ]] ||
        { rrhh_e2e_error 'iniciar PostgreSQL antes de instalar SQL'; return 1; }
    local n codigo ruta progreso modulo
    local ct_indice=0 ad3_indice=0 bolsa_indice=0 ctauth_indice=0 contexto_indice=0
    local bolsa_registrador=0 bolsa_calculador=0 ct_registrador=0
    local -a ct=() ad3=() bolsa=() ctauth=() contexto=()
    # Una instancia recién creada no puede tener B49 ni Public3 instaladas.
    # Rechazamos cualquier esquema previo, también una imagen o volumen ajenos.
    if [[ $(docker exec "$VEC_E2E_PG_CONTAINER" psql -X -At -v ON_ERROR_STOP=1 \
        -U postgres -d postgres -c "SELECT to_regnamespace('vec_bolsa_llamamientos') IS NULL AND to_regnamespace('vec_bolsa_publica_datos') IS NULL AND to_regnamespace('vec_bolsa_publica_lectura') IS NULL AND to_regnamespace('vec_bolsa_publica_publicacion') IS NULL" 2>/dev/null) != t ]]; then
        rrhh_e2e_error 'la base no está vacía: B49/Public3 u otra historia no admitida'
        return 1
    fi
    if ! docker exec -i "$VEC_E2E_PG_CONTAINER" psql -X -q -v ON_ERROR_STOP=1 \
        -U postgres -d postgres >/dev/null <<'SQL'; then
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
        rrhh_e2e_error 'falló el cierre inicial de PUBLIC'
        return 1
    fi
    rrhh_e2e_sql deploy/postgresql/contexto_actor_v1/roles_up.sql || return 1
    rrhh_e2e_sql deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql || return 1
    rrhh_e2e_sql deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql || return 1
    # En una base nueva, 000004a exige la preimagen exacta 1–3 + selector.
    # Otras familias alteran ese manifiesto simbólico; se instalan después.
    rrhh_e2e_sql deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_up.sql || return 1
    rrhh_e2e_sql deploy/postgresql/contexto_actor_v1/migraciones/000003_organizacion_corporativa_v1.up.sql || return 1
    rrhh_e2e_sql deploy/postgresql/contexto_actor_v1/migraciones/000004a_vinculo_corporativo_rrhh_v1.up.sql || return 1
    rrhh_e2e_sql deploy/postgresql/contexto_actor_v1/roles_historicos_up.sql || return 1
    rrhh_e2e_sql deploy/postgresql/autorizacion/roles_up.sql || return 1
    rrhh_e2e_sql deploy/postgresql/autorizacion/roles_v2_up.sql || return 1
    if ! docker exec -i "$VEC_E2E_PG_CONTAINER" psql -X -q -v ON_ERROR_STOP=1 \
        -U postgres -d postgres >/dev/null <<'SQL'; then
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL
        rrhh_e2e_error 'falló la preparación de pgcrypto'
        return 1
    fi
    rrhh_e2e_sql deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql || return 1
    rrhh_e2e_sql deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql || return 1
    for codigo in 000003 000004 000005 000006 000007; do
        rrhh_e2e_unica_migracion autorizacion "$codigo" || return 1
    done
    rrhh_e2e_sql deploy/postgresql/contratacion_temporal/roles_up.sql || return 1
    rrhh_e2e_sql deploy/postgresql/bolsa_llamamientos/roles_up.sql || return 1
    rrhh_e2e_sql deploy/postgresql/contratacion_temporal/migraciones_autorizacion/000001_revalidacion_analisis_v3.up.sql || return 1
    rrhh_e2e_sql deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql || return 1
    rrhh_e2e_unica_migracion contratacion_temporal 000001 || return 1
    rrhh_e2e_unica_migracion contratacion_temporal 000002 || return 1
    rrhh_e2e_sql deploy/postgresql/autorizacion_atestada_v3/roles_up.sql || return 1
    rrhh_e2e_unica_migracion autorizacion_atestada_v3 000001 || return 1
    rrhh_e2e_unica_migracion autorizacion_atestada_v3 000002 || return 1
    shopt -s nullglob
    ct=("$VEC_E2E_ROOT"/deploy/postgresql/contratacion_temporal/migraciones/*.up.sql)
    # 000004a sustituye a 000004; jamás se aplican ambas en una base nueva.
    contexto=(
        "$VEC_E2E_ROOT/deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql"
        "$VEC_E2E_ROOT/deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql"
        "$VEC_E2E_ROOT/deploy/postgresql/contexto_actor_v1/migraciones/000003_organizacion_corporativa_v1.up.sql"
        "$VEC_E2E_ROOT/deploy/postgresql/contexto_actor_v1/migraciones/000004a_vinculo_corporativo_rrhh_v1.up.sql"
        "$VEC_E2E_ROOT/deploy/postgresql/contexto_actor_v1/migraciones/000005_lectura_contexto_historico_v2.up.sql"
        "$VEC_E2E_ROOT/deploy/postgresql/contexto_actor_v1/migraciones/000006_vinculos_efectivos_temporales.up.sql"
        "$VEC_E2E_ROOT/deploy/postgresql/contexto_actor_v1/migraciones/000007_alcance_proyecciones_empleado.up.sql"
        "$VEC_E2E_ROOT/deploy/postgresql/contexto_actor_v1/migraciones/000008_acreditacion_persona_tercero.up.sql"
        "$VEC_E2E_ROOT/deploy/postgresql/contexto_actor_v1/migraciones/000009_revalidacion_vinculo_corporativo_rrhh_v1.up.sql"
    )
    ctauth=("$VEC_E2E_ROOT"/deploy/postgresql/contratacion_temporal/migraciones_autorizacion/*.up.sql)
    ad3=("$VEC_E2E_ROOT"/deploy/postgresql/autorizacion_atestada_v3/migraciones/*.up.sql)
    bolsa=("$VEC_E2E_ROOT"/deploy/postgresql/bolsa_llamamientos/migraciones/*.up.sql)
    shopt -u nullglob
    ((${#ct[@]} > 2 && ${#contexto[@]} > 2 && ${#ctauth[@]} > 1 && ${#ad3[@]} > 2 && ${#bolsa[@]} > 1)) ||
        { rrhh_e2e_error 'faltan migraciones CT, AD3 o Bolsa en el checkout'; return 1; }
    # Ya se instalaron CT1/2 y AD3-1/2. Se conserva orden de cada módulo.
    ct_indice=2; contexto_indice=4; ad3_indice=2; ctauth_indice=1
    : > "$VEC_E2E_WORK/pg_sql_aplicado"
    while ((ct_indice < ${#ct[@]} || contexto_indice < ${#contexto[@]} || ctauth_indice < ${#ctauth[@]} || ad3_indice < ${#ad3[@]} || bolsa_indice < ${#bolsa[@]})); do
        progreso=0
        for modulo in contexto ad3 ctauth bolsa ct; do
            case $modulo in
                ad3) n=$ad3_indice; ((${#ad3[@]} > n)) || continue; ruta=${ad3[n]} ;;
                contexto) n=$contexto_indice; ((${#contexto[@]} > n)) || continue; ruta=${contexto[n]} ;;
                ctauth) n=$ctauth_indice; ((${#ctauth[@]} > n)) || continue; ruta=${ctauth[n]} ;;
                bolsa) n=$bolsa_indice; ((${#bolsa[@]} > n)) || continue; ruta=${bolsa[n]} ;;
                ct) n=$ct_indice; ((${#ct[@]} > n)) || continue; ruta=${ct[n]} ;;
            esac
            codigo=${ruta##*/}
            if [[ $modulo == bolsa && $codigo == 000051_* && $bolsa_calculador == 0 ]]; then
                rrhh_e2e_sql deploy/postgresql/bolsa_llamamientos/roles_calculador_politica_up.sql || return 1
                bolsa_calculador=1
            fi
            if [[ $modulo == ct && $codigo == 000136_* && $ct_registrador == 0 ]]; then
                rrhh_e2e_sql deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql || return 1
                ct_registrador=1
            fi
            if rrhh_e2e_cola_sql "$ruta" "$modulo"; then
                printf '%s\n' "${ruta#"$VEC_E2E_ROOT/"}" >> "$VEC_E2E_WORK/pg_sql_aplicado"
                case $modulo in
                    ad3) ((ad3_indice+=1)) ;;
                    contexto) ((contexto_indice+=1)) ;;
                    ctauth) ((ctauth_indice+=1)) ;;
                    bolsa) ((bolsa_indice+=1)) ;;
                    ct) ((ct_indice+=1)) ;;
                esac
                progreso=1
                if [[ $modulo == bolsa && $codigo == 000010_* && $bolsa_registrador == 0 ]]; then
                    rrhh_e2e_sql deploy/postgresql/bolsa_llamamientos/roles_registrador_frontera_up.sql || return 1
                    bolsa_registrador=1
                fi
            fi
        done
        if ((progreso == 0)); then
            rrhh_e2e_error 'dependencias SQL no resueltas en la base sintética' || true
            for modulo in contexto ad3 ctauth bolsa ct; do
                case $modulo in
                    ad3) ruta=${ad3[ad3_indice]:-} ;;
                    contexto) ruta=${contexto[contexto_indice]:-} ;;
                    ctauth) ruta=${ctauth[ctauth_indice]:-} ;;
                    bolsa) ruta=${bolsa[bolsa_indice]:-} ;;
                    ct) ruta=${ct[ct_indice]:-} ;;
                esac
                [[ -z $ruta ]] && continue
                printf '  %s: %s\n' "$modulo" "${ruta#"$VEC_E2E_ROOT/"}" >&2
                sed -n '1,6p' "$VEC_E2E_WORK/pg_error_$modulo" >&2
            done
            return 1
        fi
    done
}
