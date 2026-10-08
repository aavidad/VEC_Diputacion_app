# Tiempo de respuesta del cuadro de Contratación temporal — 8 de octubre de 2026

El 8 de octubre, entre las 16:07:37 y las 16:07:43 UTC, se midió la consulta
vigente del cuadro de RRHH en un servidor HTTPS y un PostgreSQL 18.4 aislados.
Se usó el binario conservado del kit de desarrollo del 7 de octubre, SHA256
`eec3ce0043954fd0c2cd703d9658b2dda9d434516718cea3f11574e38a3435d8`.
El certificado de cliente, el actor y las decisiones de autorización procedían
de ese kit sintético. La base era una copia privada: 5.170 publicaciones de
5.071 expedientes, con un corte publicado que incluía las 5.000 filas de carga.

La consulta SQL era la V1 existente. Su fachada V5 coincide con el cuerpo de
CT184 (SHA256 del cuerpo normalizado
`59758e0bdab4fd4db620743f388d405256cf87c0a64121fa99b09997c62fbf9e`).
Los tres cuerpos modificados por CT187 coinciden con las huellas finales de
esa migración: materialización `174ee9497abf84536e59d96b8c1331dea84eeb9efde1d3e5777ca314dc975d76`,
totales `0b293192c96a7f193fac4160f3755d9809596226d16d4636be64b73c011201e3`
y resumen `b820203e211aa615dff50d6c49a26dce0e9f43daa2ec0854a9d3036bee401cf7`.
No se usó un lector nuevo ni se instaló SQL para esta medición.

Después de tres peticiones de calentamiento se hicieron 30 POST secuenciales
con límite de 50 expedientes. Las tres campañas Go `-race` ajenas que había en
el equipo siguieron activas durante todo el lote. Todas las respuestas fueron
HTTP 200 con TLS verificado, 50 expedientes, total 5.071, página siguiente y
la misma huella de las filas. La auditoría común pasó de 6.534 a 6.567
asientos: uno por cada petición, incluidos los tres calentamientos.

| Medida | p50 | p95 |
| --- | ---: | ---: |
| Petición HTTPS completa, cliente | 196,55 ms | 208,978 ms |
| Petición en el servidor | 193,8 ms | 206,0 ms |
| Tiempo acumulado de consultas a la base | 23,2 ms | 27,0 ms |
| Espera de conexión del pool | 0 ms | 0 ms |
| Consulta más lenta de cada petición | — | 9,2 ms |

El servidor registró 27 consultas a la base por petición. Una lectura con
límite 1 y otra con límite 100 también registraron 27 cada una, con el mismo
total de 5.071 y un asiento de auditoría por lectura. El número de consultas
no creció con el de filas de la página.

La frontera se comprobó aparte del lote: un certificado sintético de
Intervención recibió 403 y ninguna fila; sin certificado, TLS rechazó la
conexión antes de HTTP. Una petición cancelada por el cliente a los 20 ms
quedó registrada como 408, sin confirmar un consumo de lectura parcial.

El p95 HTTPS observado queda por debajo del objetivo local de 300 ms incluso
con esas tres pruebas Go concurrentes. La muestra no mide producción ni
establece una mejora causal respecto de otro binario. Tampoco prueba la ruta
CT192, que tiene otro contrato de lectura. De las 5.071 filas, 5.000 son carga
sintética en la fase de solicitud; el volumen sirve para medir el cuadro, pero
no reproduce la distribución de fases de un servicio real. El certificado
PostgreSQL del kit había caducado: se reemplazó únicamente en la copia local
por un certificado válido para el puerto de ensayo.

El commit `4378658db6273842ff47e7185fc951d8d103281f` corrige por separado
un caso N+1: si falla la preparación del catálogo de plazos, el cuadro conserva
la causa y muestra «sin calcular» sin volver a leer el catálogo por fila o
grupo. La medición HTTP anterior usó el binario del kit y no ejercitó ese fallo.
La prueba focal de 100 cálculos confirmó una preparación y ninguna relectura.
El banco local preexistente de 100 filas midió 0,583 ms por operación con
reglas preparadas y 61,065 ms con lectura fila a fila; esos tiempos son del
cálculo de plazos, no de HTTP.
