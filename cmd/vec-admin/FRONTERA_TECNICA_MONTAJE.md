# Montaje de auditoría técnica ADMIN — AD189

El modo `metadatos_v1` exige un décimo pool: `pool_frontera_tecnica`.
Es una ruta privada a un DSN, con LOGIN distinto de los otros nueve y pertenencia
exclusiva al grupo `vec_admin_frontera_tecnica_ejecutor`. El arranque acredita
su configuración vigente y los dos puntos de entrada AD189. No concede roles.
La configuración de proceso, canal y plazo es la del overlay privado; el plazo
máximo sigue siendo dos segundos. El overlay heredado debe añadir esta ruta.

La composición monta el auditor nominal existente y el registrador técnico.
Antes de V2, usa la correlación privada creada al entrar en HTTP y conserva
el mismo evento si el COMMIT queda incierto. Después de una sesión resuelta,
la marca privada `SesionResuelta` impide volver a la familia técnica. Un V2
usable mantiene actor y evidencia originales para la auditoría nominal; un V2
incompatible cierra con 503 y no inventa una Persona ni oculta errores de AD169.
Si el V2 original sigue usable, el rechazo usa la acción cerrada `consultar` para GET o `escribir` para POST
del catálogo privado y la correlación privada original del middleware, aunque
la instantánea o la correlación devueltas por el resolutor sean incompatibles.
El vector atraviesa handler, compuesto y auditor nominal: confirma un append
ERROR con ese V2 antes de escribir el 503, sin registrar nada en AD189.

Los rechazos de host, red, cadena o CRL que llegan al handler HTTP pasan por
ese auditor antes de responder. Si no se confirma el registro, devuelve 503.
Un fallo al crear la correlación privada devuelve 503 antes del despacho y
no sustituye el identificador por una cabecera. Esto no cubre errores TLS
anteriores a HTTP. La prueba de vida sin montaje de perfiles conserva su
comportamiento anterior.

El verificador común y `vec-auditoria-verificar` admiten el esquema propio
`vec.auditoria.verificacion.frontera-admin-tecnica.v1`. Cotejan el catálogo
cerrado de ocho resultados/códigos, material y enlace AD189, junto a la historia
anterior. Los esquemas anteriores no admiten la familia nueva. La salida
mantiene explícita la autenticidad del checkpoint como no comprobada.

Pruebas de los cinco paquetes modificados: normal, race y vet verdes. También
se comprobaron el parser y el CLI: normal, race y vet. Los vectores cubren
sesión resuelta incompatible, V2 original conservado, rechazo de cabeceras
como fuente de correlación, host/red/revocación/CRL en mTLS sintético real,
fallo de auditoría sin despacho ni datos, pool técnico obligatorio y exclusivo,
JSON ambiguo/null/extra/omitido, huellas y preservación de historia.
Las pruebas de configuración usan TMPDIR privado: `/tmp/.git` es ajeno y
haría que la guarda fuera de Git rechazase el material de prueba.

Semgrep local y vecsilencio no detectaron fallos nuevos. Gosec encontró sólo
tres G703 heredados en `configuracion_privada.go`: son las comprobaciones
Lstat del directorio privado y de sus ancestros Git. Las rutas son absolutas
y limpias, se rechazan symlinks, se exige propietario y 0700/0600, se abren
mediante `os.OpenRoot`/`O_NOFOLLOW` y se cotejan identidad y tamaño al leer.
Esos puntos no cambian ni permiten elegir archivos desde HTTP. El paquete
CLI del verificador no presenta avisos gosec.

El escritor del clon acreditó por separado SQL AD189 exacta
`d69b8866d90ccb756fa412640e1342bd26bb9a7381a4436ec92b2952dc74d4fa`:
UP y vector verdes una sola vez; evento 6256 con LOGIN real, replay y reinicio
con el mismo recibo, material distinto rechazado y ACL/ventana negativas.
Acta privada SHA256
`a2b228462b83356405999b0da234134e88df61372cf4e54303e4ce35d5f6b7ac`.
Este montaje no modifica SQL ni acredita una lectura favorable V3 o el
recorrido operativo completo de diez LOGIN contra PostgreSQL.
