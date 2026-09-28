# RRHH 4.08: recorrido sintético de plantillas

Base de preparación: `bb67ac3ac` (CT131/133/135/137 y AD3-96/99/100). Este
directorio no instala SQL ni arranca VEC. `ensayar.sh` usa el ensayo existente
CT137/AD3-100 en PostgreSQL 18 efímero con datos en `/dev/shm`, `docker --rm`,
ACL, negativos y reinicio; después ejecuta Go focal. Se debe correr en una
máquina con Docker y Go 1.25.12 o superior. No usar una base conservada.

```bash
./scripts/recorridos_rrhh_plantillas_web/ensayar.sh
```

`recorrer.py` requiere un servidor HTTPS **loopback** ya compuesto con API,
PostgreSQL 18 y autorización V3 reales, un expediente sintético consultable
con versión exacta, y tres certificados mTLS sintéticos distintos: editor,
publicador y lector RRHH. Cada principal necesita concesiones exactas para su
acción y organización. El tipo `ensayo_*` debe ser nuevo en el catálogo de
esa organización. El texto y la referencia de aprobación que registra el
guion son de **ensayo**, sin aprobación administrativa, firma ni envío.

```bash
python3 scripts/recorridos_rrhh_plantillas_web/recorrer.py alta \
  --origen https://localhost:8443 \
  --editor-cert /dev/shm/vec-fixture/editor.crt --editor-key /dev/shm/vec-fixture/editor.key \
  --publicador-cert /dev/shm/vec-fixture/publicador.crt --publicador-key /dev/shm/vec-fixture/publicador.key \
  --lector-cert /dev/shm/vec-fixture/lector.crt --lector-key /dev/shm/vec-fixture/lector.key \
  --expediente-ref ref:EXPEDIENTE_SINTETICO --version 1 \
  --tipo ensayo_rrhh_408_una_clave_nueva \
  --evidencia /dev/shm/vec-rrhh-plantillas-408.json
```

El guion comprueba en Chromium a 1440 y 390 px: GET del catálogo, POST de alta
`201` y su recibo, publicación `201` desde otro certificado con recibo y
huella, lista documental `200` con procedencia, descarga PDF `200`, bytes y
SHA-256, además de errores JavaScript, cookies, almacenamiento y desbordamiento.
Un fallo detiene el recorrido. Si ya ocurrió un POST, el guion deja un JSON
parcial con su recibo; hay que consultar el estado antes de repetir cualquier
operación. El navegador no declara ámbitos: la API los toma del expediente
consultado.

Reiniciar **fuera del guion** la aplicación y PostgreSQL de ese entorno
efímero, sin crear otro expediente ni repetir el alta. Luego ejecutar el mismo
comando cambiando `alta` por `recuperar`. Compara la huella del catálogo y
los bytes del PDF en los dos anchos. La evidencia se guarda solo en `/dev/shm`
sin certificados ni claves; el operador debe comprobar adicionalmente en la
base que la historia y los recibos de alta/publicación no se duplicaron.

El ensayo SQL/Go verifica contratos aislados. El recorrido navegador acredita
el camino completo únicamente cuando se ejecuta contra el ensamblado final
con las migraciones y permisos instalados. El archivo JSON de la fase `alta`
por sí solo no acredita recuperación tras reinicio.
