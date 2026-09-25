# F4b: acceso completo de Dietas (once cuentas)

Sustituye a [F4](../f4_reactivacion_dietas/README.md) y a
[D7](../d7_personal_acceso/README.md), que no constan aplicados en la
principal y quedaron incompatibles con la composición actual: F4 exige
exactamente ocho LOGIN y se detiene ante cualquier otro que alcance los grupos
Dietas, pero `vec-server` necesita además la auditoría de frontera de Dietas y
las dos cuentas de Personal. **No aplicar F4 ni D7 si se usa F4b.** Si alguno
se hubiera aplicado, F4b se detiene en su preimagen (el puntero V3 ya no sería
el `v2` de P6) y hace falta un análisis DBA, no un reintento.

En una sola transacción F4b:

1. comprueba la preimagen P6 (`v2` revocada, ocho cuentas R1D en `NOLOGIN`
   con contraseña SCRAM conservada, sin sesiones) y crea `v3` activa de la
   asignación V3 R1D con emisión y vigencia nuevas (mismo plan Go que F4, con
   actor y acto propios);
2. crea, o adopta si D7 ya las preparó en `NOLOGIN` con la forma exacta, las
   tres cuentas nuevas con su verificador SCRAM;
3. activa `LOGIN` en las once cuentas.

| LOGIN | Única membresía (`INHERIT TRUE, SET FALSE, ADMIN FALSE`) |
| --- | --- |
| `vec_dietas_r1d_{registro_identidad,revalidacion_identidad,contexto,fuente_autorizacion,registro_autorizacion,motivos,dietas,personal}_desarrollo` | las mismas que F4 |
| `vec_dietas_f4b_auditoria_frontera_desarrollo` | `vec_dietas_registrador_frontera` |
| `vec_personal_d7_asignacion` | `vec_personal_d7_ejecutor` |
| `vec_personal_d7_auditoria_frontera` | `vec_personal_registrador_frontera` |

