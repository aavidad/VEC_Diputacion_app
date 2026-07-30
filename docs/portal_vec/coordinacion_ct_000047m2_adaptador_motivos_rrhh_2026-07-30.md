# Coordinación CT-000047M2: adaptador de motivos RRHH

Fecha: 30 de julio de 2026.

Estado: diseñado y bloqueado hasta integrar M1.3/`000010` con PostgreSQL 18.4
verde y revisión independiente P0=0/P1=0.

## Resultado

Implementar `ports.ResolutorMotivoConsultaRRHH` con un adaptador PostgreSQL y
un pool nominal exclusivos. No se reutilizan el evaluador V2 ni
`PoolConsultasRRHHPostgreSQL`, porque poseen autoridades distintas o más
amplias.

Las únicas operaciones de negocio serán:

```text
ResolverMotivoCuadroRRHH(contexto, instante)
ResolverMotivoDetalleRRHH(contexto, instante)
```

No aceptan catálogo, entrada, organización, acción, finalidad, clase ni
cualquier otro selector.

## Minitareas y write-set

### M2.1 — pool y acreditación

```text
internal/modules/contrataciontemporal/adapters/postgres/
  fabrica_pool_resolucion_motivos_rrhh.go
  acreditacion_pool_resolucion_motivos_rrhh.go
  fabrica_pool_resolucion_motivos_rrhh_test.go
```

El pool:

- recibe contexto, DSN y nombre de `LOGIN` nominal separados;
- exige que el usuario de la DSN coincida exactamente con el nombre;
- aplica las primitivas TLS ya acreditadas por CT-000046;
- rechaza réplica, TLS inseguro, grupo `NOLOGIN` como usuario, configuración
  mutable o identidad distinta entre `session_user` y `current_user`;
- acredita un `LOGIN` seguro y una única membresía directa en
  `vec_autorizacion_motivos_rrhh_resolutor`, con
  `ADMIN FALSE`, `INHERIT TRUE` y `SET FALSE`;
- rechaza cualquier otra membresía, autoridad, privilegio directo o acceso a
  tablas y secuencias;
- fija propietario, firma, configuración, comentarios y ACL de las dos
  funciones M1.3;
- posee cierre idempotente y no expone `*pgxpool.Pool`.

### M2.2 — adaptador nominal

```text
internal/modules/contrataciontemporal/adapters/postgres/
  resolucion_motivos_rrhh_postgresql.go
  resolucion_motivos_rrhh_postgresql_test.go
```

El constructor público acepta solo el pool M2 real. Una frontera interna
estrecha admite dobles para las pruebas sin abrir composición productiva.

Habrá dos constantes SQL completas, no un generador ni un nombre de función
parametrizable. Cada una llama con un único `$1::timestamptz` a su fachada
nominal, limita a dos filas y devuelve cardinalidad junto con los cuatro
campos. Así se distinguen `0`, `1` y `>1`; `QueryRow` no puede ocultar una
función degradada que produzca varias referencias.

Cada operación:

1. rechaza receptor, dependencia o contexto nulos, incluso nulos tipados;
2. conserva cancelación o plazo agotado sin consultar;
3. exige instante UTC, años 1..9999 y precisión de microsegundos;
4. adquiere una conexión física;
5. abre transacción `SERIALIZABLE READ WRITE`;
6. reacredita identidad, membresía, ACL y manifiesto;
7. ejecuta únicamente su consulta literal;
8. exige cardinalidad uno y una referencia V2 válida;
9. confirma antes de devolver la referencia;
10. revierte y devuelve referencia cero ante toda incertidumbre.

No se usa `READ ONLY`: M1.3 necesita bloqueos `FOR SHARE`. La ausencia de DML
se garantiza mediante ACL, no mediante una opción incompatible.

### M2.3 — PostgreSQL real

```text
internal/modules/contrataciontemporal/adapters/postgres/
  resolucion_motivos_rrhh_postgresql_integracion_test.go

deploy/postgresql/autorizacion/
  probar_adaptador_resolucion_motivos_rrhh_m2_pg18_4.sh
```

El runner instala la cadena completa hasta `000010`, crea un `LOGIN` sintético
miembro únicamente del rol M1.R y ejecuta pruebas reales sobre PostgreSQL 18.4.
No imprime ni conserva DSN, contraseña o certificados.

### M2.4 — integración

Dirección obtiene revisión independiente, integra los commits pequeños,
actualiza estado y abre después la composición raíz. M2 no modifica bootstrap,
web ni dominio.

## Salida y errores

La única salida positiva es una
`domain.ReferenciaEntradaCatalogo` que supere
`ReferenciaMotivoAutorizacionV2Valida`.

Toda ausencia, ambigüedad, dato inválido o fallo de infraestructura devuelve
la referencia cero y conserva:

```text
ports.ErrMotivoConsultaRRHHNoDisponible
```

Cancelación y plazo agotado se unen al centinela. No se propagan mensajes,
SQLSTATE, DSN, nombres internos ni errores del controlador. La versión se
escanea como `int64`, se limita a `1..MaxInt32` y después se convierte a
`int`.

## Matriz mínima

Unitarias del pool:

- DSN inválida, usuario distinto y grupo `NOLOGIN`;
- TLS inseguro, nombre de servidor incorrecto y configuración mutable;
- identidad efectiva distinta, réplica o `LOGIN` elevado;
- membresía ausente, adicional o con opciones incorrectas;
- privilegio directo, función ausente, sobrecargada o degradada;
- fallo parcial y cierre idempotente.

Unitarias del adaptador:

- consultas literales distintas para cuadro y detalle;
- cero, una y dos filas;
- todos los campos inválidos y versión desbordada;
- contexto nulo, cancelado y vencido;
- errores de adquisición, acreditación, consulta, escaneo y `COMMIT`;
- reversión y referencia cero en todos los fallos;
- saneamiento de mensajes y recuperación de pánico;
- concurrencia sobre una instancia bajo `go test -race`.

PostgreSQL 18.4:

- referencia distinta y positiva para cada fachada;
- ausencia, retirada local, retirada V2 y fuera de vigencia;
- usuario no miembro o con autoridad adicional;
- imposibilidad de lectura y DML directo;
- `search_path` hostil y homónimos temporales;
- reinicio, reconexión y degradación posterior detectada;
- resolución frente a retirada concurrente;
- tres ejecuciones limpias.

## Puertas

Cada minitarea exige pruebas focales, `go test -race`, `go vet`, formato,
`git diff --check`, límite de 800 líneas y Gitleaks. M2.3 añade PostgreSQL
18.4 tres veces. La integración requiere P0=0 y P1=0 de un revisor que no haya
producido el código.
