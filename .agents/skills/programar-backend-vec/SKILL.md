---
name: programar-backend-vec
description: Implementa incrementos del backend Go de VEC reutilizando dominio, aplicación, puertos y adaptadores. Usar para expedientes, seguimiento, API o composición de contratación temporal.
---

# Programar backend VEC

Partir del recorrido que necesita RRHH y buscar primero su implementación con rg.
Inspeccionar internal/modules/contrataciontemporal/{domain,application,ports,adapters}
y la composición de internal/app. Distinguir un servicio existente de una ruta montada.

1. Identificar contrato, propietario de datos, punto de montaje y archivos asignados.
2. Mantener decisiones en dominio/aplicación y detalles HTTP/SQL en adaptadores.
3. Reutilizar servicios existentes. Extender una pieza con un parche acotado antes de
   introducir otra variante equivalente. Conectar las APIs que consume la interfaz.
4. Preservar referencias, versiones esperadas, recibos y errores nominales. En una
   recuperación reutilizar la operación original; no crear otra para aparentar éxito.
5. Tratar la anotación administrativa y el cambio de estado como operaciones distintas.
   Una consulta del seguimiento original no prueba cuál es su estado actual.
6. Ejecutar las pruebas focales del comportamiento modificado y entregar contrato,
   archivos y punto exacto de composición al responsable de integración.

Personal conserva relaciones/incorporaciones; Bolsa conserva selección/llamamientos.
No escribir sus agregados desde Contratación temporal. Coordinar con el propietario
SQL o de identidad antes de modificar persistencia o autorización. Aplicar
$orquestar-vec para encargar trabajo independiente sin invadir sus archivos.
