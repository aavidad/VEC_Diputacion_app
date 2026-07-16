# Flujo durable de firma de baremaciones

## Decisión adoptada

La firma, custodia, retención y confirmación de una baremación se coordinan mediante una saga durable versionada. La petición web no conserva capacidades de autorización ni mantiene el proceso completo en memoria. Cualquier réplica de la aplicación puede reanudar el expediente usando los puertos hexagonales definidos para repositorio, protección criptográfica, ejecución de efectos y generación de referencias.

La fachada deriva en cada petición la persona y el perfil desde la sesión autoritativa. La orden externa solo aporta la referencia opaca del flujo y la clave de idempotencia; no puede declarar el actor, el perfil ni permisos. La clave de idempotencia y el vínculo estable persona-perfil se almacenan como HMAC, nunca en claro.

## Estado persistido

El expediente contiene únicamente:

- referencias de negocio y de efectos;
- versión de control de concurrencia;
- HMAC de la petición, del actor y de la clave de idempotencia;
- puntos de control declarados o completados;
- recibos y huellas de resultados;
- una proyección mínima para iniciar la interacción de firma;
- el resultado final exacto cuando el flujo termina;
- un sobre de estado cifrado con AEAD y un sello HMAC de toda la representación canónica.

El sobre actual usa AES-256-GCM. El nonce, la referencia versionada de clave y la huella del cifrado forman parte del estado sellado. Los métodos genéricos JSON, texto y binario del tipo protegido fallan de forma cerrada, y su formateo para registros siempre muestra un valor expurgado.

No se deben introducir en el estado de trabajo contraseñas, cookies, credenciales temporales, direcciones firmadas de almacenamiento ni capacidades reinyectables. Cada efecto debe volver a obtener identidad y autorización desde el contexto de la petición vigente.

## Orden y puntos de control

Los pasos admitidos son, estrictamente, preparación, finalización de firma, custodia, retención, reserva de cambio y confirmación. Antes de llamar a un sistema que produzca un efecto, la fachada persiste un punto de control `declarado` con:

- una referencia de efecto estable;
- una clave HMAC de idempotencia estable;
- el instante de declaración.

Solo después se invoca el ejecutor. Al obtener un recibo válido se cifra el nuevo estado y se persiste el mismo punto como `completado`. Un reintento reutiliza la referencia y la clave ya declaradas; nunca genera otras para el mismo paso.

El ejecutor productivo deberá recuperar primero el resultado por la pareja `(referencia de efecto, HMAC de idempotencia)`. Si la llamada anterior terminó en un tiempo de espera ambiguo, esa recuperación es la que impide repetir semánticamente la firma, custodia, retención, reserva o confirmación. No existe una garantía universal de “exactamente una vez” frente a un proveedor arbitrario: el conector debe usar la idempotencia nativa del proveedor o mantener un diario transaccional durable de efectos y recibos.

## Concurrencia y recuperación

El repositorio aplica versión esperada, arrendamiento exclusivo y número de cercado creciente. Una escritura solo se admite si coinciden la versión, el propietario, el cercado y la vigencia del arrendamiento. Una réplica antigua no puede confirmar estado después de que otra haya adquirido un cercado posterior.

La exclusión no sustituye la idempotencia. Si un efecto tarda más que el arrendamiento, dos réplicas podrían intentar recuperar el mismo efecto estable, pero el ejecutor no debe producir dos efectos y solo el propietario vigente podrá guardar el resultado.

