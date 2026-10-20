# Configuración de firma de la copia

`admisiones.preparar(c)` consulta en solo lectura las cuatro asignaciones vigentes del perfil de firma. Comprueba persona, cuenta, rol AUT53, control habilitado, certificado nominal y recibo durable. Conserva las filas y sus huellas bajo `recorrido/admisiones/`. Los roles AUT53 son la semilla técnica autorizada por la excepción de Dirección del 20/10 a las 14:52; esa semilla no acredita gobierno técnico por dos administradores.

`configuracion_firma.preparar(c)` exige el plan publicado con su recibo, las cuatro admisiones cotejadas y los certificados de la copia. Copia la lista de revocación propia a `material/identidad/firma-recursos/clientes.crl` y guarda `material/identidad/firma-vec.json`. Cada firmante conserva cuatro perfiles físicos separados: firma, consulta de Contratación temporal, Documentos y consulta del circuito. El fichero declara `autoridad: no_autoritativo`; el arranque y las comprobaciones de capacidad siguen pendientes de Dirección.

La configuración de Documentos requiere una entrada `base` propia para `recorrido_copia/documentos.py:preparar_documentos`: cuentas, motivos de listado, registro externo y custodia, con las conexiones exclusivas que indica ese productor. Los datos de firmantes no aportan ese contrato. Hasta disponer de él no se crea `identidad/documentos.json` ni se anuncia acceso documental efectivo.

Prueba focal: `PYTHONPATH=deploy/principal python3 -m unittest deploy/principal/test_recorrido_copia_configuracion.py -v`. Usa cuatro casos con lecturas SQL simuladas y sin efectos de red.
