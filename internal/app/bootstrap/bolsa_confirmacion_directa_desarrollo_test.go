package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func concesionesFuenteBolsaPrueba(t *testing.T, nombre string, acciones ...string) []dominiovec.ConcesionRol {
	t.Helper()
	datos, err := os.ReadFile(filepath.Join("..", "..", "..", "deploy", "principal", nombre))
	if err != nil {
		t.Fatal(err)
	}
	var fuente struct {
		Entradas []struct {
			Concesion dominiovec.ConcesionRol `json:"concesion"`
		} `json:"entradas"`
	}
	if err := json.Unmarshal(datos, &fuente); err != nil {
		t.Fatal(err)
	}
	var resultado []dominiovec.ConcesionRol
	for _, entrada := range fuente.Entradas {
		for _, accion := range acciones {
			if entrada.Concesion.Accion == accion {
				resultado = append(resultado, entrada.Concesion)
			}
		}
	}
	if len(resultado) != len(acciones) {
		t.Fatalf("fuente %s: faltan concesiones", nombre)
	}
	return resultado
}

func TestConfirmacionDirectaAdoptaRol21GobernadoSinProvision(t *testing.T) {
	p, autoridad, datos := politicaProvisionBolsaPrueba(t, 0)
	i, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef, p.reloj.Ahora(), 6)
	if err != nil {
		t.Fatal(err)
	}
	i.VersionRol.Concesiones = append(i.VersionRol.Concesiones,
		concesionesFuenteBolsaPrueba(t, "fuentes/bolsa_carga_convoca_b1_v1.json", puertosbolsa.AccionConfirmarCargaConvoca)...)
	i.VersionRol.Concesiones = append(i.VersionRol.Concesiones,
		concesionesFuenteBolsaPrueba(t, "inscripcion_recorrido_20261013/inscripcion_bolsa_actos_v1.json",
			"bolsa.inscripcion.rrhh.motivos", "bolsa.inscripcion.rrhh.decidir", "bolsa.inscripcion.rrhh.incorporar")...)
	i.VersionRol.Concesiones = append(i.VersionRol.Concesiones,
		concesionesFuenteBolsaPrueba(t, "fuentes/bolsa_lecturas_rrhh_b103_v1.json",
			"bolsa.inscripcion.rrhh.listar", "bolsa.inscripcion.rrhh.consultar",
			"bolsa.inscripcion.rrhh.convocatorias.listar", "bolsa.rrhh.respuestas_llamamiento.consultar")...)
	i.VersionRol.Concesiones = append(i.VersionRol.Concesiones,
		concesionesFuenteBolsaPrueba(t, "fuentes/bolsa_origen_contacto_rrhh_v1.json",
			dominiovec.AccionLecturaOrigenContactoBolsa)...)
	i.VersionRol.Concesiones = append(i.VersionRol.Concesiones,
		concesionesFuenteBolsaPrueba(t, "fuentes/bolsa_transiciones_situacion_rrhh_v1.json",
			puertosbolsa.AccionConsultarTransicionesSituacion, puertosbolsa.AccionPublicarTransicionesSituacion)...)
	i.VersionRol.Concesiones = append(i.VersionRol.Concesiones,
		concesionesFuenteBolsaPrueba(t, "fuentes/bolsa_confirmacion_aceptacion_rrhh_v1.json",
			puertosbolsa.AccionConfirmarAceptacionDirecta)...)
	if len(i.VersionRol.Concesiones) != 21 {
		t.Fatalf("rol gobernado incompleto: %d", len(i.VersionRol.Concesiones))
	}
	i.VersionRol.Version = 11
	i.VersionRol.PublicadaPor = "seguridad:acto-gobernado"
	i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
	i.AsignacionPerfil.EmitidaPor = "identidad:acto-gobernado"
	i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.ActualizadoPor = i.VersionRol.PublicadaPor
	if err := i.Validar(); err != nil {
		t.Fatal(err)
	}
	autoridad.leida, autoridad.encontrada = instantaneaPublicadaDesarrollo{instantanea: i}, true
	if err := p.PublicarInicialConPoliticaOfertas(context.Background()); err != nil ||
		autoridad.publicadas != 0 || autoridad.preparadas != 0 || p.instantanea.VersionRol.Version != 11 {
		t.Fatalf("el rol de 21 concesiones no se adoptó intacto: %v", err)
	}
	for nombre, alterar := range map[string]func(*dominiovec.InstantaneaAutorizacion){
		"confirmacion_duplicada": func(x *dominiovec.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones = append(x.VersionRol.Concesiones, x.VersionRol.Concesiones[len(x.VersionRol.Concesiones)-1])
		},
		"campo_confirmacion": func(x *dominiovec.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones[len(x.VersionRol.Concesiones)-1].CamposPermitidos = []string{"persona_ref"}
		},
		"finalidad_confirmacion": func(x *dominiovec.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones[len(x.VersionRol.Concesiones)-1].Finalidades = []string{"otra_finalidad"}
		},
		"garantia_confirmacion": func(x *dominiovec.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones[len(x.VersionRol.Concesiones)-1].GarantiaMinima = dominiovec.AuthAssuranceLow
		},
		"accion_ajena": func(x *dominiovec.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones[len(x.VersionRol.Concesiones)-1].Accion = "bolsa.accion_ajena"
		},
		"campo_b103": func(x *dominiovec.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones[13].CamposPermitidos = append(x.VersionRol.Concesiones[13].CamposPermitidos, "persona_ref")
		},
		"tipo_s6": func(x *dominiovec.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones[18].TipoRecurso = "otro_recurso"
		},
		"b1_alterada": func(x *dominiovec.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones[9].TipoRecurso = "otro_recurso"
		},
		"ambito_ajeno": func(x *dominiovec.InstantaneaAutorizacion) {
			x.AsignacionPerfil.Ambitos[0].Valores[0] = "unidad:otra"
		},
		"persona_ajena": func(x *dominiovec.InstantaneaAutorizacion) {
			x.AsignacionPerfil.PrincipalID = "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			x := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(i)
			alterar(&x)
			if instantaneaBolsaConfirmacionCompatible(x, datos, p.soporte, p.reloj.Ahora(), 6) {
				t.Fatal("instantánea alterada adoptada")
			}
		})
	}
}

