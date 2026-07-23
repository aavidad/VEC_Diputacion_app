# O3-03: contrato de fuentes de análisis

Fecha: 23 de julio de 2026.

Estado: correcciones del primer `NO-GO` implementadas; re-revisión
independiente pendiente. Este documento no concede un `GO` de integración,
piloto o producción.

## Alcance

O3-03 define la frontera hexagonal común para:

- validar una retención de crédito mediante una fuente presupuestaria;
- calcular el coste previsto de una contratación temporal;
- autenticar petición y respuesta como materiales canónicos distintos;
- verificar cada respuesta mediante un TCB independiente de la fuente;
- consumir una respuesta una sola vez, con replay exacto e idempotente;
- fallar de forma cerrada ante indisponibilidad, cancelación, caducidad o
  incoherencia.

El contrato no contiene HTTP, cookies, sesiones web ni almacenamiento del
navegador. Web, escritorio, API, CLI y MCP deberán invocar los mismos casos de
uso. Ningún canal puede aportar actor, autoridad, confirmación de verificación
ni recibo de consumo.

## Petición interna

`PeticionRef`, `SolicitadaEn` y el sello de petición no son campos libres:

1. un puerto genera una referencia opaca con espacio de nombres `pet_`;
2. un reloj inyectado proporciona un instante UTC canónico;
3. el núcleo construye una preimagen binaria determinista;
4. un puerto sellador devuelve un HMAC-SHA-256;
5. la solicitud opaca conserva preimagen y sello y detecta sustituciones.

El formato del sello de petición es:

```text
hmac-sha256:fuente-analisis-v1:<64 hexadecimales no nulos>
```

### Petición RC

| Orden | Coordenada canónica |
| ---: | --- |
| 1 | Dominio `VEC-CT-FUENTE-ANALISIS-RC-V1` |
| 2–5 | Petición, organización, expediente y versión |
| 6–7 | Referencia y huella de entrada RC |
| 8–13 | Existencia, número, fecha, importe, moneda y documento declarados |
| 14 | `SolicitadaEn` |

### Petición de coste

| Orden | Coordenada canónica |
| ---: | --- |
| 1 | Dominio `VEC-CT-FUENTE-ANALISIS-COSTE-V1` |
| 2–5 | Petición, organización, expediente y versión |
| 6–9 | Categoría, grupo/subgrupo, modalidad y causa |
| 10–12 | Inicio, fin y jornada |
| 13 | `SolicitadaEn` |

## Respuesta autenticada

La respuesta no se autentica repitiendo el HMAC de petición. Tiene material y
sello propios:

```text
hmac-sha256:fuente-analisis-respuesta/v<generación>:<64 hexadecimales>
```

La preimagen de respuesta contiene:

1. dominio separado para RC o coste;
2. autoridad, generación y recibo;
3. ventana firmada `EmitidaEn`–`ValidaHasta`;
4. todas las coordenadas y el HMAC de la petición;
5. todas las salidas funcionales;
6. en RC negativa, el vínculo completo del motivo publicado.

`AutoridadRef` debe coincidir con `FuenteRef`. Existe un único `ReciboRef`: el
recibo funcional y el de la atestación deben ser exactamente el mismo. No se
admiten dos identificadores con roles probatorios ambiguos.

Para RC se cubren resultado, entrada y huella, fuente/recibo, instante, fecha,
número, importe, moneda, documento y motivo minimizado. Para coste se cubren
fuente/recibo, importe, moneda e instante. Alterar cualquiera de ellos cambia
la preimagen y deja inválido el HMAC de respuesta.

## Verificador TCB independiente

El caso de uso siempre llama a `VerificadorRespuestaFuenteAnalisis` después de
recibir la respuesta. La fuente no puede incluir una confirmación ni construir
la orden de consumo. La separación ya no se infiere comparando punteros,
interfaces o wrappers.

Antes de consultar una fuente, la composición del servidor fija una confianza
institucional que no forma parte de la petición del cliente. Cada adaptador
presenta una credencial Ed25519 firmada por una raíz gobernada. El material
firmado incluye:

