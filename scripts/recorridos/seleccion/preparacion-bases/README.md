# Ensayo HTTP de preparación de bases S2

El driver llama a las dos rutas reales de S2 por HTTPS con certificado mTLS.
Sólo admite un servidor en loopback. No crea PostgreSQL, instala migraciones ni
reinicia servicios. El responsable del clon ejecuta el recorrido cuando AD153,
Bolsa Convocatorias8 y CT168 estén revisadas y ensayadas sobre su preimagen causal.

La configuración, certificados, claves, solicitudes y continuidad permanecen
fuera de Git, en archivos del propietario con permisos `0600`. El directorio de
continuidad tiene permisos `0700`. No se imprimen respuestas ni rutas privadas.

La configuración usa el esquema
`vec.seleccion.preparacion-bases.ensayo-http.v1` y estas claves:

| Clave | Contenido |
| --- | --- |
| `origen` | Origen HTTPS local, sin ruta, credenciales ni parámetros. |
| `ca`, `certificado`, `clave` | Rutas absolutas al material mTLS privado. |
| `guardar` | Solicitud original: `esperada`, `material` y `clave_operacion`. Usar el material real del preparador existente. |
| `cas` | Solicitud con la misma preimagen original y otra clave de operación, para comprobar el conflicto. |
| `continuidad` | Archivo nuevo donde conservar respuesta y huella de la intención original. |
| `evidencia_reinicio_owner` | Acta privada del responsable del clon; se exige en la fase de reinicio. |

Ejecutar `python3 scripts/recorridos/seleccion/preparacion-bases/ensayar.py FASE CONFIGURACION`:

1. `guardar`: exige `201`, conserva la preparación y el recibo, y consulta actual
   y exacta con `200`.
2. `recuperar`: reutiliza la solicitud y clave originales, exige `200` y el mismo
   material, pendientes, recibo y fecha. Consulta actual y exacta.
3. `cas`: exige `409 version_en_conflicto`; ambas consultas deben conservar la
   versión anterior. No modifica la solicitud ni la continuidad originales.
4. `consultar`: comprueba sólo las lecturas actual y exacta.
5. Después del reinicio controlado de aplicación y PostgreSQL, `reinicio` repite
   la recuperación y las dos lecturas. El acta del responsable usa el esquema
   `vec.ensayo.reinicio-owner.v1`, con `postgresql` y `aplicacion`; cada uno contiene
   los instantes UTC `antes` y `despues`. El driver coteja esos instantes y devuelve
   la huella del acta. La observación de los procesos corresponde al responsable.

El resultado HTTP no acredita por sí solo la conservación de historia SQL.
El responsable compara por separado versiones, recibos, historia, outbox y
auditoría antes y después del CAS, replay y reinicio. El driver conserva
`historia_sql: pendiente_owner` para hacer explícita esa comprobación.

El recorrido prepara bases: no las aprueba, firma ni publica. No aporta una
evaluación alternativa; consume los pendientes del evaluador común existente.

El servidor debe disponer además del registrador común de intentos de auditoría.
Su configuración privada `auditoria-intentos.json` contiene `esquema`
(`vec.auditoria.intentos.servidor.v1`), `dsn_file`, `proceso`, `canal`
(`interna_corporativa`) y `limite_segundos` (entre 1 y 30). La cuenta tiene su
provisión DBA y su preflight propios; no se reutiliza una cuenta de negocio.
El archivo privado de S2 declara también `motivo_intento_denegado` y
`motivo_intento_error`, referencias exactas de su catálogo gobernado.

El montaje entrega un único puerto de preparación decorado a sus consumidores.
Captura la identidad registrada antes de la operación y registra los fallos
nominales después de que el servicio retorne. Si falta el acuse de auditoría,
responde `503`. Los accesos permitidos, ausencias y conflictos ya confirmados
conservan la auditoría de la transacción de Bolsa. El rechazo anterior a resolver
la sesión sigue en la auditoría común de frontera, sin inventar un actor.