func TestConfirmacionDirectaPoliticaRegistraConcesionYDenegacion(t *testing.T) {
	for _, permitida := range []bool{true, false} {
		politica, soporte, _, registro := nuevaPoliticaBorradorBolsaPrueba(t)
		if err := politica.PublicarInicial(context.Background()); err != nil {
			t.Fatal(err)
		}
		if permitida {
			politica.instantanea.VersionRol.Concesiones = append(politica.instantanea.VersionRol.Concesiones,
				dominiovec.ConcesionRol{Accion: puertosbolsa.AccionConfirmarAceptacionDirecta,
					ModuloID: "bolsa", TipoRecurso: puertosbolsa.TipoRecursoConfirmarAceptacionDirecta,
					Finalidades: []string{puertosbolsa.FinalidadConfirmarAceptacionDirecta}, GarantiaMinima: dominiovec.AuthAssuranceHigh})
		}
		generador := seguridadvec.GeneradorReferenciasCriptograficas{}
		correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generador)
		if err != nil {
			t.Fatal(err)
		}
		solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: soporte.soporteCanal.contexto.Vinculo,
			ReferenciaMotivo:          motivoCambiarSituacionParticipacionBolsaDesarrollo(),
			Accion:                    puertosbolsa.AccionConfirmarAceptacionDirecta, Finalidad: puertosbolsa.FinalidadConfirmarAceptacionDirecta,
			Recurso: dominiovec.RecursoAutorizable{Referencia: "participacion:01", ModuloID: "bolsa",
				Tipo:      puertosbolsa.TipoRecursoConfirmarAceptacionDirecta,
				Ambitos:   map[string]string{"unidad_ref": soporte.unidadRef, "ambito_ref": soporte.ambitoRef},
				Atributos: map[string]string{"confirmacion_aceptacion_sha256": strings.Repeat("a", 64)}}, Correlacion: correlacion})
		if err != nil {
			t.Fatal(err)
		}
		servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(politica, politica, politica,
			politica, relojContratacionTemporalDesarrollo{}, generador,
			aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: time.Second})
		if err != nil {
			t.Fatal(err)
		}
		_, orden, err := servicio.PrepararSolicitudLigadaV3(context.Background(), solicitud, soporte.soporteCanal.contexto.Resultado)
		if permitida {
			if err != nil {
				t.Fatal(err)
			}
			if _, err := politica.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Background(), orden); err != nil {
				t.Fatal(err)
			}
			if registro.concesiones != 1 || registro.denegaciones != 0 {
				t.Fatalf("concesión sin registrar: %+v", registro)
			}
		} else if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || registro.denegaciones != 1 || registro.concesiones != 0 {
			t.Fatalf("denegación sin registrar: err=%v registro=%+v", err, registro)
		}
	}
}

