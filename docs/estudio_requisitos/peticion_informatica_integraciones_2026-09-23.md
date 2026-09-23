# Petición a Informática — integraciones de VEC

23/09/2026. Complementa `dudas.md` (preguntas funcionales a RRHH): aquí solo lo técnico que
necesitamos de Informática de la Diputación para conectar VEC con los sistemas corporativos.
No pedimos credenciales, certificados ni datos personales reales por correo: basta con la
documentación, un entorno de pruebas y la persona responsable de cada sistema.

La primera dependencia es **identidad institucional para la aplicación interna**. Su contrato
debe cerrarse con Sistemas y Seguridad antes de habilitar la primera composición productiva
en `cmd/vec-interno`. El resto de integraciones sigue en la cola del producto; esta petición
no afirma que ninguna esté disponible ni convierte autenticación en firma documental.

## Texto para enviar

Buenos días:

VEC va a sustituir a CONVOCA en la gestión de bolsas y en los procesos selectivos, y a integrar
la contratación temporal con los sistemas corporativos. Para conectar cada sistema necesitamos,
de cada uno: responsable técnico, documentación de la interfaz (versión vigente), entorno de
pruebas con datos ficticios y el procedimiento para dar de alta a VEC como aplicación cliente.

1. **Sede electrónica (MOAD).** Inscripción telemática en procesos selectivos: ¿cómo registra
   la Sede una solicitud presentada desde otra aplicación (servicio, formato del asiento, justificante
   de registro, número y fecha)? ¿Cómo se identifica el interesado (Cl@ve, certificado, DNIe) y qué
   datos de identidad recibe VEC? ¿Cómo se notifican subsanaciones y resoluciones al aspirante?
2. **Pasarela de pago de tasas.** Interfaz para iniciar el pago de la tasa de examen, confirmación
   del cobro (síncrona o por aviso), referencia y justificante, devoluciones, exenciones y
   conciliación. ¿Qué sistema es la fuente de verdad del pago?
3. **Portafirmas.** Interfaz para enviar documentos a firma, orden de firmantes, estados, devolución
   y obtención del documento firmado con su evidencia.
4. **GINPIX.** Interfaz para comunicar altas, variaciones y ceses de personal temporal y la
   respuesta que confirma la recepción.
5. **Correo corporativo (SMTP).** Servidor, puerto, TLS, cuenta remitente y límites de envío para
   las comunicaciones a candidatos; y, si existe, servicio de **SMS** corporativo.
6. **Identidad corporativa para VEC interno — respuesta prioritaria.** Necesitamos acordar con
   Sistemas y Seguridad el contrato de una aserción **firmada por petición** que la pasarela
   interna entregue a Go por mTLS: especificación versionada, esquema, ejemplos y vectores de
   prueba ficticios, emisor y audiencia exclusivos, claves públicas y política de revocación.
   El detalle de las preguntas está debajo. También necesitamos el responsable del directorio
   o IdP de empleados (altas, cambios, bajas), las autoridades de certificación admitidas y
   los servicios OCSP/CRL. Una cuenta o certificado válido por sí solo no concede permisos VEC.

Gracias. Podemos concertar una reunión breve por sistema.

## Anexo técnico para cerrar la primera composición interna

El contrato de [acceso interno](acceso_interno_tecnicos_administracion.md) exige pasarela
interna, Kerberos/SPNEGO, certificado protegido, mTLS pasarela→Go y autorización VEC por
petición. La implementación actual recibe bytes protegidos mediante
`httpseguridad.VerificadorAsercionProtegida`, calcula garantía con
`httpseguridad.EvaluadorGarantia` y compone el listener en `composicion/interna`. Estos
puertos son puntos de integración; **faltan el emisor corporativo, sus claves y la política
aprobada**. Solicitamos respuestas documentadas a las siguientes preguntas, con valores de
prueba inventados. Las etiquetas del ejemplo no fijan el formato definitivo.

### Aserción firmada y claves de verificación

1. ¿Qué formato exacto y versión emitirá la pasarela (por ejemplo, perfil COSE o JWS),
   cómo se canonizan los bytes firmados y qué campos son obligatorios? Especificar tipo de
   transporte, codificación, tamaño máximo, cabeceras protegidas y tratamiento de campos
   desconocidos, duplicados o no canónicos. Aportar un ejemplo sintético válido y vectores
   negativos: firma, `alg`, `kid`, audiencia, canal, caducidad y repetición alterados.
2. ¿Qué algoritmos se admiten, quién genera y custodia la clave de firma y cómo se distribuye
   a VEC el catálogo **público** de `kid`/algoritmo/emisor? Definir activación, solapamiento
   de claves, retirada, revocación urgente, propagación y comportamiento ante catálogo o
   servicio de revocación indisponible. No enviar claves privadas ni certificados de cliente.
