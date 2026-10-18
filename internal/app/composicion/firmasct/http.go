package firmasct

import (
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/auditoriafirma"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La raíz entrega autoridades ya compuestas para la misma persona de cada
// petición. Los envoltorios autentican por la frontera común y sellan el
// contexto nominal antes de llegar a los adaptadores HTTP existentes.
type DependenciasHTTP struct {
	Servicios          Servicios
	AutoridadOriginal  httpinterno.AutoridadCanalRegistroFirmaVec
	Original           httpinterno.ServicioOriginalFirmableCTHTTP
	AutoridadPreflight httpinterno.AutoridadContextoCanalCircuitoRRHH
	Preflight          httpinterno.ConsultorPreflightFirmaR5V2
	AutoridadVec       httpinterno.AutoridadCanalRegistroFirmaVec
	FuenteConsulta     consultafirmasv2.FuenteContexto
	Consulta           ports.AutorizadorConsultaFirmasR5V2
	RegistroConsulta   ports.RegistroFirmasVerificadasV2
	Intentos           vecports.RegistradorIntentosAuditoria
	OrdenIntento       auditoriafirma.FabricaOrden
	EnvolverLectura    func(string, http.Handler) (http.Handler, error)
	EnvolverVec        func(http.Handler) (http.Handler, error)
}

// ComponerHTTP devuelve las cuatro rutas como un conjunto indivisible. No
// deja un registro alcanzable si falta original, preflight, consulta auditada
// o la frontera nominal. No crea identidad ni sustituye fuentes por dobles.
func ComponerHTTP(d DependenciasHTTP) ([]vechttp.RutaExacta, error) {
	if d.Servicios.Vec == nil || d.EnvolverLectura == nil || d.EnvolverVec == nil {
		return nil, ErrComposicionNoDisponible
	}
	original, err := httpinterno.NuevoManejadorOriginalFirmableCT(d.AutoridadOriginal, d.Original)
	if err != nil {
		return nil, err
	}
	preflight, err := httpinterno.NuevoManejadorPreflightFirmaR5V2(d.AutoridadPreflight, d.Preflight)
	if err != nil {
		return nil, err
	}
	consulta, err := httpinterno.NuevoManejadorConsultaFirmasR5V2(d.FuenteConsulta, d.Consulta,
		d.RegistroConsulta, d.Intentos, d.OrdenIntento)
	if err != nil {
		return nil, err
	}
	vec, err := httpinterno.NuevoManejadorRegistroFirmaVec(d.AutoridadVec, d.Servicios.Vec)
	if err != nil {
		return nil, err
	}
	rutas := []vechttp.RutaExacta{
		{Ruta: httpinterno.RutaOriginalFirmableCT, Manejador: original},
		{Ruta: httpinterno.RutaPreflightFirmaR5, Manejador: preflight},
		{Ruta: httpinterno.RutaConsultaFirmasR5V2, Manejador: consulta},
	}
	for i := range rutas {
		rutas[i].Manejador, err = d.EnvolverLectura(rutas[i].Ruta, rutas[i].Manejador)
		if err != nil || rutas[i].Manejador == nil {
			return nil, ErrComposicionNoDisponible
		}
	}
	vec, err = d.EnvolverVec(vec)
	if err != nil || vec == nil {
		return nil, ErrComposicionNoDisponible
	}
	return append(rutas, vechttp.RutaExacta{Ruta: httpinterno.RutaRegistroFirmaVec, Manejador: vec}), nil
}
