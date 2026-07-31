#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)

"$raiz/deploy/postgresql/contexto_actor_v1/probar_contexto_actor_v1_base_pg18_4.sh"
"$raiz/deploy/postgresql/contexto_actor_v1/probar_retiradas_contexto_actor_catalogos_concurrentes_pg18_4.sh"
"$raiz/deploy/postgresql/contexto_actor_v1/probar_retirada_acreditacion_estructura_acl_consumidores_v2_pg18_4.sh"
"$raiz/deploy/postgresql/contexto_actor_v1/probar_retirada_acreditacion_contexto_actor_v2_concurrencia_preservacion_ciclos_pg18_4.sh"
"$raiz/deploy/postgresql/contexto_actor_v1/probar_organizacion_corporativa_v1_pg18_4.sh"

echo 'contexto actor durable V2: integracion PostgreSQL 18 superada'
