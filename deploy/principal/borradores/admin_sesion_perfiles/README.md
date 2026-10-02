# Sesión ADMIN para gestión de perfiles: borrador IS12/CA23

Estado: preparado en código, pendiente de ensayo causal en clon y de dos revisiones independientes. No está instalado. No ejecutar UP ni DOWN en principal, cidonia ni bases con historia conservada.

CA23 consume la asignación nominal que AUT24 acredita para `rol:administracion_perfiles:v2`, su huella, control vigente y bootstrap consumido. AUT24 devuelve la audiencia administrativa de su catálogo configurado y auditado; no se siembra ninguna audiencia en estos archivos. CA23 comprueba las versiones y vigencias en CA20 y reutiliza las funciones propietarias extraídas por CA22. No crea perfiles, asignaciones ni una autoridad de provisión.

IS12 conserva una vinculación append-only de sesión para este propósito. La sesión común, cuenta privilegiada y ordinaria, persona, perfil, certificado, política y audiencia deben coincidir. Conserva la observación de revocación y los límites atestados por la frontera del certificado y CRL. Tanto efecto como replay revalidan la sesión y el estado actual dentro de SERIALIZABLE de escritura; la caducidad aplicable es el menor límite conservado. La producción sigue cerrada sin las evidencias corporativas requeridas por IS9.

El LOGIN de esta conexión hereda exclusivamente `vec_identidad_sesiones_v1_admin_perfiles`, con INHERIT TRUE, SET FALSE y ADMIN FALSE, sin ACL directas. Solo recibe las cinco funciones públicas nominales de resolución, vinculación y contexto. El revalidador de efecto tiene EXECUTE únicamente para el propietario AD3, además del propietario de Identidad. Las tablas usan RLS forzado y rechazan mutaciones y truncado.

## Interfaces

- `vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz)`: mismas 14 columnas del resolver IS11, con `rol_id=administracion_perfiles`.
- `vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text)`: entorno, host, audiencia, SHA256 del certificado, SHA256 CA, autenticación, observación de revocación, límite CRL, límite certificado, autenticación_ref y sesión_ref. Las fechas de certificado/CRL las obtiene la frontera F; no proceden del cliente.
- `vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(text,text,text,text,text,text,text,text)`: autenticación_ref, sesión_ref, cuenta privilegiada, cuenta ordinaria, persona, perfil, política y huella de política; devuelve booleano.
- `vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1()`: identidad_login y acreditada.
- `resolver_y_registrar_contexto_admin_perfiles_v1` y `reconciliar_contexto_admin_perfiles_v1`: operación, registro, cuenta y fecha solicitada; mismas ocho columnas CA22.

Dependencia exacta requerida de AUT24: `vec_autorizacion.consultar_asignacion_admin_perfiles_v1(text)` devuelve persona_ref, perfil_ref, vinculo_ref, cuenta_version, persona_version, perfil_version, vinculo_version, audiencia y vigente_hasta. Requiere USAGE del esquema y EXECUTE para el propietario CA. Si falta, CA23 rechaza la preimagen; si no hay asignación válida, la consulta no habilita sesión.

## Validación pendiente

Orden: fuentes F CA20/IS9/AUT23; CA22 e IS11 reales de K; AUT24 definiciones; CA23; IS12; AD150 sobre AD149 real. `lista_sql_borrador.txt` contiene solo nuestros dos archivos y no es una lista instalable. Dirección debe componer el orden completo con los hashes revisados antes de ensayar.

`pruebas_sql/contrato_y_denegaciones.sql` prepara en ROLLBACK un LOGIN sintético y prueba ACL, segregación de propósito, CRL vencida/infinita, sesión sin vinculación y rechazo de membresía múltiple. `pruebas_sql/sesion_nominal_y_replay.sql` requiere un fixture sintético de bootstrap AUT24, certificado IS9 y sesión común; comprueba vinculación, replay, persona ajena y revocación del certificado. Su llamador debe abrir SERIALIZABLE y revertir. Ese fixture no está incluido: no se sustituye la autoridad AUT24 por un doble.

Comprobaciones realizadas: inspección estática de nueve funciones con revocación PUBLIC inmediata, límites temporales conservados, ausencia de provisión en CA23 y `git diff --check`. No acreditan sintaxis PL/pgSQL, ACL efectiva, concurrencia ni persistencia. Semgrep con dos reglas genéricas locales, métricas desactivadas y red aislada falló en el motor al crear un hilo (`Resource temporarily unavailable`); no se obtuvo dictamen. No se ejecutó PostgreSQL ni se creó PR o publicación.
