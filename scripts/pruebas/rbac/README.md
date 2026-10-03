# Ensayo RBAC en un clon desechable

`ensayar.sh` restaura una copia fría física PG18. Conserva los roles y ACL de la
fuente. El contenedor usa `--rm`, datos propios en disco bajo `~/.local/state` y ninguna red.
No publica puertos. Los comandos SQL acceden por el socket del contenedor.
Antes de usarlos, el runner exige el ID registrado al crear el contenedor,
los mismos montajes registrados al crearlo, un único montaje enlazado al
directorio propio y el directorio de datos esperado.

```bash
scripts/pruebas/rbac/ensayar.sh $HOME/.local/state/vec-rbac-mi-ensayo init /ruta/privada/copia.tgz
scripts/pruebas/rbac/ensayar.sh $HOME/.local/state/vec-rbac-mi-ensayo status
scripts/pruebas/rbac/ensayar.sh $HOME/.local/state/vec-rbac-mi-ensayo snapshot /ruta/privada/preimagen.txt
```

El formato de fuente admitido contiene `vec-desarrollo-20260906/pgdata/`.
La restauración no extrae certificados ni configuración externa de la fuente.
Los archivos de resultados se guardan fuera de Git. La preimagen registra
privilegios, definiciones de funciones y conteos/huellas de filas, sin mostrar
valores de las filas ni contraseñas.

Cada migración exige su SHA256 exacto y dos sondas revisadas. La primera debe
devolver únicamente `missing`; la segunda, únicamente `installed`. Estas sondas
deben consultar datos y definiciones reales de su preimagen y resultado. La lista
causal pertenece al escritor y al director del corte.

```bash
scripts/pruebas/rbac/ensayar.sh $HOME/.local/state/vec-rbac-mi-ensayo apply \
  /ruta/SQL.up.sql SHA256 /ruta/privada/pre.sql /ruta/privada/post.sql
```

El runner registra el intento antes de aplicar SQL y rechaza repetirlo en el
mismo clon. No admite archivos DOWN. Un fallo conserva el clon y su log privado;
el director decide la corrección o prepara otro clon. El comando `apply` no
acredita por sí solo permisos funcionales, CAS, idempotencia ni recuperación:
esas pruebas corresponden al contrato concreto de RBAC.

Después del ensayo, detener únicamente el contenedor propio. `--rm` elimina el
contenedor, pero mantiene sus datos para inspección hasta que el director cierre
el corte. Ningún comando de este runner borra la fuente o los datos del clon.
