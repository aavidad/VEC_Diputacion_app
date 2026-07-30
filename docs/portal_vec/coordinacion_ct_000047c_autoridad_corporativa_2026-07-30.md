# Coordinación CT-000047C: autoridad corporativa de consultas RRHH

Fecha: 30 de julio de 2026.

Estado: **en curso y cerrado por defecto**. M2 está terminado, pero cuadro y
detalle aún no forman un recorrido productivo porque faltan la autoridad de
identidad de petición, la selección corporativa de perfil y ámbito, la
composición PDP y la raíz.

## Resultado perseguido

La consulta interna deberá recorrer una única cadena:

```text
canal mTLS real
→ aserción corporativa protegida y ligada al canal
→ sesión consumida y registrada
→ cápsula opaca de petición
→ selección corporativa única de perfil y ámbito
→ ContextoActor V2 registrado
→ PDP V3 y motivo M2 nominal
→ capacidad VEC-AD-3
→ consumo, lectura y acceso durable en una transacción PostgreSQL
→ respuesta HTTP minimizada
```

Ninguna referencia de autenticación, sesión, perfil, organización, rol o
ámbito puede proceder del cuerpo, URL, cabeceras, cookies, almacenamiento web,
configuración libre ni datos de demostración.

## Evidencia del hueco

- `httpseguridad.ServicioIdentidad` ya autentica el canal, consume la aserción,
  registra la sesión y proyecta una cuenta opaca.
- `CrearVinculoAutenticacionActorV2ConResultado` ya devuelve el vínculo y el
  mismo resultado ContextoActor registrado.
- Los guardianes de cuadro y detalle, M2 y
  `SesionConsultaRRHHPostgreSQL` ya existen.
- No existe una implementación productiva de
  `ports.AutoridadContextoConsultaRRHH`.
- La identidad entrega cuenta, pero no selecciona perfil ni organización.
- El resolutor ContextoActor actual exige un perfil ya elegido.
- El PDP exige principal y perfil; no debe descubrirlos ni sumar permisos de
  varios perfiles.
- `NuevoServidor` sigue devolviendo deliberadamente todas las dependencias
  como ausentes y no escucha.

## Grafo de minitareas

```text
C1 cápsula opaca de petición
  └→ C2 selección corporativa perfil-ámbito
       └→ C3 autoridad ContextoConsultaRRHH
            ├→ P1 composición PDP y emisores nominales
            └→ R1 propiedad de pools y proveedores
                  └→ R2 casos de uso y siete rutas atómicas
                       └→ R3 raíz interna
                            └→ W1 fuente web definitiva
                                 └→ E1 E2E completo
```

### C1 — cápsula opaca ligada al canal

Responsabilidad:

- proyectar cuenta y auditoría únicamente desde `ServicioIdentidad`, una
  `IdentidadSesion` válida y el mismo canal mTLS;
- conservar una huella del vínculo de canal, no la aserción;
- impedir construcción, serialización o logging accidental;
- transportarla mediante una clave privada de `context.Context`;
- conservar cancelación y denegar cápsula cero, cruzada, alterada o ausente.

C1 no define el transporte de la aserción, no lee HTTP y no selecciona perfil,
organización ni permisos.

### C2 — selección corporativa única

La autoridad corporativa debe fijar, antes del PDP:

```text
cuenta + superficie interna + uso consulta_rrhh
→ exactamente un perfil activo
→ exactamente un ámbito organizativo activo
```

La selección será gobernada, versionada, con procedencia acreditada, vigencia
`[desde,hasta)` y denegación indistinguible para cero o varias coincidencias.
No se admite `LIMIT 1`, prioridad implícita, perfil habitual ni tantear
perfiles contra el PDP hasta encontrar una concesión.

El diseño SQL exacto se cerrará antes de reservar migraciones. Debe reutilizar
las autoridades de ContextoActor y PDP sin convertir las asignaciones de
permisos en una fuente de identidad.

### C3 — autoridad de contexto RRHH

Responsabilidad:

- extraer C1;
- obtener C2;
- revalidar autenticación y registrar ContextoActor V2;
- conservar el vínculo y el mismo resultado registrado;
- construir `ContextoConsultaRRHH` sin aceptar perfil u organización del
  llamador.

El contrato actual que recibe `organizacionRef` como cadena no se considerará
procedencia suficiente para producción.

### P1 — PDP y criptografía

Compondrá el almacén PostgreSQL de autorización, servicio V3, motivo M2,
atestación, confianza y dos emisores físicamente distintos para cuadro y
detalle.

La capacidad AD-3 no será productiva mientras la decisión de Sistemas/DBA no
permita verificar dentro de PostgreSQL una clave no exportable o una capacidad
asimétrica equivalente. HMAC local permanece limitado a pruebas.

### R1/R2/R3 — raíz

- cada pool nominal tiene un único propietario;
- el cierre ocurre una vez y en orden inverso;
- las dos consultas se añaden junto a las cinco rutas existentes de forma
  atómica;
- el listener es el último recurso en construirse;
- cualquier dependencia ausente conserva cero listeners y cero API parcial.

### W1/E1 — interfaz y prueba final

La fuente web interna consumirá la misma API que escritorio, CLI y MCP, sin
fallback de presentación. El cierre exige PostgreSQL 18, TLS 1.3/mTLS,
identidad corporativa viva, revocación, reinicio, concurrencia, cero cookies,
capturas revisadas y tres ejecuciones limpias.

## Material externo pendiente

Sistemas, DBA, RRHH y DPD deben aprobar o proporcionar:

- protocolo y transporte real de la aserción Kerberos/certificado hasta Go;
- verificador, emisor, audiencia, enlace de canal, revocación y relojes;
- maestros y procedencia de cuenta, persona, perfil y ámbito organizativo;
- matriz PDP publicada;
- identidades técnicas, certificados, DSN y gestor de secretos;
- KMS/HSM, firma COSE, conjuntos de confianza y decisión MAC/asimétrica;
- conformidades ENS, EIPD y aceptación funcional.

La ausencia de estos materiales no se sustituye con cabeceras internas,
cookies, memoria, configuración de desarrollo ni datos sintéticos.

## Puertas comunes

Cada minitarea tendrá productor y revisor distintos, archivos de hasta 800
líneas, pruebas focales y de carrera, `go vet`, formato, `git diff --check`,
Gitleaks y commit pequeño en castellano. SQL exige PostgreSQL 18.4 real,
roles mínimos, estados hostiles, reinicio, concurrencia y tres ejecuciones
limpias.

La producción permanece en **NO-GO** hasta completar E1 y obtener las
conformidades organizativas. Los porcentajes oficiales no aumentan por
contratos, adaptadores o infraestructura aislados.
