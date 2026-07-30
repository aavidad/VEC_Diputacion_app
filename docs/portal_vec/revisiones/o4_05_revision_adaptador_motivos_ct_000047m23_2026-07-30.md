# Revisión PostgreSQL real del adaptador de motivos M2.3

Fecha: 30 de julio de 2026.

Resultado final: **GO**, con `P0=0`, `P1=0` y `P2=0` en las revisiones
funcional y de seguridad independientes.

Commit integrado:

```text
bd22d05f481e8d8fa7a8b1d77c80475ce087e2c9
```

## Alcance acreditado

M2.3 ejecuta el pool M2.1 y el adaptador M2.2 reales contra PostgreSQL 18.4
fijado por digest. El runner crea una instancia sin red, instala la cadena
completa hasta `000010`, usa un `LOGIN` resolutor con una única membresía
nominal y publica datos exclusivamente sintéticos mediante un `LOGIN`
proyector separado.

La matriz verifica:

- ausencia y referencias distintas de cuadro y detalle;
- retirada nominal, retirada V2 y fin de vigencia gobernado por el reloj de
  PostgreSQL;
- resolución frente a retirada concurrente, con orden causal observado
  mediante `pg_blocking_pids`;
- pérdida de membresía, autoridad adicional, privilegio directo y deriva de
  ACL posterior a construir el pool;
- lectura, `INSERT`, `UPDATE`, `DELETE`, `TRUNCATE`, `SET ROLE` y creación
  temporal denegados con `SQLSTATE 42501`;
- `search_path` hostil y homónimo en `pg_temp` sobre la misma conexión física,
  sin desviar las consultas cualificadas;
- terminación de conexiones, reconexión y reinicio del servidor conservando
  el mismo proceso y pool;
- referencia cero y texto de error opaco exacto ante toda indisponibilidad.

## Hallazgos corregidos antes del cierre

Las primeras ejecuciones identificaron defectos solo en el arnés:

1. la DSN de diagnóstico incluía parámetros exclusivos de `pgxpool`;
2. el `LOGIN` resolutor tenía un `CONNECT` directo incompatible con su
   autoridad heredada;
3. las publicaciones nominales se intentaban realizar sin el `LOGIN`
   proyector exigido;
4. un predicado con `pg_terminate_backend` podía ser reordenado y terminar la
   conexión administrativa.

Se corrigieron sin relajar producción: DSN separadas, autoridad heredada,
proyector real y selección materializada de PID antes de terminar conexiones.
Las revisiones iniciales detectaron además la ausencia de carrera causal,
homónimo temporal y reloj PostgreSQL; los tres huecos quedaron cubiertos antes
del commit final.

## Evidencia

Superaron:

```text
bash deploy/postgresql/autorizacion/\
probar_adaptador_resolucion_motivos_rrhh_m2_pg18_4.sh
# tres ejecuciones completas, cada una con contenedor y base limpios

go test -race ./internal/modules/contrataciontemporal/adapters/postgres -count=1
go test ./internal/modules/contrataciontemporal/... -count=1
go test ./... -count=1
go vet ./internal/modules/contrataciontemporal/...
go vet ./...
bash -n deploy/postgresql/autorizacion/\
probar_adaptador_resolucion_motivos_rrhh_m2_pg18_4.sh
shellcheck deploy/postgresql/autorizacion/\
probar_adaptador_resolucion_motivos_rrhh_m2_pg18_4.sh
git diff --check
gitleaks stdin
```

Un directorio vacío ajeno, `/tmp/.git`, contaminaba inicialmente la detección
de material criptográfico efímero de las pruebas globales. Se retiró con
`rmdir`; la repetición completa quedó verde. No era un defecto del código.

Tamaños y huellas del corte:

| Fichero | Líneas | SHA-256 |
| --- | ---: | --- |
| `resolucion_motivos_rrhh_postgresql_integracion_test.go` | 800 | `4895274faf221e9d6d5d37a352bebbfdccef1f713b6ca5def60db8007ecdf477` |
| `probar_adaptador_resolucion_motivos_rrhh_m2_pg18_4.sh` | 306 | `73013f40ec9d1703f6191a7eef70f7b7776514fa9fd0647845890c231efca0d0` |

Tras cada ejecución no quedaron contenedores, procesos ni directorios
temporales M2.3. No se registraron DSN, contraseñas ni secretos.

## Límite del cierre

M2 acredita la resolución nominal real de motivos, pero no compone todavía la
raíz interna ni sustituye las autoridades corporativas. Producción conserva
`NO-GO` hasta cerrar autoridad/PDP, composición, TLS/mTLS viva, E2E y
conformidades formales.
