# O4-04 — diseño de sesión TCB PostgreSQL para confirmar una decisión de cobertura

Fecha: 24 de julio de 2026.

Estado: diseño técnico previo a implementación. **No-GO productivo**.

## 1. Propósito y antecedentes

O4-04 debe convertir una `OrdenOperacionDecisionCobertura` opaca en un único
resultado durable, verificable e idempotente. La confirmación positiva debe
registrar la decisión VEC, consumir las evidencias C1, aplicar el CAS del
expediente y escribir decisión, actuación, auditoría, outbox y recibo dentro de
una sola transacción PostgreSQL `SERIALIZABLE`. La confirmación negativa debe
registrar la denegación VEC y dejar una terminación durable, sin aplicar ningún
efecto de concesión.

Este diseño continúa estos antecedentes:

- [orden opaca final de cobertura, `5be6c14`](https://github.com/aavidad/VEC_Diputacion_app/commit/5be6c14);
- [contratos nominales de confirmación y reconciliación, `9238d53`](https://github.com/aavidad/VEC_Diputacion_app/commit/9238d53);
- [acotación documental de la preparación de O4-04, `3f77abe`](https://github.com/aavidad/VEC_Diputacion_app/commit/3f77abe).

Los antecedentes no implementan la transacción productiva. En particular, el
contrato nominal actual no permite que un adaptador PostgreSQL situado en otro
paquete despliegue los campos privados de la orden. Abrir un método `Datos()`
general sobre la orden resolvería el problema de compilación, pero ampliaría la
superficie de confianza a HTTP, CLI, MCP, escritorio y cualquier otro
importador. Este documento sustituye esa salida por una sesión TCB de empuje
controlado.

## 2. Alcance

O4-04 comprende:

1. una frontera de ejecución TCB que no exponga la orden completa;
2. una transacción PostgreSQL única, serializable y de lectura-escritura;
3. el registro VEC positivo o negativo dentro de esa transacción;
4. la revalidación durable del gobierno de cobertura;
5. el consumo durable de todas las evidencias C1 en la rama positiva;
6. el CAS del agregado, la persistencia C2 y la actuación positiva;
7. auditoría, outbox, recibo y terminalidad probatoria;
8. replay exacto de una operación ya terminal;
9. reconciliación conservadora en el primario ante resultado ambiguo;
10. aislamiento estructural y por privilegios frente a canales no confiables.

Quedan fuera de este corte:

- API, pantalla y composición funcional O4-05;
- un cambio de política funcional o del canon ya publicado;
- reintentos funcionales automáticos después de una confirmación ambigua;
- coordinación distribuida entre bases de datos distintas;
- mutación del sistema externo que originó una evidencia C1;
- despliegue, explotación o autorización de producción.

VEC, Contratación Temporal y sus proyecciones de gobierno deben residir en la
misma base de datos PostgreSQL y participar en la misma transacción local. No
se presenta como equivalente una saga, una llamada de red ni una transacción
anidada.

## 3. Estado actual y condición de no-GO

El diseño es viable, pero todavía no existe una implementación productiva
completa. Faltan dos dependencias durables imprescindibles:

| Dependencia | Situación observada | Consecuencia |
| --- | --- | --- |
| Gobierno de cobertura en PostgreSQL | Existen contratos y dobles, pero no una proyección productiva completa de catálogo, política de decisión, política de actuación y puntero actual. | La transacción no puede demostrar que decidió contra publicaciones exactas, vigentes y actuales. |
| Consumo C1 durable en PostgreSQL | Existen órdenes y pruebas en memoria, pero no un consumidor productivo y transaccional. | No puede garantizarse uso único ni ligar de forma durable todas las evidencias a la decisión. |

Por ello, no se debe declarar O4-04 cerrado ni construir un adaptador que
simule estas revalidaciones. El GO de implementación exige cerrar los cortes
O4-04A a O4-04E y superar sus pruebas. El GO productivo requerirá además los
controles operativos, migraciones ensayadas, composición con roles reales y
las revisiones de seguridad aplicables.

## 4. Principios invariantes

La implementación deberá conservar estos invariantes:

- denegación por defecto ante entrada incompleta, inconsistente o no vigente;
- una orden opaca no se serializa, registra, almacena ni entrega a un canal;
- la posesión de la orden y la posesión del ejecutor TCB son capacidades
  separadas;
- el token secreto de propiedad de la reserva nunca se persiste ni atraviesa
  la frontera SQL; se persiste y coteja únicamente su SHA-256;
- concesión y denegación registran VEC dentro de la misma transacción que
  termina la reserva;
- solo una concesión consume C1, aplica CAS y persiste efectos C2;
- ninguna confirmación se considera durable antes del `COMMIT`;
- la última mutación funcional es siempre el puntero de reserva al estado
  terminal, después de haber escrito toda la prueba de la rama;
- el replay exacto de una operación terminal prevalece sobre la caducidad
  posterior de leases o publicaciones;
- un resultado ambiguo solo se reconcilia en el primario y nunca autoriza por
  sí mismo a repetir la orden;
- tablas, funciones internas y funciones VEC genéricas no se conceden a roles
  de canal.

## 5. Arquitectura de empuje

### 5.1. Separación de responsabilidades

El adaptador PostgreSQL no debe implementar directamente
`TransaccionOperacionDecisionCobertura`. El paquete funcional `cobertura`
debe proporcionar una implementación privada de ese puerto nominal y envolver
un ejecutor técnico exportado.

La secuencia conceptual es:

```text
caso de uso
    │ posee Orden opaca
    ▼
transacción nominal privada en cobertura
    │ orden.desplegarEn(sesión), método privado
    ▼
EjecutorSesionTCBOperacionDecisionCobertura
    │ abre pgx.Tx SERIALIZABLE y cede una sesión de vida transaccional
    ▼
SesionTCBOperacionDecisionCobertura
    │ recibe fragmentos cerrados, acotados y en orden
    ▼
función SQL exterior de Contratación Temporal
    │ VEC + gobierno + C1 + CAS + prueba terminal
    ▼
recibo crudo → validación nominal en núcleo → COMMIT
```

El núcleo despliega los campos privados de la orden en la sesión mediante un
método privado. El adaptador conoce únicamente fragmentos ya proyectados. No
conoce la orden, no puede construirla y no decide qué rama ejecutar.

### 5.2. Ciclo de vida de la transacción

1. El ejecutor comienza una transacción `SERIALIZABLE`, de
   lectura-escritura.
2. Configura de forma local `search_path`, zona horaria, `row_security` y
   límites de espera.
3. Crea una sesión de un solo uso ligada a ese `pgx.Tx`.
4. Invoca el callback del núcleo con la sesión.
5. La orden privada se despliega en la sesión en la secuencia permitida para
   su rama.
6. La sesión valida estructura, número, orden y coherencia de los fragmentos.
7. `Confirmar` invoca una única función SQL exterior y devuelve un recibo
   crudo.
8. El núcleo reconstruye el recibo nominal y lo valida contra la orden exacta
   antes de devolver `nil` desde el callback.
9. Solo después de esa validación el ejecutor intenta `COMMIT`.
10. El resultado nominal confirmado se publica únicamente si `COMMIT`
    terminó sin error.

Un error de callback provoca `ROLLBACK`. Un error o pérdida de respuesta
durante `COMMIT` se clasifica como resultado ambiguo y conduce a
reconciliación. El recibo observado antes del `COMMIT` no se filtra como prueba
durable.

### 5.3. Máquina de estados de la sesión

La sesión es mutable solo dentro de la transacción y no es reutilizable:

```text
nueva
  └─ Abrir → abierta
       ├─ grant: Gobierno → VEC positiva → C1{1..512} → Concesión → lista
       └─ deny:  VEC negativa → Denegación → lista
lista
  └─ Confirmar → consumida
cualquier error → inválida
```

Debe rechazar:

- llamadas fuera de orden;
- mezcla de ramas;
- gobierno o C1 en una denegación;
- concesión sin gobierno, sin VEC positiva o sin C1;
- cero evidencias o más de 512 evidencias C1 en la concesión;
- C1 duplicadas o no ordenadas cuando el canon exija orden;
- dos fragmentos terminales;
- una segunda confirmación;
- uso posterior al callback;
- uso concurrente o desde otra goroutine;
- valores no canónicos, nulos tipados y límites excedidos.

La sesión no abre goroutines y no puede escapar del callback.

## 6. Contratos y tipos técnicos

Los nombres definitivos podrán ajustarse durante O4-04A, pero la capacidad
debe mantener esta forma semántica:

```go
type EjecutorSesionTCBOperacionDecisionCobertura interface {
	EjecutarSesionTCBOperacionDecisionCobertura(
		context.Context,
		func(SesionTCBOperacionDecisionCobertura) error,
	) error
}

type SesionTCBOperacionDecisionCobertura interface {
	Abrir(CabeceraSesionTCBOperacionDecisionCobertura) error
	FijarGobierno(GobiernoSesionTCBOperacionDecisionCobertura) error
	FijarDecisionVEC(DecisionVECSesionTCBOperacionDecisionCobertura) error
	AgregarConsumoC1(ConsumoC1SesionTCBOperacionDecisionCobertura) error
	FijarConcesion(EfectoConcedidoSesionTCBOperacionDecisionCobertura) error
	FijarDenegacion(TerminalDenegadoSesionTCBOperacionDecisionCobertura) error
	Confirmar(context.Context) (DatosReciboSesionTCBOperacionDecisionCobertura, error)
}
```

Se exportan únicamente los tipos que el adaptador necesita para implementar
el contrato:

| Tipo | Contenido mínimo |
| --- | --- |
| `CabeceraSesionTCB...` | Versión de esquema, rama, huella de orden, organización, expediente, versión, reserva, referencias congeladas, HMAC semántico, SHA-256 del token, fence, análisis, lease y vigencia. |
| `GobiernoSesionTCB...` | Identidades exactas de catálogo, política de decisión y publicación de actuación que deben revalidarse. Solo concesión. |
| `DecisionVECSesionTCB...` | Solicitud, decisión candidata positiva o negativa, motivo VEC, resultado de contexto esperado, referencias, huellas y resumen esperado. |
| `ConsumoC1SesionTCB...` | Evidencia completa de `DatosOrdenConsumoCobertura` y su resumen pendiente, con identidad de vía y comprobación. |
| `EfectoConcedidoSesionTCB...` | Agregado anterior y siguiente, propuesta exacta, decisión C2, actuación e instantes de efecto y vigencia. |
| `TerminalDenegadoSesionTCB...` | Identidad terminal y datos mínimos de la denegación, sin agregado, propuesta, C1, C2 ni gobierno. |
| `DatosReciboSesionTCB...` | Proyección cruda y acotada del recibo producido por PostgreSQL, todavía no acreditada como resultado nominal. |

Los tipos técnicos no tendrán constructores públicos para sus fragmentos. Solo
el despliegue privado de la orden los construye. Sus implementaciones de
`String`, `GoString`, `Format` y `LogValue` deben ser redactadas. No deben
implementar codecs JSON, texto, binario, Gob, XML o YAML. Si el adaptador
necesita JSON para SQL, lo codifica él desde campos acotados y no mediante un
marshalling genérico de la orden.

Go no ofrece visibilidad «friend». Que los campos imprescindibles sean
legibles por el adaptador no convierte el contrato en una API de canal. El
límite efectivo se obtiene combinando posesión, composición, pruebas de
arquitectura y privilegios PostgreSQL.

## 7. Preparación de la sesión PostgreSQL

La implementación productiva debe comenzar como mínimo con:

```sql
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path = pg_catalog;
SET LOCAL row_security = on;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '15s';
SET LOCAL idle_in_transaction_session_timeout = '20s';
```

Los valores concretos de timeout serán configurables y aprobados por
Sistemas, sin relajar los límites del contrato. La función exterior debe
validar el `session_user`, el aislamiento y el modo de lectura-escritura. Debe
usar nombres cualificados de esquema y una barrera advisory compartida con
migraciones antes de tocar datos funcionales.

La carga completa se valida en forma, versión, cardinalidad, longitudes,
enumeraciones y canones antes de adquirir bloqueos de filas. Ningún valor del
canal decide tablas, funciones, columnas, esquemas ni expresiones SQL.

## 8. Secuencia SQL común

El orden de bloqueos es parte del contrato y debe permanecer igual en todas
las funciones que escriban las mismas familias de datos.

1. Validar rol, aislamiento, versión de carga, límites y canones.
2. Adquirir la barrera advisory compartida de migración.
3. Bloquear, por este orden, la reserva base, su puntero actual y la versión
   actual mediante `FOR UPDATE`.
4. Comprobar primero si la reserva ya es terminal.
5. Si es terminal, aceptar solo un replay exacto:
   - misma huella de orden, reserva, fence y referencias;
   - mismo recibo y marker terminal;
   - mismas filas probatorias exigidas por su rama.
6. Si el replay es exacto, devolver el recibo aun cuando ahora hayan caducado
   el lease o el gobierno. Si existe cualquier divergencia, fallar cerrado.
7. Si no es terminal, exigir:
   - estado reservado;
   - HMAC semántico exacto mediante comparación constante;
   - SHA-256 exacto del token de propiedad;
   - revisión de cercado propietaria;
   - referencias de análisis, VEC y recibo congeladas;
   - agregado anterior congelado cuando corresponda;
   - hora de base de datos anterior al lease y a la vigencia de la orden.

No se revalida primero la actualidad para un replay terminal, porque una
operación legítimamente terminada no se vuelve inválida por el transcurso del
tiempo. Sí se verifica siempre su integridad probatoria.

## 9. Rama de denegación VEC

La denegación no transporta ni puede producir efectos de concesión. Después de
la secuencia común:

1. Invocar el wrapper VEC específico de cobertura con la orden negativa.
2. Registrar la decisión VEC negativa mediante el registrador V3 base.
3. Verificar referencia, correlación, huella, código, rama e instantes
   devueltos contra la candidata sellada.
4. Insertar la auditoría de denegación.
5. Insertar el evento outbox de denegación con referencia reservada y versión
   de origen. Se recomienda conservarlo para que la terminalidad negativa
   también sea observable de forma fiable.
6. Insertar el recibo de denegación.
7. Insertar el marker terminal negativo con claves foráneas al registro VEC,
   auditoría, outbox y recibo.
8. Añadir una versión de reserva `denegada_vec`.
9. Actualizar el puntero de reserva como última escritura.
10. Devolver la proyección cruda del recibo.

Esta rama debe demostrar la ausencia de:

- consumo C1;
- nueva versión del agregado;
- CAS del puntero de agregado;
- decisión C2;
- actuación de concesión;
- publicación de gobierno utilizada como autoridad positiva.

La ausencia se impone mediante la unión cerrada del contrato, restricciones
del marker terminal y pruebas SQL, no mediante campos opcionales ambiguos.

## 10. Rama de concesión

Después de la secuencia común:

1. Bloquear el puntero actual del agregado y la versión anterior exacta
   mediante `FOR UPDATE`.
2. Revalidar el CAS: organización, expediente, versión, JSON/canon, huella e
   identidad O3 deben coincidir con lo congelado.
3. Bloquear mediante `FOR SHARE`, en orden estable:
   - puntero de gobierno actual para organización y acción;
   - publicación exacta del catálogo;
   - política exacta de decisión;
   - política exacta de actuación.
4. Comprobar publicación, vigencia, no retirada, referencias, versiones,
   huellas, unidad, finalidad, fase, estado, acción y correspondencia exacta
   entre las tres publicaciones.
5. Si existe motivo funcional, bloquear y resolver la barrera durable ya
   publicada, comprobando catálogo, versión, huella, entrada, clave i18n y
   actualidad. Si no procede, exigir que el motivo funcional esté ausente.
6. Ordenar de forma determinista todas las claves C1 y adquirir advisory locks
   para las que todavía no puedan bloquear una fila existente.
7. Prevalidar el lote C1 completo:
   - entre 1 y 512 elementos;
   - petición, respuesta, atestación, confirmación TCB, catálogo y verificador;
   - frescura y ventanas temporales;
   - pertenencia exacta a catálogo, vía, comprobación y propuesta;
   - ausencia de duplicados dentro del lote;
   - uso único durable por organización y petición;
   - uso único por autoridad, generación y recibo de respuesta;
   - identidad, orden y huella del lote.
8. Invocar el wrapper VEC específico para registrar y revalidar en vivo la
   concesión.
9. Exigir que el resultado VEC coincida en rama, referencia, correlación,
   huella, código, contexto e instantes; la hora de confirmación debe seguir
   dentro de todas las vigencias.
10. Insertar el lote y cada evidencia C1 consumida.
11. Insertar la versión siguiente del agregado con origen O4/cobertura.
12. Actualizar el puntero del agregado mediante CAS exacto.
13. Insertar la actuación de cobertura.
14. Insertar la decisión gobernada durable C2.
15. Bloquear mediante `FOR UPDATE` el control global de cadenas del
    expediente.
16. Insertar la auditoría específica de cobertura y el evento outbox.
17. Actualizar las cabezas de cadena correspondientes.
18. Insertar el recibo de concesión.
19. Insertar el marker terminal positivo con claves foráneas a VEC, gobierno,
    C1, agregado, actuación, C2, auditoría, outbox y recibo.
20. Añadir la versión de reserva `aplicada`.
21. Actualizar el puntero de reserva como última escritura.
22. Devolver la proyección cruda del recibo.

El instante `efectoEn` ya está sellado por C2 y no se reemplaza por el reloj de
confirmación. La transacción debe exigir:

```text
efectoEn <= confirmadaEn < mínimo de todas las vigencias aplicables
```

## 11. Integración VEC y privilegios

Debe incorporarse un wrapper PostgreSQL específico, de nombre orientativo:

```text
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1
```

El wrapper:

- valida módulo, tipo de recurso, acción y recurso exactos;
- valida la relación entre reserva, expediente y contexto VEC;
- en denegación llama al registrador V3 base, sin producir capacidad positiva;
- en concesión llama a la función de registro y revalidación viva;
- devuelve una proyección acotada que la función exterior CT vuelve a
  verificar;
- participa en la misma transacción exterior, sin `COMMIT` propio.

Solo el rol ejecutor de Contratación Temporal recibirá `EXECUTE` sobre este
wrapper. No se le concederá:

- ejecución del registrador VEC genérico;
- acceso directo a tablas VEC;
- posibilidad de seleccionar una función distinta desde la carga;
- privilegios al rol HTTP, CLI, MCP, escritorio o usuario final.

La función exterior CT será `SECURITY DEFINER`, con propietario dedicado,
`search_path` seguro y privilegios mínimos. Los propietarios de tablas no se
utilizarán como roles de conexión de la aplicación.

## 12. Proyecciones y migraciones recomendadas

### 12.1. Gobierno de cobertura

Antes de la transacción compuesta debe existir una proyección append-only con:

- publicación exacta de catálogo de vías, su canon y huella;
- publicación exacta de política de decisión ligada al catálogo;
- publicación exacta de política de actuación ligada a organización, acción,
  catálogo y política;
- finalidades, unidad, fase, estado, vigencias, motivos VEC y equivalencias de
  motivos cuando estén publicadas;
- `gobierno_cobertura_actual(organizacion_ref, accion)` como puntero a la
  publicación exacta;
- checkpoint de sincronización y barrera de actualidad;
- funciones SQL de canon y huella compatibles con los canones Go.

Todas las tablas serán inmutables, con RLS y `FORCE ROW LEVEL SECURITY`. El
ejecutor solo accederá a resolutores y barreras `SECURITY DEFINER`.

### 12.2. Reserva y terminalidad

Se recomiendan:

- `reserva_operacion_decision_cobertura`, base inmutable;
- `alias_operacion_decision_cobertura`, para ámbitos idempotentes;
- `reserva_operacion_decision_cobertura_version`, historial de estado y fence;
- `reserva_operacion_decision_cobertura_actual`, puntero;
- `confirmacion_operacion_decision_cobertura`, recibo crudo, huella de recibo
  y huella de orden;
- `terminal_operacion_decision_cobertura`, marker final con unión y
  restricciones específicas para concesión y denegación.

Separar el recibo del marker final permite escribir las pruebas sin referencias
circulares. El marker se inserta cuando todas las filas de la rama existen y
su conjunto de claves foráneas demuestra completitud.

### 12.3. Consumo C1

Se recomiendan:

- `consumo_cobertura_lote`;
- `consumo_cobertura_evidencia`.

Como mínimo se impondrá unicidad sobre:

```text
(organizacion_ref, peticion_ref)
(autoridad_ref, generacion, recibo_respuesta_ref)
```

El lote conservará identidad, orden, canon y huella. Cada evidencia conservará
la prueba local suficiente para reproducir la validación: petición, resultado,
atestación, confirmación TCB, catálogo, verificador y resumen minimizado. El
consumo es un registro durable local; no implica mutar el proveedor externo.

### 12.4. Efectos y prueba

Se reutilizarán cuando su semántica sea suficiente:

- `expediente_version_integral`;
- `expediente_integral_actual`;
- `actuacion_expediente_integral`;
- `control_cadenas_expediente_integral`;
- `outbox_expediente_integral`.

Se incorporarán:

- `decision_cobertura_gobernada_durable`;
- `auditoria_decision_cobertura`.

La auditoría genérica existente ligada por claves foráneas al análisis O3 no
debe forzarse para una semántica diferente. La auditoría de cobertura
conservará referencias y huellas probatorias sin duplicar PII innecesaria.

Todas las familias nuevas tendrán RLS, `FORCE ROW LEVEL SECURITY`, triggers de
inmutabilidad y permisos de tabla revocados a los roles ejecutores. El único
camino de escritura será la función exterior.

## 13. Reconciliación en el primario

La solicitud nominal actual conserva internamente la huella de la orden, pero
su proyección pública de coordenadas no la entrega. Añadir un getter público
reabriría la misma fuga que se evita en confirmación. Por tanto, la
reconciliación necesita una segunda frontera de empuje opaco, orientativamente:

```go
type EjecutorLecturaPrimariaTCBOperacionDecisionCobertura interface {
	EjecutarLecturaPrimariaTCBOperacionDecisionCobertura(
		context.Context,
		func(SesionLecturaPrimariaTCBOperacionDecisionCobertura) error,
	) error
}
```

El núcleo despliega en esa sesión las coordenadas minimizadas y la huella
privada de la orden. El adaptador ejecuta una transacción
`SERIALIZABLE READ ONLY` contra el primario y decodifica un resultado
estrictamente acotado.

Solo se devuelve «confirmada» cuando:

- el puntero de reserva es terminal;
- marker, recibo y huella de orden coinciden;
- registro VEC, auditoría y outbox existen y están ligados;
- en concesión existen y coinciden gobierno, lote C1 completo, agregado,
  actuación y decisión C2;
- en denegación existen las pruebas negativas y no existen efectos propios de
  concesión.

Se devuelve «no concluyente» cuando la operación está ausente, sigue
reservada, el primario no está disponible o hay divergencia probatoria. «No
concluyente» no significa rollback demostrado ni permite repetir la orden.

Los errores `40001` o `40P01` solo podrán reintentarse de forma acotada si el
ejecutor demuestra que ocurrieron antes de intentar `COMMIT`, usando la misma
orden inmutable. Desde que se haya enviado la función SQL con resultado de
transporte ambiguo, o se haya intentado `COMMIT`, el único camino es
reconciliar. Una operación funcional posterior podrá adquirir una reserva
nueva después de la caducidad y recomponer una orden nueva, pero esa decisión
pertenece al caso de uso superior, no al reconciliador.

## 14. Aislamiento frente a HTTP, CLI, MCP y escritorio

El aislamiento se aplica por capas:

1. La orden permanece como variable local del caso de uso y nunca forma parte
   de un DTO o respuesta.
2. Solo la implementación privada de la transacción nominal recibe la orden.
3. La raíz de composición inyecta el ejecutor PostgreSQL directamente en ese
   wrapper; no existe un registro de plugins seleccionable por solicitud.
4. Los handlers reciben servicios de aplicación de alto nivel, nunca la
   orden, el ejecutor ni una sesión TCB.
5. HTTP, CLI, MCP y escritorio no importan ni implementan símbolos TCB.
6. Los roles de conexión de esos canales carecen de `EXECUTE` sobre la función
   exterior y sobre el wrapper VEC, y carecen de privilegios de tabla.
7. Los logs y errores no incluyen token, HMAC, evidencia completa, PII ni
   fragmentos de orden.

La ubicación bajo `internal` ayuda frente a módulos externos, pero no crea un
paquete amigo entre núcleo y PostgreSQL porque ambos comparten ancestro. Las
pruebas de arquitectura y la lista permitida de implementadores son, por
tanto, controles obligatorios.

## 15. Estrategia de pruebas

### 15.1. Contrato Go

- máquina de estados válida para concesión y denegación;
- rechazo de mezcla de ramas y de llamadas fuera de orden;
- límites cero, uno, 512 y 513 para C1;
- rechazo de duplicados, orden alterado y nulos tipados;
- una sola confirmación y prohibición de reutilización o escape;
- recibo crudo alterado, de otra orden o de otra rama;
- `String`, `GoString`, `Format` y `LogValue` sin material sensible;
- ausencia de codecs genéricos;
- cancelación antes del inicio y durante el callback;
- error antes de `COMMIT` frente a error ambiguo de `COMMIT`.

### 15.2. Arquitectura

Pruebas AST y `go list` deben:

- prohibir importaciones y referencias TCB/orden desde HTTP, CLI, MCP y
  escritorio;
- comprobar que los constructores de handlers no reciben capacidades TCB;
- permitir implementaciones productivas del ejecutor y sesión únicamente en
  el adaptador PostgreSQL incluido en una lista cerrada;
- permitir dobles solo en ficheros `_test.go`;
- comprobar mediante aserción de compilación el único implementador
  productivo esperado;
- impedir la aparición de `Datos()`, codecs o serialización sobre la orden;
- detectar SQL en el núcleo y lógica funcional duplicada en el adaptador.

### 15.3. PostgreSQL 18

- aislamiento y modo de transacción incorrectos;
- rol o `search_path` incorrectos;
- replay terminal exacto tras caducidad;
- colisión semántica con misma clave idempotente;
- token hash, HMAC, fence, lease o referencias alterados;
- carreras entre dos propietarios y entre dos decisiones del mismo agregado;
- catálogo, política o actuación retirados durante la operación;
- motivo funcional incorrecto o retirado;
- duplicidad C1 dentro del lote y entre transacciones;
- lotes de 1 y 512 evidencias y rechazo de 0/513;
- deadlocks evitados por orden estable de locks;
- denegación sin ninguna fila positiva;
- concesión con todas las filas y FKs esperadas;
- fallo inyectado después de cada escritura, comprobando rollback total;
- fallo antes y después del registro VEC;
- fallo antes del puntero terminal;
- recibo o marker parcial imposible;
- RLS, `FORCE RLS`, revocaciones y `SECURITY DEFINER`;
- usuario de canal incapaz de ejecutar funciones o leer tablas internas;
- canon y huellas SQL idénticos a los vectores Go.

### 15.4. Reconciliación y fallos de red

- respuesta SQL perdida antes de `COMMIT`;
- respuesta de `COMMIT` perdida con operación aplicada;
- operación ausente, reservada, concedida y denegada;
- marker, recibo o cualquiera de sus dependencias alterados;
- lectura contra réplica prohibida o no concluyente;
- ningún reintento después de resultado ambiguo;
- replay exacto posterior devuelve el mismo recibo y no duplica efectos.

## 16. Plan de implementación O4-04A a O4-04E

| Corte | Contenido | Dependencias | Salida verificable |
| --- | --- | --- | --- |
| **O4-04A** | Contratos push, wrapper nominal privado, despliegue privado de la orden, sesión falsa estricta y pruebas de arquitectura/redacción. | Antecedentes `5be6c14`, `9238d53`, `3f77abe`. | La orden puede desplegarse sin `Datos()`; ningún canal obtiene capacidades TCB. No hay SQL productivo todavía. |
| **O4-04B** | Proyección PostgreSQL durable de gobierno, puntero actual, checkpoint, barrera, canones, RLS e integración PG18. | O4-04A solo para los DTO exactos que se revalidarán. | Catálogo, política y actuación se resuelven y revalidan de forma autoritativa. **Prerrequisito duro de O4-04E.** |
| **O4-04C** | Reserva, aliases, versiones, punteros, recibo, marker terminal, preparador y lector primario durable. | O4-04A. | Propiedad cercada, replay y consulta primaria funcionan sin efectos funcionales. |
| **O4-04D** | Consumo C1 durable por lote, validadores canónicos, locks ordenados, wrapper VEC específico y privilegios mínimos. | O4-04A y contratos O4-02; puede avanzar en paralelo con B/C donde no comparta migraciones. | Uso único C1 y registro VEC compuesto quedan disponibles dentro de una transacción exterior. **Prerrequisito duro de O4-04E.** |
| **O4-04E** | Función SQL compuesta, ejecutor PostgreSQL, efectos grant/deny, reconciliación, inyección de fallos, concurrencia y revisión independiente. | O4-04A, B, C y D cerrados. | Confirmación y reconciliación productivas a nivel de código y PostgreSQL; todavía no equivalen a autorización de producción. |

Orden crítico:

```text
O4-04A ─┬─ O4-04B ─┐
        ├─ O4-04C ─┼─ O4-04E
        └─ O4-04D ─┘
```

B y D son dependencias duras. C también debe estar terminado antes de E, pero
puede desarrollarse en paralelo con B y D después de fijar A. Ningún corte
debe introducir una función exterior provisional que omita una de estas
dependencias y se contabilice como transacción completa.

## 17. Criterios de aceptación y GO

O4-04 podrá considerarse implementado cuando:

- A, B, C, D y E estén integrados y revisados;
- no exista `Datos()` ni codec de la orden;
- concesión y denegación produzcan una prueba terminal completa en una sola
  transacción;
- VEC se registre dentro de esa misma transacción;
- una denegación no pueda producir ningún efecto positivo;
- una concesión revalide gobierno y consuma todo C1 antes de aplicar C2;
- el puntero terminal sea la última escritura funcional;
- el recibo se valide contra la orden antes de intentar `COMMIT`;
- toda ambigüedad se reconcilie en el primario sin reintento ciego;
- las pruebas Go, arquitectura, PostgreSQL 18, concurrencia, fallo inyectado y
  privilegios estén verdes;
- una revisión independiente no encuentre vías de elusión.

Hasta entonces, el estado permanece **diseño técnico / no-GO productivo**. Este
documento no afirma implementación, integración E2E, despliegue ni aptitud para
producción.
