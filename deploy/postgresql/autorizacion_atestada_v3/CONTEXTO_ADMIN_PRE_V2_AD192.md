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

Cotejo de recuperación propio: `cotejar_contexto_admin_pre_v2_is_v1(jsonb)`
y `cotejar_contexto_admin_pre_v2_ca_v1(jsonb)`, únicamente sus propietarios.
Admiten READ COMMITTED/READ ONLY y no escriben, crean eventos ni renuevan
configuración. Devuelven ACK5 original si familia, LOGIN y frame15 coinciden;
ausencia devuelve cero filas y material distinto con el mismo evento da23505.
No conceden acceso al LOGIN ADMIN. El enlace CA es de dominio, no otro registro
de auditoría; el cotejo lo consume después de verificar registro/enlace/vínculo.

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

Catálogo técnico cerrado mínimo: las cuatro acciones combinadas con tres
resultados. Motivos `contexto_admin_pre_v2_permitido`,
`contexto_admin_pre_v2_denegado` y `contexto_admin_pre_v2_error`, en datos SQL
y JSON del verificador. Una Persona conservada exige fuente/SHA acreditados;
no se publica un motivo humano V2 inventado.

Base causal medida por Source después de instalar estructuras
AD185→186→187→188→189→191, sin publicar claves ni efectos: CHECK v4 SHA256
`6b92079faedcd2482b720b1d0714ba6a1b09dc359badae8f6d7c0dd3f3275caf`;
núcleo source SHA256
`b7eb48be035e854928c9685c916f9139166a197e3cae731510b3597f0fe40a45`.
Se conserva el CHECK anterior íntegro y no se modifica el núcleo. No se usa
la captura anterior post184 ni se asume la futura familia AD190.

Verificador y CLI usan el esquema propio
`vec.auditoria.verificacion.contexto-admin-pre-v2.v1`. El parser de esta entrada
permite null sólo en los cuatro slots acreditativos; los parsers anteriores
permanecen cerrados. Los vectores encuadrados con Python prueban éxito,
negativa con Persona conocida, error sin identidad, cruces de familias,
colisión/cambio de acción, tipos JSON y preservación de familias históricas.

Validación focal: normal/race/vet de auditoría y CLI verdes; gofmt/diff,
Semgrep local y gosec sin hallazgos; vecsilencio sin errores nuevos. No se
repitieron puertas globales ni se invocó la base de datos. La configuración
inmutable de proceso/canal se verifica antes y después de la espera de cadena.

Pendientes: dos revisiones exactas y ensayo autorizado con productores IS/CA
reales. La configuración de LOGIN/proceso no se siembra durante UP. AD192 se
instaló después en la principal de desarrollo (H10, 4 de octubre de 2026).

AD194 (`000194_secuencia_repeticion_contexto_admin.up.sql`) corrige la repetición de registrar y el cotejo, que fallaban con «structure of query does not match function result type» porque devolvían `secuencia` como numeric(20,0) en lugar de bigint; sólo añade `::bigint` a ese valor, conserva firma, OID, propietario, ACL y proconfig, comprueba la preimagen de AD192 y se prueba con `pruebas_sql/ad194_repeticion_secuencia.sql`.

Si AD194 ya está aplicada, una segunda ejecución se detiene con código 3 y
`AD194: PARO clave=definicion … actual=468bd4…` (o `actual=4df083…` en la
función de cotejo). Ese mensaje significa «ya aplicada»: no cambia nada y no
hay que repetirla.
