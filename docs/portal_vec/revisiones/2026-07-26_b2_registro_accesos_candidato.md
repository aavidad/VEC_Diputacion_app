# Candidato B2 — registro de accesos T13

Fecha de congelación: 26 de julio de 2026.

## Identidad del candidato

- Trabajo: B2 / T13, registro durable de accesos a datos personales con
  finalidad.
- Rama: `agent/bolsa-b2-registro-accesos-20260726`.
- Worktree:
  `.worktrees/bolsa-b2-registro-accesos`.
- Base congelada del código: `5f4e0455ec16b4d2bfeeea8a0bd77b67eb50ac68`.
- La rama de integración avanzó después solo en documentación hasta
  `e3ed8eac351702155a3443db65305103ed5a0b61`; por instrucción de dirección no
  se hizo rebase.
- Sin commit: pendiente de revisión independiente e integración por dirección.
- Manifiesto: `2026-07-26_b2_registro_accesos_candidato.sha256`.

## Estado de cierre

**GO técnico acotado** para la consulta administrativa T13 aislada:

1. el recurso autorizado liga principal, actor seudonimizado HMAC, finalidad
   y los catorce campos del filtro mediante huellas SHA-256 independientes;
2. PostgreSQL revalida una decisión VEC V2 durable y el estado mutable;
3. en una transacción `SERIALIZABLE READ WRITE` consume la decisión, registra
   el acceso y ejecuta la lectura;
4. la respuesta solo sale del adaptador después del `COMMIT`;
5. el asiento de la propia consulta queda registrado pero fuera del resultado;
6. el cursor fija la frontera de páginas posteriores y excluye eventos nuevos;
7. límite máximo 100, intervalo máximo 31 días, filtros exactos y ancla
   selectiva obligatoria impiden el volcado masivo;
8. el recorrido real Go → pgx → PostgreSQL 18 demuestra éxito, replay,
   denegación, rollback y cancelación justo en la frontera de `COMMIT`; antes
   de confirmar no retorna datos ni hace visible el asiento.

**NO-GO global y productivo**:

- `AppendAudit` y `ListAudit` comunes fallan cerrados;
- el rol registrador no puede ejecutar el append genérico ni inventar actor,
  autorización, finalidad o efecto;
- el rol gobernador no puede publicar retención por mera pertenencia;
- cada vertical debe añadir un wrapper específico que revalide su decisión VEC
  y confirme operación y registro en la misma transacción;
- publicar una nueva política de retención requiere autorización VEC
  específica;
- falta composición productiva con seudonimización HMAC respaldada por HSM/KMS
  y conexión de todas las lecturas personales;
- quedan fuera de este candidato API, web, composición y datos reales.

## Matriz PostgreSQL 18

Comando:

```bash
deploy/postgresql/bolsa_registro_accesos/probar_pg18.sh
```

Imagen fijada:

```text
postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296
```

Resultado: código de salida 0 y línea final exacta:

```text
PG18 T13: instalación, down limpio, ACL, falsificación, cadena, Go-pgx, vínculo-filtro, tipos, auditoría exacta, respuesta cerrada, commit, rollback, cancelación, consumo, carrera, recuperación y reinicio OK
```

La ejecución parte de un contenedor limpio e instala las migraciones reales de
Autorización antes de T13. Verifica:

- instalación y retirada limpia sin `CASCADE`;
- rechazo del down inverso de Autorización mientras la consulta T13 sigue
  instalada, seguido del ciclo completo en el orden correcto;
- cierre DBA de `CREATE`, `PUBLIC`, tablas, funciones y `search_path`;
- roles `NOLOGIN`, sin superusuario ni `BYPASSRLS`;
- falsificación de append por registrador: SQLSTATE esperado `42501`;
- decisión durable fabricada por runtime: SQLSTATE esperado `42501`;
- aislamiento inferior a serializable: SQLSTATE esperado `22023`;
- replay de una decisión consumida: SQLSTATE esperado `42501`;
- cadena, orden, continuidad, idempotencia, adulteración y rollback;
- consulta minimizada, filtros, finalidad, límite, cursor, exclusión del asiento
  propio y estabilidad entre páginas;
- mutación uno a uno de versión, actor, módulo, acción, finalidad, recurso,
  expediente, resultado, fechas, versión de objeto, límite, cursor y finalidad
  de consulta después de crear una decisión y un recurso válidos;
- rechazo de `null` y tipo booleano en cada campo del filtro antes de cualquier
  cast, y exigencia de string no-NULL en los cinco campos de `p_prueba`;
- cotejo exacto de acción, módulo, propósito, versión, resultado, sujeto,
  correlación, actor, autenticación y campos vacíos de la auditoría T13;
