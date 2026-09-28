# Recorrido de Auditoría RRHH en navegador aislado

Comprueba RRHH 2.13 y 4.11–4.15 sobre la ruta compuesta, con las fuentes CT y
Bolsa. Requiere una copia **sintética** en PostgreSQL 18 desechable, con AD3-91,
CT132, Bolsa48 y CT136 instaladas en esa copia, roles nominales V3, dos
certificados cliente sintéticos distintos y un servidor HTTPS en loopback. El
script no instala migraciones ni arranca servicios compartidos.

Crear `/dev/shm/fixture-auditoria-web.json` con permisos `0600`:

```json
{
  "url": "https://localhost:PUERTO",
  "ca": "/ruta/privada/ca.pem",
  "ct_cert": "/ruta/privada/ct.pem",
  "ct_key": "/ruta/privada/ct.key",
  "bolsa_cert": "/ruta/privada/bolsa.pem",
  "bolsa_key": "/ruta/privada/bolsa.key",
  "ct_ref": "expediente:ct:sintetico",
  "bolsa_ref": "participacion:bolsa:sintetica",
  "ct_actor_ref": "principal:sintetico:ct",
  "bolsa_actor_ref": "principal:sintetico:bolsa",
  "bolsa_clicks": ["SELECTOR_DE_FICHA", "[data-bolsa-auditoria]"],
  "desde": "2026-09-01T00:00",
  "hasta": "2026-09-28T00:00",
  "hook": "/ruta/privada/hook-aislado"
}
```

`bolsa_clicks` son los selectores reales para abrir una participación autorizada
desde la portada y pulsar su botón de Auditoría. El último debe abrir la
referencia `bolsa_ref`. El enlace CT usa el enlace canónico
`?expediente=…#contratacion-temporal`. Ambos perfiles deben tener el permiso
propio de su fuente; no basta mostrar el módulo en el menú. El preflight
Python verifica la CA y el nombre HTTPS de cada certificado cliente sin seguir
redirecciones; Chrome omite su propia comprobación de CA solo en este loopback.

El hook privado recibe uno de `start`, `stop`, `restart_pg`, `frontera_ref`,
`consumos_v3`, `cleanup`. `start` espera a HTTPS listo y usa exclusivamente la
copia PG18. `stop` detiene esa aplicación; `restart_pg` reinicia solo el PG18
efímero; `cleanup` borra contenedor `--rm`, datos y socket de `/dev/shm`. Los
contadores devuelven únicamente JSON mínimo y las demás acciones no imprimen datos:

- `frontera_ref CORRELACION`: consulta la fila CT136 de esa correlación exacta
  y devuelve `{"count":1,"ruta":"/api/vec/auditoria/consultas",` más
  `"motivo":"acceso_denegado","actor_ref":"…"}`. Debe fallar si hay cero o
  varias filas; el script coteja la correlación de la cabecera de cada 403 y
  el actor nominal del certificado ajeno, antes y después de reiniciar.
- `consumos_v3`: devuelve `{"ct":N,"bolsa":N}`. La consulta SQL debe limitar
  la audiencia a `vec_auditoria.consulta_rrhh.v1`, operación a
  `vec.auditoria.consultar`, referencia a `ct_ref`/`bolsa_ref` y actor a
  `ct_actor_ref`/`bolsa_actor_ref` respectivamente. El script exige +2 por
  fuente antes y +2 después del reinicio; comprueba la preimagen recuperada
  antes de generar nuevas lecturas.

Las filas históricas de las dos fuentes deben caber en **una sola página** del
cliente (50 filas o menos) dentro del intervalo elegido y tener actor. Al menos
una fila CT debe enlazar motivo y recibo con valores anteriores/nuevos o sus
dos huellas. En Bolsa48, la situación aporta motivo y la traza de cambio
aporta valores: la fixture debe contener ambas filas con **el mismo recibo** y
valores anterior y nuevo visibles. Las demás pueden reflejar ausencias
históricas legítimas.
Antes de ejecutar, verificar que el volcado y certificados son
sintéticos y que el hook apunta solo al PG18 nuevo. Ejecutar:

```sh
VEC_AUDITORIA_WEB_SINTETICO=1 python3 \
  scripts/recorridos_rrhh_auditoria_web/recorrido.py \
  /dev/shm/fixture-auditoria-web.json
```

Comprueba petición/respuesta reales, actor, instante UTC, comparación con motivo
y recibo según el contrato de cada fuente, expediente, 403 cruzado durable,
igualdad de la historia
tras reiniciar app/PG18, 1440/390 px, errores JS/5xx, cookies y almacenamiento.
Las capturas quedan únicamente en `/dev/shm/vec-auditoria-web-PID/`; borrarlas
tras la revisión junto con la fixture y el hook. Una ejecución que carezca de
volcado, identidad mTLS, concesiones nominales o hook **no acredita E2E**.

El ensayo estructural anterior en `scripts/rrhh_auditoria/` cubre migraciones,
ACL y HTTP con otra preimagen. Este recorrido añade la navegación visible y la
comparación de denegaciones de CT136. `4.14` sigue sujeto a política: esta
prueba no exige ni inventa IP/equipo. Tampoco afirma que la historia anterior a
VEC tenga valores recuperables.