func TestConfirmacionDirectaAdoptaVersionPublicadaSinPublicarSemilla(t *testing.T) {
	p, autoridad, datos := politicaProvisionBolsaPrueba(t, 0)
	i, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef, p.reloj.Ahora(), 5)
	if err != nil {
		t.Fatal(err)
	}
	i.VersionRol.Version = 24
	i.VersionRol.PublicadaPor = "seguridad:acto-gobernado"
	i.AsignacionPerfil.EmitidaPor = "identidad:acto-gobernado"
	i.VersionRol.Concesiones = append(i.VersionRol.Concesiones, dominiovec.ConcesionRol{
		Accion: puertosbolsa.AccionConfirmarAceptacionDirecta, ModuloID: "bolsa",
		TipoRecurso:    puertosbolsa.TipoRecursoConfirmarAceptacionDirecta,
		Finalidades:    []string{puertosbolsa.FinalidadConfirmarAceptacionDirecta},
		GarantiaMinima: dominiovec.AuthAssuranceHigh}, concesionCargaConvocaPrueba())
	i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.ActualizadoPor = i.VersionRol.PublicadaPor
	if err := i.Validar(); err != nil {
		t.Fatal(err)
	}
	autoridad.leida, autoridad.encontrada = instantaneaPublicadaDesarrollo{instantanea: i}, true
	if err := p.PublicarInicial(context.Background()); err != nil || autoridad.publicadas != 0 ||
		autoridad.preparadas != 0 || p.instantanea.VersionRol.Version != 24 {
		t.Fatalf("se reconstruyó o rechazó la versión publicada: %v", err)
	}
}

func TestConfirmacionDirectaAdoptaSinB1ConYSinS6(t *testing.T) {
	for _, conS6 := range []bool{false, true} {
		p, autoridad, datos := politicaProvisionBolsaPrueba(t, 0)
		i, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
			datos.PrincipalID, datos.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef, p.reloj.Ahora(), 5)
		if err != nil {
			t.Fatal(err)
		}
		if conS6 {
			i.VersionRol.Concesiones = append(i.VersionRol.Concesiones,
				concesionesFuenteBolsaPrueba(t, "fuentes/bolsa_transiciones_situacion_rrhh_v1.json",
					puertosbolsa.AccionConsultarTransicionesSituacion, puertosbolsa.AccionPublicarTransicionesSituacion)...)
		}
		i.VersionRol.Concesiones = append(i.VersionRol.Concesiones,
			concesionesFuenteBolsaPrueba(t, "fuentes/bolsa_confirmacion_aceptacion_rrhh_v1.json",
				puertosbolsa.AccionConfirmarAceptacionDirecta)...)
		i.VersionRol.Version = 24
		i.VersionRol.PublicadaPor = "seguridad:acto-gobernado"
		i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
		i.AsignacionPerfil.EmitidaPor = "identidad:acto-gobernado"
		i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
		i.ControlVigenciaVersionRol.ActualizadoPor = i.VersionRol.PublicadaPor
		if err := i.Validar(); err != nil {
			t.Fatal(err)
		}
		autoridad.leida, autoridad.encontrada = instantaneaPublicadaDesarrollo{instantanea: i}, true
		if err := p.PublicarInicial(context.Background()); err != nil || autoridad.publicadas != 0 ||
			autoridad.preparadas != 0 || p.instantanea.VersionRol.Version != 24 {
			t.Fatalf("F171 sin B1 y con S6=%v no se adoptó: %v", conS6, err)
		}
	}
}

