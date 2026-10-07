# Preparar el plan privado de ADMIN

Este comando prepara o coteja un plan finito de bootstrap. El formato 2
conserva dos cuentas administrativas nominativas distintas y
sus referencias, certificados admitidos, roles, ámbitos, vigencia, fuentes
y huellas. La asignación de Sistemas, cuando figure en el plan, conserva
un perfil separado; no recibe autoridad para administrar usuarios.

La fuente y el plan deben estar fuera de Git, en directorios propios `0700`
y archivos propios `0600`, sin enlaces. Se rechazan campos desconocidos,
duplicados, incompletos o permisos de archivo abiertos. La salida contiene
solo la huella del plan. Repetir el mismo contenido lo coteja; un contenido
distinto no sobrescribe el archivo anterior.

```sh
go run ./cmd/vec-preparar-admin-bootstrap \
  -fuente /ruta/privada/fuente-admin.json \
  -plan /ruta/privada/plan-admin.json

go run ./cmd/vec-preparar-admin-bootstrap \
  -fuente /ruta/privada/fuente-admin.json \
  -plan /ruta/privada/plan-admin.json -cotejar
```

`-aplicar` exige cotejo, conexión y aprobación privadas y el destino del
recibo. En este corte valida esos datos y termina con
`provision_no_confirmada`: falta componer el proveedor durable. No abre una
conexión, no asigna perfiles ni genera un recibo de éxito. La preparación o
la aprobación local no acreditan instalación, autorización ni publicación.

La conexión futura debe usar socket local o TLS con verificación del
servidor también en todos sus destinos alternativos. El proveedor deberá
cotejar las fuentes y la huella aprobada y unir CAS, asignaciones, historia,
auditoría y recibo en una transacción. El proceso ADMIN no recibe este canal
privado de operador.

El formato 3 conserva las dos personas de Aplicación y una asignación separada
de Sistemas. Cada asignación declara organización y, cuando la fuente la admite,
unidad, con sus referencias, versiones y huellas. El catálogo aportado conserva
el identificador, la versión y la categoría de cada rol, sus límites de ámbito
y la fuente de esa categoría. El comando no publica esas definiciones.

La evidencia `fuente_reparto_aprobado` identifica el reparto ya aceptado. No
sustituye la aprobación externa por la huella del plan concreto. El mapa privado
con huellas de rol sin completar todavía no es un material válido de PlanV3.
No se rellenan huellas, cargos, organizaciones o fuentes por defecto.

```sh
go run ./cmd/vec-preparar-admin-bootstrap \
  -fuente /ruta/privada/material-v3.json \
  -plan /ruta/privada/plan-v3.json \
  -textos web/static/textos/es/admin-comprobar-perfil.json

go run ./cmd/vec-preparar-admin-bootstrap \
  -fuente /ruta/privada/material-v3.json \
  -plan /ruta/privada/plan-v3.json -cotejar \
  -aprobacion /ruta/privada/aprobacion-v3.json \
  -textos web/static/textos/es/admin-comprobar-perfil.json
```

El campo `version: 3` de la fuente selecciona ese formato. `-version-plan 3`
permite exigirlo expresamente y rechaza una versión distinta. `-textos` solo
selecciona el catálogo de idioma existente; no cambia el formato ni concede
autoridad. Para inglés, use su catálogo en `textos/en`.

El plan se guarda como `{plan, huella_plan_sha256}`. La aprobación privada tiene
la forma `{huella_plan_sha256}` y solo se coteja con `-cotejar`. La salida estándar
contiene la SHA256 calculada sobre los bytes canónicos del plan. La salida de
diagnóstico contiene código, mensaje del catálogo, límite y estado de preparación
o cotejo; no contiene nombres, certificados ni el contenido del plan.

`-aplicar` permanece cerrado para V3: no abre conexión, no crea asignaciones y
no escribe plan ni recibo. Preparar y cotejar archivos no demuestra que las
fuentes estén publicadas o que sus ámbitos sean válidos ante la autoridad real.
El proveedor deberá comprobarlos de nuevo antes de aplicar el plan aprobado.
