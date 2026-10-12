# Lectura B2 de CT tras el cese y carga de opciones

## Alcance y reutilización

La ruta existente `GET /api/vec/contratacion-temporal/incorporacion-personal-b2/plan/v1` recupera el plan y el origen nominales de CT. Si ya existe confirmación, devuelve el recibo histórico de CT junto a la versión actual del expediente. Conserva las lecturas autorizadas y auditadas de contrato, origen y detalle; no consulta de nuevo los hechos actuales de Personal, RPT o Bolsa para decidir si ocurrió la incorporación. Un plan aún sin origen sigue sujeto a las fuentes, versiones y conflictos de preparación actuales. El `POST` de preparación y confirmación no cambia.

El recorrido visible que consume esta respuesta está en la PR de interfaz #1058. Esta acta acredita la ruta real; la comprobación de la pantalla conjunta corresponde a esa PR antes de fusionarla.

## Evidencia local

- PostgreSQL 18.4 en contenedor propio con `--rm --restart=no --memory 2g`, copia de datos inventados de la principal. Aplicaciones y certificados de prueba privados fuera de Git. No se ejecutó SQL contra cidonia.
- Chrome del sistema mediante `playwright-cli`, HTTPS local con certificado de prueba. Expediente de ensayo CT con plan v8, cese posterior v9 y Personal relación y ocupación rev2 finalizadas.
- Cinco GET posteriores al cese: HTTP 200, estado `incorporacion_confirmada`, CT actual v9, intención del plan v8, mismo recibo `recibo:incorporacion-personal-b2:99aac398-2c24-4713-81b2-b98caece10eb` y fecha `2026-10-12T16:18:09.834112Z`; opciones y prerrequisitos son listas vacías, sin `null`. Sin nueva alta ni cese.
- Antes de compartir el detalle, la respuesta tomaba 408–516 ms de servidor con 47 consultas fijas. Con captura estrictamente por petición GET B2: 33 consultas fijas; cuatro peticiones calientes en Chrome tomaron 245–269 ms, con 203–227 ms de servidor y 60–73 ms de base. La primera petición tras reiniciar la aplicación tomó 373 ms en Chrome. La espera de conexión del pool fue aproximadamente nula. La reducción es de lecturas repetidas del mismo detalle, no de consultas por vacante.
- El diagnóstico de la rama `sin_plan` con 32 vacantes dio 332–407 ms calientes en Chrome (p50 363 ms, p95 407 ms) y 81 consultas fijas. La copia de carga estaba manipulada por SQL directo y seguía proyectando CT v9: no acredita una preparación prospectiva v8 completa. El objetivo de menos de 300 ms y la atribución por suboperación quedan para el corte de rendimiento siguiente. Esta PR no modifica el ensamblaje de opciones. Las vacantes ya se consultan en un lote SQL de Personal; el bucle Go solo compone DTO.

## Validación y límites

`go test -p 6 ./internal/app/bootstrap -run '^TestIncorporacionB2(GET|DetalleCompartido)' -count=1` verde; la prueba focal de captura concurrente en fuentes pasó con `-race`. La prueba de composición cubre v8→v9, cruces de organización, unidad, expediente, plan, versión y huella, denegación actual, captura de detalle solo por petición y plan sin origen. `gofmt` y `git diff --check` verdes.

Este cambio no requiere migración SQL ni configuración nueva. La revisión independiente de identidad y frontera HTTP, Semgrep local, gosec y la CI del PR se documentarán en la descripción del PR. La apertura de ficha de Personal y la presentación del recibo en pantalla se comprueban con #1058 sobre este servidor.
