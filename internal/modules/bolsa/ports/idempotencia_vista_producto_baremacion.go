package ports

import (
	"crypto/subtle"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

type vistaEfimeraTestimonioIdempotenciaBaremacion struct {
	mu             sync.Mutex
	testimonio     testimonioAtomicoIdempotenciaBaremacion
	visitasEnVuelo int
	cobertura      uint16
	cerrada        bool
}

const (
	coberturaMaterialAtestadoBaremacion uint16 = 1 << iota
	coberturaResolucionIdentidadBaremacion
	coberturaResumenIdentidadesBaremacion
	coberturaTopologiaIdentidadesBaremacion
	coberturaResumenIndicesBaremacion
	coberturaTopologiaIndicesBaremacion
	coberturaPrincipalesBaremacion
	coberturaMatrizBaremacion
	coberturaEvidenciaBaremacion
	coberturaRepresentacionesIntencionBaremacion
	coberturaObligatoriaVistaBaremacion = coberturaMaterialAtestadoBaremacion |
		coberturaResolucionIdentidadBaremacion |
		coberturaResumenIdentidadesBaremacion |
		coberturaTopologiaIdentidadesBaremacion |
		coberturaResumenIndicesBaremacion |
		coberturaTopologiaIndicesBaremacion |
		coberturaPrincipalesBaremacion |
		coberturaMatrizBaremacion |
		coberturaEvidenciaBaremacion
	coberturaPermitidaVistaBaremacion = coberturaObligatoriaVistaBaremacion |
		coberturaRepresentacionesIntencionBaremacion
)

func nuevaVistaEfimeraTestimonioIdempotenciaBaremacion(
	testimonio testimonioAtomicoIdempotenciaBaremacion,
) *vistaEfimeraTestimonioIdempotenciaBaremacion {
	return &vistaEfimeraTestimonioIdempotenciaBaremacion{testimonio: testimonio}
}

func (*vistaEfimeraTestimonioIdempotenciaBaremacion) String() string {
	return "[VISTA-EFIMERA-TESTIMONIO-IDEMPOTENCIA-BAREMACION-PROTEGIDA]"
}
func (*vistaEfimeraTestimonioIdempotenciaBaremacion) GoString() string {
	return "ports.vistaEfimeraTestimonioIdempotenciaBaremacion{[PROTEGIDA]}"
}
func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, v.String())
}
func (*vistaEfimeraTestimonioIdempotenciaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*vistaEfimeraTestimonioIdempotenciaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*vistaEfimeraTestimonioIdempotenciaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) LogValue() slog.Value {
	return slog.StringValue(v.String())
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) conTestimonio(
	visita func(testimonioAtomicoIdempotenciaBaremacion) error,
) error {
	v.mu.Lock()
	if v.cerrada || v.testimonio.validarEstructura() != nil {
		v.mu.Unlock()
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	v.visitasEnVuelo++
	clon, err := v.testimonio.clonar()
	v.mu.Unlock()
	if err != nil {
		v.mu.Lock()
		v.visitasEnVuelo--
		v.mu.Unlock()
		return err
	}
	defer func() {
		destruirTestimonioAtomicoIdempotenciaBaremacion(&clon)
		v.mu.Lock()
		v.visitasEnVuelo--
		v.mu.Unlock()
	}()
	return visita(clon)
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) marcarCobertura(marca uint16) {
	v.mu.Lock()
	if !v.cerrada {
		v.cobertura |= marca
	}
	v.mu.Unlock()
}

// cerrarYComprobarSinActividad nunca espera: marca la vista cerrada, destruye
// su copia original y devuelve false si el adaptador dejo un callback activo.
// Cada callback activo posee otra copia y la destruye en su defer al terminar.
func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) cerrarYComprobarSinActividad() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.cerrada {
		return false
	}
	v.cerrada = true
	destruirTestimonioAtomicoIdempotenciaBaremacion(&v.testimonio)
	return v.visitasEnVuelo == 0 && coberturaVistaBaremacionValida(v.cobertura)
}

