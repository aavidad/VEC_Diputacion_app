# Preparar fuentes iniciales de ADMIN

La CLI valida una fuente sintética, guarda un plan canónico privado y calcula su
SHA256. También coteja un plan existente con la misma fuente. No conecta con una
base, no genera material HMAC, no crea cuentas y no concede perfiles.

Las reglas comunes están en `domain.PlanFuentesInicialesAdminV1`: versión 1,
entorno `desarrollo`, alcance `sintetico_declarado`, organización y personas
nuevas con versión esperada 0 y evidencias de versión 1 con referencias `prc_`. Las dos personas deben
estar ordenadas por `persona_ref`. Sus cuatro operaciones de cuenta son distintas;
las referencias de las cuentas definitivas las genera Identidad al aplicar.
Las operaciones de cuenta usan `opr_`, conforme a `provisionar_cuenta_v1` de IS2;
`prc_` corresponde a la evidencia de procedencia, no a la operación.

Preparación y caducidad usan UTC con segundos (`AAAA-MM-DDTHH:MM:SSZ`). La CLI
comprueba su reloj local: `preparado_en <= ahora < caduca_en`. Las vigencias cubren
la caducidad del plan y la de cada persona no supera la de su organización. La
edad máxima de revocación debe estar entre 1 y 2147483647 segundos, conforme al
contrato SQL. Los archivos tienen un límite de 64 KiB.

```sh
GOCACHE=$HOME/.cache/go-build go build -p 8 -o /ruta/privada/vec-preparar-fuentes-admin ./cmd/vec-preparar-fuentes-admin
/ruta/privada/vec-preparar-fuentes-admin \
  --fuente /ruta/privada/fuente.json \
  --plan /ruta/privada/plan.json \
  --textos /ruta/app/web/static/textos/es/admin-fuentes-preparar.json
/ruta/privada/vec-preparar-fuentes-admin \
  --fuente /ruta/privada/fuente.json \
  --plan /ruta/privada/plan.json \
  --textos /ruta/app/web/static/textos/es/admin-fuentes-preparar.json \
  --cotejar
```

Use rutas absolutas a fuente y plan, archivos con modo 0600 y un directorio
propio con modo 0700 fuera del repositorio. El catálogo es público; existe en
castellano e inglés. El ejemplo `testdata/fuente.sintetica.json` es una fixture
con fechas fijas para las pruebas: antes de preparar otra fuente ajuste sus
fechas y referencias y aporte las huellas de sus fuentes declaradas.

El documento contiene `plan` y `huella_plan_sha256`. La huella cubre el JSON
compacto del plan, en el orden de campos del DTO, sin salto de línea final. El
documento guardado sí termina en un salto de línea. No se ordenan ni sustituyen
referencias recibidas y no se usa la representación `jsonb::text` para esta huella.

La salida JSON contiene estado, diagnóstico traducido y huella; no muestra la
fuente ni sus referencias. Los errores no incluyen rutas ni contenido. Códigos
de proceso: 0 preparado o cotejado; 1 entrada, vigencia o cotejo rechazados;
2 catálogo ausente o inválido, o fallo de escritura del diagnóstico. La CLI no
sobrescribe un plan divergente y rechaza claves JSON repetidas, campos ausentes,
campos adicionales, fechas no canónicas y cardinalidad distinta de dos personas.

La aplicación posterior pertenece a AUT39, con LOGIN, configuración privada,
aprobación externa por huella, preimagen y recibo. CA33/IS15 conservan las fuentes
y la titularidad cuenta–Persona; AD174 registra la auditoría común. Este plan no
acredita esos efectos y no sirve para producción ni para el arranque de perfiles
2+1 de AUT38.

Comprobaciones focales del corte: formato, pruebas de ambos paquetes con `-race`
y `-p 8`, `go vet`, coherencia de catálogos ES/EN y Semgrep local sin métricas.
Gosec revisa únicamente ambos paquetes: no hay hallazgos en los archivos nuevos.
Sus 25 avisos corresponden a archivos previos del paquete de dominio que este
corte no cambia (conversiones G115 y la constante de autoridad G101). Deben
mantenerse visibles en la PR; no se modifican autoridades ajenas para ocultarlos.