- adaptador real `RegistroPostgreSQL.ConsultarAccesosAdministrativos`, pgx y
  función PG18, ejecutado con detector de carreras;
- rechazo Go de filas fuera del filtro o intervalo autorizado y de
  hexadecimal mayúsculo, también en el recorrido Go → pgx;
- ausencia de respuesta y de visibilidad de auditoría mientras `COMMIT` está
  bloqueado;
- éxito después de liberar `COMMIT`, replay revertido, denegación sin efecto y
  cancelación en la frontera de `COMMIT` con `context.Canceled`;
- dos escritores concurrentes, reintento del abortado y continuidad final;
- persistencia después de reiniciar PostgreSQL;
- rechazo del `down` destructivo cuando existe historia durable.

El contenedor manual residual de desarrollo se eliminó después de la prueba.

## Corrección posterior al primer NO-GO

La revisión independiente inicial detectó que `subject_ref` no bastaba para
ligar el filtro al recurso durable, que faltaba el cotejo exacto de la traza,
que `null` podía alcanzar casts y que Go no verificaba el filtro de cada fila.
El candidato actual corrige esos cuatro puntos. Los valores opcionales vacíos
no se copian como atributos vacíos: cada uno se representa mediante la huella
SHA-256 de su valor exacto. La frontera VEC exige el conjunto completo y
PostgreSQL recompone y compara cada huella con `IS DISTINCT FROM`.

## Corrección posterior al segundo NO-GO

La segunda revisión independiente detectó dos defectos adicionales. Primero,
`verificada_en: null` producía SQL `NULL` en las expresiones booleanas y podía
alcanzar el cast y omitir la comprobación temporal. La frontera exige ahora que
los cinco valores de `p_prueba` sean strings JSON no-NULL antes de convertir
ninguno, comprueba además que el instante convertido no sea NULL y usa
`IS DISTINCT FROM` para su forma canónica. Segundo, el down de la frontera de
Autorización podía ejecutarse antes que el down principal porque la llamada
PL/pgSQL no crea una dependencia fuerte de catálogo. El down comprueba ahora
explícitamente que `consultar_accesos_administrativos_v1` ya no exista y falla
con `2BP01` sin revocar privilegios ni eliminar la función. El runner PG18
incluye ambas regresiones y el ciclo correcto de retirada/reinstalación.

## Pruebas ejecutadas

Todos estos comandos terminaron con código 0:

```bash
bash -n deploy/postgresql/bolsa_registro_accesos/probar_pg18.sh
go test ./internal/modules/bolsa/application/registroaccesos \
  ./internal/modules/bolsa/adapters/postgres/registroaccesos -count=1
go test -race ./internal/modules/bolsa/application/registroaccesos \
  ./internal/modules/bolsa/adapters/postgres/registroaccesos -count=1
go test ./... -count=1
go vet ./...
go mod verify
scripts/comprobar_tamano_ficheros.sh
git diff --check
sha256sum -c \
  docs/portal_vec/revisiones/2026-07-26_b2_registro_accesos_candidato.sha256
```

No se ejecutaron `go test -race ./...`, `govulncheck` ni la puerta completa
`scripts/verificar_calidad.sh`: la carrera se ejecutó sobre los dos paquetes
modificados; la suite normal y `go vet` sí cubrieron todo el módulo.

## Seguridad y datos

- No hay nombres, DNI, correos, expedientes reales, claves ni tokens.
- Fixtures PostgreSQL exclusivamente sintéticos y referencias opacas.
- Actor `hmac-sha256:<version-clave>:<64-hex>`; nunca SHA desnudo.
- La capacidad nominal de actor solo nace del seudonimizador común HSM/KMS.
- La autoridad procede de la decisión VEC durable, nunca de `AuditEntry` ni del
  JSON runtime.
- Sin `pg_temp` en `search_path`, acceso directo a tablas ni exportación masiva.
- Contrato del caso de uso en `application/registroaccesos`; no se creó ningún
  fichero en `*/ports`, conforme a DEC-051.

## Cambio propuesto para el tablero

No marcar T13 global como cerrado. Mantener
`Auditoría, recibos y registro de accesos` como parcial y añadir que la consulta
administrativa T13 aislada tiene candidato PostgreSQL E2E revisable; siguen
pendientes los wrappers atómicos de cada vertical, la autorización de retención,
la composición HSM/KMS, T12 y la revisión institucional.

Revisión independiente completada: sí; los dos bloqueos de la segunda revisión
quedaron corregidos y revalidados. Integración y commit: pendientes de dirección.
