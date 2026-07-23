# Revisión de seguridad y arquitectura hexagonal de O2-06A

Fecha: 23 de julio de 2026.

Revisor: especialista independiente de seguridad/hexagonal, en solo lectura.

## Dictamen

**GO condicionado** para el diseño tras la segunda revisión. **NO-GO** para
implementar o componer O2-06 hasta que exista SHA estable y GO de O2-05 y se
implementen las fronteras cerradas en el documento principal.

No se leyó ningún worktree ni cambio sin commit del productor O2-05. No se
modificó ningún archivo durante la revisión.

## Hallazgos y resolución

### 1. Material VEC-AD-3 incompleto — alto

La orden actual no contiene contexto V2, atestación, prueba, raíz ni capacidad.
La emisión existente exige esos materiales y una prueba ya validada.

Resolución: añadir el resultado de contexto a la orden y usar un proveedor
nominal VEC que derive motivo/operación/efecto y devuelva un paquete opaco para
SQL. La prueba concreta permanece dentro del broker. No se admite DTO,
serialización genérica ni entrada aportada por un canal.

### 2. Segregación emisor/ejecutor — alto

Resolución: VEC común posee el cliente, codec y broker TLS 1.3 sobre socket
Unix; el broker posee HMAC no exportable y cero credenciales SQL. Framing,
campos, hashes, códigos, ALPN, CA/EKU/SAN, `SO_PEERCRED`, ACL, límites y
deadlines son cerrados. La capacidad cruza como bytes y el cliente la envuelve
en un exportador privado redactado; no viaja una interfaz Go. El adaptador
posee el rol SQL y una credencial breve de llamada, nunca material emisor.

Evidencia:

- separación de servicio de confianza, emisor y verificador:
  `docs/portal_vec/confianza_y_capacidad_vec_ad_3_o2_05_2026-07-23.md`;
- bloqueo de codecs y redacción:
  `internal/vec/adapters/seguridad/confianzaatestacion/capacidad_v3_proteccion.go`;
- copia defensiva de capacidad:
  `capacidad_v3_sobre.go`.

### 3. Preparación separada — alto

`ServicioRegistroSolicitud` invoca hoy `PrepararAlta` antes de
`TransaccionAltas`, mientras O2-05 exige que runtime pierda permiso de reservar
por separado.

Resolución: generar solo candidatos no durables, retirar
`PreparadorAltaPostgreSQL` de O2-07 y resolver alias, reserva, expediente,
consumo, auditoría y outbox en el único `COMMIT`.

### 4. Recibo insuficiente — alto

`ReciboAlta` carece de las tres huellas previstas por O2-05.

Resolución: once campos mínimos —incluida procedencia canónica— y doce columnas
SQL contando `resultado`; canon V1, tres SHA-256, copia por valor, validación
antes/después del `COMMIT`, replay exacto y rechazo de adulteración. No
devuelve identidad, decisión, correlación, HMAC ni material VEC.

### 5. Reconciliación y reinicio — alto

Resolución: ningún error posterior al intento de `COMMIT` vuelve a confirmar.
La reconciliación usa conexión nueva, contexto interno acotado,
`READ COMMITTED READ ONLY`, misma barrera y entrada exacta; espera y consulta
son dos sentencias distintas y no hay consumo, DML, auditoría ni outbox. Una
fila válida exige el consumo de la capacidad exacta del intento. Ausencia
prueba rollback del intento; cruce/divergencia produce resultado indeterminado.
`confirmada` coteja candidatos; `replay` devuelve el recibo histórico sin
compararlo con referencias CSPRNG ni instante nuevos. El reinicio no depende
de memoria, WAL, txid, nonce efímero ni reloj cliente.

### 6. Frontera hexagonal y neutralidad — medio

Aplicación solo conoce dominio y puertos. pgx, mTLS y socket Unix viven en
adaptadores. La función exterior de contratación invoca una función interna
propiedad de VEC; nunca tablas VEC. Web, escritorio, CLI y MCP usan el mismo
`Registrar`. Cuatro contratos prueban igualdad con identidad/contexto comunes;
ningún canal accede a broker/SQL ni hereda la identidad mTLS de carga.

### 7. Errores y minimización — medio

Resolución: catálogo estable con futuras claves i18n; capacidad inválida,
revocada o expirada se presenta como denegación. SQLSTATE, texto pgx,
identidades, huellas de identidad/petición, HMAC, capacidad, DSN, secretos y
contenido personal directo se eliminan de errores, logs y recibos. Referencias,
número y huellas del recibo son seudónimos vinculables protegidos, no datos
anónimos. Los buffers se limitan antes de reservar, se copian y se
sobrescriben como mejor esfuerzo.

### 8. Desarrollo, procedencia y puertas normativas — alto

Resolución: desarrollo reutiliza la doble guarda T21 y un broker externo, con
Ed25519 VEC-AD-3 y HMAC de capacidad separadas de las demás claves. Directorio
y volumen son exclusivos y no importables a producción. SQL deriva y propaga
el canon común `vec.acto.procedencia.v1` con perfil, autoridad, proveedor y
no-migrabilidad; no presume competencia jurídica.

El diseño traza los siete puntos obligatorios: normas, datos/fuentes/
destinatarios, finalidad, controles, evidencia, conservación/acceso y
responsables. CT-CUM-02/03/04/05/06/07/10 mantienen sus bloqueos; 08/09 no
aplican a este diseño sin UI ni IA.

## Pruebas reproducidas

El revisor reprodujo correctamente las pruebas focales y de carrera de:

```text
internal/vec/adapters/seguridad/confianzaatestacion
internal/modules/contrataciontemporal/ports
internal/modules/contrataciontemporal/application
internal/modules/contrataciontemporal/adapters/postgres
```

Estas pruebas acreditan el corte integrado actual, no el futuro consumidor
O2-05 ni el adaptador O2-06.

La futura matriz cubre wire/ACL, vectores de evidencia, procedencia, efecto
cruzado, replay con candidatos nuevos, capacidad exacta, recibo adulterado,
`40001`/`40P01`, cancelación antes/después de `COMMIT`, reconciliación 0/1/>1,
reinicio, rotación/revocación durante lock, cadenas, rollback total, los
cuatro canales y barrido de fugas.

## Riesgos residuales

- Falta el SHA estable del consumidor SQL O2-05.
- Falta implementar el paquete opaco, broker segregado y ampliación T21.
- HSM/KMS, ancla anti-restauración, custodia y aprobaciones formales continúan
  bloqueando producción.
- Las puertas CT-CUM declaradas bloquean datos y actos reales.

**Dictamen final: GO condicionado para O2-06A como diseño; NO-GO para
implementación o composición anticipadas.**
