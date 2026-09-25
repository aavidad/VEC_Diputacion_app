# F4: acceso de Dietas R1D para el recorrido visible

> **Sustituido por [F4b](../f4b_acceso_dietas/README.md) (25/09/2026).** No consta
> aplicado en la principal y no encaja con la composición actual de Dietas,
> que necesita once cuentas. No aplicarlo junto con F4b.

Paquete administrativo para la vertical **crear, listar y obtener una comisión propia con recibo recuperable**. No instala migraciones, no modifica el rol V3 ni arranca la aplicación. La retirada P6 sigue intacta: F4 solo admite su puntero revocado `v2`, crea `v3` activa con emisión y vigencia nuevas, y habilita los ocho LOGIN nominales en la misma transacción. `plan.go` usa `AsignacionPerfil.Validar` y `HuellaSHA256` del dominio. La ejecución en cidonia, la identidad real del ensayo y el navegador corresponden a Dirección después de las dos revisiones E10 del mismo hash.

## Puertas antes de activar

1. PR25 y PR26 deben estar integradas y publicadas en la base que sirve el portal. Confirmar hashes y binario. No usar este paquete con otra versión del código.
2. Mantener el selector `VEC_DIETAS_BORRADORES_ENABLED` apagado y drenar la aplicación: **cero sesiones** de los ocho LOGIN durante inventario, ensayo y confirmación. Impedir conexiones nuevas externamente hasta terminar el `COMMIT`. `ALTER ROLE` no termina sesiones existentes.
3. Inventariar privadamente el puntero P6 `v2` revocado, sus huellas `v1`/`v2`, rol/control V3, rutas de membresía ascendentes y descendentes, propietarios, ACL directas y concesiones a `PUBLIC`, y sesiones. La posible novena cuenta `vec_dietas_r1d_auditoria_frontera_desarrollo` detiene F4; no se le concede LOGIN por inferencia. Un LOGIN ajeno con ruta a los grupos técnicos, un grupo que herede otro rol, una cuenta/grupo propietario o ACL `PUBLIC` en la base, esquemas, relaciones o funciones VEC detienen la transacción. Los tipos PostgreSQL se excluyen de esa afirmación: `USAGE` en un tipo no concede por sí mismo acceso a datos ni ejecución de funciones, y esta puerta no compara sus ACL para `PUBLIC`. Si la vigencia original ha caducado, se necesita una nueva decisión de autoridad, no extender fechas en este guion.
4. Usar PostgreSQL 18, base `postgres`, DBA por socket local y directorio de evidencia privado externo a Git, modo `0700`. El runner acepta el transporte de contenedor `VEC_F4_POSTGRES_CONTAINER`, `VEC_F4_PG_SOCKET_DIR`, `VEC_F4_PG_PORT` o el servicio local `PGSERVICE`, `PGSERVICEFILE`, `PGPASSFILE` privados. Sigue el protocolo de P6 sin DSN ni contraseña en argumentos o salida. **No ejecutar el preparador R1D** ni reaplicar migraciones instaladas.

## Ensayo y confirmación SQL

Con el transporte privado ya configurado y `VEC_F4_EVIDENCIA_DIR` fuera de Git:

```bash
bash deploy/principal/f4_reactivacion_dietas/ejecutar.sh --inventario
bash deploy/principal/f4_reactivacion_dietas/ejecutar.sh --rollback
VEC_F4_APLICAR=SI-F4-REVISADO bash deploy/principal/f4_reactivacion_dietas/ejecutar.sh --commit
```

