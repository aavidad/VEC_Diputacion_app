# Contrato VEC → verificador de firma (v1)

Este contrato define un servicio verificador **separado** de VEC. El adaptador
`internal/vec/documentos/adapters/verificadorhttp` lo consume. AutofirmaV2
aporta contexto de formatos y validación, pero su servicio local y sus canales
web legacy no son este destino ni acreditan la integración.

## Transporte y despliegue

- Dirección despliega el verificador en la red interna de Cidonia. VEC configura
  fuera de Git un origen `https://host[:puerto]` fijo, la CA del
  servidor y un certificado cliente con su clave. Se exige TLS 1.3 con
  autenticación mutua. El servicio valida la identidad del cliente y limita
  su autorización a esta operación.
- VEC envía `POST /v1/verificaciones`, `Content-Type: application/json`, sin
  redirecciones, proxy ambiental ni reintentos automáticos. Límite de diez
  segundos y 16 MiB por contenido original y artefacto firmado. La respuesta `200` tiene como
  máximo 16 KiB y `Content-Type: application/json`.
- El servicio nunca recibe una credencial de usuario VEC ni determina permisos
  de acceso al expediente. VEC autoriza el caso de uso antes de llamar y
  conserva la auditoría y la transición duradera bajo su autoridad.

## Solicitud

```json
{
  "version_contrato": 1,
  "documento_id": "ref:<64 caracteres hexadecimales minúsculos>",
  "version": 1,
  "huella_original_sha256": "<64 caracteres hexadecimales minúsculos>",
  "huella_firmado_sha256": "<64 caracteres hexadecimales minúsculos>",
  "contenido_original": "<base64 estándar de los bytes originales custodiados>",
  "contenido_firmado": "<base64 estándar del artefacto firmado>"
}
```

Ambas huellas se calculan sobre los bytes que viajan respectivamente en
`contenido_original` y `contenido_firmado`. VEC recupera la versión inmutable
custodiada antes de la llamada y comprueba de nuevo su huella. El verificador
debe comprobar que el artefacto firmado corresponde a **ese original exacto**
conforme al formato y política aplicables;
que una firma sea criptográficamente válida sobre otro contenido no basta.

## Respuesta

```json
{
  "version_contrato": 1,
  "estado": "valida",
  "huella_original_sha256": "<eco de la huella del original comprobado>",
  "huella_firmado_sha256": "<eco de la huella de los bytes firmados comprobados>",
  "firmante_ref": "ref:<64 caracteres hexadecimales minúsculos>",
  "certificado_huella_sha256": "<huella SHA-256 del DER del certificado>",
  "sello_tiempo_estado": "valido",
  "revocacion_estado": "vigente",
  "vinculo_original": true
}
```

`estado` es `valida`, `no_valida` o `indeterminada`. Para `valida` se exigen
ambas huellas idénticas a la solicitud, `vinculo_original: true`, firmante y certificado identificados,
sello de tiempo `valido` y revocación `vigente`. `no_valida` e
`indeterminada` pueden dejar vacíos los campos que no se pudieron comprobar,
pero devuelven las dos huellas de la solicitud. La respuesta no transporta
nombre, DNI, certificado completo ni documento en claro. El servicio conserva
la evidencia detallada bajo su propio régimen; VEC solo recibe el resultado
mínimo necesario.

Una respuesta desconocida, incompleta, demasiado grande, lenta, con huella
distinta, con error de red o con estado indeterminado **no habilita** la
transición a firmado. Tampoco la habilita un resultado válido sin sello o sin
estado positivo de revocación. VEC no infiere firma de la autenticación del
actor, de un PDF, de un borrador o de un recibo local.

## Trabajo del servicio verificador

El servicio que despliegue Dirección debe implementar esta ruta y su versión,
verificar integridad y vinculación exacta con el original, cadena de confianza,
uso y vigencia del certificado, revocación y sello de tiempo según la política
institucional admitida. Debe devolver `indeterminada` cuando una comprobación
necesaria no pueda concluirse, sin convertir indisponibilidad OCSP/CRL o de
sellado en `valida`. Su implantación, configuración de confianza, pruebas de
formatos admitidos y despliegue son una dependencia independiente de este
adaptador; ningún estado de firma se declara por la mera existencia de la ruta.
