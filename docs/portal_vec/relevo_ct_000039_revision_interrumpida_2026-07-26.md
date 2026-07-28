# Relevo de la revisión interrumpida de CT 000039

## Ubicación

- rama: `integracion/ct-000039-20260726`;
- base estable: `f744102`;
- candidato inicial recuperado: `e2bd078`;
- cierre técnico: `e558f41`;
- estado actual: `GO` técnico doble; pendiente de promoción a la rama estable.

## Corrección incorporada en esta sesión

`registrar_acceso_rrhh_interno_v2` ya no depende de que la barrera global de
consultas permanezca exactamente en versión tres. Usa el singleton estable del
registrador v2 y se añadió una prueba transaccional con barreras futuras
`20/4`.

El productor informó de un runner PostgreSQL 18.4 verde tras este cambio. La
foto modificada no recibió aún la reproducción final independiente.

## Bloqueos pendientes

### Minimización

El candidato persiste `login_tecnico` en claro. Un LOGIN nominativo puede ser
un dato personal y contradice el plan C2-D2, que prohíbe datos personales en
claro en este registro.

Debe sustituirse por `cuenta_ref` y `cuenta_ordinaria_ref` opacas procedentes
de la autoridad de Identidad. La prueba debe acreditar que el nombre del LOGIN
no aparece en tablas, cánones ni recibos.

### Safe-down

La reversión todavía no congela el catálogo semántico completo. Debe detectar,
como mínimo, deriva en:

- relaciones y columnas;
- restricciones, incluidas FK y `CHECK`;
- índices;
- políticas RLS y sus expresiones;
- ACL de tablas y columnas;
- disparadores propios e internos RI;
- reglas y dependencias posteriores.

Se reutilizará el patrón probado de la migración `000038`. La prueba añadirá
deriva estructural, de política, disparador y ACL, y comprobará que no se
destruye ninguna extensión desconocida.

### Tamaño

El fichero ascendente tiene 800 líneas exactas. Las correcciones pendientes no
deben superar el límite; se compactará o se separará prueba auxiliar sin
reducir garantías.

## Foto preservada

Hashes antes del commit de relevo:

| Fichero | SHA-256 |
|---|---|
| `000039 up` | `e6a4122442d0d21a8a54647b2a9c378403b706f090be92a8f4187742102a92ad` |
| `000039 down` | `e4156ba424ebc251c3cd2d2a9ec57b8241ace75da5f156d6f80e64f951a9d868` |
| prueba SQL | `ab362f8f743c9d9ed19c10327a7d5b7329c2bafa9cc81fd55048679dfe689ab8` |
| runner | `379530c58ec628bc5f1f61df41df4e6f14e8bdbc49668a36a26b151be5c910d6` |

## Reanudación

1. Corregir minimización y safe-down.
2. Ejecutar `bash -n`, `shellcheck`, `git diff --check` y `gitleaks`.
3. Ejecutar el runner PostgreSQL 18.4 fijado por digest.
4. Congelar hashes finales.
5. Encargar a dos revisores independientes seguridad y cobertura probatoria.
6. Solo con ambos `GO`, crear el commit definitivo y promoverlo a la rama
   estable.

La métrica oficial permanece en Contratación temporal `19/46` (41 %), O4-05
`3/5`, Bolsa `1/14` (7 %) y producción `NO-GO`.

## Segundo cierre interrumpido

Dirección ordenó detener el desarrollo y preservar el estado el 26 de julio
de 2026. Queda un avance parcial, expresamente **no integrable**:

- la migración ascendente sustituye el `login_tecnico` nominativo por
  `cuenta_ref` y `cuenta_ordinaria_ref` opacas procedentes de Identidad;
- el fichero conserva 800 líneas y su SHA-256 es
  `13c1db8e477df871ec7614fcb994fab75ce41da5a3ec46888c468d2ba60377de`;
- la prueba ejecutable todavía consulta y modifica la columna retirada
  `login_tecnico`, por lo que debe adaptarse antes de volver a ejecutar la
  matriz PostgreSQL;
- el `down` exhaustivo y las inyecciones de deriva continúan pendientes.

No se ejecutó ni se declaró verde ninguna prueba funcional sobre esta foto. Al
reanudar, deben corregirse primero el runner y la reversión segura, probar la
ausencia literal del LOGIN en tablas, cánones y recibos y reproducir toda la
matriz PostgreSQL 18.4 antes de solicitar revisión independiente.

## Cierre final del 28 de julio

El commit `e558f41` completa el trabajo interrumpido sin modificar el hash de
la migración ascendente minimizada:

- `login_tecnico` deja de persistirse y se sustituye por referencias opacas de
  Identidad;
- la reversión compara el catálogo semántico completo y usa exclusivamente
  operaciones `RESTRICT`;
- las pruebas ejercitan cruces de identidad y carreras reales de registro y
  revocación;
- una ranura lógica anterior al tráfico adverso acredita el tránsito de cuatro
  centinelas y la ausencia posterior de texto claro o hexadecimal en WAL,
  logs, recibos y trazas;
- todos los rechazos comprueban SQLSTATE y mensaje exactos.

La ejecución de dirección y las dos revisiones independientes terminaron en
verde sobre los hashes congelados. La evidencia completa está en
[la revisión final de CT 000039](revisiones/o4_05_revision_registrador_v2_ct_000039_2026-07-28.md).

Este cierre habilita CT `000040`, `000041` y `000042`; no cambia la métrica
oficial ni autoriza producción.
