# Ensayo local del cliente VEC y el validador de firma

`./scripts/firma/validador_local.sh` compila en un directorio desechable el
cliente real `validadorautofirma` de VEC y el servicio de solo verificación de
GrxFirma/AutofirmaV2. Una fuente de prueba crea un PDF PAdES, una CA de firma,
un firmante y CRL sintéticas. El servicio crea aparte su CA TLS local. El
cliente llama a `POST /verify` con TLS 1.3, comprueba el certificado del
servidor por esa CA y por el nombre `localhost`, y envía un token Bearer
efímero. No se configura confianza en el sistema operativo.

El ensayo exige estos resultados de `Cliente.VerificarMotivado`:

| Caso | Estado y motivo VEC |
| --- | --- |
| Firma y CRL vigentes | `valida / verificada` |
| Certificado revocado | `no_valida / certificado_no_valido` |
| Falta la CRL intermedia | `indeterminada / revocacion_no_acreditada` |
| CA TLS incorrecta | `indeterminada / validador_no_disponible` |
| Nombre TLS incorrecto | `indeterminada / validador_no_disponible` |
| Token Bearer incorrecto | `indeterminada / credencial_rechazada` |
| PDF firmado alterado | `indeterminada / rechazada_por_validador` |

En todos los casos comprueba las huellas calculadas por VEC sobre ambos
contenidos. El positivo exige vínculo con el original, certificado,
firmante y revocación vigentes; los otros resultados no pueden acreditar una
firma. La ruta `/health` sirve solo para esperar al servicio. El ensayo no
invoca operaciones de firma ni acepta rutas de fichero como entrada del
cliente.

Hace falta la fuente local en `~/Trabajo/AutofirmaV2-vec-verificacion`, la
caché Go poblada y `bwrap`, `rsync`, `go`, `jq`, `curl`, `base64`, `prlimit` y
`timeout`. `AUTOFIRMAV2_SOURCE` permite elegir otra copia local.
`VEC_E3_SCRATCH_PARENT` permite elegir otro directorio desechable; por defecto
usa `/dev/shm/go-build`. El proceso queda en un espacio de nombres sin red
externa, con solo loopback, entorno mínimo, fuente y toolchain de lectura,
escritura limitada a `/work` y un máximo de 600 segundos. Al salir se eliminan
PDF, claves, certificados, CRL, tokens, respuestas, registros y binarios.

La prueba usa material sintético. Acredita el contrato técnico local entre
estos dos binarios y los fallos comprobados. No acredita un prestador real,
firma oficial, instalación VEC, autorización administrativa ni efectos en un
expediente.

## Evidencia anterior, 30/09/2026

El ensayo previo de AutofirmaV2 `97b2739e98f745a55918df854faad7eb71fd2d03`
y VEC `3796cf010dd93de07c1a83a443ff15a76b889ee1` usó `curl -k` contra
`POST /verify`: tres dictámenes HTTP 200, PDF alterado HTTP 400, `/sign`
HTTP 404 y `/health` HTTP 200. No ejercitó el cliente VEC ni la confianza TLS
del consumidor. Esa evidencia se conserva con su alcance original.
