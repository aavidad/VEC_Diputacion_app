# Decisión de puentes de autoridad para consultas RRHH CT-000047B

Fecha: 29 de julio de 2026.

## Decisión

Se implementarán guardianes propietarios de Contratación temporal y una única
fachada VEC de alto nivel inyectada. No se creará ahora una nueva capacidad
opaca común.

La decisión conserva una sola autoridad:

1. Contratación temporal fija la semántica cerrada de cuadro o detalle;
2. el PDP V3 existente adopta la única decisión;
3. la fachada VEC existente encadena atestación, verificación, capacidad y
   material usando el mismo resultado registrado;
4. el guardián coteja el resultado y construye el material nominal de
   consulta;
5. PostgreSQL vuelve a verificar y consume la capacidad dentro de la
   transacción durable de lectura y registro de acceso.

Ningún navegador, DTO o manejador HTTP aporta organización, mapas, acción,
finalidad, audiencia, dominio, huella, decisión o material de confianza.

## Motivo

El código ya dispone de dos nominales propietarios:

- `ContextoConsultaRRHH`, que retiene el vínculo y resultado exactos;
- `MaterialAutorizacionConsultaRRHH`, que conserva el material de consumo.

Una tercera representación común no tiene todavía un segundo consumidor real.
Obligaría a mantener dos portadores de autoridad o a refactorizar registro,
asignación y análisis fuera del camino crítico. También introduciría una
orquestación común prematura entre PDP, KMS, confianza y material.

La alternativa descartada necesitaría al menos siete u ocho minitareas y entre
catorce y dieciocho archivos. Solo se reconsiderará cuando una segunda
vertical real necesite exactamente la misma cadena y pueda probarse un
contrato común sin semántica de Contratación temporal.

## Grafo de minitareas

```text
B1 retención nominal privada [cerrada]
├── D0 envoltorio opaco de recurso [cerrada]
│   ├── D1 fábrica cerrada de cuadro [cerrada]
│   └── D2 fábrica cerrada de detalle [en curso]
└── A2.1 raíz pública nominal [cerrada]
    └── A2.2 fachada emisora VEC de alto nivel [en curso]

D1 + D2 + A2
→ A4 guardián y puerto emisor
→ A5 cierre de la vía cruda y migración de fixtures
→ A6 autorizador de consultas RRHH
→ composición, rutas, TLS y E2E
```

Cada nodo se implementa y revisa por separado. D1 y D2 tienen write-sets
disjuntos. A2 no importa Contratación temporal y el puerto se implementará
estructuralmente.

## Recursos cerrados

Cuadro y detalle derivarán en servidor:

- organización desde el contexto opaco;
- módulo y tipo desde constantes técnicas;
- referencia desde organización o expediente, según la operación;
- ámbitos exactos, sin claves adicionales;
- dominio y huella desde la solicitud tipada;
- acción, finalidad y audiencia desde constantes del servidor.

No habrá constructor genérico que reciba cadenas o mapas ni getter de mapas
mutables. Las fábricas iniciales se limitarán al ámbito organización. Centro o
unidad requerirán otra decisión gobernada, no una ampliación implícita.

## Decisión pendiente separada

El motivo publicado y el alcance efectivo organización/centro/unidad no
proceden del HTTP y aún necesitan una resolución gobernada. No se fijarán en
código por conveniencia ni se permitirán como entrada libre al emisor. Esta
decisión debe cerrarse antes del autorizador final A6.

La frontera de identidad corporativa también sigue condicionada a que Sistemas
proporcione el verificador de la aserción protegida y los materiales reales
ligados al canal mTLS. No se sustituirá por cabeceras, cookies ni datos DEMO.
