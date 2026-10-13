# Comprobar el arranque de «Solicitudes y admisión»

El registro de arranque identifica ahora la dependencia que falla como
`componente=inscripcion_rrhh_<fase>`. Se conserva la causa interna y se muestra
el mensaje catalogado de inscripción. No se registran cuentas, certificados,
rutas privadas, DSN ni contenido del SQL.

| Fase | Qué comprobar |
| --- | --- |
| `selector` | `VEC_BOLSA_INSCRIPCIONES_ENABLED=true`, perfil de desarrollo y DSN lector RRHH definido |
| `resolvedor` | Identidad de desarrollo compuesta y lectores RRHH registrados |
| `idempotencia` | Derivador privado válido del proceso |
| `material_revisar` / `material_incorporar` | Proveedores V3 preparados para `vec_bolsa_llamamientos.inscripcion.revisar.v1` y `.incorporar.v1` |
| `fichero_privado` | `identidad/inscripcion-bolsa-rrhh.json` propio, privado y con JSON estricto: versión 1, autoridad `no_autoritativo`, entre 1 y 64 cuentas y motivos válidos distintos del mismo catálogo |
| `cuentas_lectores` | Cada certificado, sujeto y perfil coincide con un lector RRHH registrado; certificado canónico y único, cuenta no vacía |
| `dsn_lector` | `VEC_BOLSA_INSCRIPCIONES_RRHH_LECTOR_DATABASE_URL` válido y separado de los LOGIN de ejecución |
| `pool_<rol>` | Una de las seis conexiones del fichero privado: TLS verificado, rol nominal único y sin privilegios adicionales |
| `login_repetido` | Las seis conexiones usan LOGIN distintos |
| `ejecutor_bolsa` | LOGIN `vec_bolsa_llamamientos_desarrollo`, con B97 revisar e incorporar ejecutables |
| `lector_bolsa` | LOGIN `vec_bolsa_inscripciones_rrhh_lector`, grupo único `vec_bolsa_llamamientos_lector_inscripciones_rrhh` y exactamente las cuatro funciones de lectura B96 permitidas |
| `registro_sesiones` / `revalidador` | Registro y revalidación internos de identidad disponibles |
| `resolutor_contexto` / `servicio_contexto` / `autoridad_contexto` | Contexto nominal interno disponible |
| `fuente_autorizacion` / `registro_autorizacion` / `motivos` / `pdp` | Fuentes internas de autorización y catálogo de motivos disponibles |
| `nonce` / `sesion_rrhh` / `ambito_rrhh` | Sesión nominal y fuente del ámbito de gestión RRHH disponibles |
| `decisor_lecturas` / `autoridad_inscripcion` / `repositorio` / `servicio_inscripcion` / `preparador_rrhh` / `handler_rrhh` | Contratos del montaje completo admitidos |

Copiar las cuentas de `usuarios-preferencias-interna.json` no las convierte en
lectores RRHH. El resolvedor carga éstos desde `identidad/consultas-rrhh.json`:
las cuentas elegidas para inscripción deben coincidir exactamente con sus
certificados, sujetos y perfiles. Si el registro indica `cuentas_lectores`,
cotejar ambos ficheros privados y elegir únicamente las cuentas RRHH ya
registradas, sin cambiar un perfil ni conceder permisos por nombre.

Las seis entradas de conexión del fichero de inscripción son:

| Clave | Grupo nominal |
| --- | --- |
| `dsn_registro_identidad` | `vec_identidad_sesiones_v1_registrador` |
| `dsn_revalidacion_identidad` | `vec_identidad_sesiones_v1_revalidador` |
| `dsn_contexto` | `vec_contexto_actor_v1_runtime` |
| `dsn_fuente_autorizacion` | `vec_autorizacion_fuente` |
| `dsn_registro_autorizacion` | `vec_autorizacion_registro` |
| `dsn_motivos` | `vec_autorizacion_motivos_evaluador` |

El arranque de CT ya solicita los materiales V3 de revisar e incorporar cuando
está activo el selector. Si la composición alcanzó `inscripcion_bolsa_rrhh`,
no se deduce que el material falte: las primeras comprobaciones nuevas son el
fichero privado y su cruce con lectores. La nueva etiqueta permite distinguir
esas causas sin probar ajustes a ciegas.

SQL NO. CONFIG NUEVA: ninguna; se conservan las opciones y permisos actuales.
Dirección despliega el diagnóstico y recoge la fase que falle. Después corrige
la dependencia concreta y comprueba la bandeja en Chrome; esta PR identifica
el bloqueo, todavía no acredita su solución en cidonia.
