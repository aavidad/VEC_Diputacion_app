# P6: retirada controlada de Dietas R1D

Paquete administrativo **sin instalación automática**. Solo Claude, tras dos
revisiones E10 del hash final y autorización operativa de la ejecución, puede
usarlo en cidonia. No toca la asignación Personal ni borra historia. La rama
no prueba una ejecución en cidonia.

## Precondiciones

- Selector Dietas apagado, aplicación drenada para estas ocho cuentas y cero
  sesiones activas. `ALTER ROLE NOLOGIN` no cierra sesiones existentes.
- DBA de PostgreSQL 18. En cidonia se usa `podman exec -i` contra el
  contenedor PostgreSQL indicado por `VEC_P6_POSTGRES_CONTAINER` en el entorno
  privado: `psql -U postgres -d postgres` recibe el SQL por stdin. El runner
  verifica versión 18 y sesión superusuario antes de inventariar. No requiere
  `psql` en el host ni pasa DSN o contraseña en argumentos.
- Fuera de ese entorno se conserva el transporte `PGSERVICE` con
  `PGSERVICEFILE`/`PGPASSFILE` privados, propios, `0600` y externos a Git. El
  servicio nombra host, puerto, base y usuario; la contraseña queda solo en el
  passfile. Los dos transportes son excluyentes.
- Directorio de evidencia externo a Git, propio y `0700` en
  `VEC_P6_EVIDENCIA_DIR`.
- Exactamente los ocho LOGIN `vec_dietas_r1d_{registro_identidad,
  revalidacion_identidad,contexto,fuente_autorizacion,registro_autorizacion,
  motivos,dietas,personal}_desarrollo`, con atributos mínimos y una membresía
  directa esperada cada uno. El preparador actual declara también
  `vec_dietas_r1d_auditoria_frontera_desarrollo`: si existe, el paquete se
  detiene hasta inventariar y revisar ese alcance; no se le aplica NOLOGIN
  por inferencia.
- El inventario enumera rutas de membresía directas y transitivas a los
  grupos técnicos, con capacidad efectiva de herencia y `SET ROLE`. Otro
  LOGIN capaz de asumir `vec_dietas_ejecutor` o
  `vec_dietas_registrador_frontera` detiene la ejecución; los grupos de
  autorización compartidos se registran sin retirar accesos CT/Bolsa.
- Un único puntero a la asignación activa del rol
  `dietas_r1d_provisional`. `preimagen.sql` devuelve exactamente una fila.
  Cualquier versión, huella, rol, perfil o membresía inesperados detienen la
  transacción. El inventario privado conserva roles, membresías, sesiones y
  asignaciones sin contraseñas.

## Orden de uso por el ejecutor autorizado

En cidonia, como usuario operador autorizado del host, el identificador del
contenedor procede del entorno privado; no se escribe aquí ni se versiona:

```bash
export VEC_P6_POSTGRES_CONTAINER='<nombre obtenido del entorno privado>'
export VEC_P6_EVIDENCIA_DIR='<directorio privado externo a Git, modo 0700>'
bash deploy/principal/p6_retirada_dietas/ejecutar.sh --inventario
bash deploy/principal/p6_retirada_dietas/ejecutar.sh --rollback
VEC_P6_APLICAR=SI-P6-REVISADO bash deploy/principal/p6_retirada_dietas/ejecutar.sh --commit
```

En un host que tenga `psql` y su conexión privada revisada:

```bash
export PGSERVICE='<nombre de servicio privado>'
export PGSERVICEFILE='<ruta privada a pg_service.conf, modo 0600>'
export PGPASSFILE='<ruta privada a pgpass, modo 0600>'
export VEC_P6_EVIDENCIA_DIR='<directorio privado externo a Git, modo 0700>'
bash deploy/principal/p6_retirada_dietas/ejecutar.sh --inventario
bash deploy/principal/p6_retirada_dietas/ejecutar.sh --rollback
VEC_P6_APLICAR=SI-P6-REVISADO bash deploy/principal/p6_retirada_dietas/ejecutar.sh --commit
```

Cada invocación vuelve a inventariar; el ensayo `ROLLBACK` comprueba dentro de
la transacción la inserción n+1, el avance del puntero y los ocho NOLOGIN,
luego demuestra que el inventario posterior sigue activo. El `COMMIT` repite
las mismas guardas sobre preimagen actual. Un Go ejecutado desde este módulo
deserializa la asignación, exige `Validar()` y recalcula `HuellaSHA256()` de la
preimagen; produce la nueva versión revocada con identidad, ámbitos y fechas
originales, y huella con el mismo código de dominio. SQL bloquea y coteja la
preimagen antes de insertar y avanzar el puntero, fija actor/acto y deshabilita
los LOGIN en una sola transacción. No hay `DOWN` ni reactivación implícita.

Mantener cerrada la entrada de nuevas conexiones Dietas desde antes del
inventario hasta después del `COMMIT` y su inventario posterior. El SQL llama
`pg_stat_clear_snapshot()` antes de ambas lecturas de sesiones en la
transacción; el ensayo PG18 abre una conexión entre lecturas y acredita que
la segunda observación la detecta tras refrescar. El bloqueo de nuevas
conexiones depende también del drenaje externo: una comprobación SQL puntual
no sustituye esa condición operativa.

El runner guarda inventarios `p6-inventario-*`/`p6-post-*` fuera de Git con
permisos privados; borra los ficheros transitorios que contienen la preimagen
y el plan. No imprimir DSN ni adjuntar inventarios al canal o a Git. Si falla
el transporte de la transacción, el runner intenta un postinventario nuevo y
sale con error, aunque el cambio pudiera estar confirmado. No reintentar
`--commit` tras una respuesta incierta sin consultar el puntero y los roles.

Tras el `COMMIT`, Claude debe acreditar cero sesiones, rechazo de nuevas
conexiones de los ocho LOGIN, denegación de autorizaciones anteriores por
revalidación V3, y regresión CT/Bolsa mediante su barrido habitual. Registrar
el resultado real en `deploy/principal/03_entorno.md`. La comprobación de
sesiones del runner cubre antes y después de la transacción, pero el drenaje
de la aplicación es responsabilidad del ejecutor. Para reactivar F4 hace
falta otro procedimiento revisado: el preparador actual fija v1 y no restaura
LOGIN existentes.

## Ensayo aislado

`probar_pg18.sh` usa `postgres:18.4` sin red y una base desechable. Instala solo
la estructura de autorización necesaria y una asignación sintética; prueba
ROLLBACK, COMMIT y repetición denegada, rutas transitivas de grupo y refresco
de sesiones. Solo monta el repositorio en lectura y un directorio privado
desechable; emula `podman exec` sobre Docker local para probar el mismo
protocolo de stdin. No usa datos ni credenciales de cidonia. Este fixture
comprueba el mecanismo, no reemplaza el inventario
real, el cierre de sesiones ni la revalidación V3 de Claude.
