# Revisión independiente O2-09A

Fecha: 23 de julio de 2026.

## Dictamen

**NO-GO** para integrar el candidato exacto
`1323b4b382b28de21d2d8036346f13657e755242`.

La interfaz es visualmente apta y su arquitectura es reutilizable, pero acepta
dos entradas que el adaptador HTTP O2-08B rechaza. La divergencia deja al
usuario completar la revisión y enviar una solicitud que el servidor considera
inválida. No se integrará hasta alinear ambos límites y añadir sus regresiones.

## Hallazgos

### O2-09-R1 — periodo superior a cien años aceptado por la interfaz

Severidad: media.

`validarBorradorAlta` comprueba el formato y que el fin no sea anterior al
inicio, pero no aplica el máximo de cien años del adaptador O2-08B. Por ejemplo,
la interfaz acepta `2026-08-01` a `2126-08-02`; el servidor lo rechaza mediante
`MaximoAniosPeriodoAlta`.

Corrección exigida:

- aceptar exactamente cien años;
- rechazar cualquier día posterior;
- mantener fechas civiles canónicas;
- probar ambos bordes en el contrato web.

### O2-09-R2 — importe RC superior al máximo del contrato HTTP

Severidad: media.

`centimosDesdeEntrada` limita el importe a un entero seguro de JavaScript, que
es más amplio que el máximo contractual `922337203685477` céntimos de O2-08B.
La entrada `90000000000000.00` se transforma en `9000000000000000` céntimos y
la interfaz la acepta, aunque el servidor la rechaza.

Corrección exigida:

- usar en la interfaz el mismo máximo exacto de céntimos;
- aceptar el máximo;
- rechazar el máximo más un céntimo y cualquier valor superior;
- conservar aritmética entera exacta, sin redondeos de coma flotante.

## Evidencia favorable

- 20 pruebas originales y 30 repeticiones adicionales: 50/50 verdes.
- Sintaxis de todos los módulos JavaScript: válida.
- No usa cookies, almacenamiento del navegador, autoridad aportada por el
  cliente ni adaptador de datos ficticio.
- El comando conserva la UUIDv4 de intención en memoria y coincide
  estructuralmente con O2-08B.
- Teclado, foco, resumen de revisión, errores y recibo presentan estados
  comprensibles y semánticos.
- Revisión visual real a 1440 px de las pantallas de edición, revisión y recibo:
  composición profesional, tema de Diputación coherente y sin defectos
  bloqueantes. Las capturas fueron evidencia desechable y no se incorporaron al
  repositorio.
- Archivos dentro del límite de tamaño, árbol limpio y `git diff --check`
  correcto.

## Alcance del NO-GO

No se solicita rehacer la pantalla ni añadir un adaptador provisional. La
corrección queda limitada a los dos bordes contractuales, sus mensajes i18n,
sus pruebas y la documentación de integración.

Aunque se corrija, O2-09 seguirá sin cerrarse hasta conectar la misma vista con
la ruta real compuesta por O2-07/O2-08 y superar O2-10.