- organización y audiencia internas exactas;
- rol técnico;
- `AutoridadRef` y `BackendRef` canónico;
- clave pública de prueba de posesión;
- raíz, serie, generación y ventana de vigencia.

La prueba de posesión firma un desafío nuevo para cada invocación. El desafío
usa dominio propio, nonce CSPRNG de 256 bits, huella de la petición canónica,
organización, audiencia y rol. Copiar una credencial o una prueba anterior no
permite responder a otro desafío.

Las raíces admiten rotación `activa` → `retenida` → `revocada`, ventanas y
último instante autorizado de emisión. Una revocación por autoridad y serie
prevalece sobre una firma válida. La configuración rechaza raíces duplicadas.
Raíz, credencial y prueba se verifican antes de invocar la fuente; cualquier
indisponibilidad o inconsistencia falla cerrado.

Para RC se exigen tres autoridades autenticadas: fuente presupuestaria,
verificador criptográfico y publicador del catálogo. Para coste se exigen
calculador y verificador. En ambos casos deben ser distintos simultáneamente
en `AutoridadRef`, `BackendRef` firmado y clave pública. Por tanto, dos
wrappers, aliases o valores Go distintos sobre un mismo backend no crean
segregación. La autoridad autenticada de la fuente se cruza con la atestación
de respuesta; las del verificador y publicador se cruzan con sus respectivas
confirmaciones.

El verificador recibe la preimagen y la atestación opacas y devuelve una
confirmación ligada a:

- verificador TCB;
- autoridad y generación;
- recibo y sello;
- huella SHA-256 del material completo;
- ventana firmada e instante de verificación.

El núcleo comprueba de nuevo la confirmación, la identidad autenticada y el
reloj antes del consumo. Los constructores públicos de credencial,
presentación y confirmación solo cargan material probatorio: no conceden
confianza. Ninguno forma parte de DTO de cliente, cookie o cabecera libre. Solo
la verificación contra las raíces fijadas por la composición produce una
identidad interna aceptable. Un adaptador productivo deberá verificar el HMAC
con secreto externo, rotación y separación de dominios antes de emitir la
confirmación.

## Ventana y consumo

| Regla | Límite |
| --- | --- |
| Timeout total propio | 5 segundos |
| Ventana firmada de respuesta | Más de cero y máximo 5 segundos |
| Periodo previsto | 100 años exactos como máximo |
| Importe | 922.337.203.685.477 céntimos |
| `VersionExpediente` | 1 a 2^53−1 |
| `CatalogoVersion` | 1 a 2^53−1 |
| Moneda | EUR exacto; nunca coma flotante |
| Instantes | UTC canónico a microsegundo |

La ventana usa límite final exclusivo. Se comprueba al recibir, tras verificar
y justo antes de consumir. Un replay exacto dentro de la ventana puede devolver
el mismo recibo de consumo. Un replay caducado se rechaza antes de alcanzar el
consumidor.

`ConsumidorRespuestaFuenteAnalisis` tiene el contrato explícito de conservar
durablemente el primer consumo por autoridad, generación y recibo:

- misma huella: devuelve el mismo recibo;
- otra huella: `ErrRespuestaFuenteAnalisisYaConsumida`;
- el recibo de consumo queda ligado a petición, tipo y huella de respuesta.

El puerto no finge que ya exista persistencia. El adaptador durable y sus
pruebas de reinicio, concurrencia y restauración siguen pendientes. Una
cancelación posterior a un recibo de consumo válido no transforma el éxito en
un fallo ambiguo; si la dependencia falla, el contexto se comprueba
inmediatamente.

## Motivos publicados y minimizados

Una RC negativa no acepta texto libre del proveedor. El vínculo conserva:

- referencia, versión y huella del catálogo;
- entrada exacta;
- clave i18n;
- hasta ocho pares clave/valor ordenados, tipados como claves de catálogo.

El binario ya no compila nombres de parámetros ni valores funcionales.
`VerificadorPublicacionMotivoFuenteAnalisis`, con autoridad y backend
autenticados distintos de fuente y TCB, comprueba contra la publicación
gobernada la combinación exacta
referencia–versión–huella–entrada–i18n–parámetros. Su confirmación se liga
también al publicador autenticado, la huella de respuesta, autoridad y
generación.

