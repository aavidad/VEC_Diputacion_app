# O3-03: contrato de fuentes de análisis

Fecha: 23 de julio de 2026.

Estado: implementación nominal revisada. Este documento no concede un
`GO` de integración, piloto o producción.

## Alcance

O3-03 define la frontera hexagonal común para:

- validar una retención de crédito mediante una fuente presupuestaria;
- calcular el coste previsto de una contratación temporal;
- ligar cada resultado a la petición interna exacta que lo originó;
- fallar de forma cerrada ante indisponibilidad, cancelación o respuestas
  incoherentes.

El contrato no contiene HTTP, cookies, sesiones web ni almacenamiento del
navegador. Web, escritorio, API, CLI y MCP deberán invocar los mismos casos de
uso y no podrán construir estas peticiones desde parámetros de transporte.

## Modelo de confianza

`PeticionRef`, `SolicitadaEn` y el sello no son campos libres:

1. un puerto genera una referencia opaca con espacio de nombres `pet_`;
2. un reloj inyectado proporciona un instante UTC canónico;
3. el núcleo construye una preimagen binaria determinista;
4. un puerto sellador devuelve un HMAC-SHA-256 con dominio y versión;
5. la solicitud opaca conserva una copia de la preimagen y del sello;
6. la respuesta debe devolver la referencia, el sello, la organización, el
   expediente y su versión exactos;
7. el núcleo vuelve a cotejar las coordenadas y la ventana temporal antes de
   aceptar el valor.

Los adaptadores no reciben un constructor que admita `PeticionRef`, hora o
HMAC arbitrarios. La forma pública de las solicitudes expone copias de lectura
y las estructuras autoritativas permanecen encapsuladas.

El formato exigido al sello es:

```text
hmac-sha256:fuente-analisis-v1:<64 caracteres hexadecimales no nulos>
```

El prefijo separa el dominio criptográfico y permite una evolución explícita.
La clave no forma parte del sello ni del repositorio. La implementación de
producción deberá guardar la clave fuera del proceso, rotarla y acreditar qué
generación firmó cada recibo. El puerto actual define ese contrato; no aporta
por sí solo custodia de claves ni una atestación durable.

## Preimagen canónica de validación RC

Codificación: campos de texto con longitud `uint32` en big endian, enteros en
big endian, booleano en un byte e instantes UTC en microsegundos Unix. Los
textos y la preimagen completa tienen límites explícitos.

| Orden | Coordenada ligada |
| ---: | --- |
| 1 | Dominio `VEC-CT-FUENTE-ANALISIS-RC-V1` |
| 2 | `PeticionRef` generada |
| 3 | `OrganizacionRef` |
| 4 | `ExpedienteRef` |
| 5 | `VersionExpediente` |
| 6 | Referencia de la entrada RC |
| 7 | Huella SHA-256 de la entrada RC |
| 8 | Existencia declarada de RC |
| 9 | Número declarado de RC |
| 10 | Fecha declarada de RC |
| 11 | Importe declarado en céntimos |
| 12 | Moneda |
| 13 | Referencia del documento declarado |
| 14 | `SolicitadaEn` |

La respuesta RC repite las cinco coordenadas nominales comunes y coteja además
la referencia y huella de la entrada dentro de `ValidacionRC`. El HMAC de la
petición liga indirectamente el resto de campos de entrada. Una respuesta para
otra organización, expediente, versión, petición o entrada se descarta.

## Preimagen canónica de cálculo de coste

| Orden | Coordenada ligada |
| ---: | --- |
| 1 | Dominio `VEC-CT-FUENTE-ANALISIS-COSTE-V1` |
| 2 | `PeticionRef` generada |
| 3 | `OrganizacionRef` |
| 4 | `ExpedienteRef` |
| 5 | `VersionExpediente` |
| 6 | `CategoriaRef` |
| 7 | `GrupoSubgrupo` |
| 8 | Clave de modalidad |
| 9 | Clave de causa |
| 10 | Inicio del periodo |
| 11 | Fin del periodo |
| 12 | Jornada en diezmilésimas |
| 13 | `SolicitadaEn` |

La respuesta de coste devuelve las coordenadas nominales, la fuente, el
recibo, el importe y `CalculadoEn`. La igualdad del HMAC con la solicitud
impide aplicar el resultado a otras coordenadas de entrada sin que cambie el
sello.

## Límites y cronología

