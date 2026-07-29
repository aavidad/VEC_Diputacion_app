# Decisión de motivos nominales para consultas RRHH

Fecha: 30 de julio de 2026.

## Problema

El contrato `ResolutorMotivoConsultaRRHH` distingue cuadro y detalle, pero no
existe todavía una autoridad gobernada que determine qué referencia completa
de motivo corresponde a cada operación.

PostgreSQL VEC puede validar históricamente una
`ReferenciaEntradaCatalogo`, pero esa validación no autoriza a elegir:

- la versión máxima;
- la única entrada activa;
- una etiqueta;
- una coordenada de configuración;
- una referencia incrustada en código.

Cualquiera de esas alternativas introduciría una autoridad implícita y
rompería la denegación predeterminada.

## Decisión

VEC publicará una proyección append-only con dos vinculaciones nominales y
distintas:

```text
consulta de cuadro RRHH  → referencia completa de catálogo
consulta de detalle RRHH → referencia completa de catálogo
```

La publicación y retirada conservarán historia, versión, huella, vigencia,
actor técnico y prueba de la referencia VEC subyacente. Las resoluciones serán
dos operaciones SQL nominales sin selector libre y devolverán la referencia
cero ante ausencia, ambigüedad, retirada o incoherencia.

Hasta que RRHH, DPD y Jurídico aprueben la finalidad, se mantienen dos motivos
distintos. No se presupone que puedan compartir una entrada.

## Minitareas y números reservados

| ID | Migración | Alcance |
| --- | --- | --- |
| M1.1 | `000008` | Fundamento privado: historial append-only, checkpoint, referencias, RLS, ACL e inmutabilidad. |
| M1.2 | `000009` | Publicación y retirada atómicas de las dos vinculaciones, con replay idempotente. |
| M1.3 | `000010` | Dos resoluciones nominales, ACL mínima y pruebas de vigencia/retirada. |
| M2 | Adaptador Go | Implementación de los dos métodos M0 con consultas fijas y pool exclusivo. |

Cada migración será completa, reversible de forma segura y probada en
PostgreSQL 18.4 real. No se confirmarán componentes intermedios que dejen una
migración rota.

## Separación de autoridades

El adaptador M2 usará un pool nominal exclusivo de evaluación de motivos. No
reutilizará `PoolConsultasRRHHPostgreSQL`, porque esa conexión posee autoridad
sobre las proyecciones de Contratación temporal y no sobre el gobierno VEC.

Los métodos serán exactamente:

```text
ResolverMotivoCuadroRRHH(contexto, instante)
ResolverMotivoDetalleRRHH(contexto, instante)
```

No aceptarán catálogo, clave, organización, acción, finalidad ni cualquier
otro selector.

## Dependencias externas

El código y las pruebas con referencias sintéticas pueden completarse ya. La
activación real continúa bloqueada por:

- publicación aprobada del catálogo y de sus referencias;
- validación de finalidades por RRHH, DPD y Jurídico;
- repositorio maestro durable de catálogos y proyector de eventos;
- roles nominativos, TLS, certificados, secretos externos y ejecución de
  migraciones por Sistemas/DBA;
- prueba integral con infraestructura y catálogos reales.

## Consecuencia

M1 y M2 pueden avanzar en paralelo con A5 porque sus write-sets no se cruzan.
Su cierre técnico no autoriza producción ni sustituye las conformidades
organizativas.
