# Documento en custodia para la comprobación del ofrecimiento

F1-57, parte de servidor. RRHH no sube el documento ni lo elige de otro
módulo: declara el que custodia con una referencia legible y el navegador
calcula su huella SHA-256 sin enviar el archivo. Mismo patrón que las
operaciones de situación, el cese y la no incorporación.

## Qué cambia

- `documento_ref` de cada hecho deja de exigir `ref:<sha256>`. Admite la
  referencia opaca de B77 (`b77_documento_ref_opaco_v1`: 3 a 256 caracteres
  sin espacios, sin etiquetas DNI/NIE/NIF/pasaporte ni números de documento de
  identidad) o el evento de contrato que Bolsa recibe de Contratación temporal
  (`evento:ct:contrato-bolsa:<sha256>`). `ref:<sha256>` sigue siendo válida.
- La misma regla vive en Go: `ReferenciaDocumentoCustodiaValida` y
  `ReferenciaEventoContratoParticipacion` del dominio de Bolsa, usadas por
  `validarSolicitudComprobacion`.
- La lectura del histórico de contratos de la ficha añade `huella_sha256`
  (huella del evento conservado), para citar el contrato como evidencia.
  Es una función nueva `listar_contratos_participacion_v2`; la v1 queda igual.

## Activación

1. Instalar B141 una sola vez, después de B133 y B137 (lista única de main).
   No depende de B140 ni la toca: solo reemplaza
   `calcular_comprobacion_ofrecimiento_v1`, con cotejo de su cuerpo instalado
   (`27b65b67…8ab2c9`) y de su ACL, y crea la lectura v2.
   Postimagen del cálculo: `2f2ac96f3088314ad11971f801f165b637786d52ccfb11d1ef189b6018db890b`.
2. Instalar el binario. Sin B141, la lectura de contratos de la ficha responde
   503 y la comprobación sigue rechazando referencias legibles.
3. Prueba focal: `deploy/postgresql/bolsa_llamamientos/pruebas_sql/b141_documento_custodia.sql`
   (solo ROLLBACK) y `b133_calculo_y_denegacion.sql`.

Los actos ya registrados con `ref:<sha256>` no cambian. La huella no acredita
custodia ni firma del documento: solo permite comprobar después que es el mismo.

Vuelta atrás: recuperar el binario anterior. B141 no borra datos; no ejecutar
DOWN ni reinstalar B133.
