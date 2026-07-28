# O4-05 C2-D2: plan de ejecución de consultas RRHH

## Propósito

Este documento dirige el cierre de las consultas protegidas de Contratación
temporal. El resultado esperado es una vertical real y transportable:

```text
API / escritorio / CLI / MCP
            │
            ▼
servicio de aplicación
            │
            ▼
puerto SesionConsultaRRHH
            │
            ▼
adaptador PostgreSQL
            │
            ▼
función nominal exterior
            │
            ├─ consume una VEC-AD-3 nueva
            ├─ revalida identidad y autoridad
            ├─ lee una proyección minimizada
            ├─ registra acceso y alcance
            └─ emite o consume cursor cuando procede
```

La interfaz web seguirá usando los mismos servicios de aplicación. No se crea
una vía privilegiada específica para HTTP y no se aceptan cookies, cabeceras o
campos de formulario como fuente de identidad.

## Estado de partida

Están cerrados y publicados:

- consumidores nominales VEC-AD-3 de cuadro y detalle;
- rol técnico consultor RRHH;
- registro durable de accesos;
- publicación 1:1 con corte global;
- familia, emisión, consumo y revocación de cursores opacos.

Estas piezas son precondiciones probadas, pero aún no forman una consulta
productiva. Falta una función exterior que las coordine en la misma
transacción y un adaptador Go que solo entregue datos después de confirmar el
`COMMIT`.

## Hallazgos que cambian el diseño inicial

### Registrador de acceso

`registrar_acceso_rrhh_interno_v1` exige la versión uno del control de
consultas. La migración `000038` deja ese control en versión dos. Una fachada
nueva no puede reutilizar el registrador histórico.

Se añadirá un registrador nominal v2. Su contrato tendrá control propio y
estable, de modo que una migración posterior de cuadro o detalle no lo
invalide. La migración histórica `000036` no se modifica.

### Dos huellas distintas

La huella de consulta representa la petición exacta y contiene el cursor. Cada
página debe tener por tanto una huella de consulta distinta.

La huella de familia representa filtros y límite, pero excluye el cursor. Debe
permanecer idéntica durante toda la paginación. Ambos cánones serán
versionados y tendrán vectores comunes Go/PostgreSQL.

### Identidad viva

La pertenencia de `session_user` al grupo técnico solo identifica al
componente que accede a PostgreSQL. La identidad de la persona procede de la
autenticación y del contexto de actor atestados.

La función exterior deberá cotejar y revalidar, sin confiar en parámetros
sueltos:

- autenticación y su huella;
- sesión;
- control de sesión, revisión y huella;
- actor;
- perfil y versión;
- organización y ámbito.

El diseño reutilizará `vec_identidad_sesiones_v1`; no creará otra fuente de
verdad de identidades.

### Replay y revalidación final

Los consumidores VEC-AD-3 pueden reconocer un consumo anterior. Esa rama no
autoriza una segunda entrega de datos. La función exterior exigirá
`consumo_nuevo=true` antes de leer.

Además, el consumidor actual no revalida en vivo su rama de replay. Se añadirá
un revalidador final nominal, sin capacidad de consumo, que coteje las diez
piezas persistidas y vuelva a comprobar clave, configuración, raíz, decisión,
sesión y revocaciones después de formar el resultado.

### `COMMIT` ambiguo

El token siguiente solo se devuelve una vez y PostgreSQL conserva únicamente
su SHA-256. Si la conexión se corta durante el `COMMIT`, el adaptador:

1. no devuelve datos ni cursor;
2. no reintenta la misma capacidad;
3. borra de memoria la respuesta provisional;
4. devuelve un error genérico;
5. obliga a obtener autorización nueva y reiniciar desde la primera página.

Recuperar ese token exigiría persistencia cifrada y gobierno de claves. Queda
fuera de este corte.

## Orden de trabajo

