package httppersonal

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/limiteshttp"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

const RutaJustificantesMiBolsa = "/api/vec/bolsa/mi-bolsa/justificantes"
const limiteJustificanteRenuncia = 16 << 20

var errJustificanteDemasiadoGrande = errors.New("justificante_demasiado_grande")

var selectorJustificanteRenuncia = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,255}$`)

type IdentidadJustificanteRenuncia struct {
	Actor          vec.ResultadoContextoActorRegistradoV2
	Vinculo        vec.VinculoAutenticacionActorV2
	CorrelacionRef string
}

type ProveedorJustificanteRenuncia interface {
	PrepararJustificanteRenuncia(*http.Request) (IdentidadJustificanteRenuncia, error)
}

// Cada petición recibe un resolutor inmutable para su bolsa seleccionada.
// La factoría no toma identidad ni titularidad del formulario.
type FabricaJustificanteRenuncia interface {
	ServicioJustificanteRenuncia(context.Context, IdentidadJustificanteRenuncia, string) (docports.SubidorLocal, error)
}

type HandlerJustificanteRenuncia struct {
	proveedor ProveedorJustificanteRenuncia
	fabrica   FabricaJustificanteRenuncia
}

func NuevoHandlerJustificanteRenunciaNoDisponible() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		falloJustificanteRenuncia(w, docports.ErrSubidaLocalNoDisponible)
	})
}

func NuevoHandlerJustificanteRenuncia(p ProveedorJustificanteRenuncia, f FabricaJustificanteRenuncia) (*HandlerJustificanteRenuncia, error) {
	if nula(p) || nula(f) {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	return &HandlerJustificanteRenuncia{p, f}, nil
}

func (h *HandlerJustificanteRenuncia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r == nil || r.URL == nil || r.URL.Path != RutaJustificantesMiBolsa || r.URL.EscapedPath() != RutaJustificantesMiBolsa || r.URL.ForceQuery {
		responder(w, http.StatusNotFound, errorRespuesta{"recurso_no_encontrado"})
		return
	}
	if h == nil || nula(h.proveedor) || nula(h.fabrica) {
		falloJustificanteRenuncia(w, docports.ErrSubidaLocalNoDisponible)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		responder(w, http.StatusMethodNotAllowed, errorRespuesta{"metodo_no_admitido"})
		return
	}
	if r.Method == http.MethodPost && !limiteshttp.MismoOrigenEstricto(r) {
		responder(w, http.StatusForbidden, errorRespuesta{"acceso_denegado"})
		return
	}
	identidad, err := h.proveedor.PrepararJustificanteRenuncia(r)
	if err != nil {
		if errors.Is(err, ErrAutenticacionAusente) {
			responderErrorPortal(w, err)
		} else {
			falloJustificanteRenuncia(w, err)
		}
		return
	}
	if identidad.Actor.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Actor) != nil {
		falloJustificanteRenuncia(w, docports.ErrSubidaLocalDenegada)
		return
	}
	var bolsa, clave string
	var contenido []byte
	if r.Method == http.MethodGet {
		q, errorQuery := url.ParseQuery(r.URL.RawQuery)
		if errorQuery != nil || len(q) != 2 || len(q["bolsa"]) != 1 || len(q["clave_idempotencia"]) != 1 || r.ContentLength > 0 {
			responder(w, http.StatusBadRequest, errorRespuesta{"datos_no_validos"})
			return
		}
		bolsa, clave = q.Get("bolsa"), q.Get("clave_idempotencia")
	} else {
		if r.URL.RawQuery != "" {
			responder(w, http.StatusBadRequest, errorRespuesta{"datos_no_validos"})
			return
		}
		bolsa, clave, contenido, err = leerJustificanteRenuncia(w, r)
		if err != nil {
			var maximo *http.MaxBytesError
			if errors.Is(err, errJustificanteDemasiadoGrande) || errors.As(err, &maximo) {
				responder(w, http.StatusRequestEntityTooLarge, errorRespuesta{"peticion_demasiado_grande"})
			} else {
				responder(w, http.StatusBadRequest, errorRespuesta{"datos_no_validos"})
			}
			return
		}
		defer clear(contenido)
	}
	if !selectorJustificanteRenuncia.MatchString(bolsa) || !selectorJustificanteRenuncia.MatchString(clave) {
		responder(w, http.StatusBadRequest, errorRespuesta{"datos_no_validos"})
		return
	}
	servicio, err := h.fabrica.ServicioJustificanteRenuncia(r.Context(), identidad, bolsa)
	if err != nil || nula(servicio) {
		if err == nil {
			err = docports.ErrSubidaLocalNoDisponible
		}
		falloJustificanteRenuncia(w, err)
		return
	}
	var resultado docports.ResultadoSubidaLocal
	if r.Method == http.MethodGet {
		resultado, err = servicio.Consultar(r.Context(), docports.SolicitudConsultaLocal{
			ActorRegistradoV2: identidad.Actor, VinculoV2: identidad.Vinculo,
			ClaveIdempotencia: clave, CorrelacionRef: identidad.CorrelacionRef,
		})
	} else {
		resultado, err = servicio.Subir(r.Context(), docports.SolicitudSubidaLocal{
			ActorRegistradoV2: identidad.Actor, VinculoV2: identidad.Vinculo, ClaveIdempotencia: clave,
		}, contenido)
	}
	if err != nil && !errors.Is(err, docports.ErrSubidaLocalPendiente) && !errors.Is(err, docports.ErrSubidaLocalRechazada) {
		falloJustificanteRenuncia(w, err)
		return
	}
	if !resultado.Estado.Valido() || resultado.CargaID == "" {
		falloJustificanteRenuncia(w, docports.ErrSubidaLocalNoDisponible)
		return
	}
	estado := http.StatusAccepted
	if resultado.Estado == docports.EstadoSubidaLocalAdmitida {
		if resultado.ComprobanteRef == "" || resultado.ReciboRef == "" || resultado.DocumentoVersion < 1 || resultado.RegistradoEn.IsZero() {
			falloJustificanteRenuncia(w, docports.ErrSubidaLocalNoDisponible)
			return
		}
		estado = http.StatusOK
		if r.Method == http.MethodPost {
			estado = http.StatusCreated
		}
	} else if resultado.Estado == docports.EstadoSubidaLocalRechazada {
		estado = http.StatusUnprocessableEntity
	}
	salida := estadoJustificanteRenuncia{CargaID: resultado.CargaID, Estado: resultado.Estado,
		ReciboRef: resultado.ReciboRef, ComprobanteRef: resultado.ComprobanteRef,
		DocumentoVersion: resultado.DocumentoVersion, CodigoRechazo: resultado.CodigoRechazo}
	if !resultado.RegistradoEn.IsZero() {
		fecha := resultado.RegistradoEn
		salida.RegistradoEn = &fecha
	}
	responder(w, estado, struct {
		Data estadoJustificanteRenuncia `json:"data"`
	}{salida})
}

type estadoJustificanteRenuncia struct {
	CargaID          string                     `json:"carga_id"`
	Estado           docports.EstadoSubidaLocal `json:"estado"`
	ReciboRef        string                     `json:"recibo_ref,omitempty"`
	ComprobanteRef   string                     `json:"comprobante_ref,omitempty"`
	DocumentoVersion int                        `json:"documento_version,omitempty"`
	CodigoRechazo    string                     `json:"codigo_rechazo,omitempty"`
	RegistradoEn     *time.Time                 `json:"registrado_en,omitempty"`
}

func falloJustificanteRenuncia(w http.ResponseWriter, err error) {
	estado, codigo := http.StatusServiceUnavailable, "servicio_no_disponible"
	if errors.Is(err, docports.ErrSubidaLocalDenegada) || errors.Is(err, docports.ErrAccesoDenegado) || errors.Is(err, bolsaports.ErrJustificanteRenunciaDenegado) || errors.Is(err, vec.ErrAutorizacionDenegada) {
		estado, codigo = http.StatusForbidden, "acceso_denegado"
	} else if errors.Is(err, docports.ErrSubidaLocalNoEncontrada) || errors.Is(err, docports.ErrNoEncontrado) {
		estado, codigo = http.StatusNotFound, "recurso_no_encontrado"
	}
	if estado == http.StatusServiceUnavailable {
		causa := "dependencia_no_disponible"
		if errors.Is(err, context.Canceled) {
			causa = "peticion_cancelada"
		}
		if errors.Is(err, context.DeadlineExceeded) {
			causa = "plazo_agotado"
		}
		slog.Warn("justificante_renuncia_bolsa_503", "causa", causa)
	}
	responder(w, estado, errorRespuesta{codigo})
}

func leerJustificanteRenuncia(w http.ResponseWriter, r *http.Request) (bolsa, clave string, contenido []byte, err error) {
	tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "multipart/form-data" {
		return "", "", nil, docports.ErrSubidaLocalDenegada
	}
	r.Body = http.MaxBytesReader(w, r.Body, limiteJustificanteRenuncia+(64<<10))
	lector, err := r.MultipartReader()
	if err != nil {
		return "", "", nil, err
	}
	vistos := map[string]bool{}
	defer func() {
		if err != nil {
			clear(contenido)
		}
	}()
	for {
		parte, fallo := lector.NextRawPart()
		if errors.Is(fallo, io.EOF) {
			break
		}
		if fallo != nil {
			return "", "", contenido, fallo
		}
		nombre := parte.FormName()
		if vistos[nombre] || (nombre != "bolsa" && nombre != "clave_idempotencia" && nombre != "documento") || len(parte.Header.Values("Content-Disposition")) != 1 || parte.Header.Get("Content-Transfer-Encoding") != "" {
			return "", "", contenido, docports.ErrSubidaLocalDenegada
		}
		vistos[nombre] = true
		limite := int64(256)
		if nombre == "documento" {
			limite = limiteJustificanteRenuncia
		}
		valor, fallo := io.ReadAll(io.LimitReader(parte, limite+1))
		cerrar := parte.Close()
		if nombre == "documento" && int64(len(valor)) > limite {
			clear(valor)
			return "", "", contenido, errJustificanteDemasiadoGrande
		}
		if fallo != nil || cerrar != nil || int64(len(valor)) > limite || len(valor) == 0 {
			clear(valor)
			return "", "", contenido, errors.Join(docports.ErrSubidaLocalDenegada, fallo, cerrar)
		}
		if nombre == "documento" {
			if parte.FileName() == "" || http.DetectContentType(valor[:min(len(valor), 512)]) != "application/pdf" {
				clear(valor)
				return "", "", contenido, docports.ErrSubidaLocalDenegada
			}
			contenido = valor
		} else {
			if parte.FileName() != "" || strings.TrimSpace(string(valor)) != string(valor) {
				return "", "", contenido, docports.ErrSubidaLocalDenegada
			}
			if nombre == "bolsa" {
				bolsa = string(valor)
			} else {
				clave = string(valor)
			}
		}
	}
	if len(vistos) != 3 {
		return "", "", contenido, docports.ErrSubidaLocalDenegada
	}
	return bolsa, clave, contenido, nil
}