3. Confirmar identificadores exactos, estables y separados de emisor y audiencia para la
   superficie interna; su relación con cada pasarela y entorno; el identificador opaco de
   sujeto; la cuenta corporativa ordinaria o privilegiada; y el vínculo verificable entre
   cuenta, sujeto y credencial. Identificar expresamente la **fuente maestra** y el contrato
   de altas, bajas y cambios para la cadena **cuenta → persona → perfil → organización/ámbito**:
   propietario de cada dato, procedencia, identificadores opacos, versión, vigencia, estado,
   revocación y mecanismo de consulta o eventos con recuperación de cambios perdidos.
   Precisar qué autoridad valida la asignación central de perfiles y ámbitos en VEC; los
   grupos del directorio pueden informar una asignación, pero no conceden permisos por sí
   solos. Necesitamos el ciclo de alta, baja, cambio de unidad y desactivación, con objetivo
   temporal de revocación. No usar nombre, correo o DNI como prueba de igualdad ni enviar
   valores reales en esta respuesta.
4. Definir cómo se afirman y comprueban `acr`, `amr`/factores, método primario, instante de
   autenticación, principal Kerberos, referencia de certificado y referencias opacas de cada
   evidencia y **grupo criptográfico raíz**. Precisar la política y su versión para evaluar
   garantía: Kerberos y certificado deben vincularse al mismo sujeto y acreditar mecanismos
   independientes. Si PKINIT y la prueba de certificado usan la misma tarjeta/clave, indicar
   qué reto adicional aprobado demuestra independencia; no contarlos dos veces. Aclarar
   revocación de certificado, PIN/biometría y protección no exportable exigida.
5. Confirmar `jti`/ID de aserción, ID de sesión, `iat`, `nbf`, `exp`, instante de autenticación,
   tolerancia de reloj y vida máxima. El receptor limita actualmente la aserción interna a
   **3 minutos**, antigüedad de autenticación a **15 minutos** y desviación de reloj a
   **20 segundos**; confirmar compatibilidad o acordar una revisión explícita. Definir emisión
   por petición, reserva durable y consumo antirrepetición, reintentos tras fallo, revocación
   de sesión y comportamiento ante caída del registro. Una misma aserción no debe autorizar
   dos efectos; los reintentos de negocio usarán su idempotencia propia.

Ejemplo **solo ilustrativo, sintético y sin firma válida** del contenido que VEC necesita
obtener *después* de verificar la firma (los nombres/codificación finales dependen de 1):

```json
{
  "version": "ejemplo-v1",
  "jti": "asercion-sintetica-001",
  "iss": "https://idp.ejemplo.invalid/interno",
  "aud": "vec-interno-ejemplo",
  "superficie": "interna_corporativa",
  "sub": "sujeto-opaco-001",
  "cuenta": {"id": "cuenta-ficticia-001", "privilegiada": false},
  "sesion_id": "sesion-sintetica-001",
  "auth_time": "2026-09-23T09:00:00Z",
  "iat": "2026-09-23T09:00:05Z",
  "nbf": "2026-09-23T09:00:05Z",
  "exp": "2026-09-23T09:02:05Z",
  "acr": "politica-de-ejemplo-sin-aprobar",
  "metodo_primario": "kerberos",
  "factores": [
    {"amr": "kerberos", "sujeto": "sujeto-opaco-001", "principal": "principal-ficticio", "evidencia_ref": "evidencia-uno", "grupo_criptografico_ref": "grupo-uno", "verificado_en": "2026-09-23T09:00:00Z"},
    {"amr": "certificado", "sujeto": "sujeto-opaco-001", "credencial_ref": "certificado-ficticio", "evidencia_ref": "evidencia-dos", "grupo_criptografico_ref": "grupo-dos", "verificado_en": "2026-09-23T09:00:00Z"}
  ],
  "metodo_http": "POST",
  "ruta": "/api/ejemplo",
  "huella_cuerpo_sha256": "sha256:valor-sintetico-no-criptografico",
  "nonce": "nonce-sintetico-001",
  "canal_vinculado_ref": "tls-exportador:sha256:valor-sintetico-no-criptografico"
}
```

Ejemplos negativos igualmente ficticios para los vectores a facilitar: sustituir
`aud` por `vec-publico-ejemplo`; repetir `jti` en una segunda petición; conservar la
firma pero cambiar `metodo_http` a `GET`; o mantener el contenido y establecer otro
`canal_vinculado_ref`. Todos deben denegarse. El verificador de identidad actual
devuelve campos tipados, pero la comprobación de método, ruta, cuerpo y nonce exige
cerrar el formato y su acoplamiento a la petición HTTP real; este ejemplo no acredita
que esa comprobación ya esté implementada.