| Corte | Producto | Estado | Dependencias | Paralelizable |
|---|---|---|---|---|
| C2-D2-A | Contrato Go completo | GO en `9655d62` | C2-D1 | Sí |
| C2-D2-B | Revalidador final VEC-AD-3 | GO en `183e563` | consumidores comunes | Sí |
| C2-D2-C | Identidad, registrador v2 e índice *as-of* | GO en `0dc2edc` y `c286af1` | C2-D1 | Cerrado |
| C2-D2-D | Fachada nominal de detalle | Pendiente | A, B y C | Con E |
| C2-D2-E | Fachada nominal de cuadro y cursores | Pendiente | A, B y C | Con D |
| C2-D2-F | Adaptador PostgreSQL Go | Pendiente | D y E | No |
| C2-D2-G | Composición raíz, transportes y E2E | Pendiente | F | No |

Cada producto tendrá un productor y una revisión independiente. Ningún
dictamen se apoyará solo en las aserciones o los scripts del productor.

## C2-D2-A — contrato Go

Estado: `GO` técnico independiente en `9655d62`, con revisión separada, pruebas
normales, detector de carreras y `go vet`.

Entregables:

- contexto no serializable con control de sesión y versión de perfil;
- huella de filtros/familia separada de la huella de consulta;
- órdenes que exporten exclusivamente material tipado;
- recibos capaces de cotejar toda la evidencia PostgreSQL;
- límites y redacción de registros;
- vectores canónicos y pruebas de mutación.

La consulta exacta conserva texto, estado, fase, límite y cursor. La familia
conserva texto, estado, fase y límite.

## C2-D2-B — revalidador final común

Se añadirá `autorizacion_atestada_v3/migraciones/000005` con dos funciones
nominales, una para cuadro y otra para detalle. Cada función:

- será nominal para las consultas RRHH;
- recibirá las mismas diez piezas exactas;
- no consumirá ni escribirá auditoría;
- comprobará que el consumo almacenado coincide byte a byte;
- rechazará una decisión inexistente o no consumida;
- revalidará clave, gobierno, configuración, raíz y decisión vivas;
- devolverá solo evidencia mínima de revalidación.

El propietario de Contratación temporal recibirá `EXECUTE` únicamente sobre el
overload exacto. El runtime no obtendrá ese permiso.

## C2-D2-C — acceso v2 e identidad

Estado: `GO` técnico. Identidad quedó cerrada en `0dc2edc` y CT `000039`
se integró en `c286af1` tras PostgreSQL 18.4 y dos revisiones independientes.

Entregables:

- una frontera nominal de `identidad_sesiones_v1`, restringida a superficie
  corporativa, garantía alta y cuenta ordinaria;
- control estable del contrato de acceso v2;
- registrador de acceso compatible con cursores;
- revalidación de la sesión y del control de sesión actuales;
- cotejo exacto del perfil y su versión;
- índice para resolver la última versión de cada expediente dentro del corte;
- migraciones ascendentes y reversión segura;
- pruebas de ACL, deriva, identidad cruzada y carreras de revocación.

No se guardarán token, texto de filtro, material VEC ni datos personales en
claro.

La evolución SQL prevista queda separada para permitir revisión y reversión:

| Migración | Responsabilidad |
|---|---|
| Identidad `000001` específica de CT | Revalidación nominal de sesión corporativa |
| Autorización atestada `000005` | Revalidación final de consumo de cuadro/detalle |
| CT `000039` | Registrador de acceso v2 e índice as-of |
| CT `000040` | Contrato interno: tipos, cánones y controles, sin lectura |
| CT `000041` | Contrato probatorio: contenido, estados y recibo V2, sin lectura |
| CT `000042` | Ejecución interna: lectura *as-of* y confirmación de cursores |
| CT `000043` | Fachadas exteriores, orquestación atómica y privilegio mínimo |

Las barreras CT avanzarán de `18/2` a `19/3`, `20/4`, `21/5`, `22/6` y
`23/7`. Cada
reversión exigirá su estado exacto, cero dependencias posteriores y ausencia
de historia que quedaría huérfana. No se usará `CASCADE`.

Esta partición sustituye la previsión inicial de dos migraciones. La revisión
previa detectó que mezclar contrato, lectura, cursores y fachada obligaba a
comprimir una migración por encima del límite de revisión y favorecía leer
antes de consumir la autorización. La secuencia obligatoria será:

```text
canon y preflight puros
  → consumo VEC-AD-3 nuevo
  → revalidación de identidad
  → lectura protegida
  → registro de acceso
  → confirmación de cursor
  → revalidación final de identidad y VEC
```

## C2-D2-D — detalle