func TestConfirmacionDirectaAdoptaSoloExtrasExactosIncluidoB1(t *testing.T) {
	confirmacion := dominiovec.ConcesionRol{Accion: puertosbolsa.AccionConfirmarAceptacionDirecta,
		ModuloID: "bolsa", TipoRecurso: puertosbolsa.TipoRecursoConfirmarAceptacionDirecta,
		Finalidades:    []string{puertosbolsa.FinalidadConfirmarAceptacionDirecta},
		GarantiaMinima: dominiovec.AuthAssuranceHigh}
	b1 := dominiovec.ConcesionRol{Accion: puertosbolsa.AccionConfirmarCargaConvoca,
		ModuloID: puertosbolsa.ModuloCargaConvoca, TipoRecurso: puertosbolsa.TipoRecursoCargaConvoca,
		Finalidades:    []string{puertosbolsa.FinalidadConfirmarCargaConvoca},
		GarantiaMinima: dominiovec.AuthAssuranceHigh}
	if !concesionesConfirmacionDirectaExactas([]dominiovec.ConcesionRol{confirmacion, b1}) ||
		!contieneAccionConfirmacionDirecta([]dominiovec.ConcesionRol{confirmacion}) {
		t.Fatal("la concesión nominal junto a B1 exacta fue rechazada")
	}
	consultaS6 := dominiovec.ConcesionRol{Accion: puertosbolsa.AccionConsultarTransicionesSituacion,
		ModuloID: "bolsa", TipoRecurso: puertosbolsa.TipoRecursoGobiernoTransicionesSituacion,
		Finalidades:      []string{puertosbolsa.FinalidadConsultaTransicionesSituacion},
		CamposPermitidos: []string{puertosbolsa.CampoConsultaTransicionesSituacion},
		GarantiaMinima:   dominiovec.AuthAssuranceHigh}
	if !concesionesConfirmacionDirectaExactas([]dominiovec.ConcesionRol{confirmacion, b1, consultaS6}) {
		t.Fatal("la consulta S6 exacta no convivió con el acto")
	}
	alterada := b1
	alterada.TipoRecurso = "otro_recurso"
	consultaAlterada := consultaS6
	consultaAlterada.TipoRecurso = "otro_recurso"
	for _, extras := range [][]dominiovec.ConcesionRol{
		{confirmacion, alterada}, {confirmacion, b1, b1}, {confirmacion, consultaAlterada},
		{confirmacion, {Accion: "bolsa.otra_accion", GarantiaMinima: dominiovec.AuthAssuranceHigh}},
		{confirmacion, b1, {Accion: "bolsa.otra_accion", GarantiaMinima: dominiovec.AuthAssuranceHigh}},
	} {
		if concesionesConfirmacionDirectaExactas(extras) {
			t.Fatalf("se admitió concesión ajena o B1 alterada: %+v", extras)
		}
	}
}

func TestConfirmacionDirectaMuxLlegaAlHandlerNominal(t *testing.T) {
	ruta := bolsahttp.RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/respuestas/llamamiento:01/confirmacion"
	llamadas := 0
	m := &manejadorParticipacionBolsaDesarrollo{confirmacionDirecta: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusConflict)
	})}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodPost, ruta, nil))
	if w.Code != http.StatusConflict || llamadas != 1 {
		t.Fatalf("el mux no entregó el acto: estado=%d llamadas=%d", w.Code, llamadas)
	}
}