### Transporte y vínculo de la petición

6. Acordar el punto exacto donde se verifican Kerberos, certificado, revocación y mTLS del
   dispositivo; topología y DNS; identidad SAN y CA de la pasarela; protocolo de salto
   pasarela→Go; y cómo se impide acceso directo al listener interno. Go exige un handshake
   mTLS real, cadena cliente verificada y par autorizado: ni CIDR ni PEM reenviado en una
   cabecera sustituyen ese canal.
7. Definir una **única** cabecera de transporte de la aserción protegida, con nombre,
   codificación, límite y rechazo de duplicados. La pasarela debe eliminar antes todas las
   cabeceras de identidad, aserción, certificado, perfil y autorización aportadas por el
   cliente, y establecer solo la propia. Confirmar también rechazo de `Authorization` y
   cookies en la web interna, y el tratamiento de `Forwarded`/`X-Forwarded-*`.
8. La aserción debe ligar audiencia, superficie, método, ruta canónica, huella canónica del
   cuerpo, nonce y caducidad a **la misma petición**. Precisar normalización de ruta y query,
   método, cuerpo vacío y representación comprimida, así como el punto de cálculo de la
   huella. Para el canal pasarela→Go, la referencia que Go calcula es
   `tls-exportador:sha256:` + SHA-256 del resultado de `ExportKeyingMaterial` con etiqueta
   `VEC-Diputacion-Canal-Identidad-v1`, contexto `interna_corporativa` y longitud 32 bytes.
   Confirmar que la pasarela puede derivar ese valor del **mismo handshake TLS**, incorporarlo
   a los bytes firmados y emitir una aserción nueva si cambia la conexión. Aportar vectores
   ficticios de coincidencia y de canal distinto. No enviar el secreto exportado.
9. ¿Dónde y cómo elige la persona su **único perfil activo** por petición? La pasarela solo
   podrá transmitir el selector autenticado acordado; VEC resolverá su asignación central
   exacta, vigente y con ámbito, sin sumar perfiles ni convertir grupos AD en permisos.
   Separar cuenta ordinaria de privilegiada y superficie de administración; confirmar qué
   casos requieren reautenticación o dos personas distintas. Documentar origen y CSRF de las
   operaciones mutables al usar credenciales ambientales Kerberos/mTLS.

### HSM/KMS y material de autorización V3

10. ¿Qué servicio HSM/KMS corporativo puede ejecutar **HMAC-SHA-256 sin exportar la clave**
    para el alta durable de identidad? Precisar API, autenticación del servicio, separación
    por propósito (`asercion`, `sesion`, `sujeto`, `cuenta`, `cuenta_ordinaria`) y dominio ligado
    a emisor/namespace; identificador y versión de clave; rotación, recuperación de versiones
    históricas, revocación, disponibilidad y cuotas. VEC almacena solo huellas HMAC y
    coordenadas de clave; sin conector aprobado debe fallar el arranque de esta frontera.
11. Para las atestaciones de autorización **V3**, acordar el firmante o HSM, claves públicas
    verificables, `kid`, autorización de uso de clave, rotación y revocación, y entrega de
    material público por canal gestionado. El perfil operativo V3 vigente exige
    **COSE_Sign1 con Ed25519/EdDSA**; el hecho de que el verificador COSE genérico también
    soporte ES256 no lo convierte en algoritmo aprobado para V3. Cualquier cambio de suite
    requeriría decisión y versión formales, adaptación y pruebas antes de su uso. Confirmar
    el suministro del material Ed25519 y aportar vectores ficticios de firma y fallo para el
    perfil V3: cabeceras protegidas `alg` y `kid`, CBOR determinista y ninguna cabecera no
    protegida. La verificación criptográfica común no acredita por sí sola confianza,
    vigencia ni política V3. Distinguir estas claves de las de la aserción de identidad y
    de cualquier firma legal de documentos.

**Resultado solicitado:** especificación versionada y responsable de aprobación, esquema
y vectores ficticios positivos/negativos, parámetros acordados de cada entorno, procedimiento
seguro para distribuir material público y plan de pruebas con revocación, rotación, replay,
reintento y caída de dependencias. Los secretos, datos personales reales, claves privadas y
configuración privada se intercambiarán únicamente por el circuito corporativo aprobado,
nunca en esta petición ni por correo ordinario. Hasta validar este contrato, la composición
interna continúa cerrada; los contratos de Sede, tasas, Portafirmas, GINPIX y SMTP permanecen
pendientes para sus recorridos respectivos.