El arrendamiento incorpora una capacidad nominal de 256 bits que solo vive en
un cierre privado e inmutable. El repositorio conserva exclusivamente su HMAC;
ni el token ni la clave HMAC se guardan como cadenas, slices, expediente,
auditoría u outbox. La capacidad y el arrendamiento bloquean JSON, texto,
binario, gob y XML, y redactan `fmt` y `slog`. Copiar los metadatos, reutilizar
un token de otro arrendamiento o presentar un cercado obsoleto no concede
autoridad. Liberar de nuevo un arrendamiento que ya fue liberado es idempotente,
pero un token incorrecto contra uno vigente falla sin eliminarlo. El adaptador
en memoria captura una clave efímera solo para probar la semántica; el adaptador
productivo deberá realizar el HMAC en un KMS/HSM con referencias de clave
versionadas y rotación. El método actual que recibe la clave en bytes pertenece
solo al adaptador en memoria: el conector productivo deberá ofrecer una
operación de sellado con clave no exportable, o aplicar el HMAC en KMS sobre el
compromiso SHA-256 del token; nunca exportará la clave HSM al proceso.

Si el flujo ya está completado, la fachada devuelve el resultado final almacenado byte a byte en sus campos lógicos, sin volver a invocar ningún efecto. Si se reinicia la aplicación o cambia la réplica que atiende la petición, la nueva instancia puede descifrar y continuar siempre que disponga de la clave histórica identificada por `claveRef`.

La proyección de lanzamiento solo incluye referencias opacas, canal e instantes. No contiene documentos, direcciones firmadas ni credenciales. Se devuelve únicamente después de persistir el punto de preparación completado.

## Adaptadores implementados

Se han implementado los contratos del núcleo, la fachada de aplicación y adaptadores en memoria para:

- repositorio con índice único de idempotencia, control de versión, arrendamiento y cercado;
- protección AES-256-GCM con clave inyectada;
- referencias criptográficamente aleatorias.

El adaptador en memoria sirve para pruebas adversarias y desarrollo. No es almacenamiento productivo ni sobrevive al reinicio del proceso. Las pruebas cubren reintentos ambiguos en cada efecto, reanudación con otra instancia, concurrencia entre réplicas, clave de idempotencia cruzada, reutilización con otros datos, clave AEAD equivocada, estado alterado, ausencia de sesión y recuperación exacta de un flujo completado.

## Trabajo productivo pendiente

Antes de publicar esta fachada deben completarse estos conectores y controles:

1. Repositorio PostgreSQL/Oracle con transacciones, índice único sobre el HMAC de idempotencia y actualización condicional de versión, propietario y cercado. Las migraciones deben conservar el historial de puntos de control y prohibir transiciones regresivas.
2. Protector conectado a KMS o HSM. Debe cifrar con la clave activa y descifrar con un anillo de claves históricas, registrar la versión criptográfica y permitir rotación sin detener flujos abiertos.
3. Ejecutor que adapte el motor actual de baremación y firma. Debe mantener un diario durable de efectos/recibos o demostrar idempotencia nativa para Autofirma, almacenamiento, retención y repositorio de baremaciones.
4. Resolutor interno de la proyección de lanzamiento, protegido por sesión revalidada, autorización de uso único, vencimiento corto y controles anti-repetición. La referencia de lanzamiento nunca debe ser resoluble desde la superficie pública anónima.
5. Adaptadores HTTP/API/CLI que construyan el estado inicial exclusivamente desde datos ya validados por el servidor. Ningún DTO debe aceptar `EstadoTrabajoInicial`, actor, perfil, sellos o puntos de control enviados por el cliente.
6. Auditoría institucional y bandeja transaccional: registrar inicio, declaración, recuperación, finalización, error ambiguo, cambio de cercado y resultado, sin volcar estado cifrado, HMAC completos ni datos personales innecesarios.
7. Métricas y reconciliador para puntos declarados que no hayan concluido, objetos huérfanos y arrendamientos vencidos. La reparación debe recuperar recibos; no repetir efectos a ciegas.
8. Pruebas de integración con caída real de base de datos, pérdida de respuesta después de confirmar, vencimiento de arrendamiento durante un efecto, rotación de clave, dos nodos y recuperación tras reinicio completo.

Hasta terminar esos adaptadores, esta unidad demuestra y fija la semántica segura del núcleo, pero no debe presentarse como un circuito productivo conectado de extremo a extremo.