La cuenta de auditoría Dietas queda fuera del prefijo `vec_dietas_r1d_` para no
alterar los recuentos de P6/F4. Las variables y el contador de conexiones están
en [`03_entorno.md`](../03_entorno.md#dietas-con-la-composición-actual-variables-y-f4b-25092026).

## Puertas que detienen la transacción

- PostgreSQL 18, DBA por socket local, base `postgres`; cero sesiones de las
  once cuentas (`ALTER ROLE` no corta sesiones: drenar y cerrar antes).
- Exactamente ocho cuentas con prefijo `vec_dietas_r1d_`; ninguna cuenta con
  atributos privilegiados, `connlimit`/caducidad, ACL directa, propiedad,
  miembros, más de una membresía o `SET TRUE`/`ADMIN`.
- Grupos técnicos sin `LOGIN`, sin pertenecer a otro rol (p. ej.
  `pg_read_all_data`) y sin ser propietarios.
- Ningún otro `LOGIN` no superusuario con ruta a `vec_dietas_ejecutor`,
  `vec_dietas_registrador_frontera`, `vec_personal_d7_ejecutor` o
  `vec_personal_registrador_frontera`.
- Lista positiva mínima de objetos y ACL (la vertical R1D de F4, las cinco
  fachadas D7/auditoría de Personal y `registrar_auditoria_frontera_comision_v2`),
  ajustada al estado real tras Dietas 000001-000011: el ejecutor crea con
  `crear_o_recuperar_comision_catalogada_v2` (Dietas 000006 le revoca
  `…_calculada_v1`), no se exige `leer_concesion_historica_contexto_actor_v3`
  (solo la usa Contratación temporal) y el `USAGE` del ejecutor en
  `vec_autorizacion_atestada_v3`, sin el que la fachada de rutas de AD3-50 da
  42501, exige instalar antes AD3-81 (incluida en el paquete incremental
  `04_dietas_migraciones.sh --incremental`).
- Segregación por esquema: cada grupo solo puede tener `CONNECT` en la base y
  `USAGE`/`SELECT`/`EXECUTE` en sus esquemas (Dietas: `vec_dietas`,
  `vec_personal`, `vec_autorizacion_atestada_v3`; Personal D7 y su auditoría:
  `vec_personal`; auditoría Dietas: `vec_dietas`; identidad, contexto y
  autorización: el suyo). En `vec_autorizacion_atestada_v3` el ejecutor solo
  puede tener el `USAGE` de AD3-81 y el `EXECUTE` de la fachada de rutas, sin
  `SELECT` de tablas ni columnas. Escritura directa de tablas, CT/Bolsa, otra
  base o `CREATE`/`TEMP` detienen F4b.
- Ninguna concesión a `PUBLIC` en la base ni en esquemas, relaciones o
  funciones `vec_`.

La lista exacta por función de Personal no se repite aquí: la acredita el
arranque (`internal/app/bootstrap/dietas_postimagen_personal.go`) con huellas
de cuerpo y ACL exactas. `CONNECT` del grupo D7, que Personal 000012 no
concede, lo concede F4b dentro de la misma transacción.

## Secretos

- `VEC_F4B_ESTADO`: JSON privado `{"version":1,"contrasenas":{...}}`, fichero
  regular del ejecutor, modo `0600`, fuera de Git. `--preparar-estado` lo crea
  con tres contraseñas aleatorias y **nunca** lo sobrescribe ni rota.
- La contraseña no llega al servidor: el ejecutor deriva un verificador
  SCRAM-SHA-256 (4096 iteraciones, sal aleatoria) y lo pasa por stdin como
  variable psql en un temporal `0600` del directorio de evidencia que se borra
  al salir. Nada viaja por argumentos ni aparece en la evidencia (el inventario
  solo indica si hay verificador SCRAM).
- El verificador sí va en el texto de sentencias de la transacción. Por eso
  `comun.sql` fija con `SET LOCAL` `log_statement='none'`,
  `log_min_error_statement='panic'`, los registros por duración y muestreo,
  `pg_stat_statements.track='none'` y `auto_explain.log_min_duration=-1`, y
  el cliente psql usa `VERBOSITY terse`/`SHOW_CONTEXT never`. Las sentencias
  dinámicas `CREATE/ALTER ROLE … PASSWORD` convierten su error en uno sin
  CONTEXT interno. El ensayo PG18 activa `log_statement='all'` en el servidor
  y comprueba que ningún verificador ni su base64 llega al registro.
- `--sonda-tls` puede incluir en el estado las contraseñas R1D para probar las
  once cuentas; las tres nuevas son obligatorias. La conexión positiva exige
  `verify-full` y `require_auth=scram-sha-256` (psql/libpq ≥ 16): una línea
  `trust`, `password` o `md5` en `pg_hba` hace fallar la sonda. La negativa
  (`sslmode=disable`) no lleva `require_auth` para que cualquier entrada sin
  TLS cuente como fallo.

## Uso

```bash
export VEC_F4B_EVIDENCIA_DIR=<privado 0700> VEC_F4B_ESTADO=<privado>/f4b-estado.json
# transporte: VEC_F4B_POSTGRES_CONTAINER + VEC_F4B_PG_SOCKET_DIR + VEC_F4B_PG_PORT,
# o PGSERVICE/PGSERVICEFILE/PGPASSFILE privados (socket local, base postgres).
bash deploy/principal/f4b_acceso_dietas/ejecutar.sh --preparar-estado
bash deploy/principal/f4b_acceso_dietas/ejecutar.sh --inventario
bash deploy/principal/f4b_acceso_dietas/ejecutar.sh --rollback
VEC_F4B_APLICAR=SI-F4B-REVISADO bash deploy/principal/f4b_acceso_dietas/ejecutar.sh --commit
VEC_F4B_TLS_HOST=<nombre del certificado> VEC_F4B_TLS_PORT=<puerto> VEC_F4B_TLS_CA=<ca privada> \
  bash deploy/principal/f4b_acceso_dietas/ejecutar.sh --sonda-tls
```

`--sonda-tls` conecta cada cuenta del estado con `sslmode=verify-full` y exige
`pg_stat_ssl.ssl`; además exige que la misma cuenta **no** pueda entrar sin TLS
(`pg_hba` debe tener `hostssl … scram-sha-256` y rechazar `host` para ellas).
Requiere `psql` en el equipo desde el que se sondea. Cada ejecución inventaría
antes y después; tras pérdida de respuesta, consultar el inventario privado
antes de decidir. Una repetición se rechaza sin escritura.

Retirada (selector apagado, cero sesiones, puntero `v3` de F4b exacto):

```bash
bash deploy/principal/f4b_acceso_dietas/ejecutar.sh --retirar-rollback
VEC_F4B_APLICAR=SI-F4B-REVISADO bash deploy/principal/f4b_acceso_dietas/ejecutar.sh --retirar-commit
```

Crea `v4` revocada y vuelve las once cuentas a `NOLOGIN`, conservando roles,
verificadores, membresías e historia. No usar P6 de nuevo ni `DOWN`.

## Ensayo aislado

`VEC_F4B_TEST_BASE=<directorio_privado_0700> bash deploy/principal/f4b_acceso_dietas/probar_pg18.sh`

PostgreSQL 18.4 desechable sin red, con los datos en `/dev/shm/vec-pg-f4b-<pid>`
montado con `-v` (sin volumen anónimo) y borrado al terminar, CA y certificado
de servidor sintéticos y
`pg_hba` solo `hostssl` para las once cuentas. Reproduce R1D + P6, crea stubs
de las firmas requeridas y prueba: estado `0600` sin sobrescritura; estado con
permisos abiertos, LOGIN ajeno con ruta a Personal D7 o al ejecutor Dietas,
`SET TRUE`, novena cuenta R1D, ACL a CT, escritura directa, `SELECT` de tabla
o columna AD3 al ejecutor, ascenso a `pg_read_all_data`, ACL requerida ausente,
`PUBLIC` y cuenta R1D sin contraseña rechazados sin efecto; `ROLLBACK` limpio
(ni siquiera el `CONNECT` del grupo D7); adopción de una cuenta D7 ya
preparada; `COMMIT` con once LOGIN; reentrada denegada; sonda TLS 11/11 con
`verify-full` y rechazo sin TLS y con CA ajena; retirada `v4` con testigos
CT/Bolsa intactos. Acredita el paquete, no la aplicación ni la web.
