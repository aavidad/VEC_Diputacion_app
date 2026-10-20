package httpinterno

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La ausencia de fuente conserva el cuerpo de correo ya publicado.
type resolucionPortalJSON struct {
	FuenteRespuesta       string `json:"fuente_respuesta"`
	ClaveIdempotencia     string `json:"clave_idempotencia"`
	ExpedienteRef         string `json:"expediente_ref"`
	BolsaRef              string `json:"bolsa_ref"`
	LlamamientoRef        string `json:"llamamiento_ref"`
	ReciboEmisionRef      string `json:"recibo_emision_ref"`
	ParticipacionRef      string `json:"participacion_ref"`
	ReciboRespuestaRef    string `json:"recibo_respuesta_ref"`
	Respuesta             string `json:"respuesta"`
	RevisionRespuestaRRHH bool   `json:"revision_respuesta_rrhh"`
	RevisionPlazoRRHH     bool   `json:"revision_plazo_rrhh"`
	CriterioValidacionRef string `json:"criterio_validacion_ref"`
}

func (e resolucionPortalJSON) solicitud() (ports.SolicitudResolucionPortal, error) {
	s := ports.SolicitudResolucionPortal{
		ClaveIdempotencia: e.ClaveIdempotencia, ExpedienteRef: e.ExpedienteRef,
		BolsaRef: e.BolsaRef, LlamamientoRef: e.LlamamientoRef,
		ReciboEmisionRef: e.ReciboEmisionRef, ParticipacionRef: e.ParticipacionRef,
		ReciboRespuestaRef: e.ReciboRespuestaRef, Respuesta: ports.RespuestaLlamamiento(e.Respuesta),
		RevisionRespuestaRRHH: e.RevisionRespuestaRRHH, RevisionPlazoRRHH: e.RevisionPlazoRRHH,
		CriterioValidacionRef: e.CriterioValidacionRef,
	}
	if e.FuenteRespuesta != "portal" || s.ValidarEntrada() != nil {
		return ports.SolicitudResolucionPortal{}, errContenidoComunicacionLlamamientoNoValido
	}
	return s, nil
}

func leerVarianteResolucion(w http.ResponseWriter, r *http.Request) (
	ports.SolicitudResolverLlamamiento, ports.SolicitudResolucionPortal, bool, error,
) {
	var raw json.RawMessage
	if err := decodificarComunicacionLlamamiento(w, r, &raw); err != nil {
		return ports.SolicitudResolverLlamamiento{}, ports.SolicitudResolucionPortal{}, false, err
	}
	var fuente struct {
		FuenteRespuesta string `json:"fuente_respuesta"`
	}
	if json.Unmarshal(raw, &fuente) != nil {
		return ports.SolicitudResolverLlamamiento{}, ports.SolicitudResolucionPortal{}, false, errEntradaComunicacionLlamamientoInvalida
	}
	if fuente.FuenteRespuesta == "portal" {
		var entrada resolucionPortalJSON
		if err := decodificarVarianteResolucion(raw, &entrada); err != nil {
			return ports.SolicitudResolverLlamamiento{}, ports.SolicitudResolucionPortal{}, false, err
		}
		s, err := entrada.solicitud()
		return ports.SolicitudResolverLlamamiento{}, s, true, err
	}
	if fuente.FuenteRespuesta != "" {
		return ports.SolicitudResolverLlamamiento{}, ports.SolicitudResolucionPortal{}, false, errContenidoComunicacionLlamamientoNoValido
	}
	var entrada resolucionComunicacionLlamamientoJSON
	if err := decodificarVarianteResolucion(raw, &entrada); err != nil {
		return ports.SolicitudResolverLlamamiento{}, ports.SolicitudResolucionPortal{}, false, err
	}
	s, err := entrada.solicitud()
	return s, ports.SolicitudResolucionPortal{}, false, err
}

func decodificarVarianteResolucion(raw []byte, destino any) error {
	decodificador := json.NewDecoder(bytes.NewReader(raw))
	decodificador.DisallowUnknownFields()
	if decodificador.Decode(destino) != nil {
		return errEntradaComunicacionLlamamientoInvalida
	}
	canon, err := json.Marshal(destino)
	if err != nil || !bytes.Equal(raw, canon) {
		return errContenidoComunicacionLlamamientoNoValido
	}
	return nil
}

func proyectarResolucionPortal(s ports.SolicitudResolucionPortal, r ports.ResultadoResolucionPortal) (
	resolucionComunicacionLlamamientoSalidaJSON, int, bool,
) {
	// La organización solo sale del servidor; todos los demás campos deben
	// coincidir con la petición original antes de mostrar un recibo.
	original := s
	s.OrganizacionRef = r.Solicitud.OrganizacionRef
	if original.OrganizacionRef != "" || r.ValidarPara(s) != nil {
		return resolucionComunicacionLlamamientoSalidaJSON{}, 0, false
	}
	estado := http.StatusCreated
	if r.EsReplayConfirmado() {
		estado = http.StatusOK
	}
	return resolucionComunicacionLlamamientoSalidaJSON{
		Esquema:         EsquemaResolucionComunicacionLlamamiento,
		FuenteRespuesta: "portal", Respuesta: string(s.Respuesta),
		EstadoPlazo: r.EstadoPlazo, EstadoLocal: string(r.Estado),
		ResolucionRef: r.ResolucionRef, ReciboLocalRef: r.ReciboLocalRef,
		AuditoriaRef: r.AuditoriaRef, VersionResultante: r.VersionResultante,
		ResueltaEn: r.ResueltaEn.UTC().Format(time.RFC3339Nano),
	}, estado, true
}
