package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	reglasbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/reglas"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// componerContactoOrigenBolsaDesarrollo engancha la regla b29 (duda 45) al
// registro de datos de contacto y a la emisión de llamamientos ya compuestos.
// Sin catálogo, o sin Bolsa compuesta, todo conserva su conducta de siempre:
// solo contactos propios y ningún aviso.
func componerContactoOrigenBolsaDesarrollo(resolutor *reglas.Resolutor, mutador http.Handler) error {
	politica := reglasbolsa.NuevoContactoOrigen(resolutor)
	participacion, ok := mutador.(*manejadorParticipacionBolsaDesarrollo)
	if !politica.Configurada() || !ok || participacion == nil || participacion.datos == nil {
		return nil
	}
	// Una regla b29 ausente o incompleta impide arrancar, igual que un
	// paquete de reglas ilegible.
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	if _, err := politica.MarcaOrigenConvoca(ctx, time.Now().UTC()); err != nil {
		return errors.Join(errReglasEjemploNoValidas, err)
	}
	if err := participacion.datos.EstablecerPoliticaOrigenDatosContacto(politica); err != nil {
		return err
	}
	if participacion.emision == nil || participacion.fuente == nil {
		return nil
	}
	return participacion.emision.EstablecerAvisoContactoNoConfirmado(participacion.fuente)
}

// OrigenContactoParticipacion lee la marca de la versión vigente; sin contacto
// registrado no hay marca (el correo ya consta como no enviado).
func (f *fuenteCorreoParticipacionB7) OrigenContactoParticipacion(ctx context.Context, participacion string) (*dominiobolsa.MarcaOrigenDatosContacto, error) {
	if f == nil || f.repositorio == nil {
		return nil, puertosbolsa.ErrDatosContactoParticipacionNoDisponibles
	}
	r, err := f.repositorio.DatosContactoVigentes(ctx, participacion)
	if errors.Is(err, puertosbolsa.ErrDatosContactoParticipacionNoEncontrados) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.Origen, nil
}