La orden de consumo conserva el vínculo minimizado completo y el recibo de
verificación de publicación. El dominio materializa únicamente la clave i18n;
la evidencia durable no pierde el catálogo que la justificó.

Motivo, respuestas, atestaciones, confirmaciones, preimágenes y órdenes
implementan redacción para formato y `slog`. Los errores públicos no muestran
ni exponen mediante `Unwrap` la causa del proveedor. Solo
`context.Canceled` y `context.DeadlineExceeded`, sin detalle privado, se
propagan para permitir control seguro de cancelación.

## Matriz de evidencia

| Amenaza o defecto | Evidencia |
| --- | --- |
| Cruzar organización, expediente o versión | Matrices campo a campo de petición y respuesta |
| Alterar salidas después de firmar | Verificador HMAC TCB rechaza aun recompuesta la preimagen interna |
| Eco del HMAC de petición | Sello de respuesta con dominio y generación propios |
| Fuente actuando como verificador | Rol, autoridad, backend y clave distintos bajo credencial institucional |
| Wrappers/aliases del mismo backend | `BackendRef` canónico firmado y matrices por valor |
| Credencial copiada o manipulada | Firma raíz, prueba de posesión y nonce por invocación |
| Fuente autopublicando su catálogo | Tercera autoridad autenticada y prueba adversarial |
| Rotación o revocación | Raíz activa/retenida/revocada, corte de emisión y revocación por serie |
| Dos recibos probatorios | Igualdad obligatoria del único `ReciboRef` |
| Respuesta con ventana amplia | Frontera exacta 5 s y rechazo de `+1 µs` |
| Replay caducado | Rechazo antes de invocar consumo |
| Mismo recibo con otra respuesta | Conflicto idempotente |
| Catálogo compilado | Entrada y parámetros futuros aceptados estructuralmente |
| Catálogo/entrada/i18n cruzados | Confirmación de publicación no reutilizable |
| Texto o PII del proveedor | Texto libre rechazado, valores redactados y errores públicos |
| Causa privada mediante `Unwrap` | `errors.Is` no encuentra la causa del proveedor |
| Cancelación concurrente | Pruebas tras fuente y verificadores y en frontera durable |
| Desbordes | Periodo, importe y enteros exactos en 2^53−1 / 2^53 |
| Alias mutable o carrera | Copias defensivas y `go test -race` |

## Evidencia ejecutable

```text
go test ./internal/modules/contrataciontemporal/ports -count=1
go test -race ./internal/modules/contrataciontemporal/ports -count=1
go vet ./internal/modules/contrataciontemporal/ports
go test ./...
go vet ./...
scripts/comprobar_tamano_ficheros.sh
```

Los dobles de respuesta calculan y verifican HMAC-SHA-256 real sobre la
preimagen completa. El doble de consumo conserva la huella y distingue replay
exacto de conflicto.

## Condiciones pendientes antes de integrar o producir

O3-03 no queda habilitado para datos reales ni efectos jurídicos hasta disponer
al menos de:

1. generador CSPRNG de referencias opacas;
2. selladores de petición y respuesta con secretos separados, custodia y
   rotación;
3. provisión institucional de raíces, credenciales, `BackendRef` canónicos,
   rotación y revocación fuera del proceso;
4. adaptador TCB independiente que verifique la autoridad y generación reales;
5. verificador de publicaciones conectado al catálogo institucional;
6. consumidor durable con unicidad, reinicio, concurrencia y restauración
   probados;
7. recibos inmutables ligados a la transacción del expediente;
8. autorización VEC consumible, auditoría y outbox atómicos;
9. adaptadores de presupuesto y cálculo aceptados por Sistemas, RRHH e
   Intervención;
10. E2E de indisponibilidad, caducidad, replay, rotación y revocación;
11. aceptación funcional y jurídica, EIPD y categorización ENS aprobadas.

La implementación contribuye a minimización, integridad, exactitud y
trazabilidad, pero no certifica por sí sola RGPD, LOPDGDD, ENS, ENI ni la
legalidad del procedimiento.
