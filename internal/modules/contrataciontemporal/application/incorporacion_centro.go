package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var ErrServicioIncorporacionCentroInvalido = errors.New("contratacion temporal: servicio de confirmacion del centro invalido")

// ServicioIncorporacionCentro coordina la bandeja y la confirmación de la
// incorporación por el centro. El tipo de documento lo decide el catálogo
// para la modalidad del expediente; nunca el navegador.
type ServicioIncorporacionCentro struct {
	repositorio ports.RepositorioIncorporacionCentro
	reglas      ports.FuenteReglaAcreditacionIncorporacion
	reloj       ports.Reloj
}

func NuevoServicioIncorporacionCentro(repositorio ports.RepositorioIncorporacionCentro, reglas ports.FuenteReglaAcreditacionIncorporacion, reloj ports.Reloj) (*ServicioIncorporacionCentro, error) {
	if dependenciaNula(repositorio) || dependenciaNula(reglas) || dependenciaNula(reloj) {
		return nil, ErrServicioIncorporacionCentroInvalido
	}
	return &ServicioIncorporacionCentro{repositorio: repositorio, reglas: reglas, reloj: reloj}, nil
}

// ExpedienteBandejaCentro añade a la fila el documento que pide el catálogo;
// vacío si la regla no está disponible para esa modalidad.
type ExpedienteBandejaCentro struct {
	ports.ExpedienteIncorporacionCentro
	DocumentoExigido string `json:"documento_exigido"`
}

// Bandeja lista los expedientes de las peticiones del actor.
func (s *ServicioIncorporacionCentro) Bandeja(ctx context.Context, organizacionRef string, actor domain.ActorPeticionCentro) ([]ExpedienteBandejaCentro, error) {
	if s == nil || ctx == nil {
		return nil, ErrServicioIncorporacionCentroInvalido
	}
	filas, err := s.repositorio.ListarIncorporacionesCentro(ctx, ports.ConsultaIncorporacionesCentro{Modo: "bandeja", OrganizacionRef: organizacionRef, Actor: actor})
	if err != nil {
		return nil, err
	}
	salida := make([]ExpedienteBandejaCentro, 0, len(filas))
	for _, f := range filas {
		fila := ExpedienteBandejaCentro{ExpedienteIncorporacionCentro: f}
		if f.AdmiteConfirmacion() && f.ModalidadClave != "" {
			if tipo, _, err := s.reglas.DocumentoAcreditativo(ctx, domain.ClaveCatalogo(f.ModalidadClave)); err == nil {
				fila.DocumentoExigido = string(tipo)
			}
		}
		salida = append(salida, fila)
	}
	return salida, nil
}

// SolicitudConfirmarIncorporacionCentro llega del navegador del centro; el
// actor y la organización los fija la frontera autenticada.
type SolicitudConfirmarIncorporacionCentro struct {
	OrganizacionRef    string
	Actor              domain.ActorPeticionCentro
	ClaveIdempotencia  string
	PeticionRef        string
	ExpedienteRef      string
	FechaIncorporacion string
	DocumentoRef       string
	DocumentoSHA256    string
}

// Confirmar localiza el expediente en la bandeja autorizada del actor,
// resuelve el documento exigido para su modalidad y registra. SQL vuelve a
// comprobar petición, centro, fase, modalidad y unicidad; una repetición con
// la misma clave devuelve el recibo original.
func (s *ServicioIncorporacionCentro) Confirmar(ctx context.Context, sol SolicitudConfirmarIncorporacionCentro) (ports.ReciboIncorporacionCentro, error) {
	var vacio ports.ReciboIncorporacionCentro
	if s == nil || ctx == nil {
		return vacio, ErrServicioIncorporacionCentroInvalido
	}
	// La incorporación no puede ser futura (día civil peninsular); SQL lo repite.
	hoy := s.reloj.Ahora().In(zonaMadridIncorporacionCentro()).Format(time.DateOnly)
	if f, err := time.Parse(time.DateOnly, sol.FechaIncorporacion); err != nil || f.Format(time.DateOnly) != sol.FechaIncorporacion ||
		sol.FechaIncorporacion > hoy {
		return vacio, ports.ErrIncorporacionCentroInvalida
	}
	// Antes de consumir la lectura autorizada se rechaza lo que ya no vale.
	previa := ports.MaterialIncorporacionCentro{Operacion: ports.OperacionConfirmarIncorporacionCentro, ClaveIdempotencia: sol.ClaveIdempotencia,
		OrganizacionRef: sol.OrganizacionRef, Actor: sol.Actor, PeticionRef: sol.PeticionRef, ExpedienteRef: sol.ExpedienteRef,
		FechaIncorporacion: sol.FechaIncorporacion, Documento: ports.DocumentoIncorporacionCentro{Tipo: "pendiente", Referencia: sol.DocumentoRef, SHA256: sol.DocumentoSHA256},
		Regla: ports.ReglaDocumentoIncorporacion{Referencia: "pendiente:regla", HuellaSHA256: strings.Repeat("a", 64), ModalidadClave: "pendiente"}}
	if previa.Validar() != nil {
		return vacio, ports.ErrIncorporacionCentroInvalida
	}
	filas, err := s.repositorio.ListarIncorporacionesCentro(ctx, ports.ConsultaIncorporacionesCentro{Modo: "bandeja", OrganizacionRef: sol.OrganizacionRef, Actor: sol.Actor})
	if err != nil {
		return vacio, err
	}
	var fila *ports.ExpedienteIncorporacionCentro
	for i := range filas {
		if filas[i].ExpedienteRef == sol.ExpedienteRef && filas[i].PeticionRef == sol.PeticionRef {
			fila = &filas[i]
		}
	}
	// Una confirmación previa se deja a SQL: si es la misma intención
	// devuelve su recibo; si no, la rechaza.
	if fila == nil || fila.ModalidadClave == "" || fila.Fase != string(domain.FaseNombramiento) || fila.Estado != string(domain.EstadoEnCurso) {
		return vacio, ports.ErrIncorporacionCentroNoAdmitida
	}
	tipo, regla, err := s.reglas.DocumentoAcreditativo(ctx, domain.ClaveCatalogo(fila.ModalidadClave))
	if err != nil || !regla.Valida() || regla.ModalidadClave != fila.ModalidadClave {
		return vacio, ports.ErrReglaAcreditacionNoDisponible
	}
	m := ports.MaterialIncorporacionCentro{Operacion: ports.OperacionConfirmarIncorporacionCentro, ClaveIdempotencia: sol.ClaveIdempotencia,
		OrganizacionRef: sol.OrganizacionRef, Actor: sol.Actor, PeticionRef: sol.PeticionRef, ExpedienteRef: sol.ExpedienteRef,
		FechaIncorporacion: sol.FechaIncorporacion, Documento: ports.DocumentoIncorporacionCentro{Tipo: string(tipo), Referencia: sol.DocumentoRef, SHA256: sol.DocumentoSHA256},
		Regla: regla}
	if m.Validar() != nil {
		return vacio, ports.ErrIncorporacionCentroInvalida
	}
	return s.repositorio.ConfirmarIncorporacionCentro(ctx, m)
}

func zonaMadridIncorporacionCentro() *time.Location {
	if z, err := time.LoadLocation("Europe/Madrid"); err == nil {
		return z
	}
	return time.UTC
}
