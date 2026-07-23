# Revisión de seguridad y arquitectura hexagonal de O2-06A

Fecha: 23 de julio de 2026.

Revisor: especialista independiente de seguridad/hexagonal, en solo lectura.

## Dictamen

**GO condicionado** para el diseño. **NO-GO** para implementar o componer
O2-06 hasta que exista SHA estable y GO de O2-05 y se incorporen las fronteras
cerradas en el documento principal.

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

Resolución: VEC común posee el broker mTLS sobre socket Unix; el broker posee
HMAC no exportable y cero credenciales SQL. El adaptador posee el rol SQL y
una credencial breve de llamada, nunca material emisor. Sin broker/HSM/KMS la
composición productiva falla cerrada; no hay fallback local.

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

Resolución: diez campos públicos, canon V1, tres SHA-256, copia por valor,
validación antes/después del `COMMIT`, replay exacto y rechazo de adulteración.
No devuelve identidad, decisión, correlación, HMAC ni material VEC.

### 5. Reconciliación y reinicio — alto

Resolución: ningún error posterior al intento de `COMMIT` vuelve a confirmar.
La reconciliación usa conexión nueva, contexto interno acotado,
`READ COMMITTED READ ONLY`, misma barrera y entrada exacta; espera y consulta
son dos sentencias distintas y no hay consumo, DML, auditoría ni outbox. Una
fila válida devuelve el recibo; ausencia probada no autoriza un reintento
interno; divergencia, multiplicidad o timeout produce resultado indeterminado
saneado. El replay tras reinicio rederiva alias y valida internamente
decisión/correlación originales; no depende de memoria, WAL, txid, nonce
efímero ni reloj cliente.

### 6. Frontera hexagonal y neutralidad — medio

Aplicación solo conoce dominio y puertos. pgx, mTLS y socket Unix viven en
adaptadores. La función exterior de contratación invoca una función interna
propiedad de VEC; nunca tablas VEC. Web, escritorio, CLI y MCP usan el mismo
`Registrar`, sin cookies, almacenamiento de navegador ni cabeceras libres de
identidad.

### 7. Errores y minimización — medio

Resolución: catálogo estable con futuras claves i18n; capacidad inválida,
revocada o expirada se presenta como denegación. SQLSTATE, texto pgx,
identidades, huellas de identidad/petición, HMAC, capacidad, DSN, secretos y
datos personales se eliminan de errores, logs y recibos. El recibo conserva
únicamente sus tres huellas públicas de integridad. Los buffers se limitan
antes de reservar, se copian y se borran al terminar.

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

La futura matriz debe cubrir broker/ACL, autoridad y efecto cruzados, replay,
capacidad repetida, recibo adulterado, `40001`/`40P01`, cancelación antes y
después de `COMMIT`, respuesta perdida, reconciliación 0/1/>1, reinicio,
rotación/revocación/expiración durante lock, rollback total, neutralidad de
canal y barrido de fugas.

## Riesgos residuales

- Falta el SHA estable del consumidor SQL O2-05.
- Falta implementar el paquete opaco y el broker segregado.
- HSM/KMS, ancla anti-restauración, custodia y aprobaciones formales continúan
  bloqueando producción.

**Dictamen final: GO condicionado para O2-06A como diseño; NO-GO para
implementación o composición anticipadas.**
