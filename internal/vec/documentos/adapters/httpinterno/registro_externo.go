package httpinterno

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// RutaRegistroExterno anota en un expediente la referencia y la huella de un
// original que custodia otro sistema (por ejemplo, el registro de entrada).
// VEC no recibe el fichero: solo declara que RRHH lo ha comprobado con esa
// referencia y huella. No acredita firma, registro ni entrega.
const RutaRegistroExterno = "/api/vec/documentos/externos/registros"

const dominioIdentificadorRegistroExterno = "vec.documentos.registro-externo.id.v1"

// ServicioRegistroExterno es el caso de uso de Documentos que resuelve la
// política de conservación y pide después la concesión V3 ligada.
type ServicioRegistroExterno interface {
	RegistrarExternoAutorizado(context.Context, ports.AltaExterna, ports.AutorizadorRegistroExterno) (domain.Documento, error)
}

// PoliticasRegistroExterno construye la solicitud exacta de conservación de
// un tipo catalogado para un expediente. Un tipo sin política no se registra.
type PoliticasRegistroExterno interface {
	SolicitudPara(tipo, expedienteRef string) (vecports.SolicitudPoliticaConservacionDocumental, error)
}

// RegistroExternoAdmitido fija por configuración qué tipos documentales puede
// anotar esta ruta, para qué módulo y con qué custodio. El navegador solo
// elige el tipo; módulo y custodio nunca proceden del cuerpo.
type RegistroExternoAdmitido struct {
	PrefijoTipo string
	ModuloID    string
	CustodioID  string
}

// ConfiguracionRegistroExterno compone la ruta. Sin admitidos no se publica.
type ConfiguracionRegistroExterno struct {
	Servicio    ServicioRegistroExterno
	Autoridad   ports.AutorizadorRegistroExterno
	Politicas   PoliticasRegistroExterno
	Admitidos   []RegistroExternoAdmitido
	Incidencias vecports.EmisorIncidenciasTecnicas
}

type manejadorRegistroExterno struct {
	c ConfiguracionRegistroExterno
}

// NuevaRutaRegistroExterno valida la configuración: prefijos técnicos no
// solapados, módulo y custodio con forma de identificador.
func NuevaRutaRegistroExterno(c ConfiguracionRegistroExterno) (httpapi.RutaExacta, error) {
	if nula(c.Servicio) || nula(c.Autoridad) || nula(c.Politicas) || len(c.Admitidos) == 0 || len(c.Admitidos) > 32 {
		return httpapi.RutaExacta{}, ErrManejadorInvalido
	}
	if nula(c.Incidencias) {
		c.Incidencias = nil
	}
	admitidos := make([]RegistroExternoAdmitido, 0, len(c.Admitidos))
	for _, a := range c.Admitidos {
		if !strings.HasSuffix(a.PrefijoTipo, ".") || !domain.IdentificadorTecnicoValido(strings.TrimSuffix(a.PrefijoTipo, ".")) ||
			!domain.IdentificadorTecnicoValido(a.ModuloID) || !domain.IdentificadorTecnicoValido(a.CustodioID) {
			return httpapi.RutaExacta{}, ErrManejadorInvalido
		}
		for _, previo := range admitidos {
			if strings.HasPrefix(a.PrefijoTipo, previo.PrefijoTipo) || strings.HasPrefix(previo.PrefijoTipo, a.PrefijoTipo) {
				return httpapi.RutaExacta{}, ErrManejadorInvalido
			}
		}
		admitidos = append(admitidos, a)
	}
	c.Admitidos = admitidos
	return httpapi.RutaExacta{Ruta: RutaRegistroExterno, Manejador: &manejadorRegistroExterno{c: c}}, nil
}

func (h *manejadorRegistroExterno) admitido(tipo string) (RegistroExternoAdmitido, bool) {
	if !domain.IdentificadorTecnicoValido(tipo) {
		return RegistroExternoAdmitido{}, false
	}
	for _, a := range h.c.Admitidos {
		if strings.HasPrefix(tipo, a.PrefijoTipo) && len(tipo) > len(a.PrefijoTipo) {
			return a, true
		}
	}
	return RegistroExternoAdmitido{}, false
}

