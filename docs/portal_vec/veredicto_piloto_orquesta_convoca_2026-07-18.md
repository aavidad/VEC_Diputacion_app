# Veredicto del piloto de Orquesta para importar Convoca

**Fecha:** 18 de julio de 2026
**Alcance:** prueba aislada; solo ficheros sintéticos; sin promoción automática
al árbol principal.

## Decisión

**NO-GO.** No se amplía esta configuración de Orquesta a otras capacidades y
no se integra ni se commitea el código producido por el piloto.

La decisión no invalida el modelo funcional de Convoca. Separa dos resultados:

1. la entrega creada por Orquesta no supera las puertas mínimas de calidad;
2. el corte experimental ya existente del importador tampoco puede exponerse a
   datos reales hasta endurecer el tratamiento de XLS y sus invariantes.

## Resultado medido de Orquesta

| Indicador | Esperado | Observado |
| --- | --- | --- |
| Agentes de programación y revisión | 6 roles separados | 1 proceso |
| Revisiones independientes | Al menos 2 | 0 |
| Entrega | Vertical PostgreSQL completa | 6 ficheros Go, 175 líneas |
| Migraciones, función SQL, roles, ACL y RLS | Sí | No |
| Documentación y evidencias de cierre | Sí | No |
| `go test` | Verde | Falla |
| `go test -race` | Verde | Falla |
| `go vet` | Verde | Falla |
| Formato | `gofmt` limpio | 4 de 6 ficheros no conformes |
| Estado del proceso | Entrega válida | Marcador de proceso `failed` |

El panel proyectó el trabajo como entregado y un 99 %, aunque el proceso había
fallado, no existían revisiones y la fase de cierre permanecía bloqueada. Una
orden posterior de supervisión no abrió rework ni materializó revisores. Este
desajuste impide usar el porcentaje del orquestador como evidencia de calidad.

### Defectos bloqueantes de la entrega

- El adaptador llamaba a una función PostgreSQL inexistente y no aportaba DDL.
- La denominada prueba de integración no abría PostgreSQL, no ejecutaba
  migraciones y no contenía aserciones.
- El código no compilaba por un `context` no importado y por una prueba de
  arquitectura con ruta incorrecta.
- Nombres, apellidos y documento enmascarado se enviaban como JSON sin cifrado
  gobernado ni cierre seguro si faltaba el conector.
- No se demostraban idempotencia durable, concurrencia, reintentos, reinicio,
  rollback ni recuperación tras respuesta perdida.
- No se creó documentación y las pruebas adversariales eran insuficientes.

El piloto sí respetó el alcance, no creó puertos nuevos, no utilizó datos reales
y mantuvo `pgx` fuera del núcleo. Esos aspectos positivos no compensan los
bloqueos anteriores.

## Auditoría del corte experimental de Convoca

Las pruebas ordinarias, `-race`, `go vet` y `govulncheck` resultaron verdes en
los paquetes ya existentes. La revisión adversarial detectó riesgos que esas
pruebas no cubrían:

1. **Crítico: agotamiento de recursos en el parser.** La dependencia XLS procesa
   estructuras OLE/BIFF antes de que el adaptador pueda aplicar los límites de
   filas, columnas y celdas. Ciclos FAT/DIFAT/MiniFAT o contadores internos
   hostiles pueden causar bucles o consumo no acotado aun con un fichero menor
   de 16 MiB.
2. **Crítico: lectura parcial sin error.** El parser puede devolver una hoja
   parcial ante truncados o errores y no acredita de forma estricta
   BOF/BIFF8/EOF. Importar una parte de un fichero como si fuese completo rompe
   la integridad probatoria.
3. **Alto: huella y contenido pueden divergir.** El servicio calcula SHA-256 y
   entrega después el mismo `[]byte` mutable al decodificador. Debe hacer una
   copia defensiva única antes de ambas operaciones.
4. **Alto: validación incompleta del lote.** Deben rechazarse DNI sin enmascarar,
   filas duplicadas, filas aceptadas y rechazadas simultáneamente, totales no
   canónicos y valores que incumplan las invariantes.
5. **Alto antes de exponer un endpoint:** el actor no puede proceder de la
   solicitud. Debe derivarse exclusivamente de la identidad interna autenticada
   y autorizada.

Los nombres y documentos enmascarados siguen siendo datos personales. El corte
actual solo admite datos sintéticos; no está autorizado para piloto real.

## Plan de corrección y nueva puerta de calidad

1. Copiar defensivamente el contenido al entrar y probar mutación concurrente.
2. Reforzar todas las invariantes, unicidad, disyunción y postcondiciones del
   repositorio.
3. Tratar XLS en un proceso o contenedor desechable sin red, con CPU, memoria y
   tiempo limitados; endurecer o sustituir el parser para exigir OLE/BIFF8
   estricto y fallo cerrado.
4. Añadir corpus corrupto, fuzzing adversarial y pruebas de límites reales.
5. Crear la vertical PostgreSQL completa: migraciones, funciones, roles, ACL,
   RLS, cifrado gobernado, idempotencia, auditoría y pruebas con una base real.
6. Ejecutar dos revisiones independientes y las puertas `gofmt`, `go test`,
   `go test -race`, `go vet`, `govulncheck`, integración PostgreSQL y reinicio.

Solo un resultado completamente verde permitirá commitear la vertical y abrir
la siguiente tanda de dos capacidades. Hasta entonces se continuará con agentes
directos y árboles aislados; esta versión de Orquesta no programará el proyecto.
