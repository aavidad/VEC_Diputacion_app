#!/usr/bin/env bash
set -euo pipefail
umask 077
directorio=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
if (($# != 2)); then
    echo 'ensayo_publico_uso: ensayo.sh <vec-publico> <vec-server>' >&2
    exit 2
fi
publico=$(realpath "$1")
publicador=$(realpath "$2")
scratch=$(mktemp -d /var/tmp/vec-publico-ensayo-XXXXXXXX)
argumentos_web=()
if [[ -n "${VEC_PUBLICO_WEB_REPO:-}" || -n "${VEC_PUBLICO_WEB_COMMIT:-}" ]]; then
    [[ -n "${VEC_PUBLICO_WEB_REPO:-}" && -n "${VEC_PUBLICO_WEB_COMMIT:-}" ]]
    argumentos_web=(--web-source "$VEC_PUBLICO_WEB_REPO" --web-commit "$VEC_PUBLICO_WEB_COMMIT")
fi
python3 "$directorio/empaquetar.py" --binary "$publico" --destination "$scratch/artefacto" "${argumentos_web[@]}"
publico="$scratch/artefacto/vec-publico"
contenedor="vec-publico-codexb-ensayo-$$"
red="vec-publico-codexb-ensayo-$$"
pid_publico=''
limpiar() {
    if [[ -n "$pid_publico" ]]; then
        kill "$pid_publico" 2>/dev/null || true
        wait "$pid_publico" 2>/dev/null || true
    fi
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
    docker network rm "$red" >/dev/null 2>&1 || true
    rm -rf "$scratch"
}
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
imagen='postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296'
docker image inspect "$imagen" >/dev/null
python3 -c 'import secrets,sys; open(sys.argv[1],"w").write(secrets.token_urlsafe(36))' "$scratch/admin.pass"
openssl req -x509 -newkey rsa:3072 -sha256 -nodes -days 1 -subj '/CN=VEC ensayo publico CA' \
    -addext 'basicConstraints=critical,CA:TRUE' -addext 'keyUsage=critical,keyCertSign,cRLSign' \
    -keyout "$scratch/ca.key" -out "$scratch/ca.crt" >/dev/null 2>&1
openssl req -newkey rsa:3072 -sha256 -nodes -subj '/CN=localhost' \
    -keyout "$scratch/server.key" -out "$scratch/server.csr" >/dev/null 2>&1
cat >"$scratch/tls.ext" <<'EOF'
subjectAltName=DNS:localhost,IP:127.0.0.1
extendedKeyUsage=serverAuth
keyUsage=digitalSignature,keyEncipherment
EOF
openssl x509 -req -sha256 -days 1 -in "$scratch/server.csr" -CA "$scratch/ca.crt" \
    -CAkey "$scratch/ca.key" -CAcreateserial -extfile "$scratch/tls.ext" \
    -out "$scratch/server.crt" >/dev/null 2>&1
docker network create "$red" >/dev/null
puerto=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
docker run --detach --rm --name "$contenedor" --network "$red" \
    --memory 512m --cpus 2 --pids-limit 128 --publish "127.0.0.1:$puerto:5432" \
    --env POSTGRES_DB=vec_bolsa_publica_ensayo --env POSTGRES_PASSWORD_FILE="$scratch/admin.pass" \
    --mount "type=bind,src=$scratch,dst=$scratch,readonly" "$imagen" >/dev/null
esperar_pg() {
    local intento
    for ((intento=0; intento<120; intento++)); do
        if docker exec "$contenedor" psql -XqAt -U postgres -d vec_bolsa_publica_ensayo \
            -c 'SELECT 1' >/dev/null 2>&1; then return 0; fi
        sleep 0.25
    done
    return 1
}
esperar_pg
docker cp "$scratch/server.key" "$contenedor:/var/lib/postgresql/server.key"
docker cp "$scratch/server.crt" "$contenedor:/var/lib/postgresql/server.crt"
docker exec --user root "$contenedor" chown postgres:postgres /var/lib/postgresql/server.key /var/lib/postgresql/server.crt
docker exec --user root "$contenedor" chmod 600 /var/lib/postgresql/server.key
docker exec "$contenedor" psql -Xq -U postgres -d vec_bolsa_publica_ensayo \
    -c "ALTER SYSTEM SET ssl='on'" -c "ALTER SYSTEM SET ssl_cert_file='/var/lib/postgresql/server.crt'" \
    -c "ALTER SYSTEM SET ssl_key_file='/var/lib/postgresql/server.key'" -c "ALTER SYSTEM SET ssl_min_protocol_version='TLSv1.2'" >/dev/null
hba=$(docker exec "$contenedor" psql -XqAt -U postgres -d vec_bolsa_publica_ensayo -c 'SHOW hba_file')
docker exec -i --user root "$contenedor" sh -c 'cat > "$1"' -- "$hba" <<'HBA'
local all postgres trust
hostssl all all 0.0.0.0/0 scram-sha-256
hostssl all all ::/0 scram-sha-256
hostnossl all all 0.0.0.0/0 reject
hostnossl all all ::/0 reject
HBA
docker restart "$contenedor" >/dev/null
esperar_pg
[[ "$(docker port "$contenedor" 5432/tcp)" == "127.0.0.1:$puerto" ]]
identificador=$(docker exec "$contenedor" psql -XqAt -U postgres -d vec_bolsa_publica_ensayo -c 'SELECT system_identifier FROM pg_control_system()')
python3 "$directorio/fixture_ensayo.py" configure "$scratch" "$puerto" "$identificador" "$publico"
cat >"$scratch/psql" <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec /usr/bin/docker exec -i --env PGSERVICEFILE --env PGSERVICE --env PGPASSFILE --env PGCONNECT_TIMEOUT --env PGAPPNAME "$contenedor" psql "\$@"
EOF
chmod 700 "$scratch/psql"
docker exec "$contenedor" psql -Xq -U postgres -d vec_bolsa_publica_ensayo \
    -c "ALTER SYSTEM SET log_statement='all'" -c "ALTER SYSTEM SET log_min_duration_statement=0" \
    -c "ALTER SYSTEM SET log_min_duration_sample=0" -c "ALTER SYSTEM SET log_statement_sample_rate=1" \
    -c "ALTER SYSTEM SET log_transaction_sample_rate=1" -c "ALTER SYSTEM SET debug_print_parse=on" \
    -c "ALTER SYSTEM SET debug_print_rewritten=on" -c "ALTER SYSTEM SET debug_print_plan=on" \
    -c "ALTER SYSTEM SET password_encryption='md5'" -c 'SELECT pg_reload_conf()' >/dev/null
python3 "$directorio/aprovisionar.py" --config "$scratch/install.json" --psql "$scratch/psql"
VEC_PUBLICO_TEST_CONFIG="$scratch/install.json" VEC_PUBLICO_TEST_PSQL="$scratch/psql" \
    python3 -m unittest discover -s "$directorio" -p test_aprovisionar.py -v
docker logs "$contenedor" >"$scratch/postgresql.log" 2>&1
python3 "$directorio/fixture_ensayo.py" assert_no_secrets "$scratch"
docker exec "$contenedor" psql -Xq -U postgres -d vec_bolsa_publica_ensayo \
    -c 'ALTER SYSTEM RESET log_statement' -c 'ALTER SYSTEM RESET log_min_duration_statement' \
    -c 'ALTER SYSTEM RESET log_min_duration_sample' -c 'ALTER SYSTEM RESET log_statement_sample_rate' \
    -c 'ALTER SYSTEM RESET log_transaction_sample_rate' -c 'ALTER SYSTEM RESET debug_print_parse' \
    -c 'ALTER SYSTEM RESET debug_print_rewritten' -c 'ALTER SYSTEM RESET debug_print_plan' \
    -c 'ALTER SYSTEM RESET password_encryption' -c 'SELECT pg_reload_conf()' >/dev/null
python3 "$directorio/fixture_ensayo.py" fixture "$scratch"
python3 "$directorio/fixture_ensayo.py" publish "$scratch" "$puerto" "$publicador"
bash "$directorio/arrancar.sh" --config "$scratch/runtime.json" >"$scratch/process.log" 2>&1 &
pid_publico=$!
esperar_publico() {
    local intento
    for ((intento=0; intento<120; intento++)); do
        if bash "$directorio/comprobar.sh" --config "$scratch/runtime.json" >"$scratch/check.tmp" 2>/dev/null; then return 0; fi
        if ! kill -0 "$pid_publico" 2>/dev/null; then
            echo 'ensayo_publico_proceso_terminado' >&2
            python3 "$directorio/fixture_ensayo.py" diagnostic "$scratch"
            return 1
        fi
        sleep 0.25
    done
    cat "$scratch/check.tmp" >&2
    return 1
}
esperar_publico
cp "$scratch/check.tmp" "$scratch/before.jsonl"
kill "$pid_publico"
wait "$pid_publico" 2>/dev/null || true
pid_publico=''
docker restart "$contenedor" >/dev/null
esperar_pg
python3 "$directorio/aprovisionar.py" --config "$scratch/install.json" --psql "$scratch/psql" --aplicar-instancia-dedicada
bash "$directorio/arrancar.sh" --config "$scratch/runtime.json" >"$scratch/process.log" 2>&1 &
pid_publico=$!
esperar_publico
cmp "$scratch/before.jsonl" "$scratch/check.tmp"
cat "$scratch/check.tmp"
docker exec "$contenedor" psql -XqAt -U postgres -d vec_bolsa_publica_ensayo \
    -c 'SELECT count(*) FROM vec_bolsa_publica_datos.manifiesto_consumido' | grep -Fx 1 >/dev/null
echo 'ENSAYO-PUBLICO-OK'
if [[ -n "${VEC_PUBLICO_CAPTURAS:-}" ]]; then
    python3 "$directorio/probar_navegador.py" --config "$scratch/runtime.json" --captures "$VEC_PUBLICO_CAPTURAS"
fi
