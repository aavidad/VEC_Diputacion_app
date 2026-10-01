# Ensayo local del validador E3

`./scripts/firma/validador_local.sh` copia la fuente local de AutofirmaV2 a un
directorio temporal, compila su servicio de solo verificación y crea un PDF
PAdES de prueba con una CA, un firmante y dos CRL sintéticas. Envía los bytes
firmados a `POST /verify` y exige tres dictámenes: `valida` con las CRL vigentes,
`no_valida` con una CRL que revoca al firmante e `indeterminada` al retirar
una CRL. Una alteración de los bytes firmados se rechaza con HTTP 400; no se
presenta ese rechazo como dictamen. También comprueba que `/sign` responde 404.

El ensayo requiere la fuente en `~/Trabajo/AutofirmaV2-vec-verificacion`, la
caché Go ya poblada y `bwrap`, `rsync`, `go`, `jq`, `curl`, `base64`, `prlimit` y
`timeout` locales. `AUTOFIRMAV2_SOURCE` permite señalar otra copia local.
`VEC_E3_SCRATCH_PARENT` permite elegir el directorio de trabajo desechable.
La ejecución tiene 180 segundos de límite, red aislada con solo loopback,
entorno mínimo y acceso de escritura limitado al directorio temporal. Al
terminar, elimina PDF, claves, certificados, CRL, peticiones, respuestas y
registros. Ningún material sintético se guarda en Git.

Este ensayo prueba el contrato técnico local con certificados de prueba. No
acredita prestadores reales, firma oficial, TSA, instalación VEC ni efectos
administrativos.

## Comprobación del 30/09/2026

Ejecutado con AutofirmaV2 `97b2739e98f745a55918df854faad7eb71fd2d03` y
VEC base `3796cf010dd93de07c1a83a443ff15a76b889ee1`. Las tres peticiones
`POST /verify` devolvieron HTTP 200 con `valida/verificada`,
`no_valida/certificado_no_valido` e
`indeterminada/revocacion_no_acreditada`, respectivamente. El PDF alterado
devolvió HTTP 400, `/sign` devolvió 404 y `/health` devolvió 200. Al salir no
quedaron procesos del ensayo ni directorios temporales `vec-e3-validador.*`.
