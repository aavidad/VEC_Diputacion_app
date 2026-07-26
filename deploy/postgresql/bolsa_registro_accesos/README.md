# Registro de accesos de Bolsa T13

Este corte añade un almacén PostgreSQL 18 aislado para `AuditEntry`, consulta
administrativa minimizada y paginación opaca. No añade composición, API ni web.

## Estado y límites

La escritura interna es append-only, usa una cadena SHA-256 construida y
verificada por PostgreSQL, correlación idempotente, UTC, RLS forzada y una
política de retención/bloqueo versionada. No existe función de borrado ni
expurgo automático. La consulta exige intervalo de hasta 31 días, límite de 100
y al menos un ancla exacta de actor seudonimizado, recurso o expediente.
`ListAudit` y el `AppendAudit` común quedan cerrados porque esos contratos no
transportan identidad, finalidad, capacidad VEC ligada al efecto ni garantía de
atomicidad.

El actor operador no se obtiene con SHA-256 desnudo. Antes del PDP debe emitirse
`EvidenciaActorConsultaAccesos` mediante
`vecports.SeudonimizadorSujetoAlmacen`, cuya implementación productiva usa una
clave HMAC exclusiva y rotatable en HSM/KMS. El alias sigue el contrato común
`hmac-sha256:<clave-version>:<64-hex>`. Esa capacidad nominal liga principal,
ámbito T13 y alias; el recurso autorizado liga el alias a la decisión V2.
Además liga los catorce campos del filtro mediante una huella SHA-256
independiente por campo. Las huellas incluyen representaciones exactas de
valores vacíos, fechas, versiones, límite y cursor; PostgreSQL las recompone y
las coteja una a una contra el recurso canónico de la decisión durable.

La función runtime nunca considera autoritativo el JSON recibido. Una frontera
específica, propiedad de `vec_autorizacion_propietario`, coteja la decisión
contra `decision_autorizacion_solicitud_ligada_v2` y revalida en la misma
transacción asignación, rol, políticas, motivo, sesión, garantía y
`ContextoActor`. La fila `consumo_efecto_consulta` no duplica esa autoridad:
solo impide que una decisión/traza ejecute dos lecturas diferentes. Todo dato
se devuelve después del `COMMIT` que confirma decisión, consumo y auditoría.

Solo la consulta administrativa T13 queda protegida de extremo a extremo: en
una transacción revalida la decisión durable, consume la capacidad, registra el
acceso y ejecuta la lectura antes de confirmar. El rol registrador no puede
ejecutar el append genérico ni aportar actor, autorización o finalidad
arbitrarios. El gobernador tampoco puede publicar una política solo por
pertenecer al rol.

Este corte no vuelve atómicos los efectos de otras verticales. Cada vertical
debe añadir un wrapper específico que revalide una decisión VEC y confirme su
lectura o mutación junto al registro en una misma transacción. La publicación de
retención necesita igualmente una autorización VEC específica. Como composición
productiva y esos wrappers se excluyen de B2, el despliegue completo permanece
**NO-GO** hasta que composición inyecte la frontera HSM/KMS y conecte cada
operación personal a su contrato seguro.

## Instalación

Orden obligatorio, con runners separados:

1. núcleo VEC Autorización y sus roles V2;
2. `roles_up.sql`;
3. `migraciones_autorizacion/000001...up.sql` con el migrador de Autorización;
4. `migraciones/000001...up.sql` con el migrador T13;
5. `cerrar_acl_dba.sql` con DBA para retirar `CREATE` de base al propietario.

Las cuentas `LOGIN` y credenciales se aprovisionan fuera del repositorio. Los
cinco roles incluidos son `NOLOGIN`.

## Verificación

```bash
deploy/postgresql/bolsa_registro_accesos/probar_pg18.sh
go test ./internal/modules/bolsa/application/registroaccesos \
  ./internal/modules/bolsa/adapters/postgres/registroaccesos
```

La prueba efímera fija PostgreSQL 18.4, instala roles reales (sin stubs), exige
que el down inverso de Autorización falle mientras T13 siga instalado, prueba
el ciclo down/up completo sin `CASCADE`, ACL/PUBLIC/search path, decisiones
fabricadas, falsificación por registrador, límites, filtros/finalidad,
adulteración, rollback, consumo único, carrera, recuperación, continuidad y
reinicio. También comprueba que los cinco campos de `p_prueba` sean strings
obligatorios y que `verificada_en: null` no alcance ningún cast. Para aislar la
mecánica de consulta, sustituye la frontera real solo después de demostrar que
la falsificación runtime falla; el doble existe exclusivamente dentro del
contenedor desechable. El mismo runner ejecuta con `-race` el recorrido real
`RegistroPostgreSQL` → pgx → PostgreSQL 18 y prueba éxito, replay, denegación,
cancelación en la frontera de `COMMIT`, rollback y que ni la página ni su
auditoría son visibles antes de confirmar. El recorrido pgx también muta uno a
uno todos los campos del filtro después de construir una decisión válida,
inyecta `null` y tipos JSON incorrectos, prueba hexadecimal mayúsculo y
verifica que Go rechaza una fila fuera del filtro autorizado.
