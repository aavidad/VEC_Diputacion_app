package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func serializarOrden(o ports.OrdenOperacion) ([]byte, error) {
	c := domain.ComandoHecho{Esquema: domain.EsquemaComandoHecho, Accion: o.Accion, ActorRef: o.ActorRef, ClaveIdempotencia: o.ClaveIdempotencia, VersionEsperada: o.VersionEsperada, Hecho: o.Hecho, Motivo: o.Motivo, FechaCorte: o.FechaCorte}
	b, err := c.RepresentacionCanonica()
	if err != nil || o.Accion == "meritos.hecho.verificar" {
		return nil, ports.ErrRegistroNoDisponible
	}
	suma := sha256.Sum256(b)
	if hex.EncodeToString(suma[:]) != o.HuellaComando {
		return nil, ports.ErrRegistroNoDisponible
	}
	a := o.Autorizacion
	d, err := a.Solicitud.Datos()
	if err != nil {
		return nil, ports.ErrRegistroNoDisponible
	}
	vinculo, err := d.VinculoAutenticacionActor.Datos()
	finalidad, audiencia := application.EspecificacionAutorizacion(o.Accion)
	if err != nil || vinculo.PrincipalID != o.ActorRef || a.Contexto.Contexto.PersonaRef != o.ActorRef || d.Accion != o.Accion || d.Finalidad != finalidad || d.ReferenciaMotivo != o.Motivo ||
		d.Recurso.Referencia != o.Hecho.Referencia || d.Recurso.ModuloID != "meritos" || d.Recurso.Tipo != "hecho" || len(d.Recurso.Ambitos) != 3 || len(d.Recurso.Atributos) != 0 ||
		d.Recurso.Ambitos["persona_ref"] != o.Hecho.PersonaRef || d.Recurso.Ambitos["version_esperada"] != strconv.Itoa(o.VersionEsperada) || d.Recurso.Ambitos["huella_comando_sha256"] != o.HuellaComando ||
		a.Decision.ExigirProyeccionPara(a.Solicitud, []string{"hecho", "declarante_ref", "version", "recibo"}, []string{"auditar"}) != nil ||
		!vecports.MaterialAtestadoLigadoV3(a.Solicitud, a.Decision, a.Confirmacion, a.Contexto, o.Motivo, a.Material, audiencia) {
		return nil, ports.ErrRegistroNoDisponible
	}
	return b, nil
}

func leerResultado(raw []byte) (ports.ResultadoOperacion, error) {
	if len(raw) == 0 || len(raw) > 65536 {
		return ports.ResultadoOperacion{}, ports.ErrRegistroNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	var out ports.ResultadoOperacion
	if d.Decode(&out) != nil {
		return ports.ResultadoOperacion{}, ports.ErrRegistroNoDisponible
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return ports.ResultadoOperacion{}, ports.ErrRegistroNoDisponible
	}
	return out, nil
}
