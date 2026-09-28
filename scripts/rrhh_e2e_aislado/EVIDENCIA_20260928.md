# Evidencia local del arnés aislado (28-09-2026)

Base de trabajo: `trabajo/piden-rrhh-corte2-20260928`@`da48a409b7e75249fa1b2378d612a90998aa25bd`.
Todos los ensayos Docker usaron PostgreSQL 18.4 desechable, loopback y
`/dev/shm`; los contenedores se retiraron. No se usaron datos, servicios ni
secretos de la base principal.

## Ejecuciones observadas

| Comando | Resultado | Alcance |
| --- | --- | --- |
| `bash scripts/rrhh_e2e_aislado/run.sh --smoke` | Fallo de arranque VEC: faltan conexiones PostgreSQL separadas de CT para consulta, motivos, registro y revalidación de identidad/contexto. | Go compiló; sin listener, HTTP ni Chrome. |
| `bash scripts/rrhh_e2e_aislado/run.sh --preflight` | PostgreSQL 18.4 arrancó y aplicó migraciones iniciales; la cola SQL se detuvo en las precondiciones descritas abajo. | Sin fixture de negocio, API ni navegador. |
| `DOCKER_HOST=tcp://127.0.0.1:1 DOCKER_CONTEXT=invalid-remote-context bash scripts/rrhh_e2e_aislado/run.sh --preflight` | Llegó al mismo bloqueo SQL local tras fijar el socket Unix. | Verifica que el arnés ignora un destino Docker remoto heredado. |
| `navegador.py` contra HTTPS/mTLS sintético con PKCS#12, CA propia y servidor de prueba temporal | HTTP 200 a 1440/390, sin JS/cookies/storage/overflow; acceso sin certificado denegado. | Verifica la mecánica Chrome del arnés, **no VEC**. |
| `navegador.py --bolsa-ref bolsa:sintetica:1` con API sintética que devuelve 403 | Retorno 1; política HTTP 403 a 1440/390. | Verifica fallo cerrado de la sonda, **no B47 real**. |
| `bash -n` y `shellcheck -S warning` sobre los scripts; compilación Python de `navegador.py` | Verdes al cerrar la edición. | Verificación estática, no E2E. |

El último preflight identificó:

- ContextoActor `000007_alcance_proyecciones_empleado.up.sql`: contrato
  Personal 000016 ausente o divergente. Los roles históricos 000005 ya se
  instalaron en el último ensayo.
- AD3 `000011_consumidor_bolsa_llamamiento_v3_atestada.up.sql`: gobierno de
  audiencias incompatible. Falta el UP canónico AD3-7 en este checkout; están
  sus componentes, sin wrapper completo.
- Bolsa `000003_integracion_desarrollo.up.sql`: permiso denegado sobre esquema
  AD3, dependiente de la secuencia anterior.
- CT `000035_recuperacion_propia_cobertura_o4_05.up.sql`: estado anterior
  incompatible para recuperación propia O4-05.

El preflight se repitió tras corregir el orden de ContextoActor y añadir el
rol histórico; el bloqueo avanzó hasta Personal 000016. Ningún fallo se
resolvió alterando migraciones de producto, insertando historia o concediendo
permisos amplios. La puerta de `roles.sh` también detectó que un solo LOGIN
de gobierno necesitaría `SET LOCAL ROLE` sobre cuatro autoridades distintas;
no crea cuentas ni exporta DSN sin un contrato nominal aprobado.

**No hay E2E acreditado**: faltan instalación SQL completa, LOGINs nominales,
historia CT/Bolsa creada por casos de uso, publicación B47, arranque de VEC,
consultas positivas/denegadas, Chrome y recuperación tras reiniciar.
