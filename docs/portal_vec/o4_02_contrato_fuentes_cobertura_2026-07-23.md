# O4-02: contrato de fuentes de cobertura

Fecha: 23 de julio de 2026.

Estado: corrección funcional implementada en rama aislada; revisión
independiente e integración pendientes. Este documento no concede un `GO` de
integración, piloto o producción.

## Alcance

O4-02 define la frontera hexagonal para comprobar una condición de una vía de
cobertura de una contratación temporal. La vía, la comprobación y su
procedencia no están compiladas en el núcleo: pertenecen a una publicación
versionada, vigente e inmutable del catálogo de cobertura.

El contrato aporta:

- petición mínima y representación binaria canónica;
- respuesta funcional completa, inmutable y autenticada;
- autenticación institucional de fuente, verificador y publicador;
- restauración de la publicación y prueba exacta de pertenencia;
- consumo durable único, replay exacto e idempotente y conflicto explícito;
- cancelación prioritaria y errores públicos sin causas privadas;
- límites comunes para web, escritorio, API, CLI y MCP.

No existe una variante web del caso de uso. El contrato no contiene HTTP,
cookies, sesiones, almacenamiento del navegador ni cabeceras de identidad. Los
distintos canales deberán invocar la misma aplicación y no pueden aportar
raíces, credenciales de sistema, confirmaciones ni recibos.

## Flujo obligatorio

Una consulta válida sigue este orden:

1. validar petición, dependencias, organización y timeout total;
2. canonizar la petición;
3. autenticar fuente, verificador y publicador mediante tres desafíos nuevos;
4. probar que las tres autoridades tienen identidad, backend y clave distintos;
5. recuperar del publicador la publicación completa del catálogo;
6. restaurar su canon y huella y comprobar vigencia y pertenencia exacta;
7. consultar la fuente definida por el catálogo;
8. comprobar coordenadas, cronología y atestación de la respuesta;
9. verificar el HMAC mediante una autoridad institucional distinta;
10. volver a comprobar ventana y catálogo;
11. consumir de forma durable la respuesta;
12. devolver el resultado funcional solo después de validar el recibo.

Cualquier ausencia o incoherencia falla cerrado. Una fuente no puede
autoverificar su respuesta ni autopublicar el catálogo que la autoriza.

## Petición canónica

`SolicitudConsultarCobertura` solo transporta:

| Grupo | Coordenadas |
| --- | --- |
| Operación | petición, organización, expediente y versión |
| Catálogo | referencia, versión y huella SHA-256 |
| Selección | vía y comprobación exigible completas |
| Procedencia | clave y definición gobernada del conector |
| Entrada funcional | categoría y periodo previsto |
| Tiempo | instante UTC canónico de solicitud |

La representación usa el dominio
`VEC-CT-FUENTE-COBERTURA-PETICION-V1`, longitudes explícitas y enteros en
orden de red. No depende de JSON, orden de mapas, locale, zona horaria ni
formato de base de datos.

La respuesta incorpora la huella SHA-256 de esta petición canónica. Así queda
ligada también a orden, obligatoriedad, procedencia y cualquier futura
coordenada canónica, no solo a una selección nominal.

## Respuesta autenticada

La preimagen con dominio
`VEC-CT-FUENTE-COBERTURA-RESPUESTA-V1` cubre:

- petición y su huella canónica;
- organización, expediente y versión;
- identidad exacta del catálogo;
- vía, procedencia, categoría y periodo;
- clave, resultado, fuente, recibo e instante de la comprobación;
- definición gobernada de la fuente;
- autoridad, generación y único recibo de atestación;
- ventana `EmitidaEn`–`ValidaHasta`.

No se admite `Detalle` libre en la respuesta del proveedor. Esto evita que un
conector propague nombres, DNI, observaciones o mensajes internos. La
preimagen, resultado, atestación y confirmaciones se formatean de forma
redactada para `fmt` y `slog`.

`FuenteRef` debe ser exactamente la autoridad de la atestación y el recibo
funcional debe ser exactamente el recibo atestado. No existen dos identidades
o dos recibos probatorios con significado ambiguo.

El sello tiene dominio propio:

```text
hmac-sha256:fuente-cobertura-respuesta/v<generación>:<64 hexadecimales>
```

La fuente sella; el núcleo no conoce el secreto; una autoridad separada
verifica y emite una confirmación ligada a la huella de todo el material,
fuente, generación, recibo, sello y ventana. Alterar cualquier coordenada
invalida la respuesta.

## Autoridades y raíces institucionales

O4-02 reutiliza el motor común Ed25519 de O3-03. No crea otra biblioteca de
credenciales, roles, raíces o revocaciones. Añade exclusivamente estos roles
de protocolo:

- `fuente_cobertura`;
- `verificador_cobertura`;
- `publicador_catalogo_cobertura`.

Cada autoridad presenta una credencial institucional y firma un desafío que
incluye nonce CSPRNG de 256 bits, huella de petición, organización, audiencia
y rol. La confianza se fija en la composición del servidor. Nunca procede del
cliente.

