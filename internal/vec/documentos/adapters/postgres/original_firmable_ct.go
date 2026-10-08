package postgres

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
)

func validarAutorizacionOriginalFirmable(a ports.AutorizacionV3, accion string, preimagen []byte) (materialV3, error) {
	if docapp.ValidarAutorizacionOriginalFirmable(a, accion, time.Now().UTC()) != nil || len(preimagen) == 0 {
		return materialV3{}, ports.ErrSolicitudInvalida
	}
	persona, errPersona := strconv.ParseInt(strconv.FormatUint(a.Material.PersonaVersion(), 10), 10, 64)
	perfil, errPerfil := strconv.ParseInt(strconv.FormatUint(a.Material.PerfilVersion(), 10), 10, 64)
	if errPersona != nil || errPerfil != nil {
		return materialV3{}, ports.ErrSolicitudInvalida
	}
	resumen := a.Material.ResumenCapacidad()
	if resumen.Operacion() != accion || resumen.EfectoHuellaSHA256() != ports.HuellaEfectoV3(preimagen) {
		return materialV3{}, ports.ErrSolicitudInvalida
	}
	return materialV3{
		capacidad: a.Material.CapacidadCanonica(), decision: a.Material.DecisionCanonica(),
		motivo: a.Material.MotivoCanonico(), contexto: a.Material.ContextoActorCanonico(),
		persona: persona, perfil: perfil,
		payload: a.Material.PayloadVECAD3(), sobre: a.Material.SobreCOSESign1(),
		evidencia: a.Material.EvidenciaVerificacion(), raiz: a.Material.RaizPublicaSPKI(),
	}, nil
}

// ReservarOriginalFirmable usa exclusivamente la fachada Documentos-13. Cada
// intento pendiente recibe una clave de almacén nueva; el repositorio no
// accede a tablas de otro módulo ni crea referencias por su cuenta.
func (r *Repositorio) ReservarOriginalFirmable(ctx context.Context, solicitud ports.ReservaOriginalFirmable) (ports.IntentoOriginalFirmable, error) {
	preimagen, err := docapp.PreimagenReservaOriginalFirmable(solicitud)
	if err != nil || solicitud.Autorizacion.RecursoRef != solicitud.ID ||
		solicitud.Autorizacion.AmbitoRef != solicitud.ExpedienteRef {
		return ports.IntentoOriginalFirmable{}, ports.ErrSolicitudInvalida
	}
	material, err := validarAutorizacionOriginalFirmable(solicitud.Autorizacion, ports.AccionReservarOriginalFirmable, preimagen)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	auth, err := autorizacionJSON(solicitud.Autorizacion)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, ports.ErrSolicitudInvalida
	}
	raw, err := r.transaccion(ctx,
		"SELECT vec_documentos.reservar_original_firmable_v1($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		preimagen, string(auth), material.capacidad, material.decision, material.motivo, material.contexto,
		material.persona, material.perfil, material.payload, material.sobre, material.evidencia, material.raiz)
	if err != nil {
		return ports.IntentoOriginalFirmable{}, err
	}
	var respuesta struct {
		ReservaRef      string `json:"reserva_ref"`
		Estado          string `json:"estado"`
		ID              string `json:"id"`
		HuellaSHA256    string `json:"huella_sha256"`
		IntentoNum      uint64 `json:"intento_num"`
		ClaveAlmacenRef string `json:"clave_almacen_ref"`
	}
	if json.Unmarshal(raw, &respuesta) != nil {
		return ports.IntentoOriginalFirmable{}, ErrRepositorioNoDisponible
	}
	intento := ports.IntentoOriginalFirmable{
		ReservaRef: respuesta.ReservaRef, Estado: respuesta.Estado,
		DocumentoID: respuesta.ID, HuellaSHA256: respuesta.HuellaSHA256,
		Numero: respuesta.IntentoNum, ClaveAlmacenRef: respuesta.ClaveAlmacenRef,
	}
	if docapp.ValidarIntentoOriginalFirmable(intento, solicitud) != nil {
		return ports.IntentoOriginalFirmable{}, ErrRepositorioNoDisponible
	}
	return intento, nil
}

// ConfirmarOriginalFirmable coteja el objeto validado por la aplicación con
// el intento reservado y consume otra concesión V3 en la transacción SQL.
func (r *Repositorio) ConfirmarOriginalFirmable(ctx context.Context, confirmacion ports.ConfirmacionOriginalFirmable) (domain.Documento, error) {
	preimagen, err := docapp.PreimagenConfirmacionOriginalFirmable(confirmacion)
	if err != nil || confirmacion.Autorizacion.RecursoRef != confirmacion.Intento.DocumentoID ||
		!domain.ReferenciaExpedienteValida(confirmacion.Autorizacion.AmbitoRef) {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	material, err := validarAutorizacionOriginalFirmable(confirmacion.Autorizacion, ports.AccionConfirmarOriginalFirmable, preimagen)
	if err != nil {
		return domain.Documento{}, err
	}
	auth, err := autorizacionJSON(confirmacion.Autorizacion)
	if err != nil {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	objeto, err := json.Marshal(confirmacion.Objeto)
	if err != nil {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	raw, err := r.transaccion(ctx,
		"SELECT vec_documentos.confirmar_original_firmable_v1($1,$2::jsonb,$3::jsonb,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)",
		preimagen, string(objeto), string(auth), material.capacidad, material.decision,
		material.motivo, material.contexto, material.persona, material.perfil,
		material.payload, material.sobre, material.evidencia, material.raiz)
	if err != nil {
		return domain.Documento{}, err
	}
	d, err := decodificarDocumento(raw)
	if err != nil || d.ID != confirmacion.Intento.DocumentoID ||
		d.HuellaSHA256 != confirmacion.Intento.HuellaSHA256 ||
		d.ObjetoRef != confirmacion.Objeto.ObjetoRef ||
		d.ObjetoVersion != confirmacion.Objeto.ObjetoVersion {
		return domain.Documento{}, ErrRepositorioNoDisponible
	}
	return d, nil
}

var _ ports.RepositorioOriginalFirmable = (*Repositorio)(nil)
