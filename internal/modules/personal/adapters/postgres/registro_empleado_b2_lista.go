package postgres

import (
	"bytes"
	"context"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const consultaEmpleadosB2SQL = `SELECT vec_personal.consultar_empleados_rrhh_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const maxRespuestaEmpleadosB2 = 512 << 10

var _ ports.RepositorioEmpleadosRegistroB2 = (*RepositorioRegistroEmpleadoB2PostgreSQL)(nil)

// ListarEmpleadosRRHH consume la concesión V3 de la lista en Personal 000021
// y comprueba la forma cerrada de la respuesta antes de devolverla.
func (r *RepositorioRegistroEmpleadoB2PostgreSQL) ListarEmpleadosRRHH(ctx context.Context, o ports.OrdenEmpleadosB2) (ports.ResultadoEmpleadosB2, error) {
	var vacio ports.ResultadoEmpleadosB2
	if r == nil || !materialEmpleadosB2Valido(o.Material, o.Autorizacion) {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, consultaEmpleadosB2SQL, o.Material.Canonico(), o.Autorizacion, maxRespuestaEmpleadosB2, func(bruto []byte) (ports.ResultadoEmpleadosB2, error) {
		return decodificarEmpleadosB2(bruto, o)
	})
	if err != nil {
		return vacio, errorDominioRegistroEmpleadoB2(err)
	}
	return resultado, nil
}

func materialEmpleadosB2Valido(m domain.MaterialConsultaRegistroEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	if m.Operacion() != "empleados" {
		return false
	}
	reconstruido, err := domain.NuevoMaterialEmpleadosB2(domain.SolicitudEmpleadosB2{OrganismoRef: m.OrganismoRef(), Corte: m.Corte(), Limite: m.Limite(), Cursor: m.Cursor(), Actor: m.Actor()})
	return err == nil && bytes.Equal(reconstruido.Canonico(), m.Canonico()) && autorizacionConsultaEmpleadoB2Valida(m, a, domain.AccionEmpleadosB2, domain.AudienciaEmpleadosB2)
}

func decodificarEmpleadosB2(bruto []byte, o ports.OrdenEmpleadosB2) (ports.ResultadoEmpleadosB2, error) {
	var vacio ports.ResultadoEmpleadosB2
	if !formaRespuestaEmpleadosB2(bruto) {
		return vacio, errRegistroEmpleadoB2NoDisponible
	}
	var r ports.ResultadoEmpleadosB2
	if err := decodificarJSONRegistroB2(bruto, &r); err != nil || r.Pagina.ValidarPara(o.Material) != nil || !evidenciaRegistroB2Valida(r.Evidencia, o.Autorizacion) {
		return vacio, errRegistroEmpleadoB2NoDisponible
	}
	return r, nil
}

// La forma es cerrada en cada nivel: ninguna clave adicional (por ejemplo,
// una referencia de persona) puede atravesar el adaptador.
func formaRespuestaEmpleadosB2(bruto []byte) bool {
	if len(bruto) == 0 || bytes.Equal(bytes.TrimSpace(bruto), []byte("null")) || verificarJSONOrganizacionHistorica(bruto) != nil {
		return false
	}
	var raiz, pagina, evidencia, corte map[string]json.RawMessage
	if json.Unmarshal(bruto, &raiz) != nil || !clavesRegistroB2(raiz, []string{"pagina", "evidencia"}, nil) ||
		json.Unmarshal(raiz["pagina"], &pagina) != nil || json.Unmarshal(raiz["evidencia"], &evidencia) != nil ||
		!clavesRegistroB2(evidencia, []string{"recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en"}, nil) ||
		!clavesRegistroB2(pagina, []string{"organismo_ref", "corte", "limite", "cursor", "cursor_siguiente", "empleados"}, nil) ||
		json.Unmarshal(pagina["corte"], &corte) != nil || !clavesRegistroB2(corte, []string{"vigente_en", "conocido_en"}, nil) {
		return false
	}
	var empleados []map[string]json.RawMessage
	if json.Unmarshal(pagina["empleados"], &empleados) != nil || empleados == nil {
		return false
	}
	for _, e := range empleados {
		var relaciones []map[string]json.RawMessage
		if !clavesRegistroB2(e, []string{"empleado_ref", "relaciones"}, nil) || json.Unmarshal(e["relaciones"], &relaciones) != nil || relaciones == nil {
			return false
		}
		for _, r := range relaciones {
			var traza map[string]json.RawMessage
			if !clavesRegistroB2(r, []string{"relacion_ref", "estado", "unidad_ref", "unidad_denominacion", "puesto_denominacion", "regimen_denominacion", "modalidad_denominacion", "traza"}, nil) ||
				json.Unmarshal(r["traza"], &traza) != nil || !clavesRegistroB2(traza, []string{"desde", "registrada_en", "version", "acto_ref", "fuente_ref", "fuente_version"}, []string{"hasta"}) {
				return false
			}
		}
	}
	return true
}
