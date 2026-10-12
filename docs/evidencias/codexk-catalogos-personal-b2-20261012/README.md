# Catálogos de Personal en el servidor web

RRHH puede abrir Registro de Personal → Catálogos de Personal y consultar régimen, modalidad, situación y clase de servicio. Se reutilizan el servicio, el manejador HTTP y los repositorios de Personal que ya estaban montados en el proceso interno. La ruta nativa faltaba en `vec-server`; la interfaz existente ya la consumía. El GET usa la cuenta de consulta y el POST conserva la cuenta y el perfil de gobierno propios.

## Comprobación local

Código sobre `origin/main@eb4727b1b`, con la pila B2 fusionada. PostgreSQL 18.4, copia local de la principal con datos inventados, contenedor propio sin red, `--rm --restart=no --memory 2g`, datos en disco. Chrome del sistema mediante playwright-cli y certificado RRHH de ensayo.

| Tipo | HTTP | Entradas publicadas | Tiempo HTTP local |
| --- | --- | --- | --- |
| Régimen | 200 | 4 | 51 ms |
| Modalidad | 200 | 4 | 52 ms |
| Situación | 200 | 0 | 48 ms |
| Clase de servicio | 200 | 0 | 53 ms |

Cada respuesta contiene `data.evidencia.auditoria_ref`. Los registros del servidor muestran 13 consultas por petición, 47–52 ms de tiempo total y 27–29 ms de base. Son muestras locales, no un percentil de carga ni una medida de cidonia. La captura de escritorio conserva el estado vacío del catálogo de situación.

CT168 faltaba en esta copia y se instaló una vez; CT203 se instaló una vez después. El cuerpo de CT203 parte de la función instalada tras CT168, preserva su OID/ACL y añade exclusivamente la ruta exacta de catálogos de Personal. GET con certificado candidato: 403, 43 ms y un apunte durable de acceso denegado. No se ejecutó DOWN ni se reaplicó SQL instalado.

La ampliación del perfil lector cambió su plantilla. El primer arranque quedó cerrado hasta la provisión aprobada por la huella de su asignación. El mecanismo existente hizo CAS de versión 1 a 2 y el servidor arrancó. La lista de despliegue indica el par de variables para ese único arranque y exige cotejar la huella real en destino; no permite restaurar asignaciones revocadas. La revisión independiente señaló esta precondición y quedó documentada antes de abrir la PR.

Pruebas focales: `go test -p 6 ./internal/app/bootstrap -run CatalogoEmpleadoB2 -count=1`; `go test -p 6 ./internal/vec/adapters/httpapi -run ^TestRutasExactasAuditoriaEtiquetanSoloRutasNominales$ -count=1`. Formato y `git diff --check` verdes. Semgrep local con métricas desactivadas: un aviso en `ExportarMaterialParaConsumidor`, idéntico en main y fuera de las líneas cambiadas. Gosec del paquete: los 17 avisos conservados están fuera del cambio; ninguno en las líneas de esta PR. La puerta completa corresponde a la CI de la PR.

## Despliegue y siguiente paso

Usar [la lista ordenada de SQL y configuración](../../../deploy/principal/lista_sql_codexk_catalogos_personal_b2_20261012.txt). CT168 es condicional: si ya está instalada, no repetirla. CT203 requiere su preimagen exacta. La provisión del perfil es por huella y CAS en la misma autoridad existente; retirar las variables después del arranque.

Esta entrega habilita los catálogos de la pantalla. La lista y ficha de empleados aún necesitan su montaje en `vec-server`; el alta y el cese siguen en la cola de Codex-K, coordinados con la web de Codex-M. La sesión de navegador conserva incidencias previas de preferencias y sonda de empleados; no se afirma ausencia global de errores del portal ni alta/cese acreditados por este corte.
