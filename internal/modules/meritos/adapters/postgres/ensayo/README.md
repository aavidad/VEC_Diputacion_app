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

## Recorrido y recuperación comprobados

El ensayo del 1 de octubre de 2026 usó dos actores internos sintéticos con
perfiles separados: uno declaró y rectificó su hecho; el otro lo rechazó.
La secuencia fue declaración, replay, rectificación, rechazo, replay del
rechazo, CAS antiguo y clave original con otro contenido. Las primeras cinco
operaciones devolvieron `confirmada`; las otras devolvieron
`conflicto_version` y `clave_reutilizada`, sin recibo.

Para repetirlo, dirección debe restaurar su copia fría en un PGDATA nuevo,
verificar la preimagen y aplicar una sola vez la lista causal de RUM03 con
AD141/142 corregidas. La provisión OWNER inicial tiene que comprobar ausencia,
conservar la historia anterior y avanzar el checkpoint por CAS. El material
se genera y guarda únicamente en un directorio privado; no se proporciona un
fixture con permisos o claves como configuración de producto.

Registrar las sesiones justo antes de `consume`. Conservar los resultados y
las huellas de las tablas propias, detener PostgreSQL, volver a arrancarlo
con el mismo PGDATA y ejecutar `consume` en otro proceso con los mismos
comandos de negocio. Revalidar la sesión vigente o registrar una nueva por su
API si caducó. Cada reintento emite y consume una capacidad nueva: no guardar
una capacidad HMAC para usarla después del reinicio.

Comparar `recibo` y `anterior` completos, incluidos fecha, versión, auditoría
y evento originales. La `auditoria_ref` del sobre debe cambiar para acreditar
el acceso nuevo. Comparar también las huellas de `hecho_identidad`,
`hecho_version`, `operacion` y `outbox`: no deben aparecer versiones,
operaciones ni eventos nuevos por la recuperación.

El recorrido pasó de 1 hecho, 3 versiones, 3 operaciones y 3 eventos a esos
mismos registros tras reiniciar PostgreSQL y el proceso. Los accesos crecieron
de 2 a 7 y las auditorías de 7 a 14. Los siete códigos y todos los recibos,
fechas, antecedentes y huellas de historia de negocio fueron idénticos.
El driver y su adaptador no necesitaron un parche para completar el positivo.
