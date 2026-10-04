# Control de restauración preparado con transacción explícita

`Registro.Proponer` y `RevisarCAS` mantienen la materialización V3, el consumo SQL
y la validación del recibo dentro de una misma transacción `SERIALIZABLE READ WRITE`,
con zona horaria UTC. El adaptador valida orden, huella de plan, Persona y versión
antes del COMMIT y sólo devuelve el recibo cuando éste se confirma.

La composición debe aportar un `MaterializadorEnTX` nuevo y un plazo de cierre
positivo de como máximo 30 segundos. No hay valor por defecto. El proveedor recibe
`CanalSQL`, que ofrece `QueryRow` y `Exec` sobre esa conexión y no expone `Begin`,
`Commit`, `Rollback` ni `Conn`. El adaptador conserva el control de la transacción;
el proveedor no debe emitir SQL de control transaccional ni conservar el canal.
No se adapta el materializador antiguo que trabajaba fuera de la transacción.

Dentro de ella sólo se admite SQL por las fachadas autorizadas propias y de la
autoridad común. No se hacen llamadas a la red, al sistema de archivos, a otra
conexión ni a tablas privadas de otro módulo. El contrato del proveedor operativo
de K sigue pendiente; el canal limitado no acredita por sí solo su procedencia.

Si falla el material, la consulta o el recibo, se intenta cerrar mediante rollback
con `WithoutCancel` y un plazo propio, también después de desconectarse el cliente.
Los errores de consulta, COMMIT y limpieza se convierten en errores nominales,
sin causas del proveedor. Un COMMIT fallido no devuelve recibo ni demuestra que
no hubo efecto: requiere conciliación. No hay reintento automático. Una cancelación
posterior a un COMMIT confirmado conserva el recibo ya confirmado.

## Alcance y dependencias

Este corte recupera únicamente el adaptador Go preservado en `e1045fd90` sobre
`c32984285`, corrige su transacción y conserva las firmas SQL de once y doce
argumentos. No trae SQL, montaje ADMIN, roles, permisos ni un proveedor nominal.
Las reservas AD143 y Administración Copias 000001 siguen en su carril propio.

Las pruebas reutilizan la fábrica V3 sellada del paquete común de pruebas y un
doble transaccional unitario. Verifican el ciclo completo del método público,
la conexión compartida, la validación previa al COMMIT, los errores y la limpieza.
Ese doble no se compone en producción ni acredita identidad, aprobación, auditoría
común, ACL, aislamiento o COMMIT reales de PostgreSQL.

Siguen pendientes la fuente de K, el contrato de auditoría V3 común de L, la
composición causal posterior a H9, el ensayo PostgreSQL autorizado y dos revisiones
independientes de la versión final. No se abre PR antes de esas revisiones.

## Comprobación focal del 4 de octubre de 2026

Las pruebas normales del paquete y `go test -race` terminaron con código 0,
con `-p 8`, caché y temporales en disco, sin descarga de dependencias, un límite
externo de 180 segundos y 90 segundos por prueba. `go vet`, `gopls check` de los
cinco archivos Go y gosec focal terminaron con código 0. Semgrep local aplicó
42 reglas a los dos archivos de implementación, sin hallazgos y con métricas
desactivadas; los archivos de pruebas quedaron fuera de ese análisis por las
exclusiones estándar. No se ejecutaron SQL ni la campaña global.

## Fuentes de contraste

[GitLab](https://docs.gitlab.com/development/database/transaction_guidelines/)
recomienda limitar las transacciones a operaciones de base en la misma conexión,
sin red ni archivos. [Nextcloud](https://docs.nextcloud.com/server/latest/developer_manual/basics/storage/database.html)
muestra el cierre explícito con commit o rollback.
[Gitea](https://pkg.go.dev/code.gitea.io/gitea/models/db#WithTx) reutiliza una
transacción existente en `WithTx`.
[pgx](https://pkg.go.dev/github.com/jackc/pgx/v5#Conn.BeginTx) documenta que las
opciones de `BeginTx` fijan el modo y que cancelar el contexto no hace rollback
automático. El encaje anterior es una decisión de VEC, no una norma atribuida a
la Diputación.
