# Auditoría común del contexto ADMIN antes de V2 — AD192

Familia `contexto_admin_pre_v2`, módulo `administracion`, canal
`administracion_privilegiada`, finalidad `establecer_contexto_admin`.
No incorpora decisión, capacidad, sesión o contexto V2 ficticios. El LOGIN
es `session_user` real; proceso y ventana proceden de configuración privada.

Contrato de quince claves JSON, todas presentes:
`tipo_registro`, `evento_ref`, `operador_login`, `actor_ref`,
`perfil_activo_ref`, `accion`, `recurso_ref`, `resultado`, `motivo_ref`,
`proceso`, `canal`, `finalidad_ref`, `correlacion_ref`, `fuente_ref`,
`fuente_sha256`. Los campos acreditados de Actor/perfil/fuente son referencias
opacas del propietario IS/CA; los desconocidos son null, nunca valores HTTP.
Éxito exige Persona, perfil y fuente/SHA. Una negativa conserva sólo lo ya
acreditado, con perfil dependiente de Persona y fuente/SHA emparejados.

Entradas propuestas para los productores propietarios:

- `registrar_contexto_admin_pre_v2_is_v1(jsonb)`: resolver_cuenta_admin y
  vincular_sesion_admin. EXECUTE únicamente del propietario IS.
- `registrar_contexto_admin_pre_v2_ca_v1(jsonb)`: registrar_contexto_admin y
  reconciliar_contexto_admin. EXECUTE únicamente del propietario CA.

Devuelven los cinco campos del acuse común: `auditoria_ref`, `secuencia`,
`huella_sha256`, `correlacion_ref`, `registrada_en`. El LOGIN de ADMIN sólo
puede invocar las fachadas de IS/CA; no recibe append directo ni acceso a
sus tablas. El productor toma los datos desde su autoridad y liga el hash
al documento fuente real, sin atribuir autoridad a un DTO.

Resultado permitido y efecto comparten transacción. Ante denegación/error,
el productor revierte su subbloque y registra la negativa fuera; si falla
el append, no entrega datos ni un acuse provisional. Cada consulta funcional
usa un evento nuevo.

Reconciliar un COMMIT incierto es una continuación interna: conserva evento,
acción, material y acuse originales de registrar. No renombra la acción a
reconciliar ni fabrica otra consulta. `reconciliar_contexto_admin` se usa sólo
para una consulta funcional nueva con evento propio. Misma referencia con
material distinto se rechaza. El hash encuadra los quince campos en su orden
mediante el helper común; los null acreditativos usan el frame vacío, que
no puede confundirse con una cadena vacía válida.

Base causal medida por Source: CHECK v4 SHA256
`dacbd820f1679fc2f6a02bb3d001a0a69df8528c5693f8e9ea33c17cb7fe4fa5`;
núcleo source SHA256
`73cb05e1c57c82c13e066a1f3c27cf03d9bdbaed3e028184ee9d040b9b230735`.
Esta base contiene AD183/184 y no AD185–190. La migración no asumirá sus
columnas ni inventará la preimagen de una instalación futura.

Estado: contrato de producción preparado. Pendientes catálogo técnico de
motivos cerrado, SQL/verificador/vectores, dos revisiones y ensayo autorizado.
No se ha ejecutado Go ni PostgreSQL ni instalado AD192.