| Regla | Límite |
| --- | --- |
| Tiempo propio máximo de cada operación | 15 segundos |
| Periodo previsto | 100 años exactos como máximo |
| Importe | 922.337.203.685.477 céntimos como máximo |
| Representación monetaria | entero exacto, nunca coma flotante; EUR |
| Instantes | UTC canónico, precisión máxima de microsegundo |

Los límites se validan antes de consultar una dependencia y después de recibir
su resultado. El instante autoritativo del resultado no puede ser anterior a
la petición ni posterior al reloj de finalización de la operación. Tras cada
llamada a generador, sellador, fuente o reloj se comprueba inmediatamente el
contexto; un resultado concurrente con cancelación o vencimiento se descarta.

Una dependencia no disponible nunca equivale a «RC no requerida», «RC
validada» ni coste cero. Si devuelve simultáneamente un valor y un error, el
valor se elimina.

## Motivos minimizados

La fuente no puede introducir el motivo como texto libre. Un resultado
negativo utiliza `MotivoFuenteAnalisis`, compuesto por:

- referencia, versión y huella del catálogo publicado;
- clave de entrada del catálogo;
- clave de mensaje i18n;
- hasta tres parámetros ordenados y sin duplicados.

Los parámetros admiten únicamente las claves técnicas `causa`, `regla` y
`resultado`, con vocabulario técnico permitido y sin identificadores de
personas. Las decisiones funcionales y los textos traducidos permanecen en el
catálogo gobernado; el binario no compila los tipos futuros de RC. La lista
cerrada de parámetros es una defensa de minimización, no un catálogo de
decisiones de negocio.

`String`, `GoString`, formato y `slog` redactan el motivo, la preimagen y las
solicitudes. El detalle diagnóstico completo deberá conservarse, cuando sea
necesario y lícito, tras una referencia opaca en el sistema fuente; no debe
viajar en errores ni registros generales.

## Matriz de comprobación

| Amenaza o defecto | Control y prueba |
| --- | --- |
| Reutilizar una petición en otro expediente | Cruce negativo de expediente y versión |
| Reutilizarla en otra organización | Cruce negativo de organización |
| Sustituir `PeticionRef` o HMAC | Comparación estricta y constante del sello |
| Alterar una coordenada RC | Matriz campo a campo de preimagen/HMAC |
| Alterar categoría, causa, periodo o jornada | Matriz campo a campo de preimagen/HMAC |
| Aceptar un resultado anterior o futuro | Pruebas de los dos extremos y del fin de operación |
| Superar periodo o importe | Pruebas de frontera exacta y `máximo + 1` |
| Continuar tras cancelar una dependencia | Pruebas tras reloj, generador, sellador y fuente |
| Filtrar texto o PII en el motivo | Rechazo de texto libre, allowlist y pruebas de redacción |
| Aceptar interfaz con puntero nulo | Prueba de dependencia nula tipada |
| Incluir causa nula en error compuesto | Salida inválida falla cerrada con error público válido |
| Carrera o alias de punteros | Copias defensivas y `go test -race` |

## Evidencia ejecutada

En el corte documentado se ejecutan:

```text
go test ./internal/modules/contrataciontemporal/ports -count=1
go test -race ./internal/modules/contrataciontemporal/ports -count=1
go vet ./internal/modules/contrataciontemporal/ports
go test ./...
go vet ./...
scripts/comprobar_tamano_ficheros.sh
```

Los dobles de prueba calculan un HMAC-SHA-256 real sobre la preimagen. No
simulan una aceptación por cadena constante.

## Condiciones pendientes antes de integrar o producir

O3-03 no queda habilitado para datos reales ni efectos jurídicos hasta disponer
al menos de:

1. generador criptográficamente seguro de referencias opacas;
2. sellador de producción con secreto custodiado, rotación y separación de
   dominios;
3. reloj y fuentes formalmente designados como autoridades;
4. recibos durables e inmutables ligados a la transacción del expediente;
5. autorización VEC consumible, auditoría y outbox atómicos;
6. adaptadores concretos de presupuesto y cálculo revisados por Sistemas,
   RRHH e Intervención;
7. pruebas de integración y E2E con cancelación, indisponibilidad y
   concurrencia;
8. aceptación funcional y jurídica, EIPD, categorización ENS y medidas
   aplicables aprobadas.

La implementación contribuye a minimización, integridad, exactitud y
trazabilidad, pero no certifica por sí sola RGPD, LOPDGDD, ENS, ENI ni la
legalidad del procedimiento.