La fachada de detalle:

- fija acción, finalidad, audiencia y tipo de recurso;
- exige una VEC-AD-3 nueva;
- trata ausente, ajeno, denegado y versión incorrecta de forma indistinguible;
- interpreta versión cero como la versión actual coherente;
- registra exactamente un acceso y un alcance;
- revalida al final;
- devuelve como máximo 256 KiB, profundidad 24 y 16.384 nodos.

La entrada funcional solo contiene el canon versionado de detalle y el alcance
tipado. Acción, finalidad, audiencia, módulo y tipo de recurso son constantes
de la función, nunca opciones del llamador.

## C2-D2-E — cuadro

La consulta resuelve primero la última versión de cada expediente dentro del
`corte_global` y después aplica ámbito y filtros. El orden total es:

```text
actualizado_en DESC,
expediente_ref COLLATE "C" DESC
```

Se consultan `limite + 1` filas, se devuelven como máximo `limite` y el límite
del cursor siguiente es la última fila entregada, nunca la centinela.

La familia mantiene corte, identidad, ámbito, filtros, límite y un TTL máximo
no deslizante de cinco minutos. Cada continuación necesita una VEC-AD-3 nueva.

La función recibe las diez piezas VEC-AD-3 más:

```json
{
  "consulta": {
    "dominio": "vec.contratacion_temporal.consulta_rrhh.cuadro.v1",
    "version": 1,
    "texto": "",
    "estado_clave": "",
    "fase_clave": "",
    "limite": 25,
    "cursor": ""
  },
  "alcance": {
    "organizacion_ref": "…",
    "clase_ambito": "organizacion",
    "ambito_ref": "…"
  }
}
```

PostgreSQL reconstruye la misma huella de consulta y la representación
canónica del recurso autorizable. El JSON no puede ampliar la capacidad: solo
puede coincidir exactamente con lo ya atestado.

## C2-D2-F — adaptador Go

El adaptador:

- inicia una transacción explícita `SERIALIZABLE READ WRITE`;
- usa parámetros preparados y funciones nominales;
- no construye JSON de auditoría ni identidad;
- valida estructura, límites, evidencia y ámbito antes del `COMMIT`;
- confirma el `COMMIT` antes de entregar el resultado;
- normaliza errores para no crear oráculos;
- nunca reintenta una capacidad ni un cursor tras un resultado ambiguo;
- funciona igual para web, escritorio, CLI y MCP.

## C2-D2-G — composición y E2E

La composición raíz inyectará el adaptador real. Los transportes solo
traducirán DTO y errores; no resolverán identidad ni permisos por su cuenta.

La prueba E2E deberá recorrer al menos:

- cuadro primera página;
- continuación y fin de familia;
- detalle actual y versión explícita;
- ámbito de organización, centro y unidad;
- revocación, expiración, replay y carrera;
- cancelación y `COMMIT` ambiguo;
- acceso desde HTTP y desde un transporte no web;
- representación visual con estado de carga, vacío, denegación y éxito.

## Límites obligatorios

| Recurso | Límite |
|---|---:|
| Capacidad | 5 s |
| Familia de cursores | 5 min, no deslizante |
| `statement_timeout` | 4 s |
| `lock_timeout` | 1 s |
| Inactividad en transacción | 6 s |
| Filas publicadas | 100 |
| Resultado | 256 KiB |
| Profundidad JSON | 24 |
| Nodos JSON | 16.384 |

Los filtros `_` y `%` nunca actuarán como comodines. Si el contrato Go no
admite un carácter, PostgreSQL también deberá rechazarlo.

## Puerta de cierre

C2-D2 solo obtendrá `GO` cuando:

- una ejecución exitosa produzca exactamente un consumo y auditoría VEC, un
  acceso y un alcance, más los efectos de cursor necesarios;
- replay, error o cancelación no dejen una historia parcial;
- las carreras reales de consumo y revocación tengan un único orden
  observable;
- no se encuentren secretos, cursores, filtros, material VEC o PII en tablas,
  WAL lógico de prueba, logs ni trazas;
- PostgreSQL 18.4, Go, composición y E2E pasen con revisión independiente.

Hasta entonces continúa el `NO-GO` productivo y no cambia el porcentaje
oficial del procedimiento.
