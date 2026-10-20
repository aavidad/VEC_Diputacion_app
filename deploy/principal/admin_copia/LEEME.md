# Etapa ADMIN de la copia local (F1-131, primer corte)

`arranque.py` incorpora el guion de arranque de Administración para el conjunto 7.
Prepara y aplica, por separado, las fuentes sintéticas de dos personas
administradoras, una unidad y el gobierno inicial de Administración. El guion
consume las herramientas Go y el material de identidad del kit preparado para
esa copia. No crea aprobaciones externas ni confirma por sí solo el recorrido
completo de la copia.

## Entrada

El preparador de la copia genera un `config.json` privado, con permisos `0600`
en un directorio `0700`. Contiene el destino `principal_desarrollo`, la
referencia y huella de la copia de seguridad, el vector de `psql`, la
configuración del proveedor de identidad, los parámetros de fuentes, unidad y
gobierno, las rutas de los ejecutables y sus huellas, y un apartado `dba` para
cada fase. Las conexiones y aprobaciones externas también son archivos
privados. El esquema de campos corresponde a `plantilla-config.json` del kit
H10 cedido; este corte conserva ese contrato.

La configuración admite de 60 a 90 días para la vigencia de las personas,
la organización, la unidad y los perfiles de Administración. Este límite
evita preparar autoridad que caduque antes del recorrido o dure más de lo
admitido para la copia. `ca_dias` corresponde al certificado X.509 y puede
tener otra duración; las ventanas breves de preparación, aprobación y
revocación mantienen sus valores propios.

El conductor de la copia debe comprobar antes la revisión candidata limpia,
el SID de PostgreSQL y el candado del destino cuando proceda. Este guion
comprueba el destino declarado, el estado SQL de arranque, las copias de
fuente frente al manifiesto y las huellas de ejecutables y catálogos. No
comprueba por sí mismo la revisión Git, el SID ni el candado. El conductor
también prepara el kit ejecutable y el `config.json`; no se incluyen en Git
claves, certificados, conexiones ni material de personas.

El literal `HMAC_GO` de `arranque.py` forma parte del contrato del preparador
del kit H10: este lo extrae por AST para compilar el helper de identidad. Debe
conservarse íntegro aunque la ejecución de `--etapa` use el binario ya
preparado y cotejado por huella.

## Uso por el conductor

```sh
python3 deploy/principal/admin_copia/arranque.py \
  --config /ruta/privada/config.json --etapa comprobar
```

Después se invoca una etapa cada vez, en este orden:

1. `preparar_fuentes`, `configurar_fuentes`, `aplicar_fuentes`.
2. `preparar_unidad`, `configurar_unidad`, `aplicar_unidad`.
3. `preparar_bootstrap`, `configurar_bootstrap`, `aplicar_bootstrap`.
4. `verificar`.

Las etapas `configurar_*` exigen una aprobación externa privada ligada a la
huella del plan y a la referencia de la copia de seguridad. Calculan y
conservan la preimagen; un compromiso previo distinto detiene la ejecución.
Las etapas `aplicar_*` invocan las CLI existentes, que conservan la decisión,
el recibo y la auditoría del acto. El recibo de fuentes debe corresponder a
la organización y las dos personas del plan; el de unidad, a la unidad y su
fuente; el de bootstrap, a los perfiles previstos.

Si se pierde la respuesta de una aplicación, el conductor invoca
`recuperar_fuentes`, `recuperar_unidad` o `recuperar_bootstrap` con la misma
configuración y el mismo plan. La recuperación exige una respuesta marcada
como repetición y el mismo recibo, con un intento de auditoría nuevo. No se
debe volver a preparar la fase para recuperar una respuesta incierta.

## Límites de este corte

La comprobación con servicios y el reinicio vigilado corresponden al ensayo
integrado de la copia local. La etapa depende de que estén admitidas las
funciones y tablas de AUT39, Personal33 y bootstrap central, y de que el
preparador haya montado el conjunto 7 con sus ejecutables y catálogos. Una
falta de esas dependencias detiene solo esta función; no impide arrancar la
aplicación. El estado `verificar` comprueba las filas, perfiles efectivos y
recibos; deja pendiente el cotejo completo de cadena y copia fría.
