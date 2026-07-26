# O4-05: revisión del consumo y registro de accesos RRHH

Fecha: 26 de julio de 2026

Ámbito: C2-A y C2-B del consumidor PostgreSQL de consultas internas

Estado: GO técnico; no autoriza producción

## Resultado

Dos cortes aditivos han quedado probados en PostgreSQL 18.4 y revisados por
agentes distintos de sus productores:

- `a0d39c1`: consumidores nominales VEC-AD-3 de cuadro y detalle;
- `2820759`: rol consultor y registro durable de accesos RRHH.

No se ha elevado el porcentaje oficial de Contratación temporal. Aún faltan
la lectura con corte estable, las funciones exteriores de Contratación, el
adaptador Go, identidad/PDP, HTTP, composición y E2E.

## C2-A — consumidores comunes VEC-AD-3

Las migraciones `000003` y `000004` del adaptador común:

- evolucionan la lista de audiencias en dos pasos verificables;
- conservan intacta la fachada nominal de alta;
- separan cuadro y detalle por audiencia, acción, tipo de recurso y finalidad;
- reciben las diez piezas originales del conjunto probatorio;
- cotejan hashes, contexto, decisión, efecto, HMAC, configuración y raíz;
- aceptan únicamente DER-SPKI Ed25519 canónico;
- consumen la decisión y añaden un eslabón de auditoría en la misma
  transacción;
- devuelven referencia y huella reales de auditoría;
- permiten recuperar el recibo exacto sin insertar ni reparar historia.

La función interna exige una transacción `SERIALIZABLE`, de escritura, UTC,
con límites armados y un LOGIN nominativo miembro directo y exclusivo del rol
consultor. El LOGIN no puede ejecutar directamente las fachadas comunes.

Un replay devuelve `consumo_nuevo=false`. Ese resultado solo sirve para
conciliar un `COMMIT` ambiguo: C2-C deberá rechazarlo antes de leer o entregar
datos.

## Evidencia C2-A

El runner
`deploy/postgresql/autorizacion_atestada_v3/probar_consultas_rrhh_v3_pg18_4.sh`
se ejecutó de forma independiente con PostgreSQL 18.4 fijado por resumen.
Superó:

- instalación, cuatro fotografías exactas de restricciones y reversión;
- CHECK adulterado y `down` protegido;
- regresión funcional de alta;
- ACL, pertenencia nominal y cruces cuadro/detalle;
- rollback sin atestación, consumo ni auditoría residual;
- recibo completo, replay sin crecimiento y colisión múltiple;
- adulteración aislada de las diez piezas;
- nulos, X25519, RSA y DER no canónico;
- caducidad durante la revalidación RBAC;
- los dos órdenes lineales de consumo y revocación;
- bloqueo de retirada por claves, punteros o historia.

`bash -n`, ShellCheck, `git diff --check` y el límite de 800 líneas quedaron
verdes. `000003.up` ocupa 791 líneas.

## C2-B — registro durable de acceso

La migración `000036` de Contratación temporal:

- exige la barrera global 15 y la eleva a 16 al confirmar;
- crea control propio, cabeza de cadena y registro de accesos;
- fuerza RLS y concede autoridad solo al propietario;
- mantiene la función de inserción sin `EXECUTE` para `PUBLIC` ni consultor;
- diferencia cuadro y detalle mediante listas positivas;
- conserva referencias opacas y huellas autoritativas, no respuestas ni
  documentos;
- encadena cada prueba canónica con SHA-256;
- prohíbe actualización, borrado y truncado;
- impide la reversión cuando existe historia;
- restaura la barrera 15 únicamente tras una retirada limpia.

El delta de rol crea un grupo `NOLOGIN`. Cada cuenta nominativa debe ser un
LOGIN mínimo con una única membresía directa, `ADMIN FALSE`, `INHERIT TRUE` y
`SET FALSE`. También se rechaza que ese LOGIN se use como rol puente para
propagar transitivamente el permiso.

## NO-GO intermedio y corrección

La primera revisión de C2-B emitió NO-GO por dos defectos reproducidos:

1. `000035.down` podía retirar una dependencia antes que C2-B;
2. un segundo LOGIN podía heredar el consultor a través del LOGIN nominativo.

No se hizo commit. La corrección añadió la barrera global 16, estados
coherentes en el delta y cierre de la herencia saliente. La segunda revisión
comprobó:

- `15→16→15` sin estado parcial;
- rechazo de `000035.down` con C2-B vacío y con historia;
- rechazo de estados `16/sin C2-B`, `15/con C2-B` y control divergente;
- rechazo de puente simple y cadena transitiva de dos niveles;
- dos `up` concurrentes y dos `down` concurrentes con un único ganador;
- ocho escritores concurrentes con secuencia continua y cadena íntegra;
- retirada final sin ACL ni dependencias residuales.

## Evidencia C2-B

El runner
`deploy/postgresql/contratacion_temporal/probar_o4_05_registro_accesos_pg18_4.sh`
terminó verde en ejecuciones del productor, la coordinación y el revisor
independiente. También quedaron verdes `bash -n`, ShellCheck,
`git diff --check` y el límite de 800 líneas. `000036.up` ocupa 772 líneas.

## Límites deliberados

C2-B no crea cursor ni lector. `registrada_en` no es un corte de paginación:
dos transacciones pueden confirmar en orden distinto al de sus relojes o de
una secuencia asignada antes de `COMMIT`.

C2-C deberá añadir, sin reescribir migraciones históricas:

- un corte global monotónico y único alineado con el orden de confirmación;
- funciones exteriores separadas para cuadro y detalle;
- consumo VEC, lectura y registro de acceso en una única transacción;
- rechazo de replay antes de producir datos;
- conciliación opaca separada;
- límites de filas, bytes y tiempo;
- pruebas de paginación concurrente, revocación, rollback y reinicio.

Después seguirán C2-D —adaptador Go y pool acreditado— y C2-E —TLS, fallos,
revisión del TCB completo y E2E real—.

## Puerta de producción

Este documento acredita dos componentes, no el recorrido. Producción conserva
NO-GO hasta que identidad, autorización, PostgreSQL, API y web demuestren
juntos el mismo caso real, con EIPD, ENS, operación, copias y aprobaciones
organizativas cerradas.
