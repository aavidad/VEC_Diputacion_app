package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioConsultaRPTPublicaV2 struct {
	fuente      ports.FuenteRPTPublicaV2
	autorizador ports.ProveedorAutorizacionRPTPublicaV2
	consumidor  ports.ConsumidorLecturaRPTPublicaV2
}

func NuevoServicioConsultaRPTPublicaV2(f ports.FuenteRPTPublicaV2, a ports.ProveedorAutorizacionRPTPublicaV2, c ports.ConsumidorLecturaRPTPublicaV2) (*ServicioConsultaRPTPublicaV2, error) {
	if nulaRPTPublicaV2(f) || nulaRPTPublicaV2(a) || nulaRPTPublicaV2(c) {
		return nil, domain.ErrRPTPublicaV2NoDisponible
	}
	return &ServicioConsultaRPTPublicaV2{fuente: f, autorizador: a, consumidor: c}, nil
}

func (s *ServicioConsultaRPTPublicaV2) Consultar(ctx context.Context, actor vecdomain.ContextoActor, filtro domain.FiltroRPTPublicaV2) (ports.PaginaRPTPublicaV2, error) {
	var vacia ports.PaginaRPTPublicaV2
	if s == nil || ctx == nil || nulaRPTPublicaV2(s.fuente) || nulaRPTPublicaV2(s.autorizador) || nulaRPTPublicaV2(s.consumidor) {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	if actor.Validar() != nil || filtro.Validar() != nil {
		return vacia, domain.ErrRPTPublicaV2Invalida
	}
	// La fuente entrega un objeto ya validado por hash. Conservar esta misma
	// instantánea durante material, consumo transaccional y paginación evita
	// mezclar bytes si el fichero cambia entre pasos.
	snapshot, err := s.fuente.ObtenerRPTPublicaV2(ctx)
	if err != nil || snapshot.Validar() != nil {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	material, err := domain.NuevoMaterialConsultaRPTPublicaV2(domain.SolicitudConsultaRPTPublicaV2{Actor: actor, Filtro: filtro, Snapshot: snapshot})
	if err != nil {
		return vacia, domain.ErrRPTPublicaV2Invalida
	}
	autorizacion, err := s.autorizador.AutorizarConsultaRPTPublicaV2(ctx, material)
	if err != nil {
		return vacia, errorRPTPublicaV2Opaco(ctx, err)
	}
	if !autorizacionRPTPublicaV2Valida(material, autorizacion) {
		return vacia, domain.ErrRPTPublicaV2Denegada
	}
	evidencia, err := s.consumidor.ConsumirLecturaRPTPublicaV2(ctx, ports.OrdenLecturaRPTPublicaV2{Material: material, Autorizacion: autorizacion})
	if err != nil {
		return vacia, errorRPTPublicaV2Opaco(ctx, err)
	}
	if ctx.Err() != nil || !evidenciaRPTPublicaV2Valida(material, autorizacion, evidencia) {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	return paginaRPTPublicaV2(snapshot, filtro, evidencia), nil
}

func autorizacionRPTPublicaV2Valida(m domain.MaterialConsultaRPTPublicaV2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	if a.ValidarEstructura() != nil {
		return false
	}
	s := m.Solicitud()
	r := m.Recurso()
	h, err := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && a.PersonaVersion() == s.Actor.Instantanea.PersonaVersion &&
		a.PerfilVersion() == s.Actor.Instantanea.PerfilVersion &&
		x.Operacion() == domain.AccionConsultaRPTPublicaV2 && x.AudienciaConsumo() == domain.AudienciaConsultaRPTPublicaV2 &&
		x.EfectoRef() == r.Referencia && x.EfectoHuellaSHA256() == h
}

func evidenciaRPTPublicaV2Valida(m domain.MaterialConsultaRPTPublicaV2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, e ports.EvidenciaLecturaRPTPublicaV2) bool {
	x := a.ResumenCapacidad()
	_, offset := e.ConsultadaEn.Zone()
	return e.ReciboRef != "" && len(e.ReciboRef) <= 160 && e.AuditoriaRef != "" && len(e.AuditoriaRef) <= 160 &&
		e.ConsumoHuellaSHA256 != "" && len(e.ConsumoHuellaSHA256) == 64 &&
		e.DecisionRef == x.DecisionRef() && e.EfectoRef == m.Recurso().Referencia &&
		!e.ConsultadaEn.IsZero() && offset == 0 && e.ConsultadaEn.Nanosecond()%1000 == 0 &&
		!e.ConsultadaEn.Before(x.EmitidaEn()) && e.ConsultadaEn.Before(x.ExpiraEn())
}

func paginaRPTPublicaV2(s domain.SnapshotRPTPublicaV2, f domain.FiltroRPTPublicaV2, e ports.EvidenciaLecturaRPTPublicaV2) ports.PaginaRPTPublicaV2 {
	p := ports.PaginaRPTPublicaV2{Vista: f.Vista, Limite: f.Limite, Offset: f.Offset, PublicacionRef: s.PublicacionRef,
		Corte: s.Corte, HuellaSHA256: s.HuellaSHA256, Estado: s.Catalogo.Estado, Fuente: s.Catalogo.Fuente,
		Resumen: s.Catalogo.Resumen, CategoriasPendientesGrupo: append([]string(nil), s.Catalogo.CategoriasPendientesGrupo...), Evidencia: e}
	q := textoConsultaRPTV2(f.Q)
	if f.Vista == "puestos" {
		filtrados := make([]domain.PuestoRPTPublicoV2, 0, len(s.Catalogo.Puestos))
		for _, puesto := range s.Catalogo.Puestos {
			// Solo la asignación singular entra en una lista de categoría. Una
			// alternativa detectada no reparte ni duplica la dotación de la fila.
			if f.CategoriaClave != "" && puesto.CategoriaClave != f.CategoriaClave ||
				f.CentroCodigo != "" && puesto.CentroCodigo != f.CentroCodigo {
				continue
			}
			pendientes := make([]string, 0, len(puesto.CategoriasPendientes))
			for _, pendiente := range puesto.CategoriasPendientes {
				pendientes = append(pendientes, pendiente.Denominacion)
			}
			if q == "" || strings.Contains(textoConsultaRPTV2(strings.Join([]string{puesto.Codigo, puesto.Denominacion, puesto.CentroCodigo, puesto.Centro, puesto.Delegacion,
				strings.Join(puesto.Grupos, " "), puesto.Escala, puesto.CategoriaClave, strings.Join(puesto.CategoriasClaves, " "), strings.Join(pendientes, " "), puesto.Tipo, puesto.Provision}, " ")), q) {
				filtrados = append(filtrados, puesto)
			}
		}
		p.Total = len(filtrados)
		p.Puestos = paginaRPTV2(filtrados, f.Offset, f.Limite)
	} else {
		filtradas := make([]domain.CategoriaRPTPublicaV2, 0, len(s.Catalogo.Categorias))
		for _, categoria := range s.Catalogo.Categorias {
			if q == "" || strings.Contains(textoConsultaRPTV2(strings.Join([]string{categoria.Clave, categoria.Denominacion, strings.Join(categoria.Grupos, " "), strings.Join(categoria.Escalas, " ")}, " ")), q) {
				filtradas = append(filtradas, categoria)
			}
		}
		p.Total = len(filtradas)
		p.Categorias = paginaRPTV2(filtradas, f.Offset, f.Limite)
	}
	return p
}

func paginaRPTV2[T any](filas []T, offset, limite int) []T {
	if offset >= len(filas) {
		return []T{}
	}
	fin := offset + limite
	if fin > len(filas) {
		fin = len(filas)
	}
	return filas[offset:fin]
}

func textoConsultaRPTV2(valor string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, norm.NFD.String(strings.ToLower(strings.TrimSpace(valor))))
}

func errorRPTPublicaV2Opaco(ctx context.Context, err error) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if errors.Is(err, domain.ErrRPTPublicaV2Invalida) || errors.Is(err, domain.ErrRPTPublicaV2Denegada) {
		return err
	}
	return domain.ErrRPTPublicaV2NoDisponible
}

func nulaRPTPublicaV2(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}