Cada llamada inventaría de nuevo. Revisar los JSON privados y cotejar la preimagen antes de confirmar. La transacción exige la presencia exacta de las funciones, tablas, esquemas y concesiones `CONNECT`/`USAGE`/`SELECT`/`EXECUTE` enumeradas para la vertical; cualquier ausencia impide avanzar el puntero o habilitar LOGIN, también en `--rollback`. Exige además exactamente **un puntero global** al rol Dietas bajo bloqueo de tabla antes de habilitar LOGIN. `--rollback` debe conservar el puntero P6 `v2`, ocho NOLOGIN y toda la historia. `--commit` exige `v3` activa, ocho LOGIN, huella canónica nueva y versiones `v1`/`v2` intactas. Una repetición se rechaza sin escritura; tras pérdida de respuesta, consultar inventario y puntero antes de decidir cualquier acción. La credencial privada de cada cuenta se conserva: F4 no la rota ni genera otra.

## Única aceptación funcional

Tras el `COMMIT`, Dirección activa el selector y las conexiones privadas **ya revisadas**, reinicia la aplicación y comprueba que Contratación y Bolsa siguen visibles. En el portal, con identidad sintética autorizada y sin datos reales:

1. Abrir Dietas y crear una comisión. Observar `POST /api/vec/dietas/comisiones` con respuesta, referencia y recibo reales.
2. Consultar `GET /api/vec/dietas/comisiones` y `GET /api/vec/dietas/comisiones/{referencia}`. Ambos muestran la comisión propia, el mismo recibo y versión; otra identidad carece de acceso. No repetir un `POST` para salvar un fallo de consulta.
3. Reintentar exactamente la misma intención si se prueba replay: mismo recibo y una sola fila/historia. Reiniciar aplicación y PostgreSQL, volver a listar y obtener, y comparar referencia, recibo, fecha y versión. Comprobar navegador de escritorio y 390 px, respuestas, errores JS y CT/Bolsa.

Solo ese recorrido acredita la vertical. El ensayo PostgreSQL aislado verifica permisos y transacción, no demuestra la web ni el servidor. Si la identidad, el selector o el runtime impiden la prueba visible, informar **NO-GO** y no declarar Dietas recuperada.

## Retirada si falla la aplicación

Dejar el selector apagado, detener/drenar la aplicación y mantener cerrado el acceso de nuevas sesiones Dietas. Con cero sesiones y el puntero `v3` F4 exacto, ensayar y confirmar la retirada del mismo paquete:

```bash
bash deploy/principal/f4_reactivacion_dietas/ejecutar.sh --retirar-rollback
VEC_F4_APLICAR=SI-F4-REVISADO bash deploy/principal/f4_reactivacion_dietas/ejecutar.sh --retirar-commit
```

La retirada crea `v4` revocada y vuelve las ocho cuentas a NOLOGIN; preserva `v1`–`v3` y no altera CT/Bolsa. Verificar el inventario, rechazo de nuevas conexiones y denegación V3. No usar P6 de nuevo ni ejecutar `DOWN`. La retirada también requiere las dos revisiones E10 de este hash antes de uso real. Un fallo antes del `COMMIT` de activación se resuelve dejando el selector apagado e inventariando; no hay `v3` que retirar.

## Evidencia local

`go test ./deploy/principal/f4_reactivacion_dietas`, `shellcheck` y `bash -n` cubren plan y scripts. `probar_pg18.sh` usa `postgres:18.4` desechable sin red con estructura V3 y datos sintéticos; completa solo en esa base las firmas y ACL de la vertical con funciones de prueba que no sustituyen V3 real. Prueba P6 previo, ROLLBACK, COMMIT, repetición denegada, función o `USAGE` requerido ausente sin `v3` ni LOGIN, ACL/novena cuenta/huella/sesión alteradas, ascenso a `pg_read_all_data`, propiedad y `PUBLIC` inesperados, y segundo puntero con intento de `COMMIT` rechazados antes de LOGIN; conexión rechazada antes y positiva después, retirada `v4` y testigos CT/Bolsa conservados. Ejemplo: `VEC_F4_TEST_BASE=<directorio_privado_0700> bash deploy/principal/f4_reactivacion_dietas/probar_pg18.sh`.
