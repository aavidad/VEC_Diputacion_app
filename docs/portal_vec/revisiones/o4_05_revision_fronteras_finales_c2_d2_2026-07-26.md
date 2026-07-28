# O4-05: revisión de las fronteras finales C2-D2

## Dictamen

Las dos fronteras comunes que preceden a las fachadas de consulta de
Contratación temporal obtienen `GO` técnico independiente:

- revalidación final del consumo VEC-AD-3, commit `183e563`;
- revalidación de identidad corporativa, commit `0dc2edc`.

Este dictamen no autoriza producción. Todavía faltan el registrador de acceso
v2, el motor tipado de lectura, las fachadas exteriores, el adaptador Go y el
E2E desde los transportes.

## Revalidación final VEC-AD-3

La migración `autorizacion_atestada_v3/000005` añade dos funciones nominales:

- cuadro RRHH;
- detalle de expediente RRHH.

Ambas reciben exactamente las diez piezas originales. La función interna:

- reconstruye y valida capacidad, decisión y contexto;
- coteja byte a byte el material con la atestación ya consumida;
- exige consumo y auditoría durables;
- recalcula la huella del consumo;
- no inserta, no consume de nuevo y no avanza cadenas;
- impide usar un consumo de cuadro en detalle o al contrario;
- revalida decisión, contexto, RBAC y vigencias;
- comprueba de nuevo revocaciones de clave, configuración y raíz;
- devuelve únicamente decisión, huella de consumo e instante.

### Superficie y privilegios

- funciones `SECURITY DEFINER`;
- propietario NOLOGIN de autorización atestada;
- `search_path=pg_catalog`;
- helper interno sin permiso para Contratación temporal;
- fachadas nominales ejecutables únicamente por el propietario NOLOGIN de
  Contratación temporal;
- runtime y grupo consultor sin ejecución directa;
- transacción `SERIALIZABLE READ WRITE`, UTC y límites ambientales.

### Evidencia reproducida

La puerta
`deploy/postgresql/autorizacion_atestada_v3/probar_consultas_rrhh_v3_pg18_4.sh`
se ejecutó por el productor y por un revisor independiente sobre PostgreSQL
18.4 fijado por digest.

Cubrió:

- subida, bajada limpia y reinstalación;
- ACL y dependencia que bloquea la retirada;
- consumo durable sin nuevas escrituras;
- alteración individual de las diez piezas;
- cruce nominal cuadro/detalle;
- deriva RBAC posterior al consumo;
- revocación posterior al consumo;
- regresión de los consumidores anteriores.

Hashes revisados:

| Fichero | SHA-256 |
|---|---|
| `000005 up` | `5f5d94da2f06e844b299907b886e3de704f2f88ccd790e0aad2344c08e16366b` |
| `000005 down` | `7f16e4efadc35da3e09c3f783a017d7dc37043ffd1b96b573b9e0b434e6f35f5` |
| prueba SQL | `0cef454fae1623292e8fad205ac2e25f7a040513bda92d761fdb5f62fac1c30f` |
| runner | `916969b33bdcd75f85ab25b1ee7047047608639786a249f955de0d78b1d9097b` |

## Revalidación de identidad corporativa

La migración específica
`contratacion_temporal/migraciones_identidad/000001` reutiliza
`vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1`. No crea una
segunda fuente de verdad de identidad.

La función nominal:

- devuelve los diecinueve campos de la autoridad base;
- añade `login_tecnico=session_user` para la cadena probatoria;
- solo acepta superficie `interna_corporativa`;
- exige garantía `alto`;
- rechaza cuentas privilegiadas;
- exige que cuenta efectiva y ordinaria coincidan;
- identifica el LOGIN PostgreSQL como componente técnico, no como persona.

### Topología del LOGIN

La función exige que el LOGIN:

- sea nominativo y pueda iniciar sesión;
- no sea superusuario, creador de roles/bases, replicador ni `BYPASSRLS`;
- herede una única membresía directa;
- pertenezca únicamente al grupo consultor RRHH;
- no tenga opción administrativa ni `SET ROLE`;
- no forme parte de puentes o topologías indirectas.

La función rica permanece propiedad de Identidad. Solo el propietario NOLOGIN
de Contratación temporal recibe `USAGE` de esquema y `EXECUTE`; el runtime no
recibe permisos de tablas ni ejecución directa.

### Evidencia reproducida

La puerta
`deploy/postgresql/contratacion_temporal/probar_o4_05_identidad_consulta_rrhh_pg18_4.sh`
se ejecutó por productor, dirección y revisor independiente sobre PostgreSQL
18.4 fijado por digest.

Cubrió:

- instalación mínima de autoridades reales;
- alta de cuenta y sesión mediante operaciones del dominio de Identidad;
- resultado de veinte campos y LOGIN técnico exacto;
- rechazo de ejecución directa;
- rechazo de una sesión ajena;
- rechazo de una segunda membresía;
- ausencia de privilegios directos sobre tablas;
- retirada bloqueada por una fachada dependiente;
- retirada final con `RESTRICT` y sin `CASCADE`.

Hashes finales:

| Fichero | SHA-256 |
|---|---|
| `000001 up` | `bc7dcd79dfa961d9f45ed42e51438433cd5199eaca10e840d01c51cc71a014cb` |
| `000001 down` | `4db8523a2767d9cff4bf0293363f584f9e20e29b99483eb12b0a12458bdac0b8` |
| prueba SQL | `dbe52af96b144cc18964e4fc8d1d84452bcbc90c51e71f5f93b79621caad244a` |
| runner | `8441fe36243fefa4727789ded7d3d21d57a0addc14e8cc869311261779d1895c` |

## Comprobaciones auxiliares

- `bash -n`: verde;
- `shellcheck`: verde;
- `git diff --check`: verde;
- `gitleaks` sobre cada conjunto preparado: sin hallazgos;
- ningún fichero nuevo supera 800 líneas;
- contenedores de prueba efímeros y sin red.

## Trabajo siguiente

Actualización del 28 de julio: CT `000039` quedó integrado en `c286af1` con
referencias opacas de Identidad —sin persistir el LOGIN nominativo—, índice
organización–expediente–corte, reversión protegida, PostgreSQL 18.4 y doble
`GO` independiente.

El corte inmediato vigente es el motor tipado `000040`; después se
implementarán las dos fachadas exteriores `000041`, el adaptador Go y la
composición/E2E.

## Métrica

Estas fronteras eliminan bloqueos de seguridad de C2-D2, pero todavía no
cierran una vertical visible y productiva. Se mantienen:

- Contratación temporal: `19/46`, 41 % oficial;
- O4-05: `3/5` hitos;
- Bolsa: `1/14`, 7 % oficial;
- producción: `NO-GO`.