func coberturaVistaBaremacionValida(cobertura uint16) bool {
	return cobertura&coberturaObligatoriaVistaBaremacion == coberturaObligatoriaVistaBaremacion &&
		cobertura&^coberturaPermitidaVistaBaremacion == 0
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarMaterialCanonicoAtestadoBaremacion(
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		material, err := t.representacionCanonicaAtestada()
		if err != nil {
			return err
		}
		return visitarCargaProtegidaEfimeraBaremacion(material, visita)
	})
	if err == nil {
		v.marcarCobertura(coberturaMaterialAtestadoBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarResolucionIdentidadInternaEstableBaremacion(
	visita func(string, uint64, string, string, string, string, []byte) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		i := t.resolucionIdentidad
		atestacion := append([]byte(nil), i.ValorAtestacion...)
		defer borrarBytesBaremacion(atestacion)
		return visita(
			i.SnapshotRef, i.Revision, i.HuellaSHA256, i.FormatoAtestacion,
			i.EmisorAtestacionRef, i.ClaveAtestacionRef, atestacion,
		)
	})
	if err == nil {
		v.marcarCobertura(coberturaResolucionIdentidadBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) ResumenLlaveroIdentidadesBaremacion() (
	referencia string, revision uint64, cantidad uint8, huella string, err error,
) {
	err = v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		referencia, revision, cantidad, huella = t.identidades.LlaveroRef,
			t.identidades.Revision, t.identidades.Cantidad, t.identidades.HuellaSHA256
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaResumenIdentidadesBaremacion)
	}
	return
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarTopologiaIdentidadesBaremacion(
	visita func(int, uint16, uint32, string) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		for posicion, entrada := range t.identidades.Topologia {
			if err := visita(posicion, entrada.Version, entrada.GeneracionClave, entrada.ClaveHMACRef); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaTopologiaIdentidadesBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) ResumenLlaveroIndicesBaremacion() (
	referencia string, revision uint64, cantidad uint8, huella string, err error,
) {
	err = v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		referencia, revision, cantidad, huella = t.indices.LlaveroRef,
			t.indices.Revision, t.indices.Cantidad, t.indices.HuellaSHA256
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaResumenIndicesBaremacion)
	}
	return
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarTopologiaIndicesBaremacion(
	visita func(int, uint16, uint32, string) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		for posicion, entrada := range t.indices.Topologia {
			if err := visita(posicion, entrada.Version, entrada.GeneracionClave, entrada.ClaveHMACRef); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaTopologiaIndicesBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarPrincipalesBaremacion(
	visita func(int, uint16, uint32, string, string) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		for posicion, principal := range t.principales {
			if err := visita(posicion, principal.Version, principal.GeneracionClave,
				principal.ClaveHMACRef, principal.ValorHMAC); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaPrincipalesBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarMatrizIndicesBaremacion(
	visita func(int, int, uint16, uint32, string, string) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		for fila, indices := range t.matriz {
			for columna, indice := range indices {
				if err := visita(fila, columna, indice.Version, indice.GeneracionClave,
					indice.ClaveHMACRef, indice.ValorHMAC); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaMatrizBaremacion)
	}
	return err
}

// VisitarRepresentacionesCanonicasIntencionBaremacion es el puente nominal
// para que el futuro servicio privado pueda sellar, por cada celda de la
// matriz completa, el fingerprint semantico estable y conservar el sobre
// probatorio exacto. No devuelve indices, celdas ni capacidades seleccionables:
// las dos cargas solo viven dentro del callback y se borran al terminar.
//
// motivoEfimeroYaVerificado debe haber sido cotejado antes contra MotivoHMAC
// con la clave historica por la composicion privada. Este puerto no acredita
// esa verificacion, no ejecuta CAS ni concede efectos; mientras no exista
// internal/modules/bolsa/application/servicio_idempotencia_baremacion.go el
// flujo permanece expresamente NO-GO.
func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarRepresentacionesCanonicasIntencionBaremacion(
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	intencion IntencionCambioBaremacion,
	motivoEfimeroYaVerificado []byte,
	visita func(
		filaPrincipal, columnaIndice int,
		fingerprintSemantico, sobreProbatorio MaterialCanonicoEfimeroBaremacion,
	) error,
) error {
	if visita == nil || !intencionCambioBaremacionVinculadaSolicitud(intencion, solicitud) ||
		len(motivoEfimeroYaVerificado) == 0 || len(motivoEfimeroYaVerificado) > 8000 {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	bytesMotivo := append([]byte(nil), motivoEfimeroYaVerificado...)
	motivoCopia, err := NuevaCargaProtegida(bytesMotivo)
	borrarBytesBaremacion(bytesMotivo)
	if err != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	defer destruirCargaProtegidaBaremacion(&motivoCopia)

	err = v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		if subtle.ConstantTimeCompare(t.vinculoSolicitud[:], solicitud.vinculo[:]) != 1 {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
		for fila, indices := range t.matriz {
			for columna, indice := range indices {
				fingerprint, err := intencion.representacionCanonicaFingerprintSemanticoParaHMAC(
					indice, motivoCopia,
				)
				if err != nil {
					return ErrClaveIdempotenciaBaremacionInvalida
				}
				sobre, err := intencion.representacionCanonicaSobreProbatorioParaHMAC(indice)
				if err != nil {
					destruirCargaProtegidaBaremacion(&fingerprint)
					return ErrClaveIdempotenciaBaremacionInvalida
				}
				errVisita := visitarCargaProtegidaEfimeraBaremacion(
					fingerprint,
					func(fingerprintEfimero MaterialCanonicoEfimeroBaremacion) error {
						return visitarCargaProtegidaEfimeraBaremacion(
							sobre,
							func(sobreEfimero MaterialCanonicoEfimeroBaremacion) error {
								return visita(fila, columna, fingerprintEfimero, sobreEfimero)
							},
						)
					},
				)
				if errVisita != nil {
					return errVisita
				}
			}
		}
		return nil
	})
	if err == nil {
		// Las representaciones derivadas no acreditan que el adaptador haya
		// cotejado tambien los indices originales de la matriz.
		v.marcarCobertura(coberturaRepresentacionesIntencionBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarEvidenciaAtestacionBaremacion(
	visita func(string, string, string, uint64, string, []byte) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		e := t.evidencia
		copia := append([]byte(nil), e.Valor...)
		defer borrarBytesBaremacion(copia)
		return visita(e.Formato, e.EmisorRef, e.ClaveAtestacionRef, e.Revision,
			e.HuellaContenidoSHA256, copia)
	})
	if err == nil {
		v.marcarCobertura(coberturaEvidenciaBaremacion)
	}
	return err
}
