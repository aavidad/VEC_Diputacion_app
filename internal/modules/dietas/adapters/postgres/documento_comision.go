package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

var _ dietasports.RepositorioConsultaDocumento = (*RepositorioBorradorComisionPostgreSQL)(nil)

func (r *RepositorioBorradorComisionPostgreSQL) ObtenerDocumentoPropio(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, referencia string) (dietasports.ResultadoBorradorComision, error) {
	solicitud, err := application.NuevaSolicitudOperacionObtenerDocumento(referencia, identidad.Relacion.RelacionRef)
	if err != nil {
		return dietasports.ResultadoBorradorComision{}, err
	}
	bruto, err := r.consultarDocumento(ctx, identidad, solicitud, referencia)
	if err != nil {
		return dietasports.ResultadoBorradorComision{}, err
	}
	resultado, err := decodificarResultadoDocumento(bruto)
	if errors.Is(err, errResultadoBorradorNoEncontrado) {
		return dietasports.ResultadoBorradorComision{}, dietasports.ErrComisionNoEncontrada
	}
	if err != nil {
		return dietasports.ResultadoBorradorComision{}, dietasports.ErrBorradorNoDisponible
	}
	return resultado, nil
}

func (r *RepositorioBorradorComisionPostgreSQL) ListarDocumentosPropios(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, consulta dietasports.ConsultaBorradoresPropios) (dietasports.PaginaBorradoresPropios, error) {
	solicitud, err := application.NuevaSolicitudOperacionListarDocumento(consulta, identidad.Relacion.RelacionRef)
	if err != nil {
		return dietasports.PaginaBorradoresPropios{}, err
	}
	bruto, err := r.consultarDocumento(ctx, identidad, solicitud, "dietas:borradores:propios")
	if err != nil {
		return dietasports.PaginaBorradoresPropios{}, err
	}
	pagina, err := decodificarPaginaDocumento(bruto)
	limite := consulta.Limite
	if limite == 0 {
		limite = 20
	}
	if err != nil || len(pagina.Items) > limite {
		return dietasports.PaginaBorradoresPropios{}, dietasports.ErrBorradorNoDisponible
	}
	return pagina, nil
}

// Cada mutación llama una sola función nominal. La función revalida Personal,
// consume la concesión AD3 y escribe versión, historia, auditoría y recibo en
// la misma transacción serializable abierta por el adaptador.
func (r *RepositorioBorradorComisionPostgreSQL) EditarPropio(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, solicitud dietasports.SolicitudEditarComisionPropia) (dietasports.ResultadoBorradorComision, error) {
	op, err := application.NuevaSolicitudOperacionEditarBorrador(solicitud)
	if err != nil {
		return dietasports.ResultadoBorradorComision{}, err
	}
	return r.mutarPropio(ctx, identidad, op, "dietas.borrador.propio.editar", "editar_borrador_propio")
}

// RecuperarEdicionPorClave consulta un efecto ya confirmado con la intención
// declarada, antes de recalcular OSRM o tarifas que podrían haber cambiado.
// Una ausencia autorizada necesita otra concesión para la edición posterior.
func (r *RepositorioBorradorComisionPostgreSQL) RecuperarEdicionPorClave(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, solicitud dietasports.SolicitudEditarComisionPropia) (dietasports.ResultadoBorradorComision, bool, error) {
	var cero dietasports.ResultadoBorradorComision
	if err := r.valido(ctx); err != nil {
		return cero, false, err
	}
	op, err := application.NuevaSolicitudOperacionPrepararEdicion(solicitud)
	if err != nil {
		return cero, false, err
	}
	efecto, err := construirEfectoBorrador(identidad, op, "dietas.borrador.propio.editar", op.Referencia, "editar_borrador_propio")
	if err != nil {
		return cero, false, err
	}
	var bruto []byte
	if err := r.ejecutarBrutoConModo(ctx, identidad, recuperarMutacionPorClaveSQL, efecto.Material, &bruto, false); err != nil {
		return cero, false, err
	}
	var presencia struct {
		Encontrado bool `json:"encontrado"`
	}
	if json.Unmarshal(bruto, &presencia) != nil {
		return cero, false, dietasports.ErrBorradorNoDisponible
	}
	if !presencia.Encontrado {
		var objeto map[string]json.RawMessage
		if json.Unmarshal(bruto, &objeto) != nil || len(objeto) != 1 || string(objeto["encontrado"]) != "false" {
			return cero, false, dietasports.ErrBorradorNoDisponible
		}
		return cero, false, nil
	}
	resultado, err := decodificarResultadoDocumento(bruto)
	if err != nil || !resultado.Recibo.Repeticion {
		return cero, false, dietasports.ErrBorradorNoDisponible
	}
	return resultado, true, nil
}

func (r *RepositorioBorradorComisionPostgreSQL) BorrarPropio(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, solicitud dietasports.SolicitudMutacionComisionPropia) (dietasports.ResultadoBorradorComision, error) {
	op, err := application.NuevaSolicitudOperacionMutarBorrador(dietasports.OperacionBorrarBorrador, solicitud)
	if err != nil {
		return dietasports.ResultadoBorradorComision{}, err
	}
	return r.mutarPropio(ctx, identidad, op, "dietas.borrador.propio.borrar", "borrar_borrador_propio")
}

