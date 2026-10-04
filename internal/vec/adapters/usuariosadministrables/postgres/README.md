# Lecturas administrativas de usuarios

`Nueva` exige un pool PostgreSQL con LOGIN perteneciente únicamente a
`vec_admin_usuarios_lector`, el emisor central V3, la fuente de instantánea de
autorización, el registrador común de intentos, reloj y ámbito privado de
organización/unidad. Comprueba estructura del LOGIN y presencia de las dos
fachadas AUT43 y del consumidor AD185. Si falta una dependencia, no construye
la fuente.

El puerto es `ports.FuenteUsuariosAdministrables`. `ListarUsuarios` recibe los
filtros de perfil, unidad y estado, además del cursor; `ConsultarUsuario`
recibe una referencia de Persona. La organización y unidad salen de la
configuración privada del constructor. El material, conjunto y cursor siguen
el orden y las huellas de AUT43. El emisor recibe recurso, material, acción,
audiencia, correlación interna, actor, evidencia V2 e instantánea; debe devolver
una exportación V3 real. Este paquete no implementa ese emisor.

Cada lectura válida abre una transacción `SERIALIZABLE` de lectura/escritura,
llama a una fachada AUT43, comprueba el acuse V3 y la forma exacta del resultado,
y solo entrega metadatos tras un COMMIT confirmado. Un resultado nulo o vacío
también requiere decisión y consumo nuevos. Ante un COMMIT incierto devuelve
error y resultado vacío, sin volver a consultar. Tras terminar el rollback,
registra denegación o error mediante el puerto común con el resultado y vínculo
V2 originales y la configuración privada.

El resultado contiene únicamente referencia de Persona, unidad, versión
opcional de denominación y todos los perfiles del ámbito con seis campos. No
aplica límite adicional a perfiles; SQL limita la página a 50 Personas. No
monta rutas ni adapta el DTO heredado de siete fachadas.

Estado: código y pruebas focales locales. Faltan el montaje del emisor real,
el ensayo de AUT42/AD184/CA32/CA34/AUT43/AD185 en PostgreSQL, las revisiones
sensibles y el recorrido del canal ADMIN.
