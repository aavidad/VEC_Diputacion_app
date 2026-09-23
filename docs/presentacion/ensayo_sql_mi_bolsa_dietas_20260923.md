# Ensayo SQL aislado de Mi bolsa y Dietas — 23 de septiembre de 2026

## Preimagen y alcance

La estructura se obtuvo de solo lectura de `cidonia.cloud`, PostgreSQL 18.4,
mediante `pg_dump --schema-only` de los nueve esquemas `vec_*` existentes. El
volcado conserva propietarios, funciones, políticas RLS y ACL; no contiene filas.
La lista separada de 93 roles aporta solo nombres para recrearlos como `NOLOGIN`
en el contenedor desechable. Ambos archivos temporales quedan fuera de Git.

En la preimagen están AD47, Bolsa16, AD48, Bolsa17, el consumidor V3 de Mi bolsa
y la consulta de doce argumentos sin `situacion_actual` ni `ultimo_llamamiento`.
Las seis tablas de Bolsa relevantes y las ocho de Dietas tienen RLS forzosa.
En la base viva existen 12 constituciones, 390 entradas y cero vínculos de
candidato; esos datos **no** se restauraron ni alteraron en el ensayo.

La cadena pendiente exacta es:

1. `bolsa_llamamientos/000021_rellenar_vinculos_candidato`;
2. `bolsa_llamamientos/000022_consulta_mi_bolsa_situacion`;
3. `bolsa_llamamientos/000023_consulta_mi_bolsa_ultimo_llamamiento`;
4. `dietas_borradores/000005_auditoria_frontera`.

Los tres primeros pasos de Bolsa son incrementales. Dietas 5 es independiente
de ellos. El ensamblador `deploy/principal/02_migraciones.sh` incluye también
AD47/Bolsa16/AD48/Bolsa17, que tienen historia en cidonia; no se ejecuta allí
de nuevo. Bolsa 21 solo instala una función administrativa: no crea los 390
vínculos. Su relleno exige un paquete de filas acreditadas por el recuperador
de importación y una operación separada y autorizada.

## Comando reproducible para Claude en este equipo

Requiere acceso SSH de solo lectura a cidonia y Docker local con la imagen
`postgres:18.4-alpine`. El script no publica puertos y retira el contenedor al
terminar. El `trap` elimina los dos archivos temporales, incluso si falla.

```bash
set -euo pipefail
cd /home/alberto/Trabajo/VEC_wt_fase1_p2_pg18
umask 077
vec_p2_tmp=$(mktemp -d)
trap 'rm -rf "$vec_p2_tmp"' EXIT
ssh -o BatchMode=yes root@cidonia.cloud \
  "su - openclaw -c 'podman exec vec-postgresql-20260906 pg_dump -s -U postgres -d postgres -n vec_autorizacion -n vec_autorizacion_atestada_v3 -n vec_bolsa_importacion_convoca -n vec_bolsa_llamamientos -n vec_contexto_actor_v1 -n vec_contratacion_temporal -n vec_dietas -n vec_identidad_sesiones_v1 -n vec_personal'" \
  > "$vec_p2_tmp/cidonia_schema.sql"
ssh -o BatchMode=yes root@cidonia.cloud \
  "su - openclaw -c 'podman exec vec-postgresql-20260906 psql -X -At -U postgres -d postgres -c \"SELECT rolname FROM pg_roles WHERE rolname LIKE '\''vec_%'\'' ORDER BY 1\"'" \
  > "$vec_p2_tmp/roles.txt"
scripts/probar_cadena_mi_bolsa_dietas_pg18.sh \
  "$vec_p2_tmp/cidonia_schema.sql" "$vec_p2_tmp/roles.txt"
```

El resultado comprobado fue código `0`: precondiciones reales, ROLLBACK y COMMIT
de cada migración, presencia y evolución de funciones, RLS forzosa, permisos de
función y tabla, rol `NOLOGIN`, conexión nominal de Dietas, definición V3 y
ACL/RLS de tablas anteriores conservadas. El ensayo no ejecuta `DOWN`, no
reaplica migraciones con historia y no escribe en cidonia. La comparación de
historia se limita a estructura y permisos, porque `--schema-only` no copia
datos ni puede acreditar persistencia o comportamiento con las 390 entradas.

Antes de cualquier instalación en cidonia deben cerrarse dos revisiones
independientes del hash final, comprobar de nuevo la preimagen y separar el
relleno autorizado de vínculos del despliegue de funciones. La instalación y
el recorrido navegador→API→PostgreSQL quedan bajo dirección principal.
