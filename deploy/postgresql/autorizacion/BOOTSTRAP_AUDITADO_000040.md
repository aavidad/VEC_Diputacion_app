# AUT40: auditoría de cada invocación de arranque

AUT40 conserva el efecto de AUT37, su recibo, el evento `bootstrap_operador`
AD171 y el reparto de dos personas de Aplicación más un perfil Sistemas. No
reutiliza la familia de fuentes CA/IS para atribuirle una acción de arranque.

La función pública conserva OID, firma, propietario, ACL y configuración. Su
preimagen exige SHA256 de `prosrc` AUT37
`bbbb9a849f3423b2eb010e792c01d383009688d9f7d89139664817dc184a1f94`.
El cuerpo original queda en un helper privado, solo ejecutable por el propietario
AUT. La guarda del LOGIN continúa comprobando las funciones públicas exactas.
La recuperación revalida configuración y caducidad después del bloqueo.

El wrapper abre un subbloque transaccional para el efecto. Ante un error elimina
la respuesta del subbloque revertido y registra el intento real en AD179, en la
misma corriente común. El cliente no elige resultado, motivo, actor o recurso.
Cada replay tiene otro intento, aunque conserva el recibo original.

ABI pública: `registrar_bootstrap_central_admin_v3(text,text)` devuelve exactamente
`estado`, `codigo`, `recibo`, `replay` y `auditoria_intento`. Estados: `permitido`,
`denegado` o `error`. Un fallo no devuelve un recibo ni un replay. La auditoría
contiene cinco coordenadas: referencia, secuencia, huella, correlación e instante.
El consumidor confirma COMMIT también en rechazo/error antes de anunciar esas
coordenadas como persistidas.

AD179 recibe doce cadenas, con actor `session_user`, acción
`registrar_bootstrap_central_admin_v3`, módulo `administracion`, proceso real
`postgresql`, canal `operacion_tecnica_privada` y finalidad `bootstrap_admin`.
Su recurso es `solicitud_bootstrap:<32 hex>` y su huella compromete la solicitud.
Motivos: `bootstrap_registrado`, `bootstrap_replay`, `bootstrap_denegado` y
`bootstrap_error`, ligados al resultado. No se registran personas o perfiles
inventados cuando todavía no existen.

La CLI nueva `cmd/vec-aplicar-admin-bootstrap` invoca esta función con una
aprobación externa, transacción SERIALIZABLE UTC sin SET ROLE, acuse privado
nuevo por invocación y sin reintentos automáticos. Un COMMIT incierto no produce
un recibo presentado como confirmado. Los errores locales, cancelaciones o
llamadas que no alcanzan COMMIT quedan fuera de esta cobertura explícita.

El ensayo privado de dirección usa POST-H9, fuentes CA/IS reales del clon,
certificados cliente sintéticos reales y una unidad publicada por Personal33.
El positivo original AUT37 creó tres perfiles, dos vínculos de certificado y
un recibo/outbox. Tras AUT40, la CLI recuperó el recibo idéntico, registró rechazo
por otra huella y un error controlado de un getter. El reinicio conservó el
recibo y las tres asignaciones. Las actas privadas contienen referencias y
comandos exactos. No acredita principal, producción, FNMT ni firma legal.

La primera alta mediante el wrapper y la CLI se ensayó además sobre un objetivo
nuevo recuperado del frío anterior. Un fallo real inyectado en el append AD179
revirtió el primer efecto: quedaron cero bootstrap y perfiles del plan, sin
avanzar la auditoría. Al retirar el fallo, la CLI confirmó la primera alta con
replay falso; replay y reinicio conservaron el mismo recibo, tres perfiles,
dos personas Aplicación y una confirmación AD171. Cada llamada permitida añadió
su propio intento AD179. La base anterior y sus recibos permanecieron intactos.