La segregación exige diferencias simultáneas en:

- `AutoridadRef`;
- `BackendRef` canónico firmado;
- clave pública de prueba de posesión.

Dos wrappers o aliases del mismo backend no son dos autoridades. La
verificación conserva las reglas comunes de vigencia, rotación de raíces,
última emisión permitida y revocación por autoridad y serie.

## Catálogo dinámico gobernado

El publicador no devuelve un booleano. Devuelve la publicación completa. El
núcleo:

1. recalcula canon y huella con
   `RestaurarCatalogoViasCobertura`;
2. exige identidad exacta referencia–versión–huella;
3. impide usar una publicación posterior a la petición;
4. exige vigencia tanto al solicitar como al comprobar;
5. busca la vía por clave;
6. exige igualdad completa de clave, orden, obligatoriedad, procedencia y
   definición de fuente.

Una vía nueva funciona mediante una publicación nueva, sin `switch`,
allowlist compilada ni recompilación del núcleo.

## Consumo durable e idempotencia

`ConsumidorCobertura` recibe una orden inmutable ligada a:

- petición, organización, expediente y versión;
- autoridad, generación y recibo de respuesta;
- huella SHA-256 de la respuesta completa;
- atestación y confirmación del verificador;
- publicación confirmada del catálogo.

El adaptador durable deberá imponer una única clave lógica
`(autoridad, generación, recibo)`:

- misma huella: devuelve exactamente el mismo recibo;
- otra huella: devuelve `ErrRespuestaCoberturaYaConsumida`;
- respuesta caducada: el núcleo la rechaza antes del consumidor.

Los dobles adversariales prueban replay concurrente, unicidad y conflicto. No
se declara que exista todavía un adaptador productivo. La implementación
durable, su transacción con expediente/auditoría y las pruebas de reinicio
corresponden a la tarea de persistencia y composición.

## Límites e interoperabilidad

| Regla | Límite |
| --- | --- |
| Timeout total | Más de cero y máximo 5 segundos |
| Vida de respuesta | Más de cero y máximo 5 segundos |
| Fin de ventana | Exclusivo |
| Periodo | 100 años exactos como máximo |
| Versión de expediente | 1 a 2^53−1 |
| Versión de catálogo | 1 a 2^53−1 |
| Instantes | UTC canónico, precisión máxima de microsegundo |
| Preimagen | Máximo 64 KiB |

El límite 2^53−1 permite transportar versiones de forma exacta también por
clientes JSON/JavaScript. Los periodos son fechas civiles UTC y no duraciones
aproximadas.

Tras cada dependencia se comprueba primero el contexto. Solo se exponen
`context.Canceled` o `context.DeadlineExceeded`; una causa privada nunca es
alcanzable con `errors.Is`, `errors.As` o `Unwrap`. Si ya existe recibo durable
válido, no se convierte un efecto confirmado en un fallo ambiguo.

## Evidencia ejecutable

Las pruebas incluyen:

- HMAC real sobre la preimagen completa;
- mutación campo a campo, huella de petición, sello y metadatos;
- raíz, rol, audiencia, organización, credencial y desafío manipulados;
- autoridad, backend o clave reutilizados;
- publicación adulterada, suplantada o caducada;
- vía futura añadida exclusivamente por catálogo;
- replay exacto concurrente, conflicto y expiración exclusiva;
- timeout total, cancelación prioritaria y nulos tipados;
- límites 100 años, 2^53−1 y cinco segundos;
- minimización y formatos redactados sin PII.

Puertas previstas:

```text
go test ./internal/modules/contrataciontemporal/ports -count=1
go test -race ./internal/modules/contrataciontemporal/ports -count=1
go vet ./internal/modules/contrataciontemporal/ports
go test ./...
go vet ./...
scripts/comprobar_tamano_ficheros.sh
git diff --check
gitleaks detect --no-banner --redact
```

## Decisiones pendientes de Sistemas

No se han localizado conectores productivos autorizados para Bolsa, SAE u
otras fuentes de cobertura en este repositorio. Por tanto, O4-02 no lee tablas
ajenas ni simula un adaptador de producción.

Antes de habilitar datos reales, Sistemas, RRHH y seguridad deben acordar:

1. API, vista o servicio institucional de cada procedencia;
2. identidad de servicio y `BackendRef` canónico;
3. provisión, rotación y revocación de raíces y credenciales;
4. custodia y rotación del secreto de respuesta;
5. despliegue separado del verificador criptográfico;
6. publicación append-only del catálogo y autoridad publicadora;
7. consumidor durable y transacción con expediente, auditoría y outbox;
8. timeout, reintentos, circuit breaker y observabilidad sin PII;
9. pruebas E2E de caída, expiración, replay, restauración y revocación.

Los adaptadores reales deberán ser finos: traducir el puerto acordado y nunca
consultar directamente tablas propiedad de Bolsa, SAE u otro módulo.

La implementación contribuye a minimización, integridad, exactitud y
trazabilidad, pero no certifica por sí sola RGPD, LOPDGDD, ENS, ENI ni la
legalidad del procedimiento.
