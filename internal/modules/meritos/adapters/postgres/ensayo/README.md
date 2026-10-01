# Ensayo de Méritos con autorización V3

Este comando consume una provisión sintética creada previamente en una copia
efímera. Usa las funciones y adaptadores comunes de PostgreSQL: revalidación de
sesión, resolución y registro del contexto, consulta de permisos, validación del
motivo y registro de la decisión. Firma COSE con Ed25519 y verifica la firma con
el servicio común antes de emitir la capacidad HMAC. El adaptador de Méritos
consume esa capacidad y confirma la operación mediante `operar_hecho_v1`.

No instala SQL ni publica permisos, personas, sesiones o claves. Tampoco sirve
para acreditar firma documental, conformidad de RRHH o uso productivo.

## Entrada privada

El JSON se entrega por entrada estándar; nunca se guarda en el repositorio.
Los campos y tipos están definidos en `configuration`, `cryptoConfig`,
`actorConfig`, `capabilityKey`, `rbacConfig` y `operation` de `config.go`.

- `mode: descriptor` devuelve el SPKI público y las huellas de raíz y
  configuración. No abre conexiones. Permite cotejar el material que dirección
  publique en el gobierno del clon.
- `mode: consume` recibe actores y operaciones con su código esperado. Cada
  operación reconstruye una autorización nueva mediante las autoridades reales.
  Un reintento conserva el comando y la clave de negocio.
- `mode: rbac-descriptor` valida los documentos de rol, control y asignación;
  devuelve sus bytes canónicos y las huellas calculadas por el dominio común.
  `mode: plan-descriptor` devuelve el comando canónico, su huella y la del
  contexto de recurso. Ambos modos preparan datos sin conectar a PostgreSQL.
- `keys` fija al arrancar las claves de las audiencias de declarar, rectificar y
  rechazar. Su material y sus metadatos deben coincidir con el gobierno ya
  provisionado. No se cambia el gobierno durante las operaciones.

Los pools solo aceptan usuarios `vec_rum03_*`, base `postgres` y socket `/socket`.
Se separan contexto, revalidación, fuente, registro, motivos y ejecución. Los
usuarios de ejecución necesitan las membresías exactas del perfil instalado;
el comando no concede membresías.

La sesión real debe registrarse justo antes del recorrido: la API vigente limita
la aserción y la sesión a cinco minutos. Cada capacidad HMAC dura como máximo
cinco segundos y se consume inmediatamente, sin guardar un vector para después.

## Comprobación

Compilar y ejecutar dentro de un sandbox: fuente y herramientas de solo lectura,
sin red, entorno explícito, caché y temporales privados, CPU/memoria/procesos y
tiempo limitados. El socket del único clon asignado se monta solo al consumir.
La compilación focal es:

```text
go test -p 8 ./internal/modules/meritos/adapters/postgres/ensayo
go build -p 8 -o /scratch/ensayo ./internal/modules/meritos/adapters/postgres/ensayo
```

`go test` comprueba la compilación; este paquete no tiene pruebas unitarias.
El positivo requiere un recorrido `consume` y comprobar su recibo, la historia,
la auditoría y el evento en PostgreSQL. La verificación del hecho permanece
desactivada. La provisión interna exige un canal administrativo autorizado;
las funciones de publicación externa no lo sustituyen.

El recorrido actual usa la familia interna. Los publicadores externos de Bolsa
y Usuarios limitan sus acciones y no publican permisos de Méritos. Una persona
externa necesita el contrato común correspondiente antes de acreditar un
positivo. Persona y relación de empleo conservan sus autoridades respectivas.