func (h *manejadorRegistroExterno) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabeceras(w)
	if h != nil && h.c.Incidencias != nil {
		w = &escritorVigilado{ResponseWriter: w, incidencias: h.c.Incidencias}
	}
	if h == nil || nula(h.c.Servicio) || nula(h.c.Autoridad) || nula(h.c.Politicas) || r == nil || r.URL == nil {
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if !rutaExacta(r, RutaRegistroExterno) {
		responderError(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.Context().Err() != nil {
		responderError(w, http.StatusRequestTimeout, "peticion_cancelada")
		return
	}
	if !tipoContenido(r) {
		responderError(w, http.StatusUnsupportedMediaType, "tipo_no_admitido")
		return
	}
	var entrada struct {
		ClaveIdempotencia string `json:"clave_idempotencia"`
		ExpedienteRef     string `json:"expediente_ref"`
		Tipo              string `json:"tipo"`
		Referencia        string `json:"referencia"`
		HuellaSHA256      string `json:"huella_sha256"`
	}
	if err := decodificar(w, r, &entrada); err != nil || !domain.ReferenciaOpacaValida(entrada.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(entrada.ExpedienteRef) || !domain.ReferenciaCustodioValida(entrada.Referencia) ||
		len(entrada.Referencia) > 128 || !domain.HuellaValida(entrada.HuellaSHA256) {
		responderError(w, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	admitido, ok := h.admitido(entrada.Tipo)
	if !ok {
		responderError(w, http.StatusUnprocessableEntity, "tipo_no_admitido_registro")
		return
	}
	solicitud, err := h.c.Politicas.SolicitudPara(entrada.Tipo, entrada.ExpedienteRef)
	if err != nil {
		responderError(w, http.StatusUnprocessableEntity, "tipo_no_admitido_registro")
		return
	}
	custodia := domain.ReferenciaCustodiaExterna{CustodioID: admitido.CustodioID, Referencia: entrada.Referencia, HuellaSHA256: entrada.HuellaSHA256}
	alta := ports.AltaExterna{
		ID:                identificadorRegistroExterno(entrada.ExpedienteRef, entrada.ClaveIdempotencia),
		ClaveIdempotencia: entrada.ClaveIdempotencia, ModuloID: admitido.ModuloID,
		ExpedienteRef: entrada.ExpedienteRef, TipoRef: solicitud.TipoDocumentalRef(), Version: 1,
		Custodia: custodia, SolicitudPolitica: solicitud,
	}
	documento, err := h.c.Servicio.RegistrarExternoAutorizado(r.Context(), alta, h.c.Autoridad)
	if err != nil {
		responderServicio(w, err)
		return
	}
	if documento.Validar() != nil || documento.ID != alta.ID || documento.Custodia != domain.CustodiaExterna ||
		documento.ExpedienteRef != alta.ExpedienteRef || documento.CustodiaExternaRef != custodia {
		responderError(w, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	// Como en la lista, la respuesta no devuelve la referencia ni el custodio:
	// solo el número VEC y la huella que identifican la anotación.
	responderJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"estado": "registrado", "documento": struct {
		Ref          string `json:"ref"`
		Numero       string `json:"numero_vec"`
		Tipo         string `json:"tipo"`
		Version      uint64 `json:"version"`
		Huella       string `json:"huella"`
		Custodia     string `json:"custodia"`
		RegistradoEn string `json:"registrado_en"`
	}{documento.ID, documento.NumeroVEC, entrada.Tipo, documento.Version, documento.HuellaSHA256, documento.Custodia,
		documento.CreadoEn.UTC().Format(time.RFC3339Nano)}}})
}

// identificadorRegistroExterno deriva el identificador del documento de la
// clave de idempotencia: un reintento con la misma clave nombra el mismo
// documento y la fachada SQL lo reconoce en lugar de duplicarlo.
func identificadorRegistroExterno(expedienteRef, clave string) string {
	suma := sha256.Sum256([]byte(dominioIdentificadorRegistroExterno + "\x00" + expedienteRef + "\x00" + clave))
	return "ref:" + hex.EncodeToString(suma[:])
}