func (r *RepositorioBorradorComisionPostgreSQL) EnviarPropio(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, solicitud dietasports.SolicitudMutacionComisionPropia) (dietasports.ResultadoBorradorComision, error) {
	op, err := application.NuevaSolicitudOperacionMutarBorrador(dietasports.OperacionEnviarBorrador, solicitud)
	if err != nil {
		return dietasports.ResultadoBorradorComision{}, err
	}
	return r.mutarPropio(ctx, identidad, op, "dietas.borrador.propio.enviar", "enviar_borrador_propio")
}

func (r *RepositorioBorradorComisionPostgreSQL) mutarPropio(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, op dietasports.SolicitudOperacionBorrador, accion, finalidad string) (dietasports.ResultadoBorradorComision, error) {
	var cero dietasports.ResultadoBorradorComision
	if err := r.valido(ctx); err != nil {
		return cero, err
	}
	efecto, err := construirEfectoBorrador(identidad, op, accion, op.Referencia, finalidad)
	if err != nil {
		return cero, err
	}
	var bruto []byte
	if err := r.ejecutarBrutoConModo(ctx, identidad, mutarComisionPropiaSQL, efecto.Material, &bruto, true); err != nil {
		return cero, err
	}
	resultado, err := decodificarResultadoDocumento(bruto)
	if err != nil {
		// El COMMIT puede haber ocurrido; la misma clave permite recuperar el
		// recibo sin repetir la mutación ni crear otro efecto.
		return cero, dietasports.ErrResultadoBorradorIncierto
	}
	return resultado, nil
}

type resultadoDocumentoJSON struct {
	Resultado string          `json:"resultado"`
	Comision  json.RawMessage `json:"comision"`
	Recibo    reciboJSON      `json:"recibo"`
}

func decodificarResultadoDocumento(bruto []byte) (dietasports.ResultadoBorradorComision, error) {
	var cero dietasports.ResultadoBorradorComision
	var x resultadoDocumentoJSON
	if json.Unmarshal(bruto, &x) != nil {
		return cero, errors.New("resultado SQL inválido")
	}
	if x.Resultado == "no_encontrado" && len(x.Comision) == 0 && x.Recibo.Referencia == "" && x.Recibo.Version == 0 {
		return cero, errResultadoBorradorNoEncontrado
	}
	var c domain.ComisionBorrador
	if x.Resultado != "concedido" || json.Unmarshal(x.Comision, &c) != nil ||
		c.Validar() != nil || c.Version == 0 || c.Version != x.Recibo.Version ||
		!referenciaReciboBorrador.MatchString(x.Recibo.Referencia) {
		return cero, errors.New("resultado SQL inválido")
	}
	en, err := time.Parse(time.RFC3339Nano, x.Recibo.RegistradoEn)
	_, desfase := en.Zone()
	if err != nil || desfase != 0 || en.Nanosecond()%1000 != 0 {
		return cero, errors.New("instante SQL inválido")
	}
	c.CodigosRuta = append([]string{}, c.CodigosRuta...)
	return dietasports.ResultadoBorradorComision{Comision: c, Recibo: dietasports.ReciboBorradorComision{
		Referencia: x.Recibo.Referencia, Version: x.Recibo.Version, RegistradoEn: en.UTC(), Repeticion: x.Recibo.Repeticion,
	}}, nil
}

func decodificarPaginaDocumento(bruto []byte) (dietasports.PaginaBorradoresPropios, error) {
	var forma map[string]json.RawMessage
	if json.Unmarshal(bruto, &forma) != nil || len(forma["items"]) == 0 || forma["items"][0] != '[' || len(forma["siguiente_cursor"]) == 0 || forma["siguiente_cursor"][0] != '"' {
		return dietasports.PaginaBorradoresPropios{}, errors.New("página SQL inválida")
	}
	var x struct {
		Items     []json.RawMessage `json:"items"`
		Siguiente string            `json:"siguiente_cursor"`
	}
	if json.Unmarshal(bruto, &x) != nil || (x.Siguiente != "" && !referenciaComision(x.Siguiente)) {
		return dietasports.PaginaBorradoresPropios{}, errors.New("página SQL inválida")
	}
	p := dietasports.PaginaBorradoresPropios{Items: make([]dietasports.ResultadoBorradorComision, 0, len(x.Items)), SiguienteCursor: x.Siguiente}
	for _, brutoItem := range x.Items {
		item, err := decodificarResultadoDocumento(brutoItem)
		if err != nil {
			return dietasports.PaginaBorradoresPropios{}, err
		}
		p.Items = append(p.Items, item)
	}
	return p, nil
}

func efectoLecturaDocumento(identidad dietasports.IdentidadEfectivaBorrador, solicitud dietasports.SolicitudOperacionBorrador, recurso string) (dietasports.EfectoAutorizacionBorrador, error) {
	return construirEfectoBorrador(identidad, solicitud, "dietas.documento.propio.consultar", recurso, "consultar_documento_propio_dietas")
}

// El contrato de detalle/lista v2 conserva auditoría de lectura en SQL.
// Este método sirve a la integración cuando se active la migración v2.
func (r *RepositorioBorradorComisionPostgreSQL) consultarDocumento(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, solicitud dietasports.SolicitudOperacionBorrador, recurso string) ([]byte, error) {
	if err := r.valido(ctx); err != nil {
		return nil, err
	}
	efecto, err := efectoLecturaDocumento(identidad, solicitud, recurso)
	if err != nil {
		return nil, err
	}
	var bruto []byte
	if err := r.ejecutarBrutoConModo(ctx, identidad, consultarComisionesPropiasSQL, efecto.Material, &bruto, false); err != nil {
		return nil, err
	}
	return bruto, nil
}
