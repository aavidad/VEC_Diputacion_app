# Relación de Personal consultada desde RPT

`ComponerLectorRelacionSeleccionadaRPT` conecta el selector existente, el lector
nominal de Personal y el registrador común de intentos. La acción sigue siendo
`personal.relacion_rpt.consultar`, con finalidad
`conciliar_relacion_laboral_para_rpt` y audiencia `vec_personal.relacion_rpt.v1`.
La fuente mantiene certeza y cobertura no acreditadas; la lectura no decide
ocupación, reserva ni vacante RPT.

El montaje recibe identidad registrada, emisor V3, motivo de autorización,
pool de lectura, reloj, plazo de resolución de identidad, registrador nominal
y `EmisorResultadosTecnicosConContexto` común.
`ConfiguracionResultadosTecnicosLectorRPT` exige componente y etapa del catálogo
cerrado de L; no hay emisor opcional ni sustituto nulo.
`ConfiguracionIntentosLectorRPT` exige proceso, canal, recurso técnico para entrada
inválida y tres motivos de catálogo: denegado, entrada inválida y no disponible.
Son configuración confiable; ningún valor se toma del cuerpo de la petición.
El registrador implementa `RegistradorIntentosAuditoria` y
`PreflightIntentoAuditoria`, como el adaptador PostgreSQL de VEC. Su pool y LOGIN
segregados, y su configuración DBA de proceso y canal, pertenecen a ese adaptador.
La ausencia de estas dependencias mantiene el lector cerrado.

`NuevoLectorRelacionSeleccionadaRPTConIdentidad` permite aplicar la misma frontera
al selector compuesto en otros clientes internos. Antes de validar la selección,
captura una copia de la identidad registrada y su vínculo original. El proveedor
V3 coteja el actor solicitado con esa copia. Al registrar un fallo se usa esa
identidad histórica, también después de cancelar o caducar la sesión; no se
resuelve otro perfil ni se renueva un permiso. Sin identidad histórica acreditada
el puerto común no admite el intento nominal: el lector queda no disponible.

La frontera de transporte debe crear la correlación común mediante
`ConCorrelacionIncidenciasPeticion`. El lector llama a
`ReferenciaCorrelacionAutorizacionV2DePeticion` del puerto común: los 32 dígitos
hexadecimales técnicos se representan en V3 con el prefijo `correlacion_`. No
genera otra correlación ni acepta texto del cliente. Su ausencia impide iniciar
la lectura.

La lectura permitida conserva consumo V3, recibo y auditoría en la transacción de
Personal. Denegaciones y errores se registran al retornar el repositorio, después
de su rollback o cierre, mediante `AppendIntentoAuditoria`. Una relación válida se
identifica en ese registro de intento como
`personal:relacion_rpt:sha256:<SHA256 de la referencia exacta>`; el hash distingue
mayúsculas y minúsculas. No es el recurso de autorización, no altera la selección
y no contiene datos laborales. La entrada inválida usa el recurso técnico
configurado; no se conserva ni se cifra mediante hash texto arbitrario.

Cada llamada crea una referencia opaca de intento en el servidor. La orden y el
acuse quedan retenidos en su contexto de operación. Si falla el registro, se
realizan hasta dos llamadas con la misma orden, sin repetir la lectura; el acuse
se acepta únicamente si valida para esa orden. Otro material no reutiliza esa
referencia. Si no se confirma la auditoría, la respuesta queda no disponible y no
incluye datos. Esta retención permite recuperar un COMMIT ambiguo durante la
operación; no ofrece una API de recuperación del lector tras reiniciar el proceso.

Este montaje no activa rutas HTTP ni concede permisos. El ensayo PostgreSQL de
Personal27 y el consumo del puerto común se verifican aparte.

El decorador emite un resultado técnico al terminar cada llamada: correcto,
denegado, entrada inválida, cancelado o no disponible. Abarca también los rechazos
anteriores al servicio. El emisor JSONL común de L conserva la correlación y sólo
admite los campos del catálogo; no recibe persona, recurso, errores ni texto libre.
No marca la supresión HTTP. Un fallo o descarte de su cola o destino se contabiliza
sin invalidar el recibo de negocio ni repetir la lectura.

La dependencia de código es el puerto publicado por L en
`3d109adec189a9ad1c8317427b75768d4537c196` (#511). Los ensayos focales del lector
recorren ese emisor JSONL real y sus métricas, incluidos cancelación, falta de
correlación y fallos del destino técnico. No acreditan una lectura real con SQL,
HTTP o recuperación de proceso. La configuración positiva de proceso y canal del
registrador nominal pertenece a L/DBA y sigue siendo un requisito de montaje;
no se presume por conectar el emisor técnico.
