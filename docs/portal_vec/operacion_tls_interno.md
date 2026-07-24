# Material TLS de la superficie interna

`vec-interno` carga el material TLS antes de escuchar y falla cerrado si el
proceso tiene UID 0 o si algun fichero/directorio no cumple la politica.

Despliegue directo admitido:

- proceso con UID/GID no privilegiado dedicado;
- todos los directorios del camino propiedad de `root`, sin escritura para
  grupo ni otros;
- certificado, clave y CA de clientes como ficheros regulares propiedad de
  `root`, nunca enlaces simbolicos;
- clave `root:<grupo-runtime>` con modo `0440`; certificado y CA pueden usar
  `0440` o un modo de solo lectura equivalente sin bits de escritura/ejecucion;
- `VEC_INTERNO_TLS_SERVER_NAME` contiene el DNS o IP exacto validado contra el
  SAN del certificado servidor;
- el certificado servidor contiene hoja y cadena emisora; el ultimo emisor
  incluido actua como ancla explicita de esa cadena;
- tickets de sesion, callbacks TLS, HTTP/2 y h2c permanecen desactivados.

La raiz entrega una capsula `*ServidorInterno`, no un `*http.Server`. La
capsula no publica handler, listener, configuracion TLS, certificados, pools ni
metodos `Serve*`. `EscucharYServir()` abre internamente el socket una sola vez,
crea un servidor HTTP/1.1 local con registro saneado y enlaza cada peticion al
`*tls.Conn` concreto mediante un token privado y material exportado por esa
conexion. `Apagar(ctx)` es la unica operacion adicional y realiza el cierre
ordenado sin exponer el transporte. `vec-interno` la invoca ante SIGINT o
SIGTERM con un limite de diez segundos.

Los DNS de SNI se comparan sin distinguir mayusculas ASCII. No se aplica
plegado Unicode. Cuando el SAN configurado es una IP literal, el cliente debe
omitir SNI, como exige TLS.

Una clave `0400 root:root` requiere un cargador provisionador separado que
entregue los bytes antes de bajar privilegios; el cargador directo actual,
ejecutado como usuario no privilegiado, usa `0440 root:<grupo-runtime>`.
Tras materializar la configuracion, el cargador sobrescribe best-effort sus
buffers PEM/PKCS#8 temporales. Esto reduce su permanencia, pero Go y
`crypto/tls` pueden conservar copias internas y necesariamente mantienen viva
la clave parseada mientras el servidor presta servicio.

La puerta `scripts/probar_carga_tls_interna_root.sh` reproduce el contrato:
prepara el volumen como `0:10001`, comprueba que UID 0 sea rechazado y carga el
servidor como `10001:10001` desde el volumen montado de solo lectura. Solo la
fase provisionadora recibe `/tmp`; las fases runtime root y no privilegiada no
montan tmpfs innecesarios.
