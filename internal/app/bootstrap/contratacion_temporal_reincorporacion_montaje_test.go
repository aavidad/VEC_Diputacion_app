package bootstrap

import (
	"testing"
	"time"

	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestReincorporacionExigeLecturaYEscrituraV3Distintas(t *testing.T) {
	for _, caso := range []struct {
		nombre             string
		lectura, escritura bool
	}{
		{"ambas", true, true},
		{"solo lectura", true, false},
		{"solo escritura", false, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
			v, err := soporte.contexto.Vinculo.Datos()
			if err != nil {
				t.Fatal(err)
			}
			var concesiones []vecdomain.ConcesionRol
			if caso.lectura {
				concesiones = append(concesiones, vecdomain.ConcesionRol{Accion: accionConsultarSeguimientoCeseDesarrollo,
					ModuloID: ctports.ModuloContratacion, TipoRecurso: "seguimiento_contratacion_temporal",
					Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: vecdomain.AuthAssuranceHigh})
			}
			if caso.escritura {
				concesiones = append(concesiones, vecdomain.ConcesionRol{Accion: string(ctdomain.AccionRegistrarReincorporacionTitular),
					ModuloID: ctports.ModuloContratacion, TipoRecurso: ctports.TipoRecursoReincorporacionTitular,
					Finalidades: []string{ctports.FinalidadRegistrarReincorporacionTitular}, GarantiaMinima: vecdomain.AuthAssuranceHigh})
			}
			instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef,
				soporte.reloj.Ahora(), "reincorporacion_titular_ct_prueba", "Retorno titular", "retorno-titular-prueba", concesiones,
				[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
			if err != nil {
				t.Fatal(err)
			}
			soporte.reincorporacionTitular = &soporteSeguimientoCeseDesarrollo{instantanea: instantanea}
			servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(soporte, soporte, soporte, soporte,
				soporte.reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			autoridad := &autoridadSeguimientoCeseDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: soporte,
				autorizador: &autorizadorAnalisisContratacionTemporalDesarrollo{delegado: servicio, soporte: soporte, instalado: true}},
				lecturaReincorporacion: servicio}
			ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, cthttp.RutaReincorporacionesTitular)
			err = autoridad.AutorizarLecturaSeguimiento(ctx, organizacionAltaContratacionTemporalDesarrollo, "expediente:1")
			if (err == nil) != caso.lectura {
				t.Fatalf("lectura=%v error=%v", caso.lectura, err)
			}
			recurso := vecdomain.RecursoAutorizable{Referencia: "expediente:1", ModuloID: ctports.ModuloContratacion,
				Tipo: ctports.TipoRecursoReincorporacionTitular,
				Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
					"expediente_ref": "expediente:1", "fase_previa": string(ctdomain.FaseNombramiento), "estado_previo": string(ctdomain.EstadoEnCurso)}}
			_, _, _, _, err = autoridad.exigir(ctx, string(ctdomain.AccionRegistrarReincorporacionTitular),
				ctports.FinalidadRegistrarReincorporacionTitular, recurso)
			if (err == nil) != caso.escritura {
				t.Fatalf("escritura=%v error=%v", caso.escritura, err)
			}
			ajena := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, cthttp.RutaCesesNombramiento)
			if err := autoridad.AutorizarLecturaSeguimiento(ajena, organizacionAltaContratacionTemporalDesarrollo, "expediente:1"); err == nil {
				t.Fatal("capacidad de ruta ajena admitida para retorno")
			}
		})
	}
}
