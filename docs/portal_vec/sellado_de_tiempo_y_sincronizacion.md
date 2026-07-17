# Sellado de tiempo y sincronización horaria para firmas

Documento de estudio. Fecha: 18 de julio de 2026. Autor: dirección técnica
del proyecto VEC. Estado: borrador para decisión del responsable y de
Sistemas. No es una configuración aprobada.

## 1. El problema, bien planteado

Al hablar de "la hora del servidor para las firmas" se mezclan dos relojes
que son jurídica y técnicamente distintos y no deben confundirse:

1. **La hora legal de la firma (sello de tiempo cualificado).** Es la fecha
   y hora que da fe del momento en que se firmó un acto. Para tener
   presunción de exactitud (Reglamento eIDAS 910/2014, arts. 41 y 42) debe
   proceder de una **Autoridad de Sellado de Tiempo (TSA) cualificada
   externa**, mediante un sello RFC 3161 enlazado a la huella del documento.
   **No es la hora del reloj del servidor.**

2. **La hora operativa del servidor.** El `time.Now()` de la máquina. Sirve
   para ordenar la auditoría encadenada, decidir ventanas de vigencia
   (que la confianza VEC-AD-2 no esté caducada ni revocada) y coherencia
   interna. Aquí manda la sincronización NTP, y es un **requisito del ENS**
   (sincronización de relojes para la trazabilidad), pero **no es el ancla
   legal de la firma**.

Principio rector, que el dominio ya respeta: **la hora del servidor nunca
sustituye al sello cualificado**, y si el sello no puede obtenerse, la firma
que lo exige no se completa (fallo cerrado).

## 2. Lo que la aplicación ya resuelve

El dominio de firma (`internal/modules/bolsa/domain/firma_decision_validacion.go`,
tipo `FirmaDecisionTecnica`) ya modela el sello de tiempo como evidencia
externa obligatoria cuando `RequiereSelloTiempo` es cierto, y falla cerrado
en su ausencia. Campos relevantes ya existentes:

- `SelloTiempoRef` + `HuellaSelloTiempoSHA256`: referencia opaca y huella del
  sello RFC 3161 recibido de la TSA.
- `PoliticaSelloTiempoRef` + `PoliticaSelloTiempoVersion` +
  `HuellaPoliticaSelloTiempoSHA256`: la política de sellado aplicada, con
  versión, de modo que cada sello cita bajo qué reglas se emitió.
- `ValidacionSelloTiempoRef` + `HuellaValidacionSelloTiempoSHA256` y la
  validación del documento sellado: se valida el sello, no se confía a
  ciegas.
- `VinculoRevisionSelladaRef`: el sello se ata a la revisión firmada, no a
  un documento suelto.
- `SelladaEn`, `ValidadoDocumentoSelladoEn`, `ValidadaEn`: ordenación
  temporal coherente, con comprobaciones de que no retroceden.
- `RequiereAumentoLongevidad`: contempla la **firma longeva** (validez a
  largo plazo, tipo PAdES-LTA), donde el sello se refuerza para seguir
  siendo verificable años después aunque caduquen certificados.

Lo que **falta** es el adaptador que habla con una TSA real: hoy solo hay
adaptadores de memoria y pruebas. El contrato está; el conector productivo
no.

## 3. Decisiones necesarias (responsable / Sistemas / Secretaría)

1. **Proveedor de sello cualificado.** Recomendación: **TS@**, el servicio
   de sellado de tiempo cualificado de la Administración General del Estado
   (misma familia que @firma, Cl@ve y Notific@), incluido en la Lista de
   Confianza europea y gratuito para las AAPP. Alternativa: TSA cualificada
   de la **FNMT-RCM**. La elección condiciona el alta y las credenciales,
   que gestiona Sistemas. `[pendiente de decisión]`
2. **Fuente de hora del servidor.** Recomendación: `chrony` sincronizado
   contra fuentes de estrato 1 españolas: el **Real Observatorio de la
   Armada (ROA)**, guardián legal de la hora oficial en España, y/o los
   servidores NTP de **RedIRIS** para AAPP. Con monitorización de deriva y
   alarma si supera el umbral. Es competencia de Sistemas; aquí se fija como
   requisito. `[pendiente de Sistemas]`
3. **Qué actos exigen sello cualificado.** No todo acto interno necesita
   TSA. Propuesta: exigen sello cualificado las resoluciones, las
   baremaciones firmadas, las listas definitivas, los llamamientos y las
   notificaciones fehacientes; los registros internos de auditoría se
   conforman con hora operativa sellada y encadenada. `[pendiente de la
   mesa]`

## 4. Requisitos de diseño para el adaptador (T19)

- Cliente RFC 3161 sobre el contrato ya existente: envía la huella del
  documento/revisión, recibe el token, lo valida (cadena de la TSA, política,
  huella coincidente) y rellena los campos del dominio; nunca inventa la
  fecha.
- **Política de sellado versionada**: cada sello cita `PoliticaSelloTiempoRef`
  y su versión; un cambio de política es una versión nueva, no una
  sustitución silenciosa.
- **Fallo cerrado**: si la TSA no responde o el token no valida, la firma que
  requiere sello no se completa; el acto queda pendiente, nunca firmado con
  hora local.
- **Proveedor inyectable** (como el resto de adaptadores): TS@ y FNMT tras la
  misma interfaz; en pruebas, una TSA simulada determinista. Sin credenciales
  ni endpoints reales en el código ni en Git.
- **Longevidad**: soportar el aumento a validez a largo plazo (resellado
  antes de que caduque la cadena) usando `RequiereAumentoLongevidad`.
- **Reloj operativo como puerto**: el dominio ya recibe el tiempo por
  inyección (patrón `Reloj`); el adaptador productivo lo alimenta desde el
  reloj del sistema sincronizado, y la monitorización de deriva vive fuera
  del núcleo.

## 5. Encaje normativo

- **eIDAS 910/2014** arts. 41-42: presunción de exactitud e integridad del
  sello cualificado.
- **ENS (RD 311/2022)**: sincronización de relojes como soporte de la
  trazabilidad (dimensión T del paquete de cumplimiento en
  `docs/cumplimiento/`); el sellado cualificado refuerza autenticidad (A).
- **Ley 39/2015 y 40/2015**: fecha y hora fehacientes de los actos y de su
  notificación.
- Relación con el resto del expediente electrónico:
  [firma, CSV, QR y cotejo](firma_csv_qr_y_cotejo.md),
  [flujo durable de firma de baremaciones](flujo_firma_baremacion_durable.md).

## 6. Resumen

El ancla legal del tiempo es el **sello cualificado externo (TS@/FNMT)**, no
el reloj del servidor. El reloj del servidor se sincroniza por NTP contra
ROA/RedIRIS por higiene y por el ENS, y sirve para orden interno y ventanas
de vigencia. El dominio ya exige el sello y falla cerrado sin él; queda
construir el adaptador TSA productivo (T19) y tomar las tres decisiones de la
sección 3.
