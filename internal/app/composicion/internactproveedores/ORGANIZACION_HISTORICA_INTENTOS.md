# Intentos fallidos de consulta histórica de Organización

La consulta nominal conserva su lectura, consumo V3, recibo y auditoría en la
transacción existente de Personal. Este corte conecta los resultados denegados
y los errores con el registrador común de intentos, después del retorno del
repositorio y del cierre de su transacción. El éxito no añade otro intento.

## Composición

La raíz interna recibe `Configuracion.IntentosOrganizacionHistorica`, con:

- un `RegistradorIntentosOrganizacionHistorica` que implementa el puerto común
  `AppendIntentoAuditoria` y su `PreflightIntentoAuditoria`;
- proceso, canal interno y tres referencias de motivo publicadas por el gobierno
  correspondiente: denegación, entrada inválida y servicio no disponible.

El registrador usa su pool nominal segregado. Este módulo no abre otro DSN,
crea tablas, publica concesiones ni configura motivos por petición. Si faltan
esas dependencias o falla el preflight, Organización queda sin montar; el resto
de la raíz interna conserva su circuito.

`NuevaConsultaOrganizacionHistoricaConIntentos` conserva el resultado F1 y su
vínculo originales antes de validar o leer. También exige la correlación emitida
por el servidor. La captura copia el sello original para auditoría aunque el
vínculo haya caducado entre HTTP y aplicación; no renueva la sesión ni permite
leer con ella. El proveedor V3 usa esa captura y esa misma correlación; no
resuelve otro perfil al registrar el fallo. El organismo del intento procede
de la fuente vinculada, no de un filtro HTTP. Su referencia en auditoría usa
`personal:organizacion_historica:<organismo>`; el recurso V3 conserva su valor.

El servicio sólo confirma la denegación tras recibir un acuse válido. Ante un
fallo ambiguo del registrador, repite la misma orden y referencia; si vuelve a
fallar, devuelve servicio no disponible. El HTTP no añade otra entrada CT162
para esa denegación. La frontera anterior al perfil mantiene su mecanismo
existente y no atribuye una identidad supuesta al puerto posperfil.

## Alcance comprobado y dependencias pendientes

La validación previa acreditó las pruebas focales de aplicación, PostgreSQL,
proveedor, composición interna y HTTP en modo normal y con detección de carreras;
`go vet` de esos cinco paquetes también pasó. Las regresiones añadidas tras
revisión para recursos simples y caducidad entre HTTP y captura están pendientes
de ejecución. Las referencias del constructor se comprobaron con
gopls. Semgrep con reglas Go locales no encontró resultados. Gosec emitió treinta
avisos en líneas anteriores, ninguno en las líneas añadidas; esto no acredita
una auditoría completa del repositorio.

La configuración positiva del registrador nominal todavía debe proceder de la
fuente de gobierno común. No se ha demostrado aquí un montaje operativo nuevo,
un recorrido de navegador ni PostgreSQL real. Las pestañas web conservan su
activación anterior.

La publicación de fuentes sigue cerrada: Personal11 y AD3-52 sólo permiten
preparación y conciliación. Este corte no publica una RPT, una plantilla ni una
acreditación de fuente, acto o custodia. Las migraciones instaladas se conservan.
