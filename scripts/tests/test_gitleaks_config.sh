#!/usr/bin/env bash

set -euo pipefail

raiz="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if ! command -v gitleaks >/dev/null 2>&1; then
	printf '%s\n' 'gitleaks no esta instalado' >&2
	exit 2
fi

temporal="$(mktemp -d)"
trap 'rm -rf -- "$temporal"' EXIT
texto_password='pass'
texto_password+='word = "ValorReal12345"'
texto_clave='cla'
texto_clave+='ve_admin = "ValorReal12345"'
texto_ldap='LD'
texto_ldap+='AP: persona/ValorReal12345'
texto_credencial='creden'
texto_credencial+='cial: persona:ValorReal12345'
texto_sentinela='pass'
texto_sentinela+='word = "PRUEBA_NO_SECRETO_VECTOR_001"'

probar_bloqueo() {
	local nombre="$1"
	local contenido="$2"
	local caso="$temporal/$nombre"
	local estado

	mkdir -p "$caso"
	printf '%s\n' "$contenido" >"$caso/entrada.txt"
	set +e
	gitleaks dir "$caso" \
		--config "$raiz/.gitleaks.toml" \
		--redact --no-banner --no-color >/dev/null 2>&1
	estado=$?
	set -e
	if ((estado == 0)); then
		printf 'No se bloqueo el caso %s\n' "$nombre" >&2
		exit 1
	fi
}

probar_permitido() {
	local nombre="$1"
	local contenido="$2"
	local caso="$temporal/$nombre"

	mkdir -p "$caso"
	printf '%s\n' "$contenido" >"$caso/entrada.txt"
	gitleaks dir "$caso" \
		--config "$raiz/.gitleaks.toml" \
		--redact --no-banner --no-color >/dev/null
}

probar_bloqueo contrasena_literal \
	"$texto_password"
probar_bloqueo clave_castellana \
	"$texto_clave"
probar_bloqueo par_ldap_barra \
	"$texto_ldap"
probar_bloqueo par_credencial_dos_puntos \
	"$texto_credencial"

probar_permitido sustitucion_psql \
	"ALTER ROLE rol_prueba PASSWORD :'clave_generada';"
probar_permitido generacion_csprng \
	'clave_generada="$(openssl rand -hex 24)"'
probar_permitido sentinela_prueba \
	"$texto_sentinela"
probar_permitido referencia_dos_puntos \
	'referencia = "persona:identificador-publico"'
probar_permitido url_sin_credenciales \
	'url = "https://servicio.invalid/recurso"'
probar_permitido dsn_sin_credenciales \
	'dsn = "host=localhost dbname=prueba sslmode=disable"'

printf '%s\n' 'Reglas Gitleaks: casos positivos y negativos superados.'
